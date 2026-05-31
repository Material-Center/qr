package response

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

type QQCacheListResponse struct {
	List     []system.SysQQCacheRecord `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
	Stats    QQCacheExtractStats       `json:"stats"`
}

type QQCacheExtractStats struct {
	Pending          int64 `json:"pending"`
	Extracted        int64 `json:"extracted"`
	Total            int64 `json:"total"`
	BillingUnsettled int64 `json:"billingUnsettled"`
	BillingSettled   int64 `json:"billingSettled"`
}

type InternalToolQQCacheImportResponse struct {
	QQCacheRecordID uint   `json:"qqCacheRecordId"`
	QQNum           string `json:"qqNum"`
	Action          string `json:"action"`
	Force           bool   `json:"force"`
}

type InternalToolQQCacheImportItemResponse struct {
	FileName        string `json:"fileName"`
	QQCacheRecordID uint   `json:"qqCacheRecordId"`
	QQNum           string `json:"qqNum"`
	Action          string `json:"action"`
	Force           bool   `json:"force"`
	Success         bool   `json:"success"`
	Error           string `json:"error"`
}

type InternalToolQQCacheImportBatchResponse struct {
	Total   int                                     `json:"total"`
	Success int                                     `json:"success"`
	Failed  int                                     `json:"failed"`
	Results []InternalToolQQCacheImportItemResponse `json:"results"`
}

type InternalToolQQCacheExistsResponse struct {
	QQCacheRecordID uint   `json:"qqCacheRecordId"`
	QQNum           string `json:"qqNum"`
	Exists          bool   `json:"exists"`
}

type QQCacheBillingSettlementHistoryItem struct {
	SettledAt    time.Time `json:"settledAt"`
	SettledCount int64     `json:"settledCount"`
}

type QQCacheSalesSummary struct {
	Available      int64 `json:"available"`
	TodayExtracted int64 `json:"todayExtracted"`
	TodayUnsettled int64 `json:"todayUnsettled"`
	BillingSettled int64 `json:"billingSettled"`
	BillingTotal   int64 `json:"billingTotal"`
}

type QQCacheSalesHistoryItem struct {
	ID                   uint       `json:"id"`
	ExtractedAt          time.Time  `json:"extractedAt"`
	ExtractCount         int        `json:"extractCount"`
	SettledCount         int        `json:"settledCount"`
	SettlementStatus     string     `json:"settlementStatus"`
	SettlementStatusText string     `json:"settlementStatusText"`
	SettledAt            *time.Time `json:"settledAt"`
}

type QQCacheSalesHistoryResponse struct {
	List     []QQCacheSalesHistoryItem `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}

type QQCacheSalesAdminSummaryItem struct {
	ExtractorID      uint       `json:"extractorId"`
	ExtractorName    string     `json:"extractorName"`
	Username         string     `json:"username"`
	NickName         string     `json:"nickName"`
	ExtractedCount   int64      `json:"extractedCount"`
	SettledCount     int64      `json:"settledCount"`
	UnsettledCount   int64      `json:"unsettledCount"`
	LastExtractionAt *time.Time `json:"lastExtractionAt"`
}

type QQCacheSalesSettlementHistoryItem struct {
	SettledAt    time.Time `json:"settledAt"`
	SettledCount int64     `json:"settledCount"`
}
