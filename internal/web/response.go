package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/workflow"
)

type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Code    string                    `json:"code"`
	Message string                    `json:"message"`
	Issues  []conservation.FieldIssue `json:"issues,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, envelope{Data: data})
}

func writeError(w http.ResponseWriter, status int, code, message string, issues []conservation.FieldIssue) {
	writeJSON(w, status, envelope{Error: &apiError{Code: code, Message: message, Issues: issues}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "请求 JSON 无法解析："+err.Error(), nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "请求只能包含一个 JSON 对象", nil)
		return false
	}
	return true
}

func handleServiceError(w http.ResponseWriter, err error) {
	var validation *conservation.ValidationError
	var transition *conservation.TransitionError
	var auth *workflow.AuthorizationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusUnprocessableEntity, "validation_failed", validation.Error(), validation.Issues)
	case errors.As(err, &transition):
		writeError(w, http.StatusConflict, "illegal_transition", transition.Error(), nil)
	case errors.As(err, &auth):
		writeError(w, http.StatusForbidden, "forbidden", auth.Error(), nil)
	case errors.Is(err, conservation.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error(), nil)
	case workflow.IsConflict(err):
		writeError(w, http.StatusConflict, "revision_conflict", err.Error(), nil)
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "处理请求失败", nil)
	}
}
