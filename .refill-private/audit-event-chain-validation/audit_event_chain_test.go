package audit_event_chain_validation_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func TestAuditExportRejectsBrokenEventChain(t *testing.T) {
	root := t.TempDir()
	repo, err := storage.NewFileRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo)
	archiveCompleteCase(t, service)

	logs, err := filepath.Glob(filepath.Join(root, "events", "*.jsonl"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("event log lookup failed: err=%v count=%d", err, len(logs))
	}
	data, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte{'\n'})
	var final conservation.Event
	if err := json.Unmarshal(lines[len(lines)-1], &final); err != nil {
		t.Fatal(err)
	}
	final.BeforeRevision = 1
	lines[len(lines)-1], err = json.Marshal(final)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logs[0], append(bytes.Join(lines, []byte{'\n'}), '\n'), 0o640); err != nil {
		t.Fatal(err)
	}

	if pkg, err := service.BuildAuditPackage("audit-chain-case"); err == nil {
		t.Fatalf("audit export accepted a broken final revision link and issued digest %s", pkg.ContentSHA256)
	}
}

func archiveCompleteCase(t *testing.T, service *workflow.Service) {
	t.Helper()
	created, err := service.CreateCase(workflow.CreateCaseCommand{
		RequestID: "create", Actor: workflow.Actor{Name: "修复师甲", Role: workflow.RoleConservator},
		ID: "audit-chain-case", ShelfMark: "A-1", Title: "审计链测试古籍", VersionIdentifier: "刻本",
		SupportMaterial: "竹纸", CarrierCharacteristics: "线装", DamageLocations: []string{"书口"},
		InitialEvidence:        []conservation.EvidenceRef{{ID: "initial", Filename: "initial.jpg", MediaType: "image/jpeg", SHA256: "initial-hash"}},
		ResponsibleConservator: "修复师甲",
	})
	if err != nil {
		t.Fatal(err)
	}
	caseID := created.Case.ID
	_, err = service.SubmitForAssessment(caseID, workflow.CommandMeta{RequestID: "submit", ExpectedRevision: 1, Actor: workflow.Actor{Name: "修复师甲", Role: workflow.RoleConservator}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ConfirmAssessment(caseID, workflow.AssessmentCommand{
		CommandMeta: workflow.CommandMeta{RequestID: "assessment", ExpectedRevision: 2, Actor: workflow.Actor{Name: "负责人甲", Role: workflow.RoleManager}},
		Assessment: conservation.DamageAssessment{
			Severity: conservation.SeverityMinor, Locations: []string{"书口"}, Symptoms: []string{"裂口"}, ProbableCauses: []string{"磨损"},
			TreatmentGoals: []string{"加固"}, NonInterventionLimits: []string{"保留题记"}, Assessor: "负责人甲",
			EvidenceRefs:         []conservation.EvidenceRef{{ID: "assessment", Filename: "assessment.jpg", MediaType: "image/jpeg", SHA256: "assessment-hash"}},
			PartitionAssessments: []conservation.DamageLocationAssessment{{Location: "书口", Severity: conservation.SeverityMinor, Symptoms: []string{"裂口"}, ImpactScope: "局部", TreatmentGoals: []string{"加固"}, Boundaries: []string{"保留题记"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SubmitProposal(caseID, workflow.ProposalCommand{
		CommandMeta: workflow.CommandMeta{RequestID: "proposal", ExpectedRevision: 3, Actor: workflow.Actor{Name: "修复师甲", Role: workflow.RoleConservator}},
		Proposal: conservation.TreatmentProposal{
			ProposalVersion: "P1", Materials: []conservation.Material{{Name: "楮皮纸", Specification: "薄型", Purpose: "补强"}},
			ProcedureSteps: []conservation.ProcedureStep{{Order: 1, Instruction: "补强", Checkpoint: "平整"}}, EnvironmentRequirements: []string{"温度稳定"},
			ReversibilityNotes: "可受控移除", RiskControls: []string{"控制水分"}, AcceptanceCriteria: []string{"无翘曲"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for index, reviewer := range []string{"专家甲", "专家乙"} {
		_, err = service.RecordReview(caseID, workflow.ReviewCommand{
			CommandMeta: workflow.CommandMeta{RequestID: "review-" + reviewer, ExpectedRevision: int64(4 + index), Actor: workflow.Actor{Name: reviewer, Role: workflow.RoleReviewer}},
			Review:      conservation.PeerReview{ID: "review-id-" + reviewer, ProposalVersion: "P1", Reviewer: reviewer, Decision: conservation.DecisionApprove},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = service.VerifyTrial(caseID, workflow.TrialCommand{
		CommandMeta: workflow.CommandMeta{RequestID: "trial", ExpectedRevision: 6, Actor: workflow.Actor{Name: "修复师甲", Role: workflow.RoleConservator}},
		Trial: conservation.TrialRecord{ID: "trial-1", BatchCode: "T1", Method: "边角小样", Observations: []string{"平整"},
			EvidenceRefs:     []conservation.EvidenceRef{{ID: "trial", Filename: "trial.jpg", MediaType: "image/jpeg", SHA256: "trial-hash"}},
			CriterionResults: []conservation.CriterionResult{{Criterion: "无翘曲", Passed: true}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Archive(caseID, workflow.CommandMeta{RequestID: "archive", ExpectedRevision: 7, Actor: workflow.Actor{Name: "负责人甲", Role: workflow.RoleManager}})
	if err != nil {
		t.Fatal(err)
	}
}
