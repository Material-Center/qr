package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type QQCacheRouter struct{}

func (r *QQCacheRouter) InitQQCacheRouter(Router *gin.RouterGroup) {
	qqCacheRouter := Router.Group("qqCache").Use(middleware.OperationRecord())
	qqCacheRouterWithoutRecord := Router.Group("qqCache")
	{
		// upload/extract 请求体中包含敏感缓存内容，不进入操作日志
		qqCacheRouterWithoutRecord.POST("upload", qqCacheApi.Upload)
		qqCacheRouterWithoutRecord.POST("extract", qqCacheApi.Extract)
		qqCacheRouter.POST("list", qqCacheApi.List)
		qqCacheRouter.POST("resetExtract", qqCacheApi.ResetExtract)
	}
	{
		qqCacheRouterWithoutRecord.GET("roleHint", qqCacheApi.AppLoginRoleHint)
	}
}
