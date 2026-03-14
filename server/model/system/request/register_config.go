package request

type RegisterConfigUpsert struct {
	DefaultPassword string `json:"defaultPassword" form:"defaultPassword"`

	ProxyPlatform string `json:"proxyPlatform" form:"proxyPlatform"`
	ProxyAccount  string `json:"proxyAccount" form:"proxyAccount"`
	ProxyPassword string `json:"proxyPassword" form:"proxyPassword"`

	CaptchaPlatform string `json:"captchaPlatform" form:"captchaPlatform"`
	CaptchaAccount  string `json:"captchaAccount" form:"captchaAccount"`
	CaptchaPassword string `json:"captchaPassword" form:"captchaPassword"`
	CaptchaToken    string `json:"captchaToken" form:"captchaToken"`
}
