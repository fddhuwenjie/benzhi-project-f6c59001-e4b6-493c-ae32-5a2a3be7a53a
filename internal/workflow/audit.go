package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

// validateEventChain checks that the persisted event log for a case forms a
// coherent integrity chain: every event references the given case, revisions
// advance sequentially, status transitions link end-to-end, timestamps are
// non-decreasing, request identifiers are unique, and the final event's
// revision matches the case snapshot revision.
func validateEventChain(caseID string, events []conservation.Event, item *conservation.ConservationCase) error {
	if len(events) == 0 {
		return fmt.Errorf("事件链为空，无法校验完整性")
	}
	expected := int64(1)
	last := time.Time{}
	lastStatus := conservation.Status("")
	requestIDs := map[string]bool{}
	for _, event := range events {
		switch {
		case event.CaseID != caseID:
			return fmt.Errorf("事件 %s 的事项标识与请求标识不一致", event.ID)
		case event.BeforeRevision != expected-1:
			return fmt.Errorf("事件 %s 的前置修订号 %d 与期望 %d 不连续", event.ID, event.BeforeRevision, expected-1)
		case event.AfterRevision != expected:
			return fmt.Errorf("事件 %s 的后置修订号 %d 与期望 %d 不连续", event.ID, event.AfterRevision, expected)
		case !last.IsZero() && event.OccurredAt.Before(last):
			return fmt.Errorf("事件 %s 的时间早于前一事件", event.ID)
		case strings.TrimSpace(event.RequestID) == "":
			return fmt.Errorf("事件 %s 的请求标识为空", event.ID)
		case requestIDs[event.RequestID]:
			return fmt.Errorf("事件 %s 的请求标识 %q 重复", event.ID, event.RequestID)
		case event.FromStatus != lastStatus:
			return fmt.Errorf("事件 %s 的起始状态与前一事件终态不一致", event.ID)
		}
		requestIDs[event.RequestID] = true
		expected++
		last = event.OccurredAt
		lastStatus = event.ToStatus
	}
	if events[len(events)-1].AfterRevision != item.Revision {
		return fmt.Errorf("末事件修订号 %d 与封存快照修订号 %d 不一致", events[len(events)-1].AfterRevision, item.Revision)
	}
	return nil
}

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
	if err := validateEventChain(id, events, view.Case); err != nil {
		return nil, fmt.Errorf("审计包完整性校验失败: %w", err)
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
