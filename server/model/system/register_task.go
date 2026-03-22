package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

const (
	RegisterTaskStepPhoneBind      = "phone_bind"
	RegisterTaskStepChangePassword = "change_password"
	RegisterTaskStepLogin          = "login"

	RegisterTaskFailCodeTimeout      = 1001
	RegisterTaskFailCodeNoQQBound    = 1002
	RegisterTaskFailCodeManualFailed = 1003
	RegisterTaskFailCodeMockBizFail  = 1004
	RegisterTaskFailCodeMockSysFail  = -1001
)

type SysRegisterTask struct {
	global.GVA_MODEL
	Phone            string     `json:"phone" gorm:"uniqueIndex:idx_register_task_phone;comment:手机号(唯一)"`
	QQAccount        string     `json:"qqAccount" gorm:"index;comment:QQ账号"`
	QQPassword       string     `json:"qqPassword" gorm:"comment:QQ密码"`
	QQProfile        string     `json:"qqProfile" gorm:"type:text;comment:QQ账号资料(JSON)"`
	LoginCacheINI    string     `json:"loginCacheIni" gorm:"type:longtext;comment:登录缓存ini"`
	CaptchaRandstr   string     `json:"-" gorm:"size:128;comment:流程滑块randstr"`
	CaptchaTicket    string     `json:"-" gorm:"size:512;comment:流程滑块ticket"`
	IsDaren          *bool      `json:"isDaren" gorm:"comment:是否达人"`
	StatusCode       *int       `json:"statusCode" gorm:"index;comment:状态码 >0业务失败 <0系统失败 ==0成功"`
	CurrentStep      string     `json:"currentStep" gorm:"index;comment:当前步骤"`
	LastError        string     `json:"lastError" gorm:"type:text;comment:失败原因"`
	RetryCount       int        `json:"retryCount" gorm:"default:0;comment:重试次数"`
	PromoterID       uint       `json:"promoterId" gorm:"index;comment:地推账号ID"`
	LeaderID         *uint      `json:"leaderId" gorm:"index;comment:团长账号ID"`
	ChangePasswordAt *time.Time `json:"changePasswordAt" gorm:"comment:改密时间"`
	LoginAt          *time.Time `json:"loginAt" gorm:"comment:登录时间"`
	FinishedAt       *time.Time `json:"finishedAt" gorm:"index;comment:任务完成时间"`
	ExpiresAt        time.Time  `json:"expiresAt" gorm:"index;comment:任务过期时间"`

	Promoter SysUser `json:"promoter" gorm:"foreignKey:PromoterID"`
	Leader   SysUser `json:"leader" gorm:"foreignKey:LeaderID"`
}
