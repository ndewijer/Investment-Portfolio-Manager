"""
Pytest configuration and fixtures for the Investment Portfolio Manager test suite.

This module provides reusable fixtures for testing, including:
- Flask app configuration
- Database setup and teardown
- Query counting for performance tests
- Test data fixtures (can be added later)
"""

import pytest
from run import create_app
from sqlalchemy import event
from sqlalchemy.engine import Engine


@pytest.fixture(scope="session")
def app():
    """
    Create and configure a Flask app instance for testing.

    Scope: session - created once per test session.
    """
    app = create_app()
    app.config.update(
        {
            "TESTING": True,
            # Use the existing database for now - in future, create a test database
            # "SQLALCHEMY_DATABASE_URI": "sqlite:///test_portfolio.db"
        }
    )

    yield app


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


# Future fixtures to add:
# - @pytest.fixture(scope="function") def test_db(): # Clean test database
# - @pytest.fixture(scope="function") def sample_portfolio(): # Sample test data
# - @pytest.fixture(scope="function") def sample_transactions(): # Sample transactions
# - etc.
