package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type DeviceConfigRouter struct{}

func (r *DeviceConfigRouter) InitDeviceConfigRouter(Router *gin.RouterGroup) {
	deviceConfigRouter := Router.Group("deviceConfig").Use(middleware.OperationRecord())
	{
		deviceConfigRouter.POST("list", deviceConfigApi.List)
		deviceConfigRouter.POST("save", deviceConfigApi.Save)
		deviceConfigRouter.POST("delete", deviceConfigApi.Delete)
		deviceConfigRouter.GET("group/list", deviceConfigApi.GroupList)
		deviceConfigRouter.POST("group/save", deviceConfigApi.GroupSave)
		deviceConfigRouter.POST("group/delete", deviceConfigApi.GroupDelete)
	}
}
