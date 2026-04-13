package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/middleware"
)

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestAPIKeyMiddleware(t *testing.T) {
	testAPIKey := "test-api-key-12345"

	t.Run("rejects request without API key", func(t *testing.T) {
		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		mw := middleware.APIKeyMiddleware(testAPIKey)(testHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Expected request not to complete.")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["details"] != "Missing API key" {
			t.Errorf("Expected 'Missing API key' error, got '%s'", response["details"])
		}
	})

	t.Run("rejects request with invalid API key", func(t *testing.T) {
		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		mw := middleware.APIKeyMiddleware(testAPIKey)(testHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req.Header.Set("X-API-Key", "invalid")

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Expected request not to complete.")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["details"] != "Invalid API key" {
			t.Errorf("Expected 'Invalid API key' error, got '%s'", response["details"])
		}
	})

	t.Run("rejects request without time token", func(t *testing.T) {
		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		mw := middleware.APIKeyMiddleware(testAPIKey)(testHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req.Header.Set("X-API-Key", testAPIKey)

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Expected request not to complete.")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["details"] != "Missing Time token" {
			t.Errorf("Expected 'Missing Time token' error, got '%s'", response["details"])
		}
	})

	t.Run("rejects request with invalid time token", func(t *testing.T) {
		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		mw := middleware.APIKeyMiddleware(testAPIKey)(testHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("X-Time-Token", "invalid")

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Expected request not to complete.")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["details"] != "Time token is invalid or expired" {
			t.Errorf("Expected 'Time token is invalid or expired' error, got '%s'", response["details"])
		}

	})

	t.Run("allows request with valid API key and time token", func(t *testing.T) {
		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		mw := middleware.APIKeyMiddleware(testAPIKey)(testHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req.Header.Set("X-API-Key", testAPIKey)
		timeToken := middleware.GenerateTimeToken(testAPIKey)
		req.Header.Set("X-Time-Token", timeToken)

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("Expected handler to complete.")
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

	})

	t.Run("fail on not loaded internal_api_key", func(t *testing.T) {
		handlerCalled := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		mw := middleware.APIKeyMiddleware("")(testHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Expected request not to complete.")
		}
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "internal server error" {
			t.Errorf("Expected 'internal server error', got '%s'", response["error"])
		}
		// Details should NOT be exposed for 500s (security: no internal leak)
		if response["details"] != "" {
			t.Errorf("Expected empty details for 500, got '%s'", response["details"])
		}
	})
}
