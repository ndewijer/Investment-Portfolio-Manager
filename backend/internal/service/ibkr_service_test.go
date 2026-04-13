package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/ibkr"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// --- Mock IBKR Client ---

type mockIBKRClient struct {
	retreiveFunc func(ctx context.Context, token, queryID string) (ibkr.FlexQueryResponse, []byte, error)
	testFunc     func(ctx context.Context, token, queryID string) (bool, error)
}

func (m *mockIBKRClient) RetreiveIbkrFlexReport(ctx context.Context, token, queryID string) (ibkr.FlexQueryResponse, []byte, error) {
	if m.retreiveFunc != nil {
		return m.retreiveFunc(ctx, token, queryID)
	}
	return ibkr.FlexQueryResponse{}, nil, fmt.Errorf("not implemented")
}

func (m *mockIBKRClient) TestIbkrConnection(ctx context.Context, token, queryID string) (bool, error) {
	if m.testFunc != nil {
		return m.testFunc(ctx, token, queryID)
	}
	return false, fmt.Errorf("not implemented")
}

// --- Helpers ---

func generateFernetKey(t *testing.T) *fernet.Key {
	t.Helper()
	var key fernet.Key
	if err := key.Generate(); err != nil {
		t.Fatalf("failed to generate fernet key: %v", err)
	}
	return &key
}

func insertIbkrConfig(t *testing.T, db *sql.DB, id, flexToken, flexQueryID string, enabled bool) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO ibkr_config (id, flex_token, flex_query_id, auto_import_enabled, created_at, updated_at, enabled, default_allocation_enabled, default_allocations)
		VALUES (?, ?, ?, 0, datetime('now'), datetime('now'), ?, 0, '[]')`,
		id, flexToken, flexQueryID, enabled)
	if err != nil {
		t.Fatalf("failed to insert ibkr config: %v", err)
	}
}

func insertIbkrConfigWithExpiry(t *testing.T, db *sql.DB, id, flexToken, flexQueryID string, enabled bool, expiresAt time.Time) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO ibkr_config (id, flex_token, flex_query_id, token_expires_at, auto_import_enabled, created_at, updated_at, enabled, default_allocation_enabled, default_allocations)
		VALUES (?, ?, ?, ?, 0, datetime('now'), datetime('now'), ?, 0, '[]')`,
		id, flexToken, flexQueryID, expiresAt.Format("2006-01-02 15:04:05"), enabled)
	if err != nil {
		t.Fatalf("failed to insert ibkr config with expiry: %v", err)
	}
}

func insertIbkrConfigWithDefaultAllocations(t *testing.T, db *sql.DB, id, flexToken, flexQueryID string, enabled bool, defaultAlloc string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO ibkr_config (id, flex_token, flex_query_id, auto_import_enabled, created_at, updated_at, enabled, default_allocation_enabled, default_allocations)
		VALUES (?, ?, ?, 0, datetime('now'), datetime('now'), ?, 1, ?)`,
		id, flexToken, flexQueryID, enabled, defaultAlloc)
	if err != nil {
		t.Fatalf("failed to insert ibkr config with allocations: %v", err)
	}
}

// --- Encryption Tests ---

func TestIbkrService_EncryptDecryptToken(t *testing.T) {
	t.Run("round-trip encryption and decryption", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key))

		originalToken := "my-secret-ibkr-token-12345" //nolint:gosec // G101: Test credential, not a real secret

		encrypted, err := svc.ExportEncryptToken(originalToken)
		if err != nil {
			t.Fatalf("encryptToken failed: %v", err)
		}
		if encrypted == "" {
			t.Fatal("encrypted token is empty")
		}
		if encrypted == originalToken {
			t.Fatal("encrypted token should differ from original")
		}

		decrypted, err := svc.ExportDecryptToken(encrypted)
		if err != nil {
			t.Fatalf("decryptToken failed: %v", err)
		}
		if decrypted != originalToken {
			t.Errorf("expected %q, got %q", originalToken, decrypted)
		}
	})

	t.Run("decrypt fails without encryption key", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{})

		_, err := svc.ExportDecryptToken("some-token")
		if err == nil {
			t.Fatal("expected error when encryption key is nil")
		}
	})

	t.Run("encrypt fails without encryption key", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{})

		_, err := svc.ExportEncryptToken("some-token")
		if err == nil {
			t.Fatal("expected error when encryption key is nil")
		}
	})

	t.Run("decrypt fails with wrong key", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key1 := generateFernetKey(t)
		key2 := generateFernetKey(t)

		svc1 := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key1))
		svc2 := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key2))

		encrypted, err := svc1.ExportEncryptToken("my-token")
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}

		_, err = svc2.ExportDecryptToken(encrypted)
		if err == nil {
			t.Fatal("expected error when decrypting with wrong key")
		}
	})
}

// --- GetIbkrConfig Tests ---

func TestIbkrService_GetIbkrConfig(t *testing.T) {
	t.Run("returns configured=false when no config exists", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		config, err := svc.GetIbkrConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.Configured {
			t.Error("expected Configured=false when no config exists")
		}
	})

	t.Run("returns existing config", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		insertIbkrConfig(t, db, testutil.MakeID(), "enc-token", "12345", true)

		config, err := svc.GetIbkrConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.FlexQueryID != "12345" {
			t.Errorf("expected FlexQueryID=12345, got %s", config.FlexQueryID)
		}
	})

	t.Run("adds token warning when expiring within 30 days", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		expiresAt := time.Now().UTC().Add(10 * 24 * time.Hour) // 10 days from now
		insertIbkrConfigWithExpiry(t, db, testutil.MakeID(), "enc-token", "12345", true, expiresAt)

		config, err := svc.GetIbkrConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.TokenWarning == "" {
			t.Error("expected token warning for expiring token")
		}
	})

	t.Run("no token warning when expiring after 30 days", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		expiresAt := time.Now().UTC().Add(60 * 24 * time.Hour) // 60 days from now
		insertIbkrConfigWithExpiry(t, db, testutil.MakeID(), "enc-token", "12345", true, expiresAt)

		config, err := svc.GetIbkrConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.TokenWarning != "" {
			t.Errorf("expected no token warning, got %q", config.TokenWarning)
		}
	})
}

// --- UpdateIbkrConfig Tests ---

func TestIbkrService_UpdateIbkrConfig(t *testing.T) {
	t.Run("creates new config when none exists (disabled)", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key))

		enabled := false
		queryID := "99999"
		config, err := svc.UpdateIbkrConfig(context.Background(), request.UpdateIbkrConfigRequest{
			Enabled:     &enabled,
			FlexQueryID: &queryID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.FlexQueryID != "99999" {
			t.Errorf("expected FlexQueryID=99999, got %s", config.FlexQueryID)
		}
		if config.Enabled {
			t.Error("expected Enabled=false")
		}
		if !config.Configured {
			t.Error("expected Configured=true after update")
		}
	})

	t.Run("enables config with token and query id", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key))

		enabled := true
		queryID := "11111"
		token := "my-plain-token"
		config, err := svc.UpdateIbkrConfig(context.Background(), request.UpdateIbkrConfigRequest{
			Enabled:     &enabled,
			FlexQueryID: &queryID,
			FlexToken:   &token,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !config.Enabled {
			t.Error("expected Enabled=true")
		}
		if config.FlexQueryID != "11111" {
			t.Errorf("expected FlexQueryID=11111, got %s", config.FlexQueryID)
		}
	})

	t.Run("error when enabling without token", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key))

		enabled := true
		queryID := "11111"
		_, err := svc.UpdateIbkrConfig(context.Background(), request.UpdateIbkrConfigRequest{
			Enabled:     &enabled,
			FlexQueryID: &queryID,
		})
		if err == nil {
			t.Fatal("expected error when enabling without token")
		}
	})

	t.Run("disabling config sets auto-import to false", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key))

		// First enable
		enabled := true
		queryID := "11111"
		token := "my-plain-token"
		autoImport := true
		_, err := svc.UpdateIbkrConfig(context.Background(), request.UpdateIbkrConfigRequest{
			Enabled:           &enabled,
			FlexQueryID:       &queryID,
			FlexToken:         &token,
			AutoImportEnabled: &autoImport,
		})
		if err != nil {
			t.Fatalf("unexpected error creating config: %v", err)
		}

		// Now disable
		disabled := false
		config, err := svc.UpdateIbkrConfig(context.Background(), request.UpdateIbkrConfigRequest{
			Enabled: &disabled,
		})
		if err != nil {
			t.Fatalf("unexpected error disabling: %v", err)
		}
		if config.AutoImportEnabled {
			t.Error("expected AutoImportEnabled=false when disabled")
		}
	})

	t.Run("updates default allocations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithEncryptionKey(key))

		portfolio := testutil.NewPortfolio().Build(t, db)
		enabled := true
		queryID := "11111"
		token := "my-plain-token"
		pct := 100.0
		pid := portfolio.ID
		config, err := svc.UpdateIbkrConfig(context.Background(), request.UpdateIbkrConfigRequest{
			Enabled:     &enabled,
			FlexQueryID: &queryID,
			FlexToken:   &token,
			DefaultAllocations: []request.Allocation{
				{PortfolioID: &pid, Percentage: &pct},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(config.DefaultAllocations) != 1 {
			t.Errorf("expected 1 default allocation, got %d", len(config.DefaultAllocations))
		}
	})
}

// --- DeleteIbkrConfig Tests ---

func TestIbkrService_DeleteIbkrConfig(t *testing.T) {
	t.Run("deletes existing config", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		insertIbkrConfig(t, db, testutil.MakeID(), "enc-token", "12345", true)

		err := svc.DeleteIbkrConfig(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		config, err := svc.GetIbkrConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.Configured {
			t.Error("expected Configured=false after deletion")
		}
	})
}

// --- TestIbkrConnection Tests ---

func TestIbkrService_TestIbkrConnection(t *testing.T) {
	t.Run("returns true on success", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mock := &mockIBKRClient{
			testFunc: func(_ context.Context, token, queryID string) (bool, error) {
				if token == "test-token" && queryID == "12345" {
					return true, nil
				}
				return false, fmt.Errorf("unexpected args")
			},
		}
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, mock)

		ok, err := svc.TestIbkrConnection(context.Background(), request.TestIbkrConnectionRequest{
			FlexToken:   "test-token",
			FlexQueryID: "12345",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true")
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mock := &mockIBKRClient{
			testFunc: func(_ context.Context, _, _ string) (bool, error) {
				return false, fmt.Errorf("connection refused")
			},
		}
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, mock)

		_, err := svc.TestIbkrConnection(context.Background(), request.TestIbkrConnectionRequest{
			FlexToken:   "test-token",
			FlexQueryID: "12345",
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- GetActivePortfolios Tests ---

func TestIbkrService_GetActivePortfolios(t *testing.T) {
	t.Run("returns only active non-archived portfolios", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolio().Archived().Build(t, db)

		portfolios, err := svc.GetActivePortfolios()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(portfolios) != 2 {
			t.Errorf("expected 2 active portfolios, got %d", len(portfolios))
		}
	})
}

// --- GetInbox / GetInboxCount Tests ---

func TestIbkrService_GetInbox(t *testing.T) {
	t.Run("returns pending transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		inbox, err := svc.GetInbox("pending", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(inbox) != 1 {
			t.Errorf("expected 1 pending transaction, got %d", len(inbox))
		}
	})
}

func TestIbkrService_GetInboxCount(t *testing.T) {
	t.Run("counts pending transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)
		testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		count, err := svc.GetInboxCount()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count.Count != 2 {
			t.Errorf("expected count=2, got %d", count.Count)
		}
	})
}

// --- GetPendingDividends Tests ---

func TestIbkrService_GetPendingDividends(t *testing.T) {
	t.Run("returns pending dividends by ISIN", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL.NASDAQ").WithISIN("US0378331005").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Insert dividend with PENDING status (uppercase, matching the repo query)
		_, err := db.Exec(`
			INSERT INTO dividend (id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned, dividend_per_share, total_amount, reinvestment_status)
			VALUES (?, ?, ?, date('now', '-10 days'), date('now', '-5 days'), 100.0, 0.50, 50.0, 'PENDING')`,
			testutil.MakeID(), fund.ID, pf.ID)
		if err != nil {
			t.Fatalf("failed to insert dividend: %v", err)
		}

		dividends, err := svc.GetPendingDividends("", "US0378331005")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dividends) != 1 {
			t.Errorf("expected 1 pending dividend, got %d", len(dividends))
		}
	})

	t.Run("returns empty when no pending dividends", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		dividends, err := svc.GetPendingDividends("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dividends) != 0 {
			t.Errorf("expected 0 pending dividends, got %d", len(dividends))
		}
	})
}

// --- DeleteIbkrTransaction Tests ---

func TestIbkrService_DeleteIbkrTransaction(t *testing.T) {
	t.Run("deletes pending transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		tx := testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)

		err := svc.DeleteIbkrTransaction(context.Background(), tx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutil.AssertRowCount(t, db, "ibkr_transaction", 0)
	})

	t.Run("rejects deletion of processed transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		tx := testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		err := svc.DeleteIbkrTransaction(context.Background(), tx.ID)
		if !errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			t.Errorf("expected ErrIBKRTransactionAlreadyProcessed, got %v", err)
		}
	})

	t.Run("returns error for non-existent transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		err := svc.DeleteIbkrTransaction(context.Background(), testutil.MakeID())
		if err == nil {
			t.Fatal("expected error for non-existent transaction")
		}
	})
}

// --- IgnoreIbkrTransaction Tests ---

func TestIbkrService_IgnoreIbkrTransaction(t *testing.T) {
	t.Run("marks pending transaction as ignored", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		tx := testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)

		err := svc.IgnoreIbkrTransaction(context.Background(), tx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var status string
		err = db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, tx.ID).Scan(&status)
		if err != nil {
			t.Fatalf("failed to read status: %v", err)
		}
		if status != "ignored" {
			t.Errorf("expected status=ignored, got %s", status)
		}
	})

	t.Run("rejects ignoring processed transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		tx := testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)

		err := svc.IgnoreIbkrTransaction(context.Background(), tx.ID)
		if !errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			t.Errorf("expected ErrIBKRTransactionAlreadyProcessed, got %v", err)
		}
	})
}

// --- findFundByISINOrSymbol Tests ---

func TestIbkrService_FindFundByISINOrSymbol(t *testing.T) {
	t.Run("finds fund by ISIN", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)

		result, err := svc.ExportFindFundByISINOrSymbol("US0378331005", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != fund.ID {
			t.Errorf("expected fund %s, got %s", fund.ID, result.ID)
		}
	})

	t.Run("finds fund by symbol when ISIN not matched", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)

		result, err := svc.ExportFindFundByISINOrSymbol("NOMATCH", "AAPL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != fund.ID {
			t.Errorf("expected fund %s, got %s", fund.ID, result.ID)
		}
	})

	t.Run("returns ErrIBKRFundNotMatched when no match", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		_, err := svc.ExportFindFundByISINOrSymbol("NOMATCH", "NOMATCH")
		if !errors.Is(err, apperrors.ErrIBKRFundNotMatched) {
			t.Errorf("expected ErrIBKRFundNotMatched, got %v", err)
		}
	})
}

// --- GetEligiblePortfolios Tests ---

func TestIbkrService_GetEligiblePortfolios(t *testing.T) {
	t.Run("returns matched fund and portfolios", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)

		resp, err := svc.GetEligiblePortfolios(ibkrTx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.MatchInfo.Found {
			t.Error("expected matchInfo.Found=true")
		}
		if resp.MatchInfo.MatchedBy != "isin" {
			t.Errorf("expected matchedBy=isin, got %s", resp.MatchInfo.MatchedBy)
		}
		if len(resp.Portfolios) != 1 {
			t.Errorf("expected 1 portfolio, got %d", len(resp.Portfolios))
		}
	})

	t.Run("returns found=false when no fund matches", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		ibkrTx := testutil.NewIBKRTransaction().WithISIN("XX0000000000").WithSymbol("NOPE").Build(t, db)

		resp, err := svc.GetEligiblePortfolios(ibkrTx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.MatchInfo.Found {
			t.Error("expected matchInfo.Found=false")
		}
		if resp.Warning == "" {
			t.Error("expected warning message")
		}
	})

	t.Run("returns warning when fund exists but no portfolios", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		ibkrTx := testutil.NewIBKRTransaction().WithISIN("US0378331005").WithSymbol("AAPL").Build(t, db)

		resp, err := svc.GetEligiblePortfolios(ibkrTx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.MatchInfo.Found {
			t.Error("expected matchInfo.Found=true")
		}
		if resp.Warning == "" {
			t.Error("expected warning about fund not in any portfolio")
		}
		if len(resp.Portfolios) != 0 {
			t.Errorf("expected 0 portfolios, got %d", len(resp.Portfolios))
		}
	})

	t.Run("matches by symbol when ISIN mismatches", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("DE000DIFFERENT").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().WithISIN("XX0000000000").WithSymbol("AAPL").Build(t, db)

		resp, err := svc.GetEligiblePortfolios(ibkrTx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.MatchInfo.MatchedBy != "symbol" {
			t.Errorf("expected matchedBy=symbol, got %s", resp.MatchInfo.MatchedBy)
		}
	})
}

// --- GetTransactionAllocations Tests ---

func TestIbkrService_GetTransactionAllocations(t *testing.T) {
	t.Run("returns error for non-existent transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		_, err := svc.GetTransactionAllocations(testutil.MakeID())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns allocations with aggregated fees", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").
			WithSymbol("AAPL").
			WithStatus("processed").
			WithFees(5.0).
			WithQuantity(10).
			WithPrice(100).
			WithTotalAmount(1000).
			Build(t, db)

		// Create a transaction record for the allocation
		tradeTxn := testutil.NewTransaction(pf.ID).
			WithShares(10).
			WithCostPerShare(100).
			Build(t, db)

		// Insert trade allocation
		_, err := db.Exec(`
			INSERT INTO ibkr_transaction_allocation (id, ibkr_transaction_id, portfolio_id, allocation_percentage, allocated_amount, allocated_shares, transaction_id, created_at)
			VALUES (?, ?, ?, 100, 1000, 10, ?, datetime('now'))`,
			testutil.MakeID(), ibkrTx.ID, portfolio.ID, tradeTxn.ID)
		if err != nil {
			t.Fatalf("failed to insert trade allocation: %v", err)
		}

		// Create fee transaction (type comes from the transaction table)
		feeTxn := testutil.NewTransaction(pf.ID).
			WithShares(0).
			WithCostPerShare(5).
			WithType("fee").
			Build(t, db)

		// Insert fee allocation
		_, err = db.Exec(`
			INSERT INTO ibkr_transaction_allocation (id, ibkr_transaction_id, portfolio_id, allocation_percentage, allocated_amount, allocated_shares, transaction_id, created_at)
			VALUES (?, ?, ?, 100, 5, 0, ?, datetime('now'))`,
			testutil.MakeID(), ibkrTx.ID, portfolio.ID, feeTxn.ID)
		if err != nil {
			t.Fatalf("failed to insert fee allocation: %v", err)
		}

		alloc, err := svc.GetTransactionAllocations(ibkrTx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if alloc.Status != "processed" {
			t.Errorf("expected status=processed, got %s", alloc.Status)
		}
		if len(alloc.Allocations) != 1 {
			t.Fatalf("expected 1 allocation (fees aggregated), got %d", len(alloc.Allocations))
		}
		if alloc.Allocations[0].AllocatedCommission != 5.0 {
			t.Errorf("expected commission=5, got %f", alloc.Allocations[0].AllocatedCommission)
		}
	})
}

// --- parseIBKRFlexReport Tests ---

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestIbkrService_ParseIBKRFlexReport(t *testing.T) {
	t.Run("parses trades from flex report", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		report := ibkr.FlexQueryResponse{
			ImportedAt: time.Now().UTC(),
		}
		report.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				Currency:        "USD",
				CurrencyPrimary: "USD",
				Symbol:          "AAPL",
				Description:     "APPLE INC",
				Isin:            "US0378331005",
				Quantity:        10,
				TradePrice:      150.50,
				IbCommission:    -1.25,
				NetCash:         -1505.00,
				IbOrderID:       123456,
				TransactionID:   789012,
				TradeDate:       "20240115",
				Notes:           "",
				BuySell:         "BUY",
				ReportDate:      "20240115",
			},
		}

		transactions, rates, err := svc.ExportParseIBKRFlexReport(report)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(transactions) != 1 {
			t.Fatalf("expected 1 transaction, got %d", len(transactions))
		}

		tx := transactions[0]
		if tx.Symbol != "AAPL" {
			t.Errorf("expected symbol=AAPL, got %s", tx.Symbol)
		}
		if tx.ISIN != "US0378331005" {
			t.Errorf("expected ISIN=US0378331005, got %s", tx.ISIN)
		}
		if tx.TransactionType != "buy" {
			t.Errorf("expected type=buy, got %s", tx.TransactionType)
		}
		if tx.Quantity != 10 {
			t.Errorf("expected quantity=10, got %f", tx.Quantity)
		}
		if tx.Price != 150.50 {
			t.Errorf("expected price=150.50, got %f", tx.Price)
		}
		if tx.Fees != 1.25 {
			t.Errorf("expected fees=1.25, got %f", tx.Fees)
		}
		if tx.TotalAmount != 1505.00 {
			t.Errorf("expected totalAmount=1505, got %f", tx.TotalAmount)
		}
		if tx.Currency != "USD" {
			t.Errorf("expected currency=USD, got %s", tx.Currency)
		}
		if tx.Status != "pending" {
			t.Errorf("expected status=pending, got %s", tx.Status)
		}
		if tx.IBKRTransactionID != "789012_123456" {
			t.Errorf("expected ibkr id=789012_123456, got %s", tx.IBKRTransactionID)
		}
		if len(rates) != 0 {
			t.Errorf("expected 0 rates, got %d", len(rates))
		}
	})

	t.Run("parses conversion rates", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		report := ibkr.FlexQueryResponse{}
		report.FlexStatements.FlexStatement.ConversionRates.ConversionRate = []struct {
			Text         string  `xml:",chardata"`
			ReportDate   string  `xml:"reportDate,attr"`
			FromCurrency string  `xml:"fromCurrency,attr"`
			ToCurrency   string  `xml:"toCurrency,attr"`
			Rate         float64 `xml:"rate,attr"`
		}{
			{
				ReportDate:   "20240115",
				FromCurrency: "EUR",
				ToCurrency:   "USD",
				Rate:         1.0875,
			},
		}

		transactions, rates, err := svc.ExportParseIBKRFlexReport(report)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(transactions) != 0 {
			t.Errorf("expected 0 transactions, got %d", len(transactions))
		}
		if len(rates) != 1 {
			t.Fatalf("expected 1 rate, got %d", len(rates))
		}
		if rates[0].FromCurrency != "EUR" {
			t.Errorf("expected fromCurrency=EUR, got %s", rates[0].FromCurrency)
		}
		if rates[0].ToCurrency != "USD" {
			t.Errorf("expected toCurrency=USD, got %s", rates[0].ToCurrency)
		}
		if rates[0].Rate != 1.0875 {
			t.Errorf("expected rate=1.0875, got %f", rates[0].Rate)
		}
	})

	t.Run("uses CurrencyPrimary over Currency", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		report := ibkr.FlexQueryResponse{ImportedAt: time.Now().UTC()}
		report.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				Currency:        "GBP",
				CurrencyPrimary: "EUR",
				Symbol:          "TEST",
				Quantity:        5,
				TradePrice:      10,
				NetCash:         -50,
				IbCommission:    -0.5,
				IbOrderID:       1,
				TransactionID:   2,
				TradeDate:       "20240115",
				BuySell:         "BUY",
				ReportDate:      "20240115",
			},
		}

		transactions, _, err := svc.ExportParseIBKRFlexReport(report)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if transactions[0].Currency != "EUR" {
			t.Errorf("expected currency=EUR (CurrencyPrimary), got %s", transactions[0].Currency)
		}
	})

	t.Run("falls back to Currency when CurrencyPrimary is empty", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		report := ibkr.FlexQueryResponse{ImportedAt: time.Now().UTC()}
		report.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				Currency:        "GBP",
				CurrencyPrimary: "",
				Symbol:          "TEST",
				Quantity:        5,
				TradePrice:      10,
				NetCash:         -50,
				IbCommission:    -0.5,
				IbOrderID:       1,
				TransactionID:   2,
				TradeDate:       "20240115",
				BuySell:         "BUY",
				ReportDate:      "20240115",
			},
		}

		transactions, _, err := svc.ExportParseIBKRFlexReport(report)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if transactions[0].Currency != "GBP" {
			t.Errorf("expected currency=GBP (fallback), got %s", transactions[0].Currency)
		}
	})

	t.Run("normalizes quantity and fees to absolute values", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		report := ibkr.FlexQueryResponse{ImportedAt: time.Now().UTC()}
		report.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				Currency:      "USD",
				Symbol:        "TEST",
				Quantity:      -20,
				TradePrice:    50,
				NetCash:       1000,
				IbCommission:  -2.50,
				IbOrderID:     1,
				TransactionID: 2,
				TradeDate:     "20240115",
				BuySell:       "SELL",
				ReportDate:    "20240115",
			},
		}

		transactions, _, err := svc.ExportParseIBKRFlexReport(report)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if transactions[0].Quantity != 20 {
			t.Errorf("expected quantity=20 (abs), got %f", transactions[0].Quantity)
		}
		if transactions[0].Fees != 2.50 {
			t.Errorf("expected fees=2.50 (abs), got %f", transactions[0].Fees)
		}
		if transactions[0].TotalAmount != 1000 {
			t.Errorf("expected totalAmount=1000 (abs), got %f", transactions[0].TotalAmount)
		}
	})

	t.Run("returns error on invalid trade date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		report := ibkr.FlexQueryResponse{}
		report.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				TradeDate:     "bad-date",
				ReportDate:    "20240115",
				TransactionID: 1,
				IbOrderID:     1,
				BuySell:       "BUY",
			},
		}

		_, _, err := svc.ExportParseIBKRFlexReport(report)
		if err == nil {
			t.Fatal("expected error for invalid trade date")
		}
	})
}

// --- AddIbkrTransactions Tests ---

func TestIbkrService_AddIbkrTransactions(t *testing.T) {
	t.Run("adds transactions to database", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		transactions := []model.IBKRTransaction{
			{
				ID:                testutil.MakeID(),
				IBKRTransactionID: "TX_001",
				TransactionDate:   time.Now().UTC(),
				Symbol:            "AAPL",
				ISIN:              "US0378331005",
				Description:       "Buy AAPL",
				TransactionType:   "buy",
				Quantity:          10,
				Price:             150.0,
				TotalAmount:       1500.0,
				Currency:          "USD",
				Fees:              1.0,
				Status:            "pending",
				ImportedAt:        time.Now().UTC(),
				ReportDate:        time.Now().UTC(),
			},
		}

		err := svc.AddIbkrTransactions(context.Background(), transactions)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutil.AssertRowCount(t, db, "ibkr_transaction", 1)
	})
}

// --- writeImportCache Tests ---

func TestIbkrService_WriteImportCache(t *testing.T) {
	t.Run("writes cache entry to database", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		cache := model.IbkrImportCache{
			ID:        testutil.MakeID(),
			CacheKey:  "ibkr_flex_12345_2024-01-15",
			Data:      []byte("<xml>test data</xml>"),
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}

		err := svc.ExportWriteImportCache(context.Background(), cache)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutil.AssertRowCount(t, db, "ibkr_import_cache", 1)
	})
}

// --- addExchangeRates Tests ---

func TestIbkrService_AddExchangeRates(t *testing.T) {
	t.Run("inserts exchange rates", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		devRepo := repository.NewDeveloperRepository(db)
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, &mockIBKRClient{},
			service.IbkrWithDeveloperRepo(devRepo))

		rates := []model.ExchangeRate{
			{
				ID:           testutil.MakeID(),
				Date:         time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				FromCurrency: "EUR",
				ToCurrency:   "USD",
				Rate:         1.0875,
			},
		}

		err := svc.ExportAddExchangeRates(context.Background(), rates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutil.AssertRowCount(t, db, "exchange_rate", 1)
	})
}

// --- AllocateIbkrTransaction Tests ---

func TestIbkrService_AllocateIbkrTransaction(t *testing.T) {
	t.Run("allocates to single portfolio", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").
			WithSymbol("AAPL").
			WithStatus("pending").
			WithQuantity(10).
			WithPrice(100).
			WithTotalAmount(1000).
			WithFees(5).
			Build(t, db)

		allocations := []request.AllocationEntry{
			{PortfolioID: portfolio.ID, Percentage: 100},
		}
		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, allocations)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check ibkr transaction is now processed
		var status string
		if err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if status != "processed" {
			t.Errorf("expected status=processed, got %s", status)
		}

		// Check transaction created
		var txCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM "transaction"`).Scan(&txCount); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if txCount < 1 {
			t.Errorf("expected at least 1 transaction, got %d", txCount)
		}

		// Check allocation created
		testutil.AssertRowCount(t, db, "ibkr_transaction_allocation", 2) // trade + fee
	})

	t.Run("allocates to multiple portfolios", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		p1 := testutil.NewPortfolio().Build(t, db)
		p2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
		testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").
			WithSymbol("AAPL").
			WithStatus("pending").
			WithQuantity(10).
			WithPrice(100).
			WithTotalAmount(1000).
			WithFees(5).
			Build(t, db)

		allocations := []request.AllocationEntry{
			{PortfolioID: p1.ID, Percentage: 60},
			{PortfolioID: p2.ID, Percentage: 40},
		}
		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, allocations)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 2 trade allocations + 2 fee allocations = 4
		testutil.AssertRowCount(t, db, "ibkr_transaction_allocation", 4)
	})

	t.Run("rejects allocation of already processed transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		ibkrTx := testutil.NewIBKRTransaction().WithStatus("processed").Build(t, db)
		allocations := []request.AllocationEntry{
			{PortfolioID: testutil.MakeID(), Percentage: 100},
		}
		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, allocations)
		if !errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			t.Errorf("expected ErrIBKRTransactionAlreadyProcessed, got %v", err)
		}
	})

	t.Run("returns error when no allocations and no default config", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		_ = fund

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").
			WithSymbol("AAPL").
			WithStatus("pending").
			Build(t, db)

		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, nil)
		if !errors.Is(err, apperrors.ErrIBKRInvalidAllocations) {
			t.Errorf("expected ErrIBKRInvalidAllocations, got %v", err)
		}
	})

	t.Run("uses default allocations when none provided", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Set up config with default allocations
		defaultAlloc := fmt.Sprintf(`[{"portfolioId":"%s","percentage":100}]`, portfolio.ID)
		insertIbkrConfigWithDefaultAllocations(t, db, testutil.MakeID(), "token", "12345", true, defaultAlloc)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").
			WithSymbol("AAPL").
			WithStatus("pending").
			WithQuantity(10).
			WithPrice(100).
			WithTotalAmount(1000).
			WithFees(0).
			Build(t, db)

		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var status string
		if err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if status != "processed" {
			t.Errorf("expected status=processed, got %s", status)
		}
	})

	t.Run("creates portfolio_fund if it does not exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		_ = fund
		// Deliberately NOT creating portfolio_fund

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").
			WithSymbol("AAPL").
			WithStatus("pending").
			WithQuantity(10).
			WithPrice(100).
			WithTotalAmount(1000).
			WithFees(0).
			Build(t, db)

		allocations := []request.AllocationEntry{
			{PortfolioID: portfolio.ID, Percentage: 100},
		}
		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, allocations)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// portfolio_fund should have been auto-created
		testutil.AssertRowCount(t, db, "portfolio_fund", 1)
	})

	t.Run("allocation without fees does not create fee record", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").
			WithSymbol("AAPL").
			WithStatus("pending").
			WithQuantity(10).
			WithPrice(100).
			WithTotalAmount(1000).
			WithFees(0).
			Build(t, db)

		allocations := []request.AllocationEntry{
			{PortfolioID: portfolio.ID, Percentage: 100},
		}
		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, allocations)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only 1 allocation (trade), no fee
		testutil.AssertRowCount(t, db, "ibkr_transaction_allocation", 1)
	})
}

// --- BulkAllocateIbkrTransactions Tests ---

func TestIbkrService_BulkAllocateIbkrTransactions(t *testing.T) {
	t.Run("allocates multiple transactions", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx1 := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("pending").WithQuantity(10).WithPrice(100).WithTotalAmount(1000).WithFees(0).
			Build(t, db)
		tx2 := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("pending").WithQuantity(5).WithPrice(200).WithTotalAmount(1000).WithFees(0).
			Build(t, db)

		resp := svc.BulkAllocateIbkrTransactions(context.Background(), request.BulkAllocateRequest{
			TransactionIDs: []string{tx1.ID, tx2.ID},
			Allocations: []request.AllocationEntry{
				{PortfolioID: portfolio.ID, Percentage: 100},
			},
		})

		if resp.Success != 2 {
			t.Errorf("expected 2 successes, got %d", resp.Success)
		}
		if resp.Failed != 0 {
			t.Errorf("expected 0 failures, got %d (errors: %v)", resp.Failed, resp.Errors)
		}
	})

	t.Run("reports partial failures", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx1 := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("pending").WithQuantity(10).WithPrice(100).WithTotalAmount(1000).WithFees(0).
			Build(t, db)

		resp := svc.BulkAllocateIbkrTransactions(context.Background(), request.BulkAllocateRequest{
			TransactionIDs: []string{tx1.ID, testutil.MakeID()},
			Allocations: []request.AllocationEntry{
				{PortfolioID: portfolio.ID, Percentage: 100},
			},
		})

		if resp.Success != 1 {
			t.Errorf("expected 1 success, got %d", resp.Success)
		}
		if resp.Failed != 1 {
			t.Errorf("expected 1 failure, got %d", resp.Failed)
		}
	})
}

// --- UnallocateIbkrTransaction Tests ---

func TestIbkrService_UnallocateIbkrTransaction(t *testing.T) {
	t.Run("reverses allocation", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("pending").WithQuantity(10).WithPrice(100).WithTotalAmount(1000).WithFees(2).
			Build(t, db)

		// First allocate
		allocations := []request.AllocationEntry{
			{PortfolioID: portfolio.ID, Percentage: 100},
		}
		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, allocations)
		if err != nil {
			t.Fatalf("allocation failed: %v", err)
		}

		// Now unallocate
		err = svc.UnallocateIbkrTransaction(context.Background(), ibkrTx.ID)
		if err != nil {
			t.Fatalf("unallocation failed: %v", err)
		}

		// Verify status is back to pending
		var status string
		if err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if status != "pending" {
			t.Errorf("expected status=pending after unallocation, got %s", status)
		}

		// Verify allocations deleted
		testutil.AssertRowCount(t, db, "ibkr_transaction_allocation", 0)
	})

	t.Run("rejects unallocation of non-processed transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		ibkrTx := testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)

		err := svc.UnallocateIbkrTransaction(context.Background(), ibkrTx.ID)
		if !errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			t.Errorf("expected ErrIBKRTransactionAlreadyProcessed, got %v", err)
		}
	})
}

// --- ModifyAllocations Tests ---

func TestIbkrService_ModifyAllocations(t *testing.T) {
	t.Run("changes allocation percentages", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		p1 := testutil.NewPortfolio().Build(t, db)
		p2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
		testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("pending").WithQuantity(10).WithPrice(100).WithTotalAmount(1000).WithFees(0).
			Build(t, db)

		// Initial allocation: 100% to p1
		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, []request.AllocationEntry{
			{PortfolioID: p1.ID, Percentage: 100},
		})
		if err != nil {
			t.Fatalf("initial allocation failed: %v", err)
		}

		// Modify: 60% p1, 40% p2
		err = svc.ModifyAllocations(context.Background(), ibkrTx.ID, []request.AllocationEntry{
			{PortfolioID: p1.ID, Percentage: 60},
			{PortfolioID: p2.ID, Percentage: 40},
		})
		if err != nil {
			t.Fatalf("modify allocations failed: %v", err)
		}

		// Verify still processed
		var status string
		if err := db.QueryRow(`SELECT status FROM ibkr_transaction WHERE id = ?`, ibkrTx.ID).Scan(&status); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if status != "processed" {
			t.Errorf("expected status=processed, got %s", status)
		}

		// Verify new allocation count (2 trade allocs, no fees since fees=0)
		testutil.AssertRowCount(t, db, "ibkr_transaction_allocation", 2)
	})
}

// --- MatchDividend Tests ---

func TestIbkrService_MatchDividend(t *testing.T) {
	t.Run("matches dividend to allocated IBKR transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").
			WithDividendType("distributing").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create a pending dividend
		div := testutil.NewDividend(fund.ID, pf.ID).Build(t, db)

		// Create and allocate an IBKR DRIP transaction
		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("pending").
			WithType("buy").
			WithQuantity(5).WithPrice(100).WithTotalAmount(500).WithFees(0).
			WithNotes("Ri").
			Build(t, db)

		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, []request.AllocationEntry{
			{PortfolioID: portfolio.ID, Percentage: 100},
		})
		if err != nil {
			t.Fatalf("allocation failed: %v", err)
		}

		// Match dividend
		err = svc.MatchDividend(context.Background(), ibkrTx.ID, []string{div.ID})
		if err != nil {
			t.Fatalf("match dividend failed: %v", err)
		}

		// Verify dividend was updated
		var reinvestStatus string
		var reinvTxID string
		if err := db.QueryRow(`SELECT reinvestment_status, reinvestment_transaction_id FROM dividend WHERE id = ?`, div.ID).
			Scan(&reinvestStatus, &reinvTxID); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if reinvestStatus != "COMPLETED" {
			t.Errorf("expected reinvestment_status=COMPLETED, got %s", reinvestStatus)
		}
		if reinvTxID == "" {
			t.Error("expected reinvestment_transaction_id to be set")
		}
	})

	t.Run("rejects matching on non-processed transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		ibkrTx := testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)

		err := svc.MatchDividend(context.Background(), ibkrTx.ID, []string{testutil.MakeID()})
		if !errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			t.Errorf("expected ErrIBKRTransactionAlreadyProcessed, got %v", err)
		}
	})

	t.Run("rejects already matched dividend", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create a transaction for the reinvestment ref
		existingTxn := testutil.NewTransaction(pf.ID).Build(t, db)

		// Create a dividend that's already matched
		div := testutil.NewDividend(fund.ID, pf.ID).
			WithReinvestmentTransaction(existingTxn.ID).
			Build(t, db)

		// Create and allocate an IBKR transaction
		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("pending").WithType("buy").
			WithQuantity(5).WithPrice(100).WithTotalAmount(500).WithFees(0).
			WithNotes("Ri").
			Build(t, db)

		err := svc.AllocateIbkrTransaction(context.Background(), ibkrTx.ID, []request.AllocationEntry{
			{PortfolioID: portfolio.ID, Percentage: 100},
		})
		if err != nil {
			t.Fatalf("allocation failed: %v", err)
		}

		err = svc.MatchDividend(context.Background(), ibkrTx.ID, []string{div.ID})
		if err == nil {
			t.Fatal("expected error for already matched dividend")
		}
	})
}

// --- ImportFlexReport Tests ---

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestIbkrService_ImportFlexReport(t *testing.T) {
	t.Run("imports flex report from mock client", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)

		// Encrypt a token for the config
		plainToken := "test-ibkr-token" //nolint:gosec // G101: Test credential, not a real secret
		encToken, err := fernet.EncryptAndSign([]byte(plainToken), key)
		if err != nil {
			t.Fatalf("failed to encrypt token: %v", err)
		}

		insertIbkrConfig(t, db, testutil.MakeID(), string(encToken), "54321", true)

		flexResponse := ibkr.FlexQueryResponse{
			ImportedAt: time.Now().UTC(),
			QueryID:    54321,
		}
		flexResponse.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				CurrencyPrimary: "USD",
				Symbol:          "AAPL",
				Isin:            "US0378331005",
				Quantity:        10,
				TradePrice:      150,
				IbCommission:    -1,
				NetCash:         -1500,
				IbOrderID:       100,
				TransactionID:   200,
				TradeDate:       "20240115",
				BuySell:         "BUY",
				ReportDate:      "20240115",
			},
		}

		rawXML := []byte(`<FlexQueryResponse><FlexStatements count="1"><FlexStatement><Trades></Trades></FlexStatement></FlexStatements></FlexQueryResponse>`)

		mock := &mockIBKRClient{
			retreiveFunc: func(_ context.Context, token, queryID string) (ibkr.FlexQueryResponse, []byte, error) {
				if token != plainToken {
					t.Errorf("expected decrypted token %q, got %q", plainToken, token)
				}
				if queryID != "54321" {
					t.Errorf("expected queryID=54321, got %s", queryID)
				}
				return flexResponse, rawXML, nil
			},
		}

		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, mock,
			service.IbkrWithEncryptionKey(key))

		imported, skipped, err := svc.ImportFlexReport(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if imported != 1 {
			t.Errorf("expected 1 imported, got %d", imported)
		}
		if skipped != 0 {
			t.Errorf("expected 0 skipped, got %d", skipped)
		}

		testutil.AssertRowCount(t, db, "ibkr_transaction", 1)
	})

	t.Run("skips duplicate transactions on re-import", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)

		plainToken := "test-ibkr-token" //nolint:gosec // G101: Test credential, not a real secret
		encToken, err := fernet.EncryptAndSign([]byte(plainToken), key)
		if err != nil {
			t.Fatalf("failed to encrypt token: %v", err)
		}
		insertIbkrConfig(t, db, testutil.MakeID(), string(encToken), "54321", true)

		// Pre-insert transaction with same IBKR ID
		testutil.NewIBKRTransaction().
			WithStatus("pending").
			Build(t, db)

		existingIBKRTxID := "200_100"
		_, err = db.Exec(`UPDATE ibkr_transaction SET ibkr_transaction_id = ? WHERE 1=1`, existingIBKRTxID)
		if err != nil {
			t.Fatalf("failed to update ibkr transaction id: %v", err)
		}

		flexResponse := ibkr.FlexQueryResponse{
			ImportedAt: time.Now().UTC(),
			QueryID:    54321,
		}
		flexResponse.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				CurrencyPrimary: "USD",
				Symbol:          "AAPL",
				Quantity:        10,
				TradePrice:      150,
				IbCommission:    -1,
				NetCash:         -1500,
				IbOrderID:       100,
				TransactionID:   200,
				TradeDate:       "20240115",
				BuySell:         "BUY",
				ReportDate:      "20240115",
			},
		}

		mock := &mockIBKRClient{
			retreiveFunc: func(_ context.Context, _, _ string) (ibkr.FlexQueryResponse, []byte, error) {
				return flexResponse, []byte(`<xml/>`), nil
			},
		}

		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, mock,
			service.IbkrWithEncryptionKey(key))

		imported, skipped, err := svc.ImportFlexReport(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if imported != 0 {
			t.Errorf("expected 0 imported (duplicate), got %d", imported)
		}
		if skipped != 1 {
			t.Errorf("expected 1 skipped, got %d", skipped)
		}

		// Should still be 1 total
		testutil.AssertRowCount(t, db, "ibkr_transaction", 1)
	})

	t.Run("auto-allocates when default allocation enabled", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		key := generateFernetKey(t)

		plainToken := "test-ibkr-token" //nolint:gosec // G101: Test credential, not a real secret
		encToken, err := fernet.EncryptAndSign([]byte(plainToken), key)
		if err != nil {
			t.Fatalf("failed to encrypt token: %v", err)
		}

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		defaultAlloc := fmt.Sprintf(`[{"portfolioId":"%s","percentage":100}]`, portfolio.ID)
		insertIbkrConfigWithDefaultAllocations(t, db, testutil.MakeID(), string(encToken), "54321", true, defaultAlloc)

		flexResponse := ibkr.FlexQueryResponse{
			ImportedAt: time.Now().UTC(),
			QueryID:    54321,
		}
		flexResponse.FlexStatements.FlexStatement.Trades.Trade = []struct {
			Text            string  `xml:",chardata"`
			Currency        string  `xml:"currency,attr"`
			CurrencyPrimary string  `xml:"currencyPrimary,attr"`
			Symbol          string  `xml:"symbol,attr"`
			Description     string  `xml:"description,attr"`
			Isin            string  `xml:"isin,attr"`
			Quantity        float64 `xml:"quantity,attr"`
			TradePrice      float64 `xml:"tradePrice,attr"`
			IbCommission    float64 `xml:"ibCommission,attr"`
			NetCash         float64 `xml:"netCash,attr"`
			IbOrderID       int64   `xml:"ibOrderID,attr"`
			TransactionID   int64   `xml:"transactionID,attr"`
			TradeDate       string  `xml:"tradeDate,attr"`
			Notes           string  `xml:"notes,attr"`
			BuySell         string  `xml:"buySell,attr"`
			ReportDate      string  `xml:"reportDate,attr"`
		}{
			{
				CurrencyPrimary: "USD",
				Symbol:          "AAPL",
				Isin:            "US0378331005",
				Quantity:        10,
				TradePrice:      150,
				IbCommission:    -1,
				NetCash:         -1500,
				IbOrderID:       100,
				TransactionID:   200,
				TradeDate:       "20240115",
				BuySell:         "BUY",
				ReportDate:      "20240115",
			},
		}

		mock := &mockIBKRClient{
			retreiveFunc: func(_ context.Context, _, _ string) (ibkr.FlexQueryResponse, []byte, error) {
				return flexResponse, []byte(`<xml/>`), nil
			},
		}

		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, mock,
			service.IbkrWithEncryptionKey(key))

		imported, _, err := svc.ImportFlexReport(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if imported != 1 {
			t.Errorf("expected 1 imported, got %d", imported)
		}

		// Transaction should be auto-allocated (status = processed)
		var status string
		if err := db.QueryRow(`SELECT status FROM ibkr_transaction LIMIT 1`).Scan(&status); err != nil {
			t.Fatalf("scan status: %v", err)
		}
		if status != "processed" {
			t.Errorf("expected status=processed after auto-allocation, got %s", status)
		}

		// Should have allocation records (trade + fee)
		testutil.AssertRowCount(t, db, "ibkr_transaction_allocation", 2)

		// Should have created transactions
		var txCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM "transaction"`).Scan(&txCount); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if txCount < 1 {
			t.Errorf("expected at least 1 transaction, got %d", txCount)
		}
	})

	t.Run("returns error when no config", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		mock := &mockIBKRClient{}
		svc := testutil.NewTestIbkrServiceWithMockIBKR(t, db, mock)

		// No config, cache lookup will fail, and then it will try decryptToken without key
		_, _, err := svc.ImportFlexReport(context.Background())
		if err == nil {
			t.Fatal("expected error when no config/key")
		}
	})
}

// --- GetIbkrTransactionDetail Tests ---

func TestIbkrService_GetIbkrTransactionDetail(t *testing.T) {
	t.Run("returns pending transaction without allocations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		ibkrTx := testutil.NewIBKRTransaction().WithStatus("pending").Build(t, db)

		detail, err := svc.GetIbkrTransactionDetail(ibkrTx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.ID != ibkrTx.ID {
			t.Errorf("expected ID=%s, got %s", ibkrTx.ID, detail.ID)
		}
		if detail.Allocations != nil {
			t.Error("expected nil allocations for pending transaction")
		}
	})

	t.Run("returns processed transaction with allocations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithISIN("US0378331005").WithSymbol("AAPL.NASDAQ").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		ibkrTx := testutil.NewIBKRTransaction().
			WithISIN("US0378331005").WithSymbol("AAPL").
			WithStatus("processed").WithQuantity(10).WithPrice(100).WithTotalAmount(1000).WithFees(0).
			Build(t, db)

		tradeTxn := testutil.NewTransaction(pf.ID).WithShares(10).WithCostPerShare(100).Build(t, db)

		_, err := db.Exec(`
			INSERT INTO ibkr_transaction_allocation (id, ibkr_transaction_id, portfolio_id, allocation_percentage, allocated_amount, allocated_shares, transaction_id, created_at)
			VALUES (?, ?, ?, 100, 1000, 10, ?, datetime('now'))`,
			testutil.MakeID(), ibkrTx.ID, portfolio.ID, tradeTxn.ID)
		if err != nil {
			t.Fatalf("failed to insert allocation: %v", err)
		}

		detail, err := svc.GetIbkrTransactionDetail(ibkrTx.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(detail.Allocations) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(detail.Allocations))
		}
	})

	t.Run("returns error for non-existent transaction", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestIbkrService(t, db)

		_, err := svc.GetIbkrTransactionDetail(testutil.MakeID())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
