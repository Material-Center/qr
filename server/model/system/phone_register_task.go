package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

const (
	PhoneRegisterSMSModePlatformSend = "PLATFORM_SEND"
	PhoneRegisterSMSModeUserSentToTX = "USER_SENT_TO_TX"

	PhoneRegisterTaskCreateSourceManual  = "MANUAL"
	PhoneRegisterTaskCreateSourceOpenAPI = "OPENAPI"

	PhoneRegisterTaskSourceScript  = "SCRIPT"
	PhoneRegisterTaskSourceOpenAPI = "OPENAPI"

	PhoneRegisterCacheStatusPending  = "pending"
	PhoneRegisterCacheStatusUploaded = "uploaded"
	PhoneRegisterCacheStatusTimeout  = "timeout"

	PhoneRegisterStatusPending              = "pending"
	PhoneRegisterStatusRunning              = "running"
	PhoneRegisterStatusWaitingPromoterCode  = "waiting_promoter_code"
	PhoneRegisterStatusRegisteredWaitUpload = "registered_wait_upload"
	PhoneRegisterStatusSucceeded            = "succeeded"
	PhoneRegisterStatusFailed               = "failed"

	PhoneRegisterStatusCodeSucceeded               = 0
	PhoneRegisterStatusCodeDeviceExecFail          = 1001
	PhoneRegisterStatusCodeVerifyCodeTimeout       = 1002
	PhoneRegisterStatusCodeHeartbeatTimeout        = 1003
	PhoneRegisterStatusCodeTaskTimeout             = 1004
	PhoneRegisterStatusCodeCacheUploadFail         = 1005
	PhoneRegisterStatusCodeCacheToolReadFailed     = 1006
	PhoneRegisterStatusCodeRecoverTimeout          = 1007
	PhoneRegisterStatusCodePromoterManualTerminate = 1008
	PhoneRegisterStatusCodeOpenAPIFeedback         = 1009
	PhoneRegisterStatusCodeRiskFace                = 1010
	PhoneRegisterStatusCodeRiskQuota               = 1011
	PhoneRegisterStatusCodeUnknown                 = 1099

	PhoneRegisterDeviceActionEnterWaitingCode = "enter_waiting_code"
	PhoneRegisterDeviceActionConsumeCodeOK    = "consume_code_success"
	PhoneRegisterDeviceActionRegisterSuccess  = "register_success"
	PhoneRegisterDeviceActionFail             = "fail"
)

type SysPhoneRegisterTask struct {
	global.GVA_MODEL
	Phone           string     `json:"phone" gorm:"index;size:20;comment:手机号"`
	PromoterID      uint       `json:"promoterId" gorm:"index;comment:地推账号ID"`
	LeaderID        *uint      `json:"leaderId" gorm:"index;comment:团长账号ID"`
	SMSReceiveMode  string     `json:"smsReceiveMode" gorm:"index;size:32;comment:收码方式"`
	CreateSource    string     `json:"createSource" gorm:"index;size:32;comment:任务创建来源"`
	TaskSource      string     `json:"taskSource" gorm:"index;size:32;comment:任务执行来源"`
	CacheStatus     string     `json:"cacheStatus" gorm:"index;size:32;comment:缓存处理状态"`
	QQNum           string     `json:"qqNum" gorm:"index;size:32;comment:注册成功QQ号"`
	QQCacheRecordID *uint      `json:"qqCacheRecordId" gorm:"index;comment:QQ缓存记录ID"`
	PendingCode     string     `json:"-" gorm:"column:pending_code;size:32;comment:待设备消费验证码"`
	CodeRequestedAt *time.Time `json:"codeRequestedAt" gorm:"index;comment:进入待地推验证码时间"`
	Status          string     `json:"status" gorm:"index;size:32;comment:任务状态"`
	StatusCode      *int       `json:"statusCode" gorm:"index;comment:任务状态码"`
	LastError       string     `json:"lastError" gorm:"type:text;comment:最近错误摘要"`
	FinishedAt      *time.Time `json:"finishedAt" gorm:"index;comment:完成时间"`
	SettledAt       *time.Time `json:"settledAt" gorm:"index;comment:结算时间"`
	SettledBy       *uint      `json:"settledBy" gorm:"index;comment:结算管理员ID"`
	HolderDeviceID  *string    `json:"holderDeviceId" gorm:"index;size:128;comment:当前持有设备ID"`
	ClaimedAt       *time.Time `json:"claimedAt" gorm:"comment:领取时间"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt" gorm:"index;comment:最后心跳时间"`
	AvailableAt     *time.Time `json:"availableAt" gorm:"index;comment:可领取时间"`
	ExpiresAt       time.Time  `json:"expiresAt" gorm:"index;comment:任务过期时间"`
	RetryCount      int        `json:"retryCount" gorm:"default:0;comment:重试次数"`

	Promoter      SysUser          `json:"promoter" gorm:"foreignKey:PromoterID"`
	Leader        SysUser          `json:"leader" gorm:"foreignKey:LeaderID"`
	QQCacheRecord SysQQCacheRecord `json:"qqCacheRecord" gorm:"foreignKey:QQCacheRecordID"`
}

type SysPhoneRegisterTaskLog struct {
	global.GVA_MODEL
	TaskID     uint       `json:"taskId" gorm:"index;comment:手机号注册任务ID"`
	DeviceID   string     `json:"deviceId" gorm:"index;size:128;comment:设备ID"`
	ClientTime *time.Time `json:"clientTime" gorm:"index;comment:客户端上报时间"`
	Message    string     `json:"message" gorm:"type:text;comment:日志内容"`

	Task SysPhoneRegisterTask `json:"task" gorm:"foreignKey:TaskID"`
}

type SysPhoneRegisterRiskDailyStat struct {
	global.GVA_MODEL
	PromoterID         uint   `json:"promoterId" gorm:"uniqueIndex:uk_phone_register_risk_day;index;comment:地推账号ID"`
	BizDate            string `json:"bizDate" gorm:"uniqueIndex:uk_phone_register_risk_day;size:10;comment:业务日期YYYY-MM-DD"`
	SuccessReportCount int64  `json:"successReportCount" gorm:"comment:当天成功上报口径总数"`
	RiskFailCount      int64  `json:"riskFailCount" gorm:"comment:当天风控失败数"`
	LastRiskSuccessSeq int64  `json:"lastRiskSuccessSeq" gorm:"comment:最近一次风控所在成功上报序号"`
	LastRiskReason     string `json:"lastRiskReason" gorm:"size:32;comment:最近一次风控原因"`
	LastRiskGap        int64  `json:"lastRiskGap" gorm:"comment:最近一次风控间隔"`
	PreviousRiskGap    int64  `json:"previousRiskGap" gorm:"comment:上上次风控间隔"`
	PreviousRiskReason string `json:"previousRiskReason" gorm:"size:32;comment:上上次风控原因"`
}
