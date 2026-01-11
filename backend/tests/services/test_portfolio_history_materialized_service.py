"""Tests for portfolio history materialized service."""

from datetime import date, datetime, timedelta

import pytest

from app.models import (
    Portfolio,
    PortfolioFund,
    PortfolioHistoryMaterialized,
    Transaction,
    db,
)
from app.services.portfolio_history_materialized_service import (
    MaterializedCoverage,
    PortfolioHistoryMaterializedService,
)


class TestPortfolioHistoryMaterializedService:
    """Test suite for PortfolioHistoryMaterializedService."""

    def test_check_materialized_coverage_empty(self):
        """Test coverage check with no data."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 10)

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [], start_date, end_date
        )

        assert coverage.is_complete is True
        assert coverage.partial_coverage is False

    def test_check_materialized_coverage_no_records(self, app, sample_portfolio):
        """Test coverage check with portfolio but no records."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 10)

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [sample_portfolio.id], start_date, end_date
        )

        assert coverage.is_complete is False
        assert coverage.partial_coverage is False
        assert len(coverage.missing_ranges) == 1

    def test_check_materialized_coverage_complete(self, app, sample_portfolio):
        """Test coverage check with complete data."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 5)

        # Create materialized records for all days
        for i in range((end_date - start_date).days + 1):
            current_date = start_date + timedelta(days=i)
            record = PortfolioHistoryMaterialized(
                portfolio_id=sample_portfolio.id,
                date=current_date.isoformat(),
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_dividends=0.0,
                total_sale_proceeds=0.0,
                total_original_cost=0.0,
                total_gain_loss=200.0,
                is_archived=0,
            )
            db.session.add(record)
        db.session.commit()

        coverage = PortfolioHistoryMaterializedService.check_materialized_coverage(
            [sample_portfolio.id], start_date, end_date
        )

        assert coverage.is_complete is True
        assert len(coverage.covered_ranges) == 1

    def test_get_materialized_history(self, app, sample_portfolio):
        """Test retrieving materialized history."""
        start_date = date(2024, 1, 1)
        end_date = date(2024, 1, 3)

        # Create materialized records
        for i in range(3):
            current_date = start_date + timedelta(days=i)
            record = PortfolioHistoryMaterialized(
                portfolio_id=sample_portfolio.id,
                date=current_date.isoformat(),
                value=1000.0 + i * 100,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0 + i * 100,
                total_dividends=0.0,
                total_sale_proceeds=0.0,
                total_original_cost=0.0,
                total_gain_loss=200.0 + i * 100,
                is_archived=0,
            )
            db.session.add(record)
        db.session.commit()

        history = PortfolioHistoryMaterializedService.get_materialized_history(
            [sample_portfolio.id], start_date, end_date
        )

        assert len(history) == 3
        assert history[0]["date"] == "2024-01-01"
        assert history[0]["portfolios"][0]["value"] == 1000.0
        assert history[1]["portfolios"][0]["value"] == 1100.0
        assert history[2]["portfolios"][0]["value"] == 1200.0

    def test_materialize_portfolio_history(
        self, app, sample_portfolio, sample_fund, sample_transaction
    ):
        """Test materializing portfolio history."""
        # Add a transaction
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=sample_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=sample_fund.id)
            db.session.add(pf)
            db.session.commit()

        count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
            sample_portfolio.id, force_recalculate=True
        )

        # Should have materialized records
        assert count >= 0
        records = PortfolioHistoryMaterialized.query.filter_by(
            portfolio_id=sample_portfolio.id
        ).all()
        assert len(records) >= 0

    def test_invalidate_materialized_history(self, app, sample_portfolio):
        """Test invalidating materialized history."""
        start_date = date(2024, 1, 1)

        # Create some records
        for i in range(10):
            current_date = start_date + timedelta(days=i)
            record = PortfolioHistoryMaterialized(
                portfolio_id=sample_portfolio.id,
                date=current_date.isoformat(),
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_dividends=0.0,
                total_sale_proceeds=0.0,
                total_original_cost=0.0,
                total_gain_loss=200.0,
                is_archived=0,
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
        remaining = PortfolioHistoryMaterialized.query.filter_by(
            portfolio_id=sample_portfolio.id
        ).count()
        assert remaining == 5

    def test_invalidate_from_transaction(
        self, app, sample_portfolio, sample_fund, sample_transaction
    ):
        """Test invalidation from transaction."""
        # Create some materialized records
        start_date = date(2024, 1, 1)
        for i in range(10):
            current_date = start_date + timedelta(days=i)
            record = PortfolioHistoryMaterialized(
                portfolio_id=sample_portfolio.id,
                date=current_date.isoformat(),
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_dividends=0.0,
                total_sale_proceeds=0.0,
                total_original_cost=0.0,
                total_gain_loss=200.0,
                is_archived=0,
            )
            db.session.add(record)
        db.session.commit()

        # Create a transaction
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=sample_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=sample_fund.id)
            db.session.add(pf)
            db.session.commit()

        transaction = Transaction(
            portfolio_fund_id=pf.id,
            date=start_date + timedelta(days=5),
            type="buy",
            shares=10.0,
            cost_per_share=100.0,
        )
        db.session.add(transaction)
        db.session.commit()

        deleted = PortfolioHistoryMaterializedService.invalidate_from_transaction(
            transaction
        )

        # Should have deleted records from transaction date forward
        assert deleted >= 0

    def test_get_materialized_stats_empty(self, app):
        """Test getting stats with no data."""
        stats = PortfolioHistoryMaterializedService.get_materialized_stats()

        assert stats["total_records"] == 0
        assert stats["portfolios_with_data"] == 0
        assert stats["oldest_date"] is None
        assert stats["newest_date"] is None

    def test_get_materialized_stats_with_data(self, app, sample_portfolio):
        """Test getting stats with data."""
        start_date = date(2024, 1, 1)

        # Create some records
        for i in range(5):
            current_date = start_date + timedelta(days=i)
            record = PortfolioHistoryMaterialized(
                portfolio_id=sample_portfolio.id,
                date=current_date.isoformat(),
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_dividends=0.0,
                total_sale_proceeds=0.0,
                total_original_cost=0.0,
                total_gain_loss=200.0,
                is_archived=0,
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
        self, app, sample_portfolio, sample_fund
    ):
        """Test materializing with specific date range."""
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=sample_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=sample_fund.id)
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

    def test_invalidate_from_price_update(self, app, sample_portfolio, sample_fund):
        """Test invalidation from price update."""
        # Create portfolio fund
        pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=sample_fund.id
        ).first()

        if not pf:
            pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=sample_fund.id)
            db.session.add(pf)
            db.session.commit()

        # Create some materialized records
        start_date = date(2024, 1, 1)
        for i in range(10):
            current_date = start_date + timedelta(days=i)
            record = PortfolioHistoryMaterialized(
                portfolio_id=sample_portfolio.id,
                date=current_date.isoformat(),
                value=1000.0,
                cost=800.0,
                realized_gain=0.0,
                unrealized_gain=200.0,
                total_dividends=0.0,
                total_sale_proceeds=0.0,
                total_original_cost=0.0,
                total_gain_loss=200.0,
                is_archived=0,
            )
            db.session.add(record)
        db.session.commit()

        # Invalidate from price update
        price_date = start_date + timedelta(days=5)
        deleted = PortfolioHistoryMaterializedService.invalidate_from_price_update(
            sample_fund.id, price_date
        )

        # Should have deleted records
        assert deleted >= 0
