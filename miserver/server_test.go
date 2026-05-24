package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShanghaiTimeReturnsDecryptableCurrentInterfaceResponse(t *testing.T) {
	cfg := DefaultConfig()
	now := fixedLATime()
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Now:    func() time.Time { return now },
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/shanghaitime", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != float64(200) {
		t.Fatalf("code = %v, want 200", resp["code"])
	}
	if _, ok := resp["message"]; ok {
		t.Fatalf("message field should not be present in current interface response: %#v", resp)
	}

	data, ok := resp["data"].(string)
	if !ok || data == "" {
		t.Fatalf("data = %T %q, want encrypted string", resp["data"], resp["data"])
	}
	plain, err := decryptResponseStringAt(data, cfg, now)
	if err != nil {
		t.Fatalf("decrypt response: %v", err)
	}
	if plain != "2026-05-24 15:26:30" {
		t.Fatalf("plain = %q", plain)
	}
}

func TestGetDeviceReturnsCurrentInterfacePlainPayload(t *testing.T) {
	cfg := DefaultConfig()
	now := fixedLATime()
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Now:    func() time.Time { return now },
	})

	body := bytes.NewBufferString(`{"device_id":"1546c952"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/get_device", body)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["encrypted_data"]; ok {
		t.Fatalf("encrypted_data should not be present in current interface response: %#v", resp)
	}
	if resp["success"] != true {
		t.Fatalf("success = %v, want true", resp["success"])
	}
	if resp["设备id"] != "1546c952" {
		t.Fatalf("设备id = %v", resp["设备id"])
	}
	if resp["天数"] != float64(30) {
		t.Fatalf("天数 = %v, want 30", resp["天数"])
	}
	if resp["开始时间"] != "2026-05-24 15:26" {
		t.Fatalf("开始时间 = %v", resp["开始时间"])
	}
	if resp["到期时间"] != "2026-06-23 15:26:30" {
		t.Fatalf("到期时间 = %v", resp["到期时间"])
	}
}

func TestUseCodeReturnsCurrentInterfaceInvalidCodeResponse(t *testing.T) {
	cfg := DefaultConfig()
	now := fixedLATime()
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Now:    func() time.Time { return now },
	})

	body := bytes.NewBufferString(`{"device_id":"1546c952","code":"ABC123"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/use_code", body)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["encrypted_data"]; ok {
		t.Fatalf("encrypted_data should not be present in current interface response: %#v", resp)
	}
	if resp["success"] != false {
		t.Fatalf("success = %v, want false", resp["success"])
	}
	if resp["error"] != "失败,授权码无效" {
		t.Fatalf("error = %v", resp["error"])
	}
}

func TestUploadDecryptsChineseFieldsAndReturnsCurrentInterfaceMessage(t *testing.T) {
	cfg := DefaultConfig()
	srv := NewServer(ServerConfig{Crypto: cfg})

	encryptedDevice, err := encryptUploadFixtureString("1546c952", cfg)
	if err != nil {
		t.Fatalf("encrypt device: %v", err)
	}
	body := encryptedUploadFixture(t, cfg, map[string]string{
		"设备":   "1546c952",
		"当前时间": "2026-05-24 16:08:50",
		"手机号":  "13800138000",
		"账号":   "qq123",
		"密码":   "pwd123",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/上传", bytes.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantMessage := "设备 " + encryptedDevice + " 已存在相同的账号密码，不会重复保存。"
	if resp["消息"] != wantMessage {
		t.Fatalf("消息 = %v, want %q", resp["消息"], wantMessage)
	}
}

func TestAccessLogIncludesSuccessfulRequests(t *testing.T) {
	var logs bytes.Buffer
	srv := NewServer(ServerConfig{
		Crypto:    DefaultConfig(),
		LogOutput: &logs,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/get_device?debug=1", bytes.NewBufferString(`{"device_id":"1546c952"}`))
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("User-Agent", "miserver-test")
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	logLine := logs.String()
	for _, want := range []string{
		"POST /get_device?debug=1 200 ",
		"B ",
	} {
		if !bytes.Contains([]byte(logLine), []byte(want)) {
			t.Fatalf("log %q does not contain %q", logLine, want)
		}
	}
	for _, unwanted := range []string{"remote=", "proto=", "user_agent=", "path=", "method=", "status="} {
		if bytes.Contains([]byte(logLine), []byte(unwanted)) {
			t.Fatalf("log %q should not contain verbose field %q", logLine, unwanted)
		}
	}
}

func TestAccessLogIncludesNotFoundRequests(t *testing.T) {
	var logs bytes.Buffer
	srv := NewServer(ServerConfig{
		Crypto:    DefaultConfig(),
		LogOutput: &logs,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/missing/path", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	logLine := logs.String()
	for _, want := range []string{
		"POST /missing/path 404 ",
		"B ",
	} {
		if !bytes.Contains([]byte(logLine), []byte(want)) {
			t.Fatalf("log %q does not contain %q", logLine, want)
		}
	}
}

func TestEndpointsRejectNonPostMethods(t *testing.T) {
	srv := NewServer(ServerConfig{Crypto: DefaultConfig()})

	for _, path := range []string{"/shanghaitime", "/get_device", "/use_code", "/上传"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		srv.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405", path, rec.Code)
		}
	}
}

func encryptedUploadFixture(t *testing.T, cfg CryptoConfig, fields map[string]string) []byte {
	t.Helper()

	payload := make(map[string]string, len(fields))
	for key, plain := range fields {
		encrypted, err := encryptUploadFixtureString(plain, cfg)
		if err != nil {
			t.Fatalf("encrypt field %s: %v", key, err)
		}
		payload[key] = encrypted
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal upload fixture: %v", err)
	}
	return raw
}

func encryptUploadFixtureString(plain string, cfg CryptoConfig) (string, error) {
	padded := pkcs7Pad([]byte(plain), aesBlockSize)
	encrypted, err := encryptCBC(padded, cfg.Seed, cfg.IV)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func fixedLATime() time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
	return time.Date(2026, 5, 24, 0, 26, 30, 0, loc)
}
