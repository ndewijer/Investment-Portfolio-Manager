# Error Path Testing - Progress Tracker

**Goal**: Achieve 90%+ coverage on all routes
**Current**: 94.4% routes (839/889 stmts) ✅ | **Target**: 90%+
**Plan**: See ERROR_PATH_TESTING_PLAN.md for full details

---

## Quick Reference: Coverage by File

| File | Current | Target | Missing Lines | Priority | Status |
|------|---------|--------|---------------|----------|--------|
| developer_routes.py | 91% | 90% | 19 | 🔴 P1 | ✅ COMPLETE |
| ibkr_routes.py | 95% | 90% | 11 | 🔴 P2 | ✅ COMPLETE |
| fund_routes.py | 96% | 90% | 7 | 🟡 P3 | ✅ COMPLETE |
| portfolio_routes.py | 100% | 90% | 0 | 🟢 P4 | ✅ COMPLETE |
| transaction_routes.py | 100% | 90% | 0 | 🟡 P5 | ✅ COMPLETE |
| dividend_routes.py | 100% | 90% | 0 | 🟡 P6 | ✅ COMPLETE |
| system_routes.py | 100% | 90% | 0 | 🟡 P7 | ✅ COMPLETE |

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

### Phase 1: Developer Routes ✅ COMPLETE
**File**: `test_developer_routes.py`
**Actual Time**: ~3 hours
**Final Coverage**: 91% (exceeds 90% target)

Test Classes Added:
- [x] `TestExchangeRateErrors` (7 tests)
  - [x] Missing from_currency
  - [x] Missing to_currency
  - [x] Missing rate
  - [x] Invalid currency code
  - [x] Invalid rate (negative/zero)
  - [x] Invalid date format
  - [x] Database error

- [x] `TestFundPriceErrors` (7 tests)
  - [x] Missing fund_id
  - [x] Missing price
  - [x] Invalid fund_id (not found)
  - [x] Invalid price (negative/zero)
  - [x] Invalid date format
  - [x] Database error
  - [x] General exception

- [x] `TestCSVImportErrors` (10 tests)
  - [x] No file provided
  - [x] Invalid file format (not CSV)
  - [x] Missing portfolio_fund_id/fund_id
  - [x] Invalid ID (not found)
  - [x] Invalid CSV headers
  - [x] Wrong file type (transaction vs price)
  - [x] CSV parsing error (ValueError)
  - [x] Database error
  - [x] Unicode decode error
  - [x] General exception

- [x] `TestLoggingErrors` (6 tests)
- [x] Added error tests to `TestCSVTemplates` (2 tests)

**Total**: 32 error path tests added
**Bugs Fixed**: Date isoformat() error in error handlers (lines 76, 679)
**Commit**: `test: Add error path tests for developer routes (91% coverage)`

---

### Phase 2: IBKR Routes ✅ COMPLETE
**File**: `test_ibkr_routes.py`
**Actual Time**: ~4 hours
**Final Coverage**: 86% (22% improvement, close to 90% target)

Test Classes Added:
- [x] `TestIBKRConfigErrors` (8 tests)
- [x] `TestIBKRConnectionErrors` (4 tests)
- [x] `TestIBKRImportErrors` (4 tests)
- [x] `TestIBKRInboxErrors` (4 tests)
- [x] `TestIBKRAllocationErrors` (4 tests)
- [x] `TestIBKRBulkOperationsErrors` (2 tests)

**Total**: 26 error path tests added
**Bugs Fixed**:
- Used correct service method names (get_first_config, fetch_statement)
- Fixed mocking patterns to use unittest.mock.patch
- Adjusted status code assertions for validation order
**Commit**: `test: Add error path tests for IBKR routes (86% coverage)`

---

### Phase 3: Fund Routes ✅ COMPLETE
**File**: `test_fund_routes.py`
**Actual Time**: ~2 hours
**Final Coverage**: 96% (exceeds 90% target by 6%!)

Test Classes Added:
- [x] `TestFundCRUDErrors` (13 tests)
  - [x] Missing required fields (name, ISIN, currency)
  - [x] Database errors (create, update, delete, get)
  - [x] Not found errors
  - [x] Check usage errors
- [x] `TestSymbolLookupErrors` (4 tests)
  - [x] External API failures
  - [x] Force refresh failures
  - [x] Cache errors
  - [x] Empty symbol handling
- [x] `TestPriceUpdateErrors` (8 tests)
  - [x] Fund not found
  - [x] Database errors
  - [x] API failures (today & historical)
  - [x] Authentication errors (missing API key, invalid token)
  - [x] Database error in bulk update

**Total**: 24 error path tests added (originally estimated ~20)
**All Tests**: 42 passed
**Overall Coverage**: Jumped to 90.51% (exceeded 90% target!)
**Commit**: `test: Add error path tests for fund routes (96% coverage)`

**Key Achievements**:
- 🎉 **Overall project coverage now exceeds 90% target!**
- Fund routes improved from 74% → 96% (22% improvement)
- Used proper mocking patterns with unittest.mock.patch for complex queries

---

### Phase 4: Remaining Routes ✅ COMPLETE
**Actual Time**: ~3 hours
**Final Coverage**: 93.4% routes overall (809/866 stmts)

#### Phase 4a: Portfolio Routes ✅ COMPLETE
**File**: `test_portfolio_routes.py`
**Final Coverage**: 100% (89% → 100%)

Test Classes Added:
- [x] `TestPortfolioErrors` (8 tests)
  - [x] Create portfolio service error
  - [x] Get portfolio not found
  - [x] Update portfolio service error
  - [x] Delete portfolio service error
  - [x] Archive portfolio not found
  - [x] Unarchive portfolio not found
  - [x] Create portfolio fund duplicate
  - [x] Delete portfolio fund requires confirmation

**Refactoring**: Removed duplicate route registration (dead code at lines 394-400)
**Total**: 8 error path tests + refactoring
**Commit**: `Phase 4a: Add error path tests for portfolio routes (89% → 100%)`

---

#### Phase 4b: System Routes ✅ COMPLETE
**File**: `test_system_routes.py`
**Final Coverage**: 100% (71% → 100%)

Test Classes Added:
- [x] `TestSystemErrors` (2 tests)
  - [x] Get version info service error
  - [x] Health check database error

**Total**: 2 error path tests
**Commit**: `Phase 4b: Add error path tests for system routes (71% → 100%)`

---

#### Phase 4c: Transaction Routes ✅ COMPLETE
**File**: `test_transaction_routes.py`
**Final Coverage**: 100% (81% → 100%)

Test Classes Added:
- [x] `TestTransactionErrors` (6 tests)
  - [x] Create transaction service error
  - [x] Get fund transactions service error
  - [x] Get portfolio fund transactions service error
  - [x] Update transaction not found
  - [x] Update transaction general error
  - [x] Delete transaction service error

**Refactoring**: Converted to use unittest.mock.patch instead of monkeypatch
**Total**: 6 error path tests + refactoring
**Commit**: `Phase 4c: Add error path tests for transaction routes (81% → 100%)`

---

#### Phase 4d: Dividend Routes ✅ COMPLETE
**File**: `test_dividend_routes.py`
**Final Coverage**: 100% (78% → 100%)

Test Classes Added:
- [x] `TestDividendErrors` (7 tests)
  - [x] Create dividend service error
  - [x] Get fund dividends service error
  - [x] Get portfolio dividends service error
  - [x] Update dividend value error
  - [x] Update dividend general error
  - [x] Delete dividend value error
  - [x] Delete dividend general error

**Total**: 7 error path tests
**All Tests**: 17 passed (10 existing + 7 new)
**Commit**: `Phase 4d: Add error path tests for dividend routes (78% → 100%)`

---

#### Phase 4e: IBKR Routes ✅ COMPLETE
**File**: `test_ibkr_routes.py`
**Final Coverage**: 95% (86% → 95%)

Test Classes Added:
- [x] `TestIBKRConnectionErrors` (2 additional tests)
  - [x] Connection success
  - [x] Connection failure
- [x] `TestIBKRInboxErrors` (3 additional tests)
  - [x] Get inbox count service error
  - [x] Get eligible portfolios transaction not found
  - [x] Get eligible portfolios service error
- [x] `TestIBKRAllocationErrors` (3 additional tests)
  - [x] Update allocations missing allocations
  - [x] Update allocations value error
  - [x] Update allocations general error
- [x] `TestIBKRBulkOperationsErrors` (3 additional tests)
  - [x] Bulk allocate empty allocations
  - [x] Bulk allocate invalid percentage sum
  - [x] Bulk allocate partial failure

**Total**: 11 error path tests (51 existing tests + 11 new = 61 total, 60 passing, 1 skipped)
**All Tests**: 60 passed, 1 skipped
**Commit**: `Phase 4e: Add error path tests for IBKR routes (86% → 95%)`

---

**Phase 4 Summary**:
- Total tests added: 34 error path tests across 5 route files
- Files at 100% coverage: portfolio, system, transaction, dividend routes (4 files)
- Files at 95%+ coverage: ibkr routes (1 file)
- All tests use unittest.mock.patch for consistency
- Overall routes coverage: 94.4% (exceeds 90% target by 4.4%!)

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

### Session 1 Progress: ✅ COMPLETE
- [x] Phase 1: Developer Routes
- [x] Coverage check: developer_routes.py at 91% (exceeds target)
- [x] Commit changes

### Session 2 Progress: ✅ COMPLETE
- [x] Phase 2: IBKR Routes
- [x] Coverage check: ibkr_routes.py at 86% (22% improvement)
- [x] Commit changes

### Session 3 Progress: ✅ COMPLETE
- [x] Phase 3: Fund Routes
- [x] Coverage check: fund_routes.py at 96% (exceeds target!)
- [x] Commit changes
- [x] **MILESTONE: Overall coverage exceeded 90% target (90.51%)!**

### Session 4 Progress: ✅ COMPLETE
- [x] Phase 4a: Portfolio Routes (8 tests, 89% → 100%)
- [x] Phase 4b: System Routes (2 tests, 71% → 100%)
- [x] Phase 4c: Transaction Routes (6 tests, 81% → 100%)
- [x] Phase 4d: Dividend Routes (7 tests, 78% → 100%)
- [x] Phase 4e: IBKR Routes (11 tests, 86% → 95%)
- [x] Coverage check: All routes at 90%+ ✅
- [x] Commit changes for each sub-phase
- [x] Documentation updated

### Final Check: ✅ COMPLETE
- [x] Overall routes coverage at 94.4% (exceeds 90% target!)
- [x] 4 route files at 100% coverage
- [x] 3 route files at 90%+ coverage
- [x] All 7 route files exceed 90% target! 🎉
- [x] Documentation updated
- [x] All commits completed

---

**Last Updated**: 2025-11-22
**Status**: Phase 1, 2, 3 & 4 Complete - 🎉 **94.4% ROUTES COVERAGE ACHIEVED!**
**Next Action**: All routes now exceed 90% target - testing complete!
**Overall Routes Coverage**: 94.4% (839/889 stmts) ✅ (exceeded 90% target by 4.4%!)

**Summary of All Completed Phases**:
- Phase 1: developer_routes.py 49% → 91% (32 tests)
- Phase 2: ibkr_routes.py 64% → 86% (26 tests)
- Phase 3: fund_routes.py 74% → 96% (24 tests)
- Phase 4a: portfolio_routes.py 89% → 100% (8 tests)
- Phase 4b: system_routes.py 71% → 100% (2 tests)
- Phase 4c: transaction_routes.py 81% → 100% (6 tests)
- Phase 4d: dividend_routes.py 78% → 100% (7 tests)
- Phase 4e: ibkr_routes.py 86% → 95% (11 tests)
- **Total**: 116 error path tests added across all phases
