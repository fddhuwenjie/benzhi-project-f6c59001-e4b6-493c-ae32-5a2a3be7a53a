package workflow

import (
	"errors"
	"testing"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
)

func TestIdempotencyAndResponsibleConservator(t *testing.T) {
	repo, err := storage.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo)
	command := CreateCaseCommand{
		RequestID: "req-create", Actor: Actor{Name: "修复师甲", Role: RoleConservator},
		ID: "case-idem", ShelfMark: "A", Title: "古籍", VersionIdentifier: "版本",
		SupportMaterial: "纸", CarrierCharacteristics: "线装", DamageLocations: []string{"书口"},
		InitialEvidence:        []conservation.EvidenceRef{{ID: "e", Filename: "e.jpg", MediaType: "image/jpeg"}},
		ResponsibleConservator: "修复师甲",
	}
	first, err := service.CreateCase(command)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateCase(command)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || second.Case.ID != first.Case.ID {
		t.Fatalf("idempotent replay not identified: %#v", second)
	}
	_, err = service.SubmitForAssessment(first.Case.ID, CommandMeta{
		RequestID: "unauthorized", ExpectedRevision: 1,
		Actor: Actor{Name: "另一位修复师", Role: RoleConservator},
	})
	var auth *AuthorizationError
	if !errors.As(err, &auth) {
		t.Fatalf("expected owner authorization error, got %v", err)
	}
	_, err = service.SubmitForAssessment(first.Case.ID, CommandMeta{
		RequestID: "submit", ExpectedRevision: 1,
		Actor: Actor{Name: "修复师甲", Role: RoleConservator},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SubmitForAssessment(first.Case.ID, CommandMeta{
		RequestID: "stale", ExpectedRevision: 1,
		Actor: Actor{Name: "修复师甲", Role: RoleConservator},
	})
	if !IsConflict(err) {
		t.Fatalf("expected stale revision conflict, got %v", err)
	}
}
