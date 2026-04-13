package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// ---------------------------------------------------------------------------
// InsertMaterializedEntries
// ---------------------------------------------------------------------------

func TestMaterializedRepository_InsertMaterializedEntries(t *testing.T) {
	t.Run("empty slice is no-op", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		err := repo.InsertMaterializedEntries(ctx, nil)
		if err != nil {
			t.Fatalf("expected nil for empty slice, got %v", err)
		}
		testutil.AssertRowCount(t, db, "fund_history_materialized", 0)
	})

	t.Run("insert multiple entries", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)

		entries := []model.FundHistoryEntry{
			{
				ID:              testutil.MakeID(),
				PortfolioFundID: pf.ID,
				FundID:          fund.ID,
				Date:            date1,
				Shares:          100,
				Price:           10.0,
				Value:           1000,
				Cost:            900,
				RealizedGain:    50,
				UnrealizedGain:  100,
				TotalGainLoss:   150,
				Dividends:       20,
				Fees:            5,
				SaleProceeds:    200,
				OriginalCost:    150,
			},
			{
				ID:              testutil.MakeID(),
				PortfolioFundID: pf.ID,
				FundID:          fund.ID,
				Date:            date2,
				Shares:          100,
				Price:           11.0,
				Value:           1100,
				Cost:            900,
				RealizedGain:    50,
				UnrealizedGain:  200,
				TotalGainLoss:   250,
				Dividends:       20,
				Fees:            5,
				SaleProceeds:    200,
				OriginalCost:    150,
			},
		}

		err := repo.InsertMaterializedEntries(ctx, entries)
		if err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		testutil.AssertRowCount(t, db, "fund_history_materialized", 2)
	})
}

// ---------------------------------------------------------------------------
// InvalidateMaterializedTable
// ---------------------------------------------------------------------------

func TestMaterializedRepository_InvalidateMaterializedTable(t *testing.T) {
	t.Run("empty pfIDs is no-op", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		err := repo.InvalidateMaterializedTable(ctx, time.Now(), nil)
		if err != nil {
			t.Fatalf("expected nil for empty pfIDs, got %v", err)
		}
	})

	t.Run("deletes from date forward for specified pfIDs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		dates := []time.Time{
			time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC),
		}

		entries := make([]model.FundHistoryEntry, len(dates))
		for i, d := range dates {
			entries[i] = model.FundHistoryEntry{
				ID:              testutil.MakeID(),
				PortfolioFundID: pf.ID,
				FundID:          fund.ID,
				Date:            d,
				Shares:          100,
				Price:           10,
				Value:           1000,
				Cost:            900,
			}
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}
		testutil.AssertRowCount(t, db, "fund_history_materialized", 4)

		// Invalidate from March 12 onwards
		err := repo.InvalidateMaterializedTable(ctx, dates[2], []string{pf.ID})
		if err != nil {
			t.Fatalf("InvalidateMaterializedTable: %v", err)
		}

		testutil.AssertRowCount(t, db, "fund_history_materialized", 2)
	})

	t.Run("does not affect other pfIDs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund1 := testutil.NewFund().Build(t, db)
		fund2 := testutil.NewFund().Build(t, db)
		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		entries := []model.FundHistoryEntry{
			{ID: testutil.MakeID(), PortfolioFundID: pf1.ID, FundID: fund1.ID, Date: date, Shares: 10, Price: 10, Value: 100, Cost: 90},
			{ID: testutil.MakeID(), PortfolioFundID: pf2.ID, FundID: fund2.ID, Date: date, Shares: 10, Price: 10, Value: 100, Cost: 90},
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		// Only invalidate pf1
		err := repo.InvalidateMaterializedTable(ctx, date, []string{pf1.ID})
		if err != nil {
			t.Fatalf("InvalidateMaterializedTable: %v", err)
		}

		testutil.AssertRowCount(t, db, "fund_history_materialized", 1)
	})
}

// ---------------------------------------------------------------------------
// GetMaterializedHistory
// ---------------------------------------------------------------------------

func TestMaterializedRepository_GetMaterializedHistory(t *testing.T) {
	t.Run("empty portfolioIDs returns immediately", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		called := false
		err := repo.GetMaterializedHistory(nil, time.Now(), time.Now(), func(_ model.PortfolioHistoryMaterialized) error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if called {
			t.Error("callback should not be called for empty portfolioIDs")
		}
	})

	t.Run("returns aggregated records via callback", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		entries := []model.FundHistoryEntry{
			{
				ID:              testutil.MakeID(),
				PortfolioFundID: pf.ID,
				FundID:          fund.ID,
				Date:            date,
				Shares:          100,
				Price:           10,
				Value:           1000,
				Cost:            900,
				RealizedGain:    50,
				UnrealizedGain:  100,
				TotalGainLoss:   150,
				Dividends:       20,
				Fees:            5,
				SaleProceeds:    200,
				OriginalCost:    150,
			},
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		var records []model.PortfolioHistoryMaterialized
		err := repo.GetMaterializedHistory(
			[]string{portfolio.ID},
			date,
			date,
			func(record model.PortfolioHistoryMaterialized) error {
				records = append(records, record)
				return nil
			},
		)
		if err != nil {
			t.Fatalf("GetMaterializedHistory: %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("expected 1 record, got %d", len(records))
		}
		if records[0].PortfolioID != portfolio.ID {
			t.Errorf("expected portfolioID=%s, got %s", portfolio.ID, records[0].PortfolioID)
		}
		if records[0].Value != 1000 {
			t.Errorf("expected value=1000, got %f", records[0].Value)
		}
	})

	t.Run("date range filtering", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		base := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		entries := make([]model.FundHistoryEntry, 0, 5)
		for i := range 5 {
			entries = append(entries, model.FundHistoryEntry{
				ID:              testutil.MakeID(),
				PortfolioFundID: pf.ID,
				FundID:          fund.ID,
				Date:            base.AddDate(0, 0, i),
				Shares:          100,
				Price:           float64(10 + i),
				Value:           float64(100 * (10 + i)),
				Cost:            900,
			})
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		var count int
		err := repo.GetMaterializedHistory(
			[]string{portfolio.ID},
			base.AddDate(0, 0, 1), // March 11
			base.AddDate(0, 0, 3), // March 13
			func(_ model.PortfolioHistoryMaterialized) error {
				count++
				return nil
			},
		)
		if err != nil {
			t.Fatalf("GetMaterializedHistory: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 records in date range, got %d", count)
		}
	})
}

// ---------------------------------------------------------------------------
// GetLatestMaterializedDate
// ---------------------------------------------------------------------------

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestMaterializedRepository_GetLatestMaterializedDate(t *testing.T) {
	t.Run("empty portfolioIDs returns false", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		_, _, ok, err := repo.GetLatestMaterializedDate(nil)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if ok {
			t.Error("expected ok=false for empty portfolioIDs")
		}
	})

	t.Run("no data returns false", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		_, _, ok, err := repo.GetLatestMaterializedDate([]string{testutil.MakeID()})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if ok {
			t.Error("expected ok=false when no data exists")
		}
	})

	t.Run("returns latest date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		entries := []model.FundHistoryEntry{
			{ID: testutil.MakeID(), PortfolioFundID: pf.ID, FundID: fund.ID, Date: date1, Shares: 10, Price: 10, Value: 100, Cost: 90},
			{ID: testutil.MakeID(), PortfolioFundID: pf.ID, FundID: fund.ID, Date: date2, Shares: 10, Price: 11, Value: 110, Cost: 90},
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		latestDate, _, ok, err := repo.GetLatestMaterializedDate([]string{portfolio.ID})
		if err != nil {
			t.Fatalf("GetLatestMaterializedDate: %v", err)
		}
		if !ok {
			t.Fatal("expected ok=true")
		}
		if !latestDate.Equal(date2) {
			t.Errorf("expected latest date=%v, got %v", date2, latestDate)
		}
	})

	// These two tests cover the bug fixed in GetLatestMaterializedDate: the query
	// must return MIN(per-portfolio MAX(date)) so that a lagging portfolio (e.g.
	// one whose background regen failed) is not masked by a fully-covered portfolio.

	t.Run("returns minimum coverage date when portfolios lag", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		// Portfolio A covered through March 26, portfolio B only through March 25.
		portfolioA := testutil.NewPortfolio().Build(t, db)
		portfolioB := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfA := testutil.NewPortfolioFund(portfolioA.ID, fund.ID).Build(t, db)
		pfB := testutil.NewPortfolioFund(portfolioB.ID, fund.ID).Build(t, db)

		dateAhead := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
		dateBehind := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

		entries := []model.FundHistoryEntry{
			{ID: testutil.MakeID(), PortfolioFundID: pfA.ID, FundID: fund.ID, Date: dateBehind, Shares: 10, Price: 10, Value: 100, Cost: 90},
			{ID: testutil.MakeID(), PortfolioFundID: pfA.ID, FundID: fund.ID, Date: dateAhead, Shares: 10, Price: 11, Value: 110, Cost: 90},
			{ID: testutil.MakeID(), PortfolioFundID: pfB.ID, FundID: fund.ID, Date: dateBehind, Shares: 20, Price: 10, Value: 200, Cost: 180},
			// pfB has no entry for dateAhead — simulates a failed regen
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		latestDate, _, ok, err := repo.GetLatestMaterializedDate([]string{portfolioA.ID, portfolioB.ID})
		if err != nil {
			t.Fatalf("GetLatestMaterializedDate: %v", err)
		}
		if !ok {
			t.Fatal("expected ok=true")
		}
		// Must return dateBehind (the minimum), not dateAhead.
		// Before the fix this returned dateAhead, hiding that portfolioB was stale.
		if !latestDate.Equal(dateBehind) {
			t.Errorf("expected minimum coverage date=%v, got %v (portfolioB lag was masked)", dateBehind, latestDate)
		}
	})

	t.Run("returns shared date when all portfolios have equal coverage", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolioA := testutil.NewPortfolio().Build(t, db)
		portfolioB := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfA := testutil.NewPortfolioFund(portfolioA.ID, fund.ID).Build(t, db)
		pfB := testutil.NewPortfolioFund(portfolioB.ID, fund.ID).Build(t, db)

		date := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
		entries := []model.FundHistoryEntry{
			{ID: testutil.MakeID(), PortfolioFundID: pfA.ID, FundID: fund.ID, Date: date, Shares: 10, Price: 10, Value: 100, Cost: 90},
			{ID: testutil.MakeID(), PortfolioFundID: pfB.ID, FundID: fund.ID, Date: date, Shares: 20, Price: 10, Value: 200, Cost: 180},
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		latestDate, _, ok, err := repo.GetLatestMaterializedDate([]string{portfolioA.ID, portfolioB.ID})
		if err != nil {
			t.Fatalf("GetLatestMaterializedDate: %v", err)
		}
		if !ok {
			t.Fatal("expected ok=true")
		}
		if !latestDate.Equal(date) {
			t.Errorf("expected date=%v, got %v", date, latestDate)
		}
	})
}

// ---------------------------------------------------------------------------
// GetLatestSourceDates
// ---------------------------------------------------------------------------

func TestMaterializedRepository_GetLatestSourceDates(t *testing.T) {
	t.Run("empty portfolioIDs returns zero times", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		txn, price, div, err := repo.GetLatestSourceDates(nil)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !txn.IsZero() || !price.IsZero() || !div.IsZero() {
			t.Error("expected all zero times for empty portfolioIDs")
		}
	})

	t.Run("returns dates from source tables", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create transaction
		testutil.NewTransaction(pf.ID).
			WithDate(time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		// Create fund price
		testutil.NewFundPrice(fund.ID).
			WithDate(time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)).
			Build(t, db)

		// Create dividend
		testutil.NewDividend(fund.ID, pf.ID).Build(t, db)

		txnDate, priceDate, divDate, err := repo.GetLatestSourceDates([]string{portfolio.ID})
		if err != nil {
			t.Fatalf("GetLatestSourceDates: %v", err)
		}

		// Transaction created_at is set by DB, so just check it's non-zero
		if txnDate.IsZero() {
			t.Error("expected non-zero transaction date")
		}

		// Fund price date should match what we set
		expectedPriceDate := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
		if !priceDate.Equal(expectedPriceDate) {
			t.Errorf("expected price date=%v, got %v", expectedPriceDate, priceDate)
		}

		// Dividend created_at is set by DB, so just check it's non-zero
		if divDate.IsZero() {
			t.Error("expected non-zero dividend date")
		}
	})

	t.Run("no source data returns zero times", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		portfolio := testutil.NewPortfolio().Build(t, db)

		txn, price, div, err := repo.GetLatestSourceDates([]string{portfolio.ID})
		if err != nil {
			t.Fatalf("GetLatestSourceDates: %v", err)
		}
		if !txn.IsZero() || !price.IsZero() || !div.IsZero() {
			t.Error("expected all zero times when no source data exists")
		}
	})
}

// ---------------------------------------------------------------------------
// GetPortfolioSummaryLatest
// ---------------------------------------------------------------------------

func TestMaterializedRepository_GetPortfolioSummaryLatest(t *testing.T) {
	t.Run("empty portfolioIDs returns immediately", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		called := false
		err := repo.GetPortfolioSummaryLatest(nil, func(_ model.PortfolioHistoryMaterialized) error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if called {
			t.Error("callback should not be called for empty portfolioIDs")
		}
	})

	t.Run("returns only latest date", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		entries := []model.FundHistoryEntry{
			{ID: testutil.MakeID(), PortfolioFundID: pf.ID, FundID: fund.ID, Date: date1, Shares: 10, Price: 10, Value: 100, Cost: 90, RealizedGain: 5, UnrealizedGain: 10, TotalGainLoss: 15, Dividends: 2, SaleProceeds: 50, OriginalCost: 40},
			{ID: testutil.MakeID(), PortfolioFundID: pf.ID, FundID: fund.ID, Date: date2, Shares: 10, Price: 12, Value: 120, Cost: 90, RealizedGain: 5, UnrealizedGain: 30, TotalGainLoss: 35, Dividends: 2, SaleProceeds: 50, OriginalCost: 40},
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		var records []model.PortfolioHistoryMaterialized
		err := repo.GetPortfolioSummaryLatest(
			[]string{portfolio.ID},
			func(record model.PortfolioHistoryMaterialized) error {
				records = append(records, record)
				return nil
			},
		)
		if err != nil {
			t.Fatalf("GetPortfolioSummaryLatest: %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("expected 1 record (latest only), got %d", len(records))
		}
		if records[0].Value != 120 {
			t.Errorf("expected value=120 for latest date, got %f", records[0].Value)
		}
	})

	t.Run("no data returns no records", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		portfolio := testutil.NewPortfolio().Build(t, db)

		var count int
		err := repo.GetPortfolioSummaryLatest(
			[]string{portfolio.ID},
			func(_ model.PortfolioHistoryMaterialized) error {
				count++
				return nil
			},
		)
		if err != nil {
			t.Fatalf("GetPortfolioSummaryLatest: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 records, got %d", count)
		}
	})
}

// ---------------------------------------------------------------------------
// GetFundHistoryMaterialized
// ---------------------------------------------------------------------------

func TestMaterializedRepository_GetFundHistoryMaterialized(t *testing.T) {
	t.Run("returns fund-level entries via callback", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithName("Apple Inc").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		entries := []model.FundHistoryEntry{
			{
				ID:              testutil.MakeID(),
				PortfolioFundID: pf.ID,
				FundID:          fund.ID,
				Date:            date,
				Shares:          50,
				Price:           150,
				Value:           7500,
				Cost:            5000,
				RealizedGain:    200,
				UnrealizedGain:  2500,
				TotalGainLoss:   2700,
				Dividends:       100,
				Fees:            25,
				SaleProceeds:    1000,
				OriginalCost:    800,
			},
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		var results []model.FundHistoryEntry
		err := repo.GetFundHistoryMaterialized(
			portfolio.ID,
			date,
			date,
			func(entry model.FundHistoryEntry) error {
				results = append(results, entry)
				return nil
			},
		)
		if err != nil {
			t.Fatalf("GetFundHistoryMaterialized: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(results))
		}
		if results[0].FundName != "Apple Inc" {
			t.Errorf("expected FundName=Apple Inc, got %s", results[0].FundName)
		}
		if results[0].Value != 7500 {
			t.Errorf("expected Value=7500, got %f", results[0].Value)
		}
		if results[0].Shares != 50 {
			t.Errorf("expected Shares=50, got %f", results[0].Shares)
		}
	})

	t.Run("no data returns no entries", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)

		portfolio := testutil.NewPortfolio().Build(t, db)

		date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		var count int
		err := repo.GetFundHistoryMaterialized(
			portfolio.ID,
			date,
			date,
			func(_ model.FundHistoryEntry) error {
				count++
				return nil
			},
		)
		if err != nil {
			t.Fatalf("GetFundHistoryMaterialized: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 entries, got %d", count)
		}
	})

	t.Run("date range filtering", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		base := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		entries := make([]model.FundHistoryEntry, 0, 5)
		for i := range 5 {
			entries = append(entries, model.FundHistoryEntry{
				ID:              testutil.MakeID(),
				PortfolioFundID: pf.ID,
				FundID:          fund.ID,
				Date:            base.AddDate(0, 0, i),
				Shares:          10,
				Price:           float64(10 + i),
				Value:           float64(10 * (10 + i)),
				Cost:            90,
			})
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		var count int
		err := repo.GetFundHistoryMaterialized(
			portfolio.ID,
			base.AddDate(0, 0, 1),
			base.AddDate(0, 0, 3),
			func(_ model.FundHistoryEntry) error {
				count++
				return nil
			},
		)
		if err != nil {
			t.Fatalf("GetFundHistoryMaterialized: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 entries in date range, got %d", count)
		}
	})

	t.Run("callback error is propagated", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewMaterializedRepository(db)
		ctx := context.Background()

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		entries := []model.FundHistoryEntry{
			{ID: testutil.MakeID(), PortfolioFundID: pf.ID, FundID: fund.ID, Date: date, Shares: 10, Price: 10, Value: 100, Cost: 90},
		}
		if err := repo.InsertMaterializedEntries(ctx, entries); err != nil {
			t.Fatalf("InsertMaterializedEntries: %v", err)
		}

		expectedErr := errors.New("callback error")
		err := repo.GetFundHistoryMaterialized(
			portfolio.ID,
			date,
			date,
			func(_ model.FundHistoryEntry) error {
				return expectedErr
			},
		)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected callback error, got %v", err)
		}
	})
}
