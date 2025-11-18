# Transaction Routes Integration Tests

**File**: `tests/routes/test_transaction_routes.py`
**Route File**: `app/routes/transaction_routes.py`
**Test Count**: 12 tests
**Status**: ✅ All tests passing

---

## Overview

Integration tests for all transaction management API endpoints. These tests verify transaction CRUD operations, filtering, and proper interaction with the transaction service layer.

### Endpoints Tested

1. **GET /api/transactions** - List all transactions (with optional filtering)
2. **POST /api/transactions** - Create transaction
3. **GET /api/transactions/<id>** - Get transaction detail
4. **PUT /api/transactions/<id>** - Update transaction
5. **DELETE /api/transactions/<id>** - Delete transaction

### Transaction Types Supported
- **buy** - Purchase of shares
- **sell** - Sale of shares (triggers realized gain/loss calculation)
- **dividend** - Dividend reinvestment
- **fee** - Transaction fees/commissions

---

## Test Organization

### Test Classes

1. **TestTransactionList** (3 tests)
   - List transactions when empty
   - List all transactions
   - Filter transactions by portfolio_id

2. **TestTransactionCreate** (3 tests)
   - Create buy transaction
   - Create sell transaction (with prior buy for shares)
   - Create dividend transaction

3. **TestTransactionRetrieveUpdateDelete** (6 tests)
   - Get transaction detail
   - Get non-existent transaction (404)
   - Update transaction
   - Update non-existent transaction (404)
   - Delete transaction
   - Delete non-existent transaction (400)

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

### 1. Testing Sell Transactions

Sell transactions require prior buy transactions to establish share ownership:

```python
def test_create_sell_transaction(self, app_context, client, db_session):
    # ... create portfolio and fund ...

    # First buy some shares
    buy_txn = Transaction(
        portfolio_fund_id=pf.id,
        date=datetime.now().date() - timedelta(days=5),
        type="buy",
        shares=20,
        cost_per_share=Decimal("200.00"),
    )
    db_session.add(buy_txn)
    db_session.commit()

    # Now sell some shares
    payload = {
        "portfolio_fund_id": pf.id,
        "date": datetime.now().date().isoformat(),
        "type": "sell",
        "shares": 10,
        "cost_per_share": 250.00,
    }

    response = client.post("/api/transactions", json=payload)
    assert response.status_code == 200
```

**Why**: Cannot sell shares that don't exist. Service layer validates share ownership.

### 2. Testing Transaction Filtering

```python
def test_list_transactions_filtered_by_portfolio(self, app_context, client, db_session):
    # Create two portfolios with different transactions
    p1 = Portfolio(name="Portfolio 1")
    p2 = Portfolio(name="Portfolio 2")
    # ... create portfolio funds and transactions ...

    response = client.get(f"/api/transactions?portfolio_id={p1.id}")

    assert response.status_code == 200
    # Should only include transactions for portfolio 1
```

**Why**: Users need to filter transactions by portfolio for focused analysis.

### 3. Testing Transaction Updates

```python
def test_update_transaction(self, app_context, client, db_session):
    # ... create transaction ...

    payload = {
        "portfolio_fund_id": pf.id,
        "date": datetime.now().date().isoformat(),
        "type": "buy",
        "shares": 35,  # Changed from 30
        "cost_per_share": 47.00,  # Changed from 45.00
    }

    response = client.put(f"/api/transactions/{txn.id}", json=payload)

    assert response.status_code == 200
    data = response.get_json()
    assert data["shares"] == 35

    # Verify database
    db_session.refresh(txn)
    assert txn.shares == 35
```

**Pattern**: Always verify both HTTP response and database state after mutations.

---

## Transaction Data Structure

### Request Payload Format
```json
{
    "portfolio_fund_id": "uuid-string",
    "date": "2024-01-15",
    "type": "buy",
    "shares": 10.5,
    "cost_per_share": 125.75
}
```

### Response Format
```json
{
    "id": "uuid-string",
    "portfolio_fund_id": "uuid-string",
    "date": "2024-01-15",
    "type": "buy",
    "shares": 10.5,
    "cost_per_share": 125.75,
    "created_at": "2024-01-15T10:30:00"
}
```

**Note**: Sell transactions may include additional `realized_gain_loss` field.

---

## Error Handling

### 404 vs 400 Errors

Different endpoints return different error codes:

```python
# GET non-existent transaction
response = client.get(f"/api/transactions/{fake_id}")
assert response.status_code == 404  # Not Found

# DELETE non-existent transaction
response = client.delete(f"/api/transactions/{fake_id}")
assert response.status_code == 400  # Bad Request
```

**Why**: The API implementation uses different error handling patterns. Integration tests document the actual behavior rather than enforcing consistency.

---

## Transaction Business Logic

### Realized Gain/Loss Calculation

When a sell transaction is created, the service layer automatically:
1. Calculates average cost basis from prior buy transactions
2. Computes realized gain/loss: `(sell_price - avg_cost) * shares_sold`
3. Creates `RealizedGainLoss` record
4. Includes gain/loss in transaction response

**Tests verify**: Sell transactions return successfully and are created in database. Specific gain/loss calculations are tested at the service layer.

### Share Tracking

The service layer maintains accurate share counts by:
1. Adding shares on buy transactions
2. Subtracting shares on sell transactions
3. Preventing overselling (selling more shares than owned)

**Tests verify**: Transactions can be created with valid share counts. Validation logic is tested at the service layer.

---

## Running Tests

### Run all transaction route tests:
```bash
pytest tests/routes/test_transaction_routes.py -v
```

### Run specific test class:
```bash
pytest tests/routes/test_transaction_routes.py::TestTransactionCreate -v
```

### Run with timing:
```bash
pytest tests/routes/test_transaction_routes.py -v --durations=10
```

### Run without coverage (faster):
```bash
pytest tests/routes/test_transaction_routes.py -v --no-cov
```

---

## Test Results

**All 12 tests passing** ✅

### Test Execution Time
- **Average**: ~0.29 seconds for full suite
- **Fastest test**: ~0.02 seconds (empty list check)
- **Slowest test**: ~0.05 seconds (sell transaction creation)

### Coverage
Integration tests verify **all 5 transaction endpoints** work correctly with various transaction types and edge cases.

---

## Common Patterns

### Date Handling
```python
# Always use .isoformat() for dates in payloads
"date": datetime.now().date().isoformat()  # "2024-01-15"

# Transactions in the past
"date": (datetime.now().date() - timedelta(days=5)).isoformat()
```

### Decimal Precision
```python
# Use Decimal for database (model)
Transaction(
    shares=10,
    cost_per_share=Decimal("100.00")
)

# Use float for API payloads (JSON)
payload = {
    "shares": 10,
    "cost_per_share": 100.00
}
```

---

## Related Documentation

- **Service Tests**: `tests/docs/services/TRANSACTION_SERVICE_TESTS.md`
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/transaction_routes.py`
- **Transaction Service**: `app/services/transaction_service.py`

---

**Last Updated**: Phase 5 (Route Integration Tests)
**Maintainer**: See git history
