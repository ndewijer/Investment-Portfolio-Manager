package request

// CreatePortfolioRequest represents the request body for creating a portfolio
type CreatePortfolioRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	ExcludeFromOverview bool   `json:"excludeFromOverview"`
}

// UpdatePortfolioRequest is the request body for updating an existing portfolio.
type UpdatePortfolioRequest struct {
	Name                *string `json:"name,omitempty"`
	Description         *string `json:"description,omitempty"`
	IsArchived          *bool   `json:"isArchived,omitempty"`
	ExcludeFromOverview *bool   `json:"excludeFromOverview,omitempty"`
}

// CreatePortfolioFundRequest is the request body for adding a fund to a portfolio.
type CreatePortfolioFundRequest struct {
	PortfolioID string `json:"portfolioId"`
	FundID      string `json:"fundId"`
}
