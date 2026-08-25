package workflow

import (
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type CaseListItem struct {
	ID                     string              `json:"id"`
	ShelfMark              string              `json:"shelf_mark"`
	Title                  string              `json:"title"`
	ResponsibleConservator string              `json:"responsible_conservator"`
	Status                 conservation.Status `json:"status"`
	StatusLabel            string              `json:"status_label"`
	Revision               int64               `json:"revision"`
	UpdatedAt              time.Time           `json:"updated_at"`
	NextAction             string              `json:"next_action"`
}

type CaseView struct {
	Case             *conservation.ConservationCase `json:"case"`
	StatusLabel      string                         `json:"status_label"`
	ReviewConclusion conservation.ReviewConclusion  `json:"review_conclusion"`
	Timeline         []conservation.TimelineEntry   `json:"timeline"`
	AllowedActions   []string                       `json:"allowed_actions"`
	ReadOnly         bool                           `json:"read_only"`
	ReviewProgress   conservation.ReviewProgress    `json:"review_progress"`
	ArchivePrecheck  *conservation.PrecheckResult   `json:"archive_precheck,omitempty"`
}

func makeListItem(item *conservation.ConservationCase) CaseListItem {
	return CaseListItem{ID: item.ID, ShelfMark: item.ShelfMark, Title: item.Title,
		ResponsibleConservator: item.ResponsibleConservator, Status: item.Status,
		StatusLabel: item.Status.Label(), Revision: item.Revision, UpdatedAt: item.UpdatedAt,
		NextAction: nextAction(item.Status)}
}

func nextAction(status conservation.Status) string {
	switch status {
	case conservation.StatusDraft:
		return "补全档案并提交评估"
	case conservation.StatusPendingAssessment:
		return "负责人确认损伤评估"
	case conservation.StatusProposalDrafting:
		return "修复师编制方案"
	case conservation.StatusPeerReview:
		return "专家独立审议"
	case conservation.StatusPendingTrial:
		return "登记小样核验"
	case conservation.StatusPendingApproval:
		return "负责人批准封存"
	case conservation.StatusArchived:
		return "查看封存摘要"
	default:
		return ""
	}
}

func allowedActions(status conservation.Status) []string {
	switch status {
	case conservation.StatusDraft:
		return []string{"revise_draft", "submit_assessment"}
	case conservation.StatusPendingAssessment:
		return []string{"confirm_assessment"}
	case conservation.StatusProposalDrafting:
		return []string{"submit_proposal"}
	case conservation.StatusPeerReview:
		return []string{"configure_roster", "record_review"}
	case conservation.StatusPendingTrial:
		return []string{"verify_trial"}
	case conservation.StatusPendingApproval:
		return []string{"archive"}
	default:
		return []string{}
	}
}

func (s *Service) buildCaseView(id string) (*CaseView, error) {
	item, err := s.repo.Load(id)
	if err != nil {
		return nil, err
	}
	events, err := s.repo.Events(id)
	if err != nil {
		return nil, err
	}
	timeline := make([]conservation.TimelineEntry, 0, len(events))
	for _, event := range events {
		timeline = append(timeline, conservation.TimelineEntry{Type: event.Type, Label: eventLabel(event.Type), Actor: event.Actor,
			Role: event.Role, FromStatus: event.FromStatus, ToStatus: event.ToStatus, OccurredAt: event.OccurredAt})
	}
	var precheck *conservation.PrecheckResult
	if item.Status == conservation.StatusPendingApproval {
		precheck, _ = s.BuildPrecheck(id)
	} else if item.Status == conservation.StatusArchived {
		precheck = item.ArchivePrecheck
	}
	return &CaseView{Case: item, StatusLabel: item.Status.Label(), ReviewConclusion: reviewConclusion(item),
		Timeline: timeline, AllowedActions: allowedActions(item.Status), ReadOnly: item.Status == conservation.StatusArchived,
		ReviewProgress: item.ReviewProgress(), ArchivePrecheck: precheck}, nil
}

func reviewConclusion(item *conservation.ConservationCase) conservation.ReviewConclusion {
	return conservation.ReviewConclusionFor(item)
}

func eventLabel(eventType string) string {
	labels := map[string]string{"case.created": "建立损伤档案", "case.submitted_for_assessment": "提交专业评估", "assessment.confirmed": "确认损伤评估",
		"draft.revised": "修订草稿档案", "proposal.submitted": "提交修复方案", "review.roster_configured": "设置审议名册", "review.recorded": "记录同行审议", "trial.verified": "完成小样核验", "case.archived": "批准并封存"}
	if label := labels[eventType]; label != "" {
		return label
	}
	return eventType
}
