package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

const (
	QQCacheExtractBatchStatusPendingSettlement = "pending_settlement"
	QQCacheExtractBatchStatusSettled           = "settled"
)

// SysQQCacheExtractBatch 销售QQ缓存提取批次
type SysQQCacheExtractBatch struct {
	global.GVA_MODEL
	ExtractorID   uint       `json:"extractorId" gorm:"column:extractor_id;index;comment:销售用户ID"`
	ExtractorName string     `json:"extractorName" gorm:"column:extractor_name;size:128;comment:销售名称快照"`
	ExtractCount  int        `json:"extractCount" gorm:"column:extract_count;comment:本次提取账号数量"`
	SettledCount  int        `json:"settledCount" gorm:"column:settled_count;comment:已结算账号数量"`
	Status        string     `json:"status" gorm:"column:status;size:32;index;comment:结算状态"`
	ExtractedAt   time.Time  `json:"extractedAt" gorm:"column:extracted_at;index;comment:提取时间"`
	SettledAt     *time.Time `json:"settledAt" gorm:"column:settled_at;index;comment:整批结算时间"`
	SettledBy     *uint      `json:"settledBy" gorm:"column:settled_by;index;comment:结算管理员ID"`
}

func (SysQQCacheExtractBatch) TableName() string {
	return "sys_qq_cache_extract_batches"
}
