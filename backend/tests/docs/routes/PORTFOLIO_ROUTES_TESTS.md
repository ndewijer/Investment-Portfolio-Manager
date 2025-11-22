# Portfolio Routes Integration Tests

**File**: `tests/routes/test_portfolio_routes.py`
**Route File**: `app/routes/portfolio_routes.py`
**Test Count**: 30 tests (22 integration + 8 error path)
**Coverage**: 100% (110/110 statements)
**Status**: ✅ All tests passing

---

## Overview

Integration tests for all portfolio management API endpoints. These tests verify that the HTTP endpoints correctly handle requests, interact with services, and return appropriate responses.

### Endpoints Tested

1. **GET /api/portfolios** - List all portfolios
2. **POST /api/portfolios** - Create portfolio
3. **GET /api/portfolios/<id>** - Get portfolio detail with metrics
4. **PUT /api/portfolios/<id>** - Update portfolio
5. **DELETE /api/portfolios/<id>** - Delete portfolio
6. **POST /api/portfolios/<id>/archive** - Archive portfolio
7. **POST /api/portfolios/<id>/unarchive** - Unarchive portfolio
8. **GET /api/portfolio-summary** - Get portfolio summary (overview)
9. **GET /api/portfolio-history** - Get portfolio historical performance
10. **GET /api/portfolio-funds** - List portfolio funds (with optional filtering)
11. **POST /api/portfolio-funds** - Add fund to portfolio
12. **DELETE /api/portfolio-funds/<id>** - Remove fund from portfolio
13. **GET /api/portfolios/<id>/fund-history** - Get fund-specific history

---

## Test Organization

### Test Classes

1. **TestPortfolioListAndCreate** (4 tests)
   - List portfolios (empty and populated)
   - Create portfolio (full and minimal data)

2. **TestPortfolioRetrieveUpdateDelete** (6 tests)
   - Get portfolio detail with metrics
   - Get non-existent portfolio (404)
   - Get archived portfolio (404)
   - Update portfolio
   - Delete portfolio
   - Handle not found errors

3. **TestPortfolioArchiving** (2 tests)
   - Archive portfolio
   - Unarchive portfolio

4. **TestPortfolioSummaryAndHistory** (3 tests)
   - Get portfolio summary with aggregated metrics
   - Verify archived portfolios excluded
   - Get portfolio historical performance

5. **TestPortfolioFunds** (7 tests)
   - List all portfolio funds
   - List portfolio funds filtered by portfolio_id
   - Create portfolio-fund relationship
   - Delete portfolio fund without transactions
   - Delete portfolio fund with transactions (requires confirmation)
   - Get fund-specific historical data

6. **TestPortfolioErrors** (8 tests)
   - Create portfolio service error
   - Get portfolio not found
   - Update portfolio service error
   - Delete portfolio service error
   - Archive portfolio not found
   - Unarchive portfolio not found
   - Create portfolio fund duplicate error
   - Delete portfolio fund requires confirmation

---

## Helper Functions

### `create_fund()`
```python
def create_fund(isin_prefix="US", symbol_prefix="TEST", name="Test Fund",
                currency="USD", exchange="NYSE"):
    """Helper to create a Fund with all required fields."""
    return Fund(
        isin=make_isin(isin_prefix),
        symbol=make_symbol(symbol_prefix),
        name=name,
        currency=currency,
        exchange=exchange,
    )
```

**Why**: Fund model requires `currency` and `exchange` fields. This helper ensures all required fields are provided consistently across tests.

---

## Key Test Patterns

### 1. Testing Empty State
```python
def test_list_portfolios_empty(self, app_context, client, db_session):
    """Test GET /portfolios returns empty list when no portfolios exist."""
    response = client.get("/api/portfolios")

    assert response.status_code == 200
    data = response.get_json()
    assert isinstance(data, list)
    assert len(data) == 0
```

**Why**: Verifies the endpoint handles empty database state correctly.

**Important**: The `db_session` fixture is required here even though it's not directly used in the test. This fixture triggers database cleanup before the test runs, ensuring a truly empty state for testing.

### 2. Testing Full CRUD Operations
```python
def test_create_portfolio(self, app_context, client, db_session):
    payload = {"name": "My New Portfolio", "description": "Test portfolio"}
    response = client.post("/api/portfolios", json=payload)

    assert response.status_code == 200
    data = response.get_json()
    assert data["name"] == "My New Portfolio"

    # Verify database
    portfolio = db.session.get(Portfolio, data["id"])
    assert portfolio is not None
```

**Pattern**: Always verify both the HTTP response AND the database state.

### 3. Testing Archived Portfolio Behavior
```python
def test_get_archived_portfolio_returns_404(self, app_context, client, db_session):
    portfolio = Portfolio(name="Archived", is_archived=True)
    db_session.add(portfolio)
    db_session.commit()

    response = client.get(f"/api/portfolios/{portfolio.id}")

    assert response.status_code == 404
    data = response.get_json()
    assert "archived" in data["error"].lower()
```

**Why**: Archived portfolios should not be accessible via normal detail endpoint.

### 4. Testing Confirmation Flow
```python
def test_delete_portfolio_fund_with_transactions_requires_confirmation(
    self, app_context, client, db_session
):
    # ... create portfolio fund with transaction ...

    # Try delete without confirmation
    response = client.delete(f"/api/portfolio-funds/{pf.id}")

    assert response.status_code == 409  # Conflict
    assert "confirmation" in str(data).lower()
```

**Why**: Deleting portfolio funds with transactions should require explicit confirmation to prevent accidental data loss.

---

## Important Notes

### SQLAlchemy 2.0 Patterns

**Old (Deprecated)**:
```python
portfolio = Portfolio.query.get(portfolio_id)  # ❌ Deprecated
```

**New (SQLAlchemy 2.0)**:
```python
portfolio = db.session.get(Portfolio, portfolio_id)  # ✅ Correct
```

All tests use the new `db.session.get()` pattern.

### Historical Price Data

FundPrice model uses `price` field (not `close`):
```python
fund_price = FundPrice(
    fund_id=fund.id,
    date=datetime.now().date(),
    price=float(Decimal("160.00")),  # ✅ 'price', not 'close'
)
```

### Response Structure Flexibility

Integration tests focus on:
- ✅ HTTP status codes
- ✅ Response is correct type (dict, list)
- ✅ Key fields are present

They do NOT rigidly enforce:
- ❌ Exact field names (camelCase vs snake_case)
- ❌ Exact response structure

**Why**: Frontend and backend may use different naming conventions. Integration tests verify the endpoint works, not the exact serialization format.

---

## Error Path Testing (Phase 4a)

### TestPortfolioErrors Class

Added comprehensive error path tests to achieve 100% coverage on `portfolio_routes.py`. This phase also included removing dead code (duplicate route registration at lines 394-400).

**Tests Added**:
1. **test_create_portfolio_service_error** - Tests POST /portfolios handles service exceptions
2. **test_get_portfolio_not_found** - Tests GET /portfolios/<id> handles missing portfolios
3. **test_update_portfolio_service_error** - Tests PUT /portfolios/<id> handles service errors
4. **test_delete_portfolio_service_error** - Tests DELETE /portfolios/<id> handles service errors
5. **test_archive_portfolio_not_found** - Tests POST /portfolios/<id>/archive error handling
6. **test_unarchive_portfolio_not_found** - Tests POST /portfolios/<id>/unarchive error handling
7. **test_create_portfolio_fund_duplicate_error** - Tests POST /portfolio-funds handles duplicates
8. **test_delete_portfolio_fund_requires_confirmation** - Tests DELETE /portfolio-funds/<id> confirmation flow

**Coverage Improvement**: 89% → 100% (all exception handlers + dead code removal)

**Testing Pattern**:
```python
from unittest.mock import patch

def test_create_portfolio_service_error(self, client):
    """Test POST /portfolios handles service errors."""
    with patch("app.routes.portfolio_routes.PortfolioService.create_portfolio") as mock_create:
        mock_create.side_effect = Exception("Database error")

        payload = {"name": "Test Portfolio"}
        response = client.post("/api/portfolios", json=payload)

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data
```

**Why This Matters**: Error path tests ensure the API gracefully handles service layer failures, database errors, and invalid operations, returning appropriate HTTP status codes and error messages to clients.

---

## Running Tests

### Run all portfolio route tests:
```bash
pytest tests/routes/test_portfolio_routes.py -v
```

### Run specific test class:
```bash
pytest tests/routes/test_portfolio_routes.py::TestPortfolioListAndCreate -v
```

### Run specific test:
```bash
pytest tests/routes/test_portfolio_routes.py::TestPortfolioListAndCreate::test_create_portfolio -v
```

### Run without coverage (faster):
```bash
pytest tests/routes/test_portfolio_routes.py -v --no-cov
```

### Suppress SQLAlchemy warnings:
```bash
pytest tests/routes/test_portfolio_routes.py -v -W ignore::sqlalchemy.exc.LegacyAPIWarning
```

---

## Test Results

**All 30 tests passing** ✅

### Test Execution Time
- **Average**: ~0.32 seconds for full suite
- **Pattern**: Tests are fast because they use in-memory SQLite database

### Coverage
- **Route Coverage**: 100% (110/110 statements, 0 missing lines)
- **Coverage Improvement**: 89% → 100% (Phase 4a error path testing + dead code removal)
- Integration tests verify **all 13 portfolio endpoints** are accessible and return appropriate responses
- Error tests verify **all exception handlers** return appropriate status codes

---

## Related Documentation

- **Service Tests**: `tests/docs/services/PORTFOLIO_SERVICE_TESTS.md`
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/portfolio_routes.py`
- **Portfolio Service**: `app/services/portfolio_service.py`

---

## Phase 1b Fixes - Query.get() and Query.all() Deprecation

During Phase 1b of route test fixes, we identified and resolved several deprecated SQLAlchemy Query API usages in `portfolio_routes.py`:

### Issues Fixed

1. **List Portfolios Route** (Line 83)
   - **Before**: `Portfolio.query.all()`
   - **After**: `PortfolioService.get_all_portfolios()`
   - Uses SQLAlchemy 2.0 `select()` pattern internally

2. **Portfolio Fund Deletion Error Handling** (Lines 342-348)
   - **Before**: `PortfolioFund.query.options(...).get(portfolio_fund_id)`
   - **After**: `PortfolioService.get_portfolio_fund(portfolio_fund_id, with_relationships=True)`
   - Preserves eager loading of fund and portfolio relationships

3. **Transaction/Dividend Counting** (Lines 347-352)
   - **Before**: `Transaction.query.filter_by(...).count()` and `Dividend.query.filter_by(...).count()`
   - **After**: `PortfolioService.count_portfolio_fund_transactions()` and `PortfolioService.count_portfolio_fund_dividends()`
   - Uses `func.count()` with SQLAlchemy 2.0 patterns

4. **Test Isolation Fix** (test_list_portfolios_empty)
   - **Issue**: Test was missing `db_session` fixture, causing data from previous tests to leak
   - **Fix**: Added `db_session` parameter to trigger database cleanup

### Service Methods Added

In `app/services/portfolio_service.py`:
- `get_all_portfolios()` - Retrieve all portfolios without filtering
- `get_portfolio_fund(portfolio_fund_id, with_relationships=False)` - Get PortfolioFund with optional eager loading
- `count_portfolio_fund_transactions(portfolio_fund_id)` - Count transactions for a portfolio fund
- `count_portfolio_fund_dividends(portfolio_fund_id)` - Count dividends for a portfolio fund

### Test Coverage

Added 6 new service tests in `tests/services/test_portfolio_service.py`:
- `test_get_all_portfolios` - Verify all portfolios retrieved regardless of flags
- `test_get_portfolio_fund_without_relationships` - Test basic retrieval
- `test_get_portfolio_fund_with_relationships` - Test with eager loading
- `test_get_portfolio_fund_not_found` - Test 404 handling
- `test_count_portfolio_fund_transactions` - Test transaction counting
- `test_count_portfolio_fund_dividends` - Test dividend counting

### Results

✅ All 22 portfolio route tests now pass
✅ No SQLAlchemy deprecation warnings
✅ Test isolation issues resolved
✅ Service layer coverage maintained at 91%

---

**Last Updated**: Phase 5 - Phase 1b (Query API Deprecation Fixes) + Phase 4a (Error Path Testing)
**Maintainer**: See git history
