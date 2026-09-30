package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type imageClient struct {
	config      Config
	client      *http.Client
	mu          sync.Mutex
	nextRequest time.Time
}

// NewImageEditor 根据协议创建图片编辑适配器；未配置密钥时关闭 AI 能力。
func NewImageEditor(config Config) (ImageEditor, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, nil
	}
	switch config.Protocol {
	case "qwen":
		if config.BaseURL == "" {
			config.BaseURL = "https://maas.qianwenaiapi.com/api/v1"
		}
		if config.Model == "" {
			config.Model = "qwen-image-3.0"
		}
	case "openai":
		if config.BaseURL == "" {
			config.BaseURL = "https://api.openai.com/v1"
		}
		if config.Model == "" {
			config.Model = "gpt-image-2.5-sunburst"
		}
	default:
		return nil, fmt.Errorf("LLM_PROTOCOL 仅支持 qwen 或 openai")
	}
	u, err := url.Parse(config.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("LLM_BASE_URL 必须是有效 HTTPS 基础地址")
	}
	if config.TimeoutSeconds < 1 || config.TimeoutSeconds > 1800 || config.RequestsPerMinute < 1 || config.RequestsPerMinute > 600 {
		return nil, fmt.Errorf("LLM 超时或速率配置无效")
	}
	client := &imageClient{config: config, client: &http.Client{Timeout: time.Duration(config.TimeoutSeconds) * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
	if config.Protocol == "openai" {
		return &openaiEditor{imageClient: client}, nil
	}
	return &qwenEditor{imageClient: client}, nil
}

func (c *imageClient) wait(ctx context.Context) error {
	c.mu.Lock()
	start := time.Now()
	if c.nextRequest.After(start) {
		start = c.nextRequest
	}
	c.nextRequest = start.Add(time.Minute / time.Duration(c.config.RequestsPerMinute))
	c.mu.Unlock()
	timer := time.NewTimer(time.Until(start))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("AI 请求等待超时")
	case <-timer.C:
		return nil
	}
}

func validateImageInput(input ImageInput) error {
	if len(input.Data) == 0 || len(input.Data) > maxImageBytes {
		return fmt.Errorf("AI 输入图片必须在 10MB 以内")
	}
	if input.MIME != "image/jpeg" && input.MIME != "image/png" && input.MIME != "image/webp" {
		return fmt.Errorf("AI 输入图片格式不受支持")
	}
	return nil
}
