# Transaction Materialized View Invalidation Tests

## Overview

Tests verifying that transaction CRUD operations (including sell transactions with realized gains) properly invalidate the materialized fund history view.

## Test File

`tests/services/test_transaction_materialized_view_invalidation.py`

## Test Cases

| Test | What it verifies |
|------|-----------------|
| `test_create_transaction_invalidates_materialized_view` | Buy transaction creation triggers invalidation |
| `test_update_transaction_invalidates_materialized_view` | Transaction update triggers invalidation |
| `test_delete_transaction_invalidates_materialized_view` | Transaction deletion triggers invalidation |
| `test_process_sell_transaction_invalidates_materialized_view` | Sell with realized gain triggers invalidation |
| `test_invalidation_only_affects_target_portfolio` | Unrelated portfolios not invalidated |
| `test_invalidation_failure_does_not_break_transaction` | Mocked failure doesn't break the operation |

## Pattern

All tests follow the established integration test pattern:
1. Create entities (portfolio, fund, portfolio_fund)
2. Populate `FundHistoryMaterialized` records
3. Perform transaction operation
4. Assert materialized records were deleted (or preserved for unrelated portfolios)

## Related Documentation

- **Service Code**: `app/services/transaction_service.py`
- **Invalidation Helpers**: `app/services/portfolio_history_materialized_service.py`
- **Transaction Service Tests**: `TRANSACTION_SERVICE_TESTS.md`
- **IBKR Invalidation Tests**: `test_ibkr_materialized_view_invalidation.py`
- **Materialized View Docs**: `docs/MATERIALIZED_VIEW_IMPLEMENTATION.md`

---

**Document Version**: 1.5.1
**Last Updated**: 2026-02-06
