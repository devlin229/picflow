package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"picflow/backend/configs"
	"picflow/backend/internal/api"
	appLogger "picflow/backend/internal/logger"
	"picflow/backend/internal/svc"
	"picflow/backend/internal/telemetry"

	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "启动 PicFlow HTTP 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}
}

func runServer() error {
	cfg, err := configs.Load()
	if err != nil {
		return err
	}
	logger, logCloser, err := appLogger.New(cfg.Name, cfg.Env)
	if err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}
	slog.SetDefault(logger)
	defer func() {
		if closeErr := logCloser.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "关闭日志文件失败: %v\n", closeErr)
		}
	}()

	shutdownTracing, err := telemetry.Init(context.Background(), cfg.Name, cfg.Env, cfg.OTLPTraceEndpoint)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := shutdownTracing(shutdownCtx); shutdownErr != nil {
			logger.Error("关闭链路追踪失败", slog.Any("error", shutdownErr))
		}
	}()

	svcCtx, err := svc.NewServiceContext(cfg, logger)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := svcCtx.Close(); closeErr != nil {
			logger.Error("关闭服务依赖失败", slog.Any("error", closeErr))
		}
	}()

	router := api.NewRouter(svcCtx)
	server := &http.Server{
		Addr: cfg.Address(), Handler: router, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 60 * time.Second, WriteTimeout: 2 * time.Minute, IdleTimeout: 60 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	logger.Info("PicFlow HTTP 服务已启动", slog.String("address", cfg.Address()))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case err = <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("启动 HTTP 服务失败: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("正在关闭 PicFlow HTTP 服务")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("关闭 HTTP 服务失败: %w", err)
	}
	return nil
}
