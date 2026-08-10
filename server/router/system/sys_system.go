package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type SysRouter struct{}

func (s *SysRouter) InitSystemRouter(Router *gin.RouterGroup) {
	sysRouter := Router.Group("system").Use(middleware.OperationRecord())
	sysRouterWithoutRecord := Router.Group("system")
	requestSignature := middleware.RequestSignatureGuard()

	{
		sysRouter.POST("setSystemConfig", requestSignature, systemApi.SetSystemConfig) // 设置配置文件内容
		sysRouter.POST("reloadSystem", requestSignature, systemApi.ReloadSystem)       // 重启服务
	}
	{
		sysRouterWithoutRecord.POST("getSystemConfig", systemApi.GetSystemConfig) // 获取配置文件内容
		sysRouterWithoutRecord.POST("getServerInfo", systemApi.GetServerInfo)     // 获取服务器信息
	}
}
