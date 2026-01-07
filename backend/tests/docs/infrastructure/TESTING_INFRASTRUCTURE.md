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

**Production Database**: `backend/data/db/portfolio_manager.db`\
**Test Database**: `/tmp/test_portfolio_manager.db`

**Why separate**:
- Zero risk of contaminating production data
- Tests can create/modify/delete freely
- Parallel test execution possible
- Fast cleanup between tests

### Configuration

**File**: `tests/test_config.py`

```python
from pathlib import Path
from cryptography.fernet import Fernet

# Get the backend directory path
BACKEND_DIR = Path(__file__).parent.parent
DATA_DIR = BACKEND_DIR / "data" / "db"

# Ensure test database directory exists
DATA_DIR.mkdir(parents=True, exist_ok=True)

# Test database configuration
TEST_DATABASE_PATH = DATA_DIR / "test_portfolio_manager.db"
TEST_DATABASE_URI = f"sqlite:///{TEST_DATABASE_PATH}"

# Generate a valid Fernet key for testing (Phase 5)
TEST_ENCRYPTION_KEY = Fernet.generate_key().decode()

TEST_CONFIG = {
    "TESTING": True,
    "SQLALCHEMY_DATABASE_URI": TEST_DATABASE_URI,
    "SQLALCHEMY_TRACK_MODIFICATIONS": False,
    # Flask secret key for sessions
    "SECRET_KEY": "test-secret-key-not-for-production",
    # Disable CSRF protection in tests
    "WTF_CSRF_ENABLED": False,
    # IBKR encryption key for testing (Phase 5)
    "IBKR_ENCRYPTION_KEY": TEST_ENCRYPTION_KEY,
}

def cleanup_test_database():
    """Remove test database file after test session."""
    if TEST_DATABASE_PATH.exists():
        TEST_DATABASE_PATH.unlink()
```

**Key Configuration Items**:

- **TESTING**: Enables test mode, prevents production behaviors
- **SQLALCHEMY_DATABASE_URI**: Isolated test database location
- **SECRET_KEY**: Test-only secret key (never use in production)
- **WTF_CSRF_ENABLED**: Disabled for easier API testing
- **IBKR_ENCRYPTION_KEY**: Valid Fernet key for encryption tests (added Phase 5)

**IBKR Encryption Key** (Phase 5):
- Generated using `Fernet.generate_key()` at test session startup
- Provides a valid 32-byte base64-encoded key for tests
- Required for `IBKRFlexService` encryption/decryption tests
- Different from production key (which is auto-generated or set via environment)

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
    response = client.get('/api/portfolio')
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
    found = db.session.get(Fund, fund.id)
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

## Standardized Test Helpers

### Test Helper Utilities

**File**: `tests/test_helpers.py`

Standardized utilities for consistent UUID generation and test data creation across all service tests.

#### UUID Generation Functions

```python
from tests.test_helpers import (
    make_isin, make_symbol, make_id, make_ibkr_transaction_id,
    make_ibkr_txn_id, make_dividend_txn_id, make_custom_string, make_portfolio_name
)

# Generate unique ISIN (standardized)
isin = make_isin("US")  # "US1A2B3C4D5E"
isin_de = make_isin("DE")  # "DE1A2B3C4D5E"

# Generate unique symbols (standardized)
symbol = make_symbol("AAPL")  # "AAPL1A2B" (4 chars default)
symbol_long = make_symbol("MSFT", 6)  # "MSFT1A2B3C" (6 chars)

# Generate unique IDs (standardized)
test_id = make_id()  # str(uuid.uuid4())

# Generate transaction IDs (multiple formats)
txn_id = make_ibkr_transaction_id()  # "TXN1A2B3C4D5E"
ibkr_id = make_ibkr_txn_id()  # "IBKR_uuid"
div_id = make_dividend_txn_id()  # "DIV_uuid"

# Generate custom strings with flexible prefixes
cache_key = make_custom_string("test_cache_", 8)  # "test_cache_1A2B3C4D"
query_id = make_custom_string("query_", 6)  # "query_1A2B3C"

# Generate unique portfolio names
portfolio = make_portfolio_name("Test Portfolio")  # "Test Portfolio 1A2B3C"
```

**Why standardize**: Ensures consistent UUID slice lengths (10 for ISIN, 4 for symbols) and uppercase formatting across all tests.

**Standardization Status**:
- ✅ **12/12 files fully standardized** (100% complete)
- ✅ **All UUID patterns replaced** with standardized helper functions
- ✅ **Zero technical debt** remaining in UUID generation
- 📋 **Complete documentation**: See `/backend/tests/STANDARDIZATION_GUIDE.md`

#### Common Test Constants

```python
from tests.test_helpers import COMMON_CURRENCIES, INVALID_UTF8_BYTES

# Use consistent test values
currencies = COMMON_CURRENCIES  # ["USD", "EUR", "GBP", "JPY", "CAD"]
invalid_data = INVALID_UTF8_BYTES  # b'\xff\xfe\xfd'
```

### Before/After Standardization

**Before** (inconsistent patterns):
```python
# Different UUID slice lengths across tests
isin1 = f"US{uuid.uuid4().hex[:8].upper()}"   # 8 chars
isin2 = f"US{uuid.uuid4().hex[:10].upper()}"  # 10 chars
symbol1 = f"AAPL{uuid.uuid4().hex[:4]}"       # lowercase
symbol2 = f"MSFT{uuid.uuid4().hex[:6].upper()}"  # uppercase, different length
```

**After** (standardized):
```python
from tests.test_helpers import make_isin, make_symbol

# Consistent patterns everywhere
isin1 = make_isin("US")    # Always 10 chars, always uppercase
isin2 = make_isin("DE")    # Always 10 chars, always uppercase
symbol1 = make_symbol("AAPL")  # Always 4 chars default, always uppercase
symbol2 = make_symbol("MSFT", 6)  # Explicit length when needed
```

## Factory Pattern

Factories generate test data with realistic defaults using **factory_boy**.

### What Are Factories?

Factories are classes that create model instances with sensible defaults:

**Without factories** (tedious):
```python
from tests.test_helpers import make_id, make_isin, make_symbol

fund = Fund(
    id=make_id(),
    name="Test Fund",
    symbol=make_symbol("TEST"),
    isin=make_isin("US"),
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

### Database State Management Patterns

#### Standard Cleanup Approaches

**For Most Services** (relies on fixture isolation):
```python
def test_something(self, app_context, db_session):
    # Create test data
    fund = FundFactory()
    db_session.commit()

    # Test and verify
    # (test data cleaned up automatically by fixture)
```

**For Services with UNIQUE Constraints** (manual cleanup):
```python
# Example: SystemSetting, IBKRConfig have unique constraints
def test_logging_setting(self, app_context, db_session):
    # Clear existing records to prevent conflicts
    db_session.query(SystemSetting).filter_by(
        key=SystemSettingKey.LOGGING_ENABLED
    ).delete()
    db_session.commit()

    # Now create test data safely
    setting = SystemSetting(...)
    db_session.add(setting)
    db_session.commit()
```

**Autouse Fixture Pattern** (for services needing consistent cleanup):
```python
# In test file for services with unique constraints
@pytest.fixture(autouse=True)
def clean_special_table(db_session):
    """Clean special table before/after each test."""
    # Before test
    SpecialModel.query.delete()
    db_session.commit()

    yield

    # After test (optional)
    SpecialModel.query.delete()
    db_session.commit()
```

#### Mock and Patch Patterns

**Standard yfinance Mocking**:
```python
from unittest.mock import patch, MagicMock

@patch("app.services.symbol_lookup_service.yf.Ticker")
def test_symbol_lookup(self, mock_ticker, app_context, db_session):
    # Setup mock response
    mock_instance = MagicMock()
    mock_instance.info = {"longName": "Apple Inc", "symbol": "AAPL"}
    mock_ticker.return_value = mock_instance

    # Test the service
    result = SymbolLookupService.get_symbol_info("AAPL")

    # Verify
    assert result["name"] == "Apple Inc"
    mock_ticker.assert_called_once_with("AAPL")
```

**Database Error Simulation**:
```python
from unittest.mock import patch

def test_database_error_handling(self, app_context, db_session):
    with patch.object(db.session, 'commit', side_effect=Exception("Database error")):
        with pytest.raises(ValueError, match="Database error"):
            SomeService.save_data(...)
```

**HTTP Response Mocking** (using responses library):
```python
import responses

@responses.activate
def test_http_request(self, app_context):
    # Mock HTTP endpoint
    responses.add(
        responses.GET,
        "https://api.example.com/data",
        json={"result": "success"},
        status=200
    )

    # Test service that makes HTTP call
    result = SomeService.fetch_data()
    assert result["result"] == "success"
```

#### Error Testing Patterns

**Exception Testing** (for validation errors):
```python
def test_validation_error(self, app_context, db_session):
    with pytest.raises(ValueError, match=r"Invalid.*format"):
        SomeService.validate_input("invalid_data")
```

**Service Result Testing** (for operational errors):
```python
def test_service_error_result(self, app_context, db_session):
    result = SomeService.process_data(invalid_input)
    assert result["success"] is False
    assert "error message" in result["error"]
```

#### Edge Case Testing Patterns

**Zero/Empty Value Testing**:
```python
from tests.test_helpers import ZERO_VALUES

def test_edge_cases(self, app_context, db_session):
    for zero_value in ZERO_VALUES:
        result = SomeService.calculate(zero_value)
        assert result == 0  # or appropriate behavior
```

**UTF-8 and BOM Testing** (critical for CSV processing):
```python
def test_utf8_and_bom(self, app_context):
    # Test standard UTF-8
    csv_content = "date,price\n2024-03-15,150.75"
    utf8_content = csv_content.encode("utf-8")
    result1 = SomeService.process_csv(utf8_content)

    # Test UTF-8 with BOM (critical for Excel compatibility)
    bom_content = csv_content.encode("utf-8-sig")  # Includes BOM
    result2 = SomeService.process_csv(bom_content)

    # Both should produce same result
    assert result1 == result2
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

**Service layer**: 90%+ coverage\
**Route layer**: 80%+ coverage (future Phase 4)\
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
