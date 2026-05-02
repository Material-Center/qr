package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type PhoneRegisterTaskRouter struct{}

func (r *PhoneRegisterTaskRouter) InitPhoneRegisterTaskRouter(PrivateGroup *gin.RouterGroup, PublicGroup *gin.RouterGroup) {
	phoneRegisterTaskRouter := PrivateGroup.Group("phoneRegisterTask").Use(middleware.OperationRecord())
	phoneRegisterTaskRouterWithoutRecord := PrivateGroup.Group("phoneRegisterTask")
	publicPhoneRegisterTaskRouter := PublicGroup.Group("phoneRegisterTask")
	{
		phoneRegisterTaskRouter.POST("create", phoneRegisterTaskApi.CreatePhoneRegisterTask)
		phoneRegisterTaskRouter.POST("submitCode", phoneRegisterTaskApi.SubmitPhoneRegisterTaskCode)
	}
	{
		phoneRegisterTaskRouterWithoutRecord.GET("active", phoneRegisterTaskApi.GetActivePhoneRegisterTask)
		phoneRegisterTaskRouterWithoutRecord.GET("actives", phoneRegisterTaskApi.GetActivePhoneRegisterTasks)
		phoneRegisterTaskRouterWithoutRecord.POST("list", phoneRegisterTaskApi.GetPhoneRegisterTaskList)
		phoneRegisterTaskRouterWithoutRecord.GET("summary", phoneRegisterTaskApi.GetPhoneRegisterTaskSummary)
	}
	{
		publicPhoneRegisterTaskRouter.POST("device/poll", phoneRegisterTaskApi.DevicePollPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.GET("device/task", phoneRegisterTaskApi.DeviceGetPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("device/task", phoneRegisterTaskApi.DeviceGetPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("device/heartbeat", phoneRegisterTaskApi.DeviceHeartbeatPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("device/report", phoneRegisterTaskApi.DeviceReportPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.GET("device/config", phoneRegisterTaskApi.DeviceGetPhoneRegisterConfig)
	}
}
