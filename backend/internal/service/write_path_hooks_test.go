package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// =============================================================================
// TRANSACTION SERVICE — WRITE-PATH INVALIDATION HOOKS
// =============================================================================

// TestTransactionService_CreateTransaction_TriggersInvalidation verifies that
// creating a transaction triggers materialized view regeneration.
// Issue #35 Edge Case 1: Backdated transactions must invalidate the cache.
func TestTransactionService_CreateTransaction_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestTransactionService(t, db)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	_, err := svc.CreateTransaction(context.Background(), request.CreateTransactionRequest{
		PortfolioFundID: pf.ID,
		Date:            "2025-01-15",
		Type:            "buy",
		Shares:          100,
		CostPerShare:    10.0,
	})
	if err != nil {
		t.Fatalf("CreateTransaction() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after CreateTransaction, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	call := calls[0]
	expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !call.StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate %v, got %v", expectedDate, call.StartDate)
	}
	if call.PortfolioFundID != pf.ID {
		t.Errorf("Expected portfolioFundID %q, got %q", pf.ID, call.PortfolioFundID)
	}
}

// TestTransactionService_UpdateTransaction_TriggersInvalidation verifies that
// updating a transaction triggers regeneration from the transaction date.
func TestTransactionService_UpdateTransaction_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestTransactionService(t, db)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	txn := testutil.NewTransaction(pf.ID).
		WithDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
		WithShares(100).
		WithCostPerShare(10.0).
		Build(t, db)

	// Wire up mock after creating the transaction (so create doesn't trigger it)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	newShares := 150.0
	_, err := svc.UpdateTransaction(context.Background(), txn.ID, request.UpdateTransactionRequest{
		PortfolioFundID: &pf.ID,
		Shares:          &newShares,
	})
	if err != nil {
		t.Fatalf("UpdateTransaction() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after UpdateTransaction, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}
	if calls[0].PortfolioFundID != pf.ID {
		t.Errorf("Expected portfolioFundID %q, got %q", pf.ID, calls[0].PortfolioFundID)
	}
}

// TestTransactionService_DeleteTransaction_TriggersInvalidation verifies that
// deleting a transaction triggers regeneration.
func TestTransactionService_DeleteTransaction_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestTransactionService(t, db)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	txn := testutil.NewTransaction(pf.ID).
		WithDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
		WithShares(100).
		WithCostPerShare(10.0).
		Build(t, db)

	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	err := svc.DeleteTransaction(context.Background(), txn.ID)
	if err != nil {
		t.Fatalf("DeleteTransaction() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after DeleteTransaction, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}
	if calls[0].PortfolioFundID != pf.ID {
		t.Errorf("Expected portfolioFundID %q, got %q", pf.ID, calls[0].PortfolioFundID)
	}
}

// =============================================================================
// DIVIDEND SERVICE — WRITE-PATH INVALIDATION HOOKS
// =============================================================================

// TestDividendService_CreateDividend_TriggersInvalidation verifies that
// creating a dividend triggers materialized view regeneration.
// Issue #35 Edge Case 3: New dividends must invalidate the cache.
func TestDividendService_CreateDividend_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestDividendService(t, db)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	// Need a transaction so GetSharesOnDate returns shares
	testutil.NewTransaction(pf.ID).
		WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
		WithShares(100).
		WithCostPerShare(10.0).
		Build(t, db)

	_, err := svc.CreateDividend(context.Background(), request.CreateDividendRequest{
		PortfolioFundID:  pf.ID,
		RecordDate:       "2025-01-20",
		ExDividendDate:   "2025-01-18",
		DividendPerShare: 0.50,
	})
	if err != nil {
		t.Fatalf("CreateDividend() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after CreateDividend, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	call := calls[0]
	expectedDate := time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)
	if !call.StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate (ex-dividend date) %v, got %v", expectedDate, call.StartDate)
	}
	if call.PortfolioFundID != pf.ID {
		t.Errorf("Expected portfolioFundID %q, got %q", pf.ID, call.PortfolioFundID)
	}
}

// TestDividendService_UpdateDividend_UsesMinDate verifies that when the
// ex-dividend date changes, regeneration starts from the EARLIER of the
// old and new dates.
// Issue #35 Edge Case 5: Dividend date changes trigger regen from min(old, new).
func TestDividendService_UpdateDividend_UsesMinDate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestDividendService(t, db)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	testutil.NewTransaction(pf.ID).
		WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
		WithShares(100).
		WithCostPerShare(10.0).
		Build(t, db)

	// Create initial dividend with ex-dividend date Jan 18
	div := testutil.NewDividend(fund.ID, pf.ID).
		WithExDividendDate(time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)).
		WithRecordDate(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)).
		WithDividendPerShare(0.50).
		WithSharesOwned(100).
		Build(t, db)

	// Wire up mock
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	// Update ex-dividend date to Jan 15 (earlier than original Jan 18)
	newExDate := "2025-01-15"
	_, err := svc.UpdateDividend(context.Background(), div.ID, request.UpdateDividendRequest{
		ExDividendDate: &newExDate,
	})
	if err != nil {
		t.Fatalf("UpdateDividend() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after UpdateDividend, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	// Should regenerate from Jan 15 (the earlier date)
	expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !calls[0].StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate to be min(old, new) = %v, got %v", expectedDate, calls[0].StartDate)
	}
}

// TestDividendService_UpdateDividend_UsesOldDateWhenNewIsLater verifies that
// when the new ex-dividend date is LATER, we still regenerate from the OLD date.
func TestDividendService_UpdateDividend_UsesOldDateWhenNewIsLater(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestDividendService(t, db)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	testutil.NewTransaction(pf.ID).
		WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
		WithShares(100).
		WithCostPerShare(10.0).
		Build(t, db)

	// Create initial dividend with ex-dividend date Jan 15
	div := testutil.NewDividend(fund.ID, pf.ID).
		WithExDividendDate(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)).
		WithRecordDate(time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)).
		WithDividendPerShare(0.50).
		WithSharesOwned(100).
		Build(t, db)

	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	// Update ex-dividend date to Jan 20 (later than original Jan 15)
	newExDate := "2025-01-20"
	_, err := svc.UpdateDividend(context.Background(), div.ID, request.UpdateDividendRequest{
		ExDividendDate: &newExDate,
	})
	if err != nil {
		t.Fatalf("UpdateDividend() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after UpdateDividend, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	// Should regenerate from Jan 15 (the earlier/old date)
	expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !calls[0].StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate to be old date %v (earlier), got %v", expectedDate, calls[0].StartDate)
	}
}

// =============================================================================
// DEVELOPER SERVICE — WRITE-PATH INVALIDATION HOOKS
// =============================================================================

// TestDeveloperService_UpdateFundPrice_TriggersInvalidation verifies that
// manually setting a fund price triggers materialized view regeneration.
func TestDeveloperService_UpdateFundPrice_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestDeveloperService(t, db)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	fund := testutil.NewFund().Build(t, db)

	_, err := svc.UpdateFundPrice(context.Background(), request.SetFundPriceRequest{
		FundID: fund.ID,
		Date:   "2025-01-15",
		Price:  "12.50",
	})
	if err != nil {
		t.Fatalf("UpdateFundPrice() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after UpdateFundPrice, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !calls[0].StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate %v, got %v", expectedDate, calls[0].StartDate)
	}
	if calls[0].FundID != fund.ID {
		t.Errorf("Expected fundID %q, got %q", fund.ID, calls[0].FundID)
	}
}

// CSV IMPORT HOOKS
// =============================================================================

// TestDeveloperService_ImportTransactions_TriggersInvalidation verifies that
// CSV transaction import triggers regeneration from the earliest imported date.
// Issue #35 Edge Case 8.
func TestDeveloperService_ImportTransactions_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestDeveloperService(t, db)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	csv := []byte("date,type,shares,cost_per_share\n2025-01-20,buy,50,15.0\n2025-01-15,buy,100,10.0\n")

	count, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
	if err != nil {
		t.Fatalf("ImportTransactions() error: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 imported transactions, got %d", count)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after ImportTransactions, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	// Should regenerate from the EARLIEST date in the CSV (Jan 15, not Jan 20)
	expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !calls[0].StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate to be earliest imported date %v, got %v", expectedDate, calls[0].StartDate)
	}
	if calls[0].PortfolioFundID != pf.ID {
		t.Errorf("Expected portfolioFundID %q, got %q", pf.ID, calls[0].PortfolioFundID)
	}
}

// TestDeveloperService_ImportFundPrices_TriggersInvalidation verifies that
// CSV price import triggers regeneration from the earliest imported date.
// Issue #35 Edge Case 9.
func TestDeveloperService_ImportFundPrices_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestDeveloperService(t, db)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	fund := testutil.NewFund().Build(t, db)

	csv := []byte("date,price\n2025-01-20,12.50\n2025-01-10,11.00\n2025-01-15,11.75\n")

	count, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
	if err != nil {
		t.Fatalf("ImportFundPrices() error: %v", err)
	}
	if count != 3 {
		t.Fatalf("Expected 3 imported prices, got %d", count)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after ImportFundPrices, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	// Should regenerate from the EARLIEST date in the CSV (Jan 10)
	expectedDate := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	if !calls[0].StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate to be earliest imported date %v, got %v", expectedDate, calls[0].StartDate)
	}
	if calls[0].FundID != fund.ID {
		t.Errorf("Expected fundID %q, got %q", fund.ID, calls[0].FundID)
	}
}

// =============================================================================
// FUND SERVICE — WRITE-PATH INVALIDATION HOOKS
// =============================================================================

// TestFundService_UpdateHistoricalFundPrice_TriggersInvalidation verifies that
// backfilling historical prices triggers materialized view regeneration from
// the earliest backfilled date using the fundID.
// Issue #35 Edge Case 6: Historical price backfills must invalidate from the earliest date.
func TestFundService_UpdateHistoricalFundPrice_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create a mock Yahoo client that returns prices for specific dates.
	// UpdateHistoricalFundPrice computes missing dates between oldest transaction
	// and yesterday, then filters the Yahoo response to only insert those missing.
	mockYahoo := testutil.NewMockYahooClient()
	svc := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().WithSymbol("TEST").Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	// Transaction on Jan 10 — this sets the oldest date boundary
	testutil.NewTransaction(pf.ID).
		WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
		WithShares(100).
		WithCostPerShare(10.0).
		Build(t, db)

	// Pre-existing prices for Jan 10-11 — the gap starts Jan 12
	testutil.NewFundPrice(fund.ID).
		WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
		WithPrice(10.0).
		Build(t, db)
	testutil.NewFundPrice(fund.ID).
		WithDate(time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)).
		WithPrice(10.5).
		Build(t, db)

	// The mock Yahoo client returns 5 days of prices ending yesterday.
	// UpdateHistoricalFundPrice will find the gap and insert the missing ones.
	_, err := svc.UpdateHistoricalFundPrice(context.Background(), fund.ID)
	if err != nil {
		t.Fatalf("UpdateHistoricalFundPrice() error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after UpdateHistoricalFundPrice, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	if calls[0].FundID != fund.ID {
		t.Errorf("Expected fundID %q, got %q", fund.ID, calls[0].FundID)
	}
}

// =============================================================================
// TRANSACTION SERVICE — SELL WITH REALIZED GAINS
// =============================================================================

// TestTransactionService_SellTransaction_TriggersInvalidation verifies that a
// sell transaction (which creates a RealizedGainLoss record within the same DB
// transaction) still triggers materialized view regeneration after commit.
// Issue #35 Edge Case 7: Sell transactions with realized gains must invalidate.
func TestTransactionService_SellTransaction_TriggersInvalidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestTransactionService(t, db)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	// Create a buy first so there are shares to sell
	testutil.NewTransaction(pf.ID).
		WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
		WithShares(100).
		WithCostPerShare(10.0).
		Build(t, db)

	// Wire up mock after the buy (so only the sell triggers it)
	mock := testutil.NewMockMaterializedInvalidator(1)
	svc.SetMaterializedInvalidator(mock)

	sellDate := "2025-01-20"
	_, err := svc.CreateTransaction(context.Background(), request.CreateTransactionRequest{
		PortfolioFundID: pf.ID,
		Date:            sellDate,
		Type:            "sell",
		Shares:          50,
		CostPerShare:    15.0, // Selling at profit
	})
	if err != nil {
		t.Fatalf("CreateTransaction(sell) error: %v", err)
	}

	if !mock.WaitForCall(2 * time.Second) {
		t.Fatal("Expected materialized invalidator to be called after sell transaction, but it was not")
	}

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 invalidator call, got %d", len(calls))
	}

	expectedDate := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	if !calls[0].StartDate.Equal(expectedDate) {
		t.Errorf("Expected startDate %v, got %v", expectedDate, calls[0].StartDate)
	}
	if calls[0].PortfolioFundID != pf.ID {
		t.Errorf("Expected portfolioFundID %q, got %q", pf.ID, calls[0].PortfolioFundID)
	}

	// Verify the RealizedGainLoss record was also created (the whole point of Edge Case 7)
	testutil.AssertRowCount(t, db, "realized_gain_loss", 1)
}

// =============================================================================
// NIL GUARD — NO PANIC WITHOUT INVALIDATOR
// =============================================================================

// TestWritePathServices_NilInvalidator_NoPanic verifies that write operations
// work correctly when no materializedInvalidator is set (nil guard pattern).
func TestWritePathServices_NilInvalidator_NoPanic(t *testing.T) {
	t.Run("TransactionService without invalidator", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		// Intentionally NOT calling SetMaterializedInvalidator

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		_, err := svc.CreateTransaction(context.Background(), request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-01-15",
			Type:            "buy",
			Shares:          100,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() should not panic or error without invalidator: %v", err)
		}
	})

	t.Run("DividendService without invalidator", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDividendService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		_, err := svc.CreateDividend(context.Background(), request.CreateDividendRequest{
			PortfolioFundID:  pf.ID,
			RecordDate:       "2025-01-20",
			ExDividendDate:   "2025-01-18",
			DividendPerShare: 0.50,
		})
		if err != nil {
			t.Fatalf("CreateDividend() should not panic or error without invalidator: %v", err)
		}
	})

	t.Run("DeveloperService without invalidator", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		csv := []byte("date,type,shares,cost_per_share\n2025-01-15,buy,100,10.0\n")
		_, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
		if err != nil {
			t.Fatalf("ImportTransactions() should not panic or error without invalidator: %v", err)
		}
	})
}
