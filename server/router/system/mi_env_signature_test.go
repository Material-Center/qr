package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestMIEnvDeleteRoutesRequireRequestSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	global.GVA_LOG = zap.NewNop()
	oldDB := global.GVA_DB
	defer func() { global.GVA_DB = oldDB }()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&modelSystem.SysOperationRecord{}))
	global.GVA_DB = db

	router := gin.New()
	group := router.Group("")
	publicGroup := router.Group("")
	(&MIEnvRouter{}).InitMIEnvRouter(group, publicGroup)

	for _, route := range []string{"/miEnvAdmin/deleteSelected", "/miEnvAdmin/deleteAll"} {
		t.Run(route, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, route, nil)
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			var body response.Response
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			require.Equal(t, response.ERROR, body.Code)
			require.Equal(t, "用户名不存在或者密码错误", body.Msg)
		})
	}
}
