package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var matLog = logging.NewLogger("system")

// MaterializedInvalidator defines the interface for invalidating and regenerating materialized data.
// Services that modify source data (transactions, dividends, etc.) depend on this interface
// rather than on *MaterializedService directly, breaking the cyclic dependency.
type MaterializedInvalidator interface {
	RegenerateMaterializedTable(ctx context.Context, startDate time.Time, portfolioIDs []string, fundID, portfolioFundID string) error
}

// TransactionMetrics aggregates calculated metrics from processing transactions for a specific date.
// It is used internally by processTransactionsForDate to return all computed values in a single struct.
type TransactionMetrics struct {
	TotalShares    float64 // Total number of shares held
	TotalCost      float64 // Total cost basis
	TotalValue     float64 // Total market value
	TotalDividends float64 // Total dividend amounts
	TotalFees      float64 // Total fees paid
}

// MaterializedService handles history-related business logic operations.
// It coordinates between materialized views and on-demand calculations to provide
// portfolio and fund history data with fallback capabilities.
type MaterializedService struct {
	db                      *sql.DB
	materializedRepo        *repository.MaterializedRepository
	portfolioRepo           *repository.PortfolioRepository
	fundRepo                *repository.FundRepository
	fundService             *FundService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
	dataLoaderService       *DataLoaderService
	portfolioService        *PortfolioService
	pfRepo                  *repository.PortfolioFundRepository

	// regenMu protects regenInFlight from concurrent access.
	regenMu sync.Mutex
	// regenInFlight tracks which portfolio IDs currently have a background
	// regeneration job running and the startDate they're regenerating from.
	// If a new request arrives with an earlier startDate, the existing entry
	// is updated so the next job picks up the earlier date.
	regenInFlight map[string]time.Time
	// regenWriteMu serializes all materialized regen writes. Acquired inside
	// RegenerateMaterializedTable so both write-path hooks and read-path
	// fallback are serialized. SQLite only supports one writer at a time.
	regenWriteMu sync.Mutex
}

// MaterializedServiceOption is a functional option for configuring a MaterializedService.
// Pass one or more options to NewMaterializedService to inject dependencies selectively.
type MaterializedServiceOption func(*MaterializedService)

// MaterializedWithMaterializedRepository injects the MaterializedRepository dependency.
func MaterializedWithMaterializedRepository(r *repository.MaterializedRepository) MaterializedServiceOption {
	return func(s *MaterializedService) { s.materializedRepo = r }
}

// MaterializedWithPortfolioRepository injects the PortfolioRepository dependency.
func MaterializedWithPortfolioRepository(r *repository.PortfolioRepository) MaterializedServiceOption {
	return func(s *MaterializedService) { s.portfolioRepo = r }
}

// MaterializedWithFundRepository injects the FundRepository dependency.
func MaterializedWithFundRepository(r *repository.FundRepository) MaterializedServiceOption {
	return func(s *MaterializedService) { s.fundRepo = r }
}

// MaterializedWithFundService injects the FundService dependency.
func MaterializedWithFundService(ss *FundService) MaterializedServiceOption {
	return func(s *MaterializedService) { s.fundService = ss }
}

// MaterializedWithDividendService injects the DividendService dependency.
func MaterializedWithDividendService(ss *DividendService) MaterializedServiceOption {
	return func(s *MaterializedService) { s.dividendService = ss }
}

// MaterializedWithRealizedGainLossService injects the RealizedGainLossService dependency.
func MaterializedWithRealizedGainLossService(ss *RealizedGainLossService) MaterializedServiceOption {
	return func(s *MaterializedService) { s.realizedGainLossService = ss }
}

// MaterializedWithDataLoaderService injects the DataLoaderService dependency.
func MaterializedWithDataLoaderService(ss *DataLoaderService) MaterializedServiceOption {
	return func(s *MaterializedService) { s.dataLoaderService = ss }
}

// MaterializedWithPortfolioService injects the PortfolioService dependency.
func MaterializedWithPortfolioService(ss *PortfolioService) MaterializedServiceOption {
	return func(s *MaterializedService) { s.portfolioService = ss }
}

// MaterializedWithPortfolioFundRepository injects the PortfolioFundRepository dependency.
func MaterializedWithPortfolioFundRepository(r *repository.PortfolioFundRepository) MaterializedServiceOption {
	return func(s *MaterializedService) { s.pfRepo = r }
}

// NewMaterializedService creates a new MaterializedService. Pass MaterializedWith* options to
// inject dependencies. Only the options relevant to the calling context need to be provided;
// unset fields remain nil and will panic if the corresponding method is called.
func NewMaterializedService(db *sql.DB, opts ...MaterializedServiceOption) *MaterializedService {
	s := &MaterializedService{db: db}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// =============================================================================
// PORTFOLIO HISTORY METHODS
// =============================================================================

// GetPortfolioHistoryMaterialized retrieves daily portfolio valuations from the materialized view table.
//
// This method provides significantly faster performance compared to GetPortfolioHistory() by querying
// pre-calculated daily snapshots instead of recomputing values from raw transactions, dividends, and prices.
//
// The method performs the following:
//  1. Resolves portfolio(s) from the ID parameter using GetPortfoliosForRequest
//  2. Queries the materialized view (delegates to materializedRepo.GetMaterializedHistory)
//  3. Groups results by date using a callback pattern
//  4. Transforms grouped records into PortfolioSummary structs with portfolio metadata
//
// Performance Characteristics:
//   - 1 year (365 days): ~3-5ms (compared to ~50ms for on-demand calculation)
//   - 5 years (1825 days): ~8-10ms (compared to ~50ms for on-demand calculation)
//   - 6-16x faster than on-demand calculation
//
// Parameters:
//   - requestedStartDate: First date to include in returned results
//   - requestedEndDate: Last date to include in returned results
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//
// Returns:
// A slice of PortfolioHistory structs, one per date, each containing portfolio summaries for that date.
func (s *MaterializedService) GetPortfolioHistoryMaterialized(requestedStartDate, requestedEndDate time.Time, portfolioID string) ([]model.PortfolioHistory, error) {
	matLog.Debug("retrieving portfolio history from materialized view", "portfolioID", portfolioID, "startDate", requestedStartDate.Format("2006-01-02"), "endDate", requestedEndDate.Format("2006-01-02"))

	portfolios, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("get portfolios: %w", err)
	}

	portfolioIDs := make([]string, len(portfolios))
	portfolioNames := make(map[string]string, len(portfolios))
	portfolioDescription := make(map[string]string, len(portfolios))
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
		portfolioNames[p.ID] = p.Name
		portfolioDescription[p.ID] = p.Description
	}
	historyMap := make(map[string][]model.PortfolioHistoryMaterialized)

	err = s.materializedRepo.GetMaterializedHistory(
		portfolioIDs,
		requestedStartDate,
		requestedEndDate,
		func(record model.PortfolioHistoryMaterialized) error {
			historyKey := record.Date.Format("2006-01-02")
			historyMap[historyKey] = append(historyMap[historyKey], record)
			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("get materialized history: %w", err)
	}

	// Iterate over actual results (sorted keys) instead of every calendar day
	dateKeys := make([]string, 0, len(historyMap))
	for key := range historyMap {
		dateKeys = append(dateKeys, key)
	}
	sort.Strings(dateKeys)

	result := make([]model.PortfolioHistory, 0, len(dateKeys))
	for _, historyKey := range dateKeys {
		records := historyMap[historyKey]

		summaries := make([]model.PortfolioSummary, len(records))
		for i, record := range records {
			summaries[i] = model.PortfolioSummary{
				ID:                      record.PortfolioID,
				Name:                    portfolioNames[record.PortfolioID],
				Description:             portfolioDescription[record.PortfolioID],
				TotalValue:              record.Value,
				TotalCost:               record.Cost,
				TotalDividends:          record.TotalDividends,
				TotalUnrealizedGainLoss: record.UnrealizedGain,
				TotalRealizedGainLoss:   record.RealizedGain,
				TotalSaleProceeds:       record.TotalSaleProceeds,
				TotalOriginalCost:       record.TotalOriginalCost,
				TotalGainLoss:           record.TotalGainLoss,
				IsArchived:              record.IsArchived,
			}
		}

		result = append(result, model.PortfolioHistory{
			Date:       historyKey,
			Portfolios: summaries,
		})
	}
	return result, nil
}

// GetPortfolioHistory retrieves daily portfolio valuations for the requested date range.
// This method orchestrates the calculation of portfolio metrics for each day, providing
// time-series data showing how portfolio value evolved over the period.
//
// Calculation Pipeline:
//  1. Resolves portfolio(s) from the ID parameter
//  2. Batch-loads all required data (transactions, dividends, prices, realized gains)
//  3. Adjusts display date range to fit within available data boundaries
//  4. Groups transactions and dividends by portfolio for efficient calculations
//  5. Calculates daily portfolio summaries across the date range
//
// Data Loading Strategy:
// The method loads COMPLETE transaction history from the oldest transaction to present,
// regardless of the requested date range. This is necessary because share counts and
// cost basis depend on all prior transactions. However, only dates within the requested
// range are included in the returned results.
//
// The actual daily calculations are delegated to helper functions in materialized_helpers.go.
// This method focuses on orchestration and data preparation.
//
// Parameters:
//   - requestedStartDate: First date to include in returned results
//   - requestedEndDate: Last date to include in returned results
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//
// Returns a slice of PortfolioHistory, one entry per date, each containing portfolio
// summaries with metrics like TotalValue, TotalCost, TotalGainLoss, etc.
func (s *MaterializedService) GetPortfolioHistory(requestedStartDate, requestedEndDate time.Time, portfolioID string) ([]model.PortfolioHistory, error) {
	matLog.Debug("calculating portfolio history on-demand", "portfolioID", portfolioID, "startDate", requestedStartDate.Format("2006-01-02"), "endDate", requestedEndDate.Format("2006-01-02"))

	portfolios, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("get portfolios: %w", err)
	}

	data, err := s.dataLoaderService.LoadForPortfolios(portfolios, requestedStartDate, requestedEndDate)
	if err != nil {
		return nil, fmt.Errorf("load portfolio data: %w", err)
	}

	displayStart, displayEnd := s.calculateDisplayDateRange(
		requestedStartDate,
		requestedEndDate,
		data.OldestTransactionDate,
	)

	txByPortfolio, txByPFByPortfolio := s.groupTransactionsByPortfolio(
		data.TransactionsByPF,
		data.PortfolioFundToPortfolio,
	)

	divByPortfolio, divByPFByPortfolio := s.groupDividendsByPortfolio(
		data.DividendsByPF,
		data.PortfolioFundToPortfolio,
	)

	return s.calculateDailyPortfolioHistory(
		portfolios,
		data,
		txByPortfolio,
		txByPFByPortfolio,
		divByPortfolio,
		divByPFByPortfolio,
		displayStart,
		displayEnd,
	)
}

// GetPortfolioHistoryWithFallback tries to retrieve history from the materialized view,
// falling back to on-demand calculation if the materialized data is incomplete or empty.
//
// Fallback Logic:
// The method checks staleness by comparing source data timestamps (transactions, prices,
// dividends) against the materialized cache's calculated_at and date coverage via checkStaleData.
// If stale, it falls back to on-demand calculation and triggers background regeneration.
//
// This provides the best of both worlds:
//   - Fast materialized view when available and complete (~3-10ms)
//   - Reliable on-demand calculation as fallback when data is stale or missing (~50ms)
//
// Parameters:
//   - startDate: First date to include in results
//   - endDate: Last date to include in results (typically today)
//   - portfolioID: Optional portfolio ID. Empty string returns all active portfolios.
//
// Returns complete portfolio history from startDate to endDate, using the fastest available method.
func (s *MaterializedService) GetPortfolioHistoryWithFallback(
	startDate, endDate time.Time,
	portfolioID string,
) ([]model.PortfolioHistory, error) {
	matLog.Debug("getting portfolio history with fallback", "start_date", startDate.Format("2006-01-02"), "end_date", endDate.Format("2006-01-02"), "portfolio_id", portfolioID)

	portfolios, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("get portfolios: %w", err)
	}
	portfolioIDs := make([]string, len(portfolios))
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
	}

	matLog.Debug("portfolio history: resolved portfolios for range", "count", len(portfolios), "portfolioIDs", portfolioIDs, "startDate", startDate.Format("2006-01-02"), "endDate", endDate.Format("2006-01-02"))

	stale := s.checkStaleData(portfolioIDs, endDate)

	if !stale {
		materialized, mErr := s.GetPortfolioHistoryMaterialized(startDate, endDate, portfolioID)
		if mErr == nil && len(materialized) > 0 {
			matLog.Debug("portfolio history: serving from materialized view", "dates", len(materialized), "summary", summarisePortfolioResult(materialized))
			return materialized, nil
		}
		matLog.Debug("portfolio history: materialized view returned 0 entries, falling back", "error", mErr)
	}

	matLog.Debug("portfolio history: cache stale or empty, falling back to on-demand calculation")
	result, err := s.GetPortfolioHistory(startDate, endDate, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("calculate portfolio history on-demand: %w", err)
	}

	matLog.Debug("portfolio history: on-demand calculation completed", "dates", len(result), "summary", summarisePortfolioResult(result))

	s.triggerBackgroundRegeneration(portfolioIDs, startDate)

	return result, nil
}

// GetPortfolioSummaryWithFallback retrieves portfolio summaries for the latest date only.
// This is optimized for the Summary and GetPortfolio endpoints which only need current state,
// avoiding loading the entire history. If the materialized cache is stale, it falls back to
// computing the full history and returning the last entry.
//
// Parameters:
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//
// Returns portfolio summaries for the most recent available date.
func (s *MaterializedService) GetPortfolioSummaryWithFallback(portfolioID string) ([]model.PortfolioSummary, error) {
	matLog.Debug("retrieving portfolio summary with fallback", "portfolioID", portfolioID)
	portfolios, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("get portfolios: %w", err)
	}
	portfolioIDs := make([]string, len(portfolios))
	portfolioNames := make(map[string]string, len(portfolios))
	portfolioDescription := make(map[string]string, len(portfolios))
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
		portfolioNames[p.ID] = p.Name
		portfolioDescription[p.ID] = p.Description
	}

	endDate := time.Now().UTC()
	stale := s.checkStaleData(portfolioIDs, endDate)

	if !stale {
		var summaries []model.PortfolioSummary
		err := s.materializedRepo.GetPortfolioSummaryLatest(
			portfolioIDs,
			func(record model.PortfolioHistoryMaterialized) error {
				summaries = append(summaries, model.PortfolioSummary{
					ID:                      record.PortfolioID,
					Name:                    portfolioNames[record.PortfolioID],
					Description:             portfolioDescription[record.PortfolioID],
					TotalValue:              record.Value,
					TotalCost:               record.Cost,
					TotalDividends:          record.TotalDividends,
					TotalUnrealizedGainLoss: record.UnrealizedGain,
					TotalRealizedGainLoss:   record.RealizedGain,
					TotalSaleProceeds:       record.TotalSaleProceeds,
					TotalOriginalCost:       record.TotalOriginalCost,
					TotalGainLoss:           record.TotalGainLoss,
					IsArchived:              record.IsArchived,
				})
				return nil
			},
		)
		if err == nil && len(summaries) > 0 {
			matLog.Debug("portfolio summary: serving from materialized view", "portfolios", len(summaries))
			return summaries, nil
		}
		matLog.Debug("portfolio summary: materialized latest returned 0 entries, falling back", "error", err)
	}

	// Fallback: compute full history and take the last entry.
	// Pass zero-time as startDate — LoadForPortfolios unconditionally loads
	// from the oldest transaction date regardless of the requested start.
	matLog.Debug("portfolio summary: cache stale or empty, falling back to on-demand calculation")
	history, err := s.GetPortfolioHistory(time.Time{}, endDate, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("calculate portfolio history on-demand: %w", err)
	}

	// Use the actual earliest date from the result for background regen,
	// not an arbitrary epoch, per the architecture doc's startDate clamping rule.
	if len(history) > 0 {
		if regenStart, parseErr := time.Parse("2006-01-02", history[0].Date); parseErr == nil {
			s.triggerBackgroundRegeneration(portfolioIDs, regenStart)
		}
	}

	if len(history) == 0 {
		return []model.PortfolioSummary{}, nil
	}
	return history[len(history)-1].Portfolios, nil
}

// =============================================================================
// FUND HISTORY METHODS
// =============================================================================

// GetFundHistoryMaterialized retrieves historical fund data from the materialized view.
// Returns time-series data showing individual fund values within a portfolio over time.
//
// The method performs the following:
//  1. Queries the materialized view (delegates to materializedRepo.GetFundHistoryMaterialized)
//  2. Groups results by date using a callback pattern
//  3. Formats the grouped data into chronological response (delegates to formatFundHistoryFromMaterialized)
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve fund history for
//   - startDate: Inclusive start date for the query
//   - endDate: Inclusive end date for the query
//
// Returns a slice of FundHistoryResponse, one entry per date with all funds for that date.
func (s *MaterializedService) GetFundHistoryMaterialized(portfolioID string, startDate, endDate time.Time) ([]model.FundHistoryResponse, error) {
	matLog.Debug("retrieving fund history from materialized view", "portfolioID", portfolioID, "startDate", startDate.Format("2006-01-02"), "endDate", endDate.Format("2006-01-02"))
	fundHistoryByDate := make(map[string][]model.FundHistoryEntry)

	err := s.materializedRepo.GetFundHistoryMaterialized(
		portfolioID,
		startDate,
		endDate,
		func(entry model.FundHistoryEntry) error {
			dateKey := entry.Date.Format("2006-01-02")
			fundHistoryByDate[dateKey] = append(fundHistoryByDate[dateKey], entry)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get fund history materialized: %w", err)
	}

	if len(fundHistoryByDate) > 0 {
		return s.formatFundHistoryFromMaterialized(fundHistoryByDate), nil
	}

	return []model.FundHistoryResponse{}, nil
}

// formatFundHistoryFromMaterialized converts the date-keyed map structure to chronological response format.
// This helper transforms the callback-accumulated map (date string -> fund entries) into a properly
// ordered slice of FundHistoryResponse structs.
//
// The method:
//  1. Extracts all date strings from the map keys
//  2. Sorts them chronologically (string sort works for YYYY-MM-DD format)
//  3. Builds response entries in date order
//
// Parameters:
//   - fundHistoryByDate: Map of date strings to fund entries for that date
//
// Returns a slice of FundHistoryResponse sorted by date, with each entry containing all
// fund metrics for that date. Empty dates (no funds) are excluded from results.
func (s *MaterializedService) formatFundHistoryFromMaterialized(fundHistoryByDate map[string][]model.FundHistoryEntry) []model.FundHistoryResponse {

	dates := make([]string, 0, len(fundHistoryByDate))
	for date := range fundHistoryByDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	var response []model.FundHistoryResponse
	for _, dateStr := range dates {
		funds := fundHistoryByDate[dateStr]
		if len(funds) > 0 {
			response = append(response, model.FundHistoryResponse{
				Date:  funds[0].Date, // All entries have same date
				Funds: funds,
			})
		}
	}

	return response
}

// calculateFundHistoryOnFly computes fund-level history on-demand when materialized data is unavailable.
// This is the fallback calculation path that processes raw transactions, dividends, and prices to
// produce per-fund metrics for each day in the requested range.
//
// Calculation Pipeline:
//  1. Resolves portfolio from the ID parameter
//  2. Batch-loads all required data (transactions, dividends, prices, realized gains)
//  3. Maps realized gains from portfolio level to fund level
//  4. Calculates daily fund metrics across the date range
//
// This method delegates the actual calculations to calculateFundHistoryByDate and its helpers
// in materialized_helpers.go. It focuses on orchestration and data preparation.
//
// Performance Note: This method is slower than the materialized view (~50ms vs ~3-10ms) but
// ensures complete and current data when the materialized view is stale or unavailable.
//
// Parameters:
//   - portfolioID: The portfolio to calculate fund history for
//   - startDate: First date to include in results
//   - endDate: Last date to include in results
//
// Returns a slice of FundHistoryResponse, one per date, with per-fund metrics for that date.
func (s *MaterializedService) calculateFundHistoryOnFly(portfolioID string, startDate, endDate time.Time) ([]model.FundHistoryResponse, error) {
	matLog.Debug("calculating fund history on the fly", "portfolioID", portfolioID, "startDate", startDate.Format("2006-01-02"), "endDate", endDate.Format("2006-01-02"))

	portfolio, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("get portfolios: %w", err)
	}

	data, err := s.dataLoaderService.LoadForPortfolios(portfolio, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("load portfolio data: %w", err)
	}

	if len(data.PFIDs) == 0 {
		return []model.FundHistoryResponse{}, nil
	}

	// Clamp startDate to the oldest transaction date to avoid iterating
	// over thousands of empty calendar days before any data exists.
	if !data.OldestTransactionDate.IsZero() && startDate.Before(data.OldestTransactionDate) {
		startDate = data.OldestTransactionDate
	}
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)

	realizedGainsByPF := data.MapRealizedGainsByPF(portfolioID)

	return s.calculateFundHistoryByDate(
		data,
		realizedGainsByPF,
		startDate,
		endDate,
	)
}

// GetFundHistoryWithFallback tries to retrieve history from the materialized view,
// falling back to on-demand calculation if the materialized data is incomplete or empty.
//
// Fallback Logic:
// The method checks staleness by comparing source data timestamps (transactions, prices,
// dividends) against the materialized cache's calculated_at and date coverage via checkStaleData.
// If stale, it falls back to on-demand calculation and triggers background regeneration.
//
// This provides the best of both worlds:
//   - Fast materialized view when available and complete (~3-10ms)
//   - Reliable on-demand calculation as fallback when data is stale or missing (~50ms)
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve fund history for
//   - startDate: First date to include in results
//   - endDate: Last date to include in results (typically today)
//
// Returns complete fund-level history from startDate to endDate, using the fastest available method.
func (s *MaterializedService) GetFundHistoryWithFallback(
	portfolioID string,
	startDate, endDate time.Time,
) ([]model.FundHistoryResponse, error) {
	matLog.Debug("retrieving fund history with fallback", "portfolioID", portfolioID, "startDate", startDate.Format("2006-01-02"), "endDate", endDate.Format("2006-01-02"))

	stale := s.checkStaleData([]string{portfolioID}, endDate)

	if !stale {
		materialized, mErr := s.GetFundHistoryMaterialized(portfolioID, startDate, endDate)
		if mErr == nil && len(materialized) > 0 {
			matLog.Debug("fund history: serving from materialized view", "entries", len(materialized), "portfolioID", portfolioID)
			return materialized, nil
		}
	}

	matLog.Debug("fund history: cache stale or empty, falling back to on-demand calculation", "portfolioID", portfolioID)
	result, err := s.calculateFundHistoryOnFly(portfolioID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("calculate fund history on-demand: %w", err)
	}

	s.triggerBackgroundRegeneration([]string{portfolioID}, startDate)

	return result, nil
}

// =============================================================================
// STALE DETECTION & BACKGROUND REGENERATION
// =============================================================================

// checkStaleData determines whether the materialized cache is stale for the given portfolios
// by checking all three data sources: transactions, fund prices, and dividends (Issue #35).
//
// The cache is considered stale if:
//  1. No materialized data exists at all (empty cache)
//  2. The materialized date coverage doesn't reach endDate
//  3. Any source data was modified after the last cache calculation
//     - Issue #35 Edge Case 1: Backdated transactions (newer created_at)
//     - Issue #35 Edge Case 2: Price updates without transactions (newer price date)
//     - Issue #35 Edge Case 3: Dividend recording without transactions (newer created_at)
//
// Returns true if the cache is stale and should be regenerated.
func (s *MaterializedService) checkStaleData(portfolioIDs []string, endDate time.Time) bool {
	matDate, matCalc, ok, err := s.materializedRepo.GetLatestMaterializedDate(portfolioIDs)
	if err != nil {
		matLog.Debug("stale check: error getting materialized date, treating as stale", "portfolioIDs", portfolioIDs, "error", err)
		return true // Assume stale on error
	}
	if !ok {
		matLog.Debug("stale check: no materialized data found", "portfolioIDs", portfolioIDs)
		return true // No materialized data at all
	}

	matLog.Debug("stale check: materialized coverage", "portfolioIDs", portfolioIDs, "coverageEnd", matDate.Format("2006-01-02"), "calculatedAt", matCalc.Format(time.RFC3339), "endDate", endDate.Format("2006-01-02"))

	// Check date coverage (truncate to date-only since matDate is midnight UTC)
	if matDate.Before(endDate.Truncate(24 * time.Hour)) {
		matLog.Debug("stale check: stale, coverage insufficient", "portfolioIDs", portfolioIDs, "coverageEnd", matDate.Format("2006-01-02"), "needed", endDate.Format("2006-01-02"))
		return true
	}

	latestTxn, latestPrice, latestDiv, err := s.materializedRepo.GetLatestSourceDates(portfolioIDs)
	if err != nil {
		matLog.Debug("stale check: error getting source dates, treating as stale", "portfolioIDs", portfolioIDs, "error", err)
		return true
	}

	matLog.Debug("stale check: source dates", "portfolioIDs", portfolioIDs, "latestTxn", latestTxn.Format(time.RFC3339), "latestPrice", latestPrice.Format("2006-01-02"), "latestDiv", latestDiv.Format(time.RFC3339))

	// Transaction created_at is a datetime - compare directly against calculated_at
	if !latestTxn.IsZero() && latestTxn.After(matCalc) {
		matLog.Debug("stale check: stale, latest txn after calculated_at", "portfolioIDs", portfolioIDs, "latestTxn", latestTxn.Format(time.RFC3339), "calculatedAt", matCalc.Format(time.RFC3339))
		return true
	}

	// Price date is a date - if latest price date > materialized date coverage, it's stale
	if !latestPrice.IsZero() && latestPrice.After(matDate) {
		matLog.Debug("stale check: stale, latest price after materialized coverage", "portfolioIDs", portfolioIDs, "latestPrice", latestPrice.Format("2006-01-02"), "coverageEnd", matDate.Format("2006-01-02"))
		return true
	}

	// Dividend created_at is a datetime - compare against calculated_at
	if !latestDiv.IsZero() && latestDiv.After(matCalc) {
		matLog.Debug("stale check: stale, latest dividend after calculated_at", "portfolioIDs", portfolioIDs, "latestDiv", latestDiv.Format(time.RFC3339), "calculatedAt", matCalc.Format(time.RFC3339))
		return true
	}

	matLog.Debug("stale check: fresh", "portfolioIDs", portfolioIDs)
	return false
}

// triggerBackgroundRegeneration starts a background goroutine to regenerate the
// materialized table for the given portfolios. Uses regenInFlight to prevent
// duplicate jobs for the same portfolio.
//
// If a regeneration job is already running for a portfolio and the new request has
// an earlier startDate, the tracked date is updated. The running job will finish,
// then a follow-up job will be launched covering the earlier date range.
// If the new request has a later or equal startDate, it is dropped (already covered).
func (s *MaterializedService) triggerBackgroundRegeneration(portfolioIDs []string, startDate time.Time) {
	s.regenMu.Lock()
	if s.regenInFlight == nil {
		s.regenInFlight = make(map[string]time.Time)
	}

	var toStart []string
	for _, pid := range portfolioIDs {
		existing, running := s.regenInFlight[pid]
		if running {
			if startDate.Before(existing) {
				matLog.Debug("regen: superseding with earlier date", "portfolioID", pid, "newDate", startDate.Format("2006-01-02"), "oldDate", existing.Format("2006-01-02"))
				s.regenInFlight[pid] = startDate
			} else {
				matLog.Debug("regen: skipped, already in flight", "portfolioID", pid, "from", existing.Format("2006-01-02"))
			}
			continue
		}
		s.regenInFlight[pid] = startDate
		toStart = append(toStart, pid)
	}
	s.regenMu.Unlock()

	for _, pid := range toStart {
		matLog.Debug("regen: queued", "portfolioID", pid, "from", startDate.Format("2006-01-02"))
		go s.runRegenLoop(pid)
	}
}

// maxRegenRetries is the maximum number of consecutive failures before runRegenLoop
// gives up. This prevents infinite retry loops on persistent errors.
const maxRegenRetries = 3

// runRegenLoop runs regeneration for a portfolio, then checks if a follow-up job
// with an earlier startDate was requested while it was running. Repeats until no
// earlier date is pending. Aborts after maxRegenRetries consecutive failures.
func (s *MaterializedService) runRegenLoop(portfolioID string) {
	failures := 0
	for {
		s.regenMu.Lock()
		startDate, ok := s.regenInFlight[portfolioID]
		if !ok {
			s.regenMu.Unlock()
			return
		}
		s.regenMu.Unlock()

		matLog.Debug("regen: starting", "portfolioID", portfolioID, "from", startDate.Format("2006-01-02"))
		start := time.Now()
		err := s.RegenerateMaterializedTable(
			context.Background(), startDate, []string{portfolioID}, "", "",
		)
		if err != nil {
			matLog.Warn("regen: failed", "portfolioID", portfolioID, "duration", time.Since(start).Round(time.Millisecond), "error", err)
			failures++
		} else {
			matLog.Info("regen: completed", "portfolioID", portfolioID, "duration", time.Since(start).Round(time.Millisecond))
			failures = 0
		}

		if failures >= maxRegenRetries {
			matLog.Warn("regen: aborting after consecutive failures", "portfolioID", portfolioID, "failures", failures)
			s.regenMu.Lock()
			delete(s.regenInFlight, portfolioID)
			s.regenMu.Unlock()
			return
		}

		s.regenMu.Lock()
		current := s.regenInFlight[portfolioID]
		if !current.Before(startDate) {
			// No earlier date was requested while we were running — done
			delete(s.regenInFlight, portfolioID)
			s.regenMu.Unlock()
			return
		}
		// An earlier date was requested — loop again with the new date
		matLog.Debug("regen: follow-up needed", "portfolioID", portfolioID, "newFrom", current.Format("2006-01-02"), "previousFrom", startDate.Format("2006-01-02"))
		s.regenMu.Unlock()
	}
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// processTransactionsForDate calculates portfolio metrics as of the specified date.
// This is a local helper that delegates to FundService for per-fund calculations.
func (s *MaterializedService) processTransactionsForDate(transactionsMap map[string][]model.Transaction, dividendShares map[string]float64, fundMapping map[string]string, fundPriceByFund map[string][]model.FundPrice, date time.Time) (TransactionMetrics, error) {
	if len(transactionsMap) == 0 {
		return TransactionMetrics{}, nil
	}
	var totalShares, totalCost, totalValue, totalDividends, totalFees float64
	for pfID, transactions := range transactionsMap {
		fundID := fundMapping[pfID]
		prices := fundPriceByFund[fundID]

		fundMetrics, err := s.fundService.calculateFundMetrics(
			pfID, fundID, date, transactions, dividendShares[pfID], prices, false)

		if err != nil {
			return TransactionMetrics{}, fmt.Errorf("calculate fund metrics: %w", err)
		}

		totalValue += fundMetrics.Value
		totalShares += fundMetrics.Shares
		totalCost += fundMetrics.Cost
		totalDividends += fundMetrics.Dividend
		totalFees += fundMetrics.Fees
	}

	totalShares = max(0, totalShares)
	totalCost = max(0, totalCost)
	totalValue = max(0, totalValue)
	totalDividends = max(0, totalDividends)
	totalFees = max(0, totalFees)

	return TransactionMetrics{
		TotalShares:    totalShares,
		TotalCost:      totalCost,
		TotalValue:     totalValue,
		TotalDividends: totalDividends,
		TotalFees:      totalFees,
	}, nil
}

// RegenerateMaterializedTable recalculates and replaces materialized view entries from startDate
// forward. Portfolio scope is determined by the first non-empty/non-nil parameter:
//   - portfolioIDs: used directly
//   - fundID: resolved to all portfolios holding that fund
//   - portfolioFundID: resolved to the owning portfolio
//
// All calls are serialized via regenWriteMu because SQLite supports only one concurrent writer;
// without this, both write-path hooks and read-path fallback goroutines would cause SQLITE_BUSY
// errors.
//
//nolint:gocyclo // Core regen: resolve portfolios, calculate, collect pfIDs, invalidate+insert in tx
func (s *MaterializedService) RegenerateMaterializedTable(ctx context.Context, startDate time.Time, portfolioIDs []string, fundID, portfolioFundID string) error {
	matLog.DebugContext(ctx, "regenerating materialized table", "startDate", startDate.Format("2006-01-02"), "portfolioIDs", portfolioIDs, "fundID", fundID, "portfolioFundID", portfolioFundID)
	s.regenWriteMu.Lock()
	defer s.regenWriteMu.Unlock()

	if len(portfolioIDs) == 0 {
		if fundID != "" {
			pfs, err := s.pfRepo.GetPortfolioFundsbyFundID(fundID)
			if err != nil {
				return fmt.Errorf("get portfolio funds by fund ID: %w", err)
			}
			seen := make(map[string]bool)
			for _, v := range pfs {
				if !seen[v.PortfolioID] {
					portfolioIDs = append(portfolioIDs, v.PortfolioID)
					seen[v.PortfolioID] = true
				}
			}
		} else if portfolioFundID != "" {
			pf, err := s.pfRepo.GetPortfolioFund(portfolioFundID)
			if err != nil {
				return fmt.Errorf("get portfolio fund: %w", err)
			}
			portfolioIDs = append(portfolioIDs, pf.PortfolioID)
		} else {
			return fmt.Errorf("RegenerateMaterializedTable: at least one of portfolioIDs, fundID, or portfolioFundID must be provided")
		}
	}

	// Calculate new entries before starting the transaction (read-heavy, no writes)
	endDate := time.Now().UTC()
	var allEntries []model.FundHistoryResponse

	for _, pid := range portfolioIDs {
		entries, err := s.calculateFundHistoryOnFly(pid, startDate, endDate)
		if err != nil {
			return fmt.Errorf("calculate fund history: %w", err)
		}
		allEntries = append(allEntries, entries...)
	}

	fundHistoryEntries := make([]model.FundHistoryEntry, 0, len(allEntries))
	for _, v := range allEntries {
		// Propagate the response-level date to each fund entry, since
		// calculateFundEntry doesn't set FundHistoryEntry.Date itself.
		for i := range v.Funds {
			v.Funds[i].Date = v.Date
		}
		fundHistoryEntries = append(fundHistoryEntries, v.Funds...)
	}

	pfIDSet := make(map[string]bool)
	for i := range fundHistoryEntries {
		if fundHistoryEntries[i].ID == "" {
			fundHistoryEntries[i].ID = uuid.New().String()
		}
		pfIDSet[fundHistoryEntries[i].PortfolioFundID] = true
	}
	pfIDs := make([]string, 0, len(pfIDSet))
	for id := range pfIDSet {
		pfIDs = append(pfIDs, id)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if err := s.materializedRepo.WithTx(tx).InvalidateMaterializedTable(ctx, startDate, pfIDs); err != nil {
		return fmt.Errorf("invalidate materialized table: %w", err)
	}

	if err := s.materializedRepo.WithTx(tx).InsertMaterializedEntries(ctx, fundHistoryEntries); err != nil {
		return fmt.Errorf("insert materialized entries: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// summarisePortfolioResult builds a compact log string showing each portfolio's name
// and how many date entries it appears in, e.g. "3 portfolios: Savings (366), ISA (366), SIPP (200)".
func summarisePortfolioResult(history []model.PortfolioHistory) string {
	counts := make(map[string]int)   // id → count
	names := make(map[string]string) // id → name
	for _, h := range history {
		for _, p := range h.Portfolios {
			counts[p.ID]++
			names[p.ID] = p.Name
		}
	}
	if len(counts) == 0 {
		return "0 portfolios"
	}
	parts := make([]string, 0, len(counts))
	for id, n := range counts {
		parts = append(parts, fmt.Sprintf("%s (%d dates)", names[id], n))
	}
	sort.Strings(parts)
	return fmt.Sprintf("%d portfolio(s): %s", len(counts), strings.Join(parts, ", "))
}
