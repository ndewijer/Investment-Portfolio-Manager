# Portfolio History Materialized View Implementation

## Overview

This implementation introduces a materialized view pattern for portfolio history calculations to dramatically improve query performance. Instead of calculating portfolio values on-demand for every request, historical data is pre-calculated and stored in a SQLite table.

## Performance Improvements

**Before (On-Demand Calculation):**
- 5 years of daily history: ~8 seconds
- Every request recalculates all values

**After (Materialized View):**
- 5 years of daily history: ~50ms (160x faster)
- Pre-calculated data served from cache

## Architecture

### Database Schema

A new table `portfolio_history_materialized` stores pre-calculated portfolio history:

```sql
CREATE TABLE portfolio_history_materialized (
    id TEXT PRIMARY KEY,
    portfolio_id TEXT NOT NULL,
    date TEXT NOT NULL,  -- YYYY-MM-DD format
    value REAL NOT NULL,
    cost REAL NOT NULL,
    realized_gain REAL NOT NULL,
    unrealized_gain REAL NOT NULL,
    total_dividends REAL NOT NULL,
    total_sale_proceeds REAL NOT NULL,
    total_original_cost REAL NOT NULL,
    total_gain_loss REAL NOT NULL,
    is_archived INTEGER NOT NULL,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
    UNIQUE(portfolio_id, date)
);
```

**Field Naming Convention**: The database and internal Python code use snake_case (e.g., `realized_gain`, `unrealized_gain`), while the API responses use camelCase (e.g., `totalRealizedGainLoss`, `totalUnrealizedGainLoss`). The conversion happens automatically at the API boundary.

### Smart Query Router

The `PortfolioService.get_portfolio_history()` method now intelligently routes between:

1. **Materialized View (Fast Path)**: If complete data exists for the requested date range
2. **On-Demand Calculation (Slow Path)**: Falls back when materialized data is incomplete or missing

### Automatic Invalidation

When source data changes, the materialized view is automatically invalidated:

- **Transaction Create/Update/Delete**: Invalidates from transaction date forward
- **Dividend Create/Update/Delete**: Invalidates from ex-dividend date forward
- **Price Updates**: Invalidates from price date forward for all portfolios holding that fund

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

## API Changes

### Portfolio History Endpoint

The `/api/portfolio/history` endpoint now automatically uses the materialized view when available.

**Response Format (v1.4.1+)**: Returns portfolio history with camelCase field names:
- `totalValue` (previously `value`)
- `totalCost` (previously `cost`)
- `totalRealizedGainLoss` (previously `realized_gain`)
- `totalUnrealizedGainLoss` (previously `unrealized_gain`)
- `totalDividends` (previously `total_dividends`)
- `totalSaleProceeds` (previously `total_sale_proceeds`)
- `totalOriginalCost` (previously `total_original_cost`)
- `totalGainLoss` (previously `total_gain_loss`)
- `isArchived` (previously `is_archived`)

This standardizes the response format to match the summary endpoint and follows JavaScript naming conventions.

**Bypass materialized view (for testing):**

The service method accepts an optional `use_materialized=False` parameter to force on-demand calculation, useful for comparing results.

## How It Works

### 1. Data Flow

```
User Request → Portfolio Service
                    ↓
            Check Materialized Coverage
                    ↓
        ┌──────────┴──────────┐
        ↓                     ↓
    Complete              Incomplete
    Coverage              Coverage
        ↓                     ↓
    Return                Calculate
    Cached Data           On-Demand
```

### 2. Invalidation Flow

```
Transaction Created → Commit to DB
                          ↓
                  Invalidate Materialized View
                          ↓
                  Delete records >= transaction.date
                          ↓
                  (Recalculation happens on next query
                   or via background job)
```

### 3. Coverage Check

The system checks if the requested date range is fully materialized:

```python
coverage = check_materialized_coverage(portfolio_ids, start_date, end_date)

if coverage.is_complete:
    return get_materialized_history()  # Fast!
else:
    return get_portfolio_history_on_demand()  # Slow but accurate
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

### Files Changed

**New Files:**
- `backend/migrations/versions/1.4.0_add_portfolio_history_materialized.py` - Migration
- `backend/app/services/portfolio_history_materialized_service.py` - Service layer

**Modified Files:**
- `backend/app/models.py` - Added `PortfolioHistoryMaterialized` model
- `backend/app/services/portfolio_service.py` - Added smart routing logic
- `backend/app/services/transaction_service.py` - Added invalidation hooks
- `backend/app/cli_commands.py` - Added CLI commands

### Key Classes

**`PortfolioHistoryMaterialized`**: SQLAlchemy model for cached data
**`PortfolioHistoryMaterializedService`**: Service for managing materialized views
**`MaterializedCoverage`**: Data class representing coverage status

### Error Handling

All invalidation operations are wrapped in try/except blocks to ensure that materialization failures don't break core functionality. Transactions will always succeed even if invalidation fails.

## Troubleshooting

### Issue: Materialized data doesn't match on-demand calculation

**Solution:**
```bash
flask invalidate-materialized-history --portfolio-id=<id> --from-date=2020-01-01 --recalculate
```

### Issue: Queries still slow

**Check coverage:**
```bash
flask materialized-stats
```

**Materialize missing data:**
```bash
flask materialize-history --force
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
- `PortfolioHistoryMaterializedService` class docstrings
- Migration file comments
- CLI command help text: `flask materialize-history --help`

---

**Last Updated**: 2026-01-13 (Version 1.4.1)
**Maintained By**: @ndewijer
