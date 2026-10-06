package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type MIEnvRouter struct{}

func (r *MIEnvRouter) InitMIEnvRouter(_ *gin.RouterGroup, publicGroup *gin.RouterGroup) {
	internal := publicGroup.Group("internalTool/miEnv").Use(middleware.MIEnvInternalKeyGuard())
	internal.POST("/*action", miEnvApi.Handle)
	internal.GET("/*action", miEnvApi.Handle)
}
