package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSetUserInfoRefreshesLoginUserEnableCache(t *testing.T) {
	setupPhoneRegisterTaskTestDB(t)
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: 1},
		UUID:        uuid.New(),
		Username:    "promoter",
		NickName:    "promoter",
		AuthorityId: 300,
		Enable:      1,
	}
	require.NoError(t, global.GVA_DB.Create(&user).Error)
	server := newFakeRedisServer(t, nil)
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
	})

	err := (&UserService{}).SetUserInfo(modelSystem.SysUser{
		GVA_MODEL: global.GVA_MODEL{ID: user.ID},
		NickName:  user.NickName,
		Enable:    2,
	})

	require.NoError(t, err)
	require.Equal(t, 1, server.count("set"))
}
