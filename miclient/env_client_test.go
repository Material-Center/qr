package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAddEnvPostsEncryptedDataEnvelope(t *testing.T) {
	cfg := DefaultEnvConfig()
	var gotMethod string
	var gotPath string
	var gotEnvelope map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		encrypted, err := encryptString(`{"success":true,"message":"ok"}`, cfg)
		if err != nil {
			t.Fatalf("encrypt response fixture: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"data": encrypted})
	}))
	defer server.Close()

	client := NewEnvClient(server.URL, cfg)
	resp, err := client.AddEnv(EnvRecord{
		DeviceCode:       "cepheus",
		DeviceID:         "8bf9321c",
		Type:             "QQ888",
		SerialBackupName: "backup-a",
		AndroidID:        "android-a",
		Key:              "userkey-a",
	})
	if err != nil {
		t.Fatalf("AddEnv returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/add_env" {
		t.Fatalf("path = %q, want /add_env", gotPath)
	}
	if len(gotEnvelope) != 1 || gotEnvelope["data"] == "" {
		t.Fatalf("envelope = %#v, want only encrypted data", gotEnvelope)
	}
	if staticPlain, err := decryptString(gotEnvelope["data"], cfg); err == nil {
		t.Fatalf("request used static encryption, decrypted to %q", staticPlain)
	}

	plain, err := decryptResponseString(gotEnvelope["data"], cfg)
	if err != nil {
		t.Fatalf("decrypt request envelope: %v", err)
	}
	var gotPlain map[string]string
	if err := json.Unmarshal([]byte(plain), &gotPlain); err != nil {
		t.Fatalf("decode decrypted request: %v", err)
	}
	want := map[string]string{
		"设备代号":    "cepheus",
		"设备ID":    "8bf9321c",
		"类型":      "QQ888",
		"串码备份包名称": "backup-a",
		"安卓ID":    "android-a",
		"密钥":      "userkey-a",
	}
	for key, value := range want {
		if gotPlain[key] != value {
			t.Fatalf("%s = %q, want %q in %#v", key, gotPlain[key], value, gotPlain)
		}
	}
	if resp["success"] != true {
		t.Fatalf("success = %v, want true in %#v", resp["success"], resp)
	}
}

func TestGetEnvPostsFiltersAndUnwrapsDataObject(t *testing.T) {
	cfg := DefaultEnvConfig()
	var gotPath string
	var gotEnvelope map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		encrypted, err := encryptString(`{"success":true,"data":{"备份名称":"env-a","安卓ID":"android-a","密钥":"key-a"}}`, cfg)
		if err != nil {
			t.Fatalf("encrypt response fixture: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"data": encrypted})
	}))
	defer server.Close()

	client := NewEnvClient(server.URL, cfg)
	resp, err := client.GetEnv(EnvFilter{
		Type:             "QQ888",
		DeviceCode:       "cepheus",
		DeviceID:         "8bf9321c",
		MaxUsage:         intPtr(1),
		OlderThanDays:    intPtr(3),
		SerialBackupName: "backup-a",
		AndroidID:        "android-a",
	})
	if err != nil {
		t.Fatalf("GetEnv returned error: %v", err)
	}

	if gotPath != "/get_env" {
		t.Fatalf("path = %q, want /get_env", gotPath)
	}
	plain, err := decryptResponseString(gotEnvelope["data"], cfg)
	if err != nil {
		t.Fatalf("decrypt request envelope: %v", err)
	}
	var gotPlain map[string]any
	if err := json.Unmarshal([]byte(plain), &gotPlain); err != nil {
		t.Fatalf("decode decrypted request: %v", err)
	}
	if gotPlain["类型"] != "QQ888" || gotPlain["设备代号"] != "cepheus" || gotPlain["设备ID"] != "8bf9321c" {
		t.Fatalf("unexpected filter body: %#v", gotPlain)
	}
	if gotPlain["最大使用次数"] != float64(1) || gotPlain["超过天数"] != float64(3) {
		t.Fatalf("unexpected numeric filters: %#v", gotPlain)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %#v, want object", resp["data"])
	}
	if data["备份名称"] != "env-a" || data["安卓ID"] != "android-a" || data["密钥"] != "key-a" {
		t.Fatalf("data = %#v", data)
	}
}

func TestStatsUsesPlainGET(t *testing.T) {
	cfg := DefaultEnvConfig()
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"总数": 2})
	}))
	defer server.Close()

	client := NewEnvClient(server.URL, cfg)
	resp, err := client.Stats()
	if err != nil {
		t.Fatalf("Stats returned error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/stats" {
		t.Fatalf("path = %q, want /stats", gotPath)
	}
	if resp["总数"] != float64(2) {
		t.Fatalf("总数 = %v", resp["总数"])
	}
}

func TestEnvClientUnwrapsObfuscatedErrorResponse(t *testing.T) {
	cfg := DefaultEnvConfig()
	seed := responseSeeds(cfg, time.Now())[cfg.ResponseSkew]
	encrypted := encryptObfuscatedFixture("解密失败: 解密失败", seed, cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 400,
			"data": encrypted,
		})
	}))
	defer server.Close()

	client := NewEnvClient(server.URL, cfg)
	resp, err := client.AddEnv(EnvRecord{
		DeviceCode:       "cepheus",
		DeviceID:         "8bf9321c",
		Type:             "QQ888",
		SerialBackupName: "backup-a",
		AndroidID:        "android-a",
		Key:              "key-a",
	})
	if err != nil {
		t.Fatalf("AddEnv returned error: %v", err)
	}

	if resp["decrypted_data"] != "解密失败: 解密失败" {
		t.Fatalf("decrypted_data = %#v, want decrypted server error in %#v", resp["decrypted_data"], resp)
	}
	if resp["decrypt_error"] != nil {
		t.Fatalf("decrypt_error = %#v", resp["decrypt_error"])
	}
}

func TestEnvClientUnwrapsDynamicResponseWithoutWirePrefix(t *testing.T) {
	cfg := DefaultEnvConfig()
	seed := responseSeeds(cfg, time.Now())[cfg.ResponseSkew]
	encrypted := encryptDynamicFixtureWithoutWirePrefix("解密失败: 解密失败", seed, cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 400,
			"data": encrypted,
		})
	}))
	defer server.Close()

	client := NewEnvClient(server.URL, cfg)
	resp, err := client.AddEnv(EnvRecord{
		DeviceCode:       "cepheus",
		DeviceID:         "8bf9321c",
		Type:             "QQ888",
		SerialBackupName: "backup-a",
		AndroidID:        "android-a",
		Key:              "key-a",
	})
	if err != nil {
		t.Fatalf("AddEnv returned error: %v", err)
	}

	if resp["decrypted_data"] != "解密失败: 解密失败" {
		t.Fatalf("decrypted_data = %#v, want decrypted server error in %#v", resp["decrypted_data"], resp)
	}
}

func intPtr(v int) *int {
	return &v
}

func encryptDynamicFixtureWithoutWirePrefix(plain, seed string, cfg CryptoConfig) string {
	wire := encryptObfuscatedFixture(plain, seed, cfg)
	return wire[6:]
}
