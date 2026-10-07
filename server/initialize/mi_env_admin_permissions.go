package initialize

import (
	"strconv"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var miEnvAdminAPIs = []system.SysApi{
	{ApiGroup: "环境管理", Method: "POST", Path: "/miEnvAdmin/list", Description: "分页查询环境数据"},
	{ApiGroup: "环境管理", Method: "POST", Path: "/miEnvAdmin/types", Description: "查询设备环境类型"},
	{ApiGroup: "环境管理", Method: "POST", Path: "/miEnvAdmin/deleteSelected", Description: "删除所选环境数据"},
	{ApiGroup: "环境管理", Method: "POST", Path: "/miEnvAdmin/deleteAll", Description: "按条件删除全部环境数据"},
}

func ensureMIEnvAdminPermissions() error {
	if global.GVA_DB == nil {
		return nil
	}
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		menu := system.SysBaseMenu{
			MenuLevel: 0,
			Hidden:    false,
			ParentId:  0,
			Path:      "environment-manage",
			Name:      "miEnvManage",
			Component: "view/environment/miEnvManage.vue",
			Sort:      9,
			Meta:      system.Meta{Title: "环境管理", Icon: "collection"},
		}
		if err := tx.Where("name = ?", menu.Name).Attrs(menu).FirstOrCreate(&menu).Error; err != nil {
			return err
		}
		for _, api := range miEnvAdminAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return err
			}
		}
		for _, authorityID := range []uint{100, 888} {
			relation := system.SysAuthorityMenu{
				MenuId:      strconv.FormatUint(uint64(menu.ID), 10),
				AuthorityId: strconv.FormatUint(uint64(authorityID), 10),
			}
			if err := tx.Where(relation).FirstOrCreate(&relation).Error; err != nil {
				return err
			}
			for _, api := range miEnvAdminAPIs {
				rule := adapter.CasbinRule{Ptype: "p", V0: strconv.FormatUint(uint64(authorityID), 10), V1: api.Path, V2: api.Method}
				if err := tx.Where(rule).FirstOrCreate(&rule).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func logMIEnvAdminPermissionError(err error) {
	if err != nil && global.GVA_LOG != nil {
		global.GVA_LOG.Error("ensure MI environment admin permissions failed", zap.Error(err))
	}
}
