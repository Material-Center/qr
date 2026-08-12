package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLimitWithTimeSkipsWhenLimitDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark: func(key string, expire int, limit int) error {
			called = true
			return errors.New("should not be called")
		},
		Expire: 60,
		Limit:  0,
	}.LimitWithTime())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
	require.False(t, called)
}

func TestLimitWithTimeLogsClientIPWhenLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, logs := observer.New(zap.ErrorLevel)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.New(core)
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark:   func(string, int, int) error { return errors.New("limited") },
		Expire:        60,
		Limit:         1,
	}.LimitWithTime())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "203.0.113.8:12345"
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, "limit", entry.Message)
	require.Equal(t, "203.0.113.8", entry.ContextMap()["ip"])
}

func TestLimitWithTimeSkipsDeviceTaskOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	called := false
	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark: func(string, int, int) error {
			called = true
			return errors.New("limited")
		},
		Expire: 60,
		Limit:  1,
	}.LimitWithTime())
	router.POST("/phoneRegisterTask/open-api/task", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/task", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
	require.False(t, called)
}

func TestLimitWithTimeStillLimitsPromoterOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark:   func(string, int, int) error { return errors.New("limited") },
		Expire:        60,
		Limit:         1,
	}.LimitWithTime())
	router.POST("/phoneRegisterTask/open-api/promoter/task", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "limited", body["msg"])
}

func TestLimitWithTimeStillLimitsOtherOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark:   func(string, int, int) error { return errors.New("limited") },
		Expire:        60,
		Limit:         1,
	}.LimitWithTime())
	router.POST("/phoneRegisterTask/open-api/other", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/other", nil)
	router.ServeHTTP(rec, req)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "limited", body["msg"])
}

func TestDefaultLoginLimitUsesDedicatedConfigAndKey(t *testing.T) {
	global.GVA_CONFIG = config.Server{}
	global.GVA_CONFIG.System.LoginLimitCountIP = 8
	global.GVA_CONFIG.System.LoginLimitTimeIP = 300

	limit := DefaultLoginLimitConfig()

	require.Equal(t, 8, limit.Limit)
	require.Equal(t, 300, limit.Expire)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/base/login", nil)
	c.Request.RemoteAddr = "192.0.2.10:12345"
	require.Equal(t, "GVA_Login_Limit192.0.2.10", limit.GenerationKey(c))
}

func TestLoginUserAgentGuardRejectsScriptUAWithGenericFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	router := gin.New()
	router.Use(LoginUserAgentGuard())
	router.POST("/base/login", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, called)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(7), body["code"])
	require.Equal(t, "用户名不存在或者密码错误", body["msg"])
}

func TestLoginUserAgentGuardAllowsBrowserUA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(LoginUserAgentGuard())
	router.POST("/base/login", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
}
