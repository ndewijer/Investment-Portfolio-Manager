# Testing Framework Guide

**Version**: 1.3.2+
**Last Updated**: 2025-01-21
**Status**: Foundation established, expanding coverage

This guide documents the pytest testing framework introduced in version 1.3.2 and provides guidance for adding new tests.

---

## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
- [Test Structure](#test-structure)
- [Available Fixtures](#available-fixtures)
- [Running Tests](#running-tests)
- [Writing New Tests](#writing-new-tests)
- [Best Practices](#best-practices)
- [Examples](#examples)

---

## Overview

The Investment Portfolio Manager uses **pytest** for its testing framework, chosen for its:

- **Simplicity**: Easy to write and understand tests
- **Powerful fixtures**: Reusable test components
- **Great ecosystem**: Extensive plugin support
- **Clear output**: Readable test results

### Current Coverage

As of version 1.3.2:

- **Performance Tests**: `test_portfolio_performance.py` (12 tests)
  - Phase 1: Batch processing tests (8 tests)
  - Phase 2: Eager loading tests (4 tests)
  - Query count validation
  - Execution time benchmarks
  - Data structure correctness
  - Edge case handling

**Test Coverage**: 75% for `portfolio_service.py`, 34% for `transaction_service.py`, 28% overall

### Goals

- **Prevent regressions**: Catch performance/functionality regressions early
- **Document behavior**: Tests serve as executable documentation
- **Enable refactoring**: Confidence to improve code
- **Future expansion**: Foundation for comprehensive coverage

---

## Getting Started

### Prerequisites

- Python 3.13+
- Virtual environment activated
- pytest and dependencies installed

### Installation

```bash
cd backend
source .venv/bin/activate

# Install test dependencies
pip install -r requirements.txt

# Verify installation
pytest --version
```

### Dependencies

```
pytest==8.4.2           # Test framework
pytest-flask==1.3.0     # Flask integration
pytest-cov==7.0.0       # Coverage reporting
```

---

## Test Structure

### Directory Layout

```
backend/
├── app/                    # Application code
│   ├── models.py
│   ├── routes/
│   └── services/
├── tests/                  # Test suite
│   ├── __init__.py        # Package marker
│   ├── conftest.py        # Shared fixtures
│   └── test_portfolio_performance.py  # Performance tests
├── pytest.ini             # Pytest configuration
└── requirements.txt       # Dependencies
```

### Test Organization

Tests are organized by:

1. **Module**: `test_<module_name>.py`
2. **Class**: `Test<FeatureName>` (optional grouping)
3. **Function**: `test_<specific_behavior>`

**Example**:
```python
# tests/test_portfolio_service.py
class TestPortfolioHistoryPerformance:
    def test_get_portfolio_history_query_count(self):
        pass

    def test_get_portfolio_history_execution_time(self):
        pass

class TestPortfolioHistoryCorrectness:
    def test_portfolio_history_returns_data(self):
        pass
```

---

## Available Fixtures

Fixtures are reusable components defined in `tests/conftest.py`.

### `app` (session scope)

**Purpose**: Flask application instance for testing

**Scope**: Session - created once per test run

**Usage**:
```python
def test_something(app):
    assert app.config["TESTING"] == True
```

**Configuration**:
- Uses existing database (can be configured for test DB)
- TESTING mode enabled
- Same configuration as production app

### `client` (session scope)

**Purpose**: Test client for HTTP requests

**Scope**: Session - created once per test run

**Usage**:
```python
def test_api_endpoint(client):
    response = client.get('/api/portfolios')
    assert response.status_code == 200
```

**Use for**:
- Integration tests
- API endpoint testing
- End-to-end request/response validation

### `app_context` (function scope)

**Purpose**: Application context for database operations

**Scope**: Function - created for each test

**Usage**:
```python
def test_database_query(app_context):
    from app.models import Portfolio
    portfolios = Portfolio.query.all()
    assert len(portfolios) > 0
```

**Required for**:
- Database queries
- Service layer calls
- Any code requiring Flask context

### `query_counter` (function scope)

**Purpose**: Counts SQL queries executed during test

**Scope**: Function - created for each test

**Attributes**:
- `count` (int): Current query count
- `queries` (list): List of SQL statements
- `reset()`: Reset counter to zero

**Usage**:
```python
def test_query_efficiency(app_context, query_counter):
    query_counter.reset()

    # Execute code
    result = PortfolioService.get_portfolio_history()

    # Verify query count
    print(f"Queries: {query_counter.count}")
    assert query_counter.count < 100
```

**Use for**:
- Performance tests
- Query optimization validation
- Regression prevention

### `timer` (function scope)

**Purpose**: Measures execution time

**Scope**: Function - created for each test

**Methods**:
- `start()`: Start timer
- `stop()`: Stop timer and return elapsed seconds
- `elapsed` (property): Current elapsed time

**Usage**:
```python
def test_performance(app_context, timer):
    timer.start()

    # Execute code
    result = PortfolioService.get_portfolio_history()

    elapsed = timer.stop()
    print(f"Time: {elapsed:.3f}s")
    assert elapsed < 1.0
```

**Use for**:
- Performance benchmarks
- Execution time validation
- Speed regression tests

---

## Running Tests

### Basic Commands

```bash
# Run all tests
pytest

# Run with verbose output
pytest -v

# Run with output capture disabled (see print statements)
pytest -s

# Run specific test file
pytest tests/test_portfolio_performance.py

# Run specific test function
pytest tests/test_portfolio_performance.py::test_get_portfolio_history_query_count

# Run specific test class
pytest tests/test_portfolio_performance.py::TestPortfolioHistoryPerformance
```

### Coverage Reports

```bash
# Run with coverage (automatically enabled in pytest.ini)
pytest

# Generate HTML coverage report
pytest --cov=app --cov-report=html
open htmlcov/index.html

# Show missing lines
pytest --cov=app --cov-report=term-missing
```

### Markers

```bash
# Run only performance tests (future)
pytest -m performance

# Run only unit tests (future)
pytest -m unit

# Run only integration tests (future)
pytest -m integration
```

---

## Writing New Tests

### Test Template

```python
"""
Module for testing [feature name].

Brief description of what this module tests.
"""

import pytest
from app.services.some_service import SomeService


class TestFeatureName:
    """Test suite for [feature]."""

    def test_basic_functionality(self, app_context):
        """Test that basic feature works correctly."""
        # Arrange
        expected_result = "something"

        # Act
        result = SomeService.some_method()

        # Assert
        assert result == expected_result

    def test_edge_case(self, app_context):
        """Test edge case handling."""
        # Test edge case
        pass
```

### Performance Test Template

```python
class TestPerformance:
    """Performance benchmarks for [feature]."""

    def test_query_count(self, app_context, query_counter):
        """Test that query count is within target."""
        query_counter.reset()

        # Execute feature
        result = SomeService.expensive_operation()

        # Verify
        print(f"\n✓ Query count: {query_counter.count}")
        assert query_counter.count < 50, \
            f"Too many queries: {query_counter.count}"

    def test_execution_time(self, app_context, timer):
        """Test that execution completes quickly."""
        timer.start()

        # Execute feature
        result = SomeService.expensive_operation()

        elapsed = timer.stop()
        print(f"\n✓ Execution time: {elapsed:.3f}s")
        assert elapsed < 1.0, \
            f"Too slow: {elapsed:.3f}s"
```

### Integration Test Template

```python
class TestAPIEndpoint:
    """Integration tests for [endpoint]."""

    def test_endpoint_success(self, client):
        """Test successful API response."""
        response = client.get('/api/endpoint')

        assert response.status_code == 200
        data = response.get_json()
        assert 'expected_field' in data

    def test_endpoint_validation(self, client):
        """Test input validation."""
        response = client.post('/api/endpoint', json={
            'invalid': 'data'
        })

        assert response.status_code == 400
```

---

## Best Practices

### 1. Test Naming

**Good**:
```python
def test_get_portfolio_history_returns_list_of_daily_values():
    pass

def test_portfolio_fund_history_filters_by_date_range():
    pass
```

**Bad**:
```python
def test_history():  # Too vague
    pass

def test_func1():  # Meaningless name
    pass
```

### 2. Arrange-Act-Assert Pattern

```python
def test_something(app_context):
    # Arrange: Set up test data
    portfolio_id = "some-id"
    start_date = "2024-01-01"

    # Act: Execute the code under test
    result = PortfolioService.get_portfolio_history(
        start_date=start_date
    )

    # Assert: Verify the results
    assert isinstance(result, list)
    assert len(result) > 0
```

### 3. One Assertion Per Test (When Possible)

**Good**:
```python
def test_returns_list(app_context):
    result = PortfolioService.get_portfolio_history()
    assert isinstance(result, list)

def test_list_not_empty(app_context):
    result = PortfolioService.get_portfolio_history()
    assert len(result) > 0
```

**Acceptable** (related assertions):
```python
def test_response_structure(app_context):
    result = PortfolioService.get_portfolio_history()
    assert isinstance(result, list)
    assert len(result) > 0
    assert 'date' in result[0]
```

### 4. Use Descriptive Assertion Messages

```python
# Good
assert query_counter.count < 100, \
    f"Too many queries: {query_counter.count} (target: < 100)"

assert elapsed < 1.0, \
    f"Too slow: {elapsed:.3f}s (target: < 1.0s)"

# Bad
assert query_counter.count < 100
assert elapsed < 1.0
```

### 5. Test Edge Cases

```python
def test_empty_portfolio(app_context):
    """Test behavior with no transactions."""
    # Test empty case

def test_missing_price_data(app_context):
    """Test behavior when price data unavailable."""
    # Test missing data

def test_date_range_boundaries(app_context):
    """Test start and end date edge cases."""
    # Test boundaries
```

### 6. Use Fixtures for Shared Setup

```python
# In conftest.py
@pytest.fixture
def sample_portfolio(app_context):
    """Create a sample portfolio for testing."""
    portfolio = Portfolio(name="Test Portfolio")
    db.session.add(portfolio)
    db.session.commit()
    return portfolio

# In test file
def test_with_sample(sample_portfolio):
    assert sample_portfolio.name == "Test Portfolio"
```

---

## Examples

### Example 1: Service Layer Test

```python
def test_calculate_portfolio_fund_values(app_context):
    """Test fund value calculation."""
    from app.models import PortfolioFund, Portfolio

    # Get a real portfolio fund
    portfolio = Portfolio.query.filter_by(is_archived=False).first()
    if not portfolio:
        pytest.skip("No portfolio found")

    # Calculate values
    result = PortfolioService.calculate_portfolio_fund_values(
        portfolio.funds
    )

    # Verify structure
    assert isinstance(result, list)
    for fund in result:
        assert 'fund_id' in fund
        assert 'total_shares' in fund
        assert 'current_value' in fund
        assert isinstance(fund['current_value'], (int, float))
```

### Example 2: Performance Test

```python
def test_portfolio_summary_performance(
    app_context, query_counter, timer
):
    """Test portfolio summary performance."""
    query_counter.reset()
    timer.start()

    # Execute
    result = PortfolioService.get_portfolio_summary()

    elapsed = timer.stop()

    # Verify results
    assert isinstance(result, list)

    # Verify performance
    print(f"\n✓ Queries: {query_counter.count}")
    print(f"✓ Time: {elapsed:.3f}s")

    assert query_counter.count < 100
    assert elapsed < 0.5
```

### Example 3: API Integration Test

```python
def test_portfolio_history_endpoint(client):
    """Test /api/portfolio-history endpoint."""
    # Make request
    response = client.get('/api/portfolio-history?days=30')

    # Verify response
    assert response.status_code == 200

    data = response.get_json()
    assert isinstance(data, list)
    assert len(data) <= 31  # 30 days + today

    # Verify structure
    if len(data) > 0:
        assert 'date' in data[0]
        assert 'portfolios' in data[0]
```

---

## Future Expansion

### Planned Test Types

1. **Unit Tests**: Test individual functions in isolation
2. **Integration Tests**: Test API endpoints end-to-end
3. **Fixture Data**: Create reusable test portfolios/transactions
4. **Mock Objects**: Mock external services (yfinance, IBKR)
5. **Test Database**: Separate database for testing

### Areas to Cover

- **Service Layer**: All service methods
- **Routes**: All API endpoints
- **Models**: Database operations
- **Utilities**: Helper functions
- **IBKR Integration**: Import and processing logic
- **Price Updates**: yfinance integration
- **Error Handling**: Exception cases

### Test Markers (Future)

```python
@pytest.mark.unit
def test_calculation():
    pass

@pytest.mark.integration
def test_api_endpoint():
    pass

@pytest.mark.performance
def test_query_count():
    pass

@pytest.mark.slow
def test_full_import():
    pass
```

---

## Troubleshooting

### Issue: Import Errors

**Problem**: `ImportError: cannot import name 'create_app'`

**Solution**:
- Ensure `PYTHONPATH` includes backend directory
- Run pytest from backend directory
- Check conftest.py imports

### Issue: Database Not Found

**Problem**: `No such table` errors

**Solution**:
```bash
# Ensure database exists
cd backend
source .venv/bin/activate
python run.py  # Creates tables if needed
pytest
```

### Issue: Fixture Not Found

**Problem**: `fixture 'query_counter' not found`

**Solution**:
- Ensure conftest.py exists in tests directory
- Check fixture name spelling
- Verify conftest.py is not excluded in pytest.ini

### Issue: Tests Hang

**Problem**: Tests don't complete

**Solution**:
- Check for infinite loops
- Verify database queries don't timeout
- Use pytest timeout plugin:
  ```bash
  pip install pytest-timeout
  pytest --timeout=30
  ```

---

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD) or alongside implementation
2. **Test happy path and edge cases**
3. **Add performance tests** for expensive operations
4. **Update this documentation** with new patterns
5. **Maintain test coverage** above 70%

---

## Resources

- [Pytest Documentation](https://docs.pytest.org/)
- [Flask Testing Guide](https://flask.palletsprojects.com/en/latest/testing/)
- [pytest-flask Plugin](https://pytest-flask.readthedocs.io/)
- [Coverage.py Documentation](https://coverage.readthedocs.io/)

---

**Version**: 1.3.2+
**Last Updated**: 2025-01-21
**Maintainer**: @ndewijer
