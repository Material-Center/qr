package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultEnvBaseURL = "http://39.108.96.33:8888"

type EnvClient struct {
	baseURL    string
	crypto     CryptoConfig
	httpClient *http.Client
}

type EnvRecord struct {
	DeviceCode       string
	DeviceID         string
	Type             string
	SerialBackupName string
	AndroidID        string
	Key              string
	MadeCount        int
	LastUsedAt       string
}

type EnvFilter struct {
	Type             string
	DeviceCode       string
	DeviceID         string
	SerialBackupName string
	AndroidID        string
	Key              string
	Frozen           *int
	Limit            *int
	Offset           *int
	MaxUsage         *int
	MinUsage         *int
	MinMadeCount     *int
	MaxMadeCount     *int
	OlderThanDays    *int
	MinDays          *int
	MaxDays          *int
	CooldownDays     *int
	Sort             string
}

func DefaultEnvConfig() CryptoConfig {
	cfg := DefaultConfig()
	cfg.Seed = "06250511"
	return cfg
}

func NewEnvClient(baseURL string, cfg CryptoConfig) *EnvClient {
	if baseURL == "" {
		baseURL = defaultEnvBaseURL
	}
	return &EnvClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		crypto:  cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *EnvClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

func (c *EnvClient) AddEnv(record EnvRecord) (APIResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.postEncrypted("/add_env", record.payload())
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
		}
	}
	return nil, lastErr
}

func (c *EnvClient) GetEnv(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/get_env", filter.payload(false))
}

func (c *EnvClient) GetEnvEnhanced(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/get_env_enhanced", filter.payload(false))
}

func (c *EnvClient) GetEnvEnhanced2(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/get_env_enhanced2", filter.payload(false))
}

func (c *EnvClient) GetEnvForMake(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/get_env_for_make", filter.payload(false))
}

func (c *EnvClient) MakeSuccess(id int) (APIResponse, error) {
	return c.postEncrypted("/make_success", map[string]any{"环境id": id})
}

func (c *EnvClient) IncreaseMakeCount(id int) (APIResponse, error) {
	return c.postEncrypted("/increase_make_count", map[string]any{"环境id": id})
}
func (c *EnvClient) DecreaseMakeCount(id int) (APIResponse, error) {
	return c.postEncrypted("/decrease_make_count", map[string]any{"环境id": id})
}
func (c *EnvClient) ResetMakeCount(id int) (APIResponse, error) {
	return c.postEncrypted("/reset_make_count", map[string]any{"环境id": id})
}

func (c *EnvClient) GetEnvFixed(deviceID string) (APIResponse, error) {
	return c.postEncrypted("/get_env_fixed", map[string]any{"设备ID": deviceID})
}

func (c *EnvClient) StatsByType() (APIResponse, error) {
	return c.postEncrypted("/stats_by_type", map[string]any{})
}

func (c *EnvClient) StatsMakeProgress() (APIResponse, error) {
	return c.postEncrypted("/stats_make_progress", map[string]any{})
}

func (c *EnvClient) Total() (APIResponse, error) { return c.postEncrypted("/total", map[string]any{}) }
func (c *EnvClient) Available() (APIResponse, error) {
	return c.postEncrypted("/available", map[string]any{})
}
func (c *EnvClient) Frozen() (APIResponse, error) {
	return c.postEncrypted("/frozen", map[string]any{})
}
func (c *EnvClient) Unused() (APIResponse, error) {
	return c.postEncrypted("/unused", map[string]any{})
}

func (c *EnvClient) QueryEnvList(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/query_env_list", filter.payload(true))
}

func (c *EnvClient) QueryEnv(id int) (APIResponse, error) {
	return c.postEncrypted("/query_env", map[string]any{"环境id": id})
}

func (c *EnvClient) FreezeEnv(id int) (APIResponse, error) {
	return c.postEncrypted("/freeze_env", map[string]any{"环境id": id})
}

func (c *EnvClient) UnfreezeEnv(id int) (APIResponse, error) {
	return c.postEncrypted("/unfreeze_env", map[string]any{"环境id": id})
}

func (c *EnvClient) FreezeByCondition(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/freeze_by_condition", filter.payload(true))
}
func (c *EnvClient) UnfreezeByCondition(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/unfreeze_by_condition", filter.payload(true))
}

func (c *EnvClient) DeleteEnv(id int) (APIResponse, error) {
	return c.postEncrypted("/delete_env", map[string]any{"环境id": id})
}

func (c *EnvClient) DeleteByCondition(filter EnvFilter) (APIResponse, error) {
	return c.postEncrypted("/delete_by_condition", filter.payload(true))
}

func (c *EnvClient) CleanEnv() (APIResponse, error) {
	return c.postEncrypted("/clean_env", map[string]any{})
}

func (c *EnvClient) CleanEnvOlderThan(days int) (APIResponse, error) {
	return c.postEncrypted("/clean_env", map[string]any{"超过天数": days})
}

func (c *EnvClient) QueryByDevice(deviceID string, limit int) (APIResponse, error) {
	payload := map[string]any{"设备ID": deviceID}
	if limit > 0 {
		payload["limit"] = limit
	}
	return c.postEncrypted("/query_by_device", payload)
}

func (c *EnvClient) Stats() (APIResponse, error) {
	var out APIResponse
	if err := c.doJSON(context.Background(), http.MethodGet, "/stats", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *EnvClient) postEncrypted(path string, payload map[string]any) (APIResponse, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal encrypted payload: %w", err)
	}
	encrypted, err := encryptDynamicString(string(raw), c.crypto)
	if err != nil {
		return nil, fmt.Errorf("encrypt payload: %w", err)
	}

	var envelope APIResponse
	if err := c.doJSON(context.Background(), http.MethodPost, path, map[string]string{"data": encrypted}, &envelope); err != nil {
		return nil, err
	}
	return c.unwrapEncryptedResponse(envelope)
}

func (c *EnvClient) unwrapEncryptedResponse(envelope APIResponse) (APIResponse, error) {
	raw, ok := envelope["data"].(string)
	if !ok || raw == "" {
		return envelope, nil
	}

	decrypted, err := decryptResponseString(raw, c.crypto)
	if err != nil {
		envelope["decrypt_error"] = err.Error()
		return envelope, nil
	}

	var out APIResponse
	if err := json.Unmarshal([]byte(decrypted), &out); err != nil {
		envelope["decrypted_data"] = decrypted
		return envelope, nil
	}
	return out, nil
}

func (c *EnvClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	client := Client{
		baseURL:    c.baseURL,
		crypto:     c.crypto,
		httpClient: c.httpClient,
	}
	return client.doJSON(ctx, method, path, body, out)
}

func (r EnvRecord) payload() map[string]any {
	return map[string]any{
		"设备代号":    r.DeviceCode,
		"设备ID":    r.DeviceID,
		"类型":      r.Type,
		"串码备份包名称": r.SerialBackupName,
		"安卓ID":    r.AndroidID,
		"密钥":      r.Key,
	}
}

func (f EnvFilter) payload(includeListFields bool) map[string]any {
	payload := map[string]any{}
	addString(payload, "类型", f.Type)
	addString(payload, "设备代号", f.DeviceCode)
	addString(payload, "设备ID", f.DeviceID)
	addString(payload, "串码备份包名称", f.SerialBackupName)
	addString(payload, "安卓ID", f.AndroidID)
	addString(payload, "密钥", f.Key)
	addIntPtr(payload, "最大使用次数", f.MaxUsage)
	addIntPtr(payload, "最小使用次数", f.MinUsage)
	addIntPtr(payload, "最小制作次数", f.MinMadeCount)
	addIntPtr(payload, "最大制作次数", f.MaxMadeCount)
	addIntPtr(payload, "超过天数", f.OlderThanDays)
	addIntPtr(payload, "最小天数", f.MinDays)
	addIntPtr(payload, "最大天数", f.MaxDays)
	addIntPtr(payload, "冷却天数", f.CooldownDays)
	addString(payload, "排序", f.Sort)
	if includeListFields {
		addIntPtr(payload, "冻结", f.Frozen)
		addIntPtr(payload, "limit", f.Limit)
		addIntPtr(payload, "offset", f.Offset)
	}
	return payload
}

func addString(payload map[string]any, key, value string) {
	if value != "" {
		payload[key] = value
	}
}

func addIntPtr(payload map[string]any, key string, value *int) {
	if value != nil {
		payload[key] = *value
	}
}
