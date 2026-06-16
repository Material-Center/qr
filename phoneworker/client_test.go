package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func writeAPIResponse(t *testing.T, w http.ResponseWriter, code int, data any) {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal response data: %v", err)
	}
	_ = json.NewEncoder(w).Encode(apiResponse{
		Code: code,
		Data: raw,
	})
}

func TestSystemClientUsesOpenAPITokenForDeviceStats(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Header.Get("X-Open-Api-Token") != "openapi-token" {
			t.Fatalf("missing openapi token header: %q", r.Header.Get("X-Open-Api-Token"))
		}
		writeAPIResponse(t, w, 0, map[string]any{
			"deviceIdleCount": float64(3),
		})
	}))
	defer ts.Close()

	c := NewSystemClient(ts.URL, "openapi-token", time.Second)
	idle, err := c.IdleDeviceCount(t.Context())
	if err != nil {
		t.Fatalf("idle device count: %v", err)
	}
	if gotPath != "/phoneRegisterTask/open-api/promoter/device-stats" {
		t.Fatalf("path = %q, want device-stats path", gotPath)
	}
	if idle != 3 {
		t.Fatalf("idle = %d, want 3", idle)
	}
}

func TestRunOnceSkipsSourceWhenIdleDeviceCountIsNotAboveThreshold(t *testing.T) {
	sourceCalled := false
	createCalled := false
	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(1)})
		case "/phoneRegisterTask/open-api/promoter/task":
			createCalled = true
			w.WriteHeader(http.StatusInternalServerError)
		case "/base/login":
			t.Fatal("login should not be called")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceCalled = true
		_ = json.NewEncoder(w).Encode(phoneSourceResponse{Code: 0, Data: "18878309701"})
	}))
	defer source.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource:   NewPhoneSourceClient(source.URL, time.Second),
		IdleThreshold: 1,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if sourceCalled {
		t.Fatal("phone source should not be called")
	}
	if createCalled {
		t.Fatal("create task should not be called")
	}
}

func TestRunOnceSkipsSourceWhenPaused(t *testing.T) {
	dir := t.TempDir()
	pauseFile := filepath.Join(dir, "phoneworker.pause")
	if err := os.WriteFile(pauseFile, []byte("pause"), 0o644); err != nil {
		t.Fatalf("write pause file: %v", err)
	}

	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("system api should not be called while paused: %s", r.URL.Path)
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("phone source should not be called while paused")
	}))
	defer source.Close()

	worker := NewWorker(workerConfig{
		System:      NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource: NewPhoneSourceClient(source.URL, time.Second),
		PauseFile:   pauseFile,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
}

func TestRunOnceFetchesPhoneAndCreatesUserSentTaskWhenIdleDeviceCountIsAboveThreshold(t *testing.T) {
	var createdBody map[string]any
	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Open-Api-Token") != "openapi-token" {
			t.Fatalf("missing openapi token header: %q", r.Header.Get("X-Open-Api-Token"))
		}
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
		case "/phoneRegisterTask/open-api/promoter/task":
			if err := json.NewDecoder(r.Body).Decode(&createdBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			writeAPIResponse(t, w, 0, map[string]any{"id": float64(9)})
		case "/base/login":
			t.Fatal("login should not be called")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(phoneSourceResponse{Code: 0, Data: "18878309701"})
	}))
	defer source.Close()

	var logs bytes.Buffer
	worker := NewWorker(workerConfig{
		System:        NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource:   NewPhoneSourceClient(source.URL, time.Second),
		IdleThreshold: 1,
		Logger:        log.New(&logs, "", 0),
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if createdBody["phone"] != "18878309701" {
		t.Fatalf("phone = %q", createdBody["phone"])
	}
	if createdBody["smsReceiveMode"] != smsReceiveModeUserSent {
		t.Fatalf("smsReceiveMode = %q", createdBody["smsReceiveMode"])
	}
	if _, ok := createdBody["startDelaySeconds"]; ok {
		t.Fatal("startDelaySeconds should be omitted when create delay is zero")
	}
	if _, ok := createdBody["reserveDevice"]; ok {
		t.Fatal("reserveDevice should be omitted when create delay is zero")
	}
	out := logs.String()
	for _, want := range []string{
		"phone source request url=",
		"phone source response status=200",
		"phone source parsed phone=18878309701",
		"create task start phone=18878309701 mode=USER_SENT_TO_TX",
		"system api create-task request phone=18878309701",
		"system api create-task response phone=18878309701 status=200",
		"created task id=9 phone=18878309701 mode=USER_SENT_TO_TX",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs missing %q in:\n%s", want, out)
		}
	}
}

func TestRunOnceCreatesServerDelayedTaskImmediately(t *testing.T) {
	taskCreated := make(chan map[string]any, 1)
	const createDelay = 2 * time.Second

	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
		case "/phoneRegisterTask/open-api/promoter/task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			taskCreated <- body
			writeAPIResponse(t, w, 0, map[string]any{"id": float64(9)})
		case "/base/login":
			t.Fatal("login should not be called")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(phoneSourceResponse{Code: 0, Data: "18878309701"})
	}))
	defer source.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource:   NewPhoneSourceClient(source.URL, time.Second),
		IdleThreshold: 1,
		CreateDelay:   createDelay,
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	runOnceStartedAt := time.Now()
	if err := worker.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if elapsed := time.Since(runOnceStartedAt); elapsed >= 200*time.Millisecond {
		t.Fatalf("run once blocked for %s, want immediate server-delay create", elapsed)
	}
	var body map[string]any
	select {
	case body = <-taskCreated:
	case <-ctx.Done():
		t.Fatalf("wait create task: %v", ctx.Err())
	}
	if body["phone"] != "18878309701" {
		t.Fatalf("phone = %q", body["phone"])
	}
	if body["startDelaySeconds"] != float64(2) {
		t.Fatalf("startDelaySeconds = %v, want 2", body["startDelaySeconds"])
	}
	if body["reserveDevice"] != true {
		t.Fatalf("reserveDevice = %v, want true", body["reserveDevice"])
	}
}

func TestRunOnceFetchesOnePhonePerIntervalWithServerDelay(t *testing.T) {
	const createDelay = 2 * time.Second
	var sourceCalls int64
	createdPhones := make(chan string, 2)

	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(3)})
		case "/phoneRegisterTask/open-api/promoter/task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			createdPhones <- body["phone"].(string)
			writeAPIResponse(t, w, 0, map[string]any{"id": float64(9)})
		case "/base/login":
			t.Fatal("login should not be called")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next := atomic.AddInt64(&sourceCalls, 1)
		_ = json.NewEncoder(w).Encode(phoneSourceResponse{Code: 0, Data: "1887830970" + string(rune('0'+next))})
	}))
	defer source.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource:   NewPhoneSourceClient(source.URL, time.Second),
		IdleThreshold: 1,
		CreateDelay:   createDelay,
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := worker.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if got := atomic.LoadInt64(&sourceCalls); got != 1 {
		t.Fatalf("source calls after first run = %d, want 1", got)
	}
	if err := worker.RunOnce(ctx); err != nil {
		t.Fatalf("second run once: %v", err)
	}
	if got := atomic.LoadInt64(&sourceCalls); got != 2 {
		t.Fatalf("source calls after second interval = %d, want 2", got)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-createdPhones:
		case <-ctx.Done():
			t.Fatalf("wait created phone %d: %v", i+1, ctx.Err())
		}
	}
}

func TestRunOnceReturnsCreateFailureImmediatelyWithServerDelay(t *testing.T) {
	var sourceCalls int64
	var createCalls int64

	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
		case "/phoneRegisterTask/open-api/promoter/task":
			atomic.AddInt64(&createCalls, 1)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":7,"msg":"create failed"}`))
		case "/base/login":
			t.Fatal("login should not be called")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&sourceCalls, 1)
		_ = json.NewEncoder(w).Encode(phoneSourceResponse{Code: 0, Data: "18878309701"})
	}))
	defer source.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource:   NewPhoneSourceClient(source.URL, time.Second),
		IdleThreshold: 1,
		CreateDelay:   10 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := worker.RunOnce(ctx); err == nil {
		t.Fatal("run once should return create failure")
	}
	if got := atomic.LoadInt64(&createCalls); got != 1 {
		t.Fatalf("create calls = %d, want 1", got)
	}
	if got := atomic.LoadInt64(&sourceCalls); got != 1 {
		t.Fatalf("source calls = %d, want 1", got)
	}
}

func TestSystemClientCreateTaskLogsHTTPTimeout(t *testing.T) {
	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writeAPIResponse(t, w, 0, map[string]any{"id": float64(9)})
	}))
	defer system.Close()

	var logs bytes.Buffer
	client := NewSystemClient(system.URL, "openapi-token", 10*time.Millisecond)
	client.logger = log.New(&logs, "", 0)

	if _, err := client.CreateUserSentTask(t.Context(), "18878309701", 0); err == nil {
		t.Fatal("create task should time out")
	}
	out := logs.String()
	for _, want := range []string{
		"system api create-task request phone=18878309701",
		"system api create-task error phone=18878309701",
		"timeout=true",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs missing %q in:\n%s", want, out)
		}
	}
}

func waitUntil(t *testing.T, ctx context.Context, ok func() bool) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		if ok() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait condition: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func TestRunOnceDoesNotCreateTaskWhenPhoneSourceFails(t *testing.T) {
	createCalled := false
	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
		case "/phoneRegisterTask/open-api/promoter/task":
			createCalled = true
		case "/base/login":
			t.Fatal("login should not be called")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(phoneSourceResponse{Code: -1})
	}))
	defer source.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource:   NewPhoneSourceClient(source.URL, time.Second),
		IdleThreshold: 1,
	})

	if err := worker.RunOnce(t.Context()); err == nil {
		t.Fatal("run once should return phone source error")
	}
	if createCalled {
		t.Fatal("create task should not be called")
	}
}
