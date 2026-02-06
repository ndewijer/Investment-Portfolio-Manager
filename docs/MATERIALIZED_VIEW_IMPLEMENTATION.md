# Portfolio History Materialized View Implementation

## Overview

This implementation uses a fund-level materialized view pattern for portfolio history calculations to dramatically improve query performance. Instead of calculating portfolio values on-demand for every request, historical data is pre-calculated and stored at the fund level in a SQLite table.

## Performance Improvements

**Before (On-Demand Calculation):**
- 5 years of daily history: ~8 seconds
- Every request recalculates all values

**After (Materialized View):**
- 5 years of daily history: ~50ms (160x faster)
- Pre-calculated data served from cache
- Both portfolio and fund-level queries benefit from same data source

## Architecture

### Fund-Level Materialized View

```
fund_history_materialized
  ├─ portfolio_fund_id, fund_id, date, shares, price, value, cost, ...
  └─ One row per fund per date (atomic data)

/api/portfolio/history
  → SELECT SUM(...) FROM fund_history_materialized GROUP BY date, portfolio_id

/api/fund/history/{portfolioId}
  → Direct query from fund_history_materialized (fast)
```

**Benefits:**
- ✅ No data duplication (fund data is source of truth)
- ✅ Both endpoints served from same table
- ✅ Easier debugging (can trace specific fund on specific date)
- ✅ Self-healing (fix one fund, portfolio auto-updates)

### Database Schema

The `fund_history_materialized` table stores pre-calculated fund-level history:

```sql
CREATE TABLE fund_history_materialized (
    id TEXT PRIMARY KEY,
    portfolio_fund_id TEXT NOT NULL,
    fund_id TEXT NOT NULL,
    date TEXT NOT NULL,  -- YYYY-MM-DD format

    -- Fund metrics
    shares REAL NOT NULL,
    price REAL NOT NULL,
    value REAL NOT NULL,
    cost REAL NOT NULL,

    -- Gain/loss metrics
    realized_gain REAL NOT NULL,
    unrealized_gain REAL NOT NULL,
    total_gain_loss REAL NOT NULL,

    -- Income/expense metrics
    dividends REAL NOT NULL,
    fees REAL NOT NULL,

    -- Metadata
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE,
    UNIQUE(portfolio_fund_id, date)
);

-- Indexes for optimal query performance
CREATE INDEX idx_fund_history_pf_date ON fund_history_materialized(portfolio_fund_id, date);
CREATE INDEX idx_fund_history_date ON fund_history_materialized(date);
CREATE INDEX idx_fund_history_fund_id ON fund_history_materialized(fund_id);
```

**Field Naming Convention**: The database and internal Python code use snake_case (e.g., `realized_gain`, `unrealized_gain`), while the API responses use camelCase (e.g., `totalRealizedGainLoss`, `totalUnrealizedGainLoss`). The conversion happens automatically at the API boundary.

### Smart Query Router

The `PortfolioService.get_portfolio_history()` method now intelligently routes between:

1. **Materialized View (Fast Path)**: If complete data exists for the requested date range
2. **On-Demand Calculation (Slow Path)**: Falls back when materialized data is incomplete or missing

### Automatic Invalidation

When source data changes, the materialized view is automatically invalidated:

- **Transaction Create/Update/Delete**: Invalidates from transaction date forward
- **Sell Transactions**: Invalidates from transaction date forward (includes realized gain changes)
- **Dividend Create/Update/Delete**: Invalidates from ex-dividend date forward (date changes invalidate from both old and new dates)
- **Price Updates (Today)**: Invalidates from price date forward for all portfolios holding that fund
- **Price Updates (Historical)**: Invalidates from earliest updated date forward for all portfolios holding that fund
- **IBKR Process Allocation**: Invalidates from transaction date forward for each affected portfolio
- **IBKR Modify Allocations**: Invalidates for all affected portfolios (old and new)
- **IBKR Unallocate Transaction**: Invalidates from transaction date forward for each affected portfolio
- **IBKR Match Dividend**: Invalidates from ex-dividend date forward for each matched dividend

All invalidation calls are wrapped in try/except blocks with lazy imports to ensure that invalidation failures never break primary operations.

### Auto-Materialization

The fund history endpoint (`/api/fund/history/{portfolioId}`) automatically detects and fixes stale or missing data by checking **ALL** data sources.

#### Triggers Auto-Materialization When:

1. **No data exists** - Materialized view wasn't populated after an upgrade
2. **Data is stale** - Any of these conditions are true:
   - Latest **transaction** is newer than latest materialized date
   - Latest **price** is newer than latest materialized date (v1.5.3+)
   - Latest **dividend** is newer than latest materialized date (v1.5.3+)

#### Stale Data Detection (v1.5.3+)

The system checks **three** data sources to ensure completeness:

```python
latest_materialized_date = max(materialized_records.date)
latest_transaction_date  = max(transactions.date)
latest_price_date        = max(fund_prices.date)
latest_dividend_date     = max(dividends.ex_dividend_date)

# Stale if ANY source is newer
if (latest_transaction > latest_materialized OR
    latest_price > latest_materialized OR
    latest_dividend > latest_materialized):
    auto_materialize()
```

**Why all three sources matter:**

| Scenario | What Happens Without Check | Impact |
|----------|---------------------------|--------|
| Price updated, no new transactions | Invalidation deletes today's record, but no transaction to trigger re-materialization | Graphs miss today's price changes |
| Dividend recorded, no transactions | Materialized view doesn't include dividend | Graphs show wrong dividend totals |
| Transaction backdated | Invalidation deletes 0 records (none exist for that date) | Graphs miss backdated transactions |

#### Example Scenarios

**Scenario 1: Price Update (The Nightly Edge Case)**
```
Morning (Feb 6):
  - View graphs → auto-materializes through Feb 6 (using Feb 5 prices)
  - Materialized: Feb 6 row created

Night (Feb 6):
  - Price update runs → fetches Feb 6 closing price
  - Invalidation runs → deletes Feb 6 materialized record
  - Result: Materialized only through Feb 5

Next Morning (Feb 7):
  - View graphs
  - Stale check: latest_price (Feb 6) > latest_mat (Feb 5)
  - Auto-materializes Feb 6 with correct closing price ✅
  - Graphs show accurate data
```

**Scenario 2: IBKR Allocation (Backdated Transaction)**
```
1. Materialized view calculated through Feb 4
2. User allocates IBKR transaction dated Feb 5
3. Invalidation runs: deletes 0 records (none exist for Feb 5)
4. View graphs: stale check finds latest_txn (Feb 5) > latest_mat (Feb 4)
5. Auto-materializes for Feb 5 ✅
6. Graphs immediately show allocated transaction
```

**Scenario 3: Dividend Recorded**
```
1. Materialized through Feb 5
2. User records dividend with ex_dividend_date = Feb 6
3. Invalidation runs: deletes 0 records (none exist for Feb 6)
4. View graphs: stale check finds latest_dividend (Feb 6) > latest_mat (Feb 5)
5. Auto-materializes for Feb 6 with dividend included ✅
6. Graphs show correct dividend totals
```

#### Logging

Stale detection logs show exactly which data source triggered re-materialization:

```json
{
  "message": "Materialized view is stale for portfolio <id>",
  "details": {
    "latest_materialized": "2026-02-05",
    "stale_sources": ["prices"],
    "latest_price": "2026-02-06",
    "price_days_behind": 1
  }
}
```

The auto-materialization is wrapped in try/except to avoid breaking the API if materialization fails.

### Logging (v1.5.2+)

All materialized view operations now include comprehensive logging for debugging and monitoring:

**Invalidation Logging:**
```json
{
  "level": "INFO",  // INFO if records deleted, DEBUG if 0 records
  "category": "SYSTEM",
  "message": "Materialized view invalidation for portfolio <id>",
  "details": {
    "portfolio_id": "...",
    "from_date": "2026-02-05",
    "records_deleted": 0,
    "portfolio_funds_checked": 3,
    "recalculate": false
  }
}
```

**IBKR Allocation Invalidation:**
```json
{
  "level": "INFO",
  "category": "IBKR",
  "message": "Materialized view invalidation after IBKR allocation",
  "details": {
    "affected_portfolios": 3,
    "total_records_deleted": 0,
    "per_portfolio": {"portfolio-id": 0, ...}
  }
}
```

**Stale Data Detection:**
```json
{
  "level": "INFO",
  "category": "SYSTEM",
  "message": "Materialized view is stale for portfolio <id>",
  "details": {
    "latest_transaction": "2026-02-05",
    "latest_materialized": "2026-02-04",
    "days_behind": 1
  }
}
```

**Auto-Materialization:**
```json
{
  "level": "INFO",
  "category": "SYSTEM",
  "message": "Auto-materialized portfolio history (stale_data)",
  "details": {
    "reason": "stale_data",  // or "no_data"
    "records_created": 12
  }
}
```

## Edge Cases & Gotchas (IMPORTANT!)

This section documents all the edge cases discovered and how they're handled.

### 1. Invalidation Deletes 0 Records

**Problem:** When transactions are backdated or created for future dates, invalidation may delete 0 records (because no materialized data exists for those dates yet).

**Example:**
```
- Materialized through: Feb 4
- Allocate transaction dated: Feb 5
- Invalidation tries to delete records >= Feb 5
- Finds: 0 records to delete
- Result: View still only shows data through Feb 4
```

**Solution:** Stale detection checks if transactions exist beyond latest materialized date and triggers auto-materialization.

**Status:** ✅ Fixed in v1.5.2

---

### 2. Price Updates Without Transactions (The Nightly Problem)

**Problem:** Daily price updates invalidate the current day's materialized record, but if no new transactions exist, stale detection doesn't trigger recalculation.

**Example:**
```
Morning: View graphs → materializes through today (with yesterday's prices)
Night:   Price update → invalidates today's record
Morning: View graphs → stale check compares transactions, not prices
         latest_transaction == latest_materialized → NO stale detected!
Result:  Missing today's price update in graphs
```

**Solution:** Stale detection now checks prices, not just transactions (v1.5.3+).

**Status:** ✅ Fixed in v1.5.3

---

### 3. Dividend Recording Without Transactions

**Problem:** Recording a dividend doesn't create a transaction, so transaction-only stale detection wouldn't trigger recalculation.

**Example:**
```
- Materialized through: Feb 5
- Record dividend with ex_dividend_date: Feb 6
- Invalidation deletes 0 records (none exist for Feb 6)
- Stale check only looks at transactions → no stale detected
- Result: Dividend not reflected in graphs
```

**Solution:** Stale detection checks dividend dates (v1.5.3+).

**Status:** ✅ Fixed in v1.5.3

---

### 4. Multi-Portfolio Price Updates

**Problem:** One fund can be held by multiple portfolios. Price update must invalidate ALL portfolios.

**Example:**
```
- Fund AAPL held by: Portfolio A, Portfolio B, Portfolio C
- Price update for AAPL
- Must invalidate: All three portfolios
```

**Solution:** `invalidate_from_price_update()` queries all portfolio_funds for the fund and invalidates each portfolio.

**Status:** ✅ Working correctly since v1.5.1

---

### 5. Dividend Date Changes (Both Old and New)

**Problem:** When changing a dividend's ex_dividend_date, BOTH the old date range and new date range need invalidation.

**Example:**
```
- Original ex_dividend_date: Feb 10
- Updated to: Feb 15
- Must invalidate: From Feb 10 forward AND from Feb 15 forward
```

**Solution:** `update_dividend()` captures original date, invalidates from old date, then invalidates from new date.

**Status:** ✅ Working correctly since v1.5.1

---

### 6. Historical Price Backfills

**Problem:** When backfilling historical prices, must invalidate from the EARLIEST updated date, not just the latest.

**Example:**
```
- Materialized through: Feb 10
- Backfill prices for: Feb 5, Feb 6, Feb 7
- Must invalidate from: Feb 5 (earliest), not Feb 7
```

**Solution:** `update_historical_prices()` tracks earliest date in the update range and invalidates from there.

**Status:** ✅ Working correctly since v1.5.1

---

### 7. Sell Transactions with Realized Gains

**Problem:** Sell transactions create realized gain/loss records which affect history calculations.

**Example:**
```
- Sell 10 shares with $50 realized gain
- Must recalculate history from sell date forward
- Realized gain must appear in all subsequent dates
```

**Solution:** `process_sell_transaction()` triggers invalidation after creating realized gain record.

**Status:** ✅ Working correctly since v1.5.1

---

### Summary: Complete Coverage Matrix

| Data Change | Invalidation | Stale Detection | Version |
|-------------|--------------|-----------------|---------|
| Transaction CRUD | ✅ Yes | ✅ Transactions | v1.5.1 |
| Price Update (Today) | ✅ Yes | ✅ Prices | v1.5.3 |
| Price Update (Historical) | ✅ Yes | ✅ Prices | v1.5.3 |
| Dividend CRUD | ✅ Yes | ✅ Dividends | v1.5.3 |
| IBKR Allocation | ✅ Yes | ✅ Transactions | v1.5.2 |
| IBKR Modify | ✅ Yes | ✅ Transactions | v1.5.1 |
| IBKR Unallocate | ✅ Yes | ✅ Transactions | v1.5.1 |
| IBKR Match Dividend | ✅ Yes | ✅ Dividends | v1.5.3 |

**The key insight:** Every data change triggers invalidation (delete stale records), and every data source is checked during stale detection (trigger recalculation).

## Usage

### Initial Setup

After deploying this change, run the database migration:

```bash
cd backend
flask db upgrade
```

### Populating the Materialized View

**Materialize all portfolios:**
```bash
flask materialize-history
```

**Materialize specific portfolio:**
```bash
flask materialize-history --portfolio-id=<portfolio-id>
```

**Force recalculation:**
```bash
flask materialize-history --force
```

### Monitoring

**View statistics:**
```bash
flask materialized-stats
```

Output:
```
📊 Materialized View Statistics:
  Total records: 15,000
  Portfolios with data: 10
  Date range: 2020-01-01 to 2024-12-31
```

### Manual Invalidation

If you need to manually invalidate data (e.g., after a data correction):

```bash
flask invalidate-materialized-history --portfolio-id=<id> --from-date=2024-01-01
```

With automatic recalculation:
```bash
flask invalidate-materialized-history --portfolio-id=<id> --from-date=2024-01-01 --recalculate
```

## API Endpoints

### Portfolio History Endpoint

The `/api/portfolio/history` endpoint aggregates data from the fund-level materialized view.

**Response Format**: Returns portfolio history with camelCase field names:
- `totalValue` - Aggregated sum of all fund values
- `totalCost` - Aggregated sum of all fund costs
- `totalRealizedGainLoss` - Aggregated sum of realized gains
- `totalUnrealizedGainLoss` - Aggregated sum of unrealized gains
- `totalDividends` - Aggregated sum of dividends
- `totalSaleProceeds` - Calculated from realized_gain_loss table
- `totalOriginalCost` - Calculated from realized_gain_loss table
- `totalGainLoss` - Aggregated sum of total gain/loss
- `isArchived` - Portfolio archive status

### Fund History Endpoint

The `/api/fund/history/{portfolioId}` endpoint provides fund-level historical data.

**Response Format**:
```json
[
  {
    "date": "2021-09-06",
    "funds": [
      {
        "portfolio_fund_id": "...",
        "fund_id": "...",
        "fund_name": "Goldman Sachs Enhanced Index Sustainable Equity",
        "shares": 15.787812,
        "price": 31.67,
        "value": 500.00,
        "cost": 500.00,
        "realized_gain": 0,
        "unrealized_gain": 0,
        "total_gain_loss": 0,
        "dividends": 0,
        "fees": 0
      }
    ]
  }
]
```

## How It Works

### 1. Data Flow

```
User Request → Portfolio/Fund Service
                    ↓
        Fund History Materialized Table
                    ↓
        ┌──────────┴──────────┐
        ↓                     ↓
   Fund Query           Portfolio Query
        ↓                     ↓
   Direct Read         Aggregate (SUM)
        ↓                     ↓
   Fund History        Portfolio History
```

### 2. Invalidation Flow

```
Transaction Created → Commit to DB
                          ↓
                  Invalidate Fund History
                          ↓
                  Delete fund records >= transaction.date
                          ↓
                  (Recalculation happens on next query
                   or via background job)
```

### 3. Fund-Level Calculation

The system calculates fund metrics for each date and stores them:

```python
# For each portfolio_fund and each date:
FundHistoryMaterialized(
    portfolio_fund_id=pf.id,
    fund_id=pf.fund_id,
    date=date_str,
    shares=calculated_shares,
    price=latest_price,
    value=shares * price,
    cost=calculated_cost,
    realized_gain=calculated_realized,
    unrealized_gain=calculated_unrealized,
    total_gain_loss=realized + unrealized,
    dividends=calculated_dividends,
    fees=calculated_fees
)
```

Portfolio-level data is then aggregated on-the-fly:

```python
# Portfolio history aggregates from fund data
SELECT
    date,
    portfolio_id,
    SUM(value) as total_value,
    SUM(cost) as total_cost,
    SUM(realized_gain) as total_realized_gain,
    SUM(unrealized_gain) as total_unrealized_gain,
    SUM(total_gain_loss) as total_gain_loss,
    SUM(dividends) as total_dividends,
    SUM(fees) as total_fees
FROM fund_history_materialized
JOIN portfolio_fund ON fund_history_materialized.portfolio_fund_id = portfolio_fund.id
GROUP BY date, portfolio_id
```

## Maintenance

### Recommended Schedule

**Daily:**
- Materialized data is invalidated automatically on writes
- Next query will use on-demand calculation, then you can re-materialize

**Weekly (Recommended):**
```bash
flask materialize-history --force
```

This ensures any gaps are filled and all data is fresh.

### Storage Requirements

Typical storage per portfolio:
- ~1,500 days (4 years) = 150KB per portfolio
- 10 portfolios = 1.5MB total

Very reasonable for SQLite.

## Migration Guide

### Stage 1: Deploy Schema
```bash
git pull
cd backend
flask db upgrade
```

### Stage 2: Backfill Data
```bash
flask materialize-history
```

This may take a few minutes for large portfolios.

### Stage 3: Monitor Performance

Check query times in application logs. You should see dramatic improvements for history queries.

### Stage 4: Set Up Maintenance

Add weekly cron job:
```bash
0 2 * * 0 cd /path/to/app && flask materialize-history --force
```

## Rollback Plan

If issues arise:

1. **Disable feature**: Set `use_materialized=False` in the service call
2. **Investigate**: Compare materialized vs on-demand results
3. **Clear cache**: `flask invalidate-materialized-history --portfolio-id=<id> --from-date=2020-01-01`
4. **Drop table**: Run downgrade migration if needed

## Technical Details

### Key Files

**Database Migration:**
- `backend/migrations/versions/vX.Y.Z_fund_level_materialized_view.py` - Fund-level materialized view migration

**API Layer:**
- `backend/app/api/fund_namespace.py` - Fund API namespace and endpoints
- `backend/app/api/portfolio_namespace.py` - Portfolio API namespace

**Service Layer:**
- `backend/app/services/fund_service.py` - Fund-related business logic
- `backend/app/services/portfolio_service.py` - Portfolio history aggregation logic
- `backend/app/services/portfolio_history_materialized_service.py` - Materialized view management
- `backend/app/services/transaction_service.py` - Invalidation hooks

**Models:**
- `backend/app/models.py` - Contains `FundHistoryMaterialized` model

**Application Setup:**
- `backend/app/__init__.py` - Namespace registration
- `backend/app/cli_commands.py` - CLI commands for materialization management

### Key Classes

**`FundHistoryMaterialized`**: SQLAlchemy model for fund-level cached data

**`FundService`**: Service for fund-related operations including fund history retrieval

**`PortfolioHistoryMaterializedService`**: Service for managing materialized views and invalidation

**`PortfolioService`**: Aggregates portfolio history from fund data

### Architecture Benefits

**Eliminated Data Duplication**: Fund-level data is the single source of truth, with portfolio aggregates calculated on-the-fly.

**Flexible Querying**: The same underlying data serves both portfolio aggregate queries and detailed fund-level queries.

**Improved Maintainability**: When fund data is corrected or recalculated, portfolio aggregates automatically reflect the changes.

**Better Debugging**: Issues can be traced to specific funds on specific dates, rather than only seeing aggregated portfolio values.

### Error Handling

All invalidation operations are wrapped in try/except blocks to ensure that materialization failures don't break core functionality. Transactions will always succeed even if invalidation fails.

## Troubleshooting

### Issue: Materialized data doesn't match expected values

**Solution:**
```bash
flask invalidate-materialized-history --portfolio-id=<id> --from-date=2020-01-01 --recalculate
```

### Issue: Fund history endpoint returns no data

**Check if data is materialized:**
```bash
flask materialized-stats
```

**Materialize missing data:**
```bash
flask materialize-history --force
```

### Issue: Portfolio history returns incorrect aggregates

**Check fund-level data:**
```sql
-- Verify fund data exists
SELECT date, COUNT(*) as fund_count, SUM(value) as total_value
FROM fund_history_materialized
GROUP BY date
ORDER BY date DESC
LIMIT 10;
```

**Recalculate from scratch:**
```bash
flask invalidate-materialized-history --portfolio-id=<id> --from-date=2020-01-01 --recalculate
```

### Issue: Database size growing too large

Materialized views add minimal overhead (~1MB per 10 portfolios). If concerned:
```sql
-- Check table size
SELECT page_count * page_size / 1024.0 / 1024.0 AS size_mb
FROM pragma_page_count('portfolio_history_materialized')
CROSS JOIN pragma_page_size;
```

## Future Enhancements

Potential improvements for future versions:

1. **Background Jobs**: Automatic recalculation via Celery/RQ
2. **Incremental Updates**: Only recalculate affected date ranges
3. **Real-time WebSocket**: Push updates when recalculation completes
4. **Compression**: Store daily deltas instead of full snapshots
5. **Partitioning**: Separate tables per year for very large datasets

## Questions?

Check the source code documentation:
- `FundHistoryMaterialized` model in `backend/app/models.py`
- `FundService` class docstrings in `backend/app/services/fund_service.py`
- `PortfolioHistoryMaterializedService` class docstrings
- Migration file comments
- CLI command help text: `flask materialize-history --help`

---

**Document Version**: 1.5.1
**Last Updated**: 2026-02-06
**Maintained By**: @ndewijer
