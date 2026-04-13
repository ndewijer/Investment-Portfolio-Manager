package service

import (
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// enrichPortfolioFundsWithMetrics calculates and assigns current metrics to all portfolio funds.
// This method processes each fund in the PortfolioData to compute shares, costs, values,
// gains/losses, dividends, and fees as of today's date.
//
// This is the main orchestrator for enriching portfolio fund data with calculated metrics.
// It iterates through all funds and delegates the detailed calculations to
// calculateAndAssignFundMetrics for each fund.
//
// Parameters:
//   - data: Complete portfolio data including transactions, dividends, prices, etc.
//   - realizedGainsByPF: Map of realized gains keyed by portfolio fund ID
//
// Returns the input portfolio funds slice with all metric fields populated (TotalShares,
// CurrentValue, UnrealizedGainLoss, etc.). All monetary values are rounded to two decimals.
func (s *FundService) enrichPortfolioFundsWithMetrics(
	data *PortfolioData,
	realizedGainsByPF map[string][]model.RealizedGainLoss,
) ([]model.PortfolioFundResponse, error) {

	today := time.Now().UTC()
	portfolioFunds := data.PortfolioFunds

	for i := range portfolioFunds {
		fund := &portfolioFunds[i]

		err := s.calculateAndAssignFundMetrics(
			fund,
			today,
			data,
			realizedGainsByPF[fund.ID],
		)
		if err != nil {
			return nil, fmt.Errorf("calculate metrics for fund %s: %w", fund.ID, err)
		}
	}

	return portfolioFunds, nil
}

// calculateAndAssignFundMetrics computes all metrics for a single portfolio fund and assigns them in-place.
// This method performs the complete calculation pipeline for one fund:
//   - Processes dividend shares (accounting for reinvestments)
//   - Calculates fund metrics (shares, cost, value, gains)
//   - Aggregates dividend amounts
//   - Aggregates realized gains/losses
//   - Computes average cost per share
//   - Assigns all calculated values to the fund struct with rounding
//
// Parameters:
//   - fund: Pointer to the portfolio fund to enrich (modified in-place)
//   - date: The date to calculate metrics as of (typically today for current values)
//   - data: Complete portfolio data with transactions, dividends, prices
//   - realizedGains: Realized gains specific to this fund
//
// The fund parameter is modified in-place with populated metrics including:
// TotalShares, LatestPrice, AverageCost, TotalCost, CurrentValue, UnrealizedGainLoss,
// RealizedGainLoss, TotalGainLoss, TotalDividends, and TotalFees.
func (s *FundService) calculateAndAssignFundMetrics(
	fund *model.PortfolioFundResponse,
	date time.Time,
	data *PortfolioData,
	realizedGains []model.RealizedGainLoss,
) error {

	dividendSharesMap, err := s.dividendService.processDividendSharesForDate(
		data.DividendsByPF,
		data.TransactionsByPF[fund.ID],
		date,
	)
	if err != nil {
		return fmt.Errorf("process dividend shares: %w", err)
	}

	fundMetrics, err := s.calculateFundMetrics(
		fund.ID,
		fund.FundID,
		date,
		data.TransactionsByPF[fund.ID],
		dividendSharesMap[fund.ID],
		data.FundPricesByFund[fund.FundID],
		true, // Use latest price
	)
	if err != nil {
		return fmt.Errorf("calculate fund metrics: %w", err)
	}

	// Calculate dividend amount
	totalDividendAmount, err := s.dividendService.processDividendAmountForDate(
		data.DividendsByPF[fund.ID],
		date,
	)
	if err != nil {
		return fmt.Errorf("process dividend amount: %w", err)
	}

	totalRealizedGainLoss, _, _, err := s.realizedGainLossService.processRealizedGainLossForDate(
		realizedGains,
		date,
	)
	if err != nil {
		return fmt.Errorf("process realized gain loss: %w", err)
	}

	roundedShares := round(fundMetrics.Shares)
	averageCost := 0.0
	if roundedShares > 0 {
		averageCost = fundMetrics.Cost / roundedShares
	}

	fund.TotalShares = roundedShares
	fund.LatestPrice = round(fundMetrics.LatestPrice)
	fund.AverageCost = round(averageCost)
	fund.TotalCost = round(fundMetrics.Cost)
	fund.CurrentValue = round(fundMetrics.Value)
	fund.UnrealizedGainLoss = round(fundMetrics.UnrealizedGain)
	fund.RealizedGainLoss = round(totalRealizedGainLoss)
	fund.TotalGainLoss = round(fundMetrics.UnrealizedGain + totalRealizedGainLoss)
	fund.TotalDividends = round(totalDividendAmount)
	fund.TotalFees = round(fundMetrics.Fees)

	return nil
}
