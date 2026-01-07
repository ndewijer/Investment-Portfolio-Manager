# Multi-Currency Support Implementation Plan

**Status**: Planning Phase
**Priority**: High
**Complexity**: Large Feature Enhancement
**Estimated Effort**: 3 weeks

---

## Overview

Implement comprehensive multi-currency support with user-configurable base currency, allowing users to view portfolio aggregates in their preferred currency while maintaining native currency storage and display for individual funds.

---

## Core Principles

1. **Store in native currency**: All transactions stored in fund's native currency
2. **Display in base currency**: Portfolio/Overview aggregates converted to user's base currency
3. **Fund pages stay native**: Individual fund pages always show in fund's currency
4. **Frontend conversion**: Use most recent exchange rates for display conversion
5. **Efficient data management**: Cleanup unused exchange rates and symbol info

---

## Current State

- All funds/stocks have a `currency` field
- Exchange rate table exists in database
- Symbol info table exists (gets populated during typing in forms)
- All current data is EUR-based
- Number format preference exists (European vs US)
- IBKRInbox already uses `formatCurrencyWithCode(value, item.currency)`

---

## Phase 1: User Base Currency Configuration

### 1.1 Backend - Add Base Currency to Config

**File**: `backend/app/models.py` or create new `UserPreferences` model

**Changes**:
- Add `base_currency` field (default: 'EUR')
- Store alongside other preferences

**Database Schema**:
```sql
CREATE TABLE user_preferences (
    id VARCHAR PRIMARY KEY,
    base_currency VARCHAR(3) DEFAULT 'EUR',
    number_format VARCHAR(20) DEFAULT 'european',
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

**File**: `backend/app/routes/config_routes.py` (or system routes)

**New Endpoints**:
- `GET /config/preferences`
  - Returns: `{base_currency: 'EUR', number_format: 'european'}`
- `PUT /config/preferences`
  - Updates preferences
  - Body: `{base_currency: 'USD', number_format: 'us'}`

### 1.2 Frontend - Base Currency Setting

**File**: `frontend/src/pages/Config.js`

**Changes**:
- Add currency selector in User Preferences section
- Options: EUR, USD, GBP, CHF, JPY, CAD, AUD (major currencies)
- Save to backend when changed
- Update FormatContext after save

**UI Mockup**:
```
User Preferences
├── Number Format: [European ▼] / US
└── Base Currency: [EUR ▼]
    Options: EUR, USD, GBP, CHF, JPY, CAD, AUD, ...
```

**File**: `frontend/src/context/FormatContext.js`

**Changes**:
- Add `baseCurrency` state (fetched from backend)
- Add `setBaseCurrency` function
- Fetch preferences on mount
- Export in context value

---

## Phase 2: Currency Conversion System

### 2.1 Backend - Exchange Rate API Enhancement

**File**: `backend/app/routes/exchange_rate_routes.py`

**New Endpoints**:

1. `GET /exchange-rates/convert`
   - Query params: `from=USD&to=EUR&date=2024-01-15` (date optional)
   - Returns: `{rate: 0.92, from: 'USD', to: 'EUR', date: '2024-01-15'}`
   - If date is None, use most recent rate

2. `GET /exchange-rates/latest`
   - Returns latest rates for all active currencies
   - Response:
   ```json
   {
     "base": "EUR",
     "date": "2024-01-15",
     "rates": {
       "USD": 1.09,
       "GBP": 0.85,
       "CHF": 0.94,
       ...
     }
   }
   ```

**File**: `backend/app/services/exchange_rate_service.py`

**New Methods**:
- `get_conversion_rate(from_currency, to_currency, date=None)`
  - If date is None, use most recent rate
  - Returns rate or 1.0 if same currency
  - Handles inverse rates (if USD->EUR exists, can calculate EUR->USD)
- `get_latest_rates(base_currency='EUR')`
  - Returns all active currency rates
  - Cache in memory for performance (invalidate hourly)

**Performance Optimization**:
- Add in-memory caching for exchange rates (TTL: 1 hour)
- Add database index: `CREATE INDEX idx_exchange_rates_lookup ON exchange_rates(from_currency, to_currency, date DESC)`

### 2.2 Frontend - Conversion Utility

**File**: `frontend/src/utils/currencyConversion.js` (NEW)

```javascript
/**
 * Currency conversion utilities
 */

let cachedRates = null;
let cacheTimestamp = null;
const CACHE_DURATION = 3600000; // 1 hour

export const fetchExchangeRates = async () => {
  const now = Date.now();
  if (cachedRates && cacheTimestamp && (now - cacheTimestamp) < CACHE_DURATION) {
    return cachedRates;
  }

  const response = await api.get('/exchange-rates/latest');
  cachedRates = response.data;
  cacheTimestamp = now;
  return cachedRates;
};

export const convertValue = (amount, fromCurrency, toCurrency, exchangeRates) => {
  if (fromCurrency === toCurrency) return amount;

  const rate = exchangeRates.rates[toCurrency] / exchangeRates.rates[fromCurrency];
  return amount * rate;
};

export const refreshRates = () => {
  cachedRates = null;
  cacheTimestamp = null;
};
```

**File**: `frontend/src/context/FormatContext.js`

**New State & Methods**:
```javascript
const [exchangeRates, setExchangeRates] = useState(null);

useEffect(() => {
  if (baseCurrency) {
    fetchExchangeRates().then(setExchangeRates);
  }
}, [baseCurrency]);

const convertAndFormat = (value, fromCurrency) => {
  if (!exchangeRates || fromCurrency === baseCurrency) {
    return formatCurrencyWithCode(value, fromCurrency);
  }

  const converted = convertValue(value, fromCurrency, baseCurrency, exchangeRates);
  return formatCurrencyWithCode(converted, baseCurrency);
};

// Export in context value
return (
  <FormatContext.Provider value={{
    ...existing,
    baseCurrency,
    setBaseCurrency,
    exchangeRates,
    convertAndFormat,
  }}>
    {children}
  </FormatContext.Provider>
);
```

---

## Phase 3: Apply Conversions to Aggregate Views

### 3.1 Overview.js

**Current Behavior**: Shows mixed currencies or EUR by default

**New Behavior**:
1. Fetch exchange rates on mount
2. For each portfolio, convert all fund values to baseCurrency before summing
3. Display all values in baseCurrency
4. Show currency indicator: "💱 All values in EUR"

**Changes**:
```javascript
const { convertAndFormat, baseCurrency, exchangeRates } = useFormat();

// Convert portfolio totals
const portfolioTotalInBase = portfolio.funds.reduce((sum, fund) => {
  const convertedValue = convertValue(
    fund.current_value,
    fund.currency,
    baseCurrency,
    exchangeRates
  );
  return sum + convertedValue;
}, 0);

// Display
formatCurrencyWithCode(portfolioTotalInBase, baseCurrency)
```

**UI Addition**:
```jsx
<div className="currency-indicator">
  💱 All values displayed in {baseCurrency}
</div>
```

### 3.2 PortfolioDetail.js

**Current Behavior**: Shows fund native currencies

**New Behavior**:
- Add toggle: "Show in [EUR] / Show native currencies"
- When toggled to baseCurrency:
  - Convert all fund values to baseCurrency
  - Convert all transaction values to baseCurrency
  - Convert all dividend values to baseCurrency
- When toggled to native:
  - Show original currency for each fund (current behavior)

**UI Addition**:
```jsx
<div className="currency-toggle">
  <button onClick={() => setShowNative(!showNative)}>
    {showNative ? `Show in ${baseCurrency}` : 'Show Native Currencies'}
  </button>
</div>
```

### 3.3 PortfolioSummary.js

**Current Behavior**: Shows aggregates

**New Behavior**:
- Convert all fund values to baseCurrency before summing
- Display total in baseCurrency

**Changes**:
```javascript
const { convertValue, baseCurrency, exchangeRates } = useFormat();

// Convert before summing
const totalValue = portfolio.funds.reduce((sum, fund) => {
  const converted = convertValue(fund.value, fund.currency, baseCurrency, exchangeRates);
  return sum + converted;
}, 0);

formatCurrencyWithCode(totalValue, baseCurrency)
```

---

## Phase 4: Keep Native Currency Display

### 4.1 FundDetail.js

**No changes needed**
- Always display in `fund.currency`
- Price history in native currency
- This page never does conversion

**Clarification**:
```javascript
// Always use fund currency
formatCurrencyWithCode(fund.latest_price, fund.currency)
```

### 4.2 Funds.js (List Page)

**No changes needed**
- Show native currency for each fund
- No aggregation happening here

---

## Phase 5: Backend Currency Data Enhancement

### 5.1 Add Currency to API Responses

**File**: `backend/app/routes/portfolio_routes.py`

#### Endpoint: `GET /portfolio/{id}/funds`

**Current Response**:
```json
{
  "id": "...",
  "fund_name": "Vanguard Total Stock",
  "shares": 100,
  "latest_price": 150.00
}
```

**New Response**:
```json
{
  "id": "...",
  "fund_name": "Vanguard Total Stock",
  "currency": "USD",  // ← ADD THIS
  "shares": 100,
  "latest_price": 150.00
}
```

**Code Change**:
```python
# In get_portfolio_funds()
return jsonify([
    {
        "id": pf.id,
        "fund_name": pf.fund.name,
        "currency": pf.fund.currency,  # ADD THIS LINE
        "shares": pf.shares,
        ...
    }
    for pf in portfolio_funds
])
```

#### Endpoint: `GET /portfolio/{id}/transactions`

**Add** `fund_currency` field:
```python
return jsonify([
    {
        "id": t.id,
        "fund_currency": t.portfolio_fund.fund.currency,  # ADD THIS
        "price_per_share": t.price_per_share,
        ...
    }
    for t in transactions
])
```

#### Endpoint: `GET /portfolio/{id}/dividends`

**Add** `fund_currency` field:
```python
return jsonify([
    {
        "id": d.id,
        "fund_currency": d.portfolio_fund.fund.currency,  # ADD THIS
        "amount": d.amount,
        ...
    }
    for d in dividends
])
```

---

## Phase 6: Exchange Rate Management

### 6.1 Smart Currency Import

**File**: `backend/app/services/exchange_rate_service.py`

**Strategy**: Only import rates for active currencies + major currencies

**Implementation**:
```python
def get_active_currencies():
    """Get list of currencies used in funds"""
    result = db.session.query(Fund.currency).distinct().all()
    currencies = {row[0] for row in result}

    # Add major currencies
    major_currencies = {'EUR', 'USD', 'GBP', 'CHF', 'JPY', 'CAD', 'AUD'}
    currencies.update(major_currencies)

    return list(currencies)

def import_exchange_rates_for_active_currencies(date=None):
    """Only import rates for currencies we care about"""
    active_currencies = get_active_currencies()

    # Import rates only for active currencies
    # (existing import logic, but filtered)
```

**Trigger**:
- Run daily for active currencies
- When IBKR import adds new currency fund, trigger rate import for that currency

### 6.2 Exchange Rate Cleanup Job

**File**: `backend/app/services/maintenance_service.py` (NEW)

**Create Maintenance Service**:
```python
"""
Maintenance tasks for database cleanup
"""
from datetime import datetime, timedelta
from ..models import ExchangeRate, Fund, SymbolInfo

class MaintenanceService:
    @staticmethod
    def cleanup_exchange_rates():
        """Remove unused exchange rates and old data"""
        # Get active currencies
        active_currencies = get_active_currencies()

        # Delete rates for currencies NOT in active list
        deleted = ExchangeRate.query.filter(
            ~ExchangeRate.from_currency.in_(active_currencies),
            ~ExchangeRate.to_currency.in_(active_currencies)
        ).delete()

        # Keep only last 30 days for active currencies
        cutoff_date = datetime.now() - timedelta(days=30)
        ExchangeRate.query.filter(
            ExchangeRate.date < cutoff_date
        ).delete()

        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message=f"Cleaned up {deleted} unused exchange rates"
        )

    @staticmethod
    def cleanup_symbol_info():
        """Remove symbol info not associated with any fund"""
        # Get active symbols
        active_symbols = db.session.query(Fund.symbol).filter(
            Fund.symbol.isnot(None)
        ).distinct().all()
        active_symbols = {row[0] for row in active_symbols}

        # Delete symbols NOT in active list
        deleted = SymbolInfo.query.filter(
            ~SymbolInfo.symbol.in_(active_symbols)
        ).delete()

        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message=f"Cleaned up {deleted} unused symbol info entries"
        )
```

**Schedule**: Add to cron or scheduled task

**File**: `backend/app/__init__.py` or scheduler config

```python
from apscheduler.schedulers.background import BackgroundScheduler

scheduler = BackgroundScheduler()

# Run every Sunday at 3:00 AM
@scheduler.scheduled_job('cron', day_of_week='sun', hour=3)
def weekly_maintenance():
    MaintenanceService.cleanup_exchange_rates()
    MaintenanceService.cleanup_symbol_info()

scheduler.start()
```

---

## Phase 7: Symbol Info Cleanup

### 7.1 Prevent Garbage Accumulation

**File**: `backend/app/routes/fund_routes.py`

**Strategy 1**: Don't cache unsuccessful lookups

**Current Behavior**:
- Symbol lookup during typing saves to database
- Creates garbage entries

**New Behavior**:
- Only save symbol_info after successful fund creation
- During typing/lookup in forms, return data but don't persist
- Add `cache=False` parameter to lookup endpoint

**Changes**:
```python
@funds.route("/lookup-symbol-info/<string:symbol>", methods=["GET"])
def lookup_symbol_info(symbol):
    force_refresh = request.args.get("force_refresh", "false").lower() == "true"
    cache = request.args.get("cache", "true").lower() == "true"  # ADD THIS

    # If cache=False, don't save to database
    symbol_info = SymbolLookupService.get_symbol_info(
        symbol,
        force_refresh=force_refresh,
        save_to_db=cache  # ADD THIS
    )

    return jsonify(symbol_info)
```

**Frontend Change** (`frontend/src/pages/Funds.js`):
```javascript
// During typing (line ~189)
const response = await api.get(`/lookup-symbol-info/${symbol}?cache=false`);

// After fund creation success
await api.get(`/lookup-symbol-info/${fund.symbol}?cache=true&force_refresh=true`);
```

### 7.2 Weekly Cleanup (Already in Phase 6.2)

Runs weekly as part of maintenance job.

---

## Phase 8: Migration to formatCurrencyWithCode

### 8.1 Frontend Component Updates

**Files to Update**:

#### 1. Overview.js (13 usages)
```javascript
// Before
formatCurrency(value)

// After
formatCurrencyWithCode(portfolioTotalInBase, baseCurrency)
// or for individual funds
convertAndFormat(fund.value, fund.currency)
```

#### 2. FundDetail.js (1 usage)
```javascript
// Before
formatCurrency(price.price)

// After
formatCurrencyWithCode(price.price, fund.currency)
```

#### 3. PortfolioSummary.js (6 usages)
```javascript
// Before
formatCurrency(totalValue)

// After
formatCurrencyWithCode(convertedTotal, baseCurrency)
```

#### 4. FundsTable.js (10 usages)
```javascript
// Before
formatCurrency(fund.latest_price)

// After (with toggle)
showNative
  ? formatCurrencyWithCode(fund.latest_price, fund.currency)
  : convertAndFormat(fund.latest_price, fund.currency)
```

#### 5. TransactionsTable.js (4 usages)
```javascript
// Before
formatCurrency(transaction.cost_per_share)

// After
showNative
  ? formatCurrencyWithCode(transaction.cost_per_share, transaction.fund_currency)
  : convertAndFormat(transaction.cost_per_share, transaction.fund_currency)
```

#### 6. DividendsTable.js (4 usages)
```javascript
// Before
formatCurrency(dividend.amount)

// After
showNative
  ? formatCurrencyWithCode(dividend.amount, dividend.fund_currency)
  : convertAndFormat(dividend.amount, dividend.fund_currency)
```

#### 7. ValueChart.js (5 usages)
```javascript
// Add currency prop
<ValueChart
  data={data}
  lines={lines}
  currency={fund?.currency || baseCurrency}  // ADD THIS
/>

// Inside component
const { formatCurrencyWithCode } = useFormat();

// Use in tooltip and axes
formatCurrencyWithCode(value, currency)  // Use prop
```

---

## Implementation Recommendation: Frontend vs Backend Conversion

### ✅ Option A: Frontend Conversion (RECOMMENDED)

**Pros**:
- Backend stays simple, returns raw data
- Frontend has full control over display
- Can easily toggle between native and converted
- No backend computation for every request
- User preference handled client-side

**Cons**:
- Need to fetch exchange rates separately
- Conversion logic in frontend

**Implementation**:
1. Backend: Add `/exchange-rates/latest` endpoint returning all active rates
2. Frontend: Fetch rates once on app load, store in context
3. Frontend: Convert values as needed for display
4. Refresh rates periodically (every hour or on navigation)

### ❌ Option B: Backend Conversion

**Pros**:
- Single source of truth
- Can optimize queries with currency conversion

**Cons**:
- Backend needs to know user's base currency for every request
- Less flexible for toggling display
- More backend computation
- Requires passing preference in every API call

**Decision**: **Use Option A (Frontend Conversion)**

---

## UI Enhancements

### Currency Indicators

**Overview Page**:
```jsx
<div className="currency-indicator">
  💱 All values displayed in {baseCurrency}
</div>
```

**Portfolio Detail Page**:
```jsx
<div className="currency-toggle">
  <label>
    <input
      type="checkbox"
      checked={!showNative}
      onChange={() => setShowNative(!showNative)}
    />
    Display all values in {baseCurrency}
  </label>
</div>
```

**Fund Detail Page**:
```jsx
<div className="currency-note">
  (All values in {fund.currency})
</div>
```

### Config Page Enhancement

**Location**: User Preferences section

```jsx
<div className="form-field">
  <label>Base Currency</label>
  <select
    value={baseCurrency}
    onChange={(e) => handleCurrencyChange(e.target.value)}
  >
    <option value="EUR">EUR - Euro (€)</option>
    <option value="USD">USD - US Dollar ($)</option>
    <option value="GBP">GBP - British Pound (£)</option>
    <option value="CHF">CHF - Swiss Franc</option>
    <option value="JPY">JPY - Japanese Yen (¥)</option>
    <option value="CAD">CAD - Canadian Dollar (C$)</option>
    <option value="AUD">AUD - Australian Dollar (A$)</option>
  </select>
  <small>Currency used for portfolio totals and aggregated values</small>
</div>
```

---

## Testing Checklist

### Functional Testing

- [ ] Create fund in EUR, verify display
- [ ] Create fund in USD, verify display
- [ ] Create fund in CHF, verify display
- [ ] Verify Overview shows all portfolios in base currency
- [ ] Verify Portfolio Detail shows correct totals in base currency
- [ ] Toggle native currency display on Portfolio Detail
- [ ] Verify Fund Detail always shows native currency
- [ ] Change base currency from EUR to USD
  - [ ] Verify Overview updates
  - [ ] Verify Portfolio Summary updates
  - [ ] Verify totals recalculate correctly
- [ ] Change number format from European to US
  - [ ] Verify separators update (1.234,56 → 1,234.56)
  - [ ] Verify currency symbols update (€1.234,56 → $1,234.56)
- [ ] Test with missing exchange rate
  - [ ] Should fallback gracefully (show native or warning)

### Data Integrity Testing

- [ ] Verify transactions stored in native currency
- [ ] Verify dividends stored in native currency
- [ ] Verify fund prices stored in native currency
- [ ] Import IBKR transaction in USD
  - [ ] Verify stored in USD
  - [ ] Verify displays correctly in base currency on Overview
- [ ] Run exchange rate cleanup job
  - [ ] Verify unused currencies removed
  - [ ] Verify active currencies retained
- [ ] Run symbol info cleanup job
  - [ ] Verify unused symbols removed
  - [ ] Verify fund symbols retained

### Performance Testing

- [ ] Test with 50+ funds in different currencies
- [ ] Verify Overview loads quickly with conversions
- [ ] Verify exchange rate caching works (check network requests)
- [ ] Test repeated currency toggles (should be instant)

### Edge Cases

- [ ] Fund with no exchange rate available
- [ ] Same currency conversion (EUR → EUR)
- [ ] Exchange rate cache expiration
- [ ] Multiple portfolios with different currency mixes
- [ ] Negative values (losses) with conversion
- [ ] Zero values with conversion

---

## Migration Order & Timeline

### Week 1: Foundation & Backend

**Days 1-2: Database & Models**
- [ ] Create user_preferences table (or extend config)
- [ ] Add base_currency field with default 'EUR'
- [ ] Add database indexes for exchange_rates
- [ ] Migration script

**Days 3-4: Backend APIs**
- [ ] Add /config/preferences endpoints (GET, PUT)
- [ ] Add /exchange-rates/convert endpoint
- [ ] Add /exchange-rates/latest endpoint
- [ ] Enhance ExchangeRateService with caching
- [ ] Add currency field to portfolio_funds response
- [ ] Add fund_currency to transactions response
- [ ] Add fund_currency to dividends response

**Day 5: Testing & Verification**
- [ ] Test all new endpoints
- [ ] Verify currency fields in responses
- [ ] Test exchange rate caching

### Week 2: Frontend Core

**Days 1-2: Context & Utilities**
- [ ] Add baseCurrency to FormatContext
- [ ] Add exchange rate fetching to FormatContext
- [ ] Create currencyConversion.js utility
- [ ] Add convertAndFormat to FormatContext
- [ ] Test context updates

**Days 3-4: Config UI**
- [ ] Add base currency selector to Config.js
- [ ] Wire up to backend
- [ ] Add currency indicator components
- [ ] Test preference changes

**Day 5: Component Updates Start**
- [ ] Update Overview.js with conversions
- [ ] Update PortfolioSummary.js
- [ ] Test aggregates display correctly

### Week 3: Component Migrations & Cleanup

**Days 1-2: Remaining Components**
- [ ] Update FundDetail.js (verify native display)
- [ ] Update FundsTable.js with toggle
- [ ] Update TransactionsTable.js with toggle
- [ ] Update DividendsTable.js with toggle
- [ ] Update ValueChart.js with currency prop

**Days 3-4: Maintenance Tasks**
- [ ] Create MaintenanceService
- [ ] Implement cleanup_exchange_rates()
- [ ] Implement cleanup_symbol_info()
- [ ] Setup scheduler/cron
- [ ] Update symbol lookup to prevent garbage
- [ ] Test maintenance jobs

**Day 5: Testing & Documentation**
- [ ] Full regression testing
- [ ] User acceptance testing
- [ ] Update user documentation
- [ ] Create release notes

---

## Rollback Plan

### If Issues Arise

**Phase 1-2 (Backend)**:
- Can rollback database migrations
- Disable new endpoints
- Frontend continues to work with old formatCurrency

**Phase 3-4 (Frontend Display)**:
- Revert FormatContext changes
- Use old formatCurrency
- No data loss (stored in native currency)

**Phase 5-6 (Cleanup)**:
- Disable scheduled jobs
- Restore exchange rates from backup if needed

---

## Future Enhancements

### Phase 9: Advanced Features (Future)

1. **Historical Currency Conversion**
   - Convert past transactions using historical rates
   - Show portfolio value in base currency over time

2. **Portfolio Base Currency Override**
   - Allow setting base currency per portfolio
   - Different from global base currency

3. **Currency Conversion Transaction Log**
   - Track when conversions occur
   - Audit trail for converted values

4. **Real-time Exchange Rates**
   - Integrate with live exchange rate API
   - Update rates hourly instead of daily

5. **Currency Hedging Tracking**
   - Track currency hedging positions
   - Show hedged vs unhedged portfolio value

6. **Custom Exchange Rate Override**
   - Allow manual exchange rate entry
   - Useful for personal rates or broker-specific rates

---

## Dependencies

### Backend
- `apscheduler` - For scheduling maintenance jobs
- Existing: `yfinance`, `sqlalchemy`, `flask`

### Frontend
- Existing: `react`, `react-router-dom`
- No new dependencies needed

---

## Documentation Updates Needed

1. **User Guide**:
   - How to set base currency
   - Understanding native vs converted display
   - Currency toggle on Portfolio Detail

2. **Developer Guide**:
   - FormatContext API
   - Currency conversion utilities
   - Adding new currencies

3. **API Documentation**:
   - New endpoints
   - Currency fields in responses

---

## Notes & Considerations

### Mixed Currency Portfolios
Current approach shows aggregates in base currency without conversion tracking. This is display-only. Future enhancement could add proper multi-currency accounting.

### Exchange Rate Sources
Currently using exchange rate table. Future could integrate live APIs (ECB, Open Exchange Rates, etc.)

### Performance
- Exchange rate caching is critical
- Frontend conversion is faster than backend
- Consider memoization for repeated conversions

### Data Integrity
- Always store in native currency
- Never convert on write, only on read/display
- Maintain audit trail of original currency amounts

---

## Success Metrics

### User Experience
- Users can set preferred base currency
- All aggregates display in consistent currency
- Easy toggle between native and converted views

### Performance
- No noticeable slowdown with conversions
- Overview loads in < 2 seconds with 50+ funds
- Currency toggle is instant

### Data Quality
- Exchange rate table stays manageable size
- Symbol info table doesn't accumulate garbage
- All conversions use appropriate exchange rates

---

## Questions & Decisions Log

**Q**: Should we support custom exchange rates?
**A**: Future enhancement. Start with database rates only.

**Q**: What if exchange rate is missing?
**A**: Show native currency with warning indicator. Don't block display.

**Q**: Should portfolios have individual base currencies?
**A**: Not in v1. Global base currency is simpler. Can add per-portfolio later.

**Q**: How often to refresh exchange rates?
**A**: Daily for database. Hourly for frontend cache.

**Q**: Should we convert historical chart data?
**A**: Not in v1. Charts show native currency. Future enhancement.

---

## Related Issues / PRs

- #XX - Add base currency preference
- #XX - Implement currency conversion system
- #XX - Add maintenance cleanup jobs
- #XX - Migrate to formatCurrencyWithCode

---

**Last Updated**: 2025-01-04
**Document Owner**: Development Team
**Status**: Ready for Implementation
