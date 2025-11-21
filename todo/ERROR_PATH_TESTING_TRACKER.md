# Error Path Testing - Progress Tracker

**Goal**: Achieve 90%+ coverage on all routes
**Current**: 84% overall | **Target**: 90%+
**Plan**: See ERROR_PATH_TESTING_PLAN.md for full details

---

## Quick Reference: Coverage by File

| File | Current | Target | Missing Lines | Priority | Status |
|------|---------|--------|---------------|----------|--------|
| developer_routes.py | 49% | 90% | 110 | 🔴 P1 | ⏳ TODO |
| ibkr_routes.py | 64% | 90% | 78 | 🔴 P2 | ⏳ TODO |
| fund_routes.py | 74% | 90% | 42 | 🟡 P3 | ⏳ TODO |
| dividend_routes.py | 78% | 90% | 14 | 🟡 P4 | ⏳ TODO |
| transaction_routes.py | 81% | 90% | 15 | 🟡 P5 | ⏳ TODO |
| portfolio_routes.py | 89% | 90% | 13 | 🟢 P6 | ⏳ TODO |
| system_routes.py | 71% | 90% | 6 | 🟡 P7 | ⏳ TODO |

**Legend**: 🔴 High Priority | 🟡 Medium Priority | 🟢 Low Priority | ✅ Complete | ⏳ TODO

---

## Testing Categories Checklist

### For Each Route:

#### 1. Missing Required Fields
- [ ] Test each required field individually
- [ ] Test multiple missing fields
- [ ] Verify 400 status code
- [ ] Verify error message clarity

#### 2. Invalid Field Values
- [ ] Negative numbers (where not allowed)
- [ ] Zero values (where not allowed)
- [ ] Non-numeric values for numeric fields
- [ ] Invalid enum values
- [ ] Out-of-range values
- [ ] Verify 400 status code

#### 3. Resource Not Found (404)
- [ ] Invalid IDs
- [ ] Non-existent resources
- [ ] Verify 404 status code
- [ ] Verify error message

#### 4. Database Errors (Mocked)
- [ ] Mock commit() to raise exception
- [ ] Mock query operations to fail
- [ ] Verify 500 status code
- [ ] Verify graceful error handling

#### 5. External Service Failures (Mocked)
- [ ] Mock external API calls to fail
- [ ] Mock external API to return invalid data
- [ ] Verify 500 or 503 status code
- [ ] Verify fallback behavior (if applicable)

#### 6. Validation Errors
- [ ] Duplicate unique fields
- [ ] Business rule violations
- [ ] State validation failures
- [ ] Verify 400 status code

#### 7. Edge Cases
- [ ] Empty lists
- [ ] None/null values
- [ ] Boundary conditions (min/max)
- [ ] Special characters in strings
- [ ] Very long strings

---

## Implementation Phases

### Phase 1: Developer Routes (Day 1 Morning) ⏳
**File**: `test_developer_routes.py`
**Estimated Time**: 3-4 hours

Test Classes to Add:
- [ ] `TestExchangeRateErrors` (~8 tests)
  - [ ] Missing from_currency
  - [ ] Missing to_currency
  - [ ] Missing rate
  - [ ] Invalid currency code
  - [ ] Invalid rate (negative/zero)
  - [ ] Invalid date format
  - [ ] Database error
  - [ ] General exception

- [ ] `TestFundPriceErrors` (~8 tests)
  - [ ] Missing fund_id
  - [ ] Missing price
  - [ ] Invalid fund_id (not found)
  - [ ] Invalid price (negative/zero)
  - [ ] Invalid date format
  - [ ] Database error
  - [ ] Fund price not found (GET)
  - [ ] General exception

- [ ] `TestCSVImportErrors` (~10 tests)
  - [ ] No file provided
  - [ ] Invalid file format (not CSV)
  - [ ] Missing portfolio_fund_id/fund_id
  - [ ] Invalid ID (not found)
  - [ ] Invalid CSV headers
  - [ ] Wrong file type (transaction vs price)
  - [ ] CSV parsing error (ValueError)
  - [ ] Database error
  - [ ] Unicode decode error
  - [ ] General exception

- [ ] `TestLoggingErrors` (~6 tests)
  - [ ] Invalid logging level
  - [ ] Missing required fields (enabled/level)
  - [ ] Database error getting settings
  - [ ] Database error updating settings
  - [ ] Database error retrieving logs
  - [ ] Database error clearing logs

**Verification**:
```bash
pytest backend/tests/routes/test_developer_routes.py \
  --cov=backend/app/routes/developer_routes \
  --cov-report=term-missing
```
**Expected**: 90%+ coverage

---

### Phase 2: IBKR Routes (Day 1 Afternoon) ⏳
**File**: `test_ibkr_routes.py`
**Estimated Time**: 4-5 hours

Test Classes to Add:
- [ ] `TestFlexImportErrors` (~6 tests)
- [ ] `TestInboxErrors` (~5 tests)
- [ ] `TestAllocationErrors` (~15 tests)
- [ ] `TestDividendMatchingErrors` (~8 tests)
- [ ] `TestBulkOperationErrors` (~8 tests)
- [ ] `TestConfigErrors` (~4 tests)

**Verification**:
```bash
pytest backend/tests/routes/test_ibkr_routes.py \
  --cov=backend/app/routes/ibkr_routes \
  --cov-report=term-missing
```
**Expected**: 90%+ coverage

---

### Phase 3: Fund Routes (Day 2 Morning) ⏳
**File**: `test_fund_routes.py`
**Estimated Time**: 2-3 hours

Test Classes to Add:
- [ ] `TestFundCRUDErrors` (~10 tests)
- [ ] `TestSymbolLookupErrors` (~4 tests)
- [ ] `TestPriceUpdateErrors` (~6 tests)

**Verification**:
```bash
pytest backend/tests/routes/test_fund_routes.py \
  --cov=backend/app/routes/fund_routes \
  --cov-report=term-missing
```
**Expected**: 90%+ coverage

---

### Phase 4: Remaining Routes (Day 2 Afternoon) ⏳
**Estimated Time**: 2-3 hours

#### Dividend Routes
- [ ] `TestDividendErrors` (~8 tests)

#### Transaction Routes
- [ ] `TestTransactionErrors` (~8 tests)

#### Portfolio Routes
- [ ] `TestPortfolioErrors` (~6 tests)

#### System Routes
- [ ] `TestSystemErrors` (~3 tests)

**Verification**:
```bash
pytest backend/tests/routes --cov=backend/app/routes --cov-report=term-missing
```
**Expected**: All routes at 90%+

---

### Phase 5: Service Error Paths (Day 3, if needed) ⏳
**Estimated Time**: 2-3 hours

Focus on:
- [ ] ibkr_transaction_service.py (87% → 90%)
- [ ] ibkr_flex_service.py (77% → 90%)

---

## Testing Patterns Reference

### Quick Copy-Paste Templates

#### Missing Required Field Test
```python
def test_endpoint_missing_field(self, client):
    """Test endpoint rejects missing required field."""
    payload = {"field2": "value"}  # Missing field1
    response = client.post("/api/endpoint", json=payload)

    assert response.status_code == 400
    data = response.get_json()
    assert "Missing required field" in data["message"]
```

#### Invalid Value Test
```python
def test_endpoint_invalid_value(self, client):
    """Test endpoint validates field values."""
    payload = {"field": -1}  # Invalid
    response = client.post("/api/endpoint", json=payload)

    assert response.status_code == 400
```

#### Not Found Test
```python
def test_endpoint_not_found(self, client):
    """Test endpoint returns 404 for missing resource."""
    response = client.get("/api/endpoint/nonexistent-id")

    assert response.status_code == 404
```

#### Database Error Test
```python
def test_endpoint_database_error(self, client, monkeypatch):
    """Test endpoint handles database errors."""
    def mock_commit():
        raise Exception("Database error")

    monkeypatch.setattr("app.routes.module.db.session.commit", mock_commit)

    payload = {"valid": "data"}
    response = client.post("/api/endpoint", json=payload)

    assert response.status_code == 500
```

---

## Commands

### Check coverage for specific file:
```bash
pytest backend/tests/routes/test_developer_routes.py \
  --cov=backend/app/routes/developer_routes \
  --cov-report=term-missing -v
```

### Check overall routes coverage:
```bash
pytest backend/tests/routes \
  --cov=backend/app/routes \
  --cov-report=term-missing -q
```

### Check overall coverage (routes + services):
```bash
pytest backend/tests/routes backend/tests/services \
  --cov=backend/app/routes \
  --cov=backend/app/services \
  --cov-report=term-missing -q
```

### Run only error tests:
```bash
pytest backend/tests/routes -k "Error" -v
```

---

## Progress Tracking

### Session 1 Progress:
- [ ] Phase 1: Developer Routes
- [ ] Coverage check: developer_routes.py at 90%+
- [ ] Commit changes

### Session 2 Progress:
- [ ] Phase 2: IBKR Routes
- [ ] Coverage check: ibkr_routes.py at 90%+
- [ ] Commit changes

### Session 3 Progress:
- [ ] Phase 3: Fund Routes
- [ ] Coverage check: fund_routes.py at 90%+
- [ ] Commit changes

### Session 4 Progress:
- [ ] Phase 4: Remaining Routes
- [ ] Coverage check: All routes at 90%+
- [ ] Commit changes

### Final Check:
- [ ] Overall coverage at 90%+
- [ ] All route files at 90%+
- [ ] Documentation updated
- [ ] Final commit

---

**Last Updated**: 2025-11-20
**Status**: Ready to start Phase 1
**Next Action**: Implement error tests for developer_routes.py
