package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type RegisterTaskRouter struct{}

func (r *RegisterTaskRouter) InitRegisterTaskRouter(Router *gin.RouterGroup) {
	registerTaskRouter := Router.Group("registerTask").Use(middleware.OperationRecord())
	registerTaskRouterWithoutRecord := Router.Group("registerTask")
	{
		registerTaskRouter.POST("create", registerTaskApi.CreateRegisterTask) // 地推创建任务
		registerTaskRouter.POST("step", registerTaskApi.SubmitRegisterTaskStep)
	}
	{
		registerTaskRouterWithoutRecord.GET("active", registerTaskApi.GetActiveRegisterTask)
		registerTaskRouterWithoutRecord.POST("list", registerTaskApi.GetRegisterTaskList)
		registerTaskRouterWithoutRecord.GET("summary", registerTaskApi.GetRegisterTaskSummary)
	}
}
