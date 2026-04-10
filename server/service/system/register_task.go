package system

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Material-Center/qpi"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const registerTaskTimeout = 10 * time.Minute
const registerTaskLoginVerifyWaitTimeout = 3 * time.Minute
const registerTaskLoginNeedVerifyTip = "当前登录触发短信验证，请输入验证码后重试"

var registerTaskNaichaBypassPhones = map[string]struct{}{
	"15524993221": {},
	"19122164324": {},
	"19111277334": {},
	"19146017340": {},
}

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

type taskSession struct {
	PhoneBindClient *qpi.LoginClient
	ChangePwdClient *qpi.Client
	ChangePwdQQ     string
	ChangePwdRand   string
	ChangePwdTicket string
	LoginClient     *qpi.PasswordLoginClient
	LoginQQ         string
	LoginCodeCh     chan string
	CreatedAt       time.Time
}

var registerTaskSessions sync.Map // map[taskID]*taskSession

func taskLogFields(task system.SysRegisterTask) []zap.Field {
	fields := []zap.Field{
		zap.Uint("taskId", task.ID),
		zap.String("phone", task.Phone),
		zap.String("step", task.CurrentStep),
		zap.Uint("promoterId", task.PromoterID),
	}
	if task.LeaderID != nil {
		fields = append(fields, zap.Uint("leaderId", *task.LeaderID))
	}
	return fields
}

// taskLogFieldsWithOpQQ 当前步骤正在处理的 QQ（改密/登录循环），与 task 上已落库的 qq 区分
func taskLogFieldsWithOpQQ(task system.SysRegisterTask, opQQ string) []zap.Field {
	base := taskLogFields(task)
	if q := strings.TrimSpace(opQQ); q != "" {
		base = append(base, zap.String("opQQ", q))
	}
	return base
}

func buildStableLoginDeviceProfile(uin string) (androidID string, guidHex string, qimei16 string) {
	trimmedUIN := strings.TrimSpace(uin)
	androidMD5 := md5.Sum([]byte(trimmedUIN))
	androidHex := strings.ToLower(hex.EncodeToString(androidMD5[:]))
	if len(androidHex) >= 24 {
		androidID = androidHex[8:24]
	} else {
		androidID = androidHex
	}

	guidMD5 := md5.Sum([]byte(androidID + "02:00:00:00:00:00"))
	guidHex = strings.ToUpper(hex.EncodeToString(guidMD5[:]))

	qimeiMD5 := md5.Sum([]byte(androidID))
	qimeiHex := strings.ToLower(hex.EncodeToString(qimeiMD5[:]))
	if len(qimeiHex) >= 24 {
		qimei16 = qimeiHex[8:24] + "801cf1e0100014619804"
	}

	return androidID, guidHex, qimei16
}

func init() {
	startRegisterTaskRunnerDaemon()
}

func (s *RegisterTaskService) CreateTask(promoterID uint, phone string) (task system.SysRegisterTask, err error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return task, errors.New("手机号不能为空")
	}
	global.GVA_LOG.Info("【注册任务】开始创建", zap.Uint("promoterId", promoterID), zap.String("phone", phone))

	if err = s.timeoutTasksByPromoter(promoterID); err != nil {
		global.GVA_LOG.Error("【注册任务】创建前地推超时批量处理失败", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Error(err))
		return task, err
	}

	var promoter system.SysUser
	if err = global.GVA_DB.Select("id, leader_id").Where("id = ?", promoterID).First(&promoter).Error; err != nil {
		global.GVA_LOG.Warn("【注册任务】地推账号不存在", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Error(err))
		return task, errors.New("地推账号不存在")
	}

	var existed system.SysRegisterTask
	findErr := global.GVA_DB.Where("phone = ?", phone).Order("id DESC").First(&existed).Error
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		global.GVA_LOG.Error("【注册任务】创建前查重失败", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Error(findErr))
		return task, findErr
	}
	if findErr == nil {
		if existed.FinishedAt == nil {
			global.GVA_LOG.Warn("【注册任务】手机号存在进行中任务", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Uint("taskId", existed.ID))
			return task, errors.New("手机号存在进行中任务，暂不能重复提交")
		}
		if existed.StatusCode != nil && *existed.StatusCode == 0 {
			global.GVA_LOG.Warn("【注册任务】手机号存在成功任务", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Uint("taskId", existed.ID))
			return task, errors.New("手机号已提交成功任务，不能重复创建")
		}
		if existed.PromoterID != promoterID {
			global.GVA_LOG.Warn("【注册任务】手机号失败任务归属其他地推，拒绝重建",
				zap.Uint("promoterId", promoterID),
				zap.String("phone", phone),
				zap.Uint("taskId", existed.ID),
				zap.Uint("ownerPromoterId", existed.PromoterID),
			)
			return task, errors.New("手机号已有历史任务，不能由当前地推重新提交")
		}
		clearTaskSession(existed.ID)
		existed = buildResetTaskForReuse(existed, promoterID, promoter.LeaderID)
		if err = global.GVA_DB.Save(&existed).Error; err != nil {
			global.GVA_LOG.Error("【注册任务】重置失败任务失败", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Uint("taskId", existed.ID), zap.Error(err))
			return task, err
		}
		task = existed
		global.GVA_LOG.Info("【注册任务】失败任务已重置并重新开始", taskLogFields(task)...)
	} else {
		task = system.SysRegisterTask{
			Phone:       phone,
			CurrentStep: system.RegisterTaskStepPhoneBind,
			PromoterID:  promoterID,
			LeaderID:    promoter.LeaderID,
			ExpiresAt:   time.Now().Add(registerTaskTimeout),
		}
		if err = global.GVA_DB.Create(&task).Error; err != nil {
			global.GVA_LOG.Error("【注册任务】写入任务失败", zap.Uint("promoterId", promoterID), zap.String("phone", phone), zap.Error(err))
			return task, err
		}
	}
	global.GVA_LOG.Info("【注册任务】任务已落库，准备发短信", taskLogFields(task)...)
	ensureRegisterTaskRunner(task.ID, promoterID)
	task.LastError = "任务已创建，准备发送验证码"
	if err = global.GVA_DB.Save(&task).Error; err != nil {
		return task, err
	}
	if err = enqueueRegisterTaskEvent(task.ID, promoterID, registerTaskEvent{
		Action: registerTaskRunnerStartAction,
		Step:   task.CurrentStep,
	}); err != nil {
		global.GVA_LOG.Error("【注册任务】创建后自动触发失败", append(taskLogFields(task), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("【注册任务】创建完成", taskLogFields(task)...)
	return task, nil
}

func (s *RegisterTaskService) GetActiveTask(promoterID uint) (task system.SysRegisterTask, err error) {
	if err = s.normalizeTimeoutClosedTasks(); err != nil {
		global.GVA_LOG.Error("【注册任务】拉取当前任务-修正超时状态失败", zap.Uint("promoterId", promoterID), zap.Error(err))
		return task, err
	}
	if err = s.timeoutTasksByPromoter(promoterID); err != nil {
		global.GVA_LOG.Error("【注册任务】拉取当前任务-地推超时批量失败", zap.Uint("promoterId", promoterID), zap.Error(err))
		return task, err
	}
	err = global.GVA_DB.Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Order("id desc").
		First(&task).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.GVA_LOG.Error("【注册任务】拉取当前任务-查询失败", zap.Uint("promoterId", promoterID), zap.Error(err))
		}
		return task, err
	}
	global.GVA_LOG.Info("【注册任务】拉取当前任务成功", taskLogFields(task)...)
	return
}

func (s *RegisterTaskService) GetActiveTasks(promoterID uint) (tasks []system.SysRegisterTask, err error) {
	if err = s.normalizeTimeoutClosedTasks(); err != nil {
		global.GVA_LOG.Error("【注册任务】拉取当前任务列表-修正超时状态失败", zap.Uint("promoterId", promoterID), zap.Error(err))
		return nil, err
	}
	if err = s.timeoutTasksByPromoter(promoterID); err != nil {
		global.GVA_LOG.Error("【注册任务】拉取当前任务列表-地推超时批量失败", zap.Uint("promoterId", promoterID), zap.Error(err))
		return nil, err
	}
	err = global.GVA_DB.Where("promoter_id = ? AND finished_at IS NULL", promoterID).
		Order("id desc").
		Find(&tasks).Error
	if err != nil {
		global.GVA_LOG.Error("【注册任务】拉取当前任务列表-查询失败", zap.Uint("promoterId", promoterID), zap.Error(err))
		return nil, err
	}
	return tasks, nil
}

func (s *RegisterTaskService) SubmitStep(promoterID uint, req systemReq.RegisterTaskStep) (task system.SysRegisterTask, err error) {
	if req.TaskID == 0 {
		return task, errors.New("任务ID不能为空")
	}
	if req.Action == "" {
		req.Action = "submit"
	}

	if err = global.GVA_DB.Where("id = ? AND promoter_id = ?", req.TaskID, promoterID).First(&task).Error; err != nil {
		global.GVA_LOG.Warn("【注册任务】提交步骤-任务不存在", zap.Uint("promoterId", promoterID), zap.Uint("taskId", req.TaskID), zap.String("action", req.Action), zap.Error(err))
		return task, errors.New("任务不存在")
	}
	global.GVA_LOG.Info("【注册任务】提交步骤-开始",
		append(taskLogFields(task),
			zap.String("action", req.Action),
			zap.String("reqStep", req.Step),
		)...,
	)
	if task.FinishedAt != nil {
		return task, errors.New("任务已完成")
	}
	if !time.Now().Before(task.ExpiresAt) {
		return s.finishTask(task, system.RegisterTaskFailCodeTimeout, "任务超时自动完成")
	}
	switch req.Action {
	case "", "submit", "retry", "fail":
	default:
		return task, errors.New("不支持的action")
	}
	if req.Action == "" {
		req.Action = "submit"
	}
	if req.Action == "submit" && req.Step != task.CurrentStep {
		return task, errors.New("步骤状态不一致，请刷新页面后重试")
	}
	if req.Action == "submit" && strings.TrimSpace(req.VerifyCode) != "" && !isVerifyCodeReadyForCurrentStep(task) {
		return task, errors.New("验证码尚未发送成功，请先点击重试当前步骤")
	}
	if req.Action == "submit" && task.CurrentStep == system.RegisterTaskStepPhoneBind && strings.TrimSpace(req.VerifyCode) == "" {
		return task, errors.New("验证码不能为空")
	}
	ensureRegisterTaskRunner(task.ID, promoterID)

	// 登录阶段验证码提交为“直投通道”，避免重复触发登录初始化。
	if req.Action == "submit" && task.CurrentStep == system.RegisterTaskStepLogin && strings.TrimSpace(req.VerifyCode) != "" {
		currentQQ := ""
		changedQQList := splitQQList(task.QQChangedList)
		loggedQQList := splitQQList(task.QQLoggedList)
		if len(loggedQQList) < len(changedQQList) {
			currentQQ = changedQQList[len(loggedQQList)]
		}
		delivered, deliverErr := offerLoginVerifyCode(task.ID, currentQQ, req.VerifyCode)
		if deliverErr != nil {
			global.GVA_LOG.Warn("【注册任务】提交步骤-登录验证码投递失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(deliverErr))...)
			return task, deliverErr
		}
		if delivered {
			task.LastError = ""
			if err = global.GVA_DB.Save(&task).Error; err != nil {
				return task, err
			}
			global.GVA_LOG.Info("【注册任务】提交步骤-登录验证码已投递", taskLogFieldsWithOpQQ(task, currentQQ)...)
			return task, nil
		}
		return task, errors.New("登录会话已失效，请点击重试重新获取验证码")
	}

	if err = enqueueRegisterTaskEvent(task.ID, promoterID, registerTaskEvent{
		Action:      req.Action,
		Step:        req.Step,
		VerifyCode:  req.VerifyCode,
		FailMessage: req.FailMessage,
	}); err != nil {
		global.GVA_LOG.Error("【注册任务】提交步骤-事件投递失败", append(taskLogFields(task), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("【注册任务】提交步骤-事件已投递",
		append(taskLogFields(task), zap.String("action", req.Action), zap.String("reqStep", req.Step))...,
	)
	var latest system.SysRegisterTask
	if qErr := global.GVA_DB.Where("id = ? AND promoter_id = ?", task.ID, promoterID).First(&latest).Error; qErr == nil {
		return latest, nil
	}
	return task, nil
}

func isVerifyCodeReadyForCurrentStep(task system.SysRegisterTask) bool {
	lastError := strings.TrimSpace(task.LastError)
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		return hasPhoneBindSession(task.ID) || hasPhoneBindSMSSent(task)
	case system.RegisterTaskStepChangePassword:
		if strings.Contains(lastError, "改密验证码已发送") {
			return true
		}
		ps, ok := getChangePasswordSession(task.ID)
		return ok && ps != nil && ps.ChangePwdClient != nil
	case system.RegisterTaskStepLogin:
		if strings.Contains(lastError, registerTaskLoginNeedVerifyTip) || strings.Contains(lastError, "触发短信验证") {
			return true
		}
		ls, ok := getLoginSession(task.ID)
		return ok && ls != nil && ls.LoginCodeCh != nil
	default:
		return true
	}
}

func (s *RegisterTaskService) handleSubmit(task system.SysRegisterTask, req systemReq.RegisterTaskStep) (system.SysRegisterTask, error) {
	if req.Step != task.CurrentStep {
		return task, errors.New("步骤状态不一致，请刷新页面后重试")
	}
	if task.CurrentStep == system.RegisterTaskStepPhoneBind && strings.TrimSpace(req.VerifyCode) == "" {
		return task, errors.New("验证码不能为空")
	}

	now := time.Now()
	global.GVA_LOG.Info("【注册任务】执行提交-进入主流程", append(taskLogFields(task), zap.String("reqStep", req.Step))...)
	runtimeCfg, err := s.getRegisterRuntimeConfig(task.LeaderID)
	if err != nil {
		return task, err
	}

	proxyURL, err := s.allocateProxyURL(runtimeCfg, task.Phone)
	if err != nil {
		global.GVA_LOG.Error("【注册任务】分配代理失败", append(taskLogFields(task), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("【注册任务】运行时配置就绪",
		append(taskLogFields(task),
			zap.String("proxyPlatform", runtimeCfg.ProxyPlatform),
			zap.String("captchaPlatform", runtimeCfg.CaptchaPlatform),
			zap.Bool("proxyEnabled", strings.TrimSpace(proxyURL) != ""),
		)...,
	)

	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		global.GVA_LOG.Info("【注册任务】手机绑定-开始", taskLogFields(task)...)
		phoneBindClient, ok := popPhoneBindSession(task.ID)
		if !ok || phoneBindClient == nil {
			if recoverErr := s.restoreTaskProgressIfNeeded(&task); recoverErr != nil {
				return task, recoverErr
			}
			global.GVA_LOG.Warn("【注册任务】手机绑定-内存会话缺失", taskLogFields(task)...)
			return task, errors.New("验证码已发送，请先使用已收到验证码提交；若验证码失效请点击重试后再提交")
		}
		defer phoneBindClient.Close()
		global.GVA_LOG.Info("【注册任务】手机绑定-会话就绪", append(taskLogFields(task), zap.Any("code", req.VerifyCode))...)
		if err := phoneBindClient.SubmitSMS(qpi.SubmitSMSRequest{SMSCode: req.VerifyCode}); err != nil {
			global.GVA_LOG.Error("【注册任务】手机绑定-提交短信验证码失败", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		global.GVA_LOG.Info("【注册任务】手机绑定-短信验证通过", taskLogFields(task)...)
		global.GVA_LOG.Info("【注册任务】手机绑定-开始查绑QQ", taskLogFields(task)...)
		loginResp, err := phoneBindClient.LoginAccount(qpi.LoginAccountRequest{SkipList: ""})
		if err != nil {
			global.GVA_LOG.Error("【注册任务】手机绑定-查绑登录失败", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		if len(loginResp.QQList) == 0 {
			return s.finishTask(task, system.RegisterTaskFailCodeNoQQBound, "手机号无可用QQ账号")
		}
		global.GVA_LOG.Info("【注册任务】手机绑定-查绑QQ列表", append(taskLogFields(task), zap.Strings("qqList", loginResp.QQList))...)
		filteredQQList, err := s.filterQualifiedQQByNaicha(runtimeCfg, &task, loginResp.QQList)
		if err != nil {
			global.GVA_LOG.Error("【注册任务】手机绑定-奶茶筛选失败", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		global.GVA_LOG.Info("【注册任务】手机绑定-奶茶筛选完成",
			append(taskLogFields(task), zap.Int("rawQQCount", len(loginResp.QQList)), zap.Int("qualifiedCount", len(filteredQQList)))...,
		)
		if len(filteredQQList) == 0 {
			return s.finishTask(task, system.RegisterTaskFailCodeNoQQBound, "奶茶筛选后无符合条件QQ")
		}
		task.QQCandidates = strings.Join(filteredQQList, "|")
		task.CurrentStep = system.RegisterTaskStepChangePassword
		task.LastError = ""
		if err = global.GVA_DB.Save(&task).Error; err != nil {
			global.GVA_LOG.Error("【注册任务】手机绑定-切换改密步骤保存失败", append(taskLogFields(task), zap.Error(err))...)
			return task, err
		}
		global.GVA_LOG.Info("【注册任务】手机绑定后自动进入改密-开始预发验证码", taskLogFields(task)...)
		if prepErr := s.prepareChangePasswordSMSIfNeeded(&task, runtimeCfg, proxyURL, true); prepErr != nil {
			global.GVA_LOG.Error("【注册任务】手机绑定后自动进入改密-预发验证码失败", append(taskLogFields(task), zap.Error(prepErr))...)
			return task, prepErr
		}
		global.GVA_LOG.Info("【注册任务】手机绑定后自动进入改密-预发验证码成功", taskLogFields(task)...)
		global.GVA_LOG.Info("【注册任务】手机绑定-完成，进入改密",
			append(taskLogFields(task),
				zap.Strings("qualifiedQQList", filteredQQList),
			)...,
		)
	case system.RegisterTaskStepChangePassword:
		candidateQQList := splitQQList(task.QQCandidates)
		if len(candidateQQList) == 0 {
			return task, errors.New("请先完成QQ查绑及筛选步骤")
		}
		global.GVA_LOG.Info("【注册任务】改密-开始",
			append(taskLogFields(task), zap.Int("candidateCount", len(candidateQQList)))...,
		)
		changePwd := strings.TrimSpace(runtimeCfg.DefaultPassword)
		if changePwd == "" {
			global.GVA_LOG.Warn("【注册任务】改密-未配置默认密码", taskLogFields(task)...)
			return task, errors.New("管理员未配置默认改密密码")
		}
		changedQQList := splitQQList(task.QQChangedList)
		if len(changedQQList) > len(candidateQQList) {
			changedQQList = changedQQList[:len(candidateQQList)]
		}
		if len(changedQQList) >= len(candidateQQList) {
			clearChangePasswordSession(task.ID)
			global.GVA_LOG.Info("【注册任务】改密-已全部完成，进入登录",
				append(taskLogFields(task), zap.Strings("changedQQList", changedQQList))...,
			)
			task.CurrentStep = system.RegisterTaskStepLogin
			task.LastError = ""
			if err = global.GVA_DB.Save(&task).Error; err != nil {
				global.GVA_LOG.Error("【注册任务】改密-切换登录步骤保存失败", append(taskLogFields(task), zap.Error(err))...)
				return task, err
			}
			enqueueContinueLoginAfterChangePassword(task)
			break
		}
		currentQQ := candidateQQList[len(changedQQList)]
		global.GVA_LOG.Info("【注册任务】改密-处理当前QQ", taskLogFieldsWithOpQQ(task, currentQQ)...)
		verifyCode := strings.TrimSpace(req.VerifyCode)
		ps, ok := getChangePasswordSession(task.ID)
		validSession := ok && ps != nil && ps.ChangePwdClient != nil && ps.ChangePwdQQ == currentQQ
		if !validSession {
			if strings.Contains(strings.TrimSpace(task.LastError), "改密验证码已发送") {
				return task, errors.New("改密会话已失效，请点击重试重新发送验证码")
			}
			if prepErr := s.prepareChangePasswordSMSIfNeeded(&task, runtimeCfg, proxyURL, false); prepErr != nil {
				global.GVA_LOG.Error("【注册任务】改密-预发验证码失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(prepErr))...)
				return task, prepErr
			}
			return task, errors.New("改密验证码已发送，请输入改密验证码后提交")
		}
		if verifyCode == "" {
			return task, errors.New("改密验证码已发送，请输入改密验证码后提交")
		}
		if err = s.completeChangePasswordForQQ(ps, currentQQ, task.Phone, verifyCode, changePwd); err != nil {
			global.GVA_LOG.Error("【注册任务】改密-当前QQ失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(err))...)
			return task, err
		}
		clearChangePasswordSession(task.ID)
		changedQQList = append(changedQQList, currentQQ)
		global.GVA_LOG.Info("【注册任务】改密-当前QQ成功", taskLogFieldsWithOpQQ(task, currentQQ)...)
		task.QQPassword = changePwd
		task.QQChangedList = strings.Join(changedQQList, "|")
		if len(changedQQList) >= len(candidateQQList) {
			clearChangePasswordSession(task.ID)
			task.ChangePasswordAt = &now
			task.CurrentStep = system.RegisterTaskStepLogin
			task.LastError = ""
			global.GVA_LOG.Info("【注册任务】改密-全部完成，进入登录", append(taskLogFields(task), zap.Strings("changedQQList", changedQQList))...)
			if err = global.GVA_DB.Save(&task).Error; err != nil {
				global.GVA_LOG.Error("【注册任务】改密-切换登录步骤保存失败", append(taskLogFields(task), zap.Error(err))...)
				return task, err
			}
			enqueueContinueLoginAfterChangePassword(task)
		} else {
			task.LastError = ""
			if err = global.GVA_DB.Save(&task).Error; err != nil {
				global.GVA_LOG.Error("【注册任务】改密-保存进度失败", append(taskLogFields(task), zap.Error(err))...)
				return task, err
			}
			if prepErr := s.prepareChangePasswordSMSIfNeeded(&task, runtimeCfg, proxyURL, false); prepErr != nil {
				global.GVA_LOG.Error("【注册任务】改密-切换下一个QQ预发验证码失败", append(taskLogFields(task), zap.Error(prepErr))...)
				return task, prepErr
			}
			global.GVA_LOG.Info("【注册任务】改密-进度", append(taskLogFields(task), zap.Int("done", len(changedQQList)), zap.Int("total", len(candidateQQList)))...)
		}
	case system.RegisterTaskStepLogin:
		if strings.TrimSpace(task.QQPassword) == "" {
			return task, errors.New("QQ账号或密码为空，请先完成前置步骤")
		}
		changedQQList := splitQQList(task.QQChangedList)
		if len(changedQQList) == 0 {
			return task, errors.New("无可登录QQ账号，请先完成改密")
		}
		verifyCode := strings.TrimSpace(req.VerifyCode)
		loggedQQList := splitQQList(task.QQLoggedList)
		if len(loggedQQList) > len(changedQQList) {
			loggedQQList = loggedQQList[:len(changedQQList)]
		}
		global.GVA_LOG.Info("【注册任务】登录-开始",
			append(taskLogFields(task),
				zap.Int("toLoginCount", len(changedQQList)),
				zap.Int("alreadyLoggedCount", len(loggedQQList)),
				zap.Bool("verifyProvided", verifyCode != ""),
			)...,
		)
		if len(loggedQQList) >= len(changedQQList) {
			isDaren := true
			task.IsDaren = &isDaren
			task.LoginAt = &now
			global.GVA_LOG.Info("【注册任务】登录-全部完成", append(taskLogFields(task), zap.Strings("loginSuccessQQ", loggedQQList), zap.Bool("isDaren", isDaren))...)
			return s.finishTask(task, 0, "")
		}
		currentQQ := changedQQList[len(loggedQQList)]
		global.GVA_LOG.Info("【注册任务】登录-定位当前QQ",
			append(taskLogFieldsWithOpQQ(task, currentQQ),
				zap.Int("currentIndex", len(loggedQQList)+1),
				zap.Int("totalCount", len(changedQQList)),
			)...,
		)

		const needVerifyCodeTip = registerTaskLoginNeedVerifyTip
		const waitTimeoutTip = "登录验证码等待超时，请点击重试重新获取验证码"
		const sessionExpiredTip = "登录会话已失效，请点击重试重新获取验证码"
		if strings.TrimSpace(verifyCode) != "" {
			global.GVA_LOG.Info("【注册任务】登录-提交验证码",
				append(taskLogFieldsWithOpQQ(task, currentQQ),
					zap.Int("codeLength", len(verifyCode)),
				)...,
			)
			delivered, deliverErr := offerLoginVerifyCode(task.ID, currentQQ, verifyCode)
			if deliverErr != nil {
				global.GVA_LOG.Warn("【注册任务】登录-投递验证码失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(deliverErr))...)
				return task, deliverErr
			}
			if delivered {
				task.LastError = ""
				_ = global.GVA_DB.Save(&task).Error
				global.GVA_LOG.Info("【注册任务】登录-验证码已投递到等待通道", taskLogFieldsWithOpQQ(task, currentQQ)...)
				return task, nil
			}
			global.GVA_LOG.Warn("【注册任务】登录-验证码提交但无会话", taskLogFieldsWithOpQQ(task, currentQQ)...)
			return task, errors.New(sessionExpiredTip)
		}
		var loginResp *qpi.PasswordLoginResponse
		loginSession, hasLoginSession := getLoginSession(task.ID)
		if hasLoginSession && loginSession != nil && loginSession.LoginQQ == currentQQ && loginSession.LoginCodeCh != nil {
			task.LastError = needVerifyCodeTip
			_ = global.GVA_DB.Save(&task).Error
			global.GVA_LOG.Info("【注册任务】登录-当前QQ正在等待验证码",
				append(taskLogFieldsWithOpQQ(task, currentQQ),
					zap.Bool("hasLoginSession", true),
				)...,
			)
			return task, errors.New(needVerifyCodeTip)
		}
		// 严格状态保护：进入登录待验证码态后，除 retry 外不允许重复触发登录发码。
		if strings.Contains(strings.TrimSpace(task.LastError), registerTaskLoginNeedVerifyTip) {
			global.GVA_LOG.Warn("【注册任务】登录-状态守卫阻止重复发码",
				append(taskLogFieldsWithOpQQ(task, currentQQ),
					zap.String("lastError", task.LastError),
					zap.Bool("hasLoginSession", hasLoginSession),
				)...,
			)
			return task, errors.New(sessionExpiredTip)
		} else {
			global.GVA_LOG.Info("【注册任务】登录-初始化本轮会话", taskLogFieldsWithOpQQ(task, currentQQ)...)
			// 清理非当前QQ的旧会话，避免串号
			clearLoginSession(task.ID)
			tlv544Provider, pErr := buildTLV544ProviderFromConfig(runtimeCfg, task)
			if pErr != nil {
				global.GVA_LOG.Error("【注册任务】登录-TLV544初始化失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(pErr))...)
				return task, pErr
			}
			tlv553Provider, pErr := buildTLV553ProviderFromConfig(runtimeCfg, task)
			if pErr != nil {
				global.GVA_LOG.Error("【注册任务】登录-TLV553初始化失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(pErr))...)
				return task, pErr
			}
			signProvider, pErr := buildSignProviderFromConfig(runtimeCfg, task)
			if pErr != nil {
				global.GVA_LOG.Error("【注册任务】登录-SignProvider初始化失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(pErr))...)
				return task, pErr
			}
			initProvider, pErr := buildInitProviderFromConfig(runtimeCfg, task)
			if pErr != nil {
				global.GVA_LOG.Error("【注册任务】登录-InitProvider初始化失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(pErr))...)
				return task, pErr
			}

			clearLoginSession(task.ID)
			loginClient := qpi.NewPasswordLoginClient()
			if proxyURL != "" {
				if proxyErr := loginClient.SetProxy(proxyURL); proxyErr != nil {
					_ = loginClient.Close()
					global.GVA_LOG.Error("【注册任务】登录-设置代理失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(proxyErr))...)
					return task, proxyErr
				}
			}
			loginCodeCh := make(chan string, 1)
			androidID, guidHex, qimei16 := buildStableLoginDeviceProfile(currentQQ)
			loginReq := qpi.PasswordLoginRequest{
				UIN:            currentQQ,
				Password:       task.QQPassword,
				GUIDHex:        guidHex,
				AndroidID:      androidID,
				QIMEI16:        qimei16,
				TLV544Provider: tlv544Provider,
				TLV553Provider: tlv553Provider,
				SignProvider:   signProvider,
				InitProvider:   initProvider,
				CaptchaProvider: func(captchaURL string) (string, error) {
					aid, sid := parseCaptchaAidSid(captchaURL)
					global.GVA_LOG.Info("【注册任务】登录-滑块验证码参数", append(taskLogFieldsWithOpQQ(task, currentQQ),
						zap.String("captchaURL", captchaURL),
						zap.String("aid", aid),
						zap.String("sid", sid),
					)...)
					if strings.TrimSpace(aid) == "" {
						err := fmt.Errorf("登录滑块参数缺失: aid 为空, captchaURL=%s", strings.TrimSpace(captchaURL))
						global.GVA_LOG.Error("【注册任务】登录-滑块验证码失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(err))...)
						return "", err
					}
					cap, err := s.getCaptchaToken(runtimeCfg, aid, sid)
					if err != nil {
						global.GVA_LOG.Error("【注册任务】登录-滑块验证码失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(err))...)
						return "", err
					}
					global.GVA_LOG.Info("【注册任务】登录-滑块验证码获取成功", append(taskLogFieldsWithOpQQ(task, currentQQ),
						zap.String("randstr", cap.Randstr),
						zap.String("ticket", cap.Ticket),
					)...)
					return cap.Ticket, nil
				},
				SMSCodeProvider: func() (string, error) {
					task.LastError = needVerifyCodeTip
					_ = global.GVA_DB.Save(&task).Error
					global.GVA_LOG.Info("【注册任务】登录-等待验证码输入", taskLogFieldsWithOpQQ(task, currentQQ)...)
					codeCh := loginCodeCh
					if codeCh == nil {
						return "", errors.New(sessionExpiredTip)
					}
					select {
					case code, ok := <-codeCh:
						if !ok {
							return "", errors.New(sessionExpiredTip)
						}
						code = strings.TrimSpace(code)
						if code == "" {
							return "", errors.New(needVerifyCodeTip)
						}
						return code, nil
					case <-time.After(registerTaskLoginVerifyWaitTimeout):
						return "", errors.New(waitTimeoutTip)
					}
				},
			}
			setLoginSession(task.ID, loginClient, currentQQ, loginCodeCh)
			global.GVA_LOG.Info("【注册任务】登录-会话创建完成，开始登录协议", taskLogFieldsWithOpQQ(task, currentQQ)...)
			var loginErr error
			loginResp, loginErr = loginClient.Login(loginReq)
			if loginErr != nil {
				if strings.Contains(loginErr.Error(), waitTimeoutTip) {
					task.LastError = waitTimeoutTip
					_ = global.GVA_DB.Save(&task).Error
					clearLoginSession(task.ID)
					global.GVA_LOG.Warn("【注册任务】登录-等待验证码超时", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(loginErr))...)
					return task, errors.New(waitTimeoutTip)
				}
				if strings.Contains(loginErr.Error(), "短信验证码") || strings.Contains(loginErr.Error(), "触发短信验证") || strings.Contains(loginErr.Error(), needVerifyCodeTip) {
					task.LastError = needVerifyCodeTip
					_ = global.GVA_DB.Save(&task).Error
					global.GVA_LOG.Warn("【注册任务】登录-协议触发短信验证，进入待码态", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(loginErr))...)
					return task, errors.New(needVerifyCodeTip)
				}
				if isRetryableLoginNetworkErr(loginErr) {
					// 首次失败后，自动换代理重试（总共最多尝试3次：首次+2次换代理）。
					const maxProxyRetryAttempts = 2
					lastErr := loginErr
					lastProxyURL := proxyURL
					clearLoginSession(task.ID)
					for attempt := 1; attempt <= maxProxyRetryAttempts; attempt++ {
						refreshedProxyURL, pErr := s.allocateProxyURL(runtimeCfg, task.Phone)
						if pErr != nil {
							global.GVA_LOG.Error("【注册任务】登录-刷新代理失败", append(taskLogFieldsWithOpQQ(task, currentQQ),
								zap.Int("attempt", attempt),
								zap.Int("maxAttempts", maxProxyRetryAttempts),
								zap.Error(pErr),
							)...)
							return task, pErr
						}
						global.GVA_LOG.Warn("【注册任务】登录-网络异常，刷新代理后重试",
							append(taskLogFieldsWithOpQQ(task, currentQQ),
								zap.Int("attempt", attempt),
								zap.Int("maxAttempts", maxProxyRetryAttempts),
								zap.String("oldProxyURL", lastProxyURL),
								zap.String("newProxyURL", refreshedProxyURL),
								zap.Error(lastErr),
							)...,
						)
						retryClient := qpi.NewPasswordLoginClient()
						if refreshedProxyURL != "" {
							if proxyErr := retryClient.SetProxy(refreshedProxyURL); proxyErr != nil {
								_ = retryClient.Close()
								return task, proxyErr
							}
						}
						retryCodeCh := make(chan string, 1)
						retryReq := loginReq
						retryReq.SMSCodeProvider = func() (string, error) {
							task.LastError = needVerifyCodeTip
							_ = global.GVA_DB.Save(&task).Error
							global.GVA_LOG.Info("【注册任务】登录-重试后等待验证码输入", taskLogFieldsWithOpQQ(task, currentQQ)...)
							select {
							case code, ok := <-retryCodeCh:
								if !ok {
									return "", errors.New(sessionExpiredTip)
								}
								code = strings.TrimSpace(code)
								if code == "" {
									return "", errors.New(needVerifyCodeTip)
								}
								return code, nil
							case <-time.After(registerTaskLoginVerifyWaitTimeout):
								return "", errors.New(waitTimeoutTip)
							}
						}
						setLoginSession(task.ID, retryClient, currentQQ, retryCodeCh)
						loginResp, lastErr = retryClient.Login(retryReq)
						if lastErr == nil {
							clearLoginSession(task.ID)
							global.GVA_LOG.Info("【注册任务】登录-刷新代理后重试成功",
								append(taskLogFieldsWithOpQQ(task, currentQQ),
									zap.Int("attempt", attempt),
									zap.Int("maxAttempts", maxProxyRetryAttempts),
								)...,
							)
							goto loginSuccess
						}
						if strings.Contains(lastErr.Error(), waitTimeoutTip) {
							task.LastError = waitTimeoutTip
							_ = global.GVA_DB.Save(&task).Error
							clearLoginSession(task.ID)
							return task, errors.New(waitTimeoutTip)
						}
						if strings.Contains(lastErr.Error(), "短信验证码") || strings.Contains(lastErr.Error(), "触发短信验证") || strings.Contains(lastErr.Error(), needVerifyCodeTip) {
							task.LastError = needVerifyCodeTip
							_ = global.GVA_DB.Save(&task).Error
							return task, errors.New(needVerifyCodeTip)
						}
						clearLoginSession(task.ID)
						lastProxyURL = refreshedProxyURL
						if !isRetryableLoginNetworkErr(lastErr) {
							global.GVA_LOG.Error("【注册任务】登录-刷新代理后重试失败(非可重试错误)", append(taskLogFieldsWithOpQQ(task, currentQQ),
								zap.Int("attempt", attempt),
								zap.Int("maxAttempts", maxProxyRetryAttempts),
								zap.Error(lastErr),
							)...)
							return task, lastErr
						}
					}
					global.GVA_LOG.Error("【注册任务】登录-刷新代理重试已达上限仍失败", append(taskLogFieldsWithOpQQ(task, currentQQ),
						zap.Int("maxAttempts", maxProxyRetryAttempts),
						zap.Error(lastErr),
					)...)
					return task, lastErr
				}
				clearLoginSession(task.ID)
				global.GVA_LOG.Error("【注册任务】登录-当前QQ失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(loginErr))...)
				return task, loginErr
			}
			clearLoginSession(task.ID)
		}
	loginSuccess:
		global.GVA_LOG.Info("【注册任务】登录-当前QQ成功", taskLogFieldsWithOpQQ(task, currentQQ)...)
		loggedQQList = append(loggedQQList, currentQQ)
		task.QQLoggedList = strings.Join(loggedQQList, "|")
		if task.LoginCacheINI == "" {
			task.LoginCacheINI = buildCacheINI(loginResp.Cache.ToHexMap())
		} else {
			task.LoginCacheINI = task.LoginCacheINI + "\n" + buildCacheINI(loginResp.Cache.ToHexMap())
		}
		// 单个账号登录成功后立即落库，避免后续步骤失败导致进度丢失
		if err := global.GVA_DB.Save(&task).Error; err != nil {
			global.GVA_LOG.Error("【注册任务】登录-单账号成功后保存失败", append(taskLogFieldsWithOpQQ(task, currentQQ), zap.Error(err))...)
			return task, err
		}
		if len(loggedQQList) >= len(changedQQList) {
			isDaren := true
			task.IsDaren = &isDaren
			task.LoginAt = &now
			global.GVA_LOG.Info("【注册任务】登录-全部完成", append(taskLogFields(task), zap.Strings("loginSuccessQQ", loggedQQList), zap.Bool("isDaren", isDaren))...)
			return s.finishTask(task, 0, "")
		}
		task.LastError = ""
		global.GVA_LOG.Info("【注册任务】登录-进度", append(taskLogFields(task), zap.Int("done", len(loggedQQList)), zap.Int("total", len(changedQQList)))...)
	default:
		return task, errors.New("未知任务步骤")
	}

	if err := global.GVA_DB.Save(&task).Error; err != nil {
		global.GVA_LOG.Error("【注册任务】保存步骤状态失败", append(taskLogFields(task), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("【注册任务】步骤状态已保存", taskLogFields(task)...)
	return task, nil
}

func (s *RegisterTaskService) finishTask(task system.SysRegisterTask, code int, msg string) (system.SysRegisterTask, error) {
	now := time.Now()
	task.StatusCode = &code
	task.LastError = msg
	task.FinishedAt = &now
	if err := global.GVA_DB.Save(&task).Error; err != nil {
		global.GVA_LOG.Error("【注册任务】结束任务时保存失败", append(taskLogFields(task), zap.Int("statusCode", code), zap.Error(err))...)
		return task, err
	}
	global.GVA_LOG.Info("【注册任务】任务已结束",
		append(taskLogFields(task),
			zap.Int("statusCode", code),
			zap.String("message", msg),
		)...,
	)
	clearTaskSession(task.ID)
	return task, nil
}

func (s *RegisterTaskService) timeoutTasksByPromoter(promoterID uint) error {
	now := time.Now()
	failCode := system.RegisterTaskFailCodeTimeout
	err := global.GVA_DB.Model(&system.SysRegisterTask{}).
		Where("promoter_id = ? AND finished_at IS NULL AND expires_at <= ?", promoterID, now).
		Updates(map[string]interface{}{
			"status_code": failCode,
			"last_error":  "任务超时自动完成",
			"finished_at": now,
		}).Error
	if err != nil {
		global.GVA_LOG.Error("【注册任务】按地推批量超时失败", zap.Uint("promoterId", promoterID), zap.Error(err))
	}
	return err
}

func applyRegisterTaskRoleFilter(db *gorm.DB, operatorRole uint, operatorID uint, req systemReq.RegisterTaskList) (*gorm.DB, error) {
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
		return nil, errors.New("无权限查看任务")
	}
	return db, nil
}

func successLoggedQQCountSQL(column string) string {
	c := strings.TrimSpace(column)
	if c == "" {
		c = "qq_logged_list"
	}
	// 统计 pipe 分隔账号数：a|b|c => 3；空字符串/NULL => 0
	return fmt.Sprintf(`
		CASE
			WHEN %s IS NULL OR TRIM(%s) = '' THEN 0
			ELSE 1 + LENGTH(%s) - LENGTH(REPLACE(%s, '|', ''))
		END
	`, c, c, c, c)
}

func parseTaskListTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func applyRegisterTaskQueryFilters(db *gorm.DB, req systemReq.RegisterTaskList) *gorm.DB {
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

	switch strings.ToLower(strings.TrimSpace(req.Status)) {
	case "success":
		db = db.Where("finished_at IS NOT NULL AND status_code = 0")
	case "fail", "failed":
		db = db.Where("finished_at IS NOT NULL AND (status_code <> 0 OR status_code IS NULL)")
	}
	if req.Exported != nil {
		if *req.Exported {
			db = db.Where("exported_at IS NOT NULL")
		} else {
			db = db.Where("exported_at IS NULL")
		}
	}

	phone := strings.TrimSpace(req.Phone)
	if phone != "" {
		db = db.Where("phone LIKE ?", "%"+phone+"%")
	}

	if startAt, ok := parseTaskListTime(req.FinishedAtStart); ok {
		db = db.Where("finished_at >= ?", startAt)
	}
	if endAt, ok := parseTaskListTime(req.FinishedAtEnd); ok {
		db = db.Where("finished_at <= ?", endAt)
	}
	return db
}

func (s *RegisterTaskService) GetTaskList(operatorRole uint, operatorID uint, req systemReq.RegisterTaskList) (registerTaskListResult, error) {
	_ = s.normalizeTimeoutClosedTasks()
	_ = s.timeoutUnfinishedTasks()
	db := global.GVA_DB.Model(&system.SysRegisterTask{}).Preload("Promoter").Preload("Leader")
	var roleErr error
	db, roleErr = applyRegisterTaskRoleFilter(db, operatorRole, operatorID, req)
	if roleErr != nil {
		global.GVA_LOG.Warn("【注册任务】任务列表-无权限", zap.Uint("operatorRole", operatorRole), zap.Uint("operatorId", operatorID))
		return registerTaskListResult{}, roleErr
	}
	db = applyRegisterTaskQueryFilters(db, req)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		global.GVA_LOG.Error("【注册任务】任务列表-统计总数失败", zap.Uint("operatorRole", operatorRole), zap.Uint("operatorId", operatorID), zap.Error(err))
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
		global.GVA_LOG.Error("【注册任务】任务列表-查询失败", zap.Uint("operatorRole", operatorRole), zap.Uint("operatorId", operatorID), zap.Error(err))
		return registerTaskListResult{}, err
	}

	statDB := global.GVA_DB.Model(&system.SysRegisterTask{})
	statDB, roleErr = applyRegisterTaskRoleFilter(statDB, operatorRole, operatorID, req)
	if roleErr != nil {
		global.GVA_LOG.Warn("【注册任务】任务列表-无权限", zap.Uint("operatorRole", operatorRole), zap.Uint("operatorId", operatorID))
		return registerTaskListResult{}, roleErr
	}
	statDB = applyRegisterTaskQueryFilters(statDB, req)
	type taskCounter struct {
		Success    int64 `gorm:"column:success"`
		Failed     int64 `gorm:"column:failed"`
		Processing int64 `gorm:"column:processing"`
	}
	var counter taskCounter
	successQQCountExpr := successLoggedQQCountSQL("qq_logged_list")
	if err := statDB.
		Select(fmt.Sprintf(`
			COALESCE(SUM(CASE WHEN finished_at IS NOT NULL AND status_code = 0 THEN %s ELSE 0 END), 0) AS success,
			COALESCE(SUM(CASE WHEN finished_at IS NOT NULL AND (status_code <> 0 OR status_code IS NULL) THEN 1 ELSE 0 END), 0) AS failed,
			COALESCE(SUM(CASE WHEN finished_at IS NULL THEN 1 ELSE 0 END), 0) AS processing`, successQQCountExpr)).
		Scan(&counter).Error; err != nil {
		global.GVA_LOG.Error("【注册任务】任务列表-汇总统计失败", zap.Uint("operatorRole", operatorRole), zap.Uint("operatorId", operatorID), zap.Error(err))
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
		global.GVA_LOG.Warn("【注册任务】任务统计-无权限", zap.Uint("operatorRole", operatorRole), zap.Uint("operatorId", operatorID))
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

	successQQCountExpr := successLoggedQQCountSQL("t.qq_logged_list")
	db := global.GVA_DB.Table("sys_register_tasks t").
		Select(fmt.Sprintf(`
			t.leader_id,
			leader.nick_name AS leader_name,
			t.promoter_id,
			promoter.nick_name AS promoter_name,
			SUM(CASE WHEN t.finished_at IS NOT NULL AND t.status_code = 0 THEN %s ELSE 0 END) AS success_count,
			SUM(CASE WHEN t.finished_at IS NOT NULL AND (t.status_code <> 0 OR t.status_code IS NULL) THEN 1 ELSE 0 END) AS fail_count,
			SUM(CASE WHEN t.finished_at IS NULL THEN 1 ELSE 0 END) AS processing_count`, successQQCountExpr)).
		Joins("LEFT JOIN sys_users promoter ON promoter.id = t.promoter_id").
		Joins("LEFT JOIN sys_users leader ON leader.id = t.leader_id")

	if operatorRole == roleLeader {
		db = db.Where("t.leader_id = ?", operatorID)
	} else if leaderID != 0 {
		db = db.Where("t.leader_id = ?", leaderID)
	}

	var promoterRows []summaryRow
	if err := db.Group("t.leader_id, leader.nick_name, t.promoter_id, promoter.nick_name").Scan(&promoterRows).Error; err != nil {
		global.GVA_LOG.Error("【注册任务】任务统计-查询失败", zap.Uint("operatorRole", operatorRole), zap.Uint("operatorId", operatorID), zap.Uint("leaderId", leaderID), zap.Error(err))
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
	err := global.GVA_DB.Model(&system.SysRegisterTask{}).
		Where("finished_at IS NULL AND expires_at <= ?", now).
		Updates(map[string]interface{}{
			"status_code": failCode,
			"last_error":  "任务超时自动完成",
			"finished_at": now,
		}).Error
	if err != nil {
		global.GVA_LOG.Error("【注册任务】全局批量超时未完成失败", zap.Error(err))
	}
	return err
}

// normalizeTimeoutClosedTasks 兜底修正：已完成且超时，但状态码为空的数据统一标记为超时失败
func (s *RegisterTaskService) normalizeTimeoutClosedTasks() error {
	now := time.Now()
	failCode := system.RegisterTaskFailCodeTimeout
	err := global.GVA_DB.Model(&system.SysRegisterTask{}).
		Where("finished_at IS NOT NULL AND status_code IS NULL AND expires_at <= ?", now).
		Updates(map[string]interface{}{
			"status_code": failCode,
			"last_error":  "任务超时自动完成",
		}).Error
	if err != nil {
		global.GVA_LOG.Error("【注册任务】修正已超时但状态码为空失败", zap.Error(err))
	}
	return err
}

func (s *RegisterTaskService) filterQualifiedQQByNaicha(cfg systemRegisterConfig, task *system.SysRegisterTask, qqList []string) ([]string, error) {

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
	global.GVA_LOG.Info("【注册任务】奶茶筛选-DR结果", append(taskLogFields(*task), zap.Any("ret", drResults))...)
	global.GVA_LOG.Info("【注册任务】奶茶筛选-QL结果", append(taskLogFields(*task), zap.Any("ret", qlResults))...)

	bypassFilter := task != nil && shouldBypassNaichaFilterForPhone(task.Phone)
	if bypassFilter {
		global.GVA_LOG.Warn("【注册任务】奶茶筛选-测试旁路生效，跳过条件过滤",
			append(taskLogFields(*task), zap.Strings("qqList", qqList))...,
		)
	}

	drMap := make(map[string]NCDRResult)
	for _, item := range drResults {
		drMap[item.UIN] = item
	}
	qlMap := make(map[string]NCQLResult)
	for _, item := range qlResults {
		qlMap[item.UIN] = item
	}

	type naichaQQInfo struct {
		QQ             string   `json:"qq"`
		PhoneOnlineDay *int     `json:"phoneOnlineDay,omitempty"`
		QAge           *int     `json:"qAge,omitempty"`
		QLevel         *int     `json:"qLevel,omitempty"`
		Qualified      bool     `json:"qualified"`
		Reasons        []string `json:"reasons,omitempty"`
	}

	infoList := make([]naichaQQInfo, 0, len(qqList))
	filtered := make([]string, 0, len(qqList))
	for _, qq := range qqList {
		dr, ok1 := drMap[qq]
		ql, ok2 := qlMap[qq]
		if !ok1 || !ok2 {
			reasons := make([]string, 0, 2)
			if !ok1 {
				reasons = append(reasons, "缺少达人查询结果")
			}
			if !ok2 {
				reasons = append(reasons, "缺少Q龄查询结果")
			}
			infoList = append(infoList, naichaQQInfo{
				QQ:        qq,
				Qualified: false,
				Reasons:   reasons,
			})
			global.GVA_LOG.Warn("【注册任务】奶茶筛选-跳过QQ", append(
				taskLogFieldsWithOpQQ(*task, qq),
				zap.Strings("reasons", reasons),
			)...)
			continue
		}
		// 按需求过滤：达人(连续在线<1天) 且 q龄>1 且 等级>1级
		condOnline := dr.PhoneOnlineDay < 1
		condAge := ql.Age > 1
		condLevel := ql.Level > 1
		onlineDay := dr.PhoneOnlineDay
		qAge := ql.Age
		qLevel := ql.Level
		if condOnline && condAge && condLevel {
			infoList = append(infoList, naichaQQInfo{
				QQ:             qq,
				PhoneOnlineDay: &onlineDay,
				QAge:           &qAge,
				QLevel:         &qLevel,
				Qualified:      true,
			})
			global.GVA_LOG.Info("【注册任务】奶茶筛选-QQ通过", append(
				taskLogFieldsWithOpQQ(*task, qq),
				zap.Int("phoneOnlineDay", dr.PhoneOnlineDay),
				zap.Int("qAge", ql.Age),
				zap.Int("qLevel", ql.Level),
			)...)
			filtered = append(filtered, qq)
			continue
		}
		reasons := make([]string, 0, 3)
		if !condOnline {
			reasons = append(reasons, "达人条件不满足(连续在线>=1天)")
		}
		if !condAge {
			reasons = append(reasons, "Q龄条件不满足(<=1年)")
		}
		if !condLevel {
			reasons = append(reasons, "等级条件不满足(<=1级)")
		}
		infoList = append(infoList, naichaQQInfo{
			QQ:             qq,
			PhoneOnlineDay: &onlineDay,
			QAge:           &qAge,
			QLevel:         &qLevel,
			Qualified:      false,
			Reasons:        reasons,
		})
		global.GVA_LOG.Info("【注册任务】奶茶筛选-QQ未通过", append(
			taskLogFieldsWithOpQQ(*task, qq),
			zap.Int("phoneOnlineDay", dr.PhoneOnlineDay),
			zap.Int("qAge", ql.Age),
			zap.Int("qLevel", ql.Level),
			zap.Strings("reasons", reasons),
		)...)
	}
	if task != nil {
		infoBin, marshalErr := json.Marshal(infoList)
		if marshalErr != nil {
			global.GVA_LOG.Warn("【注册任务】奶茶筛选-QQ信息序列化失败", append(taskLogFields(*task), zap.Error(marshalErr))...)
		} else {
			task.QQNaichaInfo = string(infoBin)
		}
	}
	if bypassFilter {
		return qqList, nil
	}
	return filtered, nil
}

func shouldBypassNaichaFilterForPhone(phone string) bool {
	p := strings.TrimSpace(phone)
	if p == "" {
		return false
	}
	_, ok := registerTaskNaichaBypassPhones[p]
	return ok
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
	if client == nil {
		return
	}
	ts := loadOrCreateTaskSession(taskID)
	ts.PhoneBindClient = client
	ts.CreatedAt = time.Now()
	registerTaskSessions.Store(taskID, ts)
}

func hasPhoneBindSession(taskID uint) bool {
	ts, ok := getTaskSession(taskID)
	return ok && ts != nil && ts.PhoneBindClient != nil
}

func popPhoneBindSession(taskID uint) (*qpi.LoginClient, bool) {
	ts, ok := getTaskSession(taskID)
	if !ok || ts == nil || ts.PhoneBindClient == nil {
		return nil, false
	}
	client := ts.PhoneBindClient
	ts.PhoneBindClient = nil
	storeOrDeleteTaskSession(taskID, ts)
	return client, true
}

func clearPhoneBindSession(taskID uint) {
	ts, ok := getTaskSession(taskID)
	if !ok || ts == nil {
		return
	}
	if ts.PhoneBindClient != nil {
		_ = ts.PhoneBindClient.Close()
	}
	ts.PhoneBindClient = nil
	storeOrDeleteTaskSession(taskID, ts)
}

func setChangePasswordSession(taskID uint, client *qpi.Client, qq string, rand string, ticket string) {
	if client == nil {
		return
	}
	ts := loadOrCreateTaskSession(taskID)
	ts.ChangePwdClient = client
	ts.ChangePwdQQ = strings.TrimSpace(qq)
	ts.ChangePwdRand = strings.TrimSpace(rand)
	ts.ChangePwdTicket = strings.TrimSpace(ticket)
	ts.CreatedAt = time.Now()
	registerTaskSessions.Store(taskID, ts)
}

func getChangePasswordSession(taskID uint) (*taskSession, bool) {
	ts, ok := getTaskSession(taskID)
	if !ok || ts == nil || ts.ChangePwdClient == nil {
		return nil, false
	}
	return ts, true
}

func clearChangePasswordSession(taskID uint) {
	ts, ok := getTaskSession(taskID)
	if !ok || ts == nil {
		return
	}
	ts.ChangePwdClient = nil
	ts.ChangePwdQQ = ""
	ts.ChangePwdRand = ""
	ts.ChangePwdTicket = ""
	storeOrDeleteTaskSession(taskID, ts)
}

func setLoginSession(taskID uint, client *qpi.PasswordLoginClient, qq string, codeCh chan string) {
	if client == nil {
		return
	}
	ts := loadOrCreateTaskSession(taskID)
	ts.LoginClient = client
	ts.LoginQQ = strings.TrimSpace(qq)
	ts.LoginCodeCh = codeCh
	ts.CreatedAt = time.Now()
	registerTaskSessions.Store(taskID, ts)
}

func getLoginSession(taskID uint) (*taskSession, bool) {
	ts, ok := getTaskSession(taskID)
	if !ok || ts == nil || ts.LoginClient == nil {
		return nil, false
	}
	return ts, true
}

func clearLoginSession(taskID uint) {
	ts, ok := getTaskSession(taskID)
	if !ok || ts == nil {
		return
	}
	closeLoginCodeCh(ts)
	if ts.LoginClient != nil {
		_ = ts.LoginClient.Close()
	}
	ts.LoginClient = nil
	ts.LoginQQ = ""
	ts.LoginCodeCh = nil
	storeOrDeleteTaskSession(taskID, ts)
}

func offerLoginVerifyCode(taskID uint, qq string, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, errors.New("验证码不能为空")
	}
	ts, ok := getLoginSession(taskID)
	if !ok || ts == nil || ts.LoginClient == nil || ts.LoginCodeCh == nil {
		return false, nil
	}
	if strings.TrimSpace(ts.LoginQQ) != strings.TrimSpace(qq) {
		return false, nil
	}
	select {
	case ts.LoginCodeCh <- code:
		return true, nil
	default:
		return false, errors.New("验证码已提交，请勿重复提交")
	}
}

func closeLoginCodeCh(ts *taskSession) {
	if ts == nil || ts.LoginCodeCh == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	close(ts.LoginCodeCh)
}

func clearTaskSession(taskID uint) {
	v, ok := registerTaskSessions.LoadAndDelete(taskID)
	if !ok {
		return
	}
	ts, ok := v.(*taskSession)
	if !ok || ts == nil {
		return
	}
	if ts.PhoneBindClient != nil {
		_ = ts.PhoneBindClient.Close()
	}
	closeLoginCodeCh(ts)
	if ts.LoginClient != nil {
		_ = ts.LoginClient.Close()
	}
	ts.LoginCodeCh = nil
}

func getTaskSession(taskID uint) (*taskSession, bool) {
	v, ok := registerTaskSessions.Load(taskID)
	if !ok {
		return nil, false
	}
	ts, ok := v.(*taskSession)
	if !ok || ts == nil {
		return nil, false
	}
	return ts, true
}

func loadOrCreateTaskSession(taskID uint) *taskSession {
	if ts, ok := getTaskSession(taskID); ok && ts != nil {
		return ts
	}
	return &taskSession{CreatedAt: time.Now()}
}

func storeOrDeleteTaskSession(taskID uint, ts *taskSession) {
	if ts == nil {
		registerTaskSessions.Delete(taskID)
		return
	}
	if ts.PhoneBindClient == nil && ts.ChangePwdClient == nil {
		if ts.LoginClient == nil {
			registerTaskSessions.Delete(taskID)
			return
		}
	}
	if ts.PhoneBindClient == nil && ts.ChangePwdClient == nil && ts.LoginClient == nil {
		registerTaskSessions.Delete(taskID)
		return
	}
	registerTaskSessions.Store(taskID, ts)
}

func buildResetTaskForReuse(task system.SysRegisterTask, promoterID uint, leaderID *uint) system.SysRegisterTask {
	clearTaskSession(task.ID)
	task.QQPassword = ""
	task.LoginCacheINI = ""
	task.QQCandidates = ""
	task.QQNaichaInfo = ""
	task.QQChangedList = ""
	task.QQLoggedList = ""
	task.IsDaren = nil
	task.StatusCode = nil
	task.CurrentStep = system.RegisterTaskStepPhoneBind
	task.LastError = ""
	task.RetryCount = 0
	task.ExportedAt = nil
	task.ExportedBy = nil
	task.PromoterID = promoterID
	task.LeaderID = leaderID
	task.ChangePasswordAt = nil
	task.LoginAt = nil
	task.FinishedAt = nil
	task.ExpiresAt = time.Now().Add(registerTaskTimeout)
	return task
}

func (s *RegisterTaskService) preparePhoneBindSMS(task *system.SysRegisterTask, runtimeCfg systemRegisterConfig) error {
	global.GVA_LOG.Info("【注册任务】发短信-开始执行", taskLogFields(*task)...)
	captcha, err := s.getCaptchaToken(runtimeCfg, qpi.FindBindAppID, "")
	if err != nil {
		global.GVA_LOG.Error("【注册任务】发短信-获取滑块验证码失败", append(taskLogFields(*task), zap.Error(err))...)
		return err
	}
	global.GVA_LOG.Info("【注册任务】发短信-滑块验证码获取成功", append(taskLogFields(*task), zap.String("randstr", captcha.Randstr), zap.String("ticket", captcha.Ticket))...)
	proxyURL, err := s.allocateProxyURL(runtimeCfg, task.Phone)
	if err != nil {
		global.GVA_LOG.Error("【注册任务】发短信-分配代理失败", append(taskLogFields(*task), zap.Error(err))...)
		return err
	}
	global.GVA_LOG.Info("【注册任务】发短信-代理分配成功", append(taskLogFields(*task), zap.String("proxyURL", proxyURL))...)

	client := qpi.NewLoginClient()
	if proxyURL != "" {
		if err := client.SetProxy(proxyURL); err != nil {
			_ = client.Close()
			global.GVA_LOG.Error("【注册任务】发短信-设置代理失败", append(taskLogFields(*task), zap.Error(err))...)
			return err
		}
	}
	rsp, err := client.GetLoginSMS(qpi.LoginSMSRequest{
		AreaCode: "+86",
		Phone:    task.Phone,
		Randstr:  captcha.Randstr,
		Ticket:   captcha.Ticket,
	})
	if err != nil {
		_ = client.Close()
		if nonRetryable, failMsg := resolvePhoneBindNonRetryableError(err); nonRetryable {
			global.GVA_LOG.Warn("【注册任务】发短信-命中不可重试错误，直接结束任务", append(taskLogFields(*task), zap.String("failMessage", failMsg), zap.Error(err))...)
			finished, finishErr := s.finishTask(*task, system.RegisterTaskFailCodeNoQQBound, failMsg)
			if finishErr != nil {
				global.GVA_LOG.Error("【注册任务】发短信-不可重试错误结束任务失败", append(taskLogFields(*task), zap.Error(finishErr))...)
				return finishErr
			}
			*task = finished
			return nil
		}
		global.GVA_LOG.Error("【注册任务】发短信-请求登录短信失败", append(taskLogFields(*task), zap.Error(err))...)
		return err
	}
	global.GVA_LOG.Info("【注册任务】发短信-登录短信响应", append(taskLogFields(*task), zap.Any("rsp", rsp))...)

	setPhoneBindSession(task.ID, client)
	task.LastError = "验证码已发送，请输入后提交"
	if err := global.GVA_DB.Save(task).Error; err != nil {
		global.GVA_LOG.Error("【注册任务】发短信-保存任务失败", append(taskLogFields(*task), zap.Error(err))...)
		return err
	}
	global.GVA_LOG.Info("【注册任务】发短信-已发起", taskLogFields(*task)...)
	return nil
}

func (s *RegisterTaskService) restorePhoneBindSessionIfNeeded(task *system.SysRegisterTask) error {
	if task == nil || task.FinishedAt != nil || task.CurrentStep != system.RegisterTaskStepPhoneBind {
		return nil
	}
	if hasPhoneBindSession(task.ID) {
		return nil
	}
	if hasPhoneBindSMSSent(*task) {
		global.GVA_LOG.Info("【注册任务】恢复手机绑定会话-已发过验证码，跳过自动补发", taskLogFields(*task)...)
		return nil
	}
	global.GVA_LOG.Info("【注册任务】恢复手机绑定会话-开始", taskLogFields(*task)...)
	runtimeCfg, err := s.getRegisterRuntimeConfig(task.LeaderID)
	if err != nil {
		return err
	}
	if err := s.preparePhoneBindSMS(task, runtimeCfg); err != nil {
		return err
	}
	global.GVA_LOG.Info("【注册任务】恢复手机绑定会话成功", taskLogFields(*task)...)
	return nil
}

func hasPhoneBindSMSSent(task system.SysRegisterTask) bool {
	return strings.Contains(strings.TrimSpace(task.LastError), "验证码已发送")
}

func resolvePhoneBindNonRetryableError(err error) (bool, string) {
	if err == nil {
		return false, ""
	}
	var commonErr *qpi.CommonError
	if errors.As(err, &commonErr) {
		if commonErr.Code == 219 {
			msg := strings.TrimSpace(commonErr.Message)
			if msg == "" {
				msg = "当前手机号未绑定QQ号"
			}
			return true, msg
		}
		msg := strings.TrimSpace(commonErr.Message)
		if strings.Contains(msg, "未绑定QQ") {
			return true, msg
		}
	}
	msg := strings.TrimSpace(err.Error())
	if strings.Contains(msg, "当前手机号未绑定QQ号") {
		return true, "当前手机号未绑定QQ号"
	}
	if strings.Contains(msg, "code=219") && strings.Contains(msg, "未绑定QQ") {
		return true, "当前手机号未绑定QQ号"
	}
	return false, ""
}

func (s *RegisterTaskService) restoreTaskProgressIfNeeded(task *system.SysRegisterTask) error {
	if task == nil || task.FinishedAt != nil {
		return nil
	}
	global.GVA_LOG.Info("【注册任务】恢复任务进度-开始", append(taskLogFields(*task), zap.String("currentStep", task.CurrentStep))...)
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		err := s.restorePhoneBindSessionIfNeeded(task)
		if err != nil {
			global.GVA_LOG.Error("【注册任务】恢复任务进度-phone_bind失败", append(taskLogFields(*task), zap.Error(err))...)
			return err
		}
		global.GVA_LOG.Info("【注册任务】恢复任务进度-phone_bind完成", taskLogFields(*task)...)
		return nil
	case system.RegisterTaskStepChangePassword:
		// 改密步骤无类似 phone_bind 的内存会话；每次 submit 都会新建 Client 并重新取滑块。
		// 此处仅做 DB 进度修复（列表截断/步骤推进），不自动调用 qpi，避免无验证码时误触改密。
		changed := splitQQList(task.QQChangedList)
		candidates := splitQQList(task.QQCandidates)
		nextQQ := ""
		if len(candidates) > 0 && len(changed) < len(candidates) {
			nextQQ = candidates[len(changed)]
		}
		global.GVA_LOG.Info("【注册任务】恢复改密进度-说明",
			append(taskLogFields(*task),
				zap.Int("candidateCount", len(candidates)),
				zap.Int("changedCount", len(changed)),
				zap.String("nextQQ", nextQQ),
				zap.String("hint", "等待用户提交验证码后继续改密，不会在恢复阶段自动重跑协议"),
			)...,
		)
		updated := false
		if len(candidates) > 0 && len(changed) > len(candidates) {
			changed = changed[:len(candidates)]
			task.QQChangedList = strings.Join(changed, "|")
			updated = true
		}
		if len(candidates) > 0 && len(changed) >= len(candidates) {
			task.CurrentStep = system.RegisterTaskStepLogin
			task.LastError = ""
			updated = true
			global.GVA_LOG.Info("【注册任务】恢复改密进度-已满足进入登录条件", append(taskLogFields(*task), zap.Int("changed", len(changed)), zap.Int("candidates", len(candidates)))...)
		}
		if updated {
			if err := global.GVA_DB.Save(task).Error; err != nil {
				global.GVA_LOG.Error("【注册任务】恢复改密进度-保存失败", append(taskLogFields(*task), zap.Error(err))...)
				return err
			}
			global.GVA_LOG.Info("【注册任务】恢复改密进度-保存成功", append(taskLogFields(*task), zap.Int("changed", len(splitQQList(task.QQChangedList))), zap.Int("candidates", len(splitQQList(task.QQCandidates))))...)
			return nil
		}
		global.GVA_LOG.Info("【注册任务】恢复改密进度-无需修复", append(taskLogFields(*task), zap.Int("changed", len(changed)), zap.Int("candidates", len(candidates)))...)
		runtimeCfg, cfgErr := s.getRegisterRuntimeConfig(task.LeaderID)
		if cfgErr != nil {
			return cfgErr
		}
		proxyURL, pErr := s.allocateProxyURL(runtimeCfg, task.Phone)
		if pErr != nil {
			return pErr
		}
		global.GVA_LOG.Info("【注册任务】恢复改密进度-分配代理成功", append(taskLogFields(*task), zap.String("proxyURL", proxyURL))...)
		if prepErr := s.prepareChangePasswordSMSIfNeeded(task, runtimeCfg, proxyURL, false); prepErr != nil {
			global.GVA_LOG.Error("【注册任务】恢复改密进度-预发验证码失败", append(taskLogFields(*task), zap.Error(prepErr))...)
			return prepErr
		}
		if err := global.GVA_DB.Save(task).Error; err != nil {
			return err
		}
		return nil
	case system.RegisterTaskStepLogin:
		// 登录同样无持久内存会话；submit 时新建 PasswordLoginClient。此处只做 DB 修复或已全部登录时收尾。
		changed := splitQQList(task.QQChangedList)
		logged := splitQQList(task.QQLoggedList)
		nextQQ := ""
		if len(changed) > 0 && len(logged) < len(changed) {
			nextQQ = changed[len(logged)]
		}
		global.GVA_LOG.Info("【注册任务】恢复登录进度-说明",
			append(taskLogFields(*task),
				zap.Int("changedCount", len(changed)),
				zap.Int("loggedCount", len(logged)),
				zap.String("nextQQ", nextQQ),
				zap.String("hint", "改密完成后已自动进入登录步骤；登录需提交验证码后逐个处理"),
			)...,
		)
		updated := false
		if len(changed) > 0 {
			if len(logged) > len(changed) {
				logged = logged[:len(changed)]
				task.QQLoggedList = strings.Join(logged, "|")
				updated = true
			}
			if len(logged) >= len(changed) {
				isDaren := true
				task.IsDaren = &isDaren
				if task.LoginAt == nil {
					now := time.Now()
					task.LoginAt = &now
				}
				finished, err := s.finishTask(*task, 0, "")
				if err != nil {
					return err
				}
				*task = finished
				global.GVA_LOG.Info("【注册任务】恢复登录进度-任务已自动补全完成", append(taskLogFields(*task), zap.Int("logged", len(logged)), zap.Int("changed", len(changed)))...)
				return nil
			}
		}
		if updated {
			if err := global.GVA_DB.Save(task).Error; err != nil {
				global.GVA_LOG.Error("【注册任务】恢复登录进度-保存失败", append(taskLogFields(*task), zap.Error(err))...)
				return err
			}
			global.GVA_LOG.Info("【注册任务】恢复登录进度-保存成功", append(taskLogFields(*task), zap.Int("logged", len(splitQQList(task.QQLoggedList))), zap.Int("changed", len(splitQQList(task.QQChangedList))))...)
			return nil
		}
		global.GVA_LOG.Info("【注册任务】恢复登录进度-无需修复", append(taskLogFields(*task), zap.Int("logged", len(splitQQList(task.QQLoggedList))), zap.Int("changed", len(changed)))...)
		return nil
	default:
		global.GVA_LOG.Warn("【注册任务】恢复任务进度-未知步骤，跳过", append(taskLogFields(*task), zap.String("currentStep", task.CurrentStep))...)
		return nil
	}
}

func (s *RegisterTaskService) prepareChangePasswordSMSIfNeeded(task *system.SysRegisterTask, runtimeCfg systemRegisterConfig, proxyURL string, force bool) error {
	if task == nil || task.CurrentStep != system.RegisterTaskStepChangePassword {
		return nil
	}
	candidateQQList := splitQQList(task.QQCandidates)
	changedQQList := splitQQList(task.QQChangedList)
	if len(candidateQQList) == 0 || len(changedQQList) >= len(candidateQQList) {
		clearChangePasswordSession(task.ID)
		return nil
	}
	currentQQ := candidateQQList[len(changedQQList)]
	if ps, ok := getChangePasswordSession(task.ID); ok && ps != nil && ps.ChangePwdClient != nil && ps.ChangePwdQQ == currentQQ {
		return nil
	}
	if !force && strings.Contains(strings.TrimSpace(task.LastError), "改密验证码已发送") {
		return nil
	}
	clearChangePasswordSession(task.ID)

	global.GVA_LOG.Info("【注册任务】改密-开始获取滑块ticket", taskLogFields(*task)...)
	captcha, capErr := s.getCaptchaToken(runtimeCfg, qpi.ChangePasswordAppID, "")
	if capErr != nil {
		global.GVA_LOG.Error("【注册任务】改密-获取滑块ticket失败", append(taskLogFields(*task), zap.Error(capErr))...)
		return capErr
	}
	client := qpi.NewClient()
	if proxyURL != "" {
		if proxyErr := client.SetProxy(proxyURL); proxyErr != nil {
			global.GVA_LOG.Error("【注册任务】改密-设置代理失败", append(taskLogFields(*task), zap.Error(proxyErr))...)
			return proxyErr
		}
	}
	global.GVA_LOG.Info("【注册任务】改密-开始预发验证码", taskLogFields(*task)...)
	if err := s.prepareChangePasswordForQQ(client, currentQQ, task.Phone, captcha); err != nil {
		global.GVA_LOG.Error("【注册任务】改密-预发验证码失败", append(taskLogFieldsWithOpQQ(*task, currentQQ), zap.Error(err))...)
		return err
	}
	setChangePasswordSession(task.ID, client, currentQQ, captcha.Randstr, captcha.Ticket)
	task.LastError = "改密验证码已发送，请输入后提交"
	global.GVA_LOG.Info("【注册任务】改密-已自动发送验证码", taskLogFieldsWithOpQQ(*task, currentQQ)...)
	return nil
}

func (s *RegisterTaskService) prepareChangePasswordForQQ(client *qpi.Client, qq string, phone string, captcha *captchaToken) error {
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
	return sendSMSResp.Error()
}

func (s *RegisterTaskService) completeChangePasswordForQQ(ps *taskSession, qq string, phone string, verifyCode string, newPassword string) error {
	verifySMSResp, err := ps.ChangePwdClient.VerifySMSCode(qpi.SMSCodeVerifyRequest{
		QQ:       qq,
		Randstr:  ps.ChangePwdRand,
		Ticket:   ps.ChangePwdTicket,
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
	changePwdResp, err := ps.ChangePwdClient.ChangePassword(qpi.PasswordChangeRequest{
		QQ:       qq,
		Randstr:  ps.ChangePwdRand,
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

func parseCaptchaAidSid(captchaURL string) (aid, sid string) {
	captchaURL = strings.TrimSpace(captchaURL)
	if captchaURL == "" {
		return "", ""
	}
	u, err := url.Parse(captchaURL)
	if err != nil {
		return "", ""
	}
	q := u.Query()
	return strings.TrimSpace(q.Get("aid")), strings.TrimSpace(q.Get("sid"))
}

func isRetryableLoginNetworkErr(err error) bool {
	if err == nil {
		return false
	}
	// 优先使用 qpi 的结构化错误类型进行判断，避免仅靠字符串匹配。
	var ce *qpi.CommonError
	if errors.As(err, &ce) && ce != nil {
		if ce.Type == qpi.ErrorTypeBusiness && ce.Code == 237 {
			return true
		}
	}
	var be *qpi.BusinessError
	if errors.As(err, &be) && be != nil && be.Code == 237 {
		return true
	}
	// 兜底兼容：上游返回文本错误时仍可识别。
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "code=237") || strings.Contains(msg, "网络不稳定") || strings.Contains(msg, "network")
}
