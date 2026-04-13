package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

func TestPortfolioFundRepository_GetPortfolioFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	t.Run("returns portfolio fund when found", func(t *testing.T) {
		result, err := repo.GetPortfolioFund(pf.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != pf.ID {
			t.Errorf("expected ID %s, got %s", pf.ID, result.ID)
		}
		if result.PortfolioID != portfolio.ID {
			t.Errorf("expected PortfolioID %s, got %s", portfolio.ID, result.PortfolioID)
		}
		if result.FundID != fund.ID {
			t.Errorf("expected FundID %s, got %s", fund.ID, result.FundID)
		}
	})

	t.Run("returns ErrPortfolioFundNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetPortfolioFund("non-existent-id")
		if !errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			t.Errorf("expected ErrPortfolioFundNotFound, got %v", err)
		}
	})

	t.Run("returns ErrInvalidPortfolioID for empty ID", func(t *testing.T) {
		_, err := repo.GetPortfolioFund("")
		if !errors.Is(err, apperrors.ErrInvalidPortfolioID) {
			t.Errorf("expected ErrInvalidPortfolioID, got %v", err)
		}
	})
}

func TestPortfolioFundRepository_GetPortfolioFundListing(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	portfolio := testutil.NewPortfolio().WithName("ListingPortfolio").Build(t, db)
	fund := testutil.NewFund().WithName("ListingFund").WithDividendType("CASH").Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	t.Run("returns listing with enriched metadata", func(t *testing.T) {
		result, err := repo.GetPortfolioFundListing(pf.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != pf.ID {
			t.Errorf("expected ID %s, got %s", pf.ID, result.ID)
		}
		if result.PortfolioName != "ListingPortfolio" {
			t.Errorf("expected PortfolioName 'ListingPortfolio', got %s", result.PortfolioName)
		}
		if result.FundName != "ListingFund" {
			t.Errorf("expected FundName 'ListingFund', got %s", result.FundName)
		}
		if result.DividendType != "CASH" {
			t.Errorf("expected DividendType 'CASH', got %s", result.DividendType)
		}
	})

	t.Run("returns error for non-existent ID", func(t *testing.T) {
		_, err := repo.GetPortfolioFundListing("non-existent-id")
		if !errors.Is(err, apperrors.ErrFailedToRetrievePortfolioFunds) {
			t.Errorf("expected ErrFailedToRetrievePortfolioFunds, got %v", err)
		}
	})
}

func TestPortfolioFundRepository_GetAllPortfolioFundListings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	t.Run("returns empty slice when no data", func(t *testing.T) {
		result, err := repo.GetAllPortfolioFundListings()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty slice, got %d items", len(result))
		}
	})

	activePortfolio := testutil.NewPortfolio().WithName("Active").Build(t, db)
	archivedPortfolio := testutil.NewPortfolio().WithName("Archived").Archived().Build(t, db)
	fund1 := testutil.NewFund().WithName("Fund1").Build(t, db)
	fund2 := testutil.NewFund().WithName("Fund2").Build(t, db)

	testutil.NewPortfolioFund(activePortfolio.ID, fund1.ID).Build(t, db)
	testutil.NewPortfolioFund(activePortfolio.ID, fund2.ID).Build(t, db)
	testutil.NewPortfolioFund(archivedPortfolio.ID, fund1.ID).Build(t, db)

	t.Run("excludes archived portfolios", func(t *testing.T) {
		result, err := repo.GetAllPortfolioFundListings()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 listings (active only), got %d", len(result))
		}
		for _, l := range result {
			if l.PortfolioID == archivedPortfolio.ID {
				t.Error("should not include archived portfolio listings")
			}
		}
	})

	t.Run("results are ordered by portfolio name then fund name", func(t *testing.T) {
		result, err := repo.GetAllPortfolioFundListings()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) < 2 {
			t.Fatalf("expected at least 2 results, got %d", len(result))
		}
		// Both belong to "Active" portfolio, so they should be ordered by fund name
		if result[0].FundName > result[1].FundName {
			t.Errorf("expected fund names in ascending order, got %s before %s", result[0].FundName, result[1].FundName)
		}
	})
}

func TestPortfolioFundRepository_GetPortfolioFunds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	p1 := testutil.NewPortfolio().Build(t, db)
	p2 := testutil.NewPortfolio().Build(t, db)
	fund1 := testutil.NewFund().WithInvestmentType("ETF").WithDividendType("STOCK").Build(t, db)
	fund2 := testutil.NewFund().WithInvestmentType("BOND").WithDividendType("CASH").Build(t, db)

	testutil.NewPortfolioFund(p1.ID, fund1.ID).Build(t, db)
	testutil.NewPortfolioFund(p1.ID, fund2.ID).Build(t, db)
	testutil.NewPortfolioFund(p2.ID, fund1.ID).Build(t, db)

	t.Run("returns funds for a specific portfolio", func(t *testing.T) {
		result, err := repo.GetPortfolioFunds(p1.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 funds for p1, got %d", len(result))
		}
	})

	t.Run("returns all portfolio-fund relationships when portfolio ID is empty", func(t *testing.T) {
		result, err := repo.GetPortfolioFunds("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 total relationships, got %d", len(result))
		}
	})

	t.Run("returns empty slice for portfolio with no funds", func(t *testing.T) {
		emptyPortfolio := testutil.NewPortfolio().Build(t, db)
		result, err := repo.GetPortfolioFunds(emptyPortfolio.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 funds, got %d", len(result))
		}
	})

	t.Run("returns correct fund metadata", func(t *testing.T) {
		result, err := repo.GetPortfolioFunds(p1.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := false
		for _, r := range result {
			if r.FundID == fund1.ID {
				found = true
				if r.InvestmentType != "ETF" {
					t.Errorf("expected InvestmentType 'ETF', got %s", r.InvestmentType)
				}
				if r.DividendType != "STOCK" {
					t.Errorf("expected DividendType 'STOCK', got %s", r.DividendType)
				}
			}
		}
		if !found {
			t.Error("fund1 not found in results")
		}
	})
}

func TestPortfolioFundRepository_GetPortfolioFundsbyFundID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	p1 := testutil.NewPortfolio().Build(t, db)
	p2 := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	fund2 := testutil.NewFund().Build(t, db)

	pf1 := testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
	pf2 := testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

	t.Run("returns all portfolio_fund records for a fund", func(t *testing.T) {
		result, err := repo.GetPortfolioFundsbyFundID(fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 records, got %d", len(result))
		}
		ids := map[string]bool{}
		for _, r := range result {
			ids[r.ID] = true
		}
		if !ids[pf1.ID] || !ids[pf2.ID] {
			t.Errorf("expected pf1 and pf2, got %v", ids)
		}
	})

	t.Run("returns ErrPortfolioFundNotFound when fund not assigned", func(t *testing.T) {
		_, err := repo.GetPortfolioFundsbyFundID(fund2.ID)
		if !errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			t.Errorf("expected ErrPortfolioFundNotFound, got %v", err)
		}
	})

	t.Run("returns ErrInvalidFundID for empty fund ID", func(t *testing.T) {
		_, err := repo.GetPortfolioFundsbyFundID("")
		if !errors.Is(err, apperrors.ErrInvalidFundID) {
			t.Errorf("expected ErrInvalidFundID, got %v", err)
		}
	})
}

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestPortfolioFundRepository_GetPortfolioFundsOnPortfolioID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	t.Run("returns nil for all values when portfolios is empty", func(t *testing.T) {
		fundsByPortfolio, pfToPortfolio, pfToFund, pfIDs, fundIDs, err := repo.GetPortfolioFundsOnPortfolioID(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fundsByPortfolio != nil || pfToPortfolio != nil || pfToFund != nil || pfIDs != nil || fundIDs != nil {
			t.Error("expected all nil values for empty portfolios input")
		}
	})

	p1 := testutil.NewPortfolio().Build(t, db)
	p2 := testutil.NewPortfolio().Build(t, db)
	fund1 := testutil.NewFund().WithSymbol("AAPL.NASDAQ").Build(t, db)
	fund2 := testutil.NewFund().WithSymbol("GOOG.NASDAQ").Build(t, db)

	pf1 := testutil.NewPortfolioFund(p1.ID, fund1.ID).Build(t, db)
	pf2 := testutil.NewPortfolioFund(p1.ID, fund2.ID).Build(t, db)
	pf3 := testutil.NewPortfolioFund(p2.ID, fund1.ID).Build(t, db)

	t.Run("returns correct lookup structures", func(t *testing.T) {
		fundsByPortfolio, pfToPortfolio, pfToFund, pfIDs, fundIDs, err := repo.GetPortfolioFundsOnPortfolioID([]model.Portfolio{p1, p2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check fundsByPortfolio
		if len(fundsByPortfolio[p1.ID]) != 2 {
			t.Errorf("expected 2 funds for p1, got %d", len(fundsByPortfolio[p1.ID]))
		}
		if len(fundsByPortfolio[p2.ID]) != 1 {
			t.Errorf("expected 1 fund for p2, got %d", len(fundsByPortfolio[p2.ID]))
		}

		// Check pfToPortfolio map
		if pfToPortfolio[pf1.ID] != p1.ID {
			t.Errorf("expected pf1 -> p1, got %s", pfToPortfolio[pf1.ID])
		}
		if pfToPortfolio[pf3.ID] != p2.ID {
			t.Errorf("expected pf3 -> p2, got %s", pfToPortfolio[pf3.ID])
		}

		// Check pfToFund map
		if pfToFund[pf1.ID] != fund1.ID {
			t.Errorf("expected pf1 -> fund1, got %s", pfToFund[pf1.ID])
		}
		if pfToFund[pf2.ID] != fund2.ID {
			t.Errorf("expected pf2 -> fund2, got %s", pfToFund[pf2.ID])
		}

		// Check slices
		if len(pfIDs) != 3 {
			t.Errorf("expected 3 pfIDs, got %d", len(pfIDs))
		}
		if len(fundIDs) != 3 {
			t.Errorf("expected 3 fundIDs, got %d", len(fundIDs))
		}
	})

	t.Run("returns empty maps for portfolio with no funds", func(t *testing.T) {
		emptyPortfolio := testutil.NewPortfolio().Build(t, db)
		fundsByPortfolio, _, _, pfIDs, fundIDs, err := repo.GetPortfolioFundsOnPortfolioID([]model.Portfolio{emptyPortfolio})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fundsByPortfolio) != 0 {
			t.Errorf("expected empty fundsByPortfolio map, got %d entries", len(fundsByPortfolio))
		}
		if len(pfIDs) != 0 {
			t.Errorf("expected 0 pfIDs, got %d", len(pfIDs))
		}
		if len(fundIDs) != 0 {
			t.Errorf("expected 0 fundIDs, got %d", len(fundIDs))
		}
	})
}

func TestPortfolioFundRepository_GetPortfolioFundByPortfolioAndFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	t.Run("returns record when both IDs match", func(t *testing.T) {
		result, err := repo.GetPortfolioFundByPortfolioAndFund(portfolio.ID, fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != pf.ID {
			t.Errorf("expected ID %s, got %s", pf.ID, result.ID)
		}
	})

	t.Run("returns ErrPortfolioFundNotFound when portfolio does not hold the fund", func(t *testing.T) {
		otherFund := testutil.NewFund().Build(t, db)
		_, err := repo.GetPortfolioFundByPortfolioAndFund(portfolio.ID, otherFund.ID)
		if !errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			t.Errorf("expected ErrPortfolioFundNotFound, got %v", err)
		}
	})
}

func TestPortfolioFundRepository_CheckUsage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)

	t.Run("returns nil when fund is not assigned to any portfolio", func(t *testing.T) {
		unusedFund := testutil.NewFund().Build(t, db)
		result, err := repo.CheckUsage(unusedFund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	portfolio := testutil.NewPortfolio().WithName("UsagePortfolio").Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	t.Run("returns usage with zero transactions", func(t *testing.T) {
		result, err := repo.CheckUsage(fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if result[0].Name != "UsagePortfolio" {
			t.Errorf("expected portfolio name 'UsagePortfolio', got %s", result[0].Name)
		}
		if result[0].TransactionCount != 0 {
			t.Errorf("expected 0 transactions, got %d", result[0].TransactionCount)
		}
	})

	// Add transactions
	testutil.NewTransaction(pf.ID).Build(t, db)
	testutil.NewTransaction(pf.ID).Build(t, db)

	t.Run("returns correct transaction count", func(t *testing.T) {
		result, err := repo.CheckUsage(fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if result[0].TransactionCount != 2 {
			t.Errorf("expected 2 transactions, got %d", result[0].TransactionCount)
		}
	})
}

func TestPortfolioFundRepository_InsertPortfolioFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)
	ctx := context.Background()

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)

	t.Run("inserts a portfolio-fund relationship", func(t *testing.T) {
		err := repo.InsertPortfolioFund(ctx, portfolio.ID, fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it exists
		result, err := repo.GetPortfolioFundByPortfolioAndFund(portfolio.ID, fund.ID)
		if err != nil {
			t.Fatalf("failed to retrieve inserted pf: %v", err)
		}
		if result.PortfolioID != portfolio.ID {
			t.Errorf("expected PortfolioID %s, got %s", portfolio.ID, result.PortfolioID)
		}
		if result.FundID != fund.ID {
			t.Errorf("expected FundID %s, got %s", fund.ID, result.FundID)
		}
	})

	t.Run("fails with invalid foreign key for portfolio", func(t *testing.T) {
		err := repo.InsertPortfolioFund(ctx, "non-existent-portfolio", fund.ID)
		if err == nil {
			t.Fatal("expected foreign key error, got nil")
		}
	})

	t.Run("fails with invalid foreign key for fund", func(t *testing.T) {
		err := repo.InsertPortfolioFund(ctx, portfolio.ID, "non-existent-fund")
		if err == nil {
			t.Fatal("expected foreign key error, got nil")
		}
	})
}

func TestPortfolioFundRepository_DeletePortfolioFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)
	ctx := context.Background()

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

	t.Run("deletes an existing portfolio-fund relationship", func(t *testing.T) {
		err := repo.DeletePortfolioFund(ctx, pf.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = repo.GetPortfolioFund(pf.ID)
		if !errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			t.Errorf("expected ErrPortfolioFundNotFound after delete, got %v", err)
		}
	})

	t.Run("returns ErrPortfolioFundNotFound for non-existent ID", func(t *testing.T) {
		err := repo.DeletePortfolioFund(ctx, "non-existent-id")
		if !errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			t.Errorf("expected ErrPortfolioFundNotFound, got %v", err)
		}
	})
}

func TestPortfolioFundRepository_WithTx(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioFundRepository(db)
	ctx := context.Background()

	portfolio := testutil.NewPortfolio().Build(t, db)
	fund := testutil.NewFund().Build(t, db)

	t.Run("committed transaction persists insert", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		err = txRepo.InsertPortfolioFund(ctx, portfolio.ID, fund.ID)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		result, err := repo.GetPortfolioFundByPortfolioAndFund(portfolio.ID, fund.ID)
		if err != nil {
			t.Fatalf("expected pf to exist after commit: %v", err)
		}
		if result.PortfolioID != portfolio.ID {
			t.Errorf("expected PortfolioID %s, got %s", portfolio.ID, result.PortfolioID)
		}
	})

	t.Run("rolled-back transaction does not persist insert", func(t *testing.T) {
		fund2 := testutil.NewFund().Build(t, db)

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		err = txRepo.InsertPortfolioFund(ctx, portfolio.ID, fund2.ID)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Rollback()
		if err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}

		_, err = repo.GetPortfolioFundByPortfolioAndFund(portfolio.ID, fund2.ID)
		if !errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			t.Errorf("expected ErrPortfolioFundNotFound after rollback, got %v", err)
		}
	})
}
