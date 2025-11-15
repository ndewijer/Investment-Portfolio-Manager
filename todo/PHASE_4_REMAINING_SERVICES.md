# Phase 4: Remaining Service Tests + Route Integration

**Status**: Planned for future development
**Previous**: Phase 3 Complete (5 core services, 98 tests, 95%+ coverage)

---

## Remaining Service Tests (High Priority)

### 1. IBKRFlexService - Complex External API Integration
**Current**: 12% coverage (290 statements)
**Priority**: HIGH - Critical for IBKR functionality
**Complexity**: VERY HIGH - External API, XML parsing, encryption

**Key Methods to Test**:
- `fetch_flex_statement()` - IBKR API integration (mock all calls)
- `parse_flex_statement()` - XML parsing accuracy
- `_encrypt_token()` / `_decrypt_token()` - Security critical
- `import_transactions()` - Full import flow
- Cache behavior (24-hour TTL)
- Error handling for all 21 IBKR API error codes

**Testing Challenges**:
- Requires mocking complex IBKR API responses
- XML parsing validation
- Security testing for encryption
- Cache behavior testing

**Estimated**: 15-20 tests

---

### 2. IBKRTransactionService - Transaction Processing
**Current**: 11% coverage (171 statements)
**Priority**: HIGH - Business logic critical
**Depends**: IBKRFlexService (should be tested after)

**Key Methods to Test**:
- `modify_allocations()` - Allocation validation and updates
- `validate_allocations()` - Must sum to 100%
- Transaction splitting across portfolios
- Fund matching logic

**Estimated**: 8-12 tests

---

### 3. PriceUpdateService - External Price Data
**Current**: 18% coverage (90 statements)
**Priority**: MEDIUM - Data integrity important
**Complexity**: MEDIUM - External API dependency

**Key Methods to Test**:
- `update_historical_prices()` - yfinance integration (mock)
- `update_all_fund_prices()` - Bulk operations
- Error handling for missing/invalid symbols
- Currency conversion

**Testing Challenges**:
- Mock yfinance API calls
- Handle network failures
- Test various symbol formats

**Estimated**: 6-10 tests

---

## Lower Priority Services

### 4. DeveloperService - 23% coverage (149 statements)
**Priority**: LOW - Development utilities
**Estimated**: 4-6 tests

### 5. SymbolLookupService - 19% coverage (69 statements)
**Priority**: LOW - External lookups
**Estimated**: 4-6 tests

### 6. FundMatchingService - 0% coverage (62 statements)
**Priority**: LOW - Specialized functionality
**Estimated**: 3-5 tests

### 7. LoggingService - 73% coverage (56 statements)
**Priority**: LOWEST - Already decent coverage
**Estimated**: 2-3 tests for remaining gaps

---

## Route Integration Tests (Original Phase 4 Plan)

**Total Endpoints**: 58 across 6 route files

### High Priority Routes (Business Critical)
1. **Portfolio Routes** (8 endpoints) - Core functionality
2. **Transaction Routes** (5 endpoints) - Financial operations
3. **Dividend Routes** (5 endpoints) - Dividend management

### Medium Priority Routes
4. **Fund Routes** (10 endpoints) - Fund management
5. **IBKR Routes** (16 endpoints) - IBKR integration

### Lower Priority Routes
6. **System & Developer Routes** (14 endpoints) - Utilities

---

## Phase 4 Execution Strategy

### Option A: Service Tests First (Recommended)
1. Complete remaining service tests (IBKRFlexService, IBKRTransactionService, PriceUpdateService)
2. Then do route integration tests
3. Focus on high business impact

### Option B: Route Tests First
1. Start with route integration tests (more user-facing)
2. Fill in remaining service gaps after
3. Faster user-visible progress

### Option C: Mixed Approach
1. High-priority routes + High-priority services in parallel
2. Balance technical debt vs user features

---

## Success Criteria for Phase 4

**Service Testing**:
- IBKRFlexService: 80%+ coverage
- IBKRTransactionService: 85%+ coverage
- PriceUpdateService: 80%+ coverage
- Others: 70%+ coverage

**Route Testing**:
- 100% endpoint coverage (all 58 endpoints)
- Request/response validation
- Error handling verification

**Overall**:
- Maintain 80%+ backend coverage
- All tests passing
- Comprehensive documentation

---

## Estimated Timeline

**Service Tests**: 1-2 weeks
- IBKRFlexService: 3-4 days (complex)
- IBKRTransactionService: 2-3 days
- PriceUpdateService: 2-3 days
- Other services: 2-3 days

**Route Integration Tests**: 1-2 weeks
- Portfolio/Transaction/Dividend: 1 week
- Fund/IBKR routes: 1 week
- System/Developer routes: 2-3 days

**Total Phase 4**: 2-4 weeks (depending on scope)

---

## Notes from Phase 3

**What worked well**:
- Query-Specific Data testing pattern
- Comprehensive factory-based test data
- Systematic coverage improvement
- Thorough documentation

**Lessons learned**:
- External API services need careful mocking strategy
- Complex services benefit from incremental testing approach
- Documentation during development prevents knowledge loss

**Technical debt**:
- Some SQLAlchemy deprecation warnings to address
- Test isolation patterns to standardize
- Performance testing integration needed

---

**Created**: Phase 3 completion
**Next Review**: When Phase 4 begins