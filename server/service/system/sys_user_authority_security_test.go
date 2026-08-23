package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	modelSystemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSetUserAuthoritiesRejectsSelfPrivilegeEscalation(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	err := (&UserService{}).SetUserAuthorities(10, 200, 10, []uint{888})

	require.Error(t, err)
	require.Contains(t, err.Error(), "无权分配该角色")

	var user modelSystem.SysUser
	require.NoError(t, global.GVA_DB.First(&user, 10).Error)
	require.Equal(t, uint(200), user.AuthorityId)
}

func TestSetUserAuthoritiesRejectsManagingUnauthorizedTarget(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	err := (&UserService{}).SetUserAuthorities(10, 200, 11, []uint{300})

	require.Error(t, err)
	require.Contains(t, err.Error(), "无权操作该账号")
}

func TestDeleteUserRejectsManagingUnauthorizedTarget(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	err := (&UserService{}).DeleteUser(10, 200, 11)

	require.Error(t, err)
	require.Contains(t, err.Error(), "无权操作该账号")

	var count int64
	require.NoError(t, global.GVA_DB.Model(&modelSystem.SysUser{}).Where("id = ?", 11).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestDeleteUserRejectsDeletingOtherLeadersPromoter(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	err := (&UserService{}).DeleteUser(10, 200, 12)

	require.Error(t, err)
	require.Contains(t, err.Error(), "无权操作该账号")

	var count int64
	require.NoError(t, global.GVA_DB.Model(&modelSystem.SysUser{}).Where("id = ?", 12).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestDeleteUserAllowsLeaderDeletingOwnPromoter(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	err := (&UserService{}).DeleteUser(10, 200, 13)

	require.NoError(t, err)

	var count int64
	require.NoError(t, global.GVA_DB.Model(&modelSystem.SysUser{}).Where("id = ?", 13).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestGetUserInfoListLimitsLeaderToSelfAndOwnPromoters(t *testing.T) {
	setupUserAuthoritySecurityTestDB(t)

	list, total, err := (&UserService{}).GetUserInfoList(10, 200, modelSystemReq.GetUserList{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 10},
	}, false)

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	users := list.([]modelSystem.SysUser)
	require.Len(t, users, 2)
	for _, user := range users {
		require.True(t, user.ID == 10 || user.ID == 13)
	}
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
		&modelSystem.SysLoginLog{},
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
		{GVA_MODEL: global.GVA_MODEL{ID: 12}, Username: "other-promoter", AuthorityId: 300, LeaderID: uintPtr(99), Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 13}, Username: "own-promoter", AuthorityId: 300, LeaderID: uintPtr(10), Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 14}, Username: "other-leader", AuthorityId: 200, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 15}, Username: "own-admin", AuthorityId: 100, LeaderID: uintPtr(10), Enable: 1},
	}).Error)
	require.NoError(t, db.Create(&[]modelSystem.SysUserAuthority{
		{SysUserId: 10, SysAuthorityAuthorityId: 200},
		{SysUserId: 11, SysAuthorityAuthorityId: 100},
		{SysUserId: 12, SysAuthorityAuthorityId: 300},
		{SysUserId: 13, SysAuthorityAuthorityId: 300},
		{SysUserId: 14, SysAuthorityAuthorityId: 200},
		{SysUserId: 15, SysAuthorityAuthorityId: 100},
	}).Error)
}

func uintPtr(v uint) *uint {
	return &v
}
