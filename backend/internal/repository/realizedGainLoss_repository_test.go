package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// --- GetRealizedGainLossByPortfolio ---

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestRealizedGainLossRepository_GetRealizedGainLossByPortfolio(t *testing.T) {
	t.Run("returns empty map for empty portfolio slice", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewRealizedGainLossRepository(db)

		result, err := repo.GetRealizedGainLossByPortfolio([]string{}, time.Now().AddDate(-1, 0, 0), time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("returns records within date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txn := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, txn.ID).
			WithDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithShares(30).
			WithCostBasis(300).
			WithSaleProceeds(450).
			Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[portfolio.ID]) != 1 {
			t.Fatalf("expected 1 record, got %d", len(result[portfolio.ID]))
		}
		r := result[portfolio.ID][0]
		if r.SharesSold != 30 {
			t.Errorf("expected 30 shares sold, got %f", r.SharesSold)
		}
		if r.RealizedGainLoss != 150 {
			t.Errorf("expected gain of 150, got %f", r.RealizedGainLoss)
		}
	})

	t.Run("excludes records outside date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txn := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, txn.ID).
			WithDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[portfolio.ID]) != 0 {
			t.Errorf("expected 0 records, got %d", len(result[portfolio.ID]))
		}
	})

	t.Run("includes boundary dates", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		txn1 := testutil.NewTransaction(pf.ID).WithType("sell").WithDate(start).Build(t, db)
		txn2 := testutil.NewTransaction(pf.ID).WithType("sell").WithDate(end).Build(t, db)

		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, txn1.ID).WithDate(start).Build(t, db)
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, txn2.ID).WithDate(end).Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[portfolio.ID]) != 2 {
			t.Errorf("expected 2 records on boundaries, got %d", len(result[portfolio.ID]))
		}
	})

	t.Run("groups by portfolio ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		p1 := testutil.NewPortfolio().Build(t, db)
		p2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf1 := testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

		txn1 := testutil.NewTransaction(pf1.ID).WithType("sell").WithDate(date).Build(t, db)
		txn2 := testutil.NewTransaction(pf1.ID).WithType("sell").WithDate(date).Build(t, db)
		txn3 := testutil.NewTransaction(pf2.ID).WithType("sell").WithDate(date).Build(t, db)

		testutil.NewRealizedGainLoss(p1.ID, fund.ID, txn1.ID).WithDate(date).Build(t, db)
		testutil.NewRealizedGainLoss(p1.ID, fund.ID, txn2.ID).WithDate(date).Build(t, db)
		testutil.NewRealizedGainLoss(p2.ID, fund.ID, txn3.ID).WithDate(date).Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetRealizedGainLossByPortfolio([]string{p1.ID, p2.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[p1.ID]) != 2 {
			t.Errorf("expected 2 for p1, got %d", len(result[p1.ID]))
		}
		if len(result[p2.ID]) != 1 {
			t.Errorf("expected 1 for p2, got %d", len(result[p2.ID]))
		}
	})

	t.Run("returns empty map for nonexistent portfolios", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewRealizedGainLossRepository(db)

		result, err := repo.GetRealizedGainLossByPortfolio([]string{"nonexistent"}, time.Now().AddDate(-1, 0, 0), time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})
}

// --- InsertRealizedGainLoss ---

func TestRealizedGainLossRepository_InsertRealizedGainLoss(t *testing.T) {
	t.Run("inserts a record successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithDate(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		rgl := &model.RealizedGainLoss{
			ID:               testutil.MakeID(),
			PortfolioID:      portfolio.ID,
			FundID:           fund.ID,
			TransactionID:    txn.ID,
			TransactionDate:  time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			SharesSold:       50,
			CostBasis:        500,
			SaleProceeds:     750,
			RealizedGainLoss: 250,
			CreatedAt:        time.Now().UTC(),
		}

		err := repo.InsertRealizedGainLoss(context.Background(), rgl)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify by querying
		start := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, start, end)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if len(result[portfolio.ID]) != 1 {
			t.Fatalf("expected 1 record, got %d", len(result[portfolio.ID]))
		}
		if result[portfolio.ID][0].RealizedGainLoss != 250 {
			t.Errorf("expected gain 250, got %f", result[portfolio.ID][0].RealizedGainLoss)
		}
	})

	t.Run("inserts record with zero gain (break even)", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithDate(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		rgl := &model.RealizedGainLoss{
			ID:               testutil.MakeID(),
			PortfolioID:      portfolio.ID,
			FundID:           fund.ID,
			TransactionID:    txn.ID,
			TransactionDate:  time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			SharesSold:       10,
			CostBasis:        100,
			SaleProceeds:     100,
			RealizedGainLoss: 0,
			CreatedAt:        time.Now().UTC(),
		}

		err := repo.InsertRealizedGainLoss(context.Background(), rgl)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("inserts record with negative gain (loss)", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithDate(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		rgl := &model.RealizedGainLoss{
			ID:               testutil.MakeID(),
			PortfolioID:      portfolio.ID,
			FundID:           fund.ID,
			TransactionID:    txn.ID,
			TransactionDate:  time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			SharesSold:       10,
			CostBasis:        200,
			SaleProceeds:     100,
			RealizedGainLoss: -100,
			CreatedAt:        time.Now().UTC(),
		}

		err := repo.InsertRealizedGainLoss(context.Background(), rgl)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		start := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, start, start)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if result[portfolio.ID][0].RealizedGainLoss != -100 {
			t.Errorf("expected loss of -100, got %f", result[portfolio.ID][0].RealizedGainLoss)
		}
	})
}

// --- DeleteRealizedGainLossByTransactionID ---

func TestRealizedGainLossRepository_DeleteRealizedGainLossByTransactionID(t *testing.T) {
	t.Run("deletes existing record", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		txn := testutil.NewTransaction(pf.ID).WithType("sell").WithDate(date).Build(t, db)
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, txn.ID).WithDate(date).Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)
		err := repo.DeleteRealizedGainLossByTransactionID(context.Background(), txn.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deleted
		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, date, date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[portfolio.ID]) != 0 {
			t.Errorf("expected 0 records after delete, got %d", len(result[portfolio.ID]))
		}
	})

	t.Run("does not error when no record exists (idempotent)", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewRealizedGainLossRepository(db)

		err := repo.DeleteRealizedGainLossByTransactionID(context.Background(), testutil.MakeID())
		if err != nil {
			t.Errorf("expected nil for nonexistent ID (idempotent), got %v", err)
		}
	})
}

// --- WithTx ---

func TestRealizedGainLossRepository_WithTx(t *testing.T) {
	t.Run("commit makes insert visible", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithDate(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		rgl := &model.RealizedGainLoss{
			ID:               testutil.MakeID(),
			PortfolioID:      portfolio.ID,
			FundID:           fund.ID,
			TransactionID:    txn.ID,
			TransactionDate:  time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			SharesSold:       25,
			CostBasis:        250,
			SaleProceeds:     375,
			RealizedGainLoss: 125,
			CreatedAt:        time.Now().UTC(),
		}

		err = txRepo.InsertRealizedGainLoss(context.Background(), rgl)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		if err = tx.Commit(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		date := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, date, date)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if len(result[portfolio.ID]) != 1 {
			t.Errorf("expected 1 record after commit, got %d", len(result[portfolio.ID]))
		}
	})

	t.Run("rollback hides insert", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithDate(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewRealizedGainLossRepository(db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		rgl := &model.RealizedGainLoss{
			ID:               testutil.MakeID(),
			PortfolioID:      portfolio.ID,
			FundID:           fund.ID,
			TransactionID:    txn.ID,
			TransactionDate:  time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			SharesSold:       25,
			CostBasis:        250,
			SaleProceeds:     375,
			RealizedGainLoss: 125,
			CreatedAt:        time.Now().UTC(),
		}

		err = txRepo.InsertRealizedGainLoss(context.Background(), rgl)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err = tx.Rollback(); err != nil {
			t.Fatalf("rollback failed: %v", err)
		}

		date := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
		result, err := repo.GetRealizedGainLossByPortfolio([]string{portfolio.ID}, date, date)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if len(result[portfolio.ID]) != 0 {
			t.Errorf("expected 0 records after rollback, got %d", len(result[portfolio.ID]))
		}
	})
}
