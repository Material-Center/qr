package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

type LimitConfig struct {
	// GenerationKey 根据业务生成key 下面CheckOrMark查询生成
	GenerationKey func(c *gin.Context) string
	// 检查函数,用户可修改具体逻辑,更加灵活
	CheckOrMark func(key string, expire int, limit int) error
	// Expire key 过期时间
	Expire int
	// Limit 周期时间
	Limit int
}

func (l LimitConfig) LimitWithTime() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isDeviceTaskOpenAPI(c.Request.URL.Path) {
			c.Next()
			return
		}
		if l.Expire <= 0 || l.Limit <= 0 {
			c.Next()
			return
		}
		if err := l.CheckOrMark(l.GenerationKey(c), l.Expire, l.Limit); err != nil {
			global.GVA_LOG.Error("limit", zap.Error(err), zap.String("ip", c.ClientIP()))
			c.JSON(http.StatusOK, gin.H{"code": response.ERROR, "msg": err.Error()})
			c.Abort()
			return
		} else {
			c.Next()
		}
	}
}

func isDeviceTaskOpenAPI(path string) bool {
	path = strings.TrimPrefix(path, global.GVA_CONFIG.System.RouterPrefix)
	switch path {
	case "/phoneRegisterTask/open-api/task",
		"/phoneRegisterTask/open-api/verify-code",
		"/phoneRegisterTask/open-api/report",
		"/phoneRegisterTask/open-api/cache":
		return true
	default:
		return false
	}
}

// DefaultGenerationKey 默认生成key
func DefaultGenerationKey(c *gin.Context) string {
	return "GVA_Limit" + c.ClientIP()
}

func LoginGenerationKey(c *gin.Context) string {
	return "GVA_Login_Limit" + c.ClientIP()
}

func DefaultCheckOrMark(key string, expire int, limit int) (err error) {
	// 判断是否开启redis
	if global.GVA_REDIS == nil {
		return err
	}
	if err = SetLimitWithTime(key, limit, time.Duration(expire)*time.Second); err != nil {
	}
	return err
}

func DefaultLimit() gin.HandlerFunc {
	return LimitConfig{
		GenerationKey: DefaultGenerationKey,
		CheckOrMark:   DefaultCheckOrMark,
		Expire:        global.GVA_CONFIG.System.LimitTimeIP,
		Limit:         global.GVA_CONFIG.System.LimitCountIP,
	}.LimitWithTime()
}

func DefaultLoginLimitConfig() LimitConfig {
	return LimitConfig{
		GenerationKey: LoginGenerationKey,
		CheckOrMark:   DefaultCheckOrMark,
		Expire:        global.GVA_CONFIG.System.LoginLimitTimeIP,
		Limit:         global.GVA_CONFIG.System.LoginLimitCountIP,
	}
}

func DefaultLoginLimit() gin.HandlerFunc {
	return DefaultLoginLimitConfig().LimitWithTime()
}

// SetLimitWithTime 设置访问次数
func SetLimitWithTime(key string, limit int, expiration time.Duration) error {
	count, err := global.GVA_REDIS.Exists(context.Background(), key).Result()
	if err != nil {
		return err
	}
	if count == 0 {
		pipe := global.GVA_REDIS.TxPipeline()
		pipe.Incr(context.Background(), key)
		pipe.Expire(context.Background(), key, expiration)
		_, err = pipe.Exec(context.Background())
		return err
	} else {
		if ttl, err := global.GVA_REDIS.PTTL(context.Background(), key).Result(); err == nil && ttl < 0 {
			_ = global.GVA_REDIS.Expire(context.Background(), key, expiration).Err()
		}
		// 次数
		if times, err := global.GVA_REDIS.Get(context.Background(), key).Int(); err != nil {
			return err
		} else {
			if times >= limit {
				if t, err := global.GVA_REDIS.PTTL(context.Background(), key).Result(); err != nil {
					return errors.New("请求太过频繁，请稍后再试")
				} else {
					return errors.New("请求太过频繁, 请 " + t.String() + " 秒后尝试")
				}
			} else {
				return global.GVA_REDIS.Incr(context.Background(), key).Err()
			}
		}
	}
}
