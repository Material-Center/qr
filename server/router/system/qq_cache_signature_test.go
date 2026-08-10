package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestQQCacheSalesExtractRequiresRequestSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	global.GVA_LOG = zap.NewNop()

	router := gin.New()
	group := router.Group("")
	publicGroup := router.Group("")
	(&QQCacheRouter{}).InitQQCacheRouter(group, publicGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/qqCache/sales/extract", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(7), body["code"])
	require.Equal(t, "用户名不存在或者密码错误", body["msg"])
}
