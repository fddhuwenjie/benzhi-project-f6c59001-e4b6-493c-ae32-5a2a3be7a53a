package idempotency_cache_key_collision

import (
	"errors"
	"testing"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func TestIdempotencyCacheKeysIncludeOperation(t *testing.T) {
	repo, err := storage.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo)
	create := workflow.CreateCaseCommand{
		RequestID: "shared-request", Actor: workflow.Actor{Name: "修复师甲", Role: workflow.RoleConservator},
		ID: "cache-key-case", ShelfMark: "A-1", Title: "幂等缓存测试", VersionIdentifier: "刻本",
		SupportMaterial: "竹纸", CarrierCharacteristics: "线装", DamageLocations: []string{"书口"},
		InitialEvidence: []conservation.EvidenceRef{{ID: "e1", Filename: "front.jpg", MediaType: "image/jpeg"}},
		ResponsibleConservator: "修复师甲",
	}
	created, err := service.CreateCase(create)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.SubmitForAssessment(created.Case.ID, workflow.CommandMeta{
		RequestID: "shared-request", ExpectedRevision: created.Case.Revision,
		Actor: workflow.Actor{Name: "修复师甲", Role: workflow.RoleConservator},
	})
	if !errors.Is(err, storage.ErrIdempotencyConflict) {
		t.Fatalf("跨操作复用 request_id 应返回冲突，得到 %v", err)
	}
	current, err := repo.Load(created.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != conservation.StatusDraft {
		t.Fatalf("冲突请求不应推进状态，当前状态为 %s", current.Status)
	}
}
