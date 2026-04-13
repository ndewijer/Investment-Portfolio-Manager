package model

import "time"

// FundPrice represents a historical price point for a fund.
type FundPrice struct {
	ID     string    `json:"id"`
	FundID string    `json:"fundId"`
	Date   time.Time `json:"date"`
	Price  float64   `json:"price"`
}

// FundPriceUpdateResponse represents the response for fund price update operations.
// It indicates whether the update operation added new prices to the database.
type FundPriceUpdateResponse struct {
	Status    string `json:"status"`    // "success" or "error"
	Message   string `json:"message"`   // Human-readable description of the result
	NewPrices bool   `json:"newPrices"` // true if new prices were added, false if prices already existed
}

// AllFundUpdateResponse represents the response for bulk fund price update operations.
// It provides detailed information about both successful and failed fund updates,
// allowing clients to identify which funds were updated and which encountered errors.
// Success is true if at least one fund was successfully updated.
type AllFundUpdateResponse struct {
	Success      bool               `json:"success"`      // true if at least one fund was successfully updated
	UpdatedFunds []UpdatedFund      `json:"updatedFunds"` // List of successfully updated funds with details
	Errors       []UpdatedFundError `json:"errors"`       // List of funds that failed to update with error messages
	TotalUpdated int                `json:"totalUpdated"` // Count of successfully updated funds
	TotalErrors  int                `json:"totalErrors"`  // Count of funds that failed to update
}

// UpdatedFund represents a successfully updated fund with details about the operation.
// It includes the fund identification and the number of new price records added.
type UpdatedFund struct {
	FundID      string `json:"fundId"`      // Unique identifier of the fund
	Name        string `json:"name"`        // Display name of the fund
	Symbol      string `json:"symbol"`      // Trading symbol of the fund
	PricesAdded int    `json:"pricesAdded"` // Number of new historical price records added
}

// UpdatedFundError represents a fund that failed to update with error details.
// It includes fund identification and the specific error message encountered.
type UpdatedFundError struct {
	FundID string `json:"fundId"` // Unique identifier of the fund
	Name   string `json:"name"`   // Display name of the fund
	Symbol string `json:"symbol"` // Trading symbol of the fund
	Error  string `json:"error"`  // Error message describing why the update failed
}

// ExchangeRate represents a currency exchange rate for a specific date.
type ExchangeRate struct {
	ID           string    `json:"id"`           // Unique identifier for the rate
	FromCurrency string    `json:"fromCurrency"` // Source currency code
	ToCurrency   string    `json:"toCurrency"`   // Target currency code
	Rate         float64   `json:"rate"`         // Exchange rate value
	Date         time.Time `json:"date"`         // Date the rate applies to
}
