package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var pfLog = logging.NewLogger("portfolio")

// PortfolioService handles portfolio-related business logic operations.
// It coordinates between multiple repositories to compute portfolio summaries
// and aggregate metrics.
type PortfolioService struct {
	db            *sql.DB
	portfolioRepo *repository.PortfolioRepository
	pfRepo        *repository.PortfolioFundRepository
}

// NewPortfolioService creates a new PortfolioService with the provided repository dependencies.
func NewPortfolioService(
	db *sql.DB,
	portfolioRepo *repository.PortfolioRepository,
	pfRepo *repository.PortfolioFundRepository,
) *PortfolioService {
	return &PortfolioService{
		db:            db,
		portfolioRepo: portfolioRepo,
		pfRepo:        pfRepo,
	}
}

// GetAllPortfolios retrieves all portfolios from the database with no filters applied.
// This includes both archived and excluded portfolios.
func (s *PortfolioService) GetAllPortfolios() ([]model.Portfolio, error) {
	pfLog.Debug("retrieving all portfolios")
	result, err := s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: true,
		IncludeExcluded: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get all portfolios: %w", err)
	}
	return result, nil
}

// GetPortfolio retrieves a single portfolio by its ID.
// Returns the portfolio metadata including name, description, and archive status.
// This is a simple wrapper around the repository layer for portfolio lookup.
func (s *PortfolioService) GetPortfolio(portfolioID string) (model.Portfolio, error) {
	pfLog.Debug("retrieving portfolio", "portfolioID", portfolioID)
	result, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("get portfolio: %w", err)
	}
	return result, nil
}

// LoadActivePortfolios retrieves only active, non-excluded portfolios.
// Archived and excluded portfolios are filtered out.
func (s *PortfolioService) LoadActivePortfolios() ([]model.Portfolio, error) {
	pfLog.Debug("loading active portfolios")
	result, err := s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
	if err != nil {
		return nil, fmt.Errorf("load active portfolios: %w", err)
	}
	return result, nil
}

// LoadAllPortfolioFunds retrieves all funds associated with the given portfolios.
// Returns:
//   - fundsByPortfolio: map[portfolioID][]Fund
//   - portfolioFundToPortfolio: map[portfolioFundID]portfolioID
//   - portfolioFundToFund: map[portfolioFundID]fundID
//   - pfIDs: slice of all portfolio_fund IDs
//   - fundIDs: slice of all unique fund IDs
//   - error: any error encountered
func (s *PortfolioService) LoadAllPortfolioFunds(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	pfLog.Debug("loading all portfolio funds", "portfolioCount", len(portfolios))
	fundsByPortfolio, pfToPortfolio, pfToFund, pfIDs, fundIDs, err := s.pfRepo.GetPortfolioFundsOnPortfolioID(portfolios)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("load portfolio funds: %w", err)
	}
	return fundsByPortfolio, pfToPortfolio, pfToFund, pfIDs, fundIDs, nil
}

// GetPortfoliosForRequest resolves a portfolio ID parameter into a slice of portfolios.
// This is a helper method that handles the common pattern where API endpoints accept
// an optional portfolio ID - if provided, return that one portfolio; if empty, return
// all active portfolios.
//
// This centralizes the "specific portfolio vs all portfolios" logic that was duplicated
// across multiple service methods.
//
// Parameters:
//   - portfolioID: Optional portfolio ID. Empty string means "all portfolios"
//
// Returns:
//   - If portfolioID is provided: a slice containing just that portfolio
//   - If portfolioID is empty: all active, non-excluded portfolios
//   - Error if the specific portfolio ID is not found or database query fails
//
// Usage in other services:
//
//	portfolios, err := portfolioService.GetPortfoliosForRequest(portfolioID)
//	// portfolios is always a slice, simplifying downstream code
func (s *PortfolioService) GetPortfoliosForRequest(portfolioID string) ([]model.Portfolio, error) {
	pfLog.Debug("resolving portfolios for request", "portfolioID", portfolioID)
	if portfolioID != "" {
		portfolio, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
		if err != nil {
			return []model.Portfolio{}, fmt.Errorf("get portfolio: %w", err)
		}

		return []model.Portfolio{portfolio}, nil
	}

	// Load all active portfolios
	result, err := s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
	if err != nil {
		return nil, fmt.Errorf("get active portfolios: %w", err)
	}
	return result, nil
}

// CreatePortfolio creates a new portfolio record and returns the created portfolio.
func (s *PortfolioService) CreatePortfolio(ctx context.Context, req request.CreatePortfolioRequest) (*model.Portfolio, error) {
	pfLog.DebugContext(ctx, "creating portfolio", "name", req.Name)
	portfolio := &model.Portfolio{
		ID:                  uuid.New().String(),
		Name:                req.Name,
		Description:         req.Description,
		IsArchived:          false,
		ExcludeFromOverview: req.ExcludeFromOverview,
	}

	if err := s.portfolioRepo.InsertPortfolio(ctx, portfolio); err != nil {
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}

	pfLog.InfoContext(ctx, "portfolio created", "portfolioID", portfolio.ID, "name", portfolio.Name)
	return portfolio, nil
}

// UpdatePortfolio updates the name, description, and flags of an existing portfolio.
func (s *PortfolioService) UpdatePortfolio(
	ctx context.Context,
	id string,
	req request.UpdatePortfolioRequest,
) (*model.Portfolio, error) {
	pfLog.DebugContext(ctx, "updating portfolio", "portfolioID", id)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	portfolio, err := s.portfolioRepo.WithTx(tx).GetPortfolioOnID(id)
	if err != nil {
		return nil, fmt.Errorf("get portfolio: %w", err)
	}

	if req.Name != nil {
		portfolio.Name = *req.Name
	}
	if req.Description != nil {
		portfolio.Description = *req.Description
	}
	if req.IsArchived != nil {
		portfolio.IsArchived = *req.IsArchived
	}
	if req.ExcludeFromOverview != nil {
		portfolio.ExcludeFromOverview = *req.ExcludeFromOverview
	}

	if err := s.portfolioRepo.WithTx(tx).UpdatePortfolio(ctx, &portfolio); err != nil {
		return nil, fmt.Errorf("failed to update portfolio: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	pfLog.InfoContext(ctx, "portfolio updated", "portfolioID", id)
	return &portfolio, nil
}

// DeletePortfolio deletes a portfolio and all associated portfolio-fund relationships.
func (s *PortfolioService) DeletePortfolio(ctx context.Context, id string) error {
	pfLog.DebugContext(ctx, "deleting portfolio", "portfolioID", id)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	_, err = s.portfolioRepo.WithTx(tx).GetPortfolioOnID(id)
	if err != nil {
		return fmt.Errorf("get portfolio: %w", err)
	}

	err = s.portfolioRepo.WithTx(tx).DeletePortfolio(ctx, id)
	if err != nil {
		return fmt.Errorf("delete portfolio: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	pfLog.InfoContext(ctx, "portfolio deleted", "portfolioID", id)
	return nil
}
