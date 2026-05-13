package system

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PhoneRegisterTaskApi struct{}

// CreatePhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   地推创建手机号注册任务
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterTaskCreate  true  "创建参数"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterTaskActiveInfo,msg=string}
// @Router    /phoneRegisterTask/create [post]
func (a *PhoneRegisterTaskApi) CreatePhoneRegisterTask(c *gin.Context) {
	if utils.GetUserAuthorityId(c) != rtRolePromoter {
		response.FailWithMessage("仅地推可创建任务", c)
		return
	}
	var req systemReq.PhoneRegisterTaskCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, err := phoneRegisterTaskService.CreateTask(utils.GetUserID(c), req.Phone, req.SMSReceiveMode)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildPhoneRegisterActiveInfo(task), "创建成功", c)
}

// GetPhoneRegisterSubmitStatus
// @Tags      PhoneRegisterTask
// @Summary   获取手机号注册提交开关
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemRes.PhoneRegisterSubmitStatusResponse,msg=string}
// @Router    /phoneRegisterTask/submitStatus [get]
func (a *PhoneRegisterTaskApi) GetPhoneRegisterSubmitStatus(c *gin.Context) {
	enabled, message, err := phoneRegisterTaskService.IsSubmitEnabledForUser(utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.PhoneRegisterSubmitStatusResponse{
		Enabled: enabled,
		Message: message,
	}, "获取成功", c)
}

// SubmitPhoneRegisterTaskCode
// @Tags      PhoneRegisterTask
// @Summary   地推提交手机号注册验证码
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterTaskSubmitCode  true  "验证码参数"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterTaskActiveInfo,msg=string}
// @Router    /phoneRegisterTask/submitCode [post]
func (a *PhoneRegisterTaskApi) SubmitPhoneRegisterTaskCode(c *gin.Context) {
	if utils.GetUserAuthorityId(c) != rtRolePromoter {
		response.FailWithMessage("仅地推可提交验证码", c)
		return
	}
	var req systemReq.PhoneRegisterTaskSubmitCode
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, err := phoneRegisterTaskService.SubmitCode(utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildPhoneRegisterActiveInfo(task), "提交成功", c)
}

// GetActivePhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   获取地推当前未完成手机号注册任务
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemRes.PhoneRegisterTaskActiveInfo,msg=string}
// @Router    /phoneRegisterTask/active [get]
func (a *PhoneRegisterTaskApi) GetActivePhoneRegisterTask(c *gin.Context) {
	if utils.GetUserAuthorityId(c) != rtRolePromoter {
		response.FailWithMessage("仅地推可查看当前任务", c)
		return
	}
	task, err := phoneRegisterTaskService.GetActiveTask(utils.GetUserID(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.OkWithDetailed(nil, "暂无未完成任务", c)
			return
		}
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildPhoneRegisterActiveInfo(task), "获取成功", c)
}

// GetActivePhoneRegisterTasks
// @Tags      PhoneRegisterTask
// @Summary   获取地推全部未完成手机号注册任务
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]systemRes.PhoneRegisterTaskActiveInfo,msg=string}
// @Router    /phoneRegisterTask/actives [get]
func (a *PhoneRegisterTaskApi) GetActivePhoneRegisterTasks(c *gin.Context) {
	if utils.GetUserAuthorityId(c) != rtRolePromoter {
		response.FailWithMessage("仅地推可查看当前任务", c)
		return
	}
	tasks, err := phoneRegisterTaskService.GetActiveTasks(utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	items := make([]systemRes.PhoneRegisterTaskActiveInfo, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, buildPhoneRegisterActiveInfo(task))
	}
	response.OkWithDetailed(items, "获取成功", c)
}

// GetPhoneRegisterTaskList
// @Tags      PhoneRegisterTask
// @Summary   分页查询手机号注册任务
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterTaskList  true  "查询参数"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterTaskListResponse,msg=string}
// @Router    /phoneRegisterTask/list [post]
func (a *PhoneRegisterTaskApi) GetPhoneRegisterTaskList(c *gin.Context) {
	var req systemReq.PhoneRegisterTaskList
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := phoneRegisterTaskService.GetTaskList(utils.GetUserAuthorityId(c), utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	response.OkWithDetailed(systemRes.PhoneRegisterTaskListResponse{
		List:              result.List,
		Total:             result.Total,
		Page:              page,
		PageSize:          pageSize,
		SuccessCount:      result.Success,
		FailCount:         result.Failed,
		ProcessingCount:   result.Processing,
		DeviceOnlineCount: result.Device.Online,
		DeviceIdleCount:   result.Device.Idle,
	}, "获取成功", c)
}

// GetPhoneRegisterTaskSummary
// @Tags      PhoneRegisterTask
// @Summary   获取手机号注册任务统计
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     leaderId  query     int  false  "团长ID"
// @Success   200       {object}  response.Response{data=systemRes.PhoneRegisterTaskSummaryResponse,msg=string}
// @Router    /phoneRegisterTask/summary [get]
func (a *PhoneRegisterTaskApi) GetPhoneRegisterTaskSummary(c *gin.Context) {
	var req systemReq.PhoneRegisterTaskSummaryFilter
	_ = c.ShouldBindQuery(&req)
	data, err := phoneRegisterTaskService.GetSummary(utils.GetUserAuthorityId(c), utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

// SettlePhoneRegisterTaskLeader
// @Tags      PhoneRegisterTask
// @Summary   管理员结算团长手机号注册任务
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterTaskSettle  true  "结算参数"
// @Success   200   {object}  response.Response{msg=string}
// @Router    /phoneRegisterTask/settle [post]
func (a *PhoneRegisterTaskApi) SettlePhoneRegisterTaskLeader(c *gin.Context) {
	var req systemReq.PhoneRegisterTaskSettle
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := phoneRegisterTaskService.SettleLeader(utils.GetUserAuthorityId(c), utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{
		"settledAt":    result.SettledAt,
		"settledCount": result.SettledCount,
	}, "结算成功", c)
}

// GetPhoneRegisterTaskSettlementHistory
// @Tags      PhoneRegisterTask
// @Summary   管理员查询团长手机号注册任务结算历史
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     leaderId  query     int  true  "团长ID"
// @Success   200       {object}  response.Response{data=[]systemRes.PhoneRegisterTaskSettlementHistoryItem,msg=string}
// @Router    /phoneRegisterTask/settlement/history [get]
func (a *PhoneRegisterTaskApi) GetPhoneRegisterTaskSettlementHistory(c *gin.Context) {
	var req systemReq.PhoneRegisterTaskSettlementHistory
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	data, err := phoneRegisterTaskService.GetSettlementHistory(utils.GetUserAuthorityId(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

// GetPhoneRegisterTaskLogs
// @Tags      PhoneRegisterTask
// @Summary   查询手机号注册任务日志
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterTaskLogList  true  "查询参数"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterTaskLogListResponse,msg=string}
// @Router    /phoneRegisterTask/logs [post]
func (a *PhoneRegisterTaskApi) GetPhoneRegisterTaskLogs(c *gin.Context) {
	var req systemReq.PhoneRegisterTaskLogList
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	logs, total, page, pageSize, err := phoneRegisterTaskService.GetTaskLogs(utils.GetUserAuthorityId(c), utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.PhoneRegisterTaskLogListResponse{
		List:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

// DevicePollPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   设备拉取手机号注册任务
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterDevicePoll  true  "设备ID"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterDeviceTaskInfo,msg=string}
// @Router    /phoneRegisterTask/device/poll [post]
func (a *PhoneRegisterTaskApi) DevicePollPhoneRegisterTask(c *gin.Context) {
	var req systemReq.PhoneRegisterDevicePoll
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, found, err := phoneRegisterTaskService.DevicePoll(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildPhoneRegisterDeviceTaskInfo(task, found), "获取成功", c)
}

// DeviceGetPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   查询设备当前持有手机号注册任务
// @accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemRes.PhoneRegisterDeviceTaskInfo,msg=string}
// @Router    /phoneRegisterTask/device/task [get]
// @Router    /phoneRegisterTask/device/task [post]
func (a *PhoneRegisterTaskApi) DeviceGetPhoneRegisterTask(c *gin.Context) {
	var req systemReq.PhoneRegisterDeviceTask
	var err error
	if c.Request.Method == http.MethodGet {
		err = c.ShouldBindQuery(&req)
	} else {
		err = c.ShouldBindJSON(&req)
	}
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, found, err := phoneRegisterTaskService.DeviceTask(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildPhoneRegisterDeviceTaskInfo(task, found), "获取成功", c)
}

// DeviceHeartbeatPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   设备上报手机号注册任务心跳
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterDeviceHeartbeat  true  "设备ID"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterDeviceHeartbeatResponse,msg=string}
// @Router    /phoneRegisterTask/device/heartbeat [post]
func (a *PhoneRegisterTaskApi) DeviceHeartbeatPhoneRegisterTask(c *gin.Context) {
	var req systemReq.PhoneRegisterDeviceHeartbeat
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := phoneRegisterTaskService.DeviceHeartbeat(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.PhoneRegisterDeviceHeartbeatResponse{OK: true}, "心跳成功", c)
}

// DeviceReportPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   设备上报手机号注册任务进度
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterDeviceReport  true  "上报参数"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterDeviceHeartbeatResponse,msg=string}
// @Router    /phoneRegisterTask/device/report [post]
func (a *PhoneRegisterTaskApi) DeviceReportPhoneRegisterTask(c *gin.Context) {
	var req systemReq.PhoneRegisterDeviceReport
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if _, err := phoneRegisterTaskService.DeviceReport(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.PhoneRegisterDeviceHeartbeatResponse{OK: true}, "上报成功", c)
}

// DeviceLogPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   设备上报手机号注册任务日志
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterDeviceLog  true  "日志参数"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterDeviceHeartbeatResponse,msg=string}
// @Router    /phoneRegisterTask/device/log [post]
func (a *PhoneRegisterTaskApi) DeviceLogPhoneRegisterTask(c *gin.Context) {
	var req systemReq.PhoneRegisterDeviceLog
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := phoneRegisterTaskService.DeviceLog(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.PhoneRegisterDeviceHeartbeatResponse{OK: true}, "日志上报成功", c)
}

// DeviceGetPhoneRegisterConfig
// @Tags      PhoneRegisterTask
// @Summary   获取设备端手机号注册配置
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemRes.PhoneRegisterDeviceConfigResponse,msg=string}
// @Router    /phoneRegisterTask/device/config [get]
func (a *PhoneRegisterTaskApi) DeviceGetPhoneRegisterConfig(c *gin.Context) {
	data, err := phoneRegisterTaskService.GetDeviceConfig()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

func buildPhoneRegisterActiveInfo(task system.SysPhoneRegisterTask) systemRes.PhoneRegisterTaskActiveInfo {
	var codeSubmitExpiresAt *time.Time
	if task.SMSReceiveMode == system.PhoneRegisterSMSModePlatformSend &&
		task.Status == system.PhoneRegisterStatusWaitingPromoterCode &&
		task.CodeRequestedAt != nil {
		expiresAt := task.CodeRequestedAt.Add(2 * time.Minute)
		codeSubmitExpiresAt = &expiresAt
	}
	return systemRes.PhoneRegisterTaskActiveInfo{
		ID:                  task.ID,
		CreatedAt:           task.CreatedAt,
		Phone:               task.Phone,
		SMSReceiveMode:      task.SMSReceiveMode,
		TaskSource:          task.TaskSource,
		CacheStatus:         task.CacheStatus,
		Status:              task.Status,
		StatusCode:          task.StatusCode,
		LastError:           task.LastError,
		NeedPromoterCode:    task.SMSReceiveMode == system.PhoneRegisterSMSModePlatformSend && task.Status == system.PhoneRegisterStatusWaitingPromoterCode,
		CodeSubmitExpiresAt: codeSubmitExpiresAt,
		HolderDeviceID:      task.HolderDeviceID,
		ClaimedAt:           task.ClaimedAt,
		LastHeartbeatAt:     task.LastHeartbeatAt,
		ExpiresAt:           task.ExpiresAt,
		FinishedAt:          task.FinishedAt,
	}
}

func buildPhoneRegisterDeviceTaskInfo(task system.SysPhoneRegisterTask, found bool) systemRes.PhoneRegisterDeviceTaskInfo {
	if !found || task.ID == 0 {
		return systemRes.PhoneRegisterDeviceTaskInfo{TaskID: 0}
	}
	info := systemRes.PhoneRegisterDeviceTaskInfo{
		TaskID:           task.ID,
		Phone:            task.Phone,
		SMSReceiveMode:   task.SMSReceiveMode,
		TaskSource:       task.TaskSource,
		CacheStatus:      task.CacheStatus,
		Status:           task.Status,
		NeedPromoterCode: task.SMSReceiveMode == system.PhoneRegisterSMSModePlatformSend && task.Status == system.PhoneRegisterStatusWaitingPromoterCode,
		ClaimedAt:        task.ClaimedAt,
		LastHeartbeatAt:  task.LastHeartbeatAt,
	}
	expiresAt := task.ExpiresAt
	info.ExpiresAt = &expiresAt
	if strings.TrimSpace(task.PendingCode) != "" {
		info.VerifyCode = task.PendingCode
	}
	return info
}
