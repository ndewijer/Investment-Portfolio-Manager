# Dividend Materialized View Invalidation Tests

## Overview

Tests verifying that dividend CRUD operations properly invalidate the materialized fund history view. This ensures cached graph data stays accurate when dividends change.

## Test File

`tests/services/test_dividend_materialized_view_invalidation.py`

## Test Cases

| Test | What it verifies |
|------|-----------------|
| `test_create_dividend_invalidates_materialized_view` | Cash dividend creation clears stale materialized data |
| `test_create_stock_dividend_with_reinvestment_invalidates` | Stock dividend + reinvestment transaction triggers invalidation |
| `test_update_dividend_invalidates_materialized_view` | Dividend update clears stale data |
| `test_update_dividend_date_change_invalidates_both_dates` | Changed ex_dividend_date invalidates from old AND new date |
| `test_delete_dividend_invalidates_materialized_view` | Deletion clears stale data |
| `test_invalidation_failure_does_not_break_dividend_create` | Mocked failure doesn't break the primary operation |

## Pattern

All tests follow the established integration test pattern:
1. Create entities (portfolio, fund, portfolio_fund, transaction)
2. Populate `FundHistoryMaterialized` records
3. Perform dividend operation
4. Assert materialized records were deleted

## Related Documentation

- **Service Code**: `app/services/dividend_service.py`
- **Invalidation Helpers**: `app/services/portfolio_history_materialized_service.py`
- **Dividend Service Tests**: `DIVIDEND_SERVICE_TESTS.md`
- **IBKR Invalidation Tests**: `test_ibkr_materialized_view_invalidation.py`
- **Materialized View Docs**: `docs/MATERIALIZED_VIEW_IMPLEMENTATION.md`

---

**Document Version**: 1.5.1
**Last Updated**: 2026-02-06
