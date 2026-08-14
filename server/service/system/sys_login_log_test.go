package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	modelSystemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetLoginLogInfoListFiltersByIPAndAgent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	global.GVA_DB = db
	require.NoError(t, db.AutoMigrate(&modelSystem.SysLoginLog{}, &modelSystem.SysUser{}))
	require.NoError(t, db.Create(&[]modelSystem.SysLoginLog{
		{Username: "root", Ip: "192.253.229.139", Agent: "curl/8.0.1", Status: false},
		{Username: "admin", Ip: "192.253.229.140", Agent: "Mozilla/5.0 Chrome/149.0.0.0", Status: true},
		{Username: "root", Ip: "10.0.0.1", Agent: "Mozilla/5.0 Chrome/149.0.0.0", Status: false},
	}).Error)

	list, total, err := (&LoginLogService{}).GetLoginLogInfoList(modelSystemReq.SysLoginLogSearch{
		SysLoginLog: modelSystem.SysLoginLog{
			Ip:    "192.253.229",
			Agent: "curl",
		},
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 10},
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	logs := list.([]modelSystem.SysLoginLog)
	require.Len(t, logs, 1)
	require.Equal(t, "192.253.229.139", logs[0].Ip)
	require.Equal(t, "curl/8.0.1", logs[0].Agent)
}
