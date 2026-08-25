package conservation

func (c *ConservationCase) ValidateAggregate() error {
	v := &ValidationError{}
	required(v, "id", c.ID)
	if c.Revision < 1 {
		v.Add("revision", "必须大于等于 1")
	}
	if !c.Status.Valid() {
		v.Add("status", "不是已知流程状态")
	}
	required(v, "shelf_mark", c.ShelfMark)
	required(v, "title", c.Title)
	required(v, "responsible_conservator", c.ResponsibleConservator)
	if c.CreatedAt.IsZero() {
		v.Add("created_at", "不能为空")
	}
	if c.UpdatedAt.IsZero() || c.UpdatedAt.Before(c.CreatedAt) {
		v.Add("updated_at", "不能为空且不能早于创建时间")
	}
	if c.Status == StatusPendingAssessment {
		if err := c.ValidateForSubmission(); err != nil {
			copyIssues(v, err)
		}
	}
	if statusNeedsAssessment(c.Status) && c.Assessment == nil {
		v.Add("assessment", "当前阶段必须存在已确认的损伤评估")
	}
	if c.Assessment != nil {
		if c.Assessment.CaseID != c.ID {
			v.Add("assessment.case_id", "与事项标识不一致")
		}
		if c.Assessment.ConfirmedAt.IsZero() {
			v.Add("assessment.confirmed_at", "不能为空")
		}
	}
	if statusNeedsProposal(c.Status) && c.Proposal == nil {
		v.Add("proposal", "当前阶段必须存在已提交的修复方案")
	}
	if c.Proposal != nil {
		if c.Proposal.CaseID != c.ID {
			v.Add("proposal.case_id", "与事项标识不一致")
		}
		if c.Proposal.SubmittedAt.IsZero() {
			v.Add("proposal.submitted_at", "不能为空")
		}
	}
	for i, review := range c.Reviews {
		if review.CaseID != c.ID {
			v.Add("reviews["+itoa(i)+"].case_id", "与事项标识不一致")
		}
		if review.RecordedAt.IsZero() {
			v.Add("reviews["+itoa(i)+"].recorded_at", "不能为空")
		}
	}
	if statusNeedsApprovedReviews(c.Status) && summarizeWithRoster(c.Reviews, c.ReviewRoster) != ConclusionApproved {
		v.Add("reviews", "当前阶段必须具有通过的同行审议结论")
	}
	if statusNeedsPassedTrial(c.Status) && (c.Trial == nil || c.Trial.Verdict != TrialPassed) {
		v.Add("trial", "当前阶段必须具有通过的小样核验")
	}
	if c.Trial != nil {
		if c.Trial.CaseID != c.ID {
			v.Add("trial.case_id", "与事项标识不一致")
		}
		if c.Trial.VerifiedAt.IsZero() {
			v.Add("trial.verified_at", "不能为空")
		}
	}
	if c.ReviewRoster != nil {
		if err := c.ReviewRoster.Validate(c); err != nil {
			copyIssues(v, err)
		}
	}
	if c.Status == StatusArchived {
		if c.ArchivedAt == nil || c.ArchivedAt.IsZero() {
			v.Add("archived_at", "封存事项必须记录封存时间")
		}
	} else if c.ArchivedAt != nil {
		v.Add("archived_at", "未封存事项不能记录封存时间")
	}
	return v.OrNil()
}

func statusNeedsAssessment(status Status) bool {
	switch status {
	case StatusProposalDrafting, StatusPeerReview, StatusPendingTrial, StatusPendingApproval, StatusArchived:
		return true
	default:
		return false
	}
}

func statusNeedsProposal(status Status) bool {
	switch status {
	case StatusPeerReview, StatusPendingTrial, StatusPendingApproval, StatusArchived:
		return true
	default:
		return false
	}
}

func statusNeedsApprovedReviews(status Status) bool {
	return status == StatusPendingTrial || status == StatusPendingApproval || status == StatusArchived
}

func statusNeedsPassedTrial(status Status) bool {
	return status == StatusPendingApproval || status == StatusArchived
}

func copyIssues(target *ValidationError, err error) {
	if source, ok := err.(*ValidationError); ok {
		target.Issues = append(target.Issues, source.Issues...)
	}
}
