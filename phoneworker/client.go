package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const smsReceiveModeUserSent = "USER_SENT_TO_TX"

type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type listData struct {
	DeviceIdleCount int64 `json:"deviceIdleCount"`
}

type createTaskData struct {
	ID uint `json:"id"`
}

type phoneSourceResponse struct {
	Code int    `json:"code"`
	Data string `json:"data"`
}

type SystemClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewSystemClient(baseURL, token string, timeout time.Duration) *SystemClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &SystemClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *SystemClient) IdleDeviceCount(ctx context.Context) (int64, error) {
	var data listData
	if err := c.doJSON(ctx, http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil, &data); err != nil {
		return 0, err
	}
	return data.DeviceIdleCount, nil
}

func (c *SystemClient) CreateUserSentTask(ctx context.Context, phone string) (uint, error) {
	payload := map[string]string{
		"phone":          strings.TrimSpace(phone),
		"smsReceiveMode": smsReceiveModeUserSent,
	}
	var data createTaskData
	if err := c.doJSON(ctx, http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", payload, &data); err != nil {
		return 0, err
	}
	return data.ID, nil
}

func (c *SystemClient) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var reqBody io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("X-Open-Api-Token", c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s failed: http %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var wrapper apiResponse
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return fmt.Errorf("decode response: %w: %s", err, string(body))
	}
	if wrapper.Code != 0 {
		if strings.TrimSpace(wrapper.Msg) != "" {
			return errors.New(wrapper.Msg)
		}
		return fmt.Errorf("api code %d", wrapper.Code)
	}
	if out == nil || len(wrapper.Data) == 0 || string(wrapper.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		return fmt.Errorf("decode data: %w: %s", err, string(wrapper.Data))
	}
	return nil
}

type PhoneSourceClient struct {
	endpoint string
	http     *http.Client
}

func NewPhoneSourceClient(endpoint string, timeout time.Duration) *PhoneSourceClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &PhoneSourceClient{
		endpoint: strings.TrimSpace(endpoint),
		http:     &http.Client{Timeout: timeout},
	}
}

func (c *PhoneSourceClient) FetchPhone(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("phone source failed: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out phoneSourceResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode phone source: %w: %s", err, string(body))
	}
	if out.Code != 0 || strings.TrimSpace(out.Data) == "" {
		return "", fmt.Errorf("phone source returned code %d", out.Code)
	}
	phone := strings.TrimSpace(out.Data)
	if !isElevenDigitPhone(phone) {
		return "", fmt.Errorf("phone source returned invalid phone %q", phone)
	}
	return phone, nil
}

func isElevenDigitPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	for _, ch := range phone {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func buildPhoneSourceURL(base string) (string, error) {
	if strings.Contains(base, "?") {
		if _, err := url.ParseRequestURI(base); err != nil {
			return "", err
		}
		return base, nil
	}
	if _, err := url.ParseRequestURI(base); err != nil {
		return "", err
	}
	return base, nil
}
