# Developer Routes Integration Tests

**File**: `tests/routes/test_developer_routes.py`
**Route File**: `app/routes/developer_routes.py`
**Test Count**: 13 tests (11 passing, 2 skipped)
**Coverage**: Core functionality tested (Phase 2c complete)
**Status**: ✅ All core tests passing

> **💡 Detailed Test Information**: For detailed explanations of each test including
> WHY it exists and what business logic it validates, see the docstrings in the test file.
> Your IDE will show these when hovering over test names.

---

## Overview

Integration tests for developer-focused API endpoints including exchange rates, fund prices, logging configuration, and system maintenance operations. All routes delegate business logic to DeveloperService and LoggingService following the service layer architecture pattern.

### Endpoints Tested

- **GET /api/exchange-rate** - Get exchange rate for currency pair
- **POST /api/exchange-rate** - Set exchange rate for currency pair
- **POST /api/import-transactions** - Import transactions from CSV (SKIPPED - service tested)
- **POST /api/fund-price** - Set fund price for fund
- **GET /api/csv-template** - Get transaction CSV template
- **GET /api/fund-price-template** - Get fund price CSV template
- **POST /api/import-fund-prices** - Import fund prices from CSV (SKIPPED - service tested)
- **GET /api/system-settings/logging** - Get logging configuration
- **PUT /api/system-settings/logging** - Update logging configuration
- **GET /api/logs** - Get paginated and filtered logs
- **GET /api/logs?level=ERROR** - Get logs with level filter
- **GET /api/fund-price/<fund_id>** - Get fund price by fund ID
- **POST /api/logs/clear** - Clear all logs from database

---

## Test Organization

### TestExchangeRate (2 tests)
- `test_get_exchange_rate` - Verifies GET returns exchange rate for currency pair
- `test_set_exchange_rate` - Verifies POST creates new exchange rate

### TestFundPrice (2 tests)
- `test_create_fund_price` - Verifies POST creates fund price record
- `test_get_fund_price` - Verifies GET returns fund price by fund ID

### TestCSVTemplates (2 tests)
- `test_get_csv_template` - Verifies GET returns transaction template as JSON
- `test_get_fund_price_template` - Verifies GET returns fund price template as JSON

### TestImports (2 tests, both skipped)
- `test_import_transactions` - CSV upload requires complex file handling (service tested)
- `test_import_fund_prices` - CSV upload requires complex file handling (service tested)

### TestLogging (5 tests)
- `test_get_logging_settings` - Verifies GET returns LOGGING_ENABLED and LOGGING_LEVEL
- `test_update_logging_settings` - Verifies PUT updates logging configuration
- `test_get_logs` - Verifies GET returns paginated logs
- `test_get_logs_with_filters` - Verifies level/source filtering with case-insensitive enum handling
- `test_clear_logs` - Verifies POST clears all logs from database

---

## Key Patterns

**Service Layer Delegation**: All routes delegate to DeveloperService and LoggingService following the thin controller pattern. Routes handle HTTP concerns only (request/response), services handle business logic.

**SQLAlchemy 2.0 Migration**: All routes migrated from deprecated Query API (`Model.query.filter_by()`, `query.paginate()`) to SQLAlchemy 2.0 (`select()`, `db.paginate()`, `db.session.execute()`).

**Exchange Rate & Fund Price Testing**: Uses Decimal type for rates to avoid float precision issues. Uses `create_fund()` helper to ensure required fields (exchange) are provided.

**CSV Template Testing**: Templates returned as JSON with headers, examples, and descriptions. Actual CSV import logic tested in service layer tests, not route tests.

**Logging Configuration Testing**: Uses correct SystemSettingKey enums (LOGGING_ENABLED, LOGGING_LEVEL). Tests both retrieval and updates with database verification.

**Log Retrieval Testing**: Uses unique source identifiers to isolate test data. Tests filtering (level, category, source), pagination, and sorting with case-insensitive enum filtering.

**Skipped Tests**: CSV file upload endpoints (2 tests) skipped as they require complex multipart/form-data mocking. CSV parsing logic is fully tested in service layer tests.

---

## Test Status Summary

- **11 passing tests**: All core functionality tested including:
  - Exchange rate CRUD operations
  - Fund price CRUD operations
  - CSV template retrieval (JSON format)
  - Logging configuration management
  - Log retrieval with filtering and pagination
  - Log database clearing

- **2 skipped tests**: CSV file upload handling
  - Reason: Requires complex multipart/form-data mocking
  - Note: CSV parsing logic IS tested in `test_developer_service.py`

---

## Service Methods Used

**DeveloperService**:
- `get_exchange_rate()`, `set_exchange_rate()` - Exchange rate management
- `get_fund_price()`, `set_fund_price()` - Fund price management (SQLAlchemy 2.0)
- `get_csv_template()`, `get_fund_price_csv_template()` - Template retrieval
- `import_transactions_csv()`, `import_fund_prices_csv()` - CSV imports (tested in service layer)

**LoggingService** (Phase 2c):
- `get_logging_settings()`, `update_logging_settings()` - Logging configuration
- `get_logs()` - Filtered and paginated log retrieval
- `clear_logs()` - Database log clearing

---

## Running Tests

```bash
# Run all developer route tests
pytest tests/routes/test_developer_routes.py -v

# Run specific test class
pytest tests/routes/test_developer_routes.py::TestExchangeRate -v
pytest tests/routes/test_developer_routes.py::TestLogging -v
pytest tests/routes/test_developer_routes.py::TestFundPrice -v

# Run individual test
pytest tests/routes/test_developer_routes.py::TestFundPrice::test_get_fund_price -v
pytest tests/routes/test_developer_routes.py::TestLogging::test_get_logs_with_filters -v

# Run without coverage (faster)
pytest tests/routes/test_developer_routes.py -v --no-cov
```

---

## Related Documentation

- **Service Tests**: `tests/docs/services/DEVELOPER_SERVICE_TESTS.md` (CSV import logic)
- **Service Tests**: `tests/docs/services/LOGGING_SERVICE_TESTS.md` (37 tests, 98% coverage)
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **Developer Service**: `app/services/developer_service.py`
- **Logging Service**: `app/services/logging_service.py`

---

**Last Updated**: Phase 5 (Route Integration Tests) + Documentation Condensing
**SQLAlchemy 2.0 Migration**: ✅ Complete
**Service Methods Added**: 4 (LoggingService) + 1 fix (DeveloperService)
