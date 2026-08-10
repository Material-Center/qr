package initialize

import (
	"strconv"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var deviceAccountTypeApis = []system.SysApi{
	{ApiGroup: "设备管理", Method: "POST", Path: "/deviceConfig/list", Description: "分页查询设备配置"},
	{ApiGroup: "设备管理", Method: "POST", Path: "/deviceConfig/save", Description: "保存设备配置"},
	{ApiGroup: "设备管理", Method: "POST", Path: "/deviceConfig/delete", Description: "删除设备配置"},
	{ApiGroup: "QQ缓存", Method: "GET", Path: "/qqCache/accountTypes", Description: "查询QQ缓存账号类型"},
	{ApiGroup: "QQ缓存", Method: "GET", Path: "/qqCache/salesAllowedAccountTypes", Description: "读取销售可导出账号类型"},
	{ApiGroup: "QQ缓存", Method: "PUT", Path: "/qqCache/salesAllowedAccountTypes", Description: "保存销售可导出账号类型"},
}

func ensureDeviceAccountTypePermissions() error {
	if global.GVA_DB == nil {
		return nil
	}
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		menu := system.SysBaseMenu{
			MenuLevel: 0,
			Hidden:    false,
			ParentId:  0,
			Path:      "device-manage",
			Name:      "deviceManage",
			Component: "view/device/deviceConfig.vue",
			Sort:      8,
			Meta:      system.Meta{Title: "设备管理", Icon: "monitor"},
		}
		if err := tx.Where("name = ?", menu.Name).Attrs(menu).FirstOrCreate(&menu).Error; err != nil {
			return err
		}
		for _, api := range deviceAccountTypeApis {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return err
			}
		}
		for _, authorityID := range []uint{100, 888} {
			menuRelation := system.SysAuthorityMenu{
				MenuId:      strconv.FormatUint(uint64(menu.ID), 10),
				AuthorityId: strconv.FormatUint(uint64(authorityID), 10),
			}
			if err := tx.Where(menuRelation).FirstOrCreate(&menuRelation).Error; err != nil {
				return err
			}
			for _, api := range deviceAccountTypeApis {
				rule := adapter.CasbinRule{Ptype: "p", V0: strconv.FormatUint(uint64(authorityID), 10), V1: api.Path, V2: api.Method}
				if err := tx.Where(rule).FirstOrCreate(&rule).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func logDeviceAccountTypePermissionError(err error) {
	if err != nil && global.GVA_LOG != nil {
		global.GVA_LOG.Error("ensure device account type permissions failed", zap.Error(err))
	}
}
