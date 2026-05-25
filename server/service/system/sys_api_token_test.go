package system

import (
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	sysReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupApiTokenTestDB(t *testing.T) {
	t.Helper()
	global.GVA_CONFIG = config.Server{}
	global.GVA_CONFIG.JWT.SigningKey = "api-token-test-key"
	global.GVA_CONFIG.JWT.BufferTime = "1d"
	global.GVA_CONFIG.JWT.ExpiresTime = "7d"
	global.GVA_CONFIG.JWT.Issuer = "test"
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysAuthority{},
		&modelSystem.SysApiToken{},
		&modelSystem.JwtBlacklist{},
	))
	global.GVA_DB = db
}

func createApiTokenTestUser(t *testing.T, id uint, authorityID uint) {
	t.Helper()
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: id},
		UUID:        uuid.New(),
		Username:    "api-token-user",
		NickName:    "api token user",
		AuthorityId: authorityID,
		Enable:      1,
	}).Error)
}

func TestAdminCannotCreateApiTokenForSuperAdminUser(t *testing.T) {
	setupApiTokenTestDB(t)
	createApiTokenTestUser(t, 1, apiTokenRoleSuperAdmin)

	_, err := (&ApiTokenService{}).CreateApiTokenForOperator(apiTokenRoleAdmin, modelSystem.SysApiToken{
		UserID:      1,
		AuthorityID: apiTokenRoleSuperAdmin,
	}, 30)

	require.EqualError(t, err, "无权为该用户签发Token")
}

func TestAdminCanCreateApiTokenForPromoterUser(t *testing.T) {
	setupApiTokenTestDB(t)
	createApiTokenTestUser(t, 2, apiTokenRolePromoter)

	token, err := (&ApiTokenService{}).CreateApiTokenForOperator(apiTokenRoleAdmin, modelSystem.SysApiToken{
		UserID:      2,
		AuthorityID: apiTokenRolePromoter,
	}, 30)

	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestAdminCannotCreateApiTokenForLeaderUser(t *testing.T) {
	setupApiTokenTestDB(t)
	createApiTokenTestUser(t, 2, apiTokenRoleLeader)

	_, err := (&ApiTokenService{}).CreateApiTokenForOperator(apiTokenRoleAdmin, modelSystem.SysApiToken{
		UserID:      2,
		AuthorityID: apiTokenRoleLeader,
	}, 30)

	require.EqualError(t, err, "仅支持为地推账号签发OpenAPI Token")
}

func TestAdminCreatedApiTokenIsAlwaysNinetyDays(t *testing.T) {
	setupApiTokenTestDB(t)
	createApiTokenTestUser(t, 2, apiTokenRolePromoter)

	_, err := (&ApiTokenService{}).CreateApiTokenForOperator(apiTokenRoleAdmin, modelSystem.SysApiToken{
		UserID:      2,
		AuthorityID: apiTokenRolePromoter,
	}, -1)
	require.NoError(t, err)

	var token modelSystem.SysApiToken
	require.NoError(t, global.GVA_DB.First(&token).Error)
	ttl := time.Until(token.ExpiresAt)
	require.Greater(t, ttl, 89*24*time.Hour)
	require.Less(t, ttl, 91*24*time.Hour)
}

func TestAdminApiTokenListExcludesUnmanageableRoles(t *testing.T) {
	setupApiTokenTestDB(t)
	createApiTokenTestUser(t, 1, apiTokenRoleSuperAdmin)
	createApiTokenTestUser(t, 2, apiTokenRolePromoter)
	require.NoError(t, global.GVA_DB.Create(&[]modelSystem.SysApiToken{
		{UserID: 1, AuthorityID: apiTokenRoleSuperAdmin, Token: "super-token", Status: true, ExpiresAt: time.Now().Add(time.Hour)},
		{UserID: 2, AuthorityID: apiTokenRolePromoter, Token: "promoter-token", Status: true, ExpiresAt: time.Now().Add(time.Hour)},
	}).Error)

	list, total, err := (&ApiTokenService{}).GetApiTokenListForOperator(apiTokenRoleAdmin, sysReq.SysApiTokenSearch{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 10},
	})

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, "promoter-token", list[0].Token)
}
