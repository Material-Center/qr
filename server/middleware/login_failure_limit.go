package middleware

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

const (
	loginFailureLimit     = 10
	loginFailureKeyPrefix = "GVA_Login_Fail_Limit"
)

var (
	loginFailureCount = loginFailureCountFromRedis
	markLoginFailure  = markLoginFailureInRedis
	clearLoginFailure = clearLoginFailureInRedis
)

func LoginFailureGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := loginFailureCount(loginFailureKey(c))
		if err == nil && count >= loginFailureLimit {
			response.FailWithMessage("用户名不存在或者密码错误", c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func RecordLoginFailure(c *gin.Context) {
	_ = markLoginFailure(loginFailureKey(c), loginFailureWindow())
}

func ClearLoginFailure(c *gin.Context) {
	_ = clearLoginFailure(loginFailureKey(c))
}

func loginFailureKey(c *gin.Context) string {
	return loginFailureKeyPrefix + c.ClientIP()
}

func loginFailureWindow() time.Duration {
	if global.GVA_CONFIG.System.LoginLimitTimeIP > 0 {
		return time.Duration(global.GVA_CONFIG.System.LoginLimitTimeIP) * time.Second
	}
	return 30 * time.Minute
}

func loginFailureCountFromRedis(key string) (int, error) {
	if global.GVA_REDIS == nil {
		return 0, nil
	}
	return global.GVA_REDIS.Get(context.Background(), key).Int()
}

func markLoginFailureInRedis(key string, ttl time.Duration) error {
	if global.GVA_REDIS == nil {
		return nil
	}
	ctx := context.Background()
	pipe := global.GVA_REDIS.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	return count.Err()
}

func clearLoginFailureInRedis(key string) error {
	if global.GVA_REDIS == nil {
		return nil
	}
	return global.GVA_REDIS.Del(context.Background(), key).Err()
}
