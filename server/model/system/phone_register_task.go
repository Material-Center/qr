package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

const (
	PhoneRegisterSMSModePlatformSend = "PLATFORM_SEND"
	PhoneRegisterSMSModeUserSentToTX = "USER_SENT_TO_TX"

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
	QQNum           string     `json:"qqNum" gorm:"index;size:32;comment:注册成功QQ号"`
	QQCacheRecordID *uint      `json:"qqCacheRecordId" gorm:"index;comment:QQ缓存记录ID"`
	PendingCode     string     `json:"-" gorm:"column:pending_code;size:32;comment:待设备消费验证码"`
	Status          string     `json:"status" gorm:"index;size:32;comment:任务状态"`
	StatusCode      *int       `json:"statusCode" gorm:"index;comment:任务状态码"`
	LastError       string     `json:"lastError" gorm:"type:text;comment:最近错误摘要"`
	FinishedAt      *time.Time `json:"finishedAt" gorm:"index;comment:完成时间"`
	HolderDeviceID  *string    `json:"holderDeviceId" gorm:"index;size:128;comment:当前持有设备ID"`
	ClaimedAt       *time.Time `json:"claimedAt" gorm:"comment:领取时间"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt" gorm:"index;comment:最后心跳时间"`
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
