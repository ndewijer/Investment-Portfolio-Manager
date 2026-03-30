# Developer Routes Integration Tests

**File**: `tests/api/test_developer_routes.py`
**Route File**: `app/routes/developer_routes.py`
**Test Count**: 51 tests (45 passing, 6 skipped)
**Coverage**: Core functionality tested (Phase 2c complete)
**Status**: All core tests passing

> **Detailed Test Information**: For detailed explanations of each test including
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
- **GET /api/logs?skip=N&perPage=N** - Get logs with skip-based pagination and overshoot protection
- **GET /api/developer/fund-price/<fund_id>** - Get fund price by fund ID
- **POST /api/logs/clear** - Clear all logs from database
- **GET /api/developer/logs/filter-options** - Get distinct filter values for log picklists

---

## Test Organization

### TestExchangeRate (2 tests)
- `test_get_exchange_rate` - Verifies GET returns exchange rate for currency pair
- `test_set_exchange_rate` - Verifies POST creates new exchange rate

### TestFundPrice (2 tests)
- `test_create_fund_price` - Verifies POST creates fund price record
- `test_get_fund_price` - Verifies GET returns fund price by fund ID

### TestCSVTemplates (4 tests)
- `test_get_csv_template` - Verifies GET returns transaction template as JSON
- `test_get_fund_price_template` - Verifies GET returns fund price template as JSON
- `test_get_csv_template_service_error` - Verifies service errors return 500
- `test_get_fund_price_template_service_error` - Verifies service errors return 500

### TestImports (2 tests, both skipped)
- `test_import_transactions` - CSV upload requires complex file handling (service tested)
- `test_import_fund_prices` - CSV upload requires complex file handling (service tested)

### TestExchangeRateErrors (8 tests)
- `test_set_exchange_rate_missing_from_currency` - Rejects missing from_currency
- `test_set_exchange_rate_missing_to_currency` - Rejects missing to_currency
- `test_set_exchange_rate_missing_rate` - Rejects missing rate
- `test_set_exchange_rate_invalid_from_currency` - Rejects invalid currency code
- `test_set_exchange_rate_invalid_date_format` - Rejects invalid date format
- `test_set_exchange_rate_database_error` - Handles database errors gracefully
- `test_get_exchange_rate_invalid_date_format` - Rejects invalid date format on GET
- `test_get_exchange_rate_service_error` - Handles service errors gracefully

### TestFundPriceErrors (7 tests, 3 skipped)
- `test_set_fund_price_missing_fund_id` - Rejects missing fund_id
- `test_set_fund_price_missing_price` - Rejects missing price
- `test_set_fund_price_invalid_fund_id` - Rejects invalid fund_id
- `test_set_fund_price_invalid_date_format` - Rejects invalid date format
- `test_set_fund_price_service_error` - Handles service errors gracefully
- `test_get_fund_price_not_found` - Returns 404 when price not found (SKIPPED)
- `test_get_fund_price_invalid_date_format` - Rejects invalid date on GET (SKIPPED)
- `test_get_fund_price_service_error` - Handles GET service errors (SKIPPED)

### TestCSVImportErrors (10 tests)
- `test_import_transactions_no_file` - Rejects request without file
- `test_import_transactions_missing_fund_id` - Rejects missing portfolio_fund_id
- `test_import_transactions_invalid_file_format` - Rejects non-CSV files
- `test_import_transactions_invalid_csv_headers` - Rejects CSV with wrong headers
- `test_import_transactions_invalid_portfolio_fund_id` - Rejects invalid portfolio_fund_id
- `test_import_fund_prices_no_file` - Rejects request without file
- `test_import_fund_prices_missing_fund_id` - Rejects missing fund_id
- `test_import_fund_prices_invalid_file_format` - Rejects non-CSV files
- `test_import_fund_prices_invalid_csv_headers` - Rejects CSV with wrong headers
- `test_import_fund_prices_wrong_file_type` - Rejects transaction CSV on price endpoint

### TestLoggingErrors (6 tests)
- `test_update_logging_settings_missing_enabled` - Rejects missing enabled field
- `test_update_logging_settings_missing_level` - Rejects missing level field
- `test_get_logging_settings_service_error` - Handles GET service errors gracefully
- `test_update_logging_settings_service_error` - Handles PUT service errors gracefully
- `test_get_logs_service_error` - Handles log retrieval service errors
- `test_clear_logs_service_error` - Handles log clearing service errors

### TestLogging (9 tests)
- `test_get_logging_settings` - Verifies GET returns LOGGING_ENABLED and LOGGING_LEVEL
- `test_update_logging_settings` - Verifies PUT updates logging configuration
- `test_get_logs` - Verifies GET returns paginated logs
- `test_get_logs_with_filters` - Verifies level/source filtering with case-insensitive enum handling
- `test_skip_returns_correct_results` - Verifies skip=3&perPage=3 returns correct 3 results from 10 entries
- `test_skip_overshoot_returns_last_page` - Verifies skip beyond total returns last page (not empty)
- `test_skip_overshoot_with_fewer_than_perpage` - Verifies overshoot when total < perPage returns all results
- `test_skip_zero_behaves_normally` - Verifies skip=0 is equivalent to omitting skip
- `test_clear_logs` - Verifies POST clears all logs from database

### TestLogFilterOptions (6 tests)
- `test_filter_options_empty_when_no_logs` - Returns empty arrays when no logs exist
- `test_filter_options_returns_distinct_values` - Returns unique values for each filter field
- `test_filter_options_values_sorted_alphabetically` - Returns alphabetically sorted arrays
- `test_filter_options_levels_and_categories_uppercase` - Returns uppercase levels and categories
- `test_filter_options_excludes_null_values` - Excludes null values from filter results
- `test_filter_options_service_error` - Handles service errors gracefully

---

## Key Patterns

**Service Layer Delegation**: All routes delegate to DeveloperService and LoggingService following the thin controller pattern. Routes handle HTTP concerns only (request/response), services handle business logic.

**SQLAlchemy 2.0 Migration**: All routes migrated from deprecated Query API (`Model.query.filter_by()`, `query.paginate()`) to SQLAlchemy 2.0 (`select()`, `db.paginate()`, `db.session.execute()`).

**Exchange Rate & Fund Price Testing**: Uses Decimal type for rates to avoid float precision issues. Uses `create_fund()` helper to ensure required fields (exchange) are provided.

**CSV Template Testing**: Templates returned as JSON with headers, examples, and descriptions. Actual CSV import logic tested in service layer tests, not route tests.

**Logging Configuration Testing**: Uses correct SystemSettingKey enums (LOGGING_ENABLED, LOGGING_LEVEL). Tests both retrieval and updates with database verification.

**Log Retrieval Testing**: Uses unique source identifiers to isolate test data. Tests filtering (level, category, source), pagination, skip-based navigation with overshoot protection, and sorting with case-insensitive enum filtering.

**Skip Overshoot Protection Testing**: Verifies that when skip exceeds total results, the backend returns the last available page instead of an empty result set. Tests cover normal skip, overshoot with multi-page data, overshoot with sub-page data, and skip=0 equivalence.

**Skipped Tests**: CSV file upload endpoints (2 tests) and disabled fund-price GET endpoints (3 tests) skipped. CSV parsing logic is fully tested in service layer tests. Fund-price GET endpoint disabled as duplicate.

---

## Test Status Summary

- **45 passing tests**: All core functionality tested including:
  - Exchange rate CRUD operations
  - Fund price CRUD operations
  - CSV template retrieval (JSON format)
  - Logging configuration management
  - Log retrieval with filtering, pagination, and skip overshoot protection
  - Log database clearing
  - Log filter options retrieval
  - Error handling for all endpoints

- **6 skipped tests**: CSV file upload handling (2) and disabled fund-price GET (3) + disabled fund-price GET endpoint (1)
  - Reason: CSV requires complex multipart/form-data mocking; fund-price GET is disabled duplicate
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
- `get_logs()` - Filtered and paginated log retrieval with skip overshoot protection
- `clear_logs()` - Database log clearing
- `get_log_filter_options()` - Distinct filter values for log picklists

---

## Running Tests

```bash
# Run all developer route tests
uv run pytest tests/api/test_developer_routes.py -v

# Run specific test class
uv run pytest tests/api/test_developer_routes.py::TestExchangeRate -v
uv run pytest tests/api/test_developer_routes.py::TestLogging -v
uv run pytest tests/api/test_developer_routes.py::TestFundPrice -v
uv run pytest tests/api/test_developer_routes.py::TestLogFilterOptions -v

# Run individual test
uv run pytest tests/api/test_developer_routes.py::TestFundPrice::test_get_fund_price -v
uv run pytest tests/api/test_developer_routes.py::TestLogging::test_get_logs_with_filters -v
uv run pytest tests/api/test_developer_routes.py::TestLogging::test_skip_overshoot_returns_last_page -v

# Run without coverage (faster)
uv run pytest tests/api/test_developer_routes.py -v --no-cov
```

---

## Related Documentation

- **Service Tests**: `tests/docs/services/DEVELOPER_SERVICE_TESTS.md` (CSV import logic)
- **Service Tests**: `tests/docs/services/LOGGING_SERVICE_TESTS.md` (37 tests, 98% coverage)
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **Developer Service**: `app/services/developer_service.py`
- **Logging Service**: `app/services/logging_service.py`

---

**Last Updated**: 2026-03-30 (Version 2.0.0)
**Maintained By**: @ndewijer
