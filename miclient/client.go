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
const defaultUploadBaseURL = "http://120.77.84.13"

type Client struct {
	baseURL    string
	crypto     CryptoConfig
	httpClient *http.Client
}

type APIResponse map[string]any

type LicenseStatus struct {
	DeviceID  string
	ServerNow time.Time
	ExpiresAt time.Time
	Days      int
}

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

// CheckLicense mirrors the current desktop authorization state machine:
// obtain Shanghai time first, then query the device, and compare both values
// as Asia/Shanghai timestamps. It retries transient failures with bounded
// backoff and never falls back to the local clock for authorization.
func (c *Client) CheckLicense(ctx context.Context, deviceID string) (LicenseStatus, error) {
	if strings.TrimSpace(deviceID) == "" {
		return LicenseStatus{}, fmt.Errorf("device_id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		serverResp, err := c.doJSONContext(ctx, http.MethodPost, "/shanghaitime", nil)
		if err == nil {
			serverResp, err = c.decryptResponseFields(serverResp)
		}
		if err == nil {
			serverNow, parseErr := parseShanghaiTime(serverResp["decrypted_data"])
			if parseErr == nil {
				deviceResp, deviceErr := c.doJSONContext(ctx, http.MethodPost, "/get_device", map[string]string{"device_id": deviceID})
				if deviceErr == nil {
					deviceResp, deviceErr = c.decryptResponseFields(deviceResp)
				}
				if deviceErr == nil {
					expires, expiryErr := parseExpiry(deviceResp)
					if expiryErr == nil {
						status := LicenseStatus{DeviceID: deviceID, ServerNow: serverNow, ExpiresAt: expires, Days: int(expires.Sub(serverNow).Hours() / 24)}
						if serverNow.After(expires) {
							return status, fmt.Errorf("license expired at %s", expires.Format("2006-01-02 15:04:05"))
						}
						if deviceResp["success"] == false {
							return status, fmt.Errorf("device is not authorized")
						}
						return status, nil
					}
					lastErr = expiryErr
				} else {
					lastErr = deviceErr
				}
			} else {
				lastErr = parseErr
			}
		} else {
			lastErr = err
		}
		if attempt < 4 {
			delay := time.Duration(attempt+1) * 200 * time.Millisecond
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return LicenseStatus{}, ctx.Err()
			case <-timer.C:
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("license check failed")
	}
	return LicenseStatus{}, lastErr
}

func parseShanghaiTime(value any) (time.Time, error) {
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return time.Time{}, fmt.Errorf("missing Shanghai server time")
	}
	return time.ParseInLocation("2006-01-02 15:04:05", s, shanghaiLocation())
}

func parseExpiry(resp APIResponse) (time.Time, error) {
	loc := shanghaiLocation()
	for _, key := range []string{"到期时间", "expires_at", "expiry", "data"} {
		if value, ok := resp[key].(string); ok && strings.TrimSpace(value) != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", value, loc)
			if err == nil {
				return t, nil
			}
			if t, err := time.ParseInLocation("2006-01-02 15:04", value, loc); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("missing or invalid license expiry")
}

func shanghaiLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(fmt.Errorf("load Asia/Shanghai timezone: %w", err))
	}
	return loc
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

func (c *Client) doJSONContext(ctx context.Context, method, path string, body any) (APIResponse, error) {
	var out APIResponse
	if err := c.doJSON(ctx, method, path, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}
