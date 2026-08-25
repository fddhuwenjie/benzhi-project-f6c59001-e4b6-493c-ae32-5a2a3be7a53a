package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
)

type Clock func() time.Time

type Service struct {
	repo  storage.Repository
	clock Clock
}

func NewService(repo storage.Repository) *Service {
	return &Service{repo: repo, clock: time.Now}
}

func NewServiceWithClock(repo storage.Repository, clock Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

func (s *Service) Repository() storage.Repository { return s.repo }

func validateRequest(requestID string) error {
	if strings.TrimSpace(requestID) == "" {
		return &conservation.ValidationError{Issues: []conservation.FieldIssue{{Field: "request_id", Message: "不能为空"}}}
	}
	if len(requestID) > 128 {
		return &conservation.ValidationError{Issues: []conservation.FieldIssue{{Field: "request_id", Message: "不能超过 128 个字符"}}}
	}
	return nil
}

func newID(prefix string) string {
	data := make([]byte, 8)
	if _, err := rand.Read(data); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(data)
}

func (s *Service) replay(requestID, operation, caseID string, target any) (bool, error) {
	if err := validateRequest(requestID); err != nil {
		return false, err
	}
	record, err := s.repo.GetIdempotency(requestID)
	if err != nil {
		return false, err
	}
	if record == nil {
		return false, nil
	}
	if record.Operation != operation || caseID != "" && record.CaseID != caseID {
		return false, storage.ErrIdempotencyConflict
	}
	if err := json.Unmarshal(record.Result, target); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) remember(requestID, operation, caseID string, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return s.repo.PutIdempotency(storage.IdempotencyRecord{RequestID: requestID, Operation: operation, CaseID: caseID, Result: data, CompletedAt: s.clock().UTC()})
}

func makeEvent(item *conservation.ConservationCase, meta CommandMeta, eventType string, beforeRevision int64, from conservation.Status, details any, now time.Time) conservation.Event {
	data, _ := json.Marshal(struct {
		Data     any                            `json:"data,omitempty"`
		Snapshot *conservation.ConservationCase `json:"snapshot"`
	}{Data: details, Snapshot: item})
	return conservation.Event{ID: newID("evt"), CaseID: item.ID, RequestID: meta.RequestID, Type: eventType,
		Actor: strings.TrimSpace(meta.Actor.Name), Role: string(meta.Actor.Role), FromStatus: from, ToStatus: item.Status,
		BeforeRevision: beforeRevision, AfterRevision: item.Revision, OccurredAt: now.UTC(), Details: data}
}

func IsConflict(err error) bool {
	return errors.Is(err, conservation.ErrConflict) || errors.Is(err, storage.ErrIdempotencyConflict)
}

func (s *Service) ReviseDraft(caseID string, command DraftRevisionCommand) (*CaseResult, error) {
	if err := command.Actor.Validate(RoleConservator); err != nil {
		return nil, err
	}
	var cached CaseResult
	if replayed, err := s.replay(command.RequestID, "revise_draft", caseID, &cached); err != nil {
		return nil, err
	} else if replayed {
		cached.Replayed = true
		return &cached, nil
	}
	item, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(command.Actor.Name) != item.ResponsibleConservator {
		return nil, &AuthorizationError{Message: "仅档案中的责任修复师可修订草稿"}
	}
	before, from := item.Revision, item.Status
	now := s.clock()
	revision := command.Revision
	if revision.Title == "" && command.Draft.Title != "" {
		revision = command.Draft
	}
	if revision.Title == "" && command.Title != "" {
		revision = conservation.DraftRevision{Title: command.Title, VersionIdentifier: command.VersionIdentifier, CarrierCharacteristics: command.CarrierCharacteristics, DamageLocations: command.DamageLocations, InitialEvidence: command.InitialEvidence}
	}
	changes, err := item.ReviseDraft(command.ExpectedRevision, revision, now)
	if err != nil {
		return nil, err
	}
	event := makeEvent(item, command.CommandMeta, "draft.revised", before, from, map[string]any{"changed_fields": changes, "before_revision": before, "after_revision": item.Revision}, now)
	if err := s.repo.Save(item, before, event); err != nil {
		return nil, err
	}
	result := &CaseResult{Case: item}
	if err := s.remember(command.RequestID, "revise_draft", caseID, result); err != nil {
		return nil, err
	}
	return result, nil
}
