package errno

import (
	"errors"
	"fmt"
	"net/http"
)

// Error 是可以安全返回给客户端的业务错误。
type Error struct {
	Code       int
	Message    string
	HTTPStatus int
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

func New(code int, message string, httpStatus int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: httpStatus}
}

func Wrap(err error, code int, message string, httpStatus int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: httpStatus, Cause: err}
}

func InvalidArgument(message string) *Error {
	return New(CodeInvalidArgument, message, http.StatusBadRequest)
}

func Forbidden(message string) *Error {
	return New(CodeForbidden, message, http.StatusForbidden)
}

func NotFound(message string) *Error {
	return New(CodeNotFound, message, http.StatusNotFound)
}

func MethodNotAllowed() *Error {
	return New(CodeMethodNotAllowed, "请求方法不允许", http.StatusMethodNotAllowed)
}

func Conflict(message string) *Error {
	return New(CodeConflict, message, http.StatusConflict)
}

func PayloadTooLarge() *Error {
	return New(CodePayloadTooLarge, "请求内容超过大小限制", http.StatusRequestEntityTooLarge)
}

func Unavailable(err error) *Error {
	return Wrap(err, CodeUnavailable, "服务暂时不可用", http.StatusServiceUnavailable)
}

func Internal(err error) *Error {
	return Wrap(err, CodeInternal, "服务器内部错误", http.StatusInternalServerError)
}

// From 将未知错误转换为脱敏后的内部错误。
func From(err error) *Error {
	var target *Error
	if errors.As(err, &target) {
		return target
	}
	return Internal(err)
}
