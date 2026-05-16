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

type InternalToolQQCacheExistsResponse struct {
	QQCacheRecordID uint   `json:"qqCacheRecordId"`
	QQNum           string `json:"qqNum"`
	Exists          bool   `json:"exists"`
}

type QQCacheBillingSettlementHistoryItem struct {
	SettledAt    time.Time `json:"settledAt"`
	SettledCount int64     `json:"settledCount"`
}
