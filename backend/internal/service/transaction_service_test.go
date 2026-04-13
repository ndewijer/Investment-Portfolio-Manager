package service_test

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// countRealizedGainLoss returns the number of realized_gain_loss records for a transaction.
func countRealizedGainLoss(t *testing.T, db *sql.DB, transactionID string) int {
	t.Helper()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM realized_gain_loss WHERE transaction_id = ?`, transactionID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count realized_gain_loss: %v", err)
	}
	return count
}

// getRealizedGainLoss returns the realized gain/loss values for a transaction.
func getRealizedGainLoss(t *testing.T, db *sql.DB, transactionID string) (sharesSold, costBasis, saleProceeds, realizedGL float64) {
	t.Helper()
	err := db.QueryRow(
		`SELECT shares_sold, cost_basis, sale_proceeds, realized_gain_loss FROM realized_gain_loss WHERE transaction_id = ?`,
		transactionID,
	).Scan(&sharesSold, &costBasis, &saleProceeds, &realizedGL)
	if err != nil {
		t.Fatalf("failed to get realized_gain_loss: %v", err)
	}
	return
}

// getIbkrTransactionStatus returns the status of an IBKR transaction.
func getIbkrTransactionStatus(t *testing.T, db *sql.DB, ibkrTxnID string) string {
	t.Helper()
	var status string
	err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTxnID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get ibkr transaction status: %v", err)
	}
	return status
}

// almostEqual compares two floats with a tolerance for floating point precision.
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

// --- Gap 1: Realized Gain/Loss created on sell ---

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestTransactionService_CreateSellTransaction_CreatesRealizedGainLoss(t *testing.T) {
	t.Run("creates realized gain/loss record with correct values", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares @ $10
		testutil.NewTransaction(pf.ID).WithShares(100).WithCostPerShare(10.0).Build(t, db)

		// Sell 50 shares @ $15 (gain of $250)
		sellTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          50,
			CostPerShare:    15.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() returned unexpected error: %v", err)
		}

		// Verify realized gain/loss record was created
		if count := countRealizedGainLoss(t, db, sellTx.ID); count != 1 {
			t.Fatalf("Expected 1 realized_gain_loss record, got %d", count)
		}

		sharesSold, costBasis, saleProceeds, realizedGL := getRealizedGainLoss(t, db, sellTx.ID)

		if sharesSold != 50 {
			t.Errorf("Expected shares_sold=50, got %f", sharesSold)
		}
		// Cost basis: 50 shares * $10 avg = $500
		if !almostEqual(costBasis, 500.0) {
			t.Errorf("Expected cost_basis=500, got %f", costBasis)
		}
		// Sale proceeds: 50 * $15 = $750
		if !almostEqual(saleProceeds, 750.0) {
			t.Errorf("Expected sale_proceeds=750, got %f", saleProceeds)
		}
		// Realized gain: $750 - $500 = $250
		if !almostEqual(realizedGL, 250.0) {
			t.Errorf("Expected realized_gain_loss=250, got %f", realizedGL)
		}
	})

	t.Run("creates realized loss record when selling at a loss", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares @ $15
		testutil.NewTransaction(pf.ID).WithShares(100).WithCostPerShare(15.0).Build(t, db)

		// Sell 50 shares @ $10 (loss of $250)
		sellTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          50,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() returned unexpected error: %v", err)
		}

		_, costBasis, saleProceeds, realizedGL := getRealizedGainLoss(t, db, sellTx.ID)

		// Cost basis: 50 * $15 = $750
		if !almostEqual(costBasis, 750.0) {
			t.Errorf("Expected cost_basis=750, got %f", costBasis)
		}
		// Sale proceeds: 50 * $10 = $500
		if !almostEqual(saleProceeds, 500.0) {
			t.Errorf("Expected sale_proceeds=500, got %f", saleProceeds)
		}
		// Realized loss: $500 - $750 = -$250
		if !almostEqual(realizedGL, -250.0) {
			t.Errorf("Expected realized_gain_loss=-250, got %f", realizedGL)
		}
	})

	t.Run("calculates weighted average cost with multiple buys", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares @ $10 = $1000
		testutil.NewTransaction(pf.ID).WithShares(100).WithCostPerShare(10.0).Build(t, db)
		// Buy 50 shares @ $20 = $1000
		testutil.NewTransaction(pf.ID).WithShares(50).WithCostPerShare(20.0).Build(t, db)
		// Total: 150 shares, $2000 cost, avg = $13.33

		// Sell 30 shares @ $25
		sellTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          30,
			CostPerShare:    25.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() returned unexpected error: %v", err)
		}

		_, costBasis, saleProceeds, realizedGL := getRealizedGainLoss(t, db, sellTx.ID)

		// Average cost: $2000 / 150 = $13.33...
		// Cost basis: 30 * $13.33 = $400
		expectedCostBasis := (2000.0 / 150.0) * 30.0
		if !almostEqual(costBasis, expectedCostBasis) {
			t.Errorf("Expected cost_basis=%.2f, got %.2f", expectedCostBasis, costBasis)
		}
		// Sale proceeds: 30 * $25 = $750
		if !almostEqual(saleProceeds, 750.0) {
			t.Errorf("Expected sale_proceeds=750, got %f", saleProceeds)
		}
		// Gain: $750 - $400 = $350
		if !almostEqual(realizedGL, 750.0-expectedCostBasis) {
			t.Errorf("Expected realized_gain_loss=%.2f, got %.2f", 750.0-expectedCostBasis, realizedGL)
		}
	})

	t.Run("no realized gain/loss for buy transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          100,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() returned unexpected error: %v", err)
		}

		if count := countRealizedGainLoss(t, db, buyTx.ID); count != 0 {
			t.Errorf("Expected 0 realized_gain_loss records for buy, got %d", count)
		}
	})
}

// --- Gap 2: Insufficient shares validation ---

func TestTransactionService_CreateSellTransaction_InsufficientShares(t *testing.T) {
	t.Run("rejects sell with no shares owned", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		_, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          10,
			CostPerShare:    15.0,
		})

		if !errors.Is(err, apperrors.ErrInsufficientShares) {
			t.Errorf("Expected ErrInsufficientShares, got: %v", err)
		}
	})

	t.Run("rejects sell exceeding owned shares", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 50 shares
		testutil.NewTransaction(pf.ID).WithShares(50).Build(t, db)

		// Try to sell 100
		_, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          100,
			CostPerShare:    15.0,
		})

		if !errors.Is(err, apperrors.ErrInsufficientShares) {
			t.Errorf("Expected ErrInsufficientShares, got: %v", err)
		}
	})

	t.Run("allows sell of exact shares owned", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares
		testutil.NewTransaction(pf.ID).WithShares(100).WithCostPerShare(10.0).Build(t, db)

		// Sell all 100
		_, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          100,
			CostPerShare:    15.0,
		})

		if err != nil {
			t.Errorf("Expected sell of exact shares to succeed, got: %v", err)
		}
	})
}

// --- Gap 3: Update recalculates realized gains ---

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestTransactionService_UpdateTransaction_RecalculatesRealizedGains(t *testing.T) {
	t.Run("recalculates when sell shares change", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares @ $10
		testutil.NewTransaction(pf.ID).WithShares(100).WithCostPerShare(10.0).Build(t, db)

		// Sell 50 @ $15 (gain of $250)
		sellTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          50,
			CostPerShare:    15.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		// Update to sell 30 shares instead
		newShares := 30.0
		_, err = svc.UpdateTransaction(ctx, sellTx.ID, request.UpdateTransactionRequest{
			Shares: &newShares,
		})
		if err != nil {
			t.Fatalf("UpdateTransaction() error: %v", err)
		}

		sharesSold, costBasis, saleProceeds, realizedGL := getRealizedGainLoss(t, db, sellTx.ID)

		if sharesSold != 30 {
			t.Errorf("Expected shares_sold=30, got %f", sharesSold)
		}
		// Cost basis: 30 * $10 = $300
		if !almostEqual(costBasis, 300.0) {
			t.Errorf("Expected cost_basis=300, got %f", costBasis)
		}
		// Sale proceeds: 30 * $15 = $450
		if !almostEqual(saleProceeds, 450.0) {
			t.Errorf("Expected sale_proceeds=450, got %f", saleProceeds)
		}
		// Gain: $450 - $300 = $150
		if !almostEqual(realizedGL, 150.0) {
			t.Errorf("Expected realized_gain_loss=150, got %f", realizedGL)
		}
	})

	t.Run("creates realized gain/loss when changing buy to sell", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 200 shares @ $10 (covers the sell)
		testutil.NewTransaction(pf.ID).WithShares(200).WithCostPerShare(10.0).Build(t, db)

		// Create a buy transaction that we'll change to sell
		buyTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          50,
			CostPerShare:    15.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		// No realized gain/loss for buy
		if count := countRealizedGainLoss(t, db, buyTx.ID); count != 0 {
			t.Fatalf("Expected 0 RGL for buy, got %d", count)
		}

		// Change to sell
		sellType := "sell"
		_, err = svc.UpdateTransaction(ctx, buyTx.ID, request.UpdateTransactionRequest{
			Type: &sellType,
		})
		if err != nil {
			t.Fatalf("UpdateTransaction() error: %v", err)
		}

		// Now should have realized gain/loss
		if count := countRealizedGainLoss(t, db, buyTx.ID); count != 1 {
			t.Errorf("Expected 1 RGL after changing to sell, got %d", count)
		}
	})

	t.Run("removes realized gain/loss when changing sell to buy", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares
		testutil.NewTransaction(pf.ID).WithShares(100).WithCostPerShare(10.0).Build(t, db)

		// Sell 50
		sellTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          50,
			CostPerShare:    15.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		if count := countRealizedGainLoss(t, db, sellTx.ID); count != 1 {
			t.Fatalf("Expected 1 RGL for sell, got %d", count)
		}

		// Change to buy
		buyType := "buy"
		_, err = svc.UpdateTransaction(ctx, sellTx.ID, request.UpdateTransactionRequest{
			Type: &buyType,
		})
		if err != nil {
			t.Fatalf("UpdateTransaction() error: %v", err)
		}

		// Realized gain/loss should be deleted
		if count := countRealizedGainLoss(t, db, sellTx.ID); count != 0 {
			t.Errorf("Expected 0 RGL after changing to buy, got %d", count)
		}
	})

	t.Run("rejects update to sell with insufficient shares", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 10 shares
		testutil.NewTransaction(pf.ID).WithShares(10).WithCostPerShare(10.0).Build(t, db)

		// Create buy of 50 shares, then try to change to sell (only 10 available excluding this tx)
		buyTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          50,
			CostPerShare:    15.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		sellType := "sell"
		_, err = svc.UpdateTransaction(ctx, buyTx.ID, request.UpdateTransactionRequest{
			Type: &sellType,
		})
		if !errors.Is(err, apperrors.ErrInsufficientShares) {
			t.Errorf("Expected ErrInsufficientShares, got: %v", err)
		}
	})
}

// --- Gap 4: Delete cleans up realized gains ---

func TestTransactionService_DeleteTransaction_CleansUpRealizedGains(t *testing.T) {
	t.Run("deletes realized gain/loss when sell transaction is deleted", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares
		testutil.NewTransaction(pf.ID).WithShares(100).WithCostPerShare(10.0).Build(t, db)

		// Sell 50
		sellTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "sell",
			Shares:          50,
			CostPerShare:    15.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		// Verify RGL exists
		if count := countRealizedGainLoss(t, db, sellTx.ID); count != 1 {
			t.Fatalf("Expected 1 RGL, got %d", count)
		}

		// Delete the sell transaction
		err = svc.DeleteTransaction(ctx, sellTx.ID)
		if err != nil {
			t.Fatalf("DeleteTransaction() error: %v", err)
		}

		// RGL should be cleaned up
		if count := countRealizedGainLoss(t, db, sellTx.ID); count != 0 {
			t.Errorf("Expected 0 RGL after delete, got %d", count)
		}
	})

	t.Run("does not affect realized gain/loss when deleting buy transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create and delete a buy - should not touch RGL table
		buyTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          100,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		err = svc.DeleteTransaction(ctx, buyTx.ID)
		if err != nil {
			t.Fatalf("DeleteTransaction() error: %v", err)
		}

		// Verify no crash and no orphaned records
		var totalRGL int
		err = db.QueryRow(`SELECT COUNT(*) FROM realized_gain_loss`).Scan(&totalRGL)
		if err != nil {
			t.Fatalf("failed to count RGL: %v", err)
		}
		if totalRGL != 0 {
			t.Errorf("Expected 0 total RGL records, got %d", totalRGL)
		}
	})
}

// --- Gap 5: Delete reverts IBKR status ---

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestTransactionService_DeleteTransaction_RevertsIBKRStatus(t *testing.T) {
	t.Run("reverts IBKR transaction to pending when last allocation is deleted", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create a buy transaction (the one linked to IBKR)
		buyTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          100,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		// Create IBKR transaction (processed) and link via allocation
		ibkrTxnID := testutil.MakeID()
		_, err = db.Exec(
			`INSERT INTO ibkr_transaction (id, ibkr_transaction_id, transaction_date, symbol, transaction_type, quantity, price, total_amount, currency, fees, status, imported_at, report_date, notes)
			VALUES (?, ?, '2025-06-01', 'TEST', 'trade', 100, 10.0, 1000.0, 'USD', 0, 'processed', datetime('now'), '2025-06-01', '')`,
			ibkrTxnID, "IBKR-123",
		)
		if err != nil {
			t.Fatalf("Failed to create IBKR transaction: %v", err)
		}

		allocID := testutil.MakeID()
		_, err = db.Exec(
			`INSERT INTO ibkr_transaction_allocation (id, ibkr_transaction_id, portfolio_id, allocation_percentage, allocated_amount, allocated_shares, transaction_id, created_at)
			VALUES (?, ?, ?, 100.0, 1000.0, 100.0, ?, datetime('now'))`,
			allocID, ibkrTxnID, portfolio.ID, buyTx.ID,
		)
		if err != nil {
			t.Fatalf("Failed to create IBKR allocation: %v", err)
		}

		// Verify initial status is processed
		if status := getIbkrTransactionStatus(t, db, ibkrTxnID); status != "processed" {
			t.Fatalf("Expected initial status 'processed', got '%s'", status)
		}

		// Delete the transaction
		err = svc.DeleteTransaction(ctx, buyTx.ID)
		if err != nil {
			t.Fatalf("DeleteTransaction() error: %v", err)
		}

		// IBKR transaction should be reverted to pending
		if status := getIbkrTransactionStatus(t, db, ibkrTxnID); status != "pending" {
			t.Errorf("Expected IBKR status 'pending' after delete, got '%s'", status)
		}
	})

	t.Run("does not revert IBKR status when other allocations remain", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio1 := testutil.NewPortfolio().Build(t, db)
		portfolio2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio1.ID, fund.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio2.ID, fund.ID).Build(t, db)

		// Create two buy transactions (one per portfolio)
		buyTx1, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf1.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          50,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		buyTx2, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf2.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          50,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		// Create IBKR transaction with two allocations
		ibkrTxnID := testutil.MakeID()
		_, err = db.Exec(
			`INSERT INTO ibkr_transaction (id, ibkr_transaction_id, transaction_date, symbol, transaction_type, quantity, price, total_amount, currency, fees, status, imported_at, report_date, notes)
			VALUES (?, ?, '2025-06-01', 'TEST', 'trade', 100, 10.0, 1000.0, 'USD', 0, 'processed', datetime('now'), '2025-06-01', '')`,
			ibkrTxnID, "IBKR-456",
		)
		if err != nil {
			t.Fatalf("Failed to create IBKR transaction: %v", err)
		}

		_, err = db.Exec(
			`INSERT INTO ibkr_transaction_allocation (id, ibkr_transaction_id, portfolio_id, allocation_percentage, allocated_amount, allocated_shares, transaction_id, created_at)
			VALUES (?, ?, ?, 50.0, 500.0, 50.0, ?, datetime('now'))`,
			testutil.MakeID(), ibkrTxnID, portfolio1.ID, buyTx1.ID,
		)
		if err != nil {
			t.Fatalf("Failed to create IBKR allocation 1: %v", err)
		}
		_, err = db.Exec(
			`INSERT INTO ibkr_transaction_allocation (id, ibkr_transaction_id, portfolio_id, allocation_percentage, allocated_amount, allocated_shares, transaction_id, created_at)
			VALUES (?, ?, ?, 50.0, 500.0, 50.0, ?, datetime('now'))`,
			testutil.MakeID(), ibkrTxnID, portfolio2.ID, buyTx2.ID,
		)
		if err != nil {
			t.Fatalf("Failed to create IBKR allocation 2: %v", err)
		}

		// Delete only the first transaction
		err = svc.DeleteTransaction(ctx, buyTx1.ID)
		if err != nil {
			t.Fatalf("DeleteTransaction() error: %v", err)
		}

		// IBKR transaction should still be processed (one allocation remains)
		if status := getIbkrTransactionStatus(t, db, ibkrTxnID); status != "processed" {
			t.Errorf("Expected IBKR status 'processed' (other allocation remains), got '%s'", status)
		}
	})

	t.Run("no error when deleting transaction without IBKR linkage", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestTransactionService(t, db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyTx, err := svc.CreateTransaction(ctx, request.CreateTransactionRequest{
			PortfolioFundID: pf.ID,
			Date:            "2025-06-01",
			Type:            "buy",
			Shares:          100,
			CostPerShare:    10.0,
		})
		if err != nil {
			t.Fatalf("CreateTransaction() error: %v", err)
		}

		// Delete should succeed without IBKR linkage
		err = svc.DeleteTransaction(ctx, buyTx.ID)
		if err != nil {
			t.Errorf("DeleteTransaction() should succeed without IBKR linkage, got: %v", err)
		}
	})
}
