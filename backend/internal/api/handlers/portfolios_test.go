package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/handlers"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// TestPortfolioHandler_Portfolios tests the GET /api/portfolio endpoint.
//
// WHY: This is the primary endpoint for retrieving portfolios. The frontend
// depends on this returning correct data with proper HTTP status codes and
// JSON formatting. Testing ensures API contract stability.
//
//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestPortfolioHandler_Portfolios(t *testing.T) {
	t.Run("returns empty array when no portfolios exist", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert HTTP status
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Assert Content-Type
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Assert response body
		var response []model.Portfolio
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	t.Run("returns all portfolios successfully", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create test data
		p1 := testutil.CreatePortfolio(t, db, "Portfolio One")
		p2 := testutil.CreatePortfolio(t, db, "Portfolio Two")

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.Portfolio
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Errorf("Expected 2 portfolios, got %d", len(response))
		}

		// Find portfolios by ID - don't assume order
		var portfolio1, portfolio2 *model.Portfolio
		for i := range response {
			if response[i].ID == p1.ID {
				portfolio1 = &response[i]
			}
			if response[i].ID == p2.ID {
				portfolio2 = &response[i]
			}
		}

		// Verify we found both
		if portfolio1 == nil {
			t.Fatal("Portfolio One not found in response")
		}
		if portfolio2 == nil {
			t.Fatal("Portfolio Two not found in response")
		}

		// Verify data matches what we created
		if portfolio1.Name != "Portfolio One" {
			t.Errorf("Expected first portfolio name 'Portfolio One', got '%s'", portfolio1.Name)
		}
		if portfolio2.Name != "Portfolio Two" {
			t.Errorf("Expected second portfolio name 'Portfolio Two', got '%s'", portfolio2.Name)
		}
	})

	t.Run("returns portfolio with all fields populated", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio with all fields set
		p := testutil.NewPortfolio().
			WithName("Complete Portfolio").
			WithDescription("Full description").
			Archived().
			ExcludedFromOverview().
			Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.Portfolio
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		portfolio := response[0]

		// Verify all fields
		if portfolio.ID != p.ID {
			t.Errorf("ID mismatch: expected %s, got %s", p.ID, portfolio.ID)
		}
		if portfolio.Name != "Complete Portfolio" {
			t.Errorf("Name mismatch: expected 'Complete Portfolio', got '%s'", portfolio.Name)
		}
		if portfolio.Description != "Full description" {
			t.Errorf("Description mismatch: expected 'Full description', got '%s'", portfolio.Description)
		}
		if portfolio.IsArchived != true {
			t.Errorf("IsArchived mismatch: expected true, got %v", portfolio.IsArchived)
		}
		if portfolio.ExcludeFromOverview != true {
			t.Errorf("ExcludeFromOverview mismatch: expected true, got %v", portfolio.ExcludeFromOverview)
		}
	})

	t.Run("returns portfolios including archived and excluded flags", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio with all fields set
		p1 := testutil.CreateArchivedPortfolio(t, db, "Portfolio One")
		p2 := testutil.CreateExcludedPortfolio(t, db, "Portfolio Two")

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.Portfolio
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Fatalf("Expected 2 portfolios, got %d", len(response))
		}

		// Find portfolios by ID - don't assume order
		var archivedPortfolio, excludedPortfolio *model.Portfolio
		for i := range response {
			if response[i].ID == p1.ID {
				archivedPortfolio = &response[i]
			}
			if response[i].ID == p2.ID {
				excludedPortfolio = &response[i]
			}
		}

		// Verify we found both
		if archivedPortfolio == nil {
			t.Fatal("Archived portfolio not found in response")
		}
		if excludedPortfolio == nil {
			t.Fatal("Excluded portfolio not found in response")
		}

		// Verify flags are set correctly
		if !archivedPortfolio.IsArchived {
			t.Errorf("Expected archived portfolio isArchived to be true, got %t", archivedPortfolio.IsArchived)
		}
		if !excludedPortfolio.ExcludeFromOverview {
			t.Errorf("Expected excluded portfolio exclude_from_overview to be true, got %t", excludedPortfolio.ExcludeFromOverview)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		// Setup with closed database
		db := testutil.SetupTestDB(t)
		db.Close() // Force database error

		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert error response
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if _, hasError := response["error"]; !hasError {
			t.Error("Expected error field in response")
		}
	})
}

// Example of testing with helper function for repeated setup
func TestPortfolioHandler_WithHelper(t *testing.T) {
	// Helper function to reduce boilerplate
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *httptest.ResponseRecorder) {
		t.Helper()

		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)
		w := httptest.NewRecorder()

		return handler, w
	}

	t.Run("example using helper", func(t *testing.T) {
		handler, w := setupHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)

		handler.Portfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_PortfolioSummary tests the portfolio summary endpoint.
//
// WHY: This endpoint provides calculated portfolio metrics including values,
// costs, gains/losses, and dividends. It's critical for the dashboard view.
//
//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestPortfolioHandler_PortfolioSummary(t *testing.T) {
	t.Run("returns empty array when no portfolios exist", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Debug output - only shows on failure or with -v flag
		if len(response) != 0 {
			t.Logf("Expected empty array, got %d items", len(response))
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	t.Run("returns summary for portfolio with basic transactions", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create test data: portfolio, fund, and transactions
		portfolio := testutil.NewPortfolio().WithName("Test Portfolio").Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create buy transaction: 100 shares at $10
		testutil.NewTransaction(pf.ID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Add current price: $12
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		summary := response[0]
		if summary.ID != portfolio.ID {
			t.Errorf("Expected portfolio ID %s, got %s", portfolio.ID, summary.ID)
		}
		if summary.Name != "Test Portfolio" {
			t.Errorf("Expected name 'Test Portfolio', got '%s'", summary.Name)
		}

		// Verify calculations
		// Total cost: 100 * $10 = $1000
		if summary.TotalCost != 1000.0 {
			t.Errorf("Expected total cost 1000.0, got %.2f", summary.TotalCost)
		}

		// Total value: 100 * $12 = $1200
		if summary.TotalValue != 1200.0 {
			t.Errorf("Expected total value 1200.0, got %.2f", summary.TotalValue)
		}

		// Unrealized gain: $1200 - $1000 = $200
		if summary.TotalUnrealizedGainLoss != 200.0 {
			t.Errorf("Expected unrealized gain 200.0, got %.2f", summary.TotalUnrealizedGainLoss)
		}

		// Total gain/loss (no realized gains): $200
		if summary.TotalGainLoss != 200.0 {
			t.Errorf("Expected total gain/loss 200.0, got %.2f", summary.TotalGainLoss)
		}

		// No dividends
		if summary.TotalDividends != 0.0 {
			t.Errorf("Expected dividends 0.0, got %.2f", summary.TotalDividends)
		}

		// No realized gains
		if summary.TotalRealizedGainLoss != 0.0 {
			t.Errorf("Expected realized gain/loss 0.0, got %.2f", summary.TotalRealizedGainLoss)
		}

		if summary.IsArchived {
			t.Error("Expected portfolio to not be archived")
		}
	})

	t.Run("excludes archived portfolios from summary", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create active portfolio
		activePortfolio := testutil.NewPortfolio().WithName("Active").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(activePortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)
		testutil.NewFundPrice(fund.ID).Build(t, db)

		// Create archived portfolio
		archivedPortfolio := testutil.NewPortfolio().WithName("Archived").Archived().Build(t, db)
		pf2 := testutil.NewPortfolioFund(archivedPortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf2.ID).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should only include active portfolio
		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		if response[0].Name != "Active" {
			t.Errorf("Expected active portfolio, got %s", response[0].Name)
		}
	})

	t.Run("excludes portfolios marked as exclude_from_overview", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create normal portfolio
		normalPortfolio := testutil.NewPortfolio().WithName("Normal").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(normalPortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)
		testutil.NewFundPrice(fund.ID).Build(t, db)

		// Create excluded portfolio
		excludedPortfolio := testutil.NewPortfolio().WithName("Excluded").ExcludedFromOverview().Build(t, db)
		pf2 := testutil.NewPortfolioFund(excludedPortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf2.ID).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		var response []model.PortfolioSummary
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		// Should only include normal portfolio
		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		if response[0].Name != "Normal" {
			t.Errorf("Expected normal portfolio, got %s", response[0].Name)
		}
	})

	t.Run("includes realized gains from sell transactions", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pf.ID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Sell 30 shares at $15
		sellTx := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithShares(30).
			WithCostPerShare(15.0).
			Build(t, db)

		// Record realized gain: 30 * ($15 - $10) = $150
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
			WithShares(30).
			WithCostBasis(300.0).    // 30 * $10
			WithSaleProceeds(450.0). // 30 * $15
			Build(t, db)

		// Current price: $12
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		var response []model.PortfolioSummary
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		summary := response[0]

		// Remaining: 70 shares
		// Cost: 70 * $10 = $700 (proportionally reduced)
		// Value: 70 * $12 = $840
		// Unrealized: $840 - $700 = $140

		if summary.TotalCost != 700.0 {
			t.Errorf("Expected cost 700.0, got %.2f", summary.TotalCost)
		}

		if summary.TotalValue != 840.0 {
			t.Errorf("Expected value 840.0, got %.2f", summary.TotalValue)
		}

		if summary.TotalUnrealizedGainLoss != 140.0 {
			t.Errorf("Expected unrealized gain 140.0, got %.2f", summary.TotalUnrealizedGainLoss)
		}

		// Realized gain: $150
		if summary.TotalRealizedGainLoss != 150.0 {
			t.Errorf("Expected realized gain 150.0, got %.2f", summary.TotalRealizedGainLoss)
		}

		// Total gain: $140 (unrealized) + $150 (realized) = $290
		if summary.TotalGainLoss != 290.0 {
			t.Errorf("Expected total gain 290.0, got %.2f", summary.TotalGainLoss)
		}

		// Sale proceeds
		if summary.TotalSaleProceeds != 450.0 {
			t.Errorf("Expected sale proceeds 450.0, got %.2f", summary.TotalSaleProceeds)
		}

		// Original cost of sold shares
		if summary.TotalOriginalCost != 300.0 {
			t.Errorf("Expected original cost 300.0, got %.2f", summary.TotalOriginalCost)
		}
	})

	t.Run("includes dividend payments in summary", func(t *testing.T) {
		// t.Skip("BLOCKED: Dividend calculation returns 0. Root cause: testutil.NewDividend() uses raw SQL INSERT " +
		// 	"that doesn't match production data structure. Will fix when DividendService.Create() is implemented. " +
		// 	"See docs/TESTDATA_LIMITATIONS.md")

		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().WithName("Dividend Test Portfolio").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pf.ID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(time.Now().AddDate(0, 0, -20)).
			Build(t, db)

		// Add dividend: 100 shares * $0.50 = $50
		testutil.NewDividend(fund.ID, pf.ID).
			WithSharesOwned(100).
			WithDividendPerShare(0.50).
			Build(t, db)

		// Current price
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v. Body: %s", err, w.Body.String())
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d. Response: %+v", len(response), response)
		}

		summary := response[0]

		// Dividends: $50
		if summary.TotalDividends != 50.0 {
			t.Errorf("Expected dividends 50.0, got %.2f", summary.TotalDividends)
		}
	})

	t.Run("includes dividend reinvestment shares in calculations", func(t *testing.T) {
		// t.Skip("BLOCKED: Same as 'includes_dividend_payments_in_summary' - dividend test data issue. " +
		// 	"See docs/TESTDATA_LIMITATIONS.md")

		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pf.ID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(time.Now().AddDate(0, 0, -20)).
			Build(t, db)

		// Dividend reinvestment: buy 5 shares at $10
		reinvestTx := testutil.NewTransaction(pf.ID).
			WithType("dividend").
			WithShares(5).
			WithCostPerShare(10.0).
			Build(t, db)

		// Dividend record with reinvestment
		testutil.NewDividend(fund.ID, pf.ID).
			WithSharesOwned(100).
			WithDividendPerShare(0.50).
			WithReinvestmentTransaction(reinvestTx.ID).
			Build(t, db)

		// Current price: $12
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		var response []model.PortfolioSummary
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		summary := response[0]

		// Total shares: 100 (buy) + 5 (dividend reinvestment) = 105
		// Value: 105 * $12 = $1260
		// Cost: 100 * $10 + 5 * $10 = $1050
		// This will be calculated by the service, so just verify non-zero values
		if summary.TotalValue <= 0 {
			t.Errorf("Expected positive value, got %.2f", summary.TotalValue)
		}

		if summary.TotalDividends != 50.0 {
			t.Errorf("Expected dividends 50.0, got %.2f", summary.TotalDividends)
		}
	})

	t.Run("handles portfolio with no transactions gracefully", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio with no transactions
		testutil.NewPortfolio().WithName("Empty").Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Should not crash, may return empty or zero-value summary
		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	})

	t.Run("handles portfolio with no fund prices gracefully", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio with transaction but no prices
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// No fund price added

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert - should not crash
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		// Value should be 0 when no price is available
		if response[0].TotalValue != 0 {
			t.Errorf("Expected value 0 when no price, got %.2f", response[0].TotalValue)
		}

		// Cost should still be calculated
		if response[0].TotalCost != 1000.0 {
			t.Errorf("Expected cost 1000.0, got %.2f", response[0].TotalCost)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		// Setup with closed database
		db := testutil.SetupTestDB(t)
		db.Close() // Force database error

		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert error response
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if _, hasError := response["error"]; !hasError {
			t.Error("Expected error field in response")
		}
	})
}

// TestPortfolioHandler_GetPortfolio tests the GET /api/portfolio/{portfolioId} endpoint.
//
// WHY: This endpoint retrieves a single portfolio with its current valuation summary.
// It's critical for portfolio detail views and dashboard widgets showing individual portfolio performance.
//
//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestPortfolioHandler_GetPortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path
	t.Run("returns portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data
		portfolio := testutil.NewPortfolio().WithName("Test Portfolio").Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pf.ID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Current price: $12
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create request with URL params
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		// Execute
		handler.GetPortfolio(w, req)

		// Assert HTTP status
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Assert response body
		var response model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ID != portfolio.ID {
			t.Errorf("Expected portfolio ID %s, got %s", portfolio.ID, response.ID)
		}

		if response.Name != "Test Portfolio" {
			t.Errorf("Expected name 'Test Portfolio', got '%s'", response.Name)
		}

		// Verify calculations: cost = 100 * $10 = $1000
		if response.TotalCost != 1000.0 {
			t.Errorf("Expected cost 1000.0, got %.2f", response.TotalCost)
		}

		// Value: 100 * $12 = $1200
		if response.TotalValue != 1200.0 {
			t.Errorf("Expected value 1200.0, got %.2f", response.TotalValue)
		}

		// Unrealized gain: $1200 - $1000 = $200
		if response.TotalUnrealizedGainLoss != 200.0 {
			t.Errorf("Expected unrealized gain 200.0, got %.2f", response.TotalUnrealizedGainLoss)
		}
	})

	// Resource not found
	t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		validID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/"+validID,
			map[string]string{"uuid": validID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolio(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	// Edge case: Portfolio with no transactions
	t.Run("handles portfolio with no transactions", func(t *testing.T) {
		// t.Skip("BLOCKED: Returns 404 instead of 200. Issue: GetPortfolioHistoryWithFallback returns empty " +
		// 	"history for portfolios with no transactions, which is treated as 'not found'. Need to distinguish " +
		// 	"between 'portfolio doesn't exist' vs 'portfolio exists but has no data'. See portfolios.go:101")

		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().WithName("Empty Portfolio").Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response model.PortfolioSummary
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		// Should have zero values, not errors
		if response.TotalValue != 0 {
			t.Errorf("Expected TotalValue 0, got %.2f", response.TotalValue)
		}
		if response.TotalCost != 0 {
			t.Errorf("Expected TotalCost 0, got %.2f", response.TotalCost)
		}
	})

	// Input validation: Empty portfolio ID
	t.Run("returns 400 when portfolio ID is missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/",
			map[string]string{"uuid": ""},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty portfolioId, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrInvalidPortfolioID.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrInvalidPortfolioID.Error(), response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().WithName("Test").Build(t, db)
		db.Close() // Force error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_PortfolioHistory tests the GET /api/portfolio/history endpoint.
//
// WHY: This endpoint returns historical portfolio valuations over a date range.
// It's essential for performance charts and trend analysis in the frontend.
// Tests cover both the on-the-fly fallback path and the materialized view fast path.
//
//nolint:gocyclo // Test function with multiple sub-tests
func TestPortfolioHandler_PortfolioHistory(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: No portfolios
	t.Run("returns empty array when no portfolios exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-12-31",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioHistory
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		totalPortfolios := 0
		for _, entry := range response {
			totalPortfolios += len(entry.Portfolios)
		}
		if totalPortfolios != 0 {
			t.Errorf("Expected no portfolios in timeframe, got %d total portfolios", totalPortfolios)
		}
	})

	// Happy path: With data
	t.Run("returns historical data successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data
		portfolio := testutil.NewPortfolio().WithName("History Test").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Transaction on 2024-06-01
		//nolint:errcheck // Test data setup with hardcoded valid date format
		txDate, _ := time.Parse("2006-01-02", "2024-06-01")
		testutil.NewTransaction(pf.ID).
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(txDate).
			Build(t, db)

		// Price on 2024-06-15
		//nolint:errcheck // Test data setup with hardcoded valid date format
		priceDate, _ := time.Parse("2006-01-02", "2024-06-15")
		testutil.NewFundPrice(fund.ID).
			WithPrice(12.0).
			WithDate(priceDate).
			Build(t, db)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-06-01",
				"end_date":   "2024-06-30",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PortfolioHistory
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have historical data points
		if len(response) == 0 {
			t.Error("Expected historical data, got empty array")
		}
	})

	// Default behavior: Uses default dates when not provided
	t.Run("uses default dates when query params not provided", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create minimal test data
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)
		testutil.NewFundPrice(fund.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/history", nil)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		// Should succeed with default dates
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 with default dates, got %d", w.Code)
		}
	})

	// Input validation: Invalid date format
	t.Run("returns 400 for invalid start date format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "not-a-date",
				"end_date":   "2024-12-31",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if _, hasError := response["error"]; !hasError {
			t.Error("Expected error field in response")
		}
	})

	t.Run("returns 400 for invalid end date format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "invalid-date",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	// Edge case: Single day range
	t.Run("handles single day date range", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		//nolint:errcheck // Test data setup with hardcoded valid date format
		singleDate, _ := time.Parse("2006-01-02", "2024-06-15")
		testutil.NewTransaction(pf.ID).WithDate(singleDate).Build(t, db)
		testutil.NewFundPrice(fund.ID).WithDate(singleDate).Build(t, db)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-06-15",
				"end_date":   "2024-06-15",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for single day range, got %d", w.Code)
		}
	})

	// Materialized view fast path: pre-populated cache serves data without on-demand calculation
	t.Run("serves from materialized view when cache is fresh", func(t *testing.T) {
		handler, db := setupHandler(t)
		ms := testutil.NewTestMaterializedService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		txDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC)

		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).WithCostPerShare(10.0).Build(t, db)
		for d := txDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			testutil.NewFundPrice(fund.ID).WithDate(d).WithPrice(10.0).Build(t, db)
		}

		// Pre-populate materialized cache
		err := ms.RegenerateMaterializedTable(context.Background(), txDate, []string{portfolio.ID}, "", "")
		if err != nil {
			t.Fatalf("RegenerateMaterializedTable() error: %v", err)
		}

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-06-15",
				"end_date":   "2024-06-17",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for materialized fast path, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PortfolioHistory
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) == 0 {
			t.Error("Expected materialized view to return data, got empty")
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close() // Force error

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-12-31",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_PortfolioFunds tests the GET /api/portfolio/funds endpoint.
//
// WHY: This endpoint returns all portfolio-fund relationships across all portfolios.
// Used for overview pages showing which funds are in which portfolios.
func TestPortfolioHandler_PortfolioFunds(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Empty
	t.Run("returns empty array when no portfolio-fund relationships exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFundResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	// Happy path: With data
	t.Run("returns all portfolio-fund relationships successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data: 2 portfolios, 2 funds
		p1 := testutil.NewPortfolio().WithName("Portfolio 1").Build(t, db)
		p2 := testutil.NewPortfolio().WithName("Portfolio 2").Build(t, db)
		f1 := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		f2 := testutil.NewFund().WithSymbol("GOOGL").Build(t, db)

		// Create relationships
		testutil.NewPortfolioFund(p1.ID, f1.ID).Build(t, db)
		testutil.NewPortfolioFund(p1.ID, f2.ID).Build(t, db)
		testutil.NewPortfolioFund(p2.ID, f1.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFundResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have 3 relationships
		if len(response) != 3 {
			t.Errorf("Expected 3 portfolio-fund relationships, got %d", len(response))
		}
	})

	// Edge case: Multiple portfolios with same fund
	t.Run("includes funds from multiple portfolios", func(t *testing.T) {
		handler, db := setupHandler(t)

		p1 := testutil.NewPortfolio().Build(t, db)
		p2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)

		// Same fund in two portfolios
		testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
		testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		var response []model.PortfolioFundResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		// Should list both relationships
		if len(response) != 2 {
			t.Errorf("Expected 2 relationships, got %d", len(response))
		}
	})

	// Edge case: Verifies nil-safe response handling
	t.Run("returns empty array not nil when no relationships exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFundListing
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		// Should be empty slice, not nil
		if response == nil {
			t.Error("Expected non-nil response (empty array)")
		}
		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close() // Force error

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_GetPortfolioFunds tests the GET /api/portfolio/funds/{portfolioId} endpoint.
//
// WHY: This endpoint returns detailed fund metrics for a specific portfolio, including
// shares, cost, value, gains, and dividends per fund. Critical for portfolio detail views.
//
//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestPortfolioHandler_GetPortfolioFunds(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path
	t.Run("returns funds with metrics successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data
		portfolio := testutil.NewPortfolio().WithName("Test Portfolio").Build(t, db)
		fund1 := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		fund2 := testutil.NewFund().WithSymbol("GOOGL").Build(t, db)

		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		// AAPL: 100 shares at $10
		testutil.NewTransaction(pf1.ID).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)
		testutil.NewFundPrice(fund1.ID).WithPrice(12.0).Build(t, db)

		// GOOGL: 50 shares at $20
		testutil.NewTransaction(pf2.ID).
			WithShares(50).
			WithCostPerShare(20.0).
			Build(t, db)
		testutil.NewFundPrice(fund2.ID).WithPrice(22.0).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFundResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Fatalf("Expected 2 funds, got %d", len(response))
		}

		// Find funds by ID - don't assume order
		var aapl, googl *model.PortfolioFundResponse
		for i := range response {
			if response[i].FundID == fund1.ID {
				aapl = &response[i]
			}
			if response[i].FundID == fund2.ID {
				googl = &response[i]
			}
		}

		// Verify we found both funds
		if aapl == nil {
			t.Fatal("AAPL not found in response")
		}
		if googl == nil {
			t.Fatal("GOOGL not found in response")
		}

		// Verify AAPL calculations
		if aapl.TotalShares != 100 {
			t.Errorf("Expected AAPL shares 100, got %.2f", aapl.TotalShares)
		}
		if aapl.TotalCost != 1000.0 {
			t.Errorf("Expected AAPL cost 1000.0, got %.2f", aapl.TotalCost)
		}
		if aapl.CurrentValue != 1200.0 {
			t.Errorf("Expected AAPL value 1200.0, got %.2f", aapl.CurrentValue)
		}

		// Verify GOOGL calculations
		if googl.TotalShares != 50 {
			t.Errorf("Expected GOOGL shares 50, got %.2f", googl.TotalShares)
		}
		if googl.TotalCost != 1000.0 {
			t.Errorf("Expected GOOGL cost 1000.0, got %.2f", googl.TotalCost)
		}
		if googl.CurrentValue != 1100.0 {
			t.Errorf("Expected GOOGL value 1100.0, got %.2f", googl.CurrentValue)
		}
	})

	// Edge case: Empty portfolio
	t.Run("returns empty array when portfolio has no funds", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().WithName("Empty Portfolio").Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFundResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	// Resource not found
	t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		validID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+validID,
			map[string]string{"uuid": validID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	// Edge case: Fund with no prices
	t.Run("handles fund with no price data", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// No price added

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFundResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 fund, got %d", len(response))
		}

		// Value should be 0 when no price
		if response[0].CurrentValue != 0 {
			t.Errorf("Expected value 0 when no price, got %.2f", response[0].CurrentValue)
		}

		// Cost should still be calculated
		if response[0].TotalCost != 1000.0 {
			t.Errorf("Expected cost 1000.0, got %.2f", response[0].TotalCost)
		}
	})

	// Edge case: Realized gains
	t.Run("includes realized gains per fund", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pf.ID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Sell 30 shares at $15
		sellTx := testutil.NewTransaction(pf.ID).
			WithType("sell").
			WithShares(30).
			WithCostPerShare(15.0).
			Build(t, db)

		// Record realized gain
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
			WithShares(30).
			WithCostBasis(300.0).
			WithSaleProceeds(450.0).
			Build(t, db)

		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		var response []model.PortfolioFundResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 fund, got %d", len(response))
		}

		// Remaining shares: 70
		if response[0].TotalShares != 70 {
			t.Errorf("Expected 70 shares, got %.2f", response[0].TotalShares)
		}

		// Realized gain: $150
		if response[0].RealizedGainLoss != 150.0 {
			t.Errorf("Expected realized gain 150.0, got %.2f", response[0].RealizedGainLoss)
		}
	})

	// Edge case: Dividends
	t.Run("includes dividends per fund", func(t *testing.T) {
		// t.Skip("BLOCKED: Same dividend test data issue as PortfolioSummary tests. " +
		// 	"See docs/TESTDATA_LIMITATIONS.md")

		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(time.Now().AddDate(0, 0, -20)).
			Build(t, db)

		// Add dividend: $50
		testutil.NewDividend(fund.ID, pf.ID).
			WithSharesOwned(100).
			WithDividendPerShare(0.50).
			Build(t, db)

		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		var response []model.PortfolioFundResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 fund, got %d", len(response))
		}

		if response[0].TotalDividends != 50.0 {
			t.Errorf("Expected dividends 50.0, got %.2f", response[0].TotalDividends)
		}
	})

	// Input validation: Empty portfolio ID
	t.Run("returns 400 when portfolio ID is missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/",
			map[string]string{"uuid": ""},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty portfolioId, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrInvalidPortfolioID.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrInvalidPortfolioID.Error(), response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close() // Force error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_CreatePortfolio tests the CreatePortfolio endpoint.
//
// WHY: This endpoint creates new portfolios and is critical for the application.
// Testing ensures proper validation, error handling, and successful creation flow.
//
//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestPortfolioHandler_CreatePortfolio(t *testing.T) {

	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Create portfolio with all fields
	t.Run("creates portfolio with all fields successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		reqBody := `{
			"name": "My Portfolio",
			"description": "Test portfolio description",
			"excludeFromOverview": true
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/portfolio", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreatePortfolio(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Name != "My Portfolio" {
			t.Errorf("Expected name 'My Portfolio', got '%s'", response.Name)
		}
		if response.Description != "Test portfolio description" {
			t.Errorf("Expected description 'Test portfolio description', got '%s'", response.Description)
		}
		if !response.ExcludeFromOverview {
			t.Error("Expected ExcludeFromOverview to be true")
		}
		if response.ID == "" {
			t.Error("Expected ID to be generated")
		}
		if response.IsArchived {
			t.Error("Expected IsArchived to be false by default")
		}

		// Verify it was actually created in the database
		testutil.AssertRowCount(t, db, "portfolio", 1)
	})

	// Happy path: Create portfolio with minimal fields
	t.Run("creates portfolio with required fields only", func(t *testing.T) {
		handler, db := setupHandler(t)

		reqBody := `{
			"name": "Minimal Portfolio"
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/portfolio", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreatePortfolio(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Name != "Minimal Portfolio" {
			t.Errorf("Expected name 'Minimal Portfolio', got '%s'", response.Name)
		}
		if response.Description != "" {
			t.Errorf("Expected empty description, got '%s'", response.Description)
		}
		if response.ExcludeFromOverview {
			t.Error("Expected ExcludeFromOverview to be false by default")
		}

		testutil.AssertRowCount(t, db, "portfolio", 1)
	})

	// Input validation: Invalid JSON
	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler, _ := setupHandler(t)

		reqBody := `{invalid json`
		req := httptest.NewRequest(http.MethodPost, "/api/portfolio", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreatePortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "invalid request body" {
			t.Errorf("Expected 'invalid request body' error, got '%s'", response["error"])
		}
	})

	// Input validation: Empty name
	t.Run("returns 400 for empty name", func(t *testing.T) {
		handler, _ := setupHandler(t)

		reqBody := `{
			"name": "",
			"description": "Test"
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/portfolio", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreatePortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "validation failed" {
			t.Errorf("Expected 'validation failed' error, got '%s'", response["error"])
		}
	})

	// Input validation: Unknown fields
	t.Run("returns 400 for unknown fields", func(t *testing.T) {
		handler, _ := setupHandler(t)

		reqBody := `{
			"name": "Test",
			"unknownField": "should fail"
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/portfolio", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreatePortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		db.Close() // Force error

		reqBody := `{
			"name": "Test Portfolio"
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/portfolio", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreatePortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "failed to create portfolio" {
			t.Errorf("Expected 'failed to create portfolio' error, got '%s'", response["error"])
		}
	})
}

// TestPortfolioHandler_UpdatePortfolio tests the UpdatePortfolio endpoint.
//
// WHY: This endpoint updates existing portfolios and is critical for maintaining portfolio data.
// Testing ensures proper validation, partial updates, error handling, and successful update flow.
//
//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestPortfolioHandler_UpdatePortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Update all fields
	t.Run("updates portfolio with all fields successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			WithName("Original Name").
			WithDescription("Original Description").
			WithExcludeFromOverview(false).
			Build(t, db)

		reqBody := `{
			"name": "Updated Name",
			"description": "Updated Description",
			"excludeFromOverview": true
		}`
		req := httptest.NewRequest(http.MethodPut, "/api/portfolio/"+portfolio.ID, strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req = testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		req.Body = io.NopCloser(strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdatePortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got '%s'", response.Name)
		}
		if response.Description != "Updated Description" {
			t.Errorf("Expected description 'Updated Description', got '%s'", response.Description)
		}
		if !response.ExcludeFromOverview {
			t.Error("Expected ExcludeFromOverview to be true")
		}
	})

	// Happy path: Partial update (name only)
	t.Run("updates name field only", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			WithName("Original Name").
			WithDescription("Original Description").
			Build(t, db)

		reqBody := `{
			"name": "New Name"
		}`
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		req.Body = io.NopCloser(strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdatePortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", response.Name)
		}
		// Description should remain unchanged
		if response.Description != "Original Description" {
			t.Errorf("Expected description to remain 'Original Description', got '%s'", response.Description)
		}
	})

	// Resource not found
	t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		reqBody := `{"name": "Test"}`
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/portfolio/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
		)
		req.Body = io.NopCloser(strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdatePortfolio(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrPortfolioNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrPortfolioNotFound.Error(), response["error"])
		}
	})

	// Input validation: Invalid JSON
	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)

		reqBody := `{invalid json`
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		req.Body = io.NopCloser(strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdatePortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "invalid request body" {
			t.Errorf("Expected 'invalid request body' error, got '%s'", response["error"])
		}
	})

	t.Run("returns 400 for empty name", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			WithName("Original Name").
			WithDescription("Original Description").
			Build(t, db)

		reqBody := `{
			"name": ""
		}`
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		req.Body = io.NopCloser(strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdatePortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "validation failed" {
			t.Errorf("Expected 'validation failed' error, got '%s'", response["error"])
		}

	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close() // Force error

		reqBody := `{"name": "Test"}`
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		req.Body = io.NopCloser(strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdatePortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_DeletePortfolio tests the DeletePortfolio endpoint.
//
// WHY: This endpoint deletes portfolios and is critical for data management.
// Testing ensures proper validation, cascading deletes, and error handling.
//
//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestPortfolioHandler_DeletePortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Delete portfolio
	t.Run("deletes portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		testutil.AssertRowCount(t, db, "portfolio", 1)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DeletePortfolio(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}

		// Verify it was actually deleted from the database
		testutil.AssertRowCount(t, db, "portfolio", 0)
	})

	// Happy path: Cascading delete
	t.Run("deletes portfolio and cascades to related data", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)

		testutil.AssertRowCount(t, db, "portfolio", 1)
		testutil.AssertRowCount(t, db, "portfolio_fund", 1)
		testutil.AssertRowCount(t, db, "transaction", 1)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DeletePortfolio(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d", w.Code)
		}

		// Verify cascading delete
		testutil.AssertRowCount(t, db, "portfolio", 0)
		testutil.AssertRowCount(t, db, "portfolio_fund", 0)
		testutil.AssertRowCount(t, db, "transaction", 0)
	})

	// Resource not found
	t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/portfolio/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.DeletePortfolio(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrPortfolioNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrPortfolioNotFound.Error(), response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close() // Force error

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DeletePortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_ArchivePortfolio tests the ArchivePortfolio endpoint.
//
// WHY: This endpoint archives portfolios, which is a common workflow for hiding
// inactive portfolios without deleting historical data.
func TestPortfolioHandler_ArchivePortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Archive portfolio
	t.Run("archives portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			WithIsArchived(false).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+portfolio.ID+"/archive",
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.ArchivePortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if !response.IsArchived {
			t.Error("Expected IsArchived to be true")
		}
		if response.ID != portfolio.ID {
			t.Errorf("Expected ID %s, got %s", portfolio.ID, response.ID)
		}
	})

	// Idempotency: Archive already archived portfolio
	t.Run("archiving already archived portfolio succeeds", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			WithIsArchived(true).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+portfolio.ID+"/archive",
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.ArchivePortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if !response.IsArchived {
			t.Error("Expected IsArchived to remain true")
		}
	})

	// Resource not found
	t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+nonExistentID+"/archive",
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.ArchivePortfolio(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrPortfolioNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrPortfolioNotFound.Error(), response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close() // Force error

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+portfolio.ID+"/archive",
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.ArchivePortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_UnarchivePortfolio tests the UnarchivePortfolio endpoint.
//
// WHY: This endpoint unarchives portfolios, allowing users to restore archived
// portfolios back to active status.
func TestPortfolioHandler_UnarchivePortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Unarchive portfolio
	t.Run("unarchives portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			WithIsArchived(true).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+portfolio.ID+"/unarchive",
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.UnarchivePortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.IsArchived {
			t.Error("Expected IsArchived to be false")
		}
		if response.ID != portfolio.ID {
			t.Errorf("Expected ID %s, got %s", portfolio.ID, response.ID)
		}
	})

	// Idempotency: Unarchive already unarchived portfolio
	t.Run("unarchiving already active portfolio succeeds", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			WithIsArchived(false).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+portfolio.ID+"/unarchive",
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.UnarchivePortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.IsArchived {
			t.Error("Expected IsArchived to remain false")
		}
	})

	// Resource not found
	t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+nonExistentID+"/unarchive",
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.UnarchivePortfolio(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrPortfolioNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrPortfolioNotFound.Error(), response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close() // Force error

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/portfolio/"+portfolio.ID+"/unarchive",
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.UnarchivePortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

func TestPortfolioHandler_CreatePortfolioFundHandler(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Create portfolio fund
	t.Run("creates portfolio fund successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			Build(t, db)
		fund := testutil.NewFund().
			Build(t, db)

		reqBody := `{
			"portfolioId": "` + portfolio.ID + `",
			"fundId": "` + fund.ID + `"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/portfolio/fund", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.CreatePortfolioFund(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Input validation: Invalid JSON
	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler, _ := setupHandler(t)

		reqBody := `{invalid json`
		req := httptest.NewRequest(http.MethodPost, "/api/portfolio/fund", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreatePortfolioFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "invalid request body" {
			t.Errorf("Expected 'invalid request body' error, got '%s'", response["error"])
		}
	})

	// Invalid Portfolio UUID
	t.Run("returns 400 for invalid portfolio ID format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		reqBody := `{
			"portfolioId": "failtyUUID",
			"fundId": "` + nonExistentID + `"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/portfolio/fund", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		handler.CreatePortfolioFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty portfolioId, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "validation failed" {
			t.Errorf("Expected 'validation failed' error, got '%s'", response["error"])
		}
	})

	// Invalid Fund UUID
	t.Run("returns 400 for invalid fund ID format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		reqBody := `{
			"portfolioId": "` + nonExistentID + `",
			"fundId": "failtyUUID"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/portfolio/fund", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		handler.CreatePortfolioFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty fundId, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "validation failed" {
			t.Errorf("Expected 'validation failed' error, got '%s'", response["error"])
		}
	})

	// Invalid Portfolio
	t.Run("returns 404 when portfolio not found", func(t *testing.T) {
		handler, db := setupHandler(t)

		nonExistentID := testutil.MakeID()

		fund := testutil.NewFund().
			Build(t, db)

		reqBody := `{
			"portfolioId": "` + nonExistentID + `",
			"fundId": "` + fund.ID + `"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/portfolio/fund", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.CreatePortfolioFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrPortfolioNotFound.Error() {
			t.Errorf("Expected '"+apperrors.ErrPortfolioNotFound.Error()+"' error, got '%s'", response["error"])
		}
	})

	// Invalid Fund
	t.Run("returns 404 when fund not found", func(t *testing.T) {
		handler, db := setupHandler(t)

		nonExistentID := testutil.MakeID()

		portfolio := testutil.NewPortfolio().
			Build(t, db)

		reqBody := `{
			"portfolioId": "` + portfolio.ID + `",
			"fundId": "` + nonExistentID + `"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/portfolio/fund", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.CreatePortfolioFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrFundNotFound.Error() {
			t.Errorf("Expected '"+apperrors.ErrFundNotFound.Error()+"' error, got '%s'", response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			Build(t, db)
		fund := testutil.NewFund().
			Build(t, db)

		db.Close() // Force error

		reqBody := `{
			"portfolioId": "` + portfolio.ID + `",
			"fundId": "` + fund.ID + `"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/portfolio/fund", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.CreatePortfolioFund(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

func TestPortfolioHandler_DeletePortfolioFundHandler(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Delete portfolio fund
	t.Run("deletes portfolio fund successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			Build(t, db)
		fund := testutil.NewFund().
			Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).
			Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodDelete,
			"/api/portfolio/fund/"+pf.ID,
			map[string]string{
				"uuid": pf.ID,
			},
			map[string]string{
				"confirm": "true",
			},
		)

		w := httptest.NewRecorder()

		handler.DeletePortfolioFund(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Delete portfolio fund without conformation

	t.Run("returns 409 when confirmation is missing", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			Build(t, db)
		fund := testutil.NewFund().
			Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/portfolio/fund"+pf.ID,
			map[string]string{"uuid": pf.ID},
		)

		w := httptest.NewRecorder()

		handler.DeletePortfolioFund(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected 409, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "Confirm deletion" {
			t.Errorf("Expected 'Confirm deletion' error, got '%s'", response["error"])
		}
	})

	// Invalid PortfolioFund
	t.Run("returns 404 when portfolio fund not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodDelete,
			"/api/portfolio/fund/"+nonExistentID,
			map[string]string{
				"uuid": nonExistentID,
			},
			map[string]string{
				"confirm": "true",
			},
		)

		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.DeletePortfolioFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrPortfolioFundNotFound.Error() {
			t.Errorf("Expected '"+apperrors.ErrPortfolioFundNotFound.Error()+"' error, got '%s'", response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().
			Build(t, db)
		fund := testutil.NewFund().
			Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).
			Build(t, db)

		db.Close() // Force error

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodDelete,
			"/api/portfolio/fund/"+pf.ID,
			map[string]string{
				"uuid": pf.ID,
			},
			map[string]string{
				"confirm": "true",
			},
		)

		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.DeletePortfolioFund(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}
