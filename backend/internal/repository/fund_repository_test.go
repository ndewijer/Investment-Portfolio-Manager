package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

func TestFundRepository_GetAllFunds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	t.Run("returns empty slice when no funds exist", func(t *testing.T) {
		result, err := repo.GetAllFunds()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty slice, got %d items", len(result))
		}
	})

	fund1 := testutil.NewFund().WithName("Fund A").Build(t, db)
	fund2 := testutil.NewFund().WithName("Fund B").Build(t, db)

	t.Run("returns all funds", func(t *testing.T) {
		result, err := repo.GetAllFunds()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 funds, got %d", len(result))
		}
	})

	t.Run("includes latest price when fund_price exists", func(t *testing.T) {
		testutil.NewFundPrice(fund1.ID).WithPrice(100.50).WithDate(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewFundPrice(fund1.ID).WithPrice(105.75).WithDate(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)).Build(t, db)

		result, err := repo.GetAllFunds()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, f := range result {
			if f.ID == fund1.ID {
				if f.LatestPrice != 105.75 {
					t.Errorf("expected latest price 105.75, got %f", f.LatestPrice)
				}
				return
			}
		}
		t.Error("fund1 not found in results")
	})

	t.Run("returns zero price when no fund_price exists", func(t *testing.T) {
		result, err := repo.GetAllFunds()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, f := range result {
			if f.ID == fund2.ID {
				if f.LatestPrice != 0 {
					t.Errorf("expected latest price 0, got %f", f.LatestPrice)
				}
				return
			}
		}
		t.Error("fund2 not found in results")
	})
}

func TestFundRepository_GetFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	fund := testutil.NewFund().
		WithName("Apple Inc").
		WithSymbol("AAPL.NASDAQ").
		WithISIN("US0378331005").
		WithCurrency("USD").
		WithExchange("NASDAQ").
		WithInvestmentType("STOCK").
		WithDividendType("CASH").
		Build(t, db)

	t.Run("returns fund with all fields", func(t *testing.T) {
		result, err := repo.GetFund(fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != fund.ID {
			t.Errorf("expected ID %s, got %s", fund.ID, result.ID)
		}
		if result.Name != "Apple Inc" {
			t.Errorf("expected Name 'Apple Inc', got %s", result.Name)
		}
		if result.Symbol != "AAPL.NASDAQ" {
			t.Errorf("expected Symbol 'AAPL.NASDAQ', got %s", result.Symbol)
		}
		if result.Isin != "US0378331005" {
			t.Errorf("expected Isin 'US0378331005', got %s", result.Isin)
		}
		if result.Currency != "USD" {
			t.Errorf("expected Currency 'USD', got %s", result.Currency)
		}
		if result.InvestmentType != "STOCK" {
			t.Errorf("expected InvestmentType 'STOCK', got %s", result.InvestmentType)
		}
		if result.DividendType != "CASH" {
			t.Errorf("expected DividendType 'CASH', got %s", result.DividendType)
		}
	})

	t.Run("returns ErrFundNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetFund("non-existent-id")
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound, got %v", err)
		}
	})

	t.Run("includes latest price", func(t *testing.T) {
		testutil.NewFundPrice(fund.ID).WithPrice(150.0).WithDate(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewFundPrice(fund.ID).WithPrice(155.0).WithDate(time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC)).Build(t, db)

		result, err := repo.GetFund(fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.LatestPrice != 155.0 {
			t.Errorf("expected latest price 155.0, got %f", result.LatestPrice)
		}
	})
}

func TestFundRepository_GetFundPrice(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	fund1 := testutil.NewFund().Build(t, db)
	fund2 := testutil.NewFund().Build(t, db)

	d1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)

	testutil.NewFundPrice(fund1.ID).WithDate(d1).WithPrice(10.0).Build(t, db)
	testutil.NewFundPrice(fund1.ID).WithDate(d2).WithPrice(11.0).Build(t, db)
	testutil.NewFundPrice(fund1.ID).WithDate(d3).WithPrice(12.0).Build(t, db)
	testutil.NewFundPrice(fund2.ID).WithDate(d1).WithPrice(20.0).Build(t, db)
	testutil.NewFundPrice(fund2.ID).WithDate(d2).WithPrice(21.0).Build(t, db)

	t.Run("returns prices grouped by fund within date range", func(t *testing.T) {
		result, err := repo.GetFundPrice([]string{fund1.ID, fund2.ID}, d1, d3, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[fund1.ID]) != 3 {
			t.Errorf("expected 3 prices for fund1, got %d", len(result[fund1.ID]))
		}
		if len(result[fund2.ID]) != 2 {
			t.Errorf("expected 2 prices for fund2, got %d", len(result[fund2.ID]))
		}
	})

	t.Run("ascending order returns oldest first", func(t *testing.T) {
		result, err := repo.GetFundPrice([]string{fund1.ID}, d1, d3, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		prices := result[fund1.ID]
		if prices[0].Price != 10.0 || prices[2].Price != 12.0 {
			t.Errorf("expected ascending order: 10, 11, 12; got %f, %f, %f", prices[0].Price, prices[1].Price, prices[2].Price)
		}
	})

	t.Run("descending order returns newest first", func(t *testing.T) {
		result, err := repo.GetFundPrice([]string{fund1.ID}, d1, d3, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		prices := result[fund1.ID]
		if prices[0].Price != 12.0 || prices[2].Price != 10.0 {
			t.Errorf("expected descending order: 12, 11, 10; got %f, %f, %f", prices[0].Price, prices[1].Price, prices[2].Price)
		}
	})

	t.Run("respects date range boundaries", func(t *testing.T) {
		result, err := repo.GetFundPrice([]string{fund1.ID}, d1, d2, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[fund1.ID]) != 2 {
			t.Errorf("expected 2 prices in range, got %d", len(result[fund1.ID]))
		}
	})

	t.Run("returns empty map for fund with no prices in range", func(t *testing.T) {
		future := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		futureEnd := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)
		result, err := repo.GetFundPrice([]string{fund1.ID}, future, futureEnd, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[fund1.ID]) != 0 {
			t.Errorf("expected 0 prices, got %d", len(result[fund1.ID]))
		}
	})

	t.Run("returns error when startDate is after endDate", func(t *testing.T) {
		_, err := repo.GetFundPrice([]string{fund1.ID}, d3, d1, true)
		if err == nil {
			t.Fatal("expected error for invalid date range, got nil")
		}
	})
}

func TestFundRepository_GetFundBySymbolOrIsin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	fund := testutil.NewFund().
		WithSymbol("AAPL.NASDAQ").
		WithISIN("US0378331005").
		Build(t, db)

	t.Run("finds fund by symbol (strips exchange suffix)", func(t *testing.T) {
		result, err := repo.GetFundBySymbolOrIsin("AAPL", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != fund.ID {
			t.Errorf("expected fund %s, got %s", fund.ID, result.ID)
		}
	})

	t.Run("finds fund by ISIN", func(t *testing.T) {
		result, err := repo.GetFundBySymbolOrIsin("", "US0378331005")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != fund.ID {
			t.Errorf("expected fund %s, got %s", fund.ID, result.ID)
		}
	})

	t.Run("finds fund when both symbol and ISIN match", func(t *testing.T) {
		result, err := repo.GetFundBySymbolOrIsin("AAPL", "US0378331005")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != fund.ID {
			t.Errorf("expected fund %s, got %s", fund.ID, result.ID)
		}
	})

	t.Run("returns ErrFundNotFound when neither match", func(t *testing.T) {
		_, err := repo.GetFundBySymbolOrIsin("NONEXIST", "XX0000000000")
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound, got %v", err)
		}
	})

	t.Run("returns error when both symbol and ISIN are empty", func(t *testing.T) {
		_, err := repo.GetFundBySymbolOrIsin("", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFundRepository_InsertFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)
	ctx := context.Background()

	t.Run("inserts a fund successfully", func(t *testing.T) {
		f := &model.Fund{
			ID:             testutil.MakeID(),
			Name:           "Test Fund",
			Isin:           "US1234567890",
			Symbol:         "TEST.NYSE",
			Currency:       "USD",
			Exchange:       "NYSE",
			InvestmentType: "ETF",
			DividendType:   "CASH",
		}
		err := repo.InsertFund(ctx, f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetFund(f.ID)
		if err != nil {
			t.Fatalf("failed to retrieve inserted fund: %v", err)
		}
		if result.Name != "Test Fund" {
			t.Errorf("expected name 'Test Fund', got %s", result.Name)
		}
		if result.InvestmentType != "ETF" {
			t.Errorf("expected investment type 'ETF', got %s", result.InvestmentType)
		}
	})

	t.Run("fails on duplicate ID", func(t *testing.T) {
		id := testutil.MakeID()
		f1 := &model.Fund{ID: id, Name: "First", Isin: testutil.MakeISIN("US"), Symbol: testutil.MakeSymbol("A")}
		err := repo.InsertFund(ctx, f1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		f2 := &model.Fund{ID: id, Name: "Duplicate", Isin: testutil.MakeISIN("US"), Symbol: testutil.MakeSymbol("B")}
		err = repo.InsertFund(ctx, f2)
		if err == nil {
			t.Fatal("expected error on duplicate insert, got nil")
		}
	})
}

func TestFundRepository_UpdateFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)
	ctx := context.Background()

	fund := testutil.NewFund().WithName("Original Fund").Build(t, db)

	t.Run("updates an existing fund", func(t *testing.T) {
		updated := &model.Fund{
			ID:             fund.ID,
			Name:           "Updated Fund",
			Isin:           "GB9999999999",
			Symbol:         "UPD.LSE",
			Currency:       "GBP",
			Exchange:       "LSE",
			InvestmentType: "BOND",
			DividendType:   "NONE",
		}
		err := repo.UpdateFund(ctx, updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetFund(fund.ID)
		if err != nil {
			t.Fatalf("failed to retrieve updated fund: %v", err)
		}
		if result.Name != "Updated Fund" {
			t.Errorf("expected name 'Updated Fund', got %s", result.Name)
		}
		if result.Currency != "GBP" {
			t.Errorf("expected currency 'GBP', got %s", result.Currency)
		}
		if result.InvestmentType != "BOND" {
			t.Errorf("expected investment type 'BOND', got %s", result.InvestmentType)
		}
	})

	t.Run("returns ErrFundNotFound for non-existent ID", func(t *testing.T) {
		err := repo.UpdateFund(ctx, &model.Fund{ID: "non-existent-id", Name: "X"})
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound, got %v", err)
		}
	})
}

func TestFundRepository_DeleteFund(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)
	ctx := context.Background()

	fund := testutil.NewFund().Build(t, db)

	t.Run("deletes an existing fund", func(t *testing.T) {
		err := repo.DeleteFund(ctx, fund.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = repo.GetFund(fund.ID)
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound after delete, got %v", err)
		}
	})

	t.Run("returns ErrFundNotFound for non-existent ID", func(t *testing.T) {
		err := repo.DeleteFund(ctx, "non-existent-id")
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound, got %v", err)
		}
	})
}

func TestFundRepository_InsertFundPrice(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)
	ctx := context.Background()

	fund := testutil.NewFund().Build(t, db)

	t.Run("inserts a fund price", func(t *testing.T) {
		fp := model.FundPrice{
			ID:     testutil.MakeID(),
			FundID: fund.ID,
			Date:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			Price:  42.50,
		}
		err := repo.InsertFundPrice(ctx, fp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify via GetFundPrice
		result, err := repo.GetFundPrice(
			[]string{fund.ID},
			time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			true,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		prices := result[fund.ID]
		if len(prices) != 1 {
			t.Fatalf("expected 1 price, got %d", len(prices))
		}
		if prices[0].Price != 42.50 {
			t.Errorf("expected price 42.50, got %f", prices[0].Price)
		}
	})
}

func TestFundRepository_InsertFundPrices(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)
	ctx := context.Background()

	fund := testutil.NewFund().Build(t, db)

	t.Run("no-op on empty slice", func(t *testing.T) {
		err := repo.InsertFundPrices(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("inserts multiple prices in batch", func(t *testing.T) {
		prices := []model.FundPrice{
			{ID: testutil.MakeID(), FundID: fund.ID, Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), Price: 50.0},
			{ID: testutil.MakeID(), FundID: fund.ID, Date: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC), Price: 51.0},
			{ID: testutil.MakeID(), FundID: fund.ID, Date: time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC), Price: 52.0},
		}
		err := repo.InsertFundPrices(ctx, prices)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetFundPrice(
			[]string{fund.ID},
			time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
			true,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[fund.ID]) != 3 {
			t.Errorf("expected 3 prices, got %d", len(result[fund.ID]))
		}
	})
}

func TestFundRepository_UpdateFundPrice(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)
	ctx := context.Background()

	fund := testutil.NewFund().Build(t, db)
	date := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)

	t.Run("inserts when no conflict", func(t *testing.T) {
		fp := model.FundPrice{
			ID:     testutil.MakeID(),
			FundID: fund.ID,
			Date:   date,
			Price:  100.0,
		}
		err := repo.UpdateFundPrice(ctx, fp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetFundPrice([]string{fund.ID}, date, date, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result[fund.ID][0].Price != 100.0 {
			t.Errorf("expected price 100.0, got %f", result[fund.ID][0].Price)
		}
	})

	t.Run("updates price on conflict (same fund_id and date)", func(t *testing.T) {
		fp := model.FundPrice{
			ID:     testutil.MakeID(),
			FundID: fund.ID,
			Date:   date,
			Price:  200.0,
		}
		err := repo.UpdateFundPrice(ctx, fp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetFundPrice([]string{fund.ID}, date, date, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[fund.ID]) != 1 {
			t.Fatalf("expected 1 price after upsert, got %d", len(result[fund.ID]))
		}
		if result[fund.ID][0].Price != 200.0 {
			t.Errorf("expected updated price 200.0, got %f", result[fund.ID][0].Price)
		}
	})
}

func TestFundRepository_GetSymbol(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	testutil.NewSymbol().WithSymbol("AAPL").WithName("Apple Inc").WithExchange("NASDAQ").WithCurrency("USD").Build(t, db)

	t.Run("returns symbol when found", func(t *testing.T) {
		result, err := repo.GetSymbol("AAPL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Symbol != "AAPL" {
			t.Errorf("expected symbol 'AAPL', got %s", result.Symbol)
		}
		if result.Name != "Apple Inc" {
			t.Errorf("expected name 'Apple Inc', got %s", result.Name)
		}
	})

	t.Run("returns ErrSymbolNotFound for unknown symbol", func(t *testing.T) {
		_, err := repo.GetSymbol("NONEXIST")
		if !errors.Is(err, apperrors.ErrSymbolNotFound) {
			t.Errorf("expected ErrSymbolNotFound, got %v", err)
		}
	})

	t.Run("returns nil for empty symbol string", func(t *testing.T) {
		result, err := repo.GetSymbol("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for empty symbol, got %v", result)
		}
	})
}

func TestFundRepository_UpsertSymbol(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	t.Run("inserts a new symbol", func(t *testing.T) {
		s := &model.Symbol{
			ID:          testutil.MakeID(),
			Symbol:      "TSLA",
			Name:        "Tesla Inc",
			Exchange:    "NASDAQ",
			Currency:    "USD",
			Isin:        "US88160R1014",
			LastUpdated: time.Now().UTC(),
			DataSource:  "yahoo",
			IsValid:     true,
		}
		err := repo.UpsertSymbol(s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetSymbol("TSLA")
		if err != nil {
			t.Fatalf("failed to get symbol: %v", err)
		}
		if result.Name != "Tesla Inc" {
			t.Errorf("expected name 'Tesla Inc', got %s", result.Name)
		}
	})

	t.Run("updates existing symbol on conflict", func(t *testing.T) {
		s := &model.Symbol{
			ID:          testutil.MakeID(),
			Symbol:      "TSLA",
			Name:        "Tesla Inc Updated",
			Exchange:    "NYSE",
			Currency:    "EUR",
			Isin:        "US88160R1014",
			LastUpdated: time.Now().UTC(),
			DataSource:  "manual",
			IsValid:     false,
		}
		err := repo.UpsertSymbol(s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.GetSymbol("TSLA")
		if err != nil {
			t.Fatalf("failed to get symbol: %v", err)
		}
		if result.Name != "Tesla Inc Updated" {
			t.Errorf("expected name 'Tesla Inc Updated', got %s", result.Name)
		}
		if result.Exchange != "NYSE" {
			t.Errorf("expected exchange 'NYSE', got %s", result.Exchange)
		}
		if result.DataSource != "manual" {
			t.Errorf("expected data source 'manual', got %s", result.DataSource)
		}
	})
}

func TestFundRepository_PruneStaleSymbols(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	t.Run("prunes old symbols not referenced by funds", func(t *testing.T) {
		// Insert a stale symbol (older than 24 hours)
		_, err := db.Exec(`
			INSERT INTO symbol_info (id, symbol, name, last_updated, is_valid)
			VALUES (?, 'STALE', 'Stale Symbol', datetime('now', '-2 days'), 1)
		`, testutil.MakeID())
		if err != nil {
			t.Fatalf("failed to insert stale symbol: %v", err)
		}

		// Insert a fresh symbol
		_, err = db.Exec(`
			INSERT INTO symbol_info (id, symbol, name, last_updated, is_valid)
			VALUES (?, 'FRESH', 'Fresh Symbol', datetime('now'), 1)
		`, testutil.MakeID())
		if err != nil {
			t.Fatalf("failed to insert fresh symbol: %v", err)
		}

		n, err := repo.PruneStaleSymbols()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 1 {
			t.Errorf("expected 1 pruned symbol, got %d", n)
		}

		// Fresh symbol should still exist
		result, err := repo.GetSymbol("FRESH")
		if err != nil {
			t.Fatalf("fresh symbol should still exist: %v", err)
		}
		if result == nil {
			t.Error("expected fresh symbol to remain")
		}
	})

	t.Run("does not prune symbols referenced by funds", func(t *testing.T) {
		// Create a fund that references a symbol
		testutil.NewFund().WithSymbol("REFERENCED").Build(t, db)

		// Insert a stale symbol with matching symbol
		_, err := db.Exec(`
			INSERT INTO symbol_info (id, symbol, name, last_updated, is_valid)
			VALUES (?, 'REFERENCED', 'Referenced Symbol', datetime('now', '-2 days'), 1)
		`, testutil.MakeID())
		if err != nil {
			t.Fatalf("failed to insert referenced symbol: %v", err)
		}

		n, err := repo.PruneStaleSymbols()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 0 {
			t.Errorf("expected 0 pruned symbols (referenced by fund), got %d", n)
		}
	})
}

func TestFundRepository_WithTx(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)
	ctx := context.Background()

	t.Run("committed transaction persists fund insert", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		f := &model.Fund{
			ID:             testutil.MakeID(),
			Name:           "TxFund",
			Isin:           testutil.MakeISIN("US"),
			Symbol:         testutil.MakeSymbol("TX"),
			Currency:       "USD",
			Exchange:       "NYSE",
			InvestmentType: "STOCK",
			DividendType:   "NONE",
		}
		err = txRepo.InsertFund(ctx, f)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		result, err := repo.GetFund(f.ID)
		if err != nil {
			t.Fatalf("expected fund to exist after commit: %v", err)
		}
		if result.Name != "TxFund" {
			t.Errorf("expected name 'TxFund', got %s", result.Name)
		}
	})

	t.Run("rolled-back transaction does not persist fund insert", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		txRepo := repo.WithTx(tx)
		f := &model.Fund{
			ID:             testutil.MakeID(),
			Name:           "RollbackFund",
			Isin:           testutil.MakeISIN("US"),
			Symbol:         testutil.MakeSymbol("RB"),
			Currency:       "USD",
			Exchange:       "NYSE",
			InvestmentType: "STOCK",
			DividendType:   "NONE",
		}
		err = txRepo.InsertFund(ctx, f)
		if err != nil {
			_ = tx.Rollback() //nolint:errcheck // rollback in test cleanup
			t.Fatalf("unexpected error: %v", err)
		}

		err = tx.Rollback()
		if err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}

		_, err = repo.GetFund(f.ID)
		if !errors.Is(err, apperrors.ErrFundNotFound) {
			t.Errorf("expected ErrFundNotFound after rollback, got %v", err)
		}
	})
}

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestFundRepository_GetFunds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewFundRepository(db)

	fund1 := testutil.NewFund().WithName("Alpha Fund").WithCurrency("USD").Build(t, db)
	fund2 := testutil.NewFund().WithName("Beta Fund").WithCurrency("EUR").Build(t, db)
	fund3 := testutil.NewFund().WithName("Gamma Fund").WithCurrency("GBP").Build(t, db)

	t.Run("returns multiple funds by IDs", func(t *testing.T) {
		result, err := repo.GetFunds([]string{fund1.ID, fund2.ID, fund3.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 funds, got %d", len(result))
		}

		// Verify fund data is correct
		found := make(map[string]bool)
		for _, f := range result {
			found[f.ID] = true
		}
		for _, id := range []string{fund1.ID, fund2.ID, fund3.ID} {
			if !found[id] {
				t.Errorf("expected fund %s in results", id)
			}
		}
	})

	t.Run("returns single fund", func(t *testing.T) {
		result, err := repo.GetFunds([]string{fund2.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(result))
		}
		if result[0].ID != fund2.ID {
			t.Errorf("expected fund ID %s, got %s", fund2.ID, result[0].ID)
		}
		if result[0].Name != "Beta Fund" {
			t.Errorf("expected name 'Beta Fund', got %s", result[0].Name)
		}
		if result[0].Currency != "EUR" {
			t.Errorf("expected currency 'EUR', got %s", result[0].Currency)
		}
	})

	t.Run("returns empty slice for nonexistent IDs", func(t *testing.T) {
		result, err := repo.GetFunds([]string{testutil.MakeID(), testutil.MakeID()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 funds, got %d", len(result))
		}
	})

	t.Run("includes latest price from fund_price LEFT JOIN", func(t *testing.T) {
		testutil.NewFundPrice(fund1.ID).WithPrice(99.99).WithDate(time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)).Build(t, db)
		testutil.NewFundPrice(fund1.ID).WithPrice(105.50).WithDate(time.Date(2025, 5, 2, 0, 0, 0, 0, time.UTC)).Build(t, db)

		result, err := repo.GetFunds([]string{fund1.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(result))
		}
		if result[0].LatestPrice != 105.50 {
			t.Errorf("expected latest price 105.50, got %f", result[0].LatestPrice)
		}
	})

	t.Run("returns zero price for fund without prices", func(t *testing.T) {
		result, err := repo.GetFunds([]string{fund3.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(result))
		}
		if result[0].LatestPrice != 0 {
			t.Errorf("expected 0 price for fund without prices, got %f", result[0].LatestPrice)
		}
	})
}
