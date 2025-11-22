# IBKR Transaction Service Tests

**File**: `tests/services/test_ibkr_transaction_service.py`
**Service**: `app/services/ibkr_transaction_service.py`
**Test Count**: 69 tests across 13 test classes
**Coverage**: 90% (261/299 statements)
**Status**: ✅ All tests passing
**Bugs Fixed**: 1 critical (ReinvestmentStatus enum)

> **💡 Detailed Test Information**: For detailed explanations of each test including
> WHY it exists and what business logic it validates, see the docstrings in the test file.
> Your IDE will show these when hovering over test names.

---

## Overview

Comprehensive test suite for the IBKRTransactionService class, which handles processing of IBKR (Interactive Brokers) transactions and allocating them across portfolios. The service manages allocation validation, fund creation, transaction processing with multi-portfolio allocation, commission/fee allocation, transaction allocation modifications, and dividend matching. This test suite achieved 90% coverage and discovered one critical bug (ReinvestmentStatus string vs enum mismatch).

---

## Test Organization

### TestAllocationValidation (9 tests)
- `test_validate_allocations_success_100_percent` - Allocations summing to 100% are valid
- `test_validate_allocations_single_100_percent` - Single allocation of 100% is valid
- `test_validate_allocations_three_way_split` - Three-way allocation split totaling 100%
- `test_validate_allocations_floating_point_acceptable` - Small floating point errors acceptable
- `test_validate_allocations_too_high` - Allocations over 100% rejected
- `test_validate_allocations_too_low` - Allocations under 100% rejected
- `test_validate_allocations_empty_list` - Empty allocation list rejected
- `test_validate_allocations_negative_percentage` - Negative percentages rejected
- `test_validate_allocations_missing_portfolio_id` - Missing portfolio_id rejected

### TestFundCreation (5 tests)
- `test_get_existing_fund_by_isin` - Existing fund retrieved by ISIN
- `test_get_existing_fund_by_symbol` - Existing fund retrieved by symbol when ISIN doesn't match
- `test_create_new_fund_with_symbol_and_isin` - New fund created with both symbol and ISIN
- `test_create_new_fund_with_only_symbol` - New fund created with only symbol (no ISIN)
- `test_create_new_fund_name_uses_symbol` - Fund name defaults to symbol

### TestPortfolioFundCreation (2 tests)
- `test_get_existing_portfolio_fund` - Existing portfolio-fund relationship retrieved
- `test_create_new_portfolio_fund` - New portfolio-fund relationship created

### TestProcessTransactionAllocation (8 tests)
- `test_process_single_allocation_100_percent` - Transaction with single 100% allocation
- `test_process_split_allocation` - Transaction split across two portfolios
- `test_process_creates_portfolio_fund_relationship` - Processing creates portfolio-fund relationship
- `test_process_creates_transaction_record` - Processing creates Transaction record
- `test_process_already_processed_transaction` - Already-processed transaction rejected
- `test_process_nonexistent_transaction` - Nonexistent transaction handled
- `test_process_invalid_allocations` - Invalid allocations rejected
- `test_process_creates_fund_if_not_exists` - Fund created if doesn't exist

### TestDividendMatching (6 tests)
- `test_get_pending_dividends_no_filter` - All pending dividends retrieved
- `test_get_pending_dividends_filter_by_symbol` - Pending dividends filtered by symbol
- `test_match_dividend_single` - IBKR dividend matched to single existing dividend
- `test_match_dividend_multiple_portfolios` - Dividend split across multiple portfolios
- `test_match_dividend_non_dividend_transaction` - Non-dividend transaction rejected
- `test_match_dividend_already_processed` - Already-processed dividend rejected

### TestModifyAllocations (6 tests)
- `test_modify_allocations_change_percentages` - Allocation percentages for existing portfolios modified
- `test_modify_allocations_add_portfolio` - New portfolio added to existing allocations
- `test_modify_allocations_remove_portfolio` - Portfolio removed from allocations
- `test_modify_allocations_not_processed` - Unprocessed transaction modification rejected
- `test_modify_allocations_invalid_percentages` - Invalid allocations fail validation
- `test_modify_allocations_not_found` - Nonexistent transaction handled

### TestCommissionAllocation (9 tests)
- `test_process_allocation_with_zero_commission` - No fee transaction created when commission is zero
- `test_commission_allocated_proportionally` - Commission split proportionally across portfolios
- `test_commission_rounding_fractional_cents` - Fractional cents in commission allocation handled
- `test_modify_allocations_updates_fee_transactions` - Fee transactions updated when allocations modified
- `test_modify_allocations_removes_fee_transactions` - Fee transaction removed when portfolio allocation removed
- `test_modify_allocations_adds_fee_transactions` - Fee transaction created when new portfolio allocation added
- `test_fee_transaction_has_correct_structure` - Fee transactions have correct field values
- `test_fee_transaction_linked_to_ibkr` - Fee transactions linked to IBKR via IBKRTransactionAllocation
- `test_fee_transaction_linked_to_ibkr_split_allocation` - Fee transactions linked in split allocations

### TestTransactionManagement (8 tests)
- `test_get_transaction_success` - Existing transaction retrieved
- `test_get_transaction_not_found` - 404 raised for non-existent transaction
- `test_ignore_transaction_success` - Transaction marked as ignored
- `test_ignore_transaction_already_processed` - Already-processed transaction cannot be ignored
- `test_ignore_transaction_not_found` - Non-existent transaction handled
- `test_delete_transaction_success` - Transaction deleted
- `test_delete_transaction_already_processed` - Already-processed transaction cannot be deleted
- `test_delete_transaction_not_found` - Non-existent transaction handled

### TestGetInbox (5 tests)
- `test_get_inbox_default_pending` - Pending transactions returned by default, ordered by date descending
- `test_get_inbox_filter_by_status` - Transactions filtered by status
- `test_get_inbox_filter_by_transaction_type` - Transactions filtered by type
- `test_get_inbox_empty` - Empty list returned when no transactions match
- `test_get_inbox_response_format` - Serialized transaction data structure validated

### TestGetInboxCount (3 tests)
- `test_get_inbox_count_default_pending` - Pending transactions counted by default
- `test_get_inbox_count_filter_by_status` - Transactions counted by status
- `test_get_inbox_count_zero` - Zero returned when no transactions match

### TestUnallocateTransaction (4 tests)
- `test_unallocate_transaction_with_transactions` - Allocations and portfolio transactions deleted, status reverted
- `test_unallocate_transaction_orphaned_allocations` - Orphaned allocations without transaction_id handled
- `test_unallocate_transaction_not_found` - 404 returned for non-existent transaction
- `test_unallocate_transaction_not_processed` - 400 returned when transaction not processed

### TestGetTransactionAllocations (3 tests)
- `test_get_transaction_allocations` - Allocation details with portfolio info returned
- `test_get_transaction_allocations_no_allocations` - Empty allocations returned for unallocated transaction
- `test_get_transaction_allocations_not_found` - 404 returned for non-existent transaction

### TestGroupedAllocations (1 test)
- `test_get_grouped_allocations_combines_stock_and_commission` - Stock and fee transactions grouped per portfolio

---

## Key Patterns

**Allocation Validation**: All allocations must sum to exactly 100% (±0.01 for floating point), enforced before any database operations

**Proportional Splitting**: Transaction amounts, shares, and commissions are split across portfolios based on allocation percentages, with separate fee transactions (shares=0) created for commission allocation

**Test Isolation**: Each test uses unique UUIDs for all entities (funds, portfolios, transactions) to prevent test pollution and enable parallel execution

**Critical Bug Fix**: Tests discovered and validated fix for ReinvestmentStatus enum issue where service used string literal "pending" instead of enum value, completely breaking dividend matching functionality

---

## Running Tests

```bash
# Run all IBKR transaction service tests
pytest backend/tests/services/test_ibkr_transaction_service.py -v

# Run specific test class
pytest backend/tests/services/test_ibkr_transaction_service.py::TestAllocationValidation -v
pytest backend/tests/services/test_ibkr_transaction_service.py::TestDividendMatching -v
pytest backend/tests/services/test_ibkr_transaction_service.py::TestCommissionAllocation -v
pytest backend/tests/services/test_ibkr_transaction_service.py::TestProcessTransactionAllocation -v

# Run with coverage
pytest backend/tests/services/test_ibkr_transaction_service.py \
    --cov=app/services/ibkr_transaction_service \
    --cov-report=term-missing

# Run bug fix validation tests
pytest backend/tests/services/test_ibkr_transaction_service.py::TestDividendMatching::test_get_pending_dividends_no_filter -v
pytest backend/tests/services/test_ibkr_transaction_service.py::TestDividendMatching::test_get_pending_dividends_filter_by_symbol -v
```

---

## Related Documentation

- **Service Code**: `app/services/ibkr_transaction_service.py`
- **Bug Fix Documentation**: `BUG_FIXES_1.3.3.md` (Bug #4 - ReinvestmentStatus enum)
- **Related Service Tests**: `IBKR_FLEX_SERVICE_TESTS.md`, `IBKR_CONFIG_SERVICE_TESTS.md`
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **Database Models**: `app/models.py` (IBKRTransaction, IBKRTransactionAllocation, Transaction, Dividend)

---

**Last Updated**: Version 1.3.3 Phase 6 + Documentation Condensing
**Coverage**: 90% (exceeds 85% target, all critical paths tested)
