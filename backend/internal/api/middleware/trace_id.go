package middleware

import (
	"picflow/backend/internal/api/resp"

	"github.com/gin-gonic/gin"
)

// TraceID 在响应写入前设置当前请求的 Trace ID 响应头。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp.SetTraceID(c)
		c.Next()
	}
}
