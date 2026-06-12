package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestLoadImportFileParsesCodeAPIAndPhones(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "phones.txt")
	if err := os.WriteFile(path, []byte("https://example.test/code?phone=\n18507561351\n\n18507561314\n"), 0o644); err != nil {
		t.Fatalf("write import file: %v", err)
	}

	got, err := LoadImportFile(path)
	if err != nil {
		t.Fatalf("load import file: %v", err)
	}
	if got.CodeAPI != "https://example.test/code?phone=" {
		t.Fatalf("code api = %q", got.CodeAPI)
	}
	if len(got.Phones) != 2 || got.Phones[0] != "18507561351" || got.Phones[1] != "18507561314" {
		t.Fatalf("phones = %#v", got.Phones)
	}
}

func TestRunOnceCreatesReceiveTaskAndSubmitsFetchedCode(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	var createdBody map[string]any
	var submittedBody map[string]any
	var codeRequestedPhone string

	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		codeRequestedPhone = r.URL.Query().Get("phone")
		_, _ = w.Write([]byte(`{"code":0,"data":"123456"}`))
	}))
	defer codeServer.Close()

	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Open-Api-Token") != "openapi-token" {
			t.Fatalf("missing token header: %q", r.Header.Get("X-Open-Api-Token"))
		}
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			if err := json.NewDecoder(r.Body).Decode(&createdBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			})
		case "/phoneRegisterTask/open-api/promoter/submit-code":
			if err := json.NewDecoder(r.Body).Decode(&submittedBody); err != nil {
				t.Fatalf("decode submit body: %v", err)
			}
			writeAPIResponse(t, w, 0, map[string]any{
				"id":     float64(9),
				"phone":  "18507561351",
				"status": "waiting_promoter_code",
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()

	state := NewState("/tmp/phones.txt", codeServer.URL+"?phone=", []string{"18507561351"})
	worker := NewWorker(workerConfig{
		System:        NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:    NewCodeSourceClient(state.CodeAPI, time.Second),
		State:         state,
		StatePath:     statePath,
		IdleThreshold: 1,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if createdBody["phone"] != "18507561351" {
		t.Fatalf("created phone = %v", createdBody["phone"])
	}
	if createdBody["smsReceiveMode"] != smsReceiveModePlatformSend {
		t.Fatalf("created smsReceiveMode = %v", createdBody["smsReceiveMode"])
	}
	if codeRequestedPhone != "18507561351" {
		t.Fatalf("code requested phone = %q", codeRequestedPhone)
	}
	if submittedBody["taskId"] != float64(9) || submittedBody["verifyCode"] != "123456" {
		t.Fatalf("submitted body = %#v", submittedBody)
	}

	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	rec := saved.Records[0]
	if rec.TaskID != 9 || rec.Status != recordStatusCodeSubmitted || rec.VerifyCode != "123456" {
		t.Fatalf("saved record = %#v", rec)
	}
}

func TestRunOnceResumesExistingTaskWithoutCreatingDuplicate(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var createCalled bool
	var submitCalled bool
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("验证码：654321"))
	}))
	defer codeServer.Close()

	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			createCalled = true
		case "/phoneRegisterTask/open-api/promoter/submit-code":
			submitCalled = true
			writeAPIResponse(t, w, 0, map[string]any{"id": float64(9), "status": "waiting_promoter_code"})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()

	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:     NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource: NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:      loaded,
		StatePath:  statePath,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if createCalled {
		t.Fatal("should not create a duplicate task while resuming")
	}
	if !submitCalled {
		t.Fatal("submit-code should be called for resumed waiting task")
	}
}

func TestRunOnceDoesNotSubmitCodeAgainAfterCodeSubmitted(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCodeSubmitted
	state.Records[0].VerifyCode = "123456"
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var codeCalled bool
	var submitCalled bool
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		codeCalled = true
		_, _ = w.Write([]byte("654321"))
	}))
	defer codeServer.Close()

	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			})
		case "/phoneRegisterTask/open-api/promoter/submit-code":
			submitCalled = true
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()

	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:     NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource: NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:      loaded,
		StatePath:  statePath,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if codeCalled {
		t.Fatal("code api should not be called after code was already submitted")
	}
	if submitCalled {
		t.Fatal("submit-code should not be called twice")
	}
}

func TestFetchCodeLogsRawTextResponseAndParseSource(t *testing.T) {
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("验证码：654321"))
	}))
	defer codeServer.Close()

	var logs bytes.Buffer
	client := NewCodeSourceClient(codeServer.URL+"?phone=", time.Second)
	client.logger = log.New(&logs, "", 0)

	code, err := client.FetchCode(t.Context(), "18507561351")
	if err != nil {
		t.Fatalf("fetch code: %v", err)
	}
	if code != "654321" {
		t.Fatalf("code = %q", code)
	}
	out := logs.String()
	for _, want := range []string{
		"code api request phone=18507561351",
		"status=200",
		"contentType=\"text/plain; charset=utf-8\"",
		"rawBody=\"验证码：654321\"",
		"source=text",
		"code=654321",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs missing %q in:\n%s", want, out)
		}
	}
}
