package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

type DeviceConfigApi struct{}

func (a *DeviceConfigApi) List(c *gin.Context) {
	if !isQQCacheAdminRole(utils.GetUserAuthorityId(c)) {
		response.FailWithMessage("仅管理员可管理设备配置", c)
		return
	}
	var req systemReq.DeviceConfigList
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := deviceConfigService.ListDeviceConfig(req)
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
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func (a *DeviceConfigApi) Save(c *gin.Context) {
	if !isQQCacheAdminRole(utils.GetUserAuthorityId(c)) {
		response.FailWithMessage("仅管理员可管理设备配置", c)
		return
	}
	var req systemReq.DeviceConfigSave
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	record, err := deviceConfigService.SaveDeviceConfig(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(record, "保存成功", c)
}

func (a *DeviceConfigApi) Delete(c *gin.Context) {
	if !isQQCacheAdminRole(utils.GetUserAuthorityId(c)) {
		response.FailWithMessage("仅管理员可管理设备配置", c)
		return
	}
	var req systemReq.DeviceConfigDelete
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := deviceConfigService.DeleteDeviceConfig(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}
