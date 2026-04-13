package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/response"
)

// TestRespondJSON tests the respondJSON helper function.
// This is an internal test (package handlers, not handlers_test) because
// respondJSON is unexported.
func TestRespondJSON(t *testing.T) {
	t.Run("sets content-type and status code correctly", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"message": "success"}

		response.RespondJSON(w, 200, data)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
		}
	})

	t.Run("handles nil data without error", func(t *testing.T) {
		w := httptest.NewRecorder()

		response.RespondJSON(w, 204, nil)

		if w.Code != 204 {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("handles un-encodable data gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Channels cannot be JSON encoded
		data := map[string]any{
			"channel": make(chan int),
		}

		// Should not panic, just log the error
		response.RespondJSON(w, 200, data)

		// Status should still be set even if encoding fails
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Content-Type should still be set
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type to be set")
		}
	})

	t.Run("encodes valid data successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{
			"name":  "test",
			"value": "data",
		}

		response.RespondJSON(w, 200, data)

		if w.Body.Len() == 0 {
			t.Error("Expected response body to contain JSON data")
		}

		body := w.Body.String()
		if body == "" {
			t.Error("Expected non-empty response body")
		}
	})
}
