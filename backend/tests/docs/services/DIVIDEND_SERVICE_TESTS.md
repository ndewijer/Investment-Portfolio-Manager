# DividendService Test Suite Documentation

**File**: `tests/test_dividend_service.py`\
**Service**: `app/services/dividend_service.py`\
**Tests**: 21 tests\
**Coverage**: 91% (119 statements, 11 missed)\
**Created**: Version 1.3.3 (Phase 3)

---

## Overview

This test suite provides comprehensive coverage of the DividendService, including share calculations, CRUD operations for dividends, and reinvestment handling for STOCK dividends.

**Key focus areas**:
- Share ownership calculations (considering buy/sell/dividend transactions)
- CASH vs STOCK dividend behavior
- Dividend reinvestment logic
- Validation and error handling
- Edge cases

**For general testing information**, see:
- `TESTING_INFRASTRUCTURE.md` - Fixtures, factories, coverage
- `BUG_FIXES_1.3.3.md` - Bugs discovered during test development

---

## Test Organization

**File Structure**:
```python
class TestCalculateSharesOwned:         # 5 tests - Share calculations
class TestCreateDividendCash:           # 2 tests - CASH dividend creation
class TestCreateDividendStock:          # 3 tests - STOCK dividend creation
class TestUpdateDividend:               # 6 tests - Dividend updates
class TestDeleteDividend:               # 2 tests - Dividend deletion
class TestEdgeCases:                    # 3 tests - Edge cases
```

**Total**: 21 tests, all passing

---

## Dividend-Specific Testing Patterns

### Direct Object Creation

For DividendService tests, we use **direct object creation** instead of factories to avoid SubFactory conflicts.

**Pattern**:
```python
# Create related objects with factories
fund = FundFactory(dividend_type=DividendType.STOCK)
portfolio = PortfolioFactory()
portfolio_fund = PortfolioFundFactory(fund=fund, portfolio=portfolio)
db_session.commit()

# Create dividend DIRECTLY (not via factory)
from app.models import Dividend
import uuid

dividend = Dividend(
    id=str(uuid.uuid4()),
    fund_id=fund.id,
    portfolio_fund_id=portfolio_fund.id,
    record_date=date(2024, 3, 1),
    ex_dividend_date=date(2024, 2, 28),
    dividend_per_share=0.50,
    shares_owned=100,
    total_amount=50.0,
    reinvestment_status=ReinvestmentStatus.COMPLETED
)
db_session.add(dividend)
db_session.commit()
```

**Why**: See `TESTING_INFRASTRUCTURE.md` - Factory Pattern section

### Transaction Creation

Similar pattern for transactions:
```python
from app.models import Transaction
import uuid

txn = Transaction(
    id=str(uuid.uuid4()),
    portfolio_fund_id=portfolio_fund.id,
    type="buy",  # or "sell", "dividend"
    shares=100,
    cost_per_share=10.0,
    date=date(2024, 1, 1)
)
db_session.add(txn)
db_session.commit()
```

---

## Test Suite Walkthrough

### TestCalculateSharesOwned (5 tests)

**Method tested**: `DividendService.calculate_shares_owned(portfolio_fund_id, record_date)`

**What it does**: Calculates shares owned on a specific date by processing all buy/sell/dividend transactions up to that date.

---

#### Test 1: `test_calculate_shares_buy_only`

**Scenario**: Multiple buy transactions, no sells

**Test data**:
- Buy 100 shares on Jan 1
- Buy 50 shares on Feb 1
- Buy 25 shares on Mar 1
- Calculate shares on Apr 1

**Expected**: 175 shares (100 + 50 + 25)

**Why**: Tests basic accumulation logic

---

#### Test 2: `test_calculate_shares_buy_and_sell`

**Scenario**: Buys and sells

**Test data**:
- Buy 100 shares on Jan 1
- Buy 50 shares on Feb 1
- Sell 30 shares on Mar 1
- Calculate shares on Apr 1

**Expected**: 120 shares (100 + 50 - 30)

**Why**: Tests subtraction logic for sells

---

#### Test 3: `test_calculate_shares_before_first_transaction`

**Scenario**: Calculate shares before any transactions exist

**Test data**:
- First buy on Feb 1 (100 shares)
- Calculate shares on Jan 1 (BEFORE the buy)

**Expected**: 0 shares

**Why**: Tests date filtering - future transactions should be ignored

---

#### Test 4: `test_calculate_shares_only_counts_up_to_record_date`

**Scenario**: Transactions after record date should be excluded

**Test data**:
- Buy 100 shares on Jan 1
- Buy 50 shares on Feb 1
- Buy 25 shares on Apr 1 (AFTER record date)
- Calculate shares on Mar 1

**Expected**: 150 shares (excludes the Apr 1 buy)

**Why**: Critical for dividend calculations - only shares owned ON record date count

**Service logic**:
```python
Transaction.query.filter(
    Transaction.date <= record_date  # ← Key filter
)
```

---

#### Test 5: `test_calculate_shares_with_dividend_transactions`

**Scenario**: Dividend reinvestments should ADD shares

**Test data**:
- Buy 100 shares on Jan 1
- Dividend reinvestment of 5 shares on Feb 1 (type="dividend")
- Calculate shares on Mar 1

**Expected**: 105 shares (100 + 5)

**Bug validated**: This test validates **Bug Fix #1** where dividend transactions were being subtracted instead of added.

**Before fix**: Would get 95 (100 - 5)\
**After fix**: Gets 105 (100 + 5) ✅

See `BUG_FIXES_1.3.3.md` for details.

---

### TestCreateDividendCash (2 tests)

**Focus**: CASH dividends (no reinvestment)

**Context**: CASH dividends are paid as money, not shares:
- No transaction created
- Status is immediately COMPLETED
- No reinvestment tracking needed

---

#### Test 6: `test_create_cash_dividend_auto_completed`

**Scenario**: CASH dividend is auto-completed

**Test data**:
```python
dividend_data = {
    "portfolio_fund_id": portfolio_fund.id,
    "record_date": "2024-02-15",
    "ex_dividend_date": "2024-02-10",
    "dividend_per_share": 0.50,
    # NO reinvestment data
}
```

**Expected**:
- Dividend created ✅
- `reinvestment_status` = COMPLETED ✅
- NO transaction created ✅

**Why**: CASH dividends don't have pending state

---

#### Test 7: `test_create_cash_dividend_no_transaction_created`

**Scenario**: Explicitly verify no transaction

**Same as Test 6, but adds**:
```python
transactions = Transaction.query.filter_by(
    portfolio_fund_id=portfolio_fund.id
).all()
assert len(transactions) == 0  # Verify no transactions
```

**Why**: Redundant verification for critical behavior

---

### TestCreateDividendStock (3 tests)

**Focus**: STOCK dividends (optional reinvestment)

**Context**: STOCK dividends can be:
- Without reinvestment (status=PENDING)
- With reinvestment (creates Transaction, status=COMPLETED)

---

#### Test 8: `test_create_stock_dividend_without_reinvestment`

**Scenario**: Dividend declared, not yet reinvested

**Test data**:
```python
dividend_data = {
    "portfolio_fund_id": portfolio_fund.id,
    "record_date": "2024-02-15",
    "ex_dividend_date": "2024-02-10",
    "dividend_per_share": 0.50,
    # NO reinvestment_shares or reinvestment_price
}
```

**Expected**:
- Dividend created ✅
- `reinvestment_status` = PENDING ✅
- NO transaction ✅

**Why**: User might record dividend first, buy shares later

---

#### Test 9: `test_create_stock_dividend_with_reinvestment`

**Scenario**: Dividend with immediate reinvestment

**Test data**:
```python
dividend_data = {
    # ... basic fields ...
    "reinvestment_shares": 2.5,
    "reinvestment_price": 20.0,
}
```

**Expected**:
- Dividend created ✅
- `reinvestment_status` = COMPLETED ✅
- Transaction created with:
  - `type` = "dividend"
  - `shares` = 2.5
  - `cost_per_share` = 20.0
  - `date` = ex_dividend_date

**Why**: Core STOCK dividend behavior

**Assertions**:
```python
assert dividend.reinvestment_status == ReinvestmentStatus.COMPLETED
assert dividend.reinvestment_transaction_id is not None

txn = db.session.get(Transaction, dividend.reinvestment_transaction_id)
assert txn.type == "dividend"
assert txn.shares == 2.5
assert txn.cost_per_share == 20.0
```

---

#### Test 10: `test_create_stock_dividend_reinvestment_validation`

**Scenario**: Validation rejects invalid reinvestment data

**Test 1 - Negative shares**:
```python
dividend_data = {
    # ... basic fields ...
    "reinvestment_shares": -2.5,  # ❌ Invalid
    "reinvestment_price": 20.0,
}

with pytest.raises(ValueError, match="must be positive"):
    DividendService.create_dividend(dividend_data)
```

**Test 2 - Zero price**:
```python
dividend_data = {
    # ... basic fields ...
    "reinvestment_shares": 2.5,
    "reinvestment_price": 0,  # ❌ Invalid
}

with pytest.raises(ValueError, match="must be positive"):
    DividendService.create_dividend(dividend_data)
```

**Bug validated**: This test validates **Bug Fix #2** where validation was skipped when price=0.

See `BUG_FIXES_1.3.3.md` for details.

---

### TestUpdateDividend (6 tests)

**Focus**: Updating existing dividends

**Complexity**: Updates can add/modify/remove reinvestment, which affects Transactions

---

#### Test 11: `test_update_dividend_basic_fields`

**Scenario**: Simple field updates (no reinvestment changes)

**Test data**:
- Create CASH dividend
- Update `dividend_per_share` from 0.50 → 0.75
- Update dates

**Expected**:
- Fields updated ✅
- `total_amount` recalculated (100 shares × 0.75 = 75.0) ✅

**Why**: Tests basic update path

---

#### Test 12: `test_update_stock_dividend_add_reinvestment`

**Scenario**: Add reinvestment to existing STOCK dividend

**Before**:
- STOCK dividend, PENDING status, no transaction

**Update**:
```python
update_data = {
    # ... dates, dividend_per_share ...
    "reinvestment_shares": 2.5,  # ← NEW
    "reinvestment_price": 20.0,  # ← NEW
}
```

**Expected**:
- Transaction created ✅
- Status PENDING → COMPLETED ✅
- Transaction linked to dividend ✅

**Why**: Tests adding reinvestment after initial creation

---

#### Test 13: `test_update_stock_dividend_modify_reinvestment`

**Scenario**: Modify existing reinvestment

**Before**:
- STOCK dividend WITH reinvestment

**Update**:
```python
update_data = {
    # ... dates, dividend_per_share ...
    "reinvestment_shares": 3.0,  # Changed from 2.5
    "reinvestment_price": 21.0,  # Changed from 20.0
}
```

**Expected**:
- Same transaction ID (not a new one) ✅
- Transaction fields updated ✅

**Why**: Tests update vs create path

**Assertions**:
```python
original_txn_id = dividend.reinvestment_transaction_id

# After update
assert updated_dividend.reinvestment_transaction_id == original_txn_id

txn = db.session.get(Transaction, original_txn_id)
assert txn.shares == 3.0
assert txn.cost_per_share == 21.0
```

---

#### Test 14: `test_update_stock_dividend_remove_reinvestment`

**Scenario**: Remove reinvestment (delete transaction)

**Before**:
- STOCK dividend WITH reinvestment

**Update**:
```python
update_data = {
    # ... dates, dividend_per_share ...
    # NO reinvestment_shares or reinvestment_price
}
```

**Expected**:
- Transaction deleted ✅
- Status COMPLETED → PENDING ✅
- `reinvestment_transaction_id` = None ✅

**Why**: Tests cascade deletion

---

#### Test 15: `test_update_dividend_not_found`

**Scenario**: Error handling for non-existent dividend

**Test**:
```python
with pytest.raises(ValueError, match="Dividend .* not found"):
    DividendService.update_dividend(
        "non-existent-id",
        {"dividend_per_share": 0.50}
    )
```

**Why**: Tests error path

---

#### Test 16: `test_update_dividend_validation_negative_reinvestment`

**Scenario**: Validation in UPDATE (similar to Test 10 for CREATE)

**Test 1 - Negative shares**:
```python
with pytest.raises(ValueError, match="must be positive"):
    DividendService.update_dividend(
        dividend.id,
        {
            # ... required fields ...
            "reinvestment_shares": -2.5,  # ❌
            "reinvestment_price": 20.0,
        }
    )
```

**Test 2 - Zero price**:
```python
with pytest.raises(ValueError, match="must be positive"):
    DividendService.update_dividend(
        dividend.id,
        {
            # ... required fields ...
            "reinvestment_shares": 2.5,
            "reinvestment_price": 0,  # ❌
        }
    )
```

**Bug validated**: Also validates **Bug Fix #2** (update path)

**Why separate from Test 10**: Bug affected both `create_dividend()` AND `update_dividend()`

---

### TestDeleteDividend (2 tests)

**Focus**: Deletion and cascade behavior

---

#### Test 17: `test_delete_cash_dividend`

**Scenario**: Delete CASH dividend (simple case)

**Expected**: Dividend removed from database

**Test**:
```python
DividendService.delete_dividend(dividend_id)

deleted = db.session.get(Dividend, dividend_id)
assert deleted is None
```

---

#### Test 18: `test_delete_stock_dividend_with_transaction`

**Scenario**: Delete STOCK dividend WITH transaction

**Expected**:
- Dividend deleted ✅
- Transaction also deleted (cascade) ✅

**Test**:
```python
DividendService.delete_dividend(dividend_id)

assert db.session.get(Dividend, dividend_id) is None
assert db.session.get(Transaction, transaction_id) is None
```

**Why**: Tests cascade deletion

---

### TestEdgeCases (3 tests)

---

#### Test 19: `test_create_dividend_invalid_portfolio_fund`

**Scenario**: Error when portfolio-fund doesn't exist

**Test**:
```python
dividend_data = {
    "portfolio_fund_id": "non-existent-id",  # ❌
    # ... other fields ...
}

with pytest.raises(ValueError):
    DividendService.create_dividend(dividend_data)
```

---

#### Test 20: `test_calculate_shares_zero_shares`

**Scenario**: Portfolio with no transactions

**Expected**: 0 shares (not error)

**Test**:
```python
# Create portfolio_fund but NO transactions
shares = DividendService.calculate_shares_owned(
    portfolio_fund.id,
    date(2024, 3, 1)
)

assert shares == 0
```

**Why**: Edge case - empty portfolio

---

#### Test 21: `test_create_dividend_calculates_correct_total_amount`

**Scenario**: Service auto-calculates total_amount

**Formula**: `total_amount = shares_owned × dividend_per_share`

**Test**:
```python
dividend_data = {
    # ... portfolio with 100 shares ...
    "dividend_per_share": 0.50,
}

dividend = DividendService.create_dividend(dividend_data)

assert dividend.shares_owned == 100
assert dividend.total_amount == 50.0  # 100 × 0.50
```

**Why**: Tests automatic calculation

---

## Coverage Analysis

**Total**: 91% coverage (119 statements, 11 missed)

**Uncovered lines** (11 statements):
- `160-162`: Exception handler in `create_dividend()` for database errors
- `221`: Validation error in `update_dividend()`
- `278, 294, 314`: Error paths in other methods
- `356, 364-367`: Exception handlers in `delete_dividend()`

**Why uncovered**: Exception handlers require complex mocking (database failures)

**Why acceptable**: 91% exceeds 90% target, all business logic covered

---

## Running the Tests

### All dividend tests:
```bash
cd backend
source .venv/bin/activate
pytest tests/test_dividend_service.py -v
```

### Specific test class:
```bash
pytest tests/test_dividend_service.py::TestCalculateSharesOwned -v
```

### Specific test:
```bash
pytest tests/test_dividend_service.py::TestCalculateSharesOwned::test_calculate_shares_buy_only -xvs
```

### With coverage:
```bash
pytest tests/test_dividend_service.py --cov=app.services.dividend_service --cov-report=term-missing
```

---

## Related Documentation

- **Testing Infrastructure**: `TESTING_INFRASTRUCTURE.md` - General testing setup
- **Bug Fixes**: `BUG_FIXES_1.3.3.md` - Bugs found during test development
- **Test Index**: `README.md` - All test documentation
- **Service Code**: `app/services/dividend_service.py`
- **Test Code**: `tests/test_dividend_service.py`
- **Materialized View Invalidation Tests**: `DIVIDEND_MATERIALIZED_VIEW_INVALIDATION_TESTS.md`

---

**Document Version**: 1.5.1
**Last Updated**: 2026-02-06
