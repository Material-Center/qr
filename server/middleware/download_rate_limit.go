package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	downloadRateLimitEntryTTL = 10 * time.Minute
	downloadMaxConcurrency    = 20
)

type downloadRateLimitEntry struct {
	mu       sync.Mutex
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	downloadRateLimiters sync.Map
	downloadSlots        = make(chan struct{}, downloadMaxConcurrency)
)

// DownloadRateLimit limits attachment downloads per client IP and caps global
// concurrent responses. Download tickets provide the additional single-use
// credential limit.
func DownloadRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		clientIP := c.ClientIP()
		value, _ := downloadRateLimiters.LoadOrStore(clientIP, &downloadRateLimitEntry{
			limiter:  rate.NewLimiter(rate.Every(2*time.Second), 5),
			lastSeen: now,
		})
		entry := value.(*downloadRateLimitEntry)
		entry.mu.Lock()
		entry.lastSeen = now
		allowed := entry.limiter.Allow()
		entry.mu.Unlock()
		if !allowed {
			c.Header("Retry-After", "2")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": 7, "msg": "下载请求过于频繁，请稍后重试"})
			return
		}

		select {
		case downloadSlots <- struct{}{}:
			defer func() { <-downloadSlots }()
		default:
			c.Header("Retry-After", "2")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": 7, "msg": "当前下载人数较多，请稍后重试"})
			return
		}

		downloadRateLimiters.Range(func(key, value any) bool {
			if item, ok := value.(*downloadRateLimitEntry); ok {
				item.mu.Lock()
				stale := now.Sub(item.lastSeen) > downloadRateLimitEntryTTL
				item.mu.Unlock()
				if !stale {
					return true
				}
				downloadRateLimiters.Delete(key)
			}
			return true
		})
		c.Next()
	}
}
