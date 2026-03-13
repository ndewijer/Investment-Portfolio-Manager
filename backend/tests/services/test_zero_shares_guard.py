"""
Tests for the zero-shares guard fix in get_portfolio_fund_history.

This test suite verifies that:
- Funds with shares=0 but non-zero financial data (realized gains, sale proceeds,
  original cost, dividends) are NOT excluded from history
- Fully inactive funds (all financial fields zero) ARE excluded
- Sale proceeds and original cost are correctly calculated and included
"""

from datetime import date, timedelta
from decimal import Decimal

from app.models import (
    Dividend,
    Fund,
    FundPrice,
    Portfolio,
    PortfolioFund,
    RealizedGainLoss,
    Transaction,
)
from app.services.portfolio_service import PortfolioService
from tests.test_helpers import make_id, make_isin


class TestZeroSharesGuard:
    """Tests for the zero-shares guard in get_portfolio_fund_history."""

    def test_sold_fund_appears_in_history(self, app_context, db_session):
        """
        Test that a fully sold fund still appears in history with shares=0.

        WHY: When a fund is fully sold, it still has realized gains, sale proceeds,
        and original cost that should be visible in the history. The old guard
        `if shares > 0` incorrectly dropped these entries.
        """
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Sold Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, pf])
        db_session.commit()

        buy_date = date(2024, 1, 1)
        sell_date = date(2024, 1, 5)

        # Buy 10 shares at $100
        buy_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=buy_date,
            type="buy",
            shares=10.0,
            cost_per_share=Decimal("100.00"),
        )
        # Sell all 10 shares at $120
        sell_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=sell_date,
            type="sell",
            shares=10.0,
            cost_per_share=Decimal("120.00"),
        )
        db_session.add_all([buy_txn, sell_txn])
        db_session.commit()

        # Record realized gain
        rg = RealizedGainLoss(
            portfolio_id=portfolio.id,
            fund_id=fund.id,
            transaction_id=sell_txn.id,
            transaction_date=sell_date,
            shares_sold=10.0,
            sale_proceeds=1200.0,
            cost_basis=1000.0,
            realized_gain_loss=200.0,
        )
        db_session.add(rg)

        # Add prices for the date range
        for i in range(10):
            fp = FundPrice(
                fund_id=fund.id,
                date=buy_date + timedelta(days=i),
                price=100.0 + i * 5,
            )
            db_session.add(fp)
        db_session.commit()

        # Get history covering dates after the sell
        history = PortfolioService.get_portfolio_fund_history(
            portfolio.id,
            start_date=buy_date.isoformat(),
            end_date=(sell_date + timedelta(days=3)).isoformat(),
        )

        # Check that dates after sell still have fund entries
        dates_after_sell = [day for day in history if day["date"] >= sell_date.isoformat()]
        assert len(dates_after_sell) > 0, "Should have dates after sell"

        for day in dates_after_sell:
            assert len(day["funds"]) > 0, f"Fund should appear on {day['date']} even with 0 shares"
            fund_entry = day["funds"][0]
            assert fund_entry["shares"] == 0
            assert fund_entry["realizedGain"] == 200.0
            assert fund_entry["saleProceeds"] == 1200.0
            assert fund_entry["originalCost"] == 1000.0

    def test_fully_inactive_fund_excluded(self, app_context, db_session):
        """
        Test that a fund with zero shares and no financial data is excluded.

        WHY: Funds that have never had transactions or have been completely
        cleared of all financial history should not create empty entries.
        """
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        active_fund = Fund(
            id=make_id(),
            name="Active Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        inactive_fund = Fund(
            id=make_id(),
            name="Inactive Fund",
            isin=make_isin("GB"),
            currency="USD",
            exchange="NYSE",
        )
        pf_active = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=active_fund.id)
        pf_inactive = PortfolioFund(
            id=make_id(), portfolio_id=portfolio.id, fund_id=inactive_fund.id
        )
        db_session.add_all([portfolio, active_fund, inactive_fund, pf_active, pf_inactive])
        db_session.commit()

        txn_date = date(2024, 1, 1)

        # Only the active fund has a transaction
        txn = Transaction(
            portfolio_fund_id=pf_active.id,
            date=txn_date,
            type="buy",
            shares=10.0,
            cost_per_share=Decimal("100.00"),
        )
        db_session.add(txn)

        # Add price for active fund
        fp = FundPrice(fund_id=active_fund.id, date=txn_date, price=100.0)
        db_session.add(fp)
        db_session.commit()

        history = PortfolioService.get_portfolio_fund_history(
            portfolio.id,
            start_date=txn_date.isoformat(),
            end_date=txn_date.isoformat(),
        )

        assert len(history) == 1
        # Only the active fund should appear
        fund_ids = {f["fundId"] for f in history[0]["funds"]}
        assert active_fund.id in fund_ids
        assert inactive_fund.id not in fund_ids

    def test_sold_fund_with_dividends_appears(self, app_context, db_session):
        """
        Test that a sold fund with cumulative dividends still appears.

        WHY: Even if shares=0, if dividends were earned before selling,
        they should still be visible in the history.
        """
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Dividend Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, pf])
        db_session.commit()

        buy_date = date(2024, 1, 1)
        dividend_date = date(2024, 1, 3)
        sell_date = date(2024, 1, 5)

        # Buy 10 shares
        buy_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=buy_date,
            type="buy",
            shares=10.0,
            cost_per_share=Decimal("100.00"),
        )
        # Sell all shares
        sell_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=sell_date,
            type="sell",
            shares=10.0,
            cost_per_share=Decimal("110.00"),
        )
        db_session.add_all([buy_txn, sell_txn])
        db_session.commit()

        # Add dividend before sell
        dividend = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=dividend_date - timedelta(days=1),
            ex_dividend_date=dividend_date,
            dividend_per_share=2.0,
            shares_owned=10.0,
            total_amount=20.0,
        )
        db_session.add(dividend)

        # Record realized gain
        rg = RealizedGainLoss(
            portfolio_id=portfolio.id,
            fund_id=fund.id,
            transaction_id=sell_txn.id,
            transaction_date=sell_date,
            shares_sold=10.0,
            sale_proceeds=1100.0,
            cost_basis=1000.0,
            realized_gain_loss=100.0,
        )
        db_session.add(rg)

        # Add prices
        for i in range(8):
            fp = FundPrice(
                fund_id=fund.id,
                date=buy_date + timedelta(days=i),
                price=100.0 + i * 2,
            )
            db_session.add(fp)
        db_session.commit()

        # Get history for a date after sell
        after_sell = sell_date + timedelta(days=1)
        history = PortfolioService.get_portfolio_fund_history(
            portfolio.id,
            start_date=after_sell.isoformat(),
            end_date=after_sell.isoformat(),
        )

        assert len(history) == 1
        assert len(history[0]["funds"]) == 1, (
            "Fund should appear due to dividends and realized gains"
        )
        fund_entry = history[0]["funds"][0]
        assert fund_entry["shares"] == 0
        assert fund_entry["dividends"] == 20.0
        assert fund_entry["realizedGain"] == 100.0

    def test_fund_history_includes_sale_proceeds_and_original_cost(self, app_context, db_session):
        """
        Test that saleProceeds and originalCost are included in fund history output.

        WHY: The fund history dict should include sale_proceeds and original_cost
        so the materialized write path can store them.
        """
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, pf])
        db_session.commit()

        buy_date = date(2024, 1, 1)
        sell_date = date(2024, 1, 3)

        buy_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=buy_date,
            type="buy",
            shares=10.0,
            cost_per_share=Decimal("100.00"),
        )
        sell_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=sell_date,
            type="sell",
            shares=5.0,
            cost_per_share=Decimal("120.00"),
        )
        db_session.add_all([buy_txn, sell_txn])
        db_session.commit()

        rg = RealizedGainLoss(
            portfolio_id=portfolio.id,
            fund_id=fund.id,
            transaction_id=sell_txn.id,
            transaction_date=sell_date,
            shares_sold=5.0,
            sale_proceeds=600.0,
            cost_basis=500.0,
            realized_gain_loss=100.0,
        )
        db_session.add(rg)

        # Add prices
        for i in range(5):
            fp = FundPrice(
                fund_id=fund.id,
                date=buy_date + timedelta(days=i),
                price=100.0 + i * 10,
            )
            db_session.add(fp)
        db_session.commit()

        history = PortfolioService.get_portfolio_fund_history(
            portfolio.id,
            start_date=sell_date.isoformat(),
            end_date=sell_date.isoformat(),
        )

        assert len(history) == 1
        fund_entry = history[0]["funds"][0]
        assert "saleProceeds" in fund_entry
        assert "originalCost" in fund_entry
        assert "dividends" in fund_entry
        assert fund_entry["saleProceeds"] == 600.0
        assert fund_entry["originalCost"] == 500.0
