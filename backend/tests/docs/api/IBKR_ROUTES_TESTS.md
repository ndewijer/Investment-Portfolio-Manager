# IBKR Routes Integration Tests

**File**: `tests/api/test_ibkr_routes.py`
**Route File**: `app/routes/ibkr_routes.py`
**Test Count**: 61 tests (18 integration + 43 error paths)
**Coverage**: 95% (215/215 statements, 11 missing lines)
**Status**: ✅ All tests passing

> **💡 Detailed Test Information**: For detailed explanations of each test including
> WHY it exists and what business logic it validates, see the docstrings in the test file.
> Your IDE will show these when hovering over test names.

---

## Overview

Integration tests for Interactive Brokers (IBKR) transaction processing API endpoints. These tests verify IBKR configuration management, transaction inbox operations, allocation processing with percentage-based splits, dividend matching to existing records, and bulk allocation operations.

### Endpoints Tested

- **POST /api/ibkr/flex-query** - Import IBKR Flex Query (SKIPPED - external API)
- **GET /api/ibkr/config** - Get IBKR configuration
- **POST /api/ibkr/config** - Save IBKR configuration
- **POST /api/ibkr/config/test** - Test IBKR connection
- **DELETE /api/ibkr/config** - Delete IBKR configuration
- **GET /api/ibkr/inbox** - List inbox transactions with filters
- **GET /api/ibkr/inbox/count** - Get inbox transaction counts
- **GET /api/ibkr/inbox/<id>** - Get transaction detail
- **GET /api/ibkr/inbox/<id>/eligible-portfolios** - Get eligible portfolios for allocation
- **POST /api/ibkr/inbox/<id>/ignore** - Mark transaction as ignored
- **DELETE /api/ibkr/inbox/<id>** - Delete transaction
- **POST /api/ibkr/inbox/<id>/allocate** - Allocate transaction to portfolios
- **GET /api/ibkr/inbox/<id>/allocations** - Get allocation details
- **PUT /api/ibkr/inbox/<id>/allocations** - Update existing allocations
- **POST /api/ibkr/inbox/<id>/unallocate** - Remove allocations
- **GET /api/ibkr/dividends/pending** - Get pending dividends for matching
- **POST /api/ibkr/inbox/<id>/match-dividend** - Match dividend to existing records
- **POST /api/ibkr/inbox/bulk-allocate** - Bulk allocate multiple transactions

---

## Test Organization

### TestIBKRConfig (4 tests)
- `test_get_config_not_found` - Verify GET /ibkr/config returns 404 when no config exists
- `test_save_config` - Save IBKR config with flex_token and flex_query_id
- `test_get_config` - Retrieve saved IBKR configuration
- `test_delete_config` - Delete IBKR configuration

### TestIBKRImport (1 test)
- `test_import_flex_query` - SKIPPED: Import from external IBKR Flex API (requires mocking)

### TestIBKRInbox (7 tests)
- `test_get_inbox_empty` - GET /ibkr/inbox returns empty list when no transactions
- `test_get_inbox_with_transactions` - GET /ibkr/inbox returns transaction list
- `test_get_inbox_count` - GET /ibkr/inbox/count returns counts by status
- `test_get_transaction` - GET /ibkr/inbox/<id> returns transaction detail
- `test_get_eligible_portfolios` - GET /ibkr/inbox/<id>/eligible-portfolios returns active portfolios
- `test_ignore_transaction` - POST /ibkr/inbox/<id>/ignore marks transaction as ignored
- `test_delete_transaction` - DELETE /ibkr/inbox/<id> removes transaction

### TestIBKRAllocation (12 tests)
- `test_get_portfolios` - Verify portfolios endpoint returns active portfolios
- `test_allocate_transaction` - Allocate transaction 100% to single portfolio
- `test_get_pending_dividends` - GET /ibkr/dividends/pending returns unmatched dividends
- `test_match_dividend` - Match IBKR dividend transaction to existing dividend records
- `test_unallocate_transaction` - Remove all allocations from processed transaction
- `test_get_transaction_allocations` - GET /ibkr/inbox/<id>/allocations returns allocation details
- `test_update_transaction_allocations` - Modify allocations to 60/40 split across portfolios
- Additional allocation scenarios and edge cases

### TestIBKRBulkOperations (1 test)
- `test_bulk_allocate` - Allocate multiple transactions with same percentage split

### TestIBKRConfigErrors (8 tests)
- `test_save_config_missing_flex_token` - POST /ibkr/config rejects missing flex_token
- `test_save_config_missing_flex_query_id` - POST /ibkr/config rejects missing flex_query_id
- `test_save_config_empty_payload` - POST /ibkr/config rejects empty dict payload
- `test_save_config_no_payload` - POST /ibkr/config rejects null payload
- `test_save_config_invalid_token_expires_at` - POST /ibkr/config rejects invalid date format
- `test_save_config_service_error` - POST /ibkr/config handles service exceptions
- `test_delete_config_not_found` - DELETE /ibkr/config returns 404 for missing config
- `test_delete_config_service_error` - DELETE /ibkr/config handles service exceptions

### TestIBKRConnectionErrors (6 tests)
- `test_connection_missing_flex_token` - POST /ibkr/config/test rejects missing flex_token
- `test_connection_missing_flex_query_id` - POST /ibkr/config/test rejects missing flex_query_id
- `test_connection_empty_payload` - POST /ibkr/config/test rejects empty payload
- `test_connection_success` - POST /ibkr/config/test handles successful connection
- `test_connection_failure` - POST /ibkr/config/test handles failed connection
- `test_connection_api_failure` - POST /ibkr/config/test handles API exceptions

### TestIBKRImportErrors (4 tests)
- `test_import_missing_config` - POST /ibkr/flex-query returns 404 when no config
- `test_import_disabled_config` - POST /ibkr/flex-query rejects disabled config
- `test_import_api_failure` - POST /ibkr/flex-query handles IBKR API failures
- `test_import_exception` - POST /ibkr/flex-query handles general exceptions

### TestIBKRInboxErrors (7 tests)
- `test_get_transaction_not_found` - GET /ibkr/inbox/<id> returns 404 for invalid ID
- `test_ignore_transaction_not_found` - POST /ibkr/inbox/<id>/ignore returns 404
- `test_delete_transaction_not_found` - DELETE /ibkr/inbox/<id> returns 404
- `test_delete_transaction_service_error` - DELETE /ibkr/inbox/<id> handles service errors
- `test_get_inbox_count_service_error` - GET /ibkr/inbox/count handles service errors
- `test_get_eligible_portfolios_transaction_not_found` - GET eligible-portfolios returns 404
- `test_get_eligible_portfolios_service_error` - GET eligible-portfolios handles service errors

### TestIBKRAllocationErrors (8 tests)
- `test_allocate_transaction_not_found` - POST /ibkr/inbox/<id>/allocate returns 404
- `test_allocate_missing_allocations` - POST /ibkr/inbox/<id>/allocate rejects missing allocations
- `test_match_dividend_not_found` - POST /ibkr/inbox/<id>/match-dividend returns 404
- `test_match_dividend_missing_fields` - POST /ibkr/inbox/<id>/match-dividend rejects missing dividend_ids
- `test_unallocate_transaction_not_found` - POST /ibkr/inbox/<id>/unallocate returns 404
- `test_update_allocations_not_found` - PUT /ibkr/inbox/<id>/allocations returns 404
- `test_update_allocations_missing_allocations` - PUT allocations rejects missing allocations field
- `test_update_allocations_value_error` - PUT allocations handles validation errors
- `test_update_allocations_general_error` - PUT allocations handles general exceptions

### TestIBKRBulkOperationsErrors (7 tests)
- `test_bulk_allocate_missing_transaction_ids` - POST bulk-allocate rejects missing transaction_ids
- `test_bulk_allocate_empty_transaction_ids` - POST bulk-allocate rejects empty transaction_ids
- `test_bulk_allocate_missing_allocations` - POST bulk-allocate rejects missing allocations
- `test_bulk_allocate_empty_allocations` - POST bulk-allocate rejects empty allocations array
- `test_bulk_allocate_invalid_percentage_sum` - POST bulk-allocate rejects percentages not summing to 100
- `test_bulk_allocate_partial_failure` - POST bulk-allocate handles individual transaction failures
- `test_bulk_allocate_general_error` - POST bulk-allocate handles general exceptions

---

## Key Patterns

**Allocation Validation**: All allocation endpoints validate that percentages sum to exactly 100%. Service layer handles validation and raises ValueError for invalid splits, which routes convert to 400 Bad Request responses.

**Token Security**: IBKR configuration includes flex_token and flex_query_id for API authentication. Tests verify required fields are present and configuration can be safely stored and retrieved.

**Bulk Operations**: Bulk allocation endpoint processes multiple transactions with same percentage split, handling partial failures gracefully by returning individual results for each transaction.

**Service Layer Delegation**: Routes act as thin controllers, delegating all business logic to IBKRTransactionService, IBKRConfigService, and PortfolioService. Tests verify proper error handling and response formatting.

**Error Path Coverage**: Comprehensive error path testing (43 tests) ensures all validation failures, missing resources (404), and service exceptions return appropriate HTTP status codes and error messages.

---

## Running Tests

```bash
# Run all IBKR route tests
pytest tests/api/test_ibkr_routes.py -v

# Run specific test class
pytest tests/api/test_ibkr_routes.py::TestIBKRAllocation -v

# Run with coverage
pytest tests/api/test_ibkr_routes.py --cov=app/routes/ibkr_routes --cov-report=term-missing

# Run integration tests only (skip error paths)
pytest tests/api/test_ibkr_routes.py -k "not Error" -v
```

---

## Related Documentation

- **Service Tests**: `tests/docs/services/IBKR_TRANSACTION_SERVICE_TESTS.md`
- **Service Tests**: `tests/docs/services/IBKR_FLEX_SERVICE_TESTS.md`
- **Service Tests**: `tests/docs/services/IBKR_CONFIG_SERVICE_TESTS.md`
- **Remediation Plan**: `todo/ROUTE_REFACTORING_REMEDIATION_PLAN.md`

---

**Last Updated**: Phase 5 + Documentation Condensing
