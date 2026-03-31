package system

import (
	"errors"
	"strings"

	"github.com/Material-Center/qpi"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"gorm.io/gorm"
)

const (
	cfgRoleSuperAdmin = uint(888)
	cfgRoleAdmin      = uint(100)
	cfgRoleLeader     = uint(200)
)

type RegisterConfigService struct{}

func (s *RegisterConfigService) GetMyConfig(role uint, userID uint) (system.SysRegisterConfig, error) {
	ownerType, ownerID, err := getOwnerByRole(role, userID)
	if err != nil {
		return system.SysRegisterConfig{}, err
	}

	var cfg system.SysRegisterConfig
	findErr := global.GVA_DB.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).First(&cfg).Error
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		cfg.OwnerType = ownerType
		cfg.OwnerID = ownerID
		return cfg, nil
	}
	return cfg, findErr
}

func (s *RegisterConfigService) UpsertMyConfig(role uint, userID uint, req systemReq.RegisterConfigUpsert) (system.SysRegisterConfig, error) {
	ownerType, ownerID, err := getOwnerByRole(role, userID)
	if err != nil {
		return system.SysRegisterConfig{}, err
	}

	data := map[string]interface{}{}
	switch role {
	case cfgRoleSuperAdmin, cfgRoleAdmin:
		if req.DefaultPassword == "" {
			return system.SysRegisterConfig{}, errors.New("默认改密密码不能为空")
		}
		if strings.TrimSpace(req.NaichaAppID) == "" || strings.TrimSpace(req.NaichaSecret) == "" {
			return system.SysRegisterConfig{}, errors.New("奶茶平台 appId 和 secret 不能为空")
		}
		if strings.TrimSpace(req.ApiBase) == "" || strings.TrimSpace(req.ApiToken) == "" {
			return system.SysRegisterConfig{}, errors.New("登录签名服务 apiBase 和 apiToken 不能为空")
		}
		data["default_password"] = req.DefaultPassword
		data["naicha_app_id"] = req.NaichaAppID
		data["naicha_secret"] = req.NaichaSecret
		data["naicha_ck_md5"] = req.NaichaCKMd5
		data["api_base"] = req.ApiBase
		data["api_token"] = req.ApiToken
	case cfgRoleLeader:
		data["proxy_platform"] = req.ProxyPlatform
		data["proxy_account"] = req.ProxyAccount
		data["proxy_password"] = req.ProxyPassword
		data["captcha_platform"] = req.CaptchaPlatform
		data["captcha_account"] = req.CaptchaAccount
		data["captcha_password"] = req.CaptchaPassword
		data["captcha_token"] = req.CaptchaToken
	default:
		return system.SysRegisterConfig{}, errors.New("无权限修改配置")
	}

	var cfg system.SysRegisterConfig
	findErr := global.GVA_DB.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).First(&cfg).Error
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		cfg = system.SysRegisterConfig{
			OwnerType: ownerType,
			OwnerID:   ownerID,
		}
		switch role {
		case cfgRoleSuperAdmin, cfgRoleAdmin:
			cfg.DefaultPassword = req.DefaultPassword
			cfg.NaichaAppID = req.NaichaAppID
			cfg.NaichaSecret = req.NaichaSecret
			cfg.NaichaCKMd5 = req.NaichaCKMd5
			cfg.ApiBase = req.ApiBase
			cfg.ApiToken = req.ApiToken
		case cfgRoleLeader:
			cfg.ProxyPlatform = req.ProxyPlatform
			cfg.ProxyAccount = req.ProxyAccount
			cfg.ProxyPassword = req.ProxyPassword
			cfg.CaptchaPlatform = req.CaptchaPlatform
			cfg.CaptchaAccount = req.CaptchaAccount
			cfg.CaptchaPassword = req.CaptchaPassword
			cfg.CaptchaToken = req.CaptchaToken
		}
		if err = global.GVA_DB.Create(&cfg).Error; err != nil {
			return system.SysRegisterConfig{}, err
		}
		return cfg, nil
	}
	if findErr != nil {
		return system.SysRegisterConfig{}, findErr
	}

	if err = global.GVA_DB.Model(&cfg).Updates(data).Error; err != nil {
		return system.SysRegisterConfig{}, err
	}
	if err = global.GVA_DB.First(&cfg, cfg.ID).Error; err != nil {
		return system.SysRegisterConfig{}, err
	}
	return cfg, nil
}

func getOwnerByRole(role uint, userID uint) (ownerType string, ownerID uint, err error) {
	switch role {
	case cfgRoleSuperAdmin, cfgRoleAdmin:
		return system.RegisterConfigOwnerAdmin, 0, nil
	case cfgRoleLeader:
		return system.RegisterConfigOwnerLeader, userID, nil
	default:
		return "", 0, errors.New("无权限访问配置")
	}
}

func (s *RegisterConfigService) CheckMyConfig(role uint, userID uint) (map[string]interface{}, error) {
	cfgModel, err := s.GetMyConfig(role, userID)
	if err != nil {
		return nil, err
	}
	cfg := systemRegisterConfig{
		DefaultPassword: cfgModel.DefaultPassword,
		NaichaAppID:     cfgModel.NaichaAppID,
		NaichaSecret:    cfgModel.NaichaSecret,
		NaichaCKMd5:     cfgModel.NaichaCKMd5,
		ApiBase:         cfgModel.ApiBase,
		ApiToken:        cfgModel.ApiToken,
		ProxyPlatform:   cfgModel.ProxyPlatform,
		ProxyAccount:    cfgModel.ProxyAccount,
		ProxyPassword:   cfgModel.ProxyPassword,
		CaptchaPlatform: cfgModel.CaptchaPlatform,
		CaptchaAccount:  cfgModel.CaptchaAccount,
		CaptchaPassword: cfgModel.CaptchaPassword,
		CaptchaToken:    cfgModel.CaptchaToken,
	}

	result := map[string]interface{}{
		"proxy": map[string]interface{}{
			"enabled": strings.TrimSpace(cfg.ProxyPlatform) != "",
			"ok":      false,
			"message": "未配置",
		},
		"captcha": map[string]interface{}{
			"enabled": strings.TrimSpace(cfg.CaptchaPlatform) != "",
			"ok":      false,
			"message": "未配置",
		},
		"defaultPassword": map[string]interface{}{
			"enabled": role == cfgRoleSuperAdmin || role == cfgRoleAdmin,
			"ok":      strings.TrimSpace(cfg.DefaultPassword) != "",
			"message": "",
		},
		"naicha": map[string]interface{}{
			"enabled": role == cfgRoleSuperAdmin || role == cfgRoleAdmin,
			"ok":      strings.TrimSpace(cfg.NaichaAppID) != "" && strings.TrimSpace(cfg.NaichaSecret) != "",
			"message": "",
		},
		"qsign": map[string]interface{}{
			"enabled": role == cfgRoleSuperAdmin || role == cfgRoleAdmin,
			"ok":      strings.TrimSpace(cfg.ApiBase) != "" && strings.TrimSpace(cfg.ApiToken) != "",
			"message": "",
		},
	}

	if role == cfgRoleSuperAdmin || role == cfgRoleAdmin {
		if strings.TrimSpace(cfg.DefaultPassword) == "" {
			result["defaultPassword"] = map[string]interface{}{
				"enabled": true,
				"ok":      false,
				"message": "默认改密密码未配置",
			}
		} else {
			result["defaultPassword"] = map[string]interface{}{
				"enabled": true,
				"ok":      true,
				"message": "默认改密密码已配置",
			}
		}
		if strings.TrimSpace(cfg.NaichaAppID) == "" || strings.TrimSpace(cfg.NaichaSecret) == "" {
			result["naicha"] = map[string]interface{}{
				"enabled": true,
				"ok":      false,
				"message": "奶茶平台 appId/secret 未配置",
			}
		} else {
			result["naicha"] = map[string]interface{}{
				"enabled": true,
				"ok":      true,
				"message": "奶茶平台配置已就绪",
			}
		}
		if strings.TrimSpace(cfg.ApiBase) == "" || strings.TrimSpace(cfg.ApiToken) == "" {
			result["qsign"] = map[string]interface{}{
				"enabled": true,
				"ok":      false,
				"message": "登录签名服务 apiBase/apiToken 未配置",
			}
		} else {
			result["qsign"] = map[string]interface{}{
				"enabled": true,
				"ok":      true,
				"message": "登录签名服务配置已就绪",
			}
		}
	}

	if role == cfgRoleLeader {
		proxyResult := map[string]interface{}{
			"enabled": strings.TrimSpace(cfg.ProxyPlatform) != "",
			"ok":      false,
			"message": "未配置",
		}
		if strings.TrimSpace(cfg.ProxyPlatform) != "" {
			addr, pErr := allocateProxyURLFromConfig(cfg)
			if pErr != nil {
				proxyResult["message"] = pErr.Error()
			} else {
				proxyResult["ok"] = true
				proxyResult["message"] = "可用: " + addr
			}
		}
		result["proxy"] = proxyResult

		captchaResult := map[string]interface{}{
			"enabled": strings.TrimSpace(cfg.CaptchaPlatform) != "",
			"ok":      false,
			"message": "未配置",
		}
		if strings.TrimSpace(cfg.CaptchaPlatform) != "" {
			var token *captchaToken
			switch strings.ToLower(strings.TrimSpace(cfg.CaptchaPlatform)) {
			case captchaProviderYY:
				token, err = getCaptchaTokenFromYY(cfg, qpi.ChangePasswordAppID)
			case captchaProviderAC:
				token, err = getCaptchaTokenFromAC(cfg, qpi.ChangePasswordAppID)
			default:
				err = errors.New("不支持的验证码平台")
			}
			if err != nil {
				captchaResult["message"] = err.Error()
			} else {
				captchaResult["ok"] = true
				captchaResult["message"] = "可用: randstr=" + token.Randstr
			}
		}
		result["captcha"] = captchaResult
	}

	return result, nil
}
