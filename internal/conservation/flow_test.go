package conservation

import (
	"errors"
	"testing"
	"time"
)

func testEvidence(id string) []EvidenceRef {
	return []EvidenceRef{{ID: id, Filename: id + ".jpg", MediaType: "image/jpeg"}}
}

func completeDraft(t *testing.T) *ConservationCase {
	t.Helper()
	item, err := CreateCase(NewCase{
		ID: "case-1", ShelfMark: "善本-1", Title: "测试古籍", VersionIdentifier: "刻本",
		SupportMaterial: "竹纸", CarrierCharacteristics: "线装墨书",
		DamageLocations: []string{"书口"}, InitialEvidence: testEvidence("damage"),
		ResponsibleConservator: "修复师甲",
	}, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestSubmissionReturnsLocatedIssues(t *testing.T) {
	item, err := CreateCase(NewCase{ID: "case-1", ShelfMark: "A", Title: "残卷", ResponsibleConservator: "甲"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	err = item.SubmitForAssessment(1, time.Now())
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	fields := map[string]bool{}
	for _, issue := range validation.Issues {
		fields[issue.Field] = true
	}
	for _, field := range []string{"version_identifier", "support_material", "carrier_characteristics", "damage_locations", "initial_evidence"} {
		if !fields[field] {
			t.Errorf("missing issue for %s", field)
		}
	}
	if item.Status != StatusDraft || item.Revision != 1 {
		t.Fatalf("failed transition changed aggregate: %s r%d", item.Status, item.Revision)
	}
}

func TestCompleteConservationFlow(t *testing.T) {
	item := completeDraft(t)
	now := item.CreatedAt.Add(time.Hour)
	if err := item.SubmitForAssessment(1, now); err != nil {
		t.Fatal(err)
	}
	assessment := DamageAssessment{
		Severity: SeverityModerate, Locations: []string{"书口"}, Symptoms: []string{"撕裂"},
		ProbableCauses: []string{"磨损"}, TreatmentGoals: []string{"稳定"},
		NonInterventionLimits: []string{"保留题记"}, EvidenceRefs: testEvidence("assessment"), Assessor: "负责人",
	}
	if err := item.ConfirmAssessment(2, assessment, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	proposal := TreatmentProposal{
		ProposalVersion: "P1", Materials: []Material{{Name: "楮皮纸", Specification: "薄型", Purpose: "补强"}},
		ProcedureSteps:          []ProcedureStep{{Order: 1, Instruction: "补强", Checkpoint: "平整"}},
		EnvironmentRequirements: []string{"温度稳定"}, ReversibilityNotes: "可受控移除",
		RiskControls: []string{"控制水分"}, AcceptanceCriteria: []string{"无翘曲"},
	}
	if err := item.SubmitProposal(3, proposal, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	for index, reviewer := range []string{"专家甲", "专家乙"} {
		conclusion, err := item.RecordReview(int64(4+index), PeerReview{
			ID: "review-" + itoa(index), ProposalVersion: "P1", Reviewer: reviewer, Decision: DecisionApprove,
		}, now.Add(time.Duration(3+index)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 && conclusion != ConclusionPending {
			t.Fatalf("one approval should remain pending, got %s", conclusion)
		}
	}
	trial := TrialRecord{
		ID: "trial-1", BatchCode: "T1", Method: "边角小样", Observations: []string{"平整"},
		EvidenceRefs: testEvidence("trial"), CriterionResults: []CriterionResult{{Criterion: "无翘曲", Passed: true}},
	}
	if err := item.VerifyTrial(6, trial, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := item.Archive(7, now.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if item.Status != StatusArchived || item.Revision != 8 || item.ArchivedAt == nil {
		t.Fatalf("unexpected archived aggregate: %s r%d", item.Status, item.Revision)
	}
	if err := item.ValidateAggregate(); err != nil {
		t.Fatalf("archived aggregate invalid: %v", err)
	}
	if err := item.Archive(8, now.Add(7*time.Minute)); err == nil {
		t.Fatal("archived case accepted a repeated transition")
	}
}

func TestReturnReviewImmediatelyReopensProposal(t *testing.T) {
	item := completeDraft(t)
	now := item.CreatedAt.Add(time.Hour)
	if err := item.SubmitForAssessment(1, now); err != nil {
		t.Fatal(err)
	}
	assessment := DamageAssessment{Severity: SeverityMinor, Locations: []string{"书口"}, Symptoms: []string{"裂口"}, ProbableCauses: []string{"磨损"}, TreatmentGoals: []string{"加固"}, NonInterventionLimits: []string{"保留旧补"}, EvidenceRefs: testEvidence("a"), Assessor: "负责人"}
	if err := item.ConfirmAssessment(2, assessment, now); err != nil {
		t.Fatal(err)
	}
	proposal := TreatmentProposal{ProposalVersion: "P1", Materials: []Material{{Name: "纸", Specification: "薄", Purpose: "补"}}, ProcedureSteps: []ProcedureStep{{Order: 1, Instruction: "补", Checkpoint: "平"}}, EnvironmentRequirements: []string{"稳定"}, ReversibilityNotes: "可移除", RiskControls: []string{"小样"}, AcceptanceCriteria: []string{"平整"}}
	if err := item.SubmitProposal(3, proposal, now); err != nil {
		t.Fatal(err)
	}
	conclusion, err := item.RecordReview(4, PeerReview{ID: "r1", ProposalVersion: "P1", Reviewer: "专家", Decision: DecisionReturn, ReturnReason: "材料依据不足"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if conclusion != ConclusionReturned || item.Status != StatusProposalDrafting || item.Proposal != nil {
		t.Fatalf("return did not reopen proposal: conclusion=%s status=%s", conclusion, item.Status)
	}
}
