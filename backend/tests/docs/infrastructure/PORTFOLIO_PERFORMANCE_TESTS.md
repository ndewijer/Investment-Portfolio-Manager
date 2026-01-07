# Portfolio Performance Tests (v1.3.2)

**Created**: Version 1.3.2\
**File**: `tests/test_portfolio_performance.py`\
**Total Tests**: 12 tests\
**Purpose**: Validate performance optimizations and prevent regressions

---

## Overview

These tests were created in v1.3.2 to validate two major performance optimizations:

1. **Phase 1 Optimization**: Batch processing for portfolio history (eliminated 16,000+ queries)
2. **Phase 2 Optimization**: Eager loading to eliminate N+1 queries (reduced 50+ queries to <10)

**Before optimization**:
- Overview page: 16,425 queries, 5-10 seconds
- Portfolio detail: 7,665 queries, 3-5 seconds

**After optimization**:
- Overview page: <100 queries, <1 second
- Portfolio detail: <50 queries, <0.5 seconds

**Performance improvement**: 99.4% reduction in queries, 90%+ reduction in time

---

## Test Infrastructure

### Fixtures Used

**`query_counter`** - Counts SQL queries executed
- Tracks every database query
- Provides query count and query list
- See `TESTING_INFRASTRUCTURE.md` for details

**`timer`** - Measures execution time
- Tracks wall-clock time
- Used for performance benchmarks
- See `TESTING_INFRASTRUCTURE.md` for details

**`app_context`** - Provides Flask context
- Required for database operations
- Standard fixture for all tests

### Data Requirements

**Important**: These tests require production database with real data

**Why**: Performance tests need realistic data volume to be meaningful

**Running tests**:
- Tests will skip if no data found
- Use `pytest.skip()` when portfolios don't exist
- Not suitable for CI/CD (requires seeded database)

---

## Test Suite 1: Portfolio History Performance

**Class**: `TestPortfolioHistoryPerformance`\
**Tests**: 5 tests\
**Focus**: Batch processing optimization for historical calculations

### Background: The N+1 Query Problem

**Original implementation** (before v1.3.2):
```python
# For each day in range:
for date in date_range:
    # For each portfolio:
    for portfolio in portfolios:
        # For each fund in portfolio:
        for fund in portfolio.funds:
            # Query transactions for this fund on this date
            transactions = get_transactions(fund, date)  # N+1 query!
            # Query price for this fund on this date
            price = get_price(fund, date)  # Another N+1!
```

**Problem**:
- 365 days × 10 portfolios × 5 funds = 18,250 queries
- Each query is a round-trip to database
- Total time: 5-10 seconds

**Solution** (v1.3.2 optimization):
```python
# Batch load ALL transactions in date range
all_transactions = Transaction.query.filter(date.between(start, end)).all()

# Batch load ALL prices in date range
all_prices = FundPrice.query.filter(date.between(start, end)).all()

# Process in memory (no more database queries)
for date in date_range:
    for portfolio in portfolios:
        for fund in portfolio.funds:
            # Use pre-loaded data (no query)
            transactions = filter_in_memory(all_transactions, fund, date)
            price = filter_in_memory(all_prices, fund, date)
```

**Result**: ~50 queries total (batch loads only), <1 second

---

### Test 1: `test_get_portfolio_history_query_count`

**Purpose**: Verify batch processing keeps queries under target

**Scenario**: Load 365 days of portfolio history (overview page)

**Target**: <100 queries

**Test Data**:
- Date range: 365 days (past year)
- Portfolios: All non-archived portfolios
- Funds: All funds in those portfolios

**What it tests**:
```python
query_counter.reset()

# Call the optimized method
result = PortfolioService.get_portfolio_history(
    start_date=one_year_ago,
    end_date=today
)

# Verify query count
assert query_counter.count < 100
```

**Why this matters**:
- Overview page is most-used feature
- 365 days is typical user behavior
- Regression would be immediately noticeable (slow page load)

**Before optimization**: 16,425 queries\
**After optimization**: ~50 queries\
**Improvement**: 99.7% reduction

---

### Test 2: `test_get_portfolio_history_execution_time`

**Purpose**: Verify execution completes within time budget

**Scenario**: Same 365-day load as Test 1

**Target**: <1 second

**What it tests**:
```python
timer.start()

result = PortfolioService.get_portfolio_history(
    start_date=one_year_ago,
    end_date=today
)

elapsed = timer.stop()

assert elapsed < 1.0  # Must complete in under 1 second
```

**Why this matters**:
- Users expect fast page loads (<2 seconds total)
- Database query time is only part of total load time
- Need headroom for network latency, rendering, etc.

**Before optimization**: 5-10 seconds\
**After optimization**: ~0.3 seconds\
**Improvement**: 95%+ reduction

---

### Test 3: `test_get_portfolio_fund_history_query_count`

**Purpose**: Verify batch processing for portfolio detail page

**Scenario**: Load 365 days for single portfolio (drill-down view)

**Target**: <50 queries

**Test Data**:
- One portfolio (first non-archived)
- All funds in that portfolio
- 365 days of history

**What it tests**:
```python
query_counter.reset()

result = PortfolioService.get_portfolio_fund_history(
    portfolio_id=portfolio.id,
    start_date=one_year_ago,
    end_date=today
)

assert query_counter.count < 50
```

**Why this matters**:
- Portfolio detail page shows per-fund breakdown
- Users frequently drill into individual portfolios
- More detailed data = more potential for N+1 queries

**Before optimization**: 7,665 queries\
**After optimization**: ~30 queries\
**Improvement**: 99.6% reduction

**Data dependency**: Skips if no portfolio found in database

---

### Test 4: `test_get_portfolio_fund_history_execution_time`

**Purpose**: Verify portfolio detail page speed

**Scenario**: Same as Test 3

**Target**: <0.5 seconds (stricter than overview)

**What it tests**:
```python
timer.start()

result = PortfolioService.get_portfolio_fund_history(
    portfolio_id=portfolio.id,
    start_date=one_year_ago,
    end_date=today
)

elapsed = timer.stop()

assert elapsed < 0.5
```

**Why stricter target**:
- Fewer portfolios to process (just one)
- Fewer funds (only those in selected portfolio)
- Should be faster than overview page

**Before optimization**: 3-5 seconds\
**After optimization**: ~0.2 seconds\
**Improvement**: 96%+ reduction

---

### Test 5: `test_full_history_performance`

**Purpose**: Stress test with maximum data

**Scenario**: Load ALL history (no date limits) - 1,500+ days

**Target**: <150 queries, <2 seconds (more lenient)

**Test Data**:
- All portfolios
- All funds
- All historical data (from first transaction to today)

**What it tests**:
```python
query_counter.reset()
timer.start()

# No date limits = full history
result = PortfolioService.get_portfolio_history()

elapsed = timer.stop()

print(f"Days: {len(result)}")
print(f"Queries: {query_counter.count}")
print(f"Time: {elapsed:.3f}s")

assert query_counter.count < 150
assert elapsed < 2.0
```

**Why this matters**:
- Users might export all data
- Ensures optimization scales to large datasets
- Prevents memory issues with large result sets

**Why more lenient**:
- More data to process
- Rarely used feature (most users view 1 year)
- Still should be reasonable

**Expected results**:
- ~1,500 days of data
- ~80-100 queries
- ~0.8-1.5 seconds

---

## Test Suite 2: Portfolio History Correctness

**Class**: `TestPortfolioHistoryCorrectness`\
**Tests**: 3 tests\
**Focus**: Ensure optimization didn't break calculations

### Why These Tests Exist

Performance optimization can introduce bugs:
- Batch loading might miss edge cases
- In-memory filtering might have logic errors
- Date range handling might be off-by-one

These tests verify that **fast code is also correct code**.

---

### Test 6: `test_portfolio_history_returns_data`

**Purpose**: Verify data structure is valid

**Scenario**: Load default history and check structure

**What it tests**:
```python
result = PortfolioService.get_portfolio_history()

# Should be a list
assert isinstance(result, list)

if len(result) > 0:
    day = result[0]

    # Each day has required fields
    assert "date" in day
    assert "portfolios" in day
    assert isinstance(day["portfolios"], list)

    if len(day["portfolios"]) > 0:
        portfolio = day["portfolios"][0]

        # Each portfolio has required fields
        assert "id" in portfolio
        assert "name" in portfolio
        assert "value" in portfolio
        assert "cost" in portfolio
        assert "realized_gain" in portfolio
        assert "unrealized_gain" in portfolio
```

**Why this matters**:
- Frontend expects specific structure
- Missing fields cause UI errors
- Type mismatches cause rendering failures

**What it validates**:
- Return type is correct (list)
- Date entries exist
- Portfolio entries have all required fields
- Numeric fields exist (value, cost, gains)

---

### Test 7: `test_portfolio_fund_history_returns_data`

**Purpose**: Verify fund-level data structure

**Scenario**: Load portfolio detail and check structure

**What it tests**:
```python
result = PortfolioService.get_portfolio_fund_history(portfolio_id)

assert isinstance(result, list)

if len(result) > 0:
    day = result[0]

    assert "date" in day
    assert "funds" in day

    if len(day["funds"]) > 0:
        fund = day["funds"][0]

        # Each fund has required fields
        assert "portfolio_fund_id" in fund
        assert "fund_id" in fund
        assert "fund_name" in fund
        assert "value" in fund
        assert "cost" in fund
        assert "shares" in fund
        assert "price" in fund
        assert "realized_gain" in fund
        assert "unrealized_gain" in fund
```

**Why this matters**:
- Fund detail view needs more fields than overview
- Shares and price are critical for calculations
- Missing data breaks portfolio drill-down

**Data dependency**: Skips if no portfolio found

---

### Test 8: `test_date_range_filtering`

**Purpose**: Verify date range parameters work correctly

**Scenario**: Request 30 days, verify we get ~30 days back

**What it tests**:
```python
end_date = today
start_date = today - 30 days

result = PortfolioService.get_portfolio_history(
    start_date=start_date,
    end_date=end_date
)

# Should get approximately 31 days (inclusive)
assert 29 <= len(result) <= 32
```

**Why flexible range** (29-32 instead of exactly 31):
- Weekends might have no data
- Holidays might have no data
- Some days might be skipped in calculations

**Why this matters**:
- Frontend date pickers allow custom ranges
- Off-by-one errors are common in date logic
- Ensures start/end are inclusive

---

## Test Suite 3: Phase 2 Eager Loading Performance

**Class**: `TestPhase2EagerLoadingPerformance`\
**Tests**: 4 tests\
**Focus**: N+1 query elimination in summary and transaction loading

### Background: Phase 2 N+1 Problems

**Problem 1 - Portfolio Summary**:
```python
# Get all portfolios
portfolios = Portfolio.query.all()  # 1 query

for portfolio in portfolios:
    # For each portfolio, get funds
    funds = portfolio.funds  # N queries (lazy loading)

    for fund in funds:
        # For each fund, get transactions
        transactions = fund.transactions  # N×M queries
```

Total: 1 + N + (N×M) queries = **50+ queries for 10 portfolios**

**Solution - Eager Loading**:
```python
# Load portfolios WITH funds and transactions in single query
portfolios = Portfolio.query.options(
    joinedload(Portfolio.portfolio_funds)
    .joinedload(PortfolioFund.fund)
    .joinedload(PortfolioFund.transactions)
).all()  # 1 query with JOINs

# Now iterate without additional queries
for portfolio in portfolios:
    funds = portfolio.funds  # Already loaded
    transactions = fund.transactions  # Already loaded
```

Total: **3-5 queries** (depending on data)

---

### Test 9: `test_portfolio_summary_query_count`

**Purpose**: Verify eager loading eliminates N+1 in summary

**Scenario**: Load portfolio summary (overview page data)

**Target**: <10 queries

**What it tests**:
```python
query_counter.reset()

result = PortfolioService.get_portfolio_summary()

assert query_counter.count < 10
```

**Why this matters**:
- Summary loads on every page view
- Most frequent database operation
- N+1 would scale badly with more portfolios

**Before optimization**: ~50 queries for 10 portfolios\
**After optimization**: 3-5 queries\
**Improvement**: 90%+ reduction

---

### Test 10: `test_portfolio_summary_execution_time`

**Purpose**: Verify summary loads fast

**Scenario**: Same as Test 9

**Target**: <0.2 seconds

**What it tests**:
```python
timer.start()

result = PortfolioService.get_portfolio_summary()

elapsed = timer.stop()

assert elapsed < 0.2
```

**Why strict target**:
- Summary is small data volume
- Should be nearly instant
- Users see this on every page

**Expected**: ~0.05-0.1 seconds

---

### Test 11: `test_portfolio_transactions_query_count`

**Purpose**: Verify batch loading of IBKR allocations

**Scenario**: Load all transactions for a portfolio

**Target**: <5 queries

**Background - The Problem**:
```python
transactions = Transaction.query.filter_by(portfolio_fund_id=...).all()  # 1 query

for txn in transactions:
    # Get IBKR allocations for this transaction
    allocations = txn.ibkr_allocations  # N queries (230 transactions = 230 queries!)
```

**Solution**:
```python
# Batch load allocations with transactions
transactions = Transaction.query.options(
    joinedload(Transaction.ibkr_allocations)
).filter_by(...).all()  # 2 queries (transaction + allocations)

for txn in transactions:
    allocations = txn.ibkr_allocations  # Already loaded
```

**What it tests**:
```python
query_counter.reset()

result = TransactionService.get_portfolio_transactions(portfolio_id)

print(f"Queries: {query_counter.count}")
print(f"Transactions: {len(result)}")

assert query_counter.count < 5
```

**Why this matters**:
- Transaction page is heavily used
- Each portfolio might have 100+ transactions
- IBKR allocations add complexity

**Before optimization**: 231 queries for 230 transactions (1 + 230)\
**After optimization**: 2-3 queries\
**Improvement**: 98.7%+ reduction

**Data dependency**: Skips if no portfolio found

---

### Test 12: `test_portfolio_transactions_execution_time`

**Purpose**: Verify transaction loading is fast

**Scenario**: Same as Test 11

**Target**: <0.1 seconds

**What it tests**:
```python
timer.start()

result = TransactionService.get_portfolio_transactions(portfolio_id)

elapsed = timer.stop()

assert elapsed < 0.1
```

**Why strict target**:
- Even with 200+ transactions, should be fast
- Users frequently view transaction history
- Loading indicator should be barely visible

**Expected**: ~0.03-0.05 seconds

**Data dependency**: Skips if no portfolio found

---

## Performance Benchmarks Summary

### Query Count Reductions

| Operation | Before | After | Improvement |
|-----------|--------|-------|-------------|
| Portfolio history (365 days) | 16,425 | <100 | 99.4% |
| Portfolio detail (365 days) | 7,665 | <50 | 99.3% |
| Portfolio summary | ~50 | <10 | 80%+ |
| Portfolio transactions | 231 | <5 | 97.8% |

### Execution Time Improvements

| Operation | Before | After | Improvement |
|-----------|--------|-------|-------------|
| Portfolio history | 5-10s | <1s | 90%+ |
| Portfolio detail | 3-5s | <0.5s | 90%+ |
| Portfolio summary | ~0.5s | <0.2s | 60%+ |
| Portfolio transactions | ~0.4s | <0.1s | 75%+ |

---

## Running the Tests

### Run all performance tests:
```bash
cd backend
source .venv/bin/activate
pytest tests/test_portfolio_performance.py -v
```

### Run specific test class:
```bash
# History performance
pytest tests/test_portfolio_performance.py::TestPortfolioHistoryPerformance -v

# Correctness tests
pytest tests/test_portfolio_performance.py::TestPortfolioHistoryCorrectness -v

# Phase 2 tests
pytest tests/test_portfolio_performance.py::TestPhase2EagerLoadingPerformance -v
```

### Run specific test:
```bash
pytest tests/test_portfolio_performance.py::TestPortfolioHistoryPerformance::test_get_portfolio_history_query_count -xvs
```

### See query counts and timing:
```bash
pytest tests/test_portfolio_performance.py -v -s
# -s shows print statements (query counts, timing)
```

---

## Data Requirements

**Important**: These tests require a seeded database with production-like data.

**Why**:
- Performance tests are meaningless without realistic data volume
- Query optimization only shows with multiple portfolio/funds
- Timing benchmarks need actual data to process

**Running without data**:
- Tests will skip gracefully
- Uses `pytest.skip("No portfolio found")`
- No test failures

**For CI/CD**:
- Consider separate performance test suite
- Run on staging environment with real data
- Or seed test data as part of CI setup

---

## Test Markers

All tests marked with `@pytest.mark.performance`:

```python
pytestmark = [
    pytest.mark.performance,
]
```

**Usage**:
```bash
# Run only performance tests
pytest -m performance

# Skip performance tests
pytest -m "not performance"
```

---

## Maintenance Notes

### When to Update These Tests

**Update query count targets if**:
- Adding new data relationships
- Changing service layer logic
- New features require additional queries

**Update timing targets if**:
- Running on faster/slower hardware
- Database grows significantly
- Python version changes (performance characteristics)

**Don't update to make tests pass**:
- If query count increases, investigate why
- Performance regression indicates a problem
- Fix the code, don't relax the test

### Preventing Performance Regressions

**Before merging code**:
1. Run performance tests locally
2. Check query counts haven't increased
3. Check timing hasn't regressed
4. If queries increased, explain why in PR

**In code review**:
- Watch for missing `joinedload()`
- Watch for queries in loops
- Watch for lazy loading in hot paths

---

## Related Documentation

- **Testing Infrastructure**: `tests/docs/TESTING_INFRASTRUCTURE.md`
- **Fixtures Reference**: `tests/conftest.py` (query_counter, timer)
- **Performance Tests Source**: `tests/test_portfolio_performance.py`
- **Service Code**: `app/services/portfolio_service.py`, `app/services/transaction_service.py`
