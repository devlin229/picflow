package middleware

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"picflow/backend/internal/errno"

	"github.com/gin-gonic/gin"
)

const DefaultBodyLimit int64 = 1 << 20

// RouteBodyLimit 用于覆盖指定请求方法和路由的请求体大小限制。
type RouteBodyLimit struct {
	Method string
	Path   string
	Limit  int64
}

// BodyLimit 限制请求体大小，默认限制可以通过 overrides 为指定路由单独覆盖。
func BodyLimit(defaultLimit int64, overrides ...RouteBodyLimit) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := routeBodyLimit(c, defaultLimit, overrides)
		if limit <= 0 {
			c.Next()
			return
		}

		if c.Request.ContentLength > limit {
			_ = c.Error(errno.PayloadTooLarge())
			c.Abort()
			return
		}

		body := &limitedRequestBody{
			ReadCloser: http.MaxBytesReader(c.Writer, c.Request.Body, limit),
		}
		c.Request.Body = body
		c.Next()

		if body.exceeded && !c.Writer.Written() {
			_ = c.Error(errno.PayloadTooLarge())
			c.Abort()
		}
	}
}

func routeBodyLimit(c *gin.Context, defaultLimit int64, overrides []RouteBodyLimit) int64 {
	for _, override := range overrides {
		if strings.EqualFold(override.Method, c.Request.Method) && override.Path == c.FullPath() {
			return override.Limit
		}
	}
	return defaultLimit
}

type limitedRequestBody struct {
	io.ReadCloser
	exceeded bool
}

func (b *limitedRequestBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		b.exceeded = true
	}
	return n, err
}
