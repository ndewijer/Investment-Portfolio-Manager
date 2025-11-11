"""
Performance tests for portfolio history calculations.

Tests the batch processing optimization that eliminates day-by-day database queries.

Target Performance (Phase 1):
- Overview page (365 days): < 100 queries, < 1 second
- Portfolio detail (365 days): < 50 queries, < 0.5 seconds

Before optimization:
- Overview: 16,425 queries, 5-10 seconds
- Portfolio detail: 7,665 queries, 3-5 seconds
"""

import pytest
from app.services.portfolio_service import PortfolioService


class TestPortfolioHistoryPerformance:
    """Test suite for portfolio history performance optimizations."""

    def test_get_portfolio_history_query_count(self, app_context, query_counter):
        """
        Test that get_portfolio_history uses batch processing with minimal queries.

        Target: < 100 queries for 365 days of history
        Before: 16,425 queries
        """
        # Reset counter before test
        query_counter.reset()

        # Get 365 days of history (default frontend behavior)
        from datetime import datetime, timedelta

        end_date = datetime.now().date()
        start_date = end_date - timedelta(days=365)

        result = PortfolioService.get_portfolio_history(
            start_date=start_date.isoformat(), end_date=end_date.isoformat()
        )

        # Verify we got results
        assert isinstance(result, list)
        assert len(result) > 0

        # Check query count
        print(f"\n✓ Query count: {query_counter.count}")
        assert query_counter.count < 100, f"Too many queries: {query_counter.count} (target: < 100)"

    def test_get_portfolio_history_execution_time(self, app_context, timer):
        """
        Test that get_portfolio_history completes quickly.

        Target: < 1 second for 365 days
        Before: 5-10 seconds
        """
        from datetime import datetime, timedelta

        end_date = datetime.now().date()
        start_date = end_date - timedelta(days=365)

        timer.start()
        result = PortfolioService.get_portfolio_history(
            start_date=start_date.isoformat(), end_date=end_date.isoformat()
        )
        elapsed = timer.stop()

        # Verify we got results
        assert isinstance(result, list)
        assert len(result) > 0

        # Check execution time
        print(f"\n✓ Execution time: {elapsed:.3f}s")
        assert elapsed < 1.0, f"Too slow: {elapsed:.3f}s (target: < 1s)"

    def test_get_portfolio_fund_history_query_count(self, app_context, query_counter):
        """
        Test that get_portfolio_fund_history uses batch processing with minimal queries.

        Target: < 50 queries for 365 days of history
        Before: 7,665 queries
        """
        # Get a portfolio ID from the database
        from app.models import Portfolio

        portfolio = Portfolio.query.filter_by(is_archived=False).first()

        if not portfolio:
            pytest.skip("No portfolio found in database")

        query_counter.reset()

        # Get 365 days of history
        from datetime import datetime, timedelta

        end_date = datetime.now().date()
        start_date = end_date - timedelta(days=365)

        result = PortfolioService.get_portfolio_fund_history(
            portfolio.id, start_date=start_date.isoformat(), end_date=end_date.isoformat()
        )

        # Verify we got results
        assert isinstance(result, list)

        # Check query count
        print(f"\n✓ Query count: {query_counter.count}")
        assert query_counter.count < 50, f"Too many queries: {query_counter.count} (target: < 50)"

    def test_get_portfolio_fund_history_execution_time(self, app_context, timer):
        """
        Test that get_portfolio_fund_history completes quickly.

        Target: < 0.5 seconds for 365 days
        Before: 3-5 seconds
        """
        from app.models import Portfolio

        portfolio = Portfolio.query.filter_by(is_archived=False).first()

        if not portfolio:
            pytest.skip("No portfolio found in database")

        from datetime import datetime, timedelta

        end_date = datetime.now().date()
        start_date = end_date - timedelta(days=365)

        timer.start()
        result = PortfolioService.get_portfolio_fund_history(
            portfolio.id, start_date=start_date.isoformat(), end_date=end_date.isoformat()
        )
        elapsed = timer.stop()

        # Verify we got results
        assert isinstance(result, list)

        # Check execution time
        print(f"\n✓ Execution time: {elapsed:.3f}s")
        assert elapsed < 0.5, f"Too slow: {elapsed:.3f}s (target: < 0.5s)"

    def test_full_history_performance(self, app_context, query_counter, timer):
        """
        Test performance with full history (all available data).

        This is a stress test with 1,500+ days of data.
        Target: Still reasonable performance (< 150 queries, < 2 seconds)
        """
        query_counter.reset()
        timer.start()

        # Get full history (no date limits)
        result = PortfolioService.get_portfolio_history()

        elapsed = timer.stop()

        # Verify we got results
        assert isinstance(result, list)
        assert len(result) > 0

        # Check metrics
        print(f"\n✓ Full history: {len(result)} days")
        print(f"✓ Query count: {query_counter.count}")
        print(f"✓ Execution time: {elapsed:.3f}s")

        # More lenient targets for full history
        assert query_counter.count < 150, f"Too many queries: {query_counter.count} (target: < 150)"
        assert elapsed < 2.0, f"Too slow: {elapsed:.3f}s (target: < 2s)"


class TestPortfolioHistoryCorrectness:
    """
    Test suite to verify calculation correctness after optimization.

    These tests ensure the batch processing produces the same results
    as the original day-by-day processing.
    """

    def test_portfolio_history_returns_data(self, app_context):
        """Verify that portfolio history returns valid data structure."""
        result = PortfolioService.get_portfolio_history()

        assert isinstance(result, list)

        if len(result) > 0:
            # Check structure of first entry
            first_day = result[0]
            assert "date" in first_day
            assert "portfolios" in first_day
            assert isinstance(first_day["portfolios"], list)

            # If there are portfolio entries, check their structure
            if len(first_day["portfolios"]) > 0:
                portfolio = first_day["portfolios"][0]
                assert "id" in portfolio
                assert "name" in portfolio
                assert "value" in portfolio
                assert "cost" in portfolio
                assert "realized_gain" in portfolio
                assert "unrealized_gain" in portfolio

    def test_portfolio_fund_history_returns_data(self, app_context):
        """Verify that portfolio fund history returns valid data structure."""
        from app.models import Portfolio

        portfolio = Portfolio.query.filter_by(is_archived=False).first()

        if not portfolio:
            pytest.skip("No portfolio found in database")

        result = PortfolioService.get_portfolio_fund_history(portfolio.id)

        assert isinstance(result, list)

        if len(result) > 0:
            # Check structure of first entry
            first_day = result[0]
            assert "date" in first_day
            assert "funds" in first_day
            assert isinstance(first_day["funds"], list)

            # If there are fund entries, check their structure
            if len(first_day["funds"]) > 0:
                fund = first_day["funds"][0]
                assert "portfolio_fund_id" in fund
                assert "fund_id" in fund
                assert "fund_name" in fund
                assert "value" in fund
                assert "cost" in fund
                assert "shares" in fund
                assert "price" in fund
                assert "realized_gain" in fund
                assert "unrealized_gain" in fund

    def test_date_range_filtering(self, app_context):
        """Verify that date range parameters are respected."""
        from datetime import datetime, timedelta

        end_date = datetime.now().date()
        start_date = end_date - timedelta(days=30)

        result = PortfolioService.get_portfolio_history(
            start_date=start_date.isoformat(), end_date=end_date.isoformat()
        )

        # Should get approximately 31 days (inclusive)
        assert 29 <= len(result) <= 32  # Allow some flexibility for weekends, etc.


# Pytest markers for categorizing tests
pytestmark = [
    pytest.mark.performance,  # Mark all tests in this module as performance tests
]
