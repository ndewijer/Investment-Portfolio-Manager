// Package response provides utilities for sending consistent HTTP responses.
// It includes helpers for JSON responses and standardized error responses.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
)

// ErrorResponse represents a structured error response returned by the API.
// The Details field is optional and can contain additional context about the error.
type ErrorResponse struct {
	Error     string `json:"error"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// RespondJSON sends a JSON response with the given status code.
// Sets the Content-Type header to application/json and writes the status code.
// If data is nil, only the status code is sent (useful for 204 No Content).
// Logs encoding errors but does not fail the response.
func RespondJSON(w http.ResponseWriter, status int, data any) {
	if data == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// RespondError sends a structured error response with the given status code.
// Use for 4xx client errors where details are safe to expose (validation, not-found, etc.).
//
// Example:
//
//	response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
//	response.RespondError(w, http.StatusNotFound, "resource not found", "")
func RespondError(w http.ResponseWriter, status int, message string, details any) {
	resp := ErrorResponse{
		Error:   message,
		Details: details,
	}
	RespondJSON(w, status, resp)
}

// RespondInternalError sends a sanitized 500 error response.
// The full error chain is NOT exposed to the client — it's already in the logs via
// ErrorContext. Only the safe user-facing message and request ID are returned,
// so the user can reference the request ID when reporting issues.
//
// Example:
//
//	slog.ErrorContext(r.Context(), "failed to get fund", "error", err)
//	response.RespondInternalError(w, r, "failed to retrieve fund")
func RespondInternalError(w http.ResponseWriter, r *http.Request, message string) {
	resp := ErrorResponse{
		Error:     message,
		RequestID: logging.RequestIDFromContext(r.Context()),
	}
	RespondJSON(w, http.StatusInternalServerError, resp)
}
