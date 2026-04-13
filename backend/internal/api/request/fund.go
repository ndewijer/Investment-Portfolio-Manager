package request

// CreateFundRequest is the request body for creating a new fund.
type CreateFundRequest struct {
	Name           string `json:"name"`
	Isin           string `json:"isin"`
	Symbol         string `json:"symbol"`
	Currency       string `json:"currency"`
	Exchange       string `json:"exchange"`
	DividendType   string `json:"dividendType"`
	InvestmentType string `json:"investmentType"`
}

// UpdateFundRequest is the request body for updating an existing fund.
type UpdateFundRequest struct {
	Name           *string `json:"name,omitempty"`
	Isin           *string `json:"isin,omitempty"`
	Symbol         *string `json:"symbol,omitempty"`
	Currency       *string `json:"currency,omitempty"`
	Exchange       *string `json:"exchange,omitempty"`
	DividendType   *string `json:"dividendType,omitempty"`
	InvestmentType *string `json:"investmentType,omitempty"`
}
