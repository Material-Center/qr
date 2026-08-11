package system

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDeviceConfigGroupApisRejectNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		route  func(*DeviceConfigApi, *gin.Context)
	}{
		{
			name:   "list",
			method: http.MethodGet,
			path:   "/deviceConfig/group/list",
			route:  (*DeviceConfigApi).GroupList,
		},
		{
			name:   "save",
			method: http.MethodPost,
			path:   "/deviceConfig/group/save",
			body:   `{"name":"A组"}`,
			route:  (*DeviceConfigApi).GroupSave,
		},
		{
			name:   "delete",
			method: http.MethodPost,
			path:   "/deviceConfig/group/delete",
			body:   `{"id":1}`,
			route:  (*DeviceConfigApi).GroupDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			api := &DeviceConfigApi{}
			router.Handle(tt.method, tt.path, func(c *gin.Context) {
				c.Set("claims", &systemReq.CustomClaims{
					BaseClaims: systemReq.BaseClaims{AuthorityId: 600},
				})
				tt.route(api, c)
			})

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			var got response.Response
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.Equal(t, response.ERROR, got.Code)
			require.Equal(t, "仅管理员可管理设备分组", got.Msg)
		})
	}
}
