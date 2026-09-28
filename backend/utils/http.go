package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const defaultHTTPTimeout = 30 * time.Second

// HTTPClient 封装可复用的 net/http.Client，可以安全地在多个协程中使用。
type HTTPClient struct {
	client *http.Client
}

// HTTPRequest 是一次性的链式 HTTP 请求构建器，不应在多个协程中并发使用。
type HTTPRequest struct {
	client     *http.Client
	method     string
	requestURL string
	headers    map[string]string
	body       any
	sent       bool
}

// HTTPResponse 是普通 HTTP 请求的响应结果。
// Body 已经被完整读取，原始 response.Body 已由客户端关闭。
type HTTPResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// IsSuccess 判断 HTTP 状态码是否属于成功范围。
func (r *HTTPResponse) IsSuccess() bool {
	return r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices
}

// JSON 将响应内容反序列化到 target 中。
func (r *HTTPResponse) JSON(target any) error {
	if err := json.Unmarshal(r.Body, target); err != nil {
		return fmt.Errorf("decode HTTP response body: %w", err)
	}
	return nil
}

// NewHTTPClient 创建一个指定请求超时时间的客户端。
// timeout 小于或等于 0 时使用默认的 30 秒超时时间。
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &HTTPClient{client: &http.Client{
		Timeout:   timeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}}
}

// NewHTTPClientWithClient 封装已有的 http.Client。
// 需要自定义传输器、代理、TLS、重定向策略或 Cookie 时可以使用该方法。
func NewHTTPClientWithClient(client *http.Client) *HTTPClient {
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if _, instrumented := client.Transport.(*otelhttp.Transport); !instrumented {
		transport := client.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		client.Transport = otelhttp.NewTransport(transport)
	}
	return &HTTPClient{client: client}
}

// NewRequest 创建一个不携带历史状态的新请求构建器。
func (c *HTTPClient) NewRequest() *HTTPRequest {
	return &HTTPRequest{
		client:  c.client,
		headers: make(map[string]string),
	}
}

// Method 设置请求方法，该字段为必填项。
func (r *HTTPRequest) Method(method string) *HTTPRequest {
	r.method = method
	return r
}

// URL 设置请求地址，该字段为必填项。
func (r *HTTPRequest) URL(requestURL string) *HTTPRequest {
	r.requestURL = requestURL
	return r
}

// Header 设置请求头。多次调用会合并请求头，相同名称的请求头以后一次设置为准。
func (r *HTTPRequest) Header(headers map[string]string) *HTTPRequest {
	for key, value := range headers {
		r.headers[http.CanonicalHeaderKey(key)] = value
	}
	return r
}

// Body 设置请求体，支持 io.Reader、string、[]byte、url.Values，
// 以及能够序列化为 JSON 的任意类型。
func (r *HTTPRequest) Body(body any) *HTTPRequest {
	r.body = body
	return r
}

// Do 根据当前配置发送 HTTP 请求，必须在 Method 和 URL 设置完成后调用。
// 请求构建器只能发送一次，该方法会读取并关闭原始响应体。
func (r *HTTPRequest) Do(ctx context.Context) (*HTTPResponse, error) {
	response, err := r.send(ctx)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read HTTP response body: %w", err)
	}
	return &HTTPResponse{
		StatusCode: response.StatusCode,
		Header:     response.Header.Clone(),
		Body:       body,
	}, nil
}

// DoStream 根据当前配置发送流式请求，请求构建器只能发送一次。
// 调用方读取完成后必须自行关闭返回值的 Body。
func (r *HTTPRequest) DoStream(ctx context.Context) (*http.Response, error) {
	return r.send(ctx)
}

func (r *HTTPRequest) send(ctx context.Context) (*http.Response, error) {
	if strings.TrimSpace(r.method) == "" {
		return nil, fmt.Errorf("HTTP request method is required")
	}
	if strings.TrimSpace(r.requestURL) == "" {
		return nil, fmt.Errorf("HTTP request URL is required")
	}
	if r.sent {
		return nil, fmt.Errorf("HTTP request has already been sent")
	}

	requestBody, contentType, err := buildRequestBody(r.body)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, r.method, r.requestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	for key, value := range r.headers {
		request.Header.Set(key, value)
	}
	if contentType != "" && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", contentType)
	}

	r.sent = true
	response, err := r.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send HTTP request: %w", err)
	}
	return response, nil
}

func buildRequestBody(body any) (io.Reader, string, error) {
	switch value := body.(type) {
	case nil:
		return nil, "", nil
	case io.Reader:
		return value, "", nil
	case string:
		return strings.NewReader(value), "", nil
	case []byte:
		return bytes.NewReader(value), "", nil
	case url.Values:
		return strings.NewReader(value.Encode()), "application/x-www-form-urlencoded", nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, "", fmt.Errorf("encode HTTP request body: %w", err)
		}
		return bytes.NewReader(data), "application/json", nil
	}
}
