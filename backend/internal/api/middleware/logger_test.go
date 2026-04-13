package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger_RecordsStatusCode(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("teapot")) //nolint:errcheck // test handler
	})

	handler := Logger(inner)
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTeapot)
	}
}

func TestLogger_DefaultStatusOK(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// No explicit WriteHeader — should default to 200.
		_, _ = w.Write([]byte("ok")) //nolint:errcheck // test handler
	})

	handler := Logger(inner)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestClientIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")

	ip := clientIP(r)
	if ip != "10.0.0.1" {
		t.Errorf("clientIP() = %q, want %q", ip, "10.0.0.1")
	}
}

func TestClientIP_XForwardedFor_Single(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "192.168.1.1")

	ip := clientIP(r)
	if ip != "192.168.1.1" {
		t.Errorf("clientIP() = %q, want %q", ip, "192.168.1.1")
	}
}

func TestClientIP_XForwardedFor_WithSpaces(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "  10.0.0.1 , 10.0.0.2")

	ip := clientIP(r)
	if ip != "10.0.0.1" {
		t.Errorf("clientIP() = %q, want %q (should be trimmed)", ip, "10.0.0.1")
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Real-IP", "172.16.0.1")

	ip := clientIP(r)
	if ip != "172.16.0.1" {
		t.Errorf("clientIP() = %q, want %q", ip, "172.16.0.1")
	}
}

func TestClientIP_FallbackRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.0.5:4321"

	ip := clientIP(r)
	if ip != "192.168.0.5" {
		t.Errorf("clientIP() = %q, want %q", ip, "192.168.0.5")
	}
}

func TestClientIP_RemoteAddrNoPort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.0.5" // no port — SplitHostPort will fail

	ip := clientIP(r)
	if ip != "192.168.0.5" {
		t.Errorf("clientIP() = %q, want %q", ip, "192.168.0.5")
	}
}

func TestClientIP_XForwardedForPrecedence(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1")
	r.Header.Set("X-Real-IP", "172.16.0.1")
	r.RemoteAddr = "192.168.0.5:4321"

	ip := clientIP(r)
	if ip != "10.0.0.1" {
		t.Errorf("clientIP() = %q, want %q (X-Forwarded-For should take precedence)", ip, "10.0.0.1")
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusNotFound)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("underlying recorder code = %d, want %d", w.Code, http.StatusNotFound)
	}
}
