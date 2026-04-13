package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCORS_ReturnsHandler(t *testing.T) {
	c := NewCORS([]string{"https://example.com"})
	if c == nil {
		t.Fatal("NewCORS() returned nil")
	}

	// The returned cors.Cors should be usable as middleware.
	handler := c.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodOptions, "/test", nil)
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", origin, "https://example.com")
	}
}

func TestNewCORS_DisallowedOrigin(t *testing.T) {
	c := NewCORS([]string{"https://allowed.com"})
	handler := c.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodOptions, "/test", nil)
	r.Header.Set("Origin", "https://evil.com")
	r.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin == "https://evil.com" {
		t.Error("disallowed origin should not be reflected")
	}
}

func TestNewCORS_MultipleOrigins(t *testing.T) {
	c := NewCORS([]string{"https://a.com", "https://b.com"})
	handler := c.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, origin := range []string{"https://a.com", "https://b.com"} {
		r := httptest.NewRequest(http.MethodOptions, "/", nil)
		r.Header.Set("Origin", origin)
		r.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		got := w.Header().Get("Access-Control-Allow-Origin")
		if got != origin {
			t.Errorf("origin %q: Access-Control-Allow-Origin = %q", origin, got)
		}
	}
}
