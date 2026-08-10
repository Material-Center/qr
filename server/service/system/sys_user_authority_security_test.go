package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSetUserAuthoritiesRejectsSelfPrivilegeEscalation(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	err := (&UserService{}).SetUserAuthorities(200, 10, []uint{888})

	require.Error(t, err)
	require.Contains(t, err.Error(), "无权分配该角色")

	var user modelSystem.SysUser
	require.NoError(t, global.GVA_DB.First(&user, 10).Error)
	require.Equal(t, uint(200), user.AuthorityId)
}

func TestSetUserAuthoritiesRejectsManagingUnauthorizedTarget(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	err := (&UserService{}).SetUserAuthorities(200, 11, []uint{300})

	require.Error(t, err)
	require.Contains(t, err.Error(), "无权操作该账号")
}

func TestValidateAssignableAuthoritiesRejectsHiddenAuthorityIds(t *testing.T) {
	err := ValidateAssignableAuthorities(100, 200, []uint{200, 888})

	require.Error(t, err)
	require.Contains(t, err.Error(), "无权分配该角色")
}

func setupUserAuthoritySecurityTestDB(t *testing.T) {
	t.Helper()
	global.GVA_LOG = zap.NewNop()
	global.GVA_CONFIG.System.UseStrictAuth = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	global.GVA_DB = db
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysAuthority{},
		&modelSystem.SysUserAuthority{},
	))
	require.NoError(t, db.Create(&[]modelSystem.SysAuthority{
		{AuthorityId: 888, AuthorityName: "超级管理员"},
		{AuthorityId: 100, AuthorityName: "管理员"},
		{AuthorityId: 200, AuthorityName: "团长"},
		{AuthorityId: 300, AuthorityName: "地推"},
	}).Error)
	require.NoError(t, db.Create(&[]modelSystem.SysUser{
		{GVA_MODEL: global.GVA_MODEL{ID: 10}, Username: "leader", AuthorityId: 200, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 11}, Username: "admin", AuthorityId: 100, Enable: 1},
	}).Error)
	require.NoError(t, db.Create(&[]modelSystem.SysUserAuthority{
		{SysUserId: 10, SysAuthorityAuthorityId: 200},
		{SysUserId: 11, SysAuthorityAuthorityId: 100},
	}).Error)
}
