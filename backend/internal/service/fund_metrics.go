package service

import (
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// FundMetrics represents calculated metrics for a single fund at a point in time.
// This structure is returned by calculateFundMetrics and contains all per-fund valuations.
type FundMetrics struct {
	PortfolioFundID string  // Portfolio fund unique identifier
	FundID          string  // Fund identifier for price lookup
	Shares          float64 // Total number of shares held (including reinvested dividends)
	Cost            float64 // Total cost basis (weighted average cost method)
	LatestPrice     float64 // Most recent price used for valuation
	Dividend        float64 // Total dividend amounts received (not reinvested)
	Value           float64 // Current market value (shares * latestPrice)
	UnrealizedGain  float64 // Unrealized gain/loss (value - cost)
	Fees            float64 // Total fees paid
}

// calculateFundMetrics calculates detailed metrics for a single fund as of a specific date.
// This is the core calculation engine used by both per-fund endpoints and portfolio aggregation.
//
// The calculation processes all transactions up to the specified date to compute:
//   - Total shares held (buy transactions increase, sell transactions decrease)
//   - Cost basis (weighted average cost, adjusted on sales)
//   - Market value (shares * price)
//   - Unrealized gain/loss (value - cost)
//   - Dividends received
//   - Fees paid
//
// Transaction Processing Logic:
//   - "buy": Increases shares and cost
//   - "sell": Decreases shares and adjusts cost basis proportionally
//   - "dividend": Adds to dividend total (reinvestment shares come via dividendShares parameter)
//   - "fee": Adds to both cost and fees
//
// Price Strategy:
// The useLatestPrice parameter controls price selection:
//   - true: Uses the most recent available price regardless of date (for current valuations)
//   - false: Uses the price on or before the target date (for historical calculations)
//
// Parameters:
//   - pfID: Portfolio fund ID for identification
//   - fundID: Fund ID for price lookup
//   - date: Target date for calculation (only transactions on or before this date are included)
//   - transactions: All transactions for this fund, sorted by date
//   - dividendShares: Shares acquired through dividend reinvestment
//   - fundPrices: Historical price data for the fund, sorted ascending
//   - useLatestPrice: If true, uses latest available price; if false, uses price as of date
//
// Returns:
// FundMetrics struct containing all calculated values including shares, cost, value, gains, dividends, and fees.
func (s *FundService) calculateFundMetrics(
	pfID string,
	fundID string,
	date time.Time,
	transactions []model.Transaction,
	dividendShares float64,
	fundPrices []model.FundPrice,
	useLatestPrice bool,
) (FundMetrics, error) {

	var shares, cost, dividends, value, fees float64
	shares = dividendShares

	for _, transaction := range transactions {

		if transaction.Date.Before(date) || transaction.Date.Equal(date) {

			switch transaction.Type {
			case "buy":
				shares += transaction.Shares
				cost += transaction.Shares * transaction.CostPerShare
			case "dividend":
				dividends += transaction.Shares * transaction.CostPerShare
			case "sell":
				shares -= transaction.Shares
				if shares > 0.0 {
					cost = (cost / (shares + transaction.Shares)) * shares
				} else {
					cost = 0.0
				}
			case "fee":
				cost += transaction.CostPerShare
				fees += transaction.CostPerShare
			default:
				return FundMetrics{}, fmt.Errorf("unsupported transaction type: %s", transaction.Type)
			}
		} else {
			break
		}
	}
	latestPrice := 0.0
	if len(fundPrices) > 0 {
		if useLatestPrice {
			latestPrice = s.getLatestPrice(fundPrices)
		} else {
			latestPrice = s.getPriceForDate(fundPrices, date)
		}
		if latestPrice > 0 {
			value = shares * latestPrice
		}
	}

	return FundMetrics{
		PortfolioFundID: pfID,
		FundID:          fundID,
		Shares:          shares,
		Cost:            cost,
		LatestPrice:     latestPrice,
		Dividend:        dividends,
		Value:           value,
		UnrealizedGain:  value - cost,
		Fees:            fees,
	}, nil
}

// getPriceForDate finds the most recent fund price on or before the target date.
// Assumes prices are sorted in ASC order (oldest first).
// Returns 0 if no price is found on or before the target date.
func (s *FundService) getPriceForDate(prices []model.FundPrice, targetDate time.Time) float64 {
	var latestPrice float64

	// Prices are sorted ASC, so iterate forward
	for _, price := range prices {
		if price.Date.Before(targetDate) || price.Date.Equal(targetDate) {
			latestPrice = price.Price // Keep updating with more recent prices
		} else {
			break // We've passed the target date, stop
		}
	}

	return latestPrice
}

// getLatestPrice returns the most recent price available regardless of date.
// Assumes prices are sorted in ASC order (oldest first).
// Returns 0 if the prices slice is empty.
func (s *FundService) getLatestPrice(prices []model.FundPrice) float64 {
	if len(prices) == 0 {
		return 0
	}
	return prices[len(prices)-1].Price
}
