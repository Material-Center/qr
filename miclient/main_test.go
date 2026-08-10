package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunAddEnvCommand(t *testing.T) {
	cfg := DefaultEnvConfig()
	var gotPath string
	var gotEnvelope map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		encrypted, err := encryptString(`{"success":true}`, cfg)
		if err != nil {
			t.Fatalf("encrypt response fixture: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"data": encrypted})
	}))
	defer server.Close()

	var out bytes.Buffer
	err := runWithOutput([]string{
		"-env-base-url", server.URL,
		"-device", "8bf9321c",
		"-device-code", "cepheus",
		"-serial-backup-name", "backup-a",
		"-android-id", "android-a",
		"-key", "key-a",
		"add-env",
	}, &out)
	if err != nil {
		t.Fatalf("runWithOutput returned error: %v", err)
	}

	if gotPath != "/add_env" {
		t.Fatalf("path = %q, want /add_env", gotPath)
	}
	plain, err := decryptResponseString(gotEnvelope["data"], cfg)
	if err != nil {
		t.Fatalf("decrypt request envelope: %v", err)
	}
	var gotPlain map[string]string
	if err := json.Unmarshal([]byte(plain), &gotPlain); err != nil {
		t.Fatalf("decode decrypted request: %v", err)
	}
	if gotPlain["类型"] != "QQ888" || gotPlain["设备ID"] != "8bf9321c" {
		t.Fatalf("unexpected request body: %#v", gotPlain)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"success": true`)) {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRunStatsEnvCommand(t *testing.T) {
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]int{"总数": 7})
	}))
	defer server.Close()

	var out bytes.Buffer
	err := runWithOutput([]string{"-env-base-url", server.URL, "stats-env"}, &out)
	if err != nil {
		t.Fatalf("runWithOutput returned error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/stats" {
		t.Fatalf("path = %q, want /stats", gotPath)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"总数": 7`)) {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRunUploadUsesUploadBaseURL(t *testing.T) {
	var gotUploadPath string
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUploadPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer uploadServer.Close()

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("upload command should not use auth base URL; got %s", r.URL.Path)
	}))
	defer authServer.Close()

	var out bytes.Buffer
	err := runWithOutput([]string{
		"-base-url", authServer.URL,
		"-upload-base-url", uploadServer.URL,
		"-device", "1546c952",
		"-current-time", "2026-05-24 16:08:50",
		"-phone", "13800138000",
		"-account", "qq123",
		"-password", "pwd123",
		"upload",
	}, &out)
	if err != nil {
		t.Fatalf("runWithOutput returned error: %v", err)
	}

	if gotUploadPath != "/上传" {
		t.Fatalf("upload path = %q, want /上传", gotUploadPath)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"success": true`)) {
		t.Fatalf("output = %s", out.String())
	}
}
