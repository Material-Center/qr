package system

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	phoneRegisterTaskTimeout             = 30 * time.Minute
	phoneRegisterLeaseTimeout            = 5 * time.Minute
	phoneRegisterTimeoutScanEvery        = 1 * time.Minute
	phoneRegisterTimeoutScanThrottle     = 5 * time.Second
	phoneRegisterCodeSubmitWindow        = 3 * time.Minute
	phoneRegisterCacheWaitTimeout        = 3 * time.Minute
	reservedClaimGracePeriod             = 30 * time.Second
	phoneRegisterReserveSafetyTTL        = 30 * time.Second
	phoneRegisterDeviceStatsTTL          = 2 * time.Second
	phoneRegisterPendingCountCacheMaxTTL = 5 * time.Minute
	phoneRegisterOpenAPICacheCooldown    = 5 * time.Minute

	phoneRoleSuperAdmin = uint(888)
	phoneRoleAdmin      = uint(100)
	phoneRoleLeader     = uint(200)
	phoneRolePromoter   = uint(300)
)

const phoneRegisterOpenAPICacheTimeoutLog = "OpenAPI缓存上传超时未上传"

const OpenAPIDeviceCapacityNotEnoughCode = "OPENAPI_DEVICE_CAPACITY_NOT_ENOUGH"

var ErrOpenAPIDeviceCapacityNotEnough = errors.New("OpenAPI可用设备不足")

const phoneRegisterPendingClaimableTaskCountCacheKey = "phone_register:pending_claimable_count"

const (
	phoneRegisterRiskWarmupSuccessCount = 10
	phoneRegisterRiskMaxRatio           = 45
	phoneRegisterRiskReasonFace         = "人脸"
	phoneRegisterRiskReasonQuota        = "满额"
)

const phoneRegisterDeviceBusyBusiness = "phone_register"
const phoneRegisterReservationBusyPrefix = "phone_register_reserved:"

var defaultPhoneRegisterBlockedPrefixes = []string{"133", "149", "153", "173", "177", "180", "181", "189", "190", "193", "199"}

type PhoneRegisterTaskService struct{}

type phoneRegisterTaskListResult struct {
	List       []systemRes.PhoneRegisterTaskListItem
	Total      int64
	Success    int64
	Failed     int64
	Processing int64
	Device     phoneRegisterDeviceStats
}

type phoneRegisterTaskSettleResult struct {
	SettledAt    time.Time
	SettledCount int64
}

type phoneRegisterDeviceStats struct {
	Online int64
	Idle   int64
}

type PhoneRegisterTaskCreateOptions struct {
	TaskSource        string
	StartDelaySeconds int
	ReserveDevice     bool
}

func phoneRegisterTaskCreateSourceFromOptions(options PhoneRegisterTaskCreateOptions) string {
	if strings.TrimSpace(options.TaskSource) == system.PhoneRegisterTaskSourceOpenAPI {
		return system.PhoneRegisterTaskCreateSourceOpenAPI
	}
	return system.PhoneRegisterTaskCreateSourceManual
}

var phoneRegisterTaskDaemonOnce sync.Once

var phoneRegisterRiskRandomFloat = defaultPhoneRegisterRiskRandomFloat

var phoneRegisterListOnlineDeviceIDs = func() []string {
	return (&DeviceService{}).ListOnlineDeviceIDs()
}

var phoneRegisterListBusyDeviceIDs = func() []string {
	return (&DeviceService{}).ListBusyDeviceIDs()
}

var phoneRegisterListCooldownDeviceIDs = func() []string {
	return (&DeviceService{}).ListCooldownDeviceIDs()
}

var phoneRegisterMarkDeviceCooldown = func(deviceID string, ttl time.Duration) error {
	return (&DeviceService{}).MarkCooldown(deviceID, ttl)
}

var phoneRegisterDeviceStatsCache struct {
	sync.Mutex
	stats     phoneRegisterDeviceStats
	expiresAt time.Time
}

var phoneRegisterOpenAPICreateCapacityMu sync.Mutex

var phoneRegisterTimeoutScanThrottleState struct {
	sync.Mutex
	lastRun time.Time
}

type phoneRegisterDeviceTaskLookupResult struct {
	task  system.SysPhoneRegisterTask
	found bool
}

var phoneRegisterDeviceTaskLookupGroup singleflight.Group

type phoneRegisterRiskDecision struct {
	Hit        bool
	Ratio      int
	Seq        int64
	StatusCode int
	Reason     string
}

func init() {
	startPhoneRegisterTaskDaemon()
}

func startPhoneRegisterTaskDaemon() {
	phoneRegisterTaskDaemonOnce.Do(func() {
		go func() {
			svc := &PhoneRegisterTaskService{}
			_ = svc.timeoutUnfinishedTasks()
			ticker := time.NewTicker(phoneRegisterTimeoutScanEvery)
			defer ticker.Stop()
			for range ticker.C {
				_ = svc.timeoutUnfinishedTasks()
			}
		}()
	})
}

func (s *PhoneRegisterTaskService) CreateTask(promoterID uint, phone string, smsReceiveMode string, optionList ...PhoneRegisterTaskCreateOptions) (system.SysPhoneRegisterTask, error) {
	var options PhoneRegisterTaskCreateOptions
	if len(optionList) > 0 {
		options = optionList[0]
	}
	if options.StartDelaySeconds < 0 {
		return system.SysPhoneRegisterTask{}, errors.New("startDelaySeconds不能小于0")
	}
	if options.StartDelaySeconds > 600 {
		return system.SysPhoneRegisterTask{}, errors.New("startDelaySeconds不能超过600")
	}
	phone = strings.TrimSpace(phone)
	smsReceiveMode = normalizePhoneRegisterSMSMode(smsReceiveMode)
	if phone == "" {
		return system.SysPhoneRegisterTask{}, errors.New("手机号不能为空")
	}
	if !isValidPhoneRegisterTaskPhone(phone) {
		return system.SysPhoneRegisterTask{}, errors.New("手机号必须为11位数字")
	}
	blocked, err := s.IsPhoneBlocked(phone)
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	if blocked {
		return system.SysPhoneRegisterTask{}, errors.New("该手机号段暂不支持提交")
	}
	if !isValidPhoneRegisterSMSMode(smsReceiveMode) {
		return system.SysPhoneRegisterTask{}, errors.New("不支持的收码方式")
	}
	enabled, err := s.IsSubmitEnabled()
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	if !enabled {
		return system.SysPhoneRegisterTask{}, errors.New("手机号注册已关闭")
	}
	if err := s.ensureTaskCreationModeEnabled(smsReceiveMode); err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	openAPIReserveDevices := int64(0)
	createSource := phoneRegisterTaskCreateSourceFromOptions(options)
	if createSource == system.PhoneRegisterTaskCreateSourceOpenAPI {
		openAPIReserveDevices, err = s.GetOpenAPIReserveDeviceCount()
		if err != nil {
			return system.SysPhoneRegisterTask{}, err
		}
		if options.StartDelaySeconds <= 0 {
			phoneRegisterOpenAPICreateCapacityMu.Lock()
			defer phoneRegisterOpenAPICreateCapacityMu.Unlock()
			if err := s.ensureOpenAPICreateCapacityWithReserve(openAPIReserveDevices); err != nil {
				return system.SysPhoneRegisterTask{}, err
			}
		}
	}

	var promoter system.SysUser
	if err = global.GVA_DB.Select("id, leader_id, phone_register_task_disabled").Where("id = ?", promoterID).First(&promoter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return system.SysPhoneRegisterTask{}, errors.New("地推账号不存在")
		}
		return system.SysPhoneRegisterTask{}, err
	}
	if promoter.PhoneRegisterTaskDisabled != nil && *promoter.PhoneRegisterTaskDisabled {
		return system.SysPhoneRegisterTask{}, errors.New("当前账号已禁用任务创建")
	}

	now := time.Now()
	var availableAt *time.Time
	expiresBase := now
	if options.StartDelaySeconds > 0 {
		t := now.Add(time.Duration(options.StartDelaySeconds) * time.Second)
		availableAt = &t
		expiresBase = t
	}

	var reservedDeviceID string
	var reservationToken string
	if options.ReserveDevice && options.StartDelaySeconds > 0 {
		canTryReserveDevice := true
		if createSource == system.PhoneRegisterTaskCreateSourceOpenAPI {
			phoneRegisterOpenAPICreateCapacityMu.Lock()
			capacityErr := s.ensureOpenAPICreateCapacityWithReserve(openAPIReserveDevices)
			phoneRegisterOpenAPICreateCapacityMu.Unlock()
			if errors.Is(capacityErr, ErrOpenAPIDeviceCapacityNotEnough) {
				canTryReserveDevice = false
			} else if capacityErr != nil {
				return system.SysPhoneRegisterTask{}, capacityErr
			}
		}
		if canTryReserveDevice {
			reservationToken = fmt.Sprintf("%s%d", phoneRegisterReservationBusyPrefix, now.UnixNano())
			reserveTTL := time.Duration(options.StartDelaySeconds)*time.Second + reservedClaimGracePeriod + phoneRegisterReserveSafetyTTL
			deviceID, reserveErr := (&DeviceService{}).TryReserveIdleDevice(reservationToken, reserveTTL)
			if reserveErr != nil {
				return system.SysPhoneRegisterTask{}, reserveErr
			}
			reservedDeviceID = deviceID
		}
	}

	var holderDeviceID *string
	if reservedDeviceID != "" {
		holderDeviceID = &reservedDeviceID
	}
	task := system.SysPhoneRegisterTask{
		Phone:          phone,
		PromoterID:     promoterID,
		LeaderID:       promoter.LeaderID,
		SMSReceiveMode: smsReceiveMode,
		CreateSource:   createSource,
		Status:         system.PhoneRegisterStatusPending,
		HolderDeviceID: holderDeviceID,
		AvailableAt:    availableAt,
		ExpiresAt:      expiresBase.Add(phoneRegisterTaskTimeout),
	}
	if err := global.GVA_DB.Create(&task).Error; err != nil {
		if reservedDeviceID != "" {
			_ = (&DeviceService{}).ClearBusy(reservedDeviceID, reservationToken)
		}
		return system.SysPhoneRegisterTask{}, err
	}
	resetPhoneRegisterPendingClaimableTaskCountCache()
	if reservedDeviceID != "" {
		finalBusiness := phoneRegisterReservationBusyBusiness(task.ID)
		reserveTTL := time.Until(task.ExpiresAt)
		if reserveTTL < reservedClaimGracePeriod+phoneRegisterReserveSafetyTTL {
			reserveTTL = reservedClaimGracePeriod + phoneRegisterReserveSafetyTTL
		}
		if err := (&DeviceService{}).UpdateBusyIfMatching(reservedDeviceID, reservationToken, finalBusiness, reserveTTL); err != nil {
			finishedAt := time.Now()
			statusCode := system.PhoneRegisterStatusCodeUnknown
			_ = global.GVA_DB.Model(&task).Updates(map[string]interface{}{
				"status":           system.PhoneRegisterStatusFailed,
				"status_code":      &statusCode,
				"last_error":       "预占设备失败",
				"finished_at":      &finishedAt,
				"holder_device_id": nil,
			}).Error
			_ = (&DeviceService{}).ClearBusy(reservedDeviceID, reservationToken)
			_ = (&DeviceService{}).ClearBusy(reservedDeviceID, finalBusiness)
			resetPhoneRegisterPendingClaimableTaskCountCache()
			return system.SysPhoneRegisterTask{}, errors.New("预占设备失败")
		}
	}
	return task, nil
}

func phoneRegisterReservationBusyBusiness(taskID uint) string {
	return fmt.Sprintf("%s%d", phoneRegisterReservationBusyPrefix, taskID)
}

func isValidPhoneRegisterTaskPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	for _, ch := range phone {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func (s *PhoneRegisterTaskService) IsSubmitEnabled() (bool, error) {
	var cfg system.SysRegisterConfig
	err := global.GVA_DB.Select("phone_register_enabled").
		Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return cfg.PhoneRegisterEnabled == nil || *cfg.PhoneRegisterEnabled, nil
}

func (s *PhoneRegisterTaskService) ensureTaskCreationModeEnabled(smsReceiveMode string) error {
	var cfg system.SysRegisterConfig
	err := global.GVA_DB.Select("phone_register_user_sent_task_disabled, phone_register_receive_task_disabled").
		Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	switch smsReceiveMode {
	case system.PhoneRegisterSMSModeUserSentToTX:
		if cfg.PhoneRegisterUserSentTaskDisabled {
			return errors.New("自己发码任务创建已关闭")
		}
	case system.PhoneRegisterSMSModePlatformSend:
		if cfg.PhoneRegisterReceiveTaskDisabled {
			return errors.New("收码任务创建已关闭")
		}
	}
	return nil
}

func (s *PhoneRegisterTaskService) IsSubmitEnabledForUser(userID uint) (bool, string, error) {
	enabled, err := s.IsSubmitEnabled()
	if err != nil {
		return false, "", err
	}
	if !enabled {
		return false, "今日手机号注册已关闭", nil
	}
	var user system.SysUser
	if err := global.GVA_DB.Select("id, phone_register_task_disabled").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "账号不存在", nil
		}
		return false, "", err
	}
	if user.PhoneRegisterTaskDisabled != nil && *user.PhoneRegisterTaskDisabled {
		return false, "当前账号已禁用任务创建", nil
	}
	return true, "", nil
}

func (s *PhoneRegisterTaskService) IsPhoneBlocked(phone string) (bool, error) {
	var cfg system.SysRegisterConfig
	err := global.GVA_DB.Select("phone_register_blocked_prefixes").
		Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return isBlockedPhoneRegisterPhone(phone, defaultPhoneRegisterBlockedPrefixes), nil
	}
	if err != nil {
		return false, err
	}
	return isBlockedPhoneRegisterPhone(phone, phoneRegisterBlockedPrefixesFromConfig(cfg.PhoneRegisterBlockedPrefixes)), nil
}

func (s *PhoneRegisterTaskService) SubmitCode(promoterID uint, req systemReq.PhoneRegisterTaskSubmitCode) (system.SysPhoneRegisterTask, error) {
	if req.TaskID == 0 {
		return system.SysPhoneRegisterTask{}, errors.New("任务ID不能为空")
	}
	verifyCode := strings.TrimSpace(req.VerifyCode)
	if verifyCode == "" {
		return system.SysPhoneRegisterTask{}, errors.New("验证码不能为空")
	}
	_ = s.timeoutUnfinishedTasksThrottled()

	var task system.SysPhoneRegisterTask
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND promoter_id = ?", req.TaskID, promoterID).
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("任务不存在")
			}
			return err
		}
		if isPhoneRegisterTaskTerminal(task.Status, task.FinishedAt) {
			return errors.New("任务已完成")
		}
		if !time.Now().Before(task.ExpiresAt) {
			if err := s.failTaskTx(tx, &task, system.PhoneRegisterStatusCodeTaskTimeout, "任务总超时"); err != nil {
				return err
			}
			return errors.New("任务已超时")
		}
		if task.SMSReceiveMode != system.PhoneRegisterSMSModePlatformSend {
			return errors.New("当前任务收码方式不支持提交验证码")
		}
		if task.Status != system.PhoneRegisterStatusWaitingPromoterCode {
			return errors.New("当前任务未处于待地推验证码状态")
		}
		if task.CodeRequestedAt != nil && time.Now().After(task.CodeRequestedAt.Add(phoneRegisterCodeSubmitWindow)) {
			if err := s.failTaskTx(tx, &task, system.PhoneRegisterStatusCodeVerifyCodeTimeout, "验证码等待超时"); err != nil {
				return err
			}
			return errors.New("验证码已超时")
		}
		task.PendingCode = verifyCode
		task.LastError = "地推已提交验证码，等待设备消费"
		return tx.Model(&task).Select("pending_code", "last_error", "updated_at").Updates(task).Error
	})
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	return task, nil
}

func (s *PhoneRegisterTaskService) GetActiveTask(promoterID uint) (system.SysPhoneRegisterTask, error) {
	_ = s.timeoutUnfinishedTasksThrottled()
	var task system.SysPhoneRegisterTask
	err := global.GVA_DB.Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Order("id desc").
		First(&task).Error
	return task, err
}

func (s *PhoneRegisterTaskService) GetActiveTasks(promoterID uint) ([]system.SysPhoneRegisterTask, error) {
	_ = s.timeoutUnfinishedTasksThrottled()
	var tasks []system.SysPhoneRegisterTask
	err := global.GVA_DB.Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Order("id desc").
		Find(&tasks).Error
	return tasks, err
}

func (s *PhoneRegisterTaskService) GetTaskForPromoter(promoterID uint, taskID uint) (system.SysPhoneRegisterTask, error) {
	_ = s.timeoutUnfinishedTasksThrottled()
	var task system.SysPhoneRegisterTask
	err := global.GVA_DB.Where("id = ? AND promoter_id = ?", taskID, promoterID).First(&task).Error
	return task, err
}

func (s *PhoneRegisterTaskService) GetTaskList(operatorRole uint, operatorID uint, req systemReq.PhoneRegisterTaskList) (phoneRegisterTaskListResult, error) {
	_ = s.timeoutUnfinishedTasksThrottled()
	req.DayScoped = shouldUsePhoneRegisterTaskDayScoped(operatorRole, req.DayScoped)

	db := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).Preload("Promoter").Preload("Leader")
	var err error
	db, err = applyPhoneRegisterTaskRoleFilter(db, operatorRole, operatorID, req)
	if err != nil {
		return phoneRegisterTaskListResult{}, err
	}
	db = applyPhoneRegisterTaskQueryFilters(db, req)

	var total int64
	if err = db.Count(&total).Error; err != nil {
		return phoneRegisterTaskListResult{}, err
	}

	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 120 {
		pageSize = 10
	}

	var list []system.SysPhoneRegisterTask
	if err = db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return phoneRegisterTaskListResult{}, err
	}

	statDB := global.GVA_DB.Model(&system.SysPhoneRegisterTask{})
	statDB, err = applyPhoneRegisterTaskRoleFilter(statDB, operatorRole, operatorID, req)
	if err != nil {
		return phoneRegisterTaskListResult{}, err
	}
	if req.DayScoped {
		statDB = applyPhoneRegisterTaskDayRangeFilter(statDB, req.FinishedAtStart, req.FinishedAtEnd)
	} else {
		statDB = applyPhoneRegisterTaskFinishedAtRangeFilter(statDB, req.FinishedAtStart, req.FinishedAtEnd)
	}

	type counter struct {
		Success    int64 `gorm:"column:success"`
		Failed     int64 `gorm:"column:failed"`
		Processing int64 `gorm:"column:processing"`
	}
	var stat counter
	if err = statDB.Select(`
		COALESCE(SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END), 0) AS success,
		COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed,
		COALESCE(SUM(CASE WHEN status NOT IN ('succeeded', 'failed') THEN 1 ELSE 0 END), 0) AS processing`,
	).Scan(&stat).Error; err != nil {
		return phoneRegisterTaskListResult{}, err
	}
	deviceStats, err := s.GetCurrentDeviceStats()
	if err != nil {
		return phoneRegisterTaskListResult{}, err
	}
	if operatorRole == phoneRolePromoter {
		deviceStats, err = s.GetPromoterVisibleDeviceStats()
		if err != nil {
			return phoneRegisterTaskListResult{}, err
		}
	}

	return phoneRegisterTaskListResult{
		List:       buildPhoneRegisterTaskListItems(list, operatorRole != phoneRolePromoter),
		Total:      total,
		Success:    stat.Success,
		Failed:     stat.Failed,
		Processing: stat.Processing,
		Device:     deviceStats,
	}, nil
}

func (s *PhoneRegisterTaskService) GetCurrentDeviceStats() (phoneRegisterDeviceStats, error) {
	if global.GVA_DB == nil {
		return phoneRegisterDeviceStats{}, nil
	}
	now := time.Now()
	phoneRegisterDeviceStatsCache.Lock()
	defer phoneRegisterDeviceStatsCache.Unlock()
	if now.Before(phoneRegisterDeviceStatsCache.expiresAt) {
		stats := phoneRegisterDeviceStatsCache.stats
		return stats, nil
	}

	onlineDevices := map[string]struct{}{}
	for _, deviceID := range phoneRegisterListOnlineDeviceIDs() {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID != "" {
			onlineDevices[deviceID] = struct{}{}
		}
	}
	for _, deviceID := range phoneRegisterListCooldownDeviceIDs() {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID != "" {
			delete(onlineDevices, deviceID)
		}
	}

	busyDevices := map[string]struct{}{}
	for _, deviceID := range phoneRegisterListBusyDeviceIDs() {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID != "" {
			busyDevices[deviceID] = struct{}{}
		}
	}

	var idle int64
	for deviceID := range onlineDevices {
		if _, busy := busyDevices[deviceID]; !busy {
			idle++
		}
	}
	stats := phoneRegisterDeviceStats{
		Online: int64(len(onlineDevices)),
		Idle:   idle,
	}
	phoneRegisterDeviceStatsCache.stats = stats
	phoneRegisterDeviceStatsCache.expiresAt = time.Now().Add(phoneRegisterDeviceStatsTTL)
	return stats, nil
}

func (s *PhoneRegisterTaskService) GetPromoterVisibleDeviceStats() (phoneRegisterDeviceStats, error) {
	stats, err := s.GetCurrentDeviceStats()
	if err != nil {
		return phoneRegisterDeviceStats{}, err
	}
	pendingTasks, err := s.countPendingClaimableTasksWithoutHolder()
	if err != nil {
		return phoneRegisterDeviceStats{}, err
	}
	stats.Idle = maxInt64(stats.Idle-pendingTasks, 0)
	return stats, nil
}

func (s *PhoneRegisterTaskService) GetOpenAPIDeviceStats() (phoneRegisterDeviceStats, error) {
	stats, err := s.GetCurrentDeviceStats()
	if err != nil {
		return phoneRegisterDeviceStats{}, err
	}
	reserveDevices, err := s.GetOpenAPIReserveDeviceCount()
	if err != nil {
		return phoneRegisterDeviceStats{}, err
	}
	pendingTasks, err := s.countPendingClaimableTasksWithoutHolder()
	if err != nil {
		return phoneRegisterDeviceStats{}, err
	}
	stats.Online = maxInt64(stats.Online-reserveDevices, 0)
	stats.Idle = maxInt64(stats.Idle-reserveDevices-pendingTasks, 0)
	return stats, nil
}

func (s *PhoneRegisterTaskService) GetOpenAPIReserveDeviceCount() (int64, error) {
	if global.GVA_DB == nil {
		return 0, nil
	}
	var cfg system.SysRegisterConfig
	err := global.GVA_DB.Select("phone_register_open_api_reserve_devices").
		Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if cfg.PhoneRegisterOpenAPIReserveDevices < 0 {
		return 0, nil
	}
	return cfg.PhoneRegisterOpenAPIReserveDevices, nil
}

func (s *PhoneRegisterTaskService) ensureOpenAPICreateCapacityWithReserve(reserveDevices int64) error {
	stats, err := s.GetCurrentDeviceStats()
	if err != nil {
		return err
	}
	pendingTasks, err := s.countPendingClaimableTasksWithoutHolder()
	if err != nil {
		return err
	}
	if reserveDevices <= 0 && global.GVA_REDIS == nil && stats.Online == 0 && pendingTasks == 0 {
		return nil
	}
	if stats.Idle-reserveDevices-pendingTasks <= 0 {
		return ErrOpenAPIDeviceCapacityNotEnough
	}
	return nil
}

func (s *PhoneRegisterTaskService) countPendingClaimableTasksWithoutHolder() (int64, error) {
	if global.GVA_DB == nil {
		return 0, nil
	}
	if global.GVA_REDIS != nil {
		raw, err := global.GVA_REDIS.Get(context.Background(), phoneRegisterPendingClaimableTaskCountCacheKey).Result()
		if err == nil {
			count, parseErr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if parseErr == nil {
				return maxInt64(count, 0), nil
			}
		}
	}
	var count int64
	now := time.Now()
	err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("status = ? AND finished_at IS NULL", system.PhoneRegisterStatusPending).
		Where("holder_device_id IS NULL").
		Where("(available_at IS NULL OR available_at <= ?)", now).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	var nextTask system.SysPhoneRegisterTask
	nextErr := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Select("available_at").
		Where("status = ? AND finished_at IS NULL", system.PhoneRegisterStatusPending).
		Where("holder_device_id IS NULL").
		Where("available_at > ?", now).
		Order("available_at asc").
		First(&nextTask).Error
	if nextErr != nil && !errors.Is(nextErr, gorm.ErrRecordNotFound) {
		return 0, nextErr
	}
	cacheTTL := phoneRegisterPendingClaimableTaskCountCacheTTL(now, nextTask.AvailableAt)
	if global.GVA_REDIS != nil {
		_ = global.GVA_REDIS.Set(
			context.Background(),
			phoneRegisterPendingClaimableTaskCountCacheKey,
			strconv.FormatInt(count, 10),
			cacheTTL,
		).Err()
	}
	return count, err
}

func phoneRegisterPendingClaimableTaskCountCacheTTL(now time.Time, nextAvailableAt *time.Time) time.Duration {
	if nextAvailableAt != nil {
		until := nextAvailableAt.Sub(now)
		if until > 0 && until < phoneRegisterPendingCountCacheMaxTTL {
			return until
		}
	}
	return phoneRegisterPendingCountCacheMaxTTL
}

func resetPhoneRegisterPendingClaimableTaskCountCache() {
	if global.GVA_REDIS == nil {
		return
	}
	_ = global.GVA_REDIS.Del(context.Background(), phoneRegisterPendingClaimableTaskCountCacheKey).Err()
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func resetPhoneRegisterDeviceStatsCache() {
	phoneRegisterDeviceStatsCache.Lock()
	phoneRegisterDeviceStatsCache.stats = phoneRegisterDeviceStats{}
	phoneRegisterDeviceStatsCache.expiresAt = time.Time{}
	phoneRegisterDeviceStatsCache.Unlock()
}

func buildPhoneRegisterTaskListItems(tasks []system.SysPhoneRegisterTask, includeQQNum bool) []systemRes.PhoneRegisterTaskListItem {
	items := make([]systemRes.PhoneRegisterTaskListItem, 0, len(tasks))
	for i := range tasks {
		task := tasks[i]
		item := systemRes.PhoneRegisterTaskListItem{
			ID:              task.ID,
			CreatedAt:       task.CreatedAt,
			Phone:           task.Phone,
			SMSReceiveMode:  task.SMSReceiveMode,
			CreateSource:    task.CreateSource,
			TaskSource:      task.TaskSource,
			CacheStatus:     task.CacheStatus,
			Status:          task.Status,
			StatusCode:      task.StatusCode,
			LastError:       task.LastError,
			FinishedAt:      task.FinishedAt,
			SettledAt:       task.SettledAt,
			HolderDeviceID:  task.HolderDeviceID,
			ClaimedAt:       task.ClaimedAt,
			LastHeartbeatAt: task.LastHeartbeatAt,
			AvailableAt:     task.AvailableAt,
			ExpiresAt:       task.ExpiresAt,
		}
		if includeQQNum {
			item.QQNum = task.QQNum
		}
		if task.Promoter.ID != 0 {
			item.Promoter = &systemRes.PhoneRegisterTaskUserBrief{
				ID:       task.Promoter.ID,
				UserName: task.Promoter.Username,
				NickName: task.Promoter.NickName,
			}
		}
		if task.Leader.ID != 0 {
			item.Leader = &systemRes.PhoneRegisterTaskUserBrief{
				ID:       task.Leader.ID,
				UserName: task.Leader.Username,
				NickName: task.Leader.NickName,
			}
		}
		items = append(items, item)
	}
	return items
}

func (s *PhoneRegisterTaskService) GetSummary(operatorRole uint, operatorID uint, req systemReq.PhoneRegisterTaskSummaryFilter) (systemRes.PhoneRegisterTaskSummaryResponse, error) {
	if operatorRole != phoneRoleSuperAdmin && operatorRole != phoneRoleAdmin && operatorRole != phoneRoleLeader {
		return systemRes.PhoneRegisterTaskSummaryResponse{}, errors.New("无权限查看统计")
	}
	_ = s.timeoutUnfinishedTasksThrottled()
	includeRiskFailCount := operatorRole == phoneRoleSuperAdmin || operatorRole == phoneRoleAdmin

	type row struct {
		LeaderID        *uint  `gorm:"column:leader_id"`
		LeaderName      string `gorm:"column:leader_name"`
		PromoterID      uint   `gorm:"column:promoter_id"`
		PromoterName    string `gorm:"column:promoter_name"`
		SuccessCount    int64  `gorm:"column:success_count"`
		FailCount       int64  `gorm:"column:fail_count"`
		RiskFailCount   int64  `gorm:"column:risk_fail_count"`
		ProcessingCount int64  `gorm:"column:processing_count"`
		SettledCount    int64  `gorm:"column:settled_count"`
		UnsettledCount  int64  `gorm:"column:unsettled_count"`
	}

	db := global.GVA_DB.Table("sys_phone_register_tasks t").
		Select(`
			t.leader_id,
			leader.nick_name AS leader_name,
			t.promoter_id,
			promoter.nick_name AS promoter_name,
			COALESCE(SUM(CASE WHEN t.status = 'succeeded' THEN 1 ELSE 0 END), 0) AS success_count,
			COALESCE(SUM(CASE WHEN t.status = 'failed' THEN 1 ELSE 0 END), 0) AS fail_count,
			COALESCE(SUM(CASE WHEN t.status_code IN ? THEN 1 ELSE 0 END), 0) AS risk_fail_count,
			COALESCE(SUM(CASE WHEN t.status NOT IN ('succeeded', 'failed') THEN 1 ELSE 0 END), 0) AS processing_count,
			COALESCE(SUM(CASE WHEN t.status = 'succeeded' AND t.settled_at IS NOT NULL THEN 1 ELSE 0 END), 0) AS settled_count,
			COALESCE(SUM(CASE WHEN t.status = 'succeeded' AND t.settled_at IS NULL THEN 1 ELSE 0 END), 0) AS unsettled_count`, phoneRegisterRiskStatusCodes()).
		Joins("LEFT JOIN sys_users promoter ON promoter.id = t.promoter_id").
		Joins("LEFT JOIN sys_users leader ON leader.id = t.leader_id")

	if operatorRole == phoneRoleLeader {
		db = db.Where("t.leader_id = ?", operatorID)
	} else if req.LeaderID != 0 {
		db = db.Where("t.leader_id = ?", req.LeaderID)
	}
	if shouldUsePhoneRegisterTaskDayScoped(operatorRole, req.DayScoped) {
		db = applyPhoneRegisterTaskDayRangeFilterWithColumns(db, "t.finished_at", "t.created_at", req.FinishedAtStart, req.FinishedAtEnd)
	} else {
		db = applyPhoneRegisterTaskFinishedAtRangeFilterWithColumn(db, "t.finished_at", req.FinishedAtStart, req.FinishedAtEnd)
	}

	var rows []row
	if err := db.Group("t.leader_id, leader.nick_name, t.promoter_id, promoter.nick_name").Scan(&rows).Error; err != nil {
		return systemRes.PhoneRegisterTaskSummaryResponse{}, err
	}

	leaderMap := map[uint]systemRes.PhoneRegisterTaskSummaryItem{}
	promoters := make([]systemRes.PhoneRegisterTaskSummaryItem, 0, len(rows))
	for _, row := range rows {
		item := systemRes.PhoneRegisterTaskSummaryItem{
			LeaderName:      row.LeaderName,
			PromoterID:      row.PromoterID,
			PromoterName:    row.PromoterName,
			SuccessCount:    row.SuccessCount,
			FailCount:       row.FailCount,
			ProcessingCount: row.ProcessingCount,
			SettledCount:    row.SettledCount,
			UnsettledCount:  row.UnsettledCount,
		}
		if includeRiskFailCount {
			riskFailCount := row.RiskFailCount
			item.RiskFailCount = &riskFailCount
		}
		if row.LeaderID != nil {
			item.LeaderID = *row.LeaderID
		}
		promoters = append(promoters, item)
		if item.LeaderID != 0 {
			leader := leaderMap[item.LeaderID]
			leader.LeaderID = item.LeaderID
			leader.LeaderName = item.LeaderName
			leader.SuccessCount += item.SuccessCount
			leader.FailCount += item.FailCount
			if includeRiskFailCount {
				riskFailCount := row.RiskFailCount
				if leader.RiskFailCount != nil {
					riskFailCount += *leader.RiskFailCount
				}
				leader.RiskFailCount = &riskFailCount
			}
			leader.ProcessingCount += item.ProcessingCount
			leader.SettledCount += item.SettledCount
			leader.UnsettledCount += item.UnsettledCount
			leaderMap[item.LeaderID] = leader
		}
	}

	leaders := make([]systemRes.PhoneRegisterTaskSummaryItem, 0, len(leaderMap))
	for _, item := range leaderMap {
		leaders = append(leaders, item)
	}
	sort.Slice(leaders, func(i, j int) bool {
		return leaders[i].LeaderID < leaders[j].LeaderID
	})
	sort.Slice(promoters, func(i, j int) bool {
		if promoters[i].LeaderID != promoters[j].LeaderID {
			return promoters[i].LeaderID < promoters[j].LeaderID
		}
		return promoters[i].PromoterID < promoters[j].PromoterID
	})
	return systemRes.PhoneRegisterTaskSummaryResponse{
		Leaders:   leaders,
		Promoters: promoters,
	}, nil
}

func (s *PhoneRegisterTaskService) SettleLeader(operatorRole uint, operatorID uint, req systemReq.PhoneRegisterTaskSettle) (phoneRegisterTaskSettleResult, error) {
	if operatorRole != phoneRoleSuperAdmin && operatorRole != phoneRoleAdmin {
		return phoneRegisterTaskSettleResult{}, errors.New("仅管理员可结算")
	}
	if req.LeaderID == 0 {
		return phoneRegisterTaskSettleResult{}, errors.New("团长ID不能为空")
	}

	settledAt := time.Now()
	result := phoneRegisterTaskSettleResult{SettledAt: settledAt}
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		base := tx.Model(&system.SysPhoneRegisterTask{}).
			Where("leader_id = ? AND finished_at IS NOT NULL AND finished_at <= ? AND status = ? AND settled_at IS NULL", req.LeaderID, settledAt, system.PhoneRegisterStatusSucceeded)
		base = applyPhoneRegisterTaskFinishedAtRangeFilter(base, req.FinishedAtStart, req.FinishedAtEnd)
		if err := base.Count(&result.SettledCount).Error; err != nil {
			return err
		}
		if result.SettledCount <= 0 {
			return nil
		}
		updateDB := tx.Model(&system.SysPhoneRegisterTask{}).
			Where("leader_id = ? AND finished_at IS NOT NULL AND finished_at <= ? AND status = ? AND settled_at IS NULL", req.LeaderID, settledAt, system.PhoneRegisterStatusSucceeded).
			Scopes(func(db *gorm.DB) *gorm.DB {
				return applyPhoneRegisterTaskFinishedAtRangeFilter(db, req.FinishedAtStart, req.FinishedAtEnd)
			})
		return updateDB.
			Updates(map[string]interface{}{
				"settled_at": settledAt,
				"settled_by": operatorID,
			}).Error
	})
	return result, err
}

func (s *PhoneRegisterTaskService) GetSettlementHistory(operatorRole uint, req systemReq.PhoneRegisterTaskSettlementHistory) ([]systemRes.PhoneRegisterTaskSettlementHistoryItem, error) {
	if operatorRole != phoneRoleSuperAdmin && operatorRole != phoneRoleAdmin {
		return nil, errors.New("仅管理员可查看结算历史")
	}
	if req.LeaderID == 0 {
		return nil, errors.New("团长ID不能为空")
	}

	var rows []systemRes.PhoneRegisterTaskSettlementHistoryItem
	err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Select("settled_at, COUNT(1) AS settled_count").
		Where("leader_id = ? AND settled_at IS NOT NULL AND finished_at IS NOT NULL AND status = ?", req.LeaderID, system.PhoneRegisterStatusSucceeded).
		Group("settled_at").
		Order("settled_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (s *PhoneRegisterTaskService) GetTaskLogs(operatorRole uint, operatorID uint, req systemReq.PhoneRegisterTaskLogList) ([]system.SysPhoneRegisterTaskLog, int64, int, int, error) {
	if req.TaskID == 0 {
		return nil, 0, 0, 0, errors.New("taskId不能为空")
	}

	taskDB := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).Where("id = ?", req.TaskID)
	taskDB, err := applyPhoneRegisterTaskRoleFilter(taskDB, operatorRole, operatorID, systemReq.PhoneRegisterTaskList{})
	if err != nil {
		return nil, 0, 0, 0, err
	}
	var count int64
	if err = taskDB.Count(&count).Error; err != nil {
		return nil, 0, 0, 0, err
	}
	if count == 0 {
		return nil, 0, 0, 0, errors.New("无权限查看任务日志")
	}

	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 100
	}

	db := global.GVA_DB.Model(&system.SysPhoneRegisterTaskLog{}).Where("task_id = ?", req.TaskID)
	var total int64
	if err = db.Count(&total).Error; err != nil {
		return nil, 0, 0, 0, err
	}

	var logs []system.SysPhoneRegisterTaskLog
	if err = db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, 0, 0, err
	}
	return logs, total, page, pageSize, nil
}

func (s *PhoneRegisterTaskService) DevicePoll(req systemReq.PhoneRegisterDevicePoll) (system.SysPhoneRegisterTask, bool, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, false, errors.New("deviceId不能为空")
	}
	_ = (&DeviceService{}).MarkHeartbeat(deviceID)
	_ = s.timeoutUnfinishedTasksThrottled()
	_ = s.releaseExpiredReservations(time.Now())

	var task system.SysPhoneRegisterTask
	found := false
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		existing, ok, err := s.findUniqueOpenTaskByDeviceTx(tx, deviceID, true)
		if err != nil {
			return err
		}
		if ok {
			handled, handledFound, handledTask, err := s.handleHeldTaskForPollTx(tx, existing, deviceID, system.PhoneRegisterTaskSourceScript, time.Now())
			if err != nil {
				return err
			}
			if handled {
				task = handledTask
				found = handledFound
				return nil
			}
		}

		now := time.Now()
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND finished_at IS NULL", system.PhoneRegisterStatusPending).
			Where("holder_device_id IS NULL").
			Where("(available_at IS NULL OR available_at <= ?)", now).
			Order("id asc").
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				task = system.SysPhoneRegisterTask{}
				return nil
			}
			return err
		}
		task.Status = system.PhoneRegisterStatusRunning
		task.TaskSource = system.PhoneRegisterTaskSourceScript
		task.HolderDeviceID = stringPtr(deviceID)
		task.ClaimedAt = &now
		task.LastHeartbeatAt = &now
		task.LastError = ""
		if err := tx.Model(&task).
			Select("status", "task_source", "holder_device_id", "claimed_at", "last_heartbeat_at", "last_error", "updated_at").
			Updates(task).Error; err != nil {
			return err
		}
		found = true
		return nil
	})
	if err == nil && found {
		resetPhoneRegisterPendingClaimableTaskCountCache()
		_ = markPhoneRegisterDeviceBusy(deviceID)
	}
	return task, found, err
}

func (s *PhoneRegisterTaskService) OpenAPIPoll(req systemReq.PhoneRegisterOpenAPITask, verifyMode string) (system.SysPhoneRegisterTask, bool, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, false, errors.New("deviceId不能为空")
	}
	_ = (&DeviceService{}).MarkHeartbeat(deviceID)
	smsReceiveMode, err := phoneRegisterSMSModeFromOpenAPIVerifyMode(verifyMode)
	if err != nil {
		return system.SysPhoneRegisterTask{}, false, err
	}
	_ = s.timeoutUnfinishedTasksThrottled()
	_ = s.releaseExpiredReservations(time.Now())

	var task system.SysPhoneRegisterTask
	found := false
	err = global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		existing, ok, err := s.findUniqueOpenTaskByDeviceTx(tx, deviceID, true)
		if err != nil {
			return err
		}
		if ok {
			handled, handledFound, handledTask, err := s.handleHeldTaskForPollTx(tx, existing, deviceID, system.PhoneRegisterTaskSourceOpenAPI, time.Now())
			if err != nil {
				return err
			}
			if handled {
				task = handledTask
				found = handledFound
				return nil
			}
		}

		now := time.Now()
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND finished_at IS NULL", system.PhoneRegisterStatusPending).
			Where("holder_device_id IS NULL").
			Where("(available_at IS NULL OR available_at <= ?)", now)
		if smsReceiveMode != "" {
			query = query.Where("sms_receive_mode = ?", smsReceiveMode)
		}
		if err := query.Order("id asc").First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				task = system.SysPhoneRegisterTask{}
				return nil
			}
			return err
		}
		task.Status = system.PhoneRegisterStatusRunning
		task.TaskSource = system.PhoneRegisterTaskSourceOpenAPI
		task.HolderDeviceID = stringPtr(deviceID)
		task.ClaimedAt = &now
		task.LastHeartbeatAt = &now
		task.LastError = ""
		if err := tx.Model(&task).
			Select("status", "task_source", "holder_device_id", "claimed_at", "last_heartbeat_at", "last_error", "updated_at").
			Updates(task).Error; err != nil {
			return err
		}
		found = true
		return nil
	})
	if err == nil && found {
		resetPhoneRegisterPendingClaimableTaskCountCache()
		_ = markPhoneRegisterDeviceBusy(deviceID)
	}
	return task, found, err
}

func (s *PhoneRegisterTaskService) handleHeldTaskForPollTx(tx *gorm.DB, task system.SysPhoneRegisterTask, deviceID string, taskSource string, now time.Time) (bool, bool, system.SysPhoneRegisterTask, error) {
	if task.Status != system.PhoneRegisterStatusPending {
		return true, true, task, nil
	}
	if task.AvailableAt != nil && task.AvailableAt.After(now) {
		return true, false, system.SysPhoneRegisterTask{}, nil
	}
	if task.AvailableAt != nil && now.After(task.AvailableAt.Add(reservedClaimGracePeriod)) {
		if err := s.releaseReservationTx(tx, task, now); err != nil {
			return false, false, system.SysPhoneRegisterTask{}, err
		}
		return false, false, system.SysPhoneRegisterTask{}, nil
	}

	task.Status = system.PhoneRegisterStatusRunning
	task.TaskSource = strings.TrimSpace(taskSource)
	task.ClaimedAt = &now
	task.LastHeartbeatAt = &now
	task.LastError = ""
	if err := tx.Model(&task).
		Select("status", "task_source", "claimed_at", "last_heartbeat_at", "last_error", "updated_at").
		Updates(task).Error; err != nil {
		return false, false, system.SysPhoneRegisterTask{}, err
	}
	return true, true, task, nil
}

func (s *PhoneRegisterTaskService) releaseReservationTx(tx *gorm.DB, task system.SysPhoneRegisterTask, now time.Time) error {
	if task.HolderDeviceID == nil {
		return nil
	}
	deviceID := strings.TrimSpace(*task.HolderDeviceID)
	if deviceID == "" {
		return nil
	}
	if err := tx.Model(&system.SysPhoneRegisterTask{}).
		Where("id = ? AND status = ? AND holder_device_id = ?", task.ID, system.PhoneRegisterStatusPending, deviceID).
		Updates(map[string]any{
			"holder_device_id": nil,
			"updated_at":       now,
		}).Error; err != nil {
		return err
	}
	resetPhoneRegisterPendingClaimableTaskCountCache()
	_ = (&DeviceService{}).ClearBusy(deviceID, phoneRegisterReservationBusyBusiness(task.ID))
	return nil
}

func (s *PhoneRegisterTaskService) releaseExpiredReservations(now time.Time) error {
	if global.GVA_DB == nil {
		return nil
	}
	deadline := now.Add(-reservedClaimGracePeriod)
	var tasks []system.SysPhoneRegisterTask
	if err := global.GVA_DB.
		Where("status = ? AND finished_at IS NULL", system.PhoneRegisterStatusPending).
		Where("holder_device_id IS NOT NULL").
		Where("available_at IS NOT NULL AND available_at <= ?", deadline).
		Limit(100).
		Find(&tasks).Error; err != nil {
		return err
	}
	for _, task := range tasks {
		if err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
			return s.releaseReservationTx(tx, task, now)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *PhoneRegisterTaskService) DeviceTask(req systemReq.PhoneRegisterDeviceTask) (system.SysPhoneRegisterTask, bool, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, false, errors.New("deviceId不能为空")
	}
	_ = (&DeviceService{}).MarkHeartbeat(deviceID)
	_ = s.timeoutUnfinishedTasksThrottled()
	task, found, err := s.findUniqueOpenTaskByDevice(deviceID)
	if err == nil && found {
		_ = markPhoneRegisterDeviceBusy(deviceID)
	}
	return task, found, err
}

func (s *PhoneRegisterTaskService) findUniqueOpenTaskByDevice(deviceID string) (system.SysPhoneRegisterTask, bool, error) {
	value, err, _ := phoneRegisterDeviceTaskLookupGroup.Do(deviceID, func() (any, error) {
		task, found, err := s.findUniqueOpenTaskByDeviceTx(global.GVA_DB, deviceID, false)
		if err != nil {
			return phoneRegisterDeviceTaskLookupResult{}, err
		}
		return phoneRegisterDeviceTaskLookupResult{task: task, found: found}, nil
	})
	if err != nil {
		return system.SysPhoneRegisterTask{}, false, err
	}
	result, ok := value.(phoneRegisterDeviceTaskLookupResult)
	if !ok {
		return system.SysPhoneRegisterTask{}, false, errors.New("当前设备任务查询结果异常")
	}
	return result.task, result.found, nil
}

func phoneRegisterSMSModeFromOpenAPIVerifyMode(verifyMode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(verifyMode)) {
	case "":
		return "", nil
	case "receive", "platform_send", strings.ToLower(system.PhoneRegisterSMSModePlatformSend):
		return system.PhoneRegisterSMSModePlatformSend, nil
	case "send", "user_sent_to_tx", strings.ToLower(system.PhoneRegisterSMSModeUserSentToTX):
		return system.PhoneRegisterSMSModeUserSentToTX, nil
	default:
		return "", errors.New("verifyMode仅支持receive/send")
	}
}

func (s *PhoneRegisterTaskService) DeviceHeartbeat(req systemReq.PhoneRegisterDeviceHeartbeat) error {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return errors.New("deviceId不能为空")
	}
	_ = (&DeviceService{}).MarkHeartbeat(deviceID)
	_ = s.timeoutUnfinishedTasksThrottled()
	if err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		task, found, err := s.findUniqueOpenTaskByDeviceTx(tx, deviceID, true)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("当前设备暂无执行中任务")
		}
		if !time.Now().Before(task.ExpiresAt) {
			if err := s.failTaskTx(tx, &task, system.PhoneRegisterStatusCodeTaskTimeout, "任务总超时"); err != nil {
				return err
			}
			return errors.New("任务已超时")
		}
		now := time.Now()
		return tx.Model(&task).Updates(map[string]any{
			"last_heartbeat_at": now,
			"updated_at":        now,
		}).Error
	}); err != nil {
		return err
	}
	return markPhoneRegisterDeviceBusy(deviceID)
}

func (s *PhoneRegisterTaskService) DeviceReport(req systemReq.PhoneRegisterDeviceReport) (system.SysPhoneRegisterTask, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	action := strings.TrimSpace(req.Action)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, errors.New("deviceId不能为空")
	}
	if action == "" {
		return system.SysPhoneRegisterTask{}, errors.New("action不能为空")
	}
	_ = (&DeviceService{}).MarkHeartbeat(deviceID)
	_ = s.timeoutUnfinishedTasksThrottled()

	var task system.SysPhoneRegisterTask
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		current, found, err := s.findUniqueOpenTaskByDeviceTx(tx, deviceID, true)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("当前设备暂无执行中任务")
		}
		task = current
		if !time.Now().Before(task.ExpiresAt) {
			if err := s.failTaskTx(tx, &task, system.PhoneRegisterStatusCodeTaskTimeout, "任务总超时"); err != nil {
				return err
			}
			return errors.New("任务已超时")
		}
		now := time.Now()
		message := strings.TrimSpace(req.Message)
		if message == "" {
			message = phoneRegisterDefaultMessageByAction(action)
		}
		switch action {
		case system.PhoneRegisterDeviceActionEnterWaitingCode:
			if task.SMSReceiveMode != system.PhoneRegisterSMSModePlatformSend {
				return errors.New("当前任务收码方式不支持进入待码状态")
			}
			if task.Status != system.PhoneRegisterStatusWaitingPromoterCode || task.CodeRequestedAt == nil {
				task.CodeRequestedAt = &now
			}
			task.Status = system.PhoneRegisterStatusWaitingPromoterCode
			task.LastError = message
			task.LastHeartbeatAt = &now
			return tx.Model(&task).
				Select("status", "code_requested_at", "last_error", "last_heartbeat_at", "updated_at").
				Updates(task).Error
		case system.PhoneRegisterDeviceActionConsumeCodeOK:
			task.Status = system.PhoneRegisterStatusRunning
			task.PendingCode = ""
			task.CodeRequestedAt = nil
			task.LastError = message
			task.LastHeartbeatAt = &now
			return tx.Model(&task).
				Select("status", "pending_code", "code_requested_at", "last_error", "last_heartbeat_at", "updated_at").
				Updates(task).Error
		case system.PhoneRegisterDeviceActionRegisterSuccess:
			decision, riskErr := s.evaluatePhoneRegisterRiskOnSuccessTx(tx, &task, now)
			if riskErr != nil {
				return riskErr
			}
			if decision.Hit {
				return s.riskFailTaskTx(tx, &task, decision, now)
			}
			task.Status = system.PhoneRegisterStatusRegisteredWaitUpload
			task.CacheStatus = system.PhoneRegisterCacheStatusPending
			task.LastError = message
			task.LastHeartbeatAt = &now
			return tx.Model(&task).
				Select("status", "cache_status", "last_error", "last_heartbeat_at", "updated_at").
				Updates(task).Error
		case system.PhoneRegisterDeviceActionFail:
			code := system.PhoneRegisterStatusCodeDeviceExecFail
			if req.StatusCode != nil && *req.StatusCode != 0 {
				code = *req.StatusCode
			}
			return s.failTaskTx(tx, &task, code, message)
		default:
			return errors.New("不支持的action")
		}
	})
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	if action == system.PhoneRegisterDeviceActionFail {
		_ = markPhoneRegisterDeviceOffline(deviceID)
	} else {
		_ = markPhoneRegisterDeviceBusy(deviceID)
	}
	return task, nil
}

func (s *PhoneRegisterTaskService) OpenAPIReportSuccess(deviceID string, taskID uint) (system.SysPhoneRegisterTask, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, errors.New("deviceId不能为空")
	}
	if taskID == 0 {
		return system.SysPhoneRegisterTask{}, errors.New("taskId不能为空")
	}
	_ = s.timeoutUnfinishedTasksThrottled()

	var task system.SysPhoneRegisterTask
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND task_source = ?", taskID, system.PhoneRegisterTaskSourceOpenAPI).
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("当前设备暂无执行中任务")
			}
			return err
		}
		if task.HolderDeviceID == nil || strings.TrimSpace(*task.HolderDeviceID) != deviceID {
			return errors.New("taskId与当前设备任务不一致")
		}
		if isPhoneRegisterTaskTerminal(task.Status, task.FinishedAt) {
			if task.Status == system.PhoneRegisterStatusSucceeded || isPhoneRegisterRiskStatusCode(task.StatusCode) {
				return nil
			}
			return errors.New("任务已完成")
		}
		if !time.Now().Before(task.ExpiresAt) {
			if err := s.failTaskTx(tx, &task, system.PhoneRegisterStatusCodeTaskTimeout, "任务总超时"); err != nil {
				return err
			}
			return errors.New("任务已超时")
		}
		now := time.Now()
		successCode := system.PhoneRegisterStatusCodeSucceeded
		decision, riskErr := s.evaluatePhoneRegisterRiskOnSuccessTx(tx, &task, now)
		if riskErr != nil {
			return riskErr
		}
		if decision.Hit {
			return s.riskFailTaskTx(tx, &task, decision, now)
		}
		task.Status = system.PhoneRegisterStatusSucceeded
		task.StatusCode = &successCode
		task.CacheStatus = system.PhoneRegisterCacheStatusPending
		task.FinishedAt = &now
		task.LastHeartbeatAt = &now
		task.PendingCode = ""
		task.CodeRequestedAt = nil
		task.LastError = ""
		return tx.Model(&task).
			Select("status", "status_code", "cache_status", "finished_at", "last_heartbeat_at", "pending_code", "code_requested_at", "last_error", "updated_at").
			Updates(task).Error
	})
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	_ = markPhoneRegisterDeviceOffline(deviceID)
	return task, nil
}

func (s *PhoneRegisterTaskService) OpenAPIReportFailure(deviceID string, taskID uint, reason string) (system.SysPhoneRegisterTask, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, errors.New("deviceId不能为空")
	}
	if taskID == 0 {
		return system.SysPhoneRegisterTask{}, errors.New("taskId不能为空")
	}
	_ = (&DeviceService{}).MarkHeartbeat(deviceID)
	_ = s.timeoutUnfinishedTasksThrottled()

	var task system.SysPhoneRegisterTask
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND task_source = ?", taskID, system.PhoneRegisterTaskSourceOpenAPI).
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("当前设备暂无执行中任务")
			}
			return err
		}
		if task.HolderDeviceID == nil || strings.TrimSpace(*task.HolderDeviceID) != deviceID {
			return errors.New("taskId与当前设备任务不一致")
		}
		if isPhoneRegisterTaskTerminal(task.Status, task.FinishedAt) {
			if task.Status == system.PhoneRegisterStatusFailed {
				return nil
			}
			return errors.New("任务已完成")
		}
		if !time.Now().Before(task.ExpiresAt) {
			return s.failTaskTx(tx, &task, system.PhoneRegisterStatusCodeTaskTimeout, "任务总超时")
		}
		now := time.Now()
		statusCode := system.PhoneRegisterStatusCodeOpenAPIFeedback
		message := strings.TrimSpace(reason)
		if message == "" {
			message = "任务失败"
		}
		task.Status = system.PhoneRegisterStatusFailed
		task.StatusCode = &statusCode
		task.LastError = message
		task.FinishedAt = &now
		task.PendingCode = ""
		task.CodeRequestedAt = nil
		return tx.Model(&task).
			Select("status", "status_code", "last_error", "finished_at", "pending_code", "code_requested_at", "updated_at").
			Updates(task).Error
	})
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	_ = markPhoneRegisterDeviceOffline(deviceID)
	return task, nil
}

func (s *PhoneRegisterTaskService) DeviceLog(req systemReq.PhoneRegisterDeviceLog) error {
	deviceID := strings.TrimSpace(req.DeviceID)
	message := strings.TrimSpace(req.Message)
	if deviceID == "" {
		return errors.New("deviceId不能为空")
	}
	if message == "" {
		return errors.New("message不能为空")
	}
	_ = (&DeviceService{}).MarkHeartbeat(deviceID)
	if req.TaskID != 0 {
		_ = markPhoneRegisterDeviceBusy(deviceID)
	}
	var clientTime *time.Time
	if rawClientTime := strings.TrimSpace(req.ClientTime); rawClientTime != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, rawClientTime); err == nil {
			clientTime = &parsed
		}
	}
	global.GVA_LOG.Info("手机号注册设备日志",
		zap.String("deviceId", deviceID),
		zap.Uint("taskId", req.TaskID),
		zap.String("clientTime", strings.TrimSpace(req.ClientTime)),
		zap.String("message", message),
	)
	if req.TaskID != 0 {
		if err := global.GVA_DB.Create(&system.SysPhoneRegisterTaskLog{
			TaskID:     req.TaskID,
			DeviceID:   deviceID,
			ClientTime: clientTime,
			Message:    message,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *PhoneRegisterTaskService) GetDeviceConfig() (systemRes.PhoneRegisterDeviceConfigResponse, error) {
	var cfg system.SysRegisterConfig
	err := global.GVA_DB.Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).First(&cfg).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return systemRes.PhoneRegisterDeviceConfigResponse{}, err
	}
	imageVerify := systemRes.PhoneRegisterImageVerifyConfig{
		Provider: strings.TrimSpace(cfg.PhoneImageProvider),
		Question: "框出正确位置",
		System:   "",
	}
	switch strings.ToLower(strings.TrimSpace(cfg.PhoneImageProvider)) {
	case "tuling":
		imageVerify.Endpoint = "http://www.fdyscloud.com.cn/tuling/predict"
		imageVerify.Username = strings.TrimSpace(cfg.PhoneImageProviderUsername)
		imageVerify.Password = strings.TrimSpace(cfg.PhoneImageProviderPassword)
		imageVerify.RequestID = "42077360"
		imageVerify.Version = "3.1.1"
	case "tujie":
		imageVerify.Endpoint = "http://gpu1.xinyuocr.xyz:8889/api/qrcode/predict"
		imageVerify.ModelName = "普通模型"
		imageVerify.KeyCode = strings.TrimSpace(cfg.PhoneImageProviderSecretKey)
	default:
		imageVerify.Endpoint = ""
	}
	return systemRes.PhoneRegisterDeviceConfigResponse{
		ImageVerify: imageVerify,
	}, nil
}

func (s *PhoneRegisterTaskService) evaluatePhoneRegisterRiskOnSuccessTx(tx *gorm.DB, task *system.SysPhoneRegisterTask, now time.Time) (phoneRegisterRiskDecision, error) {
	if task == nil {
		return phoneRegisterRiskDecision{}, errors.New("任务不存在")
	}
	ratio, err := s.effectivePhoneRegisterRiskRatioTx(tx, task.PromoterID)
	if err != nil {
		return phoneRegisterRiskDecision{}, err
	}
	decision := phoneRegisterRiskDecision{Ratio: ratio}
	if ratio <= 0 {
		return decision, nil
	}

	stat, err := s.loadPhoneRegisterRiskDailyStatTx(tx, task.PromoterID, now)
	if err != nil {
		if isPhoneRegisterRiskStatTableMissingError(err) {
			if global.GVA_LOG != nil {
				global.GVA_LOG.Warn("手机号注册风控统计表不存在，跳过本次风控", zap.Error(err))
			}
			return decision, nil
		}
		return phoneRegisterRiskDecision{}, err
	}
	seq := stat.SuccessReportCount + 1
	decision.Seq = seq
	targetRiskCount := int64(math.Floor(float64(seq*int64(ratio)) / 100))

	shouldHit := false
	if seq > phoneRegisterRiskWarmupSuccessCount && targetRiskCount > stat.RiskFailCount {
		gap := seq - stat.LastRiskSuccessSeq
		minGap := phoneRegisterRiskMinGap(ratio, task.PromoterID, stat.BizDate, seq)
		if stat.LastRiskSuccessSeq == 0 || gap > minGap {
			probability := phoneRegisterRiskHitProbability(ratio, seq, stat.RiskFailCount, targetRiskCount, gap)
			shouldHit = phoneRegisterRiskRandomFloat(fmt.Sprintf("hit:%d:%s:%d:%d", task.PromoterID, stat.BizDate, seq, task.ID)) < probability
			if shouldHit && stat.LastRiskGap > 0 && gap == stat.LastRiskGap && gap == stat.PreviousRiskGap {
				shouldHit = false
			}
		}
	}

	updates := map[string]any{
		"success_report_count": seq,
		"updated_at":           now,
	}
	if shouldHit {
		reason := phoneRegisterRiskReason(task.PromoterID, stat.BizDate, seq, stat.LastRiskReason, stat.PreviousRiskReason)
		statusCode := system.PhoneRegisterStatusCodeRiskFace
		decision.Hit = true
		decision.StatusCode = statusCode
		decision.Reason = reason
		gap := seq - stat.LastRiskSuccessSeq
		updates["risk_fail_count"] = stat.RiskFailCount + 1
		updates["last_risk_success_seq"] = seq
		updates["last_risk_reason"] = reason
		updates["last_risk_gap"] = gap
		updates["previous_risk_gap"] = stat.LastRiskGap
		updates["previous_risk_reason"] = stat.LastRiskReason
	}
	if err := tx.Model(&system.SysPhoneRegisterRiskDailyStat{}).
		Where("id = ?", stat.ID).
		Updates(updates).Error; err != nil {
		return phoneRegisterRiskDecision{}, err
	}
	return decision, nil
}

func (s *PhoneRegisterTaskService) effectivePhoneRegisterRiskRatioTx(tx *gorm.DB, promoterID uint) (int, error) {
	if promoterID == 0 {
		return 0, nil
	}
	var promoter system.SysUser
	if err := tx.Select("id, leader_id, origin_setting").Where("id = ?", promoterID).First(&promoter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if ratio := getCacheSampleRatio(promoter.OriginSetting); ratio != nil {
		return clampPhoneRegisterRiskRatio(*ratio), nil
	}
	if promoter.LeaderID == nil || *promoter.LeaderID == 0 {
		return 0, nil
	}
	var leader system.SysUser
	if err := tx.Select("id, origin_setting").Where("id = ?", *promoter.LeaderID).First(&leader).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if ratio := getCacheSampleRatio(leader.OriginSetting); ratio != nil {
		return clampPhoneRegisterRiskRatio(*ratio), nil
	}
	return 0, nil
}

func clampPhoneRegisterRiskRatio(ratio int) int {
	if ratio < 0 {
		return 0
	}
	if ratio > phoneRegisterRiskMaxRatio {
		return phoneRegisterRiskMaxRatio
	}
	return ratio
}

func (s *PhoneRegisterTaskService) loadPhoneRegisterRiskDailyStatTx(tx *gorm.DB, promoterID uint, now time.Time) (system.SysPhoneRegisterRiskDailyStat, error) {
	bizDate, start, end := phoneRegisterRiskDayRange(now)
	var stat system.SysPhoneRegisterRiskDailyStat
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("promoter_id = ? AND biz_date = ?", promoterID, bizDate).
		First(&stat).Error
	if err == nil {
		return stat, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return system.SysPhoneRegisterRiskDailyStat{}, err
	}

	successCount, riskCount, err := countPhoneRegisterRiskDayTasksTx(tx, promoterID, start, end)
	if err != nil {
		return system.SysPhoneRegisterRiskDailyStat{}, err
	}
	stat = system.SysPhoneRegisterRiskDailyStat{
		PromoterID:         promoterID,
		BizDate:            bizDate,
		SuccessReportCount: successCount,
		RiskFailCount:      riskCount,
		LastRiskSuccessSeq: successCount,
	}
	if riskCount == 0 {
		stat.LastRiskSuccessSeq = 0
	}
	if err := tx.Create(&stat).Error; err != nil {
		if isPhoneRegisterDuplicateKeyError(err) {
			if loadErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("promoter_id = ? AND biz_date = ?", promoterID, bizDate).
				First(&stat).Error; loadErr != nil {
				return system.SysPhoneRegisterRiskDailyStat{}, loadErr
			}
			return stat, nil
		}
		return system.SysPhoneRegisterRiskDailyStat{}, err
	}
	return stat, nil
}

func countPhoneRegisterRiskDayTasksTx(tx *gorm.DB, promoterID uint, start time.Time, end time.Time) (int64, int64, error) {
	var successCount int64
	if err := tx.Model(&system.SysPhoneRegisterTask{}).
		Where("promoter_id = ?", promoterID).
		Where("finished_at >= ? AND finished_at < ?", start, end).
		Where("status_code IN ?", phoneRegisterRiskSuccessStatusCodes()).
		Count(&successCount).Error; err != nil {
		return 0, 0, err
	}
	var riskCount int64
	if err := tx.Model(&system.SysPhoneRegisterTask{}).
		Where("promoter_id = ?", promoterID).
		Where("finished_at >= ? AND finished_at < ?", start, end).
		Where("status_code IN ?", phoneRegisterRiskStatusCodes()).
		Count(&riskCount).Error; err != nil {
		return 0, 0, err
	}
	return successCount, riskCount, nil
}

func isPhoneRegisterRiskStatTableMissingError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sys_phone_register_risk_daily_stats") &&
		(strings.Contains(msg, "no such table") ||
			strings.Contains(msg, "doesn't exist") ||
			strings.Contains(msg, "does not exist") ||
			strings.Contains(msg, "unknown table"))
}

func isPhoneRegisterDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique index")
}

func phoneRegisterRiskDayRange(now time.Time) (string, time.Time, time.Time) {
	local := now.Local()
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	return start.Format("2006-01-02"), start, start.AddDate(0, 0, 1)
}

func phoneRegisterRiskSuccessStatusCodes() []int {
	return []int{
		system.PhoneRegisterStatusCodeSucceeded,
		system.PhoneRegisterStatusCodeRiskFace,
		system.PhoneRegisterStatusCodeRiskQuota,
	}
}

func phoneRegisterRiskStatusCodes() []int {
	return []int{
		system.PhoneRegisterStatusCodeRiskFace,
		system.PhoneRegisterStatusCodeRiskQuota,
	}
}

func isPhoneRegisterRiskStatusCode(statusCode *int) bool {
	if statusCode == nil {
		return false
	}
	return *statusCode == system.PhoneRegisterStatusCodeRiskFace || *statusCode == system.PhoneRegisterStatusCodeRiskQuota
}

func phoneRegisterRiskMinGap(ratio int, promoterID uint, bizDate string, seq int64) int64 {
	var low, high int64
	switch {
	case ratio <= 10:
		low, high = 5, 12
	case ratio <= 20:
		low, high = 3, 8
	case ratio <= 35:
		low, high = 2, 6
	default:
		low, high = 1, 4
	}
	value := int64(phoneRegisterRiskRandomFloat(fmt.Sprintf("gap:%d:%s:%d", promoterID, bizDate, seq)) * float64(high-low+1))
	return low + value
}

func phoneRegisterRiskHitProbability(ratio int, seq int64, currentRiskCount int64, targetRiskCount int64, gap int64) float64 {
	debt := targetRiskCount - currentRiskCount
	expectedGap := float64(100) / float64(ratio)
	base := 0.25
	debtBoost := math.Min(float64(debt)*0.22, 0.45)
	gapBoost := math.Min((float64(gap)/expectedGap)*0.18, 0.25)
	return math.Max(0.15, math.Min(base+debtBoost+gapBoost, 0.88))
}

func phoneRegisterRiskReason(_ uint, _ string, _ int64, _ string, _ string) string {
	return phoneRegisterRiskReasonFace
}

func defaultPhoneRegisterRiskRandomFloat(seed string) float64 {
	sum := sha256.Sum256([]byte(seed))
	n := binary.BigEndian.Uint64(sum[:8])
	return float64(n) / float64(^uint64(0))
}

func (s *PhoneRegisterTaskService) riskFailTaskTx(tx *gorm.DB, task *system.SysPhoneRegisterTask, decision phoneRegisterRiskDecision, now time.Time) error {
	if task == nil {
		return errors.New("任务不存在")
	}
	task.Status = system.PhoneRegisterStatusFailed
	task.StatusCode = &decision.StatusCode
	task.LastError = decision.Reason
	task.CacheStatus = system.PhoneRegisterCacheStatusPending
	task.FinishedAt = &now
	task.LastHeartbeatAt = &now
	task.PendingCode = ""
	task.CodeRequestedAt = nil
	if err := tx.Model(task).
		Select("status", "status_code", "last_error", "cache_status", "finished_at", "last_heartbeat_at", "pending_code", "code_requested_at", "updated_at").
		Updates(task).Error; err != nil {
		return err
	}
	return tx.Create(&system.SysPhoneRegisterTaskLog{
		TaskID:   task.ID,
		DeviceID: stringValue(task.HolderDeviceID),
		Message:  fmt.Sprintf("系统判定失败：%s，配置比例%d%%，当天成功上报序号%d", decision.Reason, decision.Ratio, decision.Seq),
	}).Error
}

func (s *PhoneRegisterTaskService) findRiskCacheUploadTaskByDeviceTx(tx *gorm.DB, deviceID string) (system.SysPhoneRegisterTask, bool, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, false, errors.New("deviceId不能为空")
	}
	var task system.SysPhoneRegisterTask
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("holder_device_id = ?", deviceID).
		Where("finished_at IS NOT NULL").
		Where("status = ?", system.PhoneRegisterStatusFailed).
		Where("status_code IN ?", phoneRegisterRiskStatusCodes()).
		Where("(cache_status = ? OR cache_status = '')", system.PhoneRegisterCacheStatusPending).
		Order("id desc").
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return system.SysPhoneRegisterTask{}, false, nil
		}
		return system.SysPhoneRegisterTask{}, false, err
	}
	return task, true, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (s *PhoneRegisterTaskService) CompleteTaskAfterQQCacheUploadTx(tx *gorm.DB, deviceID string, qqCacheRecordID uint, qqNum string) (system.SysPhoneRegisterTask, error) {
	task, found, err := s.findUniqueOpenTaskByDeviceTx(tx, deviceID, true)
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	if !found {
		task, found, err = s.findRiskCacheUploadTaskByDeviceTx(tx, deviceID)
		if err != nil {
			return system.SysPhoneRegisterTask{}, err
		}
		if !found {
			return system.SysPhoneRegisterTask{}, errors.New("当前设备暂无待上传缓存的手机号注册任务")
		}
	}
	if task.Status != system.PhoneRegisterStatusRegisteredWaitUpload && !isPhoneRegisterRiskStatusCode(task.StatusCode) {
		return system.SysPhoneRegisterTask{}, errors.New("当前任务未处于待上传缓存状态")
	}
	now := time.Now()
	successCode := system.PhoneRegisterStatusCodeSucceeded
	riskTask := isPhoneRegisterRiskStatusCode(task.StatusCode)
	if !riskTask {
		task.Status = system.PhoneRegisterStatusSucceeded
		task.StatusCode = &successCode
		task.LastError = ""
		task.FinishedAt = &now
	}
	task.QQNum = strings.TrimSpace(qqNum)
	task.QQCacheRecordID = &qqCacheRecordID
	task.CacheStatus = system.PhoneRegisterCacheStatusUploaded
	task.HolderDeviceID = nil
	task.PendingCode = ""
	task.CodeRequestedAt = nil
	if err := tx.Model(&task).
		Select("status", "status_code", "qq_num", "qq_cache_record_id", "cache_status", "finished_at", "holder_device_id", "pending_code", "code_requested_at", "last_error", "updated_at").
		Updates(task).Error; err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	return task, nil
}

func (s *PhoneRegisterTaskService) AttachOpenAPICacheTx(tx *gorm.DB, deviceID string, taskID uint, qqCacheRecordID uint, qqNum string) (system.SysPhoneRegisterTask, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, errors.New("deviceId不能为空")
	}
	if taskID == 0 {
		return system.SysPhoneRegisterTask{}, errors.New("taskId不能为空")
	}
	var task system.SysPhoneRegisterTask
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND task_source = ?", taskID, system.PhoneRegisterTaskSourceOpenAPI).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return system.SysPhoneRegisterTask{}, errors.New("任务不存在")
		}
		return system.SysPhoneRegisterTask{}, err
	}
	if task.HolderDeviceID == nil || strings.TrimSpace(*task.HolderDeviceID) != deviceID {
		return system.SysPhoneRegisterTask{}, errors.New("taskId与当前设备任务不一致")
	}
	if !isOpenAPICacheUploadAllowedTask(task) {
		return system.SysPhoneRegisterTask{}, errors.New("当前任务未处于可上传缓存状态")
	}
	task.QQNum = strings.TrimSpace(qqNum)
	task.QQCacheRecordID = &qqCacheRecordID
	task.CacheStatus = system.PhoneRegisterCacheStatusUploaded
	updates := map[string]any{
		"qq_num":             task.QQNum,
		"qq_cache_record_id": task.QQCacheRecordID,
		"cache_status":       task.CacheStatus,
		"updated_at":         time.Now(),
	}
	if task.Status == system.PhoneRegisterStatusSucceeded {
		task.LastError = ""
		updates["last_error"] = ""
	}
	updates["holder_device_id"] = nil
	if err := tx.Model(&task).Updates(updates).Error; err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	task.HolderDeviceID = nil
	return task, nil
}

func isOpenAPICacheUploadAllowedTask(task system.SysPhoneRegisterTask) bool {
	if task.FinishedAt == nil {
		return false
	}
	return task.Status == system.PhoneRegisterStatusSucceeded || task.Status == system.PhoneRegisterStatusFailed
}

func (s *PhoneRegisterTaskService) timeoutUnfinishedTasksThrottled() error {
	if global.GVA_DB == nil {
		return nil
	}
	now := time.Now()
	phoneRegisterTimeoutScanThrottleState.Lock()
	if !phoneRegisterTimeoutScanThrottleState.lastRun.IsZero() &&
		now.Sub(phoneRegisterTimeoutScanThrottleState.lastRun) < phoneRegisterTimeoutScanThrottle {
		phoneRegisterTimeoutScanThrottleState.Unlock()
		return nil
	}
	phoneRegisterTimeoutScanThrottleState.lastRun = now
	phoneRegisterTimeoutScanThrottleState.Unlock()

	if err := s.timeoutUnfinishedTasks(); err != nil {
		phoneRegisterTimeoutScanThrottleState.Lock()
		phoneRegisterTimeoutScanThrottleState.lastRun = time.Time{}
		phoneRegisterTimeoutScanThrottleState.Unlock()
		return err
	}
	return nil
}

func resetPhoneRegisterTimeoutScanThrottle() {
	phoneRegisterTimeoutScanThrottleState.Lock()
	phoneRegisterTimeoutScanThrottleState.lastRun = time.Time{}
	phoneRegisterTimeoutScanThrottleState.Unlock()
}

func (s *PhoneRegisterTaskService) timeoutUnfinishedTasks() error {
	if global.GVA_DB == nil {
		return nil
	}
	now := time.Now()
	heartbeatDeadline := now.Add(-phoneRegisterLeaseTimeout)
	timedOutReservations, err := s.findTimedOutPendingReservations(now)
	if err != nil {
		return err
	}
	releasedDeviceIDs, err := s.findTimeoutReleasedDeviceIDs(now, heartbeatDeadline)
	if err != nil {
		return err
	}

	if err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status NOT IN ?", []string{system.PhoneRegisterStatusSucceeded, system.PhoneRegisterStatusFailed}).
		Where("expires_at <= ?", now).
		Updates(map[string]any{
			"status":            system.PhoneRegisterStatusFailed,
			"status_code":       system.PhoneRegisterStatusCodeTaskTimeout,
			"last_error":        "任务总超时",
			"finished_at":       now,
			"holder_device_id":  nil,
			"pending_code":      "",
			"code_requested_at": nil,
			"updated_at":        now,
		}).Error; err != nil {
		return err
	}

	if err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status = ?", system.PhoneRegisterStatusWaitingPromoterCode).
		Where("code_requested_at IS NOT NULL AND code_requested_at <= ?", now.Add(-phoneRegisterCodeSubmitWindow)).
		Where("pending_code = ''").
		Updates(map[string]any{
			"status":            system.PhoneRegisterStatusFailed,
			"status_code":       system.PhoneRegisterStatusCodeVerifyCodeTimeout,
			"last_error":        "验证码等待超时",
			"finished_at":       now,
			"holder_device_id":  nil,
			"pending_code":      "",
			"code_requested_at": nil,
			"updated_at":        now,
		}).Error; err != nil {
		return err
	}

	if err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status IN ?", []string{
			system.PhoneRegisterStatusRunning,
			system.PhoneRegisterStatusWaitingPromoterCode,
		}).
		Where("holder_device_id IS NOT NULL").
		Where("last_heartbeat_at IS NOT NULL").
		Where("last_heartbeat_at <= ?", heartbeatDeadline).
		Updates(map[string]any{
			"status":            system.PhoneRegisterStatusFailed,
			"status_code":       system.PhoneRegisterStatusCodeHeartbeatTimeout,
			"last_error":        "设备心跳超时",
			"finished_at":       now,
			"holder_device_id":  nil,
			"pending_code":      "",
			"code_requested_at": nil,
			"updated_at":        now,
		}).Error; err != nil {
		return err
	}
	if err := s.recordOpenAPICacheUploadTimeouts(now); err != nil {
		return err
	}
	for _, task := range timedOutReservations {
		if task.HolderDeviceID != nil {
			_ = (&DeviceService{}).ClearBusy(*task.HolderDeviceID, phoneRegisterReservationBusyBusiness(task.ID))
		}
	}
	for _, deviceID := range releasedDeviceIDs {
		_ = markPhoneRegisterDeviceOffline(deviceID)
	}
	resetPhoneRegisterPendingClaimableTaskCountCache()
	return nil
}

func (s *PhoneRegisterTaskService) findTimedOutPendingReservations(now time.Time) ([]system.SysPhoneRegisterTask, error) {
	var tasks []system.SysPhoneRegisterTask
	if err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Select("id", "holder_device_id").
		Where("finished_at IS NULL").
		Where("status = ?", system.PhoneRegisterStatusPending).
		Where("holder_device_id IS NOT NULL").
		Where("expires_at <= ?", now).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *PhoneRegisterTaskService) findTimeoutReleasedDeviceIDs(now time.Time, heartbeatDeadline time.Time) ([]string, error) {
	deviceSet := map[string]struct{}{}
	collect := func(db *gorm.DB) error {
		var deviceIDs []string
		if err := db.Distinct("holder_device_id").Pluck("holder_device_id", &deviceIDs).Error; err != nil {
			return err
		}
		for _, deviceID := range deviceIDs {
			deviceID = strings.TrimSpace(deviceID)
			if deviceID != "" {
				deviceSet[deviceID] = struct{}{}
			}
		}
		return nil
	}
	if err := collect(global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status NOT IN ?", []string{system.PhoneRegisterStatusSucceeded, system.PhoneRegisterStatusFailed}).
		Where("status != ?", system.PhoneRegisterStatusPending).
		Where("holder_device_id IS NOT NULL").
		Where("expires_at <= ?", now)); err != nil {
		return nil, err
	}
	if err := collect(global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status = ?", system.PhoneRegisterStatusWaitingPromoterCode).
		Where("code_requested_at IS NOT NULL AND code_requested_at <= ?", now.Add(-phoneRegisterCodeSubmitWindow)).
		Where("pending_code = ''").
		Where("holder_device_id IS NOT NULL")); err != nil {
		return nil, err
	}
	if err := collect(global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status IN ?", []string{
			system.PhoneRegisterStatusRunning,
			system.PhoneRegisterStatusWaitingPromoterCode,
		}).
		Where("holder_device_id IS NOT NULL").
		Where("last_heartbeat_at IS NOT NULL").
		Where("last_heartbeat_at <= ?", heartbeatDeadline)); err != nil {
		return nil, err
	}
	deviceIDs := make([]string, 0, len(deviceSet))
	for deviceID := range deviceSet {
		deviceIDs = append(deviceIDs, deviceID)
	}
	return deviceIDs, nil
}

func (s *PhoneRegisterTaskService) recordOpenAPICacheUploadTimeouts(now time.Time) error {
	deadline := now.Add(-phoneRegisterCacheWaitTimeout)
	var tasks []system.SysPhoneRegisterTask
	if err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("task_source = ?", system.PhoneRegisterTaskSourceOpenAPI).
		Where("status = ? AND status_code = ?", system.PhoneRegisterStatusSucceeded, system.PhoneRegisterStatusCodeSucceeded).
		Where("qq_cache_record_id IS NULL").
		Where("(cache_status = ? OR cache_status = '')", system.PhoneRegisterCacheStatusPending).
		Where("last_heartbeat_at IS NOT NULL AND last_heartbeat_at <= ?", deadline).
		Limit(100).
		Find(&tasks).Error; err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}
	logs := make([]system.SysPhoneRegisterTaskLog, 0, len(tasks))
	for i := range tasks {
		deviceID := ""
		if tasks[i].HolderDeviceID != nil {
			deviceID = strings.TrimSpace(*tasks[i].HolderDeviceID)
		}
		logs = append(logs, system.SysPhoneRegisterTaskLog{
			TaskID:   tasks[i].ID,
			DeviceID: deviceID,
			Message:  phoneRegisterOpenAPICacheTimeoutLog,
		})
	}
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&logs).Error; err != nil {
			return err
		}
		ids := make([]uint, 0, len(tasks))
		for i := range tasks {
			ids = append(ids, tasks[i].ID)
		}
		return tx.Model(&system.SysPhoneRegisterTask{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"cache_status": system.PhoneRegisterCacheStatusTimeout,
				"updated_at":   now,
			}).Error
	})
}

func (s *PhoneRegisterTaskService) failTaskTx(tx *gorm.DB, task *system.SysPhoneRegisterTask, statusCode int, lastError string) error {
	if task == nil {
		return errors.New("任务不存在")
	}
	now := time.Now()
	task.Status = system.PhoneRegisterStatusFailed
	task.StatusCode = &statusCode
	task.LastError = strings.TrimSpace(lastError)
	if task.LastError == "" {
		task.LastError = "任务失败"
	}
	task.FinishedAt = &now
	task.HolderDeviceID = nil
	task.PendingCode = ""
	task.CodeRequestedAt = nil
	if err := tx.Model(task).
		Select("status", "status_code", "last_error", "finished_at", "holder_device_id", "pending_code", "code_requested_at", "updated_at").
		Updates(task).Error; err != nil {
		return err
	}
	return nil
}

func markPhoneRegisterDeviceBusy(deviceID string) error {
	return (&DeviceService{}).MarkBusy(deviceID, phoneRegisterDeviceBusyBusiness)
}

func markPhoneRegisterDeviceOffline(deviceID string) error {
	return (&DeviceService{}).MarkOffline(deviceID)
}

func (s *PhoneRegisterTaskService) findUniqueOpenTaskByDeviceTx(tx *gorm.DB, deviceID string, lock bool) (system.SysPhoneRegisterTask, bool, error) {
	db := tx.Where("holder_device_id = ? AND finished_at IS NULL AND status NOT IN ?", deviceID, []string{
		system.PhoneRegisterStatusSucceeded,
		system.PhoneRegisterStatusFailed,
	})
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var tasks []system.SysPhoneRegisterTask
	if err := db.Order("id asc").Limit(2).Find(&tasks).Error; err != nil {
		return system.SysPhoneRegisterTask{}, false, err
	}
	if len(tasks) == 0 {
		return system.SysPhoneRegisterTask{}, false, nil
	}
	if len(tasks) > 1 {
		global.GVA_LOG.Error("【手机号注册任务】设备绑定多条未完成任务", zap.String("deviceId", deviceID), zap.Int("count", len(tasks)))
		return system.SysPhoneRegisterTask{}, false, errors.New("当前设备存在多条未完成任务，请联系管理员排查")
	}
	return tasks[0], true, nil
}

func applyPhoneRegisterTaskRoleFilter(db *gorm.DB, operatorRole uint, operatorID uint, req systemReq.PhoneRegisterTaskList) (*gorm.DB, error) {
	switch operatorRole {
	case phoneRoleSuperAdmin, phoneRoleAdmin:
		if req.LeaderID != 0 {
			db = db.Where("leader_id = ?", req.LeaderID)
		}
		if req.PromoterID != 0 {
			db = db.Where("promoter_id = ?", req.PromoterID)
		}
		return db, nil
	case phoneRoleLeader:
		db = db.Where("leader_id = ?", operatorID)
		if req.PromoterID != 0 {
			db = db.Where("promoter_id = ?", req.PromoterID)
		}
		return db, nil
	case phoneRolePromoter:
		return db.Where("promoter_id = ?", operatorID), nil
	default:
		return nil, errors.New("无权限查看任务列表")
	}
}

func shouldUsePhoneRegisterTaskDayScoped(operatorRole uint, dayScoped bool) bool {
	return dayScoped && (operatorRole == phoneRoleLeader || operatorRole == phoneRolePromoter)
}

func applyPhoneRegisterTaskQueryFilters(db *gorm.DB, req systemReq.PhoneRegisterTaskList) *gorm.DB {
	if status := strings.TrimSpace(req.Status); status != "" {
		if status == "processing" {
			db = db.Where("status NOT IN ?", []string{
				system.PhoneRegisterStatusSucceeded,
				system.PhoneRegisterStatusFailed,
			})
		} else {
			db = db.Where("status = ?", status)
		}
	}
	if req.StatusCode != nil {
		db = db.Where("status_code = ?", *req.StatusCode)
	}
	switch cacheStatus := strings.TrimSpace(req.CacheStatus); cacheStatus {
	case system.PhoneRegisterCacheStatusUploaded:
		db = db.Where("cache_status = ?", system.PhoneRegisterCacheStatusUploaded)
	case "not_uploaded":
		db = db.Where("(cache_status IS NULL OR cache_status = '' OR cache_status <> ?)", system.PhoneRegisterCacheStatusUploaded)
	}
	switch createSource := strings.TrimSpace(req.CreateSource); createSource {
	case system.PhoneRegisterTaskCreateSourceOpenAPI:
		db = db.Where("(create_source = ? OR ((create_source IS NULL OR create_source = '') AND task_source = ?))",
			system.PhoneRegisterTaskCreateSourceOpenAPI,
			system.PhoneRegisterTaskSourceOpenAPI,
		)
	case system.PhoneRegisterTaskCreateSourceManual, "manual":
		db = db.Where("(create_source = ? OR ((create_source IS NULL OR create_source = '') AND (task_source IS NULL OR task_source = '' OR task_source <> ?)))",
			system.PhoneRegisterTaskCreateSourceManual,
			system.PhoneRegisterTaskSourceOpenAPI,
		)
	}
	switch taskSource := strings.TrimSpace(req.TaskSource); taskSource {
	case system.PhoneRegisterTaskSourceOpenAPI:
		db = db.Where("task_source = ?", system.PhoneRegisterTaskSourceOpenAPI)
	case "manual":
		db = db.Where("(task_source IS NULL OR task_source = '' OR task_source <> ?)", system.PhoneRegisterTaskSourceOpenAPI)
	}
	if phone := strings.TrimSpace(req.Phone); phone != "" {
		db = db.Where("phone LIKE ?", "%"+phone+"%")
	}
	if qqNum := strings.TrimSpace(req.QQNum); qqNum != "" {
		db = db.Where("qq_num LIKE ?", "%"+qqNum+"%")
	}
	if deviceID := strings.TrimSpace(req.DeviceID); deviceID != "" {
		db = db.Where("holder_device_id LIKE ?", "%"+deviceID+"%")
	}
	if mode := normalizePhoneRegisterSMSMode(req.SMSReceiveMode); mode != "" {
		db = db.Where("sms_receive_mode = ?", mode)
	}
	if req.DayScoped {
		return applyPhoneRegisterTaskDayRangeFilter(db, req.FinishedAtStart, req.FinishedAtEnd)
	}
	return applyPhoneRegisterTaskFinishedAtRangeFilter(db, req.FinishedAtStart, req.FinishedAtEnd)
}

func applyPhoneRegisterTaskFinishedAtRangeFilter(db *gorm.DB, startRaw string, endRaw string) *gorm.DB {
	return applyPhoneRegisterTaskFinishedAtRangeFilterWithColumn(db, "finished_at", startRaw, endRaw)
}

func applyPhoneRegisterTaskFinishedAtRangeFilterWithColumn(db *gorm.DB, finishedColumn string, startRaw string, endRaw string) *gorm.DB {
	if startAt, ok := parseTaskListTime(startRaw); ok {
		db = db.Where(fmt.Sprintf("%s >= ?", finishedColumn), startAt)
	}
	if endAt, ok := parseTaskListTime(endRaw); ok {
		db = db.Where(fmt.Sprintf("%s <= ?", finishedColumn), endAt)
	}
	return db
}

func applyPhoneRegisterTaskDayRangeFilter(db *gorm.DB, startRaw string, endRaw string) *gorm.DB {
	return applyPhoneRegisterTaskDayRangeFilterWithColumns(db, "finished_at", "created_at", startRaw, endRaw)
}

func applyPhoneRegisterTaskDayRangeFilterWithColumns(db *gorm.DB, finishedColumn string, createdColumn string, startRaw string, endRaw string) *gorm.DB {
	startAt, hasStart := parseTaskListTime(startRaw)
	endAt, hasEnd := parseTaskListTime(endRaw)
	if hasStart && hasEnd {
		return db.Where(
			fmt.Sprintf("((%s IS NOT NULL AND %s >= ? AND %s <= ?) OR (%s IS NULL AND %s >= ? AND %s <= ?))", finishedColumn, finishedColumn, finishedColumn, finishedColumn, createdColumn, createdColumn),
			startAt, endAt, startAt, endAt,
		)
	}
	if hasStart {
		return db.Where(
			fmt.Sprintf("((%s IS NOT NULL AND %s >= ?) OR (%s IS NULL AND %s >= ?))", finishedColumn, finishedColumn, finishedColumn, createdColumn),
			startAt, startAt,
		)
	}
	if hasEnd {
		return db.Where(
			fmt.Sprintf("((%s IS NOT NULL AND %s <= ?) OR (%s IS NULL AND %s <= ?))", finishedColumn, finishedColumn, finishedColumn, createdColumn),
			endAt, endAt,
		)
	}
	return db
}

func normalizePhoneRegisterSMSMode(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case system.PhoneRegisterSMSModePlatformSend:
		return system.PhoneRegisterSMSModePlatformSend
	case system.PhoneRegisterSMSModeUserSentToTX:
		return system.PhoneRegisterSMSModeUserSentToTX
	default:
		return strings.ToUpper(strings.TrimSpace(raw))
	}
}

func isValidPhoneRegisterSMSMode(mode string) bool {
	return mode == system.PhoneRegisterSMSModePlatformSend || mode == system.PhoneRegisterSMSModeUserSentToTX
}

func isPhoneRegisterTaskTerminal(status string, finishedAt *time.Time) bool {
	return finishedAt != nil || status == system.PhoneRegisterStatusSucceeded || status == system.PhoneRegisterStatusFailed
}

func isBlockedPhoneRegisterPhone(phone string, prefixes []string) bool {
	phone = strings.TrimSpace(phone)
	for _, prefix := range prefixes {
		if strings.HasPrefix(phone, prefix) {
			return true
		}
	}
	return false
}

func phoneRegisterBlockedPrefixesFromConfig(raw string) []string {
	normalized := normalizePhoneRegisterBlockedPrefixes(raw)
	if normalized == "" {
		return append([]string(nil), defaultPhoneRegisterBlockedPrefixes...)
	}
	return strings.Split(normalized, ",")
}

func normalizePhoneRegisterBlockedPrefixes(raw string) string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == '\r' || r == '\t' || r == ' ' || r == ';' || r == '；'
	})
	seen := map[string]struct{}{}
	prefixes := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || !isDigitsOnly(part) {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		prefixes = append(prefixes, part)
	}
	if len(prefixes) == 0 {
		return strings.Join(defaultPhoneRegisterBlockedPrefixes, ",")
	}
	return strings.Join(prefixes, ",")
}

func phoneRegisterDefaultMessageByAction(action string) string {
	switch action {
	case system.PhoneRegisterDeviceActionEnterWaitingCode:
		return "设备已进入等待验证码阶段"
	case system.PhoneRegisterDeviceActionConsumeCodeOK:
		return "设备已成功消费验证码"
	case system.PhoneRegisterDeviceActionRegisterSuccess:
		return "设备注册成功，等待缓存上传"
	case system.PhoneRegisterDeviceActionFail:
		return "设备执行失败"
	default:
		return ""
	}
}
