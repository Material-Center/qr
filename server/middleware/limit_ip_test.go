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
