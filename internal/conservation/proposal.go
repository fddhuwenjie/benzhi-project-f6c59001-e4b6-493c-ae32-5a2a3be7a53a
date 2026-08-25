package conservation

import (
	"fmt"
	"strings"
	"time"
)

type Material struct {
	Name          string `json:"name"`
	Specification string `json:"specification"`
	Purpose       string `json:"purpose"`
}

type ProcedureStep struct {
	Order       int    `json:"order"`
	Instruction string `json:"instruction"`
	Checkpoint  string `json:"checkpoint"`
}

type TreatmentProposal struct {
	CaseID                  string          `json:"case_id"`
	ProposalVersion         string          `json:"proposal_version"`
	Materials               []Material      `json:"materials"`
	ProcedureSteps          []ProcedureStep `json:"procedure_steps"`
	EnvironmentRequirements []string        `json:"environment_requirements"`
	ReversibilityNotes      string          `json:"reversibility_notes"`
	RiskControls            []string        `json:"risk_controls"`
	AcceptanceCriteria      []string        `json:"acceptance_criteria"`
	SubmittedAt             time.Time       `json:"submitted_at"`
}

type ProposalVersionRecord struct {
	Proposal       TreatmentProposal `json:"proposal"`
	ReviewComments []PeerReview      `json:"review_comments,omitempty"`
	ResponseNotes  map[string]string `json:"response_notes,omitempty"`
	Diff           []ProposalDiff    `json:"diff,omitempty"`
	RecordedAt     time.Time         `json:"recorded_at"`
}

type ProposalDiff struct {
	Field  string `json:"field"`
	Change string `json:"change"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

func compareProposals(before, after *TreatmentProposal) []ProposalDiff {
	if before == nil || after == nil {
		return nil
	}
	result := make([]ProposalDiff, 0)
	if strings.Join(before.EnvironmentRequirements, "\x00") != strings.Join(after.EnvironmentRequirements, "\x00") {
		result = append(result, ProposalDiff{Field: "environment_requirements", Change: sliceChange(len(before.EnvironmentRequirements), len(after.EnvironmentRequirements)), Before: before.EnvironmentRequirements, After: after.EnvironmentRequirements})
	}
	if before.ReversibilityNotes != after.ReversibilityNotes {
		result = append(result, ProposalDiff{Field: "reversibility_notes", Change: "modified", Before: before.ReversibilityNotes, After: after.ReversibilityNotes})
	}
	if strings.Join(before.RiskControls, "\x00") != strings.Join(after.RiskControls, "\x00") {
		result = append(result, ProposalDiff{Field: "risk_controls", Change: sliceChange(len(before.RiskControls), len(after.RiskControls)), Before: before.RiskControls, After: after.RiskControls})
	}
	if strings.Join(before.AcceptanceCriteria, "\x00") != strings.Join(after.AcceptanceCriteria, "\x00") {
		result = append(result, ProposalDiff{Field: "acceptance_criteria", Change: sliceChange(len(before.AcceptanceCriteria), len(after.AcceptanceCriteria)), Before: before.AcceptanceCriteria, After: after.AcceptanceCriteria})
	}
	if len(before.Materials) != len(after.Materials) || fmt.Sprint(before.Materials) != fmt.Sprint(after.Materials) {
		result = append(result, ProposalDiff{Field: "materials", Change: sliceChange(len(before.Materials), len(after.Materials)), Before: before.Materials, After: after.Materials})
	}
	if len(before.ProcedureSteps) != len(after.ProcedureSteps) || fmt.Sprint(before.ProcedureSteps) != fmt.Sprint(after.ProcedureSteps) {
		result = append(result, ProposalDiff{Field: "procedure_steps", Change: sliceChange(len(before.ProcedureSteps), len(after.ProcedureSteps)), Before: before.ProcedureSteps, After: after.ProcedureSteps})
	}
	return result
}

func sliceChange(before, after int) string {
	if before == 0 && after > 0 {
		return "added"
	}
	if before > 0 && after == 0 {
		return "deleted"
	}
	return "modified"
}

func (p TreatmentProposal) Validate(caseID string) error {
	v := &ValidationError{}
	if p.CaseID != "" && p.CaseID != caseID {
		v.Add("case_id", "与当前事项不一致")
	}
	required(v, "proposal_version", p.ProposalVersion)
	if len(p.Materials) == 0 {
		v.Add("materials", "至少登记一种材料")
	}
	for i, material := range p.Materials {
		required(v, "materials["+itoa(i)+"].name", material.Name)
		required(v, "materials["+itoa(i)+"].specification", material.Specification)
		required(v, "materials["+itoa(i)+"].purpose", material.Purpose)
	}
	if len(p.ProcedureSteps) == 0 {
		v.Add("procedure_steps", "至少登记一个工序")
	}
	for i, step := range p.ProcedureSteps {
		if step.Order != i+1 {
			v.Add("procedure_steps["+itoa(i)+"].order", "工序序号必须从 1 连续递增")
		}
		required(v, "procedure_steps["+itoa(i)+"].instruction", step.Instruction)
		required(v, "procedure_steps["+itoa(i)+"].checkpoint", step.Checkpoint)
	}
	if len(cleanList(p.EnvironmentRequirements)) == 0 {
		v.Add("environment_requirements", "必须说明环境要求")
	}
	required(v, "reversibility_notes", p.ReversibilityNotes)
	if len(cleanList(p.RiskControls)) == 0 {
		v.Add("risk_controls", "必须说明风险缓解措施")
	}
	if len(cleanList(p.AcceptanceCriteria)) == 0 {
		v.Add("acceptance_criteria", "至少定义一项小样验收条件")
	}
	return v.OrNil()
}

func (c *ConservationCase) SubmitProposal(expected int64, proposal TreatmentProposal, now time.Time) error {
	if err := CheckRevision(c.Revision, expected); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusProposalDrafting); err != nil {
		return err
	}
	proposal.CaseID = c.ID
	proposal.ProposalVersion = strings.TrimSpace(proposal.ProposalVersion)
	for _, history := range c.ProposalHistory {
		if history.Proposal.ProposalVersion == proposal.ProposalVersion {
			v := &ValidationError{}
			v.Add("proposal_version", "方案版本标识必须区别于全部历史版本")
			return v
		}
	}
	if c.Proposal != nil && c.Proposal.ProposalVersion == proposal.ProposalVersion {
		v := &ValidationError{}
		v.Add("proposal_version", "方案版本标识不能重复")
		return v
	}
	proposal.EnvironmentRequirements = cleanList(proposal.EnvironmentRequirements)
	proposal.RiskControls = cleanList(proposal.RiskControls)
	proposal.AcceptanceCriteria = cleanList(proposal.AcceptanceCriteria)
	if err := proposal.Validate(c.ID); err != nil {
		return err
	}
	proposal.SubmittedAt = now.UTC()
	previous := c.Proposal
	if previous != nil {
		c.ProposalHistory = append(c.ProposalHistory, ProposalVersionRecord{Proposal: *previous, ReviewComments: append([]PeerReview(nil), c.Reviews...), RecordedAt: now.UTC(), Diff: compareProposals(previous, &proposal)})
	} else if len(c.ProposalHistory) > 0 {
		last := &c.ProposalHistory[len(c.ProposalHistory)-1]
		last.Diff = compareProposals(&last.Proposal, &proposal)
	}
	c.Proposal = &proposal
	c.Reviews = nil
	c.ReviewRoster = nil
	c.Trial = nil
	c.advance(StatusPeerReview, now)
	return nil
}
