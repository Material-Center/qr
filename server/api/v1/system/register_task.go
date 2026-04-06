package system

import (
	"errors"
	"fmt"
	"net/url"
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

// GetActiveRegisterTasks
// @Tags      RegisterTask
// @Summary   获取地推全部未完成任务
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]systemRes.RegisterTaskActiveInfo,msg=string}
// @Router    /registerTask/actives [get]
func (a *RegisterTaskApi) GetActiveRegisterTasks(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRolePromoter {
		response.FailWithMessage("仅地推可查看当前任务", c)
		return
	}
	tasks, err := registerTaskService.GetActiveTasks(utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	items := make([]systemRes.RegisterTaskActiveInfo, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, buildActiveInfo(t))
	}
	response.OkWithDetailed(items, "获取成功", c)
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

// StartRegisterTaskDebugLogin
// @Tags      RegisterTask
// @Summary   管理员启动登录调试
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.RegisterTaskDebugLoginStart  true  "调试登录参数"
// @Success   200   {object}  response.Response{data=systemRes.RegisterTaskActiveInfo,msg=string}
// @Router    /registerTask/debug/login/start [post]
func (a *RegisterTaskApi) StartRegisterTaskDebugLogin(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRoleSuperAdmin && role != rtRoleAdmin {
		response.FailWithMessage("仅管理员可启动调试登录", c)
		return
	}
	var req systemReq.RegisterTaskDebugLoginStart
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, err := registerTaskService.CreateDebugLoginTask(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildActiveInfo(task), "调试任务已启动", c)
}

// SubmitRegisterTaskDebugLoginCode
// @Tags      RegisterTask
// @Summary   管理员提交调试登录验证码
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.RegisterTaskDebugLoginSubmit  true  "调试验证码"
// @Success   200   {object}  response.Response{data=systemRes.RegisterTaskActiveInfo,msg=string}
// @Router    /registerTask/debug/login/submit [post]
func (a *RegisterTaskApi) SubmitRegisterTaskDebugLoginCode(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRoleSuperAdmin && role != rtRoleAdmin {
		response.FailWithMessage("仅管理员可提交调试验证码", c)
		return
	}
	var req systemReq.RegisterTaskDebugLoginSubmit
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, err := registerTaskService.SubmitDebugLoginCode(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildActiveInfo(task), "验证码已提交", c)
}

// GetRegisterTaskDebugLoginTask
// @Tags      RegisterTask
// @Summary   管理员查询调试登录任务
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     taskId  query     int  true  "任务ID"
// @Success   200     {object}  response.Response{data=systemRes.RegisterTaskActiveInfo,msg=string}
// @Router    /registerTask/debug/login/task [get]
func (a *RegisterTaskApi) GetRegisterTaskDebugLoginTask(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRoleSuperAdmin && role != rtRoleAdmin {
		response.FailWithMessage("仅管理员可查看调试任务", c)
		return
	}
	taskID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("taskId")), 10, 64)
	task, err := registerTaskService.GetTaskByID(uint(taskID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithMessage("调试任务不存在", c)
			return
		}
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildActiveInfo(task), "获取成功", c)
}

// DownloadRegisterTaskCache
// @Tags      RegisterTask
// @Summary   下载任务登录缓存INI（仅管理员）
// @Security  ApiKeyAuth
// @Produce   application/octet-stream
// @Param     taskId  query     int  true  "任务ID"
// @Success   200     {file}    file
// @Router    /registerTask/cache/download [get]
func (a *RegisterTaskApi) DownloadRegisterTaskCache(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != rtRoleSuperAdmin && role != rtRoleAdmin {
		response.FailWithMessage("仅管理员可下载缓存", c)
		return
	}
	taskID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("taskId")), 10, 64)
	if taskID == 0 {
		response.FailWithMessage("任务ID不能为空", c)
		return
	}
	task, err := registerTaskService.GetTaskByID(uint(taskID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithMessage("任务不存在", c)
			return
		}
		response.FailWithMessage(err.Error(), c)
		return
	}
	cacheINI := strings.TrimSpace(task.LoginCacheINI)
	if cacheINI == "" {
		response.FailWithMessage("该任务暂无可下载缓存", c)
		return
	}
	filename := fmt.Sprintf("register_task_%d.ini", task.ID)
	escapedFilename := url.QueryEscape(filename)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", filename, escapedFilename))
	c.Data(200, "application/octet-stream", []byte(cacheINI))
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
	phase := deriveTaskStepPhase(task)
	if phase != "waiting_code" {
		return false
	}
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind, system.RegisterTaskStepChangePassword:
		return true
	case system.RegisterTaskStepLogin:
		return true
	default:
		return false
	}
}

func buildStepTexts(task system.SysRegisterTask) (title string, hint string, progress string, verifyLabel string, verifyPlace string, submitText string) {
	phase := deriveTaskStepPhase(task)
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		if phase == "failed" {
			return "手机号查绑QQ", "发码失败，请点击重试当前步骤后再提交验证码。", "", "短信验证码", "等待重试后再输入验证码", "等待重试"
		}
		if phase == "preparing" {
			return "手机号查绑QQ", "正在发送短信验证码，请稍候；若长时间无变化可点击重试。", "", "短信验证码", "验证码发送成功后再输入", "处理中"
		}
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
		if phase == "failed" {
			return "候选QQ改密", "改密步骤发码或执行失败，请点击重试当前步骤。", progressText, "改密验证码", "等待重试后再输入验证码", "等待重试"
		}
		if phase == "preparing" {
			return "候选QQ改密", "正在准备当前QQ改密验证码，请稍候；若长时间无变化可点击重试。", progressText, "改密验证码", "验证码发送成功后再输入", "处理中"
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
		if phase == "failed" {
			return "改密后QQ登录", "登录步骤失败，请点击重试当前步骤；仅在触发短信验证后再输入验证码。", progressText, "登录验证码", "触发短信验证后再输入验证码", "等待重试"
		}
		if phase == "preparing" || phase == "running" {
			return "改密后QQ登录", "正在执行登录流程；仅在触发短信验证后才需要输入验证码。", progressText, "登录验证码", "触发短信验证后再输入验证码", "处理中"
		}
		return "改密后QQ登录", "改密完成后自动进入登录流程；每次提交后端会按顺序处理一个QQ登录并保存缓存。", progressText, "登录验证码", "请输入当前待登录QQ对应验证码", "提交并登录下一个QQ"
	default:
		return "任务处理中", "请按后端流程提示操作。", "", "验证码", "请输入验证码", "提交"
	}
}

func deriveTaskStepPhase(task system.SysRegisterTask) string {
	lastError := strings.TrimSpace(task.LastError)
	switch task.CurrentStep {
	case system.RegisterTaskStepPhoneBind:
		if strings.Contains(lastError, "验证码已发送") || strings.Contains(lastError, "验证码不能为空") {
			return "waiting_code"
		}
		if lastError == "" || strings.Contains(lastError, "准备发送验证码") {
			return "preparing"
		}
		return "failed"
	case system.RegisterTaskStepChangePassword:
		if strings.Contains(lastError, "改密验证码已发送") ||
			strings.Contains(lastError, "改密验证码错误") ||
			strings.Contains(lastError, "验证码错误，请重新输入") ||
			strings.Contains(lastError, "验证码错误") ||
			strings.Contains(lastError, "code=2000080") {
			return "waiting_code"
		}
		if lastError == "" || strings.Contains(lastError, "准备") || strings.Contains(lastError, "处理中") {
			return "preparing"
		}
		return "failed"
	case system.RegisterTaskStepLogin:
		if strings.Contains(lastError, "触发短信验证") || strings.Contains(lastError, "登录验证码") {
			return "waiting_code"
		}
		if lastError == "" {
			return "running"
		}
		return "failed"
	default:
		if lastError == "" {
			return "running"
		}
		return "failed"
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
