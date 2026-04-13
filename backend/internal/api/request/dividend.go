package request

// CreateDividendRequest represents the request body for creating a new dividend.
// All fields are required.
type CreateDividendRequest struct {
	PortfolioFundID    string  `json:"portfolioFundId"`
	RecordDate         string  `json:"recordDate"`
	ExDividendDate     string  `json:"exDividendDate"`
	DividendPerShare   float64 `json:"dividendPerShare"`
	BuyOrderDate       string  `json:"buyOrderDate,omitempty"`
	ReinvestmentShares float64 `json:"reinvestmentShares,omitempty"`
	ReinvestmentPrice  float64 `json:"reinvestmentPrice,omitempty"`
}

// UpdateDividendRequest represents the request body for updating an existing dividend.
// All fields are optional (use pointers). Only provided fields will be updated.
type UpdateDividendRequest struct {
	PortfolioFundID    *string  `json:"portfolioFundId,omitempty"`
	RecordDate         *string  `json:"recordDate,omitempty"`
	ExDividendDate     *string  `json:"exDividendDate,omitempty"`
	DividendPerShare   *float64 `json:"dividendPerShare,omitempty"`
	BuyOrderDate       *string  `json:"buyOrderDate,omitempty"`
	ReinvestmentShares *float64 `json:"reinvestmentShares,omitempty"`
	ReinvestmentPrice  *float64 `json:"reinvestmentPrice,omitempty"`
}
