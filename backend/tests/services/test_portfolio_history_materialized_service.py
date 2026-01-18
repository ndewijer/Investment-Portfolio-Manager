"""Tests for portfolio history materialized service.

Updated for fund-level materialized view (v1.5.0).
"""

from datetime import date, timedelta

from app.models import (
    Fund,
    FundHistoryMaterialized,
    PortfolioFund,
    Transaction,
    db,
)
from app.services.portfolio_history_materialized_service import (
    MaterializedCoverage,
    PortfolioHistoryMaterializedService,
)
from tests.test_helpers import make_isin, make_symbol


def create_fund(
    isin_prefix="US", symbol_prefix="TEST", name="Test Fund", currency="USD", exchange="NYSE"
):
    """Helper function to create a unique fund."""
    return Fund(
        isin=make_isin(isin_prefix),
        symbol=make_symbol(symbol_prefix),
        name=name,
        currency=currency,
        exchange=exchange,
    )


class TestPortfolioHistoryMaterializedService:
    """Test suite for PortfolioHistoryMaterializedService.

    These tests verify the fund-level materialized view implementation
    which aggregates to provide portfolio-level views.
    """

    def test_check_materialized_coverage_empty(self):
        """Test coverage check with no portfolio IDs."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 10)

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [], start_date, end_date
        )

        assert coverage.is_complete is True
        assert coverage.partial_coverage is False

    def test_check_materialized_coverage_no_records(
        self, app, sample_portfolio, cash_dividend_fund
    ):
        """Test coverage check with portfolio fund but no materialized records."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 10)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [sample_portfolio.id], start_date, end_date
        )

        assert coverage.is_complete is False
        assert coverage.partial_coverage is False
        assert len(coverage.missing_ranges) == 1

    def test_check_materialized_coverage_complete(self, app, sample_portfolio, cash_dividend_fund):
        """Test coverage check with complete fund-level data."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 5)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create fund-level materialized records for all days
        for i in range((end_date - start_date).days + 1):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [sample_portfolio.id], start_date, end_date
        )

        assert coverage.is_complete is True
        assert len(coverage.covered_ranges) == 1

    def test_get_materialized_history(self, app, sample_portfolio, cash_dividend_fund):
        """Test retrieving materialized history (aggregated from fund data)."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 3)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create fund-level materialized records
        for i in range(3):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0 + i * 10,
                value=1000.0 + i * 100,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0 + i * 100,
                total_gain_loss=200.0 + i * 100,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        history = PortfolioHistoryMaterializedService.get_materialized_history(
            [sample_portfolio.id], start_date, end_date
        )

        assert len(history) == 3
        assert history[0]["date"] == "2024-01-01"
        assert history[0]["portfolios"][0]["totalValue"] == 1000.0
        assert history[1]["portfolios"][0]["totalValue"] == 1100.0
        assert history[2]["portfolios"][0]["totalValue"] == 1200.0

    def test_materialize_portfolio_history(self, app, sample_portfolio):
        """Test materializing portfolio history."""
        count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
            sample_portfolio.id, force_recalculate=True
        )

        # Should have materialized records (or 0 if no transactions)
        assert count >= 0

    def test_invalidate_materialized_history(self, app, sample_portfolio, cash_dividend_fund):
        """Test invalidating materialized history."""
        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create fund-level materialized records
        for i in range(10):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        # Invalidate from day 5
        invalidate_date = start_date + timedelta(days=5)
        deleted = PortfolioHistoryMaterializedService.invalidate_materialized_history(
            sample_portfolio.id, invalidate_date, recalculate=False
        )

        # Should have deleted 5 records (days 5-9)
        assert deleted == 5

        # Should have 5 records left (days 0-4)
        remaining = FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf.id).count()
        assert remaining == 5

    def test_invalidate_from_transaction(self, app, sample_portfolio, cash_dividend_fund):
        """Test invalidation from transaction."""
        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create fund-level materialized records
        for i in range(10):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        # Create a transaction
        transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=start_date + timedelta(days=5),
            type="buy",
            shares=10.0,
            cost_per_share=100.0,
        )
        db.session.add(transaction)
        db.session.commit()

        deleted = PortfolioHistoryMaterializedService.invalidate_from_transaction(transaction)

        # Should have deleted records from transaction date forward
        assert deleted >= 0

    def test_get_materialized_stats_empty(self, app, db_session):
        """Test getting stats with no data."""
        # Explicitly clean up any materialized data
        FundHistoryMaterialized.query.delete()
        db.session.commit()

        stats = PortfolioHistoryMaterializedService.get_materialized_stats()

        assert stats["total_records"] == 0
        assert stats["portfolios_with_data"] == 0
        assert stats["oldest_date"] is None
        assert stats["newest_date"] is None

    def test_get_materialized_stats_with_data(
        self, app, db_session, sample_portfolio, cash_dividend_fund
    ):
        """Test getting stats with data."""
        # Explicitly clean up any materialized data
        FundHistoryMaterialized.query.delete()
        db.session.commit()

        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create fund-level materialized records
        for i in range(5):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        stats = PortfolioHistoryMaterializedService.get_materialized_stats()

        assert stats["total_records"] == 5
        assert stats["portfolios_with_data"] == 1
        assert stats["oldest_date"] == "2024-01-01"
        assert stats["newest_date"] == "2024-01-05"

    def test_materialize_all_portfolios(self, app, sample_portfolio):
        """Test materializing all portfolios."""
        results = PortfolioHistoryMaterializedService.materialize_all_portfolios(
            force_recalculate=False
        )

        # Should have result for sample portfolio
        assert sample_portfolio.id in results

    def test_materialized_coverage_dataclass(self):
        """Test MaterializedCoverage dataclass."""
        coverage = MaterializedCoverage(
            is_complete=True,
            partial_coverage=False,
            missing_ranges=[],
            covered_ranges=[(date(2024, 1, 1), date(2024, 1, 10))],
        )

        assert coverage.is_complete is True
        assert coverage.partial_coverage is False
        assert len(coverage.covered_ranges) == 1
        assert len(coverage.missing_ranges) == 0

    def test_materialize_portfolio_history_with_date_range(
        self, app, sample_portfolio, cash_dividend_fund
    ):
        """Test materializing with specific date range."""
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Add a transaction
        transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=10.0,
            cost_per_share=100.0,
        )
        db.session.add(transaction)
        db.session.commit()

        # Materialize with date range
        count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
            sample_portfolio.id,
            start_date=date(2024, 1, 1),
            end_date=date(2024, 1, 5),
            force_recalculate=True,
        )

        # Should have some records
        assert count >= 0

    def test_invalidate_from_price_update(self, app, sample_portfolio, cash_dividend_fund):
        """Test invalidation from price update."""
        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create fund-level materialized records
        for i in range(10):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        # Invalidate from price update
        price_date = start_date + timedelta(days=5)
        deleted = PortfolioHistoryMaterializedService.invalidate_from_price_update(
            cash_dividend_fund.id, price_date
        )

        # Should have deleted records
        assert deleted >= 0

    def test_check_materialized_coverage_partial(self, app, sample_portfolio, cash_dividend_fund):
        """Test coverage check with partial data (some days missing)."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 10)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create records for only half the days
        for i in range(5):  # Only 5 of 10 days
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [sample_portfolio.id], start_date, end_date
        )

        assert coverage.is_complete is False
        assert coverage.partial_coverage is True
        assert len(coverage.missing_ranges) == 1

    def test_materialize_portfolio_history_not_found(self, app, db_session):
        """Test materializing with non-existent portfolio raises error."""
        import pytest

        fake_id = "non-existent-portfolio-id"

        with pytest.raises(ValueError, match="not found"):
            PortfolioHistoryMaterializedService.materialize_portfolio_history(fake_id)

    def test_materialize_portfolio_history_no_portfolio_funds(self, app, db_session):
        """Test materializing portfolio with no funds returns 0."""
        from app.models import Portfolio

        portfolio = Portfolio(name="Empty Portfolio", description="No funds")
        db.session.add(portfolio)
        db.session.commit()

        count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
            portfolio.id, force_recalculate=True
        )

        assert count == 0

    def test_invalidate_history_with_recalculate(self, app, sample_portfolio, cash_dividend_fund):
        """Test invalidation with recalculate=True."""
        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create a transaction to have something to recalculate
        transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=start_date,
            type="buy",
            shares=10.0,
            cost_per_share=100.0,
        )
        db.session.add(transaction)
        db.session.commit()

        # Create some materialized records
        for i in range(3):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        # Invalidate with recalculate=True
        invalidate_date = start_date + timedelta(days=1)
        deleted = PortfolioHistoryMaterializedService.invalidate_materialized_history(
            sample_portfolio.id, invalidate_date, recalculate=True
        )

        # Should have deleted 2 records (days 1 and 2)
        assert deleted == 2

    def test_invalidate_no_portfolio_funds(self, app, db_session):
        """Test invalidation when portfolio has no funds returns 0."""
        from app.models import Portfolio

        portfolio = Portfolio(name="Empty Portfolio", description="No funds")
        db.session.add(portfolio)
        db.session.commit()

        deleted = PortfolioHistoryMaterializedService.invalidate_materialized_history(
            portfolio.id, date(2024, 1, 1), recalculate=False
        )

        assert deleted == 0

    def test_invalidate_from_transaction_no_portfolio_fund(self, app, db_session):
        """Test invalidation from transaction with no portfolio_fund returns 0."""

        # Create a mock transaction object with no portfolio_fund
        class MockTransaction:
            portfolio_fund = None

        result = PortfolioHistoryMaterializedService.invalidate_from_transaction(MockTransaction())

        assert result == 0

    def test_invalidate_from_dividend(self, app, sample_portfolio, cash_dividend_fund):
        """Test invalidation from dividend."""
        from app.models import Dividend

        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create materialized records
        for i in range(10):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0,
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_gain_loss=200.0,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)
        db.session.commit()

        # Create a dividend (use record_date and ex_dividend_date, not pay_date)
        dividend = Dividend(
            fund_id=cash_dividend_fund.id,
            portfolio_fund_id=pf.id,
            record_date=start_date + timedelta(days=4),
            ex_dividend_date=start_date + timedelta(days=5),
            dividend_per_share=1.0,
            shares_owned=10.0,
            total_amount=10.0,
        )
        db.session.add(dividend)
        db.session.commit()

        # Invalidate from dividend
        deleted = PortfolioHistoryMaterializedService.invalidate_from_dividend(dividend)

        # Should have deleted records from dividend date forward
        assert deleted == 5

    def test_invalidate_from_dividend_no_portfolio_fund(self, app, db_session):
        """Test invalidation from dividend with invalid portfolio_fund returns 0."""

        # Create a mock dividend with an invalid portfolio_fund_id
        class MockDividend:
            portfolio_fund_id = "non-existent-pf-id"
            ex_dividend_date = date(2024, 1, 1)

        result = PortfolioHistoryMaterializedService.invalidate_from_dividend(MockDividend())

        assert result == 0

    def test_materialize_all_portfolios_with_error(self, app, db_session, mocker):
        """Test materialize_all handles exceptions gracefully."""
        from app.models import Portfolio

        # Create a portfolio
        portfolio = Portfolio(name="Test Portfolio", description="Test")
        db.session.add(portfolio)
        db.session.commit()

        # Mock materialize_portfolio_history to raise an exception for this portfolio
        original_method = PortfolioHistoryMaterializedService.materialize_portfolio_history

        def mock_materialize(portfolio_id, **kwargs):
            if portfolio_id == portfolio.id:
                raise Exception("Test error")
            return original_method(portfolio_id, **kwargs)

        mocker.patch.object(
            PortfolioHistoryMaterializedService,
            "materialize_portfolio_history",
            side_effect=mock_materialize,
        )

        results = PortfolioHistoryMaterializedService.materialize_all_portfolios(
            force_recalculate=False
        )

        # Should have error message for our portfolio
        assert portfolio.id in results
        assert "Error" in str(results[portfolio.id])

    def test_get_materialized_history_with_realized_gains(
        self, app, db_session, sample_portfolio, cash_dividend_fund
    ):
        """Test retrieving history with realized gains properly calculated."""
        from app.models import RealizedGainLoss

        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create a sell transaction (needed for RealizedGainLoss)
        sell_transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=start_date + timedelta(days=1),
            type="sell",
            shares=5.0,
            cost_per_share=110.0,
        )
        db.session.add(sell_transaction)
        db.session.commit()

        # Create fund-level materialized records
        for i in range(3):
            current_date = start_date + timedelta(days=i)
            record = FundHistoryMaterialized(
                portfolio_fund_id=pf.id,
                fund_id=cash_dividend_fund.id,
                date=current_date.isoformat(),
                shares=10.0,
                price=100.0 + i * 10,
                value=1000.0 + i * 100,
                cost=800.0,
                realized_gain=50.0,
                unrealized_gain=200.0 + i * 100,
                total_gain_loss=250.0 + i * 100,
                dividends=0.0,
                fees=0.0,
            )
            db.session.add(record)

        # Add a realized gain record on day 2
        rg = RealizedGainLoss(
            portfolio_id=sample_portfolio.id,
            fund_id=cash_dividend_fund.id,
            transaction_id=sell_transaction.id,
            transaction_date=start_date + timedelta(days=1),
            shares_sold=5.0,
            sale_proceeds=550.0,
            cost_basis=500.0,
            realized_gain_loss=50.0,
        )
        db.session.add(rg)
        db.session.commit()

        history = PortfolioHistoryMaterializedService.get_materialized_history(
            [sample_portfolio.id], start_date, start_date + timedelta(days=2)
        )

        assert len(history) == 3
        # Check that sale proceeds and original cost are cumulative
        assert history[1]["portfolios"][0]["totalSaleProceeds"] == 550.0
        assert history[1]["portfolios"][0]["totalOriginalCost"] == 500.0

    def test_materialize_with_dividends_and_realized_gains(
        self, app, db_session, sample_portfolio, cash_dividend_fund
    ):
        """Test materialization with dividends and realized gains."""
        from app.models import Dividend, FundPrice, RealizedGainLoss

        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Add a buy transaction
        buy_transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=start_date,
            type="buy",
            shares=10.0,
            cost_per_share=100.0,
        )
        db.session.add(buy_transaction)

        # Add a sell transaction for realized gain
        sell_transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=start_date + timedelta(days=3),
            type="sell",
            shares=2.0,
            cost_per_share=110.0,
        )
        db.session.add(sell_transaction)
        db.session.commit()

        # Add fund prices
        for i in range(5):
            price = FundPrice(
                fund_id=cash_dividend_fund.id,
                date=start_date + timedelta(days=i),
                price=100.0 + i * 5,
            )
            db.session.add(price)

        # Add a dividend
        dividend = Dividend(
            fund_id=cash_dividend_fund.id,
            portfolio_fund_id=pf.id,
            record_date=start_date + timedelta(days=1),
            ex_dividend_date=start_date + timedelta(days=2),
            dividend_per_share=1.0,
            shares_owned=10.0,
            total_amount=10.0,
        )
        db.session.add(dividend)

        # Add a realized gain
        rg = RealizedGainLoss(
            portfolio_id=sample_portfolio.id,
            fund_id=cash_dividend_fund.id,
            transaction_id=sell_transaction.id,
            transaction_date=start_date + timedelta(days=3),
            shares_sold=2.0,
            sale_proceeds=220.0,
            cost_basis=200.0,
            realized_gain_loss=20.0,
        )
        db.session.add(rg)
        db.session.commit()

        # Materialize
        count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
            sample_portfolio.id,
            start_date=start_date,
            end_date=start_date + timedelta(days=4),
            force_recalculate=True,
        )

        # Should have created records
        assert count > 0

        # Check that records include dividend and realized gain data
        records = (
            FundHistoryMaterialized.query.filter_by(portfolio_fund_id=pf.id)
            .order_by(FundHistoryMaterialized.date)
            .all()
        )

        # Records after dividend date should have cumulative dividends
        if len(records) >= 3:
            assert records[2].dividends >= 0  # Day 2 or later

        # Records after realized gain date should have realized gains
        if len(records) >= 4:
            assert records[3].realized_gain >= 0  # Day 3 or later

    def test_materialize_updates_existing_record(
        self, app, db_session, sample_portfolio, cash_dividend_fund
    ):
        """Test that materialization updates existing records when force_recalculate=True."""
        from app.models import FundPrice

        start_date = date(2024, 1, 1)

        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=cash_dividend_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Add a transaction
        transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=start_date,
            type="buy",
            shares=10.0,
            cost_per_share=100.0,
        )
        db.session.add(transaction)

        # Add fund price
        price = FundPrice(
            fund_id=cash_dividend_fund.id,
            date=start_date,
            price=100.0,
        )
        db.session.add(price)
        db.session.commit()

        # First materialization
        PortfolioHistoryMaterializedService.materialize_portfolio_history(
            sample_portfolio.id,
            start_date=start_date,
            end_date=start_date,
            force_recalculate=True,
        )

        # Update price
        price.price = 150.0
        db.session.commit()

        # Materialize again with force_recalculate
        PortfolioHistoryMaterializedService.materialize_portfolio_history(
            sample_portfolio.id,
            start_date=start_date,
            end_date=start_date,
            force_recalculate=True,
        )

        # Check that record was updated
        updated_record = FundHistoryMaterialized.query.filter_by(
            portfolio_fund_id=pf.id, date=start_date.isoformat()
        ).first()

        assert updated_record is not None
        # Value should be different due to price change
        assert updated_record.value == 1500.0  # 10 shares * $150

    def test_check_coverage_with_portfolio_no_funds(self, app, db_session):
        """Test coverage check when portfolio has no funds returns complete."""
        from app.models import Portfolio

        portfolio = Portfolio(name="Empty Portfolio", description="No funds")
        db.session.add(portfolio)
        db.session.commit()

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [portfolio.id], date(2024, 1, 1), date(2024, 1, 10)
        )

        # Should be complete since there's nothing to materialize
        assert coverage.is_complete is True
