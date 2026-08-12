package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecurityTempBanGuardBlocksBannedClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldCheck := isSecurityTempBanned
	isSecurityTempBanned = func(ip string) bool {
		require.Equal(t, "192.0.2.41", ip)
		return true
	}
	t.Cleanup(func() { isSecurityTempBanned = oldCheck })

	called := false
	router := gin.New()
	router.Use(SecurityTempBanGuard())
	router.POST("/base/login", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	req.RemoteAddr = "192.0.2.41:12345"
	router.ServeHTTP(rec, req)

	require.False(t, called)
	require.Equal(t, http.StatusOK, rec.Code)

	var body response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, response.ERROR, body.Code)
	require.Equal(t, "用户名不存在或者密码错误", body.Msg)
}

func TestRecordSecurityFailureBansIPAfterLimit(t *testing.T) {
	oldMark := markSecurityFailure
	oldBan := banSecurityFailureIP
	var markedKeys []string
	var bannedIP string
	markSecurityFailure = func(key string, ttl time.Duration) (int, error) {
		markedKeys = append(markedKeys, key)
		return securityFailureLimit, nil
	}
	banSecurityFailureIP = func(ip string, ttl time.Duration) error {
		bannedIP = ip
		require.Equal(t, securityTempBanTTL, ttl)
		return nil
	}
	t.Cleanup(func() {
		markSecurityFailure = oldMark
		banSecurityFailureIP = oldBan
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/base/login", nil)
	c.Request.RemoteAddr = "192.0.2.42:12345"

	RecordSecurityFailure(c)

	require.Equal(t, []string{"GVA_Security_Fail_Limit192.0.2.42"}, markedKeys)
	require.Equal(t, "192.0.2.42", bannedIP)
}
