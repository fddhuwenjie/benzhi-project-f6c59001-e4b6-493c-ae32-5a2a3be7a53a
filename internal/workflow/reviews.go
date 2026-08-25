package workflow

import "strings"

func (s *Service) RecordReview(caseID string, command ReviewCommand) (*CaseResult, error) {
	if err := command.Actor.Validate(RoleReviewer); err != nil {
		return nil, err
	}
	if strings.TrimSpace(command.Actor.Name) != strings.TrimSpace(command.Review.Reviewer) {
		return nil, &AuthorizationError{Message: "专家只能以本人身份提交意见"}
	}
	var cached CaseResult
	if replayed, err := s.replay(command.RequestID, "record_review", caseID, &cached); err != nil {
		return nil, err
	} else if replayed {
		cached.Replayed = true
		return &cached, nil
	}
	item, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	beforeRevision, from := item.Revision, item.Status
	now := s.clock()
	conclusion, err := item.RecordReview(command.ExpectedRevision, command.Review, now)
	if err != nil {
		return nil, err
	}
	event := makeEvent(item, command.CommandMeta, "review.recorded", beforeRevision, from, map[string]any{"decision": command.Review.Decision, "conclusion": conclusion}, now)
	if err := s.repo.Save(item, beforeRevision, event); err != nil {
		return nil, err
	}
	result := &CaseResult{Case: item, Conclusion: conclusion}
	if err := s.remember(command.RequestID, "record_review", caseID, result); err != nil {
		return nil, err
	}
	return result, nil
}
