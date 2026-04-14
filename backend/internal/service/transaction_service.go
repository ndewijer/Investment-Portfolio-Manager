package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var txLog = logging.NewLogger("transaction")

// TransactionService handles transaction-related business logic operations.
// This includes sell processing with realized gain/loss tracking, insufficient shares
// validation, and IBKR allocation cleanup on deletion.
type TransactionService struct {
	db                      *sql.DB
	transactionRepo         *repository.TransactionRepository
	pfRepo                  *repository.PortfolioFundRepository
	realizedGainLossRepo    *repository.RealizedGainLossRepository
	ibkrRepo                *repository.IbkrRepository
	materializedInvalidator MaterializedInvalidator
}

// NewTransactionService creates a new TransactionService with the provided repository dependencies.
func NewTransactionService(
	db *sql.DB,
	transactionRepo *repository.TransactionRepository,
	pfRepo *repository.PortfolioFundRepository,
	realizedGainLossRepo *repository.RealizedGainLossRepository,
	ibkrRepo *repository.IbkrRepository,
) *TransactionService {
	return &TransactionService{
		db:                   db,
		transactionRepo:      transactionRepo,
		pfRepo:               pfRepo,
		realizedGainLossRepo: realizedGainLossRepo,
		ibkrRepo:             ibkrRepo,
	}
}

// SetMaterializedInvalidator injects the MaterializedInvalidator after construction.
// This breaks the circular initialization order between TransactionService and MaterializedService.
func (s *TransactionService) SetMaterializedInvalidator(m MaterializedInvalidator) {
	s.materializedInvalidator = m
}

// getOldestTransaction returns the date of the earliest transaction across the given portfolio_fund IDs.
// This is used to determine the earliest date for which portfolio calculations can be performed.
func (s *TransactionService) getOldestTransaction(pfIDs []string) time.Time {
	return s.transactionRepo.GetOldestTransaction(pfIDs)
}

// loadTransactions retrieves transactions for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio_fund ID, allowing callers to decide how to aggregate.
func (s *TransactionService) loadTransactions(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Transaction, error) {
	result, err := s.transactionRepo.GetTransactions(pfIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get transactions: %w", err)
	}
	return result, nil
}

// GetTransactionsperPortfolio retrieves all transactions for a specific portfolio or all transactions if portfolioID is empty.
// Returns enriched transaction data including fund names and IBKR linkage status.
func (s *TransactionService) GetTransactionsperPortfolio(portfolioID string) ([]model.TransactionResponse, error) {
	txLog.Debug("retrieving transactions per portfolio", "portfolioID", portfolioID)
	result, err := s.transactionRepo.GetTransactionsPerPortfolio(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("get transactions per portfolio: %w", err)
	}
	return result, nil
}

// GetTransaction retrieves a single transaction by its ID.
// Returns enriched transaction data including fund name and IBKR linkage status.
func (s *TransactionService) GetTransaction(transactionID string) (model.TransactionResponse, error) {
	txLog.Debug("retrieving transaction", "transactionID", transactionID)
	result, err := s.transactionRepo.GetTransaction(transactionID)
	if err != nil {
		return model.TransactionResponse{}, fmt.Errorf("get transaction: %w", err)
	}
	return result, nil
}

// CreateTransaction creates a new transaction from the provided request.
// For sell transactions, validates sufficient shares and automatically creates
// a RealizedGainLoss record with the calculated cost basis and gain/loss.
//
// Returns an enriched TransactionResponse on success.
// Returns ErrInsufficientShares if selling more shares than currently held.
// Returns an error if date parsing fails or database insertion fails.
func (s *TransactionService) CreateTransaction(ctx context.Context, req request.CreateTransactionRequest) (*model.TransactionResponse, error) {
	txLog.DebugContext(ctx, "creating transaction", "portfolioFundID", req.PortfolioFundID, "type", req.Type)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	_, err = s.pfRepo.WithTx(tx).GetPortfolioFund(req.PortfolioFundID)
	if err != nil {
		return nil, fmt.Errorf("get portfolio fund: %w", err)
	}

	transactionDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}

	transaction := &model.Transaction{
		ID:              uuid.New().String(),
		PortfolioFundID: req.PortfolioFundID,
		Date:            transactionDate.UTC(),
		Type:            req.Type,
		Shares:          req.Shares,
		CostPerShare:    req.CostPerShare,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.transactionRepo.WithTx(tx).InsertTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	if req.Type == "sell" {
		if err := s.createRealizedGainLoss(ctx, tx, transaction.ID, transaction); err != nil {
			return nil, fmt.Errorf("create realized gain/loss: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), transaction.Date, nil, "", req.PortfolioFundID); err != nil {
				txLog.Warn("failed to regenerate materialized table after transaction creation", "error", err)
			}
		}()
	}

	txLog.InfoContext(ctx, "transaction created", "transactionID", transaction.ID, "type", transaction.Type, "shares", transaction.Shares)

	resp, err := s.GetTransaction(transaction.ID)
	if err != nil {
		return nil, fmt.Errorf("get created transaction response: %w", err)
	}
	return &resp, nil
}

// UpdateTransaction updates an existing transaction with the provided changes.
// Only fields present in the request (non-nil) are updated.
//
// Handles realized gain/loss lifecycle:
//   - If the old type was "sell", deletes the existing RealizedGainLoss record
//   - If the new type is "sell", validates shares and creates a new RealizedGainLoss record
//   - If both old and new are "sell", recalculates the RealizedGainLoss record
//
// When date or PortfolioFundID changes, materialized view regeneration covers the
// earlier of old/new dates, and both old and new portfolio-funds are regenerated
// separately (Issue #35, Edge Case 3).
//
// Returns an enriched TransactionResponse on success.
// Returns ErrTransactionNotFound if the transaction does not exist.
// Returns ErrInsufficientShares if selling more shares than currently held.
//
//nolint:gocyclo // Update with realized gain/loss lifecycle + old/new date invalidation
func (s *TransactionService) UpdateTransaction(
	ctx context.Context,
	id string,
	req request.UpdateTransactionRequest,
) (*model.TransactionResponse, error) {
	txLog.DebugContext(ctx, "updating transaction", "transactionID", id)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	transaction, err := s.transactionRepo.WithTx(tx).GetTransactionByID(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}

	oldType := transaction.Type
	oldDate := transaction.Date
	oldPortfolioFundID := transaction.PortfolioFundID

	if err := s.applyTransactionUpdates(tx, &transaction, req); err != nil {
		return nil, fmt.Errorf("apply transaction updates: %w", err)
	}

	if oldType == "sell" || transaction.Type == "sell" {
		if oldType == "sell" {
			if err := s.realizedGainLossRepo.WithTx(tx).DeleteRealizedGainLossByTransactionID(ctx, id); err != nil {
				return nil, fmt.Errorf("failed to delete old realized gain/loss: %w", err)
			}
		}
		if transaction.Type == "sell" {
			if err := s.createRealizedGainLoss(ctx, tx, id, &transaction); err != nil {
				return nil, fmt.Errorf("create realized gain/loss: %w", err)
			}
		}
	}

	if err := s.transactionRepo.WithTx(tx).UpdateTransaction(ctx, &transaction); err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		regenDate := transaction.Date
		if oldDate.Before(regenDate) {
			regenDate = oldDate
		}
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), regenDate, nil, "", transaction.PortfolioFundID); err != nil {
				txLog.Warn("failed to regenerate materialized table after transaction update", "error", err)
			}
		}()
		if oldPortfolioFundID != transaction.PortfolioFundID {
			//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
			go func() {
				if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), regenDate, nil, "", oldPortfolioFundID); err != nil {
					txLog.Warn("failed to regenerate materialized table for old portfolio-fund", "error", err)
				}
			}()
		}
	}

	txLog.InfoContext(ctx, "transaction updated", "transactionID", id)

	resp, err := s.GetTransaction(id)
	if err != nil {
		return nil, fmt.Errorf("get updated transaction response: %w", err)
	}
	return &resp, nil
}

// DeleteTransaction removes a transaction from the system with proper cleanup.
//
// This method handles:
//   - Realized gain/loss record deletion for sell transactions
//   - IBKR allocation cleanup and status reversion to "pending" when deleting
//     the last allocation for an IBKR transaction
//   - Materialized view invalidation
//
// Returns ErrTransactionNotFound if the transaction does not exist.
// Returns an error if the database deletion fails.
func (s *TransactionService) DeleteTransaction(ctx context.Context, id string) error {
	txLog.DebugContext(ctx, "deleting transaction", "transactionID", id)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	transaction, err := s.transactionRepo.WithTx(tx).GetTransactionByID(id)
	if err != nil {
		return fmt.Errorf("get transaction: %w", err)
	}

	if transaction.Type == "sell" {
		if err := s.realizedGainLossRepo.WithTx(tx).DeleteRealizedGainLossByTransactionID(ctx, id); err != nil {
			return fmt.Errorf("failed to delete realized gain/loss: %w", err)
		}
	}

	ibkrTxnID, err := s.ibkrRepo.WithTx(tx).GetIbkrTransactionIDByTransactionID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check ibkr allocation: %w", err)
	}

	if ibkrTxnID != "" {
		allocCount, err := s.ibkrRepo.WithTx(tx).CountIbkrAllocationsByIbkrTransactionID(ctx, ibkrTxnID)
		if err != nil {
			return fmt.Errorf("failed to count ibkr allocations: %w", err)
		}

		if allocCount == 1 {
			if err := s.ibkrRepo.WithTx(tx).UpdateIbkrTransactionStatus(ctx, ibkrTxnID, "pending", nil); err != nil {
				return fmt.Errorf("failed to revert ibkr transaction status: %w", err)
			}
		}
	}

	err = s.transactionRepo.WithTx(tx).DeleteTransaction(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), transaction.Date, nil, "", transaction.PortfolioFundID); err != nil {
				txLog.Warn("failed to regenerate materialized table after transaction deletion", "error", err)
			}
		}()
	}

	txLog.InfoContext(ctx, "transaction deleted", "transactionID", id)
	return nil
}

// calculateCurrentPosition computes the current total shares and total cost for a portfolio fund
// using the weighted average cost method. Optionally excludes a specific transaction by ID
// (used when updating an existing transaction to avoid counting it in position calculation).
//
// Transaction processing:
//   - "buy": increase shares and cost
//   - "dividend": increase shares and cost (dividend reinvestment transactions carry actual shares)
//   - "sell": decrease shares and adjust cost proportionally
//   - "fee": increase cost (fees are part of cost basis)
//
// Note: This differs from fund_metrics.go, which receives dividend reinvestment shares via a
// separate parameter. Here we count dividend transaction shares directly because this method
// operates solely on the transaction table for position validation.
//
// Returns (totalShares, totalCost, error).
func (s *TransactionService) calculateCurrentPosition(tx *sql.Tx, pfID string, excludeTransactionID string) (float64, float64, error) {
	transactions, err := s.transactionRepo.WithTx(tx).GetTransactionsByPortfolioFundID(pfID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load transactions for position: %w", err)
	}

	var shares, cost float64

	for _, t := range transactions {
		if t.ID == excludeTransactionID {
			continue
		}
		switch t.Type {
		case "buy", "dividend":
			shares += t.Shares
			cost += t.Shares * t.CostPerShare
		case "sell":
			shares -= t.Shares
			if shares > 0 {
				cost = (cost / (shares + t.Shares)) * shares
			} else {
				cost = 0
			}
		case "fee":
			cost += t.CostPerShare
		default:
			return 0, 0, fmt.Errorf("unknown transaction type %q for transaction %s", t.Type, t.ID)
		}
	}

	return shares, cost, nil
}

// applyTransactionUpdates patches a transaction with the non-nil fields from an update request.
// Validates that the portfolio fund exists if it's being changed.
func (s *TransactionService) applyTransactionUpdates(tx *sql.Tx, transaction *model.Transaction, req request.UpdateTransactionRequest) error {
	if req.PortfolioFundID != nil {
		_, err := s.pfRepo.WithTx(tx).GetPortfolioFund(*req.PortfolioFundID)
		if err != nil {
			return fmt.Errorf("get portfolio fund: %w", err)
		}
		transaction.PortfolioFundID = *req.PortfolioFundID
	}
	if req.Date != nil {
		transactionDate, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return fmt.Errorf("parse date: %w", err)
		}
		transaction.Date = transactionDate.UTC()
	}
	if req.Type != nil {
		transaction.Type = *req.Type
	}
	if req.Shares != nil {
		transaction.Shares = *req.Shares
	}
	if req.CostPerShare != nil {
		transaction.CostPerShare = *req.CostPerShare
	}
	return nil
}

// createRealizedGainLoss validates sufficient shares and creates a RealizedGainLoss record
// for a sell transaction. Used by both CreateTransaction and UpdateTransaction.
func (s *TransactionService) createRealizedGainLoss(ctx context.Context, tx *sql.Tx, transactionID string, transaction *model.Transaction) error {
	shares, cost, err := s.calculateCurrentPosition(tx, transaction.PortfolioFundID, transactionID)
	if err != nil {
		return fmt.Errorf("failed to calculate position: %w", err)
	}

	if shares < transaction.Shares {
		return apperrors.ErrInsufficientShares
	}

	pf, err := s.pfRepo.WithTx(tx).GetPortfolioFund(transaction.PortfolioFundID)
	if err != nil {
		return fmt.Errorf("get portfolio fund: %w", err)
	}

	avgCost := 0.0
	if shares > 0 {
		avgCost = cost / shares
	}
	costBasis := avgCost * transaction.Shares
	saleProceeds := transaction.Shares * transaction.CostPerShare
	realizedGL := saleProceeds - costBasis

	rgl := &model.RealizedGainLoss{
		ID:               uuid.New().String(),
		PortfolioID:      pf.PortfolioID,
		FundID:           pf.FundID,
		TransactionID:    transactionID,
		TransactionDate:  transaction.Date,
		SharesSold:       transaction.Shares,
		CostBasis:        costBasis,
		SaleProceeds:     saleProceeds,
		RealizedGainLoss: realizedGL,
		CreatedAt:        time.Now().UTC(),
	}

	if err := s.realizedGainLossRepo.WithTx(tx).InsertRealizedGainLoss(ctx, rgl); err != nil {
		return fmt.Errorf("failed to record realized gain/loss: %w", err)
	}

	return nil
}
