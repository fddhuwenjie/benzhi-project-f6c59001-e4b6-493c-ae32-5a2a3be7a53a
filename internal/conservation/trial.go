package conservation

import (
	"encoding/json"
	"strings"
	"time"
)

type CriterionResult struct {
	Criterion string `json:"criterion"`
	Passed    bool   `json:"passed"`
	Note      string `json:"note,omitempty"`
}

type TrialVerdict string

const (
	TrialPassed TrialVerdict = "passed"
	TrialFailed TrialVerdict = "failed"
)

type TrialRecord struct {
	ID                string             `json:"id"`
	CaseID            string             `json:"case_id"`
	BatchCode         string             `json:"batch_code"`
	ProposalVersion   string             `json:"proposal_version"`
	Method            string             `json:"method"`
	Observations      []string           `json:"observations"`
	Deviations        []string           `json:"deviations,omitempty"`
	DeviationRecords  []DeviationRecord  `json:"deviation_records,omitempty"`
	PreviousBatchID   string             `json:"previous_batch_id,omitempty"`
	EvidenceRefs      []EvidenceRef      `json:"evidence_refs"`
	CriterionResults  []CriterionResult  `json:"criterion_results"`
	Verdict           TrialVerdict       `json:"verdict"`
	FailedCriteria    []string           `json:"failed_criteria,omitempty"`
	RetestComparisons []RetestComparison `json:"retest_comparisons,omitempty"`
	VerifiedAt        time.Time          `json:"verified_at"`
}

type DeviationRecord struct {
	ID     string `json:"id"`
	Impact string `json:"impact"`
	Action string `json:"action"`
	Closed bool   `json:"closed"`
}

type RetestComparison struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Resolved   bool   `json:"resolved"`
	Note       string `json:"note,omitempty"`
}

func (t *TrialRecord) UnmarshalJSON(data []byte) error {
	type alias TrialRecord
	var raw struct {
		*alias
		Deviations json.RawMessage `json:"deviations"`
	}
	raw.alias = (*alias)(t)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Deviations) == 0 || string(raw.Deviations) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw.Deviations, &t.Deviations); err == nil {
		return nil
	}
	return json.Unmarshal(raw.Deviations, &t.DeviationRecords)
}

func (t *TrialRecord) evaluate(c *ConservationCase) error {
	v := &ValidationError{}
	required(v, "id", t.ID)
	required(v, "batch_code", t.BatchCode)
	required(v, "method", t.Method)
	if len(cleanList(t.Observations)) == 0 {
		v.Add("observations", "至少记录一个观察值")
	}
	seen := map[string]bool{}
	for i, deviation := range t.DeviationRecords {
		if strings.TrimSpace(deviation.ID) == "" {
			v.Add("deviation_records["+itoa(i)+"].id", "偏差标识不能为空")
		}
		if seen[deviation.ID] {
			v.Add("deviation_records["+itoa(i)+"].id", "偏差标识不能重复")
		}
		seen[deviation.ID] = true
		if strings.TrimSpace(deviation.Impact) == "" {
			v.Add("deviation_records["+itoa(i)+"].impact", "必须说明影响等级")
		}
		if strings.TrimSpace(deviation.Action) == "" {
			v.Add("deviation_records["+itoa(i)+"].action", "必须填写处置措施")
		}
	}
	validateEvidence("evidence_refs", t.EvidenceRefs, true, v)
	if c.Proposal == nil {
		v.Add("proposal", "当前方案不存在")
	} else {
		if t.ProposalVersion != "" && t.ProposalVersion != c.Proposal.ProposalVersion {
			v.Add("proposal_version", "小样批次必须绑定当前方案版本")
		}
		if len(t.CriterionResults) != len(c.Proposal.AcceptanceCriteria) {
			v.Add("criterion_results", "必须逐项核对方案验收条件")
		} else {
			for i, criterion := range c.Proposal.AcceptanceCriteria {
				if strings.TrimSpace(t.CriterionResults[i].Criterion) != criterion {
					v.Add("criterion_results["+itoa(i)+"].criterion", "与方案验收条件不一致")
				}
				if !t.CriterionResults[i].Passed && strings.TrimSpace(t.CriterionResults[i].Note) == "" {
					v.Add("criterion_results["+itoa(i)+"].note", "未通过时必须说明原因")
				}
			}
		}
	}
	if err := v.OrNil(); err != nil {
		return err
	}
	t.Verdict = TrialPassed
	for _, result := range t.CriterionResults {
		if !result.Passed {
			t.Verdict = TrialFailed
		}
	}
	return nil
}

func (c *ConservationCase) VerifyTrial(expected int64, trial TrialRecord, now time.Time) error {
	if err := CheckRevision(c.Revision, expected); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusPendingTrial); err != nil {
		return err
	}
	trial.CaseID = c.ID
	if c.Proposal != nil && strings.TrimSpace(trial.ProposalVersion) == "" {
		trial.ProposalVersion = c.Proposal.ProposalVersion
	}
	trial.ID = strings.TrimSpace(trial.ID)
	trial.BatchCode = strings.TrimSpace(trial.BatchCode)
	trial.Method = strings.TrimSpace(trial.Method)
	trial.Observations = cleanList(trial.Observations)
	trial.Deviations = cleanList(trial.Deviations)
	for i := range trial.EvidenceRefs {
		trial.EvidenceRefs[i].ID = strings.TrimSpace(trial.EvidenceRefs[i].ID)
		trial.EvidenceRefs[i].Filename = strings.TrimSpace(trial.EvidenceRefs[i].Filename)
		trial.EvidenceRefs[i].MediaType = strings.TrimSpace(trial.EvidenceRefs[i].MediaType)
		trial.EvidenceRefs[i].Note = strings.TrimSpace(trial.EvidenceRefs[i].Note)
	}
	var previous *TrialRecord
	if trial.PreviousBatchID != "" {
		for _, existing := range c.TrialBatches {
			if existing.ID == strings.TrimSpace(trial.PreviousBatchID) && existing.Verdict == TrialFailed {
				copy := existing
				previous = &copy
			}
		}
		if previous == nil {
			v := &ValidationError{}
			v.Add("previous_batch_id", "必须引用当前事项中的失败批次")
			return v
		}
	}
	for i := range trial.DeviationRecords {
		trial.DeviationRecords[i].ID = strings.TrimSpace(trial.DeviationRecords[i].ID)
		trial.DeviationRecords[i].Impact = strings.TrimSpace(trial.DeviationRecords[i].Impact)
		trial.DeviationRecords[i].Action = strings.TrimSpace(trial.DeviationRecords[i].Action)
	}
	for _, existing := range c.TrialBatches {
		if existing.BatchCode == trial.BatchCode {
			v := &ValidationError{}
			v.Add("batch_code", "事项内批次编号不能重复")
			return v
		}
	}
	if err := trial.evaluate(c); err != nil {
		return err
	}
	trial.VerifiedAt = now.UTC()
	for _, deviation := range trial.DeviationRecords {
		if !deviation.Closed {
			trial.Verdict = TrialFailed
		}
	}
	for _, result := range trial.CriterionResults {
		if !result.Passed {
			trial.FailedCriteria = append(trial.FailedCriteria, result.Criterion)
		}
	}
	if previous != nil {
		provided := map[string]bool{}
		for _, comparison := range trial.RetestComparisons {
			if comparison.Resolved {
				provided[comparison.SourceType+"\x00"+comparison.SourceID] = true
			}
		}
		trial.RetestComparisons = nil
		for _, criterion := range previous.FailedCriteria {
			resolved := provided["criterion\x00"+criterion]
			for _, current := range trial.CriterionResults {
				if current.Criterion == criterion && current.Passed {
					resolved = true
				}
			}
			trial.RetestComparisons = append(trial.RetestComparisons, RetestComparison{SourceType: "criterion", SourceID: criterion, Resolved: resolved})
			if !resolved {
				trial.Verdict = TrialFailed
			}
		}
		for _, deviation := range previous.DeviationRecords {
			if !deviation.Closed {
				resolved := provided["deviation\x00"+deviation.ID]
				for _, current := range trial.DeviationRecords {
					if current.ID == deviation.ID && current.Closed {
						resolved = true
					}
				}
				trial.RetestComparisons = append(trial.RetestComparisons, RetestComparison{SourceType: "deviation", SourceID: deviation.ID, Resolved: resolved})
				if !resolved {
					trial.Verdict = TrialFailed
				}
			}
		}
	}
	c.Trial = &trial
	c.TrialBatches = append(c.TrialBatches, trial)
	if trial.Verdict == TrialPassed {
		c.advance(StatusPendingApproval, now)
	} else {
		c.advance(StatusProposalDrafting, now)
	}
	return nil
}

func (c *ConservationCase) Archive(expected int64, now time.Time) error {
	if err := CheckRevision(c.Revision, expected); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusPendingApproval); err != nil {
		return err
	}
	if c.Assessment == nil || c.Proposal == nil || ReviewConclusionFor(c) != ConclusionApproved || c.Trial == nil || c.Trial.Verdict != TrialPassed {
		return &TransitionError{From: c.Status, Message: "评估、同行审议或小样证据链不完整，不能批准封存"}
	}
	c.advance(StatusArchived, now)
	archivedAt := now.UTC()
	c.ArchivedAt = &archivedAt
	return nil
}
