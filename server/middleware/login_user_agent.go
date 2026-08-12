package middleware

import (
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

var blockedLoginUserAgentFragments = []string{
	"curl",
	"wget",
	"python-requests",
	"python-urllib",
	"go-http-client",
	"java/",
	"httpclient",
	"libwww-perl",
	"sqlmap",
	"nikto",
	"nuclei",
	"zgrab",
	"masscan",
	"acunetix",
	"nessus",
	"wpscan",
	"dirbuster",
	"gobuster",
	"ffuf",
}

func LoginUserAgentGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isBlockedLoginUserAgent(c.Request.UserAgent()) {
			RecordSecurityFailure(c)
			response.FailWithMessage("用户名不存在或者密码错误", c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func isBlockedLoginUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return true
	}
	for _, fragment := range blockedLoginUserAgentFragments {
		if strings.Contains(ua, fragment) {
			return true
		}
	}
	return false
}
