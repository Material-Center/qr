package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDeviceConfigTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SysDeviceConfig{}))
	global.GVA_DB = db
}

func TestDeviceConfigResolveAccountTypeDefaultsWhenMissing(t *testing.T) {
	setupDeviceConfigTestDB(t)

	require.Equal(t, AccountTypeDefault, (&DeviceConfigService{}).ResolveAccountType(""))
	require.Equal(t, AccountTypeDefault, (&DeviceConfigService{}).ResolveAccountType("missing-device"))
}

func TestDeviceConfigSaveAndResolvePCAccountType(t *testing.T) {
	setupDeviceConfigTestDB(t)

	record, err := (&DeviceConfigService{}).SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    " device-001 ",
		AccountType: AccountTypePC,
		Remark:      "PC设备",
	})
	require.NoError(t, err)
	require.Equal(t, "device-001", record.DeviceID)
	require.Equal(t, AccountTypePC, record.AccountType)

	require.Equal(t, AccountTypePC, (&DeviceConfigService{}).ResolveAccountType("device-001"))
}

func TestDeviceConfigSaveRestoresSoftDeletedRecord(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	record, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "device-restore",
		AccountType: AccountTypePC,
	})
	require.NoError(t, err)
	require.NoError(t, service.DeleteDeviceConfig(systemReq.DeviceConfigDelete{ID: record.ID}))
	require.Equal(t, AccountTypeDefault, service.ResolveAccountType("device-restore"))

	restored, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "device-restore",
		AccountType: AccountTypeDefault,
		Remark:      "restored",
	})
	require.NoError(t, err)
	require.Equal(t, record.ID, restored.ID)
	require.Equal(t, AccountTypeDefault, restored.AccountType)
	require.Equal(t, "restored", restored.Remark)
}

func TestDeviceConfigUpdateRejectsDeviceIDChange(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	record, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "device-old",
		AccountType: AccountTypeDefault,
	})
	require.NoError(t, err)

	_, err = service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		ID:          record.ID,
		DeviceID:    "device-new",
		AccountType: AccountTypePC,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "设备ID不可修改")
}

func TestDeviceConfigListFilters(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	_, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{DeviceID: "pc-001", AccountType: AccountTypePC})
	require.NoError(t, err)
	_, err = service.SaveDeviceConfig(systemReq.DeviceConfigSave{DeviceID: "app-001", AccountType: AccountTypeDefault})
	require.NoError(t, err)

	list, total, err := service.ListDeviceConfig(systemReq.DeviceConfigList{
		PageInfo:    commonReq.PageInfo{Page: 1, PageSize: 10},
		DeviceID:    "pc",
		AccountType: AccountTypePC,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, "pc-001", list[0].DeviceID)
}
