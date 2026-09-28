package middleware

import (
	"log/slog"

	"picflow/backend/internal/api/resp"
	"picflow/backend/internal/errno"

	"github.com/gin-gonic/gin"
)

// Response 将 c.Error 中记录的错误转换为统一响应。
func Response(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		if errno.From(err).Code == errno.CodeInternal {
			logger.ErrorContext(
				c.Request.Context(),
				"request failed",
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
				slog.Any("error", err),
			)
		}
		resp.Fail(c, err)
	}
}
