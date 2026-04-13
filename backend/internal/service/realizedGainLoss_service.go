package service

import (
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
)

var rglLog = logging.NewLogger("transaction")

// RealizedGainLossService handles realized gain/loss calculations and record management.
type RealizedGainLossService struct {
	realizedgainlossRepo *repository.RealizedGainLossRepository
}

// NewRealizedGainLossService creates a new RealizedGainLossService with the provided repository dependencies.
func NewRealizedGainLossService(
	realizedgainlossRepo *repository.RealizedGainLossRepository,
) *RealizedGainLossService {
	return &RealizedGainLossService{
		realizedgainlossRepo: realizedgainlossRepo,
	}
}

// loadRealizedGainLoss retrieves realized gain/loss records for the given portfolios within the specified date range.
// Results are grouped by portfolio ID.
func (s *RealizedGainLossService) loadRealizedGainLoss(portfolio []string, startDate, endDate time.Time) (map[string][]model.RealizedGainLoss, error) {
	rglLog.Debug("loading realized gain/loss", "portfolios", len(portfolio), "startDate", startDate.Format("2006-01-02"), "endDate", endDate.Format("2006-01-02"))
	result, err := s.realizedgainlossRepo.GetRealizedGainLossByPortfolio(portfolio, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get realized gain/loss by portfolio: %w", err)
	}
	return result, nil
}

// processRealizedGainLossForDate calculates cumulative realized gains/losses as of the specified date.
// Only realized gains from sales that occurred on or before the target date are included.
// Returns (totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, error).
func (s *RealizedGainLossService) processRealizedGainLossForDate(realizedGainLoss []model.RealizedGainLoss, date time.Time) (float64, float64, float64, error) {
	if len(realizedGainLoss) == 0 {
		return 0.0, 0.0, 0.0, nil
	}
	var totalRealizedGainLoss, totalSaleProceeds, totalCostBasis float64

	for _, r := range realizedGainLoss {
		if r.TransactionDate.Before(date) || r.TransactionDate.Equal(date) {
			totalRealizedGainLoss += r.RealizedGainLoss
			totalSaleProceeds += r.SaleProceeds
			totalCostBasis += r.CostBasis
		}
		// No break: records are sorted by transaction_date but we iterate all to guard
		// against any insertion-order anomalies from imports or backfills.
	}

	return totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, nil
}
