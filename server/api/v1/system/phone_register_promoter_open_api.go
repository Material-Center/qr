package system

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemService "github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PromoterOpenAPIDeviceStats
// @Tags      PhoneRegisterTask
// @Summary   用户Token OpenAPI查询手机号注册可用设备
// @Produce   application/json
// @Success   200  {object}  response.Response{data=gin.H,msg=string}
// @Router    /phoneRegisterTask/open-api/promoter/device-stats [get]
func (a *PhoneRegisterTaskApi) PromoterOpenAPIDeviceStats(c *gin.Context) {
	if _, ok := requirePhoneRegisterPromoterOpenAPIToken(c); !ok {
		return
	}
	stats, err := phoneRegisterTaskService.GetCurrentDeviceStats()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{
		"deviceOnlineCount": stats.Online,
		"deviceIdleCount":   stats.Idle,
	}, "获取成功", c)
}

// PromoterOpenAPICreateTask
// @Tags      PhoneRegisterTask
// @Summary   用户Token OpenAPI创建手机号注册任务
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.PhoneRegisterTaskCreate  true  "创建参数"
// @Success   200   {object}  response.Response{data=gin.H,msg=string}
// @Router    /phoneRegisterTask/open-api/promoter/task [post]
func (a *PhoneRegisterTaskApi) PromoterOpenAPICreateTask(c *gin.Context) {
	auth, ok := requirePhoneRegisterPromoterOpenAPIToken(c)
	if !ok {
		return
	}
	var req systemReq.PhoneRegisterTaskCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.StartDelaySeconds < 0 {
		response.FailWithMessage("startDelaySeconds不能小于0", c)
		return
	}
	if req.StartDelaySeconds > 600 {
		response.FailWithMessage("startDelaySeconds不能超过600", c)
		return
	}
	task, err := phoneRegisterTaskService.CreateTask(auth.userID, req.Phone, system.PhoneRegisterSMSModeUserSentToTX, PhoneRegisterTaskCreateOptionsForOpenAPI(req))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(buildPhoneRegisterActiveInfo(task), "创建成功", c)
}

func PhoneRegisterTaskCreateOptionsForOpenAPI(req systemReq.PhoneRegisterTaskCreate) systemService.PhoneRegisterTaskCreateOptions {
	return systemService.PhoneRegisterTaskCreateOptions{
		TaskSource:        system.PhoneRegisterTaskSourceOpenAPI,
		StartDelaySeconds: req.StartDelaySeconds,
		ReserveDevice:     req.ReserveDevice,
	}
}

type phoneRegisterPromoterOpenAPIAuth struct {
	userID      uint
	authorityID uint
}

func requirePhoneRegisterPromoterOpenAPIToken(c *gin.Context) (phoneRegisterPromoterOpenAPIAuth, bool) {
	token := getPhoneRegisterPromoterOpenAPIToken(c.Request)
	if token == "" {
		response.NoAuth("缺少OpenAPI token", c)
		return phoneRegisterPromoterOpenAPIAuth{}, false
	}
	auth, err := validatePhoneRegisterPromoterOpenAPIToken(token)
	if err != nil {
		response.NoAuth(err.Error(), c)
		return phoneRegisterPromoterOpenAPIAuth{}, false
	}
	return auth, true
}

func getPhoneRegisterPromoterOpenAPIToken(req *http.Request) string {
	token := strings.TrimSpace(req.Header.Get("X-Open-Api-Token"))
	if token != "" {
		return token
	}
	auth := strings.TrimSpace(req.Header.Get("Authorization"))
	if len(auth) > len("Bearer ") && strings.EqualFold(auth[:len("Bearer ")], "Bearer ") {
		return strings.TrimSpace(auth[len("Bearer "):])
	}
	return ""
}

func validatePhoneRegisterPromoterOpenAPIToken(token string) (phoneRegisterPromoterOpenAPIAuth, error) {
	if global.GVA_DB == nil {
		return phoneRegisterPromoterOpenAPIAuth{}, errors.New("数据库未初始化")
	}
	claims, err := utils.NewJWT().ParseToken(token)
	if err != nil {
		return phoneRegisterPromoterOpenAPIAuth{}, errors.New("OpenAPI token无效")
	}
	var apiToken system.SysApiToken
	if err := global.GVA_DB.Where("token = ? AND status = ?", token, true).First(&apiToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return phoneRegisterPromoterOpenAPIAuth{}, errors.New("OpenAPI token不存在或已作废")
		}
		return phoneRegisterPromoterOpenAPIAuth{}, err
	}
	if apiToken.ExpiresAt.Before(time.Now()) {
		return phoneRegisterPromoterOpenAPIAuth{}, errors.New("OpenAPI token已过期")
	}
	if apiToken.UserID == 0 || apiToken.UserID != claims.BaseClaims.ID {
		return phoneRegisterPromoterOpenAPIAuth{}, errors.New("OpenAPI token用户不匹配")
	}
	if apiToken.AuthorityID == 0 || apiToken.AuthorityID != claims.BaseClaims.AuthorityId {
		return phoneRegisterPromoterOpenAPIAuth{}, errors.New("OpenAPI token角色不匹配")
	}
	if apiToken.AuthorityID != rtRolePromoter {
		return phoneRegisterPromoterOpenAPIAuth{}, errors.New("仅地推OpenAPI token可访问")
	}
	return phoneRegisterPromoterOpenAPIAuth{
		userID:      apiToken.UserID,
		authorityID: apiToken.AuthorityID,
	}, nil
}
