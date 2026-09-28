package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 为明确配置的来源添加跨域响应头；空配置保持同源策略。
func CORS(allowedOrigins string) gin.HandlerFunc {
	allowed := make(map[string]struct{})
	for _, origin := range strings.Split(allowedOrigins, ",") {
		if value := strings.TrimSpace(origin); value != "" {
			allowed[value] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		_, exact := allowed[origin]
		_, wildcard := allowed["*"]
		if exact || wildcard {
			value := origin
			if wildcard {
				value = "*"
			}
			c.Header("Access-Control-Allow-Origin", value)
			c.Header("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type,X-Trace-ID")
			c.Header("Access-Control-Expose-Headers", "X-Trace-ID,Content-Disposition")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			if exact || wildcard {
				c.AbortWithStatus(http.StatusNoContent)
			} else {
				c.AbortWithStatus(http.StatusForbidden)
			}
			return
		}
		c.Next()
	}
}
