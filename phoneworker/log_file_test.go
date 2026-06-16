package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDailyLogWriterCreatesNewFileAndRotatesByDay(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 16, 1, 2, 3, 0, time.Local)
	writer, err := newDailyLogWriter(dir, "phoneworker", func() time.Time { return now })
	if err != nil {
		t.Fatalf("new daily log writer: %v", err)
	}
	defer writer.Close()

	firstPath := writer.Path()
	if !strings.Contains(filepath.Base(firstPath), "phoneworker-20260616-010203") {
		t.Fatalf("first log path = %s", firstPath)
	}
	if _, err := writer.Write([]byte("first\n")); err != nil {
		t.Fatalf("write first: %v", err)
	}

	secondWriter, err := newDailyLogWriter(dir, "phoneworker", func() time.Time { return now })
	if err != nil {
		t.Fatalf("new second daily log writer: %v", err)
	}
	defer secondWriter.Close()
	secondPath := secondWriter.Path()
	if secondPath == firstPath {
		t.Fatalf("new startup should create a new log file, got same path %s", firstPath)
	}

	now = time.Date(2026, 6, 17, 0, 0, 1, 0, time.Local)
	if _, err := writer.Write([]byte("second day\n")); err != nil {
		t.Fatalf("write second day: %v", err)
	}
	rotatedPath := writer.Path()
	if rotatedPath == firstPath {
		t.Fatalf("writer did not rotate by day")
	}
	if !strings.Contains(filepath.Base(rotatedPath), "phoneworker-20260617-000001") {
		t.Fatalf("rotated log path = %s", rotatedPath)
	}

	firstRaw, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first log: %v", err)
	}
	if string(firstRaw) != "first\n" {
		t.Fatalf("first log = %q", string(firstRaw))
	}
	rotatedRaw, err := os.ReadFile(rotatedPath)
	if err != nil {
		t.Fatalf("read rotated log: %v", err)
	}
	if string(rotatedRaw) != "second day\n" {
		t.Fatalf("rotated log = %q", string(rotatedRaw))
	}
}
