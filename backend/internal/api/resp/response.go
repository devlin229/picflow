package resp

import (
	"net/http"

	"picflow/backend/internal/errno"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

const TraceIDHeader = "X-Trace-ID"

// Response 是所有 JSON 接口共用的响应结构。
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func OK(c *gin.Context, data any) {
	SetTraceID(c)
	c.JSON(http.StatusOK, Response{Code: errno.CodeSuccess, Msg: "success", Data: data})
}

func Created(c *gin.Context, data any) {
	SetTraceID(c)
	c.JSON(http.StatusCreated, Response{Code: errno.CodeSuccess, Msg: "success", Data: data})
}

func Fail(c *gin.Context, err error) {
	apiErr := errno.From(err)
	SetTraceID(c)
	c.JSON(apiErr.HTTPStatus, Response{Code: apiErr.Code, Msg: apiErr.Message, Data: nil})
}

// SetTraceID 将当前请求的 Trace ID 写入响应头，业务响应体保持 code、msg、data 三个字段。
func SetTraceID(c *gin.Context) {
	spanContext := trace.SpanContextFromContext(c.Request.Context())
	if spanContext.IsValid() {
		c.Header(TraceIDHeader, spanContext.TraceID().String())
	}
}
