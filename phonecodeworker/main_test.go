package main

import (
	"strings"
	"testing"
	"time"
)

func TestFormatStartupSettingsIncludesVersionAndMasksSecrets(t *testing.T) {
	text := formatStartupSettings(startupSettings{
		Version:       "dev",
		GitCommit:     "abc1234",
		BuildTime:     "2026-06-19T00:00:00Z",
		BaseURL:       "http://example.test/api",
		Token:         "secret-openapi-token",
		Input:         "phones.txt",
		StatePath:     "phones.txt.state.json",
		FailedPath:    "phones.txt.failed.txt",
		SuccessPath:   "phones.txt.success.txt",
		PauseFile:     "phonecodeworker.pause",
		LogDir:        "logs",
		CodeAPI:       "https://code.test/api?t=raw-secret&phone=",
		PhoneCount:    12,
		Interval:      3 * time.Second,
		IdleThreshold: 1,
		CreateDelay:   0,
		TaskSyncLimit: 3,
		Timeout:       10 * time.Second,
		Once:          false,
	})

	for _, want := range []string{
		"version=dev",
		"gitCommit=abc1234",
		"buildTime=2026-06-19T00:00:00Z",
		"baseURL=http://example.test/api",
		"input=phones.txt",
		"successOutput=phones.txt.success.txt",
		"phones=12",
		"interval=3s",
		"taskSyncLimit=3",
		"token=se***************ken",
		"codeAPI=https://code.test/api?phone=&t=***",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("startup settings missing %q in %q", want, text)
		}
	}
	for _, leaked := range []string{"secret-openapi-token", "raw-secret"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("startup settings leaked secret %q in %q", leaked, text)
		}
	}
}
