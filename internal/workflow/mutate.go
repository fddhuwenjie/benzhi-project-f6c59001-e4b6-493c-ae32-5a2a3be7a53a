package workflow

import (
	"strings"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type conservationTime struct{ Time time.Time }
type mutation func(*conservation.ConservationCase, conservationTime) error

func (s *Service) change(caseID string, meta CommandMeta, operation, eventType string, mutate mutation) (*CaseResult, error) {
	var cached CaseResult
	if replayed, err := s.replay(meta.RequestID, operation, caseID, &cached); err != nil {
		return nil, err
	} else if replayed {
		cached.Replayed = true
		return &cached, nil
	}
	item, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	if meta.Actor.Role == RoleConservator && strings.TrimSpace(meta.Actor.Name) != item.ResponsibleConservator {
		return nil, &AuthorizationError{Message: "仅档案中的责任修复师可执行此操作"}
	}
	beforeRevision, from := item.Revision, item.Status
	now := s.clock()
	if err := mutate(item, conservationTime{Time: now}); err != nil {
		return nil, err
	}
	event := makeEvent(item, meta, eventType, beforeRevision, from, map[string]any{"status_label": item.Status.Label()}, now)
	if err := s.repo.Save(item, beforeRevision, event); err != nil {
		return nil, err
	}
	result := &CaseResult{Case: item}
	if err := s.remember(meta.RequestID, operation, item.ID, result); err != nil {
		return nil, err
	}
	return result, nil
}
