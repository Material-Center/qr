package system

import (
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMIEnvServiceLifecycle(t *testing.T) {
	useMIEnvTestDB(t)

	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	id, err := MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888",
		SerialBackupName: "backup-a", AndroidID: "android-a", Key: "key-a",
	}, now)
	require.NoError(t, err)
	require.NotZero(t, id)

	record, err := MIEnvServiceApp.Consume(MIEnvFilter{Type: "QQ888", DeviceID: "device-a"}, now)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, 1, record.UsageCount)

	record, err = MIEnvServiceApp.Consume(MIEnvFilter{Type: "QQ888", DeviceID: "device-a"}, now)
	require.NoError(t, err)
	require.Nil(t, record)

	stats, err := MIEnvServiceApp.Stats()
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.Total)
	require.EqualValues(t, 1, stats.Consumed)
	require.EqualValues(t, 0, stats.Available)
}

func TestMIEnvBaseModelSoftDeleteSemantics(t *testing.T) {
	db := useMIEnvTestDB(t)
	now := time.Date(2026, 10, 7, 16, 0, 0, 0, time.UTC)
	id, err := MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888",
		SerialBackupName: "backup-a", AndroidID: "android-a", Key: "key-a",
	}, now)
	require.NoError(t, err)
	require.NoError(t, MIEnvServiceApp.Delete(id, now.Add(time.Hour)))

	var activeCount int64
	require.NoError(t, db.Model(&model.SysMIEnvRecord{}).Count(&activeCount).Error)
	require.Zero(t, activeCount)

	record, err := MIEnvServiceApp.Get(id)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.True(t, record.DeletedAt.Valid)
	require.Equal(t, now.Add(time.Hour), record.DeletedAt.Time)

	stats, err := MIEnvServiceApp.Stats()
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.Total)
	require.EqualValues(t, 1, stats.Deleted)
	require.EqualValues(t, 0, stats.Available)

	cleaned, err := MIEnvServiceApp.Clean(nil, now.Add(2*time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 1, cleaned)
	var totalCount int64
	require.NoError(t, db.Unscoped().Model(&model.SysMIEnvRecord{}).Count(&totalCount).Error)
	require.Zero(t, totalCount)
}

func TestMIEnvAddIsIdempotentOnlyForTheSameEnvironmentIdentity(t *testing.T) {
	useMIEnvTestDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	record := model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888",
		SerialBackupName: "backup-a", AndroidID: "android-a", Key: "key-a",
	}

	firstID, err := MIEnvServiceApp.Add(record, now)
	require.NoError(t, err)
	retryID, err := MIEnvServiceApp.Add(record, now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, firstID, retryID)

	record.AndroidID = "android-b"
	record.Key = "key-b"
	secondID, err := MIEnvServiceApp.Add(record, now.Add(2*time.Minute))
	require.NoError(t, err)
	require.NotEqual(t, firstID, secondID)

	items, err := MIEnvServiceApp.List(MIEnvFilter{Type: "QQ888", DeviceID: "device-a", SerialBackupName: "backup-a"}, now)
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestMIEnvListAppliesAgeAndMadeCountFilters(t *testing.T) {
	useMIEnvTestDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	oldID, err := MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-old", Type: "QQ111",
		SerialBackupName: "backup-old", AndroidID: "android-old", Key: "key-old",
	}, now.Add(-20*24*time.Hour))
	require.NoError(t, err)
	require.NoError(t, MIEnvServiceApp.AdjustMadeCount(oldID, 2, false, now.Add(-19*24*time.Hour)))

	_, err = MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-new", Type: "QQ111",
		SerialBackupName: "backup-new", AndroidID: "android-new", Key: "key-new",
	}, now.Add(-5*24*time.Hour))
	require.NoError(t, err)

	items, err := MIEnvServiceApp.List(MIEnvFilter{
		Type: "QQ111", MinMadeCount: miEnvTestIntPtr(2),
		MinDays: miEnvTestIntPtr(30), MaxDays: miEnvTestIntPtr(7),
	}, now)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, oldID, items[0].ID)
}

func TestMIEnvListForMakeExcludesConsumedRecords(t *testing.T) {
	useMIEnvTestDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	id, err := MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888",
		SerialBackupName: "backup-a", AndroidID: "android-a", Key: "key-a",
	}, now)
	require.NoError(t, err)

	consumed, err := MIEnvServiceApp.Consume(MIEnvFilter{Type: "QQ888", DeviceID: "device-a"}, now)
	require.NoError(t, err)
	require.Equal(t, id, consumed.ID)

	record, err := MIEnvServiceApp.ListForMake(MIEnvFilter{
		Type: "QQ888", DeviceCode: "cepheus", DeviceID: "device-a",
		MaxMadeCount: miEnvTestIntPtr(3), CooldownDays: miEnvTestIntPtr(1),
	}, now.Add(48*time.Hour))
	require.NoError(t, err)
	require.Nil(t, record)
}

func TestMIEnvListForMakeReservesUntilSuccessOrExpiry(t *testing.T) {
	useMIEnvTestDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	id, err := MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888",
		SerialBackupName: "backup-a", AndroidID: "android-a", Key: "key-a",
	}, now)
	require.NoError(t, err)
	filter := MIEnvFilter{Type: "QQ888", DeviceCode: "cepheus", DeviceID: "device-a", MaxMadeCount: miEnvTestIntPtr(3), CooldownDays: miEnvTestIntPtr(1)}

	reserved, err := MIEnvServiceApp.ListForMake(filter, now)
	require.NoError(t, err)
	require.Equal(t, id, reserved.ID)
	require.NotNil(t, reserved.MakeReservedUntil)
	require.Equal(t, now.Add(miEnvMakeReservationTTL), *reserved.MakeReservedUntil)

	duplicate, err := MIEnvServiceApp.ListForMake(filter, now)
	require.NoError(t, err)
	require.Nil(t, duplicate)
	consumedWhileReserved, err := MIEnvServiceApp.Consume(MIEnvFilter{Type: "QQ888", DeviceID: "device-a"}, now)
	require.NoError(t, err)
	require.Nil(t, consumedWhileReserved)

	updated, err := MIEnvServiceApp.MarkMakeSuccess(id, now.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, 2, updated.MadeCount)
	require.Nil(t, updated.MakeReservedUntil)

	_, err = MIEnvServiceApp.MarkMakeSuccess(id, now.Add(time.Hour))
	require.ErrorIs(t, err, ErrMIEnvUnavailable)

	id2, err := MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-b", Type: "QQ888",
		SerialBackupName: "backup-b", AndroidID: "android-b", Key: "key-b",
	}, now)
	require.NoError(t, err)
	filter.DeviceID = "device-b"
	first, err := MIEnvServiceApp.ListForMake(filter, now)
	require.NoError(t, err)
	require.Equal(t, id2, first.ID)
	second, err := MIEnvServiceApp.ListForMake(filter, now.Add(miEnvMakeReservationTTL+time.Second))
	require.NoError(t, err)
	require.Equal(t, id2, second.ID)
}

func TestMIEnvMutationsRejectMissingRecords(t *testing.T) {
	useMIEnvTestDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	_, err := MIEnvServiceApp.MarkMakeSuccess(999, now)
	require.ErrorIs(t, err, ErrMIEnvUnavailable)
	require.ErrorIs(t, MIEnvServiceApp.SetFrozen(999, true, now), ErrMIEnvUnavailable)
	require.ErrorIs(t, MIEnvServiceApp.AdjustMadeCount(999, 1, false, now), ErrMIEnvUnavailable)
	require.ErrorIs(t, MIEnvServiceApp.Delete(999, now), ErrMIEnvUnavailable)
}

func TestMIEnvAdminListAppliesFiltersAndPagination(t *testing.T) {
	db := useMIEnvTestDB(t)
	base := time.Date(2026, 10, 7, 8, 0, 0, 0, time.Local)
	records := []model.SysMIEnvRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: base, UpdatedAt: base.Add(time.Hour)}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ111", SerialBackupName: "a", AndroidID: "android-a", Key: "key-a", UsageCount: 1, MaxUsage: 3, MadeCount: 2, Frozen: false},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: base, UpdatedAt: base.Add(2 * time.Hour)}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888", SerialBackupName: "b", AndroidID: "android-b", Key: "key-b", UsageCount: 2, MaxUsage: 3, MadeCount: 3, Frozen: true},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: base, UpdatedAt: base.Add(3 * time.Hour)}, DeviceCode: "venus", DeviceID: "device-b", Type: "QQ888", SerialBackupName: "c", AndroidID: "android-c", Key: "key-c", UsageCount: 1, MaxUsage: 3, MadeCount: 2, Frozen: false},
	}
	require.NoError(t, db.Create(&records).Error)
	minUsage, maxUsage, minMade, maxMade := 1, 1, 2, 2
	list, total, err := MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{
		DeviceID:       "device-a",
		MinUsage:       &minUsage,
		MaxUsage:       &maxUsage,
		MinMadeCount:   &minMade,
		MaxMadeCount:   &maxMade,
		Status:         "normal",
		UpdatedAtStart: base.Format("2006-01-02 15:04:05"),
		UpdatedAtEnd:   base.Add(90 * time.Minute).Format("2006-01-02 15:04:05"),
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, "a", list[0].SerialBackupName)
	_, total, err = MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{DeviceID: "device"})
	require.NoError(t, err)
	require.Zero(t, total, "device filtering must stay exact so the device indexes remain usable")

	page, total, err := MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{PageInfo: commonReq.PageInfo{Page: 2, PageSize: 1}})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, page, 1)
	require.Equal(t, records[1].ID, page[0].ID)
}

func TestMIEnvAdminDeleteAllOnlyDeletesMatchingActiveRecords(t *testing.T) {
	db := useMIEnvTestDB(t)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.Local)
	records := []model.SysMIEnvRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ111", SerialBackupName: "a", AndroidID: "android-a", Key: "key-a", MaxUsage: 1, MadeCount: 1, Frozen: true},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888", SerialBackupName: "b", AndroidID: "android-b", Key: "key-b", MaxUsage: 1, MadeCount: 1, Frozen: false},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "venus", DeviceID: "device-b", Type: "QQ888", SerialBackupName: "c", AndroidID: "android-c", Key: "key-c", MaxUsage: 1, MadeCount: 1, Frozen: true},
	}
	require.NoError(t, db.Create(&records).Error)
	deleted, err := MIEnvServiceApp.DeleteAllForAdmin(systemReq.MIEnvAdminList{DeviceID: "device-a", Status: "frozen"}, now.Add(time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)

	var active []model.SysMIEnvRecord
	require.NoError(t, db.Where("deleted_at IS NULL").Order("id").Find(&active).Error)
	require.Len(t, active, 2)
	require.Equal(t, records[1].ID, active[0].ID)
	require.Equal(t, records[2].ID, active[1].ID)
}

func TestMIEnvAdminDeleteSelectedOnlyDeletesRequestedActiveRecords(t *testing.T) {
	db := useMIEnvTestDB(t)
	now := time.Date(2026, 10, 7, 13, 0, 0, 0, time.Local)
	records := []model.SysMIEnvRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ111", SerialBackupName: "a", AndroidID: "android-a", Key: "key-a", MaxUsage: 1, MadeCount: 1},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "venus", DeviceID: "device-b", Type: "QQ111", SerialBackupName: "b", AndroidID: "android-b", Key: "key-b", MaxUsage: 1, MadeCount: 1},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "umi", DeviceID: "device-c", Type: "QQ111", SerialBackupName: "c", AndroidID: "android-c", Key: "key-c", MaxUsage: 1, MadeCount: 1},
	}
	require.NoError(t, db.Create(&records).Error)

	deleted, err := MIEnvServiceApp.DeleteSelectedForAdmin([]uint{records[0].ID, records[2].ID, records[0].ID, 999}, now.Add(time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 2, deleted)

	var activeIDs []uint
	require.NoError(t, db.Model(&model.SysMIEnvRecord{}).Where("deleted_at IS NULL").Pluck("id", &activeIDs).Error)
	require.Equal(t, []uint{records[1].ID}, activeIDs)

	_, err = MIEnvServiceApp.DeleteSelectedForAdmin(nil, now)
	require.EqualError(t, err, "请选择要删除的环境")
	_, err = MIEnvServiceApp.DeleteSelectedForAdmin([]uint{0}, now)
	require.EqualError(t, err, "环境 ID 无效")
}

func TestMIEnvAdminDeviceGroupFilterAppliesToListAndDeleteAll(t *testing.T) {
	db := useMIEnvTestDB(t)
	now := time.Date(2026, 10, 7, 14, 0, 0, 0, time.Local)
	groups := []model.SysDeviceGroup{{Name: "A组"}, {Name: "B组"}}
	require.NoError(t, db.Create(&groups).Error)
	groupAID := groups[0].ID
	groupBID := groups[1].ID
	require.NoError(t, db.Create(&[]model.SysDeviceConfig{
		{DeviceID: "device-a", AccountType: "default", GroupID: &groupAID},
		{DeviceID: "device-b", AccountType: "default", GroupID: nil},
		{DeviceID: "device-c", AccountType: "default", GroupID: &groupBID},
	}).Error)
	records := []model.SysMIEnvRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ111", SerialBackupName: "a", AndroidID: "android-a", Key: "key-a", MaxUsage: 1, MadeCount: 1},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "venus", DeviceID: "device-b", Type: "QQ111", SerialBackupName: "b", AndroidID: "android-b", Key: "key-b", MaxUsage: 1, MadeCount: 1},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "umi", DeviceID: "device-c", Type: "QQ111", SerialBackupName: "c", AndroidID: "android-c", Key: "key-c", MaxUsage: 1, MadeCount: 1},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "apollo", DeviceID: "device-without-config", Type: "QQ111", SerialBackupName: "d", AndroidID: "android-d", Key: "key-d", MaxUsage: 1, MadeCount: 1},
	}
	require.NoError(t, db.Create(&records).Error)

	groupList, total, err := MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{GroupID: &groupAID})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, groupList, 1)
	require.Equal(t, "device-a", groupList[0].DeviceID)

	ungroupedList, total, err := MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{Ungrouped: true})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.ElementsMatch(t, []string{"device-b", "device-without-config"}, []string{ungroupedList[0].DeviceID, ungroupedList[1].DeviceID})

	deleted, err := MIEnvServiceApp.DeleteAllForAdmin(systemReq.MIEnvAdminList{GroupID: &groupAID}, now.Add(time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	var activeDeviceIDs []string
	require.NoError(t, db.Model(&model.SysMIEnvRecord{}).Where("deleted_at IS NULL").Pluck("device_id", &activeDeviceIDs).Error)
	require.NotContains(t, activeDeviceIDs, "device-a")
	require.ElementsMatch(t, []string{"device-b", "device-c", "device-without-config"}, activeDeviceIDs)
}

func TestMIEnvAdminTypeFilterAndDeviceTypeAutocomplete(t *testing.T) {
	db := useMIEnvTestDB(t)
	now := time.Date(2026, 10, 7, 15, 0, 0, 0, time.Local)
	records := []model.SysMIEnvRecord{
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ111", SerialBackupName: "a", AndroidID: "android-a", Key: "key-a", MaxUsage: 1, MadeCount: 1},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ888", SerialBackupName: "b", AndroidID: "android-b", Key: "key-b", MaxUsage: 1, MadeCount: 1},
		{GVA_MODEL: global.GVA_MODEL{CreatedAt: now, UpdatedAt: now}, DeviceCode: "venus", DeviceID: "device-b", Type: "QQ25888", SerialBackupName: "c", AndroidID: "android-c", Key: "key-c", MaxUsage: 1, MadeCount: 1},
	}
	require.NoError(t, db.Create(&records).Error)

	types, err := MIEnvServiceApp.ListTypesForAdmin(systemReq.MIEnvAdminList{DeviceID: "device-a"})
	require.NoError(t, err)
	require.Equal(t, []string{"QQ111", "QQ888"}, types)

	types, err = MIEnvServiceApp.ListTypesForAdmin(systemReq.MIEnvAdminList{DeviceID: "cepheus"})
	require.NoError(t, err)
	require.Empty(t, types, "device code must not be accepted as an admin filter")

	list, total, err := MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{DeviceID: "device-a", Type: "QQ888"})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, records[1].ID, list[0].ID)

	deleted, err := MIEnvServiceApp.DeleteAllForAdmin(systemReq.MIEnvAdminList{DeviceID: "device-a", Type: "QQ888"}, now.Add(time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	var activeTypes []string
	require.NoError(t, db.Model(&model.SysMIEnvRecord{}).Where("device_id = ? AND deleted_at IS NULL", "device-a").Pluck("type", &activeTypes).Error)
	require.Equal(t, []string{"QQ111"}, activeTypes)
}

func TestMIEnvAdminCompositeIndexesExist(t *testing.T) {
	db := useMIEnvTestDB(t)
	require.NoError(t, model.EnsureMIEnvRecordIndexes(db))
	for _, indexName := range []string{
		"idx_sys_mi_env_records_created_at",
		"idx_sys_mi_env_records_updated_at",
		"idx_mi_env_admin_active_updated",
		"idx_mi_env_admin_device_active_type",
		"idx_mi_env_admin_active_usage",
		"idx_mi_env_admin_active_made",
		"idx_mi_env_admin_active_frozen_updated",
	} {
		require.True(t, db.Migrator().HasIndex(&model.SysMIEnvRecord{}, indexName), indexName)
	}
}

func TestMIEnvAdminListRejectsInvalidFilters(t *testing.T) {
	useMIEnvTestDB(t)
	minUsage, maxUsage := 2, 1
	_, _, err := MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{MinUsage: &minUsage, MaxUsage: &maxUsage})
	require.EqualError(t, err, "使用次数最小值不能大于最大值")

	negativeMadeCount := -1
	_, _, err = MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{MinMadeCount: &negativeMadeCount})
	require.EqualError(t, err, "制作次数不能小于 0")

	_, _, err = MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{Status: "deleted"})
	require.EqualError(t, err, "状态仅支持 normal 或 frozen")

	_, _, err = MIEnvServiceApp.ListForAdmin(systemReq.MIEnvAdminList{UpdatedAtStart: "not-a-time"})
	require.EqualError(t, err, "更新时间格式无效")
}

func useMIEnvTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.SysMIEnvRecord{}, &model.SysDeviceGroup{}, &model.SysDeviceConfig{}))
	require.NoError(t, model.EnsureMIEnvRecordIndexes(db))
	previous := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previous })
	return db
}

func miEnvTestIntPtr(value int) *int { return &value }
