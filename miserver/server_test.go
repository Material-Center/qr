package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestShanghaiTimeUsesSingleClockSample(t *testing.T) {
	cfg := DefaultConfig()
	now := fixedLATime()
	clockCalls := 0
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Now: func() time.Time {
			clockCalls++
			return now.Add(time.Duration(clockCalls-1) * time.Minute)
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/shanghaitime", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if clockCalls != 1 {
		t.Fatalf("clock calls = %d, want 1", clockCalls)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	plain, err := decryptResponseStringAt(resp["data"].(string), cfg, now)
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
	if resp["天数"].(float64) < 36000 {
		t.Fatalf("天数 = %v, want a long-lived license", resp["天数"])
	}
	if resp["开始时间"] != "2026-05-24 15:26" {
		t.Fatalf("开始时间 = %v", resp["开始时间"])
	}
	if resp["到期时间"] == "" {
		t.Fatalf("到期时间 is empty")
	}
}

func TestGetDeviceReturnsLongLivedStatelessLicense(t *testing.T) {
	store := newTestStore(t)
	start := fixedLATime()
	srv := NewServer(ServerConfig{Crypto: DefaultConfig(), Store: store, Now: func() time.Time { return start }})
	body := bytes.NewBufferString(`{"device_id":"persisted-device"}`)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/get_device", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("initial status = %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["success"] != true {
		t.Fatalf("response = %#v", resp)
	}
	if resp["天数"].(float64) < 36000 {
		t.Fatalf("license is not long-lived: %#v", resp)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM licenses`).Scan(&count); err != nil {
		t.Fatalf("count license rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("get_device should be stateless, licenses rows = %d", count)
	}
}

func TestEnvHandlerForwardsToConfiguredMainServer(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internalTool/miEnv/add_env" || r.URL.RawQuery != "trace=1" {
			t.Fatalf("upstream URL = %q", r.URL.RequestURI())
		}
		if r.Header.Get("X-MI-Internal-Key") != "test-key" {
			t.Fatalf("upstream key = %q", r.Header.Get("X-MI-Internal-Key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("upstream content type = %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept-Encoding") != "identity" {
			t.Fatalf("upstream accept encoding = %q", r.Header.Get("Accept-Encoding"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":"proxied"}`))
	}))
	defer upstream.Close()

	srv := NewServer(ServerConfig{EnvProxy: &EnvProxyConfig{BaseURL: upstream.URL, Path: "/internalTool/miEnv", Key: "test-key"}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/add_env?trace=1", bytes.NewBufferString(`{"data":"payload"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.EnvHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted || rec.Body.String() != `{"data":"proxied"}` {
		t.Fatalf("proxy response = %d %q", rec.Code, rec.Body.String())
	}
}

func TestEnvProxyLogsDecryptedRequestPlaintext(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"proxied"}`))
	}))
	defer upstream.Close()

	plain := `{"设备ID":"f54dbf77","limit":10000}`
	encrypted := encryptEnvRequestFixture(t, plain, cfg, now)
	body, err := json.Marshal(map[string]string{"data": encrypted})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var logs bytes.Buffer
	srv := NewServer(ServerConfig{
		Crypto:    cfg,
		Now:       func() time.Time { return now },
		LogOutput: &logs,
		EnvProxy:  &EnvProxyConfig{BaseURL: upstream.URL, Path: "/internalTool/miEnv", Key: "test-key"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/query_env", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.EnvHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(logs.String(), `env_request service=env method=POST path="/query_env" plaintext=`+plain) {
		t.Fatalf("decrypted request log missing from %q", logs.String())
	}
}

func TestStopTimeReturnsDefaultMockState(t *testing.T) {
	cfg := DefaultConfig()
	now := fixedLATime()
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Now:    func() time.Time { return now },
	})

	encryptedDevice := encryptEnvRequestFixture(t, "1546c952", cfg, now)
	encryptedKey := encryptEnvRequestFixture(t, "diagnostic-key", cfg, now)
	body, err := json.Marshal(map[string]string{
		"encrypted_device": encryptedDevice,
		"encrypted_key":    encryptedKey,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/stoptime", bytes.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != float64(200) {
		t.Fatalf("response = %#v", resp)
	}
	encryptedData, ok := resp["data"].(string)
	if !ok || encryptedData == "" {
		t.Fatalf("encrypted data missing in %#v", resp)
	}
	plain, err := decryptResponseStringAt(encryptedData, cfg, now)
	if err != nil {
		t.Fatalf("decrypt response: %v", err)
	}
	if plain != now.In(shanghaiLocation()).AddDate(100, 0, 0).Format("2006-01-02 15:04:05") {
		t.Fatalf("expiry = %q", plain)
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
		"service=all method=POST path=\"/get_device?debug=1\" status=200",
		"bytes=",
		"remote=\"127.0.0.1:54321\"",
		"user_agent=\"miserver-test\"",
	} {
		if !bytes.Contains([]byte(logLine), []byte(want)) {
			t.Fatalf("log %q does not contain %q", logLine, want)
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
		"service=all method=POST path=\"/missing/path\" status=404",
		"bytes=",
	} {
		if !bytes.Contains([]byte(logLine), []byte(want)) {
			t.Fatalf("log %q does not contain %q", logLine, want)
		}
	}
}

func TestEndpointsRejectNonPostMethods(t *testing.T) {
	srv := NewServer(ServerConfig{Crypto: DefaultConfig()})

	for _, path := range []string{"/use_code", "/stoptime", "/上传"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		srv.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405", path, rec.Code)
		}
	}
}

func TestAuthCompatibilityAcceptsGetForTimeAndDevice(t *testing.T) {
	srv := NewServer(ServerConfig{Crypto: DefaultConfig(), Now: func() time.Time { return fixedLATime() }})

	for _, tc := range []struct {
		path string
		want int
	}{
		{path: "/shanghaitime", want: http.StatusOK},
		{path: "/get_device?device_id=1546c952", want: http.StatusOK},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		srv.AuthHandler().ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("GET %s status = %d, want %d; body=%s", tc.path, rec.Code, tc.want, rec.Body.String())
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

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := OpenStore(":memory:")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close test store: %v", err)
		}
	})
	return store
}

func fixedLATime() time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
	return time.Date(2026, 5, 24, 0, 26, 30, 0, loc)
}
