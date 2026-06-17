package source

import (
	"errors"
	"testing"
)

func TestExtractVerifyCodeFromTextResponse(t *testing.T) {
	code, detail := ExtractVerifyCodeDetail([]byte("【腾讯科技】验证码220220，20分钟内有效。"))

	if code != "220220" {
		t.Fatalf("code = %q", code)
	}
	if detail.JSONParsed {
		t.Fatalf("json parsed = true")
	}
	if detail.Source != "text" {
		t.Fatalf("source = %q", detail.Source)
	}
}

func TestExtractVerifyCodeFromJSONData(t *testing.T) {
	code, detail := ExtractVerifyCodeDetail([]byte(`{"code":0,"data":"验证码654321"}`))

	if code != "654321" {
		t.Fatalf("code = %q", code)
	}
	if !detail.JSONParsed {
		t.Fatalf("json parsed = false")
	}
	if detail.Source != "json.data" {
		t.Fatalf("source = %q", detail.Source)
	}
}

func TestExtractVerifyCodeNotReady(t *testing.T) {
	code, detail := ExtractVerifyCodeDetail([]byte(`{"code":0,"data":"暂无短信"}`))

	if code != "" {
		t.Fatalf("code = %q", code)
	}
	if !detail.JSONParsed || detail.Source != "json" {
		t.Fatalf("detail = %#v", detail)
	}
}

func TestExtractPhoneFromJSONData(t *testing.T) {
	phone, err := ExtractPhoneFromAPIResponse([]byte(`{"code":0,"data":"18507561351"}`))
	if err != nil {
		t.Fatalf("extract phone: %v", err)
	}
	if phone != "18507561351" {
		t.Fatalf("phone = %q", phone)
	}
}

func TestExtractPhoneFromTextResponse(t *testing.T) {
	phone, err := ExtractPhoneFromAPIResponse([]byte(`18507561351`))
	if err != nil {
		t.Fatalf("extract phone: %v", err)
	}
	if phone != "18507561351" {
		t.Fatalf("phone = %q", phone)
	}
}

func TestExtractPhoneReturnsNotReadyForNoPhone(t *testing.T) {
	_, err := ExtractPhoneFromAPIResponse([]byte(`{"code":0,"data":"暂无手机号"}`))
	if !errors.Is(err, ErrPhoneNotReady) {
		t.Fatalf("err = %v", err)
	}
}
