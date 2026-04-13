package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

func TestDividendHandler_GetAllDividends(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	t.Run("returns empty array when no dividends exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/dividend", nil)
		w := httptest.NewRecorder()

		handler.GetAllDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.Dividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d dividends", len(response))
		}
	})

	t.Run("returns all dividends successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		dividend1 := testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(1.00).
			WithSharesOwned(100).
			Build(t, db)
		dividend2 := testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(0.50).
			WithSharesOwned(100).
			Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/dividend", nil)
		w := httptest.NewRecorder()

		handler.GetAllDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.Dividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 dividends, got %d", len(response))
		}

		// Verify dividend IDs are present
		foundDividends := make(map[string]bool)
		for _, d := range response {
			foundDividends[d.ID] = true
		}

		if !foundDividends[dividend1.ID] {
			t.Error("Expected to find dividend1 in response")
		}
		if !foundDividends[dividend2.ID] {
			t.Error("Expected to find dividend2 in response")
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/dividend", nil)
		w := httptest.NewRecorder()

		handler.GetAllDividend(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDividendHandler_GetDividend(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	t.Run("returns dividend by ID successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		div := testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(1.50).
			WithSharesOwned(200).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/"+div.ID,
			map[string]string{"uuid": div.ID},
		)
		w := httptest.NewRecorder()

		handler.GetDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Dividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ID != div.ID {
			t.Errorf("Expected ID %s, got %s", div.ID, response.ID)
		}
		if response.DividendPerShare != 1.50 {
			t.Errorf("Expected dividendPerShare 1.50, got %f", response.DividendPerShare)
		}
	})

	t.Run("returns 404 for non-existent dividend", func(t *testing.T) {
		handler, _ := setupHandler(t)

		id := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/"+id,
			map[string]string{"uuid": id},
		)
		w := httptest.NewRecorder()

		handler.GetDividend(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		id := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/"+id,
			map[string]string{"uuid": id},
		)
		w := httptest.NewRecorder()

		handler.GetDividend(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive handler test with multiple subtests and assertions, cannot be split well.
func TestDividendHandler_CreateDividend(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	// Fixed dates: transaction in Jan, ex-dividend in Feb, buy order after ex-dividend.
	txDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	recDate := "2026-02-01"
	exDate := "2026-02-10"
	buyDate := "2026-02-15"

	t.Run("creates dividend for STOCK fund with no buy order sets PENDING", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 0.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ID == "" {
			t.Error("Expected non-empty ID")
		}
		if response.SharesOwned != 100 {
			t.Errorf("Expected 100 shares owned, got %f", response.SharesOwned)
		}
		if response.TotalAmount != 50.0 {
			t.Errorf("Expected total amount 50.0, got %f", response.TotalAmount)
		}
		if response.ReinvestmentStatus != "PENDING" {
			t.Errorf("Expected PENDING status, got %s", response.ReinvestmentStatus)
		}
		if response.DividendType != "STOCK" {
			t.Errorf("Expected DividendType STOCK, got %s", response.DividendType)
		}
	})

	t.Run("creates dividend for non-STOCK fund with no buy order sets COMPLETED", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("CASH").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(50).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 1.00
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("Expected COMPLETED status, got %s", response.ReinvestmentStatus)
		}
		if response.DividendType != "CASH" {
			t.Errorf("Expected DividendType CASH, got %s", response.DividendType)
		}
	})

	t.Run("creates dividend with buy order but missing reinvestment info sets PENDING", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 0.50,
			"buyOrderDate": "` + buyDate + `"
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ReinvestmentStatus != "PENDING" {
			t.Errorf("Expected PENDING, got %s", response.ReinvestmentStatus)
		}
		if response.ReinvestmentTransactionID != "" {
			t.Errorf("Expected no reinvestment transaction, got %s", response.ReinvestmentTransactionID)
		}
	})

	t.Run("creates STOCK dividend with full reinvestment sets COMPLETED", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		// 100 shares × 0.50 = 50.0 total dividend; 10 shares × 5.0 = 50.0 reinvested → COMPLETED
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 0.50,
			"buyOrderDate": "` + buyDate + `",
			"reinvestmentShares": 10.0,
			"reinvestmentPrice": 5.0
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("Expected COMPLETED, got %s", response.ReinvestmentStatus)
		}
		if response.ReinvestmentTransactionID == "" {
			t.Error("Expected reinvestment transaction ID to be set")
		}
	})

	t.Run("creates STOCK dividend with partial reinvestment sets PARTIAL", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		// 100 shares × 0.50 = 50.0 total dividend; 5 shares × 5.0 = 25.0 reinvested → PARTIAL
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 0.50,
			"buyOrderDate": "` + buyDate + `",
			"reinvestmentShares": 5.0,
			"reinvestmentPrice": 5.0
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ReinvestmentStatus != "PARTIAL" {
			t.Errorf("Expected PARTIAL, got %s", response.ReinvestmentStatus)
		}
	})

	t.Run("returns 400 for invalid JSON body", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", `{invalid json}`)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for validation failure", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", `{"portfolioFundId": "not-a-uuid"}`)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 for fund with DividendType None", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("None").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 0.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		body := `{
			"portfolioFundId": "` + testutil.MakeID() + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 0.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		w := httptest.NewRecorder()
		handler.CreateDividend(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive handler test with multiple subtests and assertions, cannot be split well.
func TestDividendHandler_UpdateDividend(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	// txDate is well before the default ex-dividend date (now-5d) so GetSharesOnDate picks it up.
	txDate := time.Now().UTC().AddDate(0, 0, -30)
	buyDate := time.Now().UTC().AddDate(0, 0, -3).Format("2006-01-02")

	t.Run("updates dividendPerShare and recalculates totalAmount", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)
		div := testutil.NewDividend(fund.ID, pf.ID).WithDividendPerShare(0.50).Build(t, db)

		body := `{"dividendPerShare": 1.00}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+div.ID,
			map[string]string{"uuid": div.ID},
			body,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.DividendPerShare != 1.00 {
			t.Errorf("Expected dividendPerShare 1.00, got %f", response.DividendPerShare)
		}
		if response.TotalAmount != 100.0 {
			t.Errorf("Expected totalAmount 100.0 (100 shares × 1.00), got %f", response.TotalAmount)
		}
	})

	t.Run("adds full reinvestment to PENDING STOCK dividend sets COMPLETED", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)
		// 100 shares × 0.50 = 50.0 total; 10 shares × 5.0 = 50.0 reinvested → COMPLETED
		div := testutil.NewDividend(fund.ID, pf.ID).WithDividendPerShare(0.50).Build(t, db)

		body := `{"buyOrderDate": "` + buyDate + `", "reinvestmentShares": 10.0, "reinvestmentPrice": 5.0}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+div.ID,
			map[string]string{"uuid": div.ID},
			body,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ReinvestmentStatus != "COMPLETED" {
			t.Errorf("Expected COMPLETED, got %s", response.ReinvestmentStatus)
		}
		if response.ReinvestmentTransactionID == "" {
			t.Error("Expected reinvestment transaction ID to be set")
		}
	})

	t.Run("adds partial reinvestment to PENDING STOCK dividend sets PARTIAL", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)
		// 100 shares × 0.50 = 50.0 total; 5 shares × 5.0 = 25.0 reinvested → PARTIAL
		div := testutil.NewDividend(fund.ID, pf.ID).WithDividendPerShare(0.50).Build(t, db)

		body := `{"buyOrderDate": "` + buyDate + `", "reinvestmentShares": 5.0, "reinvestmentPrice": 5.0}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+div.ID,
			map[string]string{"uuid": div.ID},
			body,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ReinvestmentStatus != "PARTIAL" {
			t.Errorf("Expected PARTIAL, got %s", response.ReinvestmentStatus)
		}
	})

	t.Run("buy order only with no reinvestment info keeps PENDING", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)
		div := testutil.NewDividend(fund.ID, pf.ID).WithDividendPerShare(0.50).Build(t, db)

		body := `{"buyOrderDate": "` + buyDate + `"}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+div.ID,
			map[string]string{"uuid": div.ID},
			body,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ReinvestmentStatus != "PENDING" {
			t.Errorf("Expected PENDING, got %s", response.ReinvestmentStatus)
		}
		if response.ReinvestmentTransactionID != "" {
			t.Errorf("Expected no reinvestment transaction, got %s", response.ReinvestmentTransactionID)
		}
	})

	t.Run("returns 404 for non-existent dividend", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+testutil.MakeID(),
			map[string]string{"uuid": testutil.MakeID()},
			`{"dividendPerShare": 1.00}`,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for invalid JSON body", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+testutil.MakeID(),
			map[string]string{"uuid": testutil.MakeID()},
			`{invalid json}`,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for validation failure", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+testutil.MakeID(),
			map[string]string{"uuid": testutil.MakeID()},
			`{"dividendPerShare": -1.0}`,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut, "/api/dividend/"+testutil.MakeID(),
			map[string]string{"uuid": testutil.MakeID()},
			`{"dividendPerShare": 1.00}`,
		)
		w := httptest.NewRecorder()
		handler.UpdateDividend(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDividendHandler_DividendPerPortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	t.Run("returns dividends for portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(1.00).
			WithSharesOwned(100).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 dividend, got %d", len(response))
		}
	})

	t.Run("returns 404 when portfolio has no portfolio_fund associations", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerPortfolio(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when portfolio exists but has no dividends", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		// portfolio_fund exists but no dividend records

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerPortfolio(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerPortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDividendHandler_DividendPerFund(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	t.Run("returns dividends for fund successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(1.00).
			WithSharesOwned(50).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerFund(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 dividend, got %d", len(response))
		}
		if response[0].FundID != fund.ID {
			t.Errorf("Expected fundId %s, got %s", fund.ID, response[0].FundID)
		}
	})

	t.Run("returns 404 when fund has no portfolio_fund associations", func(t *testing.T) {
		handler, db := setupHandler(t)

		fund := testutil.NewFund().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when fund exists but has no dividends", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		// portfolio_fund exists but no dividend records

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		fund := testutil.NewFund().Build(t, db)
		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerFund(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDividendHandler_DeleteDividend(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	txDate := time.Now().UTC().AddDate(0, 0, -30)

	t.Run("deletes dividend successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		div := testutil.NewDividend(fund.ID, pf.ID).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/dividend/"+div.ID,
			map[string]string{"uuid": div.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteDividend(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("deletes dividend with reinvestment transaction successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("STOCK").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).WithDate(txDate).WithShares(100).Build(t, db)

		// Create a dividend with full reinvestment so it has a reinvestmentTransactionId
		recDate := time.Now().UTC().AddDate(0, 0, -20).Format("2006-01-02")
		exDate := time.Now().UTC().AddDate(0, 0, -15).Format("2006-01-02")
		buyDate := time.Now().UTC().AddDate(0, 0, -10).Format("2006-01-02")
		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"recordDate": "` + recDate + `",
			"exDividendDate": "` + exDate + `",
			"dividendPerShare": 0.50,
			"buyOrderDate": "` + buyDate + `",
			"reinvestmentShares": 10.0,
			"reinvestmentPrice": 5.0
		}`
		createReq := testutil.NewRequestWithBody(http.MethodPost, "/api/dividend", body)
		createW := httptest.NewRecorder()
		handler.CreateDividend(createW, createReq)
		if createW.Code != http.StatusCreated {
			t.Fatalf("Setup failed: expected 201, got %d: %s", createW.Code, createW.Body.String())
		}
		var created model.DividendFund
		//nolint:errcheck // Test setup
		json.NewDecoder(createW.Body).Decode(&created)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/dividend/"+created.ID,
			map[string]string{"uuid": created.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteDividend(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 for non-existent dividend", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/dividend/"+testutil.MakeID(),
			map[string]string{"uuid": testutil.MakeID()},
		)
		w := httptest.NewRecorder()

		handler.DeleteDividend(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/dividend/"+testutil.MakeID(),
			map[string]string{"uuid": testutil.MakeID()},
		)
		w := httptest.NewRecorder()

		handler.DeleteDividend(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}
