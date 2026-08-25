package conservation

import "fmt"

type Status string

const (
	StatusDraft             Status = "draft"
	StatusPendingAssessment Status = "pending_assessment"
	StatusProposalDrafting  Status = "proposal_drafting"
	StatusPeerReview        Status = "peer_review"
	StatusPendingTrial      Status = "pending_trial"
	StatusPendingApproval   Status = "pending_approval"
	StatusArchived          Status = "archived"
)

var statusLabels = map[Status]string{
	StatusDraft:             "草稿",
	StatusPendingAssessment: "待评估",
	StatusProposalDrafting:  "方案编制中",
	StatusPeerReview:        "待同行审议",
	StatusPendingTrial:      "待小样核验",
	StatusPendingApproval:   "待最终批准",
	StatusArchived:          "已批准封存",
}

func (s Status) Valid() bool { _, ok := statusLabels[s]; return ok }

func (s Status) Label() string {
	if label, ok := statusLabels[s]; ok {
		return label
	}
	return string(s)
}

func requireStatus(actual Status, allowed ...Status) error {
	for _, status := range allowed {
		if actual == status {
			return nil
		}
	}
	return &TransitionError{From: actual, Message: fmt.Sprintf("当前状态“%s”不允许执行此操作", actual.Label())}
}
