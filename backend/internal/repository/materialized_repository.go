package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

var matLog = logging.NewLogger("system")

// MaterializedRepository provides data access methods for the fund_history_materialized table.
type MaterializedRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewMaterializedRepository creates a new repository instance.
func NewMaterializedRepository(db *sql.DB) *MaterializedRepository {
	return &MaterializedRepository{db: db}
}

// WithTx returns a new MaterializedRepository scoped to the provided transaction.
func (r *MaterializedRepository) WithTx(tx *sql.Tx) *MaterializedRepository {
	return &MaterializedRepository{
		db: r.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (r *MaterializedRepository) getQuerier() Querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetMaterializedHistory retrieves aggregated portfolio history by querying fund_history_materialized.
// This method streams results using a callback pattern to minimize memory usage.
//
// The method performs the following:
//  1. Builds the SQL query and parameter list (delegates to buildMaterializedQuery)
//  2. Executes the query against the fund_history_materialized table
//  3. Scans each row into a PortfolioHistoryMaterialized struct
//  4. Parses date fields (date, calculated_at)
//  5. Invokes the callback function for each record
//
// The query aggregates fund-level data from fund_history_materialized using GROUP BY.
// All values (realized_gain, sale_proceeds, original_cost, dividends) are read directly
// from pre-computed columns in the materialized table — no correlated subqueries.
// is_archived is fetched via a JOIN to the portfolio table.
//
// Parameters:
//   - portfolioIDs: Slice of portfolio IDs to retrieve history for
//   - startDate: First date to include in results (inclusive)
//   - endDate: Last date to include in results (inclusive)
//   - callback: Function called for each aggregated record, receives the record and should return error if processing fails
//
// The callback pattern allows the caller to process records one at a time without loading
// the entire result set into memory, which is efficient for large date ranges.
//
// Returns an error if the query fails, date parsing fails, or if the callback returns an error during processing.
func (r *MaterializedRepository) GetMaterializedHistory(
	portfolioIDs []string,
	startDate, endDate time.Time,
	callback func(record model.PortfolioHistoryMaterialized) error,
) error {
	matLog.Debug("getting materialized history", "portfolio_count", len(portfolioIDs), "start_date", startDate.Format("2006-01-02"), "end_date", endDate.Format("2006-01-02"))

	if len(portfolioIDs) == 0 {
		return nil
	}

	query, args := r.buildMaterializedQuery(portfolioIDs, startDate, endDate)

	rows, err := r.getQuerier().Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query fund_history_materialized: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var record model.PortfolioHistoryMaterialized
		var dateStr, calculatedAtStr string

		err := rows.Scan(
			&record.ID,
			&record.PortfolioID,
			&dateStr,
			&record.Value,
			&record.Cost,
			&record.RealizedGain,
			&record.UnrealizedGain,
			&record.TotalDividends,
			&record.TotalSaleProceeds,
			&record.TotalOriginalCost,
			&record.TotalGainLoss,
			&record.IsArchived,
			&calculatedAtStr,
		)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		record.Date, err = ParseTime(dateStr)
		if err != nil {
			return fmt.Errorf("failed to parse date: %w", err)
		}

		record.CalculatedAt, err = ParseTime(calculatedAtStr)
		if err != nil {
			return fmt.Errorf("failed to parse calculated_at: %w", err)
		}

		if err := callback(record); err != nil {
			return err
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}
	return nil
}

// buildMaterializedQuery constructs the SQL query and argument list for fetching portfolio history
// from the materialized view. This method was extracted from GetMaterializedHistory to reduce
// function complexity and improve readability.
//
// The query performs portfolio-level aggregation by:
//   - Joining fund_history_materialized with portfolio_fund and portfolio tables
//   - Grouping by date and portfolio_id to sum fund-level metrics
//   - Summing pre-computed fund-level metrics (realized gains, dividends, sale proceeds, original cost)
//   - Filtering by portfolio IDs and date range
//
// Parameters:
//   - portfolioIDs: Slice of portfolio UUIDs to include in the query
//   - startDate: Inclusive start date for the history range
//   - endDate: Inclusive end date for the history range
//
// Returns:
//   - SQL query string with placeholders for portfolioIDs and dates
//   - Argument slice containing portfolio IDs followed by formatted start and end dates
//
// Security Note: The #nosec G202 directive is used because placeholder concatenation
// is safe here - we're building "?" placeholders programmatically, not concatenating user input.
func (r *MaterializedRepository) buildMaterializedQuery(portfolioIDs []string, startDate, endDate time.Time) (string, []any) {

	placeholders := make([]string, len(portfolioIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	query := `
	SELECT
		'' as id,
		pf.portfolio_id,
		fh.date,
		SUM(fh.value) as value,
		SUM(fh.cost) as cost,
		SUM(fh.realized_gain) as realized_gain,
		SUM(fh.unrealized_gain) as unrealized_gain,
		SUM(fh.dividends) as total_dividends,
		SUM(fh.sale_proceeds) as total_sale_proceeds,
		SUM(fh.original_cost) as total_original_cost,
		SUM(fh.unrealized_gain) + SUM(fh.realized_gain) as total_gain_loss,
		p.is_archived,
		MAX(fh.calculated_at) as calculated_at
	FROM fund_history_materialized fh
	JOIN portfolio_fund pf ON fh.portfolio_fund_id = pf.id
	JOIN portfolio p ON pf.portfolio_id = p.id
	WHERE pf.portfolio_id IN (` + strings.Join(placeholders, ",") + `)
	AND fh.date >= ?
	AND fh.date <= ?
	GROUP BY fh.date, pf.portfolio_id, p.is_archived
	ORDER BY fh.date ASC
`

	args := make([]any, 0, len(portfolioIDs)+2)
	for _, id := range portfolioIDs {
		args = append(args, id)
	}
	args = append(args, startDate.Format("2006-01-02"))
	args = append(args, endDate.Format("2006-01-02"))

	return query, args
}

// GetFundHistoryMaterialized retrieves historical fund data from the materialized view.
// This method streams results using a callback pattern to minimize memory usage.
//
// The callback pattern allows the caller to process records one at a time without loading
// the entire result set into memory, which is efficient for large date ranges.
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve fund history for
//   - startDate: Inclusive start date for the query
//   - endDate: Inclusive end date for the query
//   - callback: Function called for each record found, receives the record and should return error if processing fails
//
// Returns an error if the query fails or if the callback returns an error during processing.
func (r *MaterializedRepository) GetFundHistoryMaterialized(
	portfolioID string,
	startDate, endDate time.Time,
	callback func(entry model.FundHistoryEntry) error,
) error {
	matLog.Debug("getting fund history materialized", "portfolio_id", portfolioID, "start_date", startDate.Format("2006-01-02"), "end_date", endDate.Format("2006-01-02"))
	query := `
		SELECT
			fh.id,
			fh.portfolio_fund_id,
			fh.fund_id,
			f.name as fund_name,
			fh.date,
			fh.shares,
			fh.price,
			fh.value,
			fh.cost,
			fh.realized_gain,
			fh.unrealized_gain,
			fh.total_gain_loss,
			fh.dividends,
			fh.fees,
			fh.sale_proceeds,
			fh.original_cost
		FROM fund_history_materialized fh
		JOIN portfolio_fund pf ON fh.portfolio_fund_id = pf.id
		JOIN fund f ON fh.fund_id = f.id
		WHERE pf.portfolio_id = ?
		  AND fh.date >= ?
		  AND fh.date <= ?
		ORDER BY fh.date ASC, f.name ASC
	`

	rows, err := r.getQuerier().Query(query, portfolioID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to query fund_history_materialized: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry model.FundHistoryEntry
		var dateStr string

		err := rows.Scan(
			&entry.ID,
			&entry.PortfolioFundID,
			&entry.FundID,
			&entry.FundName,
			&dateStr,
			&entry.Shares,
			&entry.Price,
			&entry.Value,
			&entry.Cost,
			&entry.RealizedGain,
			&entry.UnrealizedGain,
			&entry.TotalGainLoss,
			&entry.Dividends,
			&entry.Fees,
			&entry.SaleProceeds,
			&entry.OriginalCost,
		)
		if err != nil {
			return fmt.Errorf("failed to scan fund_history_materialized results: %w", err)
		}

		entry.Date, err = ParseTime(dateStr)
		if err != nil || entry.Date.IsZero() {
			return fmt.Errorf("failed to parse date: %w", err)
		}

		if err := callback(entry); err != nil {
			return err
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating fund_history_materialized: %w", err)
	}

	return nil
}

// GetLatestMaterializedDate returns the minimum of per-portfolio maximum dates and the
// global maximum calculated_at timestamp for the given portfolio IDs. Using the minimum
// of per-portfolio maximums ensures that if any single portfolio lags behind (e.g. due
// to a failed regeneration), the stale check correctly detects the gap rather than
// masking it with a higher-coverage portfolio's date. If no rows exist, returns
// zero-value times and false.
func (r *MaterializedRepository) GetLatestMaterializedDate(portfolioIDs []string) (latestDate time.Time, latestCalc time.Time, ok bool, err error) {
	matLog.Debug("getting latest materialized date", "portfolio_count", len(portfolioIDs))
	if len(portfolioIDs) == 0 {
		return time.Time{}, time.Time{}, false, nil
	}

	placeholders := make([]string, len(portfolioIDs))
	args := make([]any, len(portfolioIDs))
	for i, id := range portfolioIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	// MIN(max_date per portfolio) — if any portfolio lags, the group is stale.
	// MAX(max_calc per portfolio) — most recent recalculation time for source-data comparison.
	query := fmt.Sprintf(`
		SELECT MIN(max_date), MAX(max_calc)
		FROM (
			SELECT MAX(fhm.date) AS max_date, MAX(fhm.calculated_at) AS max_calc
			FROM fund_history_materialized fhm
			JOIN portfolio_fund pf ON fhm.portfolio_fund_id = pf.id
			WHERE pf.portfolio_id IN (%s)
			GROUP BY pf.portfolio_id
		) per_portfolio
	`, strings.Join(placeholders, ","))

	// MAX() aggregates lose column type information, so _texttotime won't
	// auto-parse them. Scan into strings and parse manually.
	var dateStr, calcStr sql.NullString
	if err := r.getQuerier().QueryRow(query, args...).Scan(&dateStr, &calcStr); err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("failed to get latest materialized date: %w", err)
	}

	if !dateStr.Valid || !calcStr.Valid {
		return time.Time{}, time.Time{}, false, nil
	}

	latestDate, err = time.Parse("2006-01-02", dateStr.String)
	if err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("failed to parse materialized date: %w", err)
	}

	latestCalc, err = time.Parse("2006-01-02 15:04:05", calcStr.String)
	if err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("failed to parse calculated_at: %w", err)
	}

	return latestDate, latestCalc, true, nil
}

// GetLatestSourceDates returns the most recent modification timestamps from the three
// source tables (transaction, fund_price, dividend) for the given portfolio IDs in a
// single query. fund_price uses MAX(date) since it has no created_at column.
func (r *MaterializedRepository) GetLatestSourceDates(portfolioIDs []string) (latestTxn, latestPrice, latestDiv time.Time, err error) {
	matLog.Debug("getting latest source dates", "portfolio_count", len(portfolioIDs))
	if len(portfolioIDs) == 0 {
		return time.Time{}, time.Time{}, time.Time{}, nil
	}

	placeholders := make([]string, len(portfolioIDs))
	args := make([]any, len(portfolioIDs))
	for i, id := range portfolioIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	inClause := strings.Join(placeholders, ",")

	// All three args sets are the same portfolio IDs, so triple them
	allArgs := make([]any, 0, len(args)*3)
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, args...)

	// MAX() aggregates lose column type information, so _texttotime won't
	// auto-parse them. Use COALESCE to empty string and parse manually.
	query := fmt.Sprintf(`
		SELECT
			COALESCE((SELECT MAX(t.created_at) FROM "transaction" t
				JOIN portfolio_fund pf ON t.portfolio_fund_id = pf.id
				WHERE pf.portfolio_id IN (%s)), ''),
			COALESCE((SELECT MAX(fp.date) FROM fund_price fp
				JOIN portfolio_fund pf ON fp.fund_id = pf.fund_id
				WHERE pf.portfolio_id IN (%s)), ''),
			COALESCE((SELECT MAX(d.created_at) FROM dividend d
				JOIN portfolio_fund pf ON d.portfolio_fund_id = pf.id
				WHERE pf.portfolio_id IN (%s)), '')
	`, inClause, inClause, inClause)

	var txnStr, priceStr, divStr string
	if err := r.getQuerier().QueryRow(query, allArgs...).Scan(&txnStr, &priceStr, &divStr); err != nil {
		return time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("failed to get latest source dates: %w", err)
	}

	if txnStr != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", txnStr); err == nil {
			latestTxn = parsed
		}
	}
	if priceStr != "" {
		if parsed, err := time.Parse("2006-01-02", priceStr); err == nil {
			latestPrice = parsed
		}
	}
	if divStr != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", divStr); err == nil {
			latestDiv = parsed
		}
	}

	return latestTxn, latestPrice, latestDiv, nil
}

// InvalidateMaterializedTable deletes cached entries from the given date forward,
// scoped to the specified portfolio_fund IDs. If pfIDs is empty, no rows are deleted.
func (r *MaterializedRepository) InvalidateMaterializedTable(ctx context.Context, date time.Time, pfIDs []string) error {
	matLog.DebugContext(ctx, "invalidating materialized table", "from_date", date.Format("2006-01-02"), "pf_count", len(pfIDs))
	if len(pfIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(pfIDs))
	args := make([]any, 0, len(pfIDs)+1)
	args = append(args, date.Format("2006-01-02"))
	for i, id := range pfIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		DELETE FROM fund_history_materialized
		WHERE date >= ? AND portfolio_fund_id IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := r.getQuerier().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete materialized view records: %w", err)
	}

	return nil
}

// InsertMaterializedEntries bulk-inserts pre-calculated materialized view entries into the database.
func (r *MaterializedRepository) InsertMaterializedEntries(ctx context.Context, fundHistoryEntries []model.FundHistoryEntry) error {
	matLog.DebugContext(ctx, "inserting materialized entries", "count", len(fundHistoryEntries))

	if len(fundHistoryEntries) == 0 {
		return nil
	}

	stmt, err := r.getQuerier().PrepareContext(ctx, `
        INSERT INTO fund_history_materialized (id, portfolio_fund_id, fund_id, date, shares, price, value, cost, realized_gain, unrealized_gain, total_gain_loss, dividends, fees, sale_proceeds, original_cost, calculated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	calculatedAt := time.Now().UTC().Format("2006-01-02 15:04:05")
	for _, e := range fundHistoryEntries {

		_, err := stmt.ExecContext(ctx,
			e.ID,
			e.PortfolioFundID,
			e.FundID,
			e.Date.Format("2006-01-02"),
			e.Shares,
			e.Price,
			e.Value,
			e.Cost,
			e.RealizedGain,
			e.UnrealizedGain,
			e.TotalGainLoss,
			e.Dividends,
			e.Fees,
			e.SaleProceeds,
			e.OriginalCost,
			calculatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert materialized entry for %s on %s: %w", e.PortfolioFundID, e.Date.Format("2006-01-02"), err)
		}
	}
	return nil

}

// GetPortfolioSummaryLatest retrieves aggregated portfolio metrics for the most recent date only.
// This is used by summary/detail endpoints that only need the current state, avoiding a full
// date-range scan of the materialized table.
func (r *MaterializedRepository) GetPortfolioSummaryLatest(
	portfolioIDs []string,
	callback func(record model.PortfolioHistoryMaterialized) error,
) error {
	matLog.Debug("getting portfolio summary latest", "portfolio_count", len(portfolioIDs))
	if len(portfolioIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(portfolioIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	inClause := strings.Join(placeholders, ",")

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	query := `
	SELECT
		'' as id,
		pf.portfolio_id,
		fh.date,
		SUM(fh.value) as value,
		SUM(fh.cost) as cost,
		SUM(fh.realized_gain) as realized_gain,
		SUM(fh.unrealized_gain) as unrealized_gain,
		SUM(fh.dividends) as total_dividends,
		SUM(fh.sale_proceeds) as total_sale_proceeds,
		SUM(fh.original_cost) as total_original_cost,
		SUM(fh.unrealized_gain) + SUM(fh.realized_gain) as total_gain_loss,
		p.is_archived,
		MAX(fh.calculated_at) as calculated_at
	FROM fund_history_materialized fh
	JOIN portfolio_fund pf ON fh.portfolio_fund_id = pf.id
	JOIN portfolio p ON pf.portfolio_id = p.id
	WHERE pf.portfolio_id IN (` + inClause + `)
	AND fh.date = (
		SELECT MAX(fh2.date)
		FROM fund_history_materialized fh2
		JOIN portfolio_fund pf2 ON fh2.portfolio_fund_id = pf2.id
		WHERE pf2.portfolio_id = pf.portfolio_id
	)
	GROUP BY fh.date, pf.portfolio_id, p.is_archived
	ORDER BY fh.date ASC
`

	args := make([]any, 0, len(portfolioIDs))
	for _, id := range portfolioIDs {
		args = append(args, id)
	}

	rows, err := r.getQuerier().Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query portfolio summary latest: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var record model.PortfolioHistoryMaterialized
		var dateStr, calculatedAtStr string

		err := rows.Scan(
			&record.ID,
			&record.PortfolioID,
			&dateStr,
			&record.Value,
			&record.Cost,
			&record.RealizedGain,
			&record.UnrealizedGain,
			&record.TotalDividends,
			&record.TotalSaleProceeds,
			&record.TotalOriginalCost,
			&record.TotalGainLoss,
			&record.IsArchived,
			&calculatedAtStr,
		)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		record.Date, err = ParseTime(dateStr)
		if err != nil {
			return fmt.Errorf("failed to parse date: %w", err)
		}

		record.CalculatedAt, err = ParseTime(calculatedAtStr)
		if err != nil {
			return fmt.Errorf("failed to parse calculated_at: %w", err)
		}

		if err := callback(record); err != nil {
			return err
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}
	return nil
}
