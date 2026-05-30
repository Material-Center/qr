package system

import (
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
	))
	global.GVA_DB = db
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

	summaries, err := (&QQCacheService{}).ListSalesSummaryForAdmin()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.EqualValues(t, salesID, summaries[0].ExtractorID)
	require.EqualValues(t, 2, summaries[0].ExtractedCount)
	require.EqualValues(t, 2, summaries[0].SettledCount)
	require.EqualValues(t, 0, summaries[0].UnsettledCount)
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
