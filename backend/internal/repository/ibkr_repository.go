package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

var ibkrLog = logging.NewLogger("ibkr")

// IbkrRepository provides data access methods for the ibkr table.
// It handles retrieving ibkr information.
type IbkrRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewIbkrRepository creates a new IbkrRepository with the provided database connection.
func NewIbkrRepository(db *sql.DB) *IbkrRepository {
	return &IbkrRepository{db: db}
}

// WithTx returns a new IbkrRepository scoped to the provided transaction.
func (r *IbkrRepository) WithTx(tx *sql.Tx) *IbkrRepository {
	return &IbkrRepository{
		db: r.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (r *IbkrRepository) getQuerier() Querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetIbkrConfig retrieves the IBKR integration configuration from the database.
// Returns a config with Configured=false if no configuration exists.
// Parses nullable fields (token expiration, last import date, default allocations) safely.
func (r *IbkrRepository) GetIbkrConfig() (*model.IbkrConfig, error) {
	ibkrLog.Debug("getting ibkr config")
	query := `
        SELECT id, flex_token, flex_query_id, token_expires_at, last_import_date, auto_import_enabled, created_at, updated_at, enabled, default_allocation_enabled, default_allocations
		FROM ibkr_config
      `

	var ic model.IbkrConfig
	var tokenExpiresStr, lastImportStr, defaultAllocationStr sql.NullString
	err := r.getQuerier().QueryRow(query).Scan(
		&ic.ID,
		&ic.FlexToken,
		&ic.FlexQueryID,
		&tokenExpiresStr,
		&lastImportStr,
		&ic.AutoImportEnabled,
		&ic.CreatedAt,
		&ic.UpdatedAt,
		&ic.Enabled,
		&ic.DefaultAllocationEnabled,
		&defaultAllocationStr,
	)
	if err == sql.ErrNoRows {
		return &model.IbkrConfig{}, apperrors.ErrIbkrConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ibkr_config: %w", err)
	}

	// Config exists in database
	ic.Configured = true

	if tokenExpiresStr.Valid {
		t, err := ParseTime(tokenExpiresStr.String)
		if err != nil || t.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date on TokenExpiresAt: %w", err)
		}
		ic.TokenExpiresAt = &t
	}

	if lastImportStr.Valid {
		l, err := ParseTime(lastImportStr.String)
		if err != nil || l.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date on LastImportDate: %w", err)
		}
		ic.LastImportDate = &l
	}

	if defaultAllocationStr.Valid {
		var defaultAllocation []model.Allocation
		if err := json.Unmarshal([]byte(defaultAllocationStr.String), &defaultAllocation); err != nil {
			ibkrLog.Warn("failed to parse allocation model", "error", err)
			// Continue without allocations
		} else {
			ic.DefaultAllocations = defaultAllocation
		}
	}

	return &ic, nil
}

// GetPendingDividends retrieves dividend records with reinvestment_status = 'PENDING'.
// Optionally filters by fund symbol or ISIN by joining with the fund table.
// Returns an empty slice if no pending dividends match the criteria.
func (r *IbkrRepository) GetPendingDividends(symbol, isin string) ([]model.PendingDividend, error) {
	ibkrLog.Debug("getting pending dividends", "symbol", symbol, "isin", isin)

	var query string
	var args []any
	if symbol != "" || isin != "" {
		query = `
			SELECT d.id, d.fund_id, d.portfolio_fund_id, d.record_date, d.ex_dividend_date, d.shares_owned,
				d.dividend_per_share, d.total_amount
			FROM dividend d
			INNER JOIN fund f
			ON f.id = d.fund_id
			WHERE reinvestment_status = 'PENDING'
		`

		if symbol != "" && isin != "" {
			query += " AND (f.symbol = ? OR f.isin = ?)"
			args = append(args, symbol, isin)
		} else if symbol != "" {
			query += " AND f.symbol = ?"
			args = append(args, symbol)
		} else if isin != "" {
			query += " AND f.isin = ?"
			args = append(args, isin)
		}

		query += `
			ORDER BY ex_dividend_date ASC
		`

	} else {
		query = `
			SELECT id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned,
			dividend_per_share, total_amount
			FROM dividend
			WHERE reinvestment_status = 'PENDING'
			ORDER BY ex_dividend_date ASC
		`
	}

	rows, err := r.getQuerier().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	dividend := []model.PendingDividend{}

	for rows.Next() {
		var recordDateStr, exDividendStr string
		var t model.PendingDividend

		err := rows.Scan(
			&t.ID,
			&t.FundID,
			&t.PortfolioFundID,
			&recordDateStr,
			&exDividendStr,
			&t.SharesOwned,
			&t.DividendPerShare,
			&t.TotalAmount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dividend table results: %w", err)
		}

		t.RecordDate, err = ParseTime(recordDateStr)
		if err != nil || t.RecordDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.ExDividendDate, err = ParseTime(exDividendStr)
		if err != nil || t.ExDividendDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		dividend = append(dividend, t)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dividend table: %w", err)
	}

	return dividend, nil

}

// GetInbox retrieves IBKR imported transactions from the ibkr_transaction table.
// Filters by status (defaults to "pending" if not provided) and optionally by transaction_type.
// Returns transactions ordered by transaction_date descending.
// Returns an empty slice if no transactions match the criteria.
func (r *IbkrRepository) GetInbox(status, transactionType string) ([]model.IBKRTransaction, error) {
	ibkrLog.Debug("getting ibkr inbox", "status", status, "transaction_type", transactionType)
	var query string
	var args []any

	query = `
	SELECT id, ibkr_transaction_id, transaction_date, symbol, isin, description,
         transaction_type, quantity, price, total_amount, currency, fees,
         status, imported_at, report_date, notes
	FROM ibkr_transaction
	WHERE status = ?
  `
	if status == "" {
		args = append(args, "pending")
	} else {
		args = append(args, status)
	}
	if transactionType != "" {
		query += `
			AND transaction_type = ?
		`
		args = append(args, transactionType)
	}

	query += `
		ORDER BY transaction_date DESC
	`
	rows, err := r.getQuerier().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query IBKR Transactions table: %w", err)
	}
	defer rows.Close()

	ibkrTransactions := []model.IBKRTransaction{}

	for rows.Next() {
		var transactionDateStr, importedAtStr, reportDateStr string
		t := model.IBKRTransaction{}
		err := rows.Scan(
			&t.ID,
			&t.IBKRTransactionID,
			&transactionDateStr,
			&t.Symbol,
			&t.ISIN,
			&t.Description,
			&t.TransactionType,
			&t.Quantity,
			&t.Price,
			&t.TotalAmount,
			&t.Currency,
			&t.Fees,
			&t.Status,
			&importedAtStr,
			&reportDateStr,
			&t.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan IBKR Transactions table results: %w", err)
		}

		t.TransactionDate, err = ParseTime(transactionDateStr)
		if err != nil || t.TransactionDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.ImportedAt, err = ParseTime(importedAtStr)
		if err != nil || t.ImportedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.ReportDate, err = ParseTime(reportDateStr)
		if err != nil || t.ReportDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		ibkrTransactions = append(ibkrTransactions, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating IBKR Transactions table: %w", err)
	}

	return ibkrTransactions, nil
}

// GetIbkrInboxCount retrieves the count of IBKR imported transactions.
// Uses a COUNT(*) query for efficiency rather than fetching all records.
// Returns 0 if no transactions exist.
func (r *IbkrRepository) GetIbkrInboxCount() (model.IBKRInboxCount, error) {
	ibkrLog.Debug("getting ibkr inbox count")

	query := `
        SELECT count(*)
		FROM ibkr_transaction
		WHERE status = 'pending'
      `

	count := model.IBKRInboxCount{}
	err := r.getQuerier().QueryRow(query).Scan(&count.Count)
	if err == sql.ErrNoRows {
		return model.IBKRInboxCount{
			Count: 0,
		}, nil
	}
	if err != nil {
		return model.IBKRInboxCount{}, fmt.Errorf("failed to query ibkr inbox count: %w", err)
	}

	return count, nil
}

// GetIbkrTransaction retrieves a single IBKR transaction by its ID.
// Returns ErrIBKRTransactionNotFound if the transaction does not exist.
//
// Parameters:
//   - transactionID: The UUID of the IBKR transaction to retrieve
//
// Returns the transaction details or an error if not found or database error occurs.
func (r *IbkrRepository) GetIbkrTransaction(transactionID string) (model.IBKRTransaction, error) {
	ibkrLog.Debug("getting ibkr transaction", "transaction_id", transactionID)

	query := `
        SELECT id, ibkr_transaction_id, transaction_date, symbol, isin, description, transaction_type, quantity, price, total_amount, currency, fees, status, imported_at, processed_at, report_date, notes
		FROM ibkr_transaction
		WHERE id = ?
      `

	t := model.IBKRTransaction{}
	var transactionDateStr, importedAtStr, reportDateStr string
	var proccessedDateStr sql.NullString

	err := r.getQuerier().QueryRow(query, transactionID).Scan(
		&t.ID,
		&t.IBKRTransactionID,
		&transactionDateStr,
		&t.Symbol,
		&t.ISIN,
		&t.Description,
		&t.TransactionType,
		&t.Quantity,
		&t.Price,
		&t.TotalAmount,
		&t.Currency,
		&t.Fees,
		&t.Status,
		&importedAtStr,
		&proccessedDateStr,
		&reportDateStr,
		&t.Notes)
	if err == sql.ErrNoRows {
		return model.IBKRTransaction{}, apperrors.ErrIBKRTransactionNotFound
	}
	if err != nil {
		return model.IBKRTransaction{}, fmt.Errorf("failed to query ibkr transaction: %w", err)
	}

	t.TransactionDate, err = ParseTime(transactionDateStr)
	if err != nil || t.TransactionDate.IsZero() {
		return model.IBKRTransaction{}, fmt.Errorf("failed to parse date: %w", err)
	}

	t.ImportedAt, err = ParseTime(importedAtStr)
	if err != nil || t.ImportedAt.IsZero() {
		return model.IBKRTransaction{}, fmt.Errorf("failed to parse date: %w", err)
	}

	t.ReportDate, err = ParseTime(reportDateStr)
	if err != nil || t.ReportDate.IsZero() {
		return model.IBKRTransaction{}, fmt.Errorf("failed to parse date: %w", err)
	}

	// BuyOrderDate is nullable
	if proccessedDateStr.Valid {
		parsed, err := ParseTime(proccessedDateStr.String)
		if err != nil || parsed.IsZero() {
			return model.IBKRTransaction{}, fmt.Errorf("failed to parse buy_order_date: %w", err)
		}
		t.ProcessedAt = &parsed
	}

	return t, nil
}

// CompareIbkrTransaction reports whether a transaction already exists in the database by ibkr_transaction_id.
// Returns true if the transaction exists or if a query error occurs.
// Intentional design: on DB error, we treat the transaction as already existing (fail-safe) to avoid duplicate
// inserts. The unique constraint on ibkr_transaction_id will catch any actual duplicates, and a future
// logging/alerting system is expected to surface DB errors from that layer.
func (r *IbkrRepository) CompareIbkrTransaction(t model.IBKRTransaction) bool {
	ibkrLog.Debug("comparing ibkr transaction", "ibkr_transaction_id", t.IBKRTransactionID)

	query := `
        SELECT count(*)
		FROM ibkr_transaction
		WHERE ibkr_transaction_id = ?
      `

	var count int
	err := r.getQuerier().QueryRow(query,
		t.IBKRTransactionID,
	).Scan(&count)

	if err != nil {
		ibkrLog.Warn("query error, treating as existing", "ibkr_transaction_id", t.IBKRTransactionID, "error", err)
		return true
	}

	return count > 0
}

// GetIbkrTransactionAllocations retrieves all allocation records for a specific IBKR transaction.
// Joins with portfolio and transaction tables to include portfolio names and transaction types.
// Returns allocations for both the main transaction and any associated fee transactions.
//
// Parameters:
//   - IBKRtransactionID: The UUID of the IBKR transaction
//
// Returns a slice of allocations with full details including portfolio names and transaction types,
// or an error if the database query fails.
func (r *IbkrRepository) GetIbkrTransactionAllocations(IBKRtransactionID string) ([]model.IBKRTransactionAllocation, error) {
	ibkrLog.Debug("getting ibkr transaction allocations", "ibkr_transaction_id", IBKRtransactionID)

	query := `
        SELECT i.id, i.ibkr_transaction_id, i.portfolio_id, p.name, i.allocation_percentage, i.allocated_amount, i.allocated_shares, i.transaction_id, t.type, i.created_at
		FROM ibkr_transaction_allocation i
		INNER JOIN portfolio p
		ON i.portfolio_id = p.id
		LEFT JOIN "transaction" t
		on i.transaction_id = t.id
		WHERE ibkr_transaction_id = ?
      `

	rows, err := r.getQuerier().Query(query, IBKRtransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ibkr_transaction table: %w", err)
	}
	defer rows.Close()

	allocations := []model.IBKRTransactionAllocation{}

	for rows.Next() {
		var TransactionIDStr sql.NullString
		var CreatedAtStr string
		var ta model.IBKRTransactionAllocation

		err := rows.Scan(
			&ta.ID,
			&ta.IBKRTransactionID,
			&ta.PortfolioID,
			&ta.PortfolioName,
			&ta.AllocationPercentage,
			&ta.AllocatedAmount,
			&ta.AllocatedShares,
			&TransactionIDStr,
			&ta.Type,
			&CreatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ibkr_transaction table results: %w", err)
		}

		if TransactionIDStr.Valid {
			ta.TransactionID = TransactionIDStr.String
		}

		ta.CreatedAt, err = ParseTime(CreatedAtStr)
		if err != nil || ta.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		allocations = append(allocations, ta)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ibkr_transaction table: %w", err)
	}

	return allocations, nil

}

// AddIbkrTransactions inserts a batch of IBKR transactions using a prepared statement.
// Returns nil immediately if the slice is empty.
// Returns an error if any individual insert fails.
func (r *IbkrRepository) AddIbkrTransactions(ctx context.Context, transactions []model.IBKRTransaction) error {
	ibkrLog.DebugContext(ctx, "adding ibkr transactions", "count", len(transactions))
	if len(transactions) == 0 {
		return nil
	}

	stmt, err := r.getQuerier().PrepareContext(ctx, `
        INSERT INTO ibkr_transaction (id, ibkr_transaction_id, transaction_date, symbol, isin, description, transaction_type, quantity, price, total_amount, currency, fees, status, imported_at, processed_at, raw_data, report_date, notes)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, t := range transactions {

		var processedAt sql.NullString
		if t.ProcessedAt != nil {
			processedAt.String = t.ProcessedAt.Format("2006-01-02 15:04:05")
			processedAt.Valid = true
		}
		_, err := stmt.ExecContext(ctx,
			t.ID,
			t.IBKRTransactionID,
			t.TransactionDate.Format("2006-01-02"),
			t.Symbol,
			t.ISIN,
			t.Description,
			t.TransactionType,
			t.Quantity,
			t.Price,
			t.TotalAmount,
			t.Currency,
			t.Fees,
			t.Status,
			t.ImportedAt.Format("2006-01-02 15:04:05"),
			processedAt,
			t.RawData,
			t.ReportDate.Format("2006-01-02"),
			t.Notes,
		)
		if err != nil {
			return fmt.Errorf("failed to insert IBKR Transaction for %s on %s: %w", t.IBKRTransactionID, t.TransactionDate.Format("2006-01-02"), err)
		}
	}
	return nil
}

// WriteImportCache stores a Flex report XML payload in the import cache.
// Uses INSERT OR REPLACE on the cache_key unique constraint, so repeated writes
// for the same query and date are safe and do not affect other cache entries.
func (r *IbkrRepository) WriteImportCache(ctx context.Context, t model.IbkrImportCache) error {
	ibkrLog.DebugContext(ctx, "writing import cache", "cache_key", t.CacheKey)

	query := `
		INSERT OR REPLACE INTO ibkr_import_cache (id, cache_key, data, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.getQuerier().ExecContext(ctx, query,
		t.ID,
		t.CacheKey,
		t.Data,
		t.CreatedAt.Format("2006-01-02 15:04:05"),
		t.ExpiresAt.Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return fmt.Errorf("failed to set ibkr_import_cache: %w", err)
	}

	return nil
}

// UpdateLastImportDate sets the last_import_date on the config row identified by queryID.
// Scoped to a specific flex_query_id to support multiple flex queries in the future.
func (r *IbkrRepository) UpdateLastImportDate(ctx context.Context, queryID string, t time.Time) error {
	ibkrLog.DebugContext(ctx, "updating last import date", "query_id", queryID, "date", t.Format("2006-01-02 15:04:05"))
	query := `UPDATE ibkr_config SET last_import_date = ? WHERE flex_query_id = ?`
	_, err := r.getQuerier().ExecContext(ctx, query, t.Format("2006-01-02 15:04:05"), queryID)
	if err != nil {
		return fmt.Errorf("failed to update last_import_date: %w", err)
	}
	return nil
}

// UpdateIbkrConfig persists the IBKR config using INSERT OR REPLACE.
// The table holds exactly one row. When overwriteConfig is true the existing row is deleted first,
// which is necessary when flex_query_id changes (SQLite REPLACE would leave a stale row behind
// because the PRIMARY KEY is the config ID, not the query ID).
// All fields on c must be fully populated before calling; the service layer is responsible for
// merging the request onto the existing config before invoking this method.
func (r *IbkrRepository) UpdateIbkrConfig(ctx context.Context, overwriteConfig bool, c *model.IbkrConfig) error {
	ibkrLog.DebugContext(ctx, "updating ibkr config", "overwrite", overwriteConfig)

	if overwriteConfig {
		query := `DELETE FROM ibkr_config`

		_, err := r.getQuerier().ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to delete ibkr config: %w", err)
		}
	}

	query := `
        INSERT OR REPLACE INTO ibkr_config (id, flex_token, flex_query_id, token_expires_at, last_import_date, auto_import_enabled,
	created_at, updated_at, enabled, default_allocation_enabled, default_allocations)
	values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	var tokenExpiresStr, lastImportStr sql.NullString
	if c.TokenExpiresAt != nil {
		tokenExpiresStr.String = c.TokenExpiresAt.Format("2006-01-02")
	}
	if c.LastImportDate != nil {
		lastImportStr.String = c.LastImportDate.Format("2006-01-02 15:04:05")
	}

	var defaultAllocationsStr []byte
	if len(c.DefaultAllocations) > 0 {
		var err error
		defaultAllocationsStr, err = json.Marshal(c.DefaultAllocations)
		if err != nil {
			return fmt.Errorf("failed to marshal default allocations: %w", err)
		}
	}

	_, err := r.getQuerier().ExecContext(ctx, query,
		c.ID,
		c.FlexToken,
		c.FlexQueryID,
		tokenExpiresStr,
		lastImportStr,
		c.AutoImportEnabled,
		c.CreatedAt.Format("2006-01-02 15:04:05"),
		c.UpdatedAt.Format("2006-01-02 15:04:05"),
		c.Enabled,
		c.DefaultAllocationEnabled,
		defaultAllocationsStr,
	)

	if err != nil {
		return fmt.Errorf("failed to update ibkr config: %w", err)
	}

	return nil
}

// GetIbkrImportCache retrieves the most recent cached Flex report from the database.
// Returns ErrIbkrImportCacheNotFound if the cache is empty.
// The caller should check ExpiresAt to determine whether the cached data is still valid.
func (r *IbkrRepository) GetIbkrImportCache() (model.IbkrImportCache, error) {
	ibkrLog.Debug("getting ibkr import cache")

	query := `
		SELECT id, cache_key, data, created_at, expires_at
		FROM ibkr_import_cache
		ORDER BY created_at DESC
		LIMIT 1
	`

	var c model.IbkrImportCache
	var createdAtStr, expiresAtStr string
	err := r.getQuerier().QueryRow(query).Scan(
		&c.ID,
		&c.CacheKey,
		&c.Data,
		&createdAtStr,
		&expiresAtStr,
	)
	if err == sql.ErrNoRows {
		return model.IbkrImportCache{}, apperrors.ErrIbkrImportCacheNotFound
	}

	if err != nil {
		return model.IbkrImportCache{}, fmt.Errorf("failed to query ibkr import cache: %w", err)
	}

	c.CreatedAt, err = ParseTime(createdAtStr)
	if err != nil || c.CreatedAt.IsZero() {
		return model.IbkrImportCache{}, fmt.Errorf("failed to parse date: %w", err)
	}

	c.ExpiresAt, err = ParseTime(expiresAtStr)
	if err != nil || c.ExpiresAt.IsZero() {
		return model.IbkrImportCache{}, fmt.Errorf("failed to parse date: %w", err)
	}

	return c, nil
}

// UpdateIbkrTransactionStatus updates the status and optional processed_at timestamp of an IBKR transaction.
// Returns ErrIBKRTransactionNotFound if no transaction with the given ID exists.
func (r *IbkrRepository) UpdateIbkrTransactionStatus(ctx context.Context, id, status string, processedAt *time.Time) error {
	ibkrLog.DebugContext(ctx, "updating ibkr transaction status", "transaction_id", id, "status", status)
	var processedAtStr any
	if processedAt != nil {
		processedAtStr = processedAt.Format("2006-01-02 15:04:05")
	}

	query := `UPDATE ibkr_transaction SET status = ?, processed_at = ? WHERE id = ?`
	result, err := r.getQuerier().ExecContext(ctx, query, status, processedAtStr, id)
	if err != nil {
		return fmt.Errorf("failed to update ibkr transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return apperrors.ErrIBKRTransactionNotFound
	}

	return nil
}

// DeleteIbkrTransaction removes an IBKR transaction by its ID.
// Returns ErrIBKRTransactionNotFound if no transaction with the given ID exists.
func (r *IbkrRepository) DeleteIbkrTransaction(ctx context.Context, id string) error {
	ibkrLog.DebugContext(ctx, "deleting ibkr transaction", "transaction_id", id)
	query := `DELETE FROM ibkr_transaction WHERE id = ?`
	result, err := r.getQuerier().ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete ibkr transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return apperrors.ErrIBKRTransactionNotFound
	}

	return nil
}

// InsertIbkrTransactionAllocation creates a new IBKR transaction allocation record.
func (r *IbkrRepository) InsertIbkrTransactionAllocation(ctx context.Context, a model.IBKRTransactionAllocation) error {
	ibkrLog.DebugContext(ctx, "inserting ibkr transaction allocation", "ibkr_transaction_id", a.IBKRTransactionID, "portfolio_id", a.PortfolioID)
	query := `
		INSERT INTO ibkr_transaction_allocation (id, ibkr_transaction_id, portfolio_id, allocation_percentage, allocated_amount, allocated_shares, transaction_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var transactionID sql.NullString
	if a.TransactionID != "" {
		transactionID = sql.NullString{String: a.TransactionID, Valid: true}
	}

	_, err := r.getQuerier().ExecContext(ctx, query,
		a.ID,
		a.IBKRTransactionID,
		a.PortfolioID,
		a.AllocationPercentage,
		a.AllocatedAmount,
		a.AllocatedShares,
		transactionID,
		a.CreatedAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return fmt.Errorf("failed to insert ibkr transaction allocation: %w", err)
	}

	return nil
}

// DeleteIbkrTransactionAllocations removes all allocation records for a given IBKR transaction.
// Returns ErrIBKRTransactionNotFound if no allocations exist for the given transaction.
func (r *IbkrRepository) DeleteIbkrTransactionAllocations(ctx context.Context, ibkrTransactionID string) error {
	ibkrLog.DebugContext(ctx, "deleting ibkr transaction allocations", "ibkr_transaction_id", ibkrTransactionID)
	query := `DELETE FROM ibkr_transaction_allocation WHERE ibkr_transaction_id = ?`
	result, err := r.getQuerier().ExecContext(ctx, query, ibkrTransactionID)
	if err != nil {
		return fmt.Errorf("failed to delete ibkr transaction allocations: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return apperrors.ErrIBKRTransactionNotFound
	}

	return nil
}

// GetTransactionIDsByIbkrTransaction retrieves all transaction IDs linked to an IBKR transaction via allocations.
// Used during unallocation to clean up the created Transaction records.
func (r *IbkrRepository) GetTransactionIDsByIbkrTransaction(ctx context.Context, ibkrTransactionID string) ([]string, error) {
	ibkrLog.DebugContext(ctx, "getting transaction IDs by ibkr transaction", "ibkr_transaction_id", ibkrTransactionID)
	query := `
		SELECT transaction_id FROM ibkr_transaction_allocation
		WHERE ibkr_transaction_id = ? AND transaction_id IS NOT NULL
	`

	rows, err := r.getQuerier().QueryContext(ctx, query, ibkrTransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction IDs by ibkr transaction: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan transaction ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction IDs: %w", err)
	}

	return ids, nil
}

// GetIbkrTransactionIDByTransactionID finds the IBKR transaction ID linked to a given transaction
// via the allocation table. Returns empty string if no allocation exists (not an error).
func (r *IbkrRepository) GetIbkrTransactionIDByTransactionID(ctx context.Context, transactionID string) (string, error) {
	ibkrLog.DebugContext(ctx, "getting ibkr transaction ID by transaction ID", "transaction_id", transactionID)
	query := `SELECT ibkr_transaction_id FROM ibkr_transaction_allocation WHERE transaction_id = ?`

	var ibkrTxnID string
	err := r.getQuerier().QueryRowContext(ctx, query, transactionID).Scan(&ibkrTxnID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query ibkr allocation by transaction ID: %w", err)
	}

	return ibkrTxnID, nil
}

// CountIbkrAllocationsByIbkrTransactionID counts how many allocation records exist for a given IBKR transaction.
// Used to determine whether reverting status to "pending" is needed when deleting the last allocation.
func (r *IbkrRepository) CountIbkrAllocationsByIbkrTransactionID(ctx context.Context, ibkrTransactionID string) (int, error) {
	ibkrLog.DebugContext(ctx, "counting ibkr allocations", "ibkr_transaction_id", ibkrTransactionID)
	query := `SELECT COUNT(*) FROM ibkr_transaction_allocation WHERE ibkr_transaction_id = ?`

	var count int
	err := r.getQuerier().QueryRowContext(ctx, query, ibkrTransactionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count ibkr allocations: %w", err)
	}

	return count, nil
}

// DeleteIbkrConfig removes the IBKR configuration row from the database.
// Returns ErrIbkrConfigNotFound if no config exists.
func (r *IbkrRepository) DeleteIbkrConfig(ctx context.Context) error {
	ibkrLog.DebugContext(ctx, "deleting ibkr config")
	query := `DELETE FROM ibkr_config`

	result, err := r.getQuerier().ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete ibkr config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrIbkrConfigNotFound
	}

	return nil
}
