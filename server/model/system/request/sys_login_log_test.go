package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSysLoginLogSearchBindsLowercaseQueryFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/sysLoginLog/getLoginLogList?page=1&pageSize=10&ip=192.253.229&agent=Chrome", nil)

	var search SysLoginLogSearch
	require.NoError(t, c.ShouldBindQuery(&search))
	require.Equal(t, 1, search.Page)
	require.Equal(t, 10, search.PageSize)
	require.Equal(t, "192.253.229", search.Ip)
	require.Equal(t, "Chrome", search.Agent)
}
