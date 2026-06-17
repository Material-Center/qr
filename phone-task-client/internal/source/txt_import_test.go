package source

import "testing"

func TestParseTXTImportPhonesOnlyCleansBOMSkipsBlankLinesAndDedupes(t *testing.T) {
	raw := "\ufeff18507561351\n\n18507561352\n18507561351\n"

	got, err := ParseTXTImport(raw, false)
	if err != nil {
		t.Fatalf("parse txt import: %v", err)
	}
	if got.CodeAPI != "" {
		t.Fatalf("code api = %q", got.CodeAPI)
	}
	if len(got.Phones) != 2 {
		t.Fatalf("phones = %#v", got.Phones)
	}
	if got.Phones[0].Phone != "18507561351" || got.Phones[0].LineNo != 1 {
		t.Fatalf("first phone = %#v", got.Phones[0])
	}
	if got.Phones[1].Phone != "18507561352" || got.Phones[1].LineNo != 3 {
		t.Fatalf("second phone = %#v", got.Phones[1])
	}
}

func TestParseTXTImportReceiveCodeReadsFirstLineCodeAPI(t *testing.T) {
	raw := "\ufeffhttps://example.com/code?phone={phone}\n13238381229\n18507561351\n"

	got, err := ParseTXTImport(raw, true)
	if err != nil {
		t.Fatalf("parse txt import: %v", err)
	}
	if got.CodeAPI != "https://example.com/code?phone={phone}" {
		t.Fatalf("code api = %q", got.CodeAPI)
	}
	if len(got.Phones) != 2 {
		t.Fatalf("phones = %#v", got.Phones)
	}
	if got.Phones[0].Phone != "13238381229" || got.Phones[0].LineNo != 2 {
		t.Fatalf("first phone = %#v", got.Phones[0])
	}
}

func TestParseTXTImportRejectsInvalidPhoneWithLineNumber(t *testing.T) {
	_, err := ParseTXTImport("18507561351\nbad-phone\n", false)
	if err == nil {
		t.Fatal("expected invalid phone error")
	}
	if err.Error() != "第2行手机号格式不正确: bad-phone" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestParseTXTImportRequiresCodeAPIWhenEnabled(t *testing.T) {
	_, err := ParseTXTImport("\n\n", true)
	if err == nil {
		t.Fatal("expected missing code api error")
	}
	if err.Error() != "导入文件第一行验证码API不能为空" {
		t.Fatalf("error = %q", err.Error())
	}
}
