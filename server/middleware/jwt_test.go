package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	serviceSystem "github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupJWTAuthTokenPurposeTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	global.GVA_CONFIG = config.Server{}
	global.GVA_CONFIG.JWT.SigningKey = "jwt-auth-token-purpose-test-key"
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
		&modelSystem.SysAuthority{},
		&modelSystem.SysApiToken{},
		&modelSystem.JwtBlacklist{},
	))
	global.GVA_DB = db
}

func newJWTAuthTestRouter() *gin.Engine {
	router := gin.New()
	router.GET("/private", JWTAuth(), func(c *gin.Context) {
		response.OkWithMessage("ok", c)
	})
	return router
}

func TestJWTAuthRejectsOpenAPIToken(t *testing.T) {
	setupJWTAuthTokenPurposeTest(t)
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: 1},
		UUID:        uuid.New(),
		Username:    "promoter",
		NickName:    "promoter",
		AuthorityId: 300,
		Enable:      1,
	}
	require.NoError(t, global.GVA_DB.Create(&user).Error)
	token, err := (&serviceSystem.ApiTokenService{}).CreateApiTokenForOperator(100, modelSystem.SysApiToken{
		UserID:      user.ID,
		AuthorityID: user.AuthorityId,
	}, 30)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("x-token", token)
	rec := httptest.NewRecorder()
	newJWTAuthTestRouter().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	var got response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Contains(t, got.Msg, "OpenAPI")
}

func TestJWTAuthAcceptsLoginToken(t *testing.T) {
	setupJWTAuthTokenPurposeTest(t)
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: 1},
		UUID:        uuid.New(),
		Username:    "promoter",
		NickName:    "promoter",
		AuthorityId: 300,
		Enable:      1,
	}
	token, _, err := utils.LoginToken(&user)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("x-token", token)
	rec := httptest.NewRecorder()
	newJWTAuthTestRouter().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
