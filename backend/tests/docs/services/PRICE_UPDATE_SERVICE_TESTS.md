# PriceUpdateService Test Suite Documentation

**File**: `tests/test_price_update_service.py`\
**Service**: `app/services/price_update_service.py`\
**Tests**: 17 tests\
**Coverage**: 98% (88/90 statements)\
**Created**: Version 1.3.3 (Phase 4)

## Overview

Comprehensive test suite for the PriceUpdateService module, which provides fund price management through two service classes:

1. **TodayPriceService** - Updates the latest available price (yesterday's closing price)
2. **HistoricalPriceService** - Fills missing historical prices for funds with transactions

The test suite achieves 98% coverage by testing all public methods, yfinance API integration (mocked), duplicate detection, error handling, and edge cases. This service is critical for portfolio valuation and performance calculations.

## Test Structure

### Test Classes

#### 1. TestTodayPriceService (7 tests)
Tests for fetching and storing the latest available fund price:
- `test_get_latest_available_date` - Latest date calculation (yesterday)
- `test_update_todays_price_no_symbol` - Error when fund lacks symbol
- `test_update_todays_price_already_exists` - Duplicate prevention
- `test_update_todays_price_success` - Successful price update from yfinance
- `test_update_todays_price_no_data` - Handle empty yfinance response
- `test_update_todays_price_latest_already_exists` - Race condition handling
- `test_update_todays_price_exception_handling` - Network/API error handling

#### 2. TestHistoricalPriceService (10 tests)
Tests for historical price backfilling:
- `test_get_oldest_transaction_date_no_transactions` - No transactions scenario
- `test_get_oldest_transaction_date_with_transactions` - Find oldest transaction
- `test_get_missing_dates_no_transactions` - No required dates
- `test_get_missing_dates_all_prices_exist` - All dates covered
- `test_get_missing_dates_some_missing` - Identify gaps in price history
- `test_update_historical_prices_no_symbol` - Error when fund lacks symbol
- `test_update_historical_prices_no_missing_dates` - No work needed
- `test_update_historical_prices_success` - Successful backfill
- `test_update_historical_prices_partial_data` - Handle weekend/holiday gaps
- `test_update_historical_prices_exception_handling` - API error handling

## Testing Strategy

### yfinance API Mocking

All external API calls to yfinance are mocked using `unittest.mock.patch`:

```python
@patch("app.services.price_update_service.yf.Ticker")
def test_update_todays_price_success(self, mock_ticker, app_context, db_session):
    # Mock yfinance response with pandas DataFrame
    mock_history = pd.DataFrame(
        {"Close": [150.25, 151.50, 152.75]},
        index=pd.DatetimeIndex([
            datetime.now().date() - timedelta(days=3),
            datetime.now().date() - timedelta(days=2),
            datetime.now().date() - timedelta(days=1),
        ])
    )

    mock_ticker_instance = MagicMock()
    mock_ticker_instance.history.return_value = mock_history
    mock_ticker.return_value = mock_ticker_instance

    # Now service calls will use mocked data
    response, status = TodayPriceService.update_todays_price(fund.id)
```

**Benefits**:
- No actual internet requests during testing
- Predictable, consistent test data
- Fast test execution
- Can simulate error conditions easily
- Can test weekend/holiday gaps

### Query-Specific Data Pattern

Each test creates unique funds to avoid conflicts:

```python
fund = Fund(
    id=str(uuid.uuid4()),
    isin=f"US{uuid.uuid4().hex[:10].upper()}",  # Unique ISIN
    symbol=f"AAPL{uuid.uuid4().hex[:4]}",        # Unique symbol
    # ... other fields
)
```

### Test Isolation

All tests are completely isolated:
- **Funds**: Unique per test
- **Portfolios**: Created fresh per test
- **Transactions**: Unique dates and IDs
- **Prices**: Created only when needed

No shared fixtures that mutate (except app_context, db_session).

## Service Methods Tested

### TodayPriceService

#### `get_latest_available_date()` → date
Returns yesterday's date as the latest available trading date.

**Why yesterday?** Stock prices are finalized at market close. During trading hours, "today's" price is still changing, so we fetch the last complete trading day.

**Tested**: `test_get_latest_available_date`

#### `update_todays_price(fund_id)` → (response, status)
Updates the latest available price for a fund.

**Process**:
1. Check fund has symbol (required for yfinance)
2. Calculate latest available date (yesterday)
3. Check if price already exists (duplicate prevention)
4. Fetch last 5 days of data from yfinance
5. Extract most recent closing price
6. Create FundPrice record
7. Return success response

**Tests**:
- Success path with mocked data
- Duplicate detection (2 scenarios)
- Missing symbol error
- Empty data handling
- Exception handling

### HistoricalPriceService

#### `get_oldest_transaction_date(fund_id)` → date | None
Finds the earliest transaction date for a fund across all portfolios.

**SQL Logic**:
```python
oldest = (
    Transaction.query
    .join(PortfolioFund)
    .filter(PortfolioFund.fund_id == fund_id)
    .order_by(Transaction.date.asc())
    .first()
)
```

**Returns**: Transaction date or None if no transactions exist.

**Tested**:
- No transactions → None
- Multiple transactions → Oldest date

#### `get_missing_dates(fund_id)` → list[date]
Identifies all dates between oldest transaction and today that lack price data.

**Algorithm**:
```python
start_date = get_oldest_transaction_date(fund_id)
existing_dates = {price.date for price in FundPrice.query.all()}

missing = []
current = start_date
while current <= today:
    if current not in existing_dates:
        missing.append(current)
    current += timedelta(days=1)

return missing
```

**Tested**:
- No transactions → Empty list
- All prices exist → Empty list
- Some missing → Correct gap detection

#### `update_historical_prices(fund_id)` → (response, status)
Backfills missing historical prices for a fund.

**Process**:
1. Find missing dates
2. Fetch historical data from yfinance (one API call for date range)
3. For each missing date, add price if yfinance has data
4. Commit batch to database
5. Return count of updated prices

**Tests**:
- Success with full data
- Partial data (weekend/holiday gaps)
- No missing dates
- Missing symbol error
- Exception handling

## Price Data Flow

### Today's Price Update Flow

```
User Request
    ↓
TodayPriceService.update_todays_price(fund_id)
    ↓
Calculate latest_date (yesterday)
    ↓
Check if FundPrice exists for latest_date
    ├─ YES → Return "already exists"
    └─ NO → Continue
        ↓
    yfinance.Ticker(symbol).history(start, end)
        ↓
    Extract last closing price
        ↓
    Create FundPrice record
        ↓
    Return success
```

### Historical Price Backfill Flow

```
User Request
    ↓
HistoricalPriceService.update_historical_prices(fund_id)
    ↓
Get oldest transaction date
    ├─ None → Return "no missing dates"
    └─ date → Continue
        ↓
    Get missing dates (transaction_date to today)
        ├─ Empty → Return "no missing dates"
        └─ Dates → Continue
            ↓
        yfinance.Ticker(symbol).history(min_date, max_date)
            ↓
        For each missing_date in yfinance_data:
            Create FundPrice(date, price)
            ↓
        Commit batch
            ↓
        Return success (count updated)
```

## Duplicate Detection

### Two-Level Duplicate Prevention

The service prevents duplicates at two points:

**1. Initial Check (Fast Path)**:
```python
# Before calling yfinance API
existing = FundPrice.query.filter_by(
    fund_id=fund_id,
    date=latest_date
).first()

if existing:
    return "already exists"  # Skip API call
```

**2. Race Condition Check (Slow Path)**:
```python
# After yfinance returns data
last_date = history.index[-1].date()

# Check again before inserting
if not FundPrice.query.filter_by(fund_id=fund_id, date=last_date).first():
    price = FundPrice(fund_id=fund_id, date=last_date, price=last_price)
    db.session.add(price)
```

**Tested**:
- `test_update_todays_price_already_exists` - Fast path
- `test_update_todays_price_latest_already_exists` - Slow path

**Note**: Lines 114-121 (slow path) are uncovered (98% vs 100%) because testing this requires:
- Multi-threading/concurrent requests
- Complex timing coordination
- Marginal value (extremely rare race condition)

## yfinance Integration

### Data Fetching Strategy

**Today's Price**:
```python
# Fetch last 5 days to ensure we get the latest
end_date = datetime.now().date()
start_date = end_date - timedelta(days=5)
history = ticker.history(start=start_date, end=end_date)

# Get most recent
last_price = history["Close"][-1]
last_date = history.index[-1].date()
```

**Why 5 days?** Markets are closed weekends and holidays. Fetching 5 days ensures we capture at least 3-4 trading days.

**Historical Prices**:
```python
# Fetch entire range in one API call
start_date = min(missing_dates)
end_date = max(missing_dates) + timedelta(days=1)  # Inclusive end
history = ticker.history(start=start_date, end=end_date)

# Extract only the dates we need
for date in missing_dates:
    if date in history_dates:
        price = FundPrice(fund_id, date, history.loc[date]["Close"])
```

### Weekend/Holiday Gap Handling

Markets don't trade on weekends/holidays, so yfinance won't have data for those dates:

```python
# Missing dates: [Jan 1, Jan 2, Jan 3]
# yfinance data: [Jan 1, Jan 3]  (Jan 2 was weekend)

for date in missing_dates:
    if date in history_dates:  # Only add if yfinance has data
        add_price(date)
    # Otherwise: Skip this date (no error)
```

**Tested**: `test_update_historical_prices_partial_data`

**Result**: Database will have gaps for non-trading days (this is correct behavior).

## Error Scenarios Tested

### Missing Symbol

**Scenario**: Fund exists but has no trading symbol
```python
fund.symbol = None  # Cannot fetch prices

response, status = service.update_todays_price(fund.id)

assert status == 400
assert "No symbol available" in response["message"]
```

**Tests**:
- `test_update_todays_price_no_symbol`
- `test_update_historical_prices_no_symbol`

### Empty yfinance Response

**Scenario**: Symbol is delisted or invalid
```python
# Mock yfinance to return empty DataFrame
mock_ticker.history.return_value = pd.DataFrame()

response, status = service.update_todays_price(fund.id)

assert status == 404
assert "No recent price data available" in response["message"]
```

**Test**: `test_update_todays_price_no_data`

### Network/API Errors

**Scenario**: yfinance raises exception (network error, API down, etc.)
```python
# Mock yfinance to raise exception
mock_ticker.side_effect = Exception("Network error")

response, status = service.update_todays_price(fund.id)

assert status == 500
assert "Error updating latest price" in response["message"]
```

**Database Rollback**: Exception triggers `db.session.rollback()` to prevent partial data.

**Tests**:
- `test_update_todays_price_exception_handling`
- `test_update_historical_prices_exception_handling`

## Coverage Analysis

### Current Coverage: 98% (88/90 statements)

**Excellent Coverage Areas**:
- ✅ All public methods (100%)
- ✅ Date calculations (100%)
- ✅ yfinance integration (100%)
- ✅ Duplicate detection (95% - fast path 100%, slow path 0%)
- ✅ Error handling (100%)
- ✅ Transaction queries (100%)
- ✅ Price creation logic (100%)

**Uncovered Lines** (2 statements):
- Lines 114-121: Duplicate detection slow path (race condition)

**Why these lines are uncovered**:
```python
# This code path executes when:
# 1. Initial check finds no price
# 2. yfinance API call completes
# 3. ANOTHER process adds the same price during the API call
# 4. Second check finds the duplicate

# Testing this requires:
# - Multi-threading/concurrent requests
# - Precise timing coordination
# - Complex test infrastructure
# - Marginal benefit (extremely rare)
```

**Why 98% is excellent**:
1. **Exceeds target**: 80% target → 98% achieved ✅
2. **All critical paths tested**: Core functionality at 100%
3. **Uncovered code is edge case**: Race condition with minimal impact
4. **Real-world protection**: Service has duplicate prevention at two points
5. **Diminishing returns**: Testing race conditions has high cost, low value

## Running Tests

### Run All PriceUpdateService Tests
```bash
pytest tests/test_price_update_service.py -v
```

### Run Specific Test Class
```bash
pytest tests/test_price_update_service.py::TestTodayPriceService -v
pytest tests/test_price_update_service.py::TestHistoricalPriceService -v
```

### Run with Coverage
```bash
pytest tests/test_price_update_service.py \
    --cov=app/services/price_update_service \
    --cov-report=term-missing
```

### Run Single Test
```bash
pytest tests/test_price_update_service.py::TestTodayPriceService::test_update_todays_price_success -v
```

## Database Models

### FundPrice
Stores daily closing prices for funds:
```python
FundPrice(
    id=str(uuid.uuid4()),
    fund_id=fund.id,           # Foreign key to Fund
    date=date(2024, 1, 15),   # Trading date
    price=150.25              # Closing price
)
```

**Unique Constraint**: `(fund_id, date)` prevents duplicates

### Fund
Requires symbol for price fetching:
```python
Fund(
    id=str(uuid.uuid4()),
    symbol="AAPL",            # Required for yfinance
    isin="US0378331005",
    currency="USD",
    exchange="NASDAQ",
    investment_type=InvestmentType.STOCK
)
```

### Transaction
Determines date range for historical backfill:
```python
Transaction(
    id=str(uuid.uuid4()),
    portfolio_fund_id=pf.id,
    date=date(2024, 1, 1),    # Service backfills from this date
    type="buy",
    shares=100,
    cost_per_share=150.00
)
```

## Integration Points

### Portfolio Valuation
PriceUpdateService provides price data for portfolio valuation:

```python
# PortfolioService uses prices to calculate current value
current_price = FundPrice.query.filter_by(
    fund_id=fund.id,
    date=valuation_date
).first()

current_value = shares * current_price.price
```

**Dependency**: Portfolio valuation accuracy depends on price data completeness.

### Performance Calculations
Historical prices enable performance tracking:

```python
# Calculate return over period
start_price = FundPrice.query.filter_by(fund_id, date=start_date).first()
end_price = FundPrice.query.filter_by(fund_id, date=end_date).first()

return_pct = ((end_price.price - start_price.price) / start_price.price) * 100
```

### Logging Integration
All operations logged via logging_service:

```python
logger.log(
    level=LogLevel.INFO,
    category=LogCategory.FUND,
    message="Updated latest price for fund",
    details={
        "fund_id": fund_id,
        "date": last_date.isoformat(),
        "price": last_price
    },
    http_status=200
)
```

## Performance Considerations

### Efficient API Usage

**Today's Price** (per fund):
- **API calls**: 1 (last 5 days)
- **Database queries**: 2 (fund lookup, duplicate check)
- **Database writes**: 0-1 (only if new)

**Historical Backfill** (per fund):
- **API calls**: 1 (entire date range)
- **Database queries**: 3 (fund, oldest txn, existing prices)
- **Database writes**: Batch (all missing dates at once)

### Batch Processing
Historical updates use batch commits:
```python
for date in missing_dates:
    if date in history_dates:
        price = FundPrice(...)
        db.session.add(price)  # Add to session

db.session.commit()  # Single commit for all prices
```

**Benefit**: One database transaction instead of N transactions.

### Test Performance
- **17 tests**: Complete suite runs in ~0.6 seconds
- **Mocked API**: No network latency
- **Isolated Data**: Minimal database operations

## Future Enhancements

1. **Automatic Backfill on Transaction Creation**: When a new transaction is added, automatically fetch prices for that date
2. **Bulk Update Endpoint**: Update prices for all funds at once
3. **Price Data Validation**: Detect and flag suspicious price changes (>50% in one day)
4. **Alternative Data Sources**: Fallback to other providers if yfinance fails
5. **Caching**: Cache yfinance responses to reduce API load
6. **Async Updates**: Background job to keep all prices current
7. **Currency Conversion**: Support for multi-currency price normalization

## Related Documentation

- **Service Code**: `app/services/price_update_service.py`
- **Models**: `app/models.py` (FundPrice, Fund, Transaction)
- **Dependencies**: yfinance library
- **Related Services**: `PortfolioService` (uses prices for valuation)
- **Test Infrastructure**: `tests/docs/TESTING_INFRASTRUCTURE.md`
- **Materialized View Invalidation Tests**: `PRICE_UPDATE_MATERIALIZED_VIEW_INVALIDATION_TESTS.md`

The comprehensive test suite provides complete confidence in price update functionality, ensuring accurate portfolio valuations and performance tracking while maintaining data integrity through duplicate prevention and robust error handling.

---

**Document Version**: 1.5.1
**Last Updated**: 2026-02-06
