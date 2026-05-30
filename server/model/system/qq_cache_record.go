package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// SysQQCacheRecord QQ缓存记录（新系统独立表）
type SysQQCacheRecord struct {
	global.GVA_MODEL
	Phone            *string    `json:"phone" gorm:"column:phone;size:20;comment:手机号"`
	QQNum            string     `json:"qqNum" gorm:"column:qq_num;uniqueIndex:uk_qq_cache_qq_num;comment:QQ账号"`
	QQPwd            string     `json:"qqPwd" gorm:"column:qq_pwd;comment:QQ密码"`
	ClientVersion    string     `json:"clientVersion" gorm:"column:client_version;size:64;index;comment:客户端版本号"`
	Extractor        *uint      `json:"extractor" gorm:"column:extractor;index;comment:提取者用户ID"`
	ExtractRecordID  *uint      `json:"extractRecordId" gorm:"column:extract_record_id;comment:提取记录ID"`
	ExtractionAt     *time.Time `json:"extractionAt" gorm:"column:extraction_at;comment:提取时间"`
	INI              *string    `json:"iNI" gorm:"column:ini;type:longtext;comment:缓存INI内容"`
	DeviceID         *string    `json:"deviceId" gorm:"column:device_id;size:128;comment:设备ID"`
	BillingSettledAt *time.Time `json:"billingSettledAt" gorm:"column:billing_settled_at;index;comment:计费结算时间"`
	BillingSettledBy *uint      `json:"billingSettledBy" gorm:"column:billing_settled_by;index;comment:计费结算管理员ID"`
	SalesSettledAt   *time.Time `json:"salesSettledAt" gorm:"column:sales_settled_at;index;comment:销售结算时间"`
	SalesSettledBy   *uint      `json:"salesSettledBy" gorm:"column:sales_settled_by;index;comment:销售结算管理员ID"`
}
