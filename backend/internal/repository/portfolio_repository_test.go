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

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestPortfolioRepository_GetPortfolios(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioRepository(db)

	t.Run("returns empty slice when no portfolios exist", func(t *testing.T) {
		result, err := repo.GetPortfolios(model.PortfolioFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty slice, got %d items", len(result))
		}
	})

	// Seed data for subsequent subtests
	active := testutil.NewPortfolio().WithName("Active").Build(t, db)
	archived := testutil.NewPortfolio().WithName("Archived").Archived().Build(t, db)
	excluded := testutil.NewPortfolio().WithName("Excluded").ExcludedFromOverview().Build(t, db)

	t.Run("excludes archived and excluded by default", func(t *testing.T) {
		result, err := repo.GetPortfolios(model.PortfolioFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 portfolio, got %d", len(result))
		}
		if result[0].ID != active.ID {
			t.Errorf("expected portfolio %s, got %s", active.ID, result[0].ID)
		}
	})

	t.Run("includes archived when filter is set", func(t *testing.T) {
		result, err := repo.GetPortfolios(model.PortfolioFilter{IncludeArchived: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 portfolios, got %d", len(result))
		}
		ids := map[string]bool{}
		for _, p := range result {
			ids[p.ID] = true
		}
		if !ids[active.ID] || !ids[archived.ID] {
			t.Errorf("expected active and archived portfolios, got ids: %v", ids)
		}
	})

	t.Run("includes excluded when filter is set", func(t *testing.T) {
		result, err := repo.GetPortfolios(model.PortfolioFilter{IncludeExcluded: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 portfolios, got %d", len(result))
		}
		ids := map[string]bool{}
		for _, p := range result {
			ids[p.ID] = true
		}
		if !ids[active.ID] || !ids[excluded.ID] {
			t.Errorf("expected active and excluded portfolios, got ids: %v", ids)
		}
	})

	t.Run("includes all when both filters set", func(t *testing.T) {
		result, err := repo.GetPortfolios(model.PortfolioFilter{IncludeArchived: true, IncludeExcluded: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 portfolios, got %d", len(result))
		}
	})
}

func TestPortfolioRepository_GetPortfolioOnID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioRepository(db)

	p := testutil.NewPortfolio().WithName("Lookup Test").WithDescription("desc").Build(t, db)

	t.Run("returns portfolio when found", func(t *testing.T) {
		result, err := repo.GetPortfolioOnID(p.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != p.ID {
			t.Errorf("expected ID %s, got %s", p.ID, result.ID)
		}
		if result.Name != p.Name {
			t.Errorf("expected Name %s, got %s", p.Name, result.Name)
		}
		if result.Description != p.Description {
			t.Errorf("expected Description %s, got %s", p.Description, result.Description)
		}
	})

	t.Run("returns ErrPortfolioNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetPortfolioOnID("non-existent-id")
		if !errors.Is(err, apperrors.ErrPortfolioNotFound) {
			t.Errorf("expected ErrPortfolioNotFound, got %v", err)
		}
	})
}

func TestPortfolioRepository_GetPortfoliosByFundID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioRepository(db)

	p1 := testutil.NewPortfolio().WithName("P1").Build(t, db)
	p2 := testutil.NewPortfolio().WithName("P2").Build(t, db)
	fund := testutil.NewFund().Build(t, db)
	fund2 := testutil.NewFund().Build(t, db)

	testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
	testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

	t.Run("returns portfolios that hold the fund", func(t *testing.T) {
		result, err := repo.GetPortfoliosByFundID(fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 portfolios, got %d", len(result))
		}
	})

	t.Run("returns empty slice when fund is not assigned", func(t *testing.T) {
		result, err := repo.GetPortfoliosByFundID(fund2.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 portfolios, got %d", len(result))
		}
	})

	t.Run("returns empty slice for non-existent fund ID", func(t *testing.T) {
		result, err := repo.GetPortfoliosByFundID("non-existent-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 portfolios, got %d", len(result))
		}
	})
}

func TestPortfolioRepository_InsertPortfolio(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioRepository(db)
	ctx := context.Background()

	t.Run("inserts a portfolio successfully", func(t *testing.T) {
		p := &model.Portfolio{
			ID:                  testutil.MakeID(),
			Name:                "New Portfolio",
			Description:         "New Desc",
			IsArchived:          false,
			ExcludeFromOverview: false,
		}
		err := repo.InsertPortfolio(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it was inserted
		result, err := repo.GetPortfolioOnID(p.ID)
		if err != nil {
			t.Fatalf("failed to retrieve inserted portfolio: %v", err)
		}
		if result.Name != "New Portfolio" {
			t.Errorf("expected name 'New Portfolio', got %s", result.Name)
		}
	})

	t.Run("fails on duplicate ID", func(t *testing.T) {
		p := &model.Portfolio{
			ID:   testutil.MakeID(),
			Name: "First",
		}
		err := repo.InsertPortfolio(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error on first insert: %v", err)
		}

		// Try inserting with the same ID
		p2 := &model.Portfolio{
			ID:   p.ID,
			Name: "Duplicate",
		}
		err = repo.InsertPortfolio(ctx, p2)
		if err == nil {
			t.Fatal("expected error on duplicate insert, got nil")
		}
	})
}

func TestPortfolioRepository_UpdatePortfolio(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioRepository(db)
	ctx := context.Background()

	p := testutil.NewPortfolio().WithName("Original").WithDescription("OrigDesc").Build(t, db)

	t.Run("updates an existing portfolio", func(t *testing.T) {
		updated := &model.Portfolio{
			ID:                  p.ID,
			Name:                "Updated",
			Description:         "Updated Desc",
			IsArchived:          true,
			ExcludeFromOverview: true,
		}
		err := repo.UpdatePortfolio(ctx, updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetPortfolioOnID(p.ID)
		if err != nil {
			t.Fatalf("failed to retrieve updated portfolio: %v", err)
		}
		if result.Name != "Updated" {
			t.Errorf("expected name 'Updated', got %s", result.Name)
		}
		if result.Description != "Updated Desc" {
			t.Errorf("expected description 'Updated Desc', got %s", result.Description)
		}
		if !result.IsArchived {
			t.Error("expected IsArchived to be true")
		}
		if !result.ExcludeFromOverview {
			t.Error("expected ExcludeFromOverview to be true")
		}
	})

	t.Run("returns ErrPortfolioNotFound for non-existent ID", func(t *testing.T) {
		err := repo.UpdatePortfolio(ctx, &model.Portfolio{ID: "non-existent-id", Name: "X"})
		if !errors.Is(err, apperrors.ErrPortfolioNotFound) {
			t.Errorf("expected ErrPortfolioNotFound, got %v", err)
		}
	})
}

func TestPortfolioRepository_DeletePortfolio(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioRepository(db)
	ctx := context.Background()

	p := testutil.NewPortfolio().Build(t, db)

	t.Run("deletes an existing portfolio", func(t *testing.T) {
		err := repo.DeletePortfolio(ctx, p.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = repo.GetPortfolioOnID(p.ID)
		if !errors.Is(err, apperrors.ErrPortfolioNotFound) {
			t.Errorf("expected ErrPortfolioNotFound after delete, got %v", err)
		}
	})

	t.Run("returns ErrPortfolioNotFound for non-existent ID", func(t *testing.T) {
		err := repo.DeletePortfolio(ctx, "non-existent-id")
		if !errors.Is(err, apperrors.ErrPortfolioNotFound) {
			t.Errorf("expected ErrPortfolioNotFound, got %v", err)
		}
	})
}

func TestPortfolioRepository_WithTx(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewPortfolioRepository(db)
	ctx := context.Background()

	t.Run("operations within a committed transaction persist", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)

		p := &model.Portfolio{
			ID:   testutil.MakeID(),
			Name: "TxPortfolio",
		}
		err = txRepo.InsertPortfolio(ctx, p)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		result, err := repo.GetPortfolioOnID(p.ID)
		if err != nil {
			t.Fatalf("expected portfolio to exist after commit: %v", err)
		}
		if result.Name != "TxPortfolio" {
			t.Errorf("expected name 'TxPortfolio', got %s", result.Name)
		}
	})

	t.Run("operations within a rolled-back transaction do not persist", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)

		p := &model.Portfolio{
			ID:   testutil.MakeID(),
			Name: "RollbackPortfolio",
		}
		err = txRepo.InsertPortfolio(ctx, p)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Rollback()
		if err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}

		_, err = repo.GetPortfolioOnID(p.ID)
		if !errors.Is(err, apperrors.ErrPortfolioNotFound) {
			t.Errorf("expected ErrPortfolioNotFound after rollback, got %v", err)
		}
	})
}
