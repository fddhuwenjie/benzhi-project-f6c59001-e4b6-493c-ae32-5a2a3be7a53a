package web

import (
	"net/http"
	"strings"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func (s *Server) HandleReviseDraft(w http.ResponseWriter, r *http.Request) {
	var command workflow.DraftRevisionCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.ReviseDraft(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) HandleConfigureRoster(w http.ResponseWriter, r *http.Request) {
	var command workflow.RosterCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.ConfigureReviewRoster(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) HandlePrecheck(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.BuildPrecheck(caseID(r))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func caseID(r *http.Request) string {
	return strings.TrimSpace(r.PathValue("id"))
}

func (s *Server) HandleSubmitAssessment(w http.ResponseWriter, r *http.Request) {
	var command workflow.CommandMeta
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.SubmitForAssessment(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) HandleConfirmAssessment(w http.ResponseWriter, r *http.Request) {
	var command workflow.AssessmentCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.ConfirmAssessment(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) HandleSubmitProposal(w http.ResponseWriter, r *http.Request) {
	var command workflow.ProposalCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.SubmitProposal(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) HandleRecordReview(w http.ResponseWriter, r *http.Request) {
	var command workflow.ReviewCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.RecordReview(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) HandleVerifyTrial(w http.ResponseWriter, r *http.Request) {
	var command workflow.TrialCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.VerifyTrial(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) HandleArchive(w http.ResponseWriter, r *http.Request) {
	var command workflow.ArchiveCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.ArchiveWithPrecheck(caseID(r), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
