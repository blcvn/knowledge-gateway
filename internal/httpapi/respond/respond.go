package respond

import (
	"encoding/json"
	"net/http"
)

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type ListEnvelope[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

func OK(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusOK, payload)
}

func Created(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusCreated, payload)
}

func Accepted(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusAccepted, payload)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, status int, code, message string, details any) {
	writeJSON(w, status, ErrorEnvelope{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
