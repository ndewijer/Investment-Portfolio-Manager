package model

import (
	"encoding/json"
	"time"
)

// Fund represents a fund from the database.
type Fund struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Isin           string  `json:"isin"`
	Symbol         string  `json:"symbol"`
	Currency       string  `json:"currency"`
	Exchange       string  `json:"exchange"`
	InvestmentType string  `json:"investmentType"`
	DividendType   string  `json:"dividendType"`
	LatestPrice    float64 `json:"latestPrice,omitempty"`
}

// PortfolioFund represents a portfolio_fund record from the database
type PortfolioFund struct {
	ID          string
	PortfolioID string
	FundID      string
}

// PortfolioFundResponse represents a fund held within a portfolio with calculated metrics.
// Used for detailed portfolio fund breakdowns with shares, values, and gains/losses.
type PortfolioFundResponse struct {
	ID                 string  `json:"id"`
	FundID             string  `json:"fundId"`
	FundName           string  `json:"fundName"`
	TotalShares        float64 `json:"totalShares"`
	LatestPrice        float64 `json:"latestPrice"`
	AverageCost        float64 `json:"averageCost"`
	TotalCost          float64 `json:"totalCost"`
	CurrentValue       float64 `json:"currentValue"`
	UnrealizedGainLoss float64 `json:"unrealizedGainLoss"`
	RealizedGainLoss   float64 `json:"realizedGainLoss"`
	TotalGainLoss      float64 `json:"totalGainLoss"`
	TotalDividends     float64 `json:"totalDividends"`
	TotalFees          float64 `json:"totalFees"`
	DividendType       string  `json:"dividendType"`
	InvestmentType     string  `json:"investmentType"`
}

// PortfolioFundListing represents a portfolio-fund relationship with basic metadata.
// Used for listing all portfolio-fund combinations across all portfolios.
type PortfolioFundListing struct {
	ID            string `json:"id"`            // Portfolio fund relationship ID
	PortfolioID   string `json:"portfolioId"`   // Portfolio identifier
	FundID        string `json:"fundId"`        // Fund identifier
	PortfolioName string `json:"portfolioName"` // Portfolio name
	FundName      string `json:"fundName"`      // Fund name
	DividendType  string `json:"dividendType"`  // Fund dividend type (stock/cash)
}

// FundHistoryEntry represents a single fund's metrics for a specific date.
// This structure maps to the fund_history_materialized table.
type FundHistoryEntry struct {
	ID              string    `json:"id"`              // Unique record identifier
	PortfolioFundID string    `json:"portfolioFundId"` // Portfolio fund relationship ID
	FundID          string    `json:"fundId"`          // Fund identifier
	FundName        string    `json:"fundName"`        // Fund name (from JOIN)
	Date            time.Time `json:"date"`            // Date of this snapshot
	Shares          float64   `json:"shares"`          // Total shares held
	Price           float64   `json:"price"`           // Price per share on this date
	Value           float64   `json:"value"`           // Market value (shares × price)
	Cost            float64   `json:"cost"`            // Cost basis
	RealizedGain    float64   `json:"realizedGain"`    // Realized gain/loss
	UnrealizedGain  float64   `json:"unrealizedGain"`  // Unrealized gain/loss
	TotalGainLoss   float64   `json:"totalGainLoss"`   // Total gain/loss (realized + unrealized)
	Dividends       float64   `json:"dividends"`       // Dividends received
	Fees            float64   `json:"fees"`            // Fees paid
	SaleProceeds    float64   `json:"saleProceeds"`    // Cumulative sale proceeds
	OriginalCost    float64   `json:"originalCost"`    // Cumulative original cost of sold positions
}

// FundHistoryResponse represents the JSON response for fund history endpoint.
// Returns time-series data with fund breakdowns per date.
type FundHistoryResponse struct {
	Date  time.Time          `json:"date"`  // Date of this snapshot
	Funds []FundHistoryEntry `json:"funds"` // All funds in portfolio on this date
}

// MarshalJSON formats Date as "2006-01-02" instead of the default RFC 3339.
func (r FundHistoryResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Date  string             `json:"date"`
		Funds []FundHistoryEntry `json:"funds"`
	}{
		Date:  r.Date.Format("2006-01-02"),
		Funds: r.Funds,
	})
}

// Symbol represents ticker symbol information from an external data source.
// Contains metadata about a financial instrument including exchange and currency.
type Symbol struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Name        string    `json:"name"`
	Exchange    string    `json:"exchange,omitempty"`
	Currency    string    `json:"currency,omitempty"`
	Isin        string    `json:"isin,omitempty"`
	LastUpdated time.Time `json:"lastUpdated"`
	DataSource  string    `json:"dataSource"`
	IsValid     bool      `json:"isValid"`
}

// FundUsage describes whether a fund is in use and which portfolios hold it.
type FundUsage struct {
	InUsage    bool                   `json:"inUse"`
	Portfolios []PortfolioTransaction `json:"portfolios,omitempty"`
}

// PortfolioTransaction summarises the transaction count for a portfolio that holds a fund.
type PortfolioTransaction struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	TransactionCount int    `json:"transactionCount"`
}
