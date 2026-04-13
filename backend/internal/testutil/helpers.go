package testutil

import (
	"database/sql"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/ibkr"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/yahoo"
)

// NewTestPortfolioService creates a PortfolioService wired to the provided test database.
func NewTestPortfolioService(t *testing.T, db *sql.DB) *service.PortfolioService {
	t.Helper()

	portfolioRepo := repository.NewPortfolioRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)

	return service.NewPortfolioService(
		db,
		portfolioRepo,
		pfRepo,
	)
}

// NewTestRealizedGainLossService creates a RealizedGainLossService wired to the provided test database.
func NewTestRealizedGainLossService(t *testing.T, db *sql.DB) *service.RealizedGainLossService {
	t.Helper()

	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)

	return service.NewRealizedGainLossService(
		realizedGainLossRepo,
	)
}

// NewTestTransactionService creates a TransactionService wired to the provided test database.
func NewTestTransactionService(t *testing.T, db *sql.DB) *service.TransactionService {
	t.Helper()

	transactionRepo := repository.NewTransactionRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)
	ibkrRepo := repository.NewIbkrRepository(db)

	return service.NewTransactionService(
		db,
		transactionRepo,
		pfRepo,
		realizedGainLossRepo,
		ibkrRepo,
	)
}

// NewTestDividendService creates a DividendService wired to the provided test database.
func NewTestDividendService(t *testing.T, db *sql.DB) *service.DividendService {
	t.Helper()

	dividendRepo := repository.NewDividendRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)

	return service.NewDividendService(
		db,
		dividendRepo,
		pfRepo,
		transactionRepo,
	)
}

// NewTestDataloaderService creates a DataLoaderService wired to the provided test database.
func NewTestDataloaderService(t *testing.T, db *sql.DB) *service.DataLoaderService {
	t.Helper()

	pfRepo := repository.NewPortfolioFundRepository(db)

	return service.NewDataLoaderService(
		service.DataLoaderWithPortfolioFundRepository(pfRepo),
	)
}

// NewTestFundService creates a FundService wired to the provided test database.
func NewTestFundService(t *testing.T, db *sql.DB) *service.FundService {
	t.Helper()

	fundRepo := repository.NewFundRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewTransactionService(db, transactionRepo, pfRepo, repository.NewRealizedGainLossRepository(db), repository.NewIbkrRepository(db))
	dividendService := service.NewDividendService(db, repository.NewDividendRepository(db), pfRepo, transactionRepo)
	realizedGainLossService := service.NewRealizedGainLossService(repository.NewRealizedGainLossRepository(db))
	dataloaderService := service.NewDataLoaderService(
		service.DataLoaderWithPortfolioFundRepository(pfRepo),
		service.DataLoaderWithFundRepository(fundRepo),
		service.DataLoaderWithTransactionService(transactionService),
		service.DataLoaderWithDividendService(dividendService),
		service.DataLoaderWithRealizedGainLossService(realizedGainLossService),
	)
	portfolioRepo := repository.NewPortfolioRepository(db)

	return service.NewFundService(
		db,
		service.FundWithFundRepo(fundRepo),
		service.FundWithPortfolioFundRepo(pfRepo),
		service.FundWithDataLoaderService(dataloaderService),
		service.FundWithPortfolioRepo(portfolioRepo),
		service.FundWithYahooClient(NewMockYahooClient()),
	)
}

// NewTestFundServiceWithMockYahoo creates a FundService with a mock Yahoo client for testing.
// This is useful for testing fund price update operations without making real API calls.
func NewTestFundServiceWithMockYahoo(t *testing.T, db *sql.DB, mockYahoo yahoo.Client) *service.FundService {
	t.Helper()

	fundRepo := repository.NewFundRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewTransactionService(db, transactionRepo, pfRepo, repository.NewRealizedGainLossRepository(db), repository.NewIbkrRepository(db))

	return service.NewFundService(
		db,
		service.FundWithFundRepo(fundRepo),
		service.FundWithPortfolioFundRepo(pfRepo),
		service.FundWithTransactionService(transactionService),
		service.FundWithYahooClient(mockYahoo),
	)
}

// NewTestMaterializedService creates a MaterializedService wired to the provided test database.
func NewTestMaterializedService(t *testing.T, db *sql.DB) *service.MaterializedService {
	t.Helper()

	materializedRepo := repository.NewMaterializedRepository(db)
	portfolioRepo := repository.NewPortfolioRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	fundRepo := repository.NewFundRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewTransactionService(db, transactionRepo, pfRepo, repository.NewRealizedGainLossRepository(db), repository.NewIbkrRepository(db))
	dividendService := service.NewDividendService(db, repository.NewDividendRepository(db), pfRepo, transactionRepo)
	realizedGainLossService := service.NewRealizedGainLossService(repository.NewRealizedGainLossRepository(db))
	dataloaderService := service.NewDataLoaderService(
		service.DataLoaderWithPortfolioFundRepository(pfRepo),
		service.DataLoaderWithFundRepository(fundRepo),
		service.DataLoaderWithTransactionService(transactionService),
		service.DataLoaderWithDividendService(dividendService),
		service.DataLoaderWithRealizedGainLossService(realizedGainLossService),
	)
	portfolioService := service.NewPortfolioService(db, portfolioRepo, pfRepo)
	fundService := service.NewFundService(db,
		service.FundWithFundRepo(fundRepo),
		service.FundWithPortfolioFundRepo(pfRepo),
		service.FundWithDataLoaderService(dataloaderService),
		service.FundWithPortfolioRepo(portfolioRepo),
		service.FundWithYahooClient(NewMockYahooClient()),
	)

	return service.NewMaterializedService(db,
		service.MaterializedWithMaterializedRepository(materializedRepo),
		service.MaterializedWithDataLoaderService(dataloaderService),
		service.MaterializedWithPortfolioService(portfolioService),
		service.MaterializedWithPortfolioFundRepository(pfRepo),
		service.MaterializedWithFundService(fundService),
		service.MaterializedWithDividendService(dividendService),
		service.MaterializedWithRealizedGainLossService(realizedGainLossService),
	)
}

// NewTestIbkrService creates an IbkrService wired to the provided test database.
func NewTestIbkrService(t *testing.T, db *sql.DB, opts ...service.IbkrServiceOption) *service.IbkrService {
	t.Helper()
	return NewTestIbkrServiceWithMockIBKR(t, db, ibkr.NewFinanceClient(), opts...)
}

// NewTestIbkrServiceWithMockIBKR creates an IbkrService with a mock IBKR client for testing.
// This is useful for testing import operations without making real API calls to IBKR.
// Additional IbkrServiceOption values (e.g. IbkrWithEncryptionKey) can be appended via opts.
func NewTestIbkrServiceWithMockIBKR(t *testing.T, db *sql.DB, mockIBKR ibkr.Client, opts ...service.IbkrServiceOption) *service.IbkrService {
	t.Helper()

	ibkrRepo := repository.NewIbkrRepository(db)
	fundRepo := repository.NewFundRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	dividendRepo := repository.NewDividendRepository(db)

	base := make([]service.IbkrServiceOption, 0, 7+len(opts))
	base = append(base,
		service.IbkrWithIbkrRepo(ibkrRepo),
		service.IbkrWithPortfolioRepo(repository.NewPortfolioRepository(db)),
		service.IbkrWithFundRepo(fundRepo),
		service.IbkrWithPortfolioFundRepo(pfRepo),
		service.IbkrWithTransactionRepo(transactionRepo),
		service.IbkrWithDividendRepo(dividendRepo),
		service.IbkrWithClient(mockIBKR),
	)

	return service.NewIbkrService(db, append(base, opts...)...)
}

// NewTestSystemService creates a SystemService wired to the provided test database.
func NewTestSystemService(t *testing.T, db *sql.DB) *service.SystemService {
	t.Helper()

	return service.NewSystemService(db)
}

// NewTestDeveloperService creates a DeveloperService wired to the provided test database.
func NewTestDeveloperService(t *testing.T, db *sql.DB) *service.DeveloperService {
	t.Helper()

	developerRepo := repository.NewDeveloperRepository(db)
	fundRepo := repository.NewFundRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	return service.NewDeveloperService(db, developerRepo, fundRepo, transactionRepo, pfRepo)
}

// MakeID generates a UUID string for use in tests.
//
// Example usage:
//
//	id := testutil.MakeID()
//	// Returns: "550e8400-e29b-41d4-a716-446655440000"
func MakeID() string {
	return uuid.New().String()
}

// MakeISIN generates a realistic ISIN code for testing.
//
// Example usage:
//
//	isin := testutil.MakeISIN("US")
//	// Returns: "US1A2B3C4D5E"
func MakeISIN(prefix string) string {
	if prefix == "" {
		prefix = "US"
	}
	return prefix + randomAlphanumeric(10)
}

// MakeSymbol generates a stock ticker symbol for testing.
//
// Example usage:
//
//	symbol := testutil.MakeSymbol("AAPL")
//	// Returns: "AAPL1A2B"
func MakeSymbol(base string) string {
	if base == "" {
		base = "TEST"
	}
	return base + randomAlphanumeric(4)
}

// MakePortfolioName generates a unique portfolio name for testing.
//
// Example usage:
//
//	name := testutil.MakePortfolioName("MyPortfolio")
//	// Returns: "MyPortfolio ABC123"
func MakePortfolioName(base string) string {
	if base == "" {
		base = "Portfolio"
	}
	return base + " " + randomAlphanumeric(6)
}

// MakeFundName generates a unique fund name for testing.
//
// Example usage:
//
//	name := testutil.MakeFundName("Tech Fund")
//	// Returns: "Tech Fund XYZ789"
func MakeFundName(base string) string {
	if base == "" {
		base = "Fund"
	}
	return base + " " + randomAlphanumeric(6)
}

// MakeSymbolName generates a unique fund name for testing.
//
// Example usage:
//
//	name := testutil.MakeSymbolName("Tech Symbol")
//	// Returns: "Tech Symbol XYZ789"
func MakeSymbolName(base string) string {
	if base == "" {
		base = "Symbol"
	}
	return base + " " + randomAlphanumeric(6)
}

// randomAlphanumeric generates a random alphanumeric string of specified length.
func randomAlphanumeric(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		//nolint:gosec // G404: Using math/rand for test data generation is acceptable
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// Common test constants

var (
	// CommonCurrencies contains frequently used currency codes
	CommonCurrencies = []string{"USD", "EUR", "GBP", "JPY", "CAD", "CHF", "AUD"}

	// CommonExchanges contains frequently used stock exchanges
	CommonExchanges = []string{"NASDAQ", "NYSE", "LSE", "TSE", "XETRA", "EURONEXT"}

	// CommonCountryPrefixes contains common ISIN country prefixes
	CommonCountryPrefixes = []string{"US", "GB", "DE", "FR", "JP", "CA", "CH", "AU"}
)

// RandomCurrency returns a random currency from CommonCurrencies.
func RandomCurrency() string {
	//nolint:gosec // G404: Using math/rand for test data generation is acceptable
	return CommonCurrencies[rand.Intn(len(CommonCurrencies))]
}

// RandomExchange returns a random exchange from CommonExchanges.
func RandomExchange() string {
	//nolint:gosec // G404: Using math/rand for test data generation is acceptable
	return CommonExchanges[rand.Intn(len(CommonExchanges))]
}

// RandomCountryPrefix returns a random country prefix from CommonCountryPrefixes.
func RandomCountryPrefix() string {
	//nolint:gosec // G404: Using math/rand for test data generation is acceptable
	return CommonCountryPrefixes[rand.Intn(len(CommonCountryPrefixes))]
}
