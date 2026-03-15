package system

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
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

func (s *RegisterTaskService) CreateTask(promoterID uint, phone string) (task system.SysRegisterTask, err error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return task, errors.New("手机号不能为空")
	}

	var duplicateCount int64
	if err = global.GVA_DB.Model(&system.SysRegisterTask{}).Where("phone = ?", phone).Count(&duplicateCount).Error; err != nil {
		return task, err
	}
	if duplicateCount > 0 {
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
		return task, err
	}
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
	if task.FinishedAt != nil {
		return task, errors.New("任务已完成")
	}
	if !time.Now().Before(task.ExpiresAt) {
		return s.finishTask(task, system.RegisterTaskFailCodeTimeout, "任务超时自动完成")
	}

	switch req.Action {
	case "retry":
		task.RetryCount++
		if err = global.GVA_DB.Save(&task).Error; err != nil {
			return task, err
		}
		return task, nil
	case "fail":
		failMsg := req.FailMessage
		if strings.TrimSpace(failMsg) == "" {
			failMsg = "地推手动结束任务"
		}
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
	// mock 行为：0000 系统失败，1111 业务失败，其它均视为成功
	if req.VerifyCode == "0000" {
		return s.finishTask(task, system.RegisterTaskFailCodeMockSysFail, "模拟系统失败")
	}
	if req.VerifyCode == "1111" {
		return s.finishTask(task, system.RegisterTaskFailCodeMockBizFail, "模拟业务失败")
	}

	now := time.Now()
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		if len(task.Phone) >= 4 && strings.HasSuffix(task.Phone, "0000") {
			return s.finishTask(task, system.RegisterTaskFailCodeNoQQBound, "手机号无可用QQ账号")
		}
		task.QQAccount = "qq_" + tailN(task.Phone, 6)
		task.CurrentStep = system.RegisterTaskStepChangePassword
		task.LastError = ""
	case system.RegisterTaskStepChangePassword:
		task.QQPassword = "Mock@123456"
		task.ChangePasswordAt = &now
		task.CurrentStep = system.RegisterTaskStepLogin
		task.LastError = ""
	case system.RegisterTaskStepLogin:
		isDaren := tailN(task.Phone, 1) == "8" || tailN(task.Phone, 1) == "9"
		task.IsDaren = &isDaren
		task.LoginAt = &now
		task.LoginCacheINI = fmt.Sprintf("[session]\nphone=%s\nqq=%s\ntoken=mock_token_%d\n", task.Phone, task.QQAccount, now.Unix())
		return s.finishTask(task, 0, "")
	default:
		return task, errors.New("未知任务步骤")
	}

	if err := global.GVA_DB.Save(&task).Error; err != nil {
		return task, err
	}
	return task, nil
}

func (s *RegisterTaskService) finishTask(task system.SysRegisterTask, code int, msg string) (system.SysRegisterTask, error) {
	now := time.Now()
	task.StatusCode = &code
	task.LastError = msg
	task.FinishedAt = &now
	if err := global.GVA_DB.Save(&task).Error; err != nil {
		return task, err
	}
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

func tailN(val string, n int) string {
	if len(val) <= n {
		return val
	}
	return val[len(val)-n:]
}
