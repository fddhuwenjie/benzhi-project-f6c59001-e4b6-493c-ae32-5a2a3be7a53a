package torn_event_tail_recovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func TestRecoveryIgnoresTornFinalEventRecord(t *testing.T) {
	root := t.TempDir()
	repo, err := storage.NewFileRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo)
	_, err = service.CreateCase(workflow.CreateCaseCommand{
		RequestID:              "create-torn-tail",
		Actor:                  workflow.Actor{Name: "修复师甲", Role: workflow.RoleConservator},
		ID:                     "torn-tail-case",
		ShelfMark:              "A-1",
		Title:                  "事件短写恢复测试",
		VersionIdentifier:      "刻本",
		SupportMaterial:        "竹纸",
		CarrierCharacteristics: "线装",
		DamageLocations:        []string{"书口"},
		InitialEvidence: []conservation.EvidenceRef{{
			ID: "evidence-1", Filename: "damage.jpg", MediaType: "image/jpeg",
		}},
		ResponsibleConservator: "修复师甲",
	})
	if err != nil {
		t.Fatal(err)
	}

	logs, err := filepath.Glob(filepath.Join(root, "events", "*.jsonl"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("event log lookup failed: err=%v count=%d", err, len(logs))
	}
	logFile, err := os.OpenFile(logs[0], os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := logFile.Write([]byte(`{"id":"partially-written`)); err != nil {
		logFile.Close()
		t.Fatal(err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, err := storage.NewFileRepository(root)
	if err != nil {
		t.Fatalf("repository restart rejected a valid snapshot followed by a torn final event: %v", err)
	}
	item, err := recovered.Load("torn-tail-case")
	if err != nil {
		t.Fatal(err)
	}
	if item.Revision != 1 || item.Status != conservation.StatusDraft {
		t.Fatalf("unexpected recovered snapshot: revision=%d status=%s", item.Revision, item.Status)
	}
}
