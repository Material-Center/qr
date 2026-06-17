package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"phone-task-client/internal/store"
)

func TestCreateJobFromFlagsUsesDefaultBaseURL(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "client.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	input := filepath.Join(t.TempDir(), "phones.txt")
	if err := os.WriteFile(input, []byte("https://code.test/?phone={phone}\n18507561351\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	job, err := createJobFromFlags(st, createJobOptions{
		Token:          "token-1",
		Mode:           "receive",
		PhoneSource:    "txt",
		Input:          input,
		ReserveDevices: 1,
		Interval:       3 * time.Second,
		Timeout:        10 * time.Second,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if job.BaseURLSnapshot != defaultSystemBaseURL {
		t.Fatalf("base url = %q, want %q", job.BaseURLSnapshot, defaultSystemBaseURL)
	}
}
