package web

import (
	"net/http"
	"strings"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

func (s *Server) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	writeData(w, http.StatusOK, map[string]string{"status": "ok", "service": "古籍修复方案审议工作台"})
}

func (s *Server) HandleListCases(w http.ResponseWriter, r *http.Request) {
	status := conservation.Status(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_status", "状态筛选值无效", nil)
		return
	}
	items, err := s.service.ListCases(status, r.URL.Query().Get("q"))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, items)
}

func (s *Server) HandleCreateCase(w http.ResponseWriter, r *http.Request) {
	var command workflow.CreateCaseCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	result, err := s.service.CreateCase(command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	w.Header().Set("Location", "/api/cases/"+result.Case.ID)
	writeData(w, http.StatusCreated, result)
}

func (s *Server) HandleGetCase(w http.ResponseWriter, r *http.Request) {
	view, err := s.service.GetCase(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeData(w, http.StatusOK, view)
}
