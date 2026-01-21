# Frontend Test Documentation

This directory contains comprehensive documentation for all frontend test suites, organized by category and test type.

---

## Directory Structure

```
frontend/tests/docs/
├── README.md                           # This file - navigation index
├── UNIT_TEST_COVERAGE_SUMMARY.md      # Overall coverage progress and achievements
└── infrastructure/                     # Testing infrastructure documentation
    └── COVERAGE_MONITORING.md          # Coverage monitoring and alerting guide
```

---

## Quick Start

### Running Tests

**Unit Tests (Jest):**
```bash
cd frontend

# Run all unit tests with coverage
npm test

# Run tests in watch mode
npm run test:watch

# Generate HTML coverage report
npm run test:coverage
open coverage/index.html

# Run for CI/CD
npm run test:ci
```

**E2E Tests (Playwright):**
```bash
cd frontend

# Run all E2E tests (headless)
npm run test:e2e

# Run with interactive UI
npm run test:e2e:ui

# Run in headed mode (see browser)
npm run test:e2e:headed

# Debug mode (step through tests)
npm run test:e2e:debug
```

### Coverage Status

**Overall Coverage** (Business Logic Only):
- **Lines**: 93.11% ✅
- **Statements**: 90.74% ✅
- **Branches**: 84.83% ✅
- **Functions**: 81.3% ✅

**Test Count**: 584 tests (575 passing, 9 skipped)
**Coverage Thresholds** (Mandatory): Lines 90%, Statements 90%, Branches 84%, Functions 80%

---

## Documentation Categories

### 📊 Coverage Summary

📋 **[UNIT_TEST_COVERAGE_SUMMARY.md](UNIT_TEST_COVERAGE_SUMMARY.md)**
- Overall coverage: 93.11% lines
- Category-by-category breakdown
- All hooks tested (89-94% coverage)
- Testing patterns and best practices
- Files and line counts

**Coverage by Category**:
- ✅ All utility functions: 100% coverage
- ✅ All hooks: 89-94% coverage
- ✅ Context providers: 87% coverage
- ✅ Shared components: 79% coverage

### 🔧 Infrastructure Documentation

📋 **[infrastructure/COVERAGE_MONITORING.md](infrastructure/COVERAGE_MONITORING.md)**
- How to view coverage reports (terminal, HTML, JSON)
- Mandatory coverage thresholds configuration
- Pre-commit and CI/CD integration
- Coverage alerting setup (Codecov, Slack, badges)
- Troubleshooting and best practices

**Coverage Thresholds** (Enforced):
```json
{
  "branches": 84,
  "functions": 80,
  "lines": 90,
  "statements": 90
}
```

---

## Test Coverage by Category

### ✅ Excellently Covered (80%+)

#### Utilities (100% Coverage)
- **`src/utils/portfolio/dateHelpers.js`**: 100%
  - 33 tests covering date formatting, validation, and transformations

- **`src/utils/portfolio/portfolioCalculations.js`**: 100%
  - 68 tests covering transaction calculations, chart colors, sorting, filtering

- **`src/utils/portfolio/transactionValidation.js`**: 100%
  - Tests for transaction and dividend validation logic

- **`src/utils/currency.js`**: 100%
  - 27 tests for currency symbol and formatting

- **`src/utils/numberFormat.js`**: 100%
  - 30 tests for number formatting functions

#### Hooks (89-94% Coverage)

**Portfolio Hooks** (94% average):
- **`src/hooks/portfolio/useDividendManagement.js`**: 94.89%
  - 26 tests covering dividend CRUD, cash vs stock workflows, validation

- **`src/hooks/portfolio/useFundPricing.js`**: 100%
  - 21 tests covering price caching, lookup, and cache management

- **`src/hooks/portfolio/useTransactionManagement.js`**: 95.74%
  - 24 tests covering transaction CRUD, price lookup, error handling

- **`src/hooks/portfolio/usePortfolioData.js`**: 75%
  - 11 tests (6 skipped) covering parallel data fetching

**General Hooks** (89% average):
- **`src/hooks/useApiState.js`**: 94.59%
  - 18 tests (3 skipped due to React 19) covering API state management

- **`src/hooks/useChartData.js`**: 85.71%
  - 31 tests covering progressive chart data loading and zoom handling

- **`src/hooks/useNumericInput.js`**: 100%
  - 17 tests covering European number format parsing

#### Context Providers (87% average)
- **`src/context/AppContext.js`**: 84.52%
- **`src/context/FormatContext.js`**: 93.1%

#### Shared Components (79% average)
- **`src/components/shared/`**: 79.06%
  - DataTable, FormModal, ErrorMessage, LoadingSpinner, ActionButtons

### 🚫 Excluded from Coverage

These files are intentionally excluded as they're covered by E2E tests or are infrastructure:

- **Pages** (`src/pages/**`) - 8 files covered by Playwright E2E tests
- **Config** (`src/config.js`, `src/utils/api.js`) - Infrastructure configuration
- **UI Components**:
  - ValueChart.js (1441 lines) - Complex charting, covered by E2E
  - PortfolioActions.js - CRUD UI, covered by E2E
  - TransactionsTable.js, DividendsTable.js - Display tables, covered by E2E
  - Navigation, StatusTab, FilterPopup - Layout UI, covered by E2E
- **Index files** (`**/index.js`) - Barrel files

---

## Testing Patterns

### 1. Pure Function Testing
**Target**: 100% coverage

```javascript
describe('Utility Name', () => {
  test('handles normal case', () => {
    expect(calculate(10, 5)).toBe(50);
  });

  test('handles edge cases', () => {
    expect(calculate(0, 100)).toBe(0);
    expect(calculate(null)).toBe(0);
  });
});
```

**Pattern**: Focus on edge cases (null, undefined, empty, extreme values)

### 2. React Hook Testing
**Target**: 70-80% coverage

```javascript
import { renderHook, act } from '@testing-library/react';

describe('useCustomHook', () => {
  test('initializes with correct state', () => {
    const { result } = renderHook(() => useCustomHook());
    expect(result.current.value).toBe(initial);
  });

  test('updates state', () => {
    const { result } = renderHook(() => useCustomHook());
    act(() => {
      result.current.setValue(newValue);
    });
    expect(result.current.value).toBe(newValue);
  });
});
```

**Pattern**: Mock dependencies, test state changes and side effects

### 3. Component Testing
**Target**: 60-70% coverage

```javascript
import { render, screen, fireEvent } from '@testing-library/react';

describe('Component', () => {
  test('renders correctly', () => {
    render(<Component prop="value" />);
    expect(screen.getByText('value')).toBeInTheDocument();
  });

  test('handles user interaction', () => {
    const onClick = jest.fn();
    render(<Component onClick={onClick} />);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalled();
  });
});
```

**Pattern**: Test user behavior, not implementation details

---

## Test Files Created

### Phase 1: Utilities (5 files, 160+ tests)
1. `src/utils/__tests__/api.test.js` - 295 lines (18 tests)
2. `src/utils/portfolio/__tests__/dateHelpers.test.js` - 284 lines (33 tests)
3. `src/utils/portfolio/__tests__/portfolioCalculations.test.js` - 914 lines (68 tests)
4. `src/utils/__tests__/currency.test.js` - 27 tests
5. `src/utils/__tests__/numberFormat.test.js` - 30 tests

### Phase 2: Hooks (7 files, ~150 tests)
6. `src/hooks/portfolio/__tests__/useTransactionManagement.test.js` - 492 lines (24 tests)
7. `src/hooks/portfolio/__tests__/usePortfolioData.test.js` - 399 lines (11 tests, 6 skipped)
8. `src/hooks/portfolio/__tests__/useDividendManagement.test.js` - 709 lines (26 tests)
9. `src/hooks/portfolio/__tests__/useFundPricing.test.js` - 418 lines (21 tests)
10. `src/hooks/__tests__/useChartData.test.js` - 680 lines (31 tests)
11. `src/hooks/__tests__/useApiState.test.js` - 18 tests (3 skipped)
12. `src/hooks/__tests__/useNumericInput.test.js` - 17 tests

### Phase 3: Components & Context (11 files, ~172 tests)

**Shared Components** (6 files, 73 tests):
13. `src/components/__tests__/Modal.test.js` - 346 lines
14. `src/components/__tests__/Toast.test.js` - 193 lines
15. `src/components/shared/__tests__/DataTable.test.js` - 653 lines
16. `src/components/shared/__tests__/ErrorMessage.test.js` - 285 lines
17. `src/components/shared/__tests__/FormModal.test.js` - 605 lines
18. `src/components/shared/__tests__/LoadingSpinner.test.js` - 188 lines

**Portfolio Components** (3 files, 66 tests):
19. `src/components/portfolio/__tests__/PortfolioSummary.test.js` - 224 lines (23 tests)
20. `src/components/portfolio/__tests__/PortfolioChart.test.js` - 258 lines (13 tests)
21. `src/components/portfolio/__tests__/FundsTable.test.js` - 346 lines (30 tests)

**Context Providers** (2 files, 33 tests):
22. `src/context/__tests__/FormatContext.test.js` - 384 lines (23 tests)
23. `src/context/__tests__/AppContext.test.js` - 286 lines (10 tests)

### Phase 4: E2E Tests (7 files, ~45+ tests)

**Playwright E2E Tests**:
24. `e2e/smoke.spec.js` - Basic smoke tests (6 tests)
25. `e2e/navigation.spec.js` - Navigation flow tests (11 tests)
26. `e2e/health-check.spec.js` - Backend health verification (3 tests)
27. `e2e/portfolio-management.spec.js` - Portfolio CRUD workflows
28. `e2e/transactions.spec.js` - Transaction management flows
29. `e2e/dividends.spec.js` - Dividend tracking functionality
30. `e2e/ibkr-config.spec.js` - IBKR configuration and API payload validation (6 tests)

**Total Unit Test Code**: 6,500+ lines
**Total Unit Tests**: 584 tests (575 passing, 9 skipped)
**Total E2E Tests**: 45+ tests across critical user journeys

---

## Known Issues

### React 19 + Jest Async Timing

**Issue**: 9 tests skipped due to React 19 async state update timing:
- 3 in `useApiState.test.js` (edge cases for error clearing and return values)
- 6 in `usePortfolioData.test.js` (combined loading/error state tests)

**Root Cause**: React 19 changed async state batching. When testing edge cases, `result.current` becomes `null` after async operations.

**Impact**: Minimal - core functionality fully tested, only edge cases skipped

**Workaround**: Tests marked with `test.skip()` and documented in JSDoc with references to equivalent passing tests

---

## CI/CD Integration

### Pre-commit Hooks
Tests run automatically before commits (if configured in `.pre-commit-config.yaml`):
```yaml
- repo: local
  hooks:
    - id: jest-tests
      name: Frontend Unit Tests
      entry: bash -c 'cd frontend && npm test -- --testPathIgnorePatterns=e2e'
      language: system
      pass_filenames: false
```

### GitHub Actions
Tests run on every push/PR:
```yaml
- name: Run tests with coverage
  run: |
    cd frontend
    npm ci
    npm test -- --testPathIgnorePatterns=e2e
```

**PR Merge Requirements**:
- ✅ All tests pass
- ✅ Coverage thresholds met (90% lines, 90% statements, 84% branches, 80% functions)
- ✅ Linting passes

---

## Best Practices

### 1. Document All Test Files
Every test file MUST include JSDoc comments:
```javascript
/**
 * @fileoverview Test suite for [feature name]
 *
 * Tests [functionality] including:
 * - [Key test area 1]
 * - [Key test area 2]
 * - Edge cases: null, undefined, empty, extreme values
 *
 * Total: X tests
 */
```

### 2. Test Edge Cases
Always test:
- Null and undefined inputs
- Empty strings/arrays/objects
- Zero and negative numbers
- Very large numbers
- Boundary conditions

### 3. Use Descriptive Test Names
```javascript
// Good
test('formats Euro currency with European number formatting')
test('validates stock dividend requires buy order date for future orders')

// Bad
test('works')
test('currency')
```

### 4. Mock External Dependencies
```javascript
// Mock API
jest.mock('../utils/api');

// Mock window methods
global.window.confirm = jest.fn(() => true);
global.window.alert = jest.fn();

// Mock console
global.console.error = jest.fn();
```

### 5. Use act() for State Updates
```javascript
// Synchronous
act(() => {
  result.current.setValue(newValue);
});

// Asynchronous
await act(async () => {
  await result.current.fetchData();
});
```

---

## Troubleshooting

### "Coverage threshold not met"
1. Run `npm test` to see current coverage
2. Identify files below threshold
3. Add tests OR exclude from coverage if UI/infrastructure

### "Cannot read properties of null"
- Use `act()` for all state updates
- Add `await` for async operations
- Check context provider wrapping

### Tests fail in CI but pass locally
- Ensure Node versions match
- Check for timing-dependent tests
- Verify all dependencies in package.json

---

## Related Documentation

- **Main Testing Guide**: [docs/TESTING.md](../../../docs/TESTING.md)
- **Jest Documentation**: https://jestjs.io/docs/getting-started
- **React Testing Library**: https://testing-library.com/docs/react-testing-library/intro/
- **Backend Tests**: [backend/tests/docs/README.md](../../../backend/tests/docs/README.md)

---

## Contributing

When adding new tests:

1. **Follow existing patterns** - Check similar test files for structure
2. **Document thoroughly** - JSDoc comments required
3. **Test edge cases** - Don't just test happy paths
4. **Maintain coverage** - Keep above mandatory thresholds
5. **Update documentation** - Add your test file to this README

---

**Version**: 1.3.5+
**Last Updated**: 2025-12-18
**Maintainer**: @ndewijer
