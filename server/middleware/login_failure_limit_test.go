package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLoginFailureGuardBlocksWhenClientIPReachedLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	count := 0
	oldCount := loginFailureCount
	loginFailureCount = func(key string) (int, error) {
		require.Equal(t, "GVA_Login_Fail_Limit192.0.2.30", key)
		return count, nil
	}
	t.Cleanup(func() { loginFailureCount = oldCount })

	router := gin.New()
	router.Use(LoginFailureGuard())
	router.POST("/base/login", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	count = 9
	recAllowed := httptest.NewRecorder()
	reqAllowed := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	reqAllowed.RemoteAddr = "192.0.2.30:12345"
	router.ServeHTTP(recAllowed, reqAllowed)

	require.Equal(t, http.StatusOK, recAllowed.Code)
	require.Equal(t, "ok", recAllowed.Body.String())

	count = 10
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	req.RemoteAddr = "192.0.2.30:12345"
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(7), body["code"])
	require.Equal(t, "用户名不存在或者密码错误", body["msg"])
}

func TestRecordAndClearLoginFailureUseClientIPKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	global.GVA_CONFIG = config.Server{}
	global.GVA_CONFIG.System.LoginLimitTimeIP = 1800

	var markedKey string
	var markedTTL time.Duration
	oldMark := markLoginFailure
	markLoginFailure = func(key string, ttl time.Duration) error {
		markedKey = key
		markedTTL = ttl
		return nil
	}
	var clearedKey string
	oldClear := clearLoginFailure
	clearLoginFailure = func(key string) error {
		clearedKey = key
		return nil
	}
	t.Cleanup(func() {
		markLoginFailure = oldMark
		clearLoginFailure = oldClear
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/base/login", nil)
	c.Request.RemoteAddr = "192.0.2.31:12345"

	RecordLoginFailure(c)
	ClearLoginFailure(c)

	require.Equal(t, "GVA_Login_Fail_Limit192.0.2.31", markedKey)
	require.Equal(t, 30*time.Minute, markedTTL)
	require.Equal(t, markedKey, clearedKey)
}
