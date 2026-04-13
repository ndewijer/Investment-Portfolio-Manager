package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// =============================================================================
// DividendService.GetAllDividend
// =============================================================================

func TestDividendService_GetAllDividend(t *testing.T) {
	t.Run("returns empty slice when no dividends exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		dividends, err := svc.GetAllDividend()
		if err != nil {
			t.Fatalf("GetAllDividend() error: %v", err)
		}
		if len(dividends) != 0 {
			t.Errorf("expected 0 dividends, got %d", len(dividends))
		}
	})

	t.Run("returns all dividends", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)).
			Build(t, db)
		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 2, 17, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		dividends, err := svc.GetAllDividend()
		if err != nil {
			t.Fatalf("GetAllDividend() error: %v", err)
		}
		if len(dividends) != 2 {
			t.Errorf("expected 2 dividends, got %d", len(dividends))
		}
	})
}

// =============================================================================
// DividendService.GetDividend
// =============================================================================

func TestDividendService_GetDividend(t *testing.T) {
	t.Run("returns dividend by ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)).
			WithDividendPerShare(0.50).
			Build(t, db)

		got, err := svc.GetDividend(div.ID)
		if err != nil {
			t.Fatalf("GetDividend() error: %v", err)
		}
		if got.ID != div.ID {
			t.Errorf("expected ID %q, got %q", div.ID, got.ID)
		}
		if got.DividendPerShare != 0.50 {
			t.Errorf("expected DividendPerShare=0.50, got %f", got.DividendPerShare)
		}
	})

	t.Run("returns error for nonexistent dividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		_, err := svc.GetDividend(testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for nonexistent dividend")
		}
		if !errors.Is(err, apperrors.ErrDividendNotFound) {
			t.Errorf("expected ErrDividendNotFound, got: %v", err)
		}
	})
}

// =============================================================================
// DividendService.GetDividendFund
// =============================================================================

func TestDividendService_GetDividendFund(t *testing.T) {
	t.Run("returns dividends by portfolio ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		results, err := svc.GetDividendFund(portfolio.ID, "")
		if err != nil {
			t.Fatalf("GetDividendFund() error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 dividend fund, got %d", len(results))
		}
	})

	t.Run("returns dividends by fund ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		results, err := svc.GetDividendFund("", fund.ID)
		if err != nil {
			t.Fatalf("GetDividendFund() error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 dividend fund, got %d", len(results))
		}
	})

	t.Run("returns empty for portfolio with funds but no dividends", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		// Need a portfolio_fund link so the existence check passes
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		results, err := svc.GetDividendFund(portfolio.ID, "")
		if err != nil {
			t.Fatalf("GetDividendFund() error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

// =============================================================================
// DividendService.CreateDividend
// =============================================================================

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestDividendService_CreateDividend(t *testing.T) {
	t.Run("creates dividend for CASH fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ID == "" {
			t.Error("expected non-empty dividend ID")
		}
		if div.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("expected COMPLETED for CASH fund, got %q", div.ReinvestmentStatus)
		}
		if div.TotalAmount != 50.0 {
			t.Errorf("expected TotalAmount=50.0 (100*0.50), got %f", div.TotalAmount)
		}
		if div.SharesOwned != 100.0 {
			t.Errorf("expected SharesOwned=100, got %f", div.SharesOwned)
		}
	})

	t.Run("creates dividend for STOCK fund without reinvestment as PENDING", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ReinvestmentStatus != "PENDING" {
			t.Errorf("expected PENDING for STOCK fund without reinvestment, got %q", div.ReinvestmentStatus)
		}
	})

	t.Run("creates dividend with reinvestment transaction for STOCK fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:    pf.ID,
			RecordDate:         "2025-01-20",
			ExDividendDate:     "2025-01-18",
			DividendPerShare:   0.50,
			BuyOrderDate:       "2025-01-22",
			ReinvestmentShares: 5.0,
			ReinvestmentPrice:  10.0,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ReinvestmentTransactionID == "" {
			t.Error("expected reinvestment transaction ID to be set")
		}
		// 100 * 0.50 = 50.0 total, reinvested 5*10=50.0 -> COMPLETED
		if div.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("expected COMPLETED for full reinvestment, got %q", div.ReinvestmentStatus)
		}

		// Verify transaction was created
		testutil.AssertRowCount(t, db, "transaction", 2) // original buy + reinvestment dividend
	})

	t.Run("partial reinvestment marks PARTIAL", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:    pf.ID,
			RecordDate:         "2025-01-20",
			ExDividendDate:     "2025-01-18",
			DividendPerShare:   0.50,
			BuyOrderDate:       "2025-01-22",
			ReinvestmentShares: 2.0,
			ReinvestmentPrice:  10.0,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		// 100 * 0.50 = 50.0 total, reinvested 2*10=20.0 -> PARTIAL
		if div.ReinvestmentStatus != "PARTIAL" {
			t.Errorf("expected PARTIAL for partial reinvestment, got %q", div.ReinvestmentStatus)
		}
	})

	t.Run("fails for fund with DividendType None", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("None").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		_, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err == nil {
			t.Fatal("expected error for fund with DividendType None")
		}
	})

	t.Run("fails for nonexistent portfolio fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		_, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  testutil.MakeID(),
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err == nil {
			t.Fatal("expected error for nonexistent portfolio fund")
		}
	})

	t.Run("calculates shares owned from transactions on ex-dividend date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares on Jan 10
		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		// Sell 30 shares on Jan 15
		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
			WithType("sell").WithShares(30).WithCostPerShare(15.0).
			Build(t, db)

		// Dividend with ex-date Jan 18 (after sell, so 70 shares)
		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.SharesOwned != 70.0 {
			t.Errorf("expected SharesOwned=70 (100-30), got %f", div.SharesOwned)
		}
		if div.TotalAmount != 35.0 {
			t.Errorf("expected TotalAmount=35 (70*0.50), got %f", div.TotalAmount)
		}
	})
}

// =============================================================================
// DividendService.UpdateDividend
// =============================================================================

func TestDividendService_UpdateDividend(t *testing.T) {
	t.Run("updates dividend per share and recalculates total", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)).
			WithDividendPerShare(0.50).
			WithSharesOwned(100).
			Build(t, db)

		newDPS := 0.75
		updated, err := svc.UpdateDividend(ctx, div.ID, request.UpdateDividendRequest{
			DividendPerShare: &newDPS,
		})
		if err != nil {
			t.Fatalf("UpdateDividend() error: %v", err)
		}
		if updated.DividendPerShare != 0.75 {
			t.Errorf("expected DividendPerShare=0.75, got %f", updated.DividendPerShare)
		}
		// TotalAmount should be recalculated: 100 * 0.75 = 75
		if updated.TotalAmount != 75.0 {
			t.Errorf("expected TotalAmount=75, got %f", updated.TotalAmount)
		}
	})

	t.Run("updates ex-dividend date and recalculates shares", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)
		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)).
			WithType("sell").WithShares(50).WithCostPerShare(15.0).
			Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)).
			WithDividendPerShare(0.50).
			WithSharesOwned(100).
			Build(t, db)

		// Move ex-date to Jan 25 (after the sell)
		newExDate := "2025-01-25"
		updated, err := svc.UpdateDividend(ctx, div.ID, request.UpdateDividendRequest{
			ExDividendDate: &newExDate,
		})
		if err != nil {
			t.Fatalf("UpdateDividend() error: %v", err)
		}
		// After sell: 100-50=50 shares
		if updated.SharesOwned != 50.0 {
			t.Errorf("expected SharesOwned=50 after date change, got %f", updated.SharesOwned)
		}
	})

	t.Run("fails for nonexistent dividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		newDPS := 0.75
		_, err := svc.UpdateDividend(ctx, testutil.MakeID(), request.UpdateDividendRequest{
			DividendPerShare: &newDPS,
		})
		if err == nil {
			t.Fatal("expected error for nonexistent dividend")
		}
	})

	t.Run("preserves COMPLETED status when no reinvestment info", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		// Create via service to get COMPLETED status
		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ReinvestmentStatus != "COMPLETED" {
			t.Fatalf("precondition: expected COMPLETED, got %q", div.ReinvestmentStatus)
		}

		// Update only DPS, should remain COMPLETED
		newDPS := 0.75
		updated, err := svc.UpdateDividend(ctx, div.ID, request.UpdateDividendRequest{
			DividendPerShare: &newDPS,
		})
		if err != nil {
			t.Fatalf("UpdateDividend() error: %v", err)
		}
		if updated.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("expected COMPLETED preserved, got %q", updated.ReinvestmentStatus)
		}
	})
}

// =============================================================================
// DividendService.DeleteDividend
// =============================================================================

func TestDividendService_DeleteDividend(t *testing.T) {
	t.Run("deletes dividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		err := svc.DeleteDividend(ctx, div.ID)
		if err != nil {
			t.Fatalf("DeleteDividend() error: %v", err)
		}

		testutil.AssertRowCount(t, db, "dividend", 0)
	})

	t.Run("fails for nonexistent dividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		err := svc.DeleteDividend(ctx, testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for nonexistent dividend")
		}
	})

	t.Run("deletes associated reinvestment transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:    pf.ID,
			RecordDate:         "2025-01-20",
			ExDividendDate:     "2025-01-18",
			DividendPerShare:   0.50,
			BuyOrderDate:       "2025-01-22",
			ReinvestmentShares: 5.0,
			ReinvestmentPrice:  10.0,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ReinvestmentTransactionID == "" {
			t.Fatal("precondition: expected reinvestment transaction ID")
		}

		// Should have 2 transactions: original buy + reinvestment dividend
		testutil.AssertRowCount(t, db, "transaction", 2)

		err = svc.DeleteDividend(ctx, div.ID)
		if err != nil {
			t.Fatalf("DeleteDividend() error: %v", err)
		}

		// Dividend deleted
		testutil.AssertRowCount(t, db, "dividend", 0)
		// Reinvestment transaction should also be deleted
		testutil.AssertRowCount(t, db, "transaction", 1) // only original buy remains
	})

	t.Run("triggers materialized invalidation", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		mock := testutil.NewMockMaterializedInvalidator(1)
		svc.SetMaterializedInvalidator(mock)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		err := svc.DeleteDividend(ctx, div.ID)
		if err != nil {
			t.Fatalf("DeleteDividend() error: %v", err)
		}

		if !mock.WaitForCall(2 * time.Second) {
			t.Fatal("expected invalidator call after delete")
		}
		calls := mock.Calls()
		if calls[0].PortfolioFundID != pf.ID {
			t.Errorf("expected portfolioFundID %q, got %q", pf.ID, calls[0].PortfolioFundID)
		}
	})
}

// =============================================================================
// DividendService.loadDividendPerPF (tested via exported methods)
// =============================================================================

func TestDividendService_LoadDividendPerPF_ViaCreateAndGet(t *testing.T) {
	t.Run("dividends are loadable after creation", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		_, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}

		// Verify via GetAllDividend
		all, err := svc.GetAllDividend()
		if err != nil {
			t.Fatalf("GetAllDividend() error: %v", err)
		}
		if len(all) != 1 {
			t.Errorf("expected 1 dividend, got %d", len(all))
		}
	})
}

// =============================================================================
// DividendService.updateReinvestmentTransaction (tested via UpdateDividend)
// =============================================================================
//
//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestDividendService_UpdateReinvestmentTransaction(t *testing.T) {
	t.Run("updates existing reinvestment transaction via UpdateDividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		// Create dividend with reinvestment (COMPLETED: 5*10=50 == 100*0.50=50)
		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:    pf.ID,
			RecordDate:         "2025-01-20",
			ExDividendDate:     "2025-01-18",
			DividendPerShare:   0.50,
			BuyOrderDate:       "2025-01-22",
			ReinvestmentShares: 5.0,
			ReinvestmentPrice:  10.0,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ReinvestmentStatus != "COMPLETED" {
			t.Fatalf("precondition: expected COMPLETED, got %q", div.ReinvestmentStatus)
		}
		if div.ReinvestmentTransactionID == "" {
			t.Fatal("precondition: expected reinvestment transaction ID")
		}

		origTxID := div.ReinvestmentTransactionID

		// Update reinvestment to partial: 3*10=30 != 50
		newShares := 3.0
		newPrice := 10.0
		newBuyDate := "2025-01-25"
		updated, err := svc.UpdateDividend(ctx, div.ID, request.UpdateDividendRequest{
			ReinvestmentShares: &newShares,
			ReinvestmentPrice:  &newPrice,
			BuyOrderDate:       &newBuyDate,
		})
		if err != nil {
			t.Fatalf("UpdateDividend() error: %v", err)
		}

		if updated.ReinvestmentStatus != "PARTIAL" {
			t.Errorf("expected PARTIAL after changing reinvestment shares, got %q", updated.ReinvestmentStatus)
		}
		if updated.ReinvestmentTransactionID != origTxID {
			t.Errorf("expected same reinvestment transaction ID %q, got %q", origTxID, updated.ReinvestmentTransactionID)
		}

		// Verify only 2 transactions exist (original buy + reinvestment, not a new one)
		testutil.AssertRowCount(t, db, "transaction", 2)
	})

	t.Run("updates reinvestment to COMPLETED when amounts match", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		// Create dividend with partial reinvestment: 2*10=20 != 100*0.50=50
		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:    pf.ID,
			RecordDate:         "2025-01-20",
			ExDividendDate:     "2025-01-18",
			DividendPerShare:   0.50,
			BuyOrderDate:       "2025-01-22",
			ReinvestmentShares: 2.0,
			ReinvestmentPrice:  10.0,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ReinvestmentStatus != "PARTIAL" {
			t.Fatalf("precondition: expected PARTIAL, got %q", div.ReinvestmentStatus)
		}

		// Update reinvestment to match: 5*10=50 == 100*0.50=50
		newShares := 5.0
		newPrice := 10.0
		updated, err := svc.UpdateDividend(ctx, div.ID, request.UpdateDividendRequest{
			ReinvestmentShares: &newShares,
			ReinvestmentPrice:  &newPrice,
		})
		if err != nil {
			t.Fatalf("UpdateDividend() error: %v", err)
		}

		if updated.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("expected COMPLETED after matching reinvestment, got %q", updated.ReinvestmentStatus)
		}
	})

	t.Run("creates new reinvestment when updating PENDING STOCK dividend with reinvestment info", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		// Create STOCK dividend without reinvestment -> PENDING
		div, err := svc.CreateDividend(ctx, request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err != nil {
			t.Fatalf("CreateDividend() error: %v", err)
		}
		if div.ReinvestmentStatus != "PENDING" {
			t.Fatalf("precondition: expected PENDING, got %q", div.ReinvestmentStatus)
		}

		// Only original buy transaction
		testutil.AssertRowCount(t, db, "transaction", 1)

		// Now add reinvestment info via update
		newShares := 5.0
		newPrice := 10.0
		newBuyDate := "2025-01-22"
		updated, err := svc.UpdateDividend(ctx, div.ID, request.UpdateDividendRequest{
			ReinvestmentShares: &newShares,
			ReinvestmentPrice:  &newPrice,
			BuyOrderDate:       &newBuyDate,
		})
		if err != nil {
			t.Fatalf("UpdateDividend() error: %v", err)
		}

		if updated.ReinvestmentTransactionID == "" {
			t.Error("expected reinvestment transaction ID to be set")
		}
		if updated.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("expected COMPLETED, got %q", updated.ReinvestmentStatus)
		}

		// Original buy + new reinvestment transaction
		testutil.AssertRowCount(t, db, "transaction", 2)
	})
}
