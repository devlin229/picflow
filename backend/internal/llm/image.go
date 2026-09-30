package llm

import "context"

// ImageInput 是图片编辑模型的输入，图片内容不包含文件路径。
type ImageInput struct {
	Data   []byte
	MIME   string
	Prompt string
}

// ImageEditor 隔离各供应商的图片编辑协议。
type ImageEditor interface {
	Edit(context.Context, ImageInput) ([]byte, error)
}

// Config 描述后端图片模型配置，密钥只保存在服务端。
type Config struct {
	Protocol          string
	APIKey            string
	BaseURL           string
	Model             string
	TimeoutSeconds    int
	RequestsPerMinute int
}
