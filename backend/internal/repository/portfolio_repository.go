package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

var portfolioLog = logging.NewLogger("portfolio")

// PortfolioRepository provides data access methods for portfolio and portfolio_fund tables.
// It handles retrieving portfolio metadata and their associated fund relationships.
type PortfolioRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewPortfolioRepository creates a new PortfolioRepository with the provided database connection.
func NewPortfolioRepository(db *sql.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// WithTx returns a new PortfolioRepository scoped to the provided transaction.
func (r *PortfolioRepository) WithTx(tx *sql.Tx) *PortfolioRepository {
	return &PortfolioRepository{
		db: r.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (r *PortfolioRepository) getQuerier() Querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetPortfolios retrieves portfolios from the database based on filter criteria.
// The filter allows control over whether archived and overview-excluded portfolios are included.
// Returns an empty slice if no portfolios match the filter criteria.
func (r *PortfolioRepository) GetPortfolios(filter model.PortfolioFilter) ([]model.Portfolio, error) {
	portfolioLog.Debug("getting portfolios", "include_archived", filter.IncludeArchived, "include_excluded", filter.IncludeExcluded)
	query := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
          WHERE 1=1
      `
	var args []any

	if !filter.IncludeArchived {
		query += " AND is_archived = ?"
		args = append(args, 0)
	}

	if !filter.IncludeExcluded {
		query += " AND exclude_from_overview = ?"
		args = append(args, 0)
	}

	rows, err := r.getQuerier().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolios table: %w", err)
	}
	defer rows.Close()

	portfolios := []model.Portfolio{}

	for rows.Next() {
		var p model.Portfolio

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.IsArchived,
			&p.ExcludeFromOverview,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio table results: %w", err)
		}

		portfolios = append(portfolios, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolios table: %w", err)
	}

	return portfolios, nil
}

// GetPortfolioOnID retrieves a single portfolio by its ID.
func (r *PortfolioRepository) GetPortfolioOnID(portfolioID string) (model.Portfolio, error) {
	portfolioLog.Debug("getting portfolio by ID", "portfolio_id", portfolioID)
	query := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
          WHERE id = ?
      `
	var p model.Portfolio

	err := r.getQuerier().QueryRow(query, portfolioID).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.IsArchived,
		&p.ExcludeFromOverview,
	)
	if err == sql.ErrNoRows {
		return model.Portfolio{}, apperrors.ErrPortfolioNotFound
	}
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("failed to query portfolio: %w", err)
	}

	return p, nil
}

// GetPortfoliosByFundID retrieves all portfolios that hold a specific fund.
// Joins the portfolio and portfolio_fund tables to find portfolios where the fund is assigned.
// Returns an empty slice if the fund is not assigned to any portfolios (not an error).
//
// Parameters:
//   - fundID: The UUID of the fund
//
// Returns a slice of portfolios that hold this fund, or an error if the database query fails.
func (r *PortfolioRepository) GetPortfoliosByFundID(fundID string) ([]model.Portfolio, error) {
	portfolioLog.Debug("getting portfolios by fund ID", "fund_id", fundID)

	fundQuery := `
		SELECT p.id, p.name, p.description, p.is_archived, p.exclude_from_overview
        FROM portfolio p
		INNER JOIN portfolio_fund pf
		ON pf.portfolio_id = p.id
		WHERE pf.fund_id = ?
	`

	rows, err := r.getQuerier().Query(fundQuery, fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio_fund or portfolio table: %w", err)
	}
	defer rows.Close()

	portfolios := []model.Portfolio{}

	for rows.Next() {
		var p model.Portfolio

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.IsArchived,
			&p.ExcludeFromOverview,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio_fund or portfolio table results: %w", err)
		}

		portfolios = append(portfolios, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolio_fund or portfolio table: %w", err)
	}

	return portfolios, nil
}

// InsertPortfolio inserts a new portfolio record and returns its generated ID.
func (r *PortfolioRepository) InsertPortfolio(ctx context.Context, p *model.Portfolio) error {
	portfolioLog.DebugContext(ctx, "inserting portfolio", "portfolio_id", p.ID, "name", p.Name)
	query := `
        INSERT INTO portfolio (id, name, description, is_archived, exclude_from_overview)
        VALUES (?, ?, ?, ?, ?)
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		p.ID,
		p.Name,
		p.Description,
		p.IsArchived,
		p.ExcludeFromOverview,
	)

	if err != nil {
		return fmt.Errorf("failed to insert portfolio: %w", err)
	}

	return nil
}

// UpdatePortfolio updates the name, description, and flags of an existing portfolio.
func (r *PortfolioRepository) UpdatePortfolio(ctx context.Context, p *model.Portfolio) error {
	portfolioLog.DebugContext(ctx, "updating portfolio", "portfolio_id", p.ID)
	query := `
        UPDATE portfolio
        SET name = ?, description = ?, is_archived = ?, exclude_from_overview = ?
        WHERE id = ?
    `

	result, err := r.getQuerier().ExecContext(ctx, query,
		p.Name,
		p.Description,
		p.IsArchived,
		p.ExcludeFromOverview,
		p.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrPortfolioNotFound
	}

	return nil
}

// DeletePortfolio removes a portfolio record from the database by its ID.
func (r *PortfolioRepository) DeletePortfolio(ctx context.Context, portfolioID string) error {
	portfolioLog.DebugContext(ctx, "deleting portfolio", "portfolio_id", portfolioID)
	query := `DELETE FROM portfolio WHERE id = ?`

	result, err := r.getQuerier().ExecContext(ctx, query, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to delete portfolio: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrPortfolioNotFound
	}

	return nil
}
