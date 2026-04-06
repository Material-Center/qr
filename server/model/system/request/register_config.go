package request

type RegisterConfigUpsert struct {
	DefaultPassword string `json:"defaultPassword" form:"defaultPassword"`
	NaichaAppID     string `json:"naichaAppId" form:"naichaAppId"`
	NaichaSecret    string `json:"naichaSecret" form:"naichaSecret"`
	NaichaCKMd5     string `json:"naichaCkMd5" form:"naichaCkMd5"`
	IP138Token      string `json:"ip138Token" form:"ip138Token"`
	ApiBase         string `json:"apiBase" form:"apiBase"`
	ApiToken        string `json:"apiToken" form:"apiToken"`

	ProxyPlatform  string `json:"proxyPlatform" form:"proxyPlatform"`
	ProxyAccount   string `json:"proxyAccount" form:"proxyAccount"`
	ProxyPassword  string `json:"proxyPassword" form:"proxyPassword"`
	ProxySecretID  string `json:"proxySecretId" form:"proxySecretId"`
	ProxySecretKey string `json:"proxySecretKey" form:"proxySecretKey"`

	CaptchaPlatform string `json:"captchaPlatform" form:"captchaPlatform"`
	CaptchaAccount  string `json:"captchaAccount" form:"captchaAccount"`
	CaptchaPassword string `json:"captchaPassword" form:"captchaPassword"`
	CaptchaToken    string `json:"captchaToken" form:"captchaToken"`
}
