package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	SMSReceiveModeUserSent     = "USER_SENT_TO_TX"
	SMSReceiveModePlatformSend = "PLATFORM_SEND"
)

const OpenAPIDeviceCapacityNotEnoughCode = "OPENAPI_DEVICE_CAPACITY_NOT_ENOUGH"

var ErrOpenAPIDeviceCapacityNotEnough = errors.New("OpenAPI可用设备不足")

type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type DeviceStats struct {
	DeviceOnlineCount int64 `json:"deviceOnlineCount"`
	DeviceIdleCount   int64 `json:"deviceIdleCount"`
}

type TaskInfo struct {
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
	stats, err := c.DeviceStats(ctx)
	if err != nil {
		return 0, err
	}
	return stats.DeviceIdleCount, nil
}

func (c *SystemClient) DeviceStats(ctx context.Context) (DeviceStats, error) {
	var data DeviceStats
	if err := c.doJSON(ctx, http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil, &data); err != nil {
		return DeviceStats{}, err
	}
	return data, nil
}

func (c *SystemClient) CreateSendCodeTask(ctx context.Context, phone string, createDelay time.Duration) (TaskInfo, error) {
	payload := taskPayload(phone, SMSReceiveModeUserSent, createDelay)
	var data TaskInfo
	if err := c.doJSON(ctx, http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", payload, &data); err != nil {
		return TaskInfo{}, err
	}
	return data, nil
}

func (c *SystemClient) CreateReceiveCodeTask(ctx context.Context, phone string, createDelay time.Duration) (TaskInfo, error) {
	payload := taskPayload(phone, SMSReceiveModePlatformSend, createDelay)
	var data TaskInfo
	if err := c.doJSON(ctx, http.MethodPost, "/phoneRegisterTask/open-api/promoter/receive-task", payload, &data); err != nil {
		return TaskInfo{}, err
	}
	return data, nil
}

func (c *SystemClient) GetTask(ctx context.Context, taskID uint) (TaskInfo, error) {
	var data TaskInfo
	path := fmt.Sprintf("/phoneRegisterTask/open-api/promoter/task/%d", taskID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &data); err != nil {
		return TaskInfo{}, err
	}
	return data, nil
}

func (c *SystemClient) SubmitCode(ctx context.Context, taskID uint, verifyCode string) (TaskInfo, error) {
	payload := map[string]any{
		"taskId":     taskID,
		"verifyCode": strings.TrimSpace(verifyCode),
	}
	var data TaskInfo
	if err := c.doJSON(ctx, http.MethodPost, "/phoneRegisterTask/open-api/promoter/submit-code", payload, &data); err != nil {
		return TaskInfo{}, err
	}
	return data, nil
}

func taskPayload(phone string, mode string, createDelay time.Duration) map[string]any {
	payload := map[string]any{
		"phone":          strings.TrimSpace(phone),
		"smsReceiveMode": mode,
	}
	if createDelay > 0 {
		delaySeconds := int(createDelay.Round(time.Second) / time.Second)
		if delaySeconds < 1 {
			delaySeconds = 1
		}
		payload["startDelaySeconds"] = delaySeconds
		payload["reserveDevice"] = true
	}
	return payload
}

func (c *SystemClient) doJSON(ctx context.Context, method string, path string, payload any, out any) error {
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
		return apiWrapperError(wrapper)
	}
	if out == nil || len(wrapper.Data) == 0 || string(wrapper.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		return fmt.Errorf("decode data: %w: %s", err, string(wrapper.Data))
	}
	return nil
}

func apiWrapperError(wrapper apiResponse) error {
	var data struct {
		ErrorCode string `json:"errorCode"`
	}
	if len(wrapper.Data) > 0 {
		_ = json.Unmarshal(wrapper.Data, &data)
	}
	if strings.TrimSpace(data.ErrorCode) == OpenAPIDeviceCapacityNotEnoughCode {
		return ErrOpenAPIDeviceCapacityNotEnough
	}
	if strings.TrimSpace(wrapper.Msg) != "" {
		return errors.New(wrapper.Msg)
	}
	return fmt.Errorf("api code %d", wrapper.Code)
}
