package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

var devLog = logging.NewLogger("developer")

// DeveloperRepository provides data access methods for developer and system
// utilities: logs, logging configuration, exchange rates, fund prices, and CSV imports.
type DeveloperRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewDeveloperRepository creates a new DeveloperRepository with the provided database connection.
func NewDeveloperRepository(db *sql.DB) *DeveloperRepository {
	return &DeveloperRepository{db: db}
}

// WithTx returns a new DeveloperRepository scoped to the provided transaction.
func (r *DeveloperRepository) WithTx(tx *sql.Tx) *DeveloperRepository {
	return &DeveloperRepository{
		db: r.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (r *DeveloperRepository) getQuerier() Querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetLogs retrieves log entries from the database with dynamic filtering and pagination.
// Builds a WHERE clause dynamically based on provided filters and supports multiple filter types:
//   - Level filtering: Filters by one or more log levels (debug, info, warning, error, critical)
//   - Category filtering: Filters by one or more categories (portfolio, fund, transaction, etc.)
//   - Date range filtering: Filters by start and/or end timestamps
//   - Source filtering: Partial match on source field using LIKE
//   - Message filtering: Partial match on message content using LIKE
//   - Cursor pagination: Supports efficient pagination using timestamp+id cursor
//   - Skip pagination: Offset-based pagination that can be combined with cursor filtering.
//     Includes overshoot protection: if skip exceeds the total matching rows, the offset is
//     clamped so the last page of results is returned instead of an empty set.
//
// The pagination uses a cursor-based approach for efficiency with large result sets,
// optionally combined with a skip offset. Returns one extra record beyond perPage to
// determine if more results exist.

//nolint:gocyclo,funlen // Complex filtering logic with dynamic WHERE clause requires length
func (r *DeveloperRepository) GetLogs(filters *model.LogFilters) (*model.LogResponse, error) {
	devLog.Debug("getting logs", "per_page", filters.PerPage, "sort_dir", filters.SortDir)
	// Build dynamic WHERE clause
	var whereClauses []string
	var args []any

	// 1. Level filtering (OR logic)
	if len(filters.Levels) > 0 {
		placeholders := make([]string, len(filters.Levels))
		for i, level := range filters.Levels {
			placeholders[i] = "?"
			// Convert to uppercase to match database storage
			args = append(args, strings.ToUpper(level))
		}
		whereClauses = append(whereClauses, "level IN ("+strings.Join(placeholders, ",")+")")
	}

	// 2. Category filtering (OR logic)
	if len(filters.Categories) > 0 {
		placeholders := make([]string, len(filters.Categories))
		for i, category := range filters.Categories {
			placeholders[i] = "?"
			// Convert to uppercase to match database storage
			args = append(args, strings.ToUpper(category))
		}
		whereClauses = append(whereClauses, "category IN ("+strings.Join(placeholders, ",")+")")
	}

	// 3. Date range filtering
	if filters.StartDate != nil {
		whereClauses = append(whereClauses, "timestamp >= ?")
		args = append(args, filters.StartDate.Format("2006-01-02 15:04:05.000"))
	}

	if filters.EndDate != nil {
		whereClauses = append(whereClauses, "timestamp <= ?")
		args = append(args, filters.EndDate.Format("2006-01-02 15:04:05.000"))
	}

	// 4. Source filtering (partial match)
	if filters.Source != "" {
		whereClauses = append(whereClauses, "source LIKE ?")
		args = append(args, "%"+filters.Source+"%")
	}

	// 5. Message filtering (partial match)
	if filters.Message != "" {
		whereClauses = append(whereClauses, "message LIKE ?")
		args = append(args, "%"+filters.Message+"%")
	}

	// 6. Cursor pagination
	if filters.Cursor != "" {
		parts := strings.Split(filters.Cursor, "_")
		if len(parts) == 2 {
			timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
			if err == nil {
				id := parts[1]

				if filters.SortDir == "desc" {
					// For descending: (timestamp, id) < (cursor_timestamp, cursor_id)
					whereClauses = append(whereClauses,
						"(timestamp < ? OR (timestamp = ? AND id < ?))")
					args = append(args, timestamp.Format("2006-01-02 15:04:05.000"),
						timestamp.Format("2006-01-02 15:04:05.000"), id)
				} else {
					// For ascending: (timestamp, id) > (cursor_timestamp, cursor_id)
					whereClauses = append(whereClauses,
						"(timestamp > ? OR (timestamp = ? AND id > ?))")
					args = append(args, timestamp.Format("2006-01-02 15:04:05.000"),
						timestamp.Format("2006-01-02 15:04:05.000"), id)
				}
			}
		}
	}

	// Build WHERE clause
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build ORDER BY clause based on sort_dir
	orderSQL := "ORDER BY timestamp DESC, id DESC"
	if filters.SortDir == "asc" {
		orderSQL = "ORDER BY timestamp ASC, id ASC"
	}

	// Handle skip with overshoot protection.
	// Note: when both Cursor and Skip are provided, the count and offset apply
	// to the cursor-filtered result set (rows after the cursor), not the full table.
	actualSkip := filters.Skip
	if filters.Skip > 0 {
		// Count total matching rows to avoid skipping past the end
		//nolint:gosec // G202: SQL concatenation is safe
		countQuery := "SELECT COUNT(*) FROM log " + whereSQL
		var total int
		err := r.getQuerier().QueryRow(countQuery, args...).Scan(&total)
		if err != nil {
			devLog.Error("failed to count logs for skip overshoot check", "error", err)
			return nil, fmt.Errorf("failed to count logs: %w", err)
		}
		if filters.Skip >= total {
			// Skip would overshoot - return the last page instead
			devLog.Debug("skip overshoot detected, clamping to last page",
				"requested_skip", filters.Skip, "total", total, "per_page", filters.PerPage)
			lastPage := total - filters.PerPage
			if lastPage < 0 || filters.PerPage == 0 {
				lastPage = 0
			}
			actualSkip = lastPage
		}
	}

	// Build LIMIT/OFFSET clause
	limitOffsetSQL := "LIMIT ?"
	limitOffsetArgs := []interface{}{filters.PerPage + 1}
	if actualSkip > 0 {
		limitOffsetSQL = "LIMIT ? OFFSET ?"
		limitOffsetArgs = []interface{}{filters.PerPage + 1, actualSkip}
	}

	//nolint:gosec // G202: SQL concatenation is safe - whereSQL and orderSQL contain no user input, all user values are parameterized
	query := `
		SELECT id, timestamp, level, category, message, details, source,
		       request_id, stack_trace, http_status, ip_address, user_agent
		FROM log
		` + whereSQL + `
		` + orderSQL + `
		` + limitOffsetSQL + `
	`
	args = append(args, limitOffsetArgs...)

	// Execute query
	rows, err := r.getQuerier().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query log table: %w", err)
	}
	defer rows.Close()

	// Scan results
	logs := []model.Log{}

	for rows.Next() {
		var timestampStr string
		var detailsStr, RequestIDStr, StackTraceStr, HTTPStatusStr, IPAddressStr, UserAgentStr sql.NullString
		var l model.Log

		err := rows.Scan(
			&l.ID,
			&timestampStr,
			&l.Level,
			&l.Category,
			&l.Message,
			&detailsStr,
			&l.Source,
			&RequestIDStr,
			&StackTraceStr,
			&HTTPStatusStr,
			&IPAddressStr,
			&UserAgentStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log results: %w", err)
		}

		l.Timestamp, err = ParseTime(timestampStr)
		if err != nil || l.Timestamp.IsZero() {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		// Handle nullable fields
		if detailsStr.Valid {
			l.Details = detailsStr.String
		}
		if RequestIDStr.Valid {
			l.RequestID = RequestIDStr.String
		}
		if StackTraceStr.Valid {
			l.StackTrace = StackTraceStr.String
		}
		if HTTPStatusStr.Valid {
			l.HTTPStatus = HTTPStatusStr.String
		}
		if IPAddressStr.Valid {
			l.IPAddress = IPAddressStr.String
		}
		if UserAgentStr.Valid {
			l.UserAgent = UserAgentStr.String
		}

		logs = append(logs, l)
	}

	// Generate next cursor
	hasMore := len(logs) > filters.PerPage
	var nextCursor string
	if hasMore {
		last := logs[filters.PerPage-1]
		nextCursor = fmt.Sprintf("%s_%s",
			last.Timestamp.Format(time.RFC3339Nano),
			last.ID)
		logs = logs[:filters.PerPage]
	}

	return &model.LogResponse{
		Logs:       logs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Count:      len(logs),
	}, nil
}

// GetLogFilterOptions retrieves distinct non-null values for log filter columns.
// Returns sorted lists of levels, categories, sources, and messages for frontend picklists.
// Levels and categories are returned in uppercase.
func (r *DeveloperRepository) GetLogFilterOptions() (*model.LogFilterOptions, error) {
	devLog.Debug("getting log filter options")

	options := &model.LogFilterOptions{}
	var err error

	options.Levels, err = r.queryDistinctColumn("level", true)
	if err != nil {
		return nil, err
	}
	options.Categories, err = r.queryDistinctColumn("category", true)
	if err != nil {
		return nil, err
	}
	options.Sources, err = r.queryDistinctColumn("source", false)
	if err != nil {
		return nil, err
	}
	options.Messages, err = r.queryDistinctColumn("message", false)
	if err != nil {
		return nil, err
	}

	return options, nil
}

// queryDistinctColumn returns sorted distinct non-empty values for a single log table column.
// When upper is true the values are returned in uppercase.
func (r *DeveloperRepository) queryDistinctColumn(column string, upper bool) ([]string, error) {
	colExpr := column
	if upper {
		colExpr = "UPPER(" + column + ")"
	}
	//nolint:gosec // G202: column and colExpr are hardcoded strings, not user input
	query := "SELECT DISTINCT " + colExpr + " FROM log WHERE " + column + " IS NOT NULL AND " + column + " != '' ORDER BY " + colExpr

	rows, err := r.getQuerier().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct %s: %w", column, err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var val string
		if err := rows.Scan(&val); err != nil {
			return nil, fmt.Errorf("failed to scan distinct %s: %w", column, err)
		}
		values = append(values, val)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating distinct %s: %w", column, err)
	}
	if values == nil {
		values = []string{}
	}
	return values, nil
}

// GetLoggingConfig retrieves the current logging configuration from system_setting table.
// Returns the LOGGING_ENABLED and LOGGING_LEVEL settings.
// If settings are not configured, returns default values: enabled=true, level="info".
// Logs a warning message when default values are used.
func (r *DeveloperRepository) GetLoggingConfig() (model.LoggingSetting, error) {
	devLog.Debug("getting logging config")

	queryEnabled := `
        SELECT value
		FROM system_setting
		WHERE key = 'LOGGING_ENABLED'
      `
	queryLevel := `
        SELECT value
		FROM system_setting
		WHERE key = 'LOGGING_LEVEL'
      `
	// Setting default logging mode if settings are not configured.
	conf := model.LoggingSetting{
		Enabled: true,
		Level:   "info",
	}
	err := r.getQuerier().QueryRow(queryEnabled).Scan(
		&conf.Enabled,
	)
	if err == sql.ErrNoRows {
		devLog.Debug("logging not set, defaulting to enabled")
	} else if err != nil {
		return conf, fmt.Errorf("failed to query logging enabled setting: %w", err)
	}

	err = r.getQuerier().QueryRow(queryLevel).Scan(
		&conf.Level,
	)
	if err == sql.ErrNoRows {
		devLog.Debug("level not set, defaulting to INFO")
	} else if err != nil {
		return conf, fmt.Errorf("failed to query logging level setting: %w", err)
	}

	return conf, nil
}

// SetLoggingConfig persists a new logging configuration setting to the database.
func (r *DeveloperRepository) SetLoggingConfig(ctx context.Context, setting model.SystemSetting) error {
	devLog.DebugContext(ctx, "setting logging config", "key", setting.Key, "value", setting.Value)
	query := `
        INSERT INTO system_setting (id, key, value, updated_at)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(key) DO UPDATE SET
            value = ?,
			updated_at = ?
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		setting.ID,
		setting.Key,
		setting.Value,
		setting.UpdatedAt.Format("2006-01-02 15:04:05"),
		setting.Value,
		setting.UpdatedAt.Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return fmt.Errorf("failed to upsert system setting: %w", err)
	}
	return nil
}

// GetExchangeRate retrieves an exchange rate for a specific currency pair and date.
// Queries the exchange_rate table for an exact match on from_currency, to_currency, and date.
// Returns ErrExchangeRateNotFound if no matching rate exists.
// The date parameter should be in the format YYYY-MM-DD.
func (r *DeveloperRepository) GetExchangeRate(fromCurrency, toCurrency string, dateTime time.Time) (*model.ExchangeRate, error) {
	devLog.Debug("getting exchange rate", "from", fromCurrency, "to", toCurrency, "date", dateTime.Format("2006-01-02"))

	query := `
	SELECT from_currency, to_currency, rate, date
	FROM exchange_rate
	WHERE from_currency = ?
	AND to_currency = ?
	AND date = ?
`
	rate := model.ExchangeRate{}
	var dateStr string
	err := r.getQuerier().QueryRow(query, fromCurrency, toCurrency, dateTime.Format("2006-01-02")).Scan(
		&rate.FromCurrency,
		&rate.ToCurrency,
		&rate.Rate,
		&dateStr,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.ErrExchangeRateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get exchange rate %s→%s: %w", fromCurrency, toCurrency, err)
	}

	rate.Date, err = ParseTime(dateStr)
	if err != nil || rate.Date.IsZero() {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}

	return &rate, nil
}

// UpdateExchangeRate upserts an exchange rate record.
// On conflict (same from_currency, to_currency, date), updates the rate and created_at fields.
func (r *DeveloperRepository) UpdateExchangeRate(ctx context.Context, exRate model.ExchangeRate) error {
	devLog.DebugContext(ctx, "upserting exchange rate", "from", exRate.FromCurrency, "to", exRate.ToCurrency, "date", exRate.Date.Format("2006-01-02"))

	query := `
        INSERT INTO exchange_rate (id, from_currency, to_currency, rate, date, created_at)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(from_currency, to_currency, date) DO UPDATE SET
            rate = ?,
			created_at = ?
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		exRate.ID,
		exRate.FromCurrency,
		exRate.ToCurrency,
		exRate.Rate,
		exRate.Date.Format("2006-01-02"),
		time.Now().UTC().Format("2006-01-02 15:04:05"),
		exRate.Rate,
		time.Now().UTC().Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return fmt.Errorf("failed to upsert exchange rate: %w", err)
	}

	return nil
}

// AddLog inserts a single log entry into the log table.
func (r *DeveloperRepository) AddLog(ctx context.Context, logEntry model.Log) error {
	devLog.DebugContext(ctx, "adding log entry", "level", logEntry.Level, "category", logEntry.Category)

	query := `
		INSERT INTO log (id, timestamp, level, category, message, details, source, request_id, stack_trace, http_status, ip_address, user_agent)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Convert HTTPStatus string → *int for the INTEGER column.
	// Empty string becomes NULL; non-numeric strings are dropped to NULL with a warning.
	var httpStatus *int
	if logEntry.HTTPStatus != "" {
		if v, err := strconv.Atoi(logEntry.HTTPStatus); err == nil {
			httpStatus = &v
		} else {
			devLog.Warn("non-numeric http_status in log entry, storing as NULL",
				"http_status", logEntry.HTTPStatus, "log_id", logEntry.ID)
		}
	}

	_, err := r.getQuerier().ExecContext(ctx, query,
		logEntry.ID,
		logEntry.Timestamp.Format("2006-01-02 15:04:05.000"),
		logEntry.Level,
		logEntry.Category,
		logEntry.Message,
		logEntry.Details,
		logEntry.Source,
		logEntry.RequestID,
		logEntry.StackTrace,
		httpStatus,
		logEntry.IPAddress,
		logEntry.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	return nil
}

// DeleteLogs removes all entries from the log table.
func (r *DeveloperRepository) DeleteLogs(ctx context.Context) error {
	devLog.DebugContext(ctx, "deleting all logs")
	query := `DELETE FROM log`

	_, err := r.getQuerier().ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete logs: %w", err)
	}

	return nil
}
