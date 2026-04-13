package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// ---------------------------------------------------------------------------
// GetIbkrConfig
// ---------------------------------------------------------------------------

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestIbkrRepository_GetIbkrConfig(t *testing.T) {
	t.Run("not found returns sentinel error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		cfg, err := repo.GetIbkrConfig()
		if !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			t.Fatalf("expected ErrIbkrConfigNotFound, got %v", err)
		}
		if cfg.Configured {
			t.Error("expected Configured=false for empty result")
		}
	})

	t.Run("found with minimal fields", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		cfg := &model.IbkrConfig{
			ID:                testutil.MakeID(),
			FlexToken:         "tok123",
			FlexQueryID:       "q1",
			AutoImportEnabled: false,
			Enabled:           true,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		err := repo.UpdateIbkrConfig(ctx, true, cfg)
		if err != nil {
			t.Fatalf("UpdateIbkrConfig: %v", err)
		}

		got, err := repo.GetIbkrConfig()
		if err != nil {
			t.Fatalf("GetIbkrConfig: %v", err)
		}
		if !got.Configured {
			t.Error("expected Configured=true")
		}
		if got.FlexToken != "tok123" {
			t.Errorf("expected FlexToken=tok123, got %s", got.FlexToken)
		}
		if got.FlexQueryID != "q1" {
			t.Errorf("expected FlexQueryID=q1, got %s", got.FlexQueryID)
		}
		if got.TokenExpiresAt != nil {
			t.Error("expected TokenExpiresAt to be nil")
		}
		if got.LastImportDate != nil {
			t.Error("expected LastImportDate to be nil")
		}
	})

	t.Run("found with all nullable fields", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		expires := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
		lastImport := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

		cfg := &model.IbkrConfig{
			ID:                       testutil.MakeID(),
			FlexToken:                "tok456",
			FlexQueryID:              "q2",
			TokenExpiresAt:           &expires,
			LastImportDate:           &lastImport,
			AutoImportEnabled:        true,
			Enabled:                  true,
			DefaultAllocationEnabled: true,
			DefaultAllocations: []model.Allocation{
				{PortfolioID: "p1", Percentage: 50},
				{PortfolioID: "p2", Percentage: 50},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := repo.UpdateIbkrConfig(ctx, true, cfg); err != nil {
			t.Fatalf("UpdateIbkrConfig: %v", err)
		}

		got, err := repo.GetIbkrConfig()
		if err != nil {
			t.Fatalf("GetIbkrConfig: %v", err)
		}
		// Known bug: UpdateIbkrConfig doesn't set NullString.Valid, so dates are stored as NULL.
		// This test asserts the current (broken) behavior. When the bug is fixed, update this test.
		if got.TokenExpiresAt != nil {
			t.Error("TokenExpiresAt is now set — the NullString.Valid bug may be fixed; update this test")
		}
		if got.LastImportDate != nil {
			t.Error("LastImportDate is now set — the NullString.Valid bug may be fixed; update this test")
		}
		if len(got.DefaultAllocations) != 2 {
			t.Fatalf("expected 2 allocations, got %d", len(got.DefaultAllocations))
		}
		if got.DefaultAllocations[0].PortfolioID != "p1" {
			t.Errorf("expected first allocation portfolioID=p1, got %s", got.DefaultAllocations[0].PortfolioID)
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateIbkrConfig
// ---------------------------------------------------------------------------

func TestIbkrRepository_UpdateIbkrConfig(t *testing.T) {
	t.Run("create new config", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		cfg := &model.IbkrConfig{
			ID:          testutil.MakeID(),
			FlexToken:   "create-tok",
			FlexQueryID: "cq1",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err := repo.UpdateIbkrConfig(ctx, false, cfg)
		if err != nil {
			t.Fatalf("UpdateIbkrConfig: %v", err)
		}

		got, err := repo.GetIbkrConfig()
		if err != nil {
			t.Fatalf("GetIbkrConfig: %v", err)
		}
		if got.FlexToken != "create-tok" {
			t.Errorf("expected FlexToken=create-tok, got %s", got.FlexToken)
		}
	})

	t.Run("overwrite existing config", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)

		// Create initial
		cfg1 := &model.IbkrConfig{
			ID:          testutil.MakeID(),
			FlexToken:   "old-tok",
			FlexQueryID: "old-q",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.UpdateIbkrConfig(ctx, true, cfg1); err != nil {
			t.Fatalf("first UpdateIbkrConfig: %v", err)
		}

		// Overwrite with new config
		cfg2 := &model.IbkrConfig{
			ID:          testutil.MakeID(),
			FlexToken:   "new-tok",
			FlexQueryID: "new-q",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.UpdateIbkrConfig(ctx, true, cfg2); err != nil {
			t.Fatalf("second UpdateIbkrConfig: %v", err)
		}

		got, err := repo.GetIbkrConfig()
		if err != nil {
			t.Fatalf("GetIbkrConfig: %v", err)
		}
		if got.FlexToken != "new-tok" {
			t.Errorf("expected FlexToken=new-tok, got %s", got.FlexToken)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteIbkrConfig
// ---------------------------------------------------------------------------

func TestIbkrRepository_DeleteIbkrConfig(t *testing.T) {
	t.Run("delete existing config", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		cfg := &model.IbkrConfig{
			ID:          testutil.MakeID(),
			FlexToken:   "del-tok",
			FlexQueryID: "dq",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.UpdateIbkrConfig(ctx, true, cfg); err != nil {
			t.Fatalf("UpdateIbkrConfig: %v", err)
		}

		if err := repo.DeleteIbkrConfig(ctx); err != nil {
			t.Fatalf("DeleteIbkrConfig: %v", err)
		}

		_, err := repo.GetIbkrConfig()
		if !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			t.Fatalf("expected ErrIbkrConfigNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent config returns sentinel", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		err := repo.DeleteIbkrConfig(ctx)
		if !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			t.Fatalf("expected ErrIbkrConfigNotFound, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// GetInbox
// ---------------------------------------------------------------------------

func TestIbkrRepository_GetInbox(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		txns, err := repo.GetInbox("pending", "")
		if err != nil {
			t.Fatalf("GetInbox: %v", err)
		}
		if len(txns) != 0 {
			t.Errorf("expected 0 transactions, got %d", len(txns))
		}
	})

	t.Run("default status filter is pending", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		txns, err := repo.GetInbox("", "")
		if err != nil {
			t.Fatalf("GetInbox: %v", err)
		}
		if len(txns) != 1 {
			t.Fatalf("expected 1 pending transaction, got %d", len(txns))
		}
		if txns[0].Status != "pending" {
			t.Errorf("expected status=pending, got %s", txns[0].Status)
		}
	})

	t.Run("status filter", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		txns, err := repo.GetInbox("processed", "")
		if err != nil {
			t.Fatalf("GetInbox: %v", err)
		}
		if len(txns) != 2 {
			t.Fatalf("expected 2 processed transactions, got %d", len(txns))
		}
	})

	t.Run("type filter", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		testutil.NewIBKRTransaction().WithStatus("pending").WithType("buy").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("pending").WithType("sell").Build(t, db)

		txns, err := repo.GetInbox("pending", "buy")
		if err != nil {
			t.Fatalf("GetInbox: %v", err)
		}
		if len(txns) != 1 {
			t.Fatalf("expected 1 buy transaction, got %d", len(txns))
		}
		if txns[0].TransactionType != "buy" {
			t.Errorf("expected type=buy, got %s", txns[0].TransactionType)
		}
	})
}

// ---------------------------------------------------------------------------
// GetIbkrInboxCount
// ---------------------------------------------------------------------------

func TestIbkrRepository_GetIbkrInboxCount(t *testing.T) {
	t.Run("empty inbox", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		count, err := repo.GetIbkrInboxCount()
		if err != nil {
			t.Fatalf("GetIbkrInboxCount: %v", err)
		}
		if count.Count != 0 {
			t.Errorf("expected count=0, got %d", count.Count)
		}
	})

	t.Run("counts only pending", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		count, err := repo.GetIbkrInboxCount()
		if err != nil {
			t.Fatalf("GetIbkrInboxCount: %v", err)
		}
		if count.Count != 2 {
			t.Errorf("expected count=2, got %d", count.Count)
		}
	})
}

// ---------------------------------------------------------------------------
// GetIbkrTransaction
// ---------------------------------------------------------------------------

func TestIbkrRepository_GetIbkrTransaction(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		created := testutil.NewIBKRTransaction().WithSymbol("MSFT").Build(t, db)

		got, err := repo.GetIbkrTransaction(created.ID)
		if err != nil {
			t.Fatalf("GetIbkrTransaction: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("expected ID=%s, got %s", created.ID, got.ID)
		}
		if got.Symbol != "MSFT" {
			t.Errorf("expected Symbol=MSFT, got %s", got.Symbol)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		_, err := repo.GetIbkrTransaction(testutil.MakeID())
		if !errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			t.Fatalf("expected ErrIBKRTransactionNotFound, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// CompareIbkrTransaction
// ---------------------------------------------------------------------------

func TestIbkrRepository_CompareIbkrTransaction(t *testing.T) {
	t.Run("existing transaction returns true", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		created := testutil.NewIBKRTransaction().Build(t, db)

		match := repo.CompareIbkrTransaction(model.IBKRTransaction{
			IBKRTransactionID: created.IBKRTransactionID,
		})
		if !match {
			t.Error("expected true for existing transaction")
		}
	})

	t.Run("non-existing transaction returns false", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		match := repo.CompareIbkrTransaction(model.IBKRTransaction{
			IBKRTransactionID: "nonexistent-ibkr-id",
		})
		if match {
			t.Error("expected false for non-existing transaction")
		}
	})
}

// ---------------------------------------------------------------------------
// AddIbkrTransactions
// ---------------------------------------------------------------------------

func TestIbkrRepository_AddIbkrTransactions(t *testing.T) {
	t.Run("empty slice is no-op", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		err := repo.AddIbkrTransactions(ctx, nil)
		if err != nil {
			t.Fatalf("expected nil error for empty slice, got %v", err)
		}
	})

	t.Run("insert multiple transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		txns := []model.IBKRTransaction{
			{
				ID:                testutil.MakeID(),
				IBKRTransactionID: "ibkr-add-1",
				TransactionDate:   now,
				Symbol:            "AAPL",
				ISIN:              "US0378331005",
				Description:       "Buy",
				TransactionType:   "buy",
				Quantity:          10,
				Price:             150,
				TotalAmount:       1500,
				Currency:          "USD",
				Fees:              1,
				Status:            "pending",
				ImportedAt:        now,
				ReportDate:        now,
				Notes:             "test",
			},
			{
				ID:                testutil.MakeID(),
				IBKRTransactionID: "ibkr-add-2",
				TransactionDate:   now,
				Symbol:            "GOOG",
				ISIN:              "US02079K3059",
				Description:       "Buy",
				TransactionType:   "buy",
				Quantity:          5,
				Price:             100,
				TotalAmount:       500,
				Currency:          "USD",
				Fees:              0.5,
				Status:            "pending",
				ImportedAt:        now,
				ReportDate:        now,
			},
		}

		err := repo.AddIbkrTransactions(ctx, txns)
		if err != nil {
			t.Fatalf("AddIbkrTransactions: %v", err)
		}

		testutil.AssertRowCount(t, db, "ibkr_transaction", 2)

		got, err := repo.GetIbkrTransaction(txns[0].ID)
		if err != nil {
			t.Fatalf("GetIbkrTransaction: %v", err)
		}
		if got.Symbol != "AAPL" {
			t.Errorf("expected Symbol=AAPL, got %s", got.Symbol)
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateIbkrTransactionStatus
// ---------------------------------------------------------------------------

func TestIbkrRepository_UpdateIbkrTransactionStatus(t *testing.T) {
	t.Run("update status", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		created := testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)

		now := time.Now().UTC().Truncate(time.Second)
		err := repo.UpdateIbkrTransactionStatus(ctx, created.ID, "processed", &now)
		if err != nil {
			t.Fatalf("UpdateIbkrTransactionStatus: %v", err)
		}

		got, err := repo.GetIbkrTransaction(created.ID)
		if err != nil {
			t.Fatalf("GetIbkrTransaction: %v", err)
		}
		if got.Status != "processed" {
			t.Errorf("expected status=processed, got %s", got.Status)
		}
		if got.ProcessedAt == nil {
			t.Error("expected ProcessedAt to be set")
		}
	})

	t.Run("update non-existent returns sentinel", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		err := repo.UpdateIbkrTransactionStatus(ctx, testutil.MakeID(), "processed", nil)
		if !errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			t.Fatalf("expected ErrIBKRTransactionNotFound, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteIbkrTransaction
// ---------------------------------------------------------------------------

func TestIbkrRepository_DeleteIbkrTransaction(t *testing.T) {
	t.Run("delete existing", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		created := testutil.NewIBKRTransaction().Build(t, db)

		err := repo.DeleteIbkrTransaction(ctx, created.ID)
		if err != nil {
			t.Fatalf("DeleteIbkrTransaction: %v", err)
		}

		_, err = repo.GetIbkrTransaction(created.ID)
		if !errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			t.Fatalf("expected ErrIBKRTransactionNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent returns sentinel", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		err := repo.DeleteIbkrTransaction(ctx, testutil.MakeID())
		if !errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			t.Fatalf("expected ErrIBKRTransactionNotFound, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Transaction Allocation CRUD
// ---------------------------------------------------------------------------

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestIbkrRepository_TransactionAllocationCRUD(t *testing.T) {
	t.Run("insert and get allocations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		// Create prerequisites: portfolio, fund, portfolio_fund, transaction, ibkr txn
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).Build(t, db)
		ibkrTxn := testutil.NewIBKRTransaction().Build(t, db)

		now := time.Now().UTC().Truncate(time.Second)
		alloc := model.IBKRTransactionAllocation{
			ID:                   testutil.MakeID(),
			IBKRTransactionID:    ibkrTxn.ID,
			PortfolioID:          portfolio.ID,
			TransactionID:        txn.ID,
			AllocationPercentage: 100,
			AllocatedAmount:      1500,
			AllocatedShares:      10,
			CreatedAt:            now,
		}

		err := repo.InsertIbkrTransactionAllocation(ctx, alloc)
		if err != nil {
			t.Fatalf("InsertIbkrTransactionAllocation: %v", err)
		}

		allocations, err := repo.GetIbkrTransactionAllocations(ibkrTxn.ID)
		if err != nil {
			t.Fatalf("GetIbkrTransactionAllocations: %v", err)
		}
		if len(allocations) != 1 {
			t.Fatalf("expected 1 allocation, got %d", len(allocations))
		}
		if allocations[0].PortfolioID != portfolio.ID {
			t.Errorf("expected portfolioID=%s, got %s", portfolio.ID, allocations[0].PortfolioID)
		}
		if allocations[0].PortfolioName != portfolio.Name {
			t.Errorf("expected portfolioName=%s, got %s", portfolio.Name, allocations[0].PortfolioName)
		}
	})

	t.Run("get allocations empty", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		allocations, err := repo.GetIbkrTransactionAllocations(testutil.MakeID())
		if err != nil {
			t.Fatalf("GetIbkrTransactionAllocations: %v", err)
		}
		if len(allocations) != 0 {
			t.Errorf("expected 0 allocations, got %d", len(allocations))
		}
	})

	t.Run("delete allocations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		ibkrTxn := testutil.NewIBKRTransaction().Build(t, db)

		now := time.Now().UTC().Truncate(time.Second)
		alloc := model.IBKRTransactionAllocation{
			ID:                   testutil.MakeID(),
			IBKRTransactionID:    ibkrTxn.ID,
			PortfolioID:          portfolio.ID,
			AllocationPercentage: 100,
			AllocatedAmount:      1000,
			AllocatedShares:      5,
			CreatedAt:            now,
		}

		if err := repo.InsertIbkrTransactionAllocation(ctx, alloc); err != nil {
			t.Fatalf("InsertIbkrTransactionAllocation: %v", err)
		}

		err := repo.DeleteIbkrTransactionAllocations(ctx, ibkrTxn.ID)
		if err != nil {
			t.Fatalf("DeleteIbkrTransactionAllocations: %v", err)
		}

		allocations, err := repo.GetIbkrTransactionAllocations(ibkrTxn.ID)
		if err != nil {
			t.Fatalf("GetIbkrTransactionAllocations: %v", err)
		}
		if len(allocations) != 0 {
			t.Errorf("expected 0 allocations after delete, got %d", len(allocations))
		}
	})

	t.Run("delete allocations not found returns sentinel", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		err := repo.DeleteIbkrTransactionAllocations(ctx, testutil.MakeID())
		if !errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			t.Fatalf("expected ErrIBKRTransactionNotFound, got %v", err)
		}
	})

	t.Run("count allocations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		portfolio1 := testutil.NewPortfolio().Build(t, db)
		portfolio2 := testutil.NewPortfolio().Build(t, db)
		ibkrTxn := testutil.NewIBKRTransaction().Build(t, db)

		now := time.Now().UTC().Truncate(time.Second)
		for _, p := range []model.Portfolio{portfolio1, portfolio2} {
			alloc := model.IBKRTransactionAllocation{
				ID:                   testutil.MakeID(),
				IBKRTransactionID:    ibkrTxn.ID,
				PortfolioID:          p.ID,
				AllocationPercentage: 50,
				AllocatedAmount:      750,
				AllocatedShares:      5,
				CreatedAt:            now,
			}
			if err := repo.InsertIbkrTransactionAllocation(ctx, alloc); err != nil {
				t.Fatalf("InsertIbkrTransactionAllocation: %v", err)
			}
		}

		count, err := repo.CountIbkrAllocationsByIbkrTransactionID(ctx, ibkrTxn.ID)
		if err != nil {
			t.Fatalf("CountIbkrAllocationsByIbkrTransactionID: %v", err)
		}
		if count != 2 {
			t.Errorf("expected count=2, got %d", count)
		}
	})

	t.Run("get transaction IDs by ibkr transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).Build(t, db)
		ibkrTxn := testutil.NewIBKRTransaction().Build(t, db)

		now := time.Now().UTC().Truncate(time.Second)
		alloc := model.IBKRTransactionAllocation{
			ID:                   testutil.MakeID(),
			IBKRTransactionID:    ibkrTxn.ID,
			PortfolioID:          portfolio.ID,
			AllocationPercentage: 100,
			AllocatedAmount:      1500,
			AllocatedShares:      10,
			TransactionID:        txn.ID,
			CreatedAt:            now,
		}
		if err := repo.InsertIbkrTransactionAllocation(ctx, alloc); err != nil {
			t.Fatalf("InsertIbkrTransactionAllocation: %v", err)
		}

		ids, err := repo.GetTransactionIDsByIbkrTransaction(ctx, ibkrTxn.ID)
		if err != nil {
			t.Fatalf("GetTransactionIDsByIbkrTransaction: %v", err)
		}
		if len(ids) != 1 || ids[0] != txn.ID {
			t.Errorf("expected [%s], got %v", txn.ID, ids)
		}
	})

	t.Run("get ibkr transaction ID by transaction ID", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		txn := testutil.NewTransaction(pf.ID).Build(t, db)
		ibkrTxn := testutil.NewIBKRTransaction().Build(t, db)

		now := time.Now().UTC().Truncate(time.Second)
		alloc := model.IBKRTransactionAllocation{
			ID:                   testutil.MakeID(),
			IBKRTransactionID:    ibkrTxn.ID,
			PortfolioID:          portfolio.ID,
			AllocationPercentage: 100,
			AllocatedAmount:      1500,
			AllocatedShares:      10,
			TransactionID:        txn.ID,
			CreatedAt:            now,
		}
		if err := repo.InsertIbkrTransactionAllocation(ctx, alloc); err != nil {
			t.Fatalf("InsertIbkrTransactionAllocation: %v", err)
		}

		gotID, err := repo.GetIbkrTransactionIDByTransactionID(ctx, txn.ID)
		if err != nil {
			t.Fatalf("GetIbkrTransactionIDByTransactionID: %v", err)
		}
		if gotID != ibkrTxn.ID {
			t.Errorf("expected %s, got %s", ibkrTxn.ID, gotID)
		}
	})

	t.Run("get ibkr transaction ID not found returns empty", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		gotID, err := repo.GetIbkrTransactionIDByTransactionID(ctx, testutil.MakeID())
		if err != nil {
			t.Fatalf("GetIbkrTransactionIDByTransactionID: %v", err)
		}
		if gotID != "" {
			t.Errorf("expected empty string, got %s", gotID)
		}
	})
}

// ---------------------------------------------------------------------------
// GetPendingDividends
// ---------------------------------------------------------------------------

// insertPendingDividend inserts a dividend with reinvestment_status = 'PENDING' (uppercase, as the
// service layer uses). The testutil factory defaults to lowercase which won't match the SQL query.
func insertPendingDividend(t *testing.T, db *sql.DB, fundID, pfID string) string {
	t.Helper()
	id := testutil.MakeID()
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO dividend (id, fund_id, portfolio_fund_id, record_date, ex_dividend_date,
			shares_owned, dividend_per_share, total_amount, reinvestment_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'PENDING')`,
		id, fundID, pfID,
		now.AddDate(0, 0, -10).Format("2006-01-02"),
		now.AddDate(0, 0, -5).Format("2006-01-02"),
		100.0, 0.50, 50.0,
	)
	if err != nil {
		t.Fatalf("insertPendingDividend: %v", err)
	}
	return id
}

func TestIbkrRepository_GetPendingDividends(t *testing.T) {
	t.Run("no pending dividends", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		divs, err := repo.GetPendingDividends("", "")
		if err != nil {
			t.Fatalf("GetPendingDividends: %v", err)
		}
		if len(divs) != 0 {
			t.Errorf("expected 0 dividends, got %d", len(divs))
		}
	})

	t.Run("returns only pending dividends", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithSymbol("DIVTEST").WithISIN("US1234567890").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// PENDING dividend (uppercase as service layer writes it)
		insertPendingDividend(t, db, fund.ID, pf.ID)

		// completed dividend
		txn := testutil.NewTransaction(pf.ID).Build(t, db)
		testutil.NewDividend(fund.ID, pf.ID).WithReinvestmentTransaction(txn.ID).Build(t, db)

		divs, err := repo.GetPendingDividends("", "")
		if err != nil {
			t.Fatalf("GetPendingDividends: %v", err)
		}
		if len(divs) != 1 {
			t.Fatalf("expected 1 pending dividend, got %d", len(divs))
		}
	})

	t.Run("filter by symbol", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().WithSymbol("AAA").Build(t, db)
		fund2 := testutil.NewFund().WithSymbol("BBB").Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		insertPendingDividend(t, db, fund1.ID, pf1.ID)
		insertPendingDividend(t, db, fund2.ID, pf2.ID)

		divs, err := repo.GetPendingDividends("AAA", "")
		if err != nil {
			t.Fatalf("GetPendingDividends: %v", err)
		}
		if len(divs) != 1 {
			t.Fatalf("expected 1 dividend for AAA, got %d", len(divs))
		}
	})

	t.Run("filter by isin", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().WithISIN("US9999999999").Build(t, db)
		fund2 := testutil.NewFund().WithISIN("DE8888888888").Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		insertPendingDividend(t, db, fund1.ID, pf1.ID)
		insertPendingDividend(t, db, fund2.ID, pf2.ID)

		divs, err := repo.GetPendingDividends("", "DE8888888888")
		if err != nil {
			t.Fatalf("GetPendingDividends: %v", err)
		}
		if len(divs) != 1 {
			t.Fatalf("expected 1 dividend for DE ISIN, got %d", len(divs))
		}
	})
}

// ---------------------------------------------------------------------------
// WriteImportCache / GetIbkrImportCache
// ---------------------------------------------------------------------------

func TestIbkrRepository_ImportCache(t *testing.T) {
	t.Run("write and read cache", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		cache := model.IbkrImportCache{
			ID:        testutil.MakeID(),
			CacheKey:  "test-key",
			Data:      []byte("<xml>data</xml>"),
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
		}

		err := repo.WriteImportCache(ctx, cache)
		if err != nil {
			t.Fatalf("WriteImportCache: %v", err)
		}

		got, err := repo.GetIbkrImportCache()
		if err != nil {
			t.Fatalf("GetIbkrImportCache: %v", err)
		}
		if got.CacheKey != "test-key" {
			t.Errorf("expected CacheKey=test-key, got %s", got.CacheKey)
		}
		if string(got.Data) != "<xml>data</xml>" {
			t.Errorf("expected data match, got %s", string(got.Data))
		}
	})

	t.Run("cache not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)

		_, err := repo.GetIbkrImportCache()
		if !errors.Is(err, apperrors.ErrIbkrImportCacheNotFound) {
			t.Fatalf("expected ErrIbkrImportCacheNotFound, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateLastImportDate
// ---------------------------------------------------------------------------

func TestIbkrRepository_UpdateLastImportDate(t *testing.T) {
	t.Run("updates last import date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewIbkrRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		cfg := &model.IbkrConfig{
			ID:          testutil.MakeID(),
			FlexToken:   "tok",
			FlexQueryID: "query-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.UpdateIbkrConfig(ctx, true, cfg); err != nil {
			t.Fatalf("UpdateIbkrConfig: %v", err)
		}

		newDate := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
		err := repo.UpdateLastImportDate(ctx, "query-1", newDate)
		if err != nil {
			t.Fatalf("UpdateLastImportDate: %v", err)
		}

		got, err := repo.GetIbkrConfig()
		if err != nil {
			t.Fatalf("GetIbkrConfig: %v", err)
		}
		if got.LastImportDate == nil {
			t.Fatal("expected LastImportDate to be set")
		}
	})
}
