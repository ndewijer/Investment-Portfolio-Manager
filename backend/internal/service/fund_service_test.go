package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// roundSvc mirrors the service-level rounding for assertions.
func roundSvc(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

// =============================================================================
// FundService.GetFund
// =============================================================================

func TestFundService_GetFund(t *testing.T) {
	t.Run("returns fund by ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		fund := testutil.NewFund().WithSymbol("AAPL").WithName("Apple Inc").Build(t, db)

		got, err := svc.GetFund(fund.ID)
		if err != nil {
			t.Fatalf("GetFund() error: %v", err)
		}
		if got.ID != fund.ID {
			t.Errorf("expected ID %q, got %q", fund.ID, got.ID)
		}
		if got.Symbol != "AAPL" {
			t.Errorf("expected Symbol AAPL, got %q", got.Symbol)
		}
	})

	t.Run("returns error for nonexistent fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		_, err := svc.GetFund(testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for nonexistent fund")
		}
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound, got: %v", err)
		}
	})
}

// =============================================================================
// FundService.GetAllFunds
// =============================================================================

func TestFundService_GetAllFunds(t *testing.T) {
	t.Run("returns empty slice when no funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		funds, err := svc.GetAllFunds()
		if err != nil {
			t.Fatalf("GetAllFunds() error: %v", err)
		}
		if len(funds) != 0 {
			t.Errorf("expected 0 funds, got %d", len(funds))
		}
	})

	t.Run("returns all created funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		testutil.NewFund().Build(t, db)
		testutil.NewFund().Build(t, db)
		testutil.NewFund().Build(t, db)

		funds, err := svc.GetAllFunds()
		if err != nil {
			t.Fatalf("GetAllFunds() error: %v", err)
		}
		if len(funds) != 3 {
			t.Errorf("expected 3 funds, got %d", len(funds))
		}
	})
}

// =============================================================================
// FundService.GetSymbol
// =============================================================================

func TestFundService_GetSymbol(t *testing.T) {
	t.Run("returns cached symbol from DB", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		sym := testutil.NewSymbol().WithSymbol("MSFT").WithName("Microsoft").Build(t, db)

		got, err := svc.GetSymbol("MSFT")
		if err != nil {
			t.Fatalf("GetSymbol() error: %v", err)
		}
		if got.Symbol != sym.Symbol {
			t.Errorf("expected symbol %q, got %q", sym.Symbol, got.Symbol)
		}
	})

	t.Run("fetches from Yahoo on cache miss", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		got, err := svc.GetSymbol("TEST")
		if err != nil {
			t.Fatalf("GetSymbol() error: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil symbol")
		}
		if got.Symbol != "TEST" {
			t.Errorf("expected symbol TEST, got %q", got.Symbol)
		}
	})

	t.Run("returns error when Yahoo fails", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient().WithError(fmt.Errorf("network error"))
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		_, err := svc.GetSymbol("FAIL")
		if err == nil {
			t.Fatal("expected error when Yahoo fails")
		}
	})
}

// =============================================================================
// FundService.GetAllPortfolioFundListings
// =============================================================================

func TestFundService_GetAllPortfolioFundListings(t *testing.T) {
	t.Run("returns empty when no portfolio funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		listings, err := svc.GetAllPortfolioFundListings()
		if err != nil {
			t.Fatalf("GetAllPortfolioFundListings() error: %v", err)
		}
		if len(listings) != 0 {
			t.Errorf("expected 0 listings, got %d", len(listings))
		}
	})

	t.Run("returns portfolio fund listings", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		listings, err := svc.GetAllPortfolioFundListings()
		if err != nil {
			t.Fatalf("GetAllPortfolioFundListings() error: %v", err)
		}
		if len(listings) != 1 {
			t.Errorf("expected 1 listing, got %d", len(listings))
		}
	})
}

// =============================================================================
// FundService.CheckUsage
// =============================================================================

func TestFundService_CheckUsage(t *testing.T) {
	t.Run("returns in use when linked to portfolio even without transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		fund := testutil.NewFund().Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// CheckUsage returns portfolio rows via GROUP BY even with 0 transactions,
		// so the fund is considered "in use" when it has a portfolio_fund link.
		usage, err := svc.CheckUsage(fund.ID)
		if err != nil {
			t.Fatalf("CheckUsage() error: %v", err)
		}
		if !usage.InUsage {
			t.Error("expected fund in use (has portfolio_fund link)")
		}
	})

	t.Run("returns in use when transactions exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		fund := testutil.NewFund().Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)

		usage, err := svc.CheckUsage(fund.ID)
		if err != nil {
			t.Fatalf("CheckUsage() error: %v", err)
		}
		if !usage.InUsage {
			t.Error("expected fund in use")
		}
		if len(usage.Portfolios) == 0 {
			t.Error("expected non-empty portfolios list")
		}
	})
}

// =============================================================================
// FundService.CreatePortfolioFund
// =============================================================================

func TestFundService_CreatePortfolioFund(t *testing.T) {
	t.Run("creates portfolio fund relationship", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)

		err := svc.CreatePortfolioFund(ctx, request.CreatePortfolioFundRequest{
			PortfolioID: portfolio.ID,
			FundID:      fund.ID,
		})
		if err != nil {
			t.Fatalf("CreatePortfolioFund() error: %v", err)
		}

		testutil.AssertRowCount(t, db, "portfolio_fund", 1)
	})

	t.Run("fails when portfolio does not exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		fund := testutil.NewFund().Build(t, db)

		err := svc.CreatePortfolioFund(ctx, request.CreatePortfolioFundRequest{
			PortfolioID: testutil.MakeID(),
			FundID:      fund.ID,
		})
		if err == nil {
			t.Fatal("expected error for nonexistent portfolio")
		}
	})

	t.Run("fails when fund does not exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)

		err := svc.CreatePortfolioFund(ctx, request.CreatePortfolioFundRequest{
			PortfolioID: portfolio.ID,
			FundID:      testutil.MakeID(),
		})
		if err == nil {
			t.Fatal("expected error for nonexistent fund")
		}
	})
}

// =============================================================================
// FundService.DeletePortfolioFund
// =============================================================================

func TestFundService_DeletePortfolioFund(t *testing.T) {
	t.Run("deletes existing portfolio fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		err := svc.DeletePortfolioFund(ctx, pf.ID)
		if err != nil {
			t.Fatalf("DeletePortfolioFund() error: %v", err)
		}

		testutil.AssertRowCount(t, db, "portfolio_fund", 0)
	})

	t.Run("fails for nonexistent portfolio fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		err := svc.DeletePortfolioFund(ctx, testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for nonexistent portfolio fund")
		}
	})
}

// =============================================================================
// FundService.CreateFund
// =============================================================================

func TestFundService_CreateFund(t *testing.T) {
	t.Run("creates fund with generated ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		fund, err := svc.CreateFund(ctx, request.CreateFundRequest{
			Name:           "Test Fund",
			Isin:           "US1234567890",
			Symbol:         "TFND",
			Currency:       "USD",
			Exchange:       "NASDAQ",
			InvestmentType: "STOCK",
			DividendType:   "NONE",
		})
		if err != nil {
			t.Fatalf("CreateFund() error: %v", err)
		}
		if fund.ID == "" {
			t.Error("expected non-empty fund ID")
		}
		if fund.Symbol != "TFND" {
			t.Errorf("expected Symbol TFND, got %q", fund.Symbol)
		}

		testutil.AssertRowCount(t, db, "fund", 1)
	})
}

// =============================================================================
// FundService.UpdateFund
// =============================================================================

func TestFundService_UpdateFund(t *testing.T) {
	t.Run("updates specified fields only", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		fund := testutil.NewFund().WithName("OldName").WithSymbol("OLD").Build(t, db)

		newName := "NewName"
		updated, err := svc.UpdateFund(ctx, fund.ID, request.UpdateFundRequest{
			Name: &newName,
		})
		if err != nil {
			t.Fatalf("UpdateFund() error: %v", err)
		}
		if updated.Name != "NewName" {
			t.Errorf("expected Name NewName, got %q", updated.Name)
		}
		// Symbol should remain unchanged
		if updated.Symbol != fund.Symbol {
			t.Errorf("expected Symbol %q unchanged, got %q", fund.Symbol, updated.Symbol)
		}
	})

	t.Run("updates all fields when provided", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		fund := testutil.NewFund().Build(t, db)

		newName := "Updated"
		newSymbol := "UPD"
		newCurrency := "EUR"
		newExchange := "LSE"
		newIsin := "GB1234567890"
		newInvestmentType := "ETF"
		newDividendType := "CASH"

		updated, err := svc.UpdateFund(ctx, fund.ID, request.UpdateFundRequest{
			Name:           &newName,
			Symbol:         &newSymbol,
			Currency:       &newCurrency,
			Exchange:       &newExchange,
			Isin:           &newIsin,
			InvestmentType: &newInvestmentType,
			DividendType:   &newDividendType,
		})
		if err != nil {
			t.Fatalf("UpdateFund() error: %v", err)
		}
		if updated.Name != newName || updated.Symbol != newSymbol || updated.Currency != newCurrency {
			t.Errorf("fields not updated correctly")
		}
	})

	t.Run("fails for nonexistent fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		newName := "Test"
		_, err := svc.UpdateFund(ctx, testutil.MakeID(), request.UpdateFundRequest{
			Name: &newName,
		})
		if err == nil {
			t.Fatal("expected error for nonexistent fund")
		}
	})
}

// =============================================================================
// FundService.DeleteFund
// =============================================================================

func TestFundService_DeleteFund(t *testing.T) {
	t.Run("deletes unused fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		fund := testutil.NewFund().Build(t, db)

		err := svc.DeleteFund(ctx, fund.ID)
		if err != nil {
			t.Fatalf("DeleteFund() error: %v", err)
		}
		testutil.AssertRowCount(t, db, "fund", 0)
	})

	t.Run("fails for nonexistent fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		err := svc.DeleteFund(ctx, testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for nonexistent fund")
		}
	})

	t.Run("returns ErrFundInUse when fund has transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		fund := testutil.NewFund().Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)

		err := svc.DeleteFund(ctx, fund.ID)
		if !errors.Is(err, apperrors.ErrFundInUse) {
			t.Errorf("expected ErrFundInUse, got: %v", err)
		}

		// Fund should still exist
		testutil.AssertRowCount(t, db, "fund", 1)
	})

	t.Run("rejects deletion of fund linked to portfolio even without transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)
		ctx := context.Background()

		fund := testutil.NewFund().Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// CheckUsage returns rows via GROUP BY even with 0 transactions,
		// so the fund is considered in-use and deletion is blocked.
		err := svc.DeleteFund(ctx, fund.ID)
		if !errors.Is(err, apperrors.ErrFundInUse) {
			t.Errorf("expected ErrFundInUse, got %v", err)
		}
	})
}

// =============================================================================
// FundService.LoadFundPrices
// =============================================================================

func TestFundService_LoadFundPrices(t *testing.T) {
	t.Run("returns prices for fund in date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		fund := testutil.NewFund().Build(t, db)
		d1 := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
		d2 := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
		testutil.NewFundPrice(fund.ID).WithDate(d1).WithPrice(100.0).Build(t, db)
		testutil.NewFundPrice(fund.ID).WithDate(d2).WithPrice(101.0).Build(t, db)

		prices, err := svc.LoadFundPrices([]string{fund.ID}, d1, d2, true)
		if err != nil {
			t.Fatalf("LoadFundPrices() error: %v", err)
		}
		if len(prices[fund.ID]) != 2 {
			t.Errorf("expected 2 prices, got %d", len(prices[fund.ID]))
		}
	})

	t.Run("returns empty for no matching prices", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		prices, err := svc.LoadFundPrices([]string{testutil.MakeID()},
			time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC), true)
		if err != nil {
			t.Fatalf("LoadFundPrices() error: %v", err)
		}
		if len(prices) != 0 {
			t.Errorf("expected empty map, got %d entries", len(prices))
		}
	})
}

// =============================================================================
// FundService.UpdateCurrentFundPrice
// =============================================================================

func TestFundService_UpdateCurrentFundPrice(t *testing.T) {
	t.Run("inserts new price and returns inserted=true", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)

		price, inserted, err := svc.UpdateCurrentFundPrice(context.Background(), fund.ID)
		if err != nil {
			t.Fatalf("UpdateCurrentFundPrice() error: %v", err)
		}
		if !inserted {
			t.Error("expected inserted=true for new price")
		}
		if price.Price <= 0 {
			t.Errorf("expected positive price, got %f", price.Price)
		}
	})

	t.Run("returns existing price when already present", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)

		// Insert price for yesterday
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
		testutil.NewFundPrice(fund.ID).WithDate(yesterday).WithPrice(42.0).Build(t, db)

		price, inserted, err := svc.UpdateCurrentFundPrice(context.Background(), fund.ID)
		if err != nil {
			t.Fatalf("UpdateCurrentFundPrice() error: %v", err)
		}
		if inserted {
			t.Error("expected inserted=false for existing price")
		}
		if price.Price != 42.0 {
			t.Errorf("expected price 42.0, got %f", price.Price)
		}
	})

	t.Run("fails for fund without symbol", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		fund := testutil.NewFund().WithSymbol("").Build(t, db)

		_, _, err := svc.UpdateCurrentFundPrice(context.Background(), fund.ID)
		if !errors.Is(err, apperrors.ErrInvalidSymbol) {
			t.Errorf("expected ErrInvalidSymbol, got: %v", err)
		}
	})

	t.Run("fails for nonexistent fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		_, _, err := svc.UpdateCurrentFundPrice(context.Background(), testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for nonexistent fund")
		}
	})

	t.Run("triggers materialized invalidation on insert", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		mock := testutil.NewMockMaterializedInvalidator(1)
		svc.SetMaterializedInvalidator(mock)

		fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)

		_, inserted, err := svc.UpdateCurrentFundPrice(context.Background(), fund.ID)
		if err != nil {
			t.Fatalf("UpdateCurrentFundPrice() error: %v", err)
		}
		if !inserted {
			t.Skip("price already existed, cannot test invalidation")
		}

		if !mock.WaitForCall(2 * time.Second) {
			t.Fatal("expected invalidator call after price insert")
		}
		calls := mock.Calls()
		if calls[0].FundID != fund.ID {
			t.Errorf("expected fundID %q, got %q", fund.ID, calls[0].FundID)
		}
	})

	t.Run("does not trigger invalidation when price already exists", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		mock := testutil.NewMockMaterializedInvalidator(1)
		svc.SetMaterializedInvalidator(mock)

		fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
		testutil.NewFundPrice(fund.ID).WithDate(yesterday).WithPrice(42.0).Build(t, db)

		_, _, err := svc.UpdateCurrentFundPrice(context.Background(), fund.ID)
		if err != nil {
			t.Fatalf("UpdateCurrentFundPrice() error: %v", err)
		}

		// Should NOT have triggered invalidation
		if mock.CallCount() != 0 {
			t.Errorf("expected 0 invalidator calls, got %d", mock.CallCount())
		}
	})
}

// =============================================================================
// FundService.UpdateHistoricalFundPrice
// =============================================================================

func TestFundService_UpdateHistoricalFundPrice(t *testing.T) {
	t.Run("fails for fund without symbol", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		fund := testutil.NewFund().WithSymbol("").Build(t, db)

		_, err := svc.UpdateHistoricalFundPrice(context.Background(), fund.ID)
		if !errors.Is(err, apperrors.ErrInvalidSymbol) {
			t.Errorf("expected ErrInvalidSymbol, got: %v", err)
		}
	})

	t.Run("fails when no portfolio funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)

		_, err := svc.UpdateHistoricalFundPrice(context.Background(), fund.ID)
		if err == nil {
			t.Fatal("expected error when no portfolio funds exist")
		}
	})

	t.Run("fails when no transactions exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		_, err := svc.UpdateHistoricalFundPrice(context.Background(), fund.ID)
		if err == nil {
			t.Fatal("expected error when no transactions exist")
		}
	})

	t.Run("backfills missing prices", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Transaction sets the start boundary
		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).Build(t, db)

		count, err := svc.UpdateHistoricalFundPrice(context.Background(), fund.ID)
		if err != nil {
			t.Fatalf("UpdateHistoricalFundPrice() error: %v", err)
		}
		// The mock returns 5 days of data; at least some should be inserted
		if count <= 0 {
			t.Errorf("expected at least 1 price inserted, got %d", count)
		}
	})
}

// =============================================================================
// FundService.UpdateAllFundHistory
// =============================================================================

func TestFundService_UpdateAllFundHistory(t *testing.T) {
	t.Run("returns ErrFundNotFound when no funds exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		_, err := svc.UpdateAllFundHistory(context.Background())
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound, got: %v", err)
		}
	})

	t.Run("reports errors for funds without portfolio funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)

		// Fund with no portfolio funds means UpdateHistoricalFundPrice errors
		testutil.NewFund().WithSymbol("TEST1").Build(t, db)

		_, err := svc.UpdateAllFundHistory(context.Background())
		// All funds should error, so the overall call returns an error
		if err == nil {
			t.Fatal("expected error when all funds fail")
		}
	})
}

// =============================================================================
// FundService.GetPortfolioFunds — enriched metrics
// =============================================================================

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestFundService_GetPortfolioFunds(t *testing.T) {
	t.Run("returns empty slice for portfolio with no funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 0 {
			t.Errorf("expected 0 funds, got %d", len(funds))
		}
	})

	t.Run("returns error for nonexistent portfolio", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestFundService(t, db)

		_, err := svc.GetPortfolioFunds(testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for nonexistent portfolio")
		}
	})

	t.Run("calculates metrics for buy-only scenario", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyDate := time.Now().UTC().AddDate(0, -1, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(buyDate).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Add a recent price
		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(12.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		if roundSvc(f.TotalShares) != roundSvc(100.0) {
			t.Errorf("expected TotalShares=100, got %f", f.TotalShares)
		}
		if roundSvc(f.TotalCost) != roundSvc(1000.0) {
			t.Errorf("expected TotalCost=1000, got %f", f.TotalCost)
		}
		if roundSvc(f.LatestPrice) != roundSvc(12.0) {
			t.Errorf("expected LatestPrice=12, got %f", f.LatestPrice)
		}
		if roundSvc(f.CurrentValue) != roundSvc(1200.0) {
			t.Errorf("expected CurrentValue=1200, got %f", f.CurrentValue)
		}
		expectedUnrealized := 1200.0 - 1000.0
		if roundSvc(f.UnrealizedGainLoss) != roundSvc(expectedUnrealized) {
			t.Errorf("expected UnrealizedGainLoss=%f, got %f", expectedUnrealized, f.UnrealizedGainLoss)
		}
	})

	t.Run("calculates metrics with buy and sell", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyDate := time.Now().UTC().AddDate(0, -2, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(buyDate).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		sellDate := time.Now().UTC().AddDate(0, -1, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(sellDate).
			WithType("sell").
			WithShares(40).
			WithCostPerShare(15.0).
			Build(t, db)

		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(14.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		// After selling 40 of 100 shares: 60 remain
		if roundSvc(f.TotalShares) != roundSvc(60.0) {
			t.Errorf("expected TotalShares=60, got %f", f.TotalShares)
		}
		// Cost: original 1000, after sell: (1000/100)*60 = 600
		expectedCost := (1000.0 / 100.0) * 60.0
		if roundSvc(f.TotalCost) != roundSvc(expectedCost) {
			t.Errorf("expected TotalCost=%f, got %f", expectedCost, f.TotalCost)
		}
	})

	t.Run("includes fee transactions in cost and fees", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyDate := time.Now().UTC().AddDate(0, -1, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(buyDate).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		feeDate := time.Now().UTC().AddDate(0, 0, -15)
		testutil.NewTransaction(pf.ID).
			WithDate(feeDate).
			WithType("fee").
			WithShares(0).
			WithCostPerShare(25.0).
			Build(t, db)

		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(10.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		if roundSvc(f.TotalFees) != roundSvc(25.0) {
			t.Errorf("expected TotalFees=25, got %f", f.TotalFees)
		}
		// Cost should include the fee
		expectedCost := 1000.0 + 25.0
		if roundSvc(f.TotalCost) != roundSvc(expectedCost) {
			t.Errorf("expected TotalCost=%f, got %f", expectedCost, f.TotalCost)
		}
	})
}

// =============================================================================
// Helpers
// =============================================================================

// buildFullFundService creates a FundService with all sub-services wired,
// including dividend, realized gain/loss, transaction, and data loader.
// This is needed for GetPortfolioFunds which calls enrichPortfolioFundsWithMetrics.
func buildFullFundService(t *testing.T, db *sql.DB) *service.FundService {
	t.Helper()

	fundRepo := repository.NewFundRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewTransactionService(db, transactionRepo, pfRepo, repository.NewRealizedGainLossRepository(db), repository.NewIbkrRepository(db))
	dividendService := service.NewDividendService(db, repository.NewDividendRepository(db), pfRepo, transactionRepo)
	realizedGainLossService := service.NewRealizedGainLossService(repository.NewRealizedGainLossRepository(db))
	dataloaderService := service.NewDataLoaderService(
		service.DataLoaderWithPortfolioFundRepository(pfRepo),
		service.DataLoaderWithFundRepository(fundRepo),
		service.DataLoaderWithTransactionService(transactionService),
		service.DataLoaderWithDividendService(dividendService),
		service.DataLoaderWithRealizedGainLossService(realizedGainLossService),
	)
	portfolioRepo := repository.NewPortfolioRepository(db)

	return service.NewFundService(
		db,
		service.FundWithFundRepo(fundRepo),
		service.FundWithPortfolioFundRepo(pfRepo),
		service.FundWithTransactionService(transactionService),
		service.FundWithDividendService(dividendService),
		service.FundWithRealizedGainLossService(realizedGainLossService),
		service.FundWithDataLoaderService(dataloaderService),
		service.FundWithPortfolioRepo(portfolioRepo),
		service.FundWithYahooClient(testutil.NewMockYahooClient()),
	)
}
