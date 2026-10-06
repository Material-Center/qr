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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SysMIEnvRecord{}))
	previous := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previous })

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
}
