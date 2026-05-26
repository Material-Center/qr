package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAccountListWritesRecognizedQQText(t *testing.T) {
	t.Parallel()
	out := filepath.Join(t.TempDir(), "accounts.txt")
	lines := []accountListLine{
		{QQNum: "10001", Action: "created"},
		{QQNum: "10002", Action: "updated"},
	}

	if err := writeAccountList(out, lines); err != nil {
		t.Fatalf("writeAccountList() error = %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := "10001----created\r\n10002----updated\r\n"
	if string(raw) != want {
		t.Fatalf("account list = %q, want %q", string(raw), want)
	}
}

func TestBuildAccountListOutPathUsesImportDirAndTimestamp(t *testing.T) {
	t.Parallel()

	got := buildAccountListOutPath("/tmp/imports", "", func() string {
		return "20260526_120000"
	})

	want := filepath.Join("/tmp/imports", "qq_cache_import_accounts_20260526_120000.txt")
	if got != want {
		t.Fatalf("buildAccountListOutPath() = %q, want %q", got, want)
	}
}
