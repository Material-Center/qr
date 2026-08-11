package system

import (
	"errors"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DeviceConfigService struct{}

func (s *DeviceConfigService) ResolveAccountType(deviceID string) string {
	return s.ResolveAccountTypeTx(global.GVA_DB, deviceID)
}

func (s *DeviceConfigService) ResolveAccountTypeTx(db *gorm.DB, deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || db == nil {
		return AccountTypeDefault
	}
	var config system.SysDeviceConfig
	if err := db.Where("device_id = ?", deviceID).First(&config).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) && global.GVA_LOG != nil {
			global.GVA_LOG.Warn("resolve device account type failed", zap.String("deviceID", deviceID), zap.Error(err))
		}
		return AccountTypeDefault
	}
	accountType := NormalizeQQCacheAccountType(config.AccountType)
	if accountType != strings.TrimSpace(config.AccountType) && global.GVA_LOG != nil {
		global.GVA_LOG.Warn("device account type invalid, fallback to default", zap.String("deviceID", deviceID), zap.String("accountType", config.AccountType))
	}
	return accountType
}

func (s *DeviceConfigService) ListDeviceConfig(req systemReq.DeviceConfigList) ([]system.SysDeviceConfig, int64, error) {
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 10
	}
	db := global.GVA_DB.Model(&system.SysDeviceConfig{})
	if deviceID := strings.TrimSpace(req.DeviceID); deviceID != "" {
		db = db.Where("device_id LIKE ?", "%"+deviceID+"%")
	}
	if accountType := strings.TrimSpace(req.AccountType); accountType != "" {
		if err := ValidateQQCacheAccountType(accountType); err != nil {
			return nil, 0, err
		}
		db = db.Where("account_type = ?", accountType)
	}
	if req.GroupID != nil && *req.GroupID != 0 {
		db = db.Where("group_id = ?", *req.GroupID)
	} else if req.Ungrouped {
		db = db.Where("group_id IS NULL OR group_id = 0")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []system.SysDeviceConfig
	err := db.Preload("Group").Order("updated_at desc").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *DeviceConfigService) SaveDeviceConfig(req systemReq.DeviceConfigSave) (system.SysDeviceConfig, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	accountType := strings.TrimSpace(req.AccountType)
	if deviceID == "" {
		return system.SysDeviceConfig{}, errors.New("设备ID不能为空")
	}
	if accountType == "" {
		accountType = AccountTypeDefault
	}
	if err := ValidateQQCacheAccountType(accountType); err != nil {
		return system.SysDeviceConfig{}, err
	}
	groupID := normalizeDeviceGroupID(req.GroupID)
	remark := strings.TrimSpace(req.Remark)

	var saved system.SysDeviceConfig
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := s.ensureDeviceGroupExists(tx, groupID); err != nil {
			return err
		}
		if req.ID != 0 {
			var existing system.SysDeviceConfig
			if err := tx.Where("id = ?", req.ID).First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("设备配置不存在")
				}
				return err
			}
			if existing.DeviceID != deviceID {
				return errors.New("设备ID不可修改")
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"account_type": accountType,
				"group_id":     groupID,
				"remark":       remark,
				"updated_at":   time.Now(),
			}).Error; err != nil {
				return err
			}
			return tx.Preload("Group").Where("id = ?", existing.ID).First(&saved).Error
		}

		var existing system.SysDeviceConfig
		err := tx.Unscoped().Where("device_id = ?", deviceID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			if !existing.DeletedAt.Valid {
				return errors.New("设备已存在")
			}
			if err := tx.Unscoped().Model(&existing).Updates(map[string]any{
				"account_type": accountType,
				"group_id":     groupID,
				"remark":       remark,
				"deleted_at":   nil,
				"updated_at":   time.Now(),
			}).Error; err != nil {
				return err
			}
			return tx.Preload("Group").Where("id = ?", existing.ID).First(&saved).Error
		}
		saved = system.SysDeviceConfig{
			DeviceID:    deviceID,
			AccountType: accountType,
			GroupID:     groupID,
			Remark:      remark,
		}
		if err := tx.Create(&saved).Error; err != nil {
			return err
		}
		return tx.Preload("Group").Where("id = ?", saved.ID).First(&saved).Error
	})
	return saved, err
}

func (s *DeviceConfigService) DeleteDeviceConfig(req systemReq.DeviceConfigDelete) error {
	if req.ID == 0 {
		return errors.New("设备配置ID不能为空")
	}
	return global.GVA_DB.Delete(&system.SysDeviceConfig{}, "id = ?", req.ID).Error
}

func (s *DeviceConfigService) ListDeviceGroups() ([]system.SysDeviceGroup, error) {
	var list []system.SysDeviceGroup
	err := global.GVA_DB.Order("updated_at desc").Order("id desc").Find(&list).Error
	return list, err
}

func (s *DeviceConfigService) SaveDeviceGroup(req systemReq.DeviceGroupSave) (system.SysDeviceGroup, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return system.SysDeviceGroup{}, errors.New("分组名称不能为空")
	}
	remark := strings.TrimSpace(req.Remark)

	var saved system.SysDeviceGroup
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if req.ID != 0 {
			var existing system.SysDeviceGroup
			if err := tx.Where("id = ?", req.ID).First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("设备分组不存在")
				}
				return err
			}
			if err := s.ensureDeviceGroupNameAvailable(tx, name, req.ID); err != nil {
				return err
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"name":       name,
				"remark":     remark,
				"updated_at": time.Now(),
			}).Error; err != nil {
				return err
			}
			return tx.Where("id = ?", existing.ID).First(&saved).Error
		}

		var existing system.SysDeviceGroup
		err := tx.Unscoped().Where("name = ?", name).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			if !existing.DeletedAt.Valid {
				return errors.New("设备分组已存在")
			}
			if err := tx.Unscoped().Model(&existing).Updates(map[string]any{
				"remark":     remark,
				"deleted_at": nil,
				"updated_at": time.Now(),
			}).Error; err != nil {
				return err
			}
			return tx.Where("id = ?", existing.ID).First(&saved).Error
		}
		saved = system.SysDeviceGroup{Name: name, Remark: remark}
		return tx.Create(&saved).Error
	})
	return saved, err
}

func (s *DeviceConfigService) DeleteDeviceGroup(req systemReq.DeviceGroupDelete) error {
	if req.ID == 0 {
		return errors.New("设备分组ID不能为空")
	}
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var group system.SysDeviceGroup
		if err := tx.Where("id = ?", req.ID).First(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("设备分组不存在")
			}
			return err
		}
		var count int64
		if err := tx.Model(&system.SysDeviceConfig{}).Where("group_id = ?", req.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("存在关联设备，不能删除分组")
		}
		return tx.Delete(&system.SysDeviceGroup{}, "id = ?", req.ID).Error
	})
}

func normalizeDeviceGroupID(groupID *uint) *uint {
	if groupID == nil || *groupID == 0 {
		return nil
	}
	return groupID
}

func (s *DeviceConfigService) ensureDeviceGroupExists(tx *gorm.DB, groupID *uint) error {
	if groupID == nil {
		return nil
	}
	var count int64
	if err := tx.Model(&system.SysDeviceGroup{}).Where("id = ?", *groupID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("设备分组不存在")
	}
	return nil
}

func (s *DeviceConfigService) ensureDeviceGroupNameAvailable(tx *gorm.DB, name string, currentID uint) error {
	var count int64
	if err := tx.Unscoped().Model(&system.SysDeviceGroup{}).Where("name = ? AND id <> ?", name, currentID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("设备分组已存在")
	}
	return nil
}
