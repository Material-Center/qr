package system

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/gin-gonic/gin"
)

const (
	phoneRegisterOpenAPIStatusSuccess = "success"
	phoneRegisterOpenAPIStatusFailed  = "failed"
)

var phoneRegisterOpenAPIKeys = []string{
	"pr_openapi_4be963e4c074492ecfbd563d4445609e34e1067222831d03",
}

// OpenAPIPollPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   OpenAPI获取手机号注册任务
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterOpenAPITask  true  "设备ID"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterOpenAPITaskInfo,msg=string}
// @Router    /phoneRegisterTask/open-api/task [post]
func (a *PhoneRegisterTaskApi) OpenAPIPollPhoneRegisterTask(c *gin.Context) {
	if !requirePhoneRegisterOpenAPIKey(c) {
		return
	}
	var req systemReq.PhoneRegisterOpenAPITask
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, found, err := phoneRegisterTaskService.OpenAPIPoll(req, c.Query("verifyMode"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	refreshPhoneRegisterOpenAPIHeartbeat(req.DeviceID, found)
	response.OkWithDetailed(buildPhoneRegisterOpenAPITaskInfo(task, found), "获取成功", c)
}

// OpenAPIGetPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   OpenAPI查询当前手机号注册任务
// @accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemRes.PhoneRegisterOpenAPITaskInfo,msg=string}
// @Router    /phoneRegisterTask/open-api/task [get]
func (a *PhoneRegisterTaskApi) OpenAPIGetPhoneRegisterTask(c *gin.Context) {
	if !requirePhoneRegisterOpenAPIKey(c) {
		return
	}
	var req systemReq.PhoneRegisterOpenAPITask
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
	task, found, err := phoneRegisterTaskService.DeviceTask(systemReq.PhoneRegisterDeviceTask{DeviceID: req.DeviceID})
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	refreshPhoneRegisterOpenAPIHeartbeat(req.DeviceID, found)
	response.OkWithDetailed(buildPhoneRegisterOpenAPITaskInfo(task, found), "获取成功", c)
}

// OpenAPIGetVerifyCode
// @Tags      PhoneRegisterTask
// @Summary   OpenAPI获取手机号注册验证码
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterOpenAPITask  true  "设备ID和任务ID"
// @Success   200   {object}  response.Response{data=systemRes.PhoneRegisterOpenAPIVerifyCodeResponse,msg=string}
// @Router    /phoneRegisterTask/open-api/verify-code [post]
func (a *PhoneRegisterTaskApi) OpenAPIGetVerifyCode(c *gin.Context) {
	if !requirePhoneRegisterOpenAPIKey(c) {
		return
	}
	var req systemReq.PhoneRegisterOpenAPITask
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, err := getAndValidatePhoneRegisterOpenAPITask(req.DeviceID, req.TaskID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if task.SMSReceiveMode != system.PhoneRegisterSMSModePlatformSend {
		response.FailWithMessage("当前任务验证方式不需要获取验证码", c)
		return
	}
	task, err = phoneRegisterTaskService.DeviceReport(systemReq.PhoneRegisterDeviceReport{
		DeviceID: req.DeviceID,
		Action:   system.PhoneRegisterDeviceActionEnterWaitingCode,
		Message:  "等待输入验证码",
	})
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	verifyCode := strings.TrimSpace(task.PendingCode)
	response.OkWithDetailed(systemRes.PhoneRegisterOpenAPIVerifyCodeResponse{
		TaskID:     task.ID,
		VerifyCode: verifyCode,
		HasCode:    verifyCode != "",
	}, "获取成功", c)
}

// OpenAPIReportPhoneRegisterTask
// @Tags      PhoneRegisterTask
// @Summary   OpenAPI上报手机号注册任务结果
// @accept    multipart/form-data
// @Produce   application/json
// @Param     deviceId  formData  string  true   "设备ID"
// @Param     taskId    formData  int     false  "任务ID"
// @Param     status    formData  string  true   "success/failed"
// @Param     reason    formData  string  false  "失败原因"
// @Success   200       {object}  response.Response{data=systemRes.PhoneRegisterOpenAPIReportResponse,msg=string}
// @Router    /phoneRegisterTask/open-api/report [post]
func (a *PhoneRegisterTaskApi) OpenAPIReportPhoneRegisterTask(c *gin.Context) {
	if !requirePhoneRegisterOpenAPIKey(c) {
		return
	}
	req, err := bindPhoneRegisterOpenAPIReport(c)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	switch strings.ToLower(strings.TrimSpace(req.Status)) {
	case phoneRegisterOpenAPIStatusFailed:
		currentTask, err := getAndValidatePhoneRegisterOpenAPITask(req.DeviceID, req.TaskID)
		if err != nil {
			response.FailWithMessage(err.Error(), c)
			return
		}
		task, err := phoneRegisterTaskService.DeviceReport(systemReq.PhoneRegisterDeviceReport{
			DeviceID:   req.DeviceID,
			Action:     system.PhoneRegisterDeviceActionFail,
			Message:    req.Reason,
			StatusCode: intPtr(system.PhoneRegisterStatusCodeDeviceExecFail),
		})
		if err != nil {
			response.FailWithMessage(err.Error(), c)
			return
		}
		if task.ID == 0 {
			task.ID = currentTask.ID
		}
		response.OkWithDetailed(systemRes.PhoneRegisterOpenAPIReportResponse{OK: true, TaskID: task.ID}, "上报成功", c)
	case phoneRegisterOpenAPIStatusSuccess:
		currentTask, err := getAndValidatePhoneRegisterOpenAPITask(req.DeviceID, req.TaskID)
		if err != nil {
			response.FailWithMessage(err.Error(), c)
			return
		}
		task, err := phoneRegisterTaskService.DeviceReport(systemReq.PhoneRegisterDeviceReport{
			DeviceID: req.DeviceID,
			Action:   system.PhoneRegisterDeviceActionRegisterSuccess,
			Message:  "注册成功，等待上传缓存",
		})
		if err != nil {
			response.FailWithMessage(err.Error(), c)
			return
		}
		if task.ID == 0 {
			task.ID = currentTask.ID
		}
		response.OkWithDetailed(systemRes.PhoneRegisterOpenAPIReportResponse{OK: true, TaskID: task.ID}, "上报成功", c)
	default:
		response.FailWithMessage("status仅支持success/failed", c)
	}
}

// OpenAPIUploadPhoneRegisterCache
// @Tags      PhoneRegisterTask
// @Summary   OpenAPI上传手机号注册缓存
// @accept    multipart/form-data
// @Produce   application/json
// @Param     deviceId    formData  string  true   "设备ID"
// @Param     taskId      formData  int     false  "任务ID"
// @Param     qqPwd       formData  string  false  "QQ密码"
// @Param     clientId    formData  string  false  "Android ID"
// @Param     deviceInfo  formData  string  false  "设备信息"
// @Param     cacheZip    formData  file    true   "缓存zip"
// @Success   200         {object}  response.Response{data=systemRes.PhoneRegisterOpenAPIReportResponse,msg=string}
// @Router    /phoneRegisterTask/open-api/cache [post]
func (a *PhoneRegisterTaskApi) OpenAPIUploadPhoneRegisterCache(c *gin.Context) {
	if !requirePhoneRegisterOpenAPIKey(c) {
		return
	}
	req, err := bindPhoneRegisterOpenAPICacheUpload(c)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	data, err := extractPhoneRegisterZipFromRequest(c, req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	record, task, err := qqCacheService.UploadPhoneRegister(systemReq.QQCacheUpload{
		Phone:    data.Phone,
		QQNum:    data.QQNum,
		QQPwd:    req.QQPwd,
		INI:      data.INI,
		DeviceID: req.DeviceID,
	})
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.PhoneRegisterOpenAPIReportResponse{
		OK:              true,
		TaskID:          task.ID,
		QQCacheRecordID: record.ID,
		QQNum:           record.QQNum,
	}, "上传成功", c)
}

type phoneRegisterOpenAPIExtractedCache struct {
	Phone string
	QQNum string
	INI   string
}

type phoneRegisterExtractorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Records []struct {
		QQ         string `json:"qq"`
		INIContent string `json:"iniContent"`
	} `json:"records"`
	Errors []string `json:"errors"`
}

func requirePhoneRegisterOpenAPIKey(c *gin.Context) bool {
	apiKey := strings.TrimSpace(c.GetHeader("X-Open-Api-Key"))
	if apiKey == "" {
		response.NoAuth("缺少X-Open-Api-Key", c)
		c.Abort()
		return false
	}
	if !isValidPhoneRegisterOpenAPIKey(apiKey) {
		response.NoAuth("X-Open-Api-Key无效", c)
		c.Abort()
		return false
	}
	return true
}

func isValidPhoneRegisterOpenAPIKey(apiKey string) bool {
	for _, allowed := range phoneRegisterOpenAPIKeys {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" || len(apiKey) != len(allowed) {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(allowed)) == 1 {
			return true
		}
	}
	return false
}

func bindPhoneRegisterOpenAPIReport(c *gin.Context) (systemReq.PhoneRegisterOpenAPIReport, error) {
	var req systemReq.PhoneRegisterOpenAPIReport
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.Contains(contentType, "multipart/form-data") {
		req.DeviceID = strings.TrimSpace(c.PostForm("deviceId"))
		req.Status = strings.TrimSpace(c.PostForm("status"))
		req.Reason = strings.TrimSpace(c.PostForm("reason"))
		req.QQPwd = strings.TrimSpace(c.PostForm("qqPwd"))
		req.ClientID = strings.TrimSpace(c.PostForm("clientId"))
		req.DeviceInfo = strings.TrimSpace(c.PostForm("deviceInfo"))
		if rawTaskID := strings.TrimSpace(c.PostForm("taskId")); rawTaskID != "" {
			_, _ = fmt.Sscanf(rawTaskID, "%d", &req.TaskID)
		}
	} else if err := c.ShouldBindJSON(&req); err != nil {
		return req, err
	}
	if strings.TrimSpace(req.DeviceID) == "" {
		return req, errors.New("deviceId不能为空")
	}
	if strings.TrimSpace(req.Status) == "" {
		return req, errors.New("status不能为空")
	}
	if strings.EqualFold(req.Status, phoneRegisterOpenAPIStatusFailed) && strings.TrimSpace(req.Reason) == "" {
		req.Reason = "任务失败"
	}
	return req, nil
}

func bindPhoneRegisterOpenAPICacheUpload(c *gin.Context) (systemReq.PhoneRegisterOpenAPIReport, error) {
	var req systemReq.PhoneRegisterOpenAPIReport
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if !strings.Contains(contentType, "multipart/form-data") {
		return req, errors.New("缓存上传仅支持multipart/form-data")
	}
	req.DeviceID = strings.TrimSpace(c.PostForm("deviceId"))
	req.QQPwd = strings.TrimSpace(c.PostForm("qqPwd"))
	req.ClientID = strings.TrimSpace(c.PostForm("clientId"))
	req.DeviceInfo = strings.TrimSpace(c.PostForm("deviceInfo"))
	if rawTaskID := strings.TrimSpace(c.PostForm("taskId")); rawTaskID != "" {
		_, _ = fmt.Sscanf(rawTaskID, "%d", &req.TaskID)
	}
	if req.DeviceID == "" {
		return req, errors.New("deviceId不能为空")
	}
	return req, nil
}

func extractPhoneRegisterZipFromRequest(c *gin.Context, req systemReq.PhoneRegisterOpenAPIReport) (phoneRegisterOpenAPIExtractedCache, error) {
	task, err := getAndValidatePhoneRegisterOpenAPITask(req.DeviceID, req.TaskID)
	if err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, err
	}
	if task.Status != system.PhoneRegisterStatusRegisteredWaitUpload {
		return phoneRegisterOpenAPIExtractedCache{}, errors.New("当前任务未处于待上传缓存状态")
	}
	file, header, err := c.Request.FormFile("cacheZip")
	if err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, errors.New("缓存上传需要上传cacheZip")
	}
	defer file.Close()
	if header == nil || header.Size == 0 {
		return phoneRegisterOpenAPIExtractedCache{}, errors.New("cacheZip不能为空")
	}
	zipBytes, err := io.ReadAll(file)
	if err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, err
	}
	if len(zipBytes) == 0 {
		return phoneRegisterOpenAPIExtractedCache{}, errors.New("cacheZip不能为空")
	}
	result, err := callQQCacheExtractor(zipBytes, req.ClientID, req.DeviceInfo)
	if err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, err
	}
	return phoneRegisterOpenAPIExtractedCache{
		Phone: task.Phone,
		QQNum: result.QQNum,
		INI:   result.INI,
	}, nil
}

func callQQCacheExtractor(zipBytes []byte, clientID string, deviceInfo string) (phoneRegisterOpenAPIExtractedCache, error) {
	extractorURL := strings.TrimSpace(global.GVA_CONFIG.Extra.ExtractURL)
	if extractorURL == "" {
		extractorURL = strings.TrimSpace(os.Getenv("QQ_CACHE_EXTRACTOR_URL"))
	}
	if extractorURL == "" {
		extractorURL = "http://127.0.0.1:19091/extract"
	}
	extractorURL = appendQQCacheExtractorQuery(extractorURL, clientID, deviceInfo)

	httpClient := &http.Client{Timeout: 2 * time.Minute}
	httpReq, err := http.NewRequest(http.MethodPost, extractorURL, bytes.NewReader(zipBytes))
	if err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, err
	}
	httpReq.ContentLength = int64(len(zipBytes))
	httpReq.Header.Set("Content-Type", "application/zip")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, fmt.Errorf("调用缓存提取服务失败: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return phoneRegisterOpenAPIExtractedCache{}, fmt.Errorf("缓存提取服务返回异常: %s", strings.TrimSpace(string(respBytes)))
	}
	var parsed phoneRegisterExtractorResponse
	if err = json.Unmarshal(respBytes, &parsed); err != nil {
		return phoneRegisterOpenAPIExtractedCache{}, fmt.Errorf("解析缓存提取服务响应失败: %w", err)
	}
	if len(parsed.Records) == 0 {
		if parsed.Message != "" {
			return phoneRegisterOpenAPIExtractedCache{}, errors.New(parsed.Message)
		}
		if len(parsed.Errors) > 0 {
			return phoneRegisterOpenAPIExtractedCache{}, errors.New(strings.Join(parsed.Errors, "; "))
		}
		return phoneRegisterOpenAPIExtractedCache{}, errors.New("缓存提取服务未返回账号")
	}
	record := parsed.Records[0]
	if strings.TrimSpace(record.QQ) == "" || strings.TrimSpace(record.INIContent) == "" {
		return phoneRegisterOpenAPIExtractedCache{}, errors.New("缓存提取结果缺少qq或ini")
	}
	return phoneRegisterOpenAPIExtractedCache{
		QQNum: strings.TrimSpace(record.QQ),
		INI:   strings.TrimSpace(record.INIContent),
	}, nil
}

func appendQQCacheExtractorQuery(rawURL string, clientID string, deviceInfo string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	if strings.TrimSpace(clientID) != "" {
		query.Set("clientId", strings.TrimSpace(clientID))
	}
	if strings.TrimSpace(deviceInfo) != "" {
		query.Set("deviceInfo", strings.TrimSpace(deviceInfo))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func getAndValidatePhoneRegisterOpenAPITask(deviceID string, taskID uint) (system.SysPhoneRegisterTask, error) {
	task, found, err := phoneRegisterTaskService.DeviceTask(systemReq.PhoneRegisterDeviceTask{DeviceID: deviceID})
	if err != nil {
		return system.SysPhoneRegisterTask{}, err
	}
	if !found {
		return system.SysPhoneRegisterTask{}, errors.New("当前设备暂无执行中任务")
	}
	if taskID != 0 && taskID != task.ID {
		return system.SysPhoneRegisterTask{}, errors.New("taskId与当前设备任务不一致")
	}
	return task, nil
}

func refreshPhoneRegisterOpenAPIHeartbeat(deviceID string, found bool) {
	if !found {
		return
	}
	_ = phoneRegisterTaskService.DeviceHeartbeat(systemReq.PhoneRegisterDeviceHeartbeat{DeviceID: deviceID})
}

func buildPhoneRegisterOpenAPITaskInfo(task system.SysPhoneRegisterTask, found bool) systemRes.PhoneRegisterOpenAPITaskInfo {
	if !found || task.ID == 0 {
		return systemRes.PhoneRegisterOpenAPITaskInfo{HasTask: false}
	}
	info := systemRes.PhoneRegisterOpenAPITaskInfo{
		TaskID:     task.ID,
		Phone:      task.Phone,
		VerifyMode: phoneRegisterOpenAPIVerifyMode(task.SMSReceiveMode),
		Status:     task.Status,
		HasTask:    true,
		NeedCode:   task.SMSReceiveMode == system.PhoneRegisterSMSModePlatformSend,
	}
	expiresAt := task.ExpiresAt
	info.ExpiresAt = &expiresAt
	return info
}

func phoneRegisterOpenAPIVerifyMode(smsReceiveMode string) string {
	switch smsReceiveMode {
	case system.PhoneRegisterSMSModePlatformSend:
		return "receive"
	case system.PhoneRegisterSMSModeUserSentToTX:
		return "send"
	default:
		return strings.TrimSpace(smsReceiveMode)
	}
}

func intPtr(v int) *int {
	return &v
}
