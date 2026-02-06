"""
Tests for materialized view invalidation during IBKR transaction operations.

This test suite verifies that:
- Processing IBKR transactions invalidates materialized view
- Modifying allocations invalidates materialized view for old and new portfolios
- Unallocating transactions invalidates materialized view
- Bulk allocation operations trigger proper invalidation
"""

from datetime import date, timedelta

from app.models import (
    Fund,
    FundHistoryMaterialized,
    FundPrice,
    IBKRTransaction,
    Portfolio,
    PortfolioFund,
)
from app.services.ibkr_transaction_service import IBKRTransactionService
from tests.test_helpers import make_id


class TestIBKRTransactionMaterializedViewInvalidation:
    """Tests for materialized view invalidation during IBKR operations."""

    def test_process_allocation_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that processing IBKR transaction allocation invalidates materialized view.

        WHY: When new transactions are created through IBKR import, the historical
        fund data changes. The materialized view must be invalidated from the
        transaction date forward to ensure graphs show accurate data.
        """
        # Create portfolio and fund
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin="US12345678",
            symbol="TEST",
            currency="USD",
            exchange="NYSE",
        )
        db_session.add_all([portfolio, fund])
        db_session.commit()

        # Create IBKR transaction
        transaction_date = date.today() - timedelta(days=2)
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id="12345",
            transaction_date=transaction_date,
            symbol="TEST",
            isin="US12345678",
            description="Buy TEST",
            transaction_type="buy",
            quantity=10.0,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            fees=1.0,
            status="pending",
        )
        db_session.add(ibkr_txn)
        db_session.commit()

        # Create existing materialized data for dates after the transaction
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(portfolio_fund)
        db_session.commit()

        # Add materialized records for yesterday and today
        for i in range(2):
            record_date = date.today() - timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=portfolio_fund.id,
                fund_id=fund.id,
                date=record_date.isoformat(),
                shares=5.0,
                price=100.0,
                value=500.0,
                cost=500.0,
                realized_gain=0.0,
                unrealized_gain=0.0,
                total_gain_loss=0.0,
                dividends=0.0,
                fees=0.0,
            )
            db_session.add(record)
        db_session.commit()

        # Verify records exist before processing
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 2

        # Add fund price for the transaction date
        fund_price = FundPrice(
            fund_id=fund.id,
            date=transaction_date,
            price=100.0,
        )
        db_session.add(fund_price)
        db_session.commit()

        # Process allocation
        allocations = [{"portfolio_id": portfolio.id, "percentage": 100.0}]
        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        # Verify allocation succeeded
        assert result["success"] is True

        # Verify materialized view was invalidated (records from transaction date forward deleted)
        # Records should be deleted because they're stale after the new transaction
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        # The invalidation should have removed records from the transaction date onward
        assert count_after == 0

    def test_modify_allocations_invalidates_all_affected_portfolios(self, app_context, db_session):
        """
        Test that modifying allocations invalidates both old and new portfolios.

        WHY: When allocation percentages change, multiple portfolios are affected:
        - Old portfolios that had allocations (now reduced/removed)
        - New portfolios that received allocations
        All must have their materialized views invalidated.
        """
        # Create two portfolios
        portfolio1 = Portfolio(id=make_id(), name="Portfolio 1")
        portfolio2 = Portfolio(id=make_id(), name="Portfolio 2")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin="US12345678",
            symbol="TEST",
            currency="USD",
            exchange="NYSE",
        )
        db_session.add_all([portfolio1, portfolio2, fund])
        db_session.commit()

        # Create and process IBKR transaction with initial allocation to portfolio1
        transaction_date = date.today() - timedelta(days=3)
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id="12345",
            transaction_date=transaction_date,
            symbol="TEST",
            isin="US12345678",
            description="Buy TEST",
            transaction_type="buy",
            quantity=10.0,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            fees=1.0,
            status="pending",
        )
        db_session.add(ibkr_txn)
        db_session.commit()

        # Add fund price
        fund_price = FundPrice(fund_id=fund.id, date=transaction_date, price=100.0)
        db_session.add(fund_price)
        db_session.commit()

        # Initial allocation: 100% to portfolio1
        initial_allocations = [{"portfolio_id": portfolio1.id, "percentage": 100.0}]
        result = IBKRTransactionService.process_transaction_allocation(
            ibkr_txn.id, initial_allocations
        )
        assert result["success"] is True

        # Create materialized data for both portfolios
        pf1 = PortfolioFund.query.filter_by(portfolio_id=portfolio1.id, fund_id=fund.id).first()
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund.id)
        db_session.add(pf2)
        db_session.commit()

        for pf in [pf1, pf2]:
            for i in range(2):
                record_date = date.today() - timedelta(days=i)
                record = FundHistoryMaterialized(
                    portfolio_fund_id=pf.id,
                    fund_id=fund.id,
                    date=record_date.isoformat(),
                    shares=5.0,
                    price=100.0,
                    value=500.0,
                    cost=500.0,
                    realized_gain=0.0,
                    unrealized_gain=0.0,
                    total_gain_loss=0.0,
                    dividends=0.0,
                    fees=0.0,
                )
                db_session.add(record)
        db_session.commit()

        # Verify both portfolios have materialized data
        count_p1_before = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf1.id).count()
        count_p2_before = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf2.id).count()
        assert count_p1_before == 2
        assert count_p2_before == 2

        # Modify allocation: 50% to portfolio1, 50% to portfolio2
        new_allocations = [
            {"portfolio_id": portfolio1.id, "percentage": 50.0},
            {"portfolio_id": portfolio2.id, "percentage": 50.0},
        ]
        result = IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocations)
        assert result["success"] is True

        # Verify both portfolios' materialized views were invalidated
        count_p1_after = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf1.id).count()
        count_p2_after = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf2.id).count()

        # Both should be invalidated (cleared from transaction date forward)
        assert count_p1_after == 0
        assert count_p2_after == 0

    def test_unallocate_transaction_invalidates_materialized_view(self, app_context, db_session):
        """
        Test that unallocating transactions invalidates materialized view.

        WHY: When transactions are unallocated, they're deleted from the portfolio.
        The materialized view must be invalidated so graphs don't show incorrect
        historical data including the removed transactions.
        """
        # Create portfolio and fund
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin="US12345678",
            symbol="TEST",
            currency="USD",
            exchange="NYSE",
        )
        db_session.add_all([portfolio, fund])
        db_session.commit()

        # Create and allocate IBKR transaction
        transaction_date = date.today() - timedelta(days=2)
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id="12345",
            transaction_date=transaction_date,
            symbol="TEST",
            isin="US12345678",
            description="Buy TEST",
            transaction_type="buy",
            quantity=10.0,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            fees=1.0,
            status="pending",
        )
        db_session.add(ibkr_txn)
        db_session.commit()

        # Add fund price
        fund_price = FundPrice(fund_id=fund.id, date=transaction_date, price=100.0)
        db_session.add(fund_price)
        db_session.commit()

        # Process allocation
        allocations = [{"portfolio_id": portfolio.id, "percentage": 100.0}]
        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)
        assert result["success"] is True

        # Create materialized data
        portfolio_fund = PortfolioFund.query.filter_by(
            portfolio_id=portfolio.id, fund_id=fund.id
        ).first()
        for i in range(2):
            record_date = date.today() - timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=portfolio_fund.id,
                fund_id=fund.id,
                date=record_date.isoformat(),
                shares=10.0,
                price=100.0,
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

        # Verify materialized data exists
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 2

        # Unallocate transaction
        response, status = IBKRTransactionService.unallocate_transaction(ibkr_txn.id)
        assert status == 200
        assert response["success"] is True

        # Verify materialized view was invalidated
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_bulk_allocate_invalidates_all_portfolios(self, app_context, db_session):
        """
        Test that bulk allocation invalidates materialized view for all transactions.

        WHY: Bulk allocation processes multiple transactions at once. Each transaction
        affects the portfolio's history, so all affected portfolios must have their
        materialized views invalidated.
        """
        # Create portfolio and fund
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin="US12345678",
            symbol="TEST",
            currency="USD",
            exchange="NYSE",
        )
        db_session.add_all([portfolio, fund])
        db_session.commit()

        # Create multiple IBKR transactions
        transaction_ids = []
        base_date = date.today() - timedelta(days=5)
        for i in range(3):
            txn_date = base_date + timedelta(days=i)
            ibkr_txn = IBKRTransaction(
                id=make_id(),
                ibkr_transaction_id=f"TXN{i}",
                transaction_date=txn_date,
                symbol="TEST",
                isin="US12345678",
                description=f"Buy TEST {i}",
                transaction_type="buy",
                quantity=10.0,
                price=100.0,
                total_amount=1000.0,
                currency="USD",
                fees=1.0,
                status="pending",
            )
            db_session.add(ibkr_txn)
            transaction_ids.append(ibkr_txn.id)

            # Add fund price for each date
            fund_price = FundPrice(fund_id=fund.id, date=txn_date, price=100.0)
            db_session.add(fund_price)
        db_session.commit()

        # Create pre-existing materialized data
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(portfolio_fund)
        db_session.commit()

        for i in range(6):
            record_date = date.today() - timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=portfolio_fund.id,
                fund_id=fund.id,
                date=record_date.isoformat(),
                shares=5.0,
                price=100.0,
                value=500.0,
                cost=500.0,
                realized_gain=0.0,
                unrealized_gain=0.0,
                total_gain_loss=0.0,
                dividends=0.0,
                fees=0.0,
            )
            db_session.add(record)
        db_session.commit()

        # Verify materialized data exists
        count_before = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_before == 6

        # Bulk allocate all transactions
        allocations = [{"portfolio_id": portfolio.id, "percentage": 100.0}]
        result = IBKRTransactionService.bulk_allocate_transactions(transaction_ids, allocations)
        assert result["success"] is True
        assert result["processed"] == 3

        # Verify materialized view was invalidated
        # Should be cleared from the earliest transaction date forward
        count_after = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=portfolio_fund.id
        ).count()
        assert count_after == 0

    def test_invalidation_only_affects_target_portfolio(self, app_context, db_session):
        """
        Test that invalidation only affects the specific portfolio, not others.

        WHY: Multiple portfolios may hold the same fund. When transactions are
        added to one portfolio, only that portfolio's materialized view should
        be invalidated, not other unrelated portfolios.
        """
        # Create two portfolios with the same fund
        portfolio1 = Portfolio(id=make_id(), name="Portfolio 1")
        portfolio2 = Portfolio(id=make_id(), name="Portfolio 2")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin="US12345678",
            symbol="TEST",
            currency="USD",
            exchange="NYSE",
        )
        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio1.id, fund_id=fund.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund.id)
        db_session.add_all([portfolio1, portfolio2, fund, pf1, pf2])
        db_session.commit()

        # Create IBKR transaction
        transaction_date = date.today() - timedelta(days=2)
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id="12345",
            transaction_date=transaction_date,
            symbol="TEST",
            isin="US12345678",
            description="Buy TEST",
            transaction_type="buy",
            quantity=10.0,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            fees=1.0,
            status="pending",
        )
        db_session.add(ibkr_txn)
        db_session.commit()

        # Add fund price
        fund_price = FundPrice(fund_id=fund.id, date=transaction_date, price=100.0)
        db_session.add(fund_price)
        db_session.commit()

        # Create materialized data for both portfolios
        for pf in [pf1, pf2]:
            for i in range(2):
                record_date = date.today() - timedelta(days=i)
                record = FundHistoryMaterialized(
                    portfolio_fund_id=pf.id,
                    fund_id=fund.id,
                    date=record_date.isoformat(),
                    shares=5.0,
                    price=100.0,
                    value=500.0,
                    cost=500.0,
                    realized_gain=0.0,
                    unrealized_gain=0.0,
                    total_gain_loss=0.0,
                    dividends=0.0,
                    fees=0.0,
                )
                db_session.add(record)
        db_session.commit()

        # Allocate transaction to portfolio1 only
        allocations = [{"portfolio_id": portfolio1.id, "percentage": 100.0}]
        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)
        assert result["success"] is True

        # Portfolio1 should be invalidated
        count_p1 = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf1.id).count()
        assert count_p1 == 0

        # Portfolio2 should remain untouched
        count_p2 = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf2.id).count()
        assert count_p2 == 2
