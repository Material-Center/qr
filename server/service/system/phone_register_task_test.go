package system

import (
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPhoneRegisterTaskTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysPhoneRegisterTask{},
		&modelSystem.SysPhoneRegisterTaskLog{},
		&modelSystem.SysRegisterConfig{},
	))
	global.GVA_DB = db
}

func TestTimeoutUnfinishedTasksFailsWaitingPromoterCodeAfterSubmitWindow(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	deviceID := "9130dbc0"
	codeRequestedAt := now.Add(-phoneRegisterCodeSubmitWindow - time.Second)
	task := modelSystem.SysPhoneRegisterTask{
		Phone:           "18800000000",
		PromoterID:      1,
		SMSReceiveMode:  modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:          modelSystem.PhoneRegisterStatusWaitingPromoterCode,
		HolderDeviceID:  &deviceID,
		LastHeartbeatAt: &now,
		CodeRequestedAt: &codeRequestedAt,
		PendingCode:     "",
		ExpiresAt:       now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	require.NoError(t, (&PhoneRegisterTaskService{}).timeoutUnfinishedTasks())

	var got modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&got, task.ID).Error)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, got.Status)
	require.NotNil(t, got.StatusCode)
	require.Equal(t, modelSystem.PhoneRegisterStatusCodeVerifyCodeTimeout, *got.StatusCode)
	require.Equal(t, "验证码等待超时", got.LastError)
	require.NotNil(t, got.FinishedAt)
	require.Nil(t, got.HolderDeviceID)
	require.Empty(t, got.PendingCode)
	require.Nil(t, got.CodeRequestedAt)
}

func TestCreateTaskRejectsWhenPhoneRegisterDisabled(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	disabled := false
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:            modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:              0,
		PhoneRegisterEnabled: &disabled,
	}).Error)

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000000", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.EqualError(t, err, "手机号注册已关闭")
}

func TestCreateTaskRejectsWhenPromoterTaskCreationDisabled(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	disabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		Username:                  "promoter",
		NickName:                  "地推",
		AuthorityId:               300,
		Enable:                    1,
		PhoneRegisterTaskDisabled: &disabled,
	}).Error)

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000000", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.EqualError(t, err, "当前账号已禁用任务创建")
}

func TestCreateTaskRejectsBlockedPhonePrefix(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "13300000000", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.EqualError(t, err, "该手机号段暂不支持提交")
}

func TestCreateTaskRejectsNewBlockedPhonePrefixes(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "19000000000", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.EqualError(t, err, "该手机号段暂不支持提交")

	_, err = (&PhoneRegisterTaskService{}).CreateTask(1, "19300000000", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.EqualError(t, err, "该手机号段暂不支持提交")
}

func TestCreateTaskUsesConfiguredBlockedPhonePrefixes(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                    modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                      0,
		PhoneRegisterEnabled:         &enabled,
		PhoneRegisterBlockedPrefixes: "188 199",
	}).Error)

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000000", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.EqualError(t, err, "该手机号段暂不支持提交")
}

func TestGetCurrentDeviceStatsIgnoresTaskHeartbeatWithoutDeviceHeartbeat(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	busyDevice := "busy-device"
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:           "18800000001",
		PromoterID:      1,
		SMSReceiveMode:  modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:          modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID:  &busyDevice,
		LastHeartbeatAt: &now,
		ExpiresAt:       now.Add(time.Hour),
	}).Error)

	stats, err := (&PhoneRegisterTaskService{}).GetCurrentDeviceStats()
	require.NoError(t, err)
	require.EqualValues(t, 0, stats.Online)
	require.EqualValues(t, 0, stats.Idle)
}
