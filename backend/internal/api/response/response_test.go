package response

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
)

func TestRespondJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"name": "test"}

	RespondJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var got map[string]string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if got["name"] != "test" {
		t.Errorf("body name = %q, want %q", got["name"], "test")
	}
}

func TestRespondJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusNoContent, nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
}

func TestRespondJSON_StatusCreated(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]int{"id": 42}

	RespondJSON(w, http.StatusCreated, data)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	RespondError(w, http.StatusBadRequest, "validation failed", "field X is required")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var got ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.Error != "validation failed" {
		t.Errorf("Error = %q, want %q", got.Error, "validation failed")
	}
	if got.Details != "field X is required" {
		t.Errorf("Details = %v, want %q", got.Details, "field X is required")
	}
}

func TestRespondError_EmptyDetails(t *testing.T) {
	w := httptest.NewRecorder()

	RespondError(w, http.StatusNotFound, "not found", "")

	var got ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.Error != "not found" {
		t.Errorf("Error = %q", got.Error)
	}
}

func TestRespondError_NilDetails(t *testing.T) {
	w := httptest.NewRecorder()

	RespondError(w, http.StatusConflict, "conflict", nil)

	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if _, exists := got["details"]; exists {
		t.Error("expected details to be omitted when nil")
	}
}

func TestRespondInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := logging.WithRequestInfo(context.Background(), "req-123", "1.2.3.4", "TestAgent")
	r := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	RespondInternalError(w, r, "something went wrong")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var got ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.Error != "something went wrong" {
		t.Errorf("Error = %q", got.Error)
	}
	if got.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", got.RequestID, "req-123")
	}
}

func TestRespondInternalError_NoRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	RespondInternalError(w, r, "error")

	var got ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.RequestID != "" {
		t.Errorf("RequestID = %q, want empty", got.RequestID)
	}
}
