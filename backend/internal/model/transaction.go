package model

import "time"

// TransactionType represents an allowed transaction type value.
type TransactionType string

// Transaction type constants define the valid values for the Type field of a [Transaction].
const (
	TransactionTypeBuy      TransactionType = "buy"
	TransactionTypeSell     TransactionType = "sell"
	TransactionTypeDividend TransactionType = "dividend"
	TransactionTypeFee      TransactionType = "fee"
)

// ValidTransactionTypes is the authoritative set of allowed transaction type values.
// Use this for validation anywhere transaction types are accepted as input.
var ValidTransactionTypes = map[TransactionType]bool{
	TransactionTypeBuy:      true,
	TransactionTypeSell:     true,
	TransactionTypeDividend: true,
	TransactionTypeFee:      true,
}

// Transaction represents a buy or sell transaction for a portfolio fund.
// Used internally for calculations and data processing.
type Transaction struct {
	ID              string    `json:"id"`
	PortfolioFundID string    `json:"portfolioFundId"`
	Date            time.Time `json:"date"`
	Type            string    `json:"type"`
	Shares          float64   `json:"shares"`
	CostPerShare    float64   `json:"costPerShare"`
	CreatedAt       time.Time `json:"createdAt"`
}

// TransactionResponse represents a transaction with enriched data for API responses.
// Includes fund name and IBKR linkage information.
type TransactionResponse struct {
	ID                string    `json:"id"`
	PortfolioFundID   string    `json:"portfolioFundId"`
	FundName          string    `json:"fundName"`
	Date              time.Time `json:"date"`
	Type              string    `json:"type"`
	Shares            float64   `json:"shares"`
	CostPerShare      float64   `json:"costPerShare"`
	IbkrTransactionID string    `json:"ibkrTransactionId,omitempty"`
	IbkrLinked        bool      `json:"ibkrLinked"`
}
