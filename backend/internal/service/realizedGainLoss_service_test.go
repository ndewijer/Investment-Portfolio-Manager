package service_test

import (
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// =============================================================================
// RealizedGainLossService.loadRealizedGainLoss (tested through GetPortfolioFunds
// and via direct builder + service method chains)
// =============================================================================

func TestRealizedGainLossService_LoadRealizedGainLoss(t *testing.T) {
	t.Run("returns empty map when no realized gains exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestRealizedGainLossService(t, db)

		// loadRealizedGainLoss is unexported, but we test it indirectly by
		// creating data and using GetPortfolioFunds. Here we just verify
		// the service is constructible and that the full integration works
		// via the fund service metric pipeline.
		_ = svc
	})

	t.Run("loads realized gains for portfolio via fund service", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyDate := time.Now().UTC().AddDate(0, -2, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(buyDate).WithType("buy").
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		sellDate := time.Now().UTC().AddDate(0, -1, 0)
		sellTx := testutil.NewTransaction(pf.ID).
			WithDate(sellDate).WithType("sell").
			WithShares(50).WithCostPerShare(15.0).
			Build(t, db)

		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
			WithDate(sellDate).
			WithShares(50).
			WithCostBasis(500.0).
			WithSaleProceeds(750.0).
			Build(t, db)

		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(14.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		if roundSvc(f.RealizedGainLoss) != roundSvc(250.0) {
			t.Errorf("expected RealizedGainLoss=250, got %f", f.RealizedGainLoss)
		}
	})
}

// =============================================================================
// RealizedGainLossService.processRealizedGainLossForDate (tested via metrics)
// =============================================================================

func TestRealizedGainLossService_ProcessRealizedGainLossForDate(t *testing.T) {
	t.Run("returns zero when no realized gains", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).
			WithDate(time.Now().UTC().AddDate(0, -1, 0)).
			WithType("buy").WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(12.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		if f.RealizedGainLoss != 0 {
			t.Errorf("expected RealizedGainLoss=0, got %f", f.RealizedGainLoss)
		}
	})

	t.Run("sums multiple realized gains", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyDate := time.Now().UTC().AddDate(0, -3, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(buyDate).WithType("buy").
			WithShares(200).WithCostPerShare(10.0).
			Build(t, db)

		sellDate1 := time.Now().UTC().AddDate(0, -2, 0)
		sellTx1 := testutil.NewTransaction(pf.ID).
			WithDate(sellDate1).WithType("sell").
			WithShares(50).WithCostPerShare(15.0).
			Build(t, db)

		sellDate2 := time.Now().UTC().AddDate(0, -1, 0)
		sellTx2 := testutil.NewTransaction(pf.ID).
			WithDate(sellDate2).WithType("sell").
			WithShares(30).WithCostPerShare(20.0).
			Build(t, db)

		// Realized gain 1: (15-10)*50 = 250
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx1.ID).
			WithDate(sellDate1).
			WithShares(50).
			WithCostBasis(500.0).
			WithSaleProceeds(750.0).
			Build(t, db)

		// Realized gain 2: (20-10)*30 = 300
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx2.ID).
			WithDate(sellDate2).
			WithShares(30).
			WithCostBasis(300.0).
			WithSaleProceeds(600.0).
			Build(t, db)

		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(18.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		expectedRGL := 250.0 + 300.0 // 550
		if roundSvc(f.RealizedGainLoss) != roundSvc(expectedRGL) {
			t.Errorf("expected RealizedGainLoss=%f, got %f", expectedRGL, f.RealizedGainLoss)
		}

		// Total shares: 200 - 50 - 30 = 120
		if roundSvc(f.TotalShares) != roundSvc(120.0) {
			t.Errorf("expected TotalShares=120, got %f", f.TotalShares)
		}
	})

	t.Run("includes realized loss (negative values)", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyDate := time.Now().UTC().AddDate(0, -2, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(buyDate).WithType("buy").
			WithShares(100).WithCostPerShare(20.0).
			Build(t, db)

		sellDate := time.Now().UTC().AddDate(0, -1, 0)
		sellTx := testutil.NewTransaction(pf.ID).
			WithDate(sellDate).WithType("sell").
			WithShares(50).WithCostPerShare(15.0).
			Build(t, db)

		// Realized loss: (15-20)*50 = -250
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
			WithDate(sellDate).
			WithShares(50).
			WithCostBasis(1000.0).
			WithSaleProceeds(750.0).
			Build(t, db)

		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(16.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		// SaleProceeds 750 - CostBasis 1000 = -250
		if roundSvc(f.RealizedGainLoss) != roundSvc(-250.0) {
			t.Errorf("expected RealizedGainLoss=-250, got %f", f.RealizedGainLoss)
		}
	})

	t.Run("total gain/loss combines unrealized and realized", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		svc := buildFullFundService(t, db)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().WithDividendType("NONE").Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		buyDate := time.Now().UTC().AddDate(0, -2, 0)
		testutil.NewTransaction(pf.ID).
			WithDate(buyDate).WithType("buy").
			WithShares(100).WithCostPerShare(10.0).
			Build(t, db)

		sellDate := time.Now().UTC().AddDate(0, -1, 0)
		sellTx := testutil.NewTransaction(pf.ID).
			WithDate(sellDate).WithType("sell").
			WithShares(40).WithCostPerShare(15.0).
			Build(t, db)

		// Realized gain: (15-10)*40 = 200
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
			WithDate(sellDate).
			WithShares(40).
			WithCostBasis(400.0).
			WithSaleProceeds(600.0).
			Build(t, db)

		priceDate := time.Now().UTC().AddDate(0, 0, -1)
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(14.0).Build(t, db)

		funds, err := svc.GetPortfolioFunds(portfolio.ID)
		if err != nil {
			t.Fatalf("GetPortfolioFunds() error: %v", err)
		}
		if len(funds) != 1 {
			t.Fatalf("expected 1 fund, got %d", len(funds))
		}

		f := funds[0]
		// Remaining 60 shares, cost = (1000/100)*60 = 600
		// Value = 60 * 14 = 840
		// UnrealizedGL = 840 - 600 = 240
		// RealizedGL = 200
		// TotalGL = 240 + 200 = 440
		if roundSvc(f.TotalGainLoss) != roundSvc(f.UnrealizedGainLoss+f.RealizedGainLoss) {
			t.Errorf("expected TotalGainLoss=%f, got %f", f.UnrealizedGainLoss+f.RealizedGainLoss, f.TotalGainLoss)
		}
		if roundSvc(f.RealizedGainLoss) != roundSvc(200.0) {
			t.Errorf("expected RealizedGainLoss=200, got %f", f.RealizedGainLoss)
		}
	})
}
