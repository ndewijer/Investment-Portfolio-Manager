package request

// UpdateIbkrConfigRequest is the request body for creating or updating the IBKR configuration.
type UpdateIbkrConfigRequest struct {
	Enabled                  *bool        `json:"enabled"`
	FlexToken                *string      `json:"flexToken"`
	FlexQueryID              *string      `json:"flexQueryId"`
	TokenExpiresAt           *string      `json:"tokenExpiresAt,omitempty"`
	AutoImportEnabled        *bool        `json:"autoImportEnabled"`
	DefaultAllocationEnabled *bool        `json:"defaultAllocationEnabled"`
	DefaultAllocations       []Allocation `json:"defaultAllocations,omitempty"`
}

// Allocation represents a percentage allocation of an IBKR transaction to a portfolio-fund pair.
type Allocation struct {
	PortfolioID *string  `json:"portfolioId"`
	Percentage  *float64 `json:"percentage"`
}

// TestIbkrConnectionRequest holds the credentials for a one-off IBKR connection test.
// The token is supplied in plaintext here because this endpoint is used to verify
// credentials before they are saved to the encrypted config.
type TestIbkrConnectionRequest struct {
	FlexQueryID string `json:"flexQueryId"`
	FlexToken   string `json:"flexToken"`
}

// AllocationEntry represents a single portfolio allocation with its percentage.
type AllocationEntry struct {
	PortfolioID string  `json:"portfolioId"`
	Percentage  float64 `json:"percentage"`
}

// AllocateTransactionRequest holds the allocation details for a single IBKR transaction.
type AllocateTransactionRequest struct {
	Allocations []AllocationEntry `json:"allocations"`
}

// BulkAllocateRequest holds the transaction IDs and allocation details for bulk allocation.
type BulkAllocateRequest struct {
	TransactionIDs []string          `json:"transactionIds"`
	Allocations    []AllocationEntry `json:"allocations"`
}

// ModifyAllocationsRequest holds updated allocation details for a processed IBKR transaction.
type ModifyAllocationsRequest struct {
	Allocations []AllocationEntry `json:"allocations"`
}

// MatchDividendRequest holds the dividend IDs to match against an IBKR transaction.
type MatchDividendRequest struct {
	DividendIDs []string `json:"dividendIds"`
}
