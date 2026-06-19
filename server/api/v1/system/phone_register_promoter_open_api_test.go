package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	sysReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPhoneRegisterPromoterOpenAPITest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	global.GVA_REDIS = nil
	resetPhoneRegisterPromoterOpenAPITokenCache()
	global.GVA_CONFIG = config.Server{}
	global.GVA_CONFIG.JWT.SigningKey = "phone-register-openapi-test-key"
	global.GVA_CONFIG.JWT.BufferTime = "1d"
	global.GVA_CONFIG.JWT.ExpiresTime = "7d"
	global.GVA_CONFIG.JWT.Issuer = "test"
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(7 * 24 * time.Hour))

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysApiToken{},
		&modelSystem.JwtBlacklist{},
		&modelSystem.SysPhoneRegisterTask{},
		&modelSystem.SysPhoneRegisterTaskLog{},
		&modelSystem.SysPhoneRegisterRiskDailyStat{},
		&modelSystem.SysRegisterConfig{},
	))
	global.GVA_DB = db
}

func newPhoneRegisterPromoterOpenAPIRouter() *gin.Engine {
	router := gin.New()
	api := PhoneRegisterTaskApi{}
	group := router.Group("/phoneRegisterTask/open-api/promoter")
	group.GET("device-stats", api.PromoterOpenAPIDeviceStats)
	group.POST("task", api.PromoterOpenAPICreateTask)
	group.POST("receive-task", api.PromoterOpenAPICreateReceiveTask)
	group.GET("task/:taskId", api.PromoterOpenAPIGetTask)
	group.POST("submit-code", api.PromoterOpenAPISubmitCode)
	return router
}

func createPhoneRegisterOpenAPITestUserToken(t *testing.T, userID uint, authorityID uint, active bool, expiresAt time.Time) string {
	t.Helper()
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: userID},
		UUID:        uuid.New(),
		Username:    "user-token-test",
		NickName:    "token test",
		AuthorityId: authorityID,
		Enable:      1,
	}
	require.NoError(t, global.GVA_DB.Create(&user).Error)

	claims := sysReq.CustomClaims{
		BaseClaims: sysReq.BaseClaims{
			UUID:        user.UUID,
			ID:          user.ID,
			Username:    user.Username,
			NickName:    user.NickName,
			AuthorityId: authorityID,
		},
		BufferTime: int64(24 * time.Hour / time.Second),
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"GVA"},
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Second)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    global.GVA_CONFIG.JWT.Issuer,
		},
	}
	token, err := (&utils.JWT{SigningKey: []byte(global.GVA_CONFIG.JWT.SigningKey)}).CreateToken(claims)
	require.NoError(t, err)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysApiToken{
		UserID:      user.ID,
		AuthorityID: authorityID,
		Token:       token,
		Status:      active,
		ExpiresAt:   expiresAt,
	}).Error)
	return token
}

func decodePhoneRegisterOpenAPIResponse(t *testing.T, rec *httptest.ResponseRecorder) response.Response {
	t.Helper()
	var got response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

func TestPromoterOpenAPIDeviceStatsRequiresUserToken(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()

	req := httptest.NewRequest(http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.ERROR, got.Code)
	require.Contains(t, got.Msg, "token")
}

func TestPromoterOpenAPIDeviceStatsAcceptsPromoterUserToken(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil)
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)
	raw, err := json.Marshal(got.Data)
	require.NoError(t, err)
	var data struct {
		DeviceOnlineCount int64 `json:"deviceOnlineCount"`
		DeviceIdleCount   int64 `json:"deviceIdleCount"`
	}
	require.NoError(t, json.Unmarshal(raw, &data))
	require.EqualValues(t, 0, data.DeviceOnlineCount)
	require.EqualValues(t, 0, data.DeviceIdleCount)
}

func TestPromoterOpenAPIDeviceStatsReturnsZeroWhenReservedCapacityExhausted(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 10,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil)
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)
	raw, err := json.Marshal(got.Data)
	require.NoError(t, err)
	var data struct {
		DeviceOnlineCount int64 `json:"deviceOnlineCount"`
		DeviceIdleCount   int64 `json:"deviceIdleCount"`
	}
	require.NoError(t, json.Unmarshal(raw, &data))
	require.EqualValues(t, 0, data.DeviceOnlineCount)
	require.EqualValues(t, 0, data.DeviceIdleCount)
}

func TestPromoterOpenAPITokenValidationUsesTenMinuteCache(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	auth, err := validatePhoneRegisterPromoterOpenAPIToken(token)
	require.NoError(t, err)
	require.EqualValues(t, 3001, auth.userID)

	require.NoError(t, global.GVA_DB.Model(&modelSystem.SysApiToken{}).Where("token = ?", token).Update("status", false).Error)

	auth, err = validatePhoneRegisterPromoterOpenAPIToken(token)
	require.NoError(t, err)
	require.EqualValues(t, 3001, auth.userID)
}

func TestDeleteApiTokenClearsPromoterOpenAPITokenCache(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	auth, err := validatePhoneRegisterPromoterOpenAPIToken(token)
	require.NoError(t, err)
	require.EqualValues(t, 3001, auth.userID)

	var apiToken modelSystem.SysApiToken
	require.NoError(t, global.GVA_DB.Where("token = ?", token).First(&apiToken).Error)

	router := gin.New()
	api := ApiTokenApi{}
	router.POST("/sysApiToken/deleteApiToken", func(c *gin.Context) {
		c.Set("claims", &sysReq.CustomClaims{
			BaseClaims: sysReq.BaseClaims{AuthorityId: rtRoleAdmin},
		})
		api.DeleteApiToken(c)
	})
	body := bytes.NewBufferString(fmt.Sprintf(`{"id":%d}`, apiToken.ID))
	req := httptest.NewRequest(http.MethodPost, "/sysApiToken/deleteApiToken", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)

	_, err = validatePhoneRegisterPromoterOpenAPIToken(token)
	require.EqualError(t, err, "OpenAPI token不存在或已作废")
}

func TestPromoterOpenAPICreateTaskUsesTokenUserAndUserSentMode(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	body := bytes.NewBufferString(`{"phone":"18878309701","smsReceiveMode":"PLATFORM_SEND"}`)
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)

	var task modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&task).Error)
	require.EqualValues(t, 3001, task.PromoterID)
	require.Equal(t, "18878309701", task.Phone)
	require.Equal(t, modelSystem.PhoneRegisterSMSModeUserSentToTX, task.SMSReceiveMode)
}

func TestPromoterOpenAPICreateReceiveTaskUsesPlatformSendMode(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	body := bytes.NewBufferString(`{"phone":"18878309701"}`)
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/receive-task", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)

	var task modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&task).Error)
	require.EqualValues(t, 3001, task.PromoterID)
	require.Equal(t, "18878309701", task.Phone)
	require.Equal(t, modelSystem.PhoneRegisterSMSModePlatformSend, task.SMSReceiveMode)
}

func TestPromoterOpenAPIGetTaskReturnsOnlyTokenUserTask(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))
	otherToken := createPhoneRegisterOpenAPITestUserToken(t, 3002, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))
	task := modelSystem.SysPhoneRegisterTask{
		Phone:          "18878309701",
		PromoterID:     3001,
		SMSReceiveMode: modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:         modelSystem.PhoneRegisterStatusPending,
		ExpiresAt:      time.Now().Add(30 * time.Minute),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	req := httptest.NewRequest(http.MethodGet, "/phoneRegisterTask/open-api/promoter/task/"+strconv.Itoa(int(task.ID)), nil)
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)

	req = httptest.NewRequest(http.MethodGet, "/phoneRegisterTask/open-api/promoter/task/"+strconv.Itoa(int(task.ID)), nil)
	req.Header.Set("X-Open-Api-Token", otherToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got = decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.ERROR, got.Code)
	require.Equal(t, "任务不存在", got.Msg)
}

func TestPromoterOpenAPISubmitCodeUsesTokenUser(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))
	now := time.Now()
	task := modelSystem.SysPhoneRegisterTask{
		Phone:           "18878309701",
		PromoterID:      3001,
		SMSReceiveMode:  modelSystem.PhoneRegisterSMSModePlatformSend,
		Status:          modelSystem.PhoneRegisterStatusWaitingPromoterCode,
		CodeRequestedAt: &now,
		ExpiresAt:       now.Add(30 * time.Minute),
	}
	require.NoError(t, global.GVA_DB.Create(&task).Error)

	body := bytes.NewBufferString(fmt.Sprintf(`{"taskId":%d,"verifyCode":"123456"}`, task.ID))
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/submit-code", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)

	var stored modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&stored, task.ID).Error)
	require.Equal(t, "123456", stored.PendingCode)
}

func TestPromoterOpenAPICreateTaskRejectsNegativeStartDelay(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	body := bytes.NewBufferString(`{"phone":"18878309701","startDelaySeconds":-1}`)
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.ERROR, got.Code)
	require.Equal(t, "startDelaySeconds不能小于0", got.Msg)
}

func TestPromoterOpenAPICreateTaskRejectsStartDelayOverLimit(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	body := bytes.NewBufferString(`{"phone":"18878309701","startDelaySeconds":601}`)
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.ERROR, got.Code)
	require.Equal(t, "startDelaySeconds不能超过600", got.Msg)
}

func TestPromoterOpenAPICreateTaskKeepsLegacyBehaviorWhenStartDelayIsZero(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))

	body := bytes.NewBufferString(`{"phone":"18878309701","startDelaySeconds":0,"reserveDevice":true}`)
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.SUCCESS, got.Code)

	var task modelSystem.SysPhoneRegisterTask
	require.NoError(t, global.GVA_DB.First(&task).Error)
	require.Nil(t, task.AvailableAt)
	require.Nil(t, task.HolderDeviceID)
	require.Equal(t, modelSystem.PhoneRegisterTaskSourceOpenAPI, task.TaskSource)
}

func TestPromoterOpenAPICreateTaskReturnsDeviceCapacityErrorCode(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 3001, rtRolePromoter, true, time.Now().Add(30*24*time.Hour))
	enabled := true
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterConfig{
		OwnerType:                          modelSystem.RegisterConfigOwnerAdmin,
		OwnerID:                            0,
		PhoneRegisterEnabled:               &enabled,
		PhoneRegisterOpenAPIReserveDevices: 1,
	}).Error)

	body := bytes.NewBufferString(`{"phone":"18878309701"}`)
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.ERROR, got.Code)
	require.Equal(t, "OpenAPI可用设备不足", got.Msg)
	raw, err := json.Marshal(got.Data)
	require.NoError(t, err)
	var data struct {
		ErrorCode string `json:"errorCode"`
	}
	require.NoError(t, json.Unmarshal(raw, &data))
	require.Equal(t, "OPENAPI_DEVICE_CAPACITY_NOT_ENOUGH", data.ErrorCode)
}

func TestPromoterOpenAPIRejectsNonPromoterToken(t *testing.T) {
	setupPhoneRegisterPromoterOpenAPITest(t)
	router := newPhoneRegisterPromoterOpenAPIRouter()
	token := createPhoneRegisterOpenAPITestUserToken(t, 1001, rtRoleAdmin, true, time.Now().Add(30*24*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/phoneRegisterTask/open-api/promoter/device-stats", nil)
	req.Header.Set("X-Open-Api-Token", token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	got := decodePhoneRegisterOpenAPIResponse(t, rec)
	require.Equal(t, response.ERROR, got.Code)
	require.Contains(t, got.Msg, "地推")
}
