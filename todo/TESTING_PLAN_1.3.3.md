# Testing Plan for v1.3.3: Comprehensive Backend Testing

**Version**: 1.3.3
**Created**: 2025-11-12
**Goal**: Achieve 80%+ backend test coverage with high-quality unit and integration tests
**Timeline**: 3-4 weeks (comprehensive)
**Strategy**: Refactor first (move business logic to services), then test with separate test database

---

## Phase 1: Infrastructure & Test Database Setup (Days 1-3)

### Objectives
- Create isolated test database
- Add necessary testing dependencies
- Establish test fixtures and factories
- Set up test data management

### Tasks

#### 1.1 Test Database Configuration
- Create `backend/tests/test_config.py` with test database settings
- Use SQLite in-memory or separate file: `test_portfolio_manager.db`
- Configure pytest to use test database via `conftest.py`
- Add database initialization/cleanup fixtures

#### 1.2 Testing Dependencies
Add to `requirements.txt` (latest versions from PyPI as of 2025-11-12):
```
pytest-mock==3.15.1          # Mocking support
responses==0.25.8             # HTTP request mocking
freezegun==1.5.5              # Time/date mocking
factory-boy==3.3.3            # Test data factories
faker==38.0.0                 # Fake data generation
pytest-timeout==2.4.0         # Prevent hanging tests
```

#### 1.3 Test Data Factories
Create `backend/tests/factories.py`:
- `PortfolioFactory` - create test portfolios
- `FundFactory` - create funds with different dividend types
- `TransactionFactory` - create transactions of all types
- `DividendFactory` - create dividends (CASH and STOCK)
- `IBKRTransactionFactory` - create IBKR test data

#### 1.4 Common Test Fixtures
Enhance `conftest.py`:
- `test_db` - isolated database with schema
- `sample_portfolio_with_data` - portfolio with funds, transactions, dividends
- `cash_dividend_fund` / `stock_dividend_fund` - specific fund types
- `mock_yfinance` - mock price data
- `mock_ibkr_api` - mock IBKR API responses
- `freeze_time` - control date/time for tests

**Deliverable**: Test infrastructure ready for writing tests

---

## Phase 2: Refactoring - Move Business Logic to Services (Days 4-10)

### Objectives
- Extract business logic from routes to services
- Make code testable through proper separation of concerns
- Fix the 8 critical violations identified in the route analysis

### Critical Refactorings (Priority Order)

#### 2.1 DividendService Refactoring (Day 4-5)
**Problem**: 100+ lines of business logic in `dividend_routes.py:157-263`

**Create**: `DividendService.update_dividend(dividend_id, data)`
- Handle CASH vs STOCK dividend logic
- Manage reinvestment transaction creation/update/deletion
- Recalculate total amounts
- Update reinvestment status

**Test while refactoring**: Integration test before/after to ensure behavior unchanged

#### 2.2 IBKRTransactionService Refactoring (Day 6-7)
**Problem**: 150+ lines of allocation logic in `ibkr_routes.py:811-970`

**Create**: `IBKRTransactionService.modify_allocations(transaction_id, allocations)`
- Validate percentages sum to 100%
- Handle existing allocation updates
- Create new allocations
- Delete removed allocations
- Transaction creation for each allocation

**Also Create**: `IBKRTransactionService.unallocate_transaction(transaction_id)`
- From `ibkr_routes.py:656-736`

#### 2.3 TransactionService Refactoring (Day 8)
**Problem**: Complex orchestration in `transaction_routes.py:242-347`

**Create**: `TransactionService.delete_with_cleanup(transaction_id)`
- Check IBKR allocation linkage
- Revert IBKR transaction status if needed
- Delete realized gain/loss records for sells
- Clean up transaction

**Enhance**: `TransactionService.format_transaction(transaction)`
- Include realized gain data automatically

#### 2.4 PortfolioService Refactoring (Day 9)
**Problem**: Multiple business logic leaks in portfolio_routes.py

**Create**:
- `PortfolioService.delete_portfolio_fund(portfolio_fund_id, confirmed=False)`
- `PortfolioService.format_portfolio_detail(portfolio, portfolio_funds_data)`
- `PortfolioService.get_portfolios(include_excluded=False)`

**Fix**: N+1 query in `get_portfolio_funds()` with proper eager loading

#### 2.5 FundService Refactoring (Day 10)
**Problem**: Multiple violations in fund_routes.py

**Create**:
- `FundService.create_fund(data)` - include symbol lookup orchestration
- `FundService.check_fund_usage(fund_id)` - return usage data
- `FundService.delete_fund(fund_id)` - handle deletion with checks

**Create**: `PriceUpdateService.update_all_fund_prices()`

#### 2.6 IBKRConfigService Creation (Day 10)
**Problem**: Config save logic in `ibkr_routes.py:72-150`

**Create**: `IBKRConfigService.save_config(data)`
- Handle encryption
- Parse dates
- Create or update config

**Deliverable**: All critical business logic moved to services, routes are thin controllers

---

## Phase 3: Service Layer Unit Tests (Days 11-20)

### Objectives
- Test all 11 services with unit tests
- Mock database and external dependencies
- Achieve 90%+ coverage on critical services
- Test edge cases and error handling

### 3.1 DividendService Tests (Day 11-12)
**Create**: `backend/tests/test_dividend_service.py`

**Test Coverage**:
- `test_calculate_shares_owned_*` (multiple scenarios)
  - Buy only transactions
  - Buy + sell mix
  - Multiple buys on different dates
  - Including dividend transactions
  - Edge: negative shares, zero shares
- `test_create_dividend_cash_type` - auto-completed status, no transaction
- `test_create_dividend_stock_with_reinvestment` - transaction created, shares updated
- `test_create_dividend_stock_without_reinvestment` - pending status
- `test_update_dividend_cash_to_stock` - type conversion
- `test_update_dividend_add_reinvestment` - create transaction
- `test_update_dividend_remove_reinvestment` - delete transaction
- `test_update_dividend_modify_reinvestment` - update transaction
- `test_delete_dividend_with_transaction` - cascade delete
- `test_delete_dividend_cash_only` - simple delete
- `test_dividend_validation_errors` - invalid data

**Special Tests**:
- `test_dividend_cash_vs_stock_behavior` - document differences
- `test_dividend_reinvestment_calculations` - accuracy
- `test_dividend_share_tracking` - position accuracy

**Target**: 90%+ coverage

### 3.2 TransactionService Tests (Day 13-14)
**Create**: `backend/tests/test_transaction_service.py`

**Test Coverage**:
- `test_create_transaction_*` (all types: buy, sell, dividend, fee)
- `test_process_sell_transaction` - realized gain calculation (profit, loss, exact)
- `test_calculate_current_position` - accurate tracking
- `test_delete_with_cleanup_*` - various scenarios
- `test_average_cost_tracking` - accurate calculations
- `test_overselling_validation` - error handling
- `test_format_transaction_with_realized_gain` - automatic inclusion

**Target**: 85%+ coverage

### 3.3 IBKRFlexService Tests (Day 15-16)
**Create**: `backend/tests/test_ibkr_flex_service.py`

**Test Coverage**:
- `test_encrypt_decrypt_token` - security critical
- `test_fetch_statement_*` (mocked API, all 21 error codes)
- `test_parse_flex_statement_*` - XML parsing
- `test_import_transactions` - full flow
- `test_cache_*` - 24-hour cache behavior
- `test_import_exchange_rates` - currency rates
- `test_invalid_xml_handling` - error cases
- `test_token_expiration_detection` - security

**Mock**: All IBKR API calls with `responses` library

**Target**: 80%+ coverage

### 3.4 IBKRTransactionService Tests (Day 17)
**Create**: `backend/tests/test_ibkr_transaction_service.py`

**Test Coverage**:
- `test_validate_allocations_*` - validation logic
- `test_process_transaction_allocation` - split across portfolios
- `test_modify_allocations_*` - update, add, remove scenarios
- `test_unallocate_transaction` - full unallocation
- `test_match_dividend_to_existing` - dividend matching
- `test_get_or_create_*` - fund and portfolio_fund management

**Target**: 85%+ coverage

### 3.5 PortfolioService Tests (Day 18)
**Create**: `backend/tests/test_portfolio_service.py` (expand existing)

**Add to existing 12 tests**:
- `test_calculate_portfolio_fund_values_accuracy` - correctness
- `test_format_portfolio_detail` - aggregation logic
- `test_delete_portfolio_fund_*` - various scenarios
- `test_get_portfolios_*` - filtering
- `test_realized_vs_unrealized_gains` - separation
- `test_cost_basis_adjustments` - after sales
- `test_empty_portfolio_handling` - edge case
- `test_missing_price_data` - edge case

**Target**: 90%+ coverage (from current 75%)

### 3.6 PriceUpdateService Tests (Day 19)
**Create**: `backend/tests/test_price_update_service.py`

**Test Coverage**:
- `test_update_historical_prices` (mock yfinance)
- `test_update_all_fund_prices` - bulk operation
- `test_update_fund_prices_error_handling` - missing/invalid symbols
- `test_historical_backfill` - date range
- `test_exchange_rate_updates` - currency data

**Mock**: All yfinance calls

**Target**: 80%+ coverage

### 3.7 Remaining Services Tests (Day 20)
**Create**:
- `test_fund_service.py` - CRUD operations
- `test_fund_matching_service.py` - matching logic
- `test_symbol_lookup_service.py` - external lookups (mock)
- `test_logging_service.py` - logging patterns
- `test_developer_service.py` - utilities

**Target**: 70%+ coverage

**Deliverable**: All services have comprehensive unit tests with high coverage

---

## Phase 4: Route Integration Tests (Days 21-24)

### Objectives
- Test all 58 API endpoints
- Validate request/response contracts
- Test error handling and validation

### 4.1 Portfolio Routes Integration Tests (Day 21)
**Create**: `backend/tests/test_portfolio_routes_integration.py`

**Test Coverage** (8 endpoints):
- All CRUD operations
- History endpoints
- Archive/restore flows
- Portfolio fund management

### 4.2 Transaction Routes Integration Tests (Day 21)
**Create**: `backend/tests/test_transaction_routes_integration.py`

**Test Coverage** (5 endpoints):
- All transaction types
- Update and delete with cleanup
- List with filters

### 4.3 Dividend Routes Integration Tests (Day 22)
**Create**: `backend/tests/test_dividend_routes_integration.py`

**Test Coverage** (5 endpoints):
- CASH and STOCK dividend flows
- Update scenarios
- List by fund/portfolio

### 4.4 Fund Routes Integration Tests (Day 22)
**Create**: `backend/tests/test_fund_routes_integration.py`

**Test Coverage** (10 endpoints):
- CRUD operations
- Symbol lookup integration
- Price update endpoints
- Usage checks

### 4.5 IBKR Routes Integration Tests (Day 23)
**Create**: `backend/tests/test_ibkr_routes_integration.py`

**Test Coverage** (16 endpoints):
- Config management
- Import flows (mocked)
- Allocation operations
- Bulk operations
- Dividend matching

### 4.6 System & Developer Routes Tests (Day 24)
**Create**:
- `test_system_routes_integration.py` (2 endpoints)
- `test_developer_routes_integration.py` (12 endpoints)

**Deliverable**: All 58 API endpoints tested

---

## Phase 5: Coverage Analysis & Edge Cases (Days 25-27)

### Objectives
- Measure actual coverage
- Fill gaps to reach 80%+
- Test complex edge cases
- Document test patterns

### 5.1 Coverage Analysis (Day 25)
**Tasks**:
- Run `pytest --cov=app --cov-report=html --cov-report=term-missing`
- Identify files/functions below 80%
- Prioritize untested code by criticality
- Create list of missing tests

### 5.2 Fill Coverage Gaps (Day 26)
**Focus Areas**:
- Model methods (if any business logic)
- Error handling paths
- Edge cases identified but not yet tested
- Utility functions

### 5.3 Edge Case Testing (Day 27)
**Critical Edge Cases**:
- **Dividends**: Multiple on same date, partially sold positions, zero shares
- **Transactions**: Same-day buy/sell, overselling, zero-cost transactions
- **IBKR**: Duplicate imports, network failures, token expiration
- **Portfolios**: Empty portfolios, archived, missing prices
- **Calculations**: Division by zero, negative values, large numbers
- **Dates**: Timezone handling, boundaries, historical dates

### 5.4 Documentation (Day 27)
**Update**: `docs/TESTING.md`
- Document all test fixtures
- Explain test data factories
- Show example test patterns
- Document mocking strategies
- Add troubleshooting section
- Update coverage statistics

**Deliverable**: 80%+ coverage achieved, documentation updated

---

## Phase 6: CI/CD Integration (Days 28-30)

### Objectives
- Integrate tests into pre-commit hooks
- Add tests to GitHub Actions workflow
- Block PRs with failing tests or insufficient coverage
- Ensure tests run automatically

### 6.1 Pre-commit Integration (Day 28)

**Update**: `.pre-commit-config.yaml`

Add pytest hook:
```yaml
  - repo: local
    hooks:
      - id: pytest
        name: pytest
        entry: bash -c 'cd backend && source .venv/bin/activate && pytest tests/ -v --tb=short'
        language: system
        pass_filenames: false
        always_run: true
```

**Optional**: Add coverage check hook:
```yaml
      - id: pytest-coverage
        name: pytest-coverage
        entry: bash -c 'cd backend && source .venv/bin/activate && pytest --cov=app --cov-fail-under=80'
        language: system
        pass_filenames: false
        always_run: true
```

**Configure**:
- Decide if all tests run on every commit or only on push
- Set coverage threshold (80%)
- Configure test execution time limits

### 6.2 GitHub Actions Workflow (Day 29)

**Create/Update**: `.github/workflows/backend-tests.yml`

```yaml
name: Backend Tests

on:
  pull_request:
    branches: [main]
    paths:
      - 'backend/**'
      - '.github/workflows/backend-tests.yml'
  push:
    branches: [main]
    paths:
      - 'backend/**'

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.13'

      - name: Cache dependencies
        uses: actions/cache@v4
        with:
          path: ~/.cache/pip
          key: ${{ runner.os }}-pip-${{ hashFiles('backend/requirements.txt') }}

      - name: Install dependencies
        run: |
          cd backend
          pip install -r requirements.txt

      - name: Run tests with coverage
        run: |
          cd backend
          pytest tests/ -v --cov=app --cov-report=xml --cov-report=term

      - name: Check coverage threshold
        run: |
          cd backend
          pytest --cov=app --cov-fail-under=80

      - name: Upload coverage to Codecov (optional)
        uses: codecov/codecov-action@v4
        with:
          file: ./backend/coverage.xml
          flags: backend
          name: backend-coverage
```

**Configure**:
- Set coverage threshold to 80%
- Add coverage badge to README.md
- Configure PR status checks

### 6.3 Branch Protection Rules (Day 30)

**Configure GitHub Branch Protection** for `main`:
- ✅ Require status checks to pass before merging
  - ✅ Backend Tests
  - ✅ Coverage >= 80%
- ✅ Require branches to be up to date before merging
- ✅ Require conversation resolution before merging
- ⚠️ Do not bypass the above settings for administrators

**Test the workflow**:
1. Create test PR with intentional test failure
2. Verify PR is blocked
3. Fix test, verify PR can merge
4. Create PR that reduces coverage below 80%
5. Verify PR is blocked

### 6.4 Documentation (Day 30)

**Update**: `docs/TESTING.md`
- Add CI/CD section
- Document pre-commit test execution
- Document GitHub Actions workflow
- Add troubleshooting for CI failures
- Document how to run tests locally before pushing

**Update**: `CONTRIBUTING.md` (if exists)
- Add testing requirements for PRs
- Explain coverage requirements
- Show how to run tests locally

**Update**: `README.md`
- Add test coverage badge
- Add CI status badge
- Link to testing documentation

**Deliverable**: Fully automated testing pipeline blocking bad PRs

---

## Success Criteria

### Coverage Targets
- **Overall backend coverage**: 80%+
- **Critical services** (dividend, transaction, IBKR): 90%+
- **Medium priority services** (portfolio, fund, price): 85%+
- **Lower priority services**: 70%+
- **Routes (integration tests)**: 100% endpoint coverage

### Quality Metrics
- ✅ All tests passing
- ✅ No flaky tests
- ✅ Test execution time < 2 minutes for full suite
- ✅ No warnings or deprecations
- ✅ Clear test names and documentation

### CI/CD Integration
- ✅ Tests run automatically on every PR
- ✅ PRs blocked if tests fail
- ✅ PRs blocked if coverage drops below 80%
- ✅ Pre-commit hooks run tests locally
- ✅ Coverage badge visible in README

### Deliverables
1. ✅ Separate test database setup
2. ✅ Test fixtures and factories
3. ✅ 8 critical refactorings completed
4. ✅ ~150+ unit tests written
5. ✅ ~60+ integration tests written
6. ✅ 80%+ coverage achieved
7. ✅ Updated documentation
8. ✅ Pre-commit integration
9. ✅ GitHub Actions workflow
10. ✅ Branch protection rules configured

---

## Estimated Timeline

| Phase | Days | Dates (if starting Nov 12, 2025) |
|-------|------|---------------------------|
| 1. Infrastructure Setup | 3 | Nov 12-14 |
| 2. Refactoring | 7 | Nov 15-21 |
| 3. Service Unit Tests | 10 | Nov 22-Dec 1 |
| 4. Route Integration Tests | 4 | Dec 2-5 |
| 5. Coverage & Edge Cases | 3 | Dec 6-8 |
| 6. CI/CD Integration | 3 | Dec 9-11 |
| **Total** | **30 days** | **~4 weeks** |

**Note**: 3-day buffer included for unexpected issues

---

## Future Work (v1.3.4+)

### Frontend Testing
- Jest + React Testing Library setup
- Component unit tests
- Hook tests
- Integration tests with mocked backend
- E2E tests with Cypress/Playwright

### Performance Testing
- Load testing with locust
- Database query performance tests
- Concurrent user tests
- Memory leak detection

### Security Testing
- Authentication tests
- Authorization tests
- Input validation tests
- SQL injection prevention
- XSS prevention

---

## Risk Mitigation

### Risk: Refactoring breaks existing functionality
**Mitigation**: Write integration tests BEFORE refactoring to lock in behavior

### Risk: Test database setup issues
**Mitigation**: Start with this in Phase 1, can fall back if blocked

### Risk: External API mocking complexity
**Mitigation**: Use `responses` library, start simple

### Risk: Coverage goals too ambitious
**Mitigation**: Prioritize critical code, 70% is still good

### Risk: CI/CD breaks existing workflow
**Mitigation**: Test on feature branch first, gradual rollout

### Risk: Tests too slow for pre-commit
**Mitigation**: Make pre-commit optional or run subset of tests

---

## Daily Workflow Pattern

Each development day:
1. Write tests first (TDD where possible)
2. Run tests frequently (`pytest tests/`)
3. Check coverage after each session (`pytest --cov=app`)
4. Commit tested code daily
5. Update TODO.md progress
6. Document any blockers or decisions

**After Phase 6 completion**:
- Pre-commit runs tests automatically
- Push triggers GitHub Actions
- Monitor CI results
- Fix failures before merging

---

**Last Updated**: 2025-11-12
**Status**: Planning complete, ready to begin Phase 1
