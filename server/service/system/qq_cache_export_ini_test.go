package system

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupQQCacheTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SysQQCacheRecord{}))
	global.GVA_DB = db
}

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

func TestExportIniZipByQQText(t *testing.T) {
	setupQQCacheTestDB(t)

	ini1 := "qqnum=3995613452\nguid=GUID001\n"
	ini2 := "qqnum=626384712\nguid=GUID002\n"
	ini3 := "qqnum=930634982\nguid=GUID003\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "3995613452", QQPwd: "pwd1", INI: &ini1}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "626384712", QQPwd: "pwd2", INI: &ini2}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "930634982", QQPwd: "pwd3", INI: &ini3}).Error)

	zipBytes, count, err := (&QQCacheService{}).ExportIniZipByQQText(strings.Join([]string{
		"3995613452----冻结",
		"930634982",
		"626384712----冻结",
		"3995613452----冻结",
	}, "\n"))
	require.NoError(t, err)
	require.EqualValues(t, 3, count)

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	require.True(t, names["3995613452.ini"])
	require.True(t, names["930634982.ini"])
	require.True(t, names["626384712.ini"])
	require.True(t, names["账号.txt"])
}

func TestQQCacheBillingSettlementStatsIgnoreCreatedAtFilter(t *testing.T) {
	setupQQCacheTestDB(t)

	settledAt := time.Now().Add(-time.Hour)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "10001"}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "10002"}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "10003", BillingSettledAt: &settledAt}).Error)

	pending, extracted, total, err := (&QQCacheService{}).CountExtractStatsByCreatedRange("2099-01-01 00:00:00", "2099-01-02 00:00:00")
	require.NoError(t, err)
	require.EqualValues(t, 0, pending)
	require.EqualValues(t, 0, extracted)
	require.EqualValues(t, 0, total)

	unsettled, settled, err := (&QQCacheService{}).CountBillingSettlementStats()
	require.NoError(t, err)
	require.EqualValues(t, 2, unsettled)
	require.EqualValues(t, 1, settled)
}

func TestQQCacheSettleBillingAndHistory(t *testing.T) {
	setupQQCacheTestDB(t)

	alreadySettledAt := time.Now().Add(-time.Hour)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "20001"}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "20002"}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "20003", BillingSettledAt: &alreadySettledAt}).Error)

	result, err := (&QQCacheService{}).SettleBilling(100, 88)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.SettledCount)
	require.False(t, result.SettledAt.IsZero())

	unsettled, settled, err := (&QQCacheService{}).CountBillingSettlementStats()
	require.NoError(t, err)
	require.EqualValues(t, 0, unsettled)
	require.EqualValues(t, 3, settled)

	history, err := (&QQCacheService{}).GetBillingSettlementHistory(100)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.EqualValues(t, 2, history[0].SettledCount)
	require.EqualValues(t, 1, history[1].SettledCount)
}
