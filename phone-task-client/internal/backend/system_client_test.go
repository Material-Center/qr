package backend

import (
	"encoding/json"
	"io"
	"net/http"
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
	_ = json.NewEncoder(w).Encode(apiResponse{Code: code, Data: raw})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newTestSystemClient(fn roundTripFunc) *SystemClient {
	client := NewSystemClient("http://system.test", "token-1", time.Second)
	client.http = &http.Client{Transport: fn}
	return client
}

func apiHTTPResponse(t *testing.T, code int, data any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal response data: %v", err)
	}
	wrapper, err := json.Marshal(apiResponse{Code: code, Data: raw})
	if err != nil {
		t.Fatalf("marshal wrapper: %v", err)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(wrapper))), Header: http.Header{}}
}

func TestSystemClientIdleDeviceCountUsesOpenAPIToken(t *testing.T) {
	client := newTestSystemClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/phoneRegisterTask/open-api/promoter/device-stats" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("X-Open-Api-Token") != "token-1" {
			t.Fatalf("token header = %q", r.Header.Get("X-Open-Api-Token"))
		}
		return apiHTTPResponse(t, 0, map[string]any{"deviceIdleCount": float64(8)}), nil
	})

	got, err := client.IdleDeviceCount(t.Context())
	if err != nil {
		t.Fatalf("idle device count: %v", err)
	}
	if got != 8 {
		t.Fatalf("idle = %d", got)
	}
}

func TestSystemClientCreateSendCodeTask(t *testing.T) {
	var body map[string]any
	client := newTestSystemClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/phoneRegisterTask/open-api/promoter/task" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return apiHTTPResponse(t, 0, map[string]any{"id": float64(11), "phone": "18507561351", "status": "created"}), nil
	})

	task, err := client.CreateSendCodeTask(t.Context(), " 18507561351 ", 1500*time.Millisecond)
	if err != nil {
		t.Fatalf("create send task: %v", err)
	}
	if task.ID != 11 {
		t.Fatalf("task id = %d", task.ID)
	}
	if body["phone"] != "18507561351" {
		t.Fatalf("phone = %#v", body["phone"])
	}
	if body["smsReceiveMode"] != SMSReceiveModeUserSent {
		t.Fatalf("smsReceiveMode = %#v", body["smsReceiveMode"])
	}
	if body["startDelaySeconds"] != float64(2) {
		t.Fatalf("startDelaySeconds = %#v", body["startDelaySeconds"])
	}
	if body["reserveDevice"] != true {
		t.Fatalf("reserveDevice = %#v", body["reserveDevice"])
	}
}

func TestSystemClientCreateReceiveCodeTask(t *testing.T) {
	var body map[string]any
	client := newTestSystemClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/phoneRegisterTask/open-api/promoter/receive-task" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return apiHTTPResponse(t, 0, map[string]any{
			"id":               float64(12),
			"phone":            "13238381229",
			"status":           "waiting_promoter_code",
			"needPromoterCode": true,
		}), nil
	})

	task, err := client.CreateReceiveCodeTask(t.Context(), "13238381229", 0)
	if err != nil {
		t.Fatalf("create receive task: %v", err)
	}
	if task.ID != 12 || !task.NeedPromoterCode {
		t.Fatalf("task = %#v", task)
	}
	if body["smsReceiveMode"] != SMSReceiveModePlatformSend {
		t.Fatalf("smsReceiveMode = %#v", body["smsReceiveMode"])
	}
	if _, ok := body["startDelaySeconds"]; ok {
		t.Fatalf("unexpected startDelaySeconds in body %#v", body)
	}
}

func TestSystemClientGetTaskAndSubmitCode(t *testing.T) {
	var submitBody map[string]any
	client := newTestSystemClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/phoneRegisterTask/open-api/promoter/task/12":
			return apiHTTPResponse(t, 0, map[string]any{
				"id":               float64(12),
				"phone":            "13238381229",
				"status":           "waiting_promoter_code",
				"needPromoterCode": true,
			}), nil
		case "/phoneRegisterTask/open-api/promoter/submit-code":
			if err := json.NewDecoder(r.Body).Decode(&submitBody); err != nil {
				t.Fatalf("decode submit body: %v", err)
			}
			return apiHTTPResponse(t, 0, map[string]any{
				"id":     float64(12),
				"phone":  "13238381229",
				"status": "running",
			}), nil
		default:
			t.Fatalf("unexpected path = %s", r.URL.Path)
		}
		return nil, nil
	})

	task, err := client.GetTask(t.Context(), 12)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.ID != 12 || task.Status != "waiting_promoter_code" {
		t.Fatalf("task = %#v", task)
	}

	task, err = client.SubmitCode(t.Context(), 12, " 220220 ")
	if err != nil {
		t.Fatalf("submit code: %v", err)
	}
	if task.ID != 12 {
		t.Fatalf("submitted task = %#v", task)
	}
	if submitBody["taskId"] != float64(12) || submitBody["verifyCode"] != "220220" {
		t.Fatalf("submit body = %#v", submitBody)
	}
}
