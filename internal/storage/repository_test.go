package storage_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func TestSnapshotRecoveryAndRevisionConflict(t *testing.T) {
	root := t.TempDir()
	repo, err := storage.NewFileRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo)
	result, err := service.CreateCase(workflow.CreateCaseCommand{
		RequestID: "create", Actor: workflow.Actor{Name: "甲", Role: workflow.RoleConservator},
		ID: "recover-1", ShelfMark: "A-1", Title: "待恢复古籍", VersionIdentifier: "刻本",
		SupportMaterial: "竹纸", CarrierCharacteristics: "线装", DamageLocations: []string{"书口"},
		InitialEvidence:        []conservation.EvidenceRef{{ID: "e1", Filename: "e1.jpg", MediaType: "image/jpeg"}},
		ResponsibleConservator: "甲",
	})
	if err != nil {
		t.Fatal(err)
	}
	stale := *result.Case
	stale.Revision = 2
	stale.Status = conservation.StatusPendingAssessment
	stale.UpdatedAt = time.Now().UTC()
	event := conservation.Event{ID: "bad", CaseID: stale.ID}
	if err := repo.Save(&stale, 9, event); !errors.Is(err, conservation.ErrConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	snapshots, err := filepath.Glob(filepath.Join(root, "cases", "*.json"))
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("snapshot lookup: %v, count=%d", err, len(snapshots))
	}
	if err := os.Remove(snapshots[0]); err != nil {
		t.Fatal(err)
	}
	reopened, err := storage.NewFileRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := reopened.Load("recover-1")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Revision != 1 || restored.Title != "待恢复古籍" {
		t.Fatalf("unexpected restored item: %#v", restored)
	}
	events, err := reopened.Events("recover-1")
	if err != nil || len(events) != 1 {
		t.Fatalf("events after recovery: %v count=%d", err, len(events))
	}
}
