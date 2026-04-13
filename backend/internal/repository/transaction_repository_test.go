package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// --- GetTransactions ---

func TestTransactionRepository_GetTransactions(t *testing.T) {
	t.Run("returns empty map for empty pfIDs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		result, err := repo.GetTransactions([]string{}, time.Now().AddDate(-1, 0, 0), time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("returns transactions within date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		testutil.NewTransaction(pf.ID).WithDate(date).WithShares(50).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetTransactions([]string{pf.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[pf.ID]) != 1 {
			t.Fatalf("expected 1 transaction, got %d", len(result[pf.ID]))
		}
		if result[pf.ID][0].Shares != 50 {
			t.Errorf("expected 50 shares, got %f", result[pf.ID][0].Shares)
		}
	})

	t.Run("excludes transactions outside date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Transaction before range
		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		// Transaction after range
		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetTransactions([]string{pf.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[pf.ID]) != 0 {
			t.Errorf("expected 0 transactions, got %d", len(result[pf.ID]))
		}
	})

	t.Run("includes boundary dates", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		testutil.NewTransaction(pf.ID).WithDate(start).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(end).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransactions([]string{pf.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[pf.ID]) != 2 {
			t.Errorf("expected 2 transactions on boundaries, got %d", len(result[pf.ID]))
		}
	})

	t.Run("groups by portfolio fund ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().Build(t, db)
		fund2 := testutil.NewFund().Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		testutil.NewTransaction(pf1.ID).WithDate(date).Build(t, db)
		testutil.NewTransaction(pf1.ID).WithDate(date).Build(t, db)
		testutil.NewTransaction(pf2.ID).WithDate(date).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetTransactions([]string{pf1.ID, pf2.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[pf1.ID]) != 2 {
			t.Errorf("expected 2 for pf1, got %d", len(result[pf1.ID]))
		}
		if len(result[pf2.ID]) != 1 {
			t.Errorf("expected 1 for pf2, got %d", len(result[pf2.ID]))
		}
	})

	t.Run("returns empty map when no transactions match", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		result, err := repo.GetTransactions([]string{"nonexistent-id"}, time.Now().AddDate(-1, 0, 0), time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})
}

// --- GetOldestTransaction ---

func TestTransactionRepository_GetOldestTransaction(t *testing.T) {
	t.Run("returns zero time for empty pfIDs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		result := repo.GetOldestTransaction([]string{})
		if !result.IsZero() {
			t.Errorf("expected zero time, got %v", result)
		}
	})

	t.Run("returns oldest date across multiple transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		oldest := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
		testutil.NewTransaction(pf.ID).WithDate(oldest).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result := repo.GetOldestTransaction([]string{pf.ID})

		expected := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
		if !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("returns zero time for nonexistent pfIDs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		result := repo.GetOldestTransaction([]string{"nonexistent"})
		if !result.IsZero() {
			t.Errorf("expected zero time, got %v", result)
		}
	})

	t.Run("returns oldest across multiple portfolio funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().Build(t, db)
		fund2 := testutil.NewFund().Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		testutil.NewTransaction(pf1.ID).WithDate(time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf2.ID).WithDate(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result := repo.GetOldestTransaction([]string{pf1.ID, pf2.ID})

		expected := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		if !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})
}

// --- GetTransactionsPerPortfolio ---

func TestTransactionRepository_GetTransactionsPerPortfolio(t *testing.T) {
	t.Run("returns all transactions for a portfolio", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransactionsPerPortfolio(portfolio.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 transactions, got %d", len(result))
		}
	})

	t.Run("returns all transactions when portfolioID is empty", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		p1 := testutil.NewPortfolio().Build(t, db)
		p2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf1 := testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf1.ID).WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf2.ID).WithDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransactionsPerPortfolio("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 transactions, got %d", len(result))
		}
	})

	t.Run("returns empty slice for portfolio with no transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransactionsPerPortfolio(portfolio.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 transactions, got %d", len(result))
		}
	})

	t.Run("includes fund name in response", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithName("Apple Inc").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransactionsPerPortfolio(portfolio.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 transaction, got %d", len(result))
		}
		if result[0].FundName != "Apple Inc" {
			t.Errorf("expected fund name 'Apple Inc', got '%s'", result[0].FundName)
		}
	})
}

// --- GetTransaction ---

func TestTransactionRepository_GetTransaction(t *testing.T) {
	t.Run("returns transaction by ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txn := testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)).
			WithShares(100).
			WithCostPerShare(50.0).
			Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransaction(txn.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != txn.ID {
			t.Errorf("expected ID %s, got %s", txn.ID, result.ID)
		}
		if result.Shares != 100 {
			t.Errorf("expected 100 shares, got %f", result.Shares)
		}
	})

	t.Run("returns empty for empty ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		result, err := repo.GetTransaction("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != "" {
			t.Errorf("expected empty ID, got %s", result.ID)
		}
	})

	t.Run("returns ErrTransactionNotFound for nonexistent ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		_, err := repo.GetTransaction(testutil.MakeID())
		if !errors.Is(err, apperrors.ErrTransactionNotFound) {
			t.Errorf("expected ErrTransactionNotFound, got %v", err)
		}
	})
}

// --- GetTransactionByID ---

func TestTransactionRepository_GetTransactionByID(t *testing.T) {
	t.Run("returns basic transaction model", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txn := testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC)).
			WithType("sell").
			WithShares(25).
			WithCostPerShare(75.0).
			Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransactionByID(txn.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Type != "sell" {
			t.Errorf("expected type 'sell', got '%s'", result.Type)
		}
		if result.PortfolioFundID != pf.ID {
			t.Errorf("expected portfolio fund ID %s, got %s", pf.ID, result.PortfolioFundID)
		}
	})

	t.Run("returns ErrTransactionNotFound for nonexistent ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		_, err := repo.GetTransactionByID(testutil.MakeID())
		if !errors.Is(err, apperrors.ErrTransactionNotFound) {
			t.Errorf("expected ErrTransactionNotFound, got %v", err)
		}
	})
}

// --- InsertTransaction ---

func TestTransactionRepository_InsertTransaction(t *testing.T) {
	t.Run("inserts a transaction successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		txn := &model.Transaction{
			ID:              testutil.MakeID(),
			PortfolioFundID: pf.ID,
			Date:            time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			Type:            "buy",
			Shares:          200,
			CostPerShare:    15.50,
			CreatedAt:       time.Now().UTC(),
		}

		err := repo.InsertTransaction(context.Background(), txn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it was inserted
		result, err := repo.GetTransactionByID(txn.ID)
		if err != nil {
			t.Fatalf("failed to retrieve inserted transaction: %v", err)
		}
		if result.Shares != 200 {
			t.Errorf("expected 200 shares, got %f", result.Shares)
		}
	})

	t.Run("insert with zero shares", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		txn := &model.Transaction{
			ID:              testutil.MakeID(),
			PortfolioFundID: pf.ID,
			Date:            time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			Type:            "buy",
			Shares:          0,
			CostPerShare:    0,
			CreatedAt:       time.Now().UTC(),
		}

		err := repo.InsertTransaction(context.Background(), txn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetTransactionByID(txn.ID)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if result.Shares != 0 {
			t.Errorf("expected 0 shares, got %f", result.Shares)
		}
	})
}

// --- UpdateTransaction ---

func TestTransactionRepository_UpdateTransaction(t *testing.T) {
	t.Run("updates an existing transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txn := testutil.NewTransaction(pf.ID).
			WithShares(100).
			WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewTransactionRepository(db)

		updated := &model.Transaction{
			ID:              txn.ID,
			PortfolioFundID: pf.ID,
			Date:            time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			Type:            "sell",
			Shares:          50,
			CostPerShare:    20.0,
			CreatedAt:       time.Now().UTC(),
		}

		err := repo.UpdateTransaction(context.Background(), updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetTransactionByID(txn.ID)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if result.Shares != 50 {
			t.Errorf("expected 50 shares, got %f", result.Shares)
		}
		if result.Type != "sell" {
			t.Errorf("expected type 'sell', got '%s'", result.Type)
		}
	})

	t.Run("returns ErrTransactionNotFound for nonexistent ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		txn := &model.Transaction{
			ID:              testutil.MakeID(),
			PortfolioFundID: pf.ID,
			Date:            time.Now().UTC(),
			Type:            "buy",
			Shares:          10,
			CostPerShare:    5,
			CreatedAt:       time.Now().UTC(),
		}

		err := repo.UpdateTransaction(context.Background(), txn)
		if !errors.Is(err, apperrors.ErrTransactionNotFound) {
			t.Errorf("expected ErrTransactionNotFound, got %v", err)
		}
	})
}

// --- DeleteTransaction ---

func TestTransactionRepository_DeleteTransaction(t *testing.T) {
	t.Run("deletes an existing transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txn := testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewTransactionRepository(db)
		err := repo.DeleteTransaction(context.Background(), txn.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it's gone
		_, err = repo.GetTransactionByID(txn.ID)
		if !errors.Is(err, apperrors.ErrTransactionNotFound) {
			t.Errorf("expected ErrTransactionNotFound after delete, got %v", err)
		}
	})

	t.Run("returns ErrTransactionNotFound for nonexistent ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		err := repo.DeleteTransaction(context.Background(), testutil.MakeID())
		if !errors.Is(err, apperrors.ErrTransactionNotFound) {
			t.Errorf("expected ErrTransactionNotFound, got %v", err)
		}
	})
}

// --- GetSharesOnDate ---

func TestTransactionRepository_GetSharesOnDate(t *testing.T) {
	t.Run("returns 0 when no transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		shares, err := repo.GetSharesOnDate(pf.ID, time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shares != 0 {
			t.Errorf("expected 0 shares, got %f", shares)
		}
	})

	t.Run("returns ErrInvalidPortfolioID for empty ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		_, err := repo.GetSharesOnDate("", time.Now())
		if !errors.Is(err, apperrors.ErrInvalidPortfolioID) {
			t.Errorf("expected ErrInvalidPortfolioID, got %v", err)
		}
	})

	t.Run("sums buys correctly", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).WithType("buy").WithShares(100).
			WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf.ID).WithType("buy").WithShares(50).
			WithDate(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		shares, err := repo.GetSharesOnDate(pf.ID, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shares != 150 {
			t.Errorf("expected 150 shares, got %f", shares)
		}
	})

	t.Run("subtracts sells", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).WithType("buy").WithShares(100).
			WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf.ID).WithType("sell").WithShares(30).
			WithDate(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		shares, err := repo.GetSharesOnDate(pf.ID, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shares != 70 {
			t.Errorf("expected 70 shares, got %f", shares)
		}
	})

	t.Run("adds dividend type shares", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).WithType("buy").WithShares(100).
			WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf.ID).WithType("dividend").WithShares(5).
			WithDate(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		shares, err := repo.GetSharesOnDate(pf.ID, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shares != 105 {
			t.Errorf("expected 105 shares, got %f", shares)
		}
	})

	t.Run("only counts transactions up to given date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).WithType("buy").WithShares(100).
			WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf.ID).WithType("buy").WithShares(200).
			WithDate(time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		shares, err := repo.GetSharesOnDate(pf.ID, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shares != 100 {
			t.Errorf("expected 100 shares (only first buy), got %f", shares)
		}
	})
}

// --- GetTransactionsByPortfolioFundID ---

func TestTransactionRepository_GetTransactionsByPortfolioFundID(t *testing.T) {
	t.Run("returns ErrInvalidPortfolioID for empty ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		_, err := repo.GetTransactionsByPortfolioFundID("")
		if !errors.Is(err, apperrors.ErrInvalidPortfolioID) {
			t.Errorf("expected ErrInvalidPortfolioID, got %v", err)
		}
	})

	t.Run("returns all transactions ordered by date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)

		repo := repository.NewTransactionRepository(db)
		result, err := repo.GetTransactionsByPortfolioFundID(pf.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 transactions, got %d", len(result))
		}
		// Verify ordering
		for i := 1; i < len(result); i++ {
			if result[i].Date.Before(result[i-1].Date) {
				t.Errorf("transactions not in ascending date order at index %d", i)
			}
		}
	})

	t.Run("returns nil for nonexistent portfolio fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewTransactionRepository(db)

		result, err := repo.GetTransactionsByPortfolioFundID(testutil.MakeID())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

// --- WithTx ---

func TestTransactionRepository_WithTx(t *testing.T) {
	t.Run("operations within transaction are visible after commit", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewTransactionRepository(db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		txRepo := repo.WithTx(tx)
		txn := &model.Transaction{
			ID:              testutil.MakeID(),
			PortfolioFundID: pf.ID,
			Date:            time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			Type:            "buy",
			Shares:          100,
			CostPerShare:    10,
			CreatedAt:       time.Now().UTC(),
		}

		err = txRepo.InsertTransaction(context.Background(), txn)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Should be visible from normal repo
		result, err := repo.GetTransactionByID(txn.ID)
		if err != nil {
			t.Fatalf("failed to get after commit: %v", err)
		}
		if result.ID != txn.ID {
			t.Errorf("expected ID %s, got %s", txn.ID, result.ID)
		}
	})

	t.Run("operations within rolled-back transaction are not visible", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewTransactionRepository(db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		txRepo := repo.WithTx(tx)
		txn := &model.Transaction{
			ID:              testutil.MakeID(),
			PortfolioFundID: pf.ID,
			Date:            time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			Type:            "buy",
			Shares:          100,
			CostPerShare:    10,
			CreatedAt:       time.Now().UTC(),
		}

		err = txRepo.InsertTransaction(context.Background(), txn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Rollback()
		if err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}

		// Should NOT be visible
		_, err = repo.GetTransactionByID(txn.ID)
		if !errors.Is(err, apperrors.ErrTransactionNotFound) {
			t.Errorf("expected ErrTransactionNotFound after rollback, got %v", err)
		}
	})
}
