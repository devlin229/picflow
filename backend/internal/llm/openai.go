package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

type openaiEditor struct{ *imageClient }

func (o *openaiEditor) Edit(ctx context.Context, input ImageInput) ([]byte, error) {
	if err := validateImageInput(input); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(o.config.TimeoutSeconds)*time.Second)
	defer cancel()
	if err := o.wait(ctx); err != nil {
		return nil, err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	// 只生成一张 PNG；质量使用 medium 控制费用，之后沿用本地尺寸和规格处理。
	fields := map[string]string{"model": o.config.Model, "prompt": input.Prompt, "n": "1", "quality": "medium", "size": "auto", "output_format": "png"}
	// 旧版图片模型需要显式启用高保真；2/2.5 系列默认高保真，不接受该参数。
	if o.config.Model == "gpt-image-1" || strings.HasPrefix(o.config.Model, "gpt-image-1.5") {
		fields["input_fidelity"] = "high"
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("构建 OpenAI 图片编辑请求失败")
		}
	}
	ext := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}[input.MIME]
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="image[]"; filename="source.`+ext+`"`)
	header.Set("Content-Type", input.MIME)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("构建 OpenAI 图片上传失败")
	}
	if _, err = part.Write(input.Data); err != nil {
		return nil, fmt.Errorf("构建 OpenAI 图片上传失败")
	}
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("构建 OpenAI 图片编辑请求失败")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(o.config.BaseURL, "/")+"/images/edits", &body)
	if err != nil {
		return nil, fmt.Errorf("构建 OpenAI 图片编辑请求失败")
	}
	req.Header.Set("Authorization", "Bearer "+o.config.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI 服务连接失败或超时，请检查服务配置；请求可能已计费")
	}
	defer resp.Body.Close()
	requestID := resp.Header.Get("x-request-id")
	slog.InfoContext(ctx, "OpenAI 图片请求已返回", slog.String("model", o.config.Model), slog.String("request_id", requestID), slog.Int("http_status", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case 400:
			return nil, fmt.Errorf("OpenAI 图片编辑参数无效，请检查模型名称和图片格式")
		case 401, 403:
			return nil, fmt.Errorf("OpenAI 密钥或模型权限不可用，请检查账户及模型访问权限")
		case 429:
			return nil, fmt.Errorf("OpenAI 额度不足或请求过于频繁，请检查额度和速率配置")
		default:
			return nil, fmt.Errorf("OpenAI 服务返回 HTTP %d，不会自动重试付费请求", resp.StatusCode)
		}
	}
	// 32MB 原图的 Base64 加 JSON 开销；先限制响应，再解码，避免无限分配内存。
	data, err := io.ReadAll(io.LimitReader(resp.Body, (48<<20)+1))
	if err != nil || len(data) > 48<<20 {
		return nil, fmt.Errorf("OpenAI 图片响应读取失败或超过限制")
	}
	var result struct {
		Data []struct {
			Base64 string `json:"b64_json"`
		} `json:"data"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("OpenAI 图片响应无法解析")
	}
	if result.Error != nil || len(result.Data) != 1 || result.Data[0].Base64 == "" {
		return nil, fmt.Errorf("OpenAI 未返回单张 Base64 图片，请检查服务是否支持 GPT Image 图片编辑协议")
	}
	if base64.StdEncoding.DecodedLen(len(result.Data[0].Base64)) > (32<<20)+2 {
		return nil, fmt.Errorf("OpenAI 返回图片超过 32MB")
	}
	image, err := base64.StdEncoding.DecodeString(result.Data[0].Base64)
	if err != nil || len(image) == 0 || len(image) > 32<<20 {
		return nil, fmt.Errorf("OpenAI 返回图片无效或超过 32MB")
	}
	slog.InfoContext(ctx, "OpenAI 图片结果已解码", slog.String("request_id", requestID), slog.Int("input_tokens", result.Usage.InputTokens), slog.Int("output_tokens", result.Usage.OutputTokens))
	return image, nil
}
