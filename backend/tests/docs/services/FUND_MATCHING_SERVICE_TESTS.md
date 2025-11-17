# FundMatchingService Test Documentation

**Service**: `app/services/fund_matching_service.py` \
**Test File**: `tests/services/test_fund_matching_service.py` \
**Coverage**: 100% (27 tests) \
**Status**: ✅ Complete

---

## Overview

The **FundMatchingService** handles matching of IBKR transactions to existing funds in portfolios using multiple matching strategies.

### What This Service Does

1. **Symbol Matching**: Match IBKR transactions to funds by symbol (exact or normalized)
2. **ISIN Matching**: Match using International Securities Identification Numbers
3. **Portfolio Eligibility**: Determine which portfolios can receive a matched transaction
4. **Symbol Normalization**: Handle exchange suffixes (e.g., "AAPL.NAS" → "AAPL")

### Test Suite Scope

**Full coverage testing of**:
- Symbol normalization logic (17 tests)
- Fund finding by ISIN and symbol (5 tests)
- Portfolio eligibility determination (5 tests)

**Key Achievement**: 100% statement coverage with comprehensive edge case testing.

---

## Test Organization

### Test Class Structure

```python
class TestSymbolNormalization:          # 17 tests - Symbol suffix handling
class TestFindFundMethods:              # 5 tests - Fund lookup logic
class TestGetEligiblePortfoliosForTransaction:  # 5 tests - Portfolio matching
```

### Testing Approach

**Query-Specific Data Pattern**: Each test creates unique funds/portfolios to prevent database pollution and ensure test isolation.

**Example**:
```python
# Every test uses unique identifiers
unique_isin = f"US{uuid.uuid4().hex[:10].upper()}"
unique_symbol = f"AAPL{uuid.uuid4().hex[:4]}"
```

**Why this pattern**: Prevents UNIQUE constraint violations and makes tests independent.

---

## Service-Specific Patterns

### 1. Symbol Normalization Testing

**Focus**: Exchange suffix removal for symbol matching

**Pattern**: Test various exchange suffix formats
```python
def test_normalize_symbol_nasdaq_suffix(self):
    result = FundMatchingService.normalize_symbol("AAPL.NAS")
    assert result == "AAPL"
```

### 2. Fund Creation Strategy

**Pattern**: Direct object creation (not factories)
```python
fund = Fund(
    id=str(uuid.uuid4()),
    isin=unique_isin,
    symbol=unique_symbol,
    name="Apple Inc",
    currency="USD",
    exchange="NASDAQ",
    investment_type=InvestmentType.STOCK,
)
db.session.add(fund)
db.session.commit()
```

**Why direct creation**:
- Simple data structures
- Clear test data relationships
- No complex factory interdependencies

### 3. IBKR Transaction Matching

**Pattern**: Test different transaction scenarios
```python
txn = IBKRTransaction(
    id=str(uuid.uuid4()),
    ibkr_transaction_id=f"TXN{uuid.uuid4().hex[:10]}",
    symbol="AAPL1234",  # Matches fund symbol
    isin="US9999999999",  # Different ISIN
    transaction_date=date(2024, 1, 15),
    # ... other fields
)
```

---

## Test Suite Walkthrough

### TestSymbolNormalization (17 tests)

These tests verify the symbol normalization logic that removes exchange suffixes for matching.

#### Test 1: `test_normalize_symbol_no_suffix`
**Purpose**: Verify symbols without suffixes remain unchanged \
**Test Data**: "AAPL" (plain symbol) \
**Expected**: "AAPL" (no change) \
**Why**: Base case - symbols don't always have suffixes

#### Test 2: `test_normalize_symbol_nasdaq_suffix`
**Purpose**: Remove NASDAQ exchange suffix \
**Test Data**: "AAPL.NAS" \
**Expected**: "AAPL" \
**Why**: Common NASDAQ suffix format

#### Test 3: `test_normalize_symbol_nyse_suffix`
**Purpose**: Remove NYSE exchange suffix \
**Test Data**: "IBM.N" \
**Expected**: "IBM" \
**Why**: Common NYSE suffix format

#### Test 4: `test_normalize_symbol_dot_suffix`
**Purpose**: Remove generic dot suffixes \
**Test Data**: "MSFT.O" \
**Expected**: "MSFT" \
**Why**: Generic exchange suffix pattern

#### Test 5: `test_normalize_symbol_european_suffix`
**Purpose**: Remove European exchange suffixes \
**Test Data**: "SAP.DE" \
**Expected**: "SAP" \
**Why**: German exchange (XETRA) format

#### Test 6: `test_normalize_symbol_london_suffix`
**Purpose**: Remove London Stock Exchange suffix \
**Test Data**: "SHEL.L" \
**Expected**: "SHEL" \
**Why**: London exchange format

#### Test 7: `test_normalize_symbol_multiple_dots`
**Purpose**: Handle symbols with multiple dots correctly \
**Test Data**: "BRK.A.N" \
**Expected**: "BRK.A" \
**Why**: Some symbols naturally contain dots (Berkshire Hathaway Class A)

#### Test 8: `test_normalize_symbol_short_suffix`
**Purpose**: Remove single character suffixes \
**Test Data**: "GOOG.Q" \
**Expected**: "GOOG" \
**Why**: Some exchanges use single letter suffixes

#### Test 9: `test_normalize_symbol_number_suffix`
**Purpose**: Remove numeric suffixes \
**Test Data**: "FUND.1" \
**Expected**: "FUND" \
**Why**: Some symbols have numeric identifiers

#### Test 10: `test_normalize_symbol_empty_string`
**Purpose**: Handle empty input gracefully \
**Test Data**: "" \
**Expected**: "" \
**Why**: Edge case - prevent crashes on invalid input

#### Test 11: `test_normalize_symbol_only_dot`
**Purpose**: Handle malformed input (just a dot) \
**Test Data**: "." \
**Expected**: "" \
**Why**: Edge case - malformed symbol handling

#### Test 12: `test_normalize_symbol_dot_at_end`
**Purpose**: Handle trailing dots \
**Test Data**: "AAPL." \
**Expected**: "AAPL" \
**Why**: Malformed symbol with trailing dot

#### Test 13: `test_normalize_symbol_mixed_case_suffix`
**Purpose**: Handle mixed case suffixes \
**Test Data**: "TSLA.Na" \
**Expected**: "TSLA" \
**Why**: Case insensitive suffix removal

#### Test 14: `test_normalize_symbol_long_suffix`
**Purpose**: Remove longer exchange suffixes \
**Test Data**: "VTI.ARCA" \
**Expected**: "VTI" \
**Why**: Some exchanges have longer identifiers

#### Test 15: `test_normalize_symbol_preserve_core_dots`
**Purpose**: Keep dots that are part of the core symbol \
**Test Data**: "BRK.B" \
**Expected**: "BRK.B" \
**Why**: Berkshire Hathaway Class B - dot is part of official symbol

#### Test 16: `test_normalize_symbol_complex_case`
**Purpose**: Handle complex real-world example \
**Test Data**: "GOOGL.NAS" \
**Expected**: "GOOGL" \
**Why**: Real NASDAQ-listed stock with suffix

#### Test 17: `test_normalize_symbol_canadian_suffix`
**Purpose**: Remove Canadian exchange suffix \
**Test Data**: "SHOP.TO" \
**Expected**: "SHOP" \
**Why**: Toronto Stock Exchange format

### TestFindFundMethods (5 tests)

These tests verify the fund lookup logic using ISIN and symbol matching.

#### Test 18: `test_find_fund_by_isin_match`
**Purpose**: Find fund using exact ISIN match \
**Test Data**:
- Fund ISIN: `US{uuid}` (unique)
- Transaction ISIN: Same as fund
**Expected**: Returns fund object \
**Why**: ISIN is the most reliable identifier for securities

#### Test 19: `test_find_fund_by_exact_symbol_match`
**Purpose**: Find fund using exact symbol match \
**Test Data**:
- Fund symbol: `AAPL{uuid}`
- Transaction symbol: Same as fund
- Different ISINs
**Expected**: Returns fund object \
**Why**: Symbol matching when ISIN doesn't match

#### Test 20: `test_find_fund_by_normalized_symbol`
**Purpose**: Find fund using normalized symbol match \
**Test Data**:
- Fund symbol: `WEBN{uuid}.DE`
- Transaction symbol: `WEBN{uuid}` (no suffix)
- Different ISINs
**Expected**: Returns fund object \
**Why**: Handle exchange suffix differences

#### Test 21: `test_find_fund_no_match`
**Purpose**: Return None when no fund matches \
**Test Data**:
- Fund: `AAPL{uuid1}`
- Transaction: `TSLA{uuid2}` (different symbol and ISIN)
**Expected**: Returns None \
**Why**: Handle case where transaction doesn't match any fund

#### Test 22: `test_find_fund_isin_priority_over_symbol`
**Purpose**: Verify ISIN matching takes priority over symbol \
**Test Data**:
- Fund 1: ISIN matches, symbol different
- Fund 2: Symbol matches, ISIN different
**Expected**: Returns Fund 1 (ISIN match) \
**Why**: ISIN is more reliable than symbol for identification

### TestGetEligiblePortfoliosForTransaction (5 tests)

These tests verify portfolio eligibility determination based on fund matching.

#### Test 23: `test_eligible_portfolios_success`
**Purpose**: Return portfolios containing matched fund \
**Test Data**:
- Fund in 2 portfolios
- Transaction matches fund by ISIN
**Expected**: Returns both portfolios with match info \
**Why**: Core functionality - find eligible portfolios

#### Test 24: `test_eligible_portfolios_no_fund_match`
**Purpose**: Return empty when transaction doesn't match any fund \
**Test Data**:
- Fund exists but different symbol/ISIN
- Transaction doesn't match
**Expected**: Empty portfolios list, match_info found=False \
**Why**: Handle no-match scenario gracefully

#### Test 25: `test_eligible_portfolios_fund_not_in_portfolio`
**Purpose**: Return empty when fund exists but not in any portfolio \
**Test Data**:
- Fund exists and matches transaction
- Fund not added to any portfolio
**Expected**: Empty portfolios list, match_info found=True \
**Why**: Fund match but no portfolio eligibility

#### Test 26: `test_eligible_portfolios_matched_by_exact_symbol`
**Purpose**: Verify match_info shows correct match method \
**Test Data**:
- Fund symbol matches transaction
- Different ISINs
**Expected**: match_info matched_by="exact_symbol" \
**Why**: Track how the match was made for debugging

#### Test 27: `test_eligible_portfolios_matched_by_normalized_symbol`
**Purpose**: Verify match_info for normalized symbol matches \
**Test Data**:
- Fund symbol: `WEBN{uuid}.DE`
- Transaction symbol: `WEBN{uuid}` (normalized match)
**Expected**: match_info matched_by="normalized_symbol" \
**Why**: Track normalization-based matches

---

## Coverage Analysis

### Coverage Achievement

```
Coverage: 100% (49/49 statements)
```

**Perfect Coverage**: Every line of code in FundMatchingService is executed by tests.

### What's Covered

1. **Symbol Normalization**:
   - All suffix removal patterns
   - Edge cases (empty strings, malformed input)
   - Complex symbols with multiple dots

2. **Fund Finding Logic**:
   - ISIN-based matching
   - Exact symbol matching
   - Normalized symbol matching
   - Priority ordering (ISIN > symbol)
   - No-match scenarios

3. **Portfolio Eligibility**:
   - Multiple portfolio scenarios
   - Fund matching with portfolio membership
   - Match tracking and metadata
   - Error conditions

### Why 100% Is Appropriate

This service has straightforward logic without complex error paths:
- No external API calls to mock
- No complex exception handling
- Pure business logic functions
- Clear input/output patterns

**100% coverage is both achievable and valuable** for this type of service.

---

## Running Tests

### All FundMatchingService Tests
```bash
pytest tests/services/test_fund_matching_service.py -v
```

### Specific Test Classes
```bash
# Symbol normalization tests
pytest tests/services/test_fund_matching_service.py::TestSymbolNormalization -v

# Fund finding tests
pytest tests/services/test_fund_matching_service.py::TestFindFundMethods -v

# Portfolio eligibility tests
pytest tests/services/test_fund_matching_service.py::TestGetEligiblePortfoliosForTransaction -v
```

### With Coverage Report
```bash
pytest tests/services/test_fund_matching_service.py --cov=app/services/fund_matching_service --cov-report=term-missing -v
```

### Individual Tests
```bash
# Test specific normalization
pytest tests/services/test_fund_matching_service.py::TestSymbolNormalization::test_normalize_symbol_nasdaq_suffix -v

# Test ISIN matching
pytest tests/services/test_fund_matching_service.py::TestFindFundMethods::test_find_fund_by_isin_match -v
```

---

## Related Documentation

### Service Integration
- **IBKRTransactionService**: Uses fund matching for transaction allocation
- **SymbolLookupService**: Provides symbol data that funds are matched against
- **Models**: Fund, Portfolio, PortfolioFund, IBKRTransaction

### Test Infrastructure
- [TESTING_INFRASTRUCTURE.md](../infrastructure/TESTING_INFRASTRUCTURE.md) - Test setup and fixtures
- **Query-Specific Data Pattern**: Used extensively in this test suite

### Bug Fixes
- **No bugs discovered** in this service during testing (clean implementation)

---

## Key Learnings

### Symbol Normalization Complexity
- Exchange suffixes vary significantly across markets
- Some symbols naturally contain dots (BRK.A, BRK.B)
- Normalization must be carefully designed to avoid breaking valid symbols

### Test Data Strategy
- **Unique identifiers prevent test pollution**
- **Direct object creation** works well for simple data structures
- **Comprehensive edge case testing** achieved 100% coverage

### Fund Matching Priority
- **ISIN matching is most reliable** (international standard)
- **Symbol matching is fallback** (can have variations)
- **Normalized symbol matching** handles exchange differences

---

**Last Updated**: v1.3.3 (Phase 4) \
**Test Count**: 27 tests \
**Coverage**: 100% \
**Status**: Complete ✅
