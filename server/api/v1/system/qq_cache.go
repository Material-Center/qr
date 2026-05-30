package system

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

type QQCacheApi struct{}

const (
	qqCacheRoleSuperAdmin       = uint(888)
	qqCacheRoleAdmin            = uint(100)
	qqCacheRoleAppExtract       = uint(400)
	qqCacheRoleAppUpload        = uint(500)
	qqCacheExportQQFileMaxBytes = 512 * 1024
	qqCacheInternalToolZipBytes = 500 * 1024
)

// Upload
// @Tags      QQCache
// @Summary   App上传QQ缓存
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.QQCacheUpload  true  "上传缓存参数"
// @Success   200   {object}  response.Response
// @Router    /qqCache/upload [post]
func (a *QQCacheApi) Upload(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAppUpload {
		response.FailWithMessage("仅App上传角色可上传缓存", c)
		return
	}
	var req systemReq.QQCacheUpload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	record, err := qqCacheService.UploadByApp(utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(record.QQNum, "上传成功", c)
}

// UploadPhoneRegister
// @Tags      QQCache
// @Summary   手机号注册任务上传QQ缓存并完成任务
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.QQCacheUpload  true  "上传缓存参数"
// @Success   200   {object}  response.Response{data=systemRes.QQCacheUploadPhoneRegisterResponse,msg=string}
// @Router    /qqCache/uploadPhoneRegister [post]
func (a *QQCacheApi) UploadPhoneRegister(c *gin.Context) {
	var req systemReq.QQCacheUpload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	record, _, err := qqCacheService.UploadPhoneRegister(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.QQCacheUploadPhoneRegisterResponse{
		QQCacheRecordID: record.ID,
		QQNum:           record.QQNum,
	}, "上传成功", c)
}

// InternalToolImportQQCacheZip
// @Tags      QQCache
// @Summary   内部工具导入QQ缓存zip
// @accept    multipart/form-data
// @Produce   application/json
// @Param     qqNum     formData  string  true   "QQ账号"
// @Param     qqPwd     formData  string  false  "QQ密码"
// @Param     force     formData  bool    false  "是否强制覆盖缓存字段"
// @Param     cacheZip  formData  file    true   "缓存zip"
// @Success   200       {object}  response.Response{data=systemRes.InternalToolQQCacheImportResponse,msg=string}
// @Router    /internalTool/qqCache/importZip [post]
func (a *QQCacheApi) InternalToolImportQQCacheZip(c *gin.Context) {
	force, err := strconv.ParseBool(strings.TrimSpace(c.PostForm("force")))
	if err != nil && strings.TrimSpace(c.PostForm("force")) != "" {
		response.FailWithMessage("force参数格式错误", c)
		return
	}
	a.importQQCacheZip(c, force, true)
}

// AdminImportQQCacheZip
// @Tags      QQCache
// @Summary   管理端导入QQ缓存zip
// @Security  ApiKeyAuth
// @accept    multipart/form-data
// @Produce   application/json
// @Param     cacheZip  formData  file  true  "缓存zip"
// @Success   200       {object}  response.Response{data=systemRes.InternalToolQQCacheImportResponse,msg=string}
// @Router    /qqCache/importZip [post]
func (a *QQCacheApi) AdminImportQQCacheZip(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可导入缓存", c)
		return
	}
	a.importQQCacheZip(c, true, false)
}

func (a *QQCacheApi) importQQCacheZip(c *gin.Context, force bool, skipExistingBeforeExtract bool) {
	qqNum := strings.TrimSpace(c.PostForm("qqNum"))
	if skipExistingBeforeExtract && !force && qqNum != "" {
		record, found, err := qqCacheService.InternalToolFindQQCacheByQQNum(qqNum)
		if err != nil {
			response.FailWithMessage(err.Error(), c)
			return
		}
		if found {
			response.OkWithDetailed(systemRes.InternalToolQQCacheImportResponse{
				QQCacheRecordID: record.ID,
				QQNum:           record.QQNum,
				Action:          "skipped",
				Force:           false,
			}, "导入成功", c)
			return
		}
	}

	zipBytes, err := readQQCacheInternalToolZip(c)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	extracted, err := callQQCacheExtractor(zipBytes, c.PostForm("clientId"), c.PostForm("deviceInfo"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if qqNum == "" {
		qqNum = strings.TrimSpace(extracted.QQNum)
	}
	if extracted.QQNum != "" && qqNum != "" && strings.TrimSpace(extracted.QQNum) != qqNum {
		response.FailWithMessage("zip内账号与请求账号不一致", c)
		return
	}
	req := systemReq.InternalToolQQCacheImport{
		QQNum:    qqNum,
		QQPwd:    c.PostForm("qqPwd"),
		INI:      extracted.INI,
		DeviceID: c.PostForm("deviceId"),
		Force:    force,
	}
	var (
		record systemModel.SysQQCacheRecord
		action string
	)
	if force {
		record, action, err = qqCacheService.AdminImportQQCache(req)
	} else {
		record, action, err = qqCacheService.InternalToolImportQQCache(req)
	}
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(systemRes.InternalToolQQCacheImportResponse{
		QQCacheRecordID: record.ID,
		QQNum:           record.QQNum,
		Action:          action,
		Force:           force,
	}, "导入成功", c)
}

// InternalToolCheckQQCache
// @Tags      QQCache
// @Summary   内部工具检查QQ缓存是否存在
// @accept    application/json
// @Produce   application/json
// @Param     qqNum  query  string  true  "QQ账号"
// @Success   200    {object}  response.Response{data=systemRes.InternalToolQQCacheExistsResponse,msg=string}
// @Router    /internalTool/qqCache/exists [get]
func (a *QQCacheApi) InternalToolCheckQQCache(c *gin.Context) {
	record, found, err := qqCacheService.InternalToolFindQQCacheByQQNum(c.Query("qqNum"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	resp := systemRes.InternalToolQQCacheExistsResponse{Exists: found}
	if found {
		resp.QQCacheRecordID = record.ID
		resp.QQNum = record.QQNum
	}
	response.OkWithDetailed(resp, "获取成功", c)
}

func readQQCacheInternalToolZip(c *gin.Context) ([]byte, error) {
	file, header, err := c.Request.FormFile("cacheZip")
	if err != nil {
		return nil, errors.New("cacheZip不能为空")
	}
	defer file.Close()
	if header != nil && header.Size > qqCacheInternalToolZipBytes {
		return nil, errors.New("cacheZip不能超过500K")
	}
	zipBytes, err := io.ReadAll(io.LimitReader(file, qqCacheInternalToolZipBytes+1))
	if err != nil {
		return nil, err
	}
	if len(zipBytes) == 0 {
		return nil, errors.New("cacheZip不能为空")
	}
	if len(zipBytes) > qqCacheInternalToolZipBytes {
		return nil, errors.New("cacheZip不能超过500K")
	}
	return zipBytes, nil
}

// Extract
// @Tags      QQCache
// @Summary   App提取QQ缓存
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.QQCacheExtract  true  "提取缓存参数"
// @Success   200   {object}  response.Response
// @Router    /qqCache/extract [post]
func (a *QQCacheApi) Extract(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAppExtract {
		response.FailWithMessage("仅App提取角色可提取缓存", c)
		return
	}
	var req systemReq.QQCacheExtract
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	record, err := qqCacheService.ExtractByApp(utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(record, "提取成功", c)
}

// List
// @Tags      QQCache
// @Summary   管理端分页查询QQ缓存
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.QQCacheList  true  "查询参数"
// @Success   200   {object}  response.Response{data=systemRes.QQCacheListResponse}
// @Router    /qqCache/list [post]
func (a *QQCacheApi) List(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可查看缓存管理", c)
		return
	}
	var req systemReq.QQCacheList
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := qqCacheService.ListForAdmin(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	pending, extracted, statsTotal, err := qqCacheService.CountExtractStatsByCreatedRange(req.CreatedAtStart, req.CreatedAtEnd)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	billingUnsettled, billingSettled, err := qqCacheService.CountBillingSettlementStats()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 10
	}
	response.OkWithDetailed(systemRes.QQCacheListResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Stats: systemRes.QQCacheExtractStats{
			Pending:          pending,
			Extracted:        extracted,
			Total:            statsTotal,
			BillingUnsettled: billingUnsettled,
			BillingSettled:   billingSettled,
		},
	}, "获取成功", c)
}

// SettleBilling
// @Tags      QQCache
// @Summary   管理端结算QQ缓存计费数量
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200   {object}  response.Response
// @Router    /qqCache/billing/settle [post]
func (a *QQCacheApi) SettleBilling(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可结算", c)
		return
	}
	result, err := qqCacheService.SettleBilling(role, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{
		"settledAt":    result.SettledAt,
		"settledCount": result.SettledCount,
	}, "结算成功", c)
}

// GetBillingSettlementHistory
// @Tags      QQCache
// @Summary   管理端查询QQ缓存计费结算历史
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200   {object}  response.Response
// @Router    /qqCache/billing/history [get]
func (a *QQCacheApi) GetBillingSettlementHistory(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可查看结算历史", c)
		return
	}
	rows, err := qqCacheService.GetBillingSettlementHistory(role)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(rows, "获取成功", c)
}

// ResetExtract
// @Tags      QQCache
// @Summary   管理端重置提取锁
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.QQCacheResetExtract  true  "记录ID"
// @Success   200   {object}  response.Response
// @Router    /qqCache/resetExtract [post]
func (a *QQCacheApi) ResetExtract(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可重置提取状态", c)
		return
	}
	var req systemReq.QQCacheResetExtract
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := qqCacheService.ResetExtractByID(req.ID); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("重置成功", c)
}

// ExportIniZip
// @Tags      QQCache
// @Summary   管理端批量导出缓存 INI（zip）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/zip
// @Param     data  body      systemReq.QQCacheExportIniZip  true  "记录 ID 列表"
// @Success   200   file      zip
// @Router    /qqCache/exportIniZip [post]
func (a *QQCacheApi) ExportIniZip(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可导出缓存", c)
		return
	}
	var req systemReq.QQCacheExportIniZip
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	zipBytes, count, err := qqCacheService.ExportIniZipByIDs(req.IDs)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	filename := fmt.Sprintf("qq_cache_ini_%d_%s.zip", count, time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(200, "application/zip", zipBytes)
}

// ExportPendingIniZip
// @Tags      QQCache
// @Summary   管理端按数量提取未提取缓存 INI（zip）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/zip
// @Param     data  body      systemReq.QQCacheExportPendingIniZip  true  "提取数量"
// @Success   200   file      zip
// @Router    /qqCache/exportPendingIniZip [post]
func (a *QQCacheApi) ExportPendingIniZip(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可提取缓存", c)
		return
	}
	var req systemReq.QQCacheExportPendingIniZip
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	zipBytes, count, err := qqCacheService.ExportPendingIniZipByCount(req.Count, utils.GetUserID(c), req.CreatedAtStart, req.CreatedAtEnd)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	filename := qqCacheExtractZipFilename(count)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", qqCacheAttachmentDisposition(filename))
	c.Data(200, "application/zip", zipBytes)
}

// ExportAccountList
// @Tags      QQCache
// @Summary   管理端导出QQ账号列表（txt）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   text/plain
// @Param     data  body      systemReq.QQCacheExportAccountList  true  "筛选条件或记录ID"
// @Success   200   file      txt
// @Router    /qqCache/exportAccountList [post]
func (a *QQCacheApi) ExportAccountList(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可导出账号列表", c)
		return
	}
	var req systemReq.QQCacheExportAccountList
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	text, count, err := qqCacheService.ExportAccountListText(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	filename := fmt.Sprintf("qq_account_list_%d_%s.txt", count, time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(200, "text/plain; charset=utf-8", []byte(text))
}

// ExportIniZipByQQFile
// @Tags      QQCache
// @Summary   管理端按上传TXT内QQ账号导出缓存 INI（zip）
// @Security  ApiKeyAuth
// @accept    multipart/form-data
// @Produce   application/zip
// @Param     qqFile         formData  file  true   "TXT文件，每行格式：QQ----状态"
// @Param     markExtracted  formData  bool  false  "是否标记为已提取"
// @Success   200     file      zip
// @Router    /qqCache/exportIniZipByQQFile [post]
func (a *QQCacheApi) ExportIniZipByQQFile(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	if role != qqCacheRoleAdmin && role != qqCacheRoleSuperAdmin {
		response.FailWithMessage("仅管理员可导出缓存", c)
		return
	}
	markExtractedText := strings.TrimSpace(c.PostForm("markExtracted"))
	if markExtractedText == "" {
		markExtractedText = strings.TrimSpace(c.Query("markExtracted"))
	}
	markExtracted, err := strconv.ParseBool(markExtractedText)
	if err != nil && markExtractedText != "" {
		response.FailWithMessage("markExtracted参数格式错误", c)
		return
	}
	file, _, err := c.Request.FormFile("qqFile")
	if err != nil {
		response.FailWithMessage("请上传TXT文件", c)
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, qqCacheExportQQFileMaxBytes+1))
	if err != nil {
		response.FailWithMessage("读取TXT文件失败", c)
		return
	}
	if len(raw) > qqCacheExportQQFileMaxBytes {
		response.FailWithMessage("TXT文件不能超过512KB", c)
		return
	}
	var zipBytes []byte
	var count int
	if markExtracted {
		zipBytes, count, err = qqCacheService.ExportIniZipByQQTextAndMarkExtracted(string(raw), utils.GetUserID(c))
	} else {
		zipBytes, count, err = qqCacheService.ExportIniZipByQQText(string(raw))
	}
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	filename := fmt.Sprintf("qq_cache_ini_%d_%s.zip", count, time.Now().Format("20060102_150405"))
	if markExtracted {
		filename = qqCacheExtractZipFilename(count)
	}
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	if markExtracted {
		c.Header("Content-Disposition", qqCacheAttachmentDisposition(filename))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	}
	c.Data(200, "application/zip", zipBytes)
}

func qqCacheExtractZipFilename(count int) string {
	return fmt.Sprintf("qq-%d个-%s.zip", count, time.Now().Format("20060102-1504"))
}

func qqCacheAttachmentDisposition(filename string) string {
	fallback := strings.NewReplacer("个", "").Replace(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fallback, url.PathEscape(filename))
}

// AppLoginRoleHint
// 给App侧快速识别角色提示，避免误用账号
// @Tags      QQCache
// @Summary   获取App角色提示
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=map[string]any}
// @Router    /qqCache/roleHint [get]
func (a *QQCacheApi) AppLoginRoleHint(c *gin.Context) {
	role := utils.GetUserAuthorityId(c)
	label := "未知角色"
	switch role {
	case qqCacheRoleAppExtract:
		label = "App提取"
	case qqCacheRoleAppUpload:
		label = "App上传"
	case qqCacheRoleAdmin, qqCacheRoleSuperAdmin:
		label = "管理角色"
	}
	allowExtract := role == qqCacheRoleAppExtract
	allowUpload := role == qqCacheRoleAppUpload
	if strings.TrimSpace(label) == "" {
		response.FailWithMessage("角色无效", c)
		return
	}
	response.OkWithDetailed(gin.H{
		"authorityId":   role,
		"authorityName": label,
		"allowExtract":  allowExtract,
		"allowUpload":   allowUpload,
	}, "获取成功", c)
}

func normalizeQQNum(raw string) (string, error) {
	qq := strings.TrimSpace(raw)
	if qq == "" {
		return "", errors.New("qq账号不能为空")
	}
	return qq, nil
}
