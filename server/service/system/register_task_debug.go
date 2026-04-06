package system

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"gorm.io/gorm"
)

func (s *RegisterTaskService) CreateDebugLoginTask(req systemReq.RegisterTaskDebugLoginStart) (system.SysRegisterTask, error) {
	phone := strings.TrimSpace(req.Phone)
	uin := strings.TrimSpace(req.UIN)
	password := strings.TrimSpace(req.Password)
	if phone == "" {
		return system.SysRegisterTask{}, errors.New("手机号不能为空")
	}
	if uin == "" {
		return system.SysRegisterTask{}, errors.New("UIN不能为空")
	}
	if !isDigitsOnly(uin) {
		return system.SysRegisterTask{}, errors.New("UIN必须为纯数字QQ号，请填写QQ号码而不是账号")
	}
	if password == "" {
		return system.SysRegisterTask{}, errors.New("密码不能为空")
	}

	var task system.SysRegisterTask
	findErr := global.GVA_DB.Where("phone = ?", phone).Order("id DESC").First(&task).Error
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return system.SysRegisterTask{}, findErr
	}
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		task = system.SysRegisterTask{
			Phone: phone,
		}
		task = buildDebugLoginTaskForReuse(task, uin, password, "调试登录任务已创建并发起执行")
		if err := global.GVA_DB.Create(&task).Error; err != nil {
			return system.SysRegisterTask{}, err
		}
	} else {
		task = buildDebugLoginTaskForReuse(task, uin, password, fmt.Sprintf("调试登录任务已重新发起（复用任务#%d）", task.ID))
		if err := global.GVA_DB.Save(&task).Error; err != nil {
			return system.SysRegisterTask{}, err
		}
	}
	ensureRegisterTaskRunner(task.ID, 0)
	clearTaskSession(task.ID)
	clearRegisterTaskRunnerPendingEvents(task.ID)
	if err := enqueueRegisterTaskEvent(task.ID, 0, registerTaskEvent{
		Action: "submit",
		Step:   system.RegisterTaskStepLogin,
	}); err != nil {
		return system.SysRegisterTask{}, err
	}
	return task, nil
}

func isDigitsOnly(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func buildDebugLoginTaskForReuse(task system.SysRegisterTask, uin, password, initMessage string) system.SysRegisterTask {
	task.CurrentStep = system.RegisterTaskStepLogin
	task.PromoterID = 0
	task.LeaderID = nil
	task.QQCandidates = uin
	task.QQChangedList = uin
	task.QQLoggedList = ""
	task.QQPassword = password
	task.LoginCacheINI = ""
	task.IsDaren = nil
	task.StatusCode = nil
	task.LastError = strings.TrimSpace(initMessage)
	task.RetryCount = 0
	task.ChangePasswordAt = nil
	task.LoginAt = nil
	task.FinishedAt = nil
	task.ExpiresAt = time.Now().Add(registerTaskTimeout)
	return task
}

func (s *RegisterTaskService) SubmitDebugLoginCode(req systemReq.RegisterTaskDebugLoginSubmit) (system.SysRegisterTask, error) {
	if req.TaskID == 0 {
		return system.SysRegisterTask{}, errors.New("任务ID不能为空")
	}
	code := strings.TrimSpace(req.VerifyCode)
	if code == "" {
		return system.SysRegisterTask{}, errors.New("验证码不能为空")
	}
	var task system.SysRegisterTask
	if err := global.GVA_DB.Where("id = ?", req.TaskID).First(&task).Error; err != nil {
		return system.SysRegisterTask{}, errors.New("任务不存在")
	}
	if task.FinishedAt != nil {
		return task, errors.New("任务已完成")
	}
	if task.CurrentStep != system.RegisterTaskStepLogin {
		return task, errors.New("任务不在登录步骤")
	}
	changedQQList := splitQQList(task.QQChangedList)
	loggedQQList := splitQQList(task.QQLoggedList)
	if len(loggedQQList) >= len(changedQQList) || len(changedQQList) == 0 {
		return task, errors.New("无待登录QQ")
	}
	currentQQ := changedQQList[len(loggedQQList)]
	delivered, err := offerLoginVerifyCode(task.ID, currentQQ, code)
	if err != nil {
		return task, err
	}
	if !delivered {
		return task, errors.New("登录会话已失效，请重新执行调试登录")
	}
	task.LastError = ""
	if err := global.GVA_DB.Save(&task).Error; err != nil {
		return task, err
	}
	return task, nil
}

func (s *RegisterTaskService) GetTaskByID(taskID uint) (system.SysRegisterTask, error) {
	if taskID == 0 {
		return system.SysRegisterTask{}, errors.New("任务ID不能为空")
	}
	var task system.SysRegisterTask
	if err := global.GVA_DB.Where("id = ?", taskID).First(&task).Error; err != nil {
		return system.SysRegisterTask{}, err
	}
	return task, nil
}
