package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/middleware"
)

func TestValidatePortfolioIDMiddleware(t *testing.T) {
	t.Run("passes through valid UUID", func(t *testing.T) {
		handlerCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		mw := middleware.ValidateUUIDMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("uuid", "550e8400-e29b-41d4-a716-446655440000")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("Expected next handler to be called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		handlerCalled := false
		next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
		})

		mw := middleware.ValidateUUIDMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("uuid", "invalid-id")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Expected next handler NOT to be called")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 for empty UUID", func(t *testing.T) {
		handlerCalled := false
		next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
		})

		mw := middleware.ValidateUUIDMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("uuid", "")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Expected next handler NOT to be called")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}
