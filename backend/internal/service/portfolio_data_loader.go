package service

import (
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var dlLog = logging.NewLogger("system")

// DataLoaderService centralizes the loading of all data required for portfolio calculations.
// This service eliminates code duplication by providing a single point for batch-loading
// transactions, dividends, fund prices, and realized gains across multiple portfolios.
//
// It coordinates between multiple repositories and services to gather complete datasets
// needed for portfolio history and fund metrics calculations.
type DataLoaderService struct {
	pfRepo                  *repository.PortfolioFundRepository
	fundRepo                *repository.FundRepository
	transactionService      *TransactionService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
}

// DataLoaderServiceOption is a functional option for configuring a DataLoaderService.
// Pass one or more options to NewDataLoaderService to inject dependencies selectively.
type DataLoaderServiceOption func(*DataLoaderService)

// DataLoaderWithPortfolioFundRepository injects the PortfolioFundRepository dependency.
func DataLoaderWithPortfolioFundRepository(r *repository.PortfolioFundRepository) DataLoaderServiceOption {
	return func(s *DataLoaderService) { s.pfRepo = r }
}

// DataLoaderWithFundRepository injects the FundRepository dependency.
func DataLoaderWithFundRepository(r *repository.FundRepository) DataLoaderServiceOption {
	return func(s *DataLoaderService) { s.fundRepo = r }
}

// DataLoaderWithTransactionService injects the TransactionService dependency.
func DataLoaderWithTransactionService(ss *TransactionService) DataLoaderServiceOption {
	return func(s *DataLoaderService) { s.transactionService = ss }
}

// DataLoaderWithDividendService injects the DividendService dependency.
func DataLoaderWithDividendService(ss *DividendService) DataLoaderServiceOption {
	return func(s *DataLoaderService) { s.dividendService = ss }
}

// DataLoaderWithRealizedGainLossService injects the RealizedGainLossService dependency.
func DataLoaderWithRealizedGainLossService(r *RealizedGainLossService) DataLoaderServiceOption {
	return func(s *DataLoaderService) { s.realizedGainLossService = r }
}

// NewDataLoaderService creates a new DataLoaderService. Pass DataLoaderWith* options to
// inject dependencies. Only the options relevant to the calling context need to be provided;
// unset fields remain nil and will panic if the corresponding method is called.
func NewDataLoaderService(opts ...DataLoaderServiceOption) *DataLoaderService {
	s := &DataLoaderService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// PortfolioData contains all data needed for portfolio calculations.
// This struct aggregates results from multiple data sources to provide a complete
// picture of portfolio state over a given time period.
//
// Fields are organized by scope:
//   - Portfolio-level: PortfolioFunds, PFIDs, FundIDs, OldestTransactionDate
//   - Time-series data: TransactionsByPF, DividendsByPF, FundPricesByFund
//   - Realized gains: RealizedGainsByPortfolio
//   - Mappings: PortfolioFundToPortfolio, PortfolioFundToFund
type PortfolioData struct {
	PortfolioFunds           []model.PortfolioFundResponse
	PFIDs                    []string
	FundIDs                  []string
	OldestTransactionDate    time.Time
	TransactionsByPF         map[string][]model.Transaction
	DividendsByPF            map[string][]model.Dividend
	FundPricesByFund         map[string][]model.FundPrice
	RealizedGainsByPortfolio map[string][]model.RealizedGainLoss
	PortfolioFundToPortfolio map[string]string
	PortfolioFundToFund      map[string]string
}

// MapRealizedGainsByPF transforms portfolio-level realized gains into a map keyed by portfolio fund ID.
// This is useful when you need to associate realized gains with specific funds within a portfolio.
//
// Parameters:
//   - portfolioID: The portfolio ID to filter realized gains for
//
// Returns a map where the key is portfolio_fund_id and the value is a slice of realized gain records
// for that fund. Only gains matching the fund IDs in PortfolioFunds are included.
func (data *PortfolioData) MapRealizedGainsByPF(portfolioID string) map[string][]model.RealizedGainLoss {
	result := make(map[string][]model.RealizedGainLoss)

	for _, entry := range data.RealizedGainsByPortfolio[portfolioID] {
		for _, pf := range data.PortfolioFunds {
			if entry.FundID == pf.FundID {
				result[pf.ID] = append(result[pf.ID], entry)
			}
		}
	}

	return result
}

// LoadForPortfolios loads all data required for portfolio calculations across the given portfolios.
// This method performs batch loading of transactions, dividends, fund prices, and realized gains
// for efficiency, avoiding N+1 query problems.
//
// Data Loading Strategy:
//   - Loads the COMPLETE transaction history from the oldest transaction to endDate
//   - This is necessary because share counts and cost basis depend on all prior transactions
//   - If startDate is before the oldest transaction, it's automatically adjusted
//
// Parameters:
//   - portfolios: Slice of portfolios to load data for (can be one or many)
//   - startDate: Earliest date to include in results (may be adjusted to oldest transaction)
//   - endDate: Latest date to include in results
//
// Returns:
//   - PortfolioData containing all loaded data organized by portfolio fund ID
//   - Empty PortfolioData if no portfolios provided or no portfolio funds exist
//   - Error if any data loading operation fails
//
// Usage:
//
//	portfolios, _ := portfolioService.GetPortfoliosForRequest("some-id")
//	data, err := dataLoaderService.LoadForPortfolios(portfolios, startDate, endDate)
//	if err != nil {
//	    return err
//	}
//	// Use data.TransactionsByPF, data.DividendsByPF, etc.
func (s *DataLoaderService) LoadForPortfolios(
	portfolios []model.Portfolio,
	_, endDate time.Time,
) (*PortfolioData, error) {
	dlLog.Debug("loading data for portfolios", "count", len(portfolios), "endDate", endDate.Format("2006-01-02"))

	if len(portfolios) == 0 {
		return &PortfolioData{}, nil
	}

	portfolioIDs := make([]string, len(portfolios))
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
	}

	// Load portfolio funds for all portfolios
	_, pfToPortfolio, pfToFund, pfIDs, fundIDs, err := s.pfRepo.GetPortfolioFundsOnPortfolioID(portfolios)
	if err != nil {
		return nil, fmt.Errorf("failed to load portfolio funds: %w", err)
	}
	var portfolioFunds []model.PortfolioFundResponse
	if len(portfolios) == 1 {
		portfolioFunds, err = s.pfRepo.GetPortfolioFunds(portfolios[0].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load portfolio fund details: %w", err)
		}
	}

	if len(pfIDs) == 0 {
		return &PortfolioData{
			PortfolioFunds:           portfolioFunds,
			PortfolioFundToPortfolio: pfToPortfolio,
			PortfolioFundToFund:      pfToFund,
		}, nil
	}

	// Get oldest transaction date
	oldestTxDate := s.transactionService.getOldestTransaction(pfIDs)

	// Always load from oldest transaction regardless of the requested startDate.
	// Share counts and cost basis require complete history — loading only from
	// startDate would miss prior buys, making portfolios with no recent transactions
	// appear empty. The display range (startDate/endDate) is applied by callers.
	dataStartDate := oldestTxDate

	dlLog.Debug("data loader starting",
		"portfolios", len(portfolios),
		"portfolio_funds", len(pfIDs),
		"oldest_tx", oldestTxDate.Format("2006-01-02"),
		"data_start", dataStartDate.Format("2006-01-02"),
		"end", endDate.Format("2006-01-02"),
	)

	// Batch load all data
	transactionsByPF, err := s.transactionService.loadTransactions(pfIDs, dataStartDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load transactions: %w", err)
	}

	dividendsByPF, err := s.dividendService.loadDividendPerPF(pfIDs, dataStartDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load dividends: %w", err)
	}

	fundPricesByFund, err := s.fundRepo.GetFundPrice(fundIDs, dataStartDate, endDate, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load fund prices: %w", err)
	}

	realizedGainsByPortfolio, err := s.realizedGainLossService.loadRealizedGainLoss(
		portfolioIDs,
		dataStartDate,
		endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load realized gains: %w", err)
	}

	return &PortfolioData{
		PortfolioFunds:           portfolioFunds,
		PFIDs:                    pfIDs,
		FundIDs:                  fundIDs,
		OldestTransactionDate:    oldestTxDate,
		TransactionsByPF:         transactionsByPF,
		DividendsByPF:            dividendsByPF,
		FundPricesByFund:         fundPricesByFund,
		RealizedGainsByPortfolio: realizedGainsByPortfolio,
		PortfolioFundToPortfolio: pfToPortfolio,
		PortfolioFundToFund:      pfToFund,
	}, nil
}
