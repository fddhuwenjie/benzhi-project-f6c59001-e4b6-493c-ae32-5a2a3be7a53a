package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/web"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func buildHandler(dataDir string, logger *slog.Logger) (*web.Server, error) {
	repo, err := storage.NewFileRepository(dataDir)
	if err != nil {
		return nil, fmt.Errorf("初始化仓储: %w", err)
	}
	service := workflow.NewService(repo)
	return web.NewServer(service, logger), nil
}

func runSelftest(ctx context.Context, logger *slog.Logger) error {
	dataDir, err := os.MkdirTemp("", "benzhi-conservation-selftest-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dataDir)
	app, err := buildHandler(dataDir, logger)
	if err != nil {
		return err
	}
	if err := web.RunSelfTest(ctx, app.Handler()); err != nil {
		return err
	}
	logger.Info("完整业务流程自检通过")
	return nil
}
