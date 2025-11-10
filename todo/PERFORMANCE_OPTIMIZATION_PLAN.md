# Performance Optimization Plan

**Status**: Planning
**Priority**: HIGH
**Version**: 1.3.2 (or 1.4.0)
**Last Updated**: 2024-11-10

---

## Executive Summary

The application suffers from severe performance issues caused by **day-by-day iteration through historical date ranges**, resulting in tens of thousands of database queries for simple page loads. The Overview page executes **16,425 queries** for 365 days of history (68,445 for full history), when the same result could be achieved with ~50 queries and in-memory processing.

**Current Performance:**
- Overview page: 5-10 seconds (16,460 queries)
- Portfolio Detail: 3-5 seconds (7,900 queries)

**Target Performance (after optimization):**
- Overview page: 0.5-1 second (50-100 queries)
- Portfolio Detail: 0.3-0.5 seconds (20-30 queries)

**Primary Bottleneck**: `portfolio_service.py` historical calculation methods that iterate through each day and query the database repeatedly instead of loading data once and processing in memory.

---

## 🔍 Investigation Findings

### Database Statistics
- **Active Portfolios**: 3
- **Portfolio Funds**: 7
- **Transactions**: 230 (spanning 1,521 days: 2021-09-06 to 2025-11-05)
- **Dividends**: 9
- **Historical Prices**: 938 unique dates
- **Historical Range**: 4.2 years

### Critical Issues Identified

#### 1. Day-by-Day Historical Processing (CRITICAL ⚠️)

**Location**: `backend/app/services/portfolio_service.py`
- `get_portfolio_history()` (lines 353-427)
- `get_portfolio_fund_history()` (lines 479-567)
- `_calculate_daily_values()` helper

**Problem**:
```python
while current_date <= end_date_to_use:
    # For EACH day:
    #   For EACH portfolio (3x):
    #     For EACH fund (7x):
    #       Query all transactions up to this date
    #       Query latest price up to this date
    #       Query realized gains up to this date
    current_date += timedelta(days=1)
```

**Query Count**:
- Per day: 3 portfolios × (1 RealizedGainLoss + 7 funds × (1 Transaction + 1 FundPrice)) = 45 queries/day
- **365 days: 16,425 queries**
- **Full history (1,521 days): 68,445 queries**

**Impact**: This is the #1 performance bottleneck by far (>95% of query load)

**Why It's Slow**:
- Database queries inside loops
- Repeated filtering of same 230 transactions, 365+ times
- No result caching between days
- Quadratic time complexity: O(days × records)

#### 2. N+1 Query Pattern in Portfolio Summary (HIGH)

**Location**: `backend/app/services/portfolio_service.py`
- `get_portfolio_summary()` (line 295-317)
- `_calculate_fund_metrics()` (line 113-200)

**Problem**: For each portfolio fund, separate queries for:
- Transactions
- Dividends
- Latest price
- Realized gains

**Query Count**: ~7 queries per fund × 7 funds = 49 queries
**Should be**: 4-5 queries total using eager loading

#### 3. N+1 Query in Transaction Formatting (MEDIUM)

**Location**: `backend/app/services/transaction_service.py` (line 77-79)

```python
# Called for EACH transaction (230 times)
ibkr_allocation = IBKRTransactionAllocation.query.filter_by(
    transaction_id=transaction.id
).first()
```

**Query Count**: 230 extra queries
**Should be**: Single query with JOIN

#### 4. Repeated RealizedGainLoss Queries (MEDIUM)

**Location**: `backend/app/services/portfolio_service.py` multiple locations

**Problem**: Same realized gains queried multiple times:
- In `_calculate_fund_metrics()` (line 166-169)
- In `_format_portfolio_summary()` (line 215)
- In `_calculate_daily_values()` (line 445-449)

### Good Patterns Found ✓

1. **Dividend Data Preloading** (line 237-280) - Batches queries to avoid N+1
2. **Composite Indexes** - Exist for date-based queries (migration 1.1.2)
3. **Frontend Smart Loading** - Loads 365 days initially, more only on zoom
4. **Single Query for Dividends** - Uses JOIN instead of N+1

---

## 📊 Query Analysis by Page

### Overview Page (`/overview`)

**Frontend**: `frontend/src/pages/Overview.js`

**API Calls**:
1. `/portfolio-summary` → ~30 queries (N+1 pattern)
2. `/portfolio-history?days=365` → **16,425 queries** (day-by-day processing)

**Total**: ~16,460 queries per page load

### Portfolio Detail Page (`/portfolios/:id`)

**Frontend**: `frontend/src/pages/PortfolioDetail.js`

**API Calls**:
1. `/portfolios/{id}` → ~10 queries
2. `/portfolios/{id}/fund-history?days=365` → **7,665 queries** (day-by-day)
3. `/transactions?portfolio_id={id}` → **231 queries** (1 + 230 N+1)
4. `/dividends/portfolio/{id}` → 1 query ✓

**Total**: ~7,900 queries per page load

---

## 🎯 Optimization Plan

### Phase 1: Eliminate Day-by-Day Processing (CRITICAL)

**Impact**: Reduce 16,425 queries → ~50 queries (99.7% reduction)

**Files to Modify**:
- `backend/app/services/portfolio_service.py`
  - `get_portfolio_history()` method
  - `get_portfolio_fund_history()` method
  - Add new `_build_historical_data_batch()` method

**Strategy**:
```python
# BEFORE (current):
while current_date <= end_date:
    # Query database for this specific date
    daily_values = calculate_daily_values(current_date)
    history.append(daily_values)
    current_date += timedelta(days=1)

# AFTER (proposed):
# 1. Load ALL data once upfront
all_transactions = Transaction.query.filter(...).order_by(Transaction.date).all()
all_prices = FundPrice.query.filter(...).order_by(FundPrice.fund_id, FundPrice.date).all()
all_gains = RealizedGainLoss.query.filter(...).all()

# 2. Build lookup dictionaries
prices_by_date_fund = {}  # {(date, fund_id): price}
for price in all_prices:
    prices_by_date_fund[(price.date, price.fund_id)] = price

# 3. Process chronologically in memory
current_shares = defaultdict(Decimal)
current_cost = defaultdict(Decimal)
history = []

for date in date_range:
    # Update state with transactions for this date (in-memory filter)
    day_transactions = [t for t in all_transactions if t.date == date]
    for t in day_transactions:
        if t.type == 'buy':
            current_shares[t.portfolio_fund_id] += t.shares
            current_cost[t.portfolio_fund_id] += t.shares * t.cost_per_share
        elif t.type == 'sell':
            current_shares[t.portfolio_fund_id] -= t.shares
            # FIFO cost basis adjustment

    # Calculate values using in-memory data
    daily_value = calculate_value(current_shares, current_cost, prices_by_date_fund, date)
    history.append(daily_value)
```

**Implementation Steps**:
1. Create new method `_load_historical_data_batch(portfolio_ids, start_date, end_date)`
   - Returns: `(transactions, prices, gains, dividends)` as lists
2. Create new method `_build_date_lookup_tables(data)`
   - Returns: `(prices_by_date_fund, gains_by_date, etc.)` as dicts
3. Refactor `get_portfolio_history()` to use batch loading
4. Refactor `get_portfolio_fund_history()` to use batch loading
5. Add unit tests to verify calculations match old method

**Testing Strategy**:
- Compare results between old and new methods for same date range
- Verify totals match across different time periods
- Test edge cases: first transaction, last transaction, gaps in data
- Performance test: measure query count and execution time

**Expected Results**:
- Queries: 16,425 → ~50 (99.7% reduction)
- Time: 5-10 seconds → 0.5-1 second

---

### Phase 2: Add Eager Loading & Batch Queries (HIGH)

**Impact**: Eliminate N+1 queries (~280 queries reduced)

#### 2.1: Portfolio Summary Eager Loading

**File**: `backend/app/services/portfolio_service.py`

**Changes**:
```python
from sqlalchemy.orm import joinedload, selectinload

# In get_portfolio_summary()
portfolios = Portfolio.query.options(
    selectinload(Portfolio.funds).joinedload(PortfolioFund.fund),
    selectinload(Portfolio.funds).selectinload(PortfolioFund.transactions),
    selectinload(Portfolio.realized_gains_losses)
).filter_by(is_archived=False, exclude_from_overview=False).all()
```

**Expected**: 49 queries → 4-5 queries

#### 2.2: Transaction IBKR Allocation Batching

**File**: `backend/app/services/transaction_service.py`

**Changes**:
```python
def get_portfolio_transactions(portfolio_id):
    # Load with eager loading
    transactions = Transaction.query.join(PortfolioFund).options(
        joinedload(Transaction.portfolio_fund).joinedload(PortfolioFund.fund),
        selectinload(Transaction.ibkr_allocation)  # Eager load
    ).filter(PortfolioFund.portfolio_id == portfolio_id).all()

    return [format_transaction(t) for t in transactions]

# OR batch load separately:
def get_portfolio_transactions(portfolio_id):
    transactions = Transaction.query.join(PortfolioFund)
        .filter(PortfolioFund.portfolio_id == portfolio_id).all()

    # Batch load IBKR allocations
    transaction_ids = [t.id for t in transactions]
    allocations = {a.transaction_id: a for a in
        IBKRTransactionAllocation.query.filter(
            IBKRTransactionAllocation.transaction_id.in_(transaction_ids)
        ).all()
    }

    return [format_transaction(t, allocations.get(t.id)) for t in transactions]
```

**Expected**: 231 queries → 2-3 queries

#### 2.3: Realized Gains Batching

**File**: `backend/app/services/portfolio_service.py`

**Changes**:
```python
# Load once per request, not per portfolio or fund
def get_portfolio_summary():
    portfolios = Portfolio.query.filter_by(...).all()
    portfolio_ids = [p.id for p in portfolios]

    # Batch load all realized gains
    all_gains = RealizedGainLoss.query.filter(
        RealizedGainLoss.portfolio_id.in_(portfolio_ids)
    ).all()

    # Group by portfolio_id and fund_id in Python
    gains_by_portfolio = defaultdict(lambda: defaultdict(list))
    for gain in all_gains:
        gains_by_portfolio[gain.portfolio_id][gain.fund_id].append(gain)

    # Use grouped data in calculations
    for portfolio in portfolios:
        portfolio_gains = gains_by_portfolio[portfolio.id]
        # Calculate using portfolio_gains
```

**Expected**: ~15 duplicate queries → 1 query

---

### Phase 3: Add Response Caching (MEDIUM)

**Impact**: Reduce repeated calculations for identical requests

**Requirements**:
- Add `flask-caching` to `requirements.txt`
- Configure cache (Redis for production, SimpleCache for development)

**Implementation**:
```python
from flask_caching import Cache

cache = Cache(config={
    'CACHE_TYPE': 'simple',  # or 'redis' for production
    'CACHE_DEFAULT_TIMEOUT': 300  # 5 minutes
})

@cache.memoize(timeout=300)
def get_portfolio_history(start_date, end_date):
    # Expensive calculation
    return history

# Cache invalidation on data changes
@portfolio_routes.route('/portfolios/<uuid:portfolio_id>', methods=['PUT'])
def update_portfolio(portfolio_id):
    # Update portfolio
    cache.delete_memoized(get_portfolio_history)  # Invalidate cache
    return response
```

**Cache Strategy**:
- Cache key includes: portfolio_id, start_date, end_date
- TTL: 5-10 minutes (balance between freshness and performance)
- Invalidate on: transaction create/update/delete, portfolio update
- Consider: LRU eviction for memory management

**Expected**: Repeated requests served from cache in <50ms

---

## 🧪 Testing Strategy

### Performance Testing

**Metrics to Track**:
1. Query count per request
2. Response time (ms)
3. Database load (queries/sec)
4. Memory usage

**Test Cases**:
1. Overview page with 3 portfolios, 365 days
2. Portfolio detail with 7 funds, 365 days
3. Full history load (1,521 days)
4. Edge case: Portfolio with 1 transaction
5. Edge case: Portfolio with gaps in price data

**Performance Benchmarks**:
```python
# Add to tests/performance/test_portfolio_performance.py
import time
from sqlalchemy import event
from sqlalchemy.engine import Engine

query_count = 0

@event.listens_for(Engine, "before_cursor_execute")
def before_cursor_execute(conn, cursor, statement, parameters, context, executemany):
    global query_count
    query_count += 1

def test_overview_performance():
    global query_count
    query_count = 0
    start = time.time()

    response = client.get('/portfolio-history?days=365')

    end = time.time()
    duration = (end - start) * 1000

    print(f"Queries: {query_count}, Time: {duration}ms")
    assert query_count < 100, f"Too many queries: {query_count}"
    assert duration < 1000, f"Too slow: {duration}ms"
```

### Correctness Testing

**Validation**:
- Compare old vs new calculation results for same date range
- Verify totals match (total value, total cost, total gain)
- Check edge cases maintain accuracy

```python
def test_calculation_accuracy():
    # Run both old and new methods
    old_result = get_portfolio_history_old(start, end)
    new_result = get_portfolio_history_new(start, end)

    # Compare daily values
    for old_day, new_day in zip(old_result, new_result):
        assert old_day['date'] == new_day['date']
        assert abs(old_day['total_value'] - new_day['total_value']) < 0.01
```

---

## 📋 Implementation Checklist

### Phase 1: Day-by-Day Processing

- [ ] Create branch: `feature/performance-phase1-batch-processing`
- [ ] Add query count logging/monitoring
- [ ] Create `_load_historical_data_batch()` method
- [ ] Create `_build_date_lookup_tables()` method
- [ ] Refactor `get_portfolio_history()` to use batch loading
- [ ] Refactor `get_portfolio_fund_history()` to use batch loading
- [ ] Add unit tests comparing old vs new results
- [ ] Add performance benchmarks
- [ ] Test with real data (verify accuracy)
- [ ] Performance test (verify <100 queries, <1s response)
- [ ] Code review
- [ ] Merge to main

### Phase 2: Eager Loading

- [ ] Create branch: `feature/performance-phase2-eager-loading`
- [ ] Add eager loading to portfolio summary
- [ ] Batch IBKR allocation queries
- [ ] Batch realized gains queries
- [ ] Add tests
- [ ] Performance test (verify query reduction)
- [ ] Code review
- [ ] Merge to main

### Phase 3: Caching (Optional)

- [ ] Create branch: `feature/performance-phase3-caching`
- [ ] Add flask-caching to requirements
- [ ] Implement cache decorator
- [ ] Add cache invalidation hooks
- [ ] Add cache hit/miss metrics
- [ ] Test cache behavior
- [ ] Configure production cache (Redis)
- [ ] Code review
- [ ] Merge to main

---

## 🎯 Success Criteria

### Performance Targets

| Metric | Current | Phase 1 Target | Phase 2 Target | Phase 3 Target |
|--------|---------|----------------|----------------|----------------|
| Overview queries | 16,460 | 100 | 60 | 60 (cached: 0) |
| Overview time (ms) | 5,000-10,000 | 800-1,200 | 500-800 | 500 (cached: <50) |
| Portfolio Detail queries | 7,900 | 50 | 30 | 30 (cached: 0) |
| Portfolio Detail time (ms) | 3,000-5,000 | 500-800 | 300-500 | 300 (cached: <50) |

### Acceptance Criteria

- [ ] Overview page loads in <1 second
- [ ] Portfolio Detail loads in <0.5 seconds
- [ ] Query count reduced by >95%
- [ ] Calculation accuracy maintained (no regressions)
- [ ] All tests passing
- [ ] No breaking changes to API
- [ ] Frontend unchanged (no modifications needed)

---

## 🚨 Risks & Mitigation

### Risk 1: Calculation Accuracy

**Risk**: In-memory processing might produce different results than database queries
**Mitigation**:
- Comprehensive unit tests comparing old vs new
- Manual verification with known datasets
- Gradual rollout with feature flag

### Risk 2: Memory Usage

**Risk**: Loading all historical data in memory might cause memory issues
**Mitigation**:
- Profile memory usage during testing
- Current data volume is small (230 transactions, 938 prices)
- Monitor production memory usage
- Add pagination if needed for very large portfolios

### Risk 3: Complex Refactoring

**Risk**: Historical calculation logic is complex, refactoring might introduce bugs
**Mitigation**:
- Small, incremental changes
- Each phase tested independently
- Keep old methods until new ones proven
- Feature flag to switch between implementations

### Risk 4: Cache Invalidation

**Risk**: Stale cache data if invalidation logic has bugs
**Mitigation**:
- Conservative TTL (5 minutes)
- Comprehensive invalidation hooks
- Add cache version/timestamp
- Manual cache clear endpoint for emergencies

---

## 📚 References

### Code Locations

**Backend Files**:
- `backend/app/services/portfolio_service.py` - Main optimization target
- `backend/app/services/transaction_service.py` - Transaction N+1 fix
- `backend/app/routes/portfolio_routes.py` - API endpoints

**Database Schema**:
- `backend/app/models/portfolio.py` - Portfolio model
- `backend/app/models/transaction.py` - Transaction model
- `backend/app/models/fund.py` - Fund and FundPrice models
- `backend/migrations/versions/1.1.2_indexes.py` - Existing indexes

**Frontend Files**:
- `frontend/src/pages/Overview.js` - Overview page
- `frontend/src/pages/PortfolioDetail.js` - Portfolio detail page
- `frontend/src/hooks/useChartData.js` - Chart data loading hook

### Related Documentation

- [ARCHITECTURE.md](../docs/ARCHITECTURE.md) - System architecture
- [MODELS.md](../docs/MODELS.md) - Database models
- [DATABASE.md](../docs/DATABASE.md) - Database schema

---

## 📈 Monitoring & Rollout

### Metrics to Monitor

**Application Metrics**:
- API response times (p50, p95, p99)
- Query count per request
- Database connection pool usage
- Memory usage
- Cache hit/miss rates (Phase 3)

**Database Metrics**:
- Queries per second
- Slow query log
- Lock contention
- Connection count

### Rollout Strategy

1. **Development**: Test with local data
2. **Staging**: Deploy and test with production-like data
3. **Production (Canary)**: Deploy to 10% of users, monitor metrics
4. **Production (Full)**: Deploy to 100% if metrics are good
5. **Rollback Plan**: Keep old implementation available via feature flag

### Success Validation

**Day 1**: Monitor error rates, response times
**Day 3**: Analyze performance metrics, user feedback
**Week 1**: Validate database load reduction
**Week 2**: Remove old implementation if successful

---

**Next Steps**: Begin Phase 1 implementation when ready to optimize performance.
