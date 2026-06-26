package system

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	modelCommonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	modelSystemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	modelSystemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPhoneRegisterTaskTestDB(t *testing.T) {
	t.Helper()
	global.GVA_REDIS = nil
	resetPhoneRegisterDeviceStatsCache()
	resetPhoneRegisterTimeoutScanThrottle()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysPhoneRegisterTask{},
		&modelSystem.SysPhoneRegisterTaskLog{},
		&modelSystem.SysPhoneRegisterRiskDailyStat{},
		&modelSystem.SysRegisterConfig{},
	))
	global.GVA_DB = db
}

func setupPhoneRegisterTaskTestDBWithoutRiskStat(t *testing.T) {
	t.Helper()
	global.GVA_REDIS = nil
	resetPhoneRegisterDeviceStatsCache()
	resetPhoneRegisterTimeoutScanThrottle()
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

func createPhoneRegisterTaskTestPromoter(t *testing.T, id uint) {
	t.Helper()
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: id},
		Username:    "promoter",
		NickName:    "地推",
		AuthorityId: 300,
		Enable:      1,
	}).Error)
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

func TestRequestTriggeredTimeoutScanIsThrottled(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	first := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      now.Add(-time.Second),
	}
	require.NoError(t, global.GVA_DB.Create(&first).Error)

	_, _, err := (&PhoneRegisterTaskService{}).DeviceTask(modelSystemReq.PhoneRegisterDeviceTask{DeviceID: "device-a"})
	require.NoError(t, err)

	var storedFirst modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&storedFirst, first.ID).Error)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, storedFirst.Status)

	second := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000002",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      now.Add(-time.Second),
	}
	require.NoError(t, global.GVA_DB.Create(&second).Error)

	_, _, err = (&PhoneRegisterTaskService{}).DeviceTask(modelSystemReq.PhoneRegisterDeviceTask{DeviceID: "device-a"})
	require.NoError(t, err)

	var storedSecond modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&storedSecond, second.ID).Error)
	require.Equal(t, modelSystem.PhoneRegisterStatusPending, storedSecond.Status)
}

func TestDeviceTaskCoalescesConcurrentCurrentTaskLookup(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	sqlDB, err := global.GVA_DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	now := time.Now()
	deviceID := "device-a"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:           "18800000003",
		PromoterID:      1,
		SMSReceiveMode:  modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:          modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID:  &deviceID,
		LastHeartbeatAt: &now,
		ExpiresAt:       now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	phoneRegisterTimeoutScanThrottleState.Lock()
	phoneRegisterTimeoutScanThrottleState.lastRun = time.Now()
	phoneRegisterTimeoutScanThrottleState.Unlock()

	started := make(chan struct{})
	release := make(chan struct{})
	var lookupQueries atomic.Int32
	callbackName := "phone_register_task_test:count_device_task_lookup"
	require.NoError(t, global.GVA_DB.Callback().Query().Before("gorm:query").Register(callbackName, func(db *gorm.DB) {
		if db.Statement != nil && db.Statement.Table == "sys_phone_register_tasks" {
			if lookupQueries.Add(1) == 1 {
				close(started)
				<-release
			}
		}
	}))
	defer global.GVA_DB.Callback().Query().Remove(callbackName)

	const workers = 5
	results := make([]modelSystem.SysPhoneRegisterTask, workers)
	found := make([]bool, workers)
	errs := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], found[index], errs[index] = (&PhoneRegisterTaskService{}).DeviceTask(modelSystemReq.PhoneRegisterDeviceTask{DeviceID: deviceID})
		}(i)
	}

	close(start)
	<-started
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	require.EqualValues(t, 1, lookupQueries.Load())
	for i := range results {
		require.NoError(t, errs[i])
		require.True(t, found[i])
		require.Equal(t, task.ID, results[i].ID)
	}
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

func TestCreateTaskRejectsUserSentModeWhenDisabledByConfig(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)

	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                         modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                           0,
		PhoneRegisterUserSentTaskDisabled: true,
	}).Error)

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000000", modelSystem.PhoneRegisterSMSModeUserSentToTX)
	require.EqualError(t, err, "自己发码任务创建已关闭")

	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterSMSModePlatformSend, task.SMSReceiveMode)
}

func TestCreateTaskRejectsPlatformSendModeWhenDisabledByConfig(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)

	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                        modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                          0,
		PhoneRegisterReceiveTaskDisabled: true,
	}).Error)

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000000", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.EqualError(t, err, "收码任务创建已关闭")

	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModeUserSentToTX)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterSMSModeUserSentToTX, task.SMSReceiveMode)
}

func TestCreateTaskRejectsInvalidPhoneFormat(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	cases := []string{
		"1880000000",
		"188000000000",
		"1880000000a",
	}
	for _, phone := range cases {
		_, err := (&PhoneRegisterTaskService{}).CreateTask(1, phone, modelSystem.PhoneRegisterSMSModePlatformSend)
		require.EqualError(t, err, "手机号必须为11位数字")
	}
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

func TestCreateTaskWithStartDelaySetsAvailableAtAndExpiresAfterAvailableAt(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)

	before := time.Now()
	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000000", modelSystem.PhoneRegisterSMSModeUserSentToTX, PhoneRegisterTaskCreateOptions{
		TaskSource:        modelSystem.PhoneRegisterTaskSourceOpenAPI,
		StartDelaySeconds: 120,
	})
	require.NoError(t, err)
	after := time.Now()

	require.Equal(t, modelSystem.PhoneRegisterTaskCreateSourceOpenAPI, task.CreateSource)
	require.Empty(t, task.TaskSource)
	require.Nil(t, task.HolderDeviceID)
	require.NotNil(t, task.AvailableAt)
	require.False(t, task.AvailableAt.Before(before.Add(120*time.Second)))
	require.False(t, task.AvailableAt.After(after.Add(120*time.Second)))
	require.Equal(t, task.AvailableAt.Add(phoneRegisterTaskTimeout), task.ExpiresAt)
}

func TestOpenAPICreatedTaskKeepsCreateSourceWhenClaimedByScript(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)

	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000002", modelSystem.PhoneRegisterSMSModeUserSentToTX, PhoneRegisterTaskCreateOptions{
		TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI,
	})
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterTaskCreateSourceOpenAPI, task.CreateSource)

	claimed, found, err := (&PhoneRegisterTaskService{}).DevicePoll(modelSystemReq.PhoneRegisterDevicePoll{DeviceID: "script-device"})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, task.ID, claimed.ID)

	var stored modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&stored, task.ID).Error)
	require.Equal(t, modelSystem.PhoneRegisterTaskCreateSourceOpenAPI, stored.CreateSource)
	require.Equal(t, modelSystem.PhoneRegisterTaskSourceScript, stored.TaskSource)

	list, err := (&PhoneRegisterTaskService{}).GetTaskList(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskList{
		PageInfo:     modelCommonReq.PageInfo{Page: 1, PageSize: 20},
		CreateSource: modelSystem.PhoneRegisterTaskCreateSourceOpenAPI,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, list.Total)
	require.Len(t, list.List, 1)
	require.Equal(t, modelSystem.PhoneRegisterTaskCreateSourceOpenAPI, list.List[0].CreateSource)
	require.Equal(t, modelSystem.PhoneRegisterTaskSourceScript, list.List[0].TaskSource)
}

func TestCreateTaskWithReserveDeviceFallsBackToUnlockedDelayTaskWhenNoDeviceAvailable(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)

	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModeUserSentToTX, PhoneRegisterTaskCreateOptions{
		TaskSource:        modelSystem.PhoneRegisterTaskSourceOpenAPI,
		StartDelaySeconds: 60,
		ReserveDevice:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, task.AvailableAt)
	require.Nil(t, task.HolderDeviceID)
}

func TestDevicePollReturnsEmptyForReservedTaskBeforeAvailableAtAndDoesNotClaimOtherTask(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	now := time.Now()
	deviceID := "reserved-device"
	availableAt := now.Add(time.Minute)
	reserved := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000002",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		HolderDeviceID: &deviceID,
		AvailableAt:    &availableAt,
		ExpiresAt:      availableAt.Add(phoneRegisterTaskTimeout),
	}
	require.NoError(t, global.GVA_DB.Create(&reserved).Error)
	ordinary := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000003",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      now.Add(phoneRegisterTaskTimeout),
	}
	require.NoError(t, global.GVA_DB.Create(&ordinary).Error)

	got, found, err := (&PhoneRegisterTaskService{}).DevicePoll(modelSystemReq.PhoneRegisterDevicePoll{DeviceID: deviceID})
	require.NoError(t, err)
	require.False(t, found)
	require.Zero(t, got.ID)

	var storedOrdinary modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&storedOrdinary, ordinary.ID).Error)
	require.Nil(t, storedOrdinary.HolderDeviceID)
	require.Equal(t, modelSystem.PhoneRegisterStatusPending, storedOrdinary.Status)
}

func TestDevicePollClaimsReservedTaskAfterAvailableAtWithinGrace(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	now := time.Now()
	deviceID := "reserved-device"
	availableAt := now.Add(-time.Second)
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000004",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		HolderDeviceID: &deviceID,
		AvailableAt:    &availableAt,
		ExpiresAt:      now.Add(phoneRegisterTaskTimeout),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, found, err := (&PhoneRegisterTaskService{}).DevicePoll(modelSystemReq.PhoneRegisterDevicePoll{DeviceID: deviceID})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, task.ID, got.ID)
	require.Equal(t, modelSystem.PhoneRegisterStatusRunning, got.Status)
	require.NotNil(t, got.ClaimedAt)
}

func TestDevicePollDoesNotClaimUnreservedTaskBeforeAvailableAt(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	now := time.Now()
	availableAt := now.Add(time.Minute)
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000005",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:         modelSystem.PhoneRegisterStatusPending,
		AvailableAt:    &availableAt,
		ExpiresAt:      availableAt.Add(phoneRegisterTaskTimeout),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, found, err := (&PhoneRegisterTaskService{}).DevicePoll(modelSystemReq.PhoneRegisterDevicePoll{DeviceID: "free-device"})
	require.NoError(t, err)
	require.False(t, found)
	require.Zero(t, got.ID)
}

func TestDevicePollReleasesExpiredReservationAndAllowsOtherDeviceToClaim(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	now := time.Now()
	reservedDeviceID := "missed-device"
	availableAt := now.Add(-reservedClaimGracePeriod - time.Second)
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000006",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		HolderDeviceID: &reservedDeviceID,
		AvailableAt:    &availableAt,
		ExpiresAt:      now.Add(phoneRegisterTaskTimeout),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, found, err := (&PhoneRegisterTaskService{}).DevicePoll(modelSystemReq.PhoneRegisterDevicePoll{DeviceID: "other-device"})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, task.ID, got.ID)
	require.Equal(t, "other-device", *got.HolderDeviceID)
	require.Equal(t, modelSystem.PhoneRegisterStatusRunning, got.Status)
}

func TestAttachOpenAPICacheAllowsFailedTaskAndKeepsFailure(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	statusCode := modelSystem.PhoneRegisterStatusCodeOpenAPIFeedback
	holderDeviceID := "openapi-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000000",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusFailed,
		StatusCode:     &statusCode,
		LastError:      "注册失败",
		FinishedAt:     &now,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	var got modelSystem.SysPhoneRegisterTask
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var attachErr error
		got, attachErr = (&PhoneRegisterTaskService{}).AttachOpenAPICacheTx(tx, "openapi-device", task.ID, 123, "3995613452")
		return attachErr
	})
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, got.Status)
	require.Equal(t, "注册失败", got.LastError)
	require.Equal(t, modelSystem.PhoneRegisterCacheStatusUploaded, got.CacheStatus)
	require.Equal(t, "3995613452", got.QQNum)
	require.NotNil(t, got.QQCacheRecordID)
	require.EqualValues(t, 123, *got.QQCacheRecordID)

	var stored modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&stored, task.ID).Error)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, stored.Status)
	require.Equal(t, "注册失败", stored.LastError)
	require.Equal(t, modelSystem.PhoneRegisterCacheStatusUploaded, stored.CacheStatus)
	require.Equal(t, "3995613452", stored.QQNum)
}

func TestOpenAPICacheUploadMarksDeviceCooldownForStats(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	require.NoError(t, global.GVA_DB.AutoMigrate(&modelSystem.SysQQCacheRecord{}))

	var cooldownDeviceID string
	var cooldownTTL time.Duration
	originalCooldown := phoneRegisterMarkDeviceCooldown
	phoneRegisterMarkDeviceCooldown = func(deviceID string, ttl time.Duration) error {
		cooldownDeviceID = deviceID
		cooldownTTL = ttl
		return nil
	}
	t.Cleanup(func() {
		phoneRegisterMarkDeviceCooldown = originalCooldown
	})

	now := time.Now()
	statusCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	holderDeviceID := "openapi-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusSucceeded,
		StatusCode:     &statusCode,
		CacheStatus:    modelSystem.PhoneRegisterCacheStatusPending,
		FinishedAt:     &now,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	_, _, err := (&QQCacheService{}).UploadPhoneRegister(modelSystemReq.QQCacheUpload{
		TaskID:   task.ID,
		QQNum:    "3995613452",
		QQPwd:    "pwd",
		INI:      "[3995613452]\nclientVersion=9.2.75\n",
		DeviceID: holderDeviceID,
	})
	require.NoError(t, err)
	require.Equal(t, holderDeviceID, cooldownDeviceID)
	require.Equal(t, 5*time.Minute, cooldownTTL)
}

func TestGetTaskListFiltersByCacheUploadStatus(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	tasks := []modelSystem.SysPhoneRegisterTask{
		{Phone: "1880000000001", PromoterID: 1, CacheStatus: modelSystem.PhoneRegisterCacheStatusUploaded, Status: modelSystem.PhoneRegisterStatusSucceeded, StatusCode: &successCode, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{Phone: "1880000000002", PromoterID: 1, CacheStatus: modelSystem.PhoneRegisterCacheStatusPending, Status: modelSystem.PhoneRegisterStatusSucceeded, StatusCode: &successCode, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{Phone: "1880000000003", PromoterID: 1, CacheStatus: modelSystem.PhoneRegisterCacheStatusTimeout, Status: modelSystem.PhoneRegisterStatusSucceeded, StatusCode: &successCode, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{Phone: "1880000000004", PromoterID: 1, CacheStatus: "", Status: modelSystem.PhoneRegisterStatusSucceeded, StatusCode: &successCode, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
	}
	require.NoError(t, global.GVA_DB.Create(&tasks).Error)

	uploaded, err := (&PhoneRegisterTaskService{}).GetTaskList(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskList{
		PageInfo:    modelCommonReq.PageInfo{Page: 1, PageSize: 20},
		CacheStatus: "uploaded",
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, uploaded.Total)
	require.Len(t, uploaded.List, 1)
	require.Equal(t, modelSystem.PhoneRegisterCacheStatusUploaded, uploaded.List[0].CacheStatus)

	notUploaded, err := (&PhoneRegisterTaskService{}).GetTaskList(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskList{
		PageInfo:    modelCommonReq.PageInfo{Page: 1, PageSize: 20},
		CacheStatus: "not_uploaded",
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, notUploaded.Total)
	require.Len(t, notUploaded.List, 3)
	for _, item := range notUploaded.List {
		require.NotEqual(t, modelSystem.PhoneRegisterCacheStatusUploaded, item.CacheStatus)
	}
}

func TestGetTaskListFiltersByTaskSource(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	tasks := []modelSystem.SysPhoneRegisterTask{
		{Phone: "1880000000001", PromoterID: 1, TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI, Status: modelSystem.PhoneRegisterStatusPending, ExpiresAt: now.Add(time.Hour)},
		{Phone: "1880000000002", PromoterID: 1, TaskSource: modelSystem.PhoneRegisterTaskSourceScript, Status: modelSystem.PhoneRegisterStatusPending, ExpiresAt: now.Add(time.Hour)},
		{Phone: "1880000000003", PromoterID: 1, TaskSource: "", Status: modelSystem.PhoneRegisterStatusPending, ExpiresAt: now.Add(time.Hour)},
	}
	require.NoError(t, global.GVA_DB.Create(&tasks).Error)

	openAPI, err := (&PhoneRegisterTaskService{}).GetTaskList(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskList{
		PageInfo:   modelCommonReq.PageInfo{Page: 1, PageSize: 20},
		TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, openAPI.Total)
	require.Len(t, openAPI.List, 1)
	require.Equal(t, modelSystem.PhoneRegisterTaskSourceOpenAPI, openAPI.List[0].TaskSource)

	manual, err := (&PhoneRegisterTaskService{}).GetTaskList(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskList{
		PageInfo:   modelCommonReq.PageInfo{Page: 1, PageSize: 20},
		TaskSource: "manual",
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, manual.Total)
	require.Len(t, manual.List, 2)
	for _, item := range manual.List {
		require.NotEqual(t, modelSystem.PhoneRegisterTaskSourceOpenAPI, item.TaskSource)
	}
}

func TestOpenAPIReportFailureKeepsHolderForCacheUpload(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	holderDeviceID := "openapi-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000000",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, err := (&PhoneRegisterTaskService{}).OpenAPIReportFailure(holderDeviceID, task.ID, "注册失败")
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, got.Status)
	require.Equal(t, "注册失败", got.LastError)
	require.NotNil(t, got.HolderDeviceID)
	require.Equal(t, holderDeviceID, *got.HolderDeviceID)

	var stored modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&stored, task.ID).Error)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, stored.Status)
	require.NotNil(t, stored.HolderDeviceID)
	require.Equal(t, holderDeviceID, *stored.HolderDeviceID)
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

func TestGetCurrentDeviceStatsUsesShortCache(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()

	var onlineCalls int
	var busyCalls int
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			onlineCalls++
			return []string{"device-a", "device-b"}
		},
		func() []string {
			busyCalls++
			return []string{"device-b"}
		},
	)
	defer restore()

	first, err := (&PhoneRegisterTaskService{}).GetCurrentDeviceStats()
	require.NoError(t, err)
	second, err := (&PhoneRegisterTaskService{}).GetCurrentDeviceStats()
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.EqualValues(t, 2, first.Online)
	require.EqualValues(t, 1, first.Idle)
	require.Equal(t, 1, onlineCalls)
	require.Equal(t, 1, busyCalls)
}

func TestGetCurrentDeviceStatsExcludesCooldownDevices(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()

	restoreDevices := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"cooldown-device", "idle-device", "busy-device"}
		},
		func() []string {
			return []string{"busy-device"}
		},
	)
	defer restoreDevices()
	originalCooldown := phoneRegisterListCooldownDeviceIDs
	phoneRegisterListCooldownDeviceIDs = func() []string {
		return []string{"cooldown-device"}
	}
	defer func() {
		phoneRegisterListCooldownDeviceIDs = originalCooldown
	}()

	stats, err := (&PhoneRegisterTaskService{}).GetCurrentDeviceStats()
	require.NoError(t, err)
	require.EqualValues(t, 2, stats.Online)
	require.EqualValues(t, 1, stats.Idle)
}

func TestGetTaskListPromoterDeviceIdleSubtractsClaimablePendingTasks(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()

	now := time.Now()
	future := now.Add(time.Hour)
	heldDevice := "held-device"
	tasks := []modelSystem.SysPhoneRegisterTask{
		{Phone: "18800000001", PromoterID: 1, SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend, Status: modelSystem.PhoneRegisterStatusPending, ExpiresAt: now.Add(time.Hour)},
		{Phone: "18800000002", PromoterID: 2, SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend, Status: modelSystem.PhoneRegisterStatusPending, ExpiresAt: now.Add(time.Hour)},
		{Phone: "18800000003", PromoterID: 1, SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend, Status: modelSystem.PhoneRegisterStatusPending, AvailableAt: &future, ExpiresAt: now.Add(2 * time.Hour)},
		{Phone: "18800000004", PromoterID: 1, SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend, Status: modelSystem.PhoneRegisterStatusPending, HolderDeviceID: &heldDevice, ExpiresAt: now.Add(time.Hour)},
		{Phone: "18800000005", PromoterID: 1, SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend, Status: modelSystem.PhoneRegisterStatusSucceeded, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
	}
	require.NoError(t, global.GVA_DB.Create(&tasks).Error)

	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a", "device-b", "device-c", "device-d", "device-e"}
		},
		func() []string {
			return []string{"device-e"}
		},
	)
	defer restore()

	promoterList, err := (&PhoneRegisterTaskService{}).GetTaskList(phoneRolePromoter, 1, modelSystemReq.PhoneRegisterTaskList{
		PageInfo: modelCommonReq.PageInfo{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	require.EqualValues(t, 5, promoterList.Device.Online)
	require.EqualValues(t, 2, promoterList.Device.Idle)

	adminList, err := (&PhoneRegisterTaskService{}).GetTaskList(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskList{
		PageInfo: modelCommonReq.PageInfo{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	require.EqualValues(t, 5, adminList.Device.Online)
	require.EqualValues(t, 4, adminList.Device.Idle)
}

func TestPendingClaimableTaskCountUsesRedisShortCache(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	server := newFakeRedisServer(t, nil)
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
		resetPhoneRegisterPendingClaimableTaskCountCache()
	})

	now := time.Now()
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      now.Add(time.Hour),
	}).Error)

	first, err := (&PhoneRegisterTaskService{}).countPendingClaimableTasksWithoutHolder()
	require.NoError(t, err)
	require.EqualValues(t, 1, first)

	require.NoError(t, global.GVA_DB.Model(&modelSystem.SysPhoneRegisterTask{}).
		Where("phone = ?", "18800000001").
		Update("status", modelSystem.PhoneRegisterStatusSucceeded).Error)
	second, err := (&PhoneRegisterTaskService{}).countPendingClaimableTasksWithoutHolder()
	require.NoError(t, err)
	require.EqualValues(t, 1, second)
	require.GreaterOrEqual(t, server.count("get"), 2)
}

func TestPendingClaimableTaskCountCacheInvalidatedWhenTaskClaimed(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	server := newFakeRedisServer(t, nil)
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
		resetPhoneRegisterPendingClaimableTaskCountCache()
	})

	now := time.Now()
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      now.Add(time.Hour),
	}).Error)
	cached, err := (&PhoneRegisterTaskService{}).countPendingClaimableTasksWithoutHolder()
	require.NoError(t, err)
	require.EqualValues(t, 1, cached)

	_, found, err := (&PhoneRegisterTaskService{}).DevicePoll(modelSystemReq.PhoneRegisterDevicePoll{DeviceID: "device-a"})
	require.NoError(t, err)
	require.True(t, found)

	current, err := (&PhoneRegisterTaskService{}).countPendingClaimableTasksWithoutHolder()
	require.NoError(t, err)
	require.EqualValues(t, 0, current)
	require.GreaterOrEqual(t, server.count("del"), 1)
}

func TestPendingClaimableTaskCountCacheTTLUsesNextAvailableAt(t *testing.T) {
	now := time.Now()
	soon := now.Add(30 * time.Second)
	later := now.Add(10 * time.Minute)

	require.Equal(t, 5*time.Minute, phoneRegisterPendingClaimableTaskCountCacheTTL(now, nil))
	require.Equal(t, 30*time.Second, phoneRegisterPendingClaimableTaskCountCacheTTL(now, &soon))
	require.Equal(t, 5*time.Minute, phoneRegisterPendingClaimableTaskCountCacheTTL(now, &later))
}

func TestGetOpenAPIDeviceStatsSubtractsReserveAndOpenAPITasks(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 2,
		PhoneRegisterBlockedPrefixes:       "",
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000002",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusSucceeded,
		FinishedAt:     func() *time.Time { t := time.Now(); return &t }(),
		ExpiresAt:      time.Now().Add(time.Hour),
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a", "device-b", "device-c", "device-d", "device-e"}
		},
		func() []string {
			return []string{"device-e"}
		},
	)
	defer restore()

	stats, err := (&PhoneRegisterTaskService{}).GetOpenAPIDeviceStats()
	require.NoError(t, err)
	require.EqualValues(t, 3, stats.Online)
	require.EqualValues(t, 1, stats.Idle)
}

func TestGetOpenAPIDeviceStatsSubtractsPendingOpenAPITasksWhenReserveIsZero(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
		PhoneRegisterBlockedPrefixes:       "",
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	stats, err := (&PhoneRegisterTaskService{}).GetOpenAPIDeviceStats()
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.Online)
	require.EqualValues(t, 0, stats.Idle)
}

func TestGetOpenAPIDeviceStatsSubtractsManualPendingTasks(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
		PhoneRegisterBlockedPrefixes:       "",
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		CreateSource:   modelSystem.PhoneRegisterTaskCreateSourceManual,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	stats, err := (&PhoneRegisterTaskService{}).GetOpenAPIDeviceStats()
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.Online)
	require.EqualValues(t, 0, stats.Idle)
}

func TestGetOpenAPIDeviceStatsDoesNotSubtractFutureDelayedOpenAPITasks(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()
	enabled := true
	future := time.Now().Add(time.Minute)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
		PhoneRegisterBlockedPrefixes:       "",
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000001",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		AvailableAt:    &future,
		ExpiresAt:      future.Add(time.Hour),
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	stats, err := (&PhoneRegisterTaskService{}).GetOpenAPIDeviceStats()
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.Online)
	require.EqualValues(t, 1, stats.Idle)
}

func TestCreateOpenAPITaskFailsWhenReserveConsumesCapacity(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 1,
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend, PhoneRegisterTaskCreateOptions{
		TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI,
	})
	require.EqualError(t, err, "OpenAPI可用设备不足")
}

func TestCreateDelayedOpenAPITaskFailsWhenReserveCapacityExhausted(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 1,
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend, PhoneRegisterTaskCreateOptions{
		TaskSource:        modelSystem.PhoneRegisterTaskSourceOpenAPI,
		StartDelaySeconds: 60,
		ReserveDevice:     true,
	})
	require.EqualError(t, err, "OpenAPI可用设备不足")
}

func TestCreateDelayedOpenAPITaskFailsWhenReserveDeviceUnavailableAfterCapacityCheck(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	server := newFakeRedisServer(t, nil)
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
		resetPhoneRegisterPendingClaimableTaskCountCache()
	})
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend, PhoneRegisterTaskCreateOptions{
		TaskSource:        modelSystem.PhoneRegisterTaskSourceOpenAPI,
		StartDelaySeconds: 60,
		ReserveDevice:     true,
	})
	require.EqualError(t, err, "OpenAPI可用设备不足")
}

func TestCreateOpenAPITaskFailsWhenPendingOpenAPITasksConsumeIdleCapacity(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000002",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend, PhoneRegisterTaskCreateOptions{
		TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI,
	})
	require.EqualError(t, err, "OpenAPI可用设备不足")
}

func TestCreateOpenAPITaskFailsWhenManualPendingTasksConsumeIdleCapacity(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000002",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		CreateSource:   modelSystem.PhoneRegisterTaskCreateSourceManual,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	_, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend, PhoneRegisterTaskCreateOptions{
		TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI,
	})
	require.EqualError(t, err, "OpenAPI可用设备不足")
}

func TestCreateOpenAPITaskIgnoresFutureDelayedPendingCapacity(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	future := time.Now().Add(time.Minute)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000002",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusPending,
		AvailableAt:    &future,
		ExpiresAt:      future.Add(time.Hour),
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend, PhoneRegisterTaskCreateOptions{
		TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI,
	})
	require.NoError(t, err)
	require.NotZero(t, task.ID)
}

func TestCreateOpenAPITaskDoesNotSubtractReserveWhenReserveIsZero(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 0,
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend, PhoneRegisterTaskCreateOptions{
		TaskSource: modelSystem.PhoneRegisterTaskSourceOpenAPI,
	})
	require.NoError(t, err)
	require.NotZero(t, task.ID)
}

func TestCreateManualTaskIgnoresOpenAPIReserveCapacity(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	createPhoneRegisterTaskTestPromoter(t, 1)
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 1,
	}).Error)
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			return []string{"device-a"}
		},
		func() []string {
			return nil
		},
	)
	defer restore()

	task, err := (&PhoneRegisterTaskService{}).CreateTask(1, "18800000001", modelSystem.PhoneRegisterSMSModePlatformSend)
	require.NoError(t, err)
	require.NotZero(t, task.ID)
}

func TestGetCurrentDeviceStatsRefreshesOnceUnderConcurrentMiss(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	resetPhoneRegisterDeviceStatsCache()
	defer resetPhoneRegisterDeviceStatsCache()

	started := make(chan struct{}, 5)
	release := make(chan struct{})
	var onlineCalls int
	var busyCalls int
	restore := stubPhoneRegisterDeviceIDs(
		func() []string {
			onlineCalls++
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return []string{"device-a", "device-b"}
		},
		func() []string {
			busyCalls++
			return []string{"device-b"}
		},
	)
	defer restore()

	start := make(chan struct{})
	results := make([]phoneRegisterDeviceStats, 5)
	errs := make([]error, 5)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], errs[index] = (&PhoneRegisterTaskService{}).GetCurrentDeviceStats()
		}(i)
	}

	close(start)
	<-started
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	for i := range results {
		require.NoError(t, errs[i])
		require.EqualValues(t, 2, results[i].Online)
		require.EqualValues(t, 1, results[i].Idle)
	}
	require.Equal(t, 1, onlineCalls)
	require.Equal(t, 1, busyCalls)
}

func TestOpenAPIReportSuccessDoesNotRiskBeforeWarmup(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	restore := stubPhoneRegisterRiskRandom(0)
	defer restore()

	createPhoneRegisterRiskUser(t, 1, 45)
	now := time.Now()
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	for i := 0; i < phoneRegisterRiskWarmupSuccessCount-1; i++ {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
			Phone:          "18800000000",
			PromoterID:     1,
			SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
			TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
			Status:         modelSystem.PhoneRegisterStatusSucceeded,
			StatusCode:     &successCode,
			FinishedAt:     &now,
			ExpiresAt:      now.Add(time.Hour),
		}).Error)
	}

	holderDeviceID := "openapi-warmup-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000099",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, err := (&PhoneRegisterTaskService{}).OpenAPIReportSuccess(holderDeviceID, task.ID)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusSucceeded, got.Status)
	require.NotNil(t, got.StatusCode)
	require.Equal(t, modelSystem.PhoneRegisterStatusCodeSucceeded, *got.StatusCode)
}

func TestGetSummaryIncludesRiskFailCountForPromoters(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	leaderID := uint(2)
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	riskFaceCode := modelSystem.PhoneRegisterStatusCodeRiskFace
	riskQuotaCode := modelSystem.PhoneRegisterStatusCodeRiskQuota
	realFailCode := modelSystem.PhoneRegisterStatusCodeDeviceExecFail

	tasks := []modelSystem.SysPhoneRegisterTask{
		{Phone: "18800000001", PromoterID: 1, LeaderID: &leaderID, Status: modelSystem.PhoneRegisterStatusSucceeded, StatusCode: &successCode, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{Phone: "18800000002", PromoterID: 1, LeaderID: &leaderID, Status: modelSystem.PhoneRegisterStatusFailed, StatusCode: &riskFaceCode, LastError: phoneRegisterRiskReasonFace, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{Phone: "18800000003", PromoterID: 1, LeaderID: &leaderID, Status: modelSystem.PhoneRegisterStatusFailed, StatusCode: &riskQuotaCode, LastError: phoneRegisterRiskReasonQuota, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{Phone: "18800000004", PromoterID: 1, LeaderID: &leaderID, Status: modelSystem.PhoneRegisterStatusFailed, StatusCode: &realFailCode, LastError: "真实失败", FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
	}
	require.NoError(t, global.GVA_DB.Create(&tasks).Error)

	got, err := (&PhoneRegisterTaskService{}).GetSummary(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskSummaryFilter{})
	require.NoError(t, err)
	require.Len(t, got.Promoters, 1)
	require.EqualValues(t, 1, got.Promoters[0].SuccessCount)
	require.EqualValues(t, 3, got.Promoters[0].FailCount)
	require.NotNil(t, got.Promoters[0].RiskFailCount)
	require.EqualValues(t, 2, *got.Promoters[0].RiskFailCount)
	require.Len(t, got.Leaders, 1)
	require.NotNil(t, got.Leaders[0].RiskFailCount)
	require.EqualValues(t, 2, *got.Leaders[0].RiskFailCount)
}

func TestPhoneRegisterTaskSummaryOrderIsStable(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	leaderIDs := []uint{30, 10, 20}
	for _, leaderID := range leaderIDs {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
			GVA_MODEL:   global.GVA_MODEL{ID: leaderID},
			Username:    "leader",
			NickName:    "团长",
			AuthorityId: 200,
			Enable:      1,
		}).Error)
		for _, promoterID := range []uint{leaderID + 2, leaderID + 1} {
			require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
				Phone:      "18800000000",
				PromoterID: promoterID,
				LeaderID:   &leaderID,
				Status:     modelSystem.PhoneRegisterStatusSucceeded,
				StatusCode: &successCode,
				FinishedAt: &now,
				ExpiresAt:  now.Add(time.Hour),
			}).Error)
		}
	}

	for i := 0; i < 20; i++ {
		got, err := (&PhoneRegisterTaskService{}).GetSummary(phoneRoleAdmin, 100, modelSystemReq.PhoneRegisterTaskSummaryFilter{})
		require.NoError(t, err)
		require.Equal(t, []uint{10, 20, 30}, phoneSummaryLeaderIDs(got.Leaders))
		require.Equal(t, []uint{11, 12, 21, 22, 31, 32}, phoneSummaryPromoterIDs(got.Promoters))
	}
}

func TestGetSummaryHidesRiskFailCountForLeaderRole(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)

	now := time.Now()
	leaderID := uint(2)
	riskFaceCode := modelSystem.PhoneRegisterStatusCodeRiskFace
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
		Phone:      "18800000002",
		PromoterID: 1,
		LeaderID:   &leaderID,
		Status:     modelSystem.PhoneRegisterStatusFailed,
		StatusCode: &riskFaceCode,
		LastError:  phoneRegisterRiskReasonFace,
		FinishedAt: &now,
		ExpiresAt:  now.Add(time.Hour),
	}).Error)

	got, err := (&PhoneRegisterTaskService{}).GetSummary(phoneRoleLeader, leaderID, modelSystemReq.PhoneRegisterTaskSummaryFilter{})
	require.NoError(t, err)
	require.Len(t, got.Promoters, 1)
	require.EqualValues(t, 1, got.Promoters[0].FailCount)
	require.Nil(t, got.Promoters[0].RiskFailCount)
	require.Len(t, got.Leaders, 1)
	require.Nil(t, got.Leaders[0].RiskFailCount)
	payload, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "riskFailCount")
}

func phoneSummaryLeaderIDs(items []modelSystemRes.PhoneRegisterTaskSummaryItem) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.LeaderID)
	}
	return ids
}

func phoneSummaryPromoterIDs(items []modelSystemRes.PhoneRegisterTaskSummaryItem) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.PromoterID)
	}
	return ids
}

func TestPhoneRegisterRiskReasonDoesNotUseQuota(t *testing.T) {
	require.Equal(t, phoneRegisterRiskReasonFace, phoneRegisterRiskReason(1, "2026-05-22", 11, "", ""))
	require.Equal(t, phoneRegisterRiskReasonFace, phoneRegisterRiskReason(1, "2026-05-22", 12, phoneRegisterRiskReasonFace, phoneRegisterRiskReasonFace))
	require.Equal(t, phoneRegisterRiskReasonFace, phoneRegisterRiskReason(1, "2026-05-22", 13, phoneRegisterRiskReasonQuota, phoneRegisterRiskReasonQuota))
}

func TestOpenAPIReportSuccessWithZeroRatioDoesNotRequireRiskStatTable(t *testing.T) {
	setupPhoneRegisterTaskTestDBWithoutRiskStat(t)

	createPhoneRegisterRiskUser(t, 1, 0)
	now := time.Now()
	holderDeviceID := "openapi-zero-ratio-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000099",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, err := (&PhoneRegisterTaskService{}).OpenAPIReportSuccess(holderDeviceID, task.ID)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusSucceeded, got.Status)
	require.NotNil(t, got.StatusCode)
	require.Equal(t, modelSystem.PhoneRegisterStatusCodeSucceeded, *got.StatusCode)
}

func TestOpenAPIReportSuccessSkipsRiskWhenRiskStatTableMissing(t *testing.T) {
	setupPhoneRegisterTaskTestDBWithoutRiskStat(t)
	restore := stubPhoneRegisterRiskRandom(0)
	defer restore()

	createPhoneRegisterRiskUser(t, 1, 45)
	now := time.Now()
	holderDeviceID := "openapi-missing-risk-table-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000099",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, err := (&PhoneRegisterTaskService{}).OpenAPIReportSuccess(holderDeviceID, task.ID)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusSucceeded, got.Status)
	require.NotNil(t, got.StatusCode)
	require.Equal(t, modelSystem.PhoneRegisterStatusCodeSucceeded, *got.StatusCode)
}

func TestOpenAPIReportSuccessRiskFailureStillAllowsCacheUpload(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	restore := stubPhoneRegisterRiskRandom(0)
	defer restore()

	createPhoneRegisterRiskUser(t, 1, 45)
	now := time.Now()
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	for i := 0; i < phoneRegisterRiskWarmupSuccessCount; i++ {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
			Phone:          "18800000000",
			PromoterID:     1,
			SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
			TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
			Status:         modelSystem.PhoneRegisterStatusSucceeded,
			StatusCode:     &successCode,
			FinishedAt:     &now,
			ExpiresAt:      now.Add(time.Hour),
		}).Error)
	}

	holderDeviceID := "openapi-risk-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000099",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, err := (&PhoneRegisterTaskService{}).OpenAPIReportSuccess(holderDeviceID, task.ID)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, got.Status)
	require.True(t, isPhoneRegisterRiskStatusCode(got.StatusCode))
	require.Contains(t, []string{phoneRegisterRiskReasonFace, phoneRegisterRiskReasonQuota}, got.LastError)
	require.Equal(t, modelSystem.PhoneRegisterCacheStatusPending, got.CacheStatus)
	require.NotNil(t, got.HolderDeviceID)
	require.Equal(t, holderDeviceID, *got.HolderDeviceID)

	var attached modelSystem.SysPhoneRegisterTask
	err = global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var attachErr error
		attached, attachErr = (&PhoneRegisterTaskService{}).AttachOpenAPICacheTx(tx, holderDeviceID, task.ID, 123, "3995613452")
		return attachErr
	})
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, attached.Status)
	require.True(t, isPhoneRegisterRiskStatusCode(attached.StatusCode))
	require.Contains(t, []string{phoneRegisterRiskReasonFace, phoneRegisterRiskReasonQuota}, attached.LastError)
	require.Equal(t, modelSystem.PhoneRegisterCacheStatusUploaded, attached.CacheStatus)
	require.Equal(t, "3995613452", attached.QQNum)
}

func TestRealFailuresDoNotCountTowardRiskWarmup(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	restore := stubPhoneRegisterRiskRandom(0)
	defer restore()

	createPhoneRegisterRiskUser(t, 1, 45)
	now := time.Now()
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	for i := 0; i < phoneRegisterRiskWarmupSuccessCount-1; i++ {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
			Phone:          "18800000000",
			PromoterID:     1,
			SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
			TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
			Status:         modelSystem.PhoneRegisterStatusSucceeded,
			StatusCode:     &successCode,
			FinishedAt:     &now,
			ExpiresAt:      now.Add(time.Hour),
		}).Error)
	}
	realFailCode := modelSystem.PhoneRegisterStatusCodeDeviceExecFail
	for i := 0; i < 20; i++ {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
			Phone:          "18800000001",
			PromoterID:     1,
			SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
			TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
			Status:         modelSystem.PhoneRegisterStatusFailed,
			StatusCode:     &realFailCode,
			LastError:      "真实失败",
			FinishedAt:     &now,
			ExpiresAt:      now.Add(time.Hour),
		}).Error)
	}

	holderDeviceID := "openapi-real-fail-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000099",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, err := (&PhoneRegisterTaskService{}).OpenAPIReportSuccess(holderDeviceID, task.ID)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusSucceeded, got.Status)
	require.NotNil(t, got.StatusCode)
	require.Equal(t, modelSystem.PhoneRegisterStatusCodeSucceeded, *got.StatusCode)
}

func TestRiskRatioChangeTakesEffectOnNextSuccessReport(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	restore := stubPhoneRegisterRiskRandom(0)
	defer restore()

	createPhoneRegisterRiskUser(t, 1, 0)
	now := time.Now()
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	for i := 0; i < phoneRegisterRiskWarmupSuccessCount; i++ {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
			Phone:          "18800000000",
			PromoterID:     1,
			SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
			TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
			Status:         modelSystem.PhoneRegisterStatusSucceeded,
			StatusCode:     &successCode,
			FinishedAt:     &now,
			ExpiresAt:      now.Add(time.Hour),
		}).Error)
	}

	firstDeviceID := "openapi-dynamic-first"
	firstTask := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000091",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &firstDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&firstTask).Error)
	first, err := (&PhoneRegisterTaskService{}).OpenAPIReportSuccess(firstDeviceID, firstTask.ID)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusSucceeded, first.Status)

	require.NoError(t, (&UserService{}).SetUserCacheSampleRatio(1, intPtr(45), true))

	secondDeviceID := "openapi-dynamic-second"
	secondTask := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000092",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceOpenAPI,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &secondDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&secondTask).Error)
	second, err := (&PhoneRegisterTaskService{}).OpenAPIReportSuccess(secondDeviceID, secondTask.ID)
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, second.Status)
	require.True(t, isPhoneRegisterRiskStatusCode(second.StatusCode))
}

func TestDeviceRegisterSuccessRiskFailureStillAllowsDeviceCacheUpload(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	restore := stubPhoneRegisterRiskRandom(0)
	defer restore()

	createPhoneRegisterRiskUser(t, 1, 45)
	now := time.Now()
	successCode := modelSystem.PhoneRegisterStatusCodeSucceeded
	for i := 0; i < phoneRegisterRiskWarmupSuccessCount; i++ {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysPhoneRegisterTask{
			Phone:          "18800000000",
			PromoterID:     1,
			SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
			TaskSource:     modelSystem.PhoneRegisterTaskSourceScript,
			Status:         modelSystem.PhoneRegisterStatusSucceeded,
			StatusCode:     &successCode,
			FinishedAt:     &now,
			ExpiresAt:      now.Add(time.Hour),
		}).Error)
	}

	holderDeviceID := "autox-risk-device"
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18800000099",
		PromoterID:     1,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		TaskSource:     modelSystem.PhoneRegisterTaskSourceScript,
		Status:         modelSystem.PhoneRegisterStatusRunning,
		HolderDeviceID: &holderDeviceID,
		ExpiresAt:      now.Add(time.Hour),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	got, err := (&PhoneRegisterTaskService{}).DeviceReport(modelSystemReq.PhoneRegisterDeviceReport{
		DeviceID: holderDeviceID,
		Action:   modelSystem.PhoneRegisterDeviceActionRegisterSuccess,
		Message:  "注册成功，等待上传缓存",
	})
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, got.Status)
	require.True(t, isPhoneRegisterRiskStatusCode(got.StatusCode))
	require.Equal(t, modelSystem.PhoneRegisterCacheStatusPending, got.CacheStatus)

	var completed modelSystem.SysPhoneRegisterTask
	err = global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var completeErr error
		completed, completeErr = (&PhoneRegisterTaskService{}).CompleteTaskAfterQQCacheUploadTx(tx, holderDeviceID, 123, "3995613452")
		return completeErr
	})
	require.NoError(t, err)
	require.Equal(t, modelSystem.PhoneRegisterStatusFailed, completed.Status)
	require.True(t, isPhoneRegisterRiskStatusCode(completed.StatusCode))
	require.Contains(t, []string{phoneRegisterRiskReasonFace, phoneRegisterRiskReasonQuota}, completed.LastError)
	require.Equal(t, modelSystem.PhoneRegisterCacheStatusUploaded, completed.CacheStatus)
	require.Equal(t, "3995613452", completed.QQNum)
	require.Nil(t, completed.HolderDeviceID)
}

func createPhoneRegisterRiskUser(t *testing.T, id uint, ratio int) {
	t.Helper()
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: id},
		Username:    "promoter",
		NickName:    "地推",
		AuthorityId: 300,
		Enable:      1,
		OriginSetting: map[string]interface{}{
			cacheSampleRatioKey: ratio,
		},
	}).Error)
}

func stubPhoneRegisterRiskRandom(value float64) func() {
	original := phoneRegisterRiskRandomFloat
	phoneRegisterRiskRandomFloat = func(string) float64 {
		return value
	}
	return func() {
		phoneRegisterRiskRandomFloat = original
	}
}

func stubPhoneRegisterDeviceIDs(online func() []string, busy func() []string) func() {
	originalOnline := phoneRegisterListOnlineDeviceIDs
	originalBusy := phoneRegisterListBusyDeviceIDs
	phoneRegisterListOnlineDeviceIDs = online
	phoneRegisterListBusyDeviceIDs = busy
	return func() {
		phoneRegisterListOnlineDeviceIDs = originalOnline
		phoneRegisterListBusyDeviceIDs = originalBusy
	}
}

func intPtr(v int) *int {
	return &v
}
