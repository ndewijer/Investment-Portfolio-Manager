# Fund Price Unique Constraint Migration

## Overview
Added a unique constraint on `(fund_id, date)` in the `fund_price` table to prevent duplicate price records for the same fund on the same date.

## Changes Made

### 1. Database Model (`backend/app/models.py`)
```python
__table_args__ = (
    db.UniqueConstraint("fund_id", "date", name="unique_fund_price"),  # NEW!
    db.Index("ix_fund_price_date", "date"),
    db.Index("ix_fund_price_fund_id", "fund_id"),
    db.Index("ix_fund_price_fund_id_date", "fund_id", "date"),
)
```

### 2. Price Update Services
Both `TodayPriceService` and `HistoricalPriceService` now use **upsert** logic:
- Check if price exists for the date
- Update if exists
- Insert if doesn't exist

This prevents IntegrityError exceptions from the unique constraint.

## Database Cleanup Required

### Step 1: Find Duplicates
```sql
-- Find all duplicate records
SELECT
    fund_id,
    date,
    COUNT(*) as duplicate_count,
    GROUP_CONCAT(id) as record_ids,
    GROUP_CONCAT(price) as prices
FROM fund_price
GROUP BY fund_id, date
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC, date DESC;
```

### Step 2: Check for Price Conflicts
```sql
-- Check if duplicates have DIFFERENT prices (data conflict!)
SELECT
    fund_id,
    date,
    COUNT(*) as count,
    MIN(price) as min_price,
    MAX(price) as max_price,
    MAX(price) - MIN(price) as price_difference,
    GROUP_CONCAT(id || ':' || price) as id_price_pairs
FROM fund_price
GROUP BY fund_id, date
HAVING COUNT(*) > 1 AND MIN(price) != MAX(price)
ORDER BY price_difference DESC;
```

### Step 3: Clean Up Duplicates

**Option A: Keep Most Recent Insert (Recommended)**
```sql
-- Keep the record with the latest ID (most recent insert)
DELETE FROM fund_price
WHERE id NOT IN (
    SELECT MAX(id)
    FROM fund_price
    GROUP BY fund_id, date
);
```

**Option B: Keep Highest Price**
```sql
-- Keep the highest price (if you prefer max price)
DELETE FROM fund_price
WHERE id NOT IN (
    SELECT id FROM (
        SELECT id, fund_id, date, price,
               ROW_NUMBER() OVER (
                   PARTITION BY fund_id, date
                   ORDER BY price DESC, id DESC
               ) as rn
        FROM fund_price
    ) ranked
    WHERE rn = 1
);
```

### Step 4: Apply Migration
After cleaning duplicates, apply the constraint:

**SQLite:**
```sql
-- SQLite doesn't support adding constraints to existing tables
-- Need to recreate table (handled by migration script)
```

**Migration will:**
1. Create new table with constraint
2. Copy data from old table
3. Drop old table
4. Rename new table

### Step 5: Verify
```sql
-- Should return 0 rows after migration
SELECT fund_id, date, COUNT(*)
FROM fund_price
GROUP BY fund_id, date
HAVING COUNT(*) > 1;
```

## Benefits

1. **Data Integrity**: One price per fund per day enforced at DB level
2. **No Silent Duplicates**: Attempts to create duplicates fail loudly
3. **Consistency**: Matches pattern already used in `ExchangeRate` model
4. **Query Safety**: Code can assume unique results for (fund_id, date) queries
5. **Race Condition Protection**: Concurrent price updates won't create duplicates

## Rollback Plan

If issues arise:
1. Remove constraint from models.py
2. Revert price_update_service.py to previous version
3. Run migration to drop constraint

## Testing

✅ All 17 price update service tests passing
✅ Upsert logic tested for both create and update scenarios
✅ No duplicate creation possible with constraint in place

---

**Last Updated**: 2026-02-11 (Version 1.5.4)
**Maintained By**: @ndewijer
