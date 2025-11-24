# DeveloperService Test Documentation

**Service**: `app/services/developer_service.py` \
**Test File**: `tests/services/test_developer_service.py` \
**Coverage**: 99% (59 tests) \
**Status**: ✅ Complete

---

## Overview

The **DeveloperService** provides developer-related operations including data import/export, system maintenance functions, and data sanitization utilities for CSV processing and system administration.

### What This Service Does

1. **Data Sanitization**: String, float, and date input cleaning and validation
2. **Exchange Rate Management**: Set, update, and retrieve currency exchange rates
3. **Database Queries**: Retrieve funds and portfolios for administrative operations
4. **CSV Templates**: Provide template information for data imports
5. **CSV Header Validation**: Centralized UTF-8 encoding and header validation for CSV imports
6. **CSV Processing**: UTF-8/BOM handling, field mapping, and validation
7. **Data Import**: Transaction and fund price CSV imports with error handling
8. **Fund Price Management**: Set, update, and retrieve historical fund prices

### Test Suite Scope

**Full coverage testing of**:
- Data sanitization methods (13 tests)
- Exchange rate management (8 tests)
- Database query operations (2 tests)
- CSV template generation (2 tests)
- CSV header validation (12 tests)
- CSV processing utilities (8 tests)
- Transaction import functionality (4 tests)
- Fund price management (5 tests)
- Fund price import functionality (3 tests)

**No bugs discovered**: Clean, well-implemented service with robust error handling.

---

## Test Organization

### Test Class Structure

```python
class TestSanitizationMethods:         # 13 tests - Data cleaning utilities
class TestExchangeRateManagement:      # 8 tests - Currency rate operations
class TestDatabaseQueries:             # 2 tests - Fund/portfolio retrieval
class TestCSVTemplates:                # 2 tests - Template generation
class TestCSVValidation:               # 12 tests - CSV header and encoding validation
class TestCSVProcessing:               # 8 tests - CSV parsing and validation
class TestTransactionImport:           # 4 tests - Transaction CSV import
class TestFundPriceManagement:         # 5 tests - Price data operations
class TestFundPriceImport:             # 3 tests - Price CSV import
```

### Testing Approach

**Unique Identifier Pattern**: Each test uses unique UUIDs to prevent database conflicts.

**Example**:
```python
# Unique ISIN for each test
unique_isin = f"US{uuid.uuid4().hex[:10].upper()}"
fund = Fund(
    isin=unique_isin,
    # ... other fields
)
```

**Why this pattern**: Prevents UNIQUE constraint violations when multiple tests create similar data.

**UTF-8 and BOM Testing**: Comprehensive encoding validation including Byte Order Mark handling.

---

## Service-Specific Patterns

### 1. Data Sanitization Testing

**Focus**: Input validation and cleaning for CSV data processing

**Pattern**: Test all data types with edge cases
```python
def test_sanitize_float_invalid_format(self, app_context):
    """Test sanitizing invalid float format raises ValueError."""
    with pytest.raises(ValueError, match="Invalid number format: abc123"):
        DeveloperService.sanitize_float("abc123")
```

### 2. CSV Processing with UTF-8/BOM Support

**Pattern**: Test both UTF-8 and UTF-8-BOM encoded content
```python
def test_process_csv_content_with_bom(self, app_context):
    """Test processing CSV content with BOM (Byte Order Mark)."""
    csv_content = "date,price\n2024-03-15,150.75"
    file_content = csv_content.encode("utf-8-sig")  # Includes BOM

    # Process and verify BOM is handled correctly
```

**Why this approach**:
- Real-world CSV files often have BOM from Excel exports
- UTF-8-sig encoding automatically handles BOM removal
- Prevents parsing errors from invisible BOM characters

### 3. Database Integration Testing

**Pattern**: Create complete object graphs for realistic testing
```python
# Create full fund/portfolio/portfolio_fund relationship
fund = Fund(...)
portfolio = Portfolio(...)
portfolio_fund = PortfolioFund(
    portfolio_id=portfolio.id,
    fund_id=fund.id
)
```

### 4. Error Handling and Mocking

**Pattern**: Mock database failures to test error handling
```python
with patch.object(db.session, 'commit', side_effect=Exception("Database error")):
    with pytest.raises(ValueError, match="Database error while saving"):
        DeveloperService.import_transactions_csv(file_content, portfolio_fund.id)
```

---

## Test Suite Walkthrough

### TestSanitizationMethods (13 tests)

Data cleaning and validation utilities.

#### Test 1: `test_sanitize_string_normal_input`
**Purpose**: Verify string sanitization strips whitespace \
**Test Data**: "  hello world  " \
**Expected**: "hello world" \
**Why**: Basic string cleaning for CSV data

#### Test 2: `test_sanitize_string_none_input`
**Purpose**: Verify None input returns None \
**Test Data**: None \
**Expected**: None \
**Why**: Handle optional fields gracefully

#### Test 3: `test_sanitize_string_empty_input`
**Purpose**: Verify empty string handling \
**Test Data**: "   " (spaces only) \
**Expected**: "" (empty string) \
**Why**: Distinguish between None and empty values

#### Test 4: `test_sanitize_string_non_string_input`
**Purpose**: Verify non-string inputs are converted \
**Test Data**: 12345 (integer) \
**Expected**: "12345" \
**Why**: Handle mixed data types in CSV files

#### Test 5: `test_sanitize_float_normal_input`
**Purpose**: Verify float sanitization with whitespace \
**Test Data**: "  123.45  " \
**Expected**: 123.45 \
**Why**: Clean numeric data from CSV

#### Test 6: `test_sanitize_float_integer_input`
**Purpose**: Verify integer strings convert to float \
**Test Data**: "100" \
**Expected**: 100.0 \
**Why**: Consistent numeric type handling

#### Test 7: `test_sanitize_float_none_input`
**Purpose**: Verify None handling for optional numeric fields \
**Test Data**: None \
**Expected**: None \
**Why**: Handle missing numeric data

#### Test 8: `test_sanitize_float_invalid_format`
**Purpose**: Verify invalid format raises ValueError \
**Test Data**: "abc123" \
**Expected**: ValueError with descriptive message \
**Why**: Data validation prevents corruption

#### Test 9: `test_sanitize_float_empty_string`
**Purpose**: Verify empty string handling \
**Test Data**: "" \
**Expected**: ValueError \
**Why**: Empty strings are invalid numeric data

#### Test 10: `test_sanitize_date_normal_input`
**Purpose**: Verify date parsing with whitespace \
**Test Data**: "  2024-03-15  " \
**Expected**: date(2024, 3, 15) \
**Why**: Clean date data from CSV

#### Test 11: `test_sanitize_date_none_input`
**Purpose**: Verify None handling for optional dates \
**Test Data**: None \
**Expected**: None \
**Why**: Handle missing date fields

#### Test 12: `test_sanitize_date_invalid_format`
**Purpose**: Verify invalid date format raises ValueError \
**Test Data**: "15/03/2024" (wrong format) \
**Expected**: ValueError with format guidance \
**Why**: Enforce ISO date format consistency

#### Test 13: `test_sanitize_date_invalid_date`
**Purpose**: Verify impossible dates raise ValueError \
**Test Data**: "2024-13-45" (invalid month/day) \
**Expected**: ValueError \
**Why**: Validate actual date values

### TestExchangeRateManagement (8 tests)

Currency exchange rate operations.

#### Test 14: `test_set_exchange_rate_new_rate`
**Purpose**: Verify creating new exchange rate entry \
**Test Data**: USD to EUR, rate 0.85, current date \
**Expected**: Database entry created, proper response format \
**Why**: Core functionality for currency management

#### Test 15: `test_set_exchange_rate_update_existing`
**Purpose**: Verify updating existing exchange rate \
**Test Data**: Existing USD/EUR rate, update to new value \
**Expected**: Single record updated, not duplicated \
**Why**: Prevent duplicate exchange rate entries

#### Test 16: `test_set_exchange_rate_with_specific_date`
**Purpose**: Verify setting rate for specific historical date \
**Test Data**: Historical date provided \
**Expected**: Rate stored with specified date \
**Why**: Support historical exchange rate management

#### Test 17: `test_set_exchange_rate_validation_empty_currencies`
**Purpose**: Verify validation rejects empty currency codes \
**Test Data**: Empty or None currency codes \
**Expected**: ValueError with clear message \
**Why**: Ensure data integrity

#### Test 18: `test_set_exchange_rate_validation_invalid_rate`
**Purpose**: Verify validation rejects invalid rates \
**Test Data**: Zero and negative rates \
**Expected**: ValueError \
**Why**: Exchange rates must be positive

#### Test 19: `test_get_exchange_rate_found`
**Purpose**: Verify retrieving existing exchange rate \
**Test Data**: Unique currency pair with known rate \
**Expected**: Rate data returned with proper format \
**Why**: Core lookup functionality

#### Test 20: `test_get_exchange_rate_not_found`
**Purpose**: Verify handling of non-existent exchange rate \
**Test Data**: Non-existent currency pair \
**Expected**: None returned \
**Why**: Graceful handling of missing data

### TestDatabaseQueries (2 tests)

Database query operations for administrative functions.

#### Test 21: `test_get_funds`
**Purpose**: Verify retrieving all funds from database \
**Test Data**: Multiple funds with unique ISINs \
**Expected**: All funds returned in query result \
**Why**: Administrative access to fund data

#### Test 22: `test_get_portfolios`
**Purpose**: Verify retrieving all portfolios from database \
**Test Data**: Multiple portfolios \
**Expected**: All portfolios returned in query result \
**Why**: Administrative access to portfolio data

### TestCSVTemplates (2 tests)

CSV template generation for user guidance.

#### Test 23: `test_get_csv_template`
**Purpose**: Verify transaction CSV template generation \
**Test Data**: No input required \
**Expected**: Headers, example, and description provided \
**Why**: User guidance for CSV format

#### Test 24: `test_get_fund_price_csv_template`
**Purpose**: Verify fund price CSV template generation \
**Test Data**: No input required \
**Expected**: Headers, example, and description for price data \
**Why**: User guidance for price import format

### TestCSVProcessing (8 tests)

CSV parsing and validation utilities.

#### Test 25: `test_validate_utf8_valid_content`
**Purpose**: Verify valid UTF-8 content passes validation \
**Test Data**: Valid UTF-8 encoded bytes \
**Expected**: Returns True \
**Why**: Confirm valid files are accepted

#### Test 26: `test_validate_utf8_invalid_content`
**Purpose**: Verify invalid UTF-8 content raises ValueError \
**Test Data**: Invalid byte sequence \
**Expected**: ValueError with encoding message \
**Why**: Prevent processing corrupted files

#### Test 27: `test_process_csv_content_valid_data`
**Purpose**: Verify successful CSV processing \
**Test Data**: Valid CSV with date/price columns \
**Expected**: Parsed data with row numbers \
**Why**: Core CSV processing functionality

#### Test 28: `test_process_csv_content_with_bom`
**Purpose**: Verify UTF-8 BOM (Byte Order Mark) handling \
**Test Data**: CSV encoded with UTF-8-BOM \
**Expected**: BOM stripped, data processed correctly \
**Why**: **Critical for Excel exports** - Excel often adds BOM to UTF-8 files

#### Test 29: `test_process_csv_content_missing_fields`
**Purpose**: Verify validation of required CSV fields \
**Test Data**: CSV missing required 'price' column \
**Expected**: ValueError listing missing fields \
**Why**: Data integrity validation

#### Test 30: `test_process_csv_content_invalid_encoding`
**Purpose**: Verify handling of non-UTF-8 files \
**Test Data**: Invalid UTF-8 byte sequence \
**Expected**: ValueError about encoding \
**Why**: Clear error message for encoding issues

#### Test 31: `test_process_csv_content_no_valid_records`
**Purpose**: Verify handling of empty CSV files \
**Test Data**: CSV with headers but no data rows \
**Expected**: ValueError about no valid records \
**Why**: Prevent processing empty imports

#### Test 32: `test_process_csv_content_row_processing_error`
**Purpose**: Verify handling of row processing errors \
**Test Data**: CSV with invalid price value \
**Expected**: ValueError with specific row number \
**Why**: Help users identify problematic data

### TestTransactionImport (4 tests)

Transaction CSV import functionality.

#### Test 33: `test_import_transactions_csv_success`
**Purpose**: Verify successful transaction import \
**Test Data**: Valid CSV with buy/sell transactions \
**Expected**: Transactions created in database, count returned \
**Why**: Core import functionality

#### Test 34: `test_import_transactions_csv_invalid_portfolio_fund`
**Purpose**: Verify validation of portfolio-fund relationship \
**Test Data**: Non-existent portfolio_fund_id \
**Expected**: ValueError about missing relationship \
**Why**: Data integrity validation

#### Test 35: `test_import_transactions_csv_database_error`
**Purpose**: Verify handling of database failures during import \
**Test Data**: Valid CSV, mocked database error \
**Expected**: ValueError with database error message, rollback \
**Why**: Graceful handling of system failures

#### Test 36: `test_import_transactions_csv_invalid_data_format`
**Purpose**: Verify handling of invalid transaction data \
**Test Data**: CSV with invalid date format \
**Expected**: ValueError with row-specific error \
**Why**: Data validation prevents corruption

### TestFundPriceManagement (5 tests)

Fund price data operations.

#### Test 37: `test_get_fund_price_found`
**Purpose**: Verify retrieving existing fund price \
**Test Data**: Fund with historical price entry \
**Expected**: Price data with proper format \
**Why**: Core price lookup functionality

#### Test 38: `test_get_fund_price_not_found`
**Purpose**: Verify handling of missing price data \
**Test Data**: Non-existent fund/date combination \
**Expected**: None returned \
**Why**: Graceful handling of missing data

#### Test 39: `test_set_fund_price_new_price`
**Purpose**: Verify creating new fund price entry \
**Test Data**: Fund ID, price, specific date \
**Expected**: Database entry created, response data \
**Why**: Core price management functionality

#### Test 40: `test_set_fund_price_update_existing`
**Purpose**: Verify updating existing price entry \
**Test Data**: Existing price entry, new price value \
**Expected**: Single record updated, not duplicated \
**Why**: Prevent duplicate price entries

#### Test 41: `test_set_fund_price_default_date`
**Purpose**: Verify default date handling (today) \
**Test Data**: Fund ID and price, no date specified \
**Expected**: Price entry with current date \
**Why**: Convenient API for current prices

### TestFundPriceImport (3 tests)

Fund price CSV import functionality.

#### Test 42: `test_import_fund_prices_csv_success`
**Purpose**: Verify successful price import \
**Test Data**: Valid CSV with multiple date/price entries \
**Expected**: All prices created in database, count returned \
**Why**: Core price import functionality

#### Test 43: `test_import_fund_prices_csv_database_error`
**Purpose**: Verify handling of database failures during import \
**Test Data**: Valid CSV, mocked database error \
**Expected**: ValueError with database error message \
**Why**: Graceful handling of system failures

#### Test 44: `test_import_fund_prices_csv_invalid_data`
**Purpose**: Verify handling of invalid price data \
**Test Data**: CSV with invalid price format \
**Expected**: ValueError with row-specific error \
**Why**: Data validation prevents corruption

---

## Coverage Analysis

### Coverage Achievement

```
Coverage: 99% (148/149 statements)
```

**Excellent Coverage**: Only 1 line uncovered out of 149 total statements.

### What's Covered

1. **Data Sanitization**:
   - String, float, and date cleaning
   - None value handling
   - Input validation and error messages
   - Type conversion edge cases

2. **Exchange Rate Management**:
   - Creating and updating rates
   - Historical date handling
   - Validation of currency codes and rates
   - Database lookup operations

3. **CSV Processing**:
   - UTF-8 and UTF-8-BOM encoding support
   - Field mapping and validation
   - Error handling (encoding, missing fields, malformed data)
   - Row-by-row processing with error location

4. **Data Import Operations**:
   - Transaction CSV import with validation
   - Fund price CSV import
   - Database rollback on errors
   - Portfolio-fund relationship validation

5. **Fund Price Management**:
   - Price creation and updates
   - Historical price lookup
   - Default date handling

### Uncovered Line

**Line 249**: `raise ValueError(f"CSV file error: {e!s}") from e`

**Context**: Exception handler for `csv.Error` in `_process_csv_content` method.

**Why uncovered**:
- `csv.Error` is rarely raised in practice
- Most CSV issues are caught by earlier validation (encoding, structure, data types)
- Would require malformed CSV that triggers internal parser errors

**Why acceptable**:
- 99% coverage vastly exceeds 80% target
- All practical error scenarios are tested
- Line represents defensive programming best practice

---

## Running Tests

### All DeveloperService Tests
```bash
pytest tests/services/test_developer_service.py -v
```

### Specific Test Classes
```bash
# Data sanitization tests
pytest tests/services/test_developer_service.py::TestSanitizationMethods -v

# Exchange rate management
pytest tests/services/test_developer_service.py::TestExchangeRateManagement -v

# CSV processing utilities
pytest tests/services/test_developer_service.py::TestCSVProcessing -v

# Import functionality
pytest tests/services/test_developer_service.py::TestTransactionImport -v
pytest tests/services/test_developer_service.py::TestFundPriceImport -v
```

### With Coverage Report
```bash
pytest tests/services/test_developer_service.py --cov=app/services/developer_service --cov-report=term-missing -v
```

### Individual Tests
```bash
# Test UTF-8 BOM handling (critical for Excel exports)
pytest tests/services/test_developer_service.py::TestCSVProcessing::test_process_csv_content_with_bom -v

# Test data sanitization
pytest tests/services/test_developer_service.py::TestSanitizationMethods::test_sanitize_float_invalid_format -v

# Test import functionality
pytest tests/services/test_developer_service.py::TestTransactionImport::test_import_transactions_csv_success -v
```

---

## Related Documentation

### Service Integration
- **All Services**: Use DeveloperService utilities for data cleaning and validation
- **Routes**: Developer routes use these services for administrative operations
- **Models**: Fund, Portfolio, PortfolioFund, Transaction, FundPrice, ExchangeRate

### External Dependencies
- **Python CSV Module**: Core CSV parsing functionality
- **SQLAlchemy**: Database operations for all data management
- **UUID Module**: Unique identifier generation

### Test Infrastructure
- [TESTING_INFRASTRUCTURE.md](../infrastructure/TESTING_INFRASTRUCTURE.md) - Test setup and fixtures
- **Query-Specific Data Pattern**: Used extensively to prevent database conflicts

---

## Key Learnings

### UTF-8 and BOM Handling Importance
- **Excel exports often include BOM** (Byte Order Mark) in UTF-8 files
- **utf-8-sig encoding** automatically handles BOM removal during decode
- **Testing both scenarios** ensures compatibility with real-world data

### Database Unique Constraints
- **UUID patterns** prevent test pollution: `f"US{uuid.uuid4().hex[:10].upper()}"`
- **Unique test data** eliminates UNIQUE constraint violations
- **Test isolation** improves reliability and debugging

### CSV Processing Robustness
- **Multiple error scenarios** must be handled: encoding, structure, data validation
- **Row-specific error messages** improve user experience
- **Rollback on failure** prevents partial data corruption

### Data Sanitization Strategy
- **Input cleaning** should be explicit and testable
- **Type conversion** must handle edge cases (None, empty, invalid)
- **Error messages** should guide users to correct problems

### Administrative Service Patterns
- **Simple query methods** for administrative access
- **Template generation** helps users understand expected formats
- **Comprehensive validation** prevents data corruption
- **Error handling** provides clear feedback for troubleshooting

---

**Last Updated**: v1.3.3 (Phase 5 - CSV Validation Enhancement) \
**Test Count**: 59 tests \
**Coverage**: 99% \
**Status**: Complete ✅ \
**Critical Bug Fixed**: 0 (clean implementation) \
**New Feature**: Centralized CSV header validation with comprehensive UTF-8 support
