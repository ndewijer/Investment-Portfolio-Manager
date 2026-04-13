package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/ibkr"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var ibkrLog = logging.NewLogger("ibkr")

// IbkrService handles IBKR (Interactive Brokers) integration business logic operations.
type IbkrService struct {
	db                      *sql.DB
	ibkrRepo                *repository.IbkrRepository
	portfolioRepo           *repository.PortfolioRepository
	fundRepository          *repository.FundRepository
	developerRepository     *repository.DeveloperRepository
	ibkrClient              ibkr.Client
	pfRepo                  *repository.PortfolioFundRepository
	transactionRepo         *repository.TransactionRepository
	dividendRepo            *repository.DividendRepository
	encryptionKey           *fernet.Key
	materializedInvalidator MaterializedInvalidator
}

// IbkrServiceOption is a functional option for configuring an IbkrService.
type IbkrServiceOption func(*IbkrService)

// IbkrWithIbkrRepo injects the IbkrRepository dependency.
func IbkrWithIbkrRepo(r *repository.IbkrRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.ibkrRepo = r }
}

// IbkrWithPortfolioRepo injects the PortfolioRepository dependency.
func IbkrWithPortfolioRepo(r *repository.PortfolioRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.portfolioRepo = r }
}

// IbkrWithFundRepo injects the FundRepository dependency.
func IbkrWithFundRepo(r *repository.FundRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.fundRepository = r }
}

// IbkrWithDeveloperRepo injects the DeveloperRepository dependency.
func IbkrWithDeveloperRepo(r *repository.DeveloperRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.developerRepository = r }
}

// IbkrWithClient injects the IBKR API client dependency.
func IbkrWithClient(c ibkr.Client) IbkrServiceOption {
	return func(s *IbkrService) { s.ibkrClient = c }
}

// IbkrWithPortfolioFundRepo injects the PortfolioFundRepository dependency.
func IbkrWithPortfolioFundRepo(r *repository.PortfolioFundRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.pfRepo = r }
}

// IbkrWithTransactionRepo injects the TransactionRepository dependency.
func IbkrWithTransactionRepo(r *repository.TransactionRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.transactionRepo = r }
}

// IbkrWithDividendRepo injects the DividendRepository dependency.
func IbkrWithDividendRepo(r *repository.DividendRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.dividendRepo = r }
}

// IbkrWithEncryptionKey injects the pre-decoded Fernet encryption key.
func IbkrWithEncryptionKey(key *fernet.Key) IbkrServiceOption {
	return func(s *IbkrService) { s.encryptionKey = key }
}

// NewIbkrService creates a new IbkrService. Pass IbkrWith* options to inject dependencies.
// Only the options relevant to the calling context need to be provided; unset fields remain
// nil and will panic if the corresponding method is called — a clear wiring error.
func NewIbkrService(db *sql.DB, opts ...IbkrServiceOption) *IbkrService {
	s := &IbkrService{db: db}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetMaterializedInvalidator injects the MaterializedInvalidator after construction.
// This breaks the circular initialization order between IbkrService and MaterializedService.
func (s *IbkrService) SetMaterializedInvalidator(m MaterializedInvalidator) {
	s.materializedInvalidator = m
}

// GetIbkrConfig retrieves the IBKR integration configuration.
// Adds a token expiration warning if the token expires within 30 days.
func (s *IbkrService) GetIbkrConfig() (*model.IbkrConfig, error) {
	ibkrLog.Debug("retrieving ibkr config")
	config, err := s.ibkrRepo.GetIbkrConfig()

	if err != nil {
		if errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			return &model.IbkrConfig{Configured: false}, nil
		}
		return &model.IbkrConfig{}, fmt.Errorf("get ibkr config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("unexpected nil config")
	}

	if config.TokenExpiresAt != nil && !config.TokenExpiresAt.IsZero() {
		diff := time.Until(*config.TokenExpiresAt)
		if diff.Hours() <= 720.0 {
			config.TokenWarning = fmt.Sprintf("Token expires in %d days",
				int64(diff.Hours()/24))
		}
	}

	return config, err
}

// GetActivePortfolios retrieves all active portfolios that can be used for IBKR import allocation.
// Returns portfolios that are not archived and not excluded from tracking.
func (s *IbkrService) GetActivePortfolios() ([]model.Portfolio, error) {
	ibkrLog.Debug("retrieving active portfolios for ibkr")
	portfolios, err := s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
	if err != nil {
		return nil, fmt.Errorf("get active portfolios: %w", err)
	}
	return portfolios, nil
}

// GetPendingDividends retrieves dividend records with PENDING reinvestment status.
// These dividends can be matched to incoming IBKR dividend transactions.
// Optionally filters by fund symbol or ISIN.
func (s *IbkrService) GetPendingDividends(symbol, isin string) ([]model.PendingDividend, error) {
	ibkrLog.Debug("retrieving pending dividends", "symbol", symbol, "isin", isin)
	dividends, err := s.ibkrRepo.GetPendingDividends(symbol, isin)
	if err != nil {
		return nil, fmt.Errorf("get pending dividends: %w", err)
	}
	return dividends, nil
}

// GetInbox retrieves IBKR imported transactions from the inbox.
// Returns transactions filtered by status (defaults to "pending") and optionally by transaction type.
// Used to display imported IBKR transactions that need to be allocated to portfolios.
func (s *IbkrService) GetInbox(status, transactionType string) ([]model.IBKRTransaction, error) {
	ibkrLog.Debug("retrieving inbox", "status", status, "transactionType", transactionType)
	inbox, err := s.ibkrRepo.GetInbox(status, transactionType)
	if err != nil {
		return nil, fmt.Errorf("get inbox: %w", err)
	}
	return inbox, nil
}

// GetInboxCount retrieves the count of IBKR imported transactions with status "pending".
// Returns only the count without fetching full transaction records for efficiency.
func (s *IbkrService) GetInboxCount() (model.IBKRInboxCount, error) {
	ibkrLog.Debug("retrieving inbox count")
	count, err := s.ibkrRepo.GetIbkrInboxCount()
	if err != nil {
		return model.IBKRInboxCount{}, fmt.Errorf("get inbox count: %w", err)
	}
	return count, nil
}

// GetTransactionAllocations retrieves the allocation details for an IBKR transaction.
// Fetches the transaction and its allocations, then processes and aggregates the data:
//   - Separates fee allocations from trade allocations
//   - Aggregates fees by portfolio ID and includes them in AllocatedCommission
//   - Rounds monetary values to standard precision
//   - Filters out fee transactions from the final response
//
// Parameters:
//   - transactionID: The UUID of the IBKR transaction
//
// Returns the transaction allocation summary with portfolio-level details,
// or an error if the transaction is not found or a database error occurs.
func (s *IbkrService) GetTransactionAllocations(transactionID string) (model.IBKRAllocation, error) {
	ibkrLog.Debug("retrieving transaction allocations", "transactionID", transactionID)

	ibkrTransaction, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKRAllocation{}, fmt.Errorf("get ibkr transaction: %w", err)
	}

	allocationDetails, err := s.ibkrRepo.GetIbkrTransactionAllocations(transactionID)
	if err != nil {
		return model.IBKRAllocation{}, fmt.Errorf("get transaction allocations: %w", err)
	}

	feesByID := make(map[string]float64)
	for _, allocation := range allocationDetails {
		if allocation.Type == "fee" {
			feesByID[allocation.PortfolioID] += allocation.AllocatedAmount
		}
	}

	allocationDetailsResponse := make([]model.IBKRTransactionAllocationResponse, 0, len(allocationDetails))

	for _, allocation := range allocationDetails {
		if allocation.Type == "fee" {
			continue
		}

		allocationDetailsResponse = append(allocationDetailsResponse, model.IBKRTransactionAllocationResponse{
			PortfolioID:          allocation.PortfolioID,
			PortfolioName:        allocation.PortfolioName,
			AllocationPercentage: allocation.AllocationPercentage,
			AllocatedAmount:      round(allocation.AllocatedAmount),
			AllocatedShares:      round(allocation.AllocatedShares),
			AllocatedCommission:  round(feesByID[allocation.PortfolioID]),
		})
	}

	allocationReturn := model.IBKRAllocation{
		IBKRTransactionID: ibkrTransaction.ID,
		Status:            ibkrTransaction.Status,
		Allocations:       allocationDetailsResponse,
	}

	return allocationReturn, nil
}

// GetEligiblePortfolios finds portfolios eligible for allocating an IBKR transaction.
// Matches the transaction's fund using a two-step process:
//  1. First attempts to match by ISIN (most reliable identifier)
//  2. If ISIN match fails, attempts to match by symbol
//
// Once the fund is found, retrieves all portfolios that hold this fund.
// If the fund exists but is not assigned to any portfolios, a warning is included.
//
//   - Returns 200 OK with found=false if no fund match (not an error)
//   - Uses nested match_info structure for compatibility
//
// Parameters:
//   - transactionID: The UUID of the IBKR transaction
//
// Returns:
//   - Response with matchInfo, portfolios, and optional warning
//   - Only returns error for database failures, not for "fund not found"
//   - ErrIBKRTransactionNotFound if the transaction doesn't exist
func (s *IbkrService) GetEligiblePortfolios(transactionID string) (model.IBKREligiblePortfolioResponse, error) {
	ibkrLog.Debug("retrieving eligible portfolios", "transactionID", transactionID)
	transaction, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, fmt.Errorf("get ibkr transaction: %w", err)
	}

	fund, err := s.findFundByISINOrSymbol(transaction.ISIN, transaction.Symbol)
	if errors.Is(err, apperrors.ErrIBKRFundNotMatched) {
		return model.IBKREligiblePortfolioResponse{
			MatchInfo: model.FundMatchInfo{
				Found:     false,
				MatchedBy: "",
			},
			Portfolios: []model.Portfolio{},
			Warning:    fmt.Sprintf("No fund found matching this transaction (Symbol: %s, ISIN: %s). Please add the fund to the system first.", transaction.Symbol, transaction.ISIN),
		}, nil
	}
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, fmt.Errorf("find fund by isin or symbol: %w", err)
	}

	matchedBy := "symbol"
	if fund.Isin != "" && fund.Isin == transaction.ISIN {
		matchedBy = "isin"
	}

	portfolios, err := s.portfolioRepo.GetPortfoliosByFundID(fund.ID)
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, fmt.Errorf("get portfolios by fund: %w", err)
	}

	warning := ""
	if len(portfolios) == 0 {
		warning = fmt.Sprintf("Fund '%s' (%s) exists but is not assigned to any portfolio. Please add this fund to a portfolio first.", fund.Name, fund.Symbol)
	}

	return model.IBKREligiblePortfolioResponse{
		MatchInfo: model.FundMatchInfo{
			Found:      true,
			MatchedBy:  matchedBy,
			FundID:     fund.ID,
			FundName:   fund.Name,
			FundSymbol: fund.Symbol,
			FundISIN:   fund.Isin,
		},
		Portfolios: portfolios,
		Warning:    warning,
	}, nil
}

// ImportFlexReport fetches and processes an IBKR Flex statement.
// Checks the local cache first and only calls the IBKR API if the cache is missing or expired.
// New transactions are compared against existing records and only new ones are inserted.
// Updates the last import date on the config after a successful run.
// Returns the number of imported and skipped transactions, or an error if the import fails.
//
//nolint:gocyclo // Primary Flex Report Import orchestrator. Mostly filled with error handling.
func (s *IbkrService) ImportFlexReport(ctx context.Context) (int, int, error) {
	ibkrLog.DebugContext(ctx, "starting flex report import")

	config, err := s.GetIbkrConfig()
	if err != nil {
		return 0, 0, fmt.Errorf("get ibkr config: %w", err)
	}

	cache, err := s.ibkrRepo.GetIbkrImportCache()
	if err != nil {
		// No cache is fine, error on the rest.
		if !errors.Is(err, apperrors.ErrIbkrImportCacheNotFound) {
			return 0, 0, fmt.Errorf("get import cache: %w", err)
		}
	}
	var req ibkr.FlexQueryResponse
	var body []byte
	var cacheSet bool

	if cache.ExpiresAt.IsZero() || cache.ExpiresAt.Before(time.Now().UTC()) {
		ibkrLog.DebugContext(ctx, "import cache expired or missing, fetching from ibkr api")
		token, err := s.decryptToken(config.FlexToken)
		if err != nil {
			return 0, 0, fmt.Errorf("decrypt token: %w", err)
		}

		req, body, err = s.ibkrClient.RetreiveIbkrFlexReport(ctx, token, config.FlexQueryID)
		if err != nil {
			return 0, 0, fmt.Errorf("retrieve flex report: %w", err)
		}
	} else {
		ibkrLog.DebugContext(ctx, "using cached import data", "expiresAt", cache.ExpiresAt)
		cacheSet = true
		err := xml.Unmarshal(cache.Data, &req)
		if err != nil {
			return 0, 0, fmt.Errorf("unmarshal cached flex report: %w", err)
		}
	}

	now := time.Now().UTC()

	if len(body) > 0 && !cacheSet {
		cacheKey := fmt.Sprintf("ibkr_flex_%d_%s", req.QueryID, now.Truncate(24*time.Hour).Format("2006-01-02"))
		importCache := model.IbkrImportCache{
			ID:        uuid.New().String(),
			CacheKey:  cacheKey,
			Data:      body,
			CreatedAt: now,
			ExpiresAt: now.Add(time.Hour),
		}
		if err := s.writeImportCache(ctx, importCache); err != nil {
			return 0, 0, fmt.Errorf("write import cache: %w", err)
		}
	}

	report, rates, err := s.parseIBKRFlexReport(req)
	if err != nil {
		return 0, 0, fmt.Errorf("parse flex report: %w", err)
	}

	missingTransactions := []model.IBKRTransaction{}

	for _, v := range report {
		if !s.ibkrRepo.CompareIbkrTransaction(v) {
			missingTransactions = append(missingTransactions, v)
		}
	}

	if len(missingTransactions) > 0 {
		if err := s.AddIbkrTransactions(ctx, missingTransactions); err != nil {
			return 0, 0, fmt.Errorf("add transactions: %w", err)
		}

		if config.DefaultAllocationEnabled && len(config.DefaultAllocations) > 0 {
			ibkrLog.InfoContext(ctx, "auto-allocating imported transactions", "count", len(missingTransactions))
			for _, tx := range missingTransactions {
				if err := s.AllocateIbkrTransaction(ctx, tx.ID, nil); err != nil {
					ibkrLog.ErrorContext(ctx, "auto-allocation failed", "transactionID", tx.ID, "symbol", tx.Symbol, "error", err)
				} else {
					ibkrLog.DebugContext(ctx, "auto-allocated transaction", "transactionID", tx.ID, "symbol", tx.Symbol)
				}
			}
		}
	}

	if len(rates) > 0 {
		if err := s.addExchangeRates(ctx, rates); err != nil {
			return 0, 0, fmt.Errorf("add exchange rates: %w", err)
		}
	}

	if err := s.ibkrRepo.UpdateLastImportDate(ctx, config.FlexQueryID, now); err != nil {
		return 0, 0, fmt.Errorf("ImportFlexReport: failed to update last_import_date: %w", err)
	}

	ibkrLog.InfoContext(ctx, "flex report import completed", "imported", len(missingTransactions), "skipped", len(report)-len(missingTransactions), "exchangeRates", len(rates))
	return len(missingTransactions), len(report) - len(missingTransactions), nil
}

// decryptToken decrypts a fernet-encrypted IBKR flex token using the injected encryption key.
// Returns an error if the key is nil or decryption fails.
func (s *IbkrService) decryptToken(token string) (string, error) {
	if s.encryptionKey == nil {
		return "", fmt.Errorf("IBKR_ENCRYPTION_KEY not set")
	}
	decryptedToken := fernet.VerifyAndDecrypt([]byte(token), 0, []*fernet.Key{s.encryptionKey})
	if decryptedToken == nil {
		return "", fmt.Errorf("decryption failed")
	}

	return strings.TrimSpace(string(decryptedToken)), nil
}

// encryptToken encrypts a plaintext IBKR flex token using the injected Fernet key.
// Returns an error if the key is nil or encryption fails.
func (s *IbkrService) encryptToken(token string) (string, error) {
	if s.encryptionKey == nil {
		return "", fmt.Errorf("IBKR_ENCRYPTION_KEY not set")
	}
	encryptedToken, err := fernet.EncryptAndSign([]byte(token), s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("encrypt token: %w", err)
	}
	if encryptedToken == nil {
		return "", fmt.Errorf("encryption failed")
	}

	return strings.TrimSpace(string(encryptedToken)), nil
}

// writeImportCache persists a Flex report cache entry to the database within a transaction.
// The caller is responsible for populating all fields on importCache before calling.
func (s *IbkrService) writeImportCache(ctx context.Context, importCache model.IbkrImportCache) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	if err := s.ibkrRepo.WithTx(tx).WriteImportCache(ctx, importCache); err != nil {
		return fmt.Errorf("write import cache: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// addExchangeRates upserts a slice of exchange rates within a single transaction.
// Iterates all rates and calls UpdateExchangeRate for each; rolls back on the first failure.
func (s *IbkrService) addExchangeRates(ctx context.Context, rates []model.ExchangeRate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	for _, v := range rates {

		if err = s.developerRepository.WithTx(tx).UpdateExchangeRate(ctx, v); err != nil {
			return fmt.Errorf("update exchange rate: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// AddIbkrTransactions persists a slice of IBKR transactions to the database within a single transaction.
// Returns an error if the transaction cannot be started or if any insert fails.
func (s *IbkrService) AddIbkrTransactions(ctx context.Context, transactions []model.IBKRTransaction) error {
	ibkrLog.DebugContext(ctx, "adding ibkr transactions", "count", len(transactions))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	if err = s.ibkrRepo.WithTx(tx).AddIbkrTransactions(ctx, transactions); err != nil {
		return fmt.Errorf("add ibkr transactions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// parseIBKRFlexReport converts a raw IBKR Flex report into slices of IBKRTransaction and ExchangeRate models.
// Dates are parsed from IBKR's "20060102" format. Each trade's quantity, net cash, and commission
// are normalised to absolute values. Returns an error if any date or JSON marshal step fails.
func (s *IbkrService) parseIBKRFlexReport(report ibkr.FlexQueryResponse) ([]model.IBKRTransaction, []model.ExchangeRate, error) {

	ibkrTransactions := make([]model.IBKRTransaction, len(report.FlexStatements.FlexStatement.Trades.Trade))
	ibkrExchangeRate := make([]model.ExchangeRate, len(report.FlexStatements.FlexStatement.ConversionRates.ConversionRate))
	for i, v := range report.FlexStatements.FlexStatement.Trades.Trade {
		transactionDate, err := time.Parse("20060102", v.TradeDate)
		if err != nil {
			return nil, nil, fmt.Errorf("parse trade date for transaction %d: %w", v.TransactionID, err)
		}

		reportDate, err := time.Parse("20060102", v.ReportDate)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse reportDate for transaction %d: %w", v.TransactionID, err)
		}

		rawBytes, err := json.Marshal(v)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal raw transaction data: %w", err)
		}

		currency := v.CurrencyPrimary
		if currency == "" {
			currency = v.Currency
		}

		t := model.IBKRTransaction{
			ID:                uuid.New().String(),
			IBKRTransactionID: fmt.Sprintf("%d_%d", v.TransactionID, v.IbOrderID),
			TransactionDate:   transactionDate,
			Symbol:            v.Symbol,
			ISIN:              v.Isin,
			Description:       v.Description,
			TransactionType:   strings.ToLower(v.BuySell),
			Quantity:          math.Abs(v.Quantity),
			Price:             v.TradePrice,
			TotalAmount:       math.Abs(v.NetCash),
			Currency:          currency,
			Fees:              math.Abs(v.IbCommission),
			Status:            "pending",
			ImportedAt:        report.ImportedAt,
			RawData:           rawBytes,
			Notes:             v.Notes,
			ReportDate:        reportDate,
		}

		ibkrTransactions[i] = t
	}

	for i, v := range report.FlexStatements.FlexStatement.ConversionRates.ConversionRate {
		reportDate, err := time.Parse("20060102", v.ReportDate)
		if err != nil {
			return nil, nil, fmt.Errorf("parse conversion rate report date: %w", err)
		}

		rate := model.ExchangeRate{
			ID:           uuid.New().String(),
			Date:         reportDate,
			FromCurrency: v.FromCurrency,
			ToCurrency:   v.ToCurrency,
			Rate:         v.Rate,
		}
		ibkrExchangeRate[i] = rate
	}

	return ibkrTransactions, ibkrExchangeRate, nil
}

// UpdateIbkrConfig applies a partial update to the IBKR configuration.
// Only non-nil fields in the request are applied; omitted fields retain their current values.
// FlexToken is an exception: passing an empty string also means "no change" — only a non-empty
// value overwrites the existing encrypted token.
//
// The table holds exactly one row, so the existing config is fetched and the request is merged
// onto it before persisting. overwriteConfig is set when flex_query_id changes while enabled is
// true; in that case the repository deletes the old row before inserting the new one to avoid a
// stale primary key. Once multiple IBKR accounts are supported this check will need to use the
// config ID directly rather than comparing query IDs.
//
//nolint:gocyclo,funlen // if req.X != nil pattern is intrinsic to patch-style updates in Go; no meaningful split possible
func (s *IbkrService) UpdateIbkrConfig(
	ctx context.Context,
	req request.UpdateIbkrConfigRequest,
) (*model.IbkrConfig, error) {
	ibkrLog.DebugContext(ctx, "updating ibkr config")

	var config *model.IbkrConfig
	var err error
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	config, err = s.ibkrRepo.WithTx(tx).GetIbkrConfig()
	if err != nil && !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
		return nil, fmt.Errorf("get ibkr config: %w", err)
	} else if errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
		config.ID = uuid.New().String()
		config.CreatedAt = time.Now().UTC()
	}
	var overwriteConfig bool

	if (req.FlexQueryID != nil && *req.FlexQueryID != config.FlexQueryID) && (req.Enabled != nil && *req.Enabled) {
		overwriteConfig = true
	}

	if req.FlexQueryID != nil {
		config.FlexQueryID = *req.FlexQueryID
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	config.UpdatedAt = time.Now().UTC()

	if !config.Enabled {
		config.AutoImportEnabled = false

		if err := s.ibkrRepo.WithTx(tx).UpdateIbkrConfig(ctx, overwriteConfig, config); err != nil {
			return nil, fmt.Errorf("failed to update IBKR config: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}

		config.Configured = true

		return config, nil
	}

	if req.FlexToken != nil && *req.FlexToken != "" {
		encToken, err := s.encryptToken(*req.FlexToken)
		if err != nil {
			return nil, fmt.Errorf("encrypt token: %w", err)
		}
		config.FlexToken = encToken
	}

	if config.Enabled && config.FlexToken == "" {
		return nil, fmt.Errorf("flexToken is required when enabled is true")
	}

	if req.TokenExpiresAt != nil {
		time, err := repository.ParseTime(*req.TokenExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("parse token expiration: %w", err)
		}
		config.TokenExpiresAt = &time
	}
	if req.AutoImportEnabled != nil {
		config.AutoImportEnabled = *req.AutoImportEnabled
	}
	if req.DefaultAllocationEnabled != nil {
		config.DefaultAllocationEnabled = *req.DefaultAllocationEnabled
	}

	if req.DefaultAllocations != nil {
		all := make([]model.Allocation, len(req.DefaultAllocations))
		for i, v := range req.DefaultAllocations {
			all[i].Percentage = *v.Percentage
			all[i].PortfolioID = *v.PortfolioID
		}
		config.DefaultAllocations = all
	}

	if err := s.ibkrRepo.WithTx(tx).UpdateIbkrConfig(ctx, overwriteConfig, config); err != nil {
		return nil, fmt.Errorf("failed to update IBKR config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	config.Configured = true
	ibkrLog.InfoContext(ctx, "ibkr config updated", "enabled", config.Enabled, "autoImport", config.AutoImportEnabled)

	return config, nil
}

// DeleteIbkrConfig removes the IBKR configuration from the database.
// Returns ErrIbkrConfigNotFound (propagated from the repository) if no config exists.
func (s *IbkrService) DeleteIbkrConfig(ctx context.Context) error {
	ibkrLog.DebugContext(ctx, "deleting ibkr config")
	err := s.ibkrRepo.DeleteIbkrConfig(ctx)
	if err != nil {
		return fmt.Errorf("delete ibkr config: %w", err)
	}

	ibkrLog.InfoContext(ctx, "ibkr config deleted")
	return nil
}

// TestIbkrConnection verifies that the provided credentials are accepted by IBKR.
// Unlike other token operations, the token here comes directly from the caller rather
// than from the encrypted config — this is intentional for a pre-save credential check.
// Returns true if IBKR accepts the credentials, or an error if the call fails.
func (s *IbkrService) TestIbkrConnection(ctx context.Context, req request.TestIbkrConnectionRequest) (bool, error) {
	ibkrLog.DebugContext(ctx, "testing ibkr connection", "flexQueryID", req.FlexQueryID)
	ok, err := s.ibkrClient.TestIbkrConnection(ctx, req.FlexToken, req.FlexQueryID)
	if err != nil {
		return false, fmt.Errorf("test ibkr connection: %w", err)
	}
	return ok, nil
}

// GetIbkrTransactionDetail retrieves a single IBKR transaction with its allocation details.
// If the transaction is processed, allocations are included in the response.
func (s *IbkrService) GetIbkrTransactionDetail(transactionID string) (model.IBKRTransactionDetail, error) {
	ibkrLog.Debug("retrieving ibkr transaction detail", "transactionID", transactionID)
	tx, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKRTransactionDetail{}, fmt.Errorf("get ibkr transaction: %w", err)
	}

	detail := model.IBKRTransactionDetail{IBKRTransaction: tx}

	if tx.Status == "processed" {
		alloc, err := s.GetTransactionAllocations(transactionID)
		if err != nil {
			return model.IBKRTransactionDetail{}, fmt.Errorf("get transaction allocations: %w", err)
		}
		detail.Allocations = alloc.Allocations
	}

	return detail, nil
}

// DeleteIbkrTransaction removes a pending IBKR transaction.
// Returns ErrIBKRTransactionAlreadyProcessed if the transaction is not pending.
func (s *IbkrService) DeleteIbkrTransaction(ctx context.Context, transactionID string) error {
	ibkrLog.DebugContext(ctx, "deleting ibkr transaction", "transactionID", transactionID)
	tx, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	if tx.Status != "pending" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	if err := s.ibkrRepo.DeleteIbkrTransaction(ctx, transactionID); err != nil {
		return fmt.Errorf("delete ibkr transaction: %w", err)
	}

	ibkrLog.InfoContext(ctx, "ibkr transaction deleted", "transaction_id", transactionID)
	return nil
}

// IgnoreIbkrTransaction marks a pending IBKR transaction as ignored.
// Returns ErrIBKRTransactionAlreadyProcessed if the transaction is not pending.
func (s *IbkrService) IgnoreIbkrTransaction(ctx context.Context, transactionID string) error {
	ibkrLog.DebugContext(ctx, "ignoring ibkr transaction", "transactionID", transactionID)
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	if ibkrTx.Status != "pending" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	if err := s.ibkrRepo.WithTx(dbTx).UpdateIbkrTransactionStatus(ctx, transactionID, "ignored", nil); err != nil {
		return fmt.Errorf("update ibkr transaction status: %w", err)
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	ibkrLog.InfoContext(ctx, "ibkr transaction ignored", "transaction_id", transactionID)
	return nil
}

// findFundByISINOrSymbol looks up a fund by ISIN first, then by symbol.
// Returns ErrIBKRFundNotMatched if neither matches.
func (s *IbkrService) findFundByISINOrSymbol(isin, symbol string) (model.Fund, error) {
	fund, err := s.fundRepository.GetFundBySymbolOrIsin("", isin)
	if err != nil && !errors.Is(err, apperrors.ErrFundNotFound) {
		return model.Fund{}, fmt.Errorf("find fund by isin: %w", err)
	}
	if fund.ID != "" {
		return fund, nil
	}

	fund, err = s.fundRepository.GetFundBySymbolOrIsin(symbol, "")
	if err != nil && !errors.Is(err, apperrors.ErrFundNotFound) {
		return model.Fund{}, fmt.Errorf("find fund by symbol: %w", err)
	}
	if fund.ID != "" {
		return fund, nil
	}

	return model.Fund{}, apperrors.ErrIBKRFundNotMatched
}

// AllocateIbkrTransaction allocates a pending IBKR transaction to portfolios.
// If allocations is empty and default allocation is enabled in config, uses default allocations.
// Creates Transaction and IBKRTransactionAllocation records for each portfolio, including
// separate fee transactions when fees > 0.
func (s *IbkrService) AllocateIbkrTransaction(ctx context.Context, transactionID string, allocations []request.AllocationEntry) error {
	ibkrLog.DebugContext(ctx, "allocating ibkr transaction", "transactionID", transactionID, "allocations", len(allocations))
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	if err := s.allocateIbkrTransactionTx(ctx, dbTx, transactionID, allocations); err != nil {
		return fmt.Errorf("allocate transaction: %w", err)
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	ibkrLog.InfoContext(ctx, "ibkr transaction allocated", "transactionID", transactionID)
	s.triggerRegenFromAllocations(transactionID, ibkrTx.TransactionDate)

	return nil
}

// BulkAllocateIbkrTransactions allocates multiple IBKR transactions using the same allocation split.
// Each transaction is allocated in its own DB transaction so partial success is possible.
func (s *IbkrService) BulkAllocateIbkrTransactions(ctx context.Context, req request.BulkAllocateRequest) model.BulkAllocateResponse {
	ibkrLog.DebugContext(ctx, "bulk allocating ibkr transactions", "count", len(req.TransactionIDs))
	resp := model.BulkAllocateResponse{Errors: []string{}}

	for _, txID := range req.TransactionIDs {
		if err := s.AllocateIbkrTransaction(ctx, txID, req.Allocations); err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %s", txID, err.Error()))
		} else {
			resp.Success++
		}
	}

	if resp.Failed > 0 && resp.Success > 0 {
		ibkrLog.Warn("bulk allocation partially succeeded", "success", resp.Success, "failed", resp.Failed)
	} else if resp.Failed > 0 {
		ibkrLog.Warn("bulk allocation failed for all transactions", "failed", resp.Failed)
	} else {
		ibkrLog.InfoContext(ctx, "bulk allocation completed", "success", resp.Success)
	}

	return resp
}

// unallocateIbkrTransactionTx performs unallocation within an existing DB transaction.
// Deletes linked Transaction records and IBKRTransactionAllocation records, then resets
// the IBKR transaction status to "pending".
func (s *IbkrService) unallocateIbkrTransactionTx(ctx context.Context, dbTx *sql.Tx, transactionID string) error {
	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	if ibkrTx.Status != "processed" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	txIDs, err := s.ibkrRepo.WithTx(dbTx).GetTransactionIDsByIbkrTransaction(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to get linked transaction IDs: %w", err)
	}

	// Allocations FK to transactions, so delete them first
	if err := s.ibkrRepo.WithTx(dbTx).DeleteIbkrTransactionAllocations(ctx, transactionID); err != nil {
		return fmt.Errorf("failed to delete allocations: %w", err)
	}

	for _, txID := range txIDs {
		if err := s.transactionRepo.WithTx(dbTx).DeleteTransaction(ctx, txID); err != nil {
			return fmt.Errorf("failed to delete transaction %s: %w", txID, err)
		}
	}

	if err := s.ibkrRepo.WithTx(dbTx).UpdateIbkrTransactionStatus(ctx, transactionID, "pending", nil); err != nil {
		return fmt.Errorf("failed to reset transaction status: %w", err)
	}

	return nil
}

// UnallocateIbkrTransaction reverses the allocation of a processed IBKR transaction.
// Deletes all linked Transaction and IBKRTransactionAllocation records, then resets
// the status to "pending".
func (s *IbkrService) UnallocateIbkrTransaction(ctx context.Context, transactionID string) error {
	ibkrLog.DebugContext(ctx, "unallocating ibkr transaction", "transactionID", transactionID)
	ibkrTx, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	portfolioIDs := s.collectPortfolioIDsFromAllocations(transactionID)

	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	if err := s.unallocateIbkrTransactionTx(ctx, dbTx, transactionID); err != nil {
		return fmt.Errorf("unallocate transaction: %w", err)
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	ibkrLog.InfoContext(ctx, "ibkr transaction unallocated", "transaction_id", transactionID)

	if s.materializedInvalidator != nil && len(portfolioIDs) > 0 {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), ibkrTx.TransactionDate, portfolioIDs, "", ""); err != nil {
				ibkrLog.Warn("failed to regenerate materialized table after unallocation", "error", err)
			}
		}()
	}

	return nil
}

// ModifyAllocations atomically unallocates and reallocates an IBKR transaction with new allocation percentages.
// Both operations run in a single DB transaction for atomicity.
func (s *IbkrService) ModifyAllocations(ctx context.Context, transactionID string, allocations []request.AllocationEntry) error {
	ibkrLog.DebugContext(ctx, "modifying ibkr allocations", "transactionID", transactionID, "allocations", len(allocations))
	ibkrTx, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	oldPortfolioIDs := s.collectPortfolioIDsFromAllocations(transactionID)

	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	if err := s.unallocateIbkrTransactionTx(ctx, dbTx, transactionID); err != nil {
		return fmt.Errorf("unallocate transaction: %w", err)
	}

	// Inline allocate (AllocateIbkrTransaction opens its own tx)
	if err := s.allocateIbkrTransactionTx(ctx, dbTx, transactionID, allocations); err != nil {
		return fmt.Errorf("allocate transaction: %w", err)
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	ibkrLog.InfoContext(ctx, "ibkr allocations modified", "transaction_id", transactionID)

	newPortfolioIDs := s.collectPortfolioIDsFromAllocations(transactionID)
	seen := make(map[string]bool)
	for _, pid := range newPortfolioIDs {
		seen[pid] = true
	}
	for _, pid := range oldPortfolioIDs {
		if !seen[pid] {
			newPortfolioIDs = append(newPortfolioIDs, pid)
		}
	}
	if s.materializedInvalidator != nil && len(newPortfolioIDs) > 0 {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), ibkrTx.TransactionDate, newPortfolioIDs, "", ""); err != nil {
				ibkrLog.Warn("failed to regenerate materialized table after allocation modification", "error", err)
			}
		}()
	}

	return nil
}

// allocateIbkrTransactionTx performs allocation within an existing DB transaction.
// Contains the core allocation logic shared by AllocateIbkrTransaction and ModifyAllocations.
//
//nolint:gocyclo,funlen // Allocation orchestrator with per-portfolio loop, fee handling, and auto-allocate fallback.
func (s *IbkrService) allocateIbkrTransactionTx(ctx context.Context, dbTx *sql.Tx, transactionID string, allocations []request.AllocationEntry) error {
	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	if ibkrTx.Status != "pending" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	if len(allocations) == 0 {
		config, err := s.ibkrRepo.WithTx(dbTx).GetIbkrConfig()
		if err != nil && !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			return fmt.Errorf("get ibkr config: %w", err)
		}
		if config != nil && config.DefaultAllocationEnabled && len(config.DefaultAllocations) > 0 {
			allocations = make([]request.AllocationEntry, len(config.DefaultAllocations))
			for i, a := range config.DefaultAllocations {
				allocations[i] = request.AllocationEntry{
					PortfolioID: a.PortfolioID,
					Percentage:  a.Percentage,
				}
			}
		}
	}

	if len(allocations) == 0 {
		return apperrors.ErrIBKRInvalidAllocations
	}

	fund, err := s.findFundByISINOrSymbol(ibkrTx.ISIN, ibkrTx.Symbol)
	if err != nil {
		return fmt.Errorf("find fund by isin or symbol: %w", err)
	}

	now := time.Now().UTC()

	for _, alloc := range allocations {
		pf, err := s.pfRepo.WithTx(dbTx).GetPortfolioFundByPortfolioAndFund(alloc.PortfolioID, fund.ID)
		if errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			if err := s.pfRepo.WithTx(dbTx).InsertPortfolioFund(ctx, alloc.PortfolioID, fund.ID); err != nil {
				return fmt.Errorf("failed to create portfolio_fund: %w", err)
			}
			pf, err = s.pfRepo.WithTx(dbTx).GetPortfolioFundByPortfolioAndFund(alloc.PortfolioID, fund.ID)
			if err != nil {
				return fmt.Errorf("failed to retrieve created portfolio_fund: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("get portfolio fund: %w", err)
		}

		pct := alloc.Percentage / 100.0
		allocatedAmount := round(ibkrTx.TotalAmount * pct)
		allocatedShares := round(ibkrTx.Quantity * pct)
		allocatedFees := round(ibkrTx.Fees * pct)

		txn := &model.Transaction{
			ID:              uuid.New().String(),
			PortfolioFundID: pf.ID,
			Date:            ibkrTx.TransactionDate,
			Type:            ibkrTx.TransactionType,
			Shares:          allocatedShares,
			CostPerShare:    ibkrTx.Price,
			CreatedAt:       now,
		}
		if err := s.transactionRepo.WithTx(dbTx).InsertTransaction(ctx, txn); err != nil {
			return fmt.Errorf("failed to insert transaction: %w", err)
		}

		tradeAlloc := model.IBKRTransactionAllocation{
			ID:                   uuid.New().String(),
			IBKRTransactionID:    transactionID,
			PortfolioID:          alloc.PortfolioID,
			AllocationPercentage: alloc.Percentage,
			AllocatedAmount:      allocatedAmount,
			AllocatedShares:      allocatedShares,
			TransactionID:        txn.ID,
			CreatedAt:            now,
		}
		if err := s.ibkrRepo.WithTx(dbTx).InsertIbkrTransactionAllocation(ctx, tradeAlloc); err != nil {
			return fmt.Errorf("failed to insert trade allocation: %w", err)
		}

		if allocatedFees > 0 {
			feeTxn := &model.Transaction{
				ID:              uuid.New().String(),
				PortfolioFundID: pf.ID,
				Date:            ibkrTx.TransactionDate,
				Type:            "fee",
				Shares:          0,
				CostPerShare:    allocatedFees,
				CreatedAt:       now,
			}
			if err := s.transactionRepo.WithTx(dbTx).InsertTransaction(ctx, feeTxn); err != nil {
				return fmt.Errorf("failed to insert fee transaction: %w", err)
			}

			feeAlloc := model.IBKRTransactionAllocation{
				ID:                   uuid.New().String(),
				IBKRTransactionID:    transactionID,
				PortfolioID:          alloc.PortfolioID,
				AllocationPercentage: alloc.Percentage,
				AllocatedAmount:      allocatedFees,
				AllocatedShares:      0,
				TransactionID:        feeTxn.ID,
				CreatedAt:            now,
			}
			if err := s.ibkrRepo.WithTx(dbTx).InsertIbkrTransactionAllocation(ctx, feeAlloc); err != nil {
				return fmt.Errorf("failed to insert fee allocation: %w", err)
			}
		}
	}

	if err := s.ibkrRepo.WithTx(dbTx).UpdateIbkrTransactionStatus(ctx, transactionID, "processed", &now); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

// MatchDividend links a processed IBKR transaction (DRIP) to pending dividend records.
// The IBKR transaction must be allocated first (status "processed") so that Transaction records
// exist. Each dividend's reinvestment_transaction_id is set to the allocation's transaction_id.
//
//nolint:gocyclo // Dividend matching with per-dividend portfolio lookup and validation.
func (s *IbkrService) MatchDividend(ctx context.Context, transactionID string, dividendIDs []string) error {
	ibkrLog.DebugContext(ctx, "matching dividends to ibkr transaction", "transactionID", transactionID, "dividendIDs", len(dividendIDs))
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return fmt.Errorf("get ibkr transaction: %w", err)
	}

	if ibkrTx.Status != "processed" {
		return fmt.Errorf("%w: transaction must be allocated before matching dividends", apperrors.ErrIBKRTransactionAlreadyProcessed)
	}

	if !strings.Contains(ibkrTx.Notes, "R") {
		ibkrLog.Warn("ibkr transaction notes field does not contain 'R' (DRIP code)", "transactionID", transactionID, "notes", ibkrTx.Notes)
	}

	allocationDetails, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransactionAllocations(transactionID)
	if err != nil {
		return fmt.Errorf("failed to get allocations: %w", err)
	}

	portfolioToTxID := make(map[string]string)
	for _, a := range allocationDetails {
		if a.Type != "fee" && a.TransactionID != "" {
			portfolioToTxID[a.PortfolioID] = a.TransactionID
		}
	}

	for _, dividendID := range dividendIDs {
		dividend, err := s.dividendRepo.WithTx(dbTx).GetDividend(dividendID)
		if err != nil {
			return fmt.Errorf("failed to get dividend %s: %w", dividendID, err)
		}

		if dividend.ReinvestmentTransactionID != "" {
			return fmt.Errorf("dividend %s already matched to transaction %s", dividendID, dividend.ReinvestmentTransactionID)
		}

		pf, err := s.pfRepo.WithTx(dbTx).GetPortfolioFund(dividend.PortfolioFundID)
		if err != nil {
			return fmt.Errorf("failed to get portfolio_fund for dividend %s: %w", dividendID, err)
		}

		allocTxID, ok := portfolioToTxID[pf.PortfolioID]
		if !ok {
			return fmt.Errorf("no allocation found for portfolio %s (dividend %s)", pf.PortfolioID, dividendID)
		}

		dividend.ReinvestmentTransactionID = allocTxID
		dividend.BuyOrderDate = ibkrTx.TransactionDate
		dividend.ReinvestmentStatus = "COMPLETED"

		if err := s.dividendRepo.WithTx(dbTx).UpdateDividend(ctx, &dividend); err != nil {
			return fmt.Errorf("failed to update dividend %s: %w", dividendID, err)
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	ibkrLog.InfoContext(ctx, "dividends matched to ibkr transaction", "transactionID", transactionID, "dividendCount", len(dividendIDs))
	s.triggerRegenFromAllocations(transactionID, ibkrTx.TransactionDate)

	return nil
}

// collectPortfolioIDsFromAllocations returns the unique portfolio IDs linked to
// an IBKR transaction via its allocations. Used to know which portfolios are
// affected before an unallocation deletes the records.
func (s *IbkrService) collectPortfolioIDsFromAllocations(ibkrTransactionID string) []string {
	allocs, err := s.ibkrRepo.GetIbkrTransactionAllocations(ibkrTransactionID)
	if err != nil {
		ibkrLog.Warn("failed to get allocations for regen", "error", err, "ibkrTransactionID", ibkrTransactionID)
		return nil
	}
	seen := make(map[string]bool)
	var pids []string
	for _, a := range allocs {
		if !seen[a.PortfolioID] {
			seen[a.PortfolioID] = true
			pids = append(pids, a.PortfolioID)
		}
	}
	return pids
}

// triggerRegenFromAllocations looks up the portfolio IDs from an IBKR transaction's
// allocations and triggers a single materialized view regeneration covering all of them.
func (s *IbkrService) triggerRegenFromAllocations(ibkrTransactionID string, txDate time.Time) {
	if s.materializedInvalidator == nil {
		return
	}
	portfolioIDs := s.collectPortfolioIDsFromAllocations(ibkrTransactionID)
	if len(portfolioIDs) == 0 {
		return
	}
	//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
	go func() {
		if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), txDate, portfolioIDs, "", ""); err != nil {
			ibkrLog.Warn("failed to regenerate materialized table", "error", err)
		}
	}()
}
