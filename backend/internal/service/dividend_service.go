package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var divLog = logging.NewLogger("dividend")

// DividendService handles dividend-related business logic operations.
type DividendService struct {
	db                      *sql.DB
	dividendRepo            *repository.DividendRepository
	pfRepo                  *repository.PortfolioFundRepository
	transactionRepo         *repository.TransactionRepository
	materializedInvalidator MaterializedInvalidator
}

// NewDividendService creates a new DividendService with the provided dependencies.
//
// Parameters:
//   - db: Raw database connection, used to manage database transactions in CreateDividend.
//   - dividendRepo: Repository for dividend table operations.
//   - pfRepo: Repository for portfolio-fund lookups.
//   - transactionRepo: Repository for transaction table operations, including share calculations.
func NewDividendService(
	db *sql.DB,
	dividendRepo *repository.DividendRepository,
	pfRepo *repository.PortfolioFundRepository,
	transactionRepo *repository.TransactionRepository,
) *DividendService {
	return &DividendService{
		db:              db,
		dividendRepo:    dividendRepo,
		pfRepo:          pfRepo,
		transactionRepo: transactionRepo,
	}
}

// SetMaterializedInvalidator injects the MaterializedInvalidator after construction.
// This breaks the circular initialization order between DividendService and MaterializedService.
func (s *DividendService) SetMaterializedInvalidator(m MaterializedInvalidator) {
	s.materializedInvalidator = m
}

// GetAllDividend retrieves all dividend records from the database.
// Returns raw dividend data without fund enrichment.
func (s *DividendService) GetAllDividend() ([]model.Dividend, error) {
	divLog.Debug("retrieving all dividends")
	result, err := s.dividendRepo.GetAllDividend()
	if err != nil {
		return nil, fmt.Errorf("get all dividends: %w", err)
	}
	return result, nil
}

// GetDividend retrieves a single dividend record by its ID.
// Returns ErrDividendNotFound if no record with the given ID exists.
func (s *DividendService) GetDividend(DividendID string) (model.Dividend, error) {
	divLog.Debug("retrieving dividend", "dividendID", DividendID)
	result, err := s.dividendRepo.GetDividend(DividendID)
	if err != nil {
		return model.Dividend{}, fmt.Errorf("get dividend: %w", err)
	}
	return result, nil
}

// GetDividendFund retrieves enriched dividend records filtered by either portfolio or fund.
// Exactly one of portfolioID or fundID should be non-empty; if both are empty an empty slice is returned.
//
// Parameters:
//   - portfolioID: Filter by portfolio ID (mutually exclusive with fundID)
//   - fundID:      Filter by fund ID (mutually exclusive with portfolioID; checked if portfolioID is empty)
//
// Returns a slice of DividendFund containing all historical dividend payments for the given filter.
func (s *DividendService) GetDividendFund(portfolioID, fundID string) ([]model.DividendFund, error) {
	divLog.Debug("retrieving dividend fund", "portfolioID", portfolioID, "fundID", fundID)
	dividendFund, err := s.dividendRepo.GetDividendPerPortfolioFund(portfolioID, fundID)
	if err != nil {
		return nil, fmt.Errorf("get dividend fund: %w", err)
	}
	return dividendFund, nil
}

// loadDividendPerPF retrieves dividends for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio_fund ID, allowing callers to decide how to aggregate.
func (s *DividendService) loadDividendPerPF(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	result, err := s.dividendRepo.GetDividendPerPF(pfIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get dividends per portfolio fund: %w", err)
	}
	return result, nil
}

// processDividendSharesForDate calculates shares acquired through dividend reinvestment as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
// Returns a map of portfolio_fund ID to total reinvested shares.
func (s *DividendService) processDividendSharesForDate(dividendMap map[string][]model.Dividend, transactions []model.Transaction, date time.Time) (map[string]float64, error) {
	totalDividendMap := make(map[string]float64)

	for pfID, dividend := range dividendMap {
		var dividendShares float64

		for _, div := range dividend {
			if div.ExDividendDate.Before(date) || div.ExDividendDate.Equal(date) {
				if div.ReinvestmentTransactionID != "" {
					for _, transaction := range transactions {
						if transaction.ID == div.ReinvestmentTransactionID {
							dividendShares += transaction.Shares
							break
						}
					}
				}
			} else {
				break
			}
		}
		totalDividendMap[pfID] = dividendShares
	}

	return totalDividendMap, nil
}

// processDividendAmountForDate calculates the cumulative dividend amount as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
func (s *DividendService) processDividendAmountForDate(dividend []model.Dividend, date time.Time) (float64, error) {
	if len(dividend) == 0 {
		return 0.0, nil
	}
	var totalDividend float64

	for _, d := range dividend {

		if d.ExDividendDate.Before(date) || d.ExDividendDate.Equal(date) {
			totalDividend += d.TotalAmount
		} else {
			break
		}
	}

	return totalDividend, nil
}

// CreateDividend creates a new dividend record, calculating SharesOwned and TotalAmount
// from transactions as of the ex-dividend date.
//
// If BuyOrderDate is provided, a reinvestment transaction is also created atomically
// within the same database transaction. ReinvestmentStatus is determined as follows:
//
//   - STOCK fund, no BuyOrderDate:                              "PENDING"
//   - STOCK fund, BuyOrderDate set, price/shares missing:       "PENDING"
//   - STOCK fund, BuyOrderDate set, reinvested == total amount: "COMPLETED"
//   - STOCK fund, BuyOrderDate set, reinvested < total amount:  "PARTIAL"
//   - Non-STOCK fund, no BuyOrderDate:                          "COMPLETED"
//   - Non-STOCK fund, BuyOrderDate and price/shares provided:   "COMPLETED"
//
// Note: Once a dividend is used in a portfolio (has transactions), it becomes permanent
// and cannot be deleted. This preserves portfolio history and dividend price data.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: CreateDividendRequest containing all required dividend fields
//
// Returns the created dividend with its generated ID, or an error if creation fails.
func (s *DividendService) CreateDividend(ctx context.Context, req request.CreateDividendRequest) (*model.DividendFund, error) {
	divLog.DebugContext(ctx, "creating dividend", "portfolioFundID", req.PortfolioFundID)
	portfolioFund, err := s.pfRepo.GetPortfolioFundListing(req.PortfolioFundID)
	if err != nil {
		return nil, fmt.Errorf("get portfolio fund listing: %w", err)
	}

	if portfolioFund.DividendType == "None" {
		return nil, fmt.Errorf("this fund does not pay out dividends")
	}

	recordDate, err := time.Parse("2006-01-02", req.RecordDate)
	if err != nil {
		return nil, fmt.Errorf("parse record date: %w", err)
	}

	exDividendDate, err := time.Parse("2006-01-02", req.ExDividendDate)
	if err != nil {
		return nil, fmt.Errorf("parse ex-dividend date: %w", err)
	}

	shares, err := s.transactionRepo.GetSharesOnDate(req.PortfolioFundID, exDividendDate)
	if err != nil {
		return nil, fmt.Errorf("get shares on date: %w", err)
	}

	dividend := &model.Dividend{
		ID:               uuid.New().String(),
		FundID:           portfolioFund.FundID,
		PortfolioFundID:  req.PortfolioFundID,
		RecordDate:       recordDate,
		ExDividendDate:   exDividendDate,
		DividendPerShare: req.DividendPerShare,
		SharesOwned:      shares,
		TotalAmount:      shares * req.DividendPerShare,
		CreatedAt:        time.Now().UTC(),
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if req.BuyOrderDate != "" {
		if err := s.applyReinvestment(ctx, tx, portfolioFund, dividend, req); err != nil {
			return nil, fmt.Errorf("apply reinvestment: %w", err)
		}
	} else if portfolioFund.DividendType == "STOCK" {
		dividend.ReinvestmentStatus = "PENDING"
	} else {
		dividend.ReinvestmentStatus = "COMPLETED"
	}

	if err := s.dividendRepo.WithTx(tx).InsertDividend(ctx, dividend); err != nil {
		return nil, fmt.Errorf("failed to create dividend: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), dividend.ExDividendDate, nil, "", req.PortfolioFundID); err != nil {
				divLog.Warn("failed to regenerate materialized table", "error", err)
			}
		}()
	}

	divLog.InfoContext(ctx, "dividend created", "dividendID", dividend.ID, "portfolioFundID", req.PortfolioFundID, "totalAmount", dividend.TotalAmount)
	return dividendToFund(*dividend, portfolioFund), nil
}

// UpdateDividend updates an existing dividend with the provided changes.
// Only fields present in the request (non-nil) are updated; all others retain their current values.
// SharesOwned and TotalAmount are always recalculated from transactions as of the (possibly updated)
// ex-dividend date.
//
// ReinvestmentStatus is re-evaluated after the update using the same rules as CreateDividend:
//
//   - STOCK fund, no BuyOrderDate:                                       "PENDING"
//   - STOCK fund, BuyOrderDate set, price/shares missing:                "PENDING"
//   - STOCK fund, BuyOrderDate set, reinvested == total amount:          "COMPLETED"
//   - STOCK fund, BuyOrderDate set, reinvested < total amount:           "PARTIAL"
//   - Non-STOCK fund, no BuyOrderDate:                                   "COMPLETED"
//   - Non-STOCK fund, BuyOrderDate and price/shares provided:            "COMPLETED"
//   - Any fund, existing COMPLETED status, no new reinvestment info:     "COMPLETED" (preserved)
//
// If reinvestment info is provided and no reinvestment transaction exists yet, a new one is created.
// If a reinvestment transaction already exists, it is updated in the same database transaction.
//
// When ex_dividend_date changes, materialized view regeneration starts from the earlier of
// old/new dates to ensure both ranges are recalculated (Issue #35, Edge Case 5).
//
// Returns ErrDividendNotFound if the dividend does not exist.
// Returns an error if date parsing fails or any database operation fails.
//
//nolint:gocyclo // Complex update with reinvestment lifecycle + materialized invalidation
func (s *DividendService) UpdateDividend(
	ctx context.Context,
	id string,
	req request.UpdateDividendRequest,
) (*model.DividendFund, error) {
	divLog.DebugContext(ctx, "updating dividend", "dividendID", id)
	dividend, err := s.dividendRepo.GetDividend(id)
	if err != nil {
		return nil, fmt.Errorf("get dividend: %w", err)
	}

	oldExDividendDate := dividend.ExDividendDate

	if req.PortfolioFundID != nil {
		dividend.PortfolioFundID = *req.PortfolioFundID
	}

	portfolioFund, err := s.pfRepo.GetPortfolioFundListing(dividend.PortfolioFundID)
	if err != nil {
		return nil, fmt.Errorf("get portfolio fund listing: %w", err)
	}

	if err := applyUpdateFields(&dividend, req); err != nil {
		return nil, fmt.Errorf("apply update fields: %w", err)
	}

	shares, err := s.transactionRepo.GetSharesOnDate(dividend.PortfolioFundID, dividend.ExDividendDate)
	if err != nil {
		return nil, fmt.Errorf("get shares on date: %w", err)
	}

	dividend.SharesOwned = shares
	dividend.TotalAmount = shares * dividend.DividendPerShare
	dividend.CreatedAt = time.Now().UTC() // Intentional: tracks latest modification, not original creation.

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if req.BuyOrderDate != nil || !dividend.BuyOrderDate.IsZero() {
		if err := s.applyUpdateReinvestment(ctx, tx, portfolioFund, &dividend, req); err != nil {
			return nil, fmt.Errorf("apply update reinvestment: %w", err)
		}
	} else if portfolioFund.DividendType == "STOCK" {
		dividend.ReinvestmentStatus = "PENDING"
	} else {
		dividend.ReinvestmentStatus = "COMPLETED"
	}

	if err := s.dividendRepo.WithTx(tx).UpdateDividend(ctx, &dividend); err != nil {
		return nil, fmt.Errorf("failed to update dividend: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		regenDate := dividend.ExDividendDate
		if oldExDividendDate.Before(regenDate) {
			regenDate = oldExDividendDate
		}
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), regenDate, nil, "", dividend.PortfolioFundID); err != nil {
				divLog.Warn("failed to regenerate materialized table", "error", err)
			}
		}()
	}

	divLog.InfoContext(ctx, "dividend updated", "dividendID", id)
	return dividendToFund(dividend, portfolioFund), nil
}

// applyReinvestment parses the BuyOrderDate and sets ReinvestmentStatus on the dividend.
// For STOCK funds with reinvestment price and shares, it also creates a "dividend" transaction
// within the provided database transaction.
func (s *DividendService) applyReinvestment(ctx context.Context, tx *sql.Tx, portfolioFund model.PortfolioFundListing, dividend *model.Dividend, req request.CreateDividendRequest) error {
	var err error
	dividend.BuyOrderDate, err = time.Parse("2006-01-02", req.BuyOrderDate)
	if err != nil {
		return fmt.Errorf("parse buy order date: %w", err)
	}

	hasReinvestmentInfo := req.ReinvestmentPrice > 0.0 && req.ReinvestmentShares > 0.0

	if portfolioFund.DividendType == "STOCK" && hasReinvestmentInfo {
		return s.createReinvestmentTransaction(ctx, tx, dividend, req)
	}

	if hasReinvestmentInfo {
		// Non-STOCK fund with reinvestment info: mark complete, no transaction needed.
		dividend.ReinvestmentStatus = "COMPLETED"
		return nil
	}

	dividend.ReinvestmentStatus = "PENDING"
	return nil
}

// applyUpdateReinvestment updates or creates a reinvestment transaction and recalculates
// ReinvestmentStatus for an existing dividend.
//
// If BuyOrderDate is in the request it replaces the existing value on the dividend.
// When reinvestment price and shares are both provided:
//   - STOCK funds: updates the existing reinvestment transaction (if any), or creates a new one.
//   - Non-STOCK funds: marks the dividend COMPLETED without creating a transaction.
//
// When no reinvestment info is provided, COMPLETED status is preserved; all other statuses
// are set to PENDING.
func (s *DividendService) applyUpdateReinvestment(ctx context.Context, tx *sql.Tx, portfolioFund model.PortfolioFundListing, dividend *model.Dividend, req request.UpdateDividendRequest) error {
	if req.BuyOrderDate != nil {
		buyOrderDate, err := time.Parse("2006-01-02", *req.BuyOrderDate)
		if err != nil {
			return fmt.Errorf("parse buy order date: %w", err)
		}
		dividend.BuyOrderDate = buyOrderDate.UTC()
	}

	var hasReinvestmentInfo bool
	if req.ReinvestmentShares != nil && req.ReinvestmentPrice != nil {
		hasReinvestmentInfo = *req.ReinvestmentPrice > 0.0 && *req.ReinvestmentShares > 0.0
	}

	if portfolioFund.DividendType == "STOCK" && hasReinvestmentInfo {
		if dividend.ReinvestmentTransactionID != "" {
			return s.updateReinvestmentTransaction(ctx, tx, dividend, req)
		}
		reqCreate := request.CreateDividendRequest{
			PortfolioFundID:    dividend.PortfolioFundID,
			ReinvestmentShares: *req.ReinvestmentShares,
			ReinvestmentPrice:  *req.ReinvestmentPrice,
		}
		return s.createReinvestmentTransaction(ctx, tx, dividend, reqCreate)
	}

	if hasReinvestmentInfo {
		// Non-STOCK fund with reinvestment info: mark complete, no transaction needed.
		dividend.ReinvestmentStatus = "COMPLETED"
		return nil
	}

	if dividend.ReinvestmentStatus != "COMPLETED" {
		dividend.ReinvestmentStatus = "PENDING"
	}

	return nil
}

// createReinvestmentTransaction inserts a "dividend" transaction for a STOCK fund reinvestment
// and sets ReinvestmentStatus to "COMPLETED" or "PARTIAL" based on whether the reinvested
// amount matches the total dividend amount.
func (s *DividendService) createReinvestmentTransaction(ctx context.Context, tx *sql.Tx, dividend *model.Dividend, req request.CreateDividendRequest) error {
	transactionID := uuid.New().String()

	transaction := &model.Transaction{
		ID:              transactionID,
		PortfolioFundID: req.PortfolioFundID,
		Date:            dividend.BuyOrderDate,
		Type:            "dividend",
		Shares:          req.ReinvestmentShares,
		CostPerShare:    req.ReinvestmentPrice,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.transactionRepo.WithTx(tx).InsertTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to create reinvestment transaction: %w", err)
	}

	dividend.ReinvestmentTransactionID = transactionID
	if round(req.ReinvestmentShares*req.ReinvestmentPrice) == round(dividend.TotalAmount) {
		dividend.ReinvestmentStatus = "COMPLETED"
	} else {
		dividend.ReinvestmentStatus = "PARTIAL"
	}

	return nil
}

// updateReinvestmentTransaction updates the fields of the existing reinvestment transaction
// linked to the dividend, and recalculates ReinvestmentStatus (COMPLETED or PARTIAL) based
// on whether the new reinvested amount equals the dividend's TotalAmount.
func (s *DividendService) updateReinvestmentTransaction(ctx context.Context, tx *sql.Tx, dividend *model.Dividend, req request.UpdateDividendRequest) error {
	transaction, err := s.transactionRepo.GetTransactionByID(dividend.ReinvestmentTransactionID)
	if err != nil {
		return fmt.Errorf("get reinvestment transaction: %w", err)
	}
	transaction.PortfolioFundID = dividend.PortfolioFundID
	transaction.Date = dividend.BuyOrderDate
	transaction.Shares = *req.ReinvestmentShares
	transaction.CostPerShare = *req.ReinvestmentPrice
	transaction.CreatedAt = time.Now().UTC() // Intentional: tracks latest modification, not original creation.

	if err := s.transactionRepo.WithTx(tx).UpdateTransaction(ctx, &transaction); err != nil {
		return fmt.Errorf("failed to update reinvestment transaction: %w", err)
	}

	if round(*req.ReinvestmentShares**req.ReinvestmentPrice) == round(dividend.TotalAmount) {
		dividend.ReinvestmentStatus = "COMPLETED"
	} else {
		dividend.ReinvestmentStatus = "PARTIAL"
	}

	return nil
}

// dividendToFund maps a Dividend and its associated PortfolioFundListing into a DividendFund response.
func dividendToFund(d model.Dividend, pf model.PortfolioFundListing) *model.DividendFund {
	var buyOrderDate *time.Time
	if !d.BuyOrderDate.IsZero() {
		t := d.BuyOrderDate
		buyOrderDate = &t
	}

	return &model.DividendFund{
		ID:                        d.ID,
		FundID:                    d.FundID,
		FundName:                  pf.FundName,
		PortfolioFundID:           d.PortfolioFundID,
		RecordDate:                d.RecordDate,
		ExDividendDate:            d.ExDividendDate,
		SharesOwned:               d.SharesOwned,
		DividendPerShare:          d.DividendPerShare,
		TotalAmount:               d.TotalAmount,
		ReinvestmentStatus:        d.ReinvestmentStatus,
		BuyOrderDate:              buyOrderDate,
		ReinvestmentTransactionID: d.ReinvestmentTransactionID,
		DividendType:              pf.DividendType,
	}
}

// applyUpdateFields applies optional field updates from an UpdateDividendRequest onto a Dividend.
// Parses and assigns RecordDate, ExDividendDate, and DividendPerShare when provided.
func applyUpdateFields(dividend *model.Dividend, req request.UpdateDividendRequest) error {
	if req.RecordDate != nil {
		recordDate, err := time.Parse("2006-01-02", *req.RecordDate)
		if err != nil {
			return fmt.Errorf("parse record date: %w", err)
		}
		dividend.RecordDate = recordDate.UTC()
	}

	if req.ExDividendDate != nil {
		exDividendDate, err := time.Parse("2006-01-02", *req.ExDividendDate)
		if err != nil {
			return fmt.Errorf("parse ex-dividend date: %w", err)
		}
		dividend.ExDividendDate = exDividendDate.UTC()
	}

	if req.DividendPerShare != nil {
		dividend.DividendPerShare = *req.DividendPerShare
	}

	return nil
}

// DeleteDividend removes a dividend and its associated reinvestment transaction atomically.
// The dividend row is deleted first (since it holds the FK reference to the transaction),
// then the reinvestment transaction is deleted if one exists.
//
// Returns ErrDividendNotFound if the dividend does not exist.
// Returns an error if any database operation fails; both deletions are rolled back on error.
func (s *DividendService) DeleteDividend(ctx context.Context, id string) error {
	divLog.DebugContext(ctx, "deleting dividend", "dividendID", id)

	dividend, err := s.dividendRepo.GetDividend(id)
	if err != nil {
		return fmt.Errorf("get dividend: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if err := s.dividendRepo.WithTx(tx).DeleteDividend(ctx, id); err != nil {
		return fmt.Errorf("failed to delete dividend: %w", err)
	}

	if dividend.ReinvestmentTransactionID != "" {
		if err := s.transactionRepo.WithTx(tx).DeleteTransaction(ctx, dividend.ReinvestmentTransactionID); err != nil {
			return fmt.Errorf("failed to delete reinvestment transaction: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	divLog.InfoContext(ctx, "dividend deleted", "dividendID", id)

	if s.materializedInvalidator != nil {
		//nolint:gosec // G118: Background context is intentional — goroutine outlives the HTTP request.
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), dividend.ExDividendDate, nil, "", dividend.PortfolioFundID); err != nil {
				divLog.Warn("failed to regenerate materialized table", "error", err)
			}
		}()
	}

	return nil
}
