package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/stretchr/testify/require"
)

func TestDeviceServiceNoopsWhenRedisUnavailable(t *testing.T) {
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = nil
	t.Cleanup(func() {
		global.GVA_REDIS = originalRedis
	})

	require.NoError(t, (&DeviceService{}).MarkHeartbeat("9130dbc0"))
	require.NoError(t, (&DeviceService{}).MarkBusy("9130dbc0", "phone_register"))
	require.NoError(t, (&DeviceService{}).ClearBusy("9130dbc0"))
	require.Empty(t, (&DeviceService{}).ListOnlineDeviceIDs())
	require.Empty(t, (&DeviceService{}).ListBusyDeviceIDs())
}
