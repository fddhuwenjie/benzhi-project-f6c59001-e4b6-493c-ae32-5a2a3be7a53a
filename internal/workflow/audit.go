package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type ArchiveSummary struct {
	CaseID          string                       `json:"case_id"`
	Title           string                       `json:"title"`
	ShelfMark       string                       `json:"shelf_mark"`
	ProposalVersion string                       `json:"proposal_version"`
	ReviewCount     int                          `json:"review_count"`
	TrialBatch      string                       `json:"trial_batch"`
	ArchivedAt      *time.Time                   `json:"archived_at"`
	Timeline        []conservation.TimelineEntry `json:"timeline"`
	Precheck        *conservation.PrecheckResult `json:"precheck,omitempty"`
	EvidenceIndex   []conservation.EvidenceRef   `json:"evidence_index"`
	EventChain      EventChainSummary            `json:"event_chain"`
}

type EventChainSummary struct {
	EventCount    int                 `json:"event_count"`
	FirstRevision int64               `json:"first_revision"`
	LastRevision  int64               `json:"last_revision"`
	LastStatus    conservation.Status `json:"last_status"`
}

type AuditPackage struct {
	SchemaVersion string                         `json:"schema_version"`
	ExportedAt    time.Time                      `json:"exported_at"`
	Summary       ArchiveSummary                 `json:"summary"`
	Case          *conservation.ConservationCase `json:"case"`
	Events        []conservation.Event           `json:"events"`
	ContentSHA256 string                         `json:"content_sha256"`
}

func (s *Service) BuildAuditPackage(id string) (*AuditPackage, error) {
	view, err := s.buildCaseView(id)
	if err != nil {
		return nil, err
	}
	if view.Case.Status != conservation.StatusArchived {
		return nil, &conservation.TransitionError{From: view.Case.Status, Message: "仅已封存事项可导出审计包"}
	}
	events, err := s.repo.Events(id)
	if err != nil {
		return nil, err
	}
	evidence := append([]conservation.EvidenceRef(nil), view.Case.InitialEvidence...)
	if view.Case.Assessment != nil {
		evidence = append(evidence, view.Case.Assessment.EvidenceRefs...)
	}
	for _, trial := range view.Case.TrialBatches {
		evidence = append(evidence, trial.EvidenceRefs...)
	}
	summary := ArchiveSummary{CaseID: id, Title: view.Case.Title, ShelfMark: view.Case.ShelfMark, ReviewCount: len(view.Case.Reviews), ArchivedAt: view.Case.ArchivedAt, Timeline: view.Timeline, Precheck: view.Case.ArchivePrecheck, EvidenceIndex: evidence, EventChain: EventChainSummary{EventCount: len(events)}}
	if len(events) > 0 {
		summary.EventChain.FirstRevision = events[0].AfterRevision
		summary.EventChain.LastRevision = events[len(events)-1].AfterRevision
		summary.EventChain.LastStatus = events[len(events)-1].ToStatus
	}
	if view.Case.Proposal != nil {
		summary.ProposalVersion = view.Case.Proposal.ProposalVersion
	}
	if view.Case.Trial != nil {
		summary.TrialBatch = view.Case.Trial.BatchCode
	}
	pkg := &AuditPackage{SchemaVersion: "conservation-audit-v1", ExportedAt: s.clock().UTC(), Summary: summary, Case: view.Case, Events: events}
	payload, err := json.Marshal(struct {
		Case   *conservation.ConservationCase `json:"case"`
		Events []conservation.Event           `json:"events"`
	}{view.Case, events})
	if err != nil {
		return nil, fmt.Errorf("生成审计摘要: %w", err)
	}
	digest := sha256.Sum256(payload)
	pkg.ContentSHA256 = hex.EncodeToString(digest[:])
	return pkg, nil
}
