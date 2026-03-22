package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RegisterConfigApi struct{}

// GetMyRegisterConfig
// @Tags      RegisterConfig
// @Summary   获取我的注册配置
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=system.SysRegisterConfig,msg=string}
// @Router    /registerConfig/getMyConfig [get]
func (a *RegisterConfigApi) GetMyRegisterConfig(c *gin.Context) {
	data, err := registerConfigService.GetMyConfig(utils.GetUserAuthorityId(c), utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

// SetMyRegisterConfig
// @Tags      RegisterConfig
// @Summary   修改我的注册配置
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.RegisterConfigUpsert  true  "配置内容"
// @Success   200   {object}  response.Response{data=system.SysRegisterConfig,msg=string}
// @Router    /registerConfig/setMyConfig [put]
func (a *RegisterConfigApi) SetMyRegisterConfig(c *gin.Context) {
	var req systemReq.RegisterConfigUpsert
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	data, err := registerConfigService.UpsertMyConfig(utils.GetUserAuthorityId(c), utils.GetUserID(c), req)
	if err != nil {
		global.GVA_LOG.Error("保存注册配置失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(data, "保存成功", c)
}

// CheckMyRegisterConfig
// @Tags      RegisterConfig
// @Summary   检测我的注册配置连通性
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=map[string]interface{},msg=string}
// @Router    /registerConfig/checkMyConfig [get]
func (a *RegisterConfigApi) CheckMyRegisterConfig(c *gin.Context) {
	data, err := registerConfigService.CheckMyConfig(utils.GetUserAuthorityId(c), utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("检测注册配置失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(data, "检测完成", c)
}
