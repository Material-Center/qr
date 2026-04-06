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
		registerTaskRouter.POST("debug/login/start", registerTaskApi.StartRegisterTaskDebugLogin)
		registerTaskRouter.POST("debug/login/submit", registerTaskApi.SubmitRegisterTaskDebugLoginCode)
	}
	{
		registerTaskRouterWithoutRecord.GET("active", registerTaskApi.GetActiveRegisterTask)
		registerTaskRouterWithoutRecord.GET("actives", registerTaskApi.GetActiveRegisterTasks)
		registerTaskRouterWithoutRecord.POST("list", registerTaskApi.GetRegisterTaskList)
		registerTaskRouterWithoutRecord.GET("summary", registerTaskApi.GetRegisterTaskSummary)
		registerTaskRouterWithoutRecord.GET("debug/login/task", registerTaskApi.GetRegisterTaskDebugLoginTask)
		registerTaskRouterWithoutRecord.GET("cache/download", registerTaskApi.DownloadRegisterTaskCache)
	}
}
