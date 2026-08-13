package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type AuthorityRouter struct{}

func (s *AuthorityRouter) InitAuthorityRouter(Router *gin.RouterGroup) {
	authorityRouter := Router.Group("authority").Use(middleware.OperationRecord())
	authorityRouterWithoutRecord := Router.Group("authority")
	requestSignature := middleware.RequestSignatureGuard()
	{
		authorityRouter.POST("createAuthority", requestSignature, authorityApi.CreateAuthority)   // 创建角色
		authorityRouter.POST("deleteAuthority", requestSignature, authorityApi.DeleteAuthority)   // 删除角色
		authorityRouter.PUT("updateAuthority", requestSignature, authorityApi.UpdateAuthority)    // 更新角色
		authorityRouter.POST("copyAuthority", requestSignature, authorityApi.CopyAuthority)       // 拷贝角色
		authorityRouter.POST("setDataAuthority", requestSignature, authorityApi.SetDataAuthority) // 设置角色资源权限
		authorityRouter.POST("setRoleUsers", requestSignature, authorityApi.SetRoleUsers)         // 全量覆盖角色关联用户
	}
	{
		authorityRouterWithoutRecord.POST("getAuthorityList", authorityApi.GetAuthorityList)                        // 获取角色列表
		authorityRouterWithoutRecord.GET("getUsersByAuthority", requestSignature, authorityApi.GetUsersByAuthority) // 获取角色关联用户ID列表
	}
}
