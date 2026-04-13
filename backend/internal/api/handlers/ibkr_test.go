package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fernet/fernet-go"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/ibkr"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// mockIbkrClient is a configurable test double for ibkr.Client.
// Set testConnectionFn to control the outcome of TestIbkrConnection.
// RetreiveIbkrFlexReport always returns empty results and is not exercised
// by TestIbkrConnection tests.
type mockIbkrClient struct {
	testConnectionFn func(ctx context.Context, token, queryID string) (bool, error)
}

func (m *mockIbkrClient) RetreiveIbkrFlexReport(_ context.Context, _, _ string) (ibkr.FlexQueryResponse, []byte, error) {
	return ibkr.FlexQueryResponse{}, nil, nil
}

func (m *mockIbkrClient) TestIbkrConnection(ctx context.Context, token, queryID string) (bool, error) {
	if m.testConnectionFn != nil {
		return m.testConnectionFn(ctx, token, queryID)
	}
	return true, nil
}

func TestIbkrHandler_GetConfig(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns config successfully when configured", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Insert IBKR config with token expiration
		configID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, token_expires_at, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, datetime('now', '+90 days'), ?, ?, ?, datetime('now'), datetime('now'))
		`, configID, "test_token", "123456", true, true, false)
		if err != nil {
			t.Fatalf("Failed to insert test config: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/config", nil)
		w := httptest.NewRecorder()

		handler.GetConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IbkrConfig
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if !response.Configured {
			t.Error("Expected configured to be true")
		}

		if response.FlexQueryID != "123456" {
			t.Errorf("Expected flex query ID '123456', got '%s'", response.FlexQueryID)
		}
	})

	// The service checks config.TokenExpiresAt.IsZero() without checking if TokenExpiresAt is nil first
	// This causes a panic when the config is unconfigured (TokenExpiresAt is nil)
	t.Run("returns unconfigured status when no config exists", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/config", nil)
		w := httptest.NewRecorder()

		handler.GetConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IbkrConfig
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Configured {
			t.Error("Expected configured to be false")
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/config", nil)
		w := httptest.NewRecorder()

		handler.GetConfig(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_GetActivePortfolios(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns empty array when no active portfolios exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/portfolios", nil)
		w := httptest.NewRecorder()

		handler.GetActivePortfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d portfolios", len(response))
		}
	})

	t.Run("returns active portfolios successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio1 := testutil.NewPortfolio().Build(t, db)
		portfolio2 := testutil.NewPortfolio().Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/portfolios", nil)
		w := httptest.NewRecorder()

		handler.GetActivePortfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 portfolios, got %d", len(response))
		}

		// Verify portfolio IDs are present
		foundPortfolios := make(map[string]bool)
		for _, p := range response {
			foundPortfolios[p.ID] = true
		}

		if !foundPortfolios[portfolio1.ID] {
			t.Error("Expected to find portfolio1 in response")
		}
		if !foundPortfolios[portfolio2.ID] {
			t.Error("Expected to find portfolio2 in response")
		}
	})

	t.Run("excludes archived portfolios", func(t *testing.T) {
		handler, db := setupHandler(t)

		testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolio().Archived().Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/portfolios", nil)
		w := httptest.NewRecorder()

		handler.GetActivePortfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.Portfolio
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 active portfolio, got %d", len(response))
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/portfolios", nil)
		w := httptest.NewRecorder()

		handler.GetActivePortfolios(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_GetPendingDividends(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns empty array when no pending dividends exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/dividend/pending", nil)
		w := httptest.NewRecorder()

		handler.GetPendingDividends(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PendingDividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d dividends", len(response))
		}
	})

	t.Run("returns pending dividends successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		dividend1 := testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(1.50).
			WithSharesOwned(100).
			Build(t, db)
		dividend2 := testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(2.00).
			WithSharesOwned(200).
			Build(t, db)

		// Update reinvestment status to PENDING
		_, err := db.Exec(`UPDATE dividend SET reinvestment_status = 'PENDING' WHERE id IN (?, ?)`, dividend1.ID, dividend2.ID)
		if err != nil {
			t.Fatalf("Failed to update dividend status: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/dividend/pending", nil)
		w := httptest.NewRecorder()

		handler.GetPendingDividends(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PendingDividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 pending dividends, got %d", len(response))
		}
	})

	t.Run("filters by symbol successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		fund2 := testutil.NewFund().WithSymbol("GOOGL").Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		dividend1 := testutil.NewDividend(fund1.ID, pf1.ID).Build(t, db)
		dividend2 := testutil.NewDividend(fund2.ID, pf2.ID).Build(t, db)

		// Update reinvestment status to PENDING
		_, err := db.Exec(`UPDATE dividend SET reinvestment_status = 'PENDING' WHERE id IN (?, ?)`, dividend1.ID, dividend2.ID)
		if err != nil {
			t.Fatalf("Failed to update dividend status: %v", err)
		}

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/ibkr/dividend/pending",
			map[string]string{"symbol": "AAPL"},
		)
		w := httptest.NewRecorder()

		handler.GetPendingDividends(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PendingDividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 pending dividend for AAPL, got %d", len(response))
		}

		if response[0].ID != dividend1.ID {
			t.Error("Expected to find dividend1 for AAPL")
		}
	})

	t.Run("filters by isin successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		dividend := testutil.NewDividend(fund.ID, pf.ID).Build(t, db)

		// Update reinvestment status to PENDING
		_, err := db.Exec(`UPDATE dividend SET reinvestment_status = 'PENDING' WHERE id = ?`, dividend.ID)
		if err != nil {
			t.Fatalf("Failed to update dividend status: %v", err)
		}

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/ibkr/dividend/pending",
			map[string]string{"isin": "US0378331005"},
		)
		w := httptest.NewRecorder()

		handler.GetPendingDividends(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PendingDividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 pending dividend for ISIN, got %d", len(response))
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/dividend/pending", nil)
		w := httptest.NewRecorder()

		handler.GetPendingDividends(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestIbkrHandler_GetInbox(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns empty array when no inbox transactions exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/inbox", nil)
		w := httptest.NewRecorder()

		handler.GetInbox(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.IBKRTransaction
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d transactions", len(response))
		}
	})

	t.Run("returns inbox transactions successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Insert IBKR transactions
		id1 := testutil.MakeID()
		id2 := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin,
				description, transaction_type, quantity, price, total_amount,
				currency, fees, status, imported_at, report_date, notes
			) VALUES
			(?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Buy', 'trade', 10, 150.00, 1500.00, 'USD', 1.00, 'pending', datetime('now'), date('now'), ''),
			(?, ?, datetime('now'), 'GOOGL', 'US02079K3059', 'Buy', 'trade', 5, 2800.00, 14000.00, 'USD', 2.00, 'pending', datetime('now'), date('now'), '')
		`, id1, "IBKR123", id2, "IBKR456")
		if err != nil {
			t.Fatalf("Failed to insert test transactions: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/inbox", nil)
		w := httptest.NewRecorder()

		handler.GetInbox(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.IBKRTransaction
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 inbox transactions, got %d", len(response))
		}
	})

	t.Run("filters by status successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		id1 := testutil.MakeID()
		id2 := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES
			(?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Buy AAPL', 'trade', 10, 150.00, 1500.00, 'USD', 1.00, 'pending', datetime('now'), date('now'), ''),
			(?, ?, datetime('now'), 'GOOGL', 'US02079K3059', 'Buy GOOGL', 'trade', 5, 400.00, 2000.00, 'USD', 2.00, 'allocated', datetime('now'), date('now'), '')
		`, id1, "IBKR123", id2, "IBKR456")
		if err != nil {
			t.Fatalf("Failed to insert test transactions: %v", err)
		}

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/ibkr/inbox",
			map[string]string{"status": "pending"},
		)
		w := httptest.NewRecorder()

		handler.GetInbox(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.IBKRTransaction
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 pending transaction, got %d", len(response))
		}

		if response[0].Status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", response[0].Status)
		}
	})

	t.Run("filters by transaction type successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		id1 := testutil.MakeID()
		id2 := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES
			(?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Buy AAPL', 'trade', 10, 150.00, 1500.00, 'USD', 1.00, 'pending', datetime('now'), date('now'), ''),
			(?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Dividend AAPL', 'dividend', 0, 0, 50.00, 'USD', 0.00, 'pending', datetime('now'), date('now'), '')
		`, id1, "IBKR123", id2, "IBKR456")
		if err != nil {
			t.Fatalf("Failed to insert test transactions: %v", err)
		}

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/ibkr/inbox",
			map[string]string{"transactionType": "dividend"},
		)
		w := httptest.NewRecorder()

		handler.GetInbox(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.IBKRTransaction
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 dividend transaction, got %d", len(response))
		}

		if response[0].TransactionType != "dividend" {
			t.Errorf("Expected transaction type 'dividend', got '%s'", response[0].TransactionType)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/inbox", nil)
		w := httptest.NewRecorder()

		handler.GetInbox(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_GetInboxCount(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns zero count when no inbox transactions exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/inbox/count", nil)
		w := httptest.NewRecorder()

		handler.GetInboxCount(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKRInboxCount
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Count != 0 {
			t.Errorf("Expected count 0, got %d", response.Count)
		}
	})

	t.Run("returns inbox count successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Insert 3 pending IBKR transactions
		id1 := testutil.MakeID()
		id2 := testutil.MakeID()
		id3 := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES
			(?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Buy AAPL', 'trade', 10, 150.00, 1500.00, 'USD', 1.00, 'pending', datetime('now'), date('now'), ''),
			(?, ?, datetime('now'), 'GOOGL', 'US02079K3059', 'Buy GOOGL', 'trade', 5, 400.00, 2000.00, 'USD', 2.00, 'pending', datetime('now'), date('now'), ''),
			(?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Dividend AAPL', 'dividend', 0, 0, 50.00, 'USD', 0.00, 'pending', datetime('now'), date('now'), '')
		`, id1, "IBKR123", id2, "IBKR456", id3, "IBKR789")
		if err != nil {
			t.Fatalf("Failed to insert test transactions: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/inbox/count", nil)
		w := httptest.NewRecorder()

		handler.GetInboxCount(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKRInboxCount
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.Count != 3 {
			t.Errorf("Expected count 3, got %d", response.Count)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/ibkr/inbox/count", nil)
		w := httptest.NewRecorder()

		handler.GetInboxCount(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_GetTransactionAllocations(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns transaction allocations successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create IBKR transaction
		transactionID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES (?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Buy AAPL', 'trade', 10, 150.00, 1500.00, 'USD', 1.00, 'allocated', datetime('now'), date('now'), '')
		`, transactionID, "IBKR123")
		if err != nil {
			t.Fatalf("Failed to insert test transaction: %v", err)
		}

		// Create portfolio and allocation
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		allocationID := testutil.MakeID()
		_, err = db.Exec(`
			INSERT INTO ibkr_transaction_allocation (
				id, ibkr_transaction_id, portfolio_id, allocation_percentage,
				allocated_amount, allocated_shares, transaction_id, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		`, allocationID, transactionID, portfolio.ID, 100.0, 1500.00, 10.0, tx.ID)
		if err != nil {
			t.Fatalf("Failed to insert test allocation: %v", err)
		}

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+transactionID+"/allocations",
			map[string]string{"uuid": transactionID},
		)
		w := httptest.NewRecorder()

		handler.GetTransactionAllocations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKRAllocation
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.IBKRTransactionID != transactionID {
			t.Errorf("Expected transaction ID %s, got %s", transactionID, response.IBKRTransactionID)
		}

		if len(response.Allocations) != 1 {
			t.Errorf("Expected 1 allocation, got %d", len(response.Allocations))
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+nonExistentID+"/allocations",
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.GetTransactionAllocations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		transactionID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES (?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Buy AAPL', 'trade', 10, 150.00, 1500.00, 'USD', 1.00, 'allocated', datetime('now'), date('now'), '')
		`, transactionID, "IBKR123")
		if err != nil {
			t.Fatalf("Failed to insert test transaction: %v", err)
		}

		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+transactionID+"/allocations",
			map[string]string{"uuid": transactionID},
		)
		w := httptest.NewRecorder()

		handler.GetTransactionAllocations(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestIbkrHandler_GetEligiblePortfolios(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns eligible portfolios successfully when matched by ISIN", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create fund and portfolios
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)
		portfolio1 := testutil.NewPortfolio().Build(t, db)
		portfolio2 := testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolioFund(portfolio1.ID, fund.ID).Build(t, db)
		testutil.NewPortfolioFund(portfolio2.ID, fund.ID).Build(t, db)

		// Create IBKR transaction with matching ISIN
		transactionID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES (?, ?, datetime('now'), 'AAPL', 'US0378331005', 'Buy AAPL', 'trade', 10, 150.00, 1500.00, 'USD', 1.00, 'pending', datetime('now'), date('now'), '')
		`, transactionID, "IBKR123")
		if err != nil {
			t.Fatalf("Failed to insert test transaction: %v", err)
		}

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+transactionID+"/eligible-portfolios",
			map[string]string{"uuid": transactionID},
		)
		w := httptest.NewRecorder()

		handler.GetEligiblePortfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKREligiblePortfolioResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if !response.MatchInfo.Found {
			t.Error("Expected fund to be found")
		}

		if response.MatchInfo.MatchedBy != "isin" {
			t.Errorf("Expected matched by 'isin', got '%s'", response.MatchInfo.MatchedBy)
		}

		if len(response.Portfolios) != 2 {
			t.Errorf("Expected 2 eligible portfolios, got %d", len(response.Portfolios))
		}
	})

	t.Run("returns eligible portfolios when matched by symbol", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create fund with symbol but no ISIN match
		fund := testutil.NewFund().WithSymbol("GOOGL").WithISIN("US02079K3059").Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create IBKR transaction with matching symbol but different ISIN
		transactionID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES (?, ?, datetime('now'), 'GOOGL', 'DIFFERENT_ISIN', 'Buy GOOGL', 'trade', 7, 400.00, 2800.00, 'USD', 2.00, 'pending', datetime('now'), date('now'), '')
		`, transactionID, "IBKR456")
		if err != nil {
			t.Fatalf("Failed to insert test transaction: %v", err)
		}

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+transactionID+"/eligible-portfolios",
			map[string]string{"uuid": transactionID},
		)
		w := httptest.NewRecorder()

		handler.GetEligiblePortfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKREligiblePortfolioResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if !response.MatchInfo.Found {
			t.Error("Expected fund to be found")
		}

		if response.MatchInfo.MatchedBy != "symbol" {
			t.Errorf("Expected matched by 'symbol', got '%s'", response.MatchInfo.MatchedBy)
		}

		if len(response.Portfolios) != 1 {
			t.Errorf("Expected 1 eligible portfolio, got %d", len(response.Portfolios))
		}
	})

	t.Run("returns 200 with found=false when fund not found", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create IBKR transaction with no matching fund
		transactionID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES (?, ?, datetime('now'), 'UNKNOWN', 'UNKNOWN_ISIN', 'Buy UNKNOWN', 'trade', 10, 100.00, 1000.00, 'USD', 1.00, 'pending', datetime('now'), date('now'), '')
		`, transactionID, "IBKR789")
		if err != nil {
			t.Fatalf("Failed to insert test transaction: %v", err)
		}

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+transactionID+"/eligible-portfolios",
			map[string]string{"uuid": transactionID},
		)
		w := httptest.NewRecorder()

		handler.GetEligiblePortfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKREligiblePortfolioResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.MatchInfo.Found {
			t.Error("Expected fund not to be found")
		}

		if len(response.Portfolios) != 0 {
			t.Errorf("Expected 0 portfolios, got %d", len(response.Portfolios))
		}

		if response.Warning == "" {
			t.Error("Expected warning message when fund not found")
		}
	})

	t.Run("returns warning when fund exists but has no portfolios", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create fund without any portfolio assignments
		fund := testutil.NewFund().WithSymbol("MSFT").WithISIN("US5949181045").Build(t, db)

		// Create IBKR transaction matching the fund
		transactionID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES (?, ?, datetime('now'), ?, ?, 'Buy ' || ?, 'trade', 10, 300.00, 3000.00, 'USD', 3.00, 'pending', datetime('now'), date('now'), '')
		`, transactionID, "IBKR999", fund.Symbol, fund.Isin, fund.Symbol)
		if err != nil {
			t.Fatalf("Failed to insert test transaction: %v", err)
		}

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+transactionID+"/eligible-portfolios",
			map[string]string{"uuid": transactionID},
		)
		w := httptest.NewRecorder()

		handler.GetEligiblePortfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKREligiblePortfolioResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if !response.MatchInfo.Found {
			t.Error("Expected fund to be found")
		}

		if len(response.Portfolios) != 0 {
			t.Errorf("Expected 0 portfolios, got %d", len(response.Portfolios))
		}

		if response.Warning == "" {
			t.Error("Expected warning message when fund has no portfolios")
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+nonExistentID+"/eligible-portfolios",
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.GetEligiblePortfolios(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		transactionID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction (
				id, ibkr_transaction_id, transaction_date, symbol, isin, description,
				transaction_type, quantity, price, total_amount, currency, fees, status, imported_at,
				report_date, notes
			) VALUES (?, ?, datetime('now'), 'TEST', 'TEST_ISIN', 'Buy TEST', 'trade', 10, 100.00, 1000.00, 'USD', 1.00, 'pending', datetime('now'), date('now'), '')
		`, transactionID, "IBKR999")
		if err != nil {
			t.Fatalf("Failed to insert test transaction: %v", err)
		}

		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+transactionID+"/eligible-portfolios",
			map[string]string{"uuid": transactionID},
		)
		w := httptest.NewRecorder()

		handler.GetEligiblePortfolios(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// testFernetKey is a valid 32-byte fernet key (URL-safe base64) used only in tests.
const testFernetKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

// decodedTestFernetKey returns the pre-decoded Fernet key for injection into services.
func decodedTestFernetKey(t *testing.T) *fernet.Key {
	t.Helper()
	k, err := fernet.DecodeKey(testFernetKey)
	if err != nil {
		t.Fatalf("failed to decode test fernet key: %v", err)
	}
	return k
}

// validFlexToken is a realistic 25-digit IBKR flex token for testing.
const validFlexToken = "1079673754867139037835410"

// validFlexQueryID is a short numeric string that satisfies validation rules.
const validFlexQueryID = "123456"

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestIbkrHandler_UpdateIbkrConfig(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	setupHandlerWithKey := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db, service.IbkrWithEncryptionKey(decodedTestFernetKey(t)))
		return NewIbkrHandler(is), db
	}

	t.Run("creates config when enabled is true", func(t *testing.T) {
		handler, _ := setupHandlerWithKey(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"enabled": true,
			"flexToken": "`+validFlexToken+`",
			"flexQueryId": "`+validFlexQueryID+`",
			"tokenExpiresAt": "2030-01-01"
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IbkrConfig
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if !response.Configured {
			t.Error("Expected configured to be true")
		}
		if !response.Enabled {
			t.Error("Expected enabled to be true")
		}
		if response.FlexQueryID != validFlexQueryID {
			t.Errorf("Expected flexQueryId %q, got %q", validFlexQueryID, response.FlexQueryID)
		}
	})

	t.Run("creates config when disabled", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"enabled": false
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IbkrConfig
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if response.Enabled {
			t.Error("Expected enabled to be false")
		}
	})

	t.Run("disabling also forces autoImportEnabled to false", func(t *testing.T) {
		handler, db := setupHandler(t)

		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, testutil.MakeID(), "some_token", validFlexQueryID, true, true, false)
		if err != nil {
			t.Fatalf("Failed to insert test config: %v", err)
		}

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"enabled": false
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IbkrConfig
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if response.AutoImportEnabled {
			t.Error("Expected autoImportEnabled to be false when disabled")
		}
	})

	t.Run("partial update preserves existing fields", func(t *testing.T) {
		handler, db := setupHandlerWithKey(t)

		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, testutil.MakeID(), "encrypted_token", validFlexQueryID, false, true, false)
		if err != nil {
			t.Fatalf("Failed to insert test config: %v", err)
		}

		// Only update autoImportEnabled; flexQueryId should be unchanged.
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"autoImportEnabled": true
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IbkrConfig
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if response.FlexQueryID != validFlexQueryID {
			t.Errorf("Expected flexQueryId %q to be preserved, got %q", validFlexQueryID, response.FlexQueryID)
		}
		if !response.AutoImportEnabled {
			t.Error("Expected autoImportEnabled to be updated to true")
		}
	})

	t.Run("empty flexToken does not overwrite existing token", func(t *testing.T) {
		handler, db := setupHandlerWithKey(t)

		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, testutil.MakeID(), "my_encrypted_token", validFlexQueryID, false, true, false)
		if err != nil {
			t.Fatalf("Failed to insert test config: %v", err)
		}

		// flexToken is empty string — should not overwrite existing.
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"flexToken": "",
			"autoImportEnabled": true
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		// Verify DB still holds the original token.
		var storedToken string
		err = db.QueryRow(`SELECT flex_token FROM ibkr_config`).Scan(&storedToken)
		if err != nil {
			t.Fatalf("Failed to query token: %v", err)
		}
		if storedToken != "my_encrypted_token" {
			t.Errorf("Expected token to be preserved, got %q", storedToken)
		}
	})

	t.Run("replaces config row when flexQueryId changes", func(t *testing.T) {
		handler, db := setupHandlerWithKey(t)

		oldID := testutil.MakeID()
		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, oldID, "some_token", validFlexQueryID, false, true, false)
		if err != nil {
			t.Fatalf("Failed to insert test config: %v", err)
		}

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"enabled": true,
			"flexToken": "`+validFlexToken+`",
			"flexQueryId": "654321",
			"tokenExpiresAt": "2030-01-01"
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IbkrConfig
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if response.FlexQueryID != "654321" {
			t.Errorf("Expected flexQueryId '654321', got %q", response.FlexQueryID)
		}

		// Old row must be gone — only one row should remain.
		var count int
		err = db.QueryRow(`SELECT COUNT(*) FROM ibkr_config`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count config rows: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 config row after replace, got %d", count)
		}
	})

	t.Run("returns 400 on invalid JSON body", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{invalid json}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when flexToken is wrong length", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"enabled": true,
			"flexToken": "tooshort",
			"flexQueryId": "`+validFlexQueryID+`"
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when default allocations do not add up to 100", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		portfolioID := portfolio.ID

		body := `{
			"enabled": true,
			"defaultAllocationEnabled": true,
			"defaultAllocations": [
				{"portfolioId": "` + portfolioID + `", "percentage": 50.0}
			]
		}`

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", body)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 when IBKR_ENCRYPTION_KEY is not set", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"enabled": true,
			"flexToken": "`+validFlexToken+`",
			"flexQueryId": "`+validFlexQueryID+`"
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/config", `{
			"enabled": false
		}`)
		w := httptest.NewRecorder()

		handler.UpdateIbkrConfig(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_DeleteIbkrConfig(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns 204 when config exists", func(t *testing.T) {
		handler, db := setupHandler(t)

		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, testutil.MakeID(), "some_token", validFlexQueryID, false, true, false)
		if err != nil {
			t.Fatalf("Failed to insert test config: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/ibkr/config", nil)
		w := httptest.NewRecorder()

		handler.DeleteIbkrConfig(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}

		if w.Body.Len() != 0 {
			t.Errorf("Expected empty body for 204, got: %s", w.Body.String())
		}
	})

	t.Run("config is removed from database after delete", func(t *testing.T) {
		handler, db := setupHandler(t)

		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, testutil.MakeID(), "some_token", validFlexQueryID, false, true, false)
		if err != nil {
			t.Fatalf("Failed to insert test config: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/ibkr/config", nil)
		w := httptest.NewRecorder()

		handler.DeleteIbkrConfig(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}

		var count int
		err = db.QueryRow(`SELECT COUNT(*) FROM ibkr_config`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count config rows: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 rows after delete, got %d", count)
		}
	})

	t.Run("returns 404 when no config exists", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodDelete, "/api/ibkr/config", nil)
		w := httptest.NewRecorder()

		handler.DeleteIbkrConfig(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodDelete, "/api/ibkr/config", nil)
		w := httptest.NewRecorder()

		handler.DeleteIbkrConfig(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_ImportFlexReport(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	// minimalFlexXML is a valid empty Flex report — no trades, no rates.
	// Used to exercise the cache path without needing a real IBKR token or API call.
	const minimalFlexXML = `<FlexQueryResponse queryName="test" type="AF">` +
		`<FlexStatements count="1">` +
		`<FlexStatement accountId="" fromDate="" toDate="" period="" whenGenerated="">` +
		`<Trades></Trades>` +
		`<CashTransactions></CashTransactions>` +
		`<ConversionRates></ConversionRates>` +
		`</FlexStatement>` +
		`</FlexStatements>` +
		`</FlexQueryResponse>`

	t.Run("returns 200 with 0 imported when valid cache exists", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Config row is required by GetIbkrConfig (called at start of ImportFlexReport).
		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, testutil.MakeID(), "dummy_token", 12345, false, true, false)
		if err != nil {
			t.Fatalf("Failed to insert config: %v", err)
		}

		// Insert a cache entry that expires in the future — service will use it
		// instead of calling the IBKR API, so no token or encryption key needed.
		_, err = db.Exec(`
			INSERT INTO ibkr_import_cache (id, cache_key, data, created_at, expires_at)
			VALUES (?, ?, ?, datetime('now'), datetime('now', '+1 hour'))
		`, testutil.MakeID(), "ibkr_flex_12345_today", minimalFlexXML)
		if err != nil {
			t.Fatalf("Failed to insert cache entry: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/import", nil)
		w := httptest.NewRecorder()

		handler.ImportFlexReport(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response struct {
			Success  bool `json:"success"`
			Imported int  `json:"imported"`
			Skipped  int  `json:"skipped"`
		}
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if !response.Success {
			t.Error("Expected success to be true")
		}
		if response.Imported != 0 {
			t.Errorf("Expected 0 imported, got %d", response.Imported)
		}
		if response.Skipped != 0 {
			t.Errorf("Expected 0 skipped, got %d", response.Skipped)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/import", nil)
		w := httptest.NewRecorder()

		handler.ImportFlexReport(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_TestIbkrConnection(t *testing.T) {
	// validBody contains a 24-digit token (minimum accepted) and a numeric queryId.
	const validBody = `{"flexToken":"123456789012345678901234","flexQueryId":"12345"}`

	setupHandler := func(t *testing.T, mock *mockIbkrClient) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrServiceWithMockIBKR(t, db, mock)
		return NewIbkrHandler(is), db
	}

	successMock := func() *mockIbkrClient {
		return &mockIbkrClient{
			testConnectionFn: func(_ context.Context, _, _ string) (bool, error) {
				return true, nil
			},
		}
	}

	t.Run("returns 200 with success=true on valid credentials", func(t *testing.T) {
		handler, _ := setupHandler(t, successMock())

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test", strings.NewReader(validBody))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]bool
		//nolint:errcheck // Test assertion — decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)
		if !response["success"] {
			t.Error("Expected success to be true")
		}
	})

	t.Run("returns 400 on invalid JSON body", func(t *testing.T) {
		handler, _ := setupHandler(t, successMock())

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test", strings.NewReader("{invalid}"))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when flexToken is absent", func(t *testing.T) {
		handler, _ := setupHandler(t, successMock())

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test",
			strings.NewReader(`{"flexQueryId":"12345"}`))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when flexToken is too short", func(t *testing.T) {
		handler, _ := setupHandler(t, successMock())

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test",
			strings.NewReader(`{"flexToken":"12345","flexQueryId":"12345"}`))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when flexToken contains non-digit characters", func(t *testing.T) {
		handler, _ := setupHandler(t, successMock())

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test",
			strings.NewReader(`{"flexToken":"1234567890abcdefghijklmn","flexQueryId":"12345"}`))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when flexQueryId is empty", func(t *testing.T) {
		handler, _ := setupHandler(t, successMock())

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test",
			strings.NewReader(`{"flexToken":"123456789012345678901234","flexQueryId":""}`))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when flexQueryId is not numeric", func(t *testing.T) {
		handler, _ := setupHandler(t, successMock())

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test",
			strings.NewReader(`{"flexToken":"123456789012345678901234","flexQueryId":"abc"}`))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 when IBKR client returns an error", func(t *testing.T) {
		mock := &mockIbkrClient{
			testConnectionFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, fmt.Errorf("ibkr authentication failed")
			},
		}
		handler, _ := setupHandler(t, mock)

		req := httptest.NewRequest(http.MethodPost, "/api/ibkr/config/test", strings.NewReader(validBody))
		w := httptest.NewRecorder()

		handler.TestIbkrConnection(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

}

func TestIbkrHandler_GetTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("returns pending transaction without allocations", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+ibkrTx.ID,
			map[string]string{"uuid": ibkrTx.ID},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.IBKRTransactionDetail
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if response.ID != ibkrTx.ID {
			t.Errorf("Expected ID %s, got %s", ibkrTx.ID, response.ID)
		}
		if len(response.Allocations) != 0 {
			t.Errorf("Expected 0 allocations for pending tx, got %d", len(response.Allocations))
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+nonExistentID,
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
		db.Close()

		id := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/ibkr/inbox/"+id,
			map[string]string{"uuid": id},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_DeleteTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("deletes pending transaction successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/ibkr/inbox/"+ibkrTx.ID,
			map[string]string{"uuid": ibkrTx.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteTransaction(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
		}

		// Verify deleted from DB
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 rows after delete, got %d", count)
		}
	})

	t.Run("returns 400 when transaction is processed", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/ibkr/inbox/"+ibkrTx.ID,
			map[string]string{"uuid": ibkrTx.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/ibkr/inbox/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.DeleteTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_IgnoreTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("ignores pending transaction successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/ignore",
			map[string]string{"uuid": ibkrTx.ID},
		)
		w := httptest.NewRecorder()

		handler.IgnoreTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify status changed
		var status string
		err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query status: %v", err)
		}
		if status != "ignored" {
			t.Errorf("Expected status 'ignored', got '%s'", status)
		}
	})

	t.Run("returns 400 when transaction is processed", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/ignore",
			map[string]string{"uuid": ibkrTx.ID},
		)
		w := httptest.NewRecorder()

		handler.IgnoreTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/ibkr/inbox/"+nonExistentID+"/ignore",
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.IgnoreTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestIbkrHandler_AllocateTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("allocates transaction to single portfolio", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithQuantity(10).WithPrice(150.00).WithTotalAmount(1500.00).WithFees(1.00).
			Build(t, db)

		body := `{"allocations":[{"portfolioId":"` + portfolio.ID + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify status changed to processed
		var status string
		err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query status: %v", err)
		}
		if status != "processed" {
			t.Errorf("Expected status 'processed', got '%s'", status)
		}

		// Verify allocation records created
		var allocCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM ibkr_transaction_allocation WHERE ibkr_transaction_id = ?`, ibkrTx.ID).Scan(&allocCount)
		if err != nil {
			t.Fatalf("Failed to count allocations: %v", err)
		}
		// Should have 2 allocations: 1 trade + 1 fee (fees=1.00 > 0)
		if allocCount != 2 {
			t.Errorf("Expected 2 allocation records (trade+fee), got %d", allocCount)
		}

		// Verify transaction records created
		var txCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM "transaction" t JOIN ibkr_transaction_allocation ita ON t.id = ita.transaction_id WHERE ita.ibkr_transaction_id = ?`, ibkrTx.ID).Scan(&txCount)
		if err != nil {
			t.Fatalf("Failed to count transactions: %v", err)
		}
		if txCount != 2 {
			t.Errorf("Expected 2 transaction records, got %d", txCount)
		}
	})

	t.Run("allocates to multiple portfolios", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio1 := testutil.NewPortfolio().Build(t, db)
		portfolio2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US02079K3059").WithSymbol("GOOGL").Build(t, db)
		testutil.NewPortfolioFund(portfolio1.ID, fund.ID).Build(t, db)
		testutil.NewPortfolioFund(portfolio2.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US02079K3059").WithSymbol("GOOGL").
			WithQuantity(10).WithPrice(200.00).WithTotalAmount(2000.00).WithFees(0).
			Build(t, db)

		body := `{"allocations":[` +
			`{"portfolioId":"` + portfolio1.ID + `","percentage":60},` +
			`{"portfolioId":"` + portfolio2.ID + `","percentage":40}` +
			`]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify 2 trade allocations (no fee since fees=0)
		var allocCount int
		err := db.QueryRow(`SELECT COUNT(*) FROM ibkr_transaction_allocation WHERE ibkr_transaction_id = ?`, ibkrTx.ID).Scan(&allocCount)
		if err != nil {
			t.Fatalf("Failed to count allocations: %v", err)
		}
		if allocCount != 2 {
			t.Errorf("Expected 2 allocation records (no fee), got %d", allocCount)
		}
	})

	t.Run("creates portfolio_fund if not exists", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US5949181045").WithSymbol("MSFT").Build(t, db)
		// Deliberately NOT creating portfolio_fund — allocate should create it

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US5949181045").WithSymbol("MSFT").
			WithQuantity(5).WithPrice(400.00).WithTotalAmount(2000.00).WithFees(0).
			Build(t, db)

		body := `{"allocations":[{"portfolioId":"` + portfolio.ID + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify portfolio_fund was created
		var pfCount int
		err := db.QueryRow(`SELECT COUNT(*) FROM portfolio_fund WHERE portfolio_id = ? AND fund_id = ?`,
			portfolio.ID, fund.ID).Scan(&pfCount)
		if err != nil {
			t.Fatalf("Failed to count portfolio_fund: %v", err)
		}
		if pfCount != 1 {
			t.Errorf("Expected portfolio_fund to be created, got %d", pfCount)
		}
	})

	t.Run("auto-allocates from config defaults", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("IE00B4L5Y983").WithSymbol("IWDA").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Set up config with default allocations
		configID := testutil.MakeID()
		allocJSON := fmt.Sprintf(`[{"portfolioId":"%s","percentage":100}]`, portfolio.ID)
		_, err := db.Exec(`
			INSERT INTO ibkr_config (
				id, flex_token, flex_query_id, auto_import_enabled, enabled,
				default_allocation_enabled, default_allocations, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, configID, "token", "12345", false, true, true, allocJSON)
		if err != nil {
			t.Fatalf("Failed to insert config: %v", err)
		}

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("IE00B4L5Y983").WithSymbol("IWDA").
			WithQuantity(20).WithPrice(80.00).WithTotalAmount(1600.00).WithFees(0).
			Build(t, db)

		// Empty allocations = auto-allocate
		body := `{"allocations":[]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var status string
		err = db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query status: %v", err)
		}
		if status != "processed" {
			t.Errorf("Expected status 'processed', got '%s'", status)
		}
	})

	t.Run("returns 400 when transaction is already processed", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		body := `{"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when fund not matched", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("UNKNOWN_ISIN").WithSymbol("UNKNOWN").
			Build(t, db)

		body := `{"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when allocations do not sum to 100", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db)

		body := `{"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":50}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid allocation sum, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid JSON body", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db)

		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			`{invalid}`,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		body := `{"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+nonExistentID+"/allocate",
			map[string]string{"uuid": nonExistentID},
			body,
		)
		w := httptest.NewRecorder()

		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIbkrHandler_BulkAllocate(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("allocates multiple transactions successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx1 := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").WithFees(0).
			Build(t, db)
		tx2 := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").WithFees(0).
			Build(t, db)

		body := `{"transactionIds":["` + tx1.ID + `","` + tx2.ID + `"],` +
			`"allocations":[{"portfolioId":"` + portfolio.ID + `","percentage":100}]}`
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/inbox/bulk-allocate", body)
		w := httptest.NewRecorder()

		handler.BulkAllocate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.BulkAllocateResponse
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if response.Success != 2 {
			t.Errorf("Expected 2 successes, got %d", response.Success)
		}
		if response.Failed != 0 {
			t.Errorf("Expected 0 failures, got %d", response.Failed)
		}
	})

	t.Run("handles partial failure", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx1 := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").WithFees(0).
			Build(t, db)
		// tx2 already processed → will fail
		tx2 := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").WithStatus("processed").
			Build(t, db)

		body := `{"transactionIds":["` + tx1.ID + `","` + tx2.ID + `"],` +
			`"allocations":[{"portfolioId":"` + portfolio.ID + `","percentage":100}]}`
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/inbox/bulk-allocate", body)
		w := httptest.NewRecorder()

		handler.BulkAllocate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.BulkAllocateResponse
		//nolint:errcheck
		json.NewDecoder(w.Body).Decode(&response)

		if response.Success != 1 {
			t.Errorf("Expected 1 success, got %d", response.Success)
		}
		if response.Failed != 1 {
			t.Errorf("Expected 1 failure, got %d", response.Failed)
		}
		if len(response.Errors) != 1 {
			t.Errorf("Expected 1 error message, got %d", len(response.Errors))
		}
	})

	t.Run("returns 400 on invalid JSON body", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/inbox/bulk-allocate", `{invalid}`)
		w := httptest.NewRecorder()

		handler.BulkAllocate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when no transaction IDs provided", func(t *testing.T) {
		handler, _ := setupHandler(t)

		body := `{"transactionIds":[],"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":100}]}`
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/ibkr/inbox/bulk-allocate", body)
		w := httptest.NewRecorder()

		handler.BulkAllocate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestIbkrHandler_UnallocateTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	// allocateHelper sets up a fully allocated IBKR transaction for unallocate/modify tests.
	allocateHelper := func(t *testing.T, db *sql.DB, handler *IbkrHandler) (string, string) {
		t.Helper()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithQuantity(10).WithPrice(150.00).WithTotalAmount(1500.00).WithFees(0).
			Build(t, db)

		// Allocate it
		body := `{"allocations":[{"portfolioId":"` + portfolio.ID + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()
		handler.AllocateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("allocateHelper: Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		return ibkrTx.ID, portfolio.ID
	}

	t.Run("unallocates processed transaction successfully", func(t *testing.T) {
		handler, db := setupHandler(t)
		ibkrTxID, _ := allocateHelper(t, db, handler)

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTxID+"/unallocate",
			map[string]string{"uuid": ibkrTxID},
		)
		w := httptest.NewRecorder()

		handler.UnallocateTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify status reset to pending
		var status string
		err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTxID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query status: %v", err)
		}
		if status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", status)
		}

		// Verify allocations deleted
		var allocCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM ibkr_transaction_allocation WHERE ibkr_transaction_id = ?`, ibkrTxID).Scan(&allocCount)
		if err != nil {
			t.Fatalf("Failed to count allocations: %v", err)
		}
		if allocCount != 0 {
			t.Errorf("Expected 0 allocations after unallocate, got %d", allocCount)
		}
	})

	t.Run("returns 400 when transaction is not processed", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db) // status=pending

		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/unallocate",
			map[string]string{"uuid": ibkrTx.ID},
		)
		w := httptest.NewRecorder()

		handler.UnallocateTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodPost,
			"/api/ibkr/inbox/"+nonExistentID+"/unallocate",
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.UnallocateTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestIbkrHandler_ModifyAllocations(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("modifies allocations on processed transaction", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio1 := testutil.NewPortfolio().Build(t, db)
		portfolio2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)
		testutil.NewPortfolioFund(portfolio1.ID, fund.ID).Build(t, db)
		testutil.NewPortfolioFund(portfolio2.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithQuantity(10).WithPrice(150.00).WithTotalAmount(1500.00).WithFees(0).
			Build(t, db)

		// First allocate 100% to portfolio1
		allocBody := `{"allocations":[{"portfolioId":"` + portfolio1.ID + `","percentage":100}]}`
		allocReq := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			allocBody,
		)
		aw := httptest.NewRecorder()
		handler.AllocateTransaction(aw, allocReq)
		if aw.Code != http.StatusOK {
			t.Fatalf("allocate setup failed: %d: %s", aw.Code, aw.Body.String())
		}

		// Now modify to 60/40 split
		modifyBody := `{"allocations":[` +
			`{"portfolioId":"` + portfolio1.ID + `","percentage":60},` +
			`{"portfolioId":"` + portfolio2.ID + `","percentage":40}` +
			`]}`
		modReq := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocations",
			map[string]string{"uuid": ibkrTx.ID},
			modifyBody,
		)
		mw := httptest.NewRecorder()

		handler.ModifyAllocations(mw, modReq)

		if mw.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", mw.Code, mw.Body.String())
		}

		// Verify status is still processed
		var status string
		err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query status: %v", err)
		}
		if status != "processed" {
			t.Errorf("Expected status 'processed', got '%s'", status)
		}

		// Verify new allocations exist (2 trade allocs, no fee since fees=0)
		var allocCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM ibkr_transaction_allocation WHERE ibkr_transaction_id = ?`, ibkrTx.ID).Scan(&allocCount)
		if err != nil {
			t.Fatalf("Failed to count allocations: %v", err)
		}
		if allocCount != 2 {
			t.Errorf("Expected 2 allocation records after modify, got %d", allocCount)
		}
	})

	t.Run("returns 400 when transaction is not processed", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db) // pending

		body := `{"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocations",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.ModifyAllocations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when allocations do not sum to 100", func(t *testing.T) {
		handler, _ := setupHandler(t)

		id := testutil.MakeID()
		body := `{"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":50}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/ibkr/inbox/"+id+"/allocations",
			map[string]string{"uuid": id},
			body,
		)
		w := httptest.NewRecorder()

		handler.ModifyAllocations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid JSON body", func(t *testing.T) {
		handler, _ := setupHandler(t)

		id := testutil.MakeID()
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/ibkr/inbox/"+id+"/allocations",
			map[string]string{"uuid": id},
			`{invalid}`,
		)
		w := httptest.NewRecorder()

		handler.ModifyAllocations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		body := `{"allocations":[{"portfolioId":"` + testutil.MakeID() + `","percentage":100}]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPut,
			"/api/ibkr/inbox/"+nonExistentID+"/allocations",
			map[string]string{"uuid": nonExistentID},
			body,
		)
		w := httptest.NewRecorder()

		handler.ModifyAllocations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestIbkrHandler_MatchDividend(t *testing.T) {
	setupHandler := func(t *testing.T) (*IbkrHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		is := testutil.NewTestIbkrService(t, db)
		return NewIbkrHandler(is), db
	}

	t.Run("matches dividend to allocated DRIP transaction", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create and allocate IBKR transaction with DRIP note
		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").WithNotes("R").
			WithQuantity(5).WithPrice(150.00).WithTotalAmount(750.00).WithFees(0).
			Build(t, db)

		allocBody := `{"allocations":[{"portfolioId":"` + portfolio.ID + `","percentage":100}]}`
		allocReq := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/allocate",
			map[string]string{"uuid": ibkrTx.ID},
			allocBody,
		)
		aw := httptest.NewRecorder()
		handler.AllocateTransaction(aw, allocReq)
		if aw.Code != http.StatusOK {
			t.Fatalf("allocate setup failed: %d: %s", aw.Code, aw.Body.String())
		}

		// Create a pending dividend for matching
		dividend := testutil.NewDividend(fund.ID, pf.ID).Build(t, db)
		_, err := db.Exec(`UPDATE dividend SET reinvestment_status = 'PENDING' WHERE id = ?`, dividend.ID)
		if err != nil {
			t.Fatalf("Failed to update dividend status: %v", err)
		}

		// Match the dividend
		matchBody := `{"dividendIds":["` + dividend.ID + `"]}`
		matchReq := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/match-dividend",
			map[string]string{"uuid": ibkrTx.ID},
			matchBody,
		)
		mw := httptest.NewRecorder()

		handler.MatchDividend(mw, matchReq)

		if mw.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", mw.Code, mw.Body.String())
		}

		// Verify dividend was updated
		var reinvestStatus, reinvestTxID string
		err = db.QueryRow(`SELECT reinvestment_status, reinvestment_transaction_id FROM dividend WHERE id = ?`, dividend.ID).Scan(&reinvestStatus, &reinvestTxID)
		if err != nil {
			t.Fatalf("Failed to query dividend: %v", err)
		}
		if reinvestStatus != "COMPLETED" {
			t.Errorf("Expected reinvestment_status 'COMPLETED', got '%s'", reinvestStatus)
		}
		if reinvestTxID == "" {
			t.Error("Expected reinvestment_transaction_id to be set")
		}
	})

	t.Run("returns 400 when transaction is not processed", func(t *testing.T) {
		handler, db := setupHandler(t)

		ibkrTx := testutil.NewIBKRTransaction().Build(t, db) // pending

		body := `{"dividendIds":["` + testutil.MakeID() + `"]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+ibkrTx.ID+"/match-dividend",
			map[string]string{"uuid": ibkrTx.ID},
			body,
		)
		w := httptest.NewRecorder()

		handler.MatchDividend(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()
		body := `{"dividendIds":["` + testutil.MakeID() + `"]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+nonExistentID+"/match-dividend",
			map[string]string{"uuid": nonExistentID},
			body,
		)
		w := httptest.NewRecorder()

		handler.MatchDividend(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 on invalid JSON body", func(t *testing.T) {
		handler, _ := setupHandler(t)

		id := testutil.MakeID()
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+id+"/match-dividend",
			map[string]string{"uuid": id},
			`{invalid}`,
		)
		w := httptest.NewRecorder()

		handler.MatchDividend(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when no dividend IDs provided", func(t *testing.T) {
		handler, _ := setupHandler(t)

		id := testutil.MakeID()
		body := `{"dividendIds":[]}`
		req := testutil.NewRequestWithURLParamsAndBody(
			http.MethodPost,
			"/api/ibkr/inbox/"+id+"/match-dividend",
			map[string]string{"uuid": id},
			body,
		)
		w := httptest.NewRecorder()

		handler.MatchDividend(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}
