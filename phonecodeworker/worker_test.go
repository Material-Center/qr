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
		CreateDelay:   2 * time.Second,
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
	if createdBody["startDelaySeconds"] != float64(2) {
		t.Fatalf("created startDelaySeconds = %v", createdBody["startDelaySeconds"])
	}
	if createdBody["reserveDevice"] != true {
		t.Fatalf("created reserveDevice = %v", createdBody["reserveDevice"])
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
	if rec.TaskID != 9 || rec.Status != recordStatusSucceeded || rec.VerifyCode != "123456" {
		t.Fatalf("saved record = %#v", rec)
	}
}

func TestRunOnceCreatesMultipleReceiveTasksUpToAvailableWindow(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	phones := []string{"18507561351", "18507561352", "18507561353"}
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", phones)

	createdPhones := []string{}
	nextID := uint(100)
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(3)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			phone := body["phone"].(string)
			createdPhones = append(createdPhones, phone)
			nextID++
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(nextID),
				"phone":            phone,
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called before promoter code is needed")
	}))
	defer codeServer.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:    NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:         state,
		StatePath:     statePath,
		IdleThreshold: 1,
		Interval:      time.Millisecond,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if got, want := strings.Join(createdPhones, ","), "18507561351,18507561352"; got != want {
		t.Fatalf("created phones = %s, want %s", got, want)
	}
	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	if saved.Records[0].TaskID == 0 || saved.Records[1].TaskID == 0 {
		t.Fatalf("first two records should be created: %#v", saved.Records)
	}
	if saved.Records[2].TaskID != 0 || saved.Records[2].Status != recordStatusPending {
		t.Fatalf("third record should remain pending: %#v", saved.Records[2])
	}
}

func TestRunOnceRechecksIdleCapacityAfterCreateInterval(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351", "18507561352"})

	statsCalls := 0
	createdPhones := []string{}
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			statsCalls++
			if statsCalls == 1 {
				writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
				return
			}
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(0)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			phone := body["phone"].(string)
			createdPhones = append(createdPhones, phone)
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(100 + len(createdPhones)),
				"phone":            phone,
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called before promoter code is needed")
	}))
	defer codeServer.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:    NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:         state,
		StatePath:     statePath,
		IdleThreshold: 0,
		Interval:      time.Millisecond,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if got, want := strings.Join(createdPhones, ","), "18507561351"; got != want {
		t.Fatalf("created phones = %s, want %s", got, want)
	}
	if statsCalls != 2 {
		t.Fatalf("device stats calls = %d, want 2", statsCalls)
	}
}

func TestRunOnceRetriesCodeNotReadyRecordsAfterCreateBatch(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351", "18507561352"})

	codeCalls := map[string]int{}
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		phone := r.URL.Query().Get("phone")
		codeCalls[phone]++
		if phone == "18507561351" && codeCalls[phone] == 1 {
			_, _ = w.Write([]byte("暂未收到验证码"))
			return
		}
		_, _ = w.Write([]byte("验证码：654321"))
	}))
	defer codeServer.Close()

	submitted := map[uint]string{}
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(2)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			phone := body["phone"].(string)
			id := float64(101)
			if phone == "18507561352" {
				id = 102
			}
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               id,
				"phone":            phone,
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			})
		case "/phoneRegisterTask/open-api/promoter/task/101":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(101),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			})
		case "/phoneRegisterTask/open-api/promoter/task/102":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(102),
				"phone":            "18507561352",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			})
		case "/phoneRegisterTask/open-api/promoter/submit-code":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode submit body: %v", err)
			}
			submitted[uint(body["taskId"].(float64))] = body["verifyCode"].(string)
			writeAPIResponse(t, w, 0, map[string]any{"id": body["taskId"], "status": "waiting_promoter_code"})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()

	worker := NewWorker(workerConfig{
		System:        NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:    NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:         state,
		StatePath:     statePath,
		IdleThreshold: 0,
		Interval:      time.Millisecond,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if codeCalls["18507561351"] != 2 {
		t.Fatalf("first phone code calls = %d, want 2", codeCalls["18507561351"])
	}
	if submitted[101] != "654321" || submitted[102] != "654321" {
		t.Fatalf("submitted = %#v", submitted)
	}
	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	if saved.Records[0].Status != recordStatusSucceeded || saved.Records[0].LastError != "" {
		t.Fatalf("first record = %#v", saved.Records[0])
	}
}

func TestRunOnceRecreatesTaskAfterServerTimeout(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	state.Records[0].TaskAttempts = 1
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	createdPhones := []string{}
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			code := phoneRegisterStatusCodeTaskTimeout
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "failed",
				"statusCode":       float64(code),
				"lastError":        "任务总超时",
				"needPromoterCode": false,
			})
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(1)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			phone := body["phone"].(string)
			createdPhones = append(createdPhones, phone)
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(10),
				"phone":            phone,
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called before promoter code is needed")
	}))
	defer codeServer.Close()

	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:         NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:     NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:          loaded,
		StatePath:      statePath,
		IdleThreshold:  0,
		MaxTaskRetries: 1,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if got, want := strings.Join(createdPhones, ","), "18507561351"; got != want {
		t.Fatalf("created phones = %s, want %s", got, want)
	}
	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	rec := saved.Records[0]
	if rec.TaskID != 10 || rec.Status != recordStatusCreated || rec.TaskAttempts != 2 {
		t.Fatalf("saved record = %#v", rec)
	}
}

func TestRunOnceMarksTimedOutTaskFailedAfterRetryLimit(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351"})
	state.Records[0].TaskID = 10
	state.Records[0].Status = recordStatusCreated
	state.Records[0].TaskAttempts = 2
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var createCalled bool
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/10":
			code := phoneRegisterStatusCodeTaskTimeout
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(10),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "failed",
				"statusCode":       float64(code),
				"lastError":        "任务总超时",
				"needPromoterCode": false,
			})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			createCalled = true
			t.Fatal("should not create after timeout retry limit")
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called before promoter code is needed")
	}))
	defer codeServer.Close()

	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:         NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:     NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:          loaded,
		StatePath:      statePath,
		MaxTaskRetries: 1,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if createCalled {
		t.Fatal("receive-task should not be called")
	}
	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	rec := saved.Records[0]
	if rec.Status != recordStatusFailed || rec.TaskID != 10 || rec.TaskAttempts != 2 {
		t.Fatalf("saved record = %#v", rec)
	}
}

func TestRunOnceRequeuesAfterSubmitCodeTimeout(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351"})

	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("验证码：654321"))
	}))
	defer codeServer.Close()

	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(1)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(11),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			})
		case "/phoneRegisterTask/open-api/promoter/submit-code":
			_ = json.NewEncoder(w).Encode(apiResponse{Code: 7, Msg: "验证码已超时"})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()

	worker := NewWorker(workerConfig{
		System:         NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:     NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:          state,
		StatePath:      statePath,
		IdleThreshold:  0,
		MaxTaskRetries: 1,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	rec := saved.Records[0]
	if rec.Status != recordStatusPending || rec.TaskID != 0 || rec.TaskAttempts != 1 {
		t.Fatalf("saved record = %#v", rec)
	}
	if rec.LastError != "验证码已超时" {
		t.Fatalf("last error = %q", rec.LastError)
	}
}

func TestRunOnceWritesFailedImportFile(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "phones.txt")
	statePath := filepath.Join(dir, "state.json")
	failedPath := filepath.Join(dir, "phones.failed.txt")
	codeAPI := "https://code.test/?phone="
	state := NewState(inputPath, codeAPI, []string{"18507561351", "18507561352"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	state.Records[1].Status = recordStatusSucceeded
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "failed",
				"lastError":        "设备执行失败",
				"needPromoterCode": false,
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called for failed task")
	}))
	defer codeServer.Close()

	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:     NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource: NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:      loaded,
		StatePath:  statePath,
		FailedPath: failedPath,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	raw, err := os.ReadFile(failedPath)
	if err != nil {
		t.Fatalf("read failed import file: %v", err)
	}
	if got, want := string(raw), codeAPI+"\n18507561351\n"; got != want {
		t.Fatalf("failed import file = %q, want %q", got, want)
	}
}

func TestRunOnceSyncsAllActiveTasks(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351", "18507561352"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	state.Records[1].TaskID = 10
	state.Records[1].Status = recordStatusCreated
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	synced := map[string]bool{}
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			synced["9"] = true
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		case "/phoneRegisterTask/open-api/promoter/task/10":
			synced["10"] = true
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(10),
				"phone":            "18507561352",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called before promoter code is needed")
	}))
	defer codeServer.Close()

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
	if !synced["9"] || !synced["10"] {
		t.Fatalf("synced tasks = %#v", synced)
	}
}

func TestRunOnceCreatesForIdleDevicesEvenWhenTasksAreActive(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351", "18507561352", "18507561353"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	createdPhones := []string{}
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(3)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			phone := body["phone"].(string)
			createdPhones = append(createdPhones, phone)
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(10 + len(createdPhones)),
				"phone":            phone,
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called before promoter code is needed")
	}))
	defer codeServer.Close()

	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:        NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:    NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:         loaded,
		StatePath:     statePath,
		IdleThreshold: 1,
		Interval:      time.Millisecond,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if got, want := strings.Join(createdPhones, ","), "18507561352,18507561353"; got != want {
		t.Fatalf("created phones = %s, want %s", got, want)
	}
}

func TestRunOnceReplenishesOneIdleSlotAfterActiveTaskSucceeds(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351", "18507561352", "18507561353"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	state.Records[1].TaskID = 10
	state.Records[1].Status = recordStatusCreated
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	createdPhones := []string{}
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "succeeded",
				"needPromoterCode": false,
			})
		case "/phoneRegisterTask/open-api/promoter/task/10":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(10),
				"phone":            "18507561352",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(1)})
		case "/phoneRegisterTask/open-api/promoter/receive-task":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			phone := body["phone"].(string)
			createdPhones = append(createdPhones, phone)
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(11),
				"phone":            phone,
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
			})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("code api should not be called before promoter code is needed")
	}))
	defer codeServer.Close()

	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:        NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource:    NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:         loaded,
		StatePath:     statePath,
		IdleThreshold: 0,
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if got, want := strings.Join(createdPhones, ","), "18507561353"; got != want {
		t.Fatalf("created phones = %s, want %s", got, want)
	}
	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	if saved.Records[0].Status != recordStatusSucceeded {
		t.Fatalf("finished active record = %#v", saved.Records[0])
	}
	if saved.Records[2].TaskID == 0 {
		t.Fatalf("pending record should be replenished: %#v", saved.Records[2])
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
		if r.URL.Path == "/phoneRegisterTask/open-api/promoter/submit-code" {
			submitCalled = true
		}
		t.Fatalf("code_submitted record should not call system API, got path: %s", r.URL.Path)
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

func TestRunOnceMarksMissingActiveTaskFailed(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351", "18507561314"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			_ = json.NewEncoder(w).Encode(apiResponse{Code: 7, Msg: "任务不存在"})
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(0)})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("654321"))
	}))
	defer codeServer.Close()

	var logs bytes.Buffer
	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:     NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource: NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:      loaded,
		StatePath:  statePath,
		Logger:     log.New(&logs, "", 0),
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	saved, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	if saved.Records[0].Status != recordStatusFailed || saved.Records[0].LastError != "任务不存在" {
		t.Fatalf("first record = %#v", saved.Records[0])
	}
	if saved.activeRecord() != nil {
		t.Fatalf("missing task should no longer block active record: %#v", saved.activeRecord())
	}
	if !strings.Contains(logs.String(), "active task missing task=9 phone=18507561351") {
		t.Fatalf("logs missing active task missing entry:\n%s", logs.String())
	}
}

func TestRunOnceLogsActiveTaskWaitingForPromoterCode(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state := NewState("/tmp/phones.txt", "https://code.test/?phone=", []string{"18507561351", "18507561314"})
	state.Records[0].TaskID = 9
	state.Records[0].Status = recordStatusCreated
	if err := SaveStateFile(statePath, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var codeCalled bool
	systemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/9":
			writeAPIResponse(t, w, 0, map[string]any{
				"id":               float64(9),
				"phone":            "18507561351",
				"smsReceiveMode":   smsReceiveModePlatformSend,
				"status":           "running",
				"needPromoterCode": false,
				"lastError":        "等待设备进入验证码阶段",
			})
		case "/phoneRegisterTask/open-api/promoter/device-stats":
			writeAPIResponse(t, w, 0, map[string]any{"deviceIdleCount": float64(0)})
		default:
			t.Fatalf("unexpected system path: %s", r.URL.Path)
		}
	}))
	defer systemServer.Close()
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		codeCalled = true
		_, _ = w.Write([]byte("654321"))
	}))
	defer codeServer.Close()

	var logs bytes.Buffer
	loaded, err := LoadStateFile(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	worker := NewWorker(workerConfig{
		System:     NewSystemClient(systemServer.URL, "openapi-token", time.Second),
		CodeSource: NewCodeSourceClient(codeServer.URL+"?phone=", time.Second),
		State:      loaded,
		StatePath:  statePath,
		Logger:     log.New(&logs, "", 0),
	})

	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if codeCalled {
		t.Fatal("code api should not be called before promoter code is needed")
	}
	out := logs.String()
	for _, want := range []string{
		"cycle start records=2",
		"active task phone=18507561351 task=9 localStatus=created",
		"task status task=9 phone=18507561351 remoteStatus=running needPromoterCode=false",
		"waiting promoter code task=9 phone=18507561351 remoteStatus=running lastError=\"等待设备进入验证码阶段\"",
		"state saved path=" + statePath,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs missing %q in:\n%s", want, out)
		}
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
