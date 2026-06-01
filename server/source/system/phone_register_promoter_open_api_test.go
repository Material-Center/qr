package system

import (
	"context"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPhoneRegisterPromoterOpenAPIApiSeededWithoutCasbinPermissions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&sysModel.SysApi{}, &adapter.CasbinRule{}))

	ctx := context.WithValue(context.Background(), "db", db)
	_, err = (&initApi{}).InitializeData(ctx)
	require.NoError(t, err)
	_, err = (&initCasbin{}).InitializeData(ctx)
	require.NoError(t, err)

	requireSeededAPI(t, db, "GET", "/phoneRegisterTask/open-api/promoter/device-stats")
	requireSeededAPI(t, db, "POST", "/phoneRegisterTask/open-api/promoter/task")

	for _, role := range []string{"888", "300"} {
		requireNoSeededPermission(t, db, role, "GET", "/phoneRegisterTask/open-api/promoter/device-stats")
		requireNoSeededPermission(t, db, role, "POST", "/phoneRegisterTask/open-api/promoter/task")
	}
}

func TestAdminCanManageApiTokens(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&adapter.CasbinRule{},
		&sysModel.SysAuthority{},
		&sysModel.SysBaseMenu{},
		&sysModel.SysBaseMenuParameter{},
		&sysModel.SysBaseMenuBtn{},
	))

	ctx := context.WithValue(context.Background(), "db", db)
	ctx, err = (&initAuthority{}).InitializeData(ctx)
	require.NoError(t, err)
	ctx, err = (&initMenu{}).InitializeData(ctx)
	require.NoError(t, err)
	_, err = (&initMenuAuthority{}).InitializeData(ctx)
	require.NoError(t, err)
	_, err = (&initCasbin{}).InitializeData(ctx)
	require.NoError(t, err)

	requireSeededPermission(t, db, "100", "POST", "/sysApiToken/createApiToken")
	requireSeededPermission(t, db, "100", "POST", "/sysApiToken/getApiTokenList")
	requireSeededPermission(t, db, "100", "POST", "/sysApiToken/deleteApiToken")

	var admin sysModel.SysAuthority
	require.NoError(t, db.Preload("SysBaseMenus").
		Where("authority_id = ?", uint(100)).
		First(&admin).Error)
	menuNames := make(map[string]struct{}, len(admin.SysBaseMenus))
	for _, menu := range admin.SysBaseMenus {
		menuNames[menu.Name] = struct{}{}
	}
	_, hasAPITokenMenu := menuNames["apiToken"]
	require.False(t, hasAPITokenMenu)
}

func TestQQCacheSalesAPIsAndPermissionsAreSeeded(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&sysModel.SysApi{}, &adapter.CasbinRule{}))

	ctx := context.WithValue(context.Background(), "db", db)
	_, err = (&initApi{}).InitializeData(ctx)
	require.NoError(t, err)
	_, err = (&initCasbin{}).InitializeData(ctx)
	require.NoError(t, err)

	requireSeededAPI(t, db, "GET", "/qqCache/sales/summary")
	requireSeededAPI(t, db, "POST", "/qqCache/sales/extract")
	requireSeededAPI(t, db, "POST", "/qqCache/sales/history")
	requireSeededAPI(t, db, "GET", "/qqCache/sales/summaryList")
	requireSeededAPI(t, db, "GET", "/qqCache/sales/batches")
	requireSeededAPI(t, db, "POST", "/qqCache/sales/batch/download")
	requireSeededAPI(t, db, "POST", "/qqCache/sales/settle")

	requireSeededPermission(t, db, "600", "GET", "/qqCache/sales/summary")
	requireSeededPermission(t, db, "600", "POST", "/qqCache/sales/extract")
	requireSeededPermission(t, db, "600", "POST", "/qqCache/sales/history")
	requireNoSeededPermission(t, db, "600", "POST", "/qqCache/list")
	requireNoSeededPermission(t, db, "600", "POST", "/qqCache/sales/settle")
	requireNoSeededPermission(t, db, "600", "POST", "/qqCache/sales/batch/download")

	for _, role := range []string{"100", "888"} {
		requireSeededPermission(t, db, role, "GET", "/qqCache/sales/summaryList")
		requireSeededPermission(t, db, role, "GET", "/qqCache/sales/batches")
		requireSeededPermission(t, db, role, "POST", "/qqCache/sales/batch/download")
		requireSeededPermission(t, db, role, "POST", "/qqCache/sales/settle")
	}
}

func TestLegacySeededDataDoesNotForceFullReseedForSalesRole(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&sysModel.SysAuthority{},
		&sysModel.SysBaseMenu{},
		&sysModel.SysBaseMenuParameter{},
		&sysModel.SysBaseMenuBtn{},
	))

	require.NoError(t, db.Create(&[]sysModel.SysAuthority{
		{AuthorityId: 100, AuthorityName: "管理员"},
		{AuthorityId: 200, AuthorityName: "团长"},
		{AuthorityId: 300, AuthorityName: "地推"},
		{AuthorityId: 400, AuthorityName: "App提取"},
		{AuthorityId: 500, AuthorityName: "App上传"},
	}).Error)
	menus := []sysModel.SysBaseMenu{
		{Name: "dashboard", Path: "dashboard"},
		{Name: "about", Path: "about"},
		{Name: "state", Path: "state"},
	}
	require.NoError(t, db.Create(&menus).Error)
	for _, authorityID := range []uint{100, 200, 300, 400, 500} {
		var auth sysModel.SysAuthority
		require.NoError(t, db.Where("authority_id = ?", authorityID).First(&auth).Error)
		require.NoError(t, db.Model(&auth).Association("SysBaseMenus").Replace(menus))
	}

	ctx := context.WithValue(context.Background(), "db", db)
	require.True(t, (&initAuthority{}).DataInserted(ctx))
	require.True(t, (&initMenuAuthority{}).DataInserted(ctx))
}

func requireSeededAPI(t *testing.T, db *gorm.DB, method string, path string) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&sysModel.SysApi{}).
		Where("method = ? AND path = ?", method, path).
		Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func requireSeededPermission(t *testing.T, db *gorm.DB, role string, method string, path string) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&adapter.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", role, path, method).
		Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func requireNoSeededPermission(t *testing.T, db *gorm.DB, role string, method string, path string) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&adapter.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", role, path, method).
		Count(&count).Error)
	require.Zero(t, count)
}
