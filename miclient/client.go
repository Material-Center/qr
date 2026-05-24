package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "http://py.j8nda.xyz:9999"

type Client struct {
	baseURL    string
	crypto     CryptoConfig
	httpClient *http.Client
}

type APIResponse map[string]any

func NewClient(baseURL string, cfg CryptoConfig) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		crypto:  cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

func (c *Client) ShanghaiTime() (APIResponse, error) {
	var out APIResponse
	if err := c.doJSON(context.Background(), http.MethodPost, "/shanghaitime", nil, &out); err != nil {
		return nil, err
	}
	return c.decryptResponseFields(out)
}

func (c *Client) GetDevice(deviceID string) (APIResponse, error) {
	payload := map[string]string{"device_id": deviceID}
	var out APIResponse
	if err := c.doJSON(context.Background(), http.MethodPost, "/get_device", payload, &out); err != nil {
		return nil, err
	}
	return c.decryptResponseFields(out)
}

func (c *Client) UseCode(device, code string) (APIResponse, error) {
	payload := map[string]string{"device_id": device}
	payload["code"] = code

	var out APIResponse
	if err := c.doJSON(context.Background(), http.MethodPost, "/use_code", payload, &out); err != nil {
		return nil, err
	}
	return c.decryptResponseFields(out)
}

func (c *Client) Upload(device, currentTime, phone, account, password string) (APIResponse, error) {
	payload, err := encryptedUploadPayload(device, currentTime, phone, account, password, c.crypto)
	if err != nil {
		return nil, err
	}

	var out APIResponse
	if err := c.doJSON(context.Background(), http.MethodPost, "/上传", payload, &out); err != nil {
		return nil, err
	}
	return c.decryptResponseFields(out)
}

func encryptedUploadPayload(device, currentTime, phone, account, password string, cfg CryptoConfig) (map[string]string, error) {
	fields := map[string]string{
		"设备":   device,
		"当前时间": currentTime,
		"手机号":  phone,
		"账号":   account,
		"密码":   password,
	}

	payload := make(map[string]string, len(fields))
	for key, plain := range fields {
		encrypted, err := encryptString(plain, cfg)
		if err != nil {
			return nil, fmt.Errorf("encrypt upload field %s: %w", key, err)
		}
		payload[key] = encrypted
	}
	return payload, nil
}

func (c *Client) decryptResponseFields(resp APIResponse) (APIResponse, error) {
	for _, field := range []string{"data", "encrypted_data"} {
		raw, ok := resp[field].(string)
		if !ok || raw == "" {
			continue
		}

		decrypted, err := decryptResponseString(raw, c.crypto)
		if err != nil {
			resp["decrypt_error"] = fmt.Sprintf("%s: %v", field, err)
			continue
		}

		resp["decrypted_data"] = parseDecryptedValue(decrypted)
		resp["decrypted_field"] = field
		return resp, nil
	}

	return resp, nil
}

func parseDecryptedValue(decrypted string) any {
	var nested map[string]any
	if err := json.Unmarshal([]byte(decrypted), &nested); err == nil {
		return nested
	}
	return decrypted
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode json: %w: %s", err, string(raw))
	}

	return nil
}
