package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

var pfLog = logging.NewLogger("portfolio")

// PortfolioFundRepository provides data access methods for the portfolio_fund join table.
type PortfolioFundRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewPortfolioFundRepository creates a new PortfolioFundRepository with the provided database connection.
func NewPortfolioFundRepository(db *sql.DB) *PortfolioFundRepository {
	return &PortfolioFundRepository{db: db}
}

// WithTx returns a new PortfolioFundRepository scoped to the provided transaction.
func (r *PortfolioFundRepository) WithTx(tx *sql.Tx) *PortfolioFundRepository {
	return &PortfolioFundRepository{
		db: r.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (r *PortfolioFundRepository) getQuerier() Querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetPortfolioFund retrieves a single portfolio_fund record by its ID.
// Returns ErrPortfolioFundNotFound if no record with the given ID exists.
func (r *PortfolioFundRepository) GetPortfolioFund(pfID string) (model.PortfolioFund, error) {
	pfLog.Debug("getting portfolio fund", "portfolio_fund_id", pfID)
	if pfID == "" {
		return model.PortfolioFund{}, apperrors.ErrInvalidPortfolioID
	}

	query := `
		SELECT id, portfolio_id, fund_id
		FROM portfolio_fund
		WHERE id = ?
	`

	var pf model.PortfolioFund
	err := r.getQuerier().QueryRow(query, pfID).Scan(
		&pf.ID,
		&pf.PortfolioID,
		&pf.FundID,
	)
	if err == sql.ErrNoRows {
		return model.PortfolioFund{}, apperrors.ErrPortfolioFundNotFound
	}
	if err != nil {
		return model.PortfolioFund{}, fmt.Errorf("failed to query portfolio fund: %w", err)
	}

	return pf, nil
}

// GetPortfolioFundListing retrieves a single portfolio-fund relationship with enriched metadata by its ID.
// Returns ErrFailedToRetrievePortfolioFunds if no record with the given ID exists.
// Note: includes archived portfolios — archive status is irrelevant for ID-based lookups.
func (r *PortfolioFundRepository) GetPortfolioFundListing(portfolioFundID string) (model.PortfolioFundListing, error) {
	pfLog.Debug("getting portfolio fund listing", "portfolio_fund_id", portfolioFundID)
	query := `
		SELECT
			pf.id,
			pf.portfolio_id,
			f.id as fund_id,
			p.name as portfolio_name,
			f.name as fund_name,
			f.dividend_type
		FROM portfolio_fund pf
		JOIN portfolio p ON pf.portfolio_id = p.id
		JOIN fund f ON pf.fund_id = f.id
		WHERE pf.id = ?
	`

	var l model.PortfolioFundListing
	err := r.getQuerier().QueryRow(query, portfolioFundID).Scan(
		&l.ID,
		&l.PortfolioID,
		&l.FundID,
		&l.PortfolioName,
		&l.FundName,
		&l.DividendType,
	)
	if err == sql.ErrNoRows {
		return model.PortfolioFundListing{}, apperrors.ErrFailedToRetrievePortfolioFunds
	}
	if err != nil {
		return model.PortfolioFundListing{}, fmt.Errorf("failed to query portfolio_fund listing: %w", err)
	}

	return l, nil
}

// GetAllPortfolioFundListings retrieves all portfolio-fund relationships with enriched metadata.
// Excludes archived portfolios. Used for the GET /api/portfolio/funds endpoint.
func (r *PortfolioFundRepository) GetAllPortfolioFundListings() ([]model.PortfolioFundListing, error) {
	pfLog.Debug("getting all portfolio fund listings")
	query := `
		SELECT
			pf.id,
			pf.portfolio_id,
			f.id as fund_id,
			p.name as portfolio_name,
			f.name as fund_name,
			f.dividend_type
		FROM portfolio_fund pf
		JOIN portfolio p ON pf.portfolio_id = p.id
		JOIN fund f ON pf.fund_id = f.id
		WHERE p.is_archived = 0
		ORDER BY p.name ASC, f.name ASC
	`

	rows, err := r.getQuerier().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio_fund listings: %w", err)
	}
	defer rows.Close()

	listings := []model.PortfolioFundListing{}

	for rows.Next() {
		var l model.PortfolioFundListing
		err := rows.Scan(
			&l.ID,
			&l.PortfolioID,
			&l.FundID,
			&l.PortfolioName,
			&l.FundName,
			&l.DividendType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio_fund listing: %w", err)
		}
		listings = append(listings, l)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolio_fund listings: %w", err)
	}

	return listings, nil
}

// GetPortfolioFunds retrieves funds associated with a portfolio via the portfolio_fund join table.
// If PortfolioID is empty, returns all portfolio-fund relationships.
func (r *PortfolioFundRepository) GetPortfolioFunds(PortfolioID string) ([]model.PortfolioFundResponse, error) {
	pfLog.Debug("getting portfolio funds", "portfolio_id", PortfolioID)
	fundQuery := `
		SELECT
		portfolio_fund.id,
		fund.id, fund.name, fund.investment_type, fund.dividend_type
		FROM portfolio_fund
		JOIN fund ON fund.id = portfolio_fund.fund_id
		WHERE 1=1
	`

	var fundArgs []any

	if PortfolioID != "" {
		fundQuery += " AND portfolio_fund.portfolio_id = ?"
		fundArgs = append(fundArgs, PortfolioID)
	}

	rows, err := r.getQuerier().Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve portfolio funds via portfolio_fund JOIN (portfolio_id=%s): %w", PortfolioID, err)
	}
	defer rows.Close()

	portfolioFunds := []model.PortfolioFundResponse{}

	for rows.Next() {
		var f model.PortfolioFundResponse
		err := rows.Scan(
			&f.ID,
			&f.FundID,
			&f.FundName,
			&f.InvestmentType,
			&f.DividendType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fund or portfolio_fund table results: %w", err)
		}
		portfolioFunds = append(portfolioFunds, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolio_fund JOIN results: %w", err)
	}

	return portfolioFunds, nil
}

// GetPortfolioFundsbyFundID retrieves all portfolio_fund records for a given fund ID.
// Returns ErrPortfolioFundNotFound if the fund is not assigned to any portfolio.
func (r *PortfolioFundRepository) GetPortfolioFundsbyFundID(fundID string) ([]model.PortfolioFund, error) {
	pfLog.Debug("getting portfolio funds by fund ID", "fund_id", fundID)
	if fundID == "" {
		return nil, apperrors.ErrInvalidFundID
	}

	query := `
		SELECT id, portfolio_id, fund_id
		FROM portfolio_fund
		WHERE fund_id = ?
	`

	rows, err := r.getQuerier().Query(query, fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio_fund listings: %w", err)
	}
	defer rows.Close()

	pfs := []model.PortfolioFund{}

	for rows.Next() {
		var pf model.PortfolioFund
		err := rows.Scan(
			&pf.ID,
			&pf.PortfolioID,
			&pf.FundID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio_fund listing: %w", err)
		}
		pfs = append(pfs, pf)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolio_fund listings: %w", err)
	}

	if len(pfs) == 0 {
		return nil, apperrors.ErrPortfolioFundNotFound
	}

	return pfs, nil
}

// GetPortfolioFundsOnPortfolioID retrieves funds for a set of portfolios, returning several
// lookup structures needed for calculation pipelines.
// Returns nil for all values if portfolios is empty.
func (r *PortfolioFundRepository) GetPortfolioFundsOnPortfolioID(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	pfLog.Debug("getting portfolio funds on portfolio IDs", "portfolio_count", len(portfolios))
	if len(portfolios) == 0 {
		return nil, nil, nil, nil, nil, nil
	}

	portfolioPlaceholders := make([]string, len(portfolios))
	for i := range portfolioPlaceholders {
		portfolioPlaceholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	fundQuery := `
		SELECT
		portfolio_fund.id, portfolio_fund.portfolio_id,
		fund.id, fund.name, fund.isin, fund.symbol, fund.currency, fund.exchange, fund.investment_type, fund.dividend_type
		FROM portfolio_fund
		JOIN fund ON fund.id = portfolio_fund.fund_id
		WHERE portfolio_fund.portfolio_id IN (` + strings.Join(portfolioPlaceholders, ",") + `)
	`

	fundArgs := make([]any, len(portfolios))
	for i, p := range portfolios {
		fundArgs[i] = p.ID
	}

	rows, err := r.getQuerier().Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to query portfolio_fund or funds table: %w", err)
	}
	defer rows.Close()

	fundsByPortfolio := make(map[string][]model.Fund)
	portfolioFundToPortfolio := make(map[string]string)
	portfolioFundToFund := make(map[string]string)
	var fundIDs, pfIDs []string

	for rows.Next() {
		var pfID string
		var portfolioID string
		var f model.Fund

		err := rows.Scan(
			&pfID,
			&portfolioID,
			&f.ID,
			&f.Name,
			&f.Isin,
			&f.Symbol,
			&f.Currency,
			&f.Exchange,
			&f.InvestmentType,
			&f.DividendType,
		)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to scan funds table results: %w", err)
		}

		fundsByPortfolio[portfolioID] = append(fundsByPortfolio[portfolioID], f)
		portfolioFundToPortfolio[pfID] = portfolioID
		portfolioFundToFund[pfID] = f.ID
		pfIDs = append(pfIDs, pfID)
		fundIDs = append(fundIDs, f.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("error iterating funds table: %w", err)
	}

	return fundsByPortfolio, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, nil
}

// CheckUsage checks if a fund is in use by any portfolios and returns usage details.
// Returns nil if the fund is not assigned to any portfolio.
func (r *PortfolioFundRepository) CheckUsage(fundID string) ([]model.PortfolioTransaction, error) {
	pfLog.Debug("checking fund usage", "fund_id", fundID)
	pfs, err := r.GetPortfolioFundsbyFundID(fundID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get portfolio funds by fund ID: %w", err)
	}

	pfIDs := make([]any, len(pfs))
	placeholders := make([]string, len(pfs))
	for i, v := range pfs {
		pfIDs[i] = v.ID
		placeholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	query := `
		SELECT pf.portfolio_id, p.name, COALESCE(COUNT(t.id), 0) as transaction_count
        FROM portfolio_fund pf
        JOIN portfolio p ON p.id = pf.portfolio_id
        LEFT JOIN "transaction" t ON t.portfolio_fund_id = pf.id
        WHERE pf.id IN (` + strings.Join(placeholders, ",") + `)
        GROUP BY pf.portfolio_id, p.name
	`

	PFTs := make([]model.PortfolioTransaction, 0, len(pfs))

	rows, err := r.getQuerier().Query(query, pfIDs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pft model.PortfolioTransaction
		err := rows.Scan(
			&pft.ID,
			&pft.Name,
			&pft.TransactionCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction or portfolio_fund table results: %w", err)
		}
		PFTs = append(PFTs, pft)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction or portfolio_fund listings: %w", err)
	}

	return PFTs, nil
}

// GetPortfolioFundByPortfolioAndFund retrieves a portfolio_fund record by both portfolio and fund IDs.
// Returns ErrPortfolioFundNotFound if no matching relationship exists.
func (r *PortfolioFundRepository) GetPortfolioFundByPortfolioAndFund(portfolioID, fundID string) (model.PortfolioFund, error) {
	pfLog.Debug("getting portfolio fund by portfolio and fund", "portfolio_id", portfolioID, "fund_id", fundID)
	query := `
		SELECT id, portfolio_id, fund_id
		FROM portfolio_fund
		WHERE portfolio_id = ? AND fund_id = ?
	`

	var pf model.PortfolioFund
	err := r.getQuerier().QueryRow(query, portfolioID, fundID).Scan(
		&pf.ID,
		&pf.PortfolioID,
		&pf.FundID,
	)
	if err == sql.ErrNoRows {
		return model.PortfolioFund{}, apperrors.ErrPortfolioFundNotFound
	}
	if err != nil {
		return model.PortfolioFund{}, fmt.Errorf("failed to query portfolio_fund by portfolio and fund: %w", err)
	}

	return pf, nil
}

// InsertPortfolioFund creates a new portfolio_fund relationship between a portfolio and a fund.
func (r *PortfolioFundRepository) InsertPortfolioFund(ctx context.Context, portfolioID, fundID string) error {
	pfLog.DebugContext(ctx, "inserting portfolio fund", "portfolio_id", portfolioID, "fund_id", fundID)
	query := `
        INSERT INTO portfolio_fund (id, portfolio_id, fund_id)
        VALUES (?, ?, ?)
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		uuid.New().String(),
		portfolioID,
		fundID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert portfolio_fund: %w", err)
	}

	return nil
}

// DeletePortfolioFund removes a portfolio_fund relationship by its ID.
// Returns ErrPortfolioFundNotFound if no record with the given ID exists.
func (r *PortfolioFundRepository) DeletePortfolioFund(ctx context.Context, portfolioFundID string) error {
	pfLog.DebugContext(ctx, "deleting portfolio fund", "portfolio_fund_id", portfolioFundID)
	query := `DELETE FROM portfolio_fund WHERE id = ?`

	result, err := r.getQuerier().ExecContext(ctx, query, portfolioFundID)
	if err != nil {
		return fmt.Errorf("failed to delete portfolio_fund: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrPortfolioFundNotFound
	}

	return nil
}
