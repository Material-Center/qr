package initialize

import (
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnsureMIEnvAdminPermissionsIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&system.SysBaseMenu{}, &system.SysApi{}, &system.SysAuthorityMenu{}, &adapter.CasbinRule{}))
	previous := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previous })

	require.NoError(t, ensureMIEnvAdminPermissions())
	require.NoError(t, ensureMIEnvAdminPermissions())

	var menu system.SysBaseMenu
	require.NoError(t, db.Where("name = ?", "miEnvManage").First(&menu).Error)
	require.Equal(t, "view/environment/miEnvManage.vue", menu.Component)
	for _, role := range []string{"100", "888"} {
		var menuCount int64
		require.NoError(t, db.Model(&system.SysAuthorityMenu{}).
			Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", role, menu.ID).
			Count(&menuCount).Error)
		require.EqualValues(t, 1, menuCount)
		for _, api := range miEnvAdminAPIs {
			var apiCount, ruleCount int64
			require.NoError(t, db.Model(&system.SysApi{}).Where("path = ? AND method = ?", api.Path, api.Method).Count(&apiCount).Error)
			require.EqualValues(t, 1, apiCount)
			require.NoError(t, db.Model(&adapter.CasbinRule{}).
				Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", role, api.Path, api.Method).
				Count(&ruleCount).Error)
			require.EqualValues(t, 1, ruleCount)
		}
	}
}
