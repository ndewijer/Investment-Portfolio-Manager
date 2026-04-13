package request

// CreateTransactionRequest represents the request body for creating a new transaction.
// All fields are required.
type CreateTransactionRequest struct {
	PortfolioFundID string  `json:"portfolioFundId"`
	Date            string  `json:"date"`
	Type            string  `json:"type"`
	Shares          float64 `json:"shares"`
	CostPerShare    float64 `json:"costPerShare"`
}

// UpdateTransactionRequest represents the request body for updating an existing transaction.
// All fields are optional (use pointers). Only provided fields will be updated.
type UpdateTransactionRequest struct {
	PortfolioFundID *string  `json:"portfolioFundId,omitempty"`
	Date            *string  `json:"date,omitempty"`
	Type            *string  `json:"type,omitempty"`
	Shares          *float64 `json:"shares,omitempty"`
	CostPerShare    *float64 `json:"costPerShare,omitempty"`
}
