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
	"regexp"
	"strings"
	"time"
)

const smsReceiveModePlatformSend = "PLATFORM_SEND"

var errCodeNotReady = errors.New("验证码未就绪")

type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type deviceStatsData struct {
	DeviceIdleCount int64 `json:"deviceIdleCount"`
}

type taskInfo struct {
	ID               uint       `json:"id"`
	Phone            string     `json:"phone"`
	SMSReceiveMode   string     `json:"smsReceiveMode"`
	Status           string     `json:"status"`
	StatusCode       *int       `json:"statusCode"`
	LastError        string     `json:"lastError"`
	NeedPromoterCode bool       `json:"needPromoterCode"`
	FinishedAt       *time.Time `json:"finishedAt"`
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
	var data deviceStatsData
	if err := c.doJSON(ctx, http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil, &data); err != nil {
		return 0, err
	}
	return data.DeviceIdleCount, nil
}

func (c *SystemClient) CreateReceiveTask(ctx context.Context, phone string, createDelay time.Duration) (taskInfo, error) {
	payload := map[string]any{
		"phone":          strings.TrimSpace(phone),
		"smsReceiveMode": smsReceiveModePlatformSend,
	}
	if createDelay > 0 {
		delaySeconds := int(createDelay.Round(time.Second) / time.Second)
		if delaySeconds < 1 {
			delaySeconds = 1
		}
		payload["startDelaySeconds"] = delaySeconds
		payload["reserveDevice"] = true
	}
	var data taskInfo
	if err := c.doJSON(ctx, http.MethodPost, "/phoneRegisterTask/open-api/promoter/receive-task", payload, &data); err != nil {
		return taskInfo{}, err
	}
	return data, nil
}

func (c *SystemClient) GetTask(ctx context.Context, taskID uint) (taskInfo, error) {
	var data taskInfo
	path := fmt.Sprintf("/phoneRegisterTask/open-api/promoter/task/%d", taskID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &data); err != nil {
		return taskInfo{}, err
	}
	return data, nil
}

func (c *SystemClient) SubmitCode(ctx context.Context, phone string, taskID uint, verifyCode string) (taskInfo, error) {
	payload := map[string]any{
		"taskId":     taskID,
		"verifyCode": strings.TrimSpace(verifyCode),
	}
	var data taskInfo
	if err := c.doSubmitCodeJSON(ctx, strings.TrimSpace(phone), taskID, payload, &data); err != nil {
		return taskInfo{}, err
	}
	return data, nil
}

func (c *SystemClient) doSubmitCodeJSON(ctx context.Context, phone string, taskID uint, payload any, out any) error {
	const path = "/phoneRegisterTask/open-api/promoter/submit-code"
	start := time.Now()
	c.logf("system api submit-code request phone=%s task=%d method=%s path=%s", phone, taskID, http.MethodPost, path)
	var reqBody io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			c.logf("system api submit-code marshal error phone=%s task=%d elapsed=%s err=%v", phone, taskID, time.Since(start).Round(time.Millisecond), err)
			return err
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reqBody)
	if err != nil {
		c.logf("system api submit-code build request error phone=%s task=%d elapsed=%s err=%v", phone, taskID, time.Since(start).Round(time.Millisecond), err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Open-Api-Token", c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.logf("system api submit-code error phone=%s task=%d elapsed=%s timeout=%t err=%v",
			phone, taskID, time.Since(start).Round(time.Millisecond), isContextTimeout(err), err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logf("system api submit-code read body error phone=%s task=%d status=%d elapsed=%s err=%v",
			phone, taskID, resp.StatusCode, time.Since(start).Round(time.Millisecond), err)
		return err
	}
	elapsed := time.Since(start).Round(time.Millisecond)
	c.logf("system api submit-code response phone=%s task=%d status=%d elapsed=%s bodyBytes=%d body=%q",
		phone, taskID, resp.StatusCode, elapsed, len(body), compactLogText(string(body), 2000))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s failed: http %d: %s", http.MethodPost, path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var wrapper apiResponse
	if err := json.Unmarshal(body, &wrapper); err != nil {
		c.logf("system api submit-code decode error phone=%s task=%d elapsed=%s err=%v body=%q",
			phone, taskID, elapsed, err, compactLogText(string(body), 2000))
		return fmt.Errorf("decode response: %w: %s", err, string(body))
	}
	if wrapper.Code != 0 {
		c.logf("system api submit-code api error phone=%s task=%d code=%d msg=%q elapsed=%s",
			phone, taskID, wrapper.Code, wrapper.Msg, elapsed)
		if strings.TrimSpace(wrapper.Msg) != "" {
			return errors.New(wrapper.Msg)
		}
		return fmt.Errorf("api code %d", wrapper.Code)
	}
	if out == nil || len(wrapper.Data) == 0 || string(wrapper.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		c.logf("system api submit-code decode data error phone=%s task=%d elapsed=%s err=%v data=%q",
			phone, taskID, elapsed, err, compactLogText(string(wrapper.Data), 2000))
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

type CodeSourceClient struct {
	api    string
	http   *http.Client
	logger *log.Logger
}

func NewCodeSourceClient(api string, timeout time.Duration) *CodeSourceClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &CodeSourceClient{
		api:    strings.TrimSpace(api),
		http:   &http.Client{Timeout: timeout},
		logger: log.Default(),
	}
}

func (c *CodeSourceClient) FetchCode(ctx context.Context, phone string) (string, error) {
	requestURL := buildCodeURL(c.api, phone)
	c.logf("code api request phone=%s url=%s", strings.TrimSpace(phone), requestURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
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
	contentType := resp.Header.Get("Content-Type")
	c.logf("code api response phone=%s status=%d contentType=%q bodyBytes=%d rawBody=%q",
		strings.TrimSpace(phone),
		resp.StatusCode,
		contentType,
		len(body),
		compactLogText(string(body), 2000),
	)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("code api failed: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	code, source, jsonParsed := extractVerifyCodeDetail(body)
	if code == "" {
		c.logf("code api parse phone=%s json=%t source=%s result=not_ready", strings.TrimSpace(phone), jsonParsed, source)
		return "", errCodeNotReady
	}
	c.logf("code api parse phone=%s json=%t source=%s code=%s", strings.TrimSpace(phone), jsonParsed, source, code)
	return code, nil
}

func buildCodeURL(api string, phone string) string {
	phone = url.QueryEscape(strings.TrimSpace(phone))
	if strings.Contains(api, "{phone}") {
		return strings.ReplaceAll(api, "{phone}", phone)
	}
	return api + phone
}

func extractVerifyCode(body []byte) string {
	code, _, _ := extractVerifyCodeDetail(body)
	return code
}

func extractVerifyCodeDetail(body []byte) (code string, source string, jsonParsed bool) {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err == nil {
		for _, key := range []string{"data", "code", "verifyCode", "sms"} {
			if value, ok := obj[key]; ok {
				if code := firstDigitCode(fmt.Sprint(value)); code != "" {
					return code, "json." + key, true
				}
			}
		}
		return "", "json", true
	}
	if code := firstDigitCode(string(body)); code != "" {
		return code, "text", false
	}
	return "", "text", false
}

func firstDigitCode(raw string) string {
	re := regexp.MustCompile(`\d{4,8}`)
	return re.FindString(raw)
}

func (c *CodeSourceClient) logf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func (c *SystemClient) logf(format string, args ...any) {
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
