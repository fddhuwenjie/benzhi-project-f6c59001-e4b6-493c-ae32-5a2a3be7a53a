package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := parseConfig(os.Args[1:], environment)
	if err != nil {
		logger.Error("配置无效", "error", err)
		os.Exit(2)
	}
	if cfg.selftest {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := runSelftest(ctx, logger); err != nil {
			logger.Error("自检失败", "error", err)
			os.Exit(1)
		}
		return
	}
	app, err := buildHandler(cfg.dataDir, logger)
	if err != nil {
		logger.Error("服务装配失败", "error", err)
		os.Exit(1)
	}
	server := web.HTTPServer(cfg.address, app.Handler())
	ctx, stop := signalContext()
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("服务关闭失败", "error", err)
		}
	}()
	logger.Info("服务开始监听", "addr", cfg.address, "data", cfg.dataDir)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP 服务退出", "error", err)
		os.Exit(1)
	}
}
