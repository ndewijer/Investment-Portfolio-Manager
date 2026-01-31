"""
Pytest configuration and fixtures for the Investment Portfolio Manager test suite.

This module provides reusable fixtures for testing, including:
- Flask app configuration with test database
- Database setup and teardown
- Query counting for performance tests
- Time mocking for date-sensitive tests
- Test data fixtures
"""

import pytest
from app.models import db
from run import create_app
from sqlalchemy import event
from sqlalchemy.engine import Engine

from tests.test_config import TEST_CONFIG, cleanup_test_database


@pytest.fixture(scope="session")
def app():
    """
    Create and configure a Flask app instance for testing.

    This fixture creates a Flask app with a separate test database.
    The database is created once per test session and cleaned up at the end.

    Scope: session - created once per test session.
    """
    # Pass TEST_CONFIG to create_app() so it's applied BEFORE db.init_app()
    app = create_app(config=TEST_CONFIG)

    # Clear any buffered startup logs to prevent them from being written to production DB
    # These logs are buffered during create_app() and would otherwise be flushed
    # when the first database operation occurs
    import run

    run._startup_logs.clear()

    # Create all tables in the test database
    with app.app_context():
        db.create_all()

    yield app

    # Cleanup: Drop all tables and remove test database file
    with app.app_context():
        db.drop_all()
    cleanup_test_database()


@pytest.fixture(scope="session")
def client(app):
    """
    Create a test client for making HTTP requests.

    Scope: session - created once per test session.
    """
    return app.test_client()


@pytest.fixture(scope="function")
def app_context(app):
    """
    Create an application context for each test function.

    Scope: function - created for each test that needs it.
    """
    with app.app_context():
        yield


@pytest.fixture(scope="function")
def query_counter(app_context):
    """
    Fixture that counts SQL queries executed during a test.

    Returns a QueryCounter object with:
    - count: Current number of queries
    - reset(): Reset the counter
    - queries: List of all SQL statements executed

    Usage:
        def test_something(query_counter):
            query_counter.reset()
            # ... do something ...
            assert query_counter.count < 100

    Scope: function - created for each test.
    """

    class QueryCounter:
        def __init__(self):
            self.count = 0
            self.queries = []

        def reset(self):
            self.count = 0
            self.queries = []

    counter = QueryCounter()

    @event.listens_for(Engine, "before_cursor_execute")
    def receive_before_cursor_execute(conn, cursor, statement, parameters, context, executemany):
        counter.count += 1
        counter.queries.append(statement)

    yield counter

    # Cleanup: remove the event listener after the test
    event.remove(Engine, "before_cursor_execute", receive_before_cursor_execute)


@pytest.fixture(scope="function")
def timer():
    """
    Fixture that provides a simple timer for performance testing.

    Returns a Timer object with:
    - start(): Start the timer
    - stop(): Stop the timer and return elapsed time in seconds
    - elapsed: Get elapsed time in seconds (without stopping)

    Usage:
        def test_performance(timer):
            timer.start()
            # ... do something ...
            elapsed = timer.stop()
            assert elapsed < 1.0  # Should complete in under 1 second

    Scope: function - created for each test.
    """
    import time

    class Timer:
        def __init__(self):
            self.start_time = None
            self.end_time = None

        def start(self):
            self.start_time = time.time()
            self.end_time = None

        def stop(self):
            if self.start_time is None:
                raise RuntimeError("Timer not started")
            self.end_time = time.time()
            return self.end_time - self.start_time

        @property
        def elapsed(self):
            if self.start_time is None:
                raise RuntimeError("Timer not started")
            if self.end_time is None:
                return time.time() - self.start_time
            return self.end_time - self.start_time

    return Timer()


@pytest.fixture(scope="function")
def db_session(app_context):
    """
    Provide a database session for tests with proper cleanup.

    This fixture provides access to db.session within the app context and
    ensures each test starts with a clean database state by manually
    cleaning up all test data before each test.

    Scope: function - created for each test.
    """
    # Clean up before each test
    try:
        # Roll back any active transaction
        if db.session.is_active:
            db.session.rollback()

        # Import all models for cleanup
        from app.models import (
            Dividend,
            ExchangeRate,
            Fund,
            FundHistoryMaterialized,
            FundPrice,
            IBKRConfig,
            IBKRImportCache,
            IBKRTransaction,
            IBKRTransactionAllocation,
            Log,
            Portfolio,
            PortfolioFund,
            RealizedGainLoss,
            SystemSetting,
            Transaction,
        )

        # Delete in dependency order without disabling foreign keys
        # Since foreign keys are properly enabled via run.py event listener,
        # we just need to delete in the correct order
        cleanup_tables = [
            IBKRTransactionAllocation,
            RealizedGainLoss,
            Transaction,
            FundPrice,
            Dividend,
            IBKRTransaction,
            IBKRImportCache,
            FundHistoryMaterialized,
            PortfolioFund,
            Fund,
            Portfolio,
            IBKRConfig,
            ExchangeRate,
            SystemSetting,
            Log,
        ]

        for table in cleanup_tables:
            db.session.query(table).delete()

        db.session.commit()

    except Exception as e:
        db.session.rollback()
        print(f"Warning: Pre-test database cleanup failed: {e}")
        # Try to recover by recreating all tables
        try:
            db.drop_all()
            db.create_all()
            db.session.commit()
        except Exception as recovery_error:
            print(f"Error: Failed to recover database: {recovery_error}")
            raise

    yield db.session

    # Optional: Light cleanup after test (just rollback any uncommitted changes)
    try:
        if db.session.is_active:
            db.session.rollback()
    except Exception:
        pass  # Ignore cleanup errors in teardown


@pytest.fixture(scope="function")
def mock_yfinance():
    """
    Mock yfinance API calls for testing.

    This fixture mocks yfinance so tests don't make real API calls.
    Can be customized per test to return specific data.

    Usage:
        def test_something(mock_yfinance):
            # yfinance calls will be mocked
            pass

    Scope: function - created for each test.
    """
    # This is a placeholder - will be implemented when writing price update tests
    pass


@pytest.fixture(scope="function")
def mock_ibkr_api(responses):
    """
    Mock IBKR API calls for testing.

    This fixture uses the responses library to mock HTTP calls to IBKR.
    Tests can add specific responses as needed.

    Usage:
        def test_something(mock_ibkr_api):
            # Add a mock response
            responses.add(
                responses.POST,
                "https://gdcdyn.interactivebrokers.com/...",
                json={"status": "success"},
                status=200
            )

    Scope: function - created for each test.
    """
    # The responses fixture is provided by the responses library
    # This is just documentation of how to use it
    pass


@pytest.fixture(scope="function")
def freeze_time():
    """
    Fixture that provides time freezing capability using freezegun.

    This allows tests to control the current date/time for testing
    date-sensitive logic.

    Usage:
        from freezegun import freeze_time as freeze

        @freeze("2024-01-15")
        def test_something(freeze_time):
            # Time is frozen at 2024-01-15
            pass

    Scope: function - created for each test.
    """
    # The actual freezegun decorator is used directly in tests
    # This is just documentation
    pass


# Sample data fixtures - these will be enhanced with factory_boy in factories.py
# For now, these are placeholders that will be implemented in Phase 1


@pytest.fixture(scope="function")
def sample_portfolio(app_context):
    """
    Create a sample portfolio for testing.

    Returns a Portfolio object that can be used in tests.
    Will be enhanced with factory_boy.

    Scope: function - created for each test.
    """
    from app.models import Portfolio, db

    portfolio = Portfolio(name="Test Portfolio", description="Test portfolio for tests")
    db.session.add(portfolio)
    db.session.commit()
    return portfolio


@pytest.fixture(scope="function")
def cash_dividend_fund(app_context):
    """
    Create a fund configured for CASH dividends.

    Returns a Fund object with dividend_type = CASH.

    Scope: function - created for each test.
    """
    import uuid

    from app.models import DividendType, Fund, InvestmentType, db

    # Use UUID to ensure unique ISIN for each test
    unique_isin = f"US{uuid.uuid4().hex[:10].upper()}"
    fund = Fund(
        name="Test Cash Dividend Fund",
        isin=unique_isin,
        currency="USD",
        exchange="NASDAQ",
        investment_type=InvestmentType.FUND,
        dividend_type=DividendType.CASH,
    )
    db.session.add(fund)
    db.session.commit()
    return fund


@pytest.fixture(scope="function")
def stock_dividend_fund(app_context):
    """
    Create a fund configured for STOCK dividends.

    Returns a Fund object with dividend_type = STOCK.

    Scope: function - created for each test.
    """
    import uuid

    from app.models import DividendType, Fund, InvestmentType, db

    # Use UUID to ensure unique ISIN for each test
    unique_isin = f"US{uuid.uuid4().hex[:10].upper()}"
    fund = Fund(
        name="Test Stock Dividend Fund",
        isin=unique_isin,
        currency="USD",
        exchange="NASDAQ",
        investment_type=InvestmentType.FUND,
        dividend_type=DividendType.STOCK,
    )
    db.session.add(fund)
    db.session.commit()
    return fund
