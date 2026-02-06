"""
Tests for automatic materialization of fund history when empty.

This test suite verifies that:
- Empty fund history automatically triggers materialization
- Materialization is skipped if portfolio has no transactions
- Materialization errors don't break the API
- Re-query happens after successful materialization
"""

from datetime import date, timedelta
from decimal import Decimal

from app.models import (
    Fund,
    FundHistoryMaterialized,
    FundPrice,
    Portfolio,
    PortfolioFund,
    Transaction,
)
from app.services.fund_service import FundService
from tests.test_helpers import make_id, make_isin


class TestFundHistoryAutoMaterialization:
    """Tests for automatic materialization when fund history is empty."""

    def test_auto_materialize_on_empty_with_transactions(self, app_context, db_session):
        """
        Test that empty fund history triggers automatic materialization.

        WHY: After upgrading to v1.5.0 or after invalidating without recalculation,
        the materialized view may be empty. Auto-materialization ensures graphs
        work without manual CLI intervention.
        """
        # Create portfolio with transactions
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, portfolio_fund])
        db_session.commit()

        # Add transaction and price
        transaction_date = date.today() - timedelta(days=5)
        transaction = Transaction(
            portfolio_fund_id=portfolio_fund.id,
            date=transaction_date,
            type="buy",
            shares=10.0,
            cost_per_share=Decimal("100.00"),
        )
        fund_price = FundPrice(
            fund_id=fund.id,
            date=transaction_date,
            price=105.0,
        )
        db_session.add_all([transaction, fund_price])
        db_session.commit()

        # Verify materialized view is empty
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 0

        # Call get_fund_history - should auto-materialize
        history = FundService.get_fund_history(portfolio.id)

        # Verify data was returned
        assert len(history) > 0
        assert history[0]["date"] == transaction_date.isoformat()
        assert len(history[0]["funds"]) == 1
        assert history[0]["funds"][0]["portfolioFundId"] == portfolio_fund.id

        # Verify materialized view was populated
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after > 0

    def test_no_auto_materialize_when_no_transactions(self, app_context, db_session):
        """
        Test that empty portfolios don't trigger materialization.

        WHY: Portfolios without transactions have no data to materialize.
        Attempting materialization would waste resources and return empty anyway.
        """
        # Create portfolio without transactions
        portfolio = Portfolio(id=make_id(), name="Empty Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        # Call get_fund_history
        history = FundService.get_fund_history(portfolio.id)

        # Should return empty without error
        assert history == []

        # Verify no materialization happened
        count = FundHistoryMaterialized.query.count()
        assert count == 0

    def test_auto_materialize_with_existing_data_skips(self, app_context, db_session):
        """
        Test that existing materialized data is returned without re-materialization.

        WHY: If materialized data already exists, we should use it rather than
        recalculating. This ensures fast response times.
        """
        # Create portfolio with existing materialized data
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, portfolio_fund])
        db_session.commit()

        # Add existing materialized data
        test_date = date.today() - timedelta(days=1)
        materialized_record = FundHistoryMaterialized(
            portfolio_fund_id=portfolio_fund.id,
            fund_id=fund.id,
            date=test_date.isoformat(),
            shares=10.0,
            price=100.0,
            value=1000.0,
            cost=900.0,
            realized_gain=0.0,
            unrealized_gain=100.0,
            total_gain_loss=100.0,
            dividends=0.0,
            fees=0.0,
        )
        db_session.add(materialized_record)
        db_session.commit()

        record_id = materialized_record.id

        # Call get_fund_history
        history = FundService.get_fund_history(portfolio.id)

        # Verify existing data was returned
        assert len(history) == 1
        assert history[0]["date"] == test_date.isoformat()

        # Verify no new records were created (same ID exists)
        record_after = db_session.get(FundHistoryMaterialized, record_id)
        assert record_after is not None

    def test_auto_materialize_with_multiple_funds(self, app_context, db_session):
        """
        Test auto-materialization works with multiple funds in portfolio.

        WHY: Portfolios typically hold multiple funds. Auto-materialization
        should handle all funds in a single operation.
        """
        # Create portfolio with multiple funds
        portfolio = Portfolio(id=make_id(), name="Multi-Fund Portfolio")
        fund1 = Fund(
            id=make_id(),
            name="Fund A",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        fund2 = Fund(
            id=make_id(),
            name="Fund B",
            isin=make_isin("GB"),
            currency="EUR",
            exchange="LSE",
        )
        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund1.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund2.id)
        db_session.add_all([portfolio, fund1, fund2, pf1, pf2])
        db_session.commit()

        # Add transactions and prices for both funds
        transaction_date = date.today() - timedelta(days=3)
        txn1 = Transaction(
            portfolio_fund_id=pf1.id,
            date=transaction_date,
            type="buy",
            shares=5.0,
            cost_per_share=Decimal("100.00"),
        )
        txn2 = Transaction(
            portfolio_fund_id=pf2.id,
            date=transaction_date,
            type="buy",
            shares=10.0,
            cost_per_share=Decimal("50.00"),
        )
        price1 = FundPrice(fund_id=fund1.id, date=transaction_date, price=105.0)
        price2 = FundPrice(fund_id=fund2.id, date=transaction_date, price=52.0)
        db_session.add_all([txn1, txn2, price1, price2])
        db_session.commit()

        # Call get_fund_history - should auto-materialize both funds
        history = FundService.get_fund_history(portfolio.id)

        # Verify both funds are in the history
        assert len(history) > 0
        assert len(history[0]["funds"]) == 2
        fund_ids = {f["fundId"] for f in history[0]["funds"]}
        assert fund1.id in fund_ids
        assert fund2.id in fund_ids

    def test_auto_materialize_handles_errors_gracefully(self, app_context, db_session, mocker):
        """
        Test that materialization errors don't break the API.

        WHY: If materialization fails (e.g., due to internal errors),
        the API should still respond with an empty array rather than 500 error.
        """
        # Create portfolio with fund and transaction
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, portfolio_fund])
        db_session.commit()

        # Add a transaction so auto-materialization will be attempted
        transaction = Transaction(
            portfolio_fund_id=portfolio_fund.id,
            date=date.today() - timedelta(days=1),
            type="buy",
            shares=10.0,
            cost_per_share=Decimal("100.00"),
        )
        db_session.add(transaction)
        db_session.commit()

        # Mock the materialization service to raise an exception
        from app.services import portfolio_history_materialized_service

        mock_materialize = mocker.patch.object(
            portfolio_history_materialized_service.PortfolioHistoryMaterializedService,
            "materialize_portfolio_history",
        )
        mock_materialize.side_effect = Exception("Materialization failed")

        # Call get_fund_history - should return empty without crashing
        history = FundService.get_fund_history(portfolio.id)

        # Should return empty array, not raise exception
        assert history == []

        # Verify materialization was attempted
        mock_materialize.assert_called_once()

    def test_auto_materialize_respects_date_filters(self, app_context, db_session):
        """
        Test that auto-materialization respects start_date and end_date filters.

        WHY: Users may request specific date ranges. Auto-materialization
        should materialize the full history, but the response should still
        be filtered to the requested range.
        """
        # Create portfolio with transactions over multiple days
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, portfolio_fund])
        db_session.commit()

        # Add transactions on different dates
        for i in range(5):
            txn_date = date.today() - timedelta(days=i)
            transaction = Transaction(
                portfolio_fund_id=portfolio_fund.id,
                date=txn_date,
                type="buy",
                shares=1.0,
                cost_per_share=Decimal("100.00"),
            )
            fund_price = FundPrice(
                fund_id=fund.id,
                date=txn_date,
                price=100.0 + i,
            )
            db_session.add_all([transaction, fund_price])
        db_session.commit()

        # Request filtered date range
        start_date = (date.today() - timedelta(days=2)).isoformat()
        end_date = date.today().isoformat()

        history = FundService.get_fund_history(
            portfolio.id, start_date=start_date, end_date=end_date
        )

        # Should only return filtered dates (3 days: today, -1, -2)
        assert len(history) <= 3

        # All dates should be within the filter range
        for day in history:
            assert start_date <= day["date"] <= end_date
