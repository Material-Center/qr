package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// SysQQCacheRecord QQ缓存记录（新系统独立表）
type SysQQCacheRecord struct {
	global.GVA_MODEL
	Phone           *string    `json:"phone" gorm:"column:phone;size:20;comment:手机号"`
	QQNum           string     `json:"qqNum" gorm:"column:qq_num;uniqueIndex:uk_qq_cache_qq_num;comment:QQ账号"`
	QQPwd           string     `json:"qqPwd" gorm:"column:qq_pwd;comment:QQ密码"`
	Extractor       *uint      `json:"extractor" gorm:"column:extractor;index;comment:提取者用户ID"`
	ExtractRecordID *uint      `json:"extractRecordId" gorm:"column:extract_record_id;comment:提取记录ID"`
	ExtractionAt    *time.Time `json:"extractionAt" gorm:"column:extraction_at;comment:提取时间"`
	INI             *string    `json:"iNI" gorm:"column:ini;type:longtext;comment:缓存INI内容"`
	DeviceID        *string    `json:"deviceId" gorm:"column:device_id;size:128;comment:设备ID"`
}
