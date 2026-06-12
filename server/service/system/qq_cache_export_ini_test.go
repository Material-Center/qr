package system

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
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
	require.NoError(t, db.AutoMigrate(
		&model.SysQQCacheRecord{},
		&model.SysPhoneRegisterTask{},
		&model.SysUser{},
	))
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

	got := normalizeQQCacheExportINI(raw, "abc123", "")

	if strings.Contains(got, "_device_token=") {
		t.Fatalf("expected _device_token removed, got: %q", got)
	}
	if strings.Contains(got, "_superKey=") {
		t.Fatalf("expected _superKey removed, got: %q", got)
	}
	if strings.Contains(got, "qqnum=3896349451\r\n") {
		t.Fatalf("expected qqnum removed, got: %q", got)
	}
	if !strings.Contains(got, "guid=2D2F4073897A69E82FA8124BB4293162\r\n") {
		t.Fatalf("expected guid preserved, got: %q", got)
	}
	if !strings.Contains(got, "qqpassword=abc123\r\n") {
		t.Fatalf("expected missing qqpassword filled, got: %q", got)
	}
	if strings.Contains(got, "deviceInfo=") {
		t.Fatalf("expected deviceInfo removed, got: %q", got)
	}
}

func TestNormalizeQQCacheExportINIDropsDeviceInfoWhenJSONInvalid(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"deviceInfo={bad json}",
		"",
	}, "\n")

	got := normalizeQQCacheExportINI(raw, "", "")

	require.NotContains(t, got, "deviceInfo=")
}

func TestNormalizeQQCacheExportINIKeepsExistingQQPassword(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"qqpassword=oldpwd",
		"",
	}, "\n")

	got := normalizeQQCacheExportINI(raw, "newpwd", "")

	require.Contains(t, got, "qqpassword=oldpwd\r\n")
	require.NotContains(t, got, "qqpassword=newpwd")
}

func TestNormalizeQQCacheExportINIAddsVersionAliases(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"clientVersion=9.2.75",
		"",
	}, "\n")

	got := normalizeQQCacheExportINI(raw, "", "")

	require.NotContains(t, got, "clientVersion=")
	require.Contains(t, got, "版本=9.2.75\r\n")
	require.Contains(t, got, "登录协议=9.2.75\r\n")
}

func TestNormalizeQQCacheExportINIUsesRecordClientVersionWhenMissing(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"guid=GUID001",
		"",
	}, "\n")

	got := normalizeQQCacheExportINI(raw, "", "9.2.70")

	require.Contains(t, got, "版本=9.2.70\r\n")
	require.Contains(t, got, "登录协议=9.2.70\r\n")
}

func TestNormalizeQQCacheExportINIDropsMetadataAndDuplicateKeys(t *testing.T) {
	raw := strings.Join([]string{
		"[3896349451]",
		"clientId=first",
		"extractTime=1780307037496",
		"clientId=second",
		"deviceInfo={\"model\":\"picasso\"}",
		"",
	}, "\n")

	got := normalizeQQCacheExportINI(raw, "", "")

	require.Equal(t, 1, strings.Count(got, "clientId="))
	require.Contains(t, got, "clientId=first\r\n")
	require.NotContains(t, got, "clientId=second")
	require.NotContains(t, got, "extractTime=")
	require.NotContains(t, got, "deviceInfo=")
}

func TestInternalToolImportQQCacheSkipsExistingByDefault(t *testing.T) {
	setupQQCacheTestDB(t)

	oldINI := "old ini"
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		QQNum: "10001",
		QQPwd: "oldpwd",
		INI:   &oldINI,
	}).Error)

	record, action, err := (&QQCacheService{}).InternalToolImportQQCache(systemReq.InternalToolQQCacheImport{
		QQNum: "10001",
		QQPwd: "newpwd",
		INI:   "new ini",
	})
	require.NoError(t, err)
	require.Equal(t, qqCacheInternalToolActionSkipped, action)
	require.Equal(t, "oldpwd", record.QQPwd)

	var stored model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "10001").First(&stored).Error)
	require.Equal(t, "oldpwd", stored.QQPwd)
	require.NotNil(t, stored.INI)
	require.Equal(t, oldINI, *stored.INI)
}

func TestInternalToolImportQQCacheForceUpdatesOnlyCacheFields(t *testing.T) {
	setupQQCacheTestDB(t)

	oldINI := "old ini"
	extractor := uint(88)
	extractRecordID := uint(99)
	extractionAt := time.Now().Add(-time.Hour)
	billingSettledBy := uint(100)
	billingSettledAt := time.Now().Add(-time.Minute)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		QQNum:            "10002",
		QQPwd:            "oldpwd",
		INI:              &oldINI,
		Extractor:        &extractor,
		ExtractRecordID:  &extractRecordID,
		ExtractionAt:     &extractionAt,
		BillingSettledAt: &billingSettledAt,
		BillingSettledBy: &billingSettledBy,
	}).Error)

	record, action, err := (&QQCacheService{}).InternalToolImportQQCache(systemReq.InternalToolQQCacheImport{
		QQNum: "10002",
		QQPwd: "newpwd",
		INI:   "new ini",
		Force: true,
	})
	require.NoError(t, err)
	require.Equal(t, qqCacheInternalToolActionUpdated, action)
	require.Equal(t, "newpwd", record.QQPwd)

	var stored model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "10002").First(&stored).Error)
	require.Equal(t, "newpwd", stored.QQPwd)
	require.NotNil(t, stored.INI)
	require.Equal(t, "new ini", *stored.INI)
	require.NotNil(t, stored.Extractor)
	require.Equal(t, extractor, *stored.Extractor)
	require.NotNil(t, stored.ExtractRecordID)
	require.Equal(t, extractRecordID, *stored.ExtractRecordID)
	require.NotNil(t, stored.ExtractionAt)
	require.NotNil(t, stored.BillingSettledAt)
	require.NotNil(t, stored.BillingSettledBy)
	require.Equal(t, billingSettledBy, *stored.BillingSettledBy)
}

func TestAdminImportQQCacheOverwritesExistingByDefault(t *testing.T) {
	setupQQCacheTestDB(t)

	oldINI := "old ini"
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		QQNum: "10006",
		QQPwd: "oldpwd",
		INI:   &oldINI,
	}).Error)

	record, action, err := (&QQCacheService{}).AdminImportQQCache(systemReq.InternalToolQQCacheImport{
		QQNum: "10006",
		QQPwd: "newpwd",
		INI:   "new ini",
	})
	require.NoError(t, err)
	require.Equal(t, qqCacheInternalToolActionUpdated, action)
	require.Equal(t, "newpwd", record.QQPwd)

	var stored model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "10006").First(&stored).Error)
	require.Equal(t, "newpwd", stored.QQPwd)
	require.NotNil(t, stored.INI)
	require.Equal(t, "new ini", *stored.INI)
}

func TestInternalToolImportQQCacheStoresClientVersion(t *testing.T) {
	setupQQCacheTestDB(t)

	_, action, err := (&QQCacheService{}).InternalToolImportQQCache(systemReq.InternalToolQQCacheImport{
		QQNum: "10003",
		QQPwd: "pwd",
		INI: strings.Join([]string{
			"[10003]",
			"clientVersion=9.2.70",
			"",
		}, "\n"),
	})
	require.NoError(t, err)
	require.Equal(t, qqCacheInternalToolActionCreated, action)

	var stored model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "10003").First(&stored).Error)
	require.Equal(t, "9.2.70", stored.ClientVersion)
}

func TestQQCacheListForAdminFiltersClientVersion(t *testing.T) {
	setupQQCacheTestDB(t)

	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "10004", ClientVersion: "9.2.70"}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "10005", ClientVersion: "8.9.80"}).Error)

	list, total, err := (&QQCacheService{}).ListForAdmin(systemReq.QQCacheList{
		ClientVersion: "9.2",
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, "10004", list[0].QQNum)
}

func TestQQCacheListForAdminUsesStablePaginationOrder(t *testing.T) {
	setupQQCacheTestDB(t)

	now := time.Now()
	for i := 1; i <= 3; i++ {
		require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
			GVA_MODEL: global.GVA_MODEL{
				CreatedAt: now,
				UpdatedAt: now,
			},
			QQNum: fmt.Sprintf("1000%d", i),
		}).Error)
	}

	firstPage, total, err := (&QQCacheService{}).ListForAdmin(systemReq.QQCacheList{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 2},
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, firstPage, 2)

	secondPage, total, err := (&QQCacheService{}).ListForAdmin(systemReq.QQCacheList{
		PageInfo: commonReq.PageInfo{Page: 2, PageSize: 2},
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, secondPage, 1)
	require.Greater(t, firstPage[0].ID, firstPage[1].ID)
	require.Greater(t, firstPage[1].ID, secondPage[0].ID)
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

func TestExportIniZipByQQTextAndMarkExtracted(t *testing.T) {
	setupQQCacheTestDB(t)

	ini1 := "qqnum=70001\nguid=GUID001\n"
	ini2 := "qqnum=70002\nguid=GUID002\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "70001", QQPwd: "pwd1", INI: &ini1}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "70002", QQPwd: "pwd2", INI: &ini2}).Error)

	zipBytes, count, err := (&QQCacheService{}).ExportIniZipByQQTextAndMarkExtracted(strings.Join([]string{
		"70002----updated",
		"70001----created",
	}, "\n"), 88)
	require.NoError(t, err)
	require.NotEmpty(t, zipBytes)
	require.EqualValues(t, 2, count)

	var records []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num IN ?", []string{"70001", "70002"}).Find(&records).Error)
	require.Len(t, records, 2)
	for _, record := range records {
		require.NotNil(t, record.Extractor)
		require.EqualValues(t, 88, *record.Extractor)
		require.NotNil(t, record.ExtractRecordID)
		require.EqualValues(t, record.ID, *record.ExtractRecordID)
		require.NotNil(t, record.ExtractionAt)
	}
}

func TestExportPendingIniZipByCountUsesOldestCreatedRecords(t *testing.T) {
	setupQQCacheTestDB(t)

	base := time.Date(2026, 5, 30, 10, 0, 0, 0, time.Local)
	records := []model.SysQQCacheRecord{
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: base, UpdatedAt: base.Add(3 * time.Hour)},
			QQNum:     "80001",
			INI:       stringPtr("qqnum=80001\nguid=GUID001\n"),
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: base.Add(time.Hour), UpdatedAt: base.Add(2 * time.Hour)},
			QQNum:     "80002",
			INI:       stringPtr("qqnum=80002\nguid=GUID002\n"),
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: base.Add(2 * time.Hour), UpdatedAt: base.Add(time.Hour)},
			QQNum:     "80003",
			INI:       stringPtr("qqnum=80003\nguid=GUID003\n"),
		},
	}
	require.NoError(t, global.GVA_DB.Create(&records).Error)

	_, count, err := (&QQCacheService{}).ExportPendingIniZipByCount(2, 99, "", "")
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	var extracted []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", 99).Order("created_at asc").Find(&extracted).Error)
	require.Len(t, extracted, 2)
	require.Equal(t, "80001", extracted[0].QQNum)
	require.Equal(t, "80002", extracted[1].QQNum)
}

func TestExportPendingIniZipByCountFiltersRecentMinutesWindow(t *testing.T) {
	setupQQCacheTestDB(t)

	now := time.Now()
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			QQNum:     "81001",
			INI:       stringPtr("qqnum=81001\nguid=GUID001\n"),
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-30 * time.Minute)},
			QQNum:     "81002",
			INI:       stringPtr("qqnum=81002\nguid=GUID002\n"),
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-20 * time.Minute), UpdatedAt: now.Add(-20 * time.Minute)},
			QQNum:     "81003",
		},
	}).Error)

	pending, extracted, total, err := (&QQCacheService{}).CountExtractStatsByRecentMinutes(60)
	require.NoError(t, err)
	require.EqualValues(t, 1, pending)
	require.EqualValues(t, 0, extracted)
	require.EqualValues(t, 2, total)

	pending, extracted, total, err = (&QQCacheService{}).CountExtractStatsByRecentMinutes(240)
	require.NoError(t, err)
	require.EqualValues(t, 2, pending)
	require.EqualValues(t, 0, extracted)
	require.EqualValues(t, 3, total)

	_, count, err := (&QQCacheService{}).ExportPendingIniZipByCountWithRecentMinutes(2, 99, 60)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	var extractedRecords []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", 99).Find(&extractedRecords).Error)
	require.Len(t, extractedRecords, 1)
	require.Equal(t, "81002", extractedRecords[0].QQNum)
}

func TestExportPendingIniZipByCountFiltersRecentMinutes(t *testing.T) {
	setupQQCacheTestDB(t)

	now := time.Now()
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-40 * time.Minute), UpdatedAt: now.Add(-40 * time.Minute)},
			QQNum:     "82001",
			INI:       stringPtr("qqnum=82001\nguid=GUID001\n"),
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-20 * time.Minute), UpdatedAt: now.Add(-20 * time.Minute)},
			QQNum:     "82002",
			INI:       stringPtr("qqnum=82002\nguid=GUID002\n"),
		},
	}).Error)

	pending, extracted, total, err := (&QQCacheService{}).CountExtractStatsByRecentMinutes(30)
	require.NoError(t, err)
	require.EqualValues(t, 1, pending)
	require.EqualValues(t, 0, extracted)
	require.EqualValues(t, 1, total)

	_, count, err := (&QQCacheService{}).ExportPendingIniZipByCountWithRecentMinutes(2, 99, 30)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	var extractedRecords []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", 99).Find(&extractedRecords).Error)
	require.Len(t, extractedRecords, 1)
	require.Equal(t, "82002", extractedRecords[0].QQNum)
}

func TestExportPendingIniZipByCountFiltersOlderThanRecentMinutes(t *testing.T) {
	setupQQCacheTestDB(t)

	now := time.Now()
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now.Add(-4 * time.Hour)},
			QQNum:     "83001",
			INI:       stringPtr("qqnum=83001\nguid=GUID001\n"),
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			QQNum:     "83002",
			INI:       stringPtr("qqnum=83002\nguid=GUID002\n"),
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-20 * time.Minute), UpdatedAt: now.Add(-20 * time.Minute)},
			QQNum:     "83003",
			INI:       stringPtr("qqnum=83003\nguid=GUID003\n"),
		},
	}).Error)

	pending, extracted, total, err := (&QQCacheService{}).CountExtractStatsByRecentMinutes(-180)
	require.NoError(t, err)
	require.EqualValues(t, 1, pending)
	require.EqualValues(t, 0, extracted)
	require.EqualValues(t, 1, total)

	_, count, err := (&QQCacheService{}).ExportPendingIniZipByCountWithRecentMinutes(2, 99, -180)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	var extractedRecords []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", 99).Find(&extractedRecords).Error)
	require.Len(t, extractedRecords, 1)
	require.Equal(t, "83001", extractedRecords[0].QQNum)
}

func TestExportAccountListTextBySelectedIDs(t *testing.T) {
	setupQQCacheTestDB(t)

	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "40001", ClientVersion: "9.2.70"}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "40002", ClientVersion: "8.9.80"}).Error)

	var selected model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "40001").First(&selected).Error)
	text, count, err := (&QQCacheService{}).ExportAccountListText(systemReq.QQCacheExportAccountList{
		IDs: []uint{selected.ID},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, "40001----9.2.70---------------\r\n", text)
}

func TestExportAccountListTextByFiltersDoesNotMarkExtracted(t *testing.T) {
	setupQQCacheTestDB(t)

	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "50001", ClientVersion: "9.2.70"}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "50002", ClientVersion: "8.9.80"}).Error)

	text, count, err := (&QQCacheService{}).ExportAccountListText(systemReq.QQCacheExportAccountList{
		ClientVersion: "9.2",
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, "50001----9.2.70---------------\r\n", text)

	var stored model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "50001").First(&stored).Error)
	require.Nil(t, stored.Extractor)
	require.Nil(t, stored.ExtractionAt)
}

func TestExportAccountListTextIncludesDeviceLeaderAndReceiveMode(t *testing.T) {
	setupQQCacheTestDB(t)

	leader := model.SysUser{NickName: "团长A", Username: "leader-a", AuthorityId: 200}
	require.NoError(t, global.GVA_DB.Create(&leader).Error)
	deviceID := "device-a"
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		QQNum:         "60001",
		ClientVersion: "9.2.70",
		DeviceID:      &deviceID,
	}).Error)
	var record model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "60001").First(&record).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysPhoneRegisterTask{
		QQCacheRecordID: &record.ID,
		LeaderID:        &leader.ID,
		SMSReceiveMode:  model.PhoneRegisterSMSModePlatformSend,
		QQNum:           record.QQNum,
	}).Error)

	text, count, err := (&QQCacheService{}).ExportAccountListText(systemReq.QQCacheExportAccountList{
		IDs: []uint{record.ID},
	})

	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, "60001----9.2.70----device-a----团长A----收码\r\n", text)
}

func TestExportAccountListTextFormatsUserSentModeAsSendCode(t *testing.T) {
	setupQQCacheTestDB(t)

	leader := model.SysUser{NickName: "团长B", Username: "leader-b", AuthorityId: 200}
	require.NoError(t, global.GVA_DB.Create(&leader).Error)
	deviceID := "device-b"
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		QQNum:         "60002",
		ClientVersion: "9.2.75",
		DeviceID:      &deviceID,
	}).Error)
	var record model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "60002").First(&record).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysPhoneRegisterTask{
		QQCacheRecordID: &record.ID,
		LeaderID:        &leader.ID,
		SMSReceiveMode:  model.PhoneRegisterSMSModeUserSentToTX,
		QQNum:           record.QQNum,
	}).Error)

	text, count, err := (&QQCacheService{}).ExportAccountListText(systemReq.QQCacheExportAccountList{
		IDs: []uint{record.ID},
	})

	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, "60002----9.2.75----device-b----团长B----发码\r\n", text)
}

func TestExportAccountListTextByQQTextKeepsInputOrder(t *testing.T) {
	setupQQCacheTestDB(t)

	deviceA := "device-a"
	deviceB := "device-b"
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{QQNum: "70001", ClientVersion: "9.2.70", DeviceID: &deviceA},
		{QQNum: "70002", ClientVersion: "9.2.75", DeviceID: &deviceB},
	}).Error)

	text, count, err := (&QQCacheService{}).ExportAccountListTextByQQText(strings.Join([]string{
		"70002----旧状态",
		"70001",
	}, "\n"))

	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Equal(t, strings.Join([]string{
		"70002----9.2.75----device-b----------",
		"70001----9.2.70----device-a----------",
		"",
	}, "\r\n"), text)
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
