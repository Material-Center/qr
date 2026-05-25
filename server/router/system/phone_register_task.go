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
		phoneRegisterTaskRouter.POST("settle", phoneRegisterTaskApi.SettlePhoneRegisterTaskLeader)
	}
	{
		phoneRegisterTaskRouterWithoutRecord.GET("active", phoneRegisterTaskApi.GetActivePhoneRegisterTask)
		phoneRegisterTaskRouterWithoutRecord.GET("actives", phoneRegisterTaskApi.GetActivePhoneRegisterTasks)
		phoneRegisterTaskRouterWithoutRecord.GET("submitStatus", phoneRegisterTaskApi.GetPhoneRegisterSubmitStatus)
		phoneRegisterTaskRouterWithoutRecord.POST("list", phoneRegisterTaskApi.GetPhoneRegisterTaskList)
		phoneRegisterTaskRouterWithoutRecord.POST("logs", phoneRegisterTaskApi.GetPhoneRegisterTaskLogs)
		phoneRegisterTaskRouterWithoutRecord.GET("summary", phoneRegisterTaskApi.GetPhoneRegisterTaskSummary)
		phoneRegisterTaskRouterWithoutRecord.GET("settlement/history", phoneRegisterTaskApi.GetPhoneRegisterTaskSettlementHistory)
	}
	{
		publicPhoneRegisterTaskRouter.POST("device/poll", phoneRegisterTaskApi.DevicePollPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.GET("device/task", phoneRegisterTaskApi.DeviceGetPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("device/task", phoneRegisterTaskApi.DeviceGetPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("device/heartbeat", phoneRegisterTaskApi.DeviceHeartbeatPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("device/report", phoneRegisterTaskApi.DeviceReportPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("device/log", phoneRegisterTaskApi.DeviceLogPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.GET("device/config", phoneRegisterTaskApi.DeviceGetPhoneRegisterConfig)
		publicPhoneRegisterTaskRouter.POST("open-api/task", phoneRegisterTaskApi.OpenAPIPollPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.GET("open-api/task", phoneRegisterTaskApi.OpenAPIGetPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("open-api/verify-code", phoneRegisterTaskApi.OpenAPIGetVerifyCode)
		publicPhoneRegisterTaskRouter.POST("open-api/report", phoneRegisterTaskApi.OpenAPIReportPhoneRegisterTask)
		publicPhoneRegisterTaskRouter.POST("open-api/cache", phoneRegisterTaskApi.OpenAPIUploadPhoneRegisterCache)
		publicPhoneRegisterTaskRouter.GET("open-api/promoter/device-stats", phoneRegisterTaskApi.PromoterOpenAPIDeviceStats)
		publicPhoneRegisterTaskRouter.POST("open-api/promoter/task", phoneRegisterTaskApi.PromoterOpenAPICreateTask)
	}
}
