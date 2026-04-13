package service_test

import (
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// TestPortfolioService_GetAllPortfolios tests the GetAllPortfolios method.
//
// WHY: Portfolio retrieval is a fundamental operation. This ensures the service
// correctly returns all portfolios from the database, including edge cases like
// empty databases and multiple portfolios.
func TestPortfolioService_GetAllPortfolios(t *testing.T) {
	t.Run("returns empty slice when no portfolios exist", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err != nil {
			t.Fatalf("GetAllPortfolios() returned unexpected error: %v", err)
		}

		if len(portfolios) != 0 {
			t.Errorf("Expected empty slice, got %d portfolios", len(portfolios))
		}
	})

	t.Run("returns all portfolios including archived", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Create test data
		p1 := testutil.CreatePortfolio(t, db, "Active Portfolio")
		p2 := testutil.CreateArchivedPortfolio(t, db, "Archived Portfolio")

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err != nil {
			t.Fatalf("GetAllPortfolios() returned unexpected error: %v", err)
		}

		if len(portfolios) != 2 {
			t.Errorf("Expected 2 portfolios, got %d", len(portfolios))
		}

		// Verify both portfolios are present
		foundActive := false
		foundArchived := false
		for _, p := range portfolios {
			if p.ID == p1.ID && p.Name == "Active Portfolio" {
				foundActive = true
			}
			if p.ID == p2.ID && p.Name == "Archived Portfolio" && p.IsArchived {
				foundArchived = true
			}
		}

		if !foundActive {
			t.Error("Active portfolio not found in results")
		}
		if !foundArchived {
			t.Error("Archived portfolio not found in results")
		}
	})

	t.Run("returns portfolios in correct order", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Create multiple portfolios
		testutil.CreatePortfolios(t, db, 5)

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err != nil {
			t.Fatalf("GetAllPortfolios() returned unexpected error: %v", err)
		}

		if len(portfolios) != 5 {
			t.Errorf("Expected 5 portfolios, got %d", len(portfolios))
		}
	})
}

// TestPortfolioService_GetAllPortfolios_DatabaseErrors tests error handling.
//
// WHY: The service must gracefully handle database errors without panicking,
// ensuring the application remains stable when the database is unavailable.
func TestPortfolioService_GetAllPortfolios_DatabaseErrors(t *testing.T) {
	t.Run("handles closed database connection", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Close database to force error
		db.Close()

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err == nil {
			t.Error("Expected error when database is closed, got nil")
		}

		if portfolios != nil {
			t.Errorf("Expected nil portfolios on error, got %v", portfolios)
		}
	})
}

// =============================================================================
// PortfolioService.LoadActivePortfolios
// =============================================================================

func TestPortfolioService_LoadActivePortfolios(t *testing.T) {
	t.Run("returns only active non-excluded portfolios", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		active := testutil.NewPortfolio().WithName("Active").Build(t, db)
		testutil.NewPortfolio().WithName("Archived").Archived().Build(t, db)
		testutil.NewPortfolio().WithName("Excluded").ExcludedFromOverview().Build(t, db)

		portfolios, err := svc.LoadActivePortfolios()
		if err != nil {
			t.Fatalf("LoadActivePortfolios() error: %v", err)
		}

		if len(portfolios) != 1 {
			t.Fatalf("expected 1 active portfolio, got %d", len(portfolios))
		}
		if portfolios[0].ID != active.ID {
			t.Errorf("expected portfolio ID %q, got %q", active.ID, portfolios[0].ID)
		}
	})

	t.Run("returns empty slice when all portfolios are archived", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		testutil.NewPortfolio().WithName("Archived 1").Archived().Build(t, db)
		testutil.NewPortfolio().WithName("Archived 2").Archived().Build(t, db)

		portfolios, err := svc.LoadActivePortfolios()
		if err != nil {
			t.Fatalf("LoadActivePortfolios() error: %v", err)
		}
		if len(portfolios) != 0 {
			t.Errorf("expected 0 portfolios, got %d", len(portfolios))
		}
	})

	t.Run("returns empty slice when no portfolios exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		portfolios, err := svc.LoadActivePortfolios()
		if err != nil {
			t.Fatalf("LoadActivePortfolios() error: %v", err)
		}
		if len(portfolios) != 0 {
			t.Errorf("expected 0 portfolios, got %d", len(portfolios))
		}
	})

	t.Run("returns multiple active portfolios", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		testutil.NewPortfolio().WithName("Active 1").Build(t, db)
		testutil.NewPortfolio().WithName("Active 2").Build(t, db)
		testutil.NewPortfolio().WithName("Active 3").Build(t, db)

		portfolios, err := svc.LoadActivePortfolios()
		if err != nil {
			t.Fatalf("LoadActivePortfolios() error: %v", err)
		}
		if len(portfolios) != 3 {
			t.Errorf("expected 3 portfolios, got %d", len(portfolios))
		}
	})
}

// =============================================================================
// PortfolioService.LoadAllPortfolioFunds
// =============================================================================

func TestPortfolioService_LoadAllPortfolioFunds(t *testing.T) {
	t.Run("returns fund mappings for portfolios with funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().WithName("Fund A").Build(t, db)
		fund2 := testutil.NewFund().WithName("Fund B").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		portfolios := []model.Portfolio{portfolio}
		fundsByPortfolio, pfToPortfolio, pfToFund, pfIDs, fundIDs, err := svc.LoadAllPortfolioFunds(portfolios)
		if err != nil {
			t.Fatalf("LoadAllPortfolioFunds() error: %v", err)
		}

		if len(fundsByPortfolio[portfolio.ID]) != 2 {
			t.Errorf("expected 2 funds for portfolio, got %d", len(fundsByPortfolio[portfolio.ID]))
		}
		if len(pfIDs) != 2 {
			t.Errorf("expected 2 portfolio fund IDs, got %d", len(pfIDs))
		}
		if len(fundIDs) != 2 {
			t.Errorf("expected 2 fund IDs, got %d", len(fundIDs))
		}
		if len(pfToPortfolio) != 2 {
			t.Errorf("expected 2 entries in pfToPortfolio map, got %d", len(pfToPortfolio))
		}
		if len(pfToFund) != 2 {
			t.Errorf("expected 2 entries in pfToFund map, got %d", len(pfToFund))
		}
	})

	t.Run("returns empty maps for portfolio with no funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)

		portfolios := []model.Portfolio{portfolio}
		fundsByPortfolio, _, _, pfIDs, fundIDs, err := svc.LoadAllPortfolioFunds(portfolios)
		if err != nil {
			t.Fatalf("LoadAllPortfolioFunds() error: %v", err)
		}

		if len(fundsByPortfolio[portfolio.ID]) != 0 {
			t.Errorf("expected 0 funds for empty portfolio, got %d", len(fundsByPortfolio[portfolio.ID]))
		}
		if len(pfIDs) != 0 {
			t.Errorf("expected 0 portfolio fund IDs, got %d", len(pfIDs))
		}
		if len(fundIDs) != 0 {
			t.Errorf("expected 0 fund IDs, got %d", len(fundIDs))
		}
	})

	t.Run("returns empty maps for empty portfolio slice", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		_, _, _, pfIDs, fundIDs, err := svc.LoadAllPortfolioFunds(nil)
		if err != nil {
			t.Fatalf("LoadAllPortfolioFunds() error: %v", err)
		}
		if len(pfIDs) != 0 {
			t.Errorf("expected 0 portfolio fund IDs, got %d", len(pfIDs))
		}
		if len(fundIDs) != 0 {
			t.Errorf("expected 0 fund IDs, got %d", len(fundIDs))
		}
	})
}

// Example of table-driven tests (another Go pattern)
func TestPortfolioService_GetAllPortfolios_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		setupData     func(*testing.T, *testutil.PortfolioBuilder) *testutil.PortfolioBuilder
		expectedCount int
	}{
		{
			name:          "no portfolios",
			setupData:     nil,
			expectedCount: 0,
		},
		{
			name: "single active portfolio",
			setupData: func(_ *testing.T, b *testutil.PortfolioBuilder) *testutil.PortfolioBuilder {
				return b.WithName("Single Portfolio")
			},
			expectedCount: 1,
		},
		{
			name: "single archived portfolio",
			setupData: func(_ *testing.T, b *testutil.PortfolioBuilder) *testutil.PortfolioBuilder {
				return b.WithName("Archived").Archived()
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := testutil.SetupTestDB(t)
			svc := testutil.NewTestPortfolioService(t, db)

			if tt.setupData != nil {
				builder := testutil.NewPortfolio()
				tt.setupData(t, builder).Build(t, db)
			}

			// Execute
			portfolios, err := svc.GetAllPortfolios()

			// Assert
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(portfolios) != tt.expectedCount {
				t.Errorf("Expected %d portfolios, got %d", tt.expectedCount, len(portfolios))
			}
		})
	}
}
