package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveFailedImportFileOnlyIncludesFailedRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "phones.failed.txt")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{
		"18507561351",
		"18507561352",
		"18507561353",
		"18507561354",
	})
	state.Records[0].Status = recordStatusFailed
	state.Records[1].Status = recordStatusCreated
	state.Records[1].TaskID = 9
	state.Records[2].Status = recordStatusPending
	state.Records[3].Status = recordStatusSucceeded
	state.Records[3].UpdatedAt = time.Now()

	if err := SaveFailedImportFile(path, state); err != nil {
		t.Fatalf("save failed import file: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed import file: %v", err)
	}
	if got, want := string(raw), "https://code.test/?phone=\n18507561351\n"; got != want {
		t.Fatalf("failed import file = %q, want %q", got, want)
	}
}
