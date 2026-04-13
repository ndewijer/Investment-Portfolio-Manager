package model

import "time"

// Dividend represents a dividend record from the database.
// Used internally for calculations and data processing.
type Dividend struct {
	ID                        string
	FundID                    string
	PortfolioFundID           string
	RecordDate                time.Time
	ExDividendDate            time.Time
	SharesOwned               float64
	DividendPerShare          float64
	TotalAmount               float64
	ReinvestmentStatus        string
	BuyOrderDate              time.Time
	ReinvestmentTransactionID string
	CreatedAt                 time.Time
}

// DividendFund represents a dividend payment with enriched fund information.
// This structure is used for API responses that combine dividend data with fund metadata.
// It includes all dividend payment details along with the associated fund name and dividend type.
type DividendFund struct {
	ID                        string     `json:"id"`                                  // Unique dividend record ID
	FundID                    string     `json:"fundId"`                              // Fund identifier
	FundName                  string     `json:"fundName"`                            // Name of the fund paying the dividend
	PortfolioFundID           string     `json:"portfolioFundId"`                     // Portfolio fund relationship ID
	RecordDate                time.Time  `json:"recordDate"`                          // Date of record for dividend eligibility
	ExDividendDate            time.Time  `json:"exDividendDate"`                      // Ex-dividend date
	SharesOwned               float64    `json:"sharesOwned"`                         // Number of shares owned on ex-dividend date
	DividendPerShare          float64    `json:"dividendPerShare"`                    // Dividend amount per share
	TotalAmount               float64    `json:"totalAmount"`                         // Total dividend amount (sharesOwned × dividendPerShare)
	ReinvestmentStatus        string     `json:"reinvestmentStatus"`                  // Status: "reinvested", "paid", etc.
	BuyOrderDate              *time.Time `json:"buyOrderDate,omitempty"`              // Date when reinvestment buy order was placed (nil if not reinvested)
	ReinvestmentTransactionID string     `json:"reinvestmentTransactionId,omitempty"` // Transaction ID if dividend was reinvested (empty if not)
	DividendType              string     `json:"dividendType"`                        // Type of dividend: "accumulating", "distributing"
}

// PendingDividend represents a dividend record awaiting processing or matching.
// Used for the IBKR dividend matching workflow to reconcile imported dividend transactions.
type PendingDividend struct {
	ID               string    `json:"id"`
	FundID           string    `json:"fundId"`
	PortfolioFundID  string    `json:"portfolioFundId"`
	RecordDate       time.Time `json:"recordDate"`
	ExDividendDate   time.Time `json:"exDividendDate"`
	SharesOwned      float64   `json:"sharesOwned"`
	DividendPerShare float64   `json:"dividendPerShare"`
	TotalAmount      float64   `json:"totalAmount"`
}
