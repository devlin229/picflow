package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"picflow/backend/internal/errno"

	"github.com/gin-gonic/gin"
)

// Recovery 捕获 panic，在服务端记录完整信息，并向响应中间件传递脱敏错误。
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(
					c.Request.Context(),
					"panic recovered",
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)
				_ = c.Error(errno.Internal(fmt.Errorf("panic: %v", recovered)))
				c.Abort()
			}
		}()
		c.Next()
	}
}
