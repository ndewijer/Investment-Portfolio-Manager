"""
Tests for materialized view invalidation during price update operations.

This test suite verifies that:
- Today's price update invalidates materialized view
- Historical price updates invalidate materialized view from earliest date
- Price updates invalidate multiple portfolios holding the same fund
- Duplicate prices skip invalidation
- Invalidation failures don't break price updates
"""

from datetime import date, timedelta
from unittest.mock import MagicMock, patch

import pandas as pd
from app.models import (
    Fund,
    FundHistoryMaterialized,
    FundPrice,
    InvestmentType,
    Portfolio,
    PortfolioFund,
    Transaction,
)
from app.services.price_update_service import HistoricalPriceService, TodayPriceService
from tests.test_helpers import make_id


class TestPriceUpdateMaterializedViewInvalidation:
    """Tests for materialized view invalidation during price update operations."""

    def _create_test_data(self, db_session):
        """Helper to create common test entities."""
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=f"US{make_id()[:8]}",
            symbol="TEST",
            currency="USD",
            exchange="NYSE",
            investment_type=InvestmentType.FUND,
        )
        db_session.add_all([portfolio, fund])
        db_session.commit()

        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(portfolio_fund)
        db_session.commit()

        # Create a buy transaction
        transaction = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            date=date.today() - timedelta(days=30),
            type="buy",
            shares=100.0,
            cost_per_share=10.0,
        )
        db_session.add(transaction)
        db_session.commit()

        return portfolio, fund, portfolio_fund

    def _add_materialized_records(self, db_session, portfolio_fund, fund, days=3):
        """Helper to add materialized records."""
        for i in range(days):
            record_date = date.today() - timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=portfolio_fund.id,
                fund_id=fund.id,
                date=record_date.isoformat(),
                shares=100.0,
                price=10.0,
                value=1000.0,
                cost=1000.0,
                realized_gain=0.0,
                unrealized_gain=0.0,
                total_gain_loss=0.0,
                dividends=0.0,
                fees=0.0,
            )
            db_session.add(record)
        db_session.commit()

    def _create_mock_history(self, dates_and_prices):
        """Create a mock yfinance history DataFrame."""
        index = pd.DatetimeIndex([pd.Timestamp(d) for d, _ in dates_and_prices])
        data = {"Close": [p for _, p in dates_and_prices]}
        return pd.DataFrame(data, index=index)

    def test_update_todays_price_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that updating today's price invalidates the materialized view.

        WHY: When a new price is added, the fund value changes. The materialized
        view must be invalidated from that date forward to reflect the new price.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)
        self._add_materialized_records(db_session, portfolio_fund, fund)

        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Mock yfinance to return a price
        last_date = date.today() - timedelta(days=1)
        mock_history = self._create_mock_history([(last_date, 15.0)])

        mock_ticker = MagicMock()
        mock_ticker.history.return_value = mock_history

        with patch("app.services.price_update_service.yf.Ticker", return_value=mock_ticker):
            _response, status = TodayPriceService.update_todays_price(fund.id)

        assert status == 200

        # Materialized records should be invalidated
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after < count_before

    def test_update_historical_prices_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that historical price update invalidates from the earliest updated date.

        WHY: When historical prices are backfilled, all fund values from the
        earliest new price forward are potentially wrong. The materialized view
        must be invalidated from that earliest date.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)

        # Add materialized records for 10 days
        for i in range(10):
            record_date = date.today() - timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=portfolio_fund.id,
                fund_id=fund.id,
                date=record_date.isoformat(),
                shares=100.0,
                price=10.0,
                value=1000.0,
                cost=1000.0,
                realized_gain=0.0,
                unrealized_gain=0.0,
                total_gain_loss=0.0,
                dividends=0.0,
                fees=0.0,
            )
            db_session.add(record)
        db_session.commit()

        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 10

        # Mock missing dates to return dates 5-7 days ago
        missing_dates = [date.today() - timedelta(days=i) for i in range(5, 8)]
        mock_history = self._create_mock_history([(d, 12.0) for d in missing_dates])
        mock_ticker = MagicMock()
        mock_ticker.history.return_value = mock_history

        with (
            patch(
                "app.services.price_update_service.HistoricalPriceService.get_missing_dates",
                return_value=missing_dates,
            ),
            patch(
                "app.services.price_update_service.yf.Ticker",
                return_value=mock_ticker,
            ),
        ):
            _response, status = HistoricalPriceService.update_historical_prices(fund.id)

        assert status == 200

        # Records from earliest missing date (7 days ago) forward should be invalidated
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after < count_before

    def test_price_update_invalidates_multiple_portfolios(self, app_context, db_session):
        """
        Test that price update invalidates materialized view for all portfolios holding the fund.

        WHY: Multiple portfolios can hold the same fund. When the fund's price
        changes, ALL portfolios holding it have stale cached data.
        """
        # Create two portfolios with the same fund
        portfolio1 = Portfolio(id=make_id(), name="Portfolio 1")
        portfolio2 = Portfolio(id=make_id(), name="Portfolio 2")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=f"US{make_id()[:8]}",
            symbol="TEST",
            currency="USD",
            exchange="NYSE",
            investment_type=InvestmentType.FUND,
        )
        db_session.add_all([portfolio1, portfolio2, fund])
        db_session.commit()

        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio1.id, fund_id=fund.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund.id)
        db_session.add_all([pf1, pf2])
        db_session.commit()

        # Add transactions for both
        for pf in [pf1, pf2]:
            txn = Transaction(
                id=make_id(),
                portfolio_fund_id=pf.id,
                date=date.today() - timedelta(days=30),
                type="buy",
                shares=50.0,
                cost_per_share=10.0,
            )
            db_session.add(txn)
        db_session.commit()

        # Add materialized records for both
        self._add_materialized_records(db_session, pf1, fund)
        self._add_materialized_records(db_session, pf2, fund)

        count_p1_before = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf1.id).count()
        count_p2_before = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf2.id).count()
        assert count_p1_before == 3
        assert count_p2_before == 3

        # Mock yfinance
        last_date = date.today() - timedelta(days=1)
        mock_history = self._create_mock_history([(last_date, 15.0)])
        mock_ticker = MagicMock()
        mock_ticker.history.return_value = mock_history

        with patch("app.services.price_update_service.yf.Ticker", return_value=mock_ticker):
            TodayPriceService.update_todays_price(fund.id)

        # Both portfolios should be invalidated
        count_p1_after = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf1.id).count()
        count_p2_after = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf2.id).count()
        assert count_p1_after < count_p1_before
        assert count_p2_after < count_p2_before

    def test_no_invalidation_when_price_already_exists(self, app_context, db_session):
        """
        Test that no invalidation occurs when the price already exists.

        WHY: If the price for a date already exists, no new data is written,
        so no invalidation is needed. This prevents unnecessary cache clearing.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)
        self._add_materialized_records(db_session, portfolio_fund, fund)

        # Pre-add the price that yfinance would return
        last_date = date.today() - timedelta(days=1)
        existing_price = FundPrice(fund_id=fund.id, date=last_date, price=10.0)
        db_session.add(existing_price)
        db_session.commit()

        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # The service should detect the existing price and skip
        mock_history = self._create_mock_history([(last_date, 15.0)])
        mock_ticker = MagicMock()
        mock_ticker.history.return_value = mock_history

        with patch("app.services.price_update_service.yf.Ticker", return_value=mock_ticker):
            _response, status = TodayPriceService.update_todays_price(fund.id)

        assert status == 200

        # Materialized records should remain unchanged
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == count_before

    def test_invalidation_failure_does_not_break_price_update(self, app_context, db_session):
        """
        Test that a failed invalidation doesn't break the price update.

        WHY: Invalidation is a secondary concern. If it fails, the price
        update should still succeed.
        """
        _portfolio, fund, _portfolio_fund = self._create_test_data(db_session)

        last_date = date.today() - timedelta(days=1)
        mock_history = self._create_mock_history([(last_date, 15.0)])
        mock_ticker = MagicMock()
        mock_ticker.history.return_value = mock_history

        with (
            patch(
                "app.services.price_update_service.yf.Ticker",
                return_value=mock_ticker,
            ),
            patch(
                "app.services.portfolio_history_materialized_service"
                ".PortfolioHistoryMaterializedService.invalidate_from_price_update",
                side_effect=Exception("Mock invalidation failure"),
            ),
        ):
            _response, status = TodayPriceService.update_todays_price(fund.id)

        # Price update should still succeed
        assert status == 200

        # Verify the price was actually saved
        saved_price = FundPrice.query.filter_by(fund_id=fund.id, date=last_date).first()
        assert saved_price is not None
