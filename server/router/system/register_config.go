package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type RegisterConfigRouter struct{}

func (r *RegisterConfigRouter) InitRegisterConfigRouter(Router *gin.RouterGroup) {
	registerConfigRouter := Router.Group("registerConfig").Use(middleware.OperationRecord())
	registerConfigRouterWithoutRecord := Router.Group("registerConfig")
	{
		registerConfigRouter.PUT("setMyConfig", registerConfigApi.SetMyRegisterConfig)
	}
	{
		registerConfigRouterWithoutRecord.GET("getMyConfig", registerConfigApi.GetMyRegisterConfig)
		registerConfigRouterWithoutRecord.GET("checkMyConfig", registerConfigApi.CheckMyRegisterConfig)
	}
}
