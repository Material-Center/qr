package system

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonResp "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	modelSystemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSetUserInfoRejectsOtherLeadersPromoter(t *testing.T) {
	setupUserSecurityAPITestDB(t)

	router := gin.New()
	router.PUT("/user/setUserInfo", func(c *gin.Context) {
		c.Set("claims", &modelSystemReq.CustomClaims{
			BaseClaims: modelSystemReq.BaseClaims{ID: 10, AuthorityId: 200},
		})
		(&BaseApi{}).SetUserInfo(c)
	})

	body := []byte(`{"ID":12,"nickName":"blocked","enable":1}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/user/setUserInfo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp commonResp.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, commonResp.ERROR, resp.Code)
	require.Equal(t, "无权操作该账号", resp.Msg)
}

func TestResetPasswordRejectsOtherLeadersPromoter(t *testing.T) {
	setupUserSecurityAPITestDB(t)

	router := gin.New()
	router.POST("/user/resetPassword", func(c *gin.Context) {
		c.Set("claims", &modelSystemReq.CustomClaims{
			BaseClaims: modelSystemReq.BaseClaims{ID: 10, AuthorityId: 200},
		})
		(&BaseApi{}).ResetPassword(c)
	})

	body := []byte(`{"ID":12,"password":"new-password"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/user/resetPassword", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp commonResp.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, commonResp.ERROR, resp.Code)
	require.Equal(t, "无权操作该账号", resp.Msg)
}

func TestDeputyLeaderCreatedPromoterRecordsCreator(t *testing.T) {
	setupUserSecurityAPITestDB(t)

	router := gin.New()
	router.POST("/user/admin_register", func(c *gin.Context) {
		c.Set("claims", &modelSystemReq.CustomClaims{
			BaseClaims: modelSystemReq.BaseClaims{ID: 20, AuthorityId: 210},
		})
		(&BaseApi{}).Register(c)
	})

	body := []byte(`{"userName":"deputy-created-promoter","nickName":"地推","passWord":"password","authorityId":300,"authorityIds":[300],"enable":1}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/user/admin_register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp commonResp.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, commonResp.SUCCESS, resp.Code, resp.Msg)

	var user modelSystem.SysUser
	require.NoError(t, global.GVA_DB.Where("username = ?", "deputy-created-promoter").First(&user).Error)
	require.Equal(t, uint(20), user.CreatedBy)
	require.NotNil(t, user.LeaderID)
	require.Equal(t, uint(10), *user.LeaderID)
}

func setupUserSecurityAPITestDB(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	global.GVA_LOG = zap.NewNop()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	global.GVA_DB = db
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysAuthority{},
		&modelSystem.SysUserAuthority{},
	))
	require.NoError(t, db.Create(&[]modelSystem.SysAuthority{
		{AuthorityId: 200, AuthorityName: "团长"},
		{AuthorityId: 210, AuthorityName: "副团长"},
		{AuthorityId: 300, AuthorityName: "地推"},
	}).Error)
	require.NoError(t, db.Create(&[]modelSystem.SysUser{
		{GVA_MODEL: global.GVA_MODEL{ID: 10}, Username: "leader", AuthorityId: 200, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 12}, Username: "other-promoter", AuthorityId: 300, LeaderID: uintPtr(99), Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 20}, Username: "deputy", AuthorityId: 210, LeaderID: uintPtr(10), Enable: 1},
	}).Error)
}

func uintPtr(v uint) *uint {
	return &v
}
