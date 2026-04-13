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

// --- GetAllDividend ---

func TestDividendRepository_GetAllDividend(t *testing.T) {
	t.Run("returns empty slice when no dividends", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDividendRepository(db)

		result, err := repo.GetAllDividend()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 dividends, got %d", len(result))
		}
	})

	t.Run("returns all dividends ordered by ex_dividend_date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)).
			Build(t, db)
		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		result, err := repo.GetAllDividend()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 dividends, got %d", len(result))
		}
		// Should be ordered ascending by ex_dividend_date
		if result[0].ExDividendDate.After(result[1].ExDividendDate) {
			t.Error("dividends not in ascending ex_dividend_date order")
		}
	})
}

// --- GetDividendPerPF ---

func TestDividendRepository_GetDividendPerPF(t *testing.T) {
	t.Run("returns empty map for empty pfIDs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDividendRepository(db)

		result, err := repo.GetDividendPerPF([]string{}, time.Now().AddDate(-1, 0, 0), time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("returns dividends within date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetDividendPerPF([]string{pf.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[pf.ID]) != 1 {
			t.Errorf("expected 1 dividend, got %d", len(result[pf.ID]))
		}
	})

	t.Run("excludes dividends outside date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetDividendPerPF([]string{pf.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[pf.ID]) != 0 {
			t.Errorf("expected 0 dividends, got %d", len(result[pf.ID]))
		}
	})

	t.Run("includes boundary dates", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(start).
			WithRecordDate(start.AddDate(0, 0, -3)).
			Build(t, db)
		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(end).
			WithRecordDate(end.AddDate(0, 0, -3)).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		result, err := repo.GetDividendPerPF([]string{pf.ID}, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[pf.ID]) != 2 {
			t.Errorf("expected 2 dividends on boundaries, got %d", len(result[pf.ID]))
		}
	})

	t.Run("groups by portfolio fund ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().Build(t, db)
		fund2 := testutil.NewFund().Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		exDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		recDate := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)

		testutil.NewDividend(fund1.ID, pf1.ID).WithExDividendDate(exDate).WithRecordDate(recDate).Build(t, db)
		testutil.NewDividend(fund1.ID, pf1.ID).WithExDividendDate(exDate).WithRecordDate(recDate).Build(t, db)
		testutil.NewDividend(fund2.ID, pf2.ID).WithExDividendDate(exDate).WithRecordDate(recDate).Build(t, db)

		repo := repository.NewDividendRepository(db)
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

		result, err := repo.GetDividendPerPF([]string{pf1.ID, pf2.ID}, start, end)
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
}

// --- GetDividendPerPortfolioFund ---

func TestDividendRepository_GetDividendPerPortfolioFund(t *testing.T) {
	t.Run("returns empty slice when both IDs empty", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDividendRepository(db)

		result, err := repo.GetDividendPerPortfolioFund("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d", len(result))
		}
	})

	t.Run("returns ErrPortfolioNotFound for nonexistent portfolio", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDividendRepository(db)

		_, err := repo.GetDividendPerPortfolioFund(testutil.MakeID(), "")
		if !errors.Is(err, apperrors.ErrPortfolioNotFound) {
			t.Errorf("expected ErrPortfolioNotFound, got %v", err)
		}
	})

	t.Run("returns ErrFundNotFound for nonexistent fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDividendRepository(db)

		_, err := repo.GetDividendPerPortfolioFund("", testutil.MakeID())
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound, got %v", err)
		}
	})

	t.Run("returns dividends filtered by portfolio ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("distributing").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		result, err := repo.GetDividendPerPortfolioFund(portfolio.ID, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 dividend, got %d", len(result))
		}
		if result[0].DividendType != "distributing" {
			t.Errorf("expected dividend type 'distributing', got '%s'", result[0].DividendType)
		}
	})

	t.Run("returns dividends filtered by fund ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		result, err := repo.GetDividendPerPortfolioFund("", fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 dividend, got %d", len(result))
		}
	})

	t.Run("returns empty slice when portfolio exists but has no dividends", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewDividendRepository(db)
		result, err := repo.GetDividendPerPortfolioFund(portfolio.ID, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 dividends, got %d", len(result))
		}
	})
}

// --- GetDividend ---

func TestDividendRepository_GetDividend(t *testing.T) {
	t.Run("returns dividend by ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)).
			WithSharesOwned(200).
			WithDividendPerShare(0.75).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		result, err := repo.GetDividend(div.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != div.ID {
			t.Errorf("expected ID %s, got %s", div.ID, result.ID)
		}
		if result.SharesOwned != 200 {
			t.Errorf("expected 200 shares, got %f", result.SharesOwned)
		}
		if result.DividendPerShare != 0.75 {
			t.Errorf("expected 0.75 per share, got %f", result.DividendPerShare)
		}
	})

	t.Run("returns ErrDividendNotFound for nonexistent ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDividendRepository(db)

		_, err := repo.GetDividend(testutil.MakeID())
		if !errors.Is(err, apperrors.ErrDividendNotFound) {
			t.Errorf("expected ErrDividendNotFound, got %v", err)
		}
	})
}

// --- InsertDividend ---

func TestDividendRepository_InsertDividend(t *testing.T) {
	t.Run("inserts a dividend successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewDividendRepository(db)
		div := &model.Dividend{
			ID:                 testutil.MakeID(),
			FundID:             fund.ID,
			PortfolioFundID:    pf.ID,
			RecordDate:         time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
			ExDividendDate:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			SharesOwned:        100,
			DividendPerShare:   0.50,
			TotalAmount:        50,
			ReinvestmentStatus: "pending",
			CreatedAt:          time.Now().UTC(),
		}

		err := repo.InsertDividend(context.Background(), div)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetDividend(div.ID)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if result.TotalAmount != 50 {
			t.Errorf("expected total 50, got %f", result.TotalAmount)
		}
	})

	t.Run("inserts dividend with nullable fields set", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create a transaction to reference as reinvestment
		txn := testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)).
			WithType("dividend").
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		div := &model.Dividend{
			ID:                        testutil.MakeID(),
			FundID:                    fund.ID,
			PortfolioFundID:           pf.ID,
			RecordDate:                time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
			ExDividendDate:            time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			SharesOwned:               100,
			DividendPerShare:          0.50,
			TotalAmount:               50,
			ReinvestmentStatus:        "completed",
			BuyOrderDate:              time.Date(2024, 6, 18, 0, 0, 0, 0, time.UTC),
			ReinvestmentTransactionID: txn.ID,
			CreatedAt:                 time.Now().UTC(),
		}

		err := repo.InsertDividend(context.Background(), div)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetDividend(div.ID)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if result.ReinvestmentTransactionID != txn.ID {
			t.Errorf("expected reinvestment tx ID %s, got %s", txn.ID, result.ReinvestmentTransactionID)
		}
	})
}

// --- UpdateDividend ---

func TestDividendRepository_UpdateDividend(t *testing.T) {
	t.Run("updates an existing dividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)).
			WithSharesOwned(100).
			Build(t, db)

		repo := repository.NewDividendRepository(db)

		updated := &model.Dividend{
			ID:                 div.ID,
			FundID:             fund.ID,
			PortfolioFundID:    pf.ID,
			RecordDate:         time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
			ExDividendDate:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			SharesOwned:        200,
			DividendPerShare:   1.0,
			TotalAmount:        200,
			ReinvestmentStatus: "completed",
			CreatedAt:          time.Now().UTC(),
		}

		err := repo.UpdateDividend(context.Background(), updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetDividend(div.ID)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if result.SharesOwned != 200 {
			t.Errorf("expected 200 shares, got %f", result.SharesOwned)
		}
		if result.ReinvestmentStatus != "completed" {
			t.Errorf("expected status 'completed', got '%s'", result.ReinvestmentStatus)
		}
	})

	t.Run("returns ErrDividendNotFound for nonexistent ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewDividendRepository(db)
		div := &model.Dividend{
			ID:                 testutil.MakeID(),
			FundID:             fund.ID,
			PortfolioFundID:    pf.ID,
			RecordDate:         time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
			ExDividendDate:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			SharesOwned:        100,
			DividendPerShare:   0.50,
			TotalAmount:        50,
			ReinvestmentStatus: "pending",
			CreatedAt:          time.Now().UTC(),
		}

		err := repo.UpdateDividend(context.Background(), div)
		if !errors.Is(err, apperrors.ErrDividendNotFound) {
			t.Errorf("expected ErrDividendNotFound, got %v", err)
		}
	})
}

// --- DeleteDividend ---

func TestDividendRepository_DeleteDividend(t *testing.T) {
	t.Run("deletes an existing dividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		div := testutil.NewDividend(fund.ID, pf.ID).
			WithExDividendDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
			WithRecordDate(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		repo := repository.NewDividendRepository(db)
		err := repo.DeleteDividend(context.Background(), div.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = repo.GetDividend(div.ID)
		if !errors.Is(err, apperrors.ErrDividendNotFound) {
			t.Errorf("expected ErrDividendNotFound after delete, got %v", err)
		}
	})

	t.Run("returns ErrDividendNotFound for nonexistent ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDividendRepository(db)

		err := repo.DeleteDividend(context.Background(), testutil.MakeID())
		if !errors.Is(err, apperrors.ErrDividendNotFound) {
			t.Errorf("expected ErrDividendNotFound, got %v", err)
		}
	})
}

// --- WithTx ---

func TestDividendRepository_WithTx(t *testing.T) {
	t.Run("commit makes insert visible", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewDividendRepository(db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		div := &model.Dividend{
			ID:                 testutil.MakeID(),
			FundID:             fund.ID,
			PortfolioFundID:    pf.ID,
			RecordDate:         time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
			ExDividendDate:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			SharesOwned:        100,
			DividendPerShare:   0.50,
			TotalAmount:        50,
			ReinvestmentStatus: "pending",
			CreatedAt:          time.Now().UTC(),
		}

		err = txRepo.InsertDividend(context.Background(), div)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		if err = tx.Commit(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		result, err := repo.GetDividend(div.ID)
		if err != nil {
			t.Fatalf("failed to get after commit: %v", err)
		}
		if result.ID != div.ID {
			t.Errorf("expected ID %s, got %s", div.ID, result.ID)
		}
	})

	t.Run("rollback hides insert", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		repo := repository.NewDividendRepository(db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		div := &model.Dividend{
			ID:                 testutil.MakeID(),
			FundID:             fund.ID,
			PortfolioFundID:    pf.ID,
			RecordDate:         time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
			ExDividendDate:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			SharesOwned:        100,
			DividendPerShare:   0.50,
			TotalAmount:        50,
			ReinvestmentStatus: "pending",
			CreatedAt:          time.Now().UTC(),
		}

		err = txRepo.InsertDividend(context.Background(), div)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err = tx.Rollback(); err != nil {
			t.Fatalf("rollback failed: %v", err)
		}

		_, err = repo.GetDividend(div.ID)
		if !errors.Is(err, apperrors.ErrDividendNotFound) {
			t.Errorf("expected ErrDividendNotFound after rollback, got %v", err)
		}
	})
}
