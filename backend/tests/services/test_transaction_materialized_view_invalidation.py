"""
Tests for materialized view invalidation during transaction operations.

This test suite verifies that:
- Creating buy transactions invalidates materialized view
- Updating transactions invalidates materialized view
- Deleting transactions invalidates materialized view
- Sell transactions with realized gains invalidate materialized view
- Invalidation only affects the target portfolio
- Invalidation failures don't break transaction operations
"""

from datetime import date, timedelta
from unittest.mock import patch

from app.models import (
    Fund,
    FundHistoryMaterialized,
    InvestmentType,
    Portfolio,
    PortfolioFund,
    Transaction,
)
from app.services.transaction_service import TransactionService
from tests.test_helpers import make_id


class TestTransactionMaterializedViewInvalidation:
    """Tests for materialized view invalidation during transaction operations."""

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

    def test_create_transaction_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that creating a buy transaction invalidates the materialized view.

        WHY: When a new buy transaction is added, the shares and cost basis
        change from that date forward. The materialized view must be invalidated
        to reflect the new position.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)
        self._add_materialized_records(db_session, portfolio_fund, fund)

        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Create a buy transaction
        txn_date = (date.today() - timedelta(days=5)).isoformat()
        TransactionService.create_transaction(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "date": txn_date,
                "type": "buy",
                "shares": "10",
                "cost_per_share": "10.0",
            }
        )

        # Materialized records should be invalidated
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_update_transaction_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that updating a transaction invalidates the materialized view.

        WHY: When a transaction is modified (shares, price, date), the fund
        history changes. The materialized view must be invalidated.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)

        # Create initial transaction
        txn_date = (date.today() - timedelta(days=5)).isoformat()
        transaction = TransactionService.create_transaction(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "date": txn_date,
                "type": "buy",
                "shares": "10",
                "cost_per_share": "10.0",
            }
        )

        # Re-add materialized records
        self._add_materialized_records(db_session, portfolio_fund, fund)
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Update the transaction
        TransactionService.update_transaction(
            transaction.id,
            {
                "date": txn_date,
                "type": "buy",
                "shares": "20",
                "cost_per_share": "10.0",
            },
        )

        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_delete_transaction_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that deleting a transaction invalidates the materialized view.

        WHY: When a transaction is removed, the cached history includes data
        from a transaction that no longer exists. The view must be invalidated.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)

        # Create a transaction
        txn_date = (date.today() - timedelta(days=5)).isoformat()
        transaction = TransactionService.create_transaction(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "date": txn_date,
                "type": "buy",
                "shares": "10",
                "cost_per_share": "10.0",
            }
        )

        # Re-add materialized records
        self._add_materialized_records(db_session, portfolio_fund, fund)
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Delete the transaction
        TransactionService.delete_transaction(transaction.id)

        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_process_sell_transaction_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that a sell transaction with realized gain invalidates materialized view.

        WHY: Sell transactions create realized gains and reduce the position.
        Both affect the materialized history and must trigger invalidation.
        """
        _portfolio, fund, portfolio_fund = self._create_test_data(db_session)

        # Create initial buy transaction to have shares to sell
        buy_date = date.today() - timedelta(days=10)
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            date=buy_date,
            type="buy",
            shares=100.0,
            cost_per_share=10.0,
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Add materialized records
        self._add_materialized_records(db_session, portfolio_fund, fund)
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 3

        # Create sell transaction via create_transaction (which delegates to process_sell)
        sell_date = (date.today() - timedelta(days=2)).isoformat()
        TransactionService.create_transaction(
            {
                "portfolio_fund_id": portfolio_fund.id,
                "date": sell_date,
                "type": "sell",
                "shares": "10",
                "cost_per_share": "15.0",
            }
        )

        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_invalidation_only_affects_target_portfolio(self, app_context, db_session):
        """
        Test that invalidation only affects the specific portfolio, not others.

        WHY: Multiple portfolios may hold different funds. When a transaction
        is added to one portfolio, other portfolios should not be affected.
        """
        # Create two portfolios with different funds
        portfolio1 = Portfolio(id=make_id(), name="Portfolio 1")
        portfolio2 = Portfolio(id=make_id(), name="Portfolio 2")
        fund1 = Fund(
            id=make_id(),
            name="Fund 1",
            isin=f"US{make_id()[:8]}",
            symbol="TST1",
            currency="USD",
            exchange="NYSE",
            investment_type=InvestmentType.FUND,
        )
        fund2 = Fund(
            id=make_id(),
            name="Fund 2",
            isin=f"US{make_id()[:8]}",
            symbol="TST2",
            currency="USD",
            exchange="NYSE",
            investment_type=InvestmentType.FUND,
        )
        db_session.add_all([portfolio1, portfolio2, fund1, fund2])
        db_session.commit()

        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio1.id, fund_id=fund1.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund2.id)
        db_session.add_all([pf1, pf2])
        db_session.commit()

        # Add materialized records for both
        self._add_materialized_records(db_session, pf1, fund1)
        self._add_materialized_records(db_session, pf2, fund2)

        count_p1_before = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf1.id).count()
        count_p2_before = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf2.id).count()
        assert count_p1_before == 3
        assert count_p2_before == 3

        # Create transaction only in portfolio1
        txn_date = (date.today() - timedelta(days=5)).isoformat()
        TransactionService.create_transaction(
            {
                "portfolio_fund_id": pf1.id,
                "date": txn_date,
                "type": "buy",
                "shares": "10",
                "cost_per_share": "10.0",
            }
        )

        # Portfolio1 should be invalidated
        count_p1_after = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf1.id).count()
        assert count_p1_after == 0

        # Portfolio2 should remain untouched
        count_p2_after = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf2.id).count()
        assert count_p2_after == 3

    def test_invalidation_failure_does_not_break_transaction(self, app_context, db_session):
        """
        Test that a failed invalidation doesn't break the transaction operation.

        WHY: Invalidation is a secondary concern. If it fails, the primary
        transaction operation should still succeed.
        """
        _portfolio, _fund, portfolio_fund = self._create_test_data(db_session)

        with patch(
            "app.services.portfolio_history_materialized_service"
            ".PortfolioHistoryMaterializedService.invalidate_from_transaction",
            side_effect=Exception("Mock invalidation failure"),
        ):
            txn_date = (date.today() - timedelta(days=5)).isoformat()
            transaction = TransactionService.create_transaction(
                {
                    "portfolio_fund_id": portfolio_fund.id,
                    "date": txn_date,
                    "type": "buy",
                    "shares": "10",
                    "cost_per_share": "10.0",
                }
            )

            # Transaction should still be created successfully
            assert transaction is not None
            assert transaction.id is not None
            assert transaction.shares == 10.0
