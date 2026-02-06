"""
Tests for materialized view invalidation during dividend operations.

This test suite verifies that:
- Creating dividends invalidates materialized view
- Updating dividends invalidates materialized view
- Changing ex_dividend_date invalidates from both old and new dates
- Deleting dividends invalidates materialized view
- Invalidation failures don't break dividend operations
"""

from datetime import date, timedelta
from unittest.mock import patch

from app.models import (
    DividendType,
    Fund,
    FundHistoryMaterialized,
    InvestmentType,
    Portfolio,
    PortfolioFund,
    Transaction,
)
from app.services.dividend_service import DividendService
from tests.test_helpers import make_id


class TestDividendMaterializedViewInvalidation:
    """Tests for materialized view invalidation during dividend operations."""

    def _create_test_data(self, db_session, dividend_type=DividendType.CASH):
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
            dividend_type=dividend_type,
        )
        db_session.add_all([portfolio, fund])
        db_session.commit()

        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(portfolio_fund)
        db_session.commit()

        # Create a buy transaction so shares_owned > 0
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

    def test_create_dividend_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that creating a cash dividend invalidates the materialized view.

        WHY: When a dividend is recorded, fund history changes (dividend totals
        change). The materialized view must be invalidated from the ex_dividend_date
        forward to ensure accurate data.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)
        self._add_materialized_records(db_session, portfolio_fund, fund)

        # Verify records exist
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Create dividend
        ex_date = (date.today() - timedelta(days=5)).isoformat()
        record_date = (date.today() - timedelta(days=7)).isoformat()
        DividendService.create_dividend(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "record_date": record_date,
                "ex_dividend_date": ex_date,
                "dividend_per_share": "0.50",
            }
        )

        # Materialized records should be invalidated
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_create_stock_dividend_with_reinvestment_invalidates(self, app_context, db_session):
        """
        Test that creating a stock dividend with reinvestment invalidates materialized view.

        WHY: Stock dividends with reinvestment create a new transaction, which
        also affects fund history. Both the dividend and transaction changes
        should trigger invalidation.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(
            db_session, dividend_type=DividendType.STOCK
        )
        self._add_materialized_records(db_session, portfolio_fund, fund)

        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        ex_date = (date.today() - timedelta(days=5)).isoformat()
        record_date = (date.today() - timedelta(days=7)).isoformat()
        DividendService.create_dividend(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "record_date": record_date,
                "ex_dividend_date": ex_date,
                "dividend_per_share": "0.50",
                "reinvestment_shares": "5.0",
                "reinvestment_price": "10.0",
            }
        )

        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_update_dividend_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that updating a dividend invalidates the materialized view.

        WHY: When dividend data changes (amount, dates), the cached history
        becomes stale and must be invalidated.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)

        # Create a dividend first
        ex_date = (date.today() - timedelta(days=5)).isoformat()
        record_date = (date.today() - timedelta(days=7)).isoformat()
        dividend = DividendService.create_dividend(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "record_date": record_date,
                "ex_dividend_date": ex_date,
                "dividend_per_share": "0.50",
            }
        )

        # Re-add materialized records
        self._add_materialized_records(db_session, portfolio_fund, fund)
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Update the dividend
        DividendService.update_dividend(
            dividend.id,
            {
                "record_date": record_date,
                "ex_dividend_date": ex_date,
                "dividend_per_share": "1.00",
            },
        )

        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_update_dividend_date_change_invalidates_both_dates(self, app_context, db_session):
        """
        Test that changing ex_dividend_date invalidates from both old and new dates.

        WHY: When the ex_dividend_date changes, the old date range is no longer
        valid AND the new date range also needs recalculation. Both must be
        invalidated to prevent stale data in either range.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)

        old_ex_date = (date.today() - timedelta(days=10)).isoformat()
        record_date = (date.today() - timedelta(days=12)).isoformat()
        dividend = DividendService.create_dividend(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "record_date": record_date,
                "ex_dividend_date": old_ex_date,
                "dividend_per_share": "0.50",
            }
        )

        # Add materialized records spanning both old and new dates
        for i in range(12):
            record_date_mat = date.today() - timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=portfolio_fund.id,
                fund_id=fund.id,
                date=record_date_mat.isoformat(),
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
        assert count_before == 12

        # Update with a new ex_dividend_date
        new_ex_date = (date.today() - timedelta(days=3)).isoformat()
        DividendService.update_dividend(
            dividend.id,
            {
                "record_date": (date.today() - timedelta(days=12)).isoformat(),
                "ex_dividend_date": new_ex_date,
                "dividend_per_share": "0.50",
            },
        )

        # Records from both old date (10 days ago) and new date (3 days ago)
        # forward should be invalidated. Since old date is earlier,
        # all records from 10 days ago onward should be gone.
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        # Records before the old ex_dividend_date (day 11) should remain
        assert count_after < count_before

    def test_delete_dividend_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that deleting a dividend invalidates the materialized view.

        WHY: When a dividend is removed, cached history includes stale dividend
        totals. The view must be invalidated from the ex_dividend_date forward.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)

        ex_date = (date.today() - timedelta(days=5)).isoformat()
        record_date = (date.today() - timedelta(days=7)).isoformat()
        dividend = DividendService.create_dividend(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "record_date": record_date,
                "ex_dividend_date": ex_date,
                "dividend_per_share": "0.50",
            }
        )

        # Re-add materialized records
        self._add_materialized_records(db_session, portfolio_fund, fund)
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Delete the dividend
        DividendService.delete_dividend(dividend.id)

        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_invalidation_failure_does_not_break_dividend_create(self, app_context, db_session):
        """
        Test that a failed invalidation doesn't break dividend creation.

        WHY: Invalidation is a secondary concern. If it fails (e.g., database
        issue), the primary dividend operation should still succeed.
        """
        _portfolio, _fund, portfolio_fund = self._create_test_data(db_session)

        with patch(
            "app.services.portfolio_history_materialized_service"
            ".PortfolioHistoryMaterializedService.invalidate_from_dividend",
            side_effect=Exception("Mock invalidation failure"),
        ):
            ex_date = (date.today() - timedelta(days=5)).isoformat()
            record_date = (date.today() - timedelta(days=7)).isoformat()
            dividend = DividendService.create_dividend(
                {
                    "portfolio_fund_id": portfolio_fund.id,
                    "record_date": record_date,
                    "ex_dividend_date": ex_date,
                    "dividend_per_share": "0.50",
                }
            )

            # Dividend should still be created successfully
            assert dividend is not None
            assert dividend.id is not None
