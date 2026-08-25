package conservation

import (
	"errors"
	"testing"
	"time"
)

func TestReviseDraftReplacesEvidenceAndRejectsStaleRevision(t *testing.T) {
	item := completeDraft(t)
	changes, err := item.ReviseDraft(1, DraftRevision{Title: "测试古籍", VersionIdentifier: " 刻本二版 ", CarrierCharacteristics: "线装墨书", DamageLocations: []string{"书口", "书脊"}, InitialEvidence: []EvidenceRef{{ID: " damage ", Filename: "new.jpg", MediaType: " image/jpeg "}}}, item.CreatedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if item.Revision != 2 || item.InitialEvidence[0].Filename != "new.jpg" || changes["initial_evidence"] == nil {
		t.Fatalf("unexpected revision result: %#v %#v", item, changes)
	}
	if _, err := item.ReviseDraft(1, DraftRevision{}, time.Now()); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestPartitionAssessmentReportsCoverageAndSeverityTogether(t *testing.T) {
	item := completeDraft(t)
	item.DamageLocations = []string{"书口", "书脊"}
	if err := item.SubmitForAssessment(1, time.Now()); err != nil {
		t.Fatal(err)
	}
	assessment := DamageAssessment{Severity: SeverityModerate, Locations: item.DamageLocations, Symptoms: []string{"破损"}, ProbableCauses: []string{"磨损"}, TreatmentGoals: []string{"稳定"}, NonInterventionLimits: []string{"保留题记"}, EvidenceRefs: testEvidence("a"), Assessor: "负责人", PartitionAssessments: []DamageLocationAssessment{{Location: "书脊", Severity: SeveritySevere, Symptoms: []string{"断裂"}, ImpactScope: "全脊", TreatmentGoals: []string{"加固"}, Boundaries: []string{"保留原线"}}}}
	err := item.ConfirmAssessment(2, assessment, time.Now())
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	fields := map[string]bool{}
	for _, issue := range validation.Issues {
		fields[issue.Field] = true
	}
	if !fields["severity"] || !fields["partition_assessments"] {
		t.Fatalf("expected located issues, got %#v", validation.Issues)
	}
	if item.Revision != 2 || item.Status != StatusPendingAssessment {
		t.Fatal("failed assessment changed aggregate")
	}
}

func TestReturnedProposalHistoryRejectsReusedVersion(t *testing.T) {
	item := completeDraft(t)
	now := item.CreatedAt.Add(time.Hour)
	if err := item.SubmitForAssessment(1, now); err != nil {
		t.Fatal(err)
	}
	a := DamageAssessment{Severity: SeverityMinor, Locations: []string{"书口"}, Symptoms: []string{"裂口"}, ProbableCauses: []string{"磨损"}, TreatmentGoals: []string{"加固"}, NonInterventionLimits: []string{"保留旧补"}, EvidenceRefs: testEvidence("a"), Assessor: "负责人"}
	if err := item.ConfirmAssessment(2, a, now); err != nil {
		t.Fatal(err)
	}
	p := TreatmentProposal{ProposalVersion: "P1", Materials: []Material{{Name: "纸", Specification: "薄", Purpose: "补"}}, ProcedureSteps: []ProcedureStep{{Order: 1, Instruction: "补", Checkpoint: "平"}}, EnvironmentRequirements: []string{"稳定"}, ReversibilityNotes: "可移除", RiskControls: []string{"小样"}, AcceptanceCriteria: []string{"平整"}}
	if err := item.SubmitProposal(3, p, now); err != nil {
		t.Fatal(err)
	}
	if _, err := item.RecordReview(4, PeerReview{ID: "return-1", ProposalVersion: "P1", Reviewer: "专家甲", Decision: DecisionReturn, ReturnReason: "材料依据不足"}, now); err != nil {
		t.Fatal(err)
	}
	if len(item.ProposalHistory) != 1 || item.ProposalHistory[0].Proposal.ProposalVersion != "P1" {
		t.Fatalf("proposal history missing: %#v", item.ProposalHistory)
	}
	if err := item.SubmitProposal(5, p, now); err == nil {
		t.Fatal("historical proposal version was reused")
	}
}

func TestRosterRecusalAndOpenDeviationAffectProgress(t *testing.T) {
	item := completeDraft(t)
	now := item.CreatedAt.Add(time.Hour)
	if err := item.SubmitForAssessment(1, now); err != nil {
		t.Fatal(err)
	}
	a := DamageAssessment{Severity: SeverityMinor, Locations: []string{"书口"}, Symptoms: []string{"裂口"}, ProbableCauses: []string{"磨损"}, TreatmentGoals: []string{"加固"}, NonInterventionLimits: []string{"保留旧补"}, EvidenceRefs: testEvidence("a"), Assessor: "负责人"}
	if err := item.ConfirmAssessment(2, a, now); err != nil {
		t.Fatal(err)
	}
	p := TreatmentProposal{ProposalVersion: "P1", Materials: []Material{{Name: "纸", Specification: "薄", Purpose: "补"}}, ProcedureSteps: []ProcedureStep{{Order: 1, Instruction: "补", Checkpoint: "平"}}, EnvironmentRequirements: []string{"稳定"}, ReversibilityNotes: "可移除", RiskControls: []string{"小样"}, AcceptanceCriteria: []string{"平整"}}
	if err := item.SubmitProposal(3, p, now); err != nil {
		t.Fatal(err)
	}
	item.ReviewRoster = &ReviewRoster{ProposalVersion: "P1", Quorum: 2, Members: []RosterMember{{Name: "专家甲"}, {Name: "专家乙"}, {Name: "专家丙", Recused: true, Reason: "参与编制"}}}
	if _, err := item.RecordReview(4, PeerReview{ID: "r0", ProposalVersion: "P1", Reviewer: "专家丙", Decision: DecisionApprove}, now); err == nil {
		t.Fatal("recused reviewer was accepted")
	}
	for i, name := range []string{"专家甲", "专家乙"} {
		if _, err := item.RecordReview(int64(4+i), PeerReview{ID: "r" + itoa(i+1), ProposalVersion: "P1", Reviewer: name, Decision: DecisionApprove}, now); err != nil {
			t.Fatal(err)
		}
	}
	progress := item.ReviewProgress()
	if progress.Submitted != 2 || progress.Recused != 1 || item.Status != StatusPendingTrial {
		t.Fatalf("unexpected progress: %#v status=%s", progress, item.Status)
	}
	trial := TrialRecord{ID: "t1", BatchCode: "T1", Method: "边角小样", Observations: []string{"平整"}, EvidenceRefs: testEvidence("t"), CriterionResults: []CriterionResult{{Criterion: "平整", Passed: true}}, DeviationRecords: []DeviationRecord{{ID: "d1", Impact: "中", Action: "复测", Closed: false}}}
	if err := item.VerifyTrial(6, trial, now); err != nil {
		t.Fatal(err)
	}
	if item.Trial.Verdict != TrialFailed || item.Status != StatusProposalDrafting || len(item.TrialBatches) != 1 {
		t.Fatalf("open deviation passed unexpectedly: %#v", item.Trial)
	}
}
