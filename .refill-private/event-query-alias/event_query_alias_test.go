package eventquery_test

import (
	"testing"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
)

func TestEventQueryDoesNotAliasCache(t *testing.T) {
	repo, err := storage.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	item := &conservation.ConservationCase{
		ID: "alias-case", Revision: 1, Title: "古籍", ShelfMark: "A-1",
		ResponsibleConservator: "修复师", Status: conservation.StatusDraft,
	}
	event := conservation.Event{
		ID: "evt-1", CaseID: item.ID, RequestID: "req-1", Type: "case.created",
		BeforeRevision: 0, AfterRevision: 1,
	}
	if err := repo.Create(item, event); err != nil {
		t.Fatal(err)
	}
	first, err := repo.Events(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 {
		t.Fatalf("expected one event, got %d", len(first))
	}
	first[0].Type = "tampered.by.caller"
	second, err := repo.Events(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Type != "case.created" {
		t.Fatalf("event query returned caller-mutated cached state: %#v", second)
	}
}
