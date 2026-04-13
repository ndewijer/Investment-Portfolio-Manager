package service

import (
	"fmt"
	"slices"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// calculateDisplayDateRange adjusts the requested date range to fit within available data boundaries.
// This ensures we don't try to return history for dates before the first transaction or after today.
//
// The method performs two adjustments:
//  1. If requestedStart is before the oldest transaction, move it forward to oldestTx
//  2. If requestedEnd is after today, move it back to today
//
// Both dates are normalized to midnight UTC to match database date storage format.
//
// Parameters:
//   - requestedStart: The start date requested by the user
//   - requestedEnd: The end date requested by the user
//   - oldestTx: The date of the oldest transaction in the portfolios
//
// Returns adjusted (displayStart, displayEnd) that fit within [oldestTx, today].
func (s *MaterializedService) calculateDisplayDateRange(
	requestedStart, requestedEnd, oldestTx time.Time,
) (time.Time, time.Time) {
	displayStart := requestedStart
	displayEnd := requestedEnd

	dataStart := oldestTx
	dataEnd := time.Now().UTC()

	if displayStart.Before(dataStart) {
		displayStart = dataStart
	}
	if displayEnd.After(dataEnd) {
		displayEnd = dataEnd
	}

	displayStart = time.Date(displayStart.Year(), displayStart.Month(), displayStart.Day(), 0, 0, 0, 0, time.UTC)
	displayEnd = time.Date(displayEnd.Year(), displayEnd.Month(), displayEnd.Day(), 0, 0, 0, 0, time.UTC)

	return displayStart, displayEnd
}

// groupTransactionsByPortfolio reorganizes transactions from portfolio-fund level to portfolio level.
// This method creates two views of the same transaction data:
//  1. A flat map where all transactions for a portfolio are in a single slice
//  2. A nested map preserving the portfolio-fund grouping within each portfolio
//
// Parameters:
//   - txByPF: Transactions keyed by portfolio_fund_id
//   - pfToPortfolio: Mapping from portfolio_fund_id to portfolio_id
//
// Returns:
//   - txByPortfolio: All transactions for each portfolio (flat)
//   - txByPFByPortfolio: Transactions grouped by portfolio, then by portfolio_fund
//
// The flat view is useful for finding the oldest transaction date across all funds.
// The nested view is useful for per-fund calculations within a portfolio context.
func (s *MaterializedService) groupTransactionsByPortfolio(
	txByPF map[string][]model.Transaction,
	pfToPortfolio map[string]string,
) (map[string][]model.Transaction, map[string]map[string][]model.Transaction) {

	txByPortfolio := make(map[string][]model.Transaction)
	txByPFByPortfolio := make(map[string]map[string][]model.Transaction)

	for pfID, transactions := range txByPF {
		portfolioID := pfToPortfolio[pfID]

		if txByPFByPortfolio[portfolioID] == nil {
			txByPFByPortfolio[portfolioID] = make(map[string][]model.Transaction)
		}
		txByPFByPortfolio[portfolioID][pfID] = transactions
		txByPortfolio[portfolioID] = append(txByPortfolio[portfolioID], transactions...)
	}

	return txByPortfolio, txByPFByPortfolio
}

// groupDividendsByPortfolio reorganizes dividends from portfolio-fund level to portfolio level.
// This is the dividend equivalent of groupTransactionsByPortfolio, creating both flat and nested views.
//
// Parameters:
//   - divByPF: Dividends keyed by portfolio_fund_id
//   - pfToPortfolio: Mapping from portfolio_fund_id to portfolio_id
//
// Returns:
//   - divByPortfolio: All dividends for each portfolio (flat)
//   - divByPFByPortfolio: Dividends grouped by portfolio, then by portfolio_fund
//
// The flat view is useful for calculating total dividend amounts across all funds.
// The nested view is useful for per-fund dividend calculations.
func (s *MaterializedService) groupDividendsByPortfolio(
	divByPF map[string][]model.Dividend,
	pfToPortfolio map[string]string,
) (map[string][]model.Dividend, map[string]map[string][]model.Dividend) {

	divByPortfolio := make(map[string][]model.Dividend)
	divByPFByPortfolio := make(map[string]map[string][]model.Dividend)

	for pfID, dividend := range divByPF {
		portfolioID := pfToPortfolio[pfID]

		if divByPFByPortfolio[portfolioID] == nil {
			divByPFByPortfolio[portfolioID] = make(map[string][]model.Dividend)
		}
		divByPFByPortfolio[portfolioID][pfID] = dividend
		divByPortfolio[portfolioID] = append(divByPortfolio[portfolioID], dividend...)
	}

	return divByPortfolio, divByPFByPortfolio
}

// calculateDailyPortfolioHistory computes portfolio valuations for each day in the date range.
// This method iterates through every day from the oldest transaction to today, calculating
// portfolio metrics for each date. Only dates within the displayStart/displayEnd range are
// included in the returned results.
//
// The method processes ALL days from oldest transaction to today to ensure accurate share
// counts and cost basis, but filters the results to only return the requested display range.
// This is necessary because each day's calculations depend on all prior days.
//
// Parameters:
//   - portfolios: The portfolios to calculate history for
//   - data: Complete portfolio data including transactions, dividends, prices
//   - txByPortfolio: Flat transaction map (used for oldest date checks)
//   - txByPFByPortfolio: Nested transaction map (used for per-fund calculations)
//   - divByPortfolio: Flat dividend map (used for dividend amount calculations)
//   - divByPFByPortfolio: Nested dividend map (used for per-fund calculations)
//   - displayStart: First date to include in results
//   - displayEnd: Last date to include in results
//
// Returns a slice of PortfolioHistory, one entry per date in the display range, each containing
// portfolio summaries for that date. Days with no portfolio activity are included with zero values.
func (s *MaterializedService) calculateDailyPortfolioHistory(
	portfolios []model.Portfolio,
	data *PortfolioData,
	txByPortfolio map[string][]model.Transaction,
	txByPFByPortfolio map[string]map[string][]model.Transaction,
	divByPortfolio map[string][]model.Dividend,
	divByPFByPortfolio map[string]map[string][]model.Dividend,
	displayStart, displayEnd time.Time,
) ([]model.PortfolioHistory, error) {

	var history []model.PortfolioHistory
	dataEnd := time.Now().UTC()

	for date := data.OldestTransactionDate; !date.After(dataEnd); date = date.AddDate(0, 0, 1) {
		portfolioSummary, err := s.calculatePortfolioSummaryForDate(
			date,
			portfolios,
			data,
			txByPortfolio,
			txByPFByPortfolio,
			divByPortfolio,
			divByPFByPortfolio,
		)
		if err != nil {
			return nil, fmt.Errorf("calculate portfolio summary for date %s: %w", date.Format("2006-01-02"), err)
		}

		// Only include dates in display range
		if (date.After(displayStart) || date.Equal(displayStart)) &&
			(date.Before(displayEnd) || date.Equal(displayEnd)) {
			history = append(history, model.PortfolioHistory{
				Date:       date.Format("2006-01-02"),
				Portfolios: portfolioSummary,
			})
		}
	}

	return history, nil
}

// calculatePortfolioSummaryForDate computes metrics for all portfolios on a specific date.
// This method iterates through each portfolio and calculates its summary metrics, skipping
// portfolios that have no transactions or whose first transaction is after the given date.
//
// The method checks two conditions before processing each portfolio:
//  1. The portfolio has at least one transaction (otherwise skip)
//  2. The oldest transaction date is on or before the given date (otherwise skip)
//
// These checks ensure we don't return data for portfolios with no activity or for dates
// before the portfolio had any transactions.
//
// Parameters:
//   - date: The date to calculate metrics for
//   - portfolios: All portfolios to consider
//   - data: Complete portfolio data
//   - txByPortfolio: Flat transaction map (for oldest date checks)
//   - txByPFByPortfolio: Nested transaction map (for calculations)
//   - divByPortfolio: Flat dividend map (for dividend amounts)
//   - divByPFByPortfolio: Nested dividend map (for calculations)
//
// Returns a slice of PortfolioSummary, one per portfolio that has activity on or before this date.
// Portfolios with no transactions or whose first transaction is after this date are excluded.
func (s *MaterializedService) calculatePortfolioSummaryForDate(
	date time.Time,
	portfolios []model.Portfolio,
	data *PortfolioData,
	txByPortfolio map[string][]model.Transaction,
	txByPFByPortfolio map[string]map[string][]model.Transaction,
	divByPortfolio map[string][]model.Dividend,
	divByPFByPortfolio map[string]map[string][]model.Dividend,
) ([]model.PortfolioSummary, error) {

	summary := make([]model.PortfolioSummary, 0, len(portfolios))

	for _, portfolio := range portfolios {
		if len(txByPortfolio[portfolio.ID]) == 0 {
			matLog.Debug("calculatePortfolioSummaryForDate: skipping portfolio, no transactions loaded", "date", date.Format("2006-01-02"), "portfolioID", portfolio.ID, "portfolioName", portfolio.Name)
			continue
		}

		oldest := slices.MinFunc(txByPortfolio[portfolio.ID], func(a, b model.Transaction) int {
			return a.Date.Compare(b.Date)
		})
		if oldest.Date.After(date) {
			continue
		}

		ps, err := s.calculateSinglePortfolioSummary(
			portfolio,
			date,
			data,
			txByPortfolio[portfolio.ID],
			txByPFByPortfolio[portfolio.ID],
			divByPortfolio[portfolio.ID],
			divByPFByPortfolio[portfolio.ID],
		)
		if err != nil {
			return nil, fmt.Errorf("calculate single portfolio summary: %w", err)
		}

		summary = append(summary, ps)
	}

	return summary, nil
}

// calculateSinglePortfolioSummary computes complete metrics for one portfolio on a specific date.
// This method coordinates the calculation of all portfolio-level aggregates:
//   - Processes dividend shares (accounting for reinvestments)
//   - Calculates transaction metrics (shares, cost, value) for all funds
//   - Aggregates dividend amounts
//   - Aggregates realized gains/losses
//   - Computes total gains (realized + unrealized)
//
// Parameters:
//   - portfolio: The portfolio to calculate summary for
//   - date: The date to calculate as of
//   - data: Complete portfolio data (used for fund mappings and prices)
//   - allTransactions: All transactions for this portfolio (flat)
//   - txByPF: Transactions grouped by portfolio_fund (for per-fund calculations)
//   - allDividends: All dividends for this portfolio (flat)
//   - divByPF: Dividends grouped by portfolio_fund (for per-fund calculations)
//
// Returns a PortfolioSummary with all monetary values rounded to two decimal places.
// The summary includes: TotalValue, TotalCost, TotalDividends, TotalRealizedGainLoss,
// TotalUnrealizedGainLoss, TotalGainLoss, TotalSaleProceeds, and TotalOriginalCost.
func (s *MaterializedService) calculateSinglePortfolioSummary(
	portfolio model.Portfolio,
	date time.Time,
	data *PortfolioData,
	allTransactions []model.Transaction,
	txByPF map[string][]model.Transaction,
	allDividends []model.Dividend,
	divByPF map[string][]model.Dividend,
) (model.PortfolioSummary, error) {

	totalDividendSharesPerPF, err := s.dividendService.processDividendSharesForDate(
		divByPF,
		allTransactions,
		date,
	)
	if err != nil {
		return model.PortfolioSummary{}, fmt.Errorf("process dividend shares: %w", err)
	}

	transactionMetrics, err := s.processTransactionsForDate(
		txByPF,
		totalDividendSharesPerPF,
		data.PortfolioFundToFund,
		data.FundPricesByFund,
		date,
	)
	if err != nil {
		return model.PortfolioSummary{}, fmt.Errorf("process transactions: %w", err)
	}

	// Calculate dividend amount
	totalDividendAmount, err := s.dividendService.processDividendAmountForDate(allDividends, date)
	if err != nil {
		return model.PortfolioSummary{}, fmt.Errorf("process dividends: %w", err)
	}

	// Calculate realized gains
	totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, err := s.realizedGainLossService.processRealizedGainLossForDate(
		data.RealizedGainsByPortfolio[portfolio.ID],
		date,
	)
	if err != nil {
		return model.PortfolioSummary{}, fmt.Errorf("process realized gain loss: %w", err)
	}

	// Build summary with rounding
	return model.PortfolioSummary{
		ID:                      portfolio.ID,
		Name:                    portfolio.Name,
		Description:             portfolio.Description,
		TotalValue:              round(transactionMetrics.TotalValue),
		TotalCost:               round(transactionMetrics.TotalCost),
		TotalDividends:          round(totalDividendAmount),
		TotalUnrealizedGainLoss: round(transactionMetrics.TotalValue - transactionMetrics.TotalCost),
		TotalRealizedGainLoss:   round(totalRealizedGainLoss),
		TotalSaleProceeds:       round(totalSaleProceeds),
		TotalOriginalCost:       round(totalCostBasis),
		TotalGainLoss:           round(totalRealizedGainLoss + (transactionMetrics.TotalValue - transactionMetrics.TotalCost)),
		IsArchived:              portfolio.IsArchived,
	}, nil
}

// calculateFundHistoryByDate computes per-fund metrics for each day in the date range.
// This method iterates through each date and calculates metrics for all funds on that date,
// returning time-series data showing how each fund's value evolved over time.
//
// Unlike portfolio history which aggregates funds to portfolio level, this maintains
// fund-level granularity, useful for detailed portfolio composition analysis.
//
// Parameters:
//   - data: Complete portfolio data including fund details, transactions, dividends, prices
//   - realizedGainsByPF: Map of realized gains keyed by portfolio_fund_id
//   - startDate: First date to include in results
//   - endDate: Last date to include in results
//
// Returns a slice of FundHistoryResponse, one per date, each containing metrics for all
// funds on that date. Days where no funds have data are excluded from the results.
func (s *MaterializedService) calculateFundHistoryByDate(
	data *PortfolioData,
	realizedGainsByPF map[string][]model.RealizedGainLoss,
	startDate, endDate time.Time,
) ([]model.FundHistoryResponse, error) {

	var response []model.FundHistoryResponse

	for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, 1) {
		fundsForDate, err := s.calculateFundsForDate(
			currentDate,
			data,
			realizedGainsByPF,
		)
		if err != nil {
			return nil, fmt.Errorf("calculate funds for date %s: %w", currentDate.Format("2006-01-02"), err)
		}

		if len(fundsForDate) > 0 {
			response = append(response, model.FundHistoryResponse{
				Date:  currentDate,
				Funds: fundsForDate,
			})
		}
	}

	return response, nil
}

// calculateFundsForDate computes metrics for all funds on a specific date.
// This method iterates through each fund in the portfolio data and calculates
// its shares, value, costs, gains, dividends, and fees as of the given date.
//
// Parameters:
//   - date: The date to calculate metrics for
//   - data: Complete portfolio data
//   - realizedGainsByPF: Realized gains keyed by portfolio_fund_id
//
// Returns a slice of FundHistoryEntry, one per fund, with all metric fields populated
// and rounded to two decimal places.
func (s *MaterializedService) calculateFundsForDate(
	date time.Time,
	data *PortfolioData,
	realizedGainsByPF map[string][]model.RealizedGainLoss,
) ([]model.FundHistoryEntry, error) {

	funds := make([]model.FundHistoryEntry, 0, len(data.PortfolioFunds))

	for _, pf := range data.PortfolioFunds {
		entry, err := s.calculateFundEntry(
			pf,
			date,
			data,
			realizedGainsByPF[pf.ID],
		)
		if err != nil {
			return nil, fmt.Errorf("calculate fund entry: %w", err)
		}

		// Skip entries with no meaningful data — zero shares AND no financial activity.
		// This excludes dates before the first buy, but preserves sold-fund rows
		// that carry cumulative realized_gain, dividends, sale_proceeds, or original_cost.
		if entry.Shares == 0 && entry.RealizedGain == 0 && entry.Dividends == 0 &&
			entry.SaleProceeds == 0 && entry.OriginalCost == 0 && entry.Fees == 0 {
			continue
		}

		funds = append(funds, entry)
	}

	return funds, nil
}

// calculateFundEntry computes complete metrics for a single fund on a specific date.
// This method performs the full calculation pipeline for one fund:
//   - Processes dividend shares (accounting for reinvestments)
//   - Calculates fund metrics (shares, price, value, cost, gains)
//   - Aggregates dividend amounts
//   - Aggregates realized gains/losses
//   - Rounds all monetary values to two decimals
//
// Parameters:
//   - pf: The portfolio fund to calculate metrics for
//   - date: The date to calculate as of
//   - data: Complete portfolio data (transactions, dividends, prices)
//   - realizedGains: Realized gains specific to this fund
//
// Returns a FundHistoryEntry with all fields populated: PortfolioFundID, FundID, FundName,
// Shares, Price, Value, Cost, RealizedGain, UnrealizedGain, TotalGainLoss, Dividends, Fees.
// All monetary values are rounded to two decimal places.
func (s *MaterializedService) calculateFundEntry(
	pf model.PortfolioFundResponse,
	date time.Time,
	data *PortfolioData,
	realizedGains []model.RealizedGainLoss,
) (model.FundHistoryEntry, error) {

	// Calculate dividend shares
	dividendSharesMap, err := s.dividendService.processDividendSharesForDate(
		data.DividendsByPF,
		data.TransactionsByPF[pf.ID],
		date,
	)
	if err != nil {
		return model.FundHistoryEntry{}, fmt.Errorf("process dividend shares: %w", err)
	}

	// Calculate fund metrics
	fundMetrics, err := s.fundService.calculateFundMetrics(
		pf.ID,
		pf.FundID,
		date,
		data.TransactionsByPF[pf.ID],
		dividendSharesMap[pf.ID],
		data.FundPricesByFund[pf.FundID],
		false,
	)
	if err != nil {
		return model.FundHistoryEntry{}, fmt.Errorf("calculate fund metrics: %w", err)
	}

	// Calculate dividend amount
	dividendAmount, err := s.dividendService.processDividendAmountForDate(
		data.DividendsByPF[pf.ID],
		date,
	)
	if err != nil {
		return model.FundHistoryEntry{}, fmt.Errorf("process dividends: %w", err)
	}

	// Calculate realized gains
	realizedGain, saleProceeds, costBasis, err := s.realizedGainLossService.processRealizedGainLossForDate(
		realizedGains,
		date,
	)
	if err != nil {
		return model.FundHistoryEntry{}, fmt.Errorf("process realized gain loss: %w", err)
	}

	// Build entry with rounding
	return model.FundHistoryEntry{
		PortfolioFundID: pf.ID,
		FundID:          pf.FundID,
		FundName:        pf.FundName,
		Shares:          round(fundMetrics.Shares),
		Price:           round(fundMetrics.LatestPrice),
		Value:           round(fundMetrics.Value),
		Cost:            round(fundMetrics.Cost),
		RealizedGain:    round(realizedGain),
		UnrealizedGain:  round(fundMetrics.UnrealizedGain),
		TotalGainLoss:   round(fundMetrics.UnrealizedGain + realizedGain),
		Dividends:       round(dividendAmount),
		Fees:            round(fundMetrics.Fees),
		SaleProceeds:    round(saleProceeds),
		OriginalCost:    round(costBasis),
	}, nil
}
