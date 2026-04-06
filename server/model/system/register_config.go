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
	NaichaAppID     string `json:"naichaAppId" gorm:"size:64;comment:奶茶平台appid(管理员配置)"`
	NaichaSecret    string `json:"naichaSecret" gorm:"size:256;comment:奶茶平台secret(管理员配置)"`
	NaichaCKMd5     string `json:"naichaCkMd5" gorm:"size:128;comment:奶茶平台cookie md5(管理员配置)"`
	IP138Token      string `json:"ip138Token" gorm:"size:256;comment:IP138 token(管理员配置)"`
	ApiBase         string `json:"apiBase" gorm:"size:256;comment:登录签名服务apiBase(管理员配置)"`
	ApiToken        string `json:"apiToken" gorm:"size:256;comment:登录签名服务apiToken(管理员配置)"`

	ProxyPlatform  string `json:"proxyPlatform" gorm:"size:32;comment:代理平台"`
	ProxyAccount   string `json:"proxyAccount" gorm:"size:128;comment:代理账号"`
	ProxyPassword  string `json:"proxyPassword" gorm:"size:128;comment:代理密码"`
	ProxySecretID  string `json:"proxySecretId" gorm:"size:128;comment:快代理SecretId"`
	ProxySecretKey string `json:"proxySecretKey" gorm:"size:128;comment:快代理SecretKey"`

	CaptchaPlatform string `json:"captchaPlatform" gorm:"size:32;comment:验证码平台"`
	CaptchaAccount  string `json:"captchaAccount" gorm:"size:128;comment:验证码账号"`
	CaptchaPassword string `json:"captchaPassword" gorm:"size:128;comment:验证码密码"`
	CaptchaToken    string `json:"captchaToken" gorm:"size:256;comment:验证码token"`
}
