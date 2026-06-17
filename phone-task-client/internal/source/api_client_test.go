package source

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"phone-task-client/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newTestAPIClient(fn roundTripFunc) *APIClient {
	return &APIClient{http: &http.Client{Transport: fn}}
}

func TestAPIClientFetchPhone(t *testing.T) {
	client := newTestAPIClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("t") != "abc" {
			t.Fatalf("query t = %q", r.URL.Query().Get("t"))
		}
		return textResponse(`{"code":0,"data":"18507561351"}`), nil
	})

	phone, err := client.FetchPhone(t.Context(), domain.APITemplate{
		Method: domain.HTTPMethodGET,
		URL:    "http://api.test/phone",
		Query:  map[string]string{"t": "abc"},
	}, time.Unix(100, 0))
	if err != nil {
		t.Fatalf("fetch phone: %v", err)
	}
	if phone != "18507561351" {
		t.Fatalf("phone = %q", phone)
	}
}

func TestAPIClientFetchCode(t *testing.T) {
	client := newTestAPIClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("phone") != "13238381229" {
			t.Fatalf("phone query = %q", r.URL.Query().Get("phone"))
		}
		return textResponse("验证码220220"), nil
	})

	code, detail, err := client.FetchCode(t.Context(), domain.APITemplate{
		Method: domain.HTTPMethodGET,
		URL:    "http://api.test/code",
		Query:  map[string]string{"phone": "{phone}"},
	}, "13238381229", time.Unix(100, 0))
	if err != nil {
		t.Fatalf("fetch code: %v", err)
	}
	if code != "220220" || detail.Source != "text" {
		t.Fatalf("code=%q detail=%#v", code, detail)
	}
}

func TestAPIClientFetchCodeNotReady(t *testing.T) {
	client := newTestAPIClient(func(r *http.Request) (*http.Response, error) {
		return textResponse("暂无短信"), nil
	})

	_, _, err := client.FetchCode(t.Context(), domain.APITemplate{
		Method: domain.HTTPMethodGET,
		URL:    "http://api.test/code",
	}, "13238381229", time.Unix(100, 0))
	if !errors.Is(err, ErrCodeNotReady) {
		t.Fatalf("err = %v", err)
	}
}

func textResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}
