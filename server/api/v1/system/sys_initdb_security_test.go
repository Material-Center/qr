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

func TestInitDBIsDisabledByDefault(t *testing.T) {
	t.Setenv("QR_ENABLE_INIT_API", "")
	global.GVA_DB = nil
	global.GVA_LOG = zap.NewNop()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/init/initdb", nil)

	(&DBApi{}).InitDB(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, float64(7), body["code"])
	require.Equal(t, "初始化接口已禁用", body["msg"])
}

func TestCheckDBIsDisabledByDefault(t *testing.T) {
	t.Setenv("QR_ENABLE_INIT_API", "")
	global.GVA_DB = nil
	global.GVA_LOG = zap.NewNop()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/init/checkdb", nil)

	(&DBApi{}).CheckDB(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, float64(7), body["code"])
	require.Equal(t, "初始化接口已禁用", body["msg"])
}
