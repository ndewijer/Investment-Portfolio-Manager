package service_test

import (
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// =============================================================================
// DATA LOADER SERVICE CONSTRUCTION
// =============================================================================

func TestNewDataLoaderService(t *testing.T) {
	t.Run("creates service with no options", func(t *testing.T) {
		svc := service.NewDataLoaderService()
		if svc == nil {
			t.Fatal("NewDataLoaderService() returned nil")
		}
	})

	t.Run("creates service with all options", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		pfRepo := repository.NewPortfolioFundRepository(db)
		fundRepo := repository.NewFundRepository(db)
		txSvc := testutil.NewTestTransactionService(t, db)
		divSvc := testutil.NewTestDividendService(t, db)
		rglSvc := testutil.NewTestRealizedGainLossService(t, db)

		svc := service.NewDataLoaderService(
			service.DataLoaderWithPortfolioFundRepository(pfRepo),
			service.DataLoaderWithFundRepository(fundRepo),
			service.DataLoaderWithTransactionService(txSvc),
			service.DataLoaderWithDividendService(divSvc),
			service.DataLoaderWithRealizedGainLossService(rglSvc),
		)
		if svc == nil {
			t.Fatal("NewDataLoaderService() returned nil")
		}
	})
}

// =============================================================================
// LOAD FOR PORTFOLIOS
// =============================================================================

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestDataLoaderService_LoadForPortfolios(t *testing.T) {
	t.Run("returns empty data for empty portfolios slice", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		pfRepo := repository.NewPortfolioFundRepository(db)
		fundRepo := repository.NewFundRepository(db)
		txSvc := testutil.NewTestTransactionService(t, db)
		divSvc := testutil.NewTestDividendService(t, db)
		rglSvc := testutil.NewTestRealizedGainLossService(t, db)

		svc := service.NewDataLoaderService(
			service.DataLoaderWithPortfolioFundRepository(pfRepo),
			service.DataLoaderWithFundRepository(fundRepo),
			service.DataLoaderWithTransactionService(txSvc),
			service.DataLoaderWithDividendService(divSvc),
			service.DataLoaderWithRealizedGainLossService(rglSvc),
		)

		startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

		data, err := svc.LoadForPortfolios(nil, startDate, endDate)
		if err != nil {
			t.Fatalf("LoadForPortfolios() error: %v", err)
		}
		if data == nil {
			t.Fatal("LoadForPortfolios() returned nil data")
		}
		if len(data.PFIDs) != 0 {
			t.Errorf("Expected 0 PFIDs, got %d", len(data.PFIDs))
		}
	})

	t.Run("returns empty pfIDs for portfolio with no funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		pfRepo := repository.NewPortfolioFundRepository(db)
		fundRepo := repository.NewFundRepository(db)
		txSvc := testutil.NewTestTransactionService(t, db)
		divSvc := testutil.NewTestDividendService(t, db)
		rglSvc := testutil.NewTestRealizedGainLossService(t, db)

		svc := service.NewDataLoaderService(
			service.DataLoaderWithPortfolioFundRepository(pfRepo),
			service.DataLoaderWithFundRepository(fundRepo),
			service.DataLoaderWithTransactionService(txSvc),
			service.DataLoaderWithDividendService(divSvc),
			service.DataLoaderWithRealizedGainLossService(rglSvc),
		)

		portfolio := testutil.NewPortfolio().Build(t, db)
		portfolios := []model.Portfolio{portfolio}

		startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

		data, err := svc.LoadForPortfolios(portfolios, startDate, endDate)
		if err != nil {
			t.Fatalf("LoadForPortfolios() error: %v", err)
		}
		if len(data.PFIDs) != 0 {
			t.Errorf("Expected 0 PFIDs for portfolio with no funds, got %d", len(data.PFIDs))
		}
	})

	t.Run("loads data for portfolio with transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		pfRepo := repository.NewPortfolioFundRepository(db)
		fundRepo := repository.NewFundRepository(db)
		txSvc := testutil.NewTestTransactionService(t, db)
		divSvc := testutil.NewTestDividendService(t, db)
		rglSvc := testutil.NewTestRealizedGainLossService(t, db)

		svc := service.NewDataLoaderService(
			service.DataLoaderWithPortfolioFundRepository(pfRepo),
			service.DataLoaderWithFundRepository(fundRepo),
			service.DataLoaderWithTransactionService(txSvc),
			service.DataLoaderWithDividendService(divSvc),
			service.DataLoaderWithRealizedGainLossService(rglSvc),
		)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		testutil.NewTransaction(pf.ID).
			WithDate(txDate).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		testutil.NewFundPrice(fund.ID).
			WithDate(txDate).
			WithPrice(10.0).
			Build(t, db)

		portfolios := []model.Portfolio{portfolio}
		startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

		data, err := svc.LoadForPortfolios(portfolios, startDate, endDate)
		if err != nil {
			t.Fatalf("LoadForPortfolios() error: %v", err)
		}

		if len(data.PFIDs) != 1 {
			t.Errorf("Expected 1 PFID, got %d", len(data.PFIDs))
		}
		if len(data.FundIDs) != 1 {
			t.Errorf("Expected 1 FundID, got %d", len(data.FundIDs))
		}
		if len(data.TransactionsByPF) == 0 {
			t.Error("Expected TransactionsByPF to have entries")
		}
		if len(data.FundPricesByFund) == 0 {
			t.Error("Expected FundPricesByFund to have entries")
		}

		// Check mappings
		if data.PortfolioFundToPortfolio[pf.ID] != portfolio.ID {
			t.Errorf("PortfolioFundToPortfolio[%s] = %q, want %q", pf.ID, data.PortfolioFundToPortfolio[pf.ID], portfolio.ID)
		}
		if data.PortfolioFundToFund[pf.ID] != fund.ID {
			t.Errorf("PortfolioFundToFund[%s] = %q, want %q", pf.ID, data.PortfolioFundToFund[pf.ID], fund.ID)
		}
	})

	t.Run("loads data from oldest transaction date not clamped to startDate", func(t *testing.T) {
		// Invariant: LoadForPortfolios must set dataStartDate := oldestTxDate unconditionally.
		// If startDate is AFTER the oldest transaction, data must still be loaded from
		// the transaction date so share counts and cost basis are correct.
		db := testutil.SetupTestDB(t)

		pfRepo := repository.NewPortfolioFundRepository(db)
		fundRepo := repository.NewFundRepository(db)
		txSvc := testutil.NewTestTransactionService(t, db)
		divSvc := testutil.NewTestDividendService(t, db)
		rglSvc := testutil.NewTestRealizedGainLossService(t, db)

		svc := service.NewDataLoaderService(
			service.DataLoaderWithPortfolioFundRepository(pfRepo),
			service.DataLoaderWithFundRepository(fundRepo),
			service.DataLoaderWithTransactionService(txSvc),
			service.DataLoaderWithDividendService(divSvc),
			service.DataLoaderWithRealizedGainLossService(rglSvc),
		)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Transaction on Jan 10
		txDate := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
		testutil.NewTransaction(pf.ID).
			WithDate(txDate).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Add prices from Jan 10 through Jan 25
		for d := txDate; !d.After(time.Date(2025, 1, 25, 0, 0, 0, 0, time.UTC)); d = d.AddDate(0, 0, 1) {
			testutil.NewFundPrice(fund.ID).WithDate(d).WithPrice(10.0).Build(t, db)
		}

		portfolios := []model.Portfolio{portfolio}

		// startDate is AFTER the transaction date
		startDate := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 1, 25, 0, 0, 0, 0, time.UTC)

		data, err := svc.LoadForPortfolios(portfolios, startDate, endDate)
		if err != nil {
			t.Fatalf("LoadForPortfolios() error: %v", err)
		}

		// Transactions must be loaded even though they predate startDate
		txns := data.TransactionsByPF[pf.ID]
		if len(txns) == 0 {
			t.Fatal("Expected transactions to be loaded from oldest transaction date, got none")
		}

		// Fund prices must include dates before startDate (back to transaction date)
		prices := data.FundPricesByFund[fund.ID]
		if len(prices) == 0 {
			t.Fatal("Expected fund prices to be loaded from oldest transaction date, got none")
		}

		// The earliest price loaded should be at or before the transaction date, not clamped to startDate
		earliestPrice := prices[0].Date
		if earliestPrice.After(txDate) {
			t.Errorf("Expected earliest price date to be at or before transaction date %s, got %s",
				txDate.Format("2006-01-02"), earliestPrice.Format("2006-01-02"))
		}

		// Verify the loaded transaction has non-zero shares
		if txns[0].Shares == 0 {
			t.Error("Expected loaded transaction to have non-zero shares")
		}
	})

	t.Run("loads portfolio fund details for single portfolio", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		pfRepo := repository.NewPortfolioFundRepository(db)
		fundRepo := repository.NewFundRepository(db)
		txSvc := testutil.NewTestTransactionService(t, db)
		divSvc := testutil.NewTestDividendService(t, db)
		rglSvc := testutil.NewTestRealizedGainLossService(t, db)

		svc := service.NewDataLoaderService(
			service.DataLoaderWithPortfolioFundRepository(pfRepo),
			service.DataLoaderWithFundRepository(fundRepo),
			service.DataLoaderWithTransactionService(txSvc),
			service.DataLoaderWithDividendService(divSvc),
			service.DataLoaderWithRealizedGainLossService(rglSvc),
		)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(50).WithCostPerShare(10.0).Build(t, db)
		testutil.NewFundPrice(fund.ID).WithDate(txDate).WithPrice(10.0).Build(t, db)

		portfolios := []model.Portfolio{portfolio}
		data, err := svc.LoadForPortfolios(portfolios, txDate, txDate)
		if err != nil {
			t.Fatalf("LoadForPortfolios() error: %v", err)
		}

		// Single portfolio should populate PortfolioFunds
		if len(data.PortfolioFunds) == 0 {
			t.Error("Expected PortfolioFunds to be populated for single portfolio")
		}
	})
}

// =============================================================================
// MAP REALIZED GAINS BY PF
// =============================================================================

func TestPortfolioData_MapRealizedGainsByPF(t *testing.T) {
	t.Run("maps realized gains to portfolio fund IDs", func(t *testing.T) {
		pfID1 := testutil.MakeID()
		pfID2 := testutil.MakeID()
		portfolioID := testutil.MakeID()
		fundID1 := testutil.MakeID()
		fundID2 := testutil.MakeID()

		data := &service.PortfolioData{
			PortfolioFunds: []model.PortfolioFundResponse{
				{ID: pfID1, FundID: fundID1, FundName: "Fund A"},
				{ID: pfID2, FundID: fundID2, FundName: "Fund B"},
			},
			RealizedGainsByPortfolio: map[string][]model.RealizedGainLoss{
				portfolioID: {
					{ID: "rgl1", FundID: fundID1, RealizedGainLoss: 100.0},
					{ID: "rgl2", FundID: fundID2, RealizedGainLoss: 200.0},
					{ID: "rgl3", FundID: fundID1, RealizedGainLoss: 50.0},
				},
			},
		}

		result := data.MapRealizedGainsByPF(portfolioID)

		if len(result[pfID1]) != 2 {
			t.Errorf("Expected 2 gains for pfID1, got %d", len(result[pfID1]))
		}
		if len(result[pfID2]) != 1 {
			t.Errorf("Expected 1 gain for pfID2, got %d", len(result[pfID2]))
		}
	})

	t.Run("returns empty map for portfolio with no gains", func(t *testing.T) {
		data := &service.PortfolioData{
			PortfolioFunds: []model.PortfolioFundResponse{
				{ID: "pf1", FundID: "fund1"},
			},
			RealizedGainsByPortfolio: map[string][]model.RealizedGainLoss{},
		}

		result := data.MapRealizedGainsByPF("nonexistent-portfolio")
		if len(result) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(result))
		}
	})

	t.Run("returns empty map when no portfolio funds match", func(t *testing.T) {
		portfolioID := testutil.MakeID()
		data := &service.PortfolioData{
			PortfolioFunds: []model.PortfolioFundResponse{
				{ID: "pf1", FundID: "fund-A"},
			},
			RealizedGainsByPortfolio: map[string][]model.RealizedGainLoss{
				portfolioID: {
					{ID: "rgl1", FundID: "fund-B", RealizedGainLoss: 100.0},
				},
			},
		}

		result := data.MapRealizedGainsByPF(portfolioID)
		if len(result) != 0 {
			t.Errorf("Expected empty map when no funds match, got %d entries", len(result))
		}
	})
}
