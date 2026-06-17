package template

import (
	"testing"
	"time"

	"phone-task-client/internal/domain"
)

func TestRenderGETURLUsesPhoneQueryVariable(t *testing.T) {
	got, err := RenderGETURL(domain.APITemplate{
		Method: domain.HTTPMethodGET,
		URL:    "https://example.com/code",
		Query:  map[string]string{"phone": "{phone}"},
	}, RenderInput{Phone: "13238381229"})
	if err != nil {
		t.Fatalf("render get url: %v", err)
	}
	if got != "https://example.com/code?phone=13238381229" {
		t.Fatalf("url = %q", got)
	}
}

func TestRenderGETURLReplacesPhoneInURL(t *testing.T) {
	got, err := RenderGETURL(domain.APITemplate{
		Method: domain.HTTPMethodGET,
		URL:    "https://example.com/code?phone={phone}",
	}, RenderInput{Phone: "13238381229"})
	if err != nil {
		t.Fatalf("render get url: %v", err)
	}
	if got != "https://example.com/code?phone=13238381229" {
		t.Fatalf("url = %q", got)
	}
}

func TestRenderGETURLMergesURLAndTemplateQuery(t *testing.T) {
	got, err := RenderGETURL(domain.APITemplate{
		Method: domain.HTTPMethodGET,
		URL:    "https://example.com/code?t=abc",
		Query:  map[string]string{"phone": "{phone}"},
	}, RenderInput{Phone: "13238381229"})
	if err != nil {
		t.Fatalf("render get url: %v", err)
	}
	if got != "https://example.com/code?phone=13238381229&t=abc" {
		t.Fatalf("url = %q", got)
	}
}

func TestRenderGETURLUsesTimestampVariable(t *testing.T) {
	now := time.UnixMilli(1781282012000)
	got, err := RenderGETURL(domain.APITemplate{
		Method: domain.HTTPMethodGET,
		URL:    "https://example.com/code",
		Query:  map[string]string{"ts": "{timestamp}"},
	}, RenderInput{Phone: "13238381229", Now: now})
	if err != nil {
		t.Fatalf("render get url: %v", err)
	}
	if got != "https://example.com/code?ts=1781282012000" {
		t.Fatalf("url = %q", got)
	}
}

func TestRenderGETURLRejectsNonGETTemplate(t *testing.T) {
	_, err := RenderGETURL(domain.APITemplate{
		Method: domain.HTTPMethodPOST,
		URL:    "https://example.com/code",
	}, RenderInput{Phone: "13238381229"})
	if err == nil {
		t.Fatal("expected non-get error")
	}
	if err.Error() != "api template method must be GET" {
		t.Fatalf("error = %q", err.Error())
	}
}
