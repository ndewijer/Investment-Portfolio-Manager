# Fund Routes Integration Tests

**File**: `tests/routes/test_fund_routes.py`
**Route File**: `app/routes/fund_routes.py`
**Test Count**: 19 tests (19 passing, 0 skipped)
**Status**: ✅ All tests passing (Phase 1 - Query.get_or_404() issues resolved)

---

## Overview

Integration tests for fund management API endpoints. These tests verify fund CRUD operations, symbol lookup, price management, and fund usage tracking.

### Endpoints Tested

1. **GET /api/funds** - List all funds ✅
2. **POST /api/funds** - Create fund ✅
3. **GET /api/funds/<fund_id>** - Get fund detail ✅ (FIXED in Phase 1)
4. **PUT /api/funds/<fund_id>** - Update fund ✅
5. **DELETE /api/funds/<fund_id>** - Delete fund ✅
6. **GET /api/funds/<fund_id>/check-usage** - Check fund usage ✅
7. **GET /api/lookup-symbol-info/<symbol>** - Lookup symbol info ✅
8. **GET /api/fund-prices/<fund_id>** - Get fund prices ✅ (FIXED in Phase 1)
9. **POST /api/fund-prices/<fund_id>/update** - Update fund prices ✅
10. **POST /api/funds/update-all-prices** - Update all fund prices ✅

### Phase 1 Fixes - Query.get_or_404() Session Scoping

**4 tests fixed** by moving database queries to service layer:

1. **GET /api/funds/<fund_id>** (3 tests) - Now uses `FundService.get_fund()` and `FundService.get_latest_fund_price()`
2. **GET /api/fund-prices/<fund_id>** (1 test) - Now uses `FundService.get_fund()` and `FundService.get_fund_price_history()`

Service methods use `db.session.get()` instead of deprecated `Query.get_or_404()`, resolving session scoping issues.

---

## Test Organization

### Test Classes

1. **TestFundListAndCreate** (4 tests)
   - List funds (empty and populated)
   - Create fund
   - Duplicate ISIN rejection

2. **TestFundRetrieveUpdateDelete** (8 tests)
   - Get fund detail ✅
   - Get fund with latest price ✅
   - Get fund not found (404) ✅
   - Update fund ✅
   - Update fund not found (404) ✅
   - Delete fund ✅
   - Delete fund in use (409 conflict) ✅

3. **TestFundUsage** (2 tests)
   - Check fund usage when in use ✅
   - Check fund usage when not in use ✅

4. **TestSymbolLookup** (2 tests)
   - Lookup symbol info (mocked) ✅
   - Lookup invalid symbol (404) ✅

5. **TestFundPrices** (3 tests)
   - Get fund prices ✅
   - Update today's price (mocked) ✅
   - Update historical prices (mocked) ✅

6. **TestUpdateAllPrices** (1 test)
   - Update all fund prices (with API key authentication)

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

**Consistency**: Same helper used across all route tests for uniformity.

---

## Key Test Patterns

### 1. Testing Fund Creation

```python
def test_create_fund(self, app_context, client, db_session):
    """Test POST /funds creates a new fund."""
    payload = {
        "name": "Test Fund",
        "isin": make_isin("US"),
        "symbol": "TEST",
        "currency": "USD",
        "exchange": "NYSE",
    }

    response = client.post("/api/funds", json=payload)

    assert response.status_code == 200
    data = response.get_json()
    assert "id" in data
    assert data["name"] == "Test Fund"

    # Verify database
    fund = db.session.get(Fund, data["id"])
    assert fund is not None
```

**Pattern**: Always verify both HTTP response AND database state.

### 2. Testing Duplicate ISIN Rejection

```python
def test_create_fund_duplicate_isin(self, app_context, client, db_session):
    """Test POST /funds rejects duplicate ISIN."""
    isin = make_isin("US")
    fund = create_fund("US", "VTI", "Existing Fund")
    fund.isin = isin
    db_session.add(fund)
    db_session.commit()

    payload = {
        "name": "Duplicate Fund",
        "isin": isin,  # Same ISIN
        "symbol": "DUP",
        "currency": "USD",
        "exchange": "NYSE",
    }

    response = client.post("/api/funds", json=payload)

    assert response.status_code == 400
    data = response.get_json()
    assert "error" in data or "message" in data
```

**Business Rule**: ISIN must be unique across all funds.

### 3. Testing Fund Usage Protection

```python
def test_delete_fund_in_use(self, app_context, client, db_session):
    """Test DELETE /funds/<fund_id> rejects deletion of fund in use."""
    fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
    portfolio = Portfolio(name="Test Portfolio")
    db_session.add_all([fund, portfolio])
    db_session.commit()

    # Add fund to portfolio
    pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
    db_session.add(pf)
    db_session.commit()

    response = client.delete(f"/api/funds/{fund.id}")

    assert response.status_code == 409  # Conflict
    data = response.get_json()
    assert "message" in data or "error" in data

    # Verify fund still exists
    fund_exists = db.session.get(Fund, fund.id)
    assert fund_exists is not None
```

**Business Rule**: Funds in use (attached to portfolios) cannot be deleted.

### 4. Testing Fund Usage Check

```python
def test_check_fund_usage_in_use(self, app_context, client, db_session):
    """Test GET /funds/<fund_id>/check-usage reports fund in use."""
    # ... create fund, portfolio, portfolio_fund, transaction ...

    response = client.get(f"/api/funds/{fund.id}/check-usage")

    assert response.status_code == 200
    data = response.get_json()
    assert data["in_use"] is True
    assert "portfolios" in data
    assert len(data["portfolios"]) >= 1
```

**Use Case**: Check if fund can be safely deleted before attempting deletion.

### 5. Mocking External API Calls

```python
def test_lookup_symbol_info_mock(self, app_context, client, monkeypatch):
    """Test GET /lookup-symbol-info/<symbol> with mocked response."""

    # Mock SymbolLookupService to avoid external API calls
    def mock_get_symbol_info(symbol, force_refresh=False):
        if symbol == "VTI":
            return {
                "symbol": "VTI",
                "name": "Vanguard Total Stock Market ETF",
                "currency": "USD",
                "exchange": "PCX",
            }
        return None

    from app.services import symbol_lookup_service

    monkeypatch.setattr(
        symbol_lookup_service.SymbolLookupService,
        "get_symbol_info",
        staticmethod(mock_get_symbol_info),
    )

    response = client.get("/api/lookup-symbol-info/VTI")

    assert response.status_code == 200
    data = response.get_json()
    assert data["symbol"] == "VTI"
```

**Why**: Integration tests should not make real external API calls to yfinance or other services.

### 6. Testing API Key Protected Endpoints

```python
def test_update_all_fund_prices(self, app_context, client, db_session, monkeypatch):
    """Test POST /funds/update-all-prices with API key authentication."""
    import hashlib
    from datetime import UTC, datetime

    # Set up API key authentication
    api_key = "test_api_key_12345"
    monkeypatch.setenv("INTERNAL_API_KEY", api_key)

    # Generate time-based token (same logic as @require_api_key decorator)
    current_hour = datetime.now(UTC).strftime("%Y-%m-%d-%H")
    time_token = hashlib.sha256(f"{api_key}{current_hour}".encode()).hexdigest()

    # Make request with authentication headers
    headers = {"X-API-Key": api_key, "X-Time-Token": time_token}

    response = client.post("/api/funds/update-all-prices", headers=headers)

    assert response.status_code == 200
```

**Authentication Requirements**:
- `X-API-Key` header must match `INTERNAL_API_KEY` environment variable
- `X-Time-Token` header must be SHA256 hash of `{api_key}{current_hour}`
- Token changes every hour for security

**Testing Strategy**: Set environment variable and calculate token in test for full integration testing.

---

## Important Notes

### Session Scoping Issues

Some endpoints use direct model queries (`Fund.query.get_or_404()`, `FundPrice.query.filter_by()`) which have session scoping issues in the test environment. These are documented in the route refactoring plan:

```python
# ❌ Causes session scoping issues in tests
fund = Fund.query.get_or_404(fund_id)

# ✅ Would work in tests (uses service layer)
fund = FundService.get_fund(fund_id)
```

**Affected Endpoints**:
- `GET /api/funds/<fund_id>` (Remediation Plan #8)
- `GET /api/fund-prices/<fund_id>` (Remediation Plan #9)

### API Key Protected Endpoints

The `POST /api/funds/update-all-prices` endpoint requires API key authentication via `@require_api_key` decorator. The decorator validates:
1. `X-API-Key` header matches `INTERNAL_API_KEY` environment variable
2. `X-Time-Token` header matches SHA256 hash of `{api_key}{current_hour}`

**Testing Strategy**: Set environment variable and generate time-based token in test for full integration testing (see "Testing API Key Protected Endpoints" section above).

### Fund Usage Response Structure

The fund usage endpoint returns different structures based on usage:

```python
# Fund NOT in use
{
    "in_use": False
    # No "portfolios" key
}

# Fund in use
{
    "in_use": True,
    "portfolios": [
        {"id": "...", "name": "Portfolio 1"},
        {"id": "...", "name": "Portfolio 2"}
    ],
    "transaction_count": 5,
    "dividend_count": 2
}
```

**Test Pattern**: Accept optional `portfolios` key to handle both cases.

---

## Running Tests

### Run all fund route tests:
```bash
pytest tests/routes/test_fund_routes.py -v
```

### Run specific test class:
```bash
pytest tests/routes/test_fund_routes.py::TestFundListAndCreate -v
```

### Run without skipped tests:
```bash
pytest tests/routes/test_fund_routes.py -v -k "not skip"
```

### Run without coverage (faster):
```bash
pytest tests/routes/test_fund_routes.py -v --no-cov
```

---

## Test Results

**15 tests passing, 4 tests skipped** ✅

### Test Execution Time
- **Average**: ~2.0 seconds for full suite
- **Pattern**: Tests are fast because external API calls are mocked

### Coverage

Integration tests verify:
- ✅ Fund listing and creation
- ✅ Fund updates with symbol changes
- ✅ Fund deletion with usage protection
- ✅ Duplicate ISIN rejection
- ✅ Fund usage tracking
- ✅ Symbol lookup (mocked)
- ✅ Price updates (mocked)
- ✅ Bulk price updates (with API key authentication)
- ⏭️ Fund detail retrieval (skipped - session scoping issue)
- ⏭️ Price history retrieval (skipped - session scoping issue)

---

## Related Documentation

- **Service Tests**: `tests/docs/services/FUND_SERVICE_TESTS.md` (24 tests, 100% coverage)
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/fund_routes.py`
- **Fund Service**: `app/services/fund_service.py`
- **Route Refactoring**: `todo/ROUTE_REFACTORING_REMEDIATION_PLAN.md` (#8, #9)

---

## Technical Debt

### Endpoints Requiring Refactoring

The following endpoints need refactoring to use service layer instead of direct queries:

1. **GET /api/funds/<fund_id>** (lines 153-217)
   - Currently: `Fund.query.get_or_404(fund_id)`
   - Should be: `FundService.get_fund(fund_id)`
   - Benefit: Works correctly in test environment

2. **GET /api/fund-prices/<fund_id>** (lines 436-477)
   - Currently: `Fund.query.get_or_404(fund_id)` and `FundPrice.query.filter_by()`
   - Should be: Service layer methods
   - Benefit: Consistent with other endpoints

See `todo/ROUTE_REFACTORING_REMEDIATION_PLAN.md` for implementation details.

---

**Last Updated**: Phase 5 (Route Integration Tests)
**Maintainer**: See git history
