package workflow

import (
	"strings"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/storage"
)

type CaseResult struct {
	Case       *conservation.ConservationCase `json:"case"`
	Conclusion conservation.ReviewConclusion  `json:"conclusion,omitempty"`
	Replayed   bool                           `json:"replayed,omitempty"`
}

func (s *Service) CreateCase(command CreateCaseCommand) (*CaseResult, error) {
	if err := command.Actor.Validate(RoleConservator); err != nil {
		return nil, err
	}
	var cached CaseResult
	if replayed, err := s.replay(command.RequestID, "create_case", "", &cached); err != nil {
		return nil, err
	} else if replayed {
		cached.Replayed = true
		return &cached, nil
	}
	id := strings.TrimSpace(command.ID)
	if id == "" {
		id = newID("case")
	}
	now := s.clock()
	item, err := conservation.CreateCase(conservation.NewCase{ID: id, ShelfMark: command.ShelfMark, Title: command.Title,
		VersionIdentifier: command.VersionIdentifier, SupportMaterial: command.SupportMaterial,
		CarrierCharacteristics: command.CarrierCharacteristics, DamageLocations: command.DamageLocations,
		InitialEvidence: command.InitialEvidence, ResponsibleConservator: command.ResponsibleConservator}, now)
	if err != nil {
		return nil, err
	}
	meta := CommandMeta{RequestID: command.RequestID, Actor: command.Actor}
	event := makeEvent(item, meta, "case.created", 0, "", map[string]any{"shelf_mark": item.ShelfMark}, now)
	if err := s.repo.Create(item, event); err != nil {
		return nil, err
	}
	result := &CaseResult{Case: item}
	if err := s.remember(command.RequestID, "create_case", item.ID, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) SubmitForAssessment(caseID string, meta CommandMeta) (*CaseResult, error) {
	if err := meta.Actor.Validate(RoleConservator); err != nil {
		return nil, err
	}
	return s.change(caseID, meta, "submit_assessment", "case.submitted_for_assessment", func(item *conservation.ConservationCase, nowTime conservationTime) error {
		return item.SubmitForAssessment(meta.ExpectedRevision, nowTime.Time)
	})
}

func (s *Service) ConfirmAssessment(caseID string, command AssessmentCommand) (*CaseResult, error) {
	if err := command.Actor.Validate(RoleManager); err != nil {
		return nil, err
	}
	if len(command.Assessment.PartitionAssessments) == 0 && len(command.Assessment.LocationAssessments) == 0 {
		return nil, &conservation.ValidationError{Issues: []conservation.FieldIssue{{Field: "assessment.partition_assessments", Message: "必须逐项登记全部建档损伤部位"}}}
	}
	return s.change(caseID, command.CommandMeta, "confirm_assessment", "assessment.confirmed", func(item *conservation.ConservationCase, nowTime conservationTime) error {
		return item.ConfirmAssessment(command.ExpectedRevision, command.Assessment, nowTime.Time)
	})
}

func (s *Service) SubmitProposal(caseID string, command ProposalCommand) (*CaseResult, error) {
	if err := command.Actor.Validate(RoleConservator); err != nil {
		return nil, err
	}
	return s.change(caseID, command.CommandMeta, "submit_proposal", "proposal.submitted", func(item *conservation.ConservationCase, nowTime conservationTime) error {
		if len(item.ProposalHistory) > 0 {
			history := &item.ProposalHistory[len(item.ProposalHistory)-1]
			for _, review := range history.ReviewComments {
				if (review.Decision == conservation.DecisionReturn || review.Decision == conservation.DecisionReservation) && strings.TrimSpace(command.ResponseNotes[review.ID]) == "" {
					return &conservation.ValidationError{Issues: []conservation.FieldIssue{{Field: "response_notes." + review.ID, Message: "必须逐条填写对审议意见的处理说明"}}}
				}
			}
			history.ResponseNotes = command.ResponseNotes
		}
		return item.SubmitProposal(command.ExpectedRevision, command.Proposal, nowTime.Time)
	})
}

func (s *Service) VerifyTrial(caseID string, command TrialCommand) (*CaseResult, error) {
	if err := command.Actor.Validate(RoleConservator); err != nil {
		return nil, err
	}
	if len(command.Trial.Deviations) > 0 && len(command.Trial.DeviationRecords) < len(command.Trial.Deviations) {
		return nil, &conservation.ValidationError{Issues: []conservation.FieldIssue{{Field: "trial.deviation_records", Message: "每项偏差必须登记影响等级、处置措施和关闭状态"}}}
	}
	return s.change(caseID, command.CommandMeta, "verify_trial", "trial.verified", func(item *conservation.ConservationCase, nowTime conservationTime) error {
		return item.VerifyTrial(command.ExpectedRevision, command.Trial, nowTime.Time)
	})
}

func (s *Service) Archive(caseID string, meta CommandMeta) (*CaseResult, error) {
	precheck, err := s.BuildPrecheck(caseID)
	if err != nil {
		return nil, err
	}
	hints := make([]string, 0)
	for _, item := range precheck.Items {
		if !item.Blocking && !item.Passed {
			hints = append(hints, item.Code)
		}
	}
	return s.ArchiveWithPrecheck(caseID, ArchiveCommand{CommandMeta: meta, ConfirmedHints: hints})
}

func (s *Service) ArchiveWithPrecheck(caseID string, command ArchiveCommand) (*CaseResult, error) {
	meta := command.CommandMeta
	if err := meta.Actor.Validate(RoleManager); err != nil {
		return nil, err
	}
	precheck, err := s.BuildPrecheck(caseID)
	if err != nil {
		return nil, err
	}
	if blockers := precheck.Blocking(); len(blockers) > 0 {
		v := &conservation.ValidationError{}
		for _, blocker := range blockers {
			v.Add("precheck."+blocker.Code, blocker.Message)
		}
		return nil, v
	}
	confirmed := map[string]bool{}
	for _, code := range command.ConfirmedHints {
		confirmed[strings.TrimSpace(code)] = true
	}
	for _, item := range precheck.Items {
		if !item.Blocking && !item.Passed && !confirmed[item.Code] {
			return nil, &conservation.ValidationError{Issues: []conservation.FieldIssue{{Field: "confirmed_hints", Message: "请确认提示项：" + item.Code}}}
		}
	}
	precheck.ConfirmedHints = append([]string(nil), command.ConfirmedHints...)
	return s.change(caseID, meta, "archive", "case.archived", func(item *conservation.ConservationCase, nowTime conservationTime) error {
		item.ArchivePrecheck = precheck
		return item.Archive(meta.ExpectedRevision, nowTime.Time)
	})
}

func (s *Service) GetCase(id string) (*CaseView, error) { return s.buildCaseView(id) }

func (s *Service) ListCases(status conservation.Status, query string) ([]CaseListItem, error) {
	items, err := s.repo.List(storage.CaseFilter{Status: status, Query: query})
	if err != nil {
		return nil, err
	}
	result := make([]CaseListItem, 0, len(items))
	for _, item := range items {
		result = append(result, makeListItem(item))
	}
	return result, nil
}
