package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

const (
	securityFailureLimit     = 10
	securityFailureWindow    = 30 * time.Minute
	securityTempBanTTL       = 30 * time.Minute
	securityFailureKeyPrefix = "GVA_Security_Fail_Limit"
	securityTempBanKeyPrefix = "GVA_Security_Temp_Ban"
)

var (
	isSecurityTempBanned = isSecurityTempBannedInRedis
	markSecurityFailure  = markSecurityFailureInRedis
	banSecurityFailureIP = banSecurityFailureIPInRedis
)

func SecurityTempBanGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSecurityTempBanned(c.ClientIP()) {
			c.JSON(http.StatusOK, gin.H{"code": response.ERROR, "msg": "用户名不存在或者密码错误"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RecordSecurityFailure(c *gin.Context) {
	ip := c.ClientIP()
	count, err := markSecurityFailure(securityFailureKey(ip), securityFailureWindow)
	if err == nil && count >= securityFailureLimit {
		_ = banSecurityFailureIP(ip, securityTempBanTTL)
	}
}

func securityFailureKey(ip string) string {
	return securityFailureKeyPrefix + ip
}

func securityTempBanKey(ip string) string {
	return securityTempBanKeyPrefix + ip
}

func isSecurityTempBannedInRedis(ip string) bool {
	if global.GVA_REDIS == nil || ip == "" {
		return false
	}
	ok, err := global.GVA_REDIS.Exists(context.Background(), securityTempBanKey(ip)).Result()
	return err == nil && ok > 0
}

func markSecurityFailureInRedis(key string, ttl time.Duration) (int, error) {
	if global.GVA_REDIS == nil {
		return 0, nil
	}
	ctx := context.Background()
	pipe := global.GVA_REDIS.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	if err := count.Err(); err != nil {
		return 0, err
	}
	return int(count.Val()), nil
}

func banSecurityFailureIPInRedis(ip string, ttl time.Duration) error {
	if global.GVA_REDIS == nil || ip == "" {
		return nil
	}
	return global.GVA_REDIS.Set(context.Background(), securityTempBanKey(ip), "1", ttl).Err()
}
