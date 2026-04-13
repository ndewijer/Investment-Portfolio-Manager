package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var devLog = logging.NewLogger("developer")

// DeveloperService handles Developer-related business logic operations.
type DeveloperService struct {
	db                      *sql.DB
	developerRepo           *repository.DeveloperRepository
	fundRepo                *repository.FundRepository
	transactionRepo         *repository.TransactionRepository
	pfRepo                  *repository.PortfolioFundRepository
	materializedInvalidator MaterializedInvalidator
	logHandler              *logging.LogHandler
}

// NewDeveloperService creates a new DeveloperService with the provided repository dependencies.
func NewDeveloperService(
	db *sql.DB,
	developerRepo *repository.DeveloperRepository,
	fundRepo *repository.FundRepository,
	transactionRepo *repository.TransactionRepository,
	pfRepo *repository.PortfolioFundRepository,
) *DeveloperService {
	return &DeveloperService{
		db:              db,
		developerRepo:   developerRepo,
		fundRepo:        fundRepo,
		transactionRepo: transactionRepo,
		pfRepo:          pfRepo,
	}
}

// SetMaterializedInvalidator injects the MaterializedInvalidator after construction.
// This breaks the circular initialization order between DeveloperService and MaterializedService.
func (s *DeveloperService) SetMaterializedInvalidator(m MaterializedInvalidator) {
	s.materializedInvalidator = m
}

// SetLogHandler injects the logging LogHandler for runtime config updates.
func (s *DeveloperService) SetLogHandler(h *logging.LogHandler) {
	s.logHandler = h
}

// GetLogFilterOptions retrieves distinct values for log filter columns.
// Used by the frontend to populate filter picklists.
func (s *DeveloperService) GetLogFilterOptions() (*model.LogFilterOptions, error) {
	devLog.Debug("retrieving log filter options")
	return s.developerRepo.GetLogFilterOptions()
}

// GetLogs retrieves system logs with the specified filters and pagination.
// Returns a paginated response with cursor for fetching subsequent pages.
// The context parameter is currently unused but reserved for future cancellation support.
func (s *DeveloperService) GetLogs(_ context.Context, filters *model.LogFilters) (*model.LogResponse, error) {
	devLog.Debug("retrieving logs")
	result, err := s.developerRepo.GetLogs(filters)
	if err != nil {
		return nil, fmt.Errorf("get logs: %w", err)
	}
	return result, nil
}

// GetLoggingConfig retrieves the current logging configuration settings.
// Returns the enabled status and logging level from system settings.
// Returns default values (enabled=true, level="info") if settings are not configured.
func (s *DeveloperService) GetLoggingConfig() (model.LoggingSetting, error) {
	devLog.Debug("retrieving logging config")
	config, err := s.developerRepo.GetLoggingConfig()
	if err != nil {
		return model.LoggingSetting{}, fmt.Errorf("get logging config: %w", err)
	}
	return config, nil
}

// SetLoggingConfig updates the logging configuration in system settings.
// Upserts LOGGING_ENABLED and LOGGING_LEVEL settings within a single transaction.
// Only updates fields that are present in the request.
func (s *DeveloperService) SetLoggingConfig(ctx context.Context, req request.SetLoggingConfig) (model.LoggingSetting, error) {
	devLog.DebugContext(ctx, "setting logging config", "level", req.Level)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.LoggingSetting{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.
	updateTime := time.Now().UTC()
	if req.Enabled != nil {
		settingEnabled := model.SystemSetting{
			ID:        uuid.New().String(),
			Key:       "LOGGING_ENABLED",
			Value:     req.Enabled,
			UpdatedAt: &updateTime,
		}

		if err := s.developerRepo.WithTx(tx).SetLoggingConfig(ctx, settingEnabled); err != nil {
			return model.LoggingSetting{}, fmt.Errorf("failed to update LOGGING_ENABLED: %w", err)
		}

	}
	if req.Level != "" {
		settingLevel := model.SystemSetting{
			ID:        uuid.New().String(),
			Key:       "LOGGING_LEVEL",
			Value:     req.Level,
			UpdatedAt: &updateTime,
		}
		if err := s.developerRepo.WithTx(tx).SetLoggingConfig(ctx, settingLevel); err != nil {
			return model.LoggingSetting{}, fmt.Errorf("failed to update LOGGING_LEVEL: %w", err)
		}

	}

	if err = tx.Commit(); err != nil {
		return model.LoggingSetting{}, fmt.Errorf("commit transaction: %w", err)
	}

	config, err := s.GetLoggingConfig()
	if err != nil {
		return model.LoggingSetting{}, fmt.Errorf("cannot retrieve new log settings: %w", err)
	}

	// Update the runtime logging handler immediately.
	if s.logHandler != nil {
		s.logHandler.SetEnabled(config.Enabled)
		s.logHandler.SetLevel(logging.DBStringToSlogLevel(config.Level))
	}

	devLog.InfoContext(ctx, "logging config updated", "enabled", config.Enabled, "level", config.Level)
	return config, nil
}

// GetExchangeRate retrieves the exchange rate for a specific currency pair and date.
// Returns ErrExchangeRateNotFound if no rate exists for the given parameters.
func (s *DeveloperService) GetExchangeRate(fromCurrency, toCurrency string, dateTime time.Time) (*model.ExchangeRate, error) {
	devLog.Debug("retrieving exchange rate", "from", fromCurrency, "to", toCurrency, "date", dateTime.Format("2006-01-02"))
	rate, err := s.developerRepo.GetExchangeRate(fromCurrency, toCurrency, dateTime)
	if err != nil {
		return nil, fmt.Errorf("get exchange rate: %w", err)
	}
	return rate, nil
}

// UpdateExchangeRate creates or updates an exchange rate for a given currency pair and date.
// Parses the rate from the request string and upserts the record within a transaction.
func (s *DeveloperService) UpdateExchangeRate(
	ctx context.Context,
	req request.SetExchangeRateRequest,
) (model.ExchangeRate, error) {
	devLog.DebugContext(ctx, "updating exchange rate", "from", req.FromCurrency, "to", req.ToCurrency, "date", req.Date)

	var exRate model.ExchangeRate
	var err error

	exRate.Date, err = time.Parse("2006-01-02", req.Date)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("parse date: %w", err)
	}
	exRate.ID = uuid.New().String()
	exRate.ToCurrency = req.ToCurrency
	exRate.FromCurrency = req.FromCurrency

	rateFloat, err := strconv.ParseFloat(strings.TrimSpace(req.Rate), 64)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("parse rate: %w", err)
	}
	exRate.Rate = rateFloat

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if err := s.developerRepo.WithTx(tx).UpdateExchangeRate(ctx, exRate); err != nil {
		return model.ExchangeRate{}, fmt.Errorf("failed to update exchange rate: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return model.ExchangeRate{}, fmt.Errorf("commit transaction: %w", err)
	}

	devLog.InfoContext(ctx, "exchange rate updated", "from", exRate.FromCurrency, "to", exRate.ToCurrency, "date", exRate.Date.Format("2006-01-02"))
	return exRate, nil
}

// GetFundPrice retrieves the price for a specific fund on a specific date.
// Uses the FundRepository's GetFundPrice method to fetch the price.
// Returns ErrFundPriceNotFound if no price exists for the given fund and date.
func (s *DeveloperService) GetFundPrice(fundID string, dateTime time.Time) (model.FundPrice, error) {
	devLog.Debug("retrieving fund price", "fundID", fundID, "date", dateTime.Format("2006-01-02"))

	fundIDs := []string{fundID}
	fundPrices, err := s.fundRepo.GetFundPrice(fundIDs, dateTime, dateTime, true)
	if err != nil {
		return model.FundPrice{}, fmt.Errorf("get fund price: %w", err)
	}

	prices, exists := fundPrices[fundID]
	if !exists || len(prices) == 0 {
		return model.FundPrice{}, apperrors.ErrFundPriceNotFound
	}

	return fundPrices[fundID][0], nil
}

// UpdateFundPrice creates or updates a fund price for a given fund and date.
// Parses the price from the request string and upserts the record within a transaction.
func (s *DeveloperService) UpdateFundPrice(
	ctx context.Context,
	req request.SetFundPriceRequest,
) (model.FundPrice, error) {
	devLog.DebugContext(ctx, "updating fund price", "fundID", req.FundID, "date", req.Date)

	var fp model.FundPrice
	var err error

	fp.Date, err = time.Parse("2006-01-02", req.Date)
	if err != nil {
		return model.FundPrice{}, fmt.Errorf("parse date: %w", err)
	}
	priceFloat, err := strconv.ParseFloat(strings.TrimSpace(req.Price), 64)
	if err != nil {
		return model.FundPrice{}, fmt.Errorf("parse price: %w", err)
	}
	fp.Price = priceFloat
	fp.FundID = req.FundID

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.FundPrice{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	fp.ID = uuid.New().String()

	if err := s.fundRepo.WithTx(tx).UpdateFundPrice(ctx, fp); err != nil {
		return model.FundPrice{}, fmt.Errorf("failed to update fund price: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return model.FundPrice{}, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), fp.Date, nil, fp.FundID, ""); err != nil {
				devLog.Warn("failed to regenerate materialized table after manual price update", "error", err)
			}
		}()
	}

	devLog.InfoContext(ctx, "fund price updated", "fundID", fp.FundID, "date", fp.Date.Format("2006-01-02"))
	return fp, nil
}

// DeleteLogs clears all log entries and records a single "logs cleared" audit entry.
// Both the deletion and the new audit log are performed within a single transaction.
func (s *DeveloperService) DeleteLogs(ctx context.Context, ipAddress *string, userAgent string) error {
	devLog.DebugContext(ctx, "deleting logs")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if err := s.developerRepo.WithTx(tx).DeleteLogs(ctx); err != nil {
		return fmt.Errorf("failed to delete logs: %w", err)
	}
	resetLog := model.Log{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Level:     "INFO",
		Category:  "DEVELOPER",
		Message:   "All logs cleared by user",
		Source:    "DeleteLogs",
		UserAgent: userAgent,
	}

	if ipAddress != nil {
		resetLog.IPAddress = *ipAddress
	}
	if err := s.developerRepo.WithTx(tx).AddLog(ctx, resetLog); err != nil {
		return fmt.Errorf("failed to add log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	devLog.InfoContext(ctx, "logs deleted")
	return nil
}

// ImportFundPrices parses a CSV file and upserts fund prices for the given fund.
// Validates that the fund exists, the file is valid CSV with required headers,
// and each row has a parseable date and positive price.
// Triggers materialized view regeneration from the earliest imported date (Issue #35, Edge Case 9).
//
//nolint:gocyclo // CSV parsing + validation + batch insert + materialized invalidation
func (s *DeveloperService) ImportFundPrices(ctx context.Context, fundID string, content []byte) (int, error) {
	devLog.DebugContext(ctx, "importing fund prices from CSV", "fundID", fundID)
	if _, err := s.fundRepo.GetFund(fundID); err != nil {
		return 0, apperrors.ErrFundNotFound
	}

	headers, rows, err := parseCSV(content)
	if err != nil {
		return 0, err
	}
	if err := validateCSVHeaders(headers, []string{"date", "price"}); err != nil {
		return 0, err
	}

	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[h] = i
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	count := 0
	var earliestDate time.Time
	for i, row := range rows {
		rowNum := i + 2 // 1-indexed, row 1 is headers

		date, err := time.Parse("2006-01-02", strings.TrimSpace(row[colIdx["date"]]))
		if err != nil {
			return 0, fmt.Errorf("row %d: invalid date %q: %w", rowNum, row[colIdx["date"]], err)
		}

		price, err := strconv.ParseFloat(strings.TrimSpace(row[colIdx["price"]]), 64)
		if err != nil || price <= 0 {
			return 0, fmt.Errorf("row %d: price must be a positive number, got %q", rowNum, row[colIdx["price"]])
		}

		if earliestDate.IsZero() || date.Before(earliestDate) {
			earliestDate = date
		}

		fp := model.FundPrice{
			ID:     uuid.NewString(),
			FundID: fundID,
			Date:   date,
			Price:  price,
		}
		if err := s.fundRepo.WithTx(tx).UpdateFundPrice(ctx, fp); err != nil {
			return 0, fmt.Errorf("row %d: failed to upsert fund price: %w", rowNum, err)
		}
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no data rows found in CSV")
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil && earliestDate != (time.Time{}) {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), earliestDate, nil, fundID, ""); err != nil {
				devLog.Warn("failed to regenerate materialized table after price import", "error", err)
			}
		}()
	}

	devLog.InfoContext(ctx, "fund prices imported", "fundID", fundID, "count", count)
	return count, nil
}

// transactionRow holds validated fields parsed from a single CSV transaction row.
type transactionRow struct {
	date         time.Time
	txType       string
	shares       float64
	costPerShare float64
}

// validateTransactionRow parses and validates a single CSV data row for transaction imports.
// Returns a transactionRow with the parsed values, or an error describing the invalid field.
func validateTransactionRow(row []string, colIdx map[string]int, rowNum int) (transactionRow, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(row[colIdx["date"]]))
	if err != nil {
		return transactionRow{}, fmt.Errorf("row %d: invalid date %q: %w", rowNum, row[colIdx["date"]], err)
	}

	txType := strings.TrimSpace(strings.ToLower(row[colIdx["type"]]))
	if txType == "" {
		return transactionRow{}, fmt.Errorf("row %d: type is required", rowNum)
	}
	if !model.ValidTransactionTypes[model.TransactionType(txType)] {
		return transactionRow{}, fmt.Errorf("row %d: type must be one of buy, sell, dividend, fee; got %q", rowNum, txType)
	}

	shares, err := strconv.ParseFloat(strings.TrimSpace(row[colIdx["shares"]]), 64)
	if err != nil || shares <= 0 {
		return transactionRow{}, fmt.Errorf("row %d: shares must be a positive number, got %q", rowNum, row[colIdx["shares"]])
	}

	costPerShare, err := strconv.ParseFloat(strings.TrimSpace(row[colIdx["cost_per_share"]]), 64)
	if err != nil || costPerShare < 0 {
		return transactionRow{}, fmt.Errorf("row %d: cost_per_share must be a non-negative number, got %q", rowNum, row[colIdx["cost_per_share"]])
	}

	return transactionRow{
		date:         date,
		txType:       txType,
		shares:       shares,
		costPerShare: costPerShare,
	}, nil
}

// ImportTransactions parses a CSV file and inserts transactions for the given portfolio-fund.
// Validates that the portfolio-fund relationship exists, the file is valid CSV with required
// headers, and each row has valid values.
// Triggers materialized view regeneration from the earliest imported date (Issue #35, Edge Case 8).
//
//nolint:gocyclo // CSV parsing + validation + batch insert + materialized invalidation
func (s *DeveloperService) ImportTransactions(ctx context.Context, portfolioFundID string, content []byte) (int, error) {
	devLog.DebugContext(ctx, "importing transactions from CSV", "portfolioFundID", portfolioFundID)
	if _, err := s.pfRepo.GetPortfolioFund(portfolioFundID); err != nil {
		return 0, apperrors.ErrPortfolioFundNotFound
	}

	headers, rows, err := parseCSV(content)
	if err != nil {
		return 0, err
	}
	if err := validateCSVHeaders(headers, []string{"date", "type", "shares", "cost_per_share"}); err != nil {
		return 0, err
	}

	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[h] = i
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	count := 0
	var earliestDate time.Time
	for i, row := range rows {
		rowNum := i + 2

		parsed, err := validateTransactionRow(row, colIdx, rowNum)
		if err != nil {
			return 0, err
		}

		if earliestDate.IsZero() || parsed.date.Before(earliestDate) {
			earliestDate = parsed.date
		}

		t := &model.Transaction{
			ID:              uuid.NewString(),
			PortfolioFundID: portfolioFundID,
			Date:            parsed.date,
			Type:            parsed.txType,
			Shares:          parsed.shares,
			CostPerShare:    parsed.costPerShare,
			CreatedAt:       time.Now().UTC(),
		}
		if err := s.transactionRepo.WithTx(tx).InsertTransaction(ctx, t); err != nil {
			return 0, fmt.Errorf("row %d: failed to insert transaction: %w", rowNum, err)
		}
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no data rows found in CSV")
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil && !earliestDate.IsZero() {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), earliestDate, nil, "", portfolioFundID); err != nil {
				devLog.Warn("failed to regenerate materialized table after transaction import", "error", err)
			}
		}()
	}

	devLog.InfoContext(ctx, "transactions imported", "portfolioFundID", portfolioFundID, "count", count)
	return count, nil
}

// parseCSV strips a UTF-8 BOM if present and returns parsed CSV records.
// The first returned slice is the header row; the rest are data rows.
func parseCSV(content []byte) ([]string, [][]string, error) {
	// Strip UTF-8 BOM (EF BB BF) to avoid header corruption
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})

	r := csv.NewReader(bytes.NewReader(content))
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CSV format: %w", err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("%w: CSV file is empty", apperrors.ErrInvalidCSVHeaders)
	}

	headers := make([]string, len(records[0]))
	for i, h := range records[0] {
		headers[i] = strings.TrimSpace(strings.ToLower(h))
	}

	return headers, records[1:], nil
}

// validateCSVHeaders checks that all required headers are present.
func validateCSVHeaders(headers []string, required []string) error {
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[h] = true
	}
	var missing []string
	for _, r := range required {
		if !headerSet[r] {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing required CSV columns: %s", apperrors.ErrInvalidCSVHeaders, strings.Join(missing, ", "))
	}
	return nil
}
