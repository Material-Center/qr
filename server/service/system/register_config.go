package system

import (
	"errors"

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
		data["default_password"] = req.DefaultPassword
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
