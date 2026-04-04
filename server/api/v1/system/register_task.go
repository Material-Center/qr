package system

import (
	"errors"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RegisterTaskApi struct{}

const (
	rtRoleSuperAdmin = uint(888)
	rtRoleAdmin      = uint(100)
	rtRoleLeader     = uint(200)
	rtRolePromoter   = uint(300)
)

// CreateRegisterTask
// @Tags      RegisterTask
// @Summary   地推创建注册任务
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.RegisterTaskCreate  true  "手机号"
// @Success   200   {object}  response.Response{data=systemRes.RegisterTaskActiveInfo,msg=string}
// @Router    /registerTask/create [post]
func (a *RegisterTaskApi) CreateRegisterTask(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRolePromoter {
		response.FailWithMessage("仅地推可创建任务", c)
		return
	}

	var req systemReq.RegisterTaskCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		response.FailWithMessage("手机号不能为空", c)
		return
	}

	task, err := registerTaskService.CreateTask(utils.GetUserID(c), req.Phone)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildActiveInfo(task), "创建成功", c)
}

// GetActiveRegisterTask
// @Tags      RegisterTask
// @Summary   获取地推当前未完成任务
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemRes.RegisterTaskActiveInfo,msg=string}
// @Router    /registerTask/active [get]
func (a *RegisterTaskApi) GetActiveRegisterTask(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRolePromoter {
		response.FailWithMessage("仅地推可查看当前任务", c)
		return
	}
	task, err := registerTaskService.GetActiveTask(utils.GetUserID(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.OkWithDetailed(nil, "暂无未完成任务", c)
			return
		}
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildActiveInfo(task), "获取成功", c)
}

// SubmitRegisterTaskStep
// @Tags      RegisterTask
// @Summary   提交任务步骤验证码/重试/失败
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.RegisterTaskStep  true  "任务步骤提交"
// @Success   200   {object}  response.Response{data=systemRes.RegisterTaskActiveInfo,msg=string}
// @Router    /registerTask/step [post]
func (a *RegisterTaskApi) SubmitRegisterTaskStep(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRolePromoter {
		response.FailWithMessage("仅地推可操作任务步骤", c)
		return
	}
	var req systemReq.RegisterTaskStep
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, err := registerTaskService.SubmitStep(utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildActiveInfo(task), "提交成功", c)
}

// GetRegisterTaskList
// @Tags      RegisterTask
// @Summary   分页查询注册任务
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.RegisterTaskList  true  "分页参数"
// @Success   200   {object}  response.Response{data=systemRes.RegisterTaskListResponse,msg=string}
// @Router    /registerTask/list [post]
func (a *RegisterTaskApi) GetRegisterTaskList(c *gin.Context) {
	var req systemReq.RegisterTaskList
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := registerTaskService.GetTaskList(utils.GetUserAuthorityId(c), utils.GetUserID(c), req)
	if err != nil {
		global.GVA_LOG.Error("获取注册任务列表失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	role := utils.GetUserAuthorityId(c)
	for i := range result.List {
		if role == rtRoleLeader || role == rtRolePromoter {
			result.List[i].Phone = maskPhone(result.List[i].Phone)
		}
	}
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	response.OkWithDetailed(systemRes.RegisterTaskListResponse{
		List:            result.List,
		Total:           result.Total,
		Page:            page,
		PageSize:        pageSize,
		SuccessCount:    result.Success,
		FailCount:       result.Failed,
		ProcessingCount: result.Processing,
	}, "获取成功", c)
}

// GetRegisterTaskSummary
// @Tags      RegisterTask
// @Summary   获取注册任务统计（管理员/团长）
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     leaderId  query     int  false  "按团长过滤"
// @Success   200       {object}  response.Response{data=systemRes.RegisterTaskSummaryResponse,msg=string}
// @Router    /registerTask/summary [get]
func (a *RegisterTaskApi) GetRegisterTaskSummary(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRoleSuperAdmin && role != rtRoleAdmin && role != rtRoleLeader {
		response.FailWithMessage("无权限查看统计", c)
		return
	}

	var req systemReq.RegisterTaskSummaryFilter
	_ = c.ShouldBindQuery(&req)
	data, err := registerTaskService.GetSummary(role, utils.GetUserID(c), req.LeaderID)
	if err != nil {
		global.GVA_LOG.Error("获取注册任务统计失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.RegisterTaskSummaryResponse{
		Leaders:   data.Leaders,
		Promoters: data.Promoters,
	}, "获取成功", c)
}

func buildActiveInfo(task system.SysRegisterTask) systemRes.RegisterTaskActiveInfo {
	stepTitle, stepHint, progress, verifyLabel, verifyPlace, submitText := buildStepTexts(task)
	return systemRes.RegisterTaskActiveInfo{
		ID:          task.ID,
		Phone:       task.Phone,
		CurrentStep: task.CurrentStep,
		StepTitle:   stepTitle,
		StepHint:    stepHint,
		Progress:    progress,
		VerifyLabel: verifyLabel,
		VerifyPlace: verifyPlace,
		NeedVerify:  needVerifyCode(task),
		SubmitText:  submitText,
		RetryText:   "重试当前步骤",
		FailText:    "标记任务失败",
		StatusCode:  task.StatusCode,
		LastError:   task.LastError,
		RetryCount:  task.RetryCount,
		ExpiresAt:   task.ExpiresAt,
		FinishedAt:  task.FinishedAt,
	}
}

func needVerifyCode(task system.SysRegisterTask) bool {
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind, system.RegisterTaskStepChangePassword:
		return true
	case system.RegisterTaskStepLogin:
		return strings.Contains(task.LastError, "触发短信验证")
	default:
		return true
	}
}

func buildStepTexts(task system.SysRegisterTask) (title string, hint string, progress string, verifyLabel string, verifyPlace string, submitText string) {
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		return "手机号查绑QQ", "提交当前短信验证码后，系统会自动查绑并进行奶茶筛选。", "", "短信验证码", "请输入手机号收到的验证码", "提交并查绑"
	case system.RegisterTaskStepChangePassword:
		candidates := splitPipe(task.QQCandidates)
		changed := splitPipe(task.QQChangedList)
		total := len(candidates)
		done := len(changed)
		progressText := ""
		if total > 0 {
			progressText = "改密进度 " + itoa(done) + "/" + itoa(total)
		}
		return "候选QQ改密", "进入改密后会自动发送当前QQ的改密验证码，输入验证码提交即可；全部改密完成后自动进入登录。", progressText, "改密验证码", "请输入当前待改密QQ对应验证码", "提交并改密下一个QQ"
	case system.RegisterTaskStepLogin:
		changed := splitPipe(task.QQChangedList)
		logged := splitPipe(task.QQLoggedList)
		total := len(changed)
		done := len(logged)
		progressText := ""
		if total > 0 {
			progressText = "登录进度 " + itoa(done) + "/" + itoa(total)
		}
		return "改密后QQ登录", "改密完成后自动进入登录流程；每次提交后端会按顺序处理一个QQ登录并保存缓存。", progressText, "登录验证码", "请输入当前待登录QQ对应验证码", "提交并登录下一个QQ"
	default:
		return "任务处理中", "请按后端流程提示操作。", "", "验证码", "请输入验证码", "提交"
	}
}

func splitPipe(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}
