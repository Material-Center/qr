package system

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const registerTaskRunnerRecoverAction = "__recover__"
const registerTaskRunnerIdleTTL = 5 * time.Minute
const registerTaskLastErrorMaxLen = 1024

// persistRunnerStepError 异步 runner 中 handleSubmit/retry 失败时写入 last_error，便于前端轮询 GetActiveTask 展示
func persistRunnerStepError(taskID uint, promoterID uint, err error) {
	if err == nil || taskID == 0 {
		return
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) > registerTaskLastErrorMaxLen {
		msg = msg[:registerTaskLastErrorMaxLen] + "…"
	}
	db := global.GVA_DB.Model(&system.SysRegisterTask{}).Where("id = ? AND finished_at IS NULL", taskID)
	if promoterID != 0 {
		db = db.Where("promoter_id = ?", promoterID)
	}
	if db.Update("last_error", msg).Error != nil {
		global.GVA_LOG.Warn("【注册任务】runner失败写入last_error失败",
			zap.Uint("taskId", taskID), zap.Uint("promoterId", promoterID), zap.Error(err))
	}
}

type registerTaskEvent struct {
	Action      string
	Step        string
	VerifyCode  string
	FailMessage string
}

type registerTaskRunner struct {
	taskID     uint
	promoterID uint
	eventCh    chan registerTaskEvent
}

var registerTaskRunnerOnce sync.Once
var registerTaskRunners sync.Map // map[taskID]*registerTaskRunner

func startRegisterTaskRunnerDaemon() {
	registerTaskRunnerOnce.Do(func() {
		go func() {
			svc := &RegisterTaskService{}
			// 首次启动快速拉起恢复，之后周期巡检。
			svc.scanAndEnsureRunners()
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				svc.scanAndEnsureRunners()
			}
		}()
	})
}

func ensureRegisterTaskRunner(taskID uint, promoterID uint) {
	if taskID == 0 {
		return
	}
	if v, ok := registerTaskRunners.Load(taskID); ok {
		r, _ := v.(*registerTaskRunner)
		if r != nil && promoterID != 0 && r.promoterID == 0 {
			r.promoterID = promoterID
		}
		return
	}
	runner := &registerTaskRunner{
		taskID:     taskID,
		promoterID: promoterID,
		eventCh:    make(chan registerTaskEvent, 32),
	}
	actual, loaded := registerTaskRunners.LoadOrStore(taskID, runner)
	if loaded {
		if ar, _ := actual.(*registerTaskRunner); ar != nil && promoterID != 0 && ar.promoterID == 0 {
			ar.promoterID = promoterID
		}
		return
	}
	go runRegisterTaskRunner(runner)
	_ = enqueueRegisterTaskEvent(taskID, promoterID, registerTaskEvent{Action: registerTaskRunnerRecoverAction})
}

// enqueueContinueLoginAfterChangePassword 改密步骤已落库为 login 后，再投递一次 submit/login，
// 否则 runner 只处理完改密就空闲，必须等前端再点提交才会进入登录（用户会感觉「卡住」）。
func enqueueContinueLoginAfterChangePassword(task system.SysRegisterTask) {
	if task.ID == 0 || task.PromoterID == 0 {
		return
	}
	if err := enqueueRegisterTaskEvent(task.ID, task.PromoterID, registerTaskEvent{
		Action: "submit",
		Step:   system.RegisterTaskStepLogin,
	}); err != nil {
		global.GVA_LOG.Warn("【注册任务】改密完成-自动进入登录投递失败",
			append(taskLogFields(task), zap.Error(err))...)
		return
	}
	global.GVA_LOG.Info("【注册任务】改密完成-已自动投递登录步骤", taskLogFields(task)...)
}

func enqueueRegisterTaskEvent(taskID uint, promoterID uint, event registerTaskEvent) error {
	if taskID == 0 {
		return errors.New("任务ID不能为空")
	}
	ensureRegisterTaskRunner(taskID, promoterID)
	v, ok := registerTaskRunners.Load(taskID)
	if !ok {
		return errors.New("任务runner不存在")
	}
	runner, ok := v.(*registerTaskRunner)
	if !ok || runner == nil {
		return errors.New("任务runner状态异常")
	}
	select {
	case runner.eventCh <- event:
		return nil
	default:
		return errors.New("任务处理繁忙，请稍后重试")
	}
}

func runRegisterTaskRunner(runner *registerTaskRunner) {
	if runner == nil {
		return
	}
	svc := &RegisterTaskService{}
	idleTimer := time.NewTimer(registerTaskRunnerIdleTTL)
	defer idleTimer.Stop()
	for {
		select {
		case event := <-runner.eventCh:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(registerTaskRunnerIdleTTL)
			finished, err := svc.processRunnerEvent(runner, event)
			if err != nil {
				global.GVA_LOG.Warn("【注册任务】runner处理事件失败",
					zap.Uint("taskId", runner.taskID),
					zap.Uint("promoterId", runner.promoterID),
					zap.String("action", event.Action),
					zap.Error(err),
				)
			}
			if finished {
				registerTaskRunners.Delete(runner.taskID)
				return
			}
		case <-idleTimer.C:
			registerTaskRunners.Delete(runner.taskID)
			global.GVA_LOG.Info("【注册任务】runner空闲回收",
				zap.Uint("taskId", runner.taskID),
				zap.Uint("promoterId", runner.promoterID),
			)
			return
		}
	}
}

func (s *RegisterTaskService) processRunnerEvent(runner *registerTaskRunner, event registerTaskEvent) (bool, error) {
	var task system.SysRegisterTask
	db := global.GVA_DB.Where("id = ?", runner.taskID)
	if runner.promoterID != 0 {
		db = db.Where("promoter_id = ?", runner.promoterID)
	}
	if err := db.First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	if task.FinishedAt != nil {
		clearTaskSession(task.ID)
		return true, nil
	}
	if !time.Now().Before(task.ExpiresAt) {
		if _, err := s.finishTask(task, system.RegisterTaskFailCodeTimeout, "任务超时自动完成"); err != nil {
			return false, err
		}
		return true, nil
	}

	switch strings.TrimSpace(event.Action) {
	case "", "submit":
		req := systemReq.RegisterTaskStep{
			TaskID:     task.ID,
			Step:       task.CurrentStep,
			Action:     "submit",
			VerifyCode: strings.TrimSpace(event.VerifyCode),
		}
		if strings.TrimSpace(event.Step) != "" {
			req.Step = strings.TrimSpace(event.Step)
		}
		_, err := s.handleSubmit(task, req)
		if err != nil {
			persistRunnerStepError(runner.taskID, runner.promoterID, err)
		}
		return false, err
	case "retry":
		err := s.executeRetryAction(&task)
		if err != nil {
			persistRunnerStepError(runner.taskID, runner.promoterID, err)
		}
		return false, err
	case "fail":
		failMsg := strings.TrimSpace(event.FailMessage)
		if failMsg == "" {
			failMsg = "地推手动结束任务"
		}
		_, err := s.finishTask(task, system.RegisterTaskFailCodeManualFailed, failMsg)
		return err == nil, err
	case registerTaskRunnerRecoverAction:
		if err := s.restoreTaskProgressIfNeeded(&task); err != nil {
			return false, err
		}
		if task.CurrentStep == system.RegisterTaskStepLogin {
			if ts, ok := getLoginSession(task.ID); !ok || ts == nil || ts.LoginClient == nil {
				_, err := s.handleSubmit(task, systemReq.RegisterTaskStep{
					TaskID: task.ID,
					Step:   task.CurrentStep,
					Action: "submit",
				})
				if err != nil {
					persistRunnerStepError(runner.taskID, runner.promoterID, err)
				}
				return false, err
			}
		}
		return false, nil
	default:
		return false, errors.New("不支持的action")
	}
}

func (s *RegisterTaskService) scanAndEnsureRunners() {
	if global.GVA_DB == nil {
		return
	}
	type pendingTask struct {
		ID         uint
		PromoterID uint
	}
	var tasks []pendingTask
	if err := global.GVA_DB.Model(&system.SysRegisterTask{}).
		Select("id,promoter_id").
		Where("finished_at IS NULL").
		Find(&tasks).Error; err != nil {
		global.GVA_LOG.Warn("【注册任务】runner巡检失败", zap.Error(err))
		return
	}
	for _, t := range tasks {
		ensureRegisterTaskRunner(t.ID, t.PromoterID)
	}
}

func (s *RegisterTaskService) executeRetryAction(task *system.SysRegisterTask) error {
	if task == nil {
		return errors.New("任务不存在")
	}
	task.RetryCount++
	if err := global.GVA_DB.Save(task).Error; err != nil {
		return err
	}
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		runtimeCfg, cfgErr := s.getRegisterRuntimeConfig(task.LeaderID)
		if cfgErr != nil {
			return cfgErr
		}
		clearPhoneBindSession(task.ID)
		return s.preparePhoneBindSMS(task, runtimeCfg)
	case system.RegisterTaskStepChangePassword:
		clearChangePasswordSession(task.ID)
		runtimeCfg, cfgErr := s.getRegisterRuntimeConfig(task.LeaderID)
		if cfgErr != nil {
			return cfgErr
		}
		proxyURL, pErr := s.allocateProxyURL(runtimeCfg)
		if pErr != nil {
			return pErr
		}
		return s.prepareChangePasswordSMSIfNeeded(task, runtimeCfg, proxyURL, true)
	case system.RegisterTaskStepLogin:
		clearLoginSession(task.ID)
		_, err := s.handleSubmit(*task, systemReq.RegisterTaskStep{
			TaskID: task.ID,
			Step:   task.CurrentStep,
			Action: "submit",
		})
		return err
	default:
		return nil
	}
}
