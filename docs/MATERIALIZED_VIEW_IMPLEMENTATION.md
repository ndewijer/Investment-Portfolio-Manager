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

The fund history endpoint (`/api/fund/history/{portfolioId}`) automatically detects and fixes stale or missing data:

**Triggers auto-materialization when:**
1. **No data exists** - Materialized view wasn't populated after an upgrade
2. **Data is stale** - Latest transaction is newer than latest materialized date

**Stale Data Detection** (v1.5.2+):
- Compares latest materialized date with latest transaction date
- If transaction date > materialized date: auto-materializes from that point forward
- Handles cases where invalidation deleted 0 records (e.g., transaction dated after latest materialized date)
- Logs detection reason ("no_data" or "stale_data") and records created

**Example scenario:**
```
1. Materialized view calculated through 2026-02-04
2. User allocates IBKR transaction dated 2026-02-05
3. Invalidation runs: deletes 0 records (none exist for 2026-02-05)
4. User views graphs: stale data detected (transaction > materialized)
5. Auto-materializes for 2026-02-05
6. Graphs immediately show updated data
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
