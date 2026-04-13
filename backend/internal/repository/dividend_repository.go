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

var dividendLog = logging.NewLogger("dividend")

// DividendRepository provides data access methods for the dividend table.
// It handles retrieving dividend records and reinvestment information.
type DividendRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewDividendRepository creates a new DividendRepository with the provided database connection.
func NewDividendRepository(db *sql.DB) *DividendRepository {
	return &DividendRepository{db: db}
}

// WithTx returns a new DividendRepository scoped to the provided transaction.
func (r *DividendRepository) WithTx(tx *sql.Tx) *DividendRepository {
	return &DividendRepository{
		db: r.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (r *DividendRepository) getQuerier() Querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetAllDividend retrieves all dividends.
// Dividends are filtered by ex-dividend date and sorted in ascending order by that date.
//
// Returns []Dividend.
// Handles nullable fields like buy_order_date and reinvestment_transaction_id appropriately.
// This grouping allows callers to decide how to aggregate (by portfolio, by fund, etc.) after retrieval.
func (r *DividendRepository) GetAllDividend() ([]model.Dividend, error) {
	dividendLog.Debug("getting all dividends")

	// Retrieve all dividend based on returned portfolio_fund IDs
	dividendQuery := `
		SELECT id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned,
		dividend_per_share, total_amount, reinvestment_status, buy_order_date, reinvestment_transaction_id, created_at
		FROM dividend
		ORDER BY ex_dividend_date ASC
	`

	rows, err := r.db.Query(dividendQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	dividend := []model.Dividend{}

	for rows.Next() {
		var recordDateStr, exDividendStr, createdAtStr string
		var buyOrderStr, reinvestmentTxID sql.NullString
		var t model.Dividend

		err := rows.Scan(
			&t.ID,
			&t.FundID,
			&t.PortfolioFundID,
			&recordDateStr,
			&exDividendStr,
			&t.SharesOwned,
			&t.DividendPerShare,
			&t.TotalAmount,
			&t.ReinvestmentStatus,
			&buyOrderStr,
			&reinvestmentTxID,
			&createdAtStr,
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

		// BuyOrderDate is nullable
		if buyOrderStr.Valid {
			t.BuyOrderDate, err = ParseTime(buyOrderStr.String)
			if err != nil || t.BuyOrderDate.IsZero() {
				return nil, fmt.Errorf("failed to parse buy_order_date: %w", err)
			}
		}

		// ReinvestmentTransactionID is nullable
		if reinvestmentTxID.Valid {
			t.ReinvestmentTransactionID = reinvestmentTxID.String
		}

		t.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || t.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		dividend = append(dividend, t)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dividend table: %w", err)
	}

	return dividend, nil
}

// GetDividendPerPF retrieves all dividends for the given portfolio_fund IDs within the specified date range.
// Dividends are filtered by ex-dividend date and sorted in ascending order by that date.
//
// The method performs the following:
//  1. Builds a parameterized query with placeholders for portfolio_fund IDs
//  2. Executes the query and iterates through results
//  3. Scans each row into a Dividend struct
//  4. Delegates date parsing and nullable field handling to parseDividendRecords
//  5. Groups dividends by portfolio_fund_id
//
// Date parsing is extracted to parseDividendRecords to reduce cyclomatic complexity.
//
// Parameters:
//   - pfIDs: Slice of portfolio_fund IDs to query
//   - startDate: Inclusive start date for the query (compared against ex_dividend_date)
//   - endDate: Inclusive end date for the query (compared against ex_dividend_date)
//
// Returns a map of portfolioFundID -> []Dividend, grouped for efficient per-fund calculations.
// If pfIDs is empty, returns an empty map. The grouping allows callers to aggregate by
// portfolio, fund, or other dimensions after retrieval.
func (r *DividendRepository) GetDividendPerPF(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	dividendLog.Debug("getting dividends per portfolio fund", "pf_count", len(pfIDs), "start_date", startDate.Format("2006-01-02"), "end_date", endDate.Format("2006-01-02"))
	if len(pfIDs) == 0 {
		return make(map[string][]model.Dividend), nil
	}

	dividendPlaceholders := make([]string, len(pfIDs))
	for i := range dividendPlaceholders {
		dividendPlaceholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	dividendQuery := `
		SELECT id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned,
		dividend_per_share, total_amount, reinvestment_status, buy_order_date, reinvestment_transaction_id, created_at
		FROM dividend
		WHERE portfolio_fund_id IN (` + strings.Join(dividendPlaceholders, ",") + `)
		AND ex_dividend_date >= ?
		AND ex_dividend_date <= ?
		ORDER BY ex_dividend_date ASC
	`

	dividendArgs := make([]any, 0, len(pfIDs)+2)
	for _, id := range pfIDs {
		dividendArgs = append(dividendArgs, id)
	}
	dividendArgs = append(dividendArgs, startDate.Format("2006-01-02"))
	dividendArgs = append(dividendArgs, endDate.Format("2006-01-02"))

	rows, err := r.db.Query(dividendQuery, dividendArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	dividend := make(map[string][]model.Dividend)

	for rows.Next() {
		var recordDateStr, exDividendStr, createdAtStr string
		var buyOrderStr, reinvestmentTxID sql.NullString
		var t model.Dividend

		err := rows.Scan(
			&t.ID,
			&t.FundID,
			&t.PortfolioFundID,
			&recordDateStr,
			&exDividendStr,
			&t.SharesOwned,
			&t.DividendPerShare,
			&t.TotalAmount,
			&t.ReinvestmentStatus,
			&buyOrderStr,
			&reinvestmentTxID,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dividend table results: %w", err)
		}

		err = r.parseDividendRecords(&t, recordDateStr, exDividendStr, createdAtStr, buyOrderStr, reinvestmentTxID)
		if err != nil {
			return nil, fmt.Errorf("parse dividend records: %w", err)
		}

		dividend[t.PortfolioFundID] = append(dividend[t.PortfolioFundID], t)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dividend table: %w", err)
	}

	return dividend, nil
}

// parseDividendRecords parses date strings and nullable fields from database rows into a Dividend model.
// This helper method was extracted from GetDividendPerPF to reduce cyclomatic complexity by isolating
// the field parsing logic into a dedicated function.
//
// The method handles:
//   - Parsing required date fields (RecordDate, ExDividendDate, CreatedAt)
//   - Parsing optional/nullable date fields (BuyOrderDate)
//   - Extracting nullable string fields (ReinvestmentTransactionID)
//   - Validating that parsed dates are not zero values
//
// Parameters:
//   - t: Pointer to the Dividend struct to populate (modified in-place)
//   - recordDateStr: String representation of the record date (required)
//   - exDividendStr: String representation of the ex-dividend date (required)
//   - createdAtStr: String representation of when the record was created (required)
//   - buyOrderStr: Nullable string for the buy order date
//   - reinvestmentTxID: Nullable string for the reinvestment transaction ID
//
// Returns an error if any required date fails to parse or parses to a zero value.
// Nullable fields are left at their zero values if NULL in the database.
func (r *DividendRepository) parseDividendRecords(t *model.Dividend, recordDateStr, exDividendStr, createdAtStr string, buyOrderStr, reinvestmentTxID sql.NullString) error {
	var err error

	t.RecordDate, err = ParseTime(recordDateStr)
	if err != nil || t.RecordDate.IsZero() {
		return fmt.Errorf("failed to parse date: %w", err)
	}

	t.ExDividendDate, err = ParseTime(exDividendStr)
	if err != nil || t.ExDividendDate.IsZero() {
		return fmt.Errorf("failed to parse date: %w", err)
	}

	// BuyOrderDate is nullable
	if buyOrderStr.Valid {
		t.BuyOrderDate, err = ParseTime(buyOrderStr.String)
		if err != nil || t.BuyOrderDate.IsZero() {
			return fmt.Errorf("failed to parse buy_order_date: %w", err)
		}
	}

	// ReinvestmentTransactionID is nullable
	if reinvestmentTxID.Valid {
		t.ReinvestmentTransactionID = reinvestmentTxID.String
	}

	t.CreatedAt, err = ParseTime(createdAtStr)
	if err != nil || t.CreatedAt.IsZero() {
		return fmt.Errorf("failed to parse date: %w", err)
	}

	return nil
}

// GetDividendPerPortfolioFund retrieves enriched dividend records filtered by either portfolio or fund.
// Performs a JOIN across dividend, portfolio_fund, and fund tables to return DividendFund rows
// including fund names and dividend types. Exactly one of portfolioID or fundID must be non-empty;
// if both are empty an empty slice is returned immediately.
//
// An existence check is performed first to distinguish between:
//   - Entity does not exist → ErrPortfolioNotFound or ErrFundNotFound
//   - Entity exists but has no dividends → empty slice
//
// Parameters:
//   - portfolioID: Filter by portfolio ID (mutually exclusive with fundID)
//   - fundID:      Filter by fund ID (mutually exclusive with portfolioID; checked if portfolioID is empty)
//
// Returns a slice of DividendFund ordered by ex_dividend_date ascending.
func (r *DividendRepository) GetDividendPerPortfolioFund(portfolioID, fundID string) ([]model.DividendFund, error) {
	dividendLog.Debug("getting dividends per portfolio/fund", "portfolio_id", portfolioID, "fund_id", fundID)
	var whereStatement, existsStatement, queryID string
	var notFoundErr error

	if portfolioID != "" {
		whereStatement = "WHERE pf.portfolio_id = ?"
		existsStatement = "SELECT COUNT(*) FROM portfolio_fund WHERE portfolio_id = ?"
		queryID = portfolioID
		notFoundErr = apperrors.ErrPortfolioNotFound
	} else if fundID != "" {
		whereStatement = "WHERE pf.fund_id = ?"
		existsStatement = "SELECT COUNT(*) FROM portfolio_fund WHERE fund_id = ?"
		queryID = fundID
		notFoundErr = apperrors.ErrFundNotFound
	} else {
		return []model.DividendFund{}, nil
	}

	var count int
	if err := r.db.QueryRow(existsStatement, queryID).Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to check existence: %w", err)
	}
	if count == 0 {
		return nil, notFoundErr
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	query := `
	SELECT
		d.id, d.fund_id, f.name, d.portfolio_fund_id, d.record_date, d.ex_dividend_date,
		d.shares_owned, d.dividend_per_share, d.total_amount, d.reinvestment_status,
		d.buy_order_date, d.reinvestment_transaction_id, f.dividend_type
	FROM dividend d
	INNER JOIN portfolio_fund pf ON d.portfolio_fund_id = pf.id
	INNER JOIN fund f ON pf.fund_id = f.id
	` + whereStatement + `
	ORDER BY d.ex_dividend_date ASC
	`

	rows, err := r.db.Query(query, queryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	dividendFund := []model.DividendFund{}
	for rows.Next() {
		t, err := r.scanDividendFundRow(rows)
		if err != nil {
			return nil, err
		}
		dividendFund = append(dividendFund, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dividend table: %w", err)
	}

	return dividendFund, nil
}

// scanDividendFundRow scans a single row from a DividendFund query into a DividendFund model.
// Extracted from GetDividendPerPortfolioFund to reduce cyclomatic complexity.
func (r *DividendRepository) scanDividendFundRow(rows *sql.Rows) (model.DividendFund, error) {
	var recordDateStr, exDividendStr string
	var buyOrderStr, reinvestmentTxID sql.NullString
	var t model.DividendFund

	err := rows.Scan(
		&t.ID,
		&t.FundID,
		&t.FundName,
		&t.PortfolioFundID,
		&recordDateStr,
		&exDividendStr,
		&t.SharesOwned,
		&t.DividendPerShare,
		&t.TotalAmount,
		&t.ReinvestmentStatus,
		&buyOrderStr,
		&reinvestmentTxID,
		&t.DividendType,
	)
	if err != nil {
		return model.DividendFund{}, fmt.Errorf("failed to scan dividend table results: %w", err)
	}

	t.RecordDate, err = ParseTime(recordDateStr)
	if err != nil || t.RecordDate.IsZero() {
		return model.DividendFund{}, fmt.Errorf("failed to parse date: %w", err)
	}

	t.ExDividendDate, err = ParseTime(exDividendStr)
	if err != nil || t.ExDividendDate.IsZero() {
		return model.DividendFund{}, fmt.Errorf("failed to parse date: %w", err)
	}

	if buyOrderStr.Valid {
		buyDate, err := ParseTime(buyOrderStr.String)
		if err != nil || buyDate.IsZero() {
			return model.DividendFund{}, fmt.Errorf("failed to parse buy_order_date: %w", err)
		}
		t.BuyOrderDate = &buyDate
	}

	if reinvestmentTxID.Valid {
		t.ReinvestmentTransactionID = reinvestmentTxID.String
	}

	return t, nil
}

// GetDividend retrieves a single dividend record by ID.
// Returns ErrDividendNotFound if no record with the given ID exists.
func (r *DividendRepository) GetDividend(dividendID string) (model.Dividend, error) {
	dividendLog.Debug("getting dividend by ID", "dividend_id", dividendID)
	query := `
		SELECT
			id, fund_id, portfolio_fund_id, record_date, ex_dividend_date,
			shares_owned, dividend_per_share, total_amount, reinvestment_status,
			buy_order_date, reinvestment_transaction_id
		FROM dividend
		WHERE id = ?
      `
	var d model.Dividend
	var RecordDateStr, ExDividendDateStr string
	var buyOrderDateStr, reinvestmentTransactionIDString sql.NullString

	err := r.getQuerier().QueryRow(query, dividendID).Scan(
		&d.ID,
		&d.FundID,
		&d.PortfolioFundID,
		&RecordDateStr,
		&ExDividendDateStr,
		&d.SharesOwned,
		&d.DividendPerShare,
		&d.TotalAmount,
		&d.ReinvestmentStatus,
		&buyOrderDateStr,
		&reinvestmentTransactionIDString,
	)

	if err == sql.ErrNoRows {
		return model.Dividend{}, apperrors.ErrDividendNotFound
	}
	if err != nil {
		return d, fmt.Errorf("failed to get dividend: %w", err)
	}

	d.RecordDate, err = ParseTime(RecordDateStr)
	if err != nil || d.RecordDate.IsZero() {
		return d, fmt.Errorf("failed to parse recordDate: %w", err)
	}

	d.ExDividendDate, err = ParseTime(ExDividendDateStr)
	if err != nil || d.ExDividendDate.IsZero() {
		return d, fmt.Errorf("failed to parse exDividendDate: %w", err)
	}

	if buyOrderDateStr.Valid {
		d.BuyOrderDate, err = ParseTime(buyOrderDateStr.String)
		if err != nil || d.BuyOrderDate.IsZero() {
			return d, fmt.Errorf("failed to parse buyOrderDate: %w", err)
		}
	}

	if reinvestmentTransactionIDString.Valid {
		d.ReinvestmentTransactionID = reinvestmentTransactionIDString.String
	}

	return d, nil
}

// InsertDividend inserts a new dividend record into the database.
// Nullable fields (BuyOrderDate, ReinvestmentTransactionID) are written as SQL NULL when zero/empty.
func (r *DividendRepository) InsertDividend(ctx context.Context, d *model.Dividend) error {
	dividendLog.DebugContext(ctx, "inserting dividend", "dividend_id", d.ID, "fund_id", d.FundID)
	query := `
        INSERT INTO dividend (
		id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned, dividend_per_share,
		total_amount, reinvestment_status, buy_order_date, reinvestment_transaction_id, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	var buyOrderDate any
	if !d.BuyOrderDate.IsZero() {
		buyOrderDate = d.BuyOrderDate.Format("2006-01-02")
	}

	var reinvestmentTransactionID any
	if d.ReinvestmentTransactionID != "" {
		reinvestmentTransactionID = d.ReinvestmentTransactionID
	}

	_, err := r.getQuerier().ExecContext(ctx, query,
		d.ID,
		d.FundID,
		d.PortfolioFundID,
		d.RecordDate.Format("2006-01-02 15:04:05"),
		d.ExDividendDate.Format("2006-01-02 15:04:05"),
		d.SharesOwned,
		d.DividendPerShare,
		d.TotalAmount,
		d.ReinvestmentStatus,
		buyOrderDate,
		reinvestmentTransactionID,
		d.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return fmt.Errorf("failed to insert dividend: %w", err)
	}

	return nil
}

// UpdateDividend updates an existing dividend record in the database.
// Returns ErrDividendNotFound if no row with the dividend's ID exists.
// Nullable fields (BuyOrderDate, ReinvestmentTransactionID) are written as SQL NULL when zero/empty.
func (r *DividendRepository) UpdateDividend(ctx context.Context, d *model.Dividend) error {
	dividendLog.DebugContext(ctx, "updating dividend", "dividend_id", d.ID)
	query := `
        UPDATE dividend
        SET fund_id = ?, portfolio_fund_id = ?, record_date = ?, ex_dividend_date = ?, shares_owned = ?, dividend_per_share = ?,
		total_amount = ?, reinvestment_status = ?, buy_order_date = ?, reinvestment_transaction_id = ?, created_at = ?
        WHERE id = ?
    `

	var buyOrderDate any
	if !d.BuyOrderDate.IsZero() {
		buyOrderDate = d.BuyOrderDate.Format("2006-01-02")
	}

	var reinvestmentTransactionID any
	if d.ReinvestmentTransactionID != "" {
		reinvestmentTransactionID = d.ReinvestmentTransactionID
	}

	result, err := r.getQuerier().ExecContext(ctx, query,
		d.FundID,
		d.PortfolioFundID,
		d.RecordDate.Format("2006-01-02 15:04:05"),
		d.ExDividendDate.Format("2006-01-02 15:04:05"),
		d.SharesOwned,
		d.DividendPerShare,
		d.TotalAmount,
		d.ReinvestmentStatus,
		buyOrderDate,
		reinvestmentTransactionID,
		d.CreatedAt.Format("2006-01-02 15:04:05"),
		d.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update dividend: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrDividendNotFound
	}

	return nil
}

// DeleteDividend removes a dividend record from the database by its ID.
// Returns ErrDividendNotFound if no row with the given ID exists.
func (r *DividendRepository) DeleteDividend(ctx context.Context, dividendID string) error {
	dividendLog.DebugContext(ctx, "deleting dividend", "dividend_id", dividendID)
	query := `DELETE FROM dividend WHERE id = ?`

	result, err := r.getQuerier().ExecContext(ctx, query, dividendID)
	if err != nil {
		return fmt.Errorf("failed to delete dividend: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrDividendNotFound
	}

	return nil
}
