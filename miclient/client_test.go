package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShanghaiTimeUsesPost(t *testing.T) {
	cfg := DefaultConfig()
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "2026-05-24 12:34:56"})
	}))
	defer server.Close()

	client := NewClient(server.URL, cfg)
	if _, err := client.ShanghaiTime(); err != nil {
		t.Fatalf("ShanghaiTime returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/shanghaitime" {
		t.Fatalf("path = %q, want /shanghaitime", gotPath)
	}
}

func TestUseCodePostsCodeAndDeviceID(t *testing.T) {
	cfg := DefaultConfig()
	var gotPath string
	var gotBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": "invalid code"})
	}))
	defer server.Close()

	client := NewClient(server.URL, cfg)
	resp, err := client.UseCode("1546c952", "ABC123")
	if err != nil {
		t.Fatalf("UseCode returned error: %v", err)
	}

	if gotPath != "/use_code" {
		t.Fatalf("path = %q, want /use_code", gotPath)
	}
	if gotBody["code"] != "ABC123" {
		t.Fatalf("code = %q, want ABC123", gotBody["code"])
	}
	if gotBody["device_id"] != "1546c952" {
		t.Fatalf("device_id = %q, want 1546c952", gotBody["device_id"])
	}
	if _, ok := gotBody["device"]; ok {
		t.Fatalf("device field should not be sent: %#v", gotBody)
	}
	if resp["success"] != false {
		t.Fatalf("success = %v", resp["success"])
	}
}

func TestGetDeviceUsesPostJSONBody(t *testing.T) {
	cfg := DefaultConfig()
	var gotMethod string
	var gotPath string
	var gotBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer server.Close()

	client := NewClient(server.URL, cfg)
	if _, err := client.GetDevice("1546c952"); err != nil {
		t.Fatalf("GetDevice returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/get_device" {
		t.Fatalf("path = %q, want /get_device", gotPath)
	}
	if gotBody["device_id"] != "1546c952" {
		t.Fatalf("device_id = %q", gotBody["device_id"])
	}
}

func TestUploadPostsEncryptedChineseFields(t *testing.T) {
	cfg := DefaultConfig()
	var gotMethod string
	var gotPath string
	var gotBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer server.Close()

	client := NewClient(server.URL, cfg)
	resp, err := client.Upload("1546c952", "2026-05-24 16:08:50", "13800138000", "qq123", "pwd123")
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/上传" {
		t.Fatalf("path = %q, want /上传", gotPath)
	}

	want := map[string]string{
		"设备":   "1546c952",
		"当前时间": "2026-05-24 16:08:50",
		"手机号":  "13800138000",
		"账号":   "qq123",
		"密码":   "pwd123",
	}
	for key, plain := range want {
		encrypted := gotBody[key]
		if encrypted == "" {
			t.Fatalf("missing encrypted field %q in %#v", key, gotBody)
		}
		if encrypted == plain {
			t.Fatalf("field %q was not encrypted", key)
		}
		decrypted, err := decryptString(encrypted, cfg)
		if err != nil {
			t.Fatalf("decrypt field %q: %v", key, err)
		}
		if decrypted != plain {
			t.Fatalf("field %q decrypts to %q, want %q", key, decrypted, plain)
		}
	}
	if resp["success"] != true {
		t.Fatalf("success = %v", resp["success"])
	}
}

func TestEncryptedStringResponseAddsDecryptedString(t *testing.T) {
	cfg := DefaultConfig()
	encryptedResponse, err := encryptString("2026-05-24 12:34:56", cfg)
	if err != nil {
		t.Fatalf("encrypt fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"data": encryptedResponse})
	}))
	defer server.Close()

	client := NewClient(server.URL, cfg)
	resp, err := client.ShanghaiTime()
	if err != nil {
		t.Fatalf("ShanghaiTime returned error: %v", err)
	}

	if resp["decrypted_data"] != "2026-05-24 12:34:56" {
		t.Fatalf("decrypted_data = %v", resp["decrypted_data"])
	}
}

func TestDecryptErrorIsReturnedWhenEncryptedDataCannotBeDecoded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ResponseSeedPrefix = "wrong-prefix"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 200,
			"data": "PdNuDfl3XC3Vk9KpQYJz8NbtYq6aLymBXe+aIYt3e6gbr7cSi+qqhbP/yAnQ+/dGZV2jyl",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, cfg)
	resp, err := client.ShanghaiTime()
	if err != nil {
		t.Fatalf("ShanghaiTime returned error: %v", err)
	}

	if resp["decrypted_data"] != nil {
		t.Fatalf("decrypted_data = %v, want nil", resp["decrypted_data"])
	}
	if resp["decrypt_error"] == "" {
		t.Fatalf("decrypt_error was empty")
	}
}
