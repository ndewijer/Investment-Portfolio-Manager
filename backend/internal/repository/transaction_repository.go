package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

var txnLog = logging.NewLogger("transaction")

// TransactionRepository provides data access methods for the transaction table.
// It handles retrieving and querying portfolio transactions within specified date ranges.
type TransactionRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewTransactionRepository creates a new TransactionRepository with the provided database connection.
func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// WithTx returns a new TransactionRepository scoped to the provided transaction.
func (r *TransactionRepository) WithTx(tx *sql.Tx) *TransactionRepository {
	return &TransactionRepository{
		db: r.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (r *TransactionRepository) getQuerier() Querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetTransactions retrieves all transactions for the given portfolio_fund IDs within the specified date range.
// Transactions are sorted by date in ascending order and grouped by portfolio_fund ID.
//
// Parameters:
//   - pfIDs: slice of portfolio_fund IDs to query
//   - startDate: inclusive start date for the query
//   - endDate: inclusive end date for the query
//
// Returns a map of portfolioFundID -> []Transaction. If pfIDs is empty, returns an empty map.
// This grouping allows callers to decide how to aggregate (by portfolio, by fund, etc.) after retrieval.
func (r *TransactionRepository) GetTransactions(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Transaction, error) {
	txnLog.Debug("getting transactions", "pf_count", len(pfIDs), "start_date", startDate.Format("2006-01-02"), "end_date", endDate.Format("2006-01-02"))
	if len(pfIDs) == 0 {
		return make(map[string][]model.Transaction), nil
	}

	transactionPlaceholders := make([]string, len(pfIDs))
	for i := range transactionPlaceholders {
		transactionPlaceholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	transactionQuery := `
		SELECT id, portfolio_fund_id, date, type, shares, cost_per_share, created_at
		FROM "transaction"
		WHERE portfolio_fund_id IN (` + strings.Join(transactionPlaceholders, ",") + `)
		AND date >= ?
		AND date <= ?
		ORDER BY date ASC
	`

	transactionArgs := make([]any, 0, len(pfIDs)+2)
	for _, id := range pfIDs {
		transactionArgs = append(transactionArgs, id)
	}
	transactionArgs = append(transactionArgs, startDate.Format("2006-01-02"))
	transactionArgs = append(transactionArgs, endDate.Format("2006-01-02"))

	rows, err := r.getQuerier().Query(transactionQuery, transactionArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	transactionsByPortfolioFund := make(map[string][]model.Transaction)

	for rows.Next() {

		var dateStr, createdAtStr string
		var t model.Transaction

		err := rows.Scan(
			&t.ID,
			&t.PortfolioFundID,
			&dateStr,
			&t.Type,
			&t.Shares,
			&t.CostPerShare,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction table results: %w", err)
		}
		t.Date, err = ParseTime(dateStr)
		if err != nil || t.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || t.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		transactionsByPortfolioFund[t.PortfolioFundID] = append(transactionsByPortfolioFund[t.PortfolioFundID], t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction table: %w", err)
	}

	return transactionsByPortfolioFund, nil
}

// GetOldestTransaction finds and returns the date of the earliest transaction across the given portfolio_fund IDs.
// This is used to determine the starting point for historical portfolio calculations.
//
// Returns time.Time{} (zero value) if:
//   - pfIDs is empty
//   - no transactions are found
//   - database query fails
//   - date parsing fails
func (r *TransactionRepository) GetOldestTransaction(pfIDs []string) time.Time {
	txnLog.Debug("getting oldest transaction", "pf_count", len(pfIDs))
	if len(pfIDs) == 0 {
		return time.Time{}
	}
	var oldestDateStr sql.NullString

	oldestTransactionPlaceholders := make([]string, len(pfIDs))
	for i := range oldestTransactionPlaceholders {
		oldestTransactionPlaceholders[i] = "?"
	}

	oldestTransactionQuery := `
		SELECT MIN(date)
		FROM "transaction"
		WHERE portfolio_fund_id IN (` + strings.Join(oldestTransactionPlaceholders, ",") + `)
		`

	oldestTransactionArgs := make([]any, len(pfIDs))
	for i, id := range pfIDs {
		oldestTransactionArgs[i] = id
	}

	err := r.getQuerier().QueryRow(oldestTransactionQuery, oldestTransactionArgs...).Scan(&oldestDateStr)
	if err != nil || !oldestDateStr.Valid {
		return time.Time{}
	}
	oldestDate, err := time.Parse("2006-01-02", oldestDateStr.String)
	if err != nil {
		return time.Time{}
	}

	return oldestDate.UTC()
}

// GetTransactionsPerPortfolio retrieves all transactions for a specific portfolio or all transactions if portfolioId is empty.
// Returns enriched transaction data including fund names and IBKR linkage status.
// Transactions are sorted by date in ascending order.
func (r *TransactionRepository) GetTransactionsPerPortfolio(portfolioID string) ([]model.TransactionResponse, error) {
	txnLog.Debug("getting transactions per portfolio", "portfolio_id", portfolioID)

	transactionQuery := `
		SELECT
			t.id,
			t.portfolio_fund_id,
			f.name,
			t.date,
			t.type,
			t.shares,
			t.cost_per_share,
			ita.ibkr_transaction_id,
			CASE
				WHEN ita.ibkr_transaction_id IS NOT NULL THEN 1
				ELSE 0
			END AS ibkr_linked
		FROM "transaction" t
		JOIN portfolio_fund pf ON t.portfolio_fund_id = pf.id
		JOIN portfolio p ON pf.portfolio_id = p.id
		JOIN fund f ON pf.fund_id = f.id
		LEFT JOIN ibkr_transaction_allocation ita ON t.id = ita.transaction_id
	`

	var args []any

	if portfolioID == "" {
		transactionQuery += `
		ORDER BY t.date ASC
		`
	} else {
		transactionQuery += `
		WHERE pf.portfolio_id = ?
		ORDER BY t.date ASC
		`
		args = append(args, portfolioID)
	}

	rows, err := r.getQuerier().Query(transactionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	transactionResponse := []model.TransactionResponse{}

	for rows.Next() {

		var dateStr string
		var ibkrTransactionIDStr sql.NullString
		var t model.TransactionResponse

		err := rows.Scan(
			&t.ID,
			&t.PortfolioFundID,
			&t.FundName,
			&dateStr,
			&t.Type,
			&t.Shares,
			&t.CostPerShare,
			&ibkrTransactionIDStr,
			&t.IbkrLinked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction table results: %w", err)
		}
		t.Date, err = ParseTime(dateStr)
		if err != nil || t.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		// IbkrTransactionId is nullable
		if ibkrTransactionIDStr.Valid {
			t.IbkrTransactionID = ibkrTransactionIDStr.String
		}

		transactionResponse = append(transactionResponse, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction table: %w", err)
	}

	return transactionResponse, nil
}

// GetTransaction retrieves a single transaction by its ID.
// Returns enriched transaction data including fund name and IBKR linkage status.
// Returns an empty TransactionResponse if transactionID is empty or not found.
func (r *TransactionRepository) GetTransaction(transactionID string) (model.TransactionResponse, error) {
	txnLog.Debug("getting transaction", "transaction_id", transactionID)
	if transactionID == "" {
		return model.TransactionResponse{}, nil
	}

	transactionQuery := `
		SELECT
			t.id,
			t.portfolio_fund_id,
			f.name,
			t.date,
			t.type,
			t.shares,
			t.cost_per_share,
			ita.ibkr_transaction_id,
			CASE
				WHEN ita.ibkr_transaction_id IS NOT NULL THEN 1
				ELSE 0
			END AS ibkr_linked
		FROM "transaction" t
		JOIN portfolio_fund pf ON t.portfolio_fund_id = pf.id
		JOIN portfolio p ON pf.portfolio_id = p.id
		JOIN fund f ON pf.fund_id = f.id
		LEFT JOIN ibkr_transaction_allocation ita ON t.id = ita.transaction_id
		WHERE t.id = ?
	`
	var t model.TransactionResponse
	var dateStr string
	var ibkrTransactionIDStr sql.NullString
	err := r.getQuerier().QueryRow(transactionQuery, transactionID).Scan(
		&t.ID,
		&t.PortfolioFundID,
		&t.FundName,
		&dateStr,
		&t.Type,
		&t.Shares,
		&t.CostPerShare,
		&ibkrTransactionIDStr,
		&t.IbkrLinked,
	)
	if err == sql.ErrNoRows {
		return model.TransactionResponse{}, apperrors.ErrTransactionNotFound
	}

	if err != nil {
		return t, fmt.Errorf("failed to scan transaction table results: %w", err)
	}
	t.Date, err = ParseTime(dateStr)
	if err != nil || t.Date.IsZero() {
		return t, fmt.Errorf("failed to parse date: %w", err)
	}

	if ibkrTransactionIDStr.Valid {
		t.IbkrTransactionID = ibkrTransactionIDStr.String
	}

	return t, nil
}

// GetTransactionByID retrieves a single transaction by its ID without enrichment.
// Returns the basic transaction model for internal service operations (CUD operations).
// For enriched data with fund names and IBKR linkage, use GetTransaction instead.
//
// Returns ErrTransactionNotFound if the transaction does not exist.
// Returns an error if the query fails or date parsing fails.
func (r *TransactionRepository) GetTransactionByID(transactionID string) (model.Transaction, error) {
	txnLog.Debug("getting transaction by ID", "transaction_id", transactionID)
	query := `
          SELECT id, portfolio_fund_id, date, type, shares, cost_per_share, created_at
          FROM "transaction"
          WHERE id = ?
      `
	var t model.Transaction
	var dateStr, createdAtStr string

	err := r.getQuerier().QueryRow(query, transactionID).Scan(
		&t.ID,
		&t.PortfolioFundID,
		&dateStr,
		&t.Type,
		&t.Shares,
		&t.CostPerShare,
		&createdAtStr,
	)

	if err == sql.ErrNoRows {
		return model.Transaction{}, apperrors.ErrTransactionNotFound
	}
	if err != nil {
		return t, fmt.Errorf("failed to get transaction: %w", err)
	}

	t.Date, err = ParseTime(dateStr)
	if err != nil || t.Date.IsZero() {
		return t, fmt.Errorf("failed to parse date: %w", err)
	}

	t.CreatedAt, err = ParseTime(createdAtStr)
	if err != nil || t.CreatedAt.IsZero() {
		return t, fmt.Errorf("failed to parse created_at: %w", err)
	}

	return t, nil
}

// InsertTransaction creates a new transaction record in the database.
// All transaction fields including ID must be set before calling this method.
//
// Returns an error if the insert operation fails.
func (r *TransactionRepository) InsertTransaction(ctx context.Context, t *model.Transaction) error {
	txnLog.DebugContext(ctx, "inserting transaction", "transaction_id", t.ID, "portfolio_fund_id", t.PortfolioFundID, "type", t.Type)
	query := `
        INSERT INTO "transaction" (id, portfolio_fund_id, date, type, shares, cost_per_share, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		t.ID,
		t.PortfolioFundID,
		t.Date.Format("2006-01-02"),
		t.Type,
		t.Shares,
		t.CostPerShare,
		t.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	return nil
}

// UpdateTransaction updates an existing transaction record in the database.
// All fields in the provided transaction will be written to the database.
//
// Returns ErrTransactionNotFound if no transaction exists with the given ID.
// Returns an error if the update operation fails.
func (r *TransactionRepository) UpdateTransaction(ctx context.Context, t *model.Transaction) error {
	txnLog.DebugContext(ctx, "updating transaction", "transaction_id", t.ID)
	query := `
        UPDATE "transaction"
        SET portfolio_fund_id = ?, date = ?, type = ?, shares = ?, cost_per_share = ?, created_at = ?
        WHERE id = ?
    `

	result, err := r.getQuerier().ExecContext(ctx, query,
		t.PortfolioFundID,
		t.Date.Format("2006-01-02"),
		t.Type,
		t.Shares,
		t.CostPerShare,
		t.CreatedAt.Format("2006-01-02 15:04:05"),
		t.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrTransactionNotFound
	}

	return nil
}

// DeleteTransaction removes a transaction record from the database.
//
// Returns ErrTransactionNotFound if no transaction exists with the given ID.
// Returns an error if the delete operation fails.
func (r *TransactionRepository) DeleteTransaction(ctx context.Context, transactionID string) error {
	txnLog.DebugContext(ctx, "deleting transaction", "transaction_id", transactionID)
	query := `DELETE FROM "transaction" WHERE id = ?`

	result, err := r.getQuerier().ExecContext(ctx, query, transactionID)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrTransactionNotFound
	}

	return nil
}

// GetSharesOnDate calculates the total shares held for a portfolio fund as of the given date.
// Aggregates all transactions up to and including the date using a SQL SUM:
//   - "buy" and "dividend" transactions add shares
//   - "sell" transactions subtract shares
//
// Returns 0.0 if no transactions exist for the given portfolio fund up to the date.
// Returns ErrInvalidPortfolioID if portfolioFundID is empty.
func (r *TransactionRepository) GetSharesOnDate(portfolioFundID string, date time.Time) (float64, error) {
	txnLog.Debug("getting shares on date", "portfolio_fund_id", portfolioFundID, "date", date.Format("2006-01-02"))

	if portfolioFundID == "" {
		return 0.0, apperrors.ErrInvalidPortfolioID
	}

	query := `
		SELECT COALESCE(SUM(CASE
			WHEN type IN ('buy', 'dividend') THEN shares
			WHEN type = 'sell'  THEN -shares
			ELSE 0
		END), 0)
		FROM "transaction"
		WHERE portfolio_fund_id = ?
		AND date <= ?
	`

	var f float64

	err := r.getQuerier().QueryRow(query, portfolioFundID, date.Format("2006-01-02")).Scan(
		&f,
	)
	if err != nil {
		return 0.0, fmt.Errorf("failed to query transaction table: %w", err)
	}

	return f, nil
}

// GetTransactionsByPortfolioFundID retrieves all transactions for a portfolio fund, ordered by date ascending.
// Used for calculating current position (shares held, cost basis, average cost) via weighted average.
func (r *TransactionRepository) GetTransactionsByPortfolioFundID(pfID string) ([]model.Transaction, error) {
	txnLog.Debug("getting transactions by portfolio fund ID", "portfolio_fund_id", pfID)
	if pfID == "" {
		return nil, apperrors.ErrInvalidPortfolioID
	}

	query := `
		SELECT id, portfolio_fund_id, date, type, shares, cost_per_share, created_at
		FROM "transaction"
		WHERE portfolio_fund_id = ?
		ORDER BY date ASC
	`

	rows, err := r.getQuerier().Query(query, pfID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions by portfolio fund: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction

	for rows.Next() {
		var dateStr, createdAtStr string
		var t model.Transaction

		err := rows.Scan(
			&t.ID,
			&t.PortfolioFundID,
			&dateStr,
			&t.Type,
			&t.Shares,
			&t.CostPerShare,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		t.Date, err = ParseTime(dateStr)
		if err != nil || t.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || t.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		transactions = append(transactions, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}
