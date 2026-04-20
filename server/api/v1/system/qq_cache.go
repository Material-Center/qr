package system

import (
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

type QQCacheApi struct{}

const (
	qqCacheRoleSuperAdmin = uint(888)
	qqCacheRoleAdmin      = uint(100)
	qqCacheRoleAppExtract = uint(400)
	qqCacheRoleAppUpload  = uint(500)
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
	response.OkWithDetailed(record, "上传成功", c)
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
	}, "获取成功", c)
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
