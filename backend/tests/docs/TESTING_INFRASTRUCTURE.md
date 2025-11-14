# Testing Infrastructure Guide

This document explains the testing setup used across all backend tests in the Investment Portfolio Manager.

---

## Table of Contents

1. [Overview](#overview)
2. [Test Database](#test-database)
3. [Fixtures](#fixtures)
4. [Factory Pattern](#factory-pattern)
5. [Code Coverage](#code-coverage)
6. [Test Organization](#test-organization)
7. [Running Tests](#running-tests)

---

## Overview

### Testing Philosophy

Our testing approach:
- **Integration tests** over pure unit tests
- **Real database** interactions (isolated test DB)
- **Comprehensive coverage** of business logic (90%+ target)
- **Explicit test data** (readable and maintainable)
- **Document thoroughly** (explain the "why")

### Test Types

**Service Layer Tests** (current focus):
- Test services with real database
- Test business logic thoroughly
- Test validation and error handling
- Target: 90%+ code coverage per service

**Performance Tests** (from v1.3.2):
- Test query optimization
- Test execution time
- Prevent performance regressions

**Route Integration Tests** (future - Phase 4):
- Test HTTP endpoints
- Test request/response handling
- Test authentication/authorization

---

## Test Database

### Isolation Strategy

**Production Database**: `backend/data/db/portfolio_manager.db`
**Test Database**: `/tmp/test_portfolio_manager.db`

**Why separate**:
- Zero risk of contaminating production data
- Tests can create/modify/delete freely
- Parallel test execution possible
- Fast cleanup between tests

### Configuration

**File**: `tests/test_config.py`

```python
import tempfile

TEST_CONFIG = {
    "SQLALCHEMY_DATABASE_URI": "sqlite:////tmp/test_portfolio_manager.db",
    "SQLALCHEMY_TRACK_MODIFICATIONS": False,
    "TESTING": True,
}

def cleanup_test_database():
    """Remove test database file after test session."""
    import os
    db_path = "/tmp/test_portfolio_manager.db"
    if os.path.exists(db_path):
        os.remove(db_path)
```

### Lifecycle

**Session start**:
1. Test database created at `/tmp/test_portfolio_manager.db`
2. All tables created via `db.create_all()`
3. Database is empty and ready

**During tests**:
- Each test runs in isolation
- Database state doesn't leak between tests
- All test data is temporary

**Session end**:
1. All tables dropped via `db.drop_all()`
2. Test database file deleted
3. Clean slate for next run

### Verification

**How to verify test isolation**:

```bash
# IMPORTANT: Stop all Flask dev servers first!

# 1. Count production DB logs BEFORE tests
BEFORE=$(sqlite3 backend/data/db/portfolio_manager.db "SELECT COUNT(*) FROM log;")
echo "Production DB logs before: $BEFORE"

# 2. Run tests
cd backend
source .venv/bin/activate
pytest tests/ -v

# 3. Count production DB logs AFTER tests
AFTER=$(sqlite3 data/db/portfolio_manager.db "SELECT COUNT(*) FROM log;")
echo "Production DB logs after: $AFTER"

# 4. Verify no change
if [ "$BEFORE" -eq "$AFTER" ]; then
    echo "✅ Test isolation verified"
else
    echo "❌ Test isolation failed: $(($AFTER - $BEFORE)) logs written"
fi
```

**Expected**: `$BEFORE` should equal `$AFTER` (no logs written to production)

---

## Fixtures

Fixtures are reusable test components defined in `conftest.py`.

### Core Fixtures

#### `app` (session-scoped)

**What it does**: Creates Flask application with test configuration

**Scope**: Once per test session

```python
@pytest.fixture(scope="session")
def app():
    """Create and configure Flask app for testing."""
    app = create_app(config=TEST_CONFIG)

    # Clear startup logs (prevent production DB writes)
    import run
    run._startup_logs.clear()

    # Create all tables
    with app.app_context():
        db.create_all()

    yield app

    # Cleanup
    with app.app_context():
        db.drop_all()
    cleanup_test_database()
```

**Usage**: Automatic (pytest handles it)

---

#### `client` (session-scoped)

**What it does**: Test client for HTTP requests

**Scope**: Once per test session

```python
@pytest.fixture(scope="session")
def client(app):
    """Create test client for HTTP requests."""
    return app.test_client()
```

**Usage**:
```python
def test_endpoint(client):
    response = client.get('/api/portfolios')
    assert response.status_code == 200
```

---

#### `app_context` (function-scoped)

**What it does**: Provides Flask application context

**Scope**: Once per test function

```python
@pytest.fixture(scope="function")
def app_context(app):
    """Create application context for each test."""
    with app.app_context():
        yield
```

**Why needed**: Database operations require app context

**Usage**: Add to test signature
```python
def test_something(self, app_context, db_session):
    # app_context active, database operations work
    fund = FundFactory()
    db_session.commit()
```

---

#### `db_session` (function-scoped)

**What it does**: Provides database session

**Scope**: Once per test function

```python
@pytest.fixture(scope="function")
def db_session(app_context):
    """Provide database session for tests."""
    yield db.session
```

**Usage**: All database operations
```python
def test_create_fund(self, app_context, db_session):
    # Create test data
    fund = FundFactory()
    db_session.commit()  # Save to test database

    # Query test data
    found = Fund.query.get(fund.id)
    assert found is not None
```

---

### Performance Testing Fixtures

#### `query_counter` (function-scoped)

**What it does**: Counts SQL queries executed during test

**Scope**: Once per test function

**Source**: `conftest.py:75`

```python
@pytest.fixture(scope="function")
def query_counter(app_context):
    """Count SQL queries executed during test."""
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

    # Cleanup: remove event listener
    event.remove(Engine, "before_cursor_execute", receive_before_cursor_execute)
```

**Usage**:
```python
def test_performance(self, query_counter):
    query_counter.reset()

    # Run code that makes database queries
    PortfolioService.get_portfolio_history(...)

    # Check query count
    print(f"Queries executed: {query_counter.count}")
    assert query_counter.count < 100  # Performance target
```

**Attributes**:
- `counter.count` - Total queries executed
- `counter.queries` - List of SQL statements
- `counter.reset()` - Reset counter to zero

---

#### `timer` (function-scoped)

**What it does**: Measures execution time

**Scope**: Once per test function

**Source**: `conftest.py:116`

```python
@pytest.fixture(scope="function")
def timer():
    """Simple timer for performance testing."""
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
```

**Usage**:
```python
def test_execution_time(self, timer):
    timer.start()

    # Run code to measure
    PortfolioService.get_portfolio_history(...)

    elapsed = timer.stop()

    print(f"Execution time: {elapsed:.3f}s")
    assert elapsed < 1.0  # Should complete in under 1 second
```

---

## Factory Pattern

Factories generate test data with realistic defaults using **factory_boy**.

### What Are Factories?

Factories are classes that create model instances with sensible defaults:

**Without factories** (tedious):
```python
fund = Fund(
    id=str(uuid.uuid4()),
    name="Test Fund",
    symbol="TEST",
    dividend_type=DividendType.CASH,
    currency="USD",
    fund_type="ETF",
    expense_ratio=0.05,
    # ... 10 more fields
)
db.session.add(fund)
db.session.commit()
```

**With factories** (easier):
```python
fund = FundFactory()  # All fields auto-generated with realistic defaults
db.session.commit()
```

**Override specific fields**:
```python
fund = FundFactory(
    name="My Custom Fund",
    dividend_type=DividendType.STOCK
)  # Other fields still auto-generated
db.session.commit()
```

### Available Factories

**File**: `tests/factories.py`

**Base Factory**:
```python
class BaseFactory(factory.alchemy.SQLAlchemyModelFactory):
    """Base factory for all model factories."""
    class Meta:
        sqlalchemy_session = db.session
        sqlalchemy_session_persistence = "commit"
```

**Model Factories**:
- `PortfolioFactory` - Creates Portfolio with random name
- `FundFactory` - Creates Fund with random name, symbol, etc.
- `PortfolioFundFactory` - Links Portfolio to Fund
- `TransactionFactory` - Creates Transaction
- `DividendFactory` - Creates Dividend
- `CashDividendFactory` - Dividend with CASH fund
- `StockDividendFactory` - Dividend with STOCK fund
- `IBKRTransactionFactory` - IBKR transaction data

**Example - PortfolioFactory**:
```python
class PortfolioFactory(BaseFactory):
    class Meta:
        model = Portfolio

    id = factory.LazyFunction(lambda: str(uuid.uuid4()))
    name = factory.Faker("company")
    description = factory.Faker("catch_phrase")
    is_archived = False
```

### SubFactories: The Challenge

**What are SubFactories**:
SubFactories automatically create related objects:

```python
class DividendFactory(BaseFactory):
    fund = SubFactory(FundFactory)  # Auto-creates Fund
    portfolio_fund = SubFactory(PortfolioFundFactory)  # Auto-creates PortfolioFund

    dividend_per_share = 0.50
    # ...
```

**When they're helpful**:
```python
# Want all new objects
dividend = DividendFactory()
# Creates: Fund, Portfolio, PortfolioFund, Dividend
```

**When they cause problems**:
```python
# Already have fund, want dividend for it
fund = FundFactory()
db.session.commit()

dividend = DividendFactory(fund_id=fund.id)  # ❌ SubFactory STILL creates new fund!
```

**Why**: SubFactory runs BEFORE field values are set, ignoring your `fund_id` parameter.

### When to Use Direct Creation

**Use factories when**:
- Creating "root" objects (Portfolio, Fund)
- Want realistic random data
- Don't care about specific values

**Use direct creation when**:
- Linking to existing objects
- Need precise control over values
- Avoiding SubFactory conflicts

**Example - Direct Creation**:
```python
from app.models import Dividend
import uuid

# Create related objects with factories
fund = FundFactory(dividend_type=DividendType.STOCK)
portfolio_fund = PortfolioFundFactory(fund=fund)
db_session.commit()

# Create dividend DIRECTLY (not via factory)
dividend = Dividend(
    id=str(uuid.uuid4()),
    fund_id=fund.id,  # ✅ Links to existing fund
    portfolio_fund_id=portfolio_fund.id,  # ✅ Links to existing PF
    record_date=date(2024, 3, 1),
    ex_dividend_date=date(2024, 2, 28),
    dividend_per_share=0.50,
    shares_owned=100,
    total_amount=50.0,
    reinvestment_status=ReinvestmentStatus.COMPLETED
)
db_session.add(dividend)
db_session.commit()
```

**See also**: Individual test documentation for service-specific patterns

---

## Code Coverage

### What Is Code Coverage?

Code coverage measures which lines of code are executed during tests.

### How It's Calculated

**Formula**:
```
Coverage % = (Lines Executed / Total Lines) × 100
```

**Example**:
```
91% = (108 / 119) × 100
```

This means:
- Total: 119 executable statements
- Executed: 108 statements during tests
- Missed: 11 statements never executed
- Coverage: 91%

### What Counts as a Statement?

**Counted** (executable):
```python
x = 5                          # 1 statement
if x > 0:                      # 1 statement (condition)
    y = 10                     # 1 statement
    z = 20                     # 1 statement
return result                  # 1 statement
```

**Not counted** (not executable):
```python
# Comments                     # Not counted
"""Docstrings"""              # Not counted
class ClassName:              # Not counted
    pass                       # Not counted
import module                  # Not counted (imports)
```

### Running Coverage

**Command**:
```bash
cd backend
source .venv/bin/activate

# Run tests with coverage
pytest tests/test_dividend_service.py \
  --cov=app.services.dividend_service \
  --cov-report=term-missing
```

**Output**:
```
Name                                 Stmts   Miss  Cover   Missing
------------------------------------------------------------------
app/services/dividend_service.py       119     11    91%   160-162, 278, 294, 314, 356, 364-367
```

**Reading the output**:
- **Stmts**: Total executable statements
- **Miss**: Statements not executed
- **Cover**: Coverage percentage
- **Missing**: Line numbers not covered

### Coverage Targets

**Service layer**: 90%+ coverage
**Route layer**: 80%+ coverage (future Phase 4)
**Overall backend**: 80%+ coverage

### Interpreting Coverage

**Good coverage** (90%+):
- ✅ All "happy paths" covered
- ✅ All business logic covered
- ✅ All validation covered
- ✅ Edge cases tested
- ⚠️ Some exception handlers uncovered (acceptable)

**Areas commonly uncovered**:
- Exception handlers (hard to trigger)
- Defensive error paths
- Edge cases requiring complex setup

**When 100% isn't necessary**:
- Exception handlers requiring database mocking
- Defensive code (should never execute)
- Deprecated code paths

### Coverage Reports

**Terminal** (default):
```bash
pytest tests/ --cov=app
```

**HTML Report** (detailed):
```bash
pytest tests/ --cov=app --cov-report=html
open htmlcov/index.html  # View in browser
```

**XML Report** (CI/CD):
```bash
pytest tests/ --cov=app --cov-report=xml
```

---

## Test Organization

### Directory Structure

```
backend/tests/
├── __init__.py
├── conftest.py              # Fixtures and configuration
├── factories.py             # Test data factories
├── test_config.py           # Test database configuration
│
├── test_dividend_service.py          # DividendService tests
├── test_portfolio_performance.py     # Performance tests (v1.3.2)
│
└── docs/
    ├── README.md                      # Documentation index
    ├── TESTING_INFRASTRUCTURE.md      # This file
    ├── BUG_FIXES_1.3.3.md            # Bugs found during testing
    ├── DIVIDEND_SERVICE_TESTS.md      # DividendService test guide
    └── PORTFOLIO_PERFORMANCE_TESTS.md # Performance test guide
```

### Naming Conventions

**Test files**: `test_<module>_<type>.py`
- `test_dividend_service.py` - Service unit tests
- `test_portfolio_performance.py` - Performance tests
- `test_dividend_routes_integration.py` - Route integration tests (future)

**Test classes**: `Test<MethodOrFeature>`
- `TestCalculateSharesOwned` - Tests for specific method
- `TestCreateDividend` - Tests for creation
- `TestEdgeCases` - Edge case tests

**Test methods**: `test_<what>_<condition>`
- `test_calculate_shares_buy_only` - Specific scenario
- `test_create_dividend_invalid_data` - Error condition
- `test_performance_query_count` - Performance check

### Test Structure (Arrange-Act-Assert)

```python
def test_example(self, app_context, db_session):
    """Docstring explaining what this test validates."""

    # ARRANGE: Set up test data
    portfolio = PortfolioFactory()
    fund = FundFactory()
    db_session.commit()

    # ACT: Execute the operation being tested
    result = SomeService.some_method(fund.id)

    # ASSERT: Verify the results
    assert result.field == expected_value
    assert result.status == "success"
```

---

## Running Tests

### Run All Tests

```bash
cd backend
source .venv/bin/activate
pytest tests/
```

### Run Specific Test File

```bash
pytest tests/test_dividend_service.py
```

### Run Specific Test Class

```bash
pytest tests/test_dividend_service.py::TestCalculateSharesOwned
```

### Run Specific Test

```bash
pytest tests/test_dividend_service.py::TestCalculateSharesOwned::test_calculate_shares_buy_only
```

### Verbose Output

```bash
pytest tests/ -v  # Show test names
pytest tests/ -vv  # Very verbose (show assertions)
```

### Stop on First Failure

```bash
pytest tests/ -x
```

### Show Print Statements

```bash
pytest tests/ -s
```

### Combine Options

```bash
# Verbose, stop on failure, show prints
pytest tests/test_dividend_service.py -xvs
```

### With Coverage

```bash
# Single file
pytest tests/test_dividend_service.py --cov=app.services.dividend_service

# All tests, all services
pytest tests/ --cov=app --cov-report=term-missing
```

### Performance Tests Only

```bash
pytest tests/test_portfolio_performance.py -v
```

---

## Best Practices

### 1. Always Use Fixtures

```python
# ❌ Bad
def test_something():
    app = create_app()  # Don't create app yourself
    # ...

# ✅ Good
def test_something(self, app_context, db_session):
    # Use provided fixtures
```

### 2. Commit After Factory Creation

```python
# ❌ Bad
fund = FundFactory()
result = Service.method(fund.id)  # fund.id might not exist yet!

# ✅ Good
fund = FundFactory()
db_session.commit()  # Ensure fund is saved
result = Service.method(fund.id)  # Now fund.id definitely exists
```

### 3. Use Direct Creation for Linked Objects

```python
# ❌ Problematic (SubFactory conflicts)
existing_fund = FundFactory()
db_session.commit()
dividend = DividendFactory(fund_id=existing_fund.id)  # SubFactory might override

# ✅ Better (direct creation)
dividend = Dividend(
    id=str(uuid.uuid4()),
    fund_id=existing_fund.id,  # Explicit link
    # ... other fields
)
db_session.add(dividend)
db_session.commit()
```

### 4. Test Edge Cases

```python
# Don't just test happy paths
def test_with_zero_value(self, app_context, db_session):
    result = Service.calculate(shares=0)  # Edge case
    assert result == 0

def test_with_negative_value(self, app_context, db_session):
    with pytest.raises(ValueError):
        Service.validate(price=-10)  # Error case
```

### 5. Document Test Intent

```python
def test_calculate_shares_with_dividend_transactions(self, app_context, db_session):
    """
    Test that dividend transactions are included in share calculation.

    This validates bug fix #1 where dividend transactions were being
    subtracted instead of added to the share count.
    """
    # Test code...
```

---

## Related Documentation

- **Test Organization**: `tests/docs/README.md`
- **Bug Fixes**: `tests/docs/BUG_FIXES_1.3.3.md`
- **Service Tests**: `tests/docs/*_SERVICE_TESTS.md`
- **Fixtures Source**: `tests/conftest.py`
- **Factories Source**: `tests/factories.py`
