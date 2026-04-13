package service_test

import (
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// =============================================================================
// CHECK HEALTH
// =============================================================================

func TestSystemService_CheckHealth(t *testing.T) {
	t.Run("healthy database returns nil", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := service.NewSystemService(db)

		err := svc.CheckHealth()
		if err != nil {
			t.Errorf("CheckHealth() returned unexpected error: %v", err)
		}
	})

	t.Run("closed database returns error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := service.NewSystemService(db)

		// Close the DB to simulate unhealthy state
		db.Close()

		err := svc.CheckHealth()
		if err == nil {
			t.Error("CheckHealth() expected error for closed DB, got nil")
		}
	})
}

// =============================================================================
// CHECK VERSION
// =============================================================================

func TestSystemService_CheckVersion(t *testing.T) {
	t.Run("returns version info with features", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := service.NewSystemService(db)

		info, err := svc.CheckVersion()
		if err != nil {
			t.Fatalf("CheckVersion() returned unexpected error: %v", err)
		}

		// App version should be set (default is "dev" from ldflags)
		if info.AppVersion == "" {
			t.Error("AppVersion should not be empty")
		}

		// DB version may be "unknown" since test DB uses golden schema, not goose
		// Just verify it's populated
		if info.DbVersion == "" {
			t.Error("DbVersion should not be empty")
		}

		// Features map should contain expected keys
		expectedFeatures := []string{
			"basic_portfolio_management",
			"realized_gain_loss",
			"ibkr_integration",
			"materialized_view_performance",
			"fund_level_materialized_view",
			"materialized_sale_proceeds",
		}
		for _, feature := range expectedFeatures {
			if _, ok := info.Features[feature]; !ok {
				t.Errorf("Features map missing expected key %q", feature)
			}
		}
	})

	t.Run("all features are enabled", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := service.NewSystemService(db)

		info, err := svc.CheckVersion()
		if err != nil {
			t.Fatalf("CheckVersion() error: %v", err)
		}

		for feature, enabled := range info.Features {
			if !enabled {
				t.Errorf("Feature %q should be enabled, got false", feature)
			}
		}
	})

	t.Run("migration fields are populated", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := service.NewSystemService(db)

		info, err := svc.CheckVersion()
		if err != nil {
			t.Fatalf("CheckVersion() error: %v", err)
		}

		// MigrationNeeded is a bool, so it will always have a value.
		// With golden schema (no goose table), migration check may report false.
		// Just verify the structure is valid.
		if info.MigrationNeeded && info.MigrationMessage == nil {
			t.Error("MigrationMessage should be non-nil when MigrationNeeded is true")
		}
		if !info.MigrationNeeded && info.MigrationMessage != nil {
			t.Error("MigrationMessage should be nil when MigrationNeeded is false")
		}
	})
}
