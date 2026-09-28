package logger

import (
	"io"
	"log/slog"
	"os"
)

const (
	defaultLogDirectory = "logs"
	logRetentionDays    = 30
)

// New 创建同时写入标准输出和按天轮转文件的 JSON 日志器。
func New(serviceName, environment string) (*slog.Logger, io.Closer, error) {
	fileWriter, err := NewDailyWriter(defaultLogDirectory, logRetentionDays)
	if err != nil {
		return nil, nil, err
	}

	level := slog.LevelInfo
	if environment == "debug" {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(
		io.MultiWriter(fileWriter, os.Stdout),
		&slog.HandlerOptions{
			AddSource: environment == "debug",
			Level:     level,
		},
	)
	log := slog.New(NewTraceHandler(handler)).With(
		slog.String("service", serviceName),
		slog.String("environment", environment),
	)
	return log, fileWriter, nil
}
