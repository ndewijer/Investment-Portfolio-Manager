# TransactionService Test Suite Documentation

**File**: `tests/test_transaction_service.py`\
**Service**: `app/services/transaction_service.py`\
**Tests**: 26 tests\
**Coverage**: 95% (147 statements, 8 missed)\
**Created**: Version 1.3.3 (Phase 3)

---

## Overview

This test suite provides comprehensive coverage of the TransactionService, including:
- Transaction retrieval with batch loading optimization
- Transaction formatting with IBKR allocation handling
- Transaction CRUD operations (buy, sell, dividend types)
- Realized gain/loss calculation and tracking
- IBKR allocation management and cleanup
- Current position calculation
- Sell transaction processing

**Key focus areas**:
- Buy vs sell vs dividend transaction handling
- Realized gain/loss calculation for sell transactions
- IBKR allocation linking and status management
- Average cost basis calculation
- Position tracking with multiple transaction types

**For general testing information**, see:
- `TESTING_INFRASTRUCTURE.md` - Fixtures, factories, coverage
- `BUG_FIXES_1.3.3.md` - Bugs discovered during test development

---

## Test Organization

**File Structure**:
```python
class TestGetTransactions:              # 3 tests - Transaction retrieval
class TestFormatTransaction:            # 3 tests - Transaction formatting
class TestCreateTransaction:            # 3 tests - Transaction creation
class TestUpdateTransaction:            # 3 tests - Transaction updates
class TestDeleteTransaction:            # 3 tests - Transaction deletion
class TestCalculateCurrentPosition:     # 5 tests - Position calculation
class TestProcessSellTransaction:       # 3 tests - Sell processing
class TestEdgeCases:                    # 3 tests - Error handling
```

**Total**: 26 tests, all passing

---

## Transaction-Specific Testing Patterns

### Direct Object Creation

For TransactionService tests, we use **direct object creation** for all models to avoid SubFactory conflicts.

**Pattern**:
```python
# Create related objects with factories
pf = PortfolioFundFactory()
db_session.commit()

# Create transaction DIRECTLY (not via factory)
from app.models import Transaction
import uuid

txn = Transaction(
    id=str(uuid.uuid4()),
    portfolio_fund_id=pf.id,
    type="buy",  # or "sell", "dividend"
    shares=100,
    cost_per_share=10.0,
    date=date(2024, 1, 1),
)
db_session.add(txn)
db_session.commit()
```

### IBKR Transaction Creation

IBKR models have specific field names:

```python
from app.models import IBKRTransaction, IBKRTransactionAllocation

# Create IBKR transaction
ibkr_txn = IBKRTransaction(
    id=str(uuid.uuid4()),
    ibkr_transaction_id="IBKR-UNIQUE-ID",  # Must be unique
    transaction_date=date(2024, 1, 1),     # NOT 'date'
    transaction_type="buy",                 # NOT 'type'
    quantity=100,
    price=10.0,
    total_amount=-1000.0,                   # NOT 'amount'
    currency="USD",
    symbol="VWCE",
    description="Buy 100 VWCE",
    status="processed",
)

# Create allocation
allocation = IBKRTransactionAllocation(
    id=str(uuid.uuid4()),
    ibkr_transaction_id=ibkr_txn.id,
    portfolio_id=portfolio.id,
    transaction_id=txn.id,
    allocated_shares=100.0,                 # NOT 'shares'
    allocation_percentage=100.0,
    allocated_amount=-1000.0,
)
```

---

## Test Suite Walkthrough

### TestGetTransactions (3 tests)

#### Test 1: `test_get_all_transactions`

**Purpose**: Verify retrieval of all transactions across all portfolios

**Scenario**: Multiple portfolios with transactions

**Test data**:
- Portfolio 1, Fund 1: Buy 100 shares @ $10
- Portfolio 2, Fund 2: Sell 50 shares @ $15

**Expected**: Both transactions returned as formatted dicts

**Why**: Tests basic retrieval and formatting

---

#### Test 2: `test_get_portfolio_transactions`

**Purpose**: Verify portfolio-specific retrieval with batch loading

**Scenario**: Two portfolios, only one portfolio's transactions should return

**Test data**:
- Portfolio 1: Buy 100 shares
- Portfolio 2: Buy 50 shares

**Expected**: Only Portfolio 1 transaction returned

**Why**: Tests filtering and batch loading optimization

---

#### Test 3: `test_get_portfolio_transactions_with_ibkr_allocation`

**Purpose**: Verify IBKR allocation data included in results

**Scenario**: Transaction linked to IBKR transaction

**Expected**:
- `ibkr_linked` = True
- `ibkr_transaction_id` = IBKR transaction ID

**Why**: Tests IBKR integration

---

### TestFormatTransaction (3 tests)

**Method tested**: `TransactionService.format_transaction(transaction, ibkr_allocation, portfolio_fund_lookup, batch_mode)`

#### Test 4: `test_format_transaction_basic`

**Purpose**: Test formatting without IBKR allocation

**Expected format**:
```python
{
    "id": transaction_id,
    "portfolio_fund_id": pf_id,
    "fund_name": "Fund Name",
    "date": "2024-01-01",
    "type": "buy",
    "shares": 100,
    "cost_per_share": 10.0,
    "ibkr_linked": False,
    "ibkr_transaction_id": None,
}
```

---

#### Test 5: `test_format_transaction_with_ibkr_allocation`

**Purpose**: Test querying and including IBKR data when not pre-loaded

**Expected**:
- Service queries for allocation
- `ibkr_linked` = True
- `ibkr_transaction_id` populated

**Why**: Tests backwards compatibility path

---

#### Test 6: `test_format_transaction_batch_mode`

**Purpose**: Test batch mode with pre-loaded lookups

**Scenario**: Pre-load `portfolio_fund_lookup` dict

**Expected**: Uses lookup, doesn't query relationships

**Why**: Tests N+1 query prevention

---

### TestCreateTransaction (3 tests)

**Method tested**: `TransactionService.create_transaction(data)`

#### Test 7: `test_create_buy_transaction`

**Purpose**: Test basic buy transaction creation

**Test data**:
```python
{
    "portfolio_fund_id": pf.id,
    "date": "2024-01-01",
    "type": "buy",
    "shares": 100,
    "cost_per_share": 10.0,
}
```

**Expected**: Transaction created and persisted

---

#### Test 8: `test_create_sell_transaction_with_realized_gain`

**Purpose**: Test sell transaction creates realized gain/loss record

**Scenario**:
- Existing position: 100 shares @ $10 cost
- Sell: 50 shares @ $15

**Expected**:
- Transaction created
- RealizedGainLoss record created:
  - `shares_sold` = 50
  - `cost_basis` = $500 (50 × $10)
  - `sale_proceeds` = $750 (50 × $15)
  - `realized_gain_loss` = $250 (profit)

**Why**: Tests sell transaction processing and gain calculation

---

#### Test 9: `test_create_sell_transaction_insufficient_shares`

**Purpose**: Test validation prevents selling more than owned

**Scenario**: No existing position, attempt to sell 50 shares

**Expected**: `ValueError` raised with "Insufficient shares"

---

### TestUpdateTransaction (3 tests)

**Method tested**: `TransactionService.update_transaction(transaction_id, data)`

#### Test 10: `test_update_buy_transaction`

**Purpose**: Test basic field updates for buy transaction

**Updates**:
- Date: 2024-01-01 → 2024-01-15
- Shares: 100 → 150
- Cost per share: $10 → $12

**Expected**: All fields updated correctly

---

#### Test 11: `test_update_sell_transaction_recalculates_gain`

**Purpose**: Test updating sell transaction recalculates realized gain

**Scenario**:
- Buy 100 @ $10
- Sell 50 @ $15 (original, gain = $250)
- Update sell to 50 @ $20 (new price)

**Expected**: Realized gain recalculated:
- Cost basis: 50 × $10 = $500
- Sale proceeds: 50 × $20 = $1000
- Realized gain: $500 (updated)

**Why**: Tests gain recalculation on updates

---

#### Test 12: `test_update_buy_to_sell_creates_realized_gain`

**Purpose**: Test changing transaction type from buy to sell

**Scenario**:
- Buy1: 100 shares @ $10
- Buy2: 50 shares @ $10
- Change Buy2 to Sell: 50 shares @ $15

**Expected**:
- Realized gain record created
- Gain = $250 (50 shares × ($15 - $10))

**Why**: Tests type change handling

---

### TestDeleteTransaction (3 tests)

**Method tested**: `TransactionService.delete_transaction(transaction_id)`

#### Test 13: `test_delete_buy_transaction`

**Purpose**: Test basic transaction deletion

**Expected**:
- Transaction removed
- Returns deletion details

---

#### Test 14: `test_delete_sell_transaction_removes_realized_gain`

**Purpose**: Test deleting sell also deletes realized gain record

**Expected**:
- Transaction deleted
- RealizedGainLoss record deleted (cascade)
- Returns `realized_gain_deleted` = True

**Why**: Tests cascade deletion

---

#### Test 15: `test_delete_transaction_with_ibkr_allocation`

**Purpose**: Test deleting transaction reverts IBKR status to pending

**Scenario**:
- IBKR transaction status = "processed"
- Transaction has IBKR allocation (only one)

**Expected**:
- Transaction deleted
- Allocation deleted (cascade)
- IBKR transaction status reverted to "pending"
- Returns `ibkr_reverted` = True

**Why**: Tests IBKR cleanup logic

---

### TestCalculateCurrentPosition (5 tests)

**Method tested**: `TransactionService.calculate_current_position(portfolio_fund_id)`

#### Test 16: `test_calculate_position_with_buys_only`

**Purpose**: Test position calculation with only purchases

**Transactions**:
- Buy 100 @ $10
- Buy 50 @ $12

**Expected**:
- `total_shares` = 150
- `total_cost` = $1600
- `average_cost` = $10.67

---

#### Test 17: `test_calculate_position_with_buys_and_sells`

**Purpose**: Test position with buys and sells (validates bug fix)

**Transactions**:
- Buy 100 @ $10 = $1000
- Buy 50 @ $12 = $600
- Sell 30 @ $15

**Expected (after bug fix)**:
- Position before sell: 150 shares, $1600 cost
- Average cost: $10.67
- Sell 30 shares reduces cost by: 30 × $10.67 = $320
- Final position:
  - `total_shares` = 120
  - `total_cost` = $1280
  - `average_cost` = $10.67

**Bug fix**: Cost basis now reduced by average cost, not sale price

---

#### Test 18: `test_calculate_position_with_dividend_shares`

**Purpose**: Test position includes dividend reinvestment shares

**Transactions**:
- Buy 100 @ $10
- Dividend reinvestment: 5 shares @ $10

**Expected**: `total_shares` = 105

**Why**: Dividend shares must be included in position

---

#### Test 19: `test_calculate_position_empty`

**Purpose**: Test empty portfolio returns zeros

**Expected**:
- `total_shares` = 0
- `total_cost` = 0
- `average_cost` = 0

---

#### Test 20: `test_calculate_position_all_shares_sold`

**Purpose**: Test complete liquidation returns zeros

**Transactions**:
- Buy 100 @ $10
- Sell 100 @ $15

**Expected**: All values = 0

**Why**: Tests complete position closure

---

### TestProcessSellTransaction (3 tests)

**Method tested**: `TransactionService.process_sell_transaction(portfolio_fund_id, shares, price, date)`

#### Test 21: `test_process_sell_transaction`

**Purpose**: Test sell processing creates transaction and realized gain

**Scenario**:
- Buy 100 @ $10
- Process sell: 50 @ $15

**Expected**:
- Transaction created (type="sell")
- `realized_gain_loss` = $250
- RealizedGainLoss record in database

---

#### Test 22: `test_process_sell_transaction_insufficient_shares`

**Purpose**: Test validation prevents overselling

**Scenario**: No shares, attempt to sell

**Expected**: `ValueError` raised

---

#### Test 23: `test_process_sell_transaction_realized_loss`

**Purpose**: Test loss calculation (negative gain)

**Scenario**:
- Buy 100 @ $15
- Sell 50 @ $10 (loss)

**Expected**:
- Cost basis: $750
- Sale proceeds: $500
- Realized loss: -$250

---

### TestEdgeCases (3 tests)

#### Test 24: `test_delete_nonexistent_transaction`

**Purpose**: Test error handling for invalid ID

**Expected**: `ValueError` with "not found"

---

#### Test 25: `test_update_nonexistent_transaction`

**Purpose**: Test 404 handling

**Expected**: `NotFound` exception

---

#### Test 26: `test_get_portfolio_transactions_empty_portfolio`

**Purpose**: Test empty result handling

**Expected**: Empty list `[]`

---

## Coverage Analysis

**Total**: 95% coverage (147 statements, 8 missed)

**Uncovered lines** (8 statements):
- `240`: Validation error in `update_transaction()` (insufficient shares check)
- `363-365`: Exception handler in `delete_transaction()` (rollback)
- `420`: ValueError in `calculate_current_position()` (insufficient shares)
- `479-481`: Exception handler in `process_sell_transaction()` (rollback)

**Why uncovered**: Exception handlers and error paths requiring complex scenarios

**Why acceptable**: 95% far exceeds 90% target, all business logic covered

---

## Bug Fix: Cost Basis Calculation

**Bug discovered**: `calculate_current_position()` was reducing cost basis by **sale price** instead of **average cost**.

**Impact**: All sell transactions had incorrect cost basis and realized gains.

**Fix**: Line 415-418 in `transaction_service.py`:
```python
# BEFORE (BUGGY)
total_cost -= transaction.cost_per_share * transaction.shares  # Sale price!

# AFTER (FIXED)
average_cost = total_cost / total_shares if total_shares > 0 else 0
total_cost -= average_cost * transaction.shares  # Average cost ✅
```

**Example**:
- Buy 100 @ $10 = $1000 cost
- Sell 50 @ $15
- **Buggy**: Cost reduced by 50 × $15 = $750 → Remaining cost = $250 ❌
- **Fixed**: Cost reduced by 50 × $10 = $500 → Remaining cost = $500 ✅

**Validated by**: Tests 8, 11, 12, 17, 21, 23

See `BUG_FIXES_1.3.3.md` for full analysis.

---

## Running Tests

### All transaction tests:
```bash
cd backend
source .venv/bin/activate
pytest tests/test_transaction_service.py -v
```

### Specific test class:
```bash
pytest tests/test_transaction_service.py::TestCreateTransaction -v
```

### Specific test:
```bash
pytest tests/test_transaction_service.py::TestCalculateCurrentPosition::test_calculate_position_with_buys_and_sells -xvs
```

### With coverage:
```bash
pytest tests/test_transaction_service.py --cov=app.services.transaction_service --cov-report=term-missing
```

---

## Related Documentation

- **Testing Infrastructure**: `TESTING_INFRASTRUCTURE.md` - General testing setup
- **Bug Fixes**: `BUG_FIXES_1.3.3.md` - Bugs found during test development
- **Dividend Tests**: `DIVIDEND_SERVICE_TESTS.md` - Related service tests
- **Test Index**: `README.md` - All test documentation
- **Service Code**: `app/services/transaction_service.py`
- **Test Code**: `tests/test_transaction_service.py`
- **Materialized View Invalidation Tests**: `TRANSACTION_MATERIALIZED_VIEW_INVALIDATION_TESTS.md`

---

**Document Version**: 1.5.1
**Last Updated**: 2026-02-06
