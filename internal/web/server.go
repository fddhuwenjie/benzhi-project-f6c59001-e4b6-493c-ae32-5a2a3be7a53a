package web

import (
	"log/slog"
	"net/http"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

type Server struct {
	service *workflow.Service
	logger  *slog.Logger
	mux     *http.ServeMux
}

func NewServer(service *workflow.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{service: service, logger: logger, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.HandleWorkspace)
	s.mux.HandleFunc("GET /assets/app.css", s.HandleCSS)
	s.mux.HandleFunc("GET /assets/app.js", s.HandleJS)
	s.mux.HandleFunc("GET /api/health", s.HandleHealth)
	s.mux.HandleFunc("GET /api/cases", s.HandleListCases)
	s.mux.HandleFunc("POST /api/cases", s.HandleCreateCase)
	s.mux.HandleFunc("GET /api/cases/{id}", s.HandleGetCase)
	s.mux.HandleFunc("POST /api/cases/{id}/submit", s.HandleSubmitAssessment)
	s.mux.HandleFunc("POST /api/cases/{id}/draft", s.HandleReviseDraft)
	s.mux.HandleFunc("POST /api/cases/{id}/assessment", s.HandleConfirmAssessment)
	s.mux.HandleFunc("POST /api/cases/{id}/proposal", s.HandleSubmitProposal)
	s.mux.HandleFunc("POST /api/cases/{id}/reviews/roster", s.HandleConfigureRoster)
	s.mux.HandleFunc("POST /api/cases/{id}/reviews", s.HandleRecordReview)
	s.mux.HandleFunc("POST /api/cases/{id}/trial", s.HandleVerifyTrial)
	s.mux.HandleFunc("POST /api/cases/{id}/archive", s.HandleArchive)
	s.mux.HandleFunc("GET /api/cases/{id}/precheck", s.HandlePrecheck)
	s.mux.HandleFunc("GET /api/cases/{id}/audit", s.HandleAuditPackage)
}

func (s *Server) Handler() http.Handler {
	return s.recoverPanic(s.requestLog(s.securityHeaders(s.mux)))
}

func HTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
