package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// =============================================================================
// GET LOGS
// =============================================================================

func TestDeveloperService_GetLogs(t *testing.T) {
	t.Run("returns empty logs when no logs exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		filters := &model.LogFilters{
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := svc.GetLogs(context.Background(), filters)
		if err != nil {
			t.Fatalf("GetLogs() returned unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("GetLogs() returned nil response")
		}
		if len(resp.Logs) != 0 {
			t.Errorf("Expected 0 logs, got %d", len(resp.Logs))
		}
	})

	t.Run("returns logs after DeleteLogs creates audit entry", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		// DeleteLogs creates an audit log entry
		ip := "127.0.0.1"
		err := svc.DeleteLogs(context.Background(), &ip, "test-agent")
		if err != nil {
			t.Fatalf("DeleteLogs() returned unexpected error: %v", err)
		}

		filters := &model.LogFilters{
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := svc.GetLogs(context.Background(), filters)
		if err != nil {
			t.Fatalf("GetLogs() returned unexpected error: %v", err)
		}
		if resp.Count == 0 {
			t.Error("Expected at least 1 log after DeleteLogs, got 0")
		}
	})

	t.Run("filters by level", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		// Create a log entry via DeleteLogs (creates INFO level log)
		err := svc.DeleteLogs(context.Background(), nil, "test-agent")
		if err != nil {
			t.Fatalf("DeleteLogs() error: %v", err)
		}

		// Filter for INFO level
		filters := &model.LogFilters{
			Levels:  []string{"INFO"},
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := svc.GetLogs(context.Background(), filters)
		if err != nil {
			t.Fatalf("GetLogs() error: %v", err)
		}
		for _, log := range resp.Logs {
			if log.Level != "INFO" {
				t.Errorf("Expected level INFO, got %s", log.Level)
			}
		}

		// Filter for ERROR level - should return 0
		filters.Levels = []string{"ERROR"}
		resp, err = svc.GetLogs(context.Background(), filters)
		if err != nil {
			t.Fatalf("GetLogs() error: %v", err)
		}
		if len(resp.Logs) != 0 {
			t.Errorf("Expected 0 ERROR logs, got %d", len(resp.Logs))
		}
	})
}

// =============================================================================
// LOGGING CONFIG
// =============================================================================

func TestDeveloperService_GetLoggingConfig(t *testing.T) {
	t.Run("returns default config when no settings exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		config, err := svc.GetLoggingConfig()
		if err != nil {
			t.Fatalf("GetLoggingConfig() error: %v", err)
		}
		// Default values: enabled=true, level="info"
		if !config.Enabled {
			t.Error("Expected default Enabled=true")
		}
		if config.Level != "info" {
			t.Errorf("Expected default Level='info', got %q", config.Level)
		}
	})

	t.Run("returns configured settings", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		// Set config via system_setting table
		testutil.NewLoggingEnabled(false).Build(t, db)
		testutil.NewLoggingLevel("warning").Build(t, db)

		config, err := svc.GetLoggingConfig()
		if err != nil {
			t.Fatalf("GetLoggingConfig() error: %v", err)
		}
		if config.Enabled {
			t.Error("Expected Enabled=false after setting")
		}
		if config.Level != "warning" {
			t.Errorf("Expected Level='warning', got %q", config.Level)
		}
	})
}

func TestDeveloperService_SetLoggingConfig(t *testing.T) {
	t.Run("updates logging level", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		req := request.SetLoggingConfig{
			Level: "debug",
		}
		config, err := svc.SetLoggingConfig(context.Background(), req)
		if err != nil {
			t.Fatalf("SetLoggingConfig() error: %v", err)
		}
		if config.Level != "debug" {
			t.Errorf("Expected Level='debug', got %q", config.Level)
		}
	})

	t.Run("updates enabled flag", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		enabled := false
		req := request.SetLoggingConfig{
			Enabled: &enabled,
		}
		config, err := svc.SetLoggingConfig(context.Background(), req)
		if err != nil {
			t.Fatalf("SetLoggingConfig() error: %v", err)
		}
		if config.Enabled {
			t.Error("Expected Enabled=false after update")
		}
	})

	t.Run("updates both level and enabled", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		enabled := true
		req := request.SetLoggingConfig{
			Enabled: &enabled,
			Level:   "error",
		}
		config, err := svc.SetLoggingConfig(context.Background(), req)
		if err != nil {
			t.Fatalf("SetLoggingConfig() error: %v", err)
		}
		if !config.Enabled {
			t.Error("Expected Enabled=true")
		}
		if config.Level != "error" {
			t.Errorf("Expected Level='error', got %q", config.Level)
		}
	})
}

// =============================================================================
// EXCHANGE RATE
// =============================================================================

func TestDeveloperService_GetExchangeRate(t *testing.T) {
	t.Run("returns exchange rate when it exists", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		testutil.NewExchangeRate("USD", "EUR", "2025-01-15", 0.85).Build(t, db)

		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		rate, err := svc.GetExchangeRate("USD", "EUR", date)
		if err != nil {
			t.Fatalf("GetExchangeRate() error: %v", err)
		}
		if rate == nil {
			t.Fatal("GetExchangeRate() returned nil")
		}
		if rate.Rate != 0.85 {
			t.Errorf("Expected rate 0.85, got %f", rate.Rate)
		}
	})

	t.Run("returns error when exchange rate not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		_, err := svc.GetExchangeRate("USD", "JPY", date)
		if err == nil {
			t.Error("Expected error for non-existent exchange rate, got nil")
		}
	})
}

func TestDeveloperService_UpdateExchangeRate(t *testing.T) {
	t.Run("creates new exchange rate", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		req := request.SetExchangeRateRequest{
			Date:         "2025-01-15",
			FromCurrency: "USD",
			ToCurrency:   "GBP",
			Rate:         "0.79",
		}
		result, err := svc.UpdateExchangeRate(context.Background(), req)
		if err != nil {
			t.Fatalf("UpdateExchangeRate() error: %v", err)
		}
		if result.Rate != 0.79 {
			t.Errorf("Expected rate 0.79, got %f", result.Rate)
		}
		if result.FromCurrency != "USD" {
			t.Errorf("Expected FromCurrency 'USD', got %q", result.FromCurrency)
		}
		if result.ToCurrency != "GBP" {
			t.Errorf("Expected ToCurrency 'GBP', got %q", result.ToCurrency)
		}
	})

	t.Run("returns error for invalid date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		req := request.SetExchangeRateRequest{
			Date:         "not-a-date",
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Rate:         "0.85",
		}
		_, err := svc.UpdateExchangeRate(context.Background(), req)
		if err == nil {
			t.Error("Expected error for invalid date, got nil")
		}
	})

	t.Run("returns error for invalid rate", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		req := request.SetExchangeRateRequest{
			Date:         "2025-01-15",
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Rate:         "not-a-number",
		}
		_, err := svc.UpdateExchangeRate(context.Background(), req)
		if err == nil {
			t.Error("Expected error for invalid rate, got nil")
		}
	})
}

// =============================================================================
// FUND PRICE
// =============================================================================

func TestDeveloperService_GetFundPrice(t *testing.T) {
	t.Run("returns fund price when it exists", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		testutil.NewFundPrice(fund.ID).WithDate(date).WithPrice(150.50).Build(t, db)

		fp, err := svc.GetFundPrice(fund.ID, date)
		if err != nil {
			t.Fatalf("GetFundPrice() error: %v", err)
		}
		if fp.Price != 150.50 {
			t.Errorf("Expected price 150.50, got %f", fp.Price)
		}
		if fp.FundID != fund.ID {
			t.Errorf("Expected FundID %q, got %q", fund.ID, fp.FundID)
		}
	})

	t.Run("returns ErrFundPriceNotFound when no price exists", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

		_, err := svc.GetFundPrice(fund.ID, date)
		if !errors.Is(err, apperrors.ErrFundPriceNotFound) {
			t.Errorf("Expected ErrFundPriceNotFound, got %v", err)
		}
	})
}

func TestDeveloperService_UpdateFundPrice(t *testing.T) {
	t.Run("creates new fund price", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		req := request.SetFundPriceRequest{
			Date:   "2025-01-15",
			FundID: fund.ID,
			Price:  "155.75",
		}
		result, err := svc.UpdateFundPrice(context.Background(), req)
		if err != nil {
			t.Fatalf("UpdateFundPrice() error: %v", err)
		}
		if result.Price != 155.75 {
			t.Errorf("Expected price 155.75, got %f", result.Price)
		}
		if result.FundID != fund.ID {
			t.Errorf("Expected FundID %q, got %q", fund.ID, result.FundID)
		}
	})

	t.Run("returns error for invalid date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		req := request.SetFundPriceRequest{
			Date:   "bad-date",
			FundID: "some-id",
			Price:  "10.0",
		}
		_, err := svc.UpdateFundPrice(context.Background(), req)
		if err == nil {
			t.Error("Expected error for invalid date, got nil")
		}
	})

	t.Run("returns error for invalid price", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		req := request.SetFundPriceRequest{
			Date:   "2025-01-15",
			FundID: "some-id",
			Price:  "abc",
		}
		_, err := svc.UpdateFundPrice(context.Background(), req)
		if err == nil {
			t.Error("Expected error for invalid price, got nil")
		}
	})
}

// =============================================================================
// IMPORT FUND PRICES
// =============================================================================

func TestDeveloperService_ImportFundPrices(t *testing.T) {
	t.Run("imports valid CSV", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		csv := []byte("date,price\n2025-01-15,100.50\n2025-01-16,101.25\n2025-01-17,99.75\n")

		count, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err != nil {
			t.Fatalf("ImportFundPrices() error: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 rows imported, got %d", count)
		}
	})

	t.Run("handles CSV with BOM", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		// UTF-8 BOM prefix
		csv := append([]byte{0xEF, 0xBB, 0xBF}, []byte("date,price\n2025-01-15,50.00\n")...)

		count, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err != nil {
			t.Fatalf("ImportFundPrices() error: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 row imported, got %d", count)
		}
	})

	t.Run("returns error for missing headers", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		csv := []byte("wrong,headers\n2025-01-15,100.50\n")

		_, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err == nil {
			t.Error("Expected error for missing headers, got nil")
		}
		if !errors.Is(err, apperrors.ErrInvalidCSVHeaders) {
			t.Errorf("Expected ErrInvalidCSVHeaders, got %v", err)
		}
	})

	t.Run("returns error for invalid date in row", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		csv := []byte("date,price\nbad-date,100.50\n")

		_, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err == nil {
			t.Error("Expected error for invalid date, got nil")
		}
	})

	t.Run("returns error for negative price", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		csv := []byte("date,price\n2025-01-15,-10.00\n")

		_, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err == nil {
			t.Error("Expected error for negative price, got nil")
		}
	})

	t.Run("returns error for non-existent fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		csv := []byte("date,price\n2025-01-15,100.50\n")

		_, err := svc.ImportFundPrices(context.Background(), "non-existent-fund-id", csv)
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("Expected ErrFundNotFound, got %v", err)
		}
	})

	t.Run("returns error for empty CSV", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		csv := []byte("")

		_, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err == nil {
			t.Error("Expected error for empty CSV, got nil")
		}
	})

	t.Run("returns error for headers only (no data rows)", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		csv := []byte("date,price\n")

		_, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err == nil {
			t.Error("Expected error for headers-only CSV, got nil")
		}
	})

	t.Run("returns error for non-numeric price", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		fund := testutil.NewFund().Build(t, db)
		csv := []byte("date,price\n2025-01-15,abc\n")

		_, err := svc.ImportFundPrices(context.Background(), fund.ID, csv)
		if err == nil {
			t.Error("Expected error for non-numeric price, got nil")
		}
	})
}

// =============================================================================
// IMPORT TRANSACTIONS
// =============================================================================

func TestDeveloperService_ImportTransactions(t *testing.T) {
	t.Run("imports valid CSV", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		csv := []byte("date,type,shares,cost_per_share\n2025-01-15,buy,100,10.50\n2025-01-16,sell,50,11.00\n")

		count, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
		if err != nil {
			t.Fatalf("ImportTransactions() error: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 rows imported, got %d", count)
		}
	})

	t.Run("returns error for missing headers", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		csv := []byte("date,wrong,headers\n2025-01-15,buy,100\n")

		_, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
		if err == nil {
			t.Error("Expected error for missing headers, got nil")
		}
		if !errors.Is(err, apperrors.ErrInvalidCSVHeaders) {
			t.Errorf("Expected ErrInvalidCSVHeaders, got %v", err)
		}
	})

	t.Run("returns error for invalid transaction type", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		csv := []byte("date,type,shares,cost_per_share\n2025-01-15,invalid_type,100,10.50\n")

		_, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
		if err == nil {
			t.Error("Expected error for invalid transaction type, got nil")
		}
	})

	t.Run("returns error for negative shares", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		csv := []byte("date,type,shares,cost_per_share\n2025-01-15,buy,-100,10.50\n")

		_, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
		if err == nil {
			t.Error("Expected error for negative shares, got nil")
		}
	})

	t.Run("returns error for negative cost_per_share", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		csv := []byte("date,type,shares,cost_per_share\n2025-01-15,buy,100,-10.50\n")

		_, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
		if err == nil {
			t.Error("Expected error for negative cost_per_share, got nil")
		}
	})

	t.Run("returns error for non-existent portfolio fund", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		csv := []byte("date,type,shares,cost_per_share\n2025-01-15,buy,100,10.50\n")

		_, err := svc.ImportTransactions(context.Background(), "non-existent-pf-id", csv)
		if !errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			t.Errorf("Expected ErrPortfolioFundNotFound, got %v", err)
		}
	})

	t.Run("accepts all valid transaction types", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Need an initial buy for sell to work with
		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).
			WithShares(1000).
			WithCostPerShare(10.0).
			Build(t, db)

		csv := []byte("date,type,shares,cost_per_share\n2025-01-15,buy,100,10.50\n2025-01-16,sell,50,11.00\n2025-01-17,dividend,10,0.50\n2025-01-18,fee,1,5.00\n")

		count, err := svc.ImportTransactions(context.Background(), pf.ID, csv)
		if err != nil {
			t.Fatalf("ImportTransactions() error: %v", err)
		}
		if count != 4 {
			t.Errorf("Expected 4 rows imported, got %d", count)
		}
	})
}

// =============================================================================
// DELETE LOGS
// =============================================================================

func TestDeveloperService_DeleteLogs(t *testing.T) {
	t.Run("deletes logs and creates audit entry", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		ip := "192.168.1.1"
		err := svc.DeleteLogs(context.Background(), &ip, "Mozilla/5.0")
		if err != nil {
			t.Fatalf("DeleteLogs() error: %v", err)
		}

		// Should have exactly 1 audit log entry
		filters := &model.LogFilters{
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := svc.GetLogs(context.Background(), filters)
		if err != nil {
			t.Fatalf("GetLogs() error: %v", err)
		}
		if resp.Count != 1 {
			t.Errorf("Expected 1 audit log entry, got %d", resp.Count)
		}
		if resp.Logs[0].Message != "All logs cleared by user" {
			t.Errorf("Expected audit message, got %q", resp.Logs[0].Message)
		}
	})

	t.Run("works with nil IP address", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		err := svc.DeleteLogs(context.Background(), nil, "test-agent")
		if err != nil {
			t.Fatalf("DeleteLogs() error: %v", err)
		}
	})

	t.Run("clears previous logs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		// Create some logs by deleting twice
		err := svc.DeleteLogs(context.Background(), nil, "agent1")
		if err != nil {
			t.Fatalf("first DeleteLogs() error: %v", err)
		}

		// Second delete should clear the first audit entry and create a new one
		err = svc.DeleteLogs(context.Background(), nil, "agent2")
		if err != nil {
			t.Fatalf("second DeleteLogs() error: %v", err)
		}

		filters := &model.LogFilters{
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := svc.GetLogs(context.Background(), filters)
		if err != nil {
			t.Fatalf("GetLogs() error: %v", err)
		}
		// Should only have 1 entry (from the second delete)
		if resp.Count != 1 {
			t.Errorf("Expected 1 log entry after second delete, got %d", resp.Count)
		}
	})
}

// =============================================================================
// GET LOG FILTER OPTIONS
// =============================================================================

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestDeveloperService_GetLogFilterOptions(t *testing.T) {
	t.Run("returns empty options when no logs exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		opts, err := svc.GetLogFilterOptions()
		if err != nil {
			t.Fatalf("GetLogFilterOptions() returned unexpected error: %v", err)
		}
		if opts == nil {
			t.Fatal("GetLogFilterOptions() returned nil")
		}
		if len(opts.Levels) != 0 {
			t.Errorf("Expected 0 levels, got %d", len(opts.Levels))
		}
		if len(opts.Categories) != 0 {
			t.Errorf("Expected 0 categories, got %d", len(opts.Categories))
		}
		if len(opts.Sources) != 0 {
			t.Errorf("Expected 0 sources, got %d", len(opts.Sources))
		}
		if len(opts.Messages) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(opts.Messages))
		}
	})

	t.Run("returns populated options after logs are created", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		// Seed a log entry via DeleteLogs (creates an INFO-level audit log)
		err := svc.DeleteLogs(context.Background(), nil, "test-agent")
		if err != nil {
			t.Fatalf("DeleteLogs() error: %v", err)
		}

		opts, err := svc.GetLogFilterOptions()
		if err != nil {
			t.Fatalf("GetLogFilterOptions() returned unexpected error: %v", err)
		}
		if len(opts.Levels) == 0 {
			t.Error("Expected at least one level after seeding logs")
		}
		if len(opts.Categories) == 0 {
			t.Error("Expected at least one category after seeding logs")
		}
		if len(opts.Sources) == 0 {
			t.Error("Expected at least one source after seeding logs")
		}
		if len(opts.Messages) == 0 {
			t.Error("Expected at least one message after seeding logs")
		}
	})

	t.Run("levels are returned in uppercase", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		// Seed a log via DeleteLogs
		err := svc.DeleteLogs(context.Background(), nil, "test-agent")
		if err != nil {
			t.Fatalf("DeleteLogs() error: %v", err)
		}

		opts, err := svc.GetLogFilterOptions()
		if err != nil {
			t.Fatalf("GetLogFilterOptions() error: %v", err)
		}
		for _, level := range opts.Levels {
			if level != strings.ToUpper(level) {
				t.Errorf("Expected level %q to be uppercase", level)
			}
		}
	})

	t.Run("categories are returned in uppercase", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestDeveloperService(t, db)

		// Seed a log via DeleteLogs
		err := svc.DeleteLogs(context.Background(), nil, "test-agent")
		if err != nil {
			t.Fatalf("DeleteLogs() error: %v", err)
		}

		opts, err := svc.GetLogFilterOptions()
		if err != nil {
			t.Fatalf("GetLogFilterOptions() error: %v", err)
		}
		for _, cat := range opts.Categories {
			if cat != strings.ToUpper(cat) {
				t.Errorf("Expected category %q to be uppercase", cat)
			}
		}
	})
}
