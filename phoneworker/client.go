package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
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
	logger  *log.Logger
}

func NewSystemClient(baseURL, token string, timeout time.Duration) *SystemClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &SystemClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		http:    &http.Client{Timeout: timeout},
		logger:  log.Default(),
	}
}

func (c *SystemClient) IdleDeviceCount(ctx context.Context) (int64, error) {
	var data listData
	if err := c.doJSON(ctx, http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil, &data); err != nil {
		return 0, err
	}
	return data.DeviceIdleCount, nil
}

func (c *SystemClient) CreateUserSentTask(ctx context.Context, phone string, createDelay time.Duration) (uint, error) {
	phone = strings.TrimSpace(phone)
	payload := map[string]any{
		"phone":          phone,
		"smsReceiveMode": smsReceiveModeUserSent,
	}
	if createDelay > 0 {
		delaySeconds := int(createDelay.Round(time.Second) / time.Second)
		if delaySeconds < 1 {
			delaySeconds = 1
		}
		payload["startDelaySeconds"] = delaySeconds
		payload["reserveDevice"] = true
	}
	var data createTaskData
	if err := c.doCreateTaskJSON(ctx, phone, payload, &data); err != nil {
		return 0, err
	}
	return data.ID, nil
}

func (c *SystemClient) doCreateTaskJSON(ctx context.Context, phone string, payload any, out any) error {
	const path = "/phoneRegisterTask/open-api/promoter/task"
	start := time.Now()
	c.logf("system api create-task request phone=%s method=%s path=%s", phone, http.MethodPost, path)
	var reqBody io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			c.logf("system api create-task marshal error phone=%s elapsed=%s err=%v", phone, time.Since(start).Round(time.Millisecond), err)
			return err
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reqBody)
	if err != nil {
		c.logf("system api create-task build request error phone=%s elapsed=%s err=%v", phone, time.Since(start).Round(time.Millisecond), err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Open-Api-Token", c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.logf("system api create-task error phone=%s elapsed=%s timeout=%t err=%v",
			phone, time.Since(start).Round(time.Millisecond), isContextTimeout(err), err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logf("system api create-task read body error phone=%s status=%d elapsed=%s err=%v",
			phone, resp.StatusCode, time.Since(start).Round(time.Millisecond), err)
		return err
	}
	elapsed := time.Since(start).Round(time.Millisecond)
	c.logf("system api create-task response phone=%s status=%d elapsed=%s bodyBytes=%d body=%q",
		phone, resp.StatusCode, elapsed, len(body), compactLogText(string(body), 2000))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s failed: http %d: %s", http.MethodPost, path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var wrapper apiResponse
	if err := json.Unmarshal(body, &wrapper); err != nil {
		c.logf("system api create-task decode error phone=%s elapsed=%s err=%v body=%q",
			phone, elapsed, err, compactLogText(string(body), 2000))
		return fmt.Errorf("decode response: %w: %s", err, string(body))
	}
	if wrapper.Code != 0 {
		c.logf("system api create-task api error phone=%s code=%d msg=%q elapsed=%s",
			phone, wrapper.Code, wrapper.Msg, elapsed)
		if strings.TrimSpace(wrapper.Msg) != "" {
			return errors.New(wrapper.Msg)
		}
		return fmt.Errorf("api code %d", wrapper.Code)
	}
	if out == nil || len(wrapper.Data) == 0 || string(wrapper.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		c.logf("system api create-task decode data error phone=%s elapsed=%s err=%v data=%q",
			phone, elapsed, err, compactLogText(string(wrapper.Data), 2000))
		return fmt.Errorf("decode data: %w: %s", err, string(wrapper.Data))
	}
	return nil
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
	logger   *log.Logger
}

func NewPhoneSourceClient(endpoint string, timeout time.Duration) *PhoneSourceClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &PhoneSourceClient{
		endpoint: strings.TrimSpace(endpoint),
		http:     &http.Client{Timeout: timeout},
		logger:   log.Default(),
	}
}

func (c *PhoneSourceClient) FetchPhone(ctx context.Context) (string, error) {
	start := time.Now()
	c.logf("phone source request url=%s", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		c.logf("phone source build request error elapsed=%s err=%v", time.Since(start).Round(time.Millisecond), err)
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.logf("phone source error elapsed=%s timeout=%t err=%v", time.Since(start).Round(time.Millisecond), isContextTimeout(err), err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logf("phone source read body error status=%d elapsed=%s err=%v", resp.StatusCode, time.Since(start).Round(time.Millisecond), err)
		return "", err
	}
	elapsed := time.Since(start).Round(time.Millisecond)
	c.logf("phone source response status=%d elapsed=%s bodyBytes=%d body=%q",
		resp.StatusCode, elapsed, len(body), compactLogText(string(body), 2000))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("phone source failed: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out phoneSourceResponse
	if err := json.Unmarshal(body, &out); err != nil {
		c.logf("phone source decode error elapsed=%s err=%v body=%q", elapsed, err, compactLogText(string(body), 2000))
		return "", fmt.Errorf("decode phone source: %w: %s", err, string(body))
	}
	if out.Code != 0 || strings.TrimSpace(out.Data) == "" {
		c.logf("phone source api error code=%d elapsed=%s data=%q", out.Code, elapsed, out.Data)
		return "", fmt.Errorf("phone source returned code %d", out.Code)
	}
	phone := strings.TrimSpace(out.Data)
	if !isElevenDigitPhone(phone) {
		c.logf("phone source invalid phone=%q elapsed=%s", phone, elapsed)
		return "", fmt.Errorf("phone source returned invalid phone %q", phone)
	}
	c.logf("phone source parsed phone=%s elapsed=%s", phone, elapsed)
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

func (c *SystemClient) logf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func (c *PhoneSourceClient) logf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func isContextTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func compactLogText(raw string, maxLen int) string {
	text := strings.TrimSpace(raw)
	text = strings.ReplaceAll(text, "\r", "\\r")
	text = strings.ReplaceAll(text, "\n", "\\n")
	if maxLen > 0 && len(text) > maxLen {
		return text[:maxLen] + "...(truncated)"
	}
	return text
}
