package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestRunOnceFetchesPhoneAndCreatesUserSentTaskWhenIdleDeviceCountIsAboveThreshold(t *testing.T) {
	var createdBody map[string]string
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

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(system.URL, "openapi-token", time.Second),
		PhoneSource:   NewPhoneSourceClient(source.URL, time.Second),
		IdleThreshold: 1,
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
}

func TestRunOnceWaitsAfterFetchingPhoneBeforeCreatingTask(t *testing.T) {
	sourceFetchedAt := make(chan time.Time, 1)
	taskCreatedAt := make(chan time.Time, 1)
	const createDelay = 50 * time.Millisecond

	system := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
		case "/phoneRegisterTask/open-api/promoter/task":
			taskCreatedAt <- time.Now()
			writeAPIResponse(t, w, 0, map[string]any{"id": float64(9)})
		case "/base/login":
			t.Fatal("login should not be called")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer system.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceFetchedAt <- time.Now()
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
	if err := worker.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}
	fetchedAt := <-sourceFetchedAt
	createdAt := <-taskCreatedAt
	if elapsed := createdAt.Sub(fetchedAt); elapsed < createDelay {
		t.Fatalf("create elapsed = %s, want at least %s", elapsed, createDelay)
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
