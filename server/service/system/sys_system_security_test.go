package system

import (
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestSetSystemConfigRejectsSecurityCriticalChanges(t *testing.T) {
	setupSystemConfigTest(t)

	next := global.GVA_CONFIG
	next.JWT.SigningKey = "attacker-controlled"
	next.System.UseMultipoint = false
	next.Mysql.Path = "attacker-db"

	err := (&SystemConfigService{}).SetSystemConfig(modelSystem.System{Config: next})

	require.Error(t, err)
	require.Contains(t, err.Error(), "禁止通过接口修改安全配置")
}

func TestSetSystemConfigAllowsNonSecurityChanges(t *testing.T) {
	configPath := setupSystemConfigTest(t)

	next := global.GVA_CONFIG
	next.Zap.Level = "info"

	err := (&SystemConfigService{}).SetSystemConfig(modelSystem.System{Config: next})

	require.NoError(t, err)
	require.FileExists(t, configPath)
}

func setupSystemConfigTest(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	vp := viper.New()
	vp.SetConfigFile(configPath)
	global.GVA_VP = vp
	global.GVA_CONFIG = config.Server{
		JWT: config.JWT{
			SigningKey:  "stable-signing-key",
			ExpiresTime: "7d",
			BufferTime:  "1d",
			Issuer:      "qr",
		},
		System: config.System{
			DbType:        "mysql",
			Addr:          8888,
			LimitCountIP:  15000,
			LimitTimeIP:   3600,
			UseMultipoint: true,
			UseRedis:      true,
			UseStrictAuth: false,
		},
		Zap: config.Zap{
			Level: "debug",
		},
	}
	return configPath
}
