package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type QQCacheRouter struct{}

func (r *QQCacheRouter) InitQQCacheRouter(Router *gin.RouterGroup, PublicGroup *gin.RouterGroup) {
	qqCacheRouter := Router.Group("qqCache").Use(middleware.OperationRecord())
	qqCacheRouterWithoutRecord := Router.Group("qqCache")
	publicQQCacheRouter := PublicGroup.Group("qqCache")
	{
		// upload/extract/exportIniZip 请求体或响应中含敏感缓存内容，不进入操作日志
		qqCacheRouterWithoutRecord.POST("upload", qqCacheApi.Upload)
		publicQQCacheRouter.POST("uploadPhoneRegister", qqCacheApi.UploadPhoneRegister)
		qqCacheRouterWithoutRecord.POST("extract", qqCacheApi.Extract)
		qqCacheRouterWithoutRecord.POST("exportIniZip", qqCacheApi.ExportIniZip)
		qqCacheRouterWithoutRecord.POST("exportPendingIniZip", qqCacheApi.ExportPendingIniZip)
		qqCacheRouterWithoutRecord.POST("exportIniZipByQQFile", qqCacheApi.ExportIniZipByQQFile)
		qqCacheRouter.POST("list", qqCacheApi.List)
		qqCacheRouter.POST("resetExtract", qqCacheApi.ResetExtract)
		qqCacheRouter.POST("billing/settle", qqCacheApi.SettleBilling)
		qqCacheRouterWithoutRecord.GET("billing/history", qqCacheApi.GetBillingSettlementHistory)
	}
	{
		qqCacheRouterWithoutRecord.GET("roleHint", qqCacheApi.AppLoginRoleHint)
	}
}
