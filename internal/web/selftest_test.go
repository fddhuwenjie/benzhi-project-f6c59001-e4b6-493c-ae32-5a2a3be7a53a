package web

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func TestRunSelfTest(t *testing.T) {
	repo, err := storage.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer(workflow.NewService(repo), logger)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := RunSelfTest(ctx, server.Handler()); err != nil {
		t.Fatal(err)
	}
}
