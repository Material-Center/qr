package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type BaseRouter struct{}

func (s *BaseRouter) InitBaseRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	baseRouter := Router.Group("base")
	{
		baseRouter.POST("login", middleware.DefaultLoginLimit(), middleware.LoginUserAgentGuard(), middleware.RequestSignatureGuard(), baseApi.Login)
		baseRouter.POST("appLogin", middleware.DefaultLoginLimit(), middleware.LoginUserAgentGuard(), baseApi.AppLogin)
		baseRouter.POST("captcha", middleware.RequestSignatureGuard(), baseApi.Captcha)
	}
	return baseRouter
}
