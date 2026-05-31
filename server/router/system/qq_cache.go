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
	publicInternalToolRouter := PublicGroup.Group("internalTool")
	{
		// upload/extract/import/export 请求体或响应中含敏感缓存内容，不进入操作日志
		qqCacheRouterWithoutRecord.POST("upload", qqCacheApi.Upload)
		publicQQCacheRouter.POST("uploadPhoneRegister", qqCacheApi.UploadPhoneRegister)
		publicInternalToolRouter.POST("qqCache/importZip", qqCacheApi.InternalToolImportQQCacheZip)
		publicInternalToolRouter.GET("qqCache/exists", qqCacheApi.InternalToolCheckQQCache)
		qqCacheRouterWithoutRecord.POST("importZip", qqCacheApi.AdminImportQQCacheZip)
		qqCacheRouterWithoutRecord.POST("extract", qqCacheApi.Extract)
		qqCacheRouterWithoutRecord.POST("exportIniZip", qqCacheApi.ExportIniZip)
		qqCacheRouterWithoutRecord.POST("exportPendingIniZip", qqCacheApi.ExportPendingIniZip)
		qqCacheRouterWithoutRecord.POST("exportAccountList", qqCacheApi.ExportAccountList)
		qqCacheRouterWithoutRecord.POST("exportIniZipByQQFile", qqCacheApi.ExportIniZipByQQFile)
		qqCacheRouter.POST("list", qqCacheApi.List)
		qqCacheRouter.POST("resetExtract", qqCacheApi.ResetExtract)
		qqCacheRouter.POST("billing/settle", qqCacheApi.SettleBilling)
		qqCacheRouterWithoutRecord.GET("billing/history", qqCacheApi.GetBillingSettlementHistory)
		qqCacheRouterWithoutRecord.GET("sales/summary", qqCacheApi.GetSalesSummary)
		qqCacheRouterWithoutRecord.POST("sales/extract", qqCacheApi.SalesExtract)
		qqCacheRouter.POST("sales/history", qqCacheApi.GetSalesHistory)
		qqCacheRouterWithoutRecord.GET("sales/summaryList", qqCacheApi.GetSalesSummaryList)
		qqCacheRouter.POST("sales/settle", qqCacheApi.SettleSalesBilling)
		qqCacheRouterWithoutRecord.GET("sales/settlement/history", qqCacheApi.GetSalesSettlementHistory)
	}
	{
		qqCacheRouterWithoutRecord.GET("roleHint", qqCacheApi.AppLoginRoleHint)
	}
}
