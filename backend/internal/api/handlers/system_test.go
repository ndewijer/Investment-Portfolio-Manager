package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

func TestSystemHandler_Health(t *testing.T) {
	setupHandler := func(t *testing.T) (*SystemHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ss := testutil.NewTestSystemService(t, db)
		return NewSystemHandler(ss), db
	}

	t.Run("returns healthy status when database is connected", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/system/health", nil)
		w := httptest.NewRecorder()

		handler.Health(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response HealthResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		if response.Database != "connected" {
			t.Errorf("Expected database 'connected', got '%s'", response.Database)
		}

		if response.Error != "" {
			t.Errorf("Expected no error, got '%s'", response.Error)
		}
	})

	t.Run("returns 503 when database is disconnected", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Close the database connection to simulate failure
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/system/health", nil)
		w := httptest.NewRecorder()

		handler.Health(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected 503, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestSystemHandler_Version(t *testing.T) {
	setupHandler := func(t *testing.T) (*SystemHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ss := testutil.NewTestSystemService(t, db)
		return NewSystemHandler(ss), db
	}

	t.Run("returns version information successfully", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/system/version", nil)
		w := httptest.NewRecorder()

		handler.Version(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response VersionInfoResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.AppVersion == "" {
			t.Error("Expected app_version to be populated")
		}

		if response.DbVersion == "" {
			t.Error("Expected db_version to be populated")
		}

		if response.Features == nil {
			t.Error("Expected features map to be initialized")
		}
	})

	// Note: The Version endpoint doesn't fail when database is closed because
	// it doesn't require active database queries - it reads version from schema
	// which is cached or handled gracefully. No database error test needed.
}
