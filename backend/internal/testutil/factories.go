package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// PortfolioBuilder provides a fluent interface for creating test portfolios.
//
// Example usage:
//
//	// Simple creation with defaults
//	portfolio := testutil.NewPortfolio().Build(t, db)
//
//	// Customized portfolio
//	portfolio := testutil.NewPortfolio().
//	    WithName("Custom Portfolio").
//	    WithDescription("My description").
//	    Archived().
//	    Build(t, db)
type PortfolioBuilder struct {
	ID                  string
	Name                string
	Description         string
	IsArchived          bool
	ExcludeFromOverview bool
}

// NewPortfolio creates a PortfolioBuilder with sensible defaults.
func NewPortfolio() *PortfolioBuilder {
	return &PortfolioBuilder{
		ID:                  MakeID(),
		Name:                MakePortfolioName("Test Portfolio"),
		Description:         "Test description",
		IsArchived:          false,
		ExcludeFromOverview: false,
	}
}

// WithID sets a custom ID.
func (b *PortfolioBuilder) WithID(id string) *PortfolioBuilder {
	b.ID = id
	return b
}

// WithName sets a custom name.
func (b *PortfolioBuilder) WithName(name string) *PortfolioBuilder {
	b.Name = name
	return b
}

// WithDescription sets a custom description.
func (b *PortfolioBuilder) WithDescription(desc string) *PortfolioBuilder {
	b.Description = desc
	return b
}

// WithIsArchived sets the archived flag for the portfolio.
func (b *PortfolioBuilder) WithIsArchived(archived bool) *PortfolioBuilder {
	b.IsArchived = archived
	return b
}

// WithExcludeFromOverview sets the exclude-from-overview flag for the portfolio.
func (b *PortfolioBuilder) WithExcludeFromOverview(exclude bool) *PortfolioBuilder {
	b.IsArchived = exclude
	return b
}

// Archived marks the portfolio as archived.
func (b *PortfolioBuilder) Archived() *PortfolioBuilder {
	b.IsArchived = true
	return b
}

// ExcludedFromOverview marks the portfolio as excluded from overview.
func (b *PortfolioBuilder) ExcludedFromOverview() *PortfolioBuilder {
	b.ExcludeFromOverview = true
	return b
}

// Build creates the portfolio in the database and returns it.
func (b *PortfolioBuilder) Build(t *testing.T, db *sql.DB) model.Portfolio {
	t.Helper()

	query := `
		INSERT INTO portfolio (id, name, description, is_archived, exclude_from_overview)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, b.ID, b.Name, b.Description, b.IsArchived, b.ExcludeFromOverview)
	if err != nil {
		t.Fatalf("Failed to create test portfolio: %v", err)
	}

	return model.Portfolio{
		ID:                  b.ID,
		Name:                b.Name,
		Description:         b.Description,
		IsArchived:          b.IsArchived,
		ExcludeFromOverview: b.ExcludeFromOverview,
	}
}

// Convenience functions

// CreatePortfolio creates a portfolio with the given name and default values.
//
// Example usage:
//
//	portfolio := testutil.CreatePortfolio(t, db, "My Portfolio")
func CreatePortfolio(t *testing.T, db *sql.DB, name string) model.Portfolio {
	t.Helper()
	return NewPortfolio().WithName(name).Build(t, db)
}

// CreatePortfolios creates multiple portfolios with unique names.
//
// Example usage:
//
//	portfolios := testutil.CreatePortfolios(t, db, 5)
//	// Creates 5 portfolios with auto-generated names
func CreatePortfolios(t *testing.T, db *sql.DB, count int) []model.Portfolio {
	t.Helper()

	portfolios := make([]model.Portfolio, count)
	for i := range count {
		portfolios[i] = NewPortfolio().Build(t, db)
	}
	return portfolios
}

// CreateArchivedPortfolio creates an archived portfolio.
//
// Example usage:
//
//	portfolio := testutil.CreateArchivedPortfolio(t, db, "Old Portfolio")
func CreateArchivedPortfolio(t *testing.T, db *sql.DB, name string) model.Portfolio {
	t.Helper()
	return NewPortfolio().WithName(name).Archived().Build(t, db)
}

// CreateExcludedPortfolio creates an portfolio excluded by overview.
//
// Example usage:
//
//	portfolio := testutil.CreateExcludedPortfolio(t, db, "Old Portfolio")
func CreateExcludedPortfolio(t *testing.T, db *sql.DB, name string) model.Portfolio {
	t.Helper()
	return NewPortfolio().WithName(name).ExcludedFromOverview().Build(t, db)
}

// FundBuilder provides a fluent interface for creating test funds.
//
// Example usage:
//
//	fund := testutil.NewFund().
//	    WithSymbol("AAPL").
//	    WithCurrency("USD").
//	    Build(t, db)
type FundBuilder struct {
	ID             string
	Name           string
	ISIN           string
	Symbol         string
	Currency       string
	Exchange       string
	InvestmentType string
	DividendType   string
}

// NewFund creates a FundBuilder with sensible defaults.
func NewFund() *FundBuilder {
	return &FundBuilder{
		ID:             MakeID(),
		Name:           MakeFundName("Test Fund"),
		ISIN:           MakeISIN("US"),
		Symbol:         MakeSymbol("TEST"),
		Currency:       "USD",
		Exchange:       "NASDAQ",
		InvestmentType: "STOCK",
		DividendType:   "NONE",
	}
}

// WithName sets a custom name.
func (b *FundBuilder) WithName(name string) *FundBuilder {
	b.Name = name
	return b
}

// WithISIN sets a custom ISIN.
func (b *FundBuilder) WithISIN(isin string) *FundBuilder {
	b.ISIN = isin
	return b
}

// WithSymbol sets a custom symbol.
func (b *FundBuilder) WithSymbol(symbol string) *FundBuilder {
	b.Symbol = symbol
	return b
}

// WithCurrency sets the currency.
func (b *FundBuilder) WithCurrency(currency string) *FundBuilder {
	b.Currency = currency
	return b
}

// WithExchange sets the exchange.
func (b *FundBuilder) WithExchange(exchange string) *FundBuilder {
	b.Exchange = exchange
	return b
}

// WithInvestmentType sets the investment type.
func (b *FundBuilder) WithInvestmentType(investmentType string) *FundBuilder {
	b.InvestmentType = investmentType
	return b
}

// WithDividendType sets the dividend type.
func (b *FundBuilder) WithDividendType(dividendType string) *FundBuilder {
	b.DividendType = dividendType
	return b
}

// Build creates the fund in the database and returns it.
func (b *FundBuilder) Build(t *testing.T, db *sql.DB) model.Fund {
	t.Helper()

	query := `
		INSERT INTO fund (id, name, isin, symbol, currency, exchange, investment_type, dividend_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, b.ID, b.Name, b.ISIN, b.Symbol, b.Currency, b.Exchange, b.InvestmentType, b.DividendType)
	if err != nil {
		t.Fatalf("Failed to create test fund: %v", err)
	}

	return model.Fund{
		ID:             b.ID,
		Name:           b.Name,
		Isin:           b.ISIN,
		Symbol:         b.Symbol,
		Currency:       b.Currency,
		Exchange:       b.Exchange,
		InvestmentType: b.InvestmentType,
		DividendType:   b.DividendType,
	}
}

// CreateFund creates a fund with the given symbol and default values.
func CreateFund(t *testing.T, db *sql.DB, symbol string) model.Fund {
	t.Helper()
	return NewFund().WithSymbol(symbol).Build(t, db)
}

// CreateFunds creates multiple funds with unique symbols.
func CreateFunds(t *testing.T, db *sql.DB, count int) []model.Fund {
	t.Helper()

	funds := make([]model.Fund, count)
	for i := range count {
		funds[i] = NewFund().Build(t, db)
	}
	return funds
}

// PortfolioFundBuilder provides a fluent interface for creating portfolio-fund relationships
type PortfolioFundBuilder struct {
	ID          string
	PortfolioID string
	FundID      string
}

// NewPortfolioFund creates a PortfolioFundBuilder
func NewPortfolioFund(portfolioID, fundID string) *PortfolioFundBuilder {
	return &PortfolioFundBuilder{
		ID:          MakeID(),
		PortfolioID: portfolioID,
		FundID:      fundID,
	}
}

// Build creates the portfolio_fund in the database
func (b *PortfolioFundBuilder) Build(t *testing.T, db *sql.DB) model.PortfolioFund {
	t.Helper()

	query := `
		INSERT INTO portfolio_fund (id, portfolio_id, fund_id)
		VALUES (?, ?, ?)
	`

	_, err := db.Exec(query, b.ID, b.PortfolioID, b.FundID)
	if err != nil {
		t.Fatalf("Failed to create portfolio_fund: %v", err)
	}

	return model.PortfolioFund{
		ID:          b.ID,
		PortfolioID: b.PortfolioID,
		FundID:      b.FundID,
	}
}

// TransactionBuilder provides a fluent interface for creating transactions
type TransactionBuilder struct {
	ID              string
	PortfolioFundID string
	Date            time.Time
	Type            string
	Shares          float64
	CostPerShare    float64
}

// NewTransaction creates a TransactionBuilder with defaults
func NewTransaction(portfolioFundID string) *TransactionBuilder {
	return &TransactionBuilder{
		ID:              MakeID(),
		PortfolioFundID: portfolioFundID,
		Date:            time.Now().UTC(),
		Type:            "buy",
		Shares:          100.0,
		CostPerShare:    10.0,
	}
}

// WithID sets a custom ID
func (b *TransactionBuilder) WithID(id string) *TransactionBuilder {
	b.ID = id
	return b
}

// WithDate sets the transaction date
func (b *TransactionBuilder) WithDate(date time.Time) *TransactionBuilder {
	b.Date = date
	return b
}

// WithType sets the transaction type
func (b *TransactionBuilder) WithType(txType string) *TransactionBuilder {
	b.Type = txType
	return b
}

// WithShares sets the number of shares
func (b *TransactionBuilder) WithShares(shares float64) *TransactionBuilder {
	b.Shares = shares
	return b
}

// WithCostPerShare sets the cost per share
func (b *TransactionBuilder) WithCostPerShare(cost float64) *TransactionBuilder {
	b.CostPerShare = cost
	return b
}

// Build creates the transaction in the database
func (b *TransactionBuilder) Build(t *testing.T, db *sql.DB) model.Transaction {
	t.Helper()

	query := `
		INSERT INTO "transaction" (id, portfolio_fund_id, date, type, shares, cost_per_share)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, b.ID, b.PortfolioFundID, b.Date.Format("2006-01-02"), b.Type, b.Shares, b.CostPerShare)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	return model.Transaction{
		ID:              b.ID,
		PortfolioFundID: b.PortfolioFundID,
		Date:            b.Date,
		Type:            b.Type,
		Shares:          b.Shares,
		CostPerShare:    b.CostPerShare,
		CreatedAt:       time.Now().UTC(),
	}
}

// FundPriceBuilder provides a fluent interface for creating fund prices
type FundPriceBuilder struct {
	ID     string
	FundID string
	Date   time.Time
	Price  float64
}

// NewFundPrice creates a FundPriceBuilder
func NewFundPrice(fundID string) *FundPriceBuilder {
	return &FundPriceBuilder{
		ID:     MakeID(),
		FundID: fundID,
		Date:   time.Now().UTC(),
		Price:  12.0,
	}
}

// WithDate sets the price date
func (b *FundPriceBuilder) WithDate(date time.Time) *FundPriceBuilder {
	b.Date = date
	return b
}

// WithPrice sets the price
func (b *FundPriceBuilder) WithPrice(price float64) *FundPriceBuilder {
	b.Price = price
	return b
}

// Build creates the fund price in the database
func (b *FundPriceBuilder) Build(t *testing.T, db *sql.DB) model.FundPrice {
	t.Helper()

	query := `
		INSERT INTO fund_price (id, fund_id, date, price)
		VALUES (?, ?, ?, ?)
	`

	_, err := db.Exec(query, b.ID, b.FundID, b.Date.Format("2006-01-02"), b.Price)
	if err != nil {
		t.Fatalf("Failed to create fund price: %v", err)
	}

	return model.FundPrice{
		ID:     b.ID,
		FundID: b.FundID,
		Date:   b.Date,
		Price:  b.Price,
	}
}

// DividendBuilder provides a fluent interface for creating dividends
type DividendBuilder struct {
	ID                        string
	FundID                    string
	PortfolioFundID           string
	RecordDate                time.Time
	ExDividendDate            time.Time
	SharesOwned               float64
	DividendPerShare          float64
	TotalAmount               float64
	ReinvestmentStatus        string
	BuyOrderDate              *time.Time
	ReinvestmentTransactionID string
}

// NewDividend creates a DividendBuilder
func NewDividend(fundID, portfolioFundID string) *DividendBuilder {
	return &DividendBuilder{
		ID:                 MakeID(),
		FundID:             fundID,
		PortfolioFundID:    portfolioFundID,
		RecordDate:         time.Now().UTC().AddDate(0, 0, -10),
		ExDividendDate:     time.Now().UTC().AddDate(0, 0, -5),
		SharesOwned:        100.0,
		DividendPerShare:   0.50,
		TotalAmount:        50.0,
		ReinvestmentStatus: "pending",
	}
}

// WithExDividendDate sets the ex-dividend date.
func (b *DividendBuilder) WithExDividendDate(date time.Time) *DividendBuilder {
	b.ExDividendDate = date
	return b
}

// WithRecordDate sets the record date.
func (b *DividendBuilder) WithRecordDate(date time.Time) *DividendBuilder {
	b.RecordDate = date
	return b
}

// WithReinvestmentTransaction sets the reinvestment transaction ID
func (b *DividendBuilder) WithReinvestmentTransaction(txID string) *DividendBuilder {
	b.ReinvestmentTransactionID = txID
	b.ReinvestmentStatus = "completed"
	return b
}

// WithSharesOwned sets shares owned
func (b *DividendBuilder) WithSharesOwned(shares float64) *DividendBuilder {
	b.SharesOwned = shares
	return b
}

// WithDividendPerShare sets dividend per share
func (b *DividendBuilder) WithDividendPerShare(amount float64) *DividendBuilder {
	b.DividendPerShare = amount
	b.TotalAmount = amount * b.SharesOwned
	return b
}

// Build creates the dividend in the database
func (b *DividendBuilder) Build(t *testing.T, db *sql.DB) model.Dividend {
	t.Helper()

	query := `
		INSERT INTO dividend (id, fund_id, portfolio_fund_id, record_date, ex_dividend_date,
		                     shares_owned, dividend_per_share, total_amount, reinvestment_status,
		                     buy_order_date, reinvestment_transaction_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var buyOrderDate any
	if b.BuyOrderDate != nil {
		buyOrderDate = b.BuyOrderDate.Format("2006-01-02")
	}

	var reinvTxID any
	if b.ReinvestmentTransactionID != "" {
		reinvTxID = b.ReinvestmentTransactionID
	}

	_, err := db.Exec(query,
		b.ID, b.FundID, b.PortfolioFundID,
		b.RecordDate.Format("2006-01-02"),
		b.ExDividendDate.Format("2006-01-02"),
		b.SharesOwned, b.DividendPerShare, b.TotalAmount,
		b.ReinvestmentStatus, buyOrderDate, reinvTxID)

	if err != nil {
		t.Fatalf("Failed to create dividend: %v", err)
	}

	return model.Dividend{
		ID:                        b.ID,
		FundID:                    b.FundID,
		PortfolioFundID:           b.PortfolioFundID,
		RecordDate:                b.RecordDate,
		ExDividendDate:            b.ExDividendDate,
		SharesOwned:               b.SharesOwned,
		DividendPerShare:          b.DividendPerShare,
		TotalAmount:               b.TotalAmount,
		ReinvestmentStatus:        b.ReinvestmentStatus,
		ReinvestmentTransactionID: b.ReinvestmentTransactionID,
		CreatedAt:                 time.Now().UTC(),
	}
}

// RealizedGainLossBuilder provides a fluent interface for creating realized gain/loss records
type RealizedGainLossBuilder struct {
	ID               string
	PortfolioID      string
	FundID           string
	TransactionID    string
	TransactionDate  time.Time
	SharesSold       float64
	CostBasis        float64
	SaleProceeds     float64
	RealizedGainLoss float64
}

// NewRealizedGainLoss creates a RealizedGainLossBuilder
func NewRealizedGainLoss(portfolioID, fundID, transactionID string) *RealizedGainLossBuilder {
	return &RealizedGainLossBuilder{
		ID:               MakeID(),
		PortfolioID:      portfolioID,
		FundID:           fundID,
		TransactionID:    transactionID,
		TransactionDate:  time.Now().UTC(),
		SharesSold:       30.0,
		CostBasis:        300.0,
		SaleProceeds:     450.0,
		RealizedGainLoss: 150.0,
	}
}

// WithShares sets the shares sold
func (b *RealizedGainLossBuilder) WithShares(shares float64) *RealizedGainLossBuilder {
	b.SharesSold = shares
	return b
}

// WithCostBasis sets the cost basis
func (b *RealizedGainLossBuilder) WithCostBasis(cost float64) *RealizedGainLossBuilder {
	b.CostBasis = cost
	return b
}

// WithSaleProceeds sets the sale proceeds
func (b *RealizedGainLossBuilder) WithSaleProceeds(proceeds float64) *RealizedGainLossBuilder {
	b.SaleProceeds = proceeds
	b.RealizedGainLoss = proceeds - b.CostBasis
	return b
}

// WithDate sets the transaction date
func (b *RealizedGainLossBuilder) WithDate(date time.Time) *RealizedGainLossBuilder {
	b.TransactionDate = date
	return b
}

// Build creates the realized gain/loss in the database
func (b *RealizedGainLossBuilder) Build(t *testing.T, db *sql.DB) model.RealizedGainLoss {
	t.Helper()

	query := `
		INSERT INTO realized_gain_loss (id, portfolio_id, fund_id, transaction_id, transaction_date,
		                                shares_sold, cost_basis, sale_proceeds, realized_gain_loss)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		b.ID, b.PortfolioID, b.FundID, b.TransactionID,
		b.TransactionDate.Format("2006-01-02"),
		b.SharesSold, b.CostBasis, b.SaleProceeds, b.RealizedGainLoss)

	if err != nil {
		t.Fatalf("Failed to create realized gain/loss: %v", err)
	}

	return model.RealizedGainLoss{
		ID:               b.ID,
		PortfolioID:      b.PortfolioID,
		FundID:           b.FundID,
		TransactionID:    b.TransactionID,
		TransactionDate:  b.TransactionDate,
		SharesSold:       b.SharesSold,
		CostBasis:        b.CostBasis,
		SaleProceeds:     b.SaleProceeds,
		RealizedGainLoss: b.RealizedGainLoss,
		CreatedAt:        time.Now().UTC(),
	}
}

// SymbolInfoBuilder provides a fluent interface for creating test Symbols.
//
// Example usage:
//
//	Symbol := testutil.NewSymbol().
//	    WithSymbol("AAPL").
//	    WithName("Apple").
//	    Build(t, db)
type SymbolInfoBuilder struct {
	ID       string
	Symbol   string
	Name     string
	Exchange string
	Currency string
	Isin     string
}

// NewSymbol creates a SymbolInfoBuilder with sensible defaults.
func NewSymbol() *SymbolInfoBuilder {
	return &SymbolInfoBuilder{
		ID:       MakeID(),
		Symbol:   MakeSymbol("TEST"),
		Name:     MakeSymbolName("Test Symbol"),
		Exchange: "NASDAQ",
		Currency: "USD",
		Isin:     MakeISIN("US"),
	}
}

// WithName sets a custom name.
func (b *SymbolInfoBuilder) WithName(name string) *SymbolInfoBuilder {
	b.Name = name
	return b
}

// WithISIN sets a custom ISIN.
func (b *SymbolInfoBuilder) WithISIN(isin string) *SymbolInfoBuilder {
	b.Isin = isin
	return b
}

// WithSymbol sets a custom symbol.
func (b *SymbolInfoBuilder) WithSymbol(symbol string) *SymbolInfoBuilder {
	b.Symbol = symbol
	return b
}

// WithCurrency sets the currency.
func (b *SymbolInfoBuilder) WithCurrency(currency string) *SymbolInfoBuilder {
	b.Currency = currency
	return b
}

// WithExchange sets the exchange.
func (b *SymbolInfoBuilder) WithExchange(exchange string) *SymbolInfoBuilder {
	b.Exchange = exchange
	return b
}

// Build creates the Symbol in the database and returns it.
func (b *SymbolInfoBuilder) Build(t *testing.T, db *sql.DB) model.Symbol {
	t.Helper()

	query := `
		INSERT INTO symbol_info (id, symbol, name, exchange, currency, isin)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, b.ID, b.Symbol, b.Name, b.Exchange, b.Currency, b.Isin)
	if err != nil {
		t.Fatalf("Failed to create test Symbol: %v", err)
	}

	return model.Symbol{
		ID:       b.ID,
		Symbol:   b.Symbol,
		Name:     b.Name,
		Exchange: b.Exchange,
		Currency: b.Currency,
		Isin:     b.Isin,
	}
}

// CreateSymbol creates a Symbol with the given symbol and default values.
func CreateSymbol(t *testing.T, db *sql.DB, symbol string) model.Symbol {
	t.Helper()
	return NewSymbol().WithSymbol(symbol).Build(t, db)
}

// SystemSettingBuilder helps create test system settings.
type SystemSettingBuilder struct {
	ID        string
	Key       string
	Value     string
	UpdatedAt *time.Time
}

// NewSystemSetting creates a SystemSettingBuilder with sensible defaults.
func NewSystemSetting(key, value string) *SystemSettingBuilder {
	now := time.Now().UTC()
	return &SystemSettingBuilder{
		ID:        MakeID(),
		Key:       key,
		Value:     value,
		UpdatedAt: &now,
	}
}

// WithValue sets a custom value.
func (b *SystemSettingBuilder) WithValue(value string) *SystemSettingBuilder {
	b.Value = value
	return b
}

// WithUpdatedAt sets the updated_at timestamp.
func (b *SystemSettingBuilder) WithUpdatedAt(t time.Time) *SystemSettingBuilder {
	b.UpdatedAt = &t
	return b
}

// Build creates the system setting in the database and returns it.
func (b *SystemSettingBuilder) Build(t *testing.T, db *sql.DB) model.SystemSetting {
	t.Helper()

	query := `
        INSERT INTO system_setting (id, key, value, updated_at)
        VALUES (?, ?, ?, ?)
    `

	_, err := db.Exec(query, b.ID, b.Key, b.Value, b.UpdatedAt)
	if err != nil {
		t.Fatalf("Failed to create test system setting: %v", err)
	}

	return model.SystemSetting{
		ID:        b.ID,
		Key:       b.Key,
		Value:     b.Value,
		UpdatedAt: b.UpdatedAt,
	}
}

// Convenience functions for specific settings

// NewLoggingEnabled creates a builder for the LOGGING_ENABLED setting.
func NewLoggingEnabled(enabled bool) *SystemSettingBuilder {
	value := "false"
	if enabled {
		value = "true"
	}
	return NewSystemSetting("LOGGING_ENABLED", value)
}

// NewLoggingLevel creates a builder for the LOGGING_LEVEL setting.
// Valid levels: debug, info, warning, error, critical
func NewLoggingLevel(level string) *SystemSettingBuilder {
	validLevels := map[string]bool{
		"debug":    true,
		"info":     true,
		"warning":  true,
		"error":    true,
		"critical": true,
	}

	if !validLevels[level] {
		panic(fmt.Sprintf("invalid logging level: %s (must be debug, info, warning, error, or critical)", level))
	}

	return NewSystemSetting("LOGGING_LEVEL", level)
}

// IBKRTransactionBuilder provides a fluent interface for creating IBKR transactions.
type IBKRTransactionBuilder struct {
	ID                string
	IBKRTransactionID string
	TransactionDate   time.Time
	Symbol            string
	ISIN              string
	Description       string
	TransactionType   string
	Quantity          float64
	Price             float64
	TotalAmount       float64
	Currency          string
	Fees              float64
	Status            string
	ReportDate        time.Time
	Notes             string
}

// NewIBKRTransaction creates an IBKRTransactionBuilder with sensible defaults.
func NewIBKRTransaction() *IBKRTransactionBuilder {
	return &IBKRTransactionBuilder{
		ID:                MakeID(),
		IBKRTransactionID: "IBKR_" + randomAlphanumeric(10),
		TransactionDate:   time.Now().UTC(),
		Symbol:            "AAPL",
		ISIN:              "US0378331005",
		Description:       "Buy AAPL",
		TransactionType:   "buy",
		Quantity:          10,
		Price:             150.00,
		TotalAmount:       1500.00,
		Currency:          "USD",
		Fees:              1.00,
		Status:            "pending",
		ReportDate:        time.Now().UTC(),
		Notes:             "",
	}
}

// WithStatus sets the transaction status.
func (b *IBKRTransactionBuilder) WithStatus(status string) *IBKRTransactionBuilder {
	b.Status = status
	return b
}

// WithSymbol sets the symbol.
func (b *IBKRTransactionBuilder) WithSymbol(symbol string) *IBKRTransactionBuilder {
	b.Symbol = symbol
	return b
}

// WithISIN sets the ISIN.
func (b *IBKRTransactionBuilder) WithISIN(isin string) *IBKRTransactionBuilder {
	b.ISIN = isin
	return b
}

// WithNotes sets the notes field.
func (b *IBKRTransactionBuilder) WithNotes(notes string) *IBKRTransactionBuilder {
	b.Notes = notes
	return b
}

// WithFees sets the fees.
func (b *IBKRTransactionBuilder) WithFees(fees float64) *IBKRTransactionBuilder {
	b.Fees = fees
	return b
}

// WithType sets the transaction type.
func (b *IBKRTransactionBuilder) WithType(txType string) *IBKRTransactionBuilder {
	b.TransactionType = txType
	return b
}

// WithQuantity sets the quantity.
func (b *IBKRTransactionBuilder) WithQuantity(qty float64) *IBKRTransactionBuilder {
	b.Quantity = qty
	return b
}

// WithPrice sets the price.
func (b *IBKRTransactionBuilder) WithPrice(price float64) *IBKRTransactionBuilder {
	b.Price = price
	return b
}

// WithTotalAmount sets the total amount.
func (b *IBKRTransactionBuilder) WithTotalAmount(amount float64) *IBKRTransactionBuilder {
	b.TotalAmount = amount
	return b
}

// Build creates the IBKR transaction in the database and returns it.
func (b *IBKRTransactionBuilder) Build(t *testing.T, db *sql.DB) model.IBKRTransaction {
	t.Helper()

	query := `
		INSERT INTO ibkr_transaction (
			id, ibkr_transaction_id, transaction_date, symbol, isin, description,
			transaction_type, quantity, price, total_amount, currency, fees,
			status, imported_at, report_date, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
	`

	_, err := db.Exec(query,
		b.ID, b.IBKRTransactionID, b.TransactionDate.Format("2006-01-02"),
		b.Symbol, b.ISIN, b.Description, b.TransactionType,
		b.Quantity, b.Price, b.TotalAmount, b.Currency, b.Fees,
		b.Status, b.ReportDate.Format("2006-01-02"), b.Notes,
	)
	if err != nil {
		t.Fatalf("Failed to create IBKR transaction: %v", err)
	}

	return model.IBKRTransaction{
		ID:                b.ID,
		IBKRTransactionID: b.IBKRTransactionID,
		TransactionDate:   b.TransactionDate,
		Symbol:            b.Symbol,
		ISIN:              b.ISIN,
		Description:       b.Description,
		TransactionType:   b.TransactionType,
		Quantity:          b.Quantity,
		Price:             b.Price,
		TotalAmount:       b.TotalAmount,
		Currency:          b.Currency,
		Fees:              b.Fees,
		Status:            b.Status,
		ReportDate:        b.ReportDate,
		Notes:             b.Notes,
	}
}

// ExchangeRateBuilder provides a fluent interface for creating exchange rates.
//
// Example usage:
//
//	rate := testutil.NewExchangeRate("USD", "EUR", "2024-01-01", 0.85).Build(t, db)
type ExchangeRateBuilder struct {
	ID           string
	FromCurrency string
	ToCurrency   string
	Rate         float64
	Date         time.Time
}

// NewExchangeRate creates an ExchangeRateBuilder with the provided parameters.
func NewExchangeRate(fromCurrency, toCurrency, date string, rate float64) *ExchangeRateBuilder {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		panic(fmt.Sprintf("invalid date format: %s (must be YYYY-MM-DD)", date))
	}

	return &ExchangeRateBuilder{
		ID:           MakeID(),
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Rate:         rate,
		Date:         parsedDate,
	}
}

// WithRate sets a custom rate.
func (b *ExchangeRateBuilder) WithRate(rate float64) *ExchangeRateBuilder {
	b.Rate = rate
	return b
}

// Build creates the exchange rate in the database and returns it.
func (b *ExchangeRateBuilder) Build(t *testing.T, db *sql.DB) model.ExchangeRate {
	t.Helper()

	query := `
		INSERT INTO exchange_rate (id, from_currency, to_currency, rate, date, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := db.Exec(query, b.ID, b.FromCurrency, b.ToCurrency, b.Rate, b.Date.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("Failed to create test exchange rate: %v", err)
	}

	return model.ExchangeRate{
		FromCurrency: b.FromCurrency,
		ToCurrency:   b.ToCurrency,
		Rate:         b.Rate,
		Date:         b.Date,
	}
}
