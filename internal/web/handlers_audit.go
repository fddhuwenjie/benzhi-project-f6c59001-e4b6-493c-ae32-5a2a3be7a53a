package web

import (
	"fmt"
	"net/http"
)

func (s *Server) HandleAuditPackage(w http.ResponseWriter, r *http.Request) {
	pkg, err := s.service.BuildAuditPackage(caseID(r))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "audit-"+caseID(r)+".json"))
	writeData(w, http.StatusOK, pkg)
}
