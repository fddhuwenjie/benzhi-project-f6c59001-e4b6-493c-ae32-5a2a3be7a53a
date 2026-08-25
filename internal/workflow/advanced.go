package workflow

import (
	"strings"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

func (s *Service) ConfigureReviewRoster(caseID string, command RosterCommand) (*CaseResult, error) {
	if err := command.Actor.Validate(RoleManager); err != nil {
		return nil, err
	}
	return s.change(caseID, command.CommandMeta, "configure_roster", "review.roster_configured", func(item *conservation.ConservationCase, now conservationTime) error {
		if err := requireRosterStatus(item.Status); err != nil {
			return err
		}
		roster := command.Roster
		if len(roster.Members) == 0 {
			roster.Members = roster.Experts
		}
		roster.ProposalVersion = strings.TrimSpace(roster.ProposalVersion)
		for i := range roster.Members {
			roster.Members[i].Name = strings.TrimSpace(roster.Members[i].Name)
			if roster.Members[i].Reason == "" {
				roster.Members[i].Reason = roster.Members[i].RecusalReason
			}
			roster.Members[i].Reason = strings.TrimSpace(roster.Members[i].Reason)
		}
		if item.Proposal != nil && roster.ProposalVersion == "" {
			roster.ProposalVersion = item.Proposal.ProposalVersion
		}
		if err := roster.Validate(item); err != nil {
			return err
		}
		roster.ConfiguredAt = now.Time.UTC()
		item.ReviewRoster = &roster
		item.Revision++
		item.UpdatedAt = now.Time.UTC()
		return nil
	})
}

func requireRosterStatus(status conservation.Status) error {
	if status != conservation.StatusPeerReview {
		return &conservation.TransitionError{From: status, Message: "仅待同行审议事项可配置专家名册"}
	}
	return nil
}

func (s *Service) BuildPrecheck(id string) (*conservation.PrecheckResult, error) {
	item, err := s.repo.Load(id)
	if err != nil {
		return nil, err
	}
	events, err := s.repo.Events(id)
	if err != nil {
		return nil, err
	}
	result := &conservation.PrecheckResult{CheckedAt: s.clock().UTC()}
	add := func(code, stage, message string, blocking, passed bool) {
		item := conservation.PrecheckItem{Code: code, Stage: stage, Message: message, Blocking: blocking, Passed: passed}
		result.Items = append(result.Items, item)
		if blocking && !passed {
			result.BlockingItems = append(result.BlockingItems, item)
		}
		if !blocking && !passed {
			result.HintItems = append(result.HintItems, item)
		}
	}
	add("assessment", "评估", "已确认分区损伤评估", true, item.Assessment != nil)
	add("proposal", "方案", "当前方案版本已提交", true, item.Proposal != nil)
	add("reviews", "审议", "有效审议意见达到法定人数且无保留/退回", true, reviewConclusion(item) == conservation.ConclusionApproved)
	add("trial", "小样", "存在通过且偏差已关闭的小样批次", true, item.Trial != nil && item.Trial.Verdict == conservation.TrialPassed)
	mediaOK := true
	hashesOK := true
	for _, ref := range item.InitialEvidence {
		if strings.TrimSpace(ref.MediaType) == "" {
			mediaOK = false
		}
		if strings.TrimSpace(ref.SHA256) == "" {
			hashesOK = false
		}
	}
	if item.Assessment != nil {
		for _, ref := range item.Assessment.EvidenceRefs {
			if strings.TrimSpace(ref.MediaType) == "" {
				mediaOK = false
			}
			if strings.TrimSpace(ref.SHA256) == "" {
				hashesOK = false
			}
		}
	}
	if item.Trial != nil {
		for _, ref := range item.Trial.EvidenceRefs {
			if strings.TrimSpace(ref.MediaType) == "" {
				mediaOK = false
			}
			if strings.TrimSpace(ref.SHA256) == "" {
				hashesOK = false
			}
		}
	}
	add("evidence.media_type", "证据", "阶段证据媒体类型完整", true, mediaOK)
	add("evidence.sha256", "证据", "部分证据未登记内容摘要，负责人需确认", false, hashesOK)
	chainOK := len(events) > 0
	expected := int64(1)
	last := time.Time{}
	lastStatus := conservation.Status("")
	requestIDs := map[string]bool{}
	for _, event := range events {
		if event.CaseID != id || event.BeforeRevision != expected-1 || event.AfterRevision != expected || (!last.IsZero() && event.OccurredAt.Before(last)) || strings.TrimSpace(event.RequestID) == "" || requestIDs[event.RequestID] || event.FromStatus != lastStatus {
			chainOK = false
		}
		requestIDs[event.RequestID] = true
		expected++
		last = event.OccurredAt
		lastStatus = event.ToStatus
	}
	if chainOK && len(events) > 0 && events[len(events)-1].AfterRevision != item.Revision {
		chainOK = false
	}
	add("event_chain", "审计链", "事件链与快照修订号一致", true, chainOK)
	if item.Status == conservation.StatusPendingApproval {
		item.ArchivePrecheck = result
	}
	return result, nil
}
