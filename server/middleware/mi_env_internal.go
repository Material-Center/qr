package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
)

const MIEnvInternalKeyHeader = "X-MI-Internal-Key"

const DefaultMIEnvInternalKey = "cd5d1c1b4bd95fcb561d2a3f2b5407de82b9088d8b1f4eb3ebce9c28331ef42f"

func MIEnvInternalKeyGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(global.GVA_CONFIG.Extra.MIEnvInternalKey)
		if expected == "" {
			expected = DefaultMIEnvInternalKey
		}
		provided := strings.TrimSpace(c.GetHeader(MIEnvInternalKeyHeader))
		if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "msg": "invalid internal key"})
			c.Abort()
			return
		}
		c.Next()
	}
}
