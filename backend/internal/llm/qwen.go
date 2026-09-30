package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxImageBytes = 10 << 20

type qwenEditor struct {
	*imageClient
}

func (q *qwenEditor) Edit(ctx context.Context, input ImageInput) ([]byte, error) {
	if err := validateImageInput(input); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(q.config.TimeoutSeconds)*time.Second)
	defer cancel()
	if err := q.wait(ctx); err != nil {
		return nil, err
	}
	payload := struct {
		Model string `json:"model"`
		Input struct {
			Messages []message `json:"messages"`
		} `json:"input"`
		Parameters struct {
			N                int    `json:"n"`
			PromptExtend     bool   `json:"prompt_extend"`
			PromptExtendMode string `json:"prompt_extend_mode"`
			EnableThinking   bool   `json:"enable_thinking"`
			Watermark        bool   `json:"watermark"`
		} `json:"parameters"`
	}{Model: q.config.Model}
	payload.Input.Messages = []message{{Role: "user", Content: []content{{Image: "data:" + input.MIME + ";base64," + base64.StdEncoding.EncodeToString(input.Data)}, {Text: input.Prompt}}}}
	payload.Parameters.N = 1
	payload.Parameters.PromptExtend = true
	payload.Parameters.PromptExtendMode = "direct"
	payload.Parameters.EnableThinking = true
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("构建 AI 请求失败")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(q.config.BaseURL, "/")+"/services/aigc/multimodal-generation/generation", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构建 AI 请求失败")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+q.config.APIKey)
	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI 服务连接失败或超时，请检查服务配置")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI 服务返回 HTTP %d，请检查密钥、额度和模型权限", resp.StatusCode)
	}
	var result struct {
		RequestID string `json:"request_id"`
		Code      string `json:"code"`
		Usage     struct {
			InputImageCount  int `json:"input_image_count"`
			OutputImageCount int `json:"output_image_count"`
		} `json:"usage"`
		Output struct {
			Choices []struct {
				Message message `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("AI 服务响应无法解析")
	}
	if result.Code != "" {
		return nil, fmt.Errorf("AI 服务未完成图片编辑，请检查模型配置和服务额度")
	}
	slog.InfoContext(ctx, "Qwen 图片请求已返回", slog.String("model", q.config.Model), slog.String("request_id", result.RequestID), slog.Int("input_image_count", result.Usage.InputImageCount), slog.Int("output_image_count", result.Usage.OutputImageCount))
	for _, choice := range result.Output.Choices {
		for _, item := range choice.Message.Content {
			if item.Image != "" {
				return q.download(ctx, item.Image)
			}
		}
	}
	return nil, fmt.Errorf("AI 服务没有返回图片")
}

type message struct {
	Role    string    `json:"role"`
	Content []content `json:"content"`
}
type content struct {
	Image string `json:"image,omitempty"`
	Text  string `json:"text,omitempty"`
}

func (q *qwenEditor) download(ctx context.Context, address string) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return nil, fmt.Errorf("AI 返回的图片地址无效")
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, ".aliyuncs.com") && !strings.HasSuffix(host, ".alicdn.com") {
		return nil, fmt.Errorf("AI 返回的图片地址不是受支持的阿里云存储域名")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("构建 AI 图片下载请求失败")
	}
	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 AI 图片失败或超时")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 AI 图片失败，HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20+1))
	if err != nil || len(data) > 32<<20 {
		return nil, fmt.Errorf("AI 图片读取失败或超过 32MB")
	}
	return data, nil
}
