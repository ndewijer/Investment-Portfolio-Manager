# Portfolio Routes Integration Tests

**File**: `tests/api/test_portfolio_routes.py`
**Route File**: `app/routes/portfolio_routes.py`
**Test Count**: 30 tests (22 integration + 8 error path)
**Coverage**: 100% (110/110 statements)
**Status**: ✅ All tests passing

---

## Docstring Reference

All test behavior and implementation details are documented in the test file docstrings. This document provides test organization and running instructions.

---

## Overview

Integration tests for all portfolio management API endpoints. Verifies HTTP endpoints correctly handle requests, interact with services, and return appropriate responses.

**Endpoints Tested**: 13 total
- Portfolio CRUD: list, create, get, update, delete
- Portfolio archiving: archive, unarchive
- Portfolio analytics: summary, history, fund-specific history
- Portfolio funds: list (with filtering), add, remove

---

## Test Organization

### TestPortfolioListAndCreate (4 tests)
- List portfolios when empty
- List portfolios with data
- Create portfolio with full data
- Create portfolio with minimal data

### TestPortfolioRetrieveUpdateDelete (6 tests)
- Get portfolio detail with metrics
- Get non-existent portfolio returns 404
- Get archived portfolio returns 404
- Update portfolio successfully
- Delete portfolio successfully
- Delete non-existent portfolio returns 404

### TestPortfolioArchiving (2 tests)
- Archive portfolio successfully
- Unarchive portfolio successfully

### TestPortfolioSummaryAndHistory (3 tests)
- Get portfolio summary with aggregated metrics
- Verify archived portfolios excluded from summary
- Get portfolio historical performance data
  - Note (v1.4.1+): Returns camelCase field names (e.g., `totalValue`, `totalRealizedGainLoss`)

### TestPortfolioFunds (7 tests)
- List all portfolio funds
- List portfolio funds filtered by portfolio_id
- Create portfolio-fund relationship
- Delete portfolio fund without transactions
- Delete portfolio fund with transactions requires confirmation
- Verify confirmation flag bypasses transaction check
- Get fund-specific historical data for portfolio

### TestPortfolioErrors (8 tests)
- Create portfolio service error returns 500
- Get portfolio not found returns 404
- Update portfolio service error returns 500
- Delete portfolio service error returns 500
- Archive portfolio not found returns 404
- Unarchive portfolio not found returns 404
- Create portfolio fund duplicate error returns 409
- Delete portfolio fund requires confirmation returns 409

---

## Helper Functions

### `create_fund()`
Helper to create Fund instances with all required fields (isin, symbol, name, currency, exchange). Ensures consistency across tests and handles required field validation.

---

## Key Test Patterns

- **Empty State Testing**: `db_session` fixture triggers database cleanup to ensure true empty state
- **Full CRUD Verification**: Tests verify both HTTP response and database state changes
- **Archived Portfolio Behavior**: Archived portfolios return 404 on detail endpoint, excluded from list/summary
- **Confirmation Flow**: Portfolio fund deletion with transactions requires `confirm=true` query parameter
- **Error Path Testing**: Mocks simulate service failures to verify error handlers return appropriate status codes
- **SQLAlchemy 2.0 Patterns**: Uses `db.session.get(Model, id)` instead of deprecated `Model.query.get(id)`
- **Historical Data**: FundPrice model uses `price` field (not `close`)
- **Response Flexibility**: Verifies status codes and key fields, not exact naming conventions
- **API Response Format (v1.4.1+)**: Portfolio history endpoint returns camelCase field names (internal Python uses snake_case, conversion happens at API boundary)

---

## Phase 1b: Query API Deprecation Fixes

Fixed deprecated SQLAlchemy Query API usages in `portfolio_routes.py`:

**Routes Updated**:
1. List portfolios: Changed to `PortfolioService.get_all_portfolios()`
2. Portfolio fund deletion: Changed to `PortfolioService.get_portfolio_fund()` with eager loading
3. Transaction/dividend counting: Changed to service methods using `func.count()`

**Service Methods Added**:
- `get_all_portfolios()` - Retrieve all portfolios without filtering
- `get_portfolio_fund(id, with_relationships)` - Get PortfolioFund with optional eager loading
- `count_portfolio_fund_transactions(id)` - Count transactions for portfolio fund
- `count_portfolio_fund_dividends(id)` - Count dividends for portfolio fund

**Test Isolation Fix**: Added `db_session` fixture to `test_list_portfolios_empty` to prevent data leakage.

---

## Phase 4a: Error Path Testing

Added TestPortfolioErrors class with 8 error path tests to achieve 100% coverage. Also removed dead code (duplicate route registration at lines 394-400).

**Coverage Improvement**: 89% → 100% (all exception handlers + dead code removal)

**Testing Approach**: Uses `unittest.mock.patch` to simulate service layer failures and verify error handling returns appropriate status codes and error messages.

---

## Running Tests

### Run all portfolio route tests:
```bash
pytest tests/api/test_portfolio_routes.py -v
```

### Run specific test class:
```bash
pytest tests/api/test_portfolio_routes.py::TestPortfolioListAndCreate -v
```

### Run specific test:
```bash
pytest tests/api/test_portfolio_routes.py::TestPortfolioListAndCreate::test_create_portfolio -v
```

### Run without coverage (faster):
```bash
pytest tests/api/test_portfolio_routes.py -v --no-cov
```

---

## Related Documentation

- **Service Tests**: `tests/docs/services/PORTFOLIO_SERVICE_TESTS.md`
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/portfolio_routes.py`
- **Portfolio Service**: `app/services/portfolio_service.py`

---

**Last Updated**: 2026-01-13 (Version 1.4.1)
**Maintained By**: @ndewijer
