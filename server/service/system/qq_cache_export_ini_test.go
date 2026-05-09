package system

import (
	"strings"
	"testing"
	"time"

	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

func TestNormalizeQQCacheExportINI(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"qqnum=3896349451",
		"guid=2D2F4073897A69E82FA8124BB4293162",
		"_device_token=A2D20B043EC29240AAAE8D40F283DF3A",
		"_superKey=62512D7967633475396F62685A6432336D74642D4167754D3646676F75795045773363374D576D766875675F",
		"deviceInfo={\"brand\":\"Redmi\",\"model\":\"22041211AC\",\"androidVersion\":\"13\",\"apiLevel\":33,\"serialNumber\":\"HETSWSZHK7F6L7YL\",\"androidId\":\"a28f3561bfa79033\"}",
		"",
	}, "\n")

	got := normalizeQQCacheExportINI(raw)

	if strings.Contains(got, "_device_token=") {
		t.Fatalf("expected _device_token removed, got: %q", got)
	}
	if strings.Contains(got, "_superKey=") {
		t.Fatalf("expected _superKey removed, got: %q", got)
	}
	if !strings.Contains(got, "qqnum=3896349451\r\n") {
		t.Fatalf("expected qqnum preserved, got: %q", got)
	}
	if !strings.Contains(got, "guid=2D2F4073897A69E82FA8124BB4293162\r\n") {
		t.Fatalf("expected guid preserved, got: %q", got)
	}
	if !strings.Contains(got, "\"model\":\"XiaoMi 17\"") {
		t.Fatalf("expected deviceInfo model normalized, got: %q", got)
	}
	if strings.Contains(got, "\"model\":\"22041211AC\"") {
		t.Fatalf("expected old model removed, got: %q", got)
	}
}

func TestNormalizeQQCacheExportINIKeepDeviceInfoWhenJSONInvalid(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"deviceInfo={bad json}",
		"",
	}, "\n")

	got := normalizeQQCacheExportINI(raw)

	if !strings.Contains(got, "deviceInfo={bad json}\r\n") {
		t.Fatalf("expected invalid json line kept as-is, got: %q", got)
	}
}

func TestExtractQQCacheGUID(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"qqnum=3896349451",
		"guid = \"2D2F4073897A69E82FA8124BB4293162\"",
	}, "\n")

	got := extractQQCacheGUID(raw)

	if got != "2D2F4073897A69E82FA8124BB4293162" {
		t.Fatalf("expected guid extracted, got: %q", got)
	}
}

func TestBuildQQCacheAccountLine(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 1, 39, 49, 0, time.Local)
	rec := model.SysQQCacheRecord{
		QQNum: "3896349451",
		QQPwd: "abc123",
	}
	rec.CreatedAt = createdAt

	got := buildQQCacheAccountLine(rec, "guid=2D2F4073897A69E82FA8124BB4293162")
	want := "3896349451----abc123----2D2F4073897A69E82FA8124BB4293162----2026-05-10 01:39:49"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
