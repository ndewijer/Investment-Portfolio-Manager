# SymbolLookupService Test Documentation

**Service**: `app/services/symbol_lookup_service.py`
**Test File**: `tests/services/test_symbol_lookup_service.py`
**Coverage**: 100% (20 tests)
**Status**: ✅ Complete

---

## Overview

The **SymbolLookupService** provides symbol information lookup and caching functionality, integrating with yfinance for real-time financial data.

### What This Service Does

1. **Symbol Information Lookup**: Fetch company/fund names, exchanges, currencies, and ISINs
2. **Intelligent Caching**: 7-day cache with validity tracking and automatic refresh
3. **ISIN-Based Lookup**: Find symbols by International Securities Identification Number
4. **Manual Symbol Updates**: Allow manual correction of symbol information
5. **yfinance Integration**: Real-time data fetching with error handling

### Test Suite Scope

**Full coverage testing of**:
- Symbol information retrieval with caching (10 tests)
- ISIN-based symbol lookup (4 tests)
- Manual symbol information updates (6 tests)

**Critical Bug Found**: UNIQUE constraint error with invalid cache entries (discovered and fixed).

---

## Test Organization

### Test Class Structure

```python
class TestGetSymbolInfo:         # 10 tests - Core symbol lookup functionality
class TestGetSymbolByIsin:       # 4 tests - ISIN-based lookups
class TestUpdateSymbolInfo:      # 6 tests - Manual symbol updates
```

### Testing Approach

**Query-Specific Data Pattern**: Each test uses unique symbols and ISINs to prevent database conflicts.

**Example**:
```python
# Unique identifiers per test
unique_symbol = f"AAPL{uuid.uuid4().hex[:4]}"
unique_isin = f"US{uuid.uuid4().hex[:10].upper()}"
```

**yfinance Mocking Strategy**: Using `unittest.mock.patch` to mock external API calls.

---

## Service-Specific Patterns

### 1. Cache Testing Strategy

**Focus**: Test cache hit, miss, expiration, and invalid states

**Pattern**: Direct database manipulation to create specific cache states
```python
# Create cache entry with specific state
symbol_info = SymbolInfo(
    symbol=unique_symbol,
    last_updated=datetime.now(UTC) - timedelta(days=8),  # Expired
    is_valid=True
)
```

### 2. yfinance Mocking

**Pattern**: Mock external API responses
```python
@patch("app.services.symbol_lookup_service.yf.Ticker")
def test_method(self, mock_ticker, app_context, db_session):
    mock_info = {
        "longName": "Apple Inc",
        "exchange": "NASDAQ",
        "currency": "USD"
    }
    mock_ticker_instance = MagicMock()
    mock_ticker_instance.info = mock_info
    mock_ticker.return_value = mock_ticker_instance
```

**Why this approach**:
- Prevents external API calls during testing
- Allows testing of different response scenarios
- Fast and reliable test execution

### 3. Invalid Cache Handling

**Critical Bug Testing**: Tests for UNIQUE constraint errors when invalid cache exists

**Bug Context**: Service tried to INSERT new records instead of UPDATing existing invalid cache entries.

---

## Test Suite Walkthrough

### TestGetSymbolInfo (10 tests)

Core symbol lookup functionality with caching logic.

#### Test 1: `test_get_symbol_info_cache_hit`
**Purpose**: Verify cache returns valid data without external API call
**Test Data**:
- Symbol: `AAPL{uuid}` (unique per test)
- Cached data: Valid, recent (within 7 days)
**Expected**: Returns cached data, no API call made
**Why**: Cache should prevent unnecessary external requests

#### Test 2: `test_get_symbol_info_cache_expired`
**Purpose**: Verify expired cache triggers fresh data fetch
**Test Data**:
- Cached data: Valid but 8 days old (expired)
- Mock API response: Updated company name
**Expected**: Returns fresh data from API, cache updated
**Why**: Stale data should be refreshed automatically

#### Test 3: `test_get_symbol_info_no_cache`
**Purpose**: Test fresh symbol lookup without existing cache
**Test Data**:
- Symbol: `TSLA{uuid}` (not in cache)
- Mock API response: Tesla data
**Expected**: Returns API data, creates cache entry
**Why**: New symbols should be fetched and cached

#### Test 4: `test_get_symbol_info_force_refresh`
**Purpose**: Test force refresh ignores valid cache
**Test Data**:
- Cached data: Valid and recent
- force_refresh=True parameter
**Expected**: Returns fresh API data, ignores cache
**Why**: Users should be able to force data refresh

#### Test 5: `test_get_symbol_info_yfinance_no_data`
**Purpose**: Handle case when yfinance returns no data
**Test Data**:
- Mock API response: None (symbol not found)
**Expected**: Returns None
**Why**: Invalid symbols should be handled gracefully

#### Test 6: `test_get_symbol_info_invalidates_cache_on_no_data`
**Purpose**: Mark cache as invalid when API returns no data
**Test Data**:
- Existing cache: Valid but expired
- Mock API response: None
**Expected**: Cache marked invalid, returns None
**Why**: Prevent returning stale data for invalid symbols

#### Test 7: `test_get_symbol_info_uses_short_name_fallback`
**Purpose**: Use shortName when longName not available
**Test Data**:
- Mock API response: Only has shortName field
**Expected**: Returns shortName as the name
**Why**: Handle incomplete API responses gracefully

#### Test 8: `test_get_symbol_info_uses_unknown_name_fallback`
**Purpose**: Use "Unknown" when no name fields available
**Test Data**:
- Mock API response: No name fields
**Expected**: Returns "Unknown" as name
**Why**: Prevent null/empty names in database

#### Test 9: `test_get_symbol_info_handles_exception`
**Purpose**: Handle yfinance API exceptions gracefully
**Test Data**:
- Mock API: Raises exception
**Expected**: Returns None, no crash
**Why**: External API failures shouldn't break the application

#### Test 10: `test_get_symbol_info_skips_invalid_cache`
**Purpose**: Skip invalid cache entries and update them with fresh data
**Test Data**:
- Cached data: is_valid=False
- Mock API response: Valid new data
**Expected**: Returns fresh data, cache updated and marked valid
**Why**: **Critical Bug Fix** - Invalid cache was causing UNIQUE constraint errors

**Bug Details**: This test discovered a critical bug where the service queried cache by `symbol AND is_valid=True`, causing UNIQUE constraint errors when trying to INSERT a new record for a symbol that already existed but was invalid.

**Fix**: Changed query to `symbol` only, then check validity in the condition.

### TestGetSymbolByIsin (4 tests)

ISIN-based symbol lookup functionality.

#### Test 11: `test_get_symbol_by_isin_cache_hit`
**Purpose**: Find symbol using ISIN from cache
**Test Data**:
- ISIN: `US{uuid}` (unique)
- Cached symbol with matching ISIN
**Expected**: Returns symbol info
**Why**: ISIN is a reliable identifier for securities

#### Test 12: `test_get_symbol_by_isin_not_found`
**Purpose**: Return None when ISIN not in cache
**Test Data**:
- ISIN: `US{uuid}` (not in cache)
**Expected**: Returns None
**Why**: Handle missing ISIN gracefully

#### Test 13: `test_get_symbol_by_isin_invalid_entry`
**Purpose**: Skip invalid cache entries during ISIN lookup
**Test Data**:
- Cached entry: Has ISIN but is_valid=False
**Expected**: Returns None (invalid entry skipped)
**Why**: Don't return invalid cached data

#### Test 14: `test_get_symbol_by_isin_handles_exception`
**Purpose**: Handle database exceptions during ISIN lookup
**Test Data**:
- Mock database query to raise exception
**Expected**: Returns None, error logged
**Why**: Database failures shouldn't crash the service

### TestUpdateSymbolInfo (6 tests)

Manual symbol information update functionality.

#### Test 15: `test_update_existing_symbol`
**Purpose**: Update an existing symbol's information
**Test Data**:
- Existing symbol in cache
- Update data: New name and ISIN
**Expected**: Symbol updated, data_source="manual"
**Why**: Allow manual correction of symbol data

#### Test 16: `test_update_creates_new_symbol`
**Purpose**: Create new symbol entry via update
**Test Data**:
- Symbol not in cache
- Complete symbol data provided
**Expected**: New symbol created with data_source="manual"
**Why**: Manual addition of symbols not in yfinance

#### Test 17: `test_update_symbol_partial_data`
**Purpose**: Update only specific fields, preserve others
**Test Data**:
- Existing symbol: Name, exchange, currency
- Update data: Only ISIN
**Expected**: ISIN updated, other fields preserved
**Why**: Allow partial updates without overwriting existing data

#### Test 18: `test_update_symbol_all_fields`
**Purpose**: Update all fields of existing symbol
**Test Data**:
- Existing symbol with old data
- Update data: All fields changed
**Expected**: All fields updated to new values
**Why**: Complete symbol information overhaul

#### Test 19: `test_update_symbol_handles_exception`
**Purpose**: Handle database exceptions during updates
**Test Data**:
- Mock database query to raise exception
**Expected**: Returns None, error logged
**Why**: Database failures shouldn't crash update operations

#### Test 20: `test_update_symbol_empty_data`
**Purpose**: Handle empty update data gracefully
**Test Data**:
- Existing symbol
- Empty update dictionary
**Expected**: last_updated field updated, data_source="manual"
**Why**: Empty updates should at least update timestamp

---

## Coverage Analysis

### Coverage Achievement

```
Coverage: 100% (69/69 statements)
```

**Perfect Coverage**: Every line of code in SymbolLookupService is executed by tests.

### What's Covered

1. **Cache Logic**:
   - Cache hit/miss scenarios
   - Expiration handling
   - Invalid cache handling
   - Force refresh logic

2. **yfinance Integration**:
   - Successful API calls
   - API failures and exceptions
   - Data parsing (longName, shortName, fallbacks)
   - Response validation

3. **ISIN Lookups**:
   - Cache-based ISIN searches
   - Invalid entry handling
   - Database exceptions

4. **Manual Updates**:
   - Creating new symbols
   - Updating existing symbols
   - Partial vs. complete updates
   - Exception handling

### Critical Bug Discovered

**Bug**: UNIQUE constraint error when invalid cache exists
**Impact**: Service crashed when trying to refresh previously invalid symbols
**Root Cause**: Query filtered by `is_valid=True`, causing INSERT instead of UPDATE
**Fix**: Query by symbol only, check validity in condition
**Test**: `test_get_symbol_info_skips_invalid_cache` validates the fix

---

## Running Tests

### All SymbolLookupService Tests
```bash
pytest tests/services/test_symbol_lookup_service.py -v
```

### Specific Test Classes
```bash
# Symbol information tests
pytest tests/services/test_symbol_lookup_service.py::TestGetSymbolInfo -v

# ISIN lookup tests
pytest tests/services/test_symbol_lookup_service.py::TestGetSymbolByIsin -v

# Update functionality tests
pytest tests/services/test_symbol_lookup_service.py::TestUpdateSymbolInfo -v
```

### With Coverage Report
```bash
pytest tests/services/test_symbol_lookup_service.py --cov=app/services/symbol_lookup_service --cov-report=term-missing -v
```

### Bug Fix Test
```bash
# Test the critical bug fix
pytest tests/services/test_symbol_lookup_service.py::TestGetSymbolInfo::test_get_symbol_info_skips_invalid_cache -v
```

---

## Related Documentation

### Service Integration
- **PriceUpdateService**: Uses symbol lookup for price data
- **FundMatchingService**: Relies on symbol information for matching
- **Models**: SymbolInfo model for caching

### External Dependencies
- **yfinance**: Financial data provider (mocked in tests)
- **Database**: SymbolInfo table for caching

### Bug Fixes
- [BUG_FIXES_1.3.3.md](../phases/BUG_FIXES_1.3.3.md) - Bug #5: UNIQUE constraint error

### Test Infrastructure
- [TESTING_INFRASTRUCTURE.md](../infrastructure/TESTING_INFRASTRUCTURE.md) - Test setup and patterns

---

## Key Learnings

### Cache Invalidation Complexity
- **Invalid cache must be handled carefully** to avoid database conflicts
- **Query filtering** must consider all possible record states
- **UPDATE vs INSERT** logic is critical for unique constraints

### External API Integration Testing
- **Mock all external calls** for reliable testing
- **Test various response scenarios** (success, failure, partial data)
- **Handle API exceptions gracefully** in production code

### Test Data Isolation
- **Unique identifiers prevent test pollution**
- **Database state isolation** is critical for cache testing
- **Query-specific data patterns** work well for services with database state

### yfinance API Quirks
- **Multiple name fields** (longName, shortName) with inconsistent availability
- **Exception handling** is essential for external API reliability
- **Fallback strategies** improve user experience

---

**Last Updated**: v1.3.3 (Phase 4)
**Test Count**: 20 tests
**Coverage**: 100%
**Status**: Complete ✅
**Critical Bug Fixed**: 1 (UNIQUE constraint with invalid cache)
