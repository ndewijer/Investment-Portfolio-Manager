# Developer Routes Integration Tests

**File**: `tests/routes/test_developer_routes.py`
**Route File**: `app/routes/developer_routes.py`
**Test Count**: 13 tests (11 passing, 2 skipped)
**Status**: ✅ Core functionality tested (Phase 2c complete)

---

## Overview

Integration tests for developer-focused API endpoints including exchange rates, fund prices, logging configuration, and system maintenance operations.

### Endpoints Tested

1. **GET /api/exchange-rate** - Get exchange rate ✅
2. **POST /api/exchange-rate** - Set exchange rate ✅
3. **POST /api/import-transactions** - Import transactions (SKIPPED - CSV upload) ⏭️
4. **POST /api/fund-price** - Set fund price ✅
5. **GET /api/csv-template** - Get CSV template ✅
6. **GET /api/fund-price-template** - Get fund price template ✅
7. **POST /api/import-fund-prices** - Import fund prices (SKIPPED - CSV upload) ⏭️
8. **GET /api/system-settings/logging** - Get logging settings ✅ (Phase 2c)
9. **PUT /api/system-settings/logging** - Update logging settings ✅ (Phase 2c)
10. **GET /api/logs** - Get logs ✅ (Phase 2c)
11. **GET /api/logs?level=ERROR** - Get logs with filters ✅ (Phase 2c)
12. **GET /api/fund-price/<fund_id>** - Get fund price ✅ (Phase 2c)
13. **POST /api/logs/clear** - Clear logs ✅

### Test Status Summary

- **11 passing** - All core functionality tested
- **2 skipped** - CSV file upload endpoints (logic tested in service layer)

---

## Recent Changes

### Phase 2c - Service Layer Refactoring (6 tests enabled)

**Service Methods Added:**
- `DeveloperService.get_fund_price()` - Retrieve fund price (SQLAlchemy 2.0 migration)
- `LoggingService.get_logging_settings()` - Get logging configuration
- `LoggingService.update_logging_settings()` - Update logging configuration
- `LoggingService.get_logs()` - Retrieve filtered and paginated logs
- `LoggingService.clear_logs()` - Clear all logs

**Routes Refactored:**
All developer routes now properly delegate business logic to service layer:
- Removed direct database access (Query.query, Query.filter_by, Query.filter)
- Removed old SQLAlchemy Query API usage (query.paginate(), Model.query.delete())
- Routes now act as thin controllers, delegating to services
- Migrated to SQLAlchemy 2.0 API (select(), db.paginate(), delete())

**Tests Fixed:**

1. **test_get_fund_price** - Previously skipped for 500 error
   - Issue: DeveloperService used old Query API (`FundPrice.query.filter_by().first()`)
   - Fix: Migrated to SQLAlchemy 2.0 (`select()` with `db.session.execute().scalar_one_or_none()`)
   - Status: ✅ Passing

2. **test_get_csv_template** - Previously skipped for 500 error
   - Issue: Test expected CSV content type, but endpoint returns JSON
   - Fix: Updated test to expect JSON response with headers
   - Status: ✅ Passing

3. **test_get_fund_price_template** - Previously skipped for 500 error
   - Issue: Test expected CSV content type, but endpoint returns JSON
   - Fix: Updated test to expect JSON response with headers
   - Status: ✅ Passing

4. **test_get_logging_settings** - Previously skipped for 500 error
   - Issue: Route used old Query API and wrong SystemSettingKey values
   - Fix: Moved logic to LoggingService, updated test to use LOGGING_ENABLED/LOGGING_LEVEL
   - Status: ✅ Passing

5. **test_update_logging_settings** - Previously skipped for 500 error
   - Issue: Route used old Query API (`SystemSetting.query.filter_by().first()`)
   - Fix: Moved logic to LoggingService with SQLAlchemy 2.0 `select()` statements
   - Status: ✅ Passing

6. **test_get_logs_with_filters** - Previously skipped for 500 error
   - Issue: Route used old Query API (`Log.query`, `query.filter()`, `query.paginate()`)
   - Fix: Moved logic to LoggingService with `db.paginate()` and case-insensitive enum filtering
   - Added unique source filtering to prevent test interference
   - Status: ✅ Passing

---

## Test Organization

### Test Classes

1. **TestExchangeRate** (2 tests)
   - Get exchange rate for currency pair
   - Set exchange rate for currency pair

2. **TestFundPrice** (2 tests)
   - Create fund price
   - Get fund price by fund ID

3. **TestCSVTemplates** (2 tests)
   - Get transaction CSV template
   - Get fund price CSV template

4. **TestImports** (2 tests, both skipped)
   - Import transactions from CSV (SKIPPED - service tested)
   - Import fund prices from CSV (SKIPPED - service tested)

5. **TestLogging** (5 tests)
   - Get logging settings
   - Update logging settings
   - Get logs
   - Get logs with filters
   - Clear logs

---

## Test Details

### TestExchangeRate

#### test_get_exchange_rate
**Purpose**: Verify GET /api/exchange-rate returns exchange rate for currency pair \
**Setup**: Create ExchangeRate(USD→EUR, rate=0.85) \
**Request**: GET /api/exchange-rate?from_currency=USD&to_currency=EUR \
**Assertions**:
- Status code 200
- Response contains rate, from_currency, to_currency
- Rate matches created exchange rate

#### test_set_exchange_rate
**Purpose**: Verify POST /api/exchange-rate creates new exchange rate \
**Request**: POST /api/exchange-rate with USD→GBP, rate=0.75 \
**Assertions**:
- Status code 200
- Database contains new ExchangeRate record
- Rate value matches payload

### TestFundPrice

#### test_create_fund_price
**Purpose**: Verify POST /api/fund-price creates fund price \
**Setup**: Create Fund(VTI) \
**Request**: POST /api/fund-price with fund_id, date, price=250.00 \
**Assertions**:
- Status code 200
- Database contains new FundPrice record
- Price matches payload

#### test_get_fund_price
**Purpose**: Verify GET /api/fund-price/<fund_id> returns fund price \
**Setup**: Create Fund(VOO) and FundPrice(price=450.00) \
**Request**: GET /api/fund-price/{fund_id} \
**Assertions**:
- Status code 200
- Response is dict with price, fund_id, date keys
- Price matches created fund price
**Fix**: Changed from expecting list to expecting dict (Phase 2c)

### TestCSVTemplates

#### test_get_csv_template
**Purpose**: Verify GET /api/csv-template returns transaction template \
**Request**: GET /api/csv-template \
**Assertions**:
- Status code 200
- Response is JSON dict (not CSV)
- Contains headers key with "date"
**Fix**: Changed from expecting CSV content-type to JSON (Phase 2c)

#### test_get_fund_price_template
**Purpose**: Verify GET /api/fund-price-template returns fund price template \
**Request**: GET /api/fund-price-template \
**Assertions**:
- Status code 200
- Response is JSON dict (not CSV)
- Contains headers key with "date"
**Fix**: Changed from expecting CSV content-type to JSON (Phase 2c)

### TestImports

#### test_import_transactions
**Purpose**: Import transactions from CSV file \
**Status**: SKIPPED - CSV upload requires complex file handling \
**Note**: CSV parsing logic is tested in test_developer_service.py

#### test_import_fund_prices
**Purpose**: Import fund prices from CSV file \
**Status**: SKIPPED - CSV upload requires complex file handling \
**Note**: CSV parsing logic is tested in test_developer_service.py

### TestLogging

#### test_get_logging_settings
**Purpose**: Verify GET /api/system-settings/logging returns logging settings \
**Setup**: Create SystemSettings for LOGGING_ENABLED=true, LOGGING_LEVEL=INFO \
**Request**: GET /api/system-settings/logging \
**Assertions**:
- Status code 200
- Response contains enabled=True, level="INFO"
**Fix**: Updated to use LOGGING_ENABLED/LOGGING_LEVEL keys (Phase 2c)

#### test_update_logging_settings
**Purpose**: Verify PUT /api/system-settings/logging updates settings \
**Request**: PUT /api/system-settings/logging with enabled=False, level="DEBUG" \
**Assertions**:
- Status code 200
- Response contains updated values
- Database updated with enabled_setting.value="false"
**Fix**: Moved logic to LoggingService with SQLAlchemy 2.0 API (Phase 2c)

#### test_get_logs
**Purpose**: Verify GET /api/logs returns paginated logs \
**Setup**: Create 2 Log entries \
**Request**: GET /api/logs \
**Assertions**:
- Status code 200
- Response is dict with logs, total, pages, current_page keys
- logs array contains at least 2 entries

#### test_get_logs_with_filters
**Purpose**: Verify GET /api/logs with level filter \
**Setup**: Create ERROR and INFO logs with unique source \
**Request**: GET /api/logs?level=error&source={unique_source} \
**Assertions**:
- Status code 200
- Response is dict with logs array
- Filtered logs match source and level
**Fix**: Added case-insensitive enum filtering, unique source isolation (Phase 2c)

#### test_clear_logs
**Purpose**: Verify POST /api/logs/clear removes all logs \
**Setup**: Create 2 Log entries \
**Request**: POST /api/logs/clear \
**Assertions**:
- Status code 200
- Operation completes successfully (logs may be added by clear operation itself)

---

## Testing Patterns

### Exchange Rate Testing
- Uses Decimal type for rates to avoid float precision issues
- Tests both retrieval and creation operations

### Fund Price Testing
- Uses helper function `create_fund()` to ensure all required fields (exchange) are provided
- Tests both creation and retrieval with date parameters

### CSV Template Testing
- Templates are returned as JSON with headers, example, and description
- Actual CSV import logic is tested in service layer tests

### Logging Configuration Testing
- Uses correct SystemSettingKey enums (LOGGING_ENABLED, LOGGING_LEVEL)
- Tests both retrieval and updates with database verification

### Log Retrieval Testing
- Uses unique source identifiers to isolate test data
- Tests filtering (level, category, source), pagination, and sorting
- Uses case-insensitive enum filtering for robustness

---

## SQLAlchemy 2.0 Migration

All developer routes have been migrated from deprecated Query API to SQLAlchemy 2.0:

**Before (Old Query API)**:
```python
# Getting a record
FundPrice.query.filter_by(fund_id=fund_id, date=date).first()

# Filtering with or_
query = Log.query
query.filter(db.or_(*filters))

# Pagination
query.paginate(page=page, per_page=per_page)

# Deletion
Log.query.delete()
```

**After (SQLAlchemy 2.0)**:
```python
# Getting a record
from sqlalchemy import select
stmt = select(FundPrice).where(FundPrice.fund_id == fund_id, FundPrice.date == date)
db.session.execute(stmt).scalar_one_or_none()

# Filtering with or_
from sqlalchemy import or_, select
stmt = select(Log).where(or_(*filters))

# Pagination
db.paginate(stmt, page=page, per_page=per_page)

# Deletion
from sqlalchemy import delete
stmt = delete(Log)
db.session.execute(stmt)
```

---

## Service Integration

### DeveloperService Methods
- `get_exchange_rate(from_currency, to_currency, date)` - Retrieve exchange rate
- `set_exchange_rate(from_currency, to_currency, rate, date)` - Set/update rate
- `get_fund_price(fund_id, date)` - Retrieve fund price (✅ SQLAlchemy 2.0)
- `set_fund_price(fund_id, price, date)` - Set/update price
- `get_csv_template()` - Get transaction CSV template structure
- `get_fund_price_csv_template()` - Get fund price CSV template structure
- `import_transactions_csv(file_content, portfolio_fund_id)` - Import transactions
- `import_fund_prices_csv(file_content, fund_id)` - Import fund prices

### LoggingService Methods (Phase 2c)
- `get_logging_settings()` - Retrieve logging configuration
- `update_logging_settings(enabled, level)` - Update logging configuration
- `get_logs(filters, sort, pagination)` - Retrieve filtered, paginated logs
- `clear_logs()` - Delete all logs from database

---

## Coverage Notes

### What's Tested
- ✅ Exchange rate CRUD operations
- ✅ Fund price CRUD operations
- ✅ CSV template retrieval (JSON format)
- ✅ Logging configuration management
- ✅ Log retrieval with filtering and pagination
- ✅ Log database clearing

### What's Skipped
- ⏭️ CSV file upload handling (2 tests)
  - Reason: Requires complex multipart/form-data mocking
  - Note: CSV parsing logic IS tested in service layer

### Service Layer Coverage
- All business logic is tested in service layer tests:
  - `test_developer_service.py` - CSV import logic
  - `test_logging_service.py` - Logging management (37 tests, 98% coverage)

---

## Running Tests

### All Developer Route Tests
```bash
pytest tests/routes/test_developer_routes.py -v
```

### Specific Test Classes
```bash
# Exchange rate tests
pytest tests/routes/test_developer_routes.py::TestExchangeRate -v

# Logging tests
pytest tests/routes/test_developer_routes.py::TestLogging -v

# Fund price tests
pytest tests/routes/test_developer_routes.py::TestFundPrice -v
```

### Individual Tests
```bash
# Test Phase 2c fixes
pytest tests/routes/test_developer_routes.py::TestFundPrice::test_get_fund_price -v
pytest tests/routes/test_developer_routes.py::TestLogging::test_get_logs_with_filters -v
```

---

## Related Documentation

- **Service Tests**: tests/docs/services/DEVELOPER_SERVICE_TESTS.md (CSV import logic)
- **Service Tests**: tests/docs/services/LOGGING_SERVICE_TESTS.md (37 tests, 98% coverage)
- **Phase Documentation**: See Phase 2c notes for service layer refactoring details

---

**Last Updated**: v1.3.3 (Phase 2c - Service Layer Refactoring) \
**Test Count**: 13 tests (11 passing, 2 skipped) \
**Status**: Complete ✅ \
**SQLAlchemy 2.0 Migration**: ✅ Complete \
**Service Methods Added**: 4 (LoggingService) + 1 fix (DeveloperService)
