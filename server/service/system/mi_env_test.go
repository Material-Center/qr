package system

import (
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
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

func useMIEnvTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.SysMIEnvRecord{}))
	previous := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previous })
	return db
}

func miEnvTestIntPtr(value int) *int { return &value }
