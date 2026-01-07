# Dividend Routes Integration Tests

**File**: `tests/api/test_dividend_routes.py`
**Route File**: `app/routes/dividend_routes.py`
**Test Count**: 17 tests (10 integration + 7 error path)
**Coverage**: 100% (65/65 statements)
**Status**: ✅ All tests passing

---

## Docstring Reference

See test file docstrings for detailed test documentation and implementation notes.

---

## Overview

Integration tests for dividend management API endpoints. Verifies dividend CRUD operations, filtering by fund/portfolio, and proper calculation of dividend amounts based on transaction history.

### Endpoints Tested

1. **POST /api/dividend** - Create dividend
2. **GET /api/dividend/fund/<fund_id>** - Get dividends by fund
3. **GET /api/dividend/portfolio/<portfolio_id>** - Get dividends by portfolio
4. **PUT /api/dividend/<dividend_id>** - Update dividend
5. **DELETE /api/dividend/<dividend_id>** - Delete dividend

---

## Test Organization

### TestDividendCreate (2 tests)
- Create dividend with transaction history
- Verify total amount calculation (shares_owned × dividend_per_share)

### TestDividendRetrieve (4 tests)
- Get dividends by fund
- Get dividends by fund (not found)
- Get dividends by portfolio
- Get dividends by portfolio (not found)

### TestDividendUpdateDelete (4 tests)
- Update dividend
- Update non-existent dividend (404)
- Delete dividend
- Delete non-existent dividend (404/500)

### TestDividendErrors (7 tests)
- Create dividend service error
- Get fund dividends service error
- Get portfolio dividends service error
- Update dividend value error
- Update dividend general error
- Delete dividend value error
- Delete dividend general error

---

## Key Test Patterns

### 1. Dividend Creation Requires Transaction History

**CRITICAL**: The dividend service calculates `shares_owned` from transaction history, NOT from the request payload. Tests must create transactions before creating dividends to ensure accurate share calculations.

**Formula**: `total_amount = shares_owned × dividend_per_share`

### 2. Filtering by Fund vs Portfolio

- **By Fund**: Returns all dividends for a specific fund across all portfolios
- **By Portfolio**: Returns all dividends for all funds within a specific portfolio

### 3. Dividend Updates Recalculate Shares

When updating a dividend, the service recalculates `shares_owned` based on the (potentially new) record date and transaction history.

### 4. Error Path Testing

All error paths use `unittest.mock.patch` to simulate service layer exceptions. Tests verify graceful error handling with appropriate HTTP status codes (400/404/500).

### 5. Dividend Types Support

Tests include `dividend_type` parameter in `create_fund()` helper to support both CASH and STOCK dividend types.

---

## Running Tests

### Run all dividend route tests:
```bash
pytest tests/api/test_dividend_routes.py -v
```

### Run specific test class:
```bash
pytest tests/api/test_dividend_routes.py::TestDividendCreate -v
```

### Run with transaction debugging:
```bash
pytest tests/api/test_dividend_routes.py -v -s
```

### Run without coverage (faster):
```bash
pytest tests/api/test_dividend_routes.py -v --no-cov
```

---

## Test Results

**All 17 tests passing** ✅

- **Route Coverage**: 100% (65/65 statements, 0 missing lines)
- **Coverage Improvement**: 78% → 100% (Phase 4d error path testing)
- **Average Execution Time**: ~0.35 seconds for full suite

---

## Related Documentation

- **Service Tests**: `tests/docs/services/DIVIDEND_SERVICE_TESTS.md` (21 tests, 91% coverage)
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/dividend_routes.py`
- **Dividend Service**: `app/services/dividend_service.py`
- **Bug Fixes**: `tests/docs/phases/BUG_FIXES_1.3.3.md` - 2 critical dividend bugs discovered

---

**Last Updated**: Phase 5 (Route Integration Tests) + Phase 4d (Error Path Testing)
**Maintainer**: See git history
