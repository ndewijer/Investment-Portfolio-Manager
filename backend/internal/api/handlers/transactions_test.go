package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

func TestTransactionHandler_AllTransactions(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("returns empty array when no transactions exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/transaction", nil)
		w := httptest.NewRecorder()

		handler.AllTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d transactions", len(response))
		}
	})

	t.Run("returns all transactions successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx1 := testutil.NewTransaction(pf.ID).Build(t, db)
		tx2 := testutil.NewTransaction(pf.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/transaction", nil)
		w := httptest.NewRecorder()

		handler.AllTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(response))
		}

		// Verify transaction IDs are present
		foundTransactions := make(map[string]bool)
		for _, tx := range response {
			foundTransactions[tx.ID] = true
		}

		if !foundTransactions[tx1.ID] {
			t.Error("Expected to find tx1 in response")
		}
		if !foundTransactions[tx2.ID] {
			t.Error("Expected to find tx2 in response")
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/transaction", nil)
		w := httptest.NewRecorder()

		handler.AllTransactions(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTransactionHandler_TransactionPerPortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("returns transactions for portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.TransactionPerPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(response))
		}
	})

	t.Run("returns empty array when portfolio has no transactions", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.TransactionPerPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d transactions", len(response))
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.TransactionPerPortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTransactionHandler_GetTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("returns transaction by ID successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ID != tx.ID {
			t.Errorf("Expected transaction ID %s, got %s", tx.ID, response.ID)
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTransactionHandler_CreateTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("creates transaction successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"date": "2024-01-15",
			"type": "buy",
			"shares": 50.0,
			"costPerShare": 100.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Transaction
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ID == "" {
			t.Error("Expected transaction ID to be set")
		}
		if response.PortfolioFundID != pf.ID {
			t.Errorf("Expected portfolioFundId %s, got %s", pf.ID, response.PortfolioFundID)
		}
		if response.Type != "buy" {
			t.Errorf("Expected type buy, got %s", response.Type)
		}
		if response.Shares != 50.0 {
			t.Errorf("Expected shares 50.0, got %f", response.Shares)
		}
		if response.CostPerShare != 100.50 {
			t.Errorf("Expected costPerShare 100.50, got %f", response.CostPerShare)
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", "invalid json")
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on missing required fields", func(t *testing.T) {
		handler, _ := setupHandler(t)

		body := `{
			"date": "2024-01-15",
			"type": "buy"
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid transaction type", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"date": "2024-01-15",
			"type": "invalid_type",
			"shares": 50.0,
			"costPerShare": 100.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid date format", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"date": "15-01-2024",
			"type": "buy",
			"shares": 50.0,
			"costPerShare": 100.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid portfolio fund ID", func(t *testing.T) {
		handler, _ := setupHandler(t)

		body := `{
			"portfolioFundId": "not-a-uuid",
			"date": "2024-01-15",
			"type": "buy",
			"shares": 50.0,
			"costPerShare": 100.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on insufficient shares for sell", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"date": "2024-01-15",
			"type": "sell",
			"shares": 100.0,
			"costPerShare": 10.0
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on non-existent portfolio fund", func(t *testing.T) {
		handler, _ := setupHandler(t)

		body := `{
			"portfolioFundId": "` + testutil.MakeID() + `",
			"date": "2024-01-15",
			"type": "buy",
			"shares": 50.0,
			"costPerShare": 100.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		db.Close()

		body := `{
			"portfolioFundId": "` + pf.ID + `",
			"date": "2024-01-15",
			"type": "buy",
			"shares": 50.0,
			"costPerShare": 100.50
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/transaction", body)
		w := httptest.NewRecorder()

		handler.CreateTransaction(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTransactionHandler_UpdateTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("updates transaction successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		body := `{
			"shares": 75.0,
			"costPerShare": 150.75
		}`

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Transaction
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ID != tx.ID {
			t.Errorf("Expected transaction ID %s, got %s", tx.ID, response.ID)
		}
		if response.Shares != 75.0 {
			t.Errorf("Expected shares 75.0, got %f", response.Shares)
		}
		if response.CostPerShare != 150.75 {
			t.Errorf("Expected costPerShare 150.75, got %f", response.CostPerShare)
		}
	})

	t.Run("updates transaction type successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		// Create a buy transaction first so there are shares available to sell
		testutil.NewTransaction(pf.ID).WithShares(200).Build(t, db)
		// This is the transaction we'll change to sell
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		body := `{
			"type": "sell"
		}`

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Transaction
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Type != "sell" {
			t.Errorf("Expected type sell, got %s", response.Type)
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		body := `{
			"shares": 75.0
		}`

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
			body,
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
			"invalid json",
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid transaction type", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		body := `{
			"type": "invalid_type"
		}`

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid date format", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		body := `{
			"date": "15-01-2024"
		}`

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on insufficient shares when changing to sell", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		// Buy 10 shares, then try to change a 50-share buy to sell
		testutil.NewTransaction(pf.ID).WithShares(10).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).WithShares(50).Build(t, db)

		body := `{
			"type": "sell"
		}`

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		db.Close()

		body := `{
			"shares": 75.0
		}`

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.UpdateTransaction(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTransactionHandler_DeleteTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("deletes transaction successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteTransaction(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}

		// Verify transaction is deleted
		req2 := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
		)
		w2 := httptest.NewRecorder()

		handler.GetTransaction(w2, req2)

		if w2.Code != http.StatusNotFound {
			t.Errorf("Expected transaction to be deleted, but got status %d", w2.Code)
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/transaction/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.DeleteTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/transaction/"+tx.ID,
			map[string]string{"uuid": tx.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteTransaction(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}
