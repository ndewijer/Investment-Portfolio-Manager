package request

// SetExchangeRateRequest is the request body for creating or updating an exchange rate.
type SetExchangeRateRequest struct {
	Date         string `json:"date"`         // Date is the exchange rate date in YYYY-MM-DD format.
	FromCurrency string `json:"fromCurrency"` // FromCurrency is the source currency code (e.g. "USD").
	ToCurrency   string `json:"toCurrency"`   // ToCurrency is the target currency code (e.g. "EUR").
	Rate         string `json:"rate"`         // Rate is the exchange rate as a decimal string.
}

// SetFundPriceRequest is the request body for creating or updating a fund price.
type SetFundPriceRequest struct {
	Date   string `json:"date"`   // Date is the price date in YYYY-MM-DD format.
	FundID string `json:"fundId"` // FundID is the UUID of the fund.
	Price  string `json:"price"`  // Price is the fund price as a decimal string.
}

// SetLoggingConfig is the request body for updating the logging configuration.
type SetLoggingConfig struct {
	Enabled *bool  `json:"enabled"` // Enabled controls whether logging is active. Required.
	Level   string `json:"level"`   // Level is the minimum log level. Must be one of: debug, info, warning, error, critical.
}
