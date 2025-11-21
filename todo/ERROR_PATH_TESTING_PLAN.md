# Error Path Testing Plan - Route Coverage to 90%

**Created**: 2025-11-20
**Goal**: Achieve 90%+ coverage on all route files by systematically testing error paths
**Current Status**: 84% overall (Services: 90%+, Routes: 49-89%)

---

## Coverage Gaps by Priority

### Priority 1: developer_routes.py (49% → 90%)
**Missing**: 110 lines | **Current**: 49% | **Target**: 90%

#### Areas to Cover:

1. **Exchange Rate Endpoints**
   - ❌ Missing required fields (from_currency, to_currency, rate)
   - ❌ Invalid currency codes (not in valid_currencies set)
   - ❌ Invalid rate values (negative, zero, non-numeric)
   - ❌ Invalid date format
   - ❌ Database errors (mock db.session.commit to raise exception)
   - ❌ General exception handling

2. **Fund Price Endpoints**
   - ❌ Missing required fields (fund_id, price)
   - ❌ Invalid fund_id (fund not found)
   - ❌ Invalid price values (negative, zero, non-numeric)
   - ❌ Invalid date format
   - ❌ Database errors
   - ❌ Fund price not found (404 case)

3. **CSV Import Endpoints**
   - ❌ No file provided
   - ❌ Invalid file format (not CSV)
   - ❌ Invalid CSV headers
   - ❌ Wrong file type (transaction file vs price file)
   - ❌ Invalid portfolio_fund_id/fund_id
   - ❌ CSV parsing errors (ValueError)
   - ❌ Database errors during import

4. **Logging Endpoints**
   - ❌ Invalid logging level values
   - ❌ Invalid filter parameters
   - ❌ Database errors retrieving settings
   - ❌ Database errors updating settings
   - ❌ Database errors retrieving logs
   - ❌ Database errors clearing logs

**Test Strategy**:
```python
# Example: Test missing required field
def test_set_exchange_rate_missing_from_currency(client):
    payload = {"to_currency": "EUR", "rate": 0.85}  # Missing from_currency
    response = client.post("/api/exchange-rate", json=payload)
    assert response.status_code == 400
    assert "Missing required field: from_currency" in response.get_json()["message"]

# Example: Test database error
def test_set_exchange_rate_database_error(client, monkeypatch):
    def mock_commit():
        raise Exception("Database error")
    monkeypatch.setattr("app.routes.developer_routes.db.session.commit", mock_commit)

    payload = {"from_currency": "USD", "to_currency": "EUR", "rate": 0.85}
    response = client.post("/api/exchange-rate", json=payload)
    assert response.status_code == 500
```

---

### Priority 2: ibkr_routes.py (64% → 90%)
**Missing**: 78 lines | **Current**: 64% | **Target**: 90%

#### Areas to Cover:

1. **Flex Import Endpoint** (currently skipped)
   - ❌ Missing API key
   - ❌ Invalid API key format
   - ❌ External API failure (mock)
   - ❌ Invalid flex query response
   - ❌ Database errors during import

2. **Inbox Endpoints**
   - ❌ Invalid transaction_id (not found)
   - ❌ Invalid status parameter values
   - ❌ Invalid transaction_type values
   - ❌ Database errors

3. **Allocation Endpoints**
   - ❌ Missing required fields (allocations, portfolio_id, percentage)
   - ❌ Invalid portfolio_id
   - ❌ Invalid percentage values (negative, >100, non-numeric)
   - ❌ Allocation percentage doesn't sum to 100
   - ❌ Transaction not in pending status
   - ❌ Transaction already allocated
   - ❌ Database errors

4. **Dividend Matching**
   - ❌ Missing required fields (dividend_ids, isin)
   - ❌ Invalid dividend_id
   - ❌ Dividend already matched
   - ❌ ISIN mismatch
   - ❌ Database errors

5. **Bulk Operations**
   - ❌ Missing transaction_ids array
   - ❌ Empty transaction_ids array
   - ❌ Some transactions not found
   - ❌ Some transactions not in pending status
   - ❌ Partial failure scenarios
   - ❌ Database errors

6. **Config Endpoints**
   - ❌ Missing required fields
   - ❌ Invalid API key format
   - ❌ Database errors saving config

**Test Strategy**:
```python
# Example: Test allocation percentage validation
def test_allocate_transaction_invalid_percentage(client, db_session):
    txn = create_ibkr_transaction(status="pending")
    portfolio = create_portfolio()

    payload = {
        "allocations": [
            {"portfolio_id": portfolio.id, "percentage": 150}  # Invalid >100
        ]
    }
    response = client.post(f"/api/ibkr/inbox/{txn.id}/allocate", json=payload)
    assert response.status_code == 400

# Example: Test transaction not found
def test_allocate_transaction_not_found(client):
    response = client.post("/api/ibkr/inbox/invalid-id/allocate", json={
        "allocations": [{"portfolio_id": "id", "percentage": 100}]
    })
    assert response.status_code == 404
```

---

### Priority 3: fund_routes.py (74% → 90%)
**Missing**: 42 lines | **Current**: 74% | **Target**: 90%

#### Areas to Cover:

1. **Fund CRUD Endpoints**
   - ❌ Missing required fields (isin, symbol, name, currency, exchange)
   - ❌ Duplicate ISIN (unique constraint)
   - ❌ Invalid currency code
   - ❌ Invalid exchange code
   - ❌ Fund not found (404)
   - ❌ Fund in use (cannot delete)
   - ❌ Database errors

2. **Symbol Lookup**
   - ❌ Symbol not found
   - ❌ External API failure (mock)
   - ❌ Invalid symbol format
   - ❌ Cache errors

3. **Price Update Endpoints**
   - ❌ Fund not found
   - ❌ External API failure (mock)
   - ❌ Invalid price data from API
   - ❌ Database errors
   - ❌ Authentication failure (missing API key)
   - ❌ Rate limiting

**Test Strategy**:
```python
# Example: Test duplicate ISIN
def test_create_fund_duplicate_isin(client, db_session):
    fund = create_fund(isin="US1234567890")
    db_session.add(fund)
    db_session.commit()

    payload = {
        "isin": "US1234567890",  # Duplicate
        "symbol": "NEW",
        "name": "New Fund",
        "currency": "USD",
        "exchange": "NYSE"
    }
    response = client.post("/api/funds", json=payload)
    assert response.status_code == 400
    assert "already exists" in response.get_json()["message"]
```

---

### Priority 4: dividend_routes.py (78% → 90%)
**Missing**: 14 lines | **Current**: 78% | **Target**: 90%

#### Areas to Cover:

1. **Dividend Creation**
   - ❌ Missing required fields (fund_id, ex_dividend_date, amount)
   - ❌ Invalid fund_id (fund not found)
   - ❌ Invalid date format
   - ❌ Invalid amount (negative, zero)
   - ❌ Database errors

2. **Dividend Retrieval**
   - ❌ Fund not found
   - ❌ Portfolio not found
   - ❌ No dividends found (empty list)
   - ❌ Database errors

3. **Dividend Update/Delete**
   - ❌ Dividend not found (404)
   - ❌ Invalid update data
   - ❌ Database errors

**Test Strategy**:
```python
# Example: Test invalid amount
def test_create_dividend_negative_amount(client, db_session):
    fund = create_fund()
    db_session.add(fund)
    db_session.commit()

    payload = {
        "fund_id": fund.id,
        "ex_dividend_date": "2024-01-01",
        "amount": -5.00  # Negative
    }
    response = client.post("/api/dividends", json=payload)
    assert response.status_code == 400
```

---

### Priority 5: transaction_routes.py (81% → 90%)
**Missing**: 15 lines | **Current**: 81% | **Target**: 90%

#### Areas to Cover:

1. **Transaction Creation**
   - ❌ Missing required fields (portfolio_fund_id, date, type, shares, cost_per_share)
   - ❌ Invalid portfolio_fund_id (not found)
   - ❌ Invalid transaction type (not buy/sell/dividend)
   - ❌ Invalid shares (negative, zero)
   - ❌ Invalid cost_per_share (negative)
   - ❌ Invalid date format
   - ❌ Database errors

2. **Transaction Update/Delete**
   - ❌ Transaction not found (404)
   - ❌ Invalid update data
   - ❌ Database errors

**Test Strategy**:
```python
# Example: Test invalid transaction type
def test_create_transaction_invalid_type(client, db_session):
    pf = create_portfolio_fund()

    payload = {
        "portfolio_fund_id": pf.id,
        "date": "2024-01-01",
        "type": "invalid",  # Not buy/sell/dividend
        "shares": 10,
        "cost_per_share": 100
    }
    response = client.post("/api/transactions", json=payload)
    assert response.status_code == 400
```

---

### Priority 6: portfolio_routes.py (89% → 90%)
**Missing**: 13 lines | **Current**: 89% | **Target**: 90%

#### Areas to Cover:

1. **Portfolio CRUD**
   - ❌ Missing required field (name)
   - ❌ Database errors
   - ❌ Portfolio not found (404)

2. **Portfolio-Fund Relationships**
   - ❌ Missing required fields (portfolio_id, fund_id)
   - ❌ Invalid portfolio_id or fund_id
   - ❌ Duplicate relationship
   - ❌ Database errors

**Test Strategy**:
```python
# Example: Test missing name
def test_create_portfolio_missing_name(client):
    payload = {"description": "Test"}  # Missing name
    response = client.post("/api/portfolios", json=payload)
    assert response.status_code == 400
```

---

## Service Error Path Coverage

### Services Below 90%:

1. **ibkr_transaction_service.py (87% → 90%)**
   - ❌ Edge cases in allocation logic
   - ❌ Orphaned allocation scenarios
   - ❌ Boundary conditions in percentage calculations
   - ❌ Transaction state validation failures

2. **ibkr_flex_service.py (77% → 90%)**
   - ❌ XML parsing errors
   - ❌ Invalid flex query responses
   - ❌ Missing required fields in response
   - ❌ Date parsing errors
   - ❌ Currency conversion errors

---

## Testing Implementation Strategy

### Phase 1: Developer Routes (Highest Impact)
**Estimated Time**: 3-4 hours
**Tests to Add**: ~30 error path tests

1. Create `TestExchangeRateErrors` class
2. Create `TestFundPriceErrors` class
3. Create `TestCSVImportErrors` class
4. Create `TestLoggingErrors` class

### Phase 2: IBKR Routes
**Estimated Time**: 4-5 hours
**Tests to Add**: ~40 error path tests

1. Create error test classes for each endpoint group
2. Mock external IBKR API failures
3. Test allocation validation thoroughly

### Phase 3: Fund Routes
**Estimated Time**: 2-3 hours
**Tests to Add**: ~20 error path tests

1. Add validation error tests
2. Mock symbol lookup failures
3. Test price update errors

### Phase 4: Remaining Routes
**Estimated Time**: 2-3 hours
**Tests to Add**: ~15 error path tests

1. Add error tests for dividend routes
2. Add error tests for transaction routes
3. Add error tests for portfolio routes

### Phase 5: Service Error Paths
**Estimated Time**: 2-3 hours
**Tests to Add**: ~10-15 tests

1. Focus on ibkr_transaction_service
2. Focus on ibkr_flex_service

---

## Testing Patterns to Use

### Pattern 1: Missing Required Fields
```python
def test_endpoint_missing_field(client):
    """Test endpoint rejects payload with missing required field."""
    payload = {"field2": "value"}  # Missing field1
    response = client.post("/api/endpoint", json=payload)

    assert response.status_code == 400
    data = response.get_json()
    assert "Missing required field" in data["message"]
```

### Pattern 2: Invalid Field Values
```python
def test_endpoint_invalid_value(client):
    """Test endpoint validates field values."""
    payload = {"field": -1}  # Invalid negative value
    response = client.post("/api/endpoint", json=payload)

    assert response.status_code == 400
    data = response.get_json()
    assert "Invalid" in data["message"]
```

### Pattern 3: Not Found (404)
```python
def test_endpoint_not_found(client):
    """Test endpoint returns 404 for non-existent resource."""
    response = client.get("/api/endpoint/nonexistent-id")

    assert response.status_code == 404
    data = response.get_json()
    assert "not found" in data["message"].lower()
```

### Pattern 4: Database Errors (Mocked)
```python
def test_endpoint_database_error(client, monkeypatch):
    """Test endpoint handles database errors gracefully."""
    def mock_commit():
        raise Exception("Database error")

    monkeypatch.setattr("app.routes.module.db.session.commit", mock_commit)

    payload = {"valid": "data"}
    response = client.post("/api/endpoint", json=payload)

    assert response.status_code == 500
    data = response.get_json()
    assert "error" in data["message"].lower()
```

### Pattern 5: External Service Failures (Mocked)
```python
def test_endpoint_external_api_failure(client, monkeypatch):
    """Test endpoint handles external API failures."""
    def mock_external_call(*args, **kwargs):
        raise Exception("API unavailable")

    monkeypatch.setattr(
        "app.services.service_name.ServiceClass.method",
        staticmethod(mock_external_call)
    )

    response = client.post("/api/endpoint", json={"data": "value"})

    assert response.status_code in [500, 503]
```

### Pattern 6: Validation Errors
```python
def test_endpoint_validation_error(client, db_session):
    """Test endpoint validates business rules."""
    # Setup data that violates business rule
    existing = create_resource(unique_field="value")
    db_session.add(existing)
    db_session.commit()

    # Try to create duplicate
    payload = {"unique_field": "value"}  # Duplicate
    response = client.post("/api/endpoint", json=payload)

    assert response.status_code == 400
    assert "already exists" in response.get_json()["message"]
```

---

## Success Criteria

### Per-File Targets:
- ✅ developer_routes.py: 90%+ (currently 49%)
- ✅ ibkr_routes.py: 90%+ (currently 64%)
- ✅ fund_routes.py: 90%+ (currently 74%)
- ✅ dividend_routes.py: 90%+ (currently 78%)
- ✅ transaction_routes.py: 90%+ (currently 81%)
- ✅ portfolio_routes.py: 90%+ (currently 89%)
- ✅ system_routes.py: 90%+ (currently 71%)

### Overall Targets:
- ✅ Routes: 90%+ coverage (currently varies)
- ✅ Services: Maintain 90%+ coverage (currently achieved)
- ✅ Combined: 90%+ coverage (currently 84%)

---

## Execution Plan

### Day 1 Morning (3-4 hours):
1. ✅ Implement error tests for developer_routes.py
2. ✅ Verify coverage increases to 90%+
3. ✅ Commit: "test: Add error path tests for developer routes"

### Day 1 Afternoon (4-5 hours):
1. ✅ Implement error tests for ibkr_routes.py
2. ✅ Verify coverage increases to 90%+
3. ✅ Commit: "test: Add error path tests for IBKR routes"

### Day 2 Morning (2-3 hours):
1. ✅ Implement error tests for fund_routes.py
2. ✅ Verify coverage increases to 90%+
3. ✅ Commit: "test: Add error path tests for fund routes"

### Day 2 Afternoon (2-3 hours):
1. ✅ Implement error tests for remaining routes
2. ✅ Verify all routes at 90%+
3. ✅ Commit: "test: Add error path tests for dividend, transaction, portfolio routes"

### Day 3 (if needed) (2-3 hours):
1. ✅ Implement service error path tests
2. ✅ Verify overall coverage at 90%+
3. ✅ Final commit: "test: Complete error path testing - 90% coverage achieved"

---

## Commands for Testing

### Check Coverage by File:
```bash
pytest backend/tests/routes --cov=backend/app/routes --cov-report=term-missing -q
```

### Check Overall Coverage:
```bash
pytest backend/tests --cov=backend/app --cov-report=term-missing -q
```

### Run Specific Test File:
```bash
pytest backend/tests/routes/test_developer_routes.py -v
```

### Check Coverage for Single Module:
```bash
pytest backend/tests/routes/test_developer_routes.py \
  --cov=backend/app/routes/developer_routes \
  --cov-report=term-missing
```

---

## Notes

- Focus on **systematic coverage** rather than achieving 100%
- Use `# pragma: no cover` sparingly for truly unreachable code
- Mock external dependencies (APIs, database errors) rather than hitting them
- Each test should be independent and isolated
- Use descriptive test names that explain what error is being tested
- Group related error tests in test classes

---

**Estimated Total Time**: 12-18 hours
**Expected Outcome**: 90%+ coverage on all routes and services
**Next Session Start**: Implement Phase 1 (Developer Routes error tests)

---

**Created by**: Claude Code
**Last Updated**: 2025-11-20
**Status**: Ready for implementation
