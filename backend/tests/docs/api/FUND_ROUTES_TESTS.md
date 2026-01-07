# Fund Routes Integration Tests

**File**: `tests/api/test_fund_routes.py`
**Route File**: `app/routes/fund_routes.py`
**Test Count**: 19 tests (19 passing, 0 skipped)
**Status**: ✅ All tests passing (Phase 1 - Query.get_or_404() issues resolved)

---

## Docstring Reference

All test implementation details, parameters, and assertions are documented in the test file docstrings. Refer to `tests/api/test_fund_routes.py` for complete test specifications.

---

## Overview

Integration tests for fund management API endpoints, verifying CRUD operations, symbol lookup, price management, and fund usage tracking.

### Endpoints Tested

1. **GET /api/fund** - List all funds
2. **POST /api/fund** - Create fund
3. **GET /api/fund/<fund_id>** - Get fund detail (uses FundService)
4. **PUT /api/fund/<fund_id>** - Update fund
5. **DELETE /api/fund/<fund_id>** - Delete fund
6. **GET /api/fund/<fund_id>/check-usage** - Check fund usage
7. **GET /api/lookup-symbol-info/<symbol>** - Lookup symbol info
8. **GET /api/fund-prices/<fund_id>** - Get fund prices (uses FundService)
9. **POST /api/fund-prices/<fund_id>/update** - Update fund prices
10. **POST /api/fund/update-all-prices** - Update all fund prices

**Phase 1 Refactoring**: Endpoints #3 and #8 refactored to use service layer (`FundService.get_fund()`, `FundService.get_latest_fund_price()`, `FundService.get_fund_price_history()`) instead of `Query.get_or_404()`.

---

## Test Organization

### Test Classes

**TestFundListAndCreate** (4 tests)
- List funds (empty) - Returns empty array for no funds
- List funds (populated) - Returns all funds with details
- Create fund - Creates fund with valid data
- Duplicate ISIN rejection - Rejects fund with existing ISIN (400)

**TestFundRetrieveUpdateDelete** (8 tests)
- Get fund detail - Returns fund by ID
- Get fund with latest price - Returns fund with most recent price
- Get fund not found - Returns 404 for invalid ID
- Update fund - Updates fund details successfully
- Update fund not found - Returns 404 for invalid ID
- Delete fund - Deletes fund not in use
- Delete fund not found - Returns 404 for invalid ID
- Delete fund in use - Rejects deletion of fund attached to portfolios (409)

**TestFundUsage** (2 tests)
- Check fund usage (in use) - Returns portfolios and transaction counts
- Check fund usage (not in use) - Returns in_use: false

**TestSymbolLookup** (2 tests)
- Lookup symbol info - Returns symbol details (mocked yfinance)
- Lookup invalid symbol - Returns 404 for unknown symbol

**TestFundPrices** (3 tests)
- Get fund prices - Returns price history for fund
- Update today's price - Updates latest price from API (mocked)
- Update historical prices - Backfills price history (mocked)

**TestUpdateAllPrices** (1 test)
- Update all fund prices - Bulk updates with API key authentication

---

## Key Patterns

### Test Infrastructure
- **Fixtures**: `app_context`, `client`, `db_session` for isolated test environment
- **Helper**: `create_fund()` creates Fund instances with consistent required fields
- **Uniqueness**: `make_isin()` and `make_symbol()` generate unique identifiers

### Business Rules Tested
- **ISIN Uniqueness**: Duplicate ISIN rejected with 400 error
- **Fund Usage Protection**: Funds in portfolios cannot be deleted (409 conflict)
- **Fund Usage Check**: Endpoint reports portfolios, transaction counts, dividend counts
- **Symbol Validation**: Invalid symbols return 404

### Testing Strategies
- **Response + Database**: Always verify both HTTP response and database state
- **Mocking External APIs**: All yfinance calls mocked to avoid external dependencies
- **API Key Authentication**: Time-based token (SHA256 of `{api_key}{current_hour}`) required for bulk operations
- **Service Layer**: Tests verify routes delegate to service methods properly

### Authentication Pattern
Protected endpoints (`/fund/update-all-prices`) require:
- `X-API-Key` header matching `INTERNAL_API_KEY` env variable
- `X-Time-Token` header with SHA256(`{api_key}{YYYY-MM-DD-HH}`)
- Tests set env variable and calculate token for full integration testing

### Fund Usage Response Structure
```
Not in use: {"in_use": false}
In use: {"in_use": true, "portfolios": [...], "transaction_count": N, "dividend_count": N}
```

---

## Running Tests

```bash
# All fund route tests
pytest tests/api/test_fund_routes.py -v

# Specific test class
pytest tests/api/test_fund_routes.py::TestFundListAndCreate -v

# Without coverage (faster)
pytest tests/api/test_fund_routes.py -v --no-cov
```

**Execution Time**: ~2.0 seconds (mocked external APIs)

---

## Related Documentation

- **Service Tests**: `tests/docs/services/FUND_SERVICE_TESTS.md`
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/fund_routes.py`
- **Fund Service**: `app/services/fund_service.py`

---

**Last Updated**: Phase 5 (Route Integration Tests)
