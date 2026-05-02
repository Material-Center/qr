package system

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	phoneRegisterTaskTimeout      = 30 * time.Minute
	phoneRegisterLeaseTimeout     = 5 * time.Minute
	phoneRegisterTimeoutScanEvery = 1 * time.Minute

	phoneRoleSuperAdmin = uint(888)
	phoneRoleAdmin      = uint(100)
	phoneRoleLeader     = uint(200)
	phoneRolePromoter   = uint(300)
)

type PhoneRegisterTaskService struct{}

type phoneRegisterTaskListResult struct {
	List       []system.SysPhoneRegisterTask
	Total      int64
	Success    int64
	Failed     int64
	Processing int64
}

var phoneRegisterTaskDaemonOnce sync.Once

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

func (s *PhoneRegisterTaskService) CreateTask(promoterID uint, phone string, smsReceiveMode string) (system.SysPhoneRegisterTask, error) {
	phone = strings.TrimSpace(phone)
	smsReceiveMode = normalizePhoneRegisterSMSMode(smsReceiveMode)
	if phone == "" {
		return system.SysPhoneRegisterTask{}, errors.New("手机号不能为空")
	}
	if !isValidPhoneRegisterSMSMode(smsReceiveMode) {
		return system.SysPhoneRegisterTask{}, errors.New("不支持的收码方式")
	}

	var promoter system.SysUser
	if err := global.GVA_DB.Select("id, leader_id").Where("id = ?", promoterID).First(&promoter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return system.SysPhoneRegisterTask{}, errors.New("地推账号不存在")
		}
		return system.SysPhoneRegisterTask{}, err
	}

	task := system.SysPhoneRegisterTask{
		Phone:          phone,
		PromoterID:     promoterID,
		LeaderID:       promoter.LeaderID,
		SMSReceiveMode: smsReceiveMode,
		Status:         system.PhoneRegisterStatusPending,
		ExpiresAt:      time.Now().Add(phoneRegisterTaskTimeout),
	}
	if err := global.GVA_DB.Create(&task).Error; err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	return task, nil
}

func (s *PhoneRegisterTaskService) SubmitCode(promoterID uint, req systemReq.PhoneRegisterTaskSubmitCode) (system.SysPhoneRegisterTask, error) {
	if req.TaskID == 0 {
		return system.SysPhoneRegisterTask{}, errors.New("任务ID不能为空")
	}
	verifyCode := strings.TrimSpace(req.VerifyCode)
	if verifyCode == "" {
		return system.SysPhoneRegisterTask{}, errors.New("验证码不能为空")
	}
	_ = s.timeoutUnfinishedTasks()

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
	_ = s.timeoutUnfinishedTasks()
	var task system.SysPhoneRegisterTask
	err := global.GVA_DB.Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Order("id desc").
		First(&task).Error
	return task, err
}

func (s *PhoneRegisterTaskService) GetActiveTasks(promoterID uint) ([]system.SysPhoneRegisterTask, error) {
	_ = s.timeoutUnfinishedTasks()
	var tasks []system.SysPhoneRegisterTask
	err := global.GVA_DB.Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Order("id desc").
		Find(&tasks).Error
	return tasks, err
}

func (s *PhoneRegisterTaskService) GetTaskList(operatorRole uint, operatorID uint, req systemReq.PhoneRegisterTaskList) (phoneRegisterTaskListResult, error) {
	_ = s.timeoutUnfinishedTasks()

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
	if pageSize <= 0 || pageSize > 100 {
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
	statDB = applyPhoneRegisterTaskQueryFilters(statDB, req)

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

	return phoneRegisterTaskListResult{
		List:       list,
		Total:      total,
		Success:    stat.Success,
		Failed:     stat.Failed,
		Processing: stat.Processing,
	}, nil
}

func (s *PhoneRegisterTaskService) GetSummary(operatorRole uint, operatorID uint, leaderID uint) (systemRes.PhoneRegisterTaskSummaryResponse, error) {
	if operatorRole != phoneRoleSuperAdmin && operatorRole != phoneRoleAdmin && operatorRole != phoneRoleLeader {
		return systemRes.PhoneRegisterTaskSummaryResponse{}, errors.New("无权限查看统计")
	}
	_ = s.timeoutUnfinishedTasks()

	type row struct {
		LeaderID        *uint  `gorm:"column:leader_id"`
		LeaderName      string `gorm:"column:leader_name"`
		PromoterID      uint   `gorm:"column:promoter_id"`
		PromoterName    string `gorm:"column:promoter_name"`
		SuccessCount    int64  `gorm:"column:success_count"`
		FailCount       int64  `gorm:"column:fail_count"`
		ProcessingCount int64  `gorm:"column:processing_count"`
	}

	db := global.GVA_DB.Table("sys_phone_register_tasks t").
		Select(`
			t.leader_id,
			leader.nick_name AS leader_name,
			t.promoter_id,
			promoter.nick_name AS promoter_name,
			COALESCE(SUM(CASE WHEN t.status = 'succeeded' THEN 1 ELSE 0 END), 0) AS success_count,
			COALESCE(SUM(CASE WHEN t.status = 'failed' THEN 1 ELSE 0 END), 0) AS fail_count,
			COALESCE(SUM(CASE WHEN t.status NOT IN ('succeeded', 'failed') THEN 1 ELSE 0 END), 0) AS processing_count`).
		Joins("LEFT JOIN sys_users promoter ON promoter.id = t.promoter_id").
		Joins("LEFT JOIN sys_users leader ON leader.id = t.leader_id")

	if operatorRole == phoneRoleLeader {
		db = db.Where("t.leader_id = ?", operatorID)
	} else if leaderID != 0 {
		db = db.Where("t.leader_id = ?", leaderID)
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
			leader.ProcessingCount += item.ProcessingCount
			leaderMap[item.LeaderID] = leader
		}
	}

	leaders := make([]systemRes.PhoneRegisterTaskSummaryItem, 0, len(leaderMap))
	for _, item := range leaderMap {
		leaders = append(leaders, item)
	}
	return systemRes.PhoneRegisterTaskSummaryResponse{
		Leaders:   leaders,
		Promoters: promoters,
	}, nil
}

func (s *PhoneRegisterTaskService) DevicePoll(req systemReq.PhoneRegisterDevicePoll) (system.SysPhoneRegisterTask, bool, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, false, errors.New("deviceId不能为空")
	}
	_ = s.timeoutUnfinishedTasks()

	var task system.SysPhoneRegisterTask
	found := false
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		existing, ok, err := s.findUniqueOpenTaskByDeviceTx(tx, deviceID, true)
		if err != nil {
			return err
		}
		if ok {
			task = existing
			found = true
			return nil
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND finished_at IS NULL", system.PhoneRegisterStatusPending).
			Order("id asc").
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				task = system.SysPhoneRegisterTask{}
				return nil
			}
			return err
		}
		now := time.Now()
		task.Status = system.PhoneRegisterStatusRunning
		task.HolderDeviceID = stringPtr(deviceID)
		task.ClaimedAt = &now
		task.LastHeartbeatAt = &now
		task.LastError = ""
		if err := tx.Model(&task).
			Select("status", "holder_device_id", "claimed_at", "last_heartbeat_at", "last_error", "updated_at").
			Updates(task).Error; err != nil {
			return err
		}
		found = true
		return nil
	})
	return task, found, err
}

func (s *PhoneRegisterTaskService) DeviceTask(req systemReq.PhoneRegisterDeviceTask) (system.SysPhoneRegisterTask, bool, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return system.SysPhoneRegisterTask{}, false, errors.New("deviceId不能为空")
	}
	_ = s.timeoutUnfinishedTasks()
	return s.findUniqueOpenTaskByDeviceTx(global.GVA_DB, deviceID, false)
}

func (s *PhoneRegisterTaskService) DeviceHeartbeat(req systemReq.PhoneRegisterDeviceHeartbeat) error {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return errors.New("deviceId不能为空")
	}
	_ = s.timeoutUnfinishedTasks()
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
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
	})
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
	_ = s.timeoutUnfinishedTasks()

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
			task.Status = system.PhoneRegisterStatusWaitingPromoterCode
			task.LastError = message
			task.LastHeartbeatAt = &now
			return tx.Model(&task).
				Select("status", "last_error", "last_heartbeat_at", "updated_at").
				Updates(task).Error
		case system.PhoneRegisterDeviceActionConsumeCodeOK:
			task.Status = system.PhoneRegisterStatusRunning
			task.PendingCode = ""
			task.LastError = message
			task.LastHeartbeatAt = &now
			return tx.Model(&task).
				Select("status", "pending_code", "last_error", "last_heartbeat_at", "updated_at").
				Updates(task).Error
		case system.PhoneRegisterDeviceActionRegisterSuccess:
			task.Status = system.PhoneRegisterStatusRegisteredWaitUpload
			task.LastError = message
			task.LastHeartbeatAt = &now
			return tx.Model(&task).
				Select("status", "last_error", "last_heartbeat_at", "updated_at").
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
	return task, nil
}

func (s *PhoneRegisterTaskService) GetDeviceConfig() (systemRes.PhoneRegisterDeviceConfigResponse, error) {
	var cfg system.SysRegisterConfig
	err := global.GVA_DB.Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).First(&cfg).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return systemRes.PhoneRegisterDeviceConfigResponse{}, err
	}
	return systemRes.PhoneRegisterDeviceConfigResponse{
		ImageProvider: systemRes.PhoneRegisterImageProviderConfig{
			Provider:  cfg.PhoneImageProvider,
			Username:  cfg.PhoneImageProviderUsername,
			Password:  cfg.PhoneImageProviderPassword,
			SecretKey: cfg.PhoneImageProviderSecretKey,
		},
	}, nil
}

func (s *PhoneRegisterTaskService) CompleteTaskAfterQQCacheUploadTx(tx *gorm.DB, deviceID string, qqCacheRecordID uint, qqNum string) (system.SysPhoneRegisterTask, error) {
	task, found, err := s.findUniqueOpenTaskByDeviceTx(tx, deviceID, true)
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	if !found {
		return system.SysPhoneRegisterTask{}, errors.New("当前设备暂无待上传缓存的手机号注册任务")
	}
	if task.Status != system.PhoneRegisterStatusRegisteredWaitUpload {
		return system.SysPhoneRegisterTask{}, errors.New("当前任务未处于待上传缓存状态")
	}
	now := time.Now()
	successCode := system.PhoneRegisterStatusCodeSucceeded
	task.Status = system.PhoneRegisterStatusSucceeded
	task.StatusCode = &successCode
	task.QQNum = strings.TrimSpace(qqNum)
	task.QQCacheRecordID = &qqCacheRecordID
	task.FinishedAt = &now
	task.HolderDeviceID = nil
	task.PendingCode = ""
	task.LastError = ""
	if err := tx.Model(&task).
		Select("status", "status_code", "qq_num", "qq_cache_record_id", "finished_at", "holder_device_id", "pending_code", "last_error", "updated_at").
		Updates(task).Error; err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	return task, nil
}

func (s *PhoneRegisterTaskService) timeoutUnfinishedTasks() error {
	if global.GVA_DB == nil {
		return nil
	}
	now := time.Now()
	heartbeatDeadline := now.Add(-phoneRegisterLeaseTimeout)

	if err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status NOT IN ?", []string{system.PhoneRegisterStatusSucceeded, system.PhoneRegisterStatusFailed}).
		Where("expires_at <= ?", now).
		Updates(map[string]any{
			"status":           system.PhoneRegisterStatusFailed,
			"status_code":      system.PhoneRegisterStatusCodeTaskTimeout,
			"last_error":       "任务总超时",
			"finished_at":      now,
			"holder_device_id": nil,
			"pending_code":     "",
			"updated_at":       now,
		}).Error; err != nil {
		return err
	}

	if err := global.GVA_DB.Model(&system.SysPhoneRegisterTask{}).
		Where("finished_at IS NULL").
		Where("status IN ?", []string{
			system.PhoneRegisterStatusRunning,
			system.PhoneRegisterStatusWaitingPromoterCode,
			system.PhoneRegisterStatusRegisteredWaitUpload,
		}).
		Where("holder_device_id IS NOT NULL").
		Where("last_heartbeat_at IS NOT NULL").
		Where("last_heartbeat_at <= ?", heartbeatDeadline).
		Updates(map[string]any{
			"status":           system.PhoneRegisterStatusFailed,
			"status_code":      system.PhoneRegisterStatusCodeHeartbeatTimeout,
			"last_error":       "设备心跳超时",
			"finished_at":      now,
			"holder_device_id": nil,
			"pending_code":     "",
			"updated_at":       now,
		}).Error; err != nil {
		return err
	}
	return nil
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
	return tx.Model(task).
		Select("status", "status_code", "last_error", "finished_at", "holder_device_id", "pending_code", "updated_at").
		Updates(task).Error
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

func applyPhoneRegisterTaskQueryFilters(db *gorm.DB, req systemReq.PhoneRegisterTaskList) *gorm.DB {
	if status := strings.TrimSpace(req.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if req.StatusCode != nil {
		db = db.Where("status_code = ?", *req.StatusCode)
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
	if req.FinishedAtStart != "" {
		db = db.Where("finished_at >= ?", req.FinishedAtStart)
	}
	if req.FinishedAtEnd != "" {
		db = db.Where("finished_at <= ?", req.FinishedAtEnd)
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
