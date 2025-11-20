# IBKR Routes Integration Tests

**File**: `tests/routes/test_ibkr_routes.py`
**Route File**: `app/routes/ibkr_routes.py`
**Test Count**: 20 tests (19 passing, 1 skipped)
**Status**: ✅ Core functionality tested (Phase 2b complete)

---

## Overview

Integration tests for Interactive Brokers (IBKR) transaction processing API endpoints. These tests verify IBKR transaction inbox management, allocation processing, dividend matching, and bulk operations.

### Endpoints Tested

1. **POST /api/ibkr/flex-query** - Import IBKR Flex Query (SKIPPED - external API) ⏭️
2. **GET /api/ibkr/inbox** - List inbox transactions ✅
3. **GET /api/ibkr/inbox/<transaction_id>** - Get transaction detail ✅
4. **POST /api/ibkr/inbox/<transaction_id>/ignore** - Ignore transaction ✅
5. **DELETE /api/ibkr/inbox/<transaction_id>** - Delete transaction ✅
6. **POST /api/ibkr/inbox/<transaction_id>/allocate** - Allocate transaction ✅ (Phase 2a)
7. **GET /api/ibkr/inbox/<transaction_id>/allocations** - Get allocations ✅ (Phase 2b)
8. **PUT /api/ibkr/inbox/<transaction_id>/allocations** - Update allocations ✅ (Phase 2a)
9. **POST /api/ibkr/inbox/<transaction_id>/unallocate** - Unallocate transaction ✅ (Phase 2b)
10. **GET /api/ibkr/dividends/pending** - Get pending dividends ✅
11. **POST /api/ibkr/inbox/<transaction_id>/match-dividend** - Match dividend ✅ (Phase 2a)
12. **POST /api/ibkr/inbox/bulk-allocate** - Bulk allocate transactions ✅ (Phase 2a)
13. **GET /api/ibkr/config** - Get IBKR config ✅ (Phase 1)
14. **POST /api/ibkr/config** - Save IBKR config ✅ (Phase 1)

### Test Status Summary

- **19 passing** - All core functionality tested
- **1 skipped** - External IBKR Flex API integration (requires mocking)

---

## Recent Changes

### Phase 2b - Service Layer Refactoring (2 tests enabled)

**Service Methods Added:**
- `IBKRConfigService.get_first_config()` - Get IBKR configuration
- `IBKRTransactionService.get_inbox()` - Get inbox transactions with filters
- `IBKRTransactionService.get_inbox_count()` - Count transactions by status
- `IBKRTransactionService.unallocate_transaction()` - Remove transaction allocations
- `IBKRTransactionService.get_transaction_allocations()` - Get allocation details
- `PortfolioService.get_active_portfolios()` - Get non-archived portfolios

**Routes Refactored:**
All IBKR routes now properly delegate business logic to service layer:
- Removed direct database access (Query.query, Query.filter_by, db.session.get)
- Routes now act as thin controllers, delegating to services
- Consistent with Phase 1 & 2a patterns

**Tests Fixed:**
1. **test_get_transaction_allocations** - Previously skipped for 500 error
   - Issue: Route had business logic, test expected wrong response format
   - Fix: Moved logic to service, fixed test to expect dict not list
   - Status: ✅ Passing

2. **test_unallocate_transaction** - Previously skipped for 500 error
   - Issue: Route had business logic, test used wrong transaction status
   - Fix: Moved logic to service, fixed test to use "processed" status
   - Added handling for orphaned allocations (allocations without transactions)
   - Status: ✅ Passing

### Phase 2a - Validation & Payload Fixes (4 tests enabled)

These tests were failing due to incorrect request payload formats. All have been fixed to match the actual API requirements:

1. **test_allocate_transaction** - Fixed allocation payload format
   - Issue: Payload used `portfolio_fund_id` and `shares`
   - Fix: Changed to `allocations` array with `portfolio_id` and `percentage`
   - Status: ✅ Passing

2. **test_match_dividend** - Fixed dividend matching payload format
   - Issue: Payload used singular `dividend_id`
   - Fix: Changed to `dividend_ids` array, added `isin` to IBKRTransaction
   - Status: ✅ Passing

3. **test_update_transaction_allocations** - Fixed modification payload format
   - Issue: Payload used `portfolio_fund_id` and `shares`
   - Fix: Changed to `allocations` array, properly allocates transaction first
   - Status: ✅ Passing

4. **test_bulk_allocate** - Fixed bulk allocation payload format
   - Issue: Payload missing `allocations` array
   - Fix: Added both `transaction_ids` and `allocations` to payload
   - Status: ✅ Passing

---

## Test Organization

### Test Classes

1. **TestIBKRFlexImport** (2 tests, 1 skipped)
   - Import Flex Query (SKIPPED - external API)
   - List inbox transactions

2. **TestIBKRInbox** (3 tests)
   - Get inbox transaction detail
   - Ignore transaction
   - Delete transaction

3. **TestIBKRAllocation** (10 tests, all passing)
   - Allocate transaction (100% single portfolio)
   - Get transaction allocations
   - Update transaction allocations (60/40 split)
   - Unallocate transaction
   - Get pending dividends
   - Match dividend to existing records

4. **TestIBKRBulkOperations** (1 test)
   - Bulk allocate multiple transactions

5. **TestIBKRConfig** (2 tests, all passing)
   - Get IBKR configuration status
   - Save IBKR configuration

---

## Helper Functions

### `create_fund()`
```python
def create_fund(isin_prefix="US", symbol_prefix="TEST", name="Test Fund",
                currency="USD", exchange="NYSE"):
    """Helper to create a Fund with all required fields."""
    return Fund(
        isin=make_isin(isin_prefix),
        symbol=make_symbol(symbol_prefix),
        name=name,
        currency=currency,
        exchange=exchange,
    )
```

**Consistency**: Same helper used across all route tests for uniformity.

---

## Key Test Patterns

### 1. Testing Transaction Allocation

```python
def test_allocate_transaction(self, app_context, client, db_session):
    """Test POST /ibkr/inbox/<transaction_id>/allocate."""
    # Create fund and portfolio
    fund = create_fund("US", "AAPL", "Apple Inc")
    portfolio = Portfolio(name="Test Portfolio")
    db_session.add_all([fund, portfolio])
    db_session.commit()

    # Create IBKR transaction
    txn = IBKRTransaction(
        ibkr_transaction_id=make_id(),
        transaction_date=datetime.now().date(),
        symbol=fund.symbol,
        isin=fund.isin,  # Required for fund matching
        description="Apple Inc",
        transaction_type="buy",
        quantity=10,
        price=150.00,
        total_amount=1500.00,
        currency="USD",
        status="pending",
    )
    db_session.add(txn)
    db_session.commit()

    # Allocate 100% to portfolio
    payload = {
        "allocations": [
            {
                "portfolio_id": portfolio.id,
                "percentage": 100.0
            }
        ]
    }

    response = client.post(f"/api/ibkr/inbox/{txn.id}/allocate", json=payload)

    assert response.status_code == 200
    data = response.get_json()
    assert data["success"] is True
    assert "created_transactions" in data
```

**Key Points**:
- Payload format: `{"allocations": [{"portfolio_id": str, "percentage": float}]}`
- Allocations must sum to 100%
- IBKRTransaction requires `isin` field for fund matching
- Service layer creates Transaction records and IBKRTransactionAllocation records

### 2. Testing Dividend Matching

```python
def test_match_dividend(self, app_context, client, db_session):
    """Test POST /ibkr/inbox/<transaction_id>/match-dividend."""
    # Create fund, portfolio, and dividend
    fund = create_fund("US", "AAPL", "Apple Inc")
    portfolio = Portfolio(name="Test Portfolio")
    db_session.add_all([fund, portfolio])
    db_session.commit()

    pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
    db_session.add(pf)
    db_session.commit()

    dividend = Dividend(
        fund_id=fund.id,
        portfolio_fund_id=pf.id,
        record_date=datetime.now().date(),
        ex_dividend_date=datetime.now().date() - timedelta(days=1),
        shares_owned=100,
        dividend_per_share=Decimal("0.50"),
        total_amount=Decimal("0"),  # Will be set by matching
        reinvestment_status=ReinvestmentStatus.PENDING,
    )
    db_session.add(dividend)
    db_session.commit()

    # Create IBKR dividend transaction
    txn = IBKRTransaction(
        ibkr_transaction_id=make_id(),
        transaction_date=datetime.now().date(),
        symbol=fund.symbol,
        isin=fund.isin,
        description="Apple Inc - Dividend",
        transaction_type="dividend",
        total_amount=50.00,
        currency="USD",
        status="pending",
    )
    db_session.add(txn)
    db_session.commit()

    # Match dividend
    payload = {"dividend_ids": [dividend.id]}

    response = client.post(f"/api/ibkr/inbox/{txn.id}/match-dividend", json=payload)

    assert response.status_code == 200
    data = response.get_json()
    assert data["success"] is True
    assert data["updated_dividends"] == 1
```

**Key Points**:
- Payload format: `{"dividend_ids": [str, ...]}`
- Dividend `total_amount` starts at 0, set by matching
- IBKRTransaction type must be "dividend"
- Service allocates amount across multiple dividends by shares owned

### 3. Testing Allocation Modification

```python
def test_update_transaction_allocations(self, app_context, client, db_session):
    """Test PUT /ibkr/inbox/<transaction_id>/allocations."""
    # Create 2 portfolios
    fund = create_fund("US", "AAPL", "Apple Inc")
    portfolio1 = Portfolio(name="Portfolio 1")
    portfolio2 = Portfolio(name="Portfolio 2")
    db_session.add_all([fund, portfolio1, portfolio2])
    db_session.commit()

    # Create and allocate transaction
    txn = IBKRTransaction(...)
    db_session.add(txn)
    db_session.commit()

    # First allocate 100% to portfolio1
    from app.services.ibkr_transaction_service import IBKRTransactionService
    IBKRTransactionService.process_transaction_allocation(
        txn.id,
        [{"portfolio_id": portfolio1.id, "percentage": 100.0}]
    )

    # Modify to 60/40 split
    payload = {
        "allocations": [
            {"portfolio_id": portfolio1.id, "percentage": 60.0},
            {"portfolio_id": portfolio2.id, "percentage": 40.0}
        ]
    }

    response = client.put(f"/api/ibkr/inbox/{txn.id}/allocations", json=payload)

    assert response.status_code == 200
    data = response.get_json()
    assert data["success"] is True
```

**Key Points**:
- Transaction must be processed (status="processed") before modification
- Service layer updates existing Transaction records and allocations
- Allocations must still sum to 100%
- Can add/remove portfolios or change percentages

### 4. Testing Bulk Operations

```python
def test_bulk_allocate(self, app_context, client, db_session):
    """Test POST /ibkr/inbox/bulk-allocate."""
    # Create fund and portfolio
    fund = create_fund("US", "AAPL", "Apple Inc")
    portfolio = Portfolio(name="Test Portfolio")
    db_session.add_all([fund, portfolio])
    db_session.commit()

    # Create 2 IBKR transactions
    txn1 = IBKRTransaction(...)
    txn2 = IBKRTransaction(...)
    db_session.add_all([txn1, txn2])
    db_session.commit()

    # Bulk allocate both to same portfolio
    payload = {
        "transaction_ids": [txn1.id, txn2.id],
        "allocations": [
            {"portfolio_id": portfolio.id, "percentage": 100.0}
        ]
    }

    response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

    assert response.status_code == 200
    data = response.get_json()
    assert data.get("success") is True or "results" in data
```

**Key Points**:
- Payload includes both `transaction_ids` array and `allocations` array
- All transactions allocated with same percentages
- Efficient for processing multiple similar transactions

---

## Mocking Strategy

### External IBKR Flex API

The IBKR Flex Query import endpoint requires external API calls and is skipped in integration tests:

```python
@pytest.mark.skip(
    reason="Endpoint requires external IBKR Flex API calls. "
    "Testing requires mocking complex external API interactions."
)
def test_import_flex_query(self, app_context, client):
    """Test POST /ibkr/flex-query imports from IBKR."""
    # This would require mocking:
    # - IBKR authentication
    # - Flex Query execution
    # - XML response parsing
    # Complex external API - tested at service layer instead
```

**Rationale**: External API complexity is better tested at the service layer with controlled mocks.

---

## Skipped Tests Analysis

### Legitimate Skips (1 test)

1. **test_import_flex_query** - Requires external IBKR Flex API
   - **Reason**: Complex external API interactions
   - **Coverage**: Tested at service layer (`test_ibkr_flex_service.py`)
   - **Action**: No fix needed

### 500-Error Issues (4 tests) - Pending Investigation

These tests return 500 errors and require business logic investigation:

1. **test_get_allocations** (line 316) - GET /api/ibkr/inbox/<transaction_id>/allocations
   - Returns 500 error
   - May require specific transaction state

2. **test_unallocate_transaction** (line 486) - POST /api/ibkr/inbox/<transaction_id>/unallocate
   - Returns 500 error
   - May require specific allocation state

3. **test_get_config** (line 537) - GET /api/ibkr/config
   - Returns 500 error
   - Config storage mechanism may need initialization

4. **test_save_config** (SKIPPED) - POST /api/ibkr/config
   - Returns 500 error
   - Related to config storage

---

## Database Verification Pattern

All CRUD tests verify database state after API calls:

```python
# After allocation
allocations = IBKRTransactionAllocation.query.filter_by(
    ibkr_transaction_id=txn.id
).all()
assert len(allocations) >= 1

# After dividend matching
updated_div = db.session.get(Dividend, dividend.id)
assert updated_div.total_amount == 50.00

# After transaction marking
db_session.refresh(txn)
assert txn.status == "processed"
```

**Purpose**: Ensures API changes persist correctly to database.

---

## Error Handling Tests

### Transaction Validation

```python
def test_ignore_transaction_already_processed(self, app_context, client, db_session):
    """Test ignoring already-processed transaction fails."""
    txn = create_transaction(status="processed")

    response = client.post(f"/api/ibkr/inbox/{txn.id}/ignore")

    assert response.status_code == 400
    assert "error" in response.get_json()
```

Tests verify:
- Cannot ignore/delete processed transactions
- Cannot match non-dividend transactions as dividends
- Allocations must sum to 100%
- Transaction must exist (404 for invalid IDs)

---

## Service Layer Integration

IBKR routes delegate to `IBKRTransactionService` for business logic:

- `process_transaction_allocation()` - Create transactions and allocations
- `modify_allocations()` - Update existing allocations
- `match_dividend()` - Link IBKR dividends to existing Dividend records
- `get_pending_dividends()` - Query pending dividends for matching
- `validate_allocations()` - Ensure allocations sum to 100%

**Pattern**: Routes handle HTTP/validation, service handles business logic.

---

## Test Statistics

### By Test Class

| Class | Total | Passing | Skipped |
|-------|-------|---------|---------|
| TestIBKRFlexImport | 2 | 1 | 1 |
| TestIBKRInbox | 3 | 3 | 0 |
| TestIBKRAllocation | 10 | 8 | 2 |
| TestIBKRBulkOperations | 1 | 1 | 0 |
| TestIBKRConfig | 2 | 0 | 2 |
| **Total** | **18** | **13** | **5** |

### By Skip Reason

| Reason | Count | Action Needed |
|--------|-------|---------------|
| External API | 1 | None - tested at service layer |
| 500 Errors | 4 | Investigation required |

---

## Related Documentation

- **Service Tests**: `tests/docs/services/IBKR_TRANSACTION_SERVICE_TESTS.md`
- **Service Tests**: `tests/docs/services/IBKR_FLEX_SERVICE_TESTS.md`
- **Service Tests**: `tests/docs/services/IBKR_CONFIG_SERVICE_TESTS.md`
- **Remediation Plan**: `todo/ROUTE_REFACTORING_REMEDIATION_PLAN.md`

---

**Last Updated**: 2025-01-18 (Phase 2a - Fixed 4 validation tests)
**Status**: 15/20 tests passing (75% coverage)
**Next Steps**: Investigate 4 remaining 500-error tests
