package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/handlers"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

func TestFundHandler_GetFundPrices(t *testing.T) {
	t.Run("returns fund price history", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/fund-prices/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFundPrices(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.FundPrice
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Fund without prices should return empty array
		if len(response) != 0 {
			t.Errorf("Expected empty price array, got %d prices", len(response))
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		db.Close() // Force database error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/fund-prices/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFundPrices(w, req)

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

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestFundHandler_UpdateFundPrice_Today(t *testing.T) {
	t.Run("successfully updates today's price when price doesn't exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund with symbol
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "today"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response model.FundPriceUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v. Body: %s", err, w.Body.String())
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		if !response.NewPrices {
			t.Error("Expected new_prices to be true since price was inserted")
		}

		if mockYahoo.QueryCount != 1 {
			t.Errorf("Expected 1 Yahoo API call, got %d", mockYahoo.QueryCount)
		}
	})

	t.Run("returns success with new_prices=false when price already exists", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund with symbol
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)

		// Insert yesterday's price
		now := time.Now().UTC()
		yesterday := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
		testutil.NewFundPrice(fund.ID).
			WithDate(yesterday).
			WithPrice(100.0).
			Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "today"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response model.FundPriceUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		if response.NewPrices {
			t.Error("Expected new_prices to be false since price already existed")
		}

		if mockYahoo.QueryCount != 0 {
			t.Errorf("Expected 0 Yahoo API calls since price exists, got %d", mockYahoo.QueryCount)
		}
	})

	t.Run("returns error when fund has no symbol", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund WITHOUT symbol
		fund := testutil.NewFund().WithSymbol("").Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "today"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if response["error"] != "cannot update current fund price" {
			t.Errorf("Expected 'cannot update current fund price', got '%s'", response["error"])
		}
	})

	t.Run("returns 404 when fund does not exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/nonexistent/update",
			map[string]string{"uuid": "nonexistent"},
			map[string]string{"type": "today"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if response["error"] != apperrors.ErrFundNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrFundNotFound.Error(), response["error"])
		}
	})

	t.Run("returns error when Yahoo API fails", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient().WithError(fmt.Errorf("yahoo api error"))
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "today"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("returns error when Yahoo returns no data", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient().WithEmptyResponse()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "today"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("returns bad request when type parameter is invalid", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "invalid"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns bad request when type parameter is missing", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestFundHandler_UpdateFundPrice_Historical(t *testing.T) {
	t.Run("successfully updates historical prices when missing dates exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		// Create mock that returns 5 days of data
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund and portfolio
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create a transaction 7 days ago
		now := time.Now().UTC()
		sevenDaysAgo := now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		testutil.NewTransaction(pf.ID).
			WithDate(sevenDaysAgo).
			Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "historical"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response model.FundPriceUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		if !response.NewPrices {
			t.Error("Expected new_prices to be true since prices were inserted")
		}

		if mockYahoo.QueryCount != 1 {
			t.Errorf("Expected 1 Yahoo API call, got %d", mockYahoo.QueryCount)
		}
	})

	t.Run("returns success with new_prices=false when no missing dates", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund and portfolio
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create a transaction 3 days ago
		now := time.Now().UTC()
		threeDaysAgo := now.AddDate(0, 0, -3).Truncate(24 * time.Hour)
		testutil.NewTransaction(pf.ID).
			WithDate(threeDaysAgo).
			Build(t, db)

		// Insert all prices from transaction date to yesterday
		yesterday := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
		for d := threeDaysAgo; !d.After(yesterday); d = d.AddDate(0, 0, 1) {
			testutil.NewFundPrice(fund.ID).
				WithDate(d).
				WithPrice(100.0).
				Build(t, db)
		}

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "historical"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response model.FundPriceUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		if response.NewPrices {
			t.Error("Expected new_prices to be false since all prices exist")
		}

		if mockYahoo.QueryCount != 0 {
			t.Errorf("Expected 0 Yahoo API calls, got %d", mockYahoo.QueryCount)
		}
	})

	t.Run("returns error when fund has no transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund and portfolio WITHOUT transactions
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "historical"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("returns error when fund has no portfolio funds", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund WITHOUT portfolio_fund relationship
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "historical"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("handles partial Yahoo data gracefully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)

		// Mock will return 5 days but we need 7 days
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Transaction 10 days ago but Yahoo only returns 5 days
		now := time.Now().UTC()
		tenDaysAgo := now.AddDate(0, 0, -10).Truncate(24 * time.Hour)
		testutil.NewTransaction(pf.ID).
			WithDate(tenDaysAgo).
			Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodPost,
			"/api/fund/fund-prices/"+fund.ID+"/update",
			map[string]string{"uuid": fund.ID},
			map[string]string{"type": "historical"},
		)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response model.FundPriceUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should still succeed with whatever data was available
		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		// Should have added the prices that were available (5 days)
		if !response.NewPrices {
			t.Error("Expected new_prices to be true since some prices were added")
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestFundHandler_UpdateAllFundHistory(t *testing.T) {
	t.Run("successfully updates all funds with full success", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create portfolio and funds with transactions
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().WithSymbol("AAPL").WithName("Apple").Build(t, db)
		fund2 := testutil.NewFund().WithSymbol("GOOGL").WithName("Google").Build(t, db)

		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		now := time.Now().UTC()
		fiveDaysAgo := now.AddDate(0, 0, -5).Truncate(24 * time.Hour)
		testutil.NewTransaction(pf1.ID).WithDate(fiveDaysAgo).Build(t, db)
		testutil.NewTransaction(pf2.ID).WithDate(fiveDaysAgo).Build(t, db)

		req := httptest.NewRequest(http.MethodPost, "/api/fund/update-all-prices", nil)
		w := httptest.NewRecorder()

		handler.UpdateAllFundHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.AllFundUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}

		if response.TotalUpdated != 2 {
			t.Errorf("Expected 2 funds updated, got %d", response.TotalUpdated)
		}

		if response.TotalErrors != 0 {
			t.Errorf("Expected 0 errors, got %d", response.TotalErrors)
		}

		if len(response.UpdatedFunds) != 2 {
			t.Errorf("Expected 2 updated funds, got %d", len(response.UpdatedFunds))
		}

		if len(response.Errors) != 0 {
			t.Errorf("Expected 0 errors, got %d", len(response.Errors))
		}

		// Verify fund details are present
		for _, uf := range response.UpdatedFunds {
			if uf.FundID == "" {
				t.Error("Expected fund ID to be populated")
			}
			if uf.Name == "" {
				t.Error("Expected fund name to be populated")
			}
			if uf.Symbol == "" {
				t.Error("Expected fund symbol to be populated")
			}
			if uf.PricesAdded <= 0 {
				t.Error("Expected prices to be added")
			}
		}
	})

	t.Run("handles partial success with some funds failing", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create funds: one with symbol (will succeed) and one without (will fail)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fundSuccess := testutil.NewFund().WithSymbol("AAPL").WithName("Apple").Build(t, db)
		fundFail := testutil.NewFund().WithSymbol("").WithName("NoSymbol").Build(t, db)

		pfSuccess := testutil.NewPortfolioFund(portfolio.ID, fundSuccess.ID).Build(t, db)
		pfFail := testutil.NewPortfolioFund(portfolio.ID, fundFail.ID).Build(t, db)

		now := time.Now().UTC()
		fiveDaysAgo := now.AddDate(0, 0, -5).Truncate(24 * time.Hour)
		testutil.NewTransaction(pfSuccess.ID).WithDate(fiveDaysAgo).Build(t, db)
		testutil.NewTransaction(pfFail.ID).WithDate(fiveDaysAgo).Build(t, db)

		req := httptest.NewRequest(http.MethodPost, "/api/fund/update-all-prices", nil)
		w := httptest.NewRecorder()

		handler.UpdateAllFundHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.AllFundUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true for partial success")
		}

		if response.TotalUpdated != 1 {
			t.Errorf("Expected 1 fund updated, got %d", response.TotalUpdated)
		}

		if response.TotalErrors != 1 {
			t.Errorf("Expected 1 error, got %d", response.TotalErrors)
		}

		if len(response.UpdatedFunds) != 1 {
			t.Errorf("Expected 1 updated fund, got %d", len(response.UpdatedFunds))
		}

		if len(response.Errors) != 1 {
			t.Errorf("Expected 1 error entry, got %d", len(response.Errors))
		}

		// Verify error details
		if len(response.Errors) > 0 {
			errEntry := response.Errors[0]
			if errEntry.FundID == "" {
				t.Error("Expected error fund ID to be populated")
			}
			if errEntry.Error == "" {
				t.Error("Expected error message to be populated")
			}
		}
	})

	t.Run("returns 500 with details when all funds fail", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create funds without symbols or transactions (will fail)
		fund1 := testutil.NewFund().WithSymbol("").WithName("Fund1").Build(t, db)
		fund2 := testutil.NewFund().WithSymbol("").WithName("Fund2").Build(t, db)

		// Create portfolio associations but no transactions
		portfolio := testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodPost, "/api/fund/update-all-prices", nil)
		w := httptest.NewRecorder()

		handler.UpdateAllFundHistory(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		var response model.AllFundUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Success {
			t.Error("Expected success to be false for total failure")
		}

		if response.TotalUpdated != 0 {
			t.Errorf("Expected 0 funds updated, got %d", response.TotalUpdated)
		}

		if response.TotalErrors != 2 {
			t.Errorf("Expected 2 errors, got %d", response.TotalErrors)
		}

		if len(response.Errors) != 2 {
			t.Errorf("Expected 2 error entries, got %d", len(response.Errors))
		}
	})

	t.Run("returns 404 when no funds exist in database", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		req := httptest.NewRequest(http.MethodPost, "/api/fund/update-all-prices", nil)
		w := httptest.NewRecorder()

		handler.UpdateAllFundHistory(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if response["error"] != apperrors.ErrFundNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrFundNotFound.Error(), response["error"])
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create a fund first to ensure GetAllFunds would have data
		testutil.NewFund().Build(t, db)

		db.Close() // Force database error

		req := httptest.NewRequest(http.MethodPost, "/api/fund/update-all-prices", nil)
		w := httptest.NewRecorder()

		handler.UpdateAllFundHistory(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		// Database errors return structured AllFundUpdateResponse with empty data
		var response model.AllFundUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if response.Success {
			t.Error("Expected success to be false on database error")
		}
	})

	t.Run("returns success when no prices need updating", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mockYahoo := testutil.NewMockYahooClient()
		fs := testutil.NewTestFundServiceWithMockYahoo(t, db, mockYahoo)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		// Create fund with transaction and all prices already populated
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		now := time.Now().UTC()
		threeDaysAgo := now.AddDate(0, 0, -3).Truncate(24 * time.Hour)
		testutil.NewTransaction(pf.ID).WithDate(threeDaysAgo).Build(t, db)

		// Populate all prices from transaction date to yesterday
		yesterday := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
		for d := threeDaysAgo; !d.After(yesterday); d = d.AddDate(0, 0, 1) {
			testutil.NewFundPrice(fund.ID).
				WithDate(d).
				WithPrice(100.0).
				Build(t, db)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/fund/update-all-prices", nil)
		w := httptest.NewRecorder()

		handler.UpdateAllFundHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response model.AllFundUpdateResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}

		if response.TotalUpdated != 1 {
			t.Errorf("Expected 1 fund processed, got %d", response.TotalUpdated)
		}

		// Should have 0 prices added since all exist
		if len(response.UpdatedFunds) > 0 && response.UpdatedFunds[0].PricesAdded != 0 {
			t.Errorf("Expected 0 prices added, got %d", response.UpdatedFunds[0].PricesAdded)
		}

		// Should not call Yahoo API since all prices exist
		if mockYahoo.QueryCount != 0 {
			t.Errorf("Expected 0 Yahoo API calls, got %d", mockYahoo.QueryCount)
		}
	})
}
