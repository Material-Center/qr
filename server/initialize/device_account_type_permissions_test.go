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

func TestEnsureDeviceAccountTypePermissionsIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&system.SysBaseMenu{}, &system.SysApi{}, &system.SysAuthorityMenu{}, &adapter.CasbinRule{}))
	global.GVA_DB = db

	require.NoError(t, db.Create(&system.SysAuthorityMenu{MenuId: "999", AuthorityId: "600"}).Error)
	require.NoError(t, ensureDeviceAccountTypePermissions())
	require.NoError(t, ensureDeviceAccountTypePermissions())

	var menuCount int64
	require.NoError(t, db.Model(&system.SysBaseMenu{}).Where("name = ?", "deviceManage").Count(&menuCount).Error)
	require.EqualValues(t, 1, menuCount)

	for _, api := range deviceAccountTypeApis {
		var apiCount int64
		require.NoError(t, db.Model(&system.SysApi{}).Where("path = ? AND method = ?", api.Path, api.Method).Count(&apiCount).Error)
		require.EqualValues(t, 1, apiCount)
		for _, role := range []string{"100", "888"} {
			var ruleCount int64
			require.NoError(t, db.Model(&adapter.CasbinRule{}).
				Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", role, api.Path, api.Method).
				Count(&ruleCount).Error)
			require.EqualValues(t, 1, ruleCount)
		}
	}
	for _, expected := range []struct {
		path   string
		method string
	}{
		{path: "/deviceConfig/group/list", method: "GET"},
		{path: "/deviceConfig/group/save", method: "POST"},
		{path: "/deviceConfig/group/delete", method: "POST"},
	} {
		var apiCount int64
		require.NoError(t, db.Model(&system.SysApi{}).Where("path = ? AND method = ?", expected.path, expected.method).Count(&apiCount).Error)
		require.EqualValues(t, 1, apiCount)
		for _, role := range []string{"100", "888"} {
			var ruleCount int64
			require.NoError(t, db.Model(&adapter.CasbinRule{}).
				Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", role, expected.path, expected.method).
				Count(&ruleCount).Error)
			require.EqualValues(t, 1, ruleCount)
		}
	}

	var salesMenuCount int64
	require.NoError(t, db.Model(&system.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", "600").Count(&salesMenuCount).Error)
	require.EqualValues(t, 1, salesMenuCount)
}
