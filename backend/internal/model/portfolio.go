package model

import "time"

// Portfolio represents a portfolio from the database.
type Portfolio struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	IsArchived          bool   `json:"isArchived"`
	ExcludeFromOverview bool   `json:"excludeFromOverview"`
}

// PortfolioFilter holds filter options for querying portfolios.
type PortfolioFilter struct {
	IncludeArchived bool
	IncludeExcluded bool
}

// PortfolioSummary represents the current state of a portfolio at a specific point in time.
// It includes valuation, cost basis, gains/losses (both realized and unrealized),
// dividends, and sale information. All monetary values are rounded to two decimal places.
type PortfolioSummary struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Description             string  `json:"description"`
	TotalValue              float64 `json:"totalValue"`              // Current market value
	TotalCost               float64 `json:"totalCost"`               // Current cost basis
	TotalDividends          float64 `json:"totalDividends"`          // Cumulative dividends
	TotalUnrealizedGainLoss float64 `json:"totalUnrealizedGainLoss"` // Unrealized gain/loss
	TotalRealizedGainLoss   float64 `json:"totalRealizedGainLoss"`   // Realized gain/loss from sales
	TotalSaleProceeds       float64 `json:"totalSaleProceeds"`       // Total proceeds from sales
	TotalOriginalCost       float64 `json:"totalOriginalCost"`       // Original cost of sold positions
	TotalGainLoss           float64 `json:"totalGainLoss"`           // Combined realized + unrealized
	IsArchived              bool    `json:"isArchived"`
}

// PortfolioHistory represents portfolio valuations for a single date.
// It contains one entry per portfolio showing their state on that specific date.
type PortfolioHistory struct {
	Date       string             `json:"date"`       // Date in YYYY-MM-DD format
	Portfolios []PortfolioSummary `json:"portfolios"` // Portfolio states for this date
}

// PortfolioHistoryMaterialized represents a pre-calculated portfolio state for a specific date.
// This is used for fast retrieval of historical portfolio data from the materialized view table.
type PortfolioHistoryMaterialized struct {
	ID                string    // Primary key
	PortfolioID       string    // Portfolio identifier
	Date              time.Time // Date of this snapshot
	Value             float64   // Market value on this date
	Cost              float64   // Cost basis on this date
	RealizedGain      float64   // Realized gains/losses as of this date
	UnrealizedGain    float64   // Unrealized gains/losses on this date
	TotalDividends    float64   // Cumulative dividends as of this date
	TotalSaleProceeds float64   // Total proceeds from sales
	TotalOriginalCost float64   // Original cost of sold positions
	TotalGainLoss     float64   // Combined realized + unrealized gain/loss
	IsArchived        bool      // Whether portfolio is archived
	CalculatedAt      time.Time // When this record was calculated
}
