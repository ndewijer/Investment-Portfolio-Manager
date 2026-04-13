package service

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/yahoo"
)

var fundLog = logging.NewLogger("fund")

// FundService handles fund-related business logic operations.
type FundService struct {
	db                      *sql.DB
	fundRepo                *repository.FundRepository
	pfRepo                  *repository.PortfolioFundRepository
	transactionService      *TransactionService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
	dataLoaderService       *DataLoaderService
	portfolioRepo           *repository.PortfolioRepository
	yahooClient             yahoo.Client
	materializedInvalidator MaterializedInvalidator
}

// FundServiceOption is a functional option for configuring a FundService.
// Pass one or more options to NewFundService to inject dependencies selectively.
type FundServiceOption func(*FundService)

// FundWithFundRepo injects the FundRepository dependency.
func FundWithFundRepo(r *repository.FundRepository) FundServiceOption {
	return func(s *FundService) { s.fundRepo = r }
}

// FundWithPortfolioFundRepo injects the PortfolioFundRepository dependency.
func FundWithPortfolioFundRepo(r *repository.PortfolioFundRepository) FundServiceOption {
	return func(s *FundService) { s.pfRepo = r }
}

// FundWithTransactionService injects the TransactionService dependency.
func FundWithTransactionService(ss *TransactionService) FundServiceOption {
	return func(s *FundService) { s.transactionService = ss }
}

// FundWithDividendService injects the DividendService dependency.
func FundWithDividendService(ss *DividendService) FundServiceOption {
	return func(s *FundService) { s.dividendService = ss }
}

// FundWithRealizedGainLossService injects the RealizedGainLossService dependency.
func FundWithRealizedGainLossService(ss *RealizedGainLossService) FundServiceOption {
	return func(s *FundService) { s.realizedGainLossService = ss }
}

// FundWithDataLoaderService injects the DataLoaderService dependency.
func FundWithDataLoaderService(ss *DataLoaderService) FundServiceOption {
	return func(s *FundService) { s.dataLoaderService = ss }
}

// FundWithPortfolioRepo injects the PortfolioRepository dependency.
func FundWithPortfolioRepo(r *repository.PortfolioRepository) FundServiceOption {
	return func(s *FundService) { s.portfolioRepo = r }
}

// FundWithYahooClient injects the Yahoo Finance client dependency.
// Swap for a mock implementation in tests to avoid real network calls.
func FundWithYahooClient(c yahoo.Client) FundServiceOption {
	return func(s *FundService) { s.yahooClient = c }
}

// NewFundService creates a new FundService. Pass FundWith* options to inject dependencies.
// Only the options relevant to the calling context need to be provided; unset fields remain
// nil and will panic if the corresponding method is called — a clear wiring error.
func NewFundService(db *sql.DB, opts ...FundServiceOption) *FundService {
	s := &FundService{db: db}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetMaterializedInvalidator injects the MaterializedInvalidator after construction.
// This breaks the circular initialization order between FundService and MaterializedService.
func (s *FundService) SetMaterializedInvalidator(m MaterializedInvalidator) {
	s.materializedInvalidator = m
}

// GetFund retrieves fund from the database.
// Returns fund metadata including latest prices.
func (s *FundService) GetFund(fundID string) (model.Fund, error) {
	fundLog.Debug("retrieving fund", "fundID", fundID)
	fund, err := s.fundRepo.GetFund(fundID)
	if err != nil {
		return model.Fund{}, fmt.Errorf("get fund: %w", err)
	}
	return fund, nil
}

// GetAllFunds retrieves all funds from the database.
// Returns fund metadata including latest prices for all funds in the system.
func (s *FundService) GetAllFunds() ([]model.Fund, error) {
	fundLog.Debug("retrieving all funds")
	funds, err := s.fundRepo.GetAllFunds()
	if err != nil {
		return nil, fmt.Errorf("get all funds: %w", err)
	}
	return funds, nil
}

// GetSymbol retrieves symbol information by ticker symbol.
// Checks the local symbol_info cache first. On cache miss, queries the Yahoo Finance
// v8 chart endpoint, caches the result, and returns it. Prunes stale unreferenced
// cache entries on each call.
func (s *FundService) GetSymbol(symbol string) (*model.Symbol, error) {
	fundLog.Debug("looking up symbol", "symbol", symbol)
	_, err := s.fundRepo.PruneStaleSymbols()
	if err != nil {
		fundLog.Warn("failed to prune stale symbols", "error", err)
	}

	sym, err := s.fundRepo.GetSymbol(symbol)
	if err == nil {
		return sym, nil
	}

	raw, err := s.yahooClient.QueryYahooFiveDaySymbol(context.Background(), symbol)
	if err != nil {
		return nil, fmt.Errorf("yahoo lookup failed: %w", err)
	}
	chart, err := s.yahooClient.ParseChart(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing yahoo response: %w", err)
	}

	sym = &model.Symbol{
		ID:          uuid.New().String(),
		Symbol:      chart.Symbol,
		Name:        chart.LongName,
		Exchange:    chart.ExchangeName,
		Currency:    chart.Currency,
		LastUpdated: time.Now().UTC(),
		DataSource:  "yahoo",
		IsValid:     true,
	}

	if err := s.fundRepo.UpsertSymbol(sym); err != nil {
		return nil, fmt.Errorf("caching symbol: %w", err)
	}

	return sym, nil
}

// GetAllPortfolioFundListings retrieves all portfolio-fund relationships with basic metadata.
// Returns a listing of funds across all portfolios (non-archived) with portfolio and fund names.
// Used for the GET /api/portfolio/funds endpoint.
func (s *FundService) GetAllPortfolioFundListings() ([]model.PortfolioFundListing, error) {
	fundLog.Debug("retrieving all portfolio fund listings")
	listings, err := s.pfRepo.GetAllPortfolioFundListings()
	if err != nil {
		return nil, fmt.Errorf("get portfolio fund listings: %w", err)
	}
	return listings, nil
}

// GetPortfolioFunds retrieves detailed fund metrics for all funds in a portfolio.
// This method orchestrates the complete calculation pipeline to produce enriched fund data
// with current valuations, gains/losses, dividends, and fees.
//
// Calculation Pipeline:
//  1. Resolves portfolio(s) from the ID parameter (specific portfolio or all active)
//  2. Batch-loads all required data (transactions, dividends, prices, realized gains)
//  3. Maps realized gains from portfolio level to fund level
//  4. Enriches each fund with calculated metrics using historical data
//
// The actual calculations (share counts, cost basis, valuations) are delegated to
// enrichPortfolioFundsWithMetrics and its helpers. This method focuses on orchestration
// and data loading.
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve funds for. Must be non-empty; callers are
//     expected to validate this before calling (the HTTP handler rejects empty IDs).
//
// Returns a slice of PortfolioFund structs with all metric fields populated:
// TotalShares, LatestPrice, AverageCost, TotalCost, CurrentValue, UnrealizedGainLoss,
// RealizedGainLoss, TotalGainLoss, TotalDividends, and TotalFees.
// All monetary values are rounded to two decimal places.
func (s *FundService) GetPortfolioFunds(portfolioID string) ([]model.PortfolioFundResponse, error) {
	fundLog.Debug("retrieving portfolio funds", "portfolioID", portfolioID)

	// Direct repo call (not via PortfolioService) because the handler guarantees a non-empty
	// portfolioID, making the multi-portfolio resolution in GetPortfoliosForRequest unnecessary.
	portfolio, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("get portfolio: %w", err)
	}

	data, err := s.dataLoaderService.LoadForPortfolios([]model.Portfolio{portfolio}, time.Time{}, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("load portfolio data: %w", err)
	}

	if len(data.PFIDs) == 0 {
		return []model.PortfolioFundResponse{}, nil
	}

	// Override fund prices with only the last 30 days. enrichPortfolioFundsWithMetrics
	// uses useLatestPrice=true, so it only needs the most recent price per fund —
	// loading all prices from the oldest transaction wastes I/O.
	recentStart := time.Now().UTC().AddDate(0, -1, 0)
	recentPrices, err := s.LoadFundPrices(data.FundIDs, recentStart, time.Now().UTC(), true)
	if err != nil {
		return nil, fmt.Errorf("failed to load recent fund prices: %w", err)
	}
	data.FundPricesByFund = recentPrices

	realizedGainsByPF := data.MapRealizedGainsByPF(portfolioID)

	return s.enrichPortfolioFundsWithMetrics(data, realizedGainsByPF)
}

// LoadFundPrices retrieves fund prices for the given fund IDs within the specified date range.
// Prices are sorted by date based on the ascending parameter (true=ASC, false=DESC).
// Results are grouped by fund ID, allowing per-fund price lookups.
//
// Parameters:
//   - fundIDs: Slice of fund IDs to retrieve prices for
//   - startDate: Inclusive start date for the price range
//   - endDate: Inclusive end date for the price range
//   - ascending: If true, sort prices oldest-first (ASC); if false, newest-first (DESC)
//
// Returns a map of fundID -> []FundPrice, where prices are sorted according to the ascending parameter.
// ASC order is typically used for date-aware price lookups (getPriceForDate),
// while DESC order is efficient for latest-price queries.
func (s *FundService) LoadFundPrices(fundIDs []string, startDate, endDate time.Time, ascending bool) (map[string][]model.FundPrice, error) {
	fundLog.Debug("loading fund prices", "fundCount", len(fundIDs), "startDate", startDate.Format("2006-01-02"), "endDate", endDate.Format("2006-01-02"))
	prices, err := s.fundRepo.GetFundPrice(fundIDs, startDate, endDate, ascending)
	if err != nil {
		return nil, fmt.Errorf("get fund prices: %w", err)
	}
	return prices, nil
}

// CheckUsage checks if a fund is currently in use by any portfolios.
// A fund is considered "in use" if it has portfolio_fund relationships with transactions.
// This check is critical for data integrity - funds with usage history should not be deleted
// to preserve portfolio history and fund price data for historical calculations.
//
// Returns a FundUsage object:
//   - InUsage: true if the fund has been used in any portfolio (has transactions)
//   - Portfolios: list of portfolios using the fund with transaction counts
//   - Empty Portfolios slice means the fund can be safely deleted
//
// Use case: Call before deletion to prevent losing historical data.
func (s *FundService) CheckUsage(fundID string) (model.FundUsage, error) {
	fundLog.Debug("checking fund usage", "fundID", fundID)
	checkUsage, err := s.pfRepo.CheckUsage(fundID)
	if err != nil {
		return model.FundUsage{}, fmt.Errorf("check fund usage: %w", err)
	}
	var fundUsage model.FundUsage
	if len(checkUsage) == 0 {
		fundUsage.InUsage = false
	} else {
		fundUsage.InUsage = true
		fundUsage.Portfolios = checkUsage
	}

	return fundUsage, nil
}

// CreatePortfolioFund creates a relationship between a portfolio and a fund.
// Validates that both the portfolio and fund exist before creating the relationship.
// This allows a fund to be tracked within a specific portfolio.
func (s *FundService) CreatePortfolioFund(ctx context.Context, req request.CreatePortfolioFundRequest) error {
	fundLog.DebugContext(ctx, "creating portfolio fund", "portfolioID", req.PortfolioID, "fundID", req.FundID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	// Direct repo call via WithTx (not via PortfolioService) to include the existence check
	// within the service-level transaction without passing tx through service boundaries.
	_, err = s.portfolioRepo.WithTx(tx).GetPortfolioOnID(req.PortfolioID)
	if err != nil {
		return fmt.Errorf("get portfolio: %w", err)
	}

	_, err = s.fundRepo.WithTx(tx).GetFund(req.FundID)
	if err != nil {
		return fmt.Errorf("get fund: %w", err)
	}

	if err := s.pfRepo.WithTx(tx).InsertPortfolioFund(ctx, req.PortfolioID, req.FundID); err != nil {
		return fmt.Errorf("failed to create portfolio_fund: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	fundLog.InfoContext(ctx, "portfolio fund created", "portfolioID", req.PortfolioID, "fundID", req.FundID)
	return nil
}

// DeletePortfolioFund removes the relationship between a portfolio and a fund.
// Validates that the portfolio-fund relationship exists before deletion.
// This does not delete the fund itself, only removes it from the portfolio.
func (s *FundService) DeletePortfolioFund(ctx context.Context, pfID string) error {
	fundLog.DebugContext(ctx, "deleting portfolio fund", "portfolioFundID", pfID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	_, err = s.pfRepo.WithTx(tx).GetPortfolioFund(pfID)
	if err != nil {
		return fmt.Errorf("get portfolio fund: %w", err)
	}

	err = s.pfRepo.WithTx(tx).DeletePortfolioFund(ctx, pfID)
	if err != nil {
		return fmt.Errorf("delete portfolio fund: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fundLog.InfoContext(ctx, "portfolio fund deleted", "portfolio_fund_id", pfID)
	return nil
}

// CreateFund creates a new fund with the provided details.
// Generates a new UUID for the fund and inserts it into the database.
//
// Note: Once a fund is used in a portfolio (has transactions), it becomes permanent
// and cannot be deleted. This preserves portfolio history and fund price data.
// Only delete unused funds (e.g., created by mistake).
//
// Parameters:
//   - ctx: Context for the operation
//   - req: CreateFundRequest containing all required fund fields
//
// Returns the created fund with its generated ID, or an error if creation fails.
func (s *FundService) CreateFund(ctx context.Context, req request.CreateFundRequest) (*model.Fund, error) {
	fundLog.DebugContext(ctx, "creating fund", "name", req.Name, "symbol", req.Symbol)
	fund := &model.Fund{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Isin:           req.Isin,
		Symbol:         req.Symbol,
		Exchange:       req.Exchange,
		Currency:       req.Currency,
		InvestmentType: req.InvestmentType,
		DividendType:   req.DividendType,
	}

	if err := s.fundRepo.InsertFund(ctx, fund); err != nil {
		return nil, fmt.Errorf("failed to create fund: %w", err)
	}

	fundLog.InfoContext(ctx, "fund created", "fundID", fund.ID, "symbol", fund.Symbol)
	return fund, nil
}

// UpdateFund updates an existing fund with the provided fields.
// Only provided fields in the request are updated; omitted fields remain unchanged.
// Validates that the fund exists before updating.
//
// Parameters:
//   - ctx: Context for the operation
//   - id: The fund ID to update
//   - req: UpdateFundRequest containing the fields to update
//
// Returns the updated fund or an error if the fund doesn't exist or update fails.
func (s *FundService) UpdateFund(
	ctx context.Context,
	id string,
	req request.UpdateFundRequest,
) (*model.Fund, error) {
	fundLog.DebugContext(ctx, "updating fund", "fundID", id)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	fund, err := s.fundRepo.WithTx(tx).GetFund(id)
	if err != nil {
		return nil, fmt.Errorf("get fund: %w", err)
	}
	if req.Name != nil {
		fund.Name = *req.Name
	}
	if req.Isin != nil {
		fund.Isin = *req.Isin
	}
	if req.Symbol != nil {
		fund.Symbol = *req.Symbol
	}
	if req.Currency != nil {
		fund.Currency = *req.Currency
	}
	if req.Exchange != nil {
		fund.Exchange = *req.Exchange
	}
	if req.InvestmentType != nil {
		fund.InvestmentType = *req.InvestmentType
	}
	if req.DividendType != nil {
		fund.DividendType = *req.DividendType
	}

	if err := s.fundRepo.WithTx(tx).UpdateFund(ctx, &fund); err != nil {
		return nil, fmt.Errorf("failed to update fund: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	fundLog.InfoContext(ctx, "fund updated", "fundID", id)
	return &fund, nil
}

// DeleteFund removes a fund from the database.
// Validates that the fund exists and is not in use before deletion.
// A fund is considered "in use" if it has been associated with any portfolios
// (has transactions). This preserves portfolio history and fund price data.
//
// Parameters:
//   - ctx: Context for the operation
//   - id: The fund ID to delete
//
// Returns:
//   - apperrors.ErrFundNotFound if the fund doesn't exist
//   - apperrors.ErrFundInUse if the fund is being used by portfolios
//   - error if deletion fails
func (s *FundService) DeleteFund(ctx context.Context, id string) error {
	fundLog.DebugContext(ctx, "deleting fund", "fundID", id)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	_, err = s.fundRepo.WithTx(tx).GetFund(id)
	if err != nil {
		return fmt.Errorf("get fund: %w", err)
	}

	usage, err := s.pfRepo.WithTx(tx).CheckUsage(id)
	if err != nil {
		return fmt.Errorf("failed to check fund usage: %w", err)
	}

	if len(usage) > 0 {
		return apperrors.ErrFundInUse
	}

	err = s.fundRepo.WithTx(tx).DeleteFund(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete fund: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fundLog.InfoContext(ctx, "fund deleted", "fundID", id)
	return nil
}

// UpdateCurrentFundPrice fetches and stores the latest available price for a fund.
// This method always targets yesterday's date to ensure we only get final closing prices,
// never provisional intraday data from today's open market. Stock markets provide the
// previous day's close as the most recent complete data point.
//
// The method follows this workflow:
//  1. Validates that the fund exists and has a ticker symbol
//  2. Checks if yesterday's price already exists in the database (early return if found)
//  3. Fetches the last 5 days of price data from Yahoo Finance
//  4. Attempts to extract yesterday's price from the data
//  5. Falls back to the most recent available price if yesterday is not found
//  6. Inserts the price into the database if it doesn't already exist
//
// Duplicate Prevention:
// The method includes multiple safeguards against duplicate insertions:
//   - Early return if yesterday's price already exists
//   - Fallback path checks if the fallback date already exists
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - fundID: The unique identifier of the fund to update
//
// Returns:
//   - FundPrice: The inserted or existing price record
//   - bool: true if a new price was inserted, false if price already existed
//   - error: If the fund doesn't exist, has no symbol, or Yahoo Finance query fails
//
// Note: This method triggers materialized view regeneration after a successful price insert (Issue #35).
func (s *FundService) UpdateCurrentFundPrice(ctx context.Context, fundID string) (model.FundPrice, bool, error) {
	fundLog.DebugContext(ctx, "updating current fund price", "fundID", fundID)
	fund, err := s.GetFund(fundID)
	if err != nil {
		return model.FundPrice{}, false, fmt.Errorf("get fund: %w", err)
	}

	if fund.Symbol == "" {
		return model.FundPrice{}, false, apperrors.ErrInvalidSymbol
	}

	yesterdayDate := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)

	if existingPrice, exists := s.checkExistingPrice(fundID, yesterdayDate); exists {
		return existingPrice, false, nil
	}

	chart, err := s.fetchYahooChart(ctx, fund.Symbol)
	if err != nil {
		return model.FundPrice{}, false, fmt.Errorf("fetch yahoo chart: %w", err)
	}

	if len(chart.Indicators) == 0 {
		return model.FundPrice{}, false, fmt.Errorf("no price data available from Yahoo Finance")
	}

	indicator, ok := chart.GetIndicatorForDate(yesterdayDate)
	if !ok {
		indicator = chart.Indicators[len(chart.Indicators)-1]
		yesterdayDate = indicator.Date.Truncate(24 * time.Hour)

		if existingPrice, exists := s.checkExistingPrice(fundID, yesterdayDate); exists {
			return existingPrice, false, nil
		}
	}

	if indicator.PriceClose <= 0 {
		return model.FundPrice{}, false, fmt.Errorf("invalid price for date %s: %.2f", yesterdayDate.Format("2006-01-02"), indicator.PriceClose)
	}

	fundPrice := model.FundPrice{
		ID:     uuid.New().String(),
		FundID: fund.ID,
		Date:   yesterdayDate,
		Price:  indicator.PriceClose,
	}

	if err = s.fundRepo.InsertFundPrice(ctx, fundPrice); err != nil {
		return model.FundPrice{}, false, fmt.Errorf("insert fund price: %w", err)
	}

	if s.materializedInvalidator != nil {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), fundPrice.Date, nil, fundPrice.FundID, ""); err != nil {
				fundLog.Warn("failed to regenerate materialized table after price update", "error", err)
			}
		}()
	}

	fundLog.InfoContext(ctx, "fund price updated", "fundID", fundID, "date", fundPrice.Date.Format("2006-01-02"), "price", fundPrice.Price)
	return fundPrice, true, nil

}

// checkExistingPrice checks if a price already exists for the given date.
func (s *FundService) checkExistingPrice(fundID string, date time.Time) (model.FundPrice, bool) {
	fundPrices, err := s.fundRepo.GetFundPrice([]string{fundID}, date, date, true)
	if err != nil {
		fundLog.Warn("failed to check existing price", "error", err, "fund_id", fundID)
		return model.FundPrice{}, false
	}

	prices, exists := fundPrices[fundID]
	if exists && len(prices) > 0 {
		return prices[0], true
	}
	return model.FundPrice{}, false
}

// fetchYahooChart fetches and parses Yahoo Finance data for a symbol.
func (s *FundService) fetchYahooChart(ctx context.Context, symbol string) (yahoo.PriceChart, error) {
	raw, err := s.yahooClient.QueryYahooFiveDaySymbol(ctx, symbol)
	if err != nil {
		return yahoo.PriceChart{}, fmt.Errorf("query yahoo five day: %w", err)
	}
	chart, err := s.yahooClient.ParseChart(raw)
	if err != nil {
		return yahoo.PriceChart{}, fmt.Errorf("parse yahoo chart: %w", err)
	}
	return chart, nil
}

// buildMissingDatesMap creates a map of date strings that are missing from the existing prices.
// This helper function reduces cyclomatic complexity in UpdateHistoricalFundPrice.
func (s *FundService) buildMissingDatesMap(existingPrices []model.FundPrice, startDate, endDate time.Time) map[string]bool {
	existingDates := make(map[string]bool)
	for _, fp := range existingPrices {
		existingDates[fp.Date.UTC().Truncate(24*time.Hour).Format("2006-01-02")] = true
	}

	missingDates := make(map[string]bool)
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		key := d.UTC().Truncate(24 * time.Hour).Format("2006-01-02")
		if !existingDates[key] {
			missingDates[key] = true
		}
	}

	return missingDates
}

// filterMissingPrices filters Yahoo Finance indicators to only include dates that are missing.
// This helper function reduces cyclomatic complexity in UpdateHistoricalFundPrice.
func (s *FundService) filterMissingPrices(indicators []yahoo.Indicators, missingDates map[string]bool, fundID string) []model.FundPrice {
	// Use a map keyed by date string to deduplicate: Yahoo Finance can return
	// multiple timestamps for the same trading day (e.g. open and close).
	// Later entries overwrite earlier ones, so we keep the latest timestamp's price.
	byDate := make(map[string]model.FundPrice, len(missingDates))
	for _, v := range indicators {
		sanitizedDate := v.Date.Truncate(24 * time.Hour).Format("2006-01-02")
		if missingDates[sanitizedDate] {
			if v.PriceClose <= 0 {
				continue
			}
			byDate[sanitizedDate] = model.FundPrice{
				ID:     uuid.New().String(),
				FundID: fundID,
				Price:  v.PriceClose,
				Date:   v.Date.Truncate(24 * time.Hour),
			}
		}
	}

	missingFundPrices := make([]model.FundPrice, 0, len(byDate))
	for _, fp := range byDate {
		missingFundPrices = append(missingFundPrices, fp)
	}
	return missingFundPrices
}

// UpdateHistoricalFundPrice backfills missing historical prices for a fund.
// This method identifies all missing price dates from the fund's earliest transaction
// to yesterday, fetches the data from Yahoo Finance, and performs a batch insert of
// all missing prices.
//
// The method follows this workflow:
//  1. Validates that the fund exists and has a ticker symbol
//  2. Retrieves all portfolio_fund relationships for this fund
//  3. Finds the earliest transaction date across all portfolios using this fund
//  4. Queries existing prices in the database for the date range
//  5. Identifies missing dates by comparing all dates in range with existing prices
//  6. Returns early if no missing dates are found (no-op)
//  7. Fetches historical data from Yahoo Finance for the entire date range
//  8. Filters Yahoo data to only include missing dates
//  9. Performs a single batch insert of all missing prices
//
// Efficiency Notes:
// This implementation is more efficient than the Python equivalent as it:
//   - Uses map-based lookups for O(1) date existence checks
//   - Builds all prices in memory before insertion
//   - Performs a single batch insert instead of individual inserts
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - fundID: The unique identifier of the fund to update
//
// Returns:
//   - int: The number of new prices added to the database
//   - error: If the fund doesn't exist, has no symbol, has no transactions,
//     or Yahoo Finance query fails
//
// Note: This method triggers materialized view regeneration after a successful price insert (Issue #35).
//
//nolint:gocyclo,funlen // Multi-step pipeline: validate, load, diff, fetch, filter, insert, invalidate
func (s *FundService) UpdateHistoricalFundPrice(ctx context.Context, fundID string) (int, error) {
	fundLog.DebugContext(ctx, "updating historical fund prices", "fundID", fundID)
	fund, err := s.GetFund(fundID)
	if err != nil {
		return 0, fmt.Errorf("get fund: %w", err)
	}

	if fund.Symbol == "" {
		return 0, apperrors.ErrInvalidSymbol
	}

	portfolioFunds, err := s.pfRepo.GetPortfolioFundsbyFundID(fundID)
	if err != nil {
		return 0, fmt.Errorf("get portfolio funds by fund ID: %w", err)
	}
	if len(portfolioFunds) == 0 {
		return 0, fmt.Errorf("no portfolio funds found for fund %s", fundID)
	}

	pfIDs := make([]string, len(portfolioFunds))
	for i, v := range portfolioFunds {
		pfIDs[i] = v.ID
	}

	oldestDate := s.transactionService.getOldestTransaction(pfIDs)
	if oldestDate.IsZero() {
		return 0, fmt.Errorf("no transactions found for fund %s", fundID)
	}
	now := time.Now().UTC().Truncate(24 * time.Hour)
	yesterdayDate := now.AddDate(0, 0, -1)

	existingPrices, err := s.fundRepo.GetFundPrice([]string{fundID}, oldestDate, now, true)
	if err != nil {
		return 0, fmt.Errorf("get existing fund prices: %w", err)
	}

	missingDates := s.buildMissingDatesMap(existingPrices[fundID], oldestDate, yesterdayDate)
	if len(missingDates) == 0 {
		return 0, nil // nothing to do
	}

	raw, err := s.yahooClient.QueryYahooSymbolByDateRange(ctx, fund.Symbol, oldestDate, yesterdayDate)
	if err != nil {
		return 0, fmt.Errorf("query yahoo by date range: %w", err)
	}
	chart, err := s.yahooClient.ParseChart(raw)
	if err != nil {
		return 0, fmt.Errorf("parse yahoo chart: %w", err)
	}

	missingFundPrices := s.filterMissingPrices(chart.Indicators, missingDates, fundID)
	if len(missingFundPrices) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	err = s.fundRepo.WithTx(tx).InsertFundPrices(ctx, missingFundPrices)
	if err != nil {
		return 0, fmt.Errorf("insert fund prices: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	oldestPrice := slices.MinFunc(missingFundPrices, func(a, b model.FundPrice) int {
		return a.Date.Compare(b.Date)
	})
	if s.materializedInvalidator != nil {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), oldestPrice.Date, nil, oldestPrice.FundID, ""); err != nil {
				fundLog.Warn("failed to regenerate materialized table after historical price update", "error", err)
			}
		}()
	}

	fundLog.InfoContext(ctx, "historical fund prices updated", "fundID", fundID, "pricesAdded", len(missingFundPrices))
	return len(missingFundPrices), nil
}

// UpdateAllFundHistory updates historical price data for all funds in the database.
// It iterates through all funds and attempts to fetch and store missing historical prices.
//
// The function collects both successes and failures for each fund, continuing to process
// remaining funds even if individual updates fail. This ensures maximum data collection
// in a single operation.
//
// Returns:
//   - AllFundUpdateResponse containing detailed results for each fund (successes and errors)
//   - error is nil if at least one fund was successfully updated (partial success)
//   - error is non-nil only if:
//   - No funds exist in the database (returns apperrors.ErrFundNotFound)
//   - GetAllFunds fails
//   - All fund updates failed (zero successes)
//
// The response always includes detailed information about which funds succeeded or failed,
// regardless of whether an error is returned.
func (s *FundService) UpdateAllFundHistory(ctx context.Context) (model.AllFundUpdateResponse, error) {
	fundLog.DebugContext(ctx, "updating all fund history")
	funds, err := s.GetAllFunds()
	if err != nil {
		return model.AllFundUpdateResponse{}, fmt.Errorf("get all funds: %w", err)
	}

	if len(funds) == 0 {
		return model.AllFundUpdateResponse{}, apperrors.ErrFundNotFound
	}
	var fundUpdate model.AllFundUpdateResponse
	var errors []model.UpdatedFundError
	var fundResults []model.UpdatedFund

	for _, f := range funds {
		result, err := s.UpdateHistoricalFundPrice(ctx, f.ID)
		if err != nil {
			fundPriceError := model.UpdatedFundError{
				FundID: f.ID,
				Name:   f.Name,
				Symbol: f.Symbol,
				Error:  err.Error(),
			}
			errors = append(errors, fundPriceError)
			continue
		}

		updateResult := model.UpdatedFund{
			FundID:      f.ID,
			Name:        f.Name,
			Symbol:      f.Symbol,
			PricesAdded: result,
		}

		fundResults = append(fundResults, updateResult)
	}

	fundUpdate.Errors = errors
	fundUpdate.UpdatedFunds = fundResults
	fundUpdate.TotalErrors = len(errors)
	fundUpdate.TotalUpdated = len(fundResults)

	if len(fundResults) == 0 && len(errors) > 0 {
		fundUpdate.Success = false
		return fundUpdate, fmt.Errorf("failed to update any funds: %d errors occurred", len(errors))
	}

	fundUpdate.Success = true
	if len(errors) > 0 {
		fundLog.Warn("fund history update completed with errors", "updated", len(fundResults), "errors", len(errors))
	} else {
		fundLog.InfoContext(ctx, "all fund history updated", "updated", len(fundResults))
	}
	return fundUpdate, nil
}
