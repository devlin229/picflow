package configs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const megabyte int64 = 1 << 20

// Config 保存 PicFlow 运行所需的配置。
type Config struct {
	Name               string
	Env                string
	Host               string
	Port               int
	OTLPTraceEndpoint  string
	CORSAllowedOrigins string
	DataDir            string
	WebDir             string
	MaxUploadSize      int64
	WorkerCount        int
}

// Address 返回 HTTP 服务监听地址。
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// DatabasePath 返回 SQLite 数据库文件路径。
func (c *Config) DatabasePath() string {
	return filepath.Join(c.DataDir, "picflow.db")
}

// Load 从可选的 .env 文件和系统环境变量读取配置，系统环境变量优先级更高。
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	v.SetDefault("NAME", "picflow")
	v.SetDefault("ENV", "debug")
	v.SetDefault("HOST", "0.0.0.0")
	v.SetDefault("DATA_DIR", "./data")
	v.SetDefault("MAX_UPLOAD_SIZE_MB", 20)
	v.SetDefault("WORKER_COUNT", 2)

	if err := v.ReadInConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("读取 .env 失败: %w", err)
	}
	if strings.TrimSpace(v.GetString("PORT")) == "" {
		return nil, fmt.Errorf("校验 .env 失败: PORT 不能为空")
	}

	dataDir, err := filepath.Abs(v.GetString("DATA_DIR"))
	if err != nil {
		return nil, fmt.Errorf("解析 DATA_DIR 失败: %w", err)
	}
	webDir := strings.TrimSpace(v.GetString("WEB_DIR"))
	if webDir != "" {
		webDir, err = filepath.Abs(webDir)
		if err != nil {
			return nil, fmt.Errorf("解析 WEB_DIR 失败: %w", err)
		}
	}
	cfg := &Config{
		Name:               strings.TrimSpace(v.GetString("NAME")),
		Env:                strings.TrimSpace(v.GetString("ENV")),
		Host:               strings.TrimSpace(v.GetString("HOST")),
		Port:               v.GetInt("PORT"),
		OTLPTraceEndpoint:  strings.TrimSpace(v.GetString("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")),
		CORSAllowedOrigins: strings.TrimSpace(v.GetString("CORS_ALLOWED_ORIGINS")),
		DataDir:            dataDir,
		WebDir:             webDir,
		MaxUploadSize:      v.GetInt64("MAX_UPLOAD_SIZE_MB") * megabyte,
		WorkerCount:        v.GetInt("WORKER_COUNT"),
	}
	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("校验配置失败: %w", err)
	}
	return cfg, nil
}

// Validate 检查配置是否可以安全运行。
func (c *Config) Validate() error {
	switch c.Env {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("ENV 只能是 debug、release 或 test")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT 必须在 1 到 65535 之间")
	}
	if c.DataDir == "" {
		return fmt.Errorf("DATA_DIR 不能为空")
	}
	if c.MaxUploadSize < megabyte || c.MaxUploadSize > 100*megabyte {
		return fmt.Errorf("MAX_UPLOAD_SIZE_MB 必须在 1 到 100 之间")
	}
	if c.WorkerCount < 1 || c.WorkerCount > 16 {
		return fmt.Errorf("WORKER_COUNT 必须在 1 到 16 之间")
	}
	return nil
}
