package system

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupQQCacheSalesTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.SysUser{},
		&model.SysQQCacheRecord{},
		&model.SysQQCacheExtractBatch{},
		&model.SysParams{},
	))
	global.GVA_DB = db
	require.NoError(t, (&QQCacheService{}).SaveSalesAllowedAccountTypes([]string{AccountTypeDefault, AccountTypePC}))
}

func clearQQCacheSalesAllowedAccountTypes(t *testing.T) {
	t.Helper()
	require.NoError(t, global.GVA_DB.Where("`key` = ?", qqCacheSalesAllowedAccountTypesKey).Delete(&model.SysParams{}).Error)
}

func TestQQCacheSalesSummaryUsesGlobalAvailableAndTodaySalesCounts(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6001)
	today := time.Now()
	yesterday := today.Add(-24 * time.Hour)
	settledAt := today.Add(-time.Hour)
	ini := "qqnum=10001\nguid=GUID001\n"

	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: yesterday, UpdatedAt: yesterday}, QQNum: "10001", INI: &ini},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: today, UpdatedAt: today}, QQNum: "10002", INI: &ini},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: today, UpdatedAt: today}, QQNum: "10003"},
		{QQNum: "20001", Extractor: &salesID, ExtractionAt: &today, INI: &ini},
		{QQNum: "20002", Extractor: &salesID, ExtractionAt: &today, SalesSettledAt: &settledAt, INI: &ini},
		{QQNum: "20003", Extractor: &salesID, ExtractionAt: &yesterday, INI: &ini},
	}).Error)

	summary, err := (&QQCacheService{}).GetSalesSummary(salesID, "")
	require.NoError(t, err)
	require.EqualValues(t, 2, summary.Available)
	require.EqualValues(t, 2, summary.TodayExtracted)
	require.EqualValues(t, 1, summary.TodayUnsettled)
}

func TestQQCacheSalesMissingAllowedAccountTypesBlocksExport(t *testing.T) {
	setupQQCacheSalesTestDB(t)
	clearQQCacheSalesAllowedAccountTypes(t)

	salesID := uint(6010)
	ini := "qqnum=91001\nguid=GUID91001\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		QQNum:       "91001",
		INI:         &ini,
		AccountType: AccountTypeDefault,
	}).Error)

	summary, err := (&QQCacheService{}).GetSalesSummary(salesID, "")
	require.NoError(t, err)
	require.EqualValues(t, 0, summary.Available)

	_, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(1, salesID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "未配置可导出账号类型")
	require.Zero(t, count)
	require.Zero(t, batch.ID)
}

func TestQQCacheSalesAllowedAccountTypesFiltersSummaryAndExtract(t *testing.T) {
	setupQQCacheSalesTestDB(t)
	require.NoError(t, (&QQCacheService{}).SaveSalesAllowedAccountTypes([]string{AccountTypePC}))

	salesID := uint(6011)
	ini := "qqnum=92001\nguid=GUID92001\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_pc",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{QQNum: "92001", INI: &ini, AccountType: AccountTypePC},
		{QQNum: "92002", INI: &ini, AccountType: AccountTypeDefault},
	}).Error)

	summary, err := (&QQCacheService{}).GetSalesSummary(salesID, "")
	require.NoError(t, err)
	require.EqualValues(t, 1, summary.Available)

	_, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(2, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.EqualValues(t, 1, batch.ExtractCount)

	var extracted []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", salesID).Find(&extracted).Error)
	require.Len(t, extracted, 1)
	require.Equal(t, "92001", extracted[0].QQNum)
}

func TestQQCacheSalesAllowedAccountTypesReadsLatestDuplicateAndSaveConverges(t *testing.T) {
	setupQQCacheSalesTestDB(t)
	clearQQCacheSalesAllowedAccountTypes(t)

	require.NoError(t, global.GVA_DB.Create(&model.SysParams{
		Name:  "旧配置",
		Key:   qqCacheSalesAllowedAccountTypesKey,
		Value: `["default"]`,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysParams{
		Name:  "新配置",
		Key:   qqCacheSalesAllowedAccountTypesKey,
		Value: `["pc"]`,
	}).Error)

	types, err := (&QQCacheService{}).GetSalesAllowedAccountTypes()
	require.NoError(t, err)
	require.Equal(t, []string{AccountTypePC}, types)

	require.NoError(t, (&QQCacheService{}).SaveSalesAllowedAccountTypes([]string{AccountTypeDefault, AccountTypePC}))
	var rows []model.SysParams
	require.NoError(t, global.GVA_DB.Where("`key` = ?", qqCacheSalesAllowedAccountTypesKey).Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, row := range rows {
		require.JSONEq(t, `["default","pc"]`, row.Value)
	}
}

func TestQQCacheSalesAllowedAccountTypesSaveSameValueDoesNotCreateDuplicate(t *testing.T) {
	setupQQCacheSalesTestDB(t)
	clearQQCacheSalesAllowedAccountTypes(t)

	require.NoError(t, (&QQCacheService{}).SaveSalesAllowedAccountTypes([]string{AccountTypeDefault}))
	require.NoError(t, (&QQCacheService{}).SaveSalesAllowedAccountTypes([]string{AccountTypeDefault}))

	var total int64
	require.NoError(t, global.GVA_DB.Model(&model.SysParams{}).Where("`key` = ?", qqCacheSalesAllowedAccountTypesKey).Count(&total).Error)
	require.EqualValues(t, 1, total)
}

func TestQQCacheSalesAllowedDefaultIncludesHistoricalInvalidType(t *testing.T) {
	setupQQCacheSalesTestDB(t)
	require.NoError(t, (&QQCacheService{}).SaveSalesAllowedAccountTypes([]string{AccountTypeDefault}))

	salesID := uint(6013)
	ini := "qqnum=93001\nguid=GUID93001\n"
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{QQNum: "93001", INI: &ini, AccountType: AccountTypeDefault},
		{QQNum: "93002", INI: &ini, AccountType: "mobile"},
		{QQNum: "93003", INI: &ini, AccountType: AccountTypePC},
	}).Error)

	summary, err := (&QQCacheService{}).GetSalesSummary(salesID, "")
	require.NoError(t, err)
	require.EqualValues(t, 2, summary.Available)
}

func TestQQCacheSalesAllowedDefaultOnlyExtractsDefaultAndHistoricalInvalidType(t *testing.T) {
	setupQQCacheSalesTestDB(t)
	require.NoError(t, (&QQCacheService{}).SaveSalesAllowedAccountTypes([]string{AccountTypeDefault}))

	salesID := uint(6014)
	ini := "qqnum=93101\nguid=GUID93101\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_default",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{QQNum: "93101", INI: &ini, AccountType: AccountTypeDefault},
		{QQNum: "93102", INI: &ini, AccountType: "mobile"},
		{QQNum: "93103", INI: &ini, AccountType: AccountTypePC},
	}).Error)

	_, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(3, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.EqualValues(t, 2, batch.ExtractCount)

	var extracted []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", salesID).Order("qq_num asc").Find(&extracted).Error)
	require.Len(t, extracted, 2)
	require.Equal(t, "93101", extracted[0].QQNum)
	require.Equal(t, "93102", extracted[1].QQNum)
}

func TestQQCacheSalesSummaryAndExtractFilterRecentMinutes(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6012)
	now := time.Now()
	iniOld := "qqnum=10101\nguid=GUID101\n"
	iniRecent := "qqnum=10102\nguid=GUID102\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_recent",
		NickName:    "最近提取销售",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			QQNum:     "10101",
			INI:       &iniOld,
		},
		{
			GVA_MODEL: global.GVA_MODEL{CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-30 * time.Minute)},
			QQNum:     "10102",
			INI:       &iniRecent,
		},
	}).Error)

	summary, err := (&QQCacheService{}).GetSalesSummaryWithRecentMinutes(salesID, "", 60)
	require.NoError(t, err)
	require.EqualValues(t, 1, summary.Available)

	_, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCountWithRecentMinutes(2, salesID, 60)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.EqualValues(t, 1, batch.ExtractCount)

	var extracted []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", salesID).Find(&extracted).Error)
	require.Len(t, extracted, 1)
	require.Equal(t, "10102", extracted[0].QQNum)
}

func TestQQCacheSalesExtractCreatesBatchAndHistory(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6002)
	ini1 := "qqnum=30001\nguid=GUID001\n"
	ini2 := "qqnum=30002\nguid=GUID002\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_a",
		NickName:    "销售A",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{QQNum: "30001", QQPwd: "pwd1", INI: &ini1},
		{QQNum: "30002", QQPwd: "pwd2", INI: &ini2},
	}).Error)

	zipBytes, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(2, salesID)
	require.NoError(t, err)
	require.NotEmpty(t, zipBytes)
	require.EqualValues(t, 2, count)
	require.EqualValues(t, salesID, batch.ExtractorID)
	require.Equal(t, "销售A", batch.ExtractorName)
	require.EqualValues(t, 2, batch.ExtractCount)
	require.Equal(t, model.QQCacheExtractBatchStatusPendingSettlement, batch.Status)
	require.False(t, batch.ExtractedAt.IsZero())

	var extracted []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", salesID).Find(&extracted).Error)
	require.Len(t, extracted, 2)
	for _, record := range extracted {
		require.NotNil(t, record.ExtractRecordID)
		require.Equal(t, batch.ID, *record.ExtractRecordID)
		require.NotNil(t, record.ExtractionAt)
	}

	history, total, err := (&QQCacheService{}).ListSalesExtractHistory(salesID, systemReq.QQCacheSalesHistory{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, history, 1)
	require.EqualValues(t, 2, history[0].ExtractCount)
	require.Equal(t, "待结算", history[0].SettlementStatusText)
}

func TestQQCacheSettleSalesBillingUpdatesBatchStatus(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6003)
	ini := "qqnum=40001\nguid=GUID001\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_b",
		NickName:    "销售B",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	_, _, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(0, salesID)
	require.Error(t, err)
	require.Zero(t, batch.ID)

	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{QQNum: "40001", INI: &ini},
		{QQNum: "40002", INI: &ini},
	}).Error)
	_, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(2, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	result, err := (&QQCacheService{}).SettleSalesBilling(100, 88, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.SettledCount)

	var storedBatch model.SysQQCacheExtractBatch
	require.NoError(t, global.GVA_DB.First(&storedBatch, batch.ID).Error)
	require.Equal(t, model.QQCacheExtractBatchStatusSettled, storedBatch.Status)
	require.EqualValues(t, 2, storedBatch.SettledCount)
	require.NotNil(t, storedBatch.SettledAt)
	require.NotNil(t, storedBatch.SettledBy)
	require.EqualValues(t, 88, *storedBatch.SettledBy)

	var settledRecords []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", salesID).Find(&settledRecords).Error)
	require.Len(t, settledRecords, 2)
	for _, record := range settledRecords {
		require.NotNil(t, record.SalesSettledAt)
		require.NotNil(t, record.SalesSettledBy)
		require.Nil(t, record.BillingSettledAt)
		require.Nil(t, record.BillingSettledBy)
	}

	summaries, err := (&QQCacheService{}).ListSalesSummaryForAdmin("", "", qqCacheSalesTimeFilterCreatedAt)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.EqualValues(t, salesID, summaries[0].ExtractorID)
	require.EqualValues(t, 2, summaries[0].ExtractedCount)
	require.EqualValues(t, 2, summaries[0].SettledCount)
	require.EqualValues(t, 0, summaries[0].UnsettledCount)

	history, err := (&QQCacheService{}).GetSalesSettlementHistory(100, salesID)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.EqualValues(t, 2, history[0].SettledCount)
}

func TestQQCacheSettleSalesBillingRefreshesOnlyAffectedBatches(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6033)
	operatorID := uint(88)
	oldOperatorID := uint(11)
	oldSettledAt := time.Now().Add(-24 * time.Hour)
	now := time.Now()
	ini := "qqnum=40331\nguid=GUID40331\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_scoped_settle",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	oldBatch := model.SysQQCacheExtractBatch{
		ExtractorID:  salesID,
		ExtractCount: 1,
		SettledCount: 1,
		Status:       model.QQCacheExtractBatchStatusSettled,
		ExtractedAt:  oldSettledAt.Add(-time.Hour),
		SettledAt:    &oldSettledAt,
		SettledBy:    &oldOperatorID,
	}
	pendingBatch := model.SysQQCacheExtractBatch{
		ExtractorID:  salesID,
		ExtractCount: 2,
		SettledCount: 1,
		Status:       model.QQCacheExtractBatchStatusPendingSettlement,
		ExtractedAt:  now,
	}
	require.NoError(t, global.GVA_DB.Create(&oldBatch).Error)
	require.NoError(t, global.GVA_DB.Create(&pendingBatch).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{
			QQNum:           "40331",
			INI:             &ini,
			Extractor:       &salesID,
			ExtractRecordID: &oldBatch.ID,
			ExtractionAt:    &oldBatch.ExtractedAt,
			SalesSettledAt:  &oldSettledAt,
			SalesSettledBy:  &oldOperatorID,
		},
		{
			QQNum:           "40332",
			INI:             &ini,
			Extractor:       &salesID,
			ExtractRecordID: &pendingBatch.ID,
			ExtractionAt:    &now,
		},
		{
			QQNum:           "40333",
			INI:             &ini,
			Extractor:       &salesID,
			ExtractRecordID: &pendingBatch.ID,
			ExtractionAt:    &now,
			SalesSettledAt:  &oldSettledAt,
			SalesSettledBy:  &oldOperatorID,
		},
	}).Error)

	result, err := (&QQCacheService{}).SettleSalesBilling(100, operatorID, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.SettledCount)

	var storedOldBatch model.SysQQCacheExtractBatch
	require.NoError(t, global.GVA_DB.First(&storedOldBatch, oldBatch.ID).Error)
	require.Equal(t, model.QQCacheExtractBatchStatusSettled, storedOldBatch.Status)
	require.EqualValues(t, 1, storedOldBatch.SettledCount)
	require.NotNil(t, storedOldBatch.SettledAt)
	require.True(t, oldSettledAt.Equal(*storedOldBatch.SettledAt))
	require.NotNil(t, storedOldBatch.SettledBy)
	require.EqualValues(t, oldOperatorID, *storedOldBatch.SettledBy)

	var storedPendingBatch model.SysQQCacheExtractBatch
	require.NoError(t, global.GVA_DB.First(&storedPendingBatch, pendingBatch.ID).Error)
	require.Equal(t, model.QQCacheExtractBatchStatusSettled, storedPendingBatch.Status)
	require.EqualValues(t, 2, storedPendingBatch.SettledCount)
	require.NotNil(t, storedPendingBatch.SettledAt)
	require.NotNil(t, storedPendingBatch.SettledBy)
	require.EqualValues(t, operatorID, *storedPendingBatch.SettledBy)
}

func TestQQCacheSettleSalesBillingRejectsNonSalesExtractor(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	adminExtractorID := uint(1001)
	now := time.Now()
	ini := "qqnum=50001\nguid=GUID001\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: adminExtractorID},
		Username:    "admin_a",
		AuthorityId: 100,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		QQNum:        "50001",
		INI:          &ini,
		Extractor:    &adminExtractorID,
		ExtractionAt: &now,
	}).Error)

	result, err := (&QQCacheService{}).SettleSalesBilling(100, 88, adminExtractorID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "销售账号")
	require.Zero(t, result.SettledCount)

	var record model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "50001").First(&record).Error)
	require.Nil(t, record.SalesSettledAt)
	require.Nil(t, record.SalesSettledBy)
	require.Nil(t, record.BillingSettledAt)
	require.Nil(t, record.BillingSettledBy)
}

func TestQQCacheSalesSummaryListFiltersSalesWithoutExtractsInRange(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesWithExtractID := uint(6101)
	salesWithoutExtractID := uint(6102)
	adminExtractorID := uint(1101)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	ini := "qqnum=51001\nguid=GUID001\n"
	require.NoError(t, global.GVA_DB.Create(&[]model.SysUser{
		{GVA_MODEL: global.GVA_MODEL{ID: salesWithExtractID}, Username: "sales_with", NickName: "已提取销售", AuthorityId: 600, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: salesWithoutExtractID}, Username: "sales_empty", NickName: "未提取销售", AuthorityId: 600, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: adminExtractorID}, Username: "admin_extractor", AuthorityId: 100, Enable: 1},
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, QQNum: "51001", INI: &ini, Extractor: &salesWithExtractID, ExtractionAt: &now},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, QQNum: "51002", INI: &ini, Extractor: &adminExtractorID, ExtractionAt: &now},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: yesterday, UpdatedAt: yesterday}, QQNum: "51003", INI: &ini, Extractor: &salesWithExtractID, ExtractionAt: &yesterday},
	}).Error)

	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	summaries, err := (&QQCacheService{}).ListSalesSummaryForAdmin(
		todayStart.Format("2006-01-02 15:04:05"),
		todayEnd.Format("2006-01-02 15:04:05"),
		qqCacheSalesTimeFilterCreatedAt,
	)
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	byID := map[uint]systemRes.QQCacheSalesAdminSummaryItem{}
	for _, item := range summaries {
		byID[item.ExtractorID] = item
	}
	require.EqualValues(t, 1, byID[salesWithExtractID].ExtractedCount)
	require.EqualValues(t, 1, byID[salesWithExtractID].UnsettledCount)
	require.NotContains(t, byID, salesWithoutExtractID)
	require.NotContains(t, byID, adminExtractorID)

	allSummaries, err := (&QQCacheService{}).ListSalesSummaryForAdmin("", "", qqCacheSalesTimeFilterCreatedAt)
	require.NoError(t, err)
	for _, item := range allSummaries {
		byID[item.ExtractorID] = item
	}
	require.EqualValues(t, 2, byID[salesWithExtractID].ExtractedCount)
}

func TestQQCacheSalesSummaryTimeFilter(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6103)
	uploadedAt := time.Date(2026, 7, 19, 12, 0, 0, 0, time.Local)
	extractedAt := time.Date(2026, 7, 20, 18, 0, 0, 0, time.Local)
	ini := "qqnum=51004\nguid=GUID004\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_time_filter",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	batch := model.SysQQCacheExtractBatch{
		ExtractorID:  salesID,
		ExtractCount: 1,
		Status:       model.QQCacheExtractBatchStatusPendingSettlement,
		ExtractedAt:  extractedAt,
	}
	require.NoError(t, global.GVA_DB.Create(&batch).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{
		GVA_MODEL:       global.GVA_MODEL{CreatedAt: uploadedAt, UpdatedAt: uploadedAt},
		QQNum:           "51004",
		INI:             &ini,
		Extractor:       &salesID,
		ExtractRecordID: &batch.ID,
		ExtractionAt:    &extractedAt,
	}).Error)

	uploadStart := time.Date(2026, 7, 19, 0, 0, 0, 0, time.Local)
	uploadEnd := time.Date(2026, 7, 19, 23, 59, 59, 0, time.Local)
	extractStart := uploadStart.AddDate(0, 0, 1)
	extractEnd := uploadEnd.AddDate(0, 0, 1)

	uploadSummaries, err := (&QQCacheService{}).ListSalesSummaryForAdmin(
		uploadStart.Format("2006-01-02 15:04:05"),
		uploadEnd.Format("2006-01-02 15:04:05"),
		qqCacheSalesTimeFilterCreatedAt,
	)
	require.NoError(t, err)
	require.Len(t, uploadSummaries, 1)

	uploadExtractionSummaries, err := (&QQCacheService{}).ListSalesSummaryForAdmin(
		uploadStart.Format("2006-01-02 15:04:05"),
		uploadEnd.Format("2006-01-02 15:04:05"),
		qqCacheSalesTimeFilterExtractedAt,
	)
	require.NoError(t, err)
	require.Empty(t, uploadExtractionSummaries)

	extractSummaries, err := (&QQCacheService{}).ListSalesSummaryForAdmin(
		extractStart.Format("2006-01-02 15:04:05"),
		extractEnd.Format("2006-01-02 15:04:05"),
		qqCacheSalesTimeFilterExtractedAt,
	)
	require.NoError(t, err)
	require.Len(t, extractSummaries, 1)

	uploadBatches, err := (&QQCacheService{}).ListSalesExtractBatchesForAdmin(
		100,
		salesID,
		uploadStart.Format("2006-01-02 15:04:05"),
		uploadEnd.Format("2006-01-02 15:04:05"),
		qqCacheSalesTimeFilterExtractedAt,
	)
	require.NoError(t, err)
	require.Empty(t, uploadBatches)

	extractBatches, err := (&QQCacheService{}).ListSalesExtractBatchesForAdmin(
		100,
		salesID,
		extractStart.Format("2006-01-02 15:04:05"),
		extractEnd.Format("2006-01-02 15:04:05"),
		qqCacheSalesTimeFilterExtractedAt,
	)
	require.NoError(t, err)
	require.Len(t, extractBatches, 1)
	require.EqualValues(t, batch.ID, extractBatches[0].ID)
}

func TestQQCacheGlobalBillingDoesNotSettleSalesBatch(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6004)
	ini := "qqnum=60001\nguid=GUID001\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_c",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{QQNum: "60001", INI: &ini},
		{QQNum: "60002", INI: &ini},
	}).Error)
	_, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(2, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	globalResult, err := (&QQCacheService{}).SettleBilling(100, 88)
	require.NoError(t, err)
	require.EqualValues(t, 2, globalResult.SettledCount)

	var storedBatch model.SysQQCacheExtractBatch
	require.NoError(t, global.GVA_DB.First(&storedBatch, batch.ID).Error)
	require.Equal(t, model.QQCacheExtractBatchStatusPendingSettlement, storedBatch.Status)
	require.EqualValues(t, 0, storedBatch.SettledCount)
	require.Nil(t, storedBatch.SettledAt)
	require.Nil(t, storedBatch.SettledBy)

	var records []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extractor = ?", salesID).Find(&records).Error)
	require.Len(t, records, 2)
	for _, record := range records {
		require.NotNil(t, record.BillingSettledAt)
		require.NotNil(t, record.BillingSettledBy)
		require.Nil(t, record.SalesSettledAt)
		require.Nil(t, record.SalesSettledBy)
	}
}

func TestQQCacheResetExtractRejectsSalesBatchRecord(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6005)
	ini := "qqnum=70001\nguid=GUID001\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_d",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&model.SysQQCacheRecord{QQNum: "70001", INI: &ini}).Error)
	_, count, _, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(1, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	var record model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("qq_num = ?", "70001").First(&record).Error)
	require.NotNil(t, record.Extractor)
	require.NotNil(t, record.ExtractRecordID)

	err = (&QQCacheService{}).ResetExtractByID(record.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "销售提取记录不可重置")

	var stored model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.First(&stored, record.ID).Error)
	require.NotNil(t, stored.Extractor)
	require.NotNil(t, stored.ExtractRecordID)
	require.NotNil(t, stored.ExtractionAt)
}

func TestQQCacheSalesBatchRedownloadUsesCreatedAtRangeAndDoesNotMutateState(t *testing.T) {
	setupQQCacheSalesTestDB(t)

	salesID := uint(6006)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	iniA := "qqnum=80001\nguid=GUID001\n"
	iniB := "qqnum=80002\nguid=GUID002\n"
	require.NoError(t, global.GVA_DB.Create(&model.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: salesID},
		Username:    "sales_e",
		NickName:    "销售E",
		AuthorityId: 600,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]model.SysQQCacheRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: yesterday, UpdatedAt: yesterday}, QQNum: "80001", INI: &iniA},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, QQNum: "80002", INI: &iniB},
	}).Error)
	_, count, batch, err := (&QQCacheService{}).ExportSalesPendingIniZipByCount(2, salesID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	batches, err := (&QQCacheService{}).ListSalesExtractBatchesForAdmin(
		100,
		salesID,
		todayStart.Format("2006-01-02 15:04:05"),
		todayEnd.Format("2006-01-02 15:04:05"),
		qqCacheSalesTimeFilterCreatedAt,
	)
	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.EqualValues(t, batch.ID, batches[0].ID)
	require.EqualValues(t, 1, batches[0].ExtractCount)

	zipBytes, exportedCount, err := (&QQCacheService{}).ExportSalesExtractBatchIniZipForAdmin(
		100,
		salesID,
		batch.ID,
		todayStart.Format("2006-01-02 15:04:05"),
		todayEnd.Format("2006-01-02 15:04:05"),
	)
	require.NoError(t, err)
	require.EqualValues(t, 1, exportedCount)
	requireQQCacheZipEntries(t, zipBytes, []string{"80002.ini", "账号.txt"})

	var storedBatch model.SysQQCacheExtractBatch
	require.NoError(t, global.GVA_DB.First(&storedBatch, batch.ID).Error)
	require.Equal(t, model.QQCacheExtractBatchStatusPendingSettlement, storedBatch.Status)
	require.EqualValues(t, 0, storedBatch.SettledCount)
	require.Nil(t, storedBatch.SettledAt)
	require.Nil(t, storedBatch.SettledBy)

	var records []model.SysQQCacheRecord
	require.NoError(t, global.GVA_DB.Where("extract_record_id = ?", batch.ID).Find(&records).Error)
	require.Len(t, records, 2)
	for _, record := range records {
		require.EqualValues(t, salesID, *record.Extractor)
		require.Nil(t, record.SalesSettledAt)
		require.Nil(t, record.SalesSettledBy)
	}
}

func requireQQCacheZipEntries(t *testing.T, zipBytes []byte, expected []string) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	require.NoError(t, err)
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	require.ElementsMatch(t, expected, names)
}
