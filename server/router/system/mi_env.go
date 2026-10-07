package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type MIEnvRouter struct{}

func (r *MIEnvRouter) InitMIEnvRouter(privateGroup *gin.RouterGroup, publicGroup *gin.RouterGroup) {
	admin := privateGroup.Group("miEnvAdmin")
	adminMutation := privateGroup.Group("miEnvAdmin").Use(middleware.OperationRecord())
	admin.POST("list", miEnvApi.AdminList)
	admin.POST("types", miEnvApi.AdminTypes)
	adminMutation.POST("deleteSelected", middleware.RequestSignatureGuard(), miEnvApi.AdminDeleteSelected)
	adminMutation.POST("deleteAll", middleware.RequestSignatureGuard(), miEnvApi.AdminDeleteAll)

	internal := publicGroup.Group("internalTool/miEnv").Use(middleware.MIEnvInternalKeyGuard())
	internal.POST("/*action", miEnvApi.Handle)
	internal.GET("/*action", miEnvApi.Handle)
}
