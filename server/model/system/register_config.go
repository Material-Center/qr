package system

import "github.com/flipped-aurora/gin-vue-admin/server/global"

const (
	RegisterConfigOwnerAdmin  = "admin"
	RegisterConfigOwnerLeader = "leader"
)

type SysRegisterConfig struct {
	global.GVA_MODEL
	OwnerType string `json:"ownerType" gorm:"index;size:20;comment:配置归属类型 admin/leader"`
	OwnerID   uint   `json:"ownerId" gorm:"index;comment:归属ID，admin为0，leader为用户ID"`

	DefaultPassword string `json:"defaultPassword" gorm:"size:128;comment:默认改密密码(管理员配置)"`

	ProxyPlatform string `json:"proxyPlatform" gorm:"size:32;comment:代理平台"`
	ProxyAccount  string `json:"proxyAccount" gorm:"size:128;comment:代理账号"`
	ProxyPassword string `json:"proxyPassword" gorm:"size:128;comment:代理密码"`

	CaptchaPlatform string `json:"captchaPlatform" gorm:"size:32;comment:验证码平台"`
	CaptchaAccount  string `json:"captchaAccount" gorm:"size:128;comment:验证码账号"`
	CaptchaPassword string `json:"captchaPassword" gorm:"size:128;comment:验证码密码"`
	CaptchaToken    string `json:"captchaToken" gorm:"size:256;comment:验证码token"`
}
