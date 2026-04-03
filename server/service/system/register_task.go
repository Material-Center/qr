package system

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Material-Center/qpi"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"go.uber.org/zap"
)

const registerTaskTimeout = 10 * time.Minute

const (
	roleSuperAdmin = uint(888)
	roleAdmin      = uint(100)
	roleLeader     = uint(200)
	rolePromoter   = uint(300)
)

type RegisterTaskService struct{}

type registerTaskListResult struct {
	List       []system.SysRegisterTask
	Total      int64
	Success    int64
	Failed     int64
	Processing int64
}

type phoneBindSession struct {
	Client    *qpi.LoginClient
	CreatedAt time.Time
}

var registerTaskPhoneBindSessions sync.Map // map[taskID]*phoneBindSession

func taskLogFields(task system.SysRegisterTask) []zap.Field {
	fields := []zap.Field{
		zap.Uint("taskId", task.ID),
		zap.String("phone", task.Phone),
		zap.String("currentStep", task.CurrentStep),
		zap.Uint("promoterId", task.PromoterID),
	}
	if task.LeaderID != nil {
		fields = append(fields, zap.Uint("leaderId", *task.LeaderID))
	}
	return fields
}

func init() {
	qpi.SetLogger(global.GVA_LOG)
}

func (s *RegisterTaskService) CreateTask(promoterID uint, phone string) (task system.SysRegisterTask, err error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return task, errors.New("手机号不能为空")
	}
	global.GVA_LOG.Info("register_task_create_start", zap.Uint("promoterId", promoterID), zap.String("phone", phone))

	var duplicateCount int64
	if err = global.GVA_DB.Model(&system.SysRegisterTask{}).Where("phone = ?", phone).Count(&duplicateCount).Error; err != nil {
		global.GVA_LOG.Error("register_task_create_db_check_failed", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Error(err))
		return task, err
	}
	if duplicateCount > 0 {
		global.GVA_LOG.Warn("register_task_create_duplicate_phone", zap.Uint("promoterId", promoterID), zap.String("phone", phone))
		return task, errors.New("手机号已提交过任务，不能重复创建")
	}

	if err = s.timeoutTasksByPromoter(promoterID); err != nil {
		return task, err
	}

	var unfinishedCount int64
	if err = global.GVA_DB.Model(&system.SysRegisterTask{}).
		Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Count(&unfinishedCount).Error; err != nil {
		return task, err
	}
	if unfinishedCount > 0 {
		return task, errors.New("存在未完成任务，请先完成当前任务")
	}

	var promoter system.SysUser
	if err = global.GVA_DB.Select("id, leader_id").Where("id = ?", promoterID).First(&promoter).Error; err != nil {
		return task, errors.New("地推账号不存在")
	}

	task = system.SysRegisterTask{
		Phone:       phone,
		CurrentStep: system.RegisterTaskStepPhoneBind,
		PromoterID:  promoterID,
		LeaderID:    promoter.LeaderID,
		ExpiresAt:   time.Now().Add(registerTaskTimeout),
	}
	if err = global.GVA_DB.Create(&task).Error; err != nil {
		global.GVA_LOG.Error("register_task_create_failed", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Error(err))
		return task, err
	}
	runtimeCfg, cfgErr := s.getRegisterRuntimeConfig(task.LeaderID)
	if cfgErr != nil {
		return task, cfgErr
	}
	if prepErr := s.preparePhoneBindSMS(&task, runtimeCfg); prepErr != nil {
		return task, prepErr
	}
	global.GVA_LOG.Info("register_task_create_success", taskLogFields(task)...)
	return task, nil
}

func (s *RegisterTaskService) GetActiveTask(promoterID uint) (task system.SysRegisterTask, err error) {
	if err = s.normalizeTimeoutClosedTasks(); err != nil {
		return task, err
	}
	if err = s.timeoutTasksByPromoter(promoterID); err != nil {
		return task, err
	}
	err = global.GVA_DB.Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Order("id desc").
		First(&task).Error
	if err != nil {
		return task, err
	}
	if recoverErr := s.restorePhoneBindSessionIfNeeded(&task); recoverErr != nil {
		return task, recoverErr
	}
	return
}

func (s *RegisterTaskService) SubmitStep(promoterID uint, req systemReq.RegisterTaskStep) (task system.SysRegisterTask, err error) {
	if req.TaskID == 0 {
		return task, errors.New("任务ID不能为空")
	}
	if req.Action == "" {
		req.Action = "submit"
	}

	if err = global.GVA_DB.Where("id = ? AND promoter_id = ?", req.TaskID, promoterID).First(&task).Error; err != nil {
		return task, errors.New("任务不存在")
	}
	global.GVA_LOG.Info("register_task_submit_step_start",
		append(taskLogFields(task),
			zap.String("action", req.Action),
			zap.String("step", req.Step),
		)...,
	)
	if task.FinishedAt != nil {
		return task, errors.New("任务已完成")
	}
	if !time.Now().Before(task.ExpiresAt) {
		return s.finishTask(task, system.RegisterTaskFailCodeTimeout, "任务超时自动完成")
	}

	switch req.Action {
	case "retry":
		if task.CurrentStep == system.RegisterTaskStepPhoneBind {
			runtimeCfg, cfgErr := s.getRegisterRuntimeConfig(task.LeaderID)
			if cfgErr != nil {
				return task, cfgErr
			}
			if prepErr := s.preparePhoneBindSMS(&task, runtimeCfg); prepErr != nil {
				return task, prepErr
			}
			return task, nil
		}
		task.RetryCount++
		if err = global.GVA_DB.Save(&task).Error; err != nil {
			global.GVA_LOG.Error("register_task_retry_save_failed", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		global.GVA_LOG.Info("register_task_retry_saved", append(taskLogFields(task), zap.Int("retryCount", task.RetryCount))...)
		return task, nil
	case "fail":
		failMsg := req.FailMessage
		if strings.TrimSpace(failMsg) == "" {
			failMsg = "地推手动结束任务"
		}
		global.GVA_LOG.Warn("register_task_manual_fail", append(taskLogFields(task), zap.String("reason", failMsg))...)
		return s.finishTask(task, system.RegisterTaskFailCodeManualFailed, failMsg)
	case "submit":
		return s.handleSubmit(task, req)
	default:
		return task, errors.New("不支持的action")
	}
}

func (s *RegisterTaskService) handleSubmit(task system.SysRegisterTask, req systemReq.RegisterTaskStep) (system.SysRegisterTask, error) {
	if req.Step != task.CurrentStep {
		return task, errors.New("步骤状态不一致，请刷新页面后重试")
	}
	if strings.TrimSpace(req.VerifyCode) == "" {
		return task, errors.New("验证码不能为空")
	}

	now := time.Now()
	runtimeCfg, err := s.getRegisterRuntimeConfig(task.LeaderID)
	if err != nil {
		return task, err
	}

	proxyURL, err := s.allocateProxyURL(runtimeCfg)
	if err != nil {
		global.GVA_LOG.Error("register_task_allocate_proxy_failed", append(taskLogFields(task), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("register_task_runtime_prepared",
		append(taskLogFields(task),
			zap.String("proxyPlatform", runtimeCfg.ProxyPlatform),
			zap.String("captchaPlatform", runtimeCfg.CaptchaPlatform),
			zap.Bool("proxyEnabled", strings.TrimSpace(proxyURL) != ""),
		)...,
	)

	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		global.GVA_LOG.Info("register_task_phone_bind_begin", taskLogFields(task)...)
		ps, ok := popPhoneBindSession(task.ID)
		if !ok || ps == nil || ps.Client == nil {
			if recoverErr := s.restorePhoneBindSessionIfNeeded(&task); recoverErr != nil {
				return task, recoverErr
			}
			return task, errors.New("服务重启后已重新发送验证码，请填写新验证码后再次提交")
		}
		defer ps.Client.Close()
		if err := ps.Client.SubmitSMS(qpi.SubmitSMSRequest{SMSCode: req.VerifyCode}); err != nil {
			global.GVA_LOG.Error("register_task_phone_bind_submit_sms_failed", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		loginResp, err := ps.Client.LoginAccount(qpi.LoginAccountRequest{SkipList: ""})
		if err != nil {
			global.GVA_LOG.Error("register_task_phone_bind_login_account_failed", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		if len(loginResp.QQList) == 0 {
			return s.finishTask(task, system.RegisterTaskFailCodeNoQQBound, "手机号无可用QQ账号")
		}
		global.GVA_LOG.Info("register_task_phone_bind_qq_list", append(taskLogFields(task), zap.Strings("qqList", loginResp.QQList))...)
		filteredQQList, err := s.filterQualifiedQQByNaicha(runtimeCfg, loginResp.QQList)
		if err != nil {
			global.GVA_LOG.Error("register_task_phone_bind_naicha_filter_failed", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		if len(filteredQQList) == 0 {
			return s.finishTask(task, system.RegisterTaskFailCodeNoQQBound, "奶茶筛选后无符合条件QQ")
		}
		task.QQCandidates = strings.Join(filteredQQList, "|")
		task.QQAccount = filteredQQList[0]
		task.CurrentStep = system.RegisterTaskStepChangePassword
		task.LastError = ""
		global.GVA_LOG.Info("register_task_phone_bind_success",
			append(taskLogFields(task),
				zap.Strings("qualifiedQQList", filteredQQList),
			)...,
		)
	case system.RegisterTaskStepChangePassword:
		global.GVA_LOG.Info("register_task_change_password_begin", taskLogFields(task)...)
		candidateQQList := splitQQList(task.QQCandidates)
		if len(candidateQQList) == 0 {
			candidateQQList = splitQQList(task.QQAccount)
		}
		if len(candidateQQList) == 0 {
			return task, errors.New("请先完成QQ查绑及筛选步骤")
		}
		changePwd := strings.TrimSpace(runtimeCfg.DefaultPassword)
		if changePwd == "" {
			global.GVA_LOG.Warn("register_task_change_password_missing_default_password", taskLogFields(task)...)
			return task, errors.New("管理员未配置默认改密密码")
		}
		changedQQList := splitQQList(task.QQChangedList)
		if len(changedQQList) > len(candidateQQList) {
			changedQQList = changedQQList[:len(candidateQQList)]
		}
		if len(changedQQList) >= len(candidateQQList) {
			task.CurrentStep = system.RegisterTaskStepLogin
			task.LastError = ""
			break
		}
		currentQQ := candidateQQList[len(changedQQList)]
		captcha, capErr := s.getCaptchaToken(runtimeCfg, qpi.ChangePasswordAppID)
		if capErr != nil {
			global.GVA_LOG.Error("register_task_change_password_get_captcha_failed", append(taskLogFields(task), zap.String("qq", currentQQ), zap.Error(capErr))...)
			return task, capErr
		}
		client := qpi.NewClient()
		if proxyURL != "" {
			if proxyErr := client.SetProxy(proxyURL); proxyErr != nil {
				global.GVA_LOG.Error("register_task_change_password_set_proxy_failed", append(taskLogFields(task), zap.String("qq", currentQQ), zap.Error(proxyErr))...)
				return task, proxyErr
			}
		}
		if err = s.changePasswordForQQ(client, currentQQ, task.Phone, req.VerifyCode, captcha, changePwd); err != nil {
			global.GVA_LOG.Error("register_task_change_password_single_failed", append(taskLogFields(task), zap.String("qq", currentQQ), zap.Error(err))...)
			return task, err
		}
		changedQQList = append(changedQQList, currentQQ)
		global.GVA_LOG.Info("register_task_change_password_single_success", append(taskLogFields(task), zap.String("qq", currentQQ))...)
		task.QQPassword = changePwd
		task.QQChangedList = strings.Join(changedQQList, "|")
		task.QQAccount = changedQQList[0]
		if len(changedQQList) >= len(candidateQQList) {
			task.ChangePasswordAt = &now
			task.CurrentStep = system.RegisterTaskStepLogin
			task.LastError = ""
			global.GVA_LOG.Info("register_task_change_password_success", append(taskLogFields(task), zap.Strings("changedQQList", changedQQList))...)
		} else {
			task.LastError = ""
			global.GVA_LOG.Info("register_task_change_password_progress", append(taskLogFields(task), zap.Int("done", len(changedQQList)), zap.Int("total", len(candidateQQList)))...)
		}
	case system.RegisterTaskStepLogin:
		global.GVA_LOG.Info("register_task_login_begin", taskLogFields(task)...)
		if strings.TrimSpace(task.QQPassword) == "" {
			return task, errors.New("QQ账号或密码为空，请先完成前置步骤")
		}
		changedQQList := splitQQList(task.QQChangedList)
		if len(changedQQList) == 0 {
			changedQQList = splitQQList(task.QQAccount)
		}
		if len(changedQQList) == 0 {
			return task, errors.New("无可登录QQ账号，请先完成改密")
		}
		tlv544Provider, pErr := buildTLV544ProviderFromConfig(runtimeCfg)
		if pErr != nil {
			return task, pErr
		}
		tlv553Provider, pErr := buildTLV553ProviderFromConfig(runtimeCfg)
		if pErr != nil {
			return task, pErr
		}
		loggedQQList := splitQQList(task.QQLoggedList)
		if len(loggedQQList) > len(changedQQList) {
			loggedQQList = loggedQQList[:len(changedQQList)]
		}
		if len(loggedQQList) >= len(changedQQList) {
			isDaren := true
			task.IsDaren = &isDaren
			task.QQAccount = loggedQQList[0]
			task.LoginAt = &now
			global.GVA_LOG.Info("register_task_login_success", append(taskLogFields(task), zap.Strings("loginSuccessQQ", loggedQQList), zap.Bool("isDaren", isDaren))...)
			return s.finishTask(task, 0, "")
		}
		currentQQ := changedQQList[len(loggedQQList)]
		loginClient := qpi.NewPasswordLoginClient()
		if proxyURL != "" {
			if proxyErr := loginClient.SetProxy(proxyURL); proxyErr != nil {
				_ = loginClient.Close()
				global.GVA_LOG.Error("register_task_login_set_proxy_failed", append(taskLogFields(task), zap.String("qq", currentQQ), zap.Error(proxyErr))...)
				return task, proxyErr
			}
		}
		loginReq := qpi.PasswordLoginRequest{
			UIN:            currentQQ,
			Password:       task.QQPassword,
			TLV544Provider: tlv544Provider,
			TLV553Provider: tlv553Provider,
			CaptchaProvider: func(captchaURL string) (string, error) {
				cap, err := s.getCaptchaToken(runtimeCfg, qpi.ChangePasswordAppID)
				if err != nil {
					return "", err
				}
				return cap.Ticket, nil
			},
			SMSCodeProvider: func() (string, error) {
				return req.VerifyCode, nil
			},
		}
		loginResp, loginErr := loginClient.Login(loginReq)
		_ = loginClient.Close()
		if loginErr != nil {
			global.GVA_LOG.Error("register_task_login_single_failed", append(taskLogFields(task), zap.String("qq", currentQQ), zap.Error(loginErr))...)
			return task, loginErr
		}
		global.GVA_LOG.Info("register_task_login_single_success", append(taskLogFields(task), zap.String("qq", currentQQ))...)
		loggedQQList = append(loggedQQList, currentQQ)
		task.QQLoggedList = strings.Join(loggedQQList, "|")
		if task.LoginCacheINI == "" {
			task.LoginCacheINI = buildCacheINI(loginResp.Cache.ToHexMap())
		} else {
			task.LoginCacheINI = task.LoginCacheINI + "\n" + buildCacheINI(loginResp.Cache.ToHexMap())
		}
		if len(loggedQQList) >= len(changedQQList) {
			isDaren := true
			task.IsDaren = &isDaren
			task.QQAccount = loggedQQList[0]
			task.LoginAt = &now
			global.GVA_LOG.Info("register_task_login_success", append(taskLogFields(task), zap.Strings("loginSuccessQQ", loggedQQList), zap.Bool("isDaren", isDaren))...)
			return s.finishTask(task, 0, "")
		}
		task.LastError = ""
		global.GVA_LOG.Info("register_task_login_progress", append(taskLogFields(task), zap.Int("done", len(loggedQQList)), zap.Int("total", len(changedQQList)))...)
	default:
		return task, errors.New("未知任务步骤")
	}

	if err := global.GVA_DB.Save(&task).Error; err != nil {
		global.GVA_LOG.Error("register_task_step_save_failed", append(taskLogFields(task), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("register_task_step_state_saved", taskLogFields(task)...)
	return task, nil
}

func (s *RegisterTaskService) finishTask(task system.SysRegisterTask, code int, msg string) (system.SysRegisterTask, error) {
	now := time.Now()
	task.StatusCode = &code
	task.LastError = msg
	task.FinishedAt = &now
	if err := global.GVA_DB.Save(&task).Error; err != nil {
		global.GVA_LOG.Error("register_task_finish_save_failed", append(taskLogFields(task), zap.Int("statusCode", code), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("register_task_finished",
		append(taskLogFields(task),
			zap.Int("statusCode", code),
			zap.String("message", msg),
		)...,
	)
	return task, nil
}

func (s *RegisterTaskService) timeoutTasksByPromoter(promoterID uint) error {
	now := time.Now()
	failCode := system.RegisterTaskFailCodeTimeout
	return global.GVA_DB.Model(&system.SysRegisterTask{}).
		Where("promoter_id = ? AND finished_at IS NULL AND expires_at <= ?", promoterID, now).
		Updates(map[string]interface{}{
			"status_code": failCode,
			"last_error":  "任务超时自动完成",
			"finished_at": now,
		}).Error
}

func (s *RegisterTaskService) GetTaskList(operatorRole uint, operatorID uint, req systemReq.RegisterTaskList) (registerTaskListResult, error) {
	_ = s.normalizeTimeoutClosedTasks()
	_ = s.timeoutUnfinishedTasks()
	db := global.GVA_DB.Model(&system.SysRegisterTask{}).Preload("Promoter").Preload("Leader")

	switch operatorRole {
	case roleSuperAdmin, roleAdmin:
		if req.LeaderID != 0 {
			db = db.Where("leader_id = ?", req.LeaderID)
		}
		if req.PromoterID != 0 {
			db = db.Where("promoter_id = ?", req.PromoterID)
		}
	case roleLeader:
		db = db.Where("leader_id = ?", operatorID)
		if req.PromoterID != 0 {
			db = db.Where("promoter_id = ?", req.PromoterID)
		}
	case rolePromoter:
		db = db.Where("promoter_id = ?", operatorID)
	default:
		return registerTaskListResult{}, errors.New("无权限查看任务")
	}

	if req.StatusCode != nil {
		db = db.Where("status_code = ?", *req.StatusCode)
	}
	if req.Unfinished != nil {
		if *req.Unfinished {
			db = db.Where("finished_at IS NULL")
		} else {
			db = db.Where("finished_at IS NOT NULL")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return registerTaskListResult{}, err
	}

	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var list []system.SysRegisterTask
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&list).Error; err != nil {
		return registerTaskListResult{}, err
	}

	statDB := global.GVA_DB.Model(&system.SysRegisterTask{})
	switch operatorRole {
	case roleSuperAdmin, roleAdmin:
		if req.LeaderID != 0 {
			statDB = statDB.Where("leader_id = ?", req.LeaderID)
		}
		if req.PromoterID != 0 {
			statDB = statDB.Where("promoter_id = ?", req.PromoterID)
		}
	case roleLeader:
		statDB = statDB.Where("leader_id = ?", operatorID)
		if req.PromoterID != 0 {
			statDB = statDB.Where("promoter_id = ?", req.PromoterID)
		}
	case rolePromoter:
		statDB = statDB.Where("promoter_id = ?", operatorID)
	}
	type taskCounter struct {
		Success    int64 `gorm:"column:success"`
		Failed     int64 `gorm:"column:failed"`
		Processing int64 `gorm:"column:processing"`
	}
	var counter taskCounter
	if err := statDB.
		Select(`
			COALESCE(SUM(CASE WHEN finished_at IS NOT NULL AND status_code = 0 THEN 1 ELSE 0 END), 0) AS success,
			COALESCE(SUM(CASE WHEN finished_at IS NOT NULL AND (status_code <> 0 OR status_code IS NULL) THEN 1 ELSE 0 END), 0) AS failed,
			COALESCE(SUM(CASE WHEN finished_at IS NULL THEN 1 ELSE 0 END), 0) AS processing`).
		Scan(&counter).Error; err != nil {
		return registerTaskListResult{}, err
	}

	return registerTaskListResult{
		List:       list,
		Total:      total,
		Success:    counter.Success,
		Failed:     counter.Failed,
		Processing: counter.Processing,
	}, nil
}

func (s *RegisterTaskService) GetSummary(operatorRole uint, operatorID uint, leaderID uint) (systemRes.RegisterTaskSummaryResponse, error) {
	if operatorRole != roleSuperAdmin && operatorRole != roleAdmin && operatorRole != roleLeader {
		return systemRes.RegisterTaskSummaryResponse{}, errors.New("无权限查看统计")
	}

	_ = s.normalizeTimeoutClosedTasks()
	_ = s.timeoutUnfinishedTasks()

	type summaryRow struct {
		LeaderID        *uint  `gorm:"column:leader_id"`
		LeaderName      string `gorm:"column:leader_name"`
		PromoterID      uint   `gorm:"column:promoter_id"`
		PromoterName    string `gorm:"column:promoter_name"`
		SuccessCount    int64  `gorm:"column:success_count"`
		FailCount       int64  `gorm:"column:fail_count"`
		ProcessingCount int64  `gorm:"column:processing_count"`
	}

	db := global.GVA_DB.Table("sys_register_tasks t").
		Select(`
			t.leader_id,
			leader.nick_name AS leader_name,
			t.promoter_id,
			promoter.nick_name AS promoter_name,
			SUM(CASE WHEN t.finished_at IS NOT NULL AND t.status_code = 0 THEN 1 ELSE 0 END) AS success_count,
			SUM(CASE WHEN t.finished_at IS NOT NULL AND (t.status_code <> 0 OR t.status_code IS NULL) THEN 1 ELSE 0 END) AS fail_count,
			SUM(CASE WHEN t.finished_at IS NULL THEN 1 ELSE 0 END) AS processing_count`).
		Joins("LEFT JOIN sys_users promoter ON promoter.id = t.promoter_id").
		Joins("LEFT JOIN sys_users leader ON leader.id = t.leader_id")

	if operatorRole == roleLeader {
		db = db.Where("t.leader_id = ?", operatorID)
	} else if leaderID != 0 {
		db = db.Where("t.leader_id = ?", leaderID)
	}

	var promoterRows []summaryRow
	if err := db.Group("t.leader_id, leader.nick_name, t.promoter_id, promoter.nick_name").Scan(&promoterRows).Error; err != nil {
		return systemRes.RegisterTaskSummaryResponse{}, err
	}

	leaderMap := map[uint]systemRes.RegisterTaskSummaryItem{}
	promoters := make([]systemRes.RegisterTaskSummaryItem, 0, len(promoterRows))
	for _, row := range promoterRows {
		item := systemRes.RegisterTaskSummaryItem{
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
			leaderAgg := leaderMap[item.LeaderID]
			leaderAgg.LeaderID = item.LeaderID
			leaderAgg.LeaderName = item.LeaderName
			leaderAgg.SuccessCount += item.SuccessCount
			leaderAgg.FailCount += item.FailCount
			leaderAgg.ProcessingCount += item.ProcessingCount
			leaderMap[item.LeaderID] = leaderAgg
		}
	}

	leaders := make([]systemRes.RegisterTaskSummaryItem, 0, len(leaderMap))
	for _, item := range leaderMap {
		leaders = append(leaders, item)
	}
	return systemRes.RegisterTaskSummaryResponse{
		Leaders:   leaders,
		Promoters: promoters,
	}, nil
}

func (s *RegisterTaskService) timeoutUnfinishedTasks() error {
	now := time.Now()
	failCode := system.RegisterTaskFailCodeTimeout
	return global.GVA_DB.Model(&system.SysRegisterTask{}).
		Where("finished_at IS NULL AND expires_at <= ?", now).
		Updates(map[string]interface{}{
			"status_code": failCode,
			"last_error":  "任务超时自动完成",
			"finished_at": now,
		}).Error
}

// normalizeTimeoutClosedTasks 兜底修正：已完成且超时，但状态码为空的数据统一标记为超时失败
func (s *RegisterTaskService) normalizeTimeoutClosedTasks() error {
	now := time.Now()
	failCode := system.RegisterTaskFailCodeTimeout
	return global.GVA_DB.Model(&system.SysRegisterTask{}).
		Where("finished_at IS NOT NULL AND status_code IS NULL AND expires_at <= ?", now).
		Updates(map[string]interface{}{
			"status_code": failCode,
			"last_error":  "任务超时自动完成",
		}).Error
}

func (s *RegisterTaskService) filterQualifiedQQByNaicha(cfg systemRegisterConfig, qqList []string) ([]string, error) {
	appID := strings.TrimSpace(cfg.NaichaAppID)
	secret := strings.TrimSpace(cfg.NaichaSecret)
	ckMd5 := strings.TrimSpace(cfg.NaichaCKMd5)
	if appID == "" || secret == "" {
		return nil, errors.New("管理员未配置奶茶平台 appId/secret")
	}
	client := NewNCClient(appID, secret, ckMd5)
	drResults, err := client.NcQueryDR(qqList, 120000, 4)
	if err != nil {
		return nil, err
	}
	qlResults, err := client.NcQueryQL(qqList, 120000, 4)
	if err != nil {
		return nil, err
	}

	drMap := make(map[string]NCDRResult)
	for _, item := range drResults {
		drMap[item.UIN] = item
	}
	qlMap := make(map[string]NCQLResult)
	for _, item := range qlResults {
		qlMap[item.UIN] = item
	}

	filtered := make([]string, 0, len(qqList))
	for _, qq := range qqList {
		dr, ok1 := drMap[qq]
		ql, ok2 := qlMap[qq]
		if !ok1 || !ok2 {
			continue
		}
		// 按需求过滤：达人(连续在线<1天) 且 q龄>1 且 等级>1级
		if dr.PhoneOnlineDay < 1 && ql.Age > 1 && ql.Level > 1 {
			filtered = append(filtered, qq)
		}
	}
	return filtered, nil
}

func (s *RegisterTaskService) changePasswordForQQ(client *qpi.Client, qq string, phone string, verifyCode string, captcha *captchaToken, newPassword string) error {
	verifyCaptchaResp, err := client.VerifyCaptcha(qpi.CaptchaVerifyRequest{
		QQ:      qq,
		Randstr: captcha.Randstr,
		Ticket:  captcha.Ticket,
	})
	if err != nil {
		return err
	}
	if err := verifyCaptchaResp.Error(); err != nil {
		return err
	}
	riskResp, err := client.CheckRisk(qpi.RiskCheckRequest{
		QQ:      qq,
		Randstr: captcha.Randstr,
		Ticket:  captcha.Ticket,
	})
	if err != nil {
		return err
	}
	if err := riskResp.Error(); err != nil {
		return err
	}
	queryPhoneResp, err := client.QueryPhone(qpi.PhoneQueryRequest{
		QQ:      qq,
		Randstr: captcha.Randstr,
	})
	if err != nil {
		return err
	}
	if err := queryPhoneResp.Error(); err != nil {
		return err
	}
	verifyPhoneResp, err := client.VerifyPhone(qpi.PhoneVerifyRequest{
		QQ:       qq,
		Randstr:  captcha.Randstr,
		Ticket:   captcha.Ticket,
		Phone:    phone,
		AreaCode: "+86",
	})
	if err != nil {
		return err
	}
	if err := verifyPhoneResp.Error(); err != nil {
		return err
	}
	sendSMSResp, err := client.SendSMS(qpi.SMSSendRequest{
		QQ:       qq,
		Randstr:  captcha.Randstr,
		Ticket:   captcha.Ticket,
		Phone:    phone,
		AreaCode: "+86",
	})
	if err != nil {
		return err
	}
	if err := sendSMSResp.Error(); err != nil {
		return err
	}
	verifySMSResp, err := client.VerifySMSCode(qpi.SMSCodeVerifyRequest{
		QQ:       qq,
		Randstr:  captcha.Randstr,
		Ticket:   captcha.Ticket,
		Phone:    phone,
		Code:     verifyCode,
		AreaCode: "+86",
	})
	if err != nil {
		return err
	}
	if err := verifySMSResp.Error(); err != nil {
		return err
	}
	changePwdResp, err := client.ChangePassword(qpi.PasswordChangeRequest{
		QQ:       qq,
		Randstr:  captcha.Randstr,
		Password: newPassword,
		Key:      verifySMSResp.Key,
		Phone:    phone,
		AreaCode: "+86",
	})
	if err != nil {
		return err
	}
	return changePwdResp.Error()
}

func splitQQList(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func setPhoneBindSession(taskID uint, client *qpi.LoginClient) {
	registerTaskPhoneBindSessions.Store(taskID, &phoneBindSession{
		Client:    client,
		CreatedAt: time.Now(),
	})
}

func hasPhoneBindSession(taskID uint) bool {
	_, ok := registerTaskPhoneBindSessions.Load(taskID)
	return ok
}

func popPhoneBindSession(taskID uint) (*phoneBindSession, bool) {
	v, ok := registerTaskPhoneBindSessions.LoadAndDelete(taskID)
	if !ok {
		return nil, false
	}
	ps, ok := v.(*phoneBindSession)
	if !ok || ps == nil {
		return nil, false
	}
	return ps, true
}

func (s *RegisterTaskService) preparePhoneBindSMS(task *system.SysRegisterTask, runtimeCfg systemRegisterConfig) error {
	captcha, err := s.getCaptchaToken(runtimeCfg, qpi.FindBindAppID)
	if err != nil {
		return err
	}
	proxyURL, err := s.allocateProxyURL(runtimeCfg)
	if err != nil {
		global.GVA_LOG.Error("allocate proxy url failed", zap.Error(err))
		return err
	}

	client := qpi.NewLoginClient()
	if proxyURL != "" {
		if err := client.SetProxy(proxyURL); err != nil {
			_ = client.Close()
			global.GVA_LOG.Error("set proxy failed", zap.Error(err))
			return err
		}
	}
	if _, err := client.GetLoginSMS(qpi.LoginSMSRequest{
		AreaCode: "+86",
		Phone:    task.Phone,
		Randstr:  captcha.Randstr,
		Ticket:   captcha.Ticket,
	}); err != nil {
		_ = client.Close()
		global.GVA_LOG.Error("get login sms failed", zap.Error(err))
		return err
	}

	setPhoneBindSession(task.ID, client)
	task.CaptchaRandstr = captcha.Randstr
	task.CaptchaTicket = captcha.Ticket
	task.LastError = "验证码已发送，请输入后提交"
	if err := global.GVA_DB.Save(task).Error; err != nil {
		global.GVA_LOG.Error("save task failed", zap.Error(err))
		return err
	}
	global.GVA_LOG.Info("phone send sms success", taskLogFields(*task)...)
	return nil
}

func (s *RegisterTaskService) restorePhoneBindSessionIfNeeded(task *system.SysRegisterTask) error {
	if task == nil || task.FinishedAt != nil || task.CurrentStep != system.RegisterTaskStepPhoneBind {
		return nil
	}
	if hasPhoneBindSession(task.ID) {
		return nil
	}
	runtimeCfg, err := s.getRegisterRuntimeConfig(task.LeaderID)
	if err != nil {
		return err
	}
	if err := s.preparePhoneBindSMS(task, runtimeCfg); err != nil {
		return err
	}
	global.GVA_LOG.Info("phone bind session recovered after restart", taskLogFields(*task)...)
	return nil
}
