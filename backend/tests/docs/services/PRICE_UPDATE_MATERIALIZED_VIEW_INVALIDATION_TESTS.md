# Price Update Materialized View Invalidation Tests

## Overview

Tests verifying that price update operations properly invalidate the materialized fund history view. This ensures cached graph data stays accurate when fund prices change.

## Test File

`tests/services/test_price_update_materialized_view_invalidation.py`

## Test Cases

| Test | What it verifies |
|------|-----------------|
| `test_update_todays_price_invalidates_materialized_view` | Today's price update invalidates (mocked yfinance) |
| `test_update_historical_prices_invalidates_materialized_view` | Historical update invalidates from earliest date |
| `test_price_update_invalidates_multiple_portfolios` | Two portfolios with same fund both get invalidated |
| `test_no_invalidation_when_price_already_exists` | Duplicate price skips invalidation |
| `test_invalidation_failure_does_not_break_price_update` | Mocked failure doesn't break the update |

## Pattern

All tests mock yfinance to avoid real API calls. The pattern:
1. Create entities (portfolio, fund, portfolio_fund, transaction)
2. Populate `FundHistoryMaterialized` records
3. Perform price update with mocked yfinance
4. Assert materialized records were deleted (or preserved for duplicates)

## Related Documentation

- **Service Code**: `app/services/price_update_service.py`
- **Invalidation Helpers**: `app/services/portfolio_history_materialized_service.py`
- **Price Update Service Tests**: `PRICE_UPDATE_SERVICE_TESTS.md`
- **Materialized View Docs**: `docs/MATERIALIZED_VIEW_IMPLEMENTATION.md`

---

**Document Version**: 1.5.1
**Last Updated**: 2026-02-06
