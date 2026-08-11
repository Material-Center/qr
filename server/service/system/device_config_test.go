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
	require.NoError(t, db.AutoMigrate(&model.SysDeviceGroup{}, &model.SysDeviceConfig{}))
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

func TestDeviceGroupSaveListAndAssignDevice(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	group, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{
		Name:   " PC设备 ",
		Remark: "pc group",
	})
	require.NoError(t, err)
	require.Equal(t, "PC设备", group.Name)
	require.Equal(t, "pc group", group.Remark)

	device, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "pc-group-001",
		AccountType: AccountTypePC,
		GroupID:     &group.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, device.GroupID)
	require.Equal(t, group.ID, *device.GroupID)
	require.Equal(t, "PC设备", device.Group.Name)

	groups, err := service.ListDeviceGroups()
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "PC设备", groups[0].Name)
}

func TestDeviceConfigListFiltersByGroupAndUngrouped(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	group, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{Name: "A组"})
	require.NoError(t, err)
	_, err = service.SaveDeviceConfig(systemReq.DeviceConfigSave{DeviceID: "grouped-001", AccountType: AccountTypePC, GroupID: &group.ID})
	require.NoError(t, err)
	_, err = service.SaveDeviceConfig(systemReq.DeviceConfigSave{DeviceID: "ungrouped-001", AccountType: AccountTypeDefault})
	require.NoError(t, err)

	grouped, total, err := service.ListDeviceConfig(systemReq.DeviceConfigList{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 10},
		GroupID:  &group.ID,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, "grouped-001", grouped[0].DeviceID)
	require.Equal(t, "A组", grouped[0].Group.Name)

	ungrouped, total, err := service.ListDeviceConfig(systemReq.DeviceConfigList{
		PageInfo:  commonReq.PageInfo{Page: 1, PageSize: 10},
		Ungrouped: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, "ungrouped-001", ungrouped[0].DeviceID)
	require.Nil(t, ungrouped[0].GroupID)
}

func TestDeviceGroupDeleteRejectsReferencedGroup(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	group, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{Name: "不可删除"})
	require.NoError(t, err)
	_, err = service.SaveDeviceConfig(systemReq.DeviceConfigSave{DeviceID: "device-ref", AccountType: AccountTypeDefault, GroupID: &group.ID})
	require.NoError(t, err)

	err = service.DeleteDeviceGroup(systemReq.DeviceGroupDelete{ID: group.ID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "存在关联设备")
}

func TestDeviceGroupDeleteRejectsMissingGroup(t *testing.T) {
	setupDeviceConfigTestDB(t)

	err := (&DeviceConfigService{}).DeleteDeviceGroup(systemReq.DeviceGroupDelete{ID: 999})
	require.Error(t, err)
	require.Contains(t, err.Error(), "设备分组不存在")
}

func TestDeviceGroupRenameRejectsSoftDeletedName(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	active, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{Name: "当前分组"})
	require.NoError(t, err)
	deleted, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{Name: "已删分组"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteDeviceGroup(systemReq.DeviceGroupDelete{ID: deleted.ID}))

	_, err = service.SaveDeviceGroup(systemReq.DeviceGroupSave{
		ID:   active.ID,
		Name: "已删分组",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "设备分组已存在")
}

func TestDeviceConfigSaveRejectsMissingGroup(t *testing.T) {
	setupDeviceConfigTestDB(t)

	missingGroupID := uint(999)
	_, err := (&DeviceConfigService{}).SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "device-missing-group",
		AccountType: AccountTypeDefault,
		GroupID:     &missingGroupID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "设备分组不存在")
}

func TestDeviceConfigBatchUpdateOnlySelectedFields(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	groupA, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{Name: "A组"})
	require.NoError(t, err)
	groupB, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{Name: "B组"})
	require.NoError(t, err)
	deviceA, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "batch-001",
		AccountType: AccountTypeDefault,
		GroupID:     &groupA.ID,
		Remark:      "旧备注A",
	})
	require.NoError(t, err)
	deviceB, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "batch-002",
		AccountType: AccountTypeDefault,
		Remark:      "旧备注B",
	})
	require.NoError(t, err)

	updated, err := service.BatchUpdateDeviceConfig(systemReq.DeviceConfigBatchUpdate{
		IDs:               []uint{deviceA.ID, deviceB.ID},
		UpdateAccountType: true,
		AccountType:       AccountTypePC,
		UpdateGroup:       true,
		GroupID:           &groupB.ID,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, updated)

	var devices []model.SysDeviceConfig
	require.NoError(t, global.GVA_DB.Order("device_id asc").Find(&devices).Error)
	require.Len(t, devices, 2)
	for _, device := range devices {
		require.Equal(t, AccountTypePC, device.AccountType)
		require.NotNil(t, device.GroupID)
		require.Equal(t, groupB.ID, *device.GroupID)
	}
	require.Equal(t, "旧备注A", devices[0].Remark)
	require.Equal(t, "旧备注B", devices[1].Remark)
}

func TestDeviceConfigBatchUpdateCanClearGroupAndRemark(t *testing.T) {
	setupDeviceConfigTestDB(t)

	service := &DeviceConfigService{}
	group, err := service.SaveDeviceGroup(systemReq.DeviceGroupSave{Name: "待清空"})
	require.NoError(t, err)
	device, err := service.SaveDeviceConfig(systemReq.DeviceConfigSave{
		DeviceID:    "batch-clear",
		AccountType: AccountTypePC,
		GroupID:     &group.ID,
		Remark:      "待清空备注",
	})
	require.NoError(t, err)

	updated, err := service.BatchUpdateDeviceConfig(systemReq.DeviceConfigBatchUpdate{
		IDs:          []uint{device.ID},
		UpdateGroup:  true,
		GroupID:      nil,
		UpdateRemark: true,
		Remark:       " ",
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)

	var stored model.SysDeviceConfig
	require.NoError(t, global.GVA_DB.First(&stored, device.ID).Error)
	require.Equal(t, AccountTypePC, stored.AccountType)
	require.Nil(t, stored.GroupID)
	require.Empty(t, stored.Remark)
}

func TestDeviceConfigBatchUpdateRejectsEmptyFields(t *testing.T) {
	setupDeviceConfigTestDB(t)

	updated, err := (&DeviceConfigService{}).BatchUpdateDeviceConfig(systemReq.DeviceConfigBatchUpdate{
		IDs: []uint{1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "请选择要更新的字段")
	require.Zero(t, updated)
}
