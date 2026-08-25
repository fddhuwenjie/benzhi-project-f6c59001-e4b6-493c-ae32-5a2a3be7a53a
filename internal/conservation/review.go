package conservation

import (
	"strings"
	"time"
)

type ReviewDecision string

const (
	DecisionApprove     ReviewDecision = "approve"
	DecisionReservation ReviewDecision = "reservation"
	DecisionReturn      ReviewDecision = "return"
)

type ReviewConclusion string

const (
	ConclusionPending     ReviewConclusion = "pending"
	ConclusionApproved    ReviewConclusion = "approved"
	ConclusionReservation ReviewConclusion = "reservation"
	ConclusionReturned    ReviewConclusion = "returned"
)

type PeerReview struct {
	ID              string         `json:"id"`
	CaseID          string         `json:"case_id"`
	ProposalVersion string         `json:"proposal_version"`
	Reviewer        string         `json:"reviewer"`
	Decision        ReviewDecision `json:"decision"`
	Reservations    string         `json:"reservations,omitempty"`
	ReturnReason    string         `json:"return_reason,omitempty"`
	RecordedAt      time.Time      `json:"recorded_at"`
}

type RosterMember struct {
	Name          string `json:"name"`
	Recused       bool   `json:"recused"`
	Reason        string `json:"reason,omitempty"`
	RecusalReason string `json:"recusal_reason,omitempty"`
}

type ReviewRoster struct {
	ProposalVersion string         `json:"proposal_version"`
	Members         []RosterMember `json:"members"`
	Experts         []RosterMember `json:"experts,omitempty"`
	Quorum          int            `json:"quorum"`
	ConfiguredAt    time.Time      `json:"configured_at"`
}

type ReviewProgress struct {
	Submitted int `json:"submitted"`
	Pending   int `json:"pending"`
	Recused   int `json:"recused"`
	Quorum    int `json:"quorum"`
	Remaining int `json:"remaining"`
}

func (r ReviewRoster) Validate(c *ConservationCase) error {
	v := &ValidationError{}
	members := r.Members
	if len(members) == 0 {
		members = r.Experts
	}
	if len(members) < 2 {
		v.Add("members", "专家名册至少需要两人")
	}
	active := 0
	for _, member := range members {
		if !member.Recused {
			active++
		}
	}
	if r.Quorum < 2 || r.Quorum > active {
		v.Add("quorum", "法定人数必须不小于 2 且不超过名册人数")
	}
	seen := map[string]bool{}
	for i, member := range members {
		name := strings.TrimSpace(member.Name)
		if name == "" {
			v.Add("members["+itoa(i)+"].name", "专家姓名不能为空")
		}
		if seen[name] {
			v.Add("members["+itoa(i)+"].name", "专家不能重名")
		}
		seen[name] = true
		if name == c.ResponsibleConservator {
			v.Add("members["+itoa(i)+"].name", "责任修复师不能进入名册")
		}
		reason := member.Reason
		if reason == "" {
			reason = member.RecusalReason
		}
		if member.Recused && strings.TrimSpace(reason) == "" {
			v.Add("members["+itoa(i)+"].reason", "回避必须说明原因")
		}
	}
	if r.ProposalVersion != "" && (c.Proposal == nil || r.ProposalVersion != c.Proposal.ProposalVersion) {
		v.Add("proposal_version", "必须对应当前方案版本")
	}
	return v.OrNil()
}

func (c *ConservationCase) ReviewProgress() ReviewProgress {
	if c.ReviewRoster == nil {
		submitted := len(c.Reviews)
		pending := 2 - submitted
		if pending < 0 {
			pending = 0
		}
		return ReviewProgress{Submitted: submitted, Pending: pending, Quorum: 2, Remaining: pending}
	}
	progress := ReviewProgress{Quorum: c.ReviewRoster.Quorum}
	seen := map[string]bool{}
	members := c.ReviewRoster.Members
	if len(members) == 0 {
		members = c.ReviewRoster.Experts
	}
	for _, member := range members {
		if member.Recused {
			progress.Recused++
			continue
		}
		if !seen[member.Name] {
			progress.Pending++
		}
		seen[member.Name] = false
	}
	for _, review := range c.Reviews {
		if _, ok := seen[review.Reviewer]; ok {
			progress.Submitted++
			seen[review.Reviewer] = true
		}
	}
	progress.Pending = len(members) - progress.Recused - progress.Submitted
	if progress.Pending < 0 {
		progress.Pending = 0
	}
	progress.Remaining = progress.Quorum - progress.Submitted
	if progress.Remaining < 0 {
		progress.Remaining = 0
	}
	return progress
}

func (r PeerReview) Validate(c *ConservationCase) error {
	v := &ValidationError{}
	required(v, "id", r.ID)
	required(v, "reviewer", r.Reviewer)
	if r.Decision != DecisionApprove && r.Decision != DecisionReservation && r.Decision != DecisionReturn {
		v.Add("decision", "必须为 approve、reservation 或 return")
	}
	if r.Decision == DecisionReservation {
		required(v, "reservations", r.Reservations)
	}
	if r.Decision == DecisionReturn {
		required(v, "return_reason", r.ReturnReason)
	}
	if c.Proposal == nil || r.ProposalVersion != c.Proposal.ProposalVersion {
		v.Add("proposal_version", "必须对应当前方案版本")
	}
	for _, existing := range c.Reviews {
		if existing.Reviewer == strings.TrimSpace(r.Reviewer) {
			v.Add("reviewer", "该专家已提交当前方案意见")
		}
		if existing.ID == strings.TrimSpace(r.ID) {
			v.Add("id", "审议记录标识已存在")
		}
	}
	return v.OrNil()
}

func SummarizeReviews(reviews []PeerReview) ReviewConclusion {
	approvals := 0
	reservations := 0
	for _, review := range reviews {
		switch review.Decision {
		case DecisionReturn:
			return ConclusionReturned
		case DecisionApprove:
			approvals++
		case DecisionReservation:
			reservations++
		}
	}
	if len(reviews) < 2 {
		return ConclusionPending
	}
	if reservations > 0 {
		return ConclusionReservation
	}
	if approvals >= 2 {
		return ConclusionApproved
	}
	return ConclusionPending
}

func summarizeWithRoster(reviews []PeerReview, roster *ReviewRoster) ReviewConclusion {
	if roster == nil {
		return SummarizeReviews(reviews)
	}
	approvals, reservations := 0, 0
	allowed := map[string]bool{}
	members := roster.Members
	if len(members) == 0 {
		members = roster.Experts
	}
	for _, member := range members {
		if !member.Recused {
			allowed[member.Name] = true
		}
	}
	for _, review := range reviews {
		if !allowed[review.Reviewer] {
			continue
		}
		switch review.Decision {
		case DecisionReturn:
			return ConclusionReturned
		case DecisionApprove:
			approvals++
		case DecisionReservation:
			reservations++
		}
	}
	if reservations > 0 {
		return ConclusionReservation
	}
	if approvals >= roster.Quorum {
		return ConclusionApproved
	}
	return ConclusionPending
}

func ReviewConclusionFor(c *ConservationCase) ReviewConclusion {
	return summarizeWithRoster(c.Reviews, c.ReviewRoster)
}

func (c *ConservationCase) RecordReview(expected int64, review PeerReview, now time.Time) (ReviewConclusion, error) {
	if err := CheckRevision(c.Revision, expected); err != nil {
		return "", err
	}
	if err := requireStatus(c.Status, StatusPeerReview); err != nil {
		return "", err
	}
	review.CaseID = c.ID
	review.ID = strings.TrimSpace(review.ID)
	review.Reviewer = strings.TrimSpace(review.Reviewer)
	if c.ReviewRoster != nil {
		if review.ProposalVersion != c.ReviewRoster.ProposalVersion {
			return "", &ValidationError{Issues: []FieldIssue{{Field: "proposal_version", Message: "意见必须绑定当前方案版本"}}}
		}
		members := c.ReviewRoster.Members
		if len(members) == 0 {
			members = c.ReviewRoster.Experts
		}
		for _, member := range members {
			if member.Name == review.Reviewer && member.Recused {
				return "", &ValidationError{Issues: []FieldIssue{{Field: "reviewer", Message: "回避专家不能提交意见"}}}
			}
		}
		found := false
		for _, member := range members {
			if member.Name == review.Reviewer {
				found = true
				break
			}
		}
		if !found {
			return "", &ValidationError{Issues: []FieldIssue{{Field: "reviewer", Message: "专家不在当前名册内"}}}
		}
	}
	if err := review.Validate(c); err != nil {
		return "", err
	}
	review.RecordedAt = now.UTC()
	c.Reviews = append(c.Reviews, review)
	conclusion := summarizeWithRoster(c.Reviews, c.ReviewRoster)
	if conclusion == ConclusionReturned || conclusion == ConclusionReservation {
		if c.Proposal != nil {
			c.ProposalHistory = append(c.ProposalHistory, ProposalVersionRecord{Proposal: *c.Proposal, ReviewComments: append([]PeerReview(nil), c.Reviews...), RecordedAt: now.UTC()})
		}
		c.Proposal = nil
		c.ReviewRoster = nil
		c.advance(StatusProposalDrafting, now)
		return conclusion, nil
	}
	if conclusion == ConclusionApproved {
		c.advance(StatusPendingTrial, now)
	} else {
		c.advance(StatusPeerReview, now)
	}
	return conclusion, nil
}
