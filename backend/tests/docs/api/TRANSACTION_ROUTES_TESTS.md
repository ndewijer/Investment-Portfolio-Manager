# Transaction Routes Integration Tests

**File**: `tests/api/test_transaction_routes.py`
**Route File**: `app/routes/transaction_routes.py`
**Test Count**: 18 tests (12 integration + 6 error path)
**Coverage**: 100% (78/78 statements)
**Status**: ✅ All tests passing

---

## Docstring Reference

All test functions include detailed docstrings explaining their purpose. Refer to the source file for implementation details.

---

## Overview

Integration tests for transaction management API endpoints. Tests verify transaction CRUD operations, filtering, and interaction with the transaction service layer.

**Endpoints Tested**:
- GET /api/transaction - List/filter transactions
- POST /api/transaction - Create transaction
- GET /api/transaction/<id> - Get transaction detail
- PUT /api/transaction/<id> - Update transaction
- DELETE /api/transaction/<id> - Delete transaction

**Transaction Types**: buy, sell, dividend, fee

---

## Test Organization

### TestTransactionList (3 tests)
- `test_list_transactions_empty` - Empty transaction list returns empty array
- `test_list_all_transactions` - Returns all transactions across portfolios
- `test_list_transactions_filtered_by_portfolio` - Filter by portfolio_id parameter

### TestTransactionCreate (3 tests)
- `test_create_buy_transaction` - Create buy transaction successfully
- `test_create_sell_transaction` - Create sell with prior buy for shares
- `test_create_dividend_transaction` - Create dividend transaction

### TestTransactionRetrieveUpdateDelete (6 tests)
- `test_get_transaction` - Retrieve transaction detail
- `test_get_nonexistent_transaction` - Returns 404 for missing transaction
- `test_update_transaction` - Update transaction fields
- `test_update_nonexistent_transaction` - Returns 404 for missing transaction
- `test_delete_transaction` - Delete transaction successfully
- `test_delete_nonexistent_transaction` - Returns 400 for missing transaction

### TestTransactionErrors (6 tests)
- `test_create_transaction_service_error` - POST handles service exceptions (500)
- `test_get_fund_transactions_service_error` - GET with fund_id handles errors
- `test_get_portfolio_fund_transactions_service_error` - GET with portfolio_fund_id handles errors
- `test_update_transaction_not_found` - PUT handles ValueError (404)
- `test_update_transaction_general_error` - PUT handles general exceptions (500)
- `test_delete_transaction_service_error` - DELETE handles service errors (500)

---

## Key Patterns

### Testing Approach
- **Sell transactions**: Require prior buy transactions to establish share ownership
- **Filtering**: Tests verify portfolio_id query parameter filtering
- **Mutations**: Always verify both HTTP response and database state
- **Error paths**: Use `unittest.mock.patch` to simulate service failures

### Helper Functions
- `create_fund()` - Creates Fund with all required fields (consistent across route tests)

### Error Handling
- GET non-existent: Returns 404
- DELETE non-existent: Returns 400 (documents actual API behavior)
- Service errors: Return 500 with error message

### Business Logic Verification
- **Realized Gain/Loss**: Sell transactions trigger automatic calculation (tested at service layer)
- **Share Tracking**: Service validates share counts and prevents overselling

### Phase 4c Error Path Testing
- Converted all tests from `monkeypatch` to `unittest.mock.patch` for consistency
- Added comprehensive error path tests covering all exception handlers
- Coverage improvement: 81% → 100%

---

## Running Tests

### Run all transaction route tests:
```bash
pytest tests/api/test_transaction_routes.py -v
```

### Run specific test class:
```bash
pytest tests/api/test_transaction_routes.py::TestTransactionCreate -v
```

### Run with timing:
```bash
pytest tests/api/test_transaction_routes.py -v --durations=10
```

### Run without coverage (faster):
```bash
pytest tests/api/test_transaction_routes.py -v --no-cov
```

---

## Test Results

**All 18 tests passing** ✅

**Execution Time**: ~0.31 seconds (full suite)
**Coverage**: 100% (78/78 statements, 0 missing lines)

**Coverage History**: 81% → 100% (Phase 4c error path testing)

---

## Related Documentation

- **Service Tests**: `tests/docs/services/TRANSACTION_SERVICE_TESTS.md`
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/transaction_routes.py`
- **Transaction Service**: `app/services/transaction_service.py`

---

**Last Updated**: Phase 5 (Route Integration Tests) + Phase 4c (Error Path Testing)
**Maintainer**: See git history
