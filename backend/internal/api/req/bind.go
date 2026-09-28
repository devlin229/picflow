package req

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"picflow/backend/internal/errno"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Bind 将 URI、Query、Header 和 Body 中的参数统一绑定到 target，并在最后执行一次校验。
// Query 和表单字段使用 form 标签，URI 参数使用 uri 标签，请求头使用 header 标签。
func Bind(c *gin.Context, target any) error {
	if target == nil {
		return errno.InvalidArgument("request target is required")
	}

	if err := binding.MapFormWithTag(target, c.Request.URL.Query(), "form"); err != nil {
		return bindError(err)
	}
	if err := binding.MapFormWithTag(target, c.Request.Header, "header"); err != nil {
		return bindError(err)
	}
	if err := bindBody(c.Request, target); err != nil {
		return bindError(err)
	}
	if err := binding.MapFormWithTag(target, uriParams(c), "uri"); err != nil {
		return bindError(err)
	}
	if err := binding.Validator.ValidateStruct(target); err != nil {
		return bindError(err)
	}
	return nil
}

func bindBody(request *http.Request, target any) error {
	if request.Body == nil || request.Body == http.NoBody || request.ContentLength == 0 {
		return nil
	}

	mediaType := request.Header.Get("Content-Type")
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}

	switch {
	case mediaType == "", mediaType == "application/json", strings.HasSuffix(mediaType, "+json"):
		return decodeJSON(request.Body, target)
	case mediaType == "application/xml", mediaType == "text/xml", strings.HasSuffix(mediaType, "+xml"):
		return decodeXML(request.Body, target)
	case mediaType == "application/x-www-form-urlencoded":
		if err := request.ParseForm(); err != nil {
			return err
		}
		return binding.MapFormWithTag(target, request.PostForm, "form")
	case strings.HasPrefix(mediaType, "multipart/form-data"):
		if err := request.ParseMultipartForm(32 << 20); err != nil {
			return err
		}
		return binding.MapFormWithTag(target, request.MultipartForm.Value, "form")
	default:
		return fmt.Errorf("unsupported content type %q", mediaType)
	}
}

func decodeJSON(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body must contain only one JSON value")
		}
		return err
	}
	return nil
}

func decodeXML(body io.Reader, target any) error {
	if err := xml.NewDecoder(body).Decode(target); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func uriParams(c *gin.Context) map[string][]string {
	params := make(map[string][]string, len(c.Params))
	for _, param := range c.Params {
		params[param.Key] = []string{param.Value}
	}
	return params
}

func bindError(err error) error {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return errno.PayloadTooLarge()
	}
	return errno.Wrap(err, errno.CodeInvalidArgument, "invalid request parameters", http.StatusBadRequest)
}
