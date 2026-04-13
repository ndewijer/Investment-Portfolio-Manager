# Testing Framework Guide

This guide documents the testing framework and provides guidance for adding new tests.

---

## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
- [Test Structure](#test-structure)
- [Test Helpers and Utilities](#test-helpers-and-utilities)
- [Running Tests](#running-tests)
- [Writing New Tests](#writing-new-tests)
- [Best Practices](#best-practices)
- [Examples](#examples)

---

## Overview

The Investment Portfolio Manager uses **Go's built-in `testing` package** for backend testing, chosen for its:

- **Zero Dependencies**: Part of the Go standard library
- **Speed**: Tests run fast with in-memory SQLite databases
- **Isolation**: Each test gets its own database instance via `testutil.SetupTestDB()`
- **Subtests**: `t.Run()` for organized, hierarchical test groups
- **Clear output**: Built-in verbose and short modes

### Coverage

- **Handler Tests**: HTTP handler tests covering all route groups
  - Portfolio handlers: comprehensive CRUD + archive/unarchive + fund management
  - Fund handlers: CRUD + price updates + symbol lookup + usage checking
  - Transaction handlers: CRUD + portfolio-specific queries
  - Dividend handlers: CRUD + portfolio/fund-specific queries
  - IBKR handlers: config management + import + inbox + allocation + dividend matching
  - Developer handlers: logs + settings + CSV import + exchange rates + fund prices
  - System handlers: health check + version

- **Service Tests**: Business logic coverage across all major services
  - Portfolio service: CRUD, archive, fund management, summary, history
  - Fund service: CRUD, price updates, usage checking, symbol lookup
  - Transaction service: CRUD, gain/loss calculation, portfolio queries
  - Dividend service: CRUD, portfolio/fund queries, reinvestment
  - IBKR services: config, flex client, transaction processing, allocation
  - Materialized service: cache management, invalidation, coverage detection
  - Developer service: logging, exchange rates, CSV import

- **Critical Bugs Found Through Testing**: 5 bugs
  - ReinvestmentStatus enum vs string mismatch
  - Dividend share calculation error
  - Cost basis calculation error
  - Validation bypassing with zero values
  - UNIQUE constraint errors with invalid cache

### Goals

- **Prevent regressions**: Catch performance/functionality regressions early
- **Document behavior**: Tests serve as executable documentation
- **Enable refactoring**: Confidence to improve code
- **Comprehensive coverage**: Maintain high test coverage across the codebase
- **No mocks**: Tests use real SQLite databases (in-memory, per-test) for realistic behavior

---

## Getting Started

### Prerequisites

- Go 1.26+
- Make (for convenience targets)

### Running Tests

```bash
# From project root using Make
make test              # Run all tests with race detector
make test-short        # Skip slow tests
make test-verbose      # Verbose output

# From backend directory using Go directly
cd backend
go test ./...                              # Run all tests
go test -race ./...                        # With race detector
go test ./internal/service/...             # Specific package
go test -run TestCreatePortfolio ./...     # Specific test
go test -v ./internal/api/handlers/...    # Verbose for handlers
```

### Coverage

```bash
# Using Make targets
make coverage              # Run tests with coverage summary
make coverage-html         # Generate HTML coverage report
make coverage-func         # Per-function coverage
make coverage-by-file      # Per-file coverage sorted by %
make coverage-gaps         # Files below 100% coverage
make coverage-threshold    # Check coverage meets 75% threshold (CI)

# Using Go directly
cd backend
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out
```

---

## Test Structure

### Directory Layout

```
backend/
├── internal/
│   ├── api/
│   │   └── handlers/
│   │       ├── portfolios.go
│   │       ├── portfolios_test.go       # Handler tests (external _test package)
│   │       ├── funds.go
│   │       ├── funds_test.go
│   │       ├── transactions.go
│   │       ├── transactions_test.go
│   │       ├── dividends.go
│   │       ├── dividends_test.go
│   │       ├── ibkr.go
│   │       ├── ibkr_test.go
│   │       ├── developer.go
│   │       ├── developer_test.go
│   │       ├── system.go
│   │       ├── system_test.go
│   │       └── shared_test.go           # Internal test helpers
│   ├── service/
│   │   ├── portfolio_service.go
│   │   ├── portfolio_service_test.go
│   │   ├── fund_service.go
│   │   ├── fund_service_test.go
│   │   ├── transaction_service.go
│   │   ├── transaction_service_test.go
│   │   ├── dividend_service.go
│   │   ├── dividend_service_test.go
│   │   ├── ibkr_config_service.go
│   │   ├── ibkr_config_service_test.go
│   │   ├── ibkr_transaction_service.go
│   │   ├── ibkr_transaction_service_test.go
│   │   ├── materialized_service.go
│   │   ├── materialized_service_test.go
│   │   └── ...
│   └── testutil/
│       ├── database.go                  # Test database setup (in-memory SQLite)
│       ├── factories.go                 # Builder pattern for test data
│       ├── helpers.go                   # Service factory helpers
│       └── http_helpers.go              # HTTP request/response test helpers
└── Makefile                             # Test and coverage targets
```

### Test Organization

Tests are organized by:

1. **Package**: Tests live alongside the code they test (`_test.go` suffix)
2. **Function**: `TestFeatureName_SpecificBehavior`
3. **Subtests**: `t.Run("description", func(t *testing.T) { ... })` for grouping related cases

**Example**:
```go
// internal/api/handlers/portfolios_test.go
func TestPortfolioHandler_GetAllPortfolios(t *testing.T) {
    t.Run("returns empty list when no portfolios exist", func(t *testing.T) {
        // ...
    })

    t.Run("returns all portfolios", func(t *testing.T) {
        // ...
    })
}

func TestPortfolioHandler_GetPortfolio(t *testing.T) {
    t.Run("returns portfolio by ID", func(t *testing.T) {
        // ...
    })

    t.Run("returns 404 for non-existent portfolio", func(t *testing.T) {
        // ...
    })
}
```

---

## Test Helpers and Utilities

Test utilities are defined in `backend/internal/testutil/`.

### `SetupTestDB(t)` — In-Memory Database

**Purpose**: Creates an isolated in-memory SQLite database for each test with the full production schema applied.

**Usage**:
```go
func TestSomething(t *testing.T) {
    db := testutil.SetupTestDB(t)
    // db is automatically cleaned up when the test ends via t.Cleanup()
}
```

**Key Properties**:
- Uses in-memory SQLite for speed
- Full production schema created for each test
- Automatic cleanup via `t.Cleanup()`
- Complete test isolation — no shared state between tests

### Builder Pattern Factories

**Purpose**: Fluent builders for creating test data with sensible defaults.

**Usage**:
```go
// Create a portfolio with defaults
portfolio := testutil.NewPortfolio().Build(t, db)

// Create a portfolio with custom values
portfolio := testutil.NewPortfolio().
    WithName("Custom Portfolio").
    WithDescription("Testing").
    Archived().
    Build(t, db)

// Create a fund with custom values
fund := testutil.NewFund().
    WithSymbol("AAPL").
    WithCurrency("USD").
    Build(t, db)

// Create a transaction
transaction := testutil.NewTransaction().
    WithPortfolioFundID(pf.ID).
    WithType("buy").
    WithShares(100).
    WithCostPerShare(150.00).
    Build(t, db)
```

**Benefits**:
- Self-documenting test setup
- Sensible defaults for optional fields
- Readable at a glance
- Consistent test data creation across all test files

### Service Factory Helpers

**Purpose**: Convenience functions to create fully-wired service instances for testing.

**Usage**:
```go
db := testutil.SetupTestDB(t)
ps := testutil.NewTestPortfolioService(t, db)
fs := testutil.NewTestFundService(t, db)
ms := testutil.NewTestMaterializedService(t, db)
```

### HTTP Request Helpers

**Purpose**: Helpers for creating HTTP requests with Chi URL parameters and parsing responses.

**Usage**:
```go
// Create a request with URL parameters (Chi-compatible)
req := testutil.NewRequestWithURLParams(
    http.MethodGet,
    "/api/portfolio/"+id,
    map[string]string{"portfolioId": id},
)

// Assert row counts in the database
testutil.AssertRowCount(t, db, "portfolio", 1)
```

---

## Running Tests

### Basic Commands

```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Run tests skipping slow tests (tagged with t.Skip for short mode)
make test-short

# Run specific test file
cd backend && go test ./internal/service/ -run TestPortfolioService

# Run specific test function
cd backend && go test ./internal/api/handlers/ -run TestPortfolioHandler_GetAllPortfolios

# Run specific subtest
cd backend && go test ./internal/api/handlers/ -run "TestPortfolioHandler_GetAllPortfolios/returns_empty_list"

# Run all handler tests
cd backend && go test ./internal/api/handlers/...

# Run all service tests
cd backend && go test ./internal/service/...
```

### Coverage Reports

```bash
# Run with coverage summary
make coverage

# Generate HTML coverage report
make coverage-html
open backend/coverage.html

# Per-function coverage
make coverage-func

# Show files below 100% coverage
make coverage-gaps

# Check coverage meets 75% threshold
make coverage-threshold
```

### Race Detector

The default `make test` target includes the `-race` flag to detect data races:

```bash
make test  # Includes -race by default
```

---

## Writing New Tests

### Handler Test Template

```go
func TestNewHandler_Endpoint(t *testing.T) {
    // WHY: Verify that the endpoint correctly handles [scenario]

    t.Run("returns expected result for valid input", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := testutil.NewTestSomeService(t, db)
        handler := handlers.NewSomeHandler(svc)

        // Create test data
        item := testutil.NewItem().WithName("Test").Build(t, db)

        // Build request
        req := testutil.NewRequestWithURLParams(
            http.MethodGet,
            "/api/items/"+item.ID,
            map[string]string{"itemId": item.ID},
        )
        w := httptest.NewRecorder()

        // Execute
        handler.GetItem(w, req)

        // Assert
        assert.Equal(t, http.StatusOK, w.Code)

        var result model.Item
        err := json.NewDecoder(w.Body).Decode(&result)
        require.NoError(t, err)
        assert.Equal(t, "Test", result.Name)
    })

    t.Run("returns 404 for non-existent item", func(t *testing.T) {
        db := testutil.SetupTestDB(t)
        svc := testutil.NewTestSomeService(t, db)
        handler := handlers.NewSomeHandler(svc)

        req := testutil.NewRequestWithURLParams(
            http.MethodGet,
            "/api/items/"+testutil.MakeID(),
            map[string]string{"itemId": testutil.MakeID()},
        )
        w := httptest.NewRecorder()

        handler.GetItem(w, req)

        assert.Equal(t, http.StatusNotFound, w.Code)
    })
}
```

### Service Test Template

```go
func TestSomeService_Method(t *testing.T) {
    t.Run("performs expected operation", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := testutil.NewTestSomeService(t, db)

        // Create test data
        item := testutil.NewItem().WithName("Test").Build(t, db)

        // Execute
        result, err := svc.GetItem(context.Background(), item.ID)

        // Assert
        require.NoError(t, err)
        assert.Equal(t, "Test", result.Name)
    })

    t.Run("returns error for non-existent item", func(t *testing.T) {
        db := testutil.SetupTestDB(t)
        svc := testutil.NewTestSomeService(t, db)

        _, err := svc.GetItem(context.Background(), testutil.MakeID())

        assert.Error(t, err)
        assert.True(t, errors.Is(err, apperrors.ErrItemNotFound))
    })
}
```

### Table-Driven Test Template

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name           string
        input          string
        expectedStatus int
        expectedError  string
    }{
        {"empty ID", "", http.StatusBadRequest, "ID is required"},
        {"invalid UUID", "not-a-uuid", http.StatusBadRequest, "invalid UUID format"},
        {"non-existent", testutil.MakeID(), http.StatusNotFound, "not found"},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            db := testutil.SetupTestDB(t)
            svc := testutil.NewTestSomeService(t, db)
            handler := handlers.NewSomeHandler(svc)

            req := testutil.NewRequestWithURLParams(
                http.MethodGet,
                "/api/items/"+tc.input,
                map[string]string{"itemId": tc.input},
            )
            w := httptest.NewRecorder()

            handler.GetItem(w, req)

            assert.Equal(t, tc.expectedStatus, w.Code)
        })
    }
}
```

---

## Best Practices

### 1. Test Naming

**Good**:
```go
func TestPortfolioService_DeletePortfolio_ReturnsErrorWhenFundsAttached(t *testing.T) {}

func TestTransactionHandler_CreateTransaction_ValidatesCostPerShare(t *testing.T) {}
```

**Bad**:
```go
func TestDelete(t *testing.T) {}    // Too vague

func TestFunc1(t *testing.T) {}     // Meaningless name
```

### 2. WHY Comments

Document the purpose of test groups with WHY comments:

```go
func TestPortfolioHandler_ArchivePortfolio(t *testing.T) {
    // WHY: Archiving should set is_archived=true and still allow read access
    // but prevent new transactions from being added.

    t.Run("archives portfolio successfully", func(t *testing.T) {
        // ...
    })
}
```

### 3. One Assertion Focus Per Subtest

**Good**:
```go
t.Run("returns list of portfolios", func(t *testing.T) {
    result, err := svc.GetAllPortfolios(ctx)
    require.NoError(t, err)
    assert.Len(t, result, 2)
})

t.Run("portfolios contain expected fields", func(t *testing.T) {
    result, err := svc.GetAllPortfolios(ctx)
    require.NoError(t, err)
    assert.NotEmpty(t, result[0].ID)
    assert.NotEmpty(t, result[0].Name)
})
```

**Acceptable** (related assertions):
```go
t.Run("response structure is correct", func(t *testing.T) {
    result, err := svc.GetAllPortfolios(ctx)
    require.NoError(t, err)
    assert.Len(t, result, 2)
    assert.NotEmpty(t, result[0].ID)
    assert.NotEmpty(t, result[0].Name)
})
```

### 4. Test Edge Cases

```go
t.Run("handles empty portfolio with no transactions", func(t *testing.T) {
    // Test empty case
})

t.Run("handles missing price data gracefully", func(t *testing.T) {
    // Test missing data
})

t.Run("handles date range boundaries correctly", func(t *testing.T) {
    // Test boundaries
})
```

### 5. Use Builder Factories for Test Data

```go
// Good: Clear, self-documenting
portfolio := testutil.NewPortfolio().
    WithName("Test Portfolio").
    Archived().
    Build(t, db)

// Bad: Manual SQL inserts scattered through tests
db.Exec("INSERT INTO portfolio (id, name, is_archived) VALUES (?, ?, ?)", id, "Test", true)
```

### 6. Test Isolation

Every test gets its own database instance. Never rely on state from another test:

```go
// Good: Each subtest creates its own DB
t.Run("first test", func(t *testing.T) {
    db := testutil.SetupTestDB(t)
    testutil.NewPortfolio().Build(t, db)
    testutil.AssertRowCount(t, db, "portfolio", 1)
})

t.Run("second test starts clean", func(t *testing.T) {
    db := testutil.SetupTestDB(t)
    testutil.AssertRowCount(t, db, "portfolio", 0)
})
```

---

## Examples

### Example 1: Handler Test with HTTP Response Verification

```go
func TestPortfolioHandler_CreatePortfolio(t *testing.T) {
    t.Run("creates portfolio successfully", func(t *testing.T) {
        db := testutil.SetupTestDB(t)
        ps := testutil.NewTestPortfolioService(t, db)
        fs := testutil.NewTestFundService(t, db)
        ms := testutil.NewTestMaterializedService(t, db)
        handler := handlers.NewPortfolioHandler(ps, fs, ms)

        body := `{"name": "Test Portfolio", "description": "Testing"}`
        req := httptest.NewRequest(http.MethodPost, "/api/portfolio", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        handler.CreatePortfolio(w, req)

        assert.Equal(t, http.StatusCreated, w.Code)

        var result model.Portfolio
        err := json.NewDecoder(w.Body).Decode(&result)
        require.NoError(t, err)
        assert.Equal(t, "Test Portfolio", result.Name)
        assert.NotEmpty(t, result.ID)
    })
}
```

### Example 2: Service Test with Error Verification

```go
func TestPortfolioService_DeletePortfolio(t *testing.T) {
    t.Run("returns error when portfolio has funds", func(t *testing.T) {
        db := testutil.SetupTestDB(t)
        ps := testutil.NewTestPortfolioService(t, db)

        portfolio := testutil.NewPortfolio().Build(t, db)
        fund := testutil.NewFund().Build(t, db)
        testutil.NewPortfolioFund().
            WithPortfolioID(portfolio.ID).
            WithFundID(fund.ID).
            Build(t, db)

        err := ps.DeletePortfolio(context.Background(), portfolio.ID)

        assert.Error(t, err)
        // Portfolio should still exist
        testutil.AssertRowCount(t, db, "portfolio", 1)
    })
}
```

### Example 3: Table-Driven Validation Test

```go
func TestTransactionHandler_CreateTransaction_Validation(t *testing.T) {
    tests := []struct {
        name           string
        body           string
        expectedStatus int
    }{
        {
            name:           "missing type",
            body:           `{"portfolio_fund_id": "some-id", "date": "2024-01-15", "shares": 100}`,
            expectedStatus: http.StatusBadRequest,
        },
        {
            name:           "negative shares",
            body:           `{"portfolio_fund_id": "some-id", "date": "2024-01-15", "type": "buy", "shares": -1}`,
            expectedStatus: http.StatusBadRequest,
        },
        {
            name:           "invalid date format",
            body:           `{"portfolio_fund_id": "some-id", "date": "not-a-date", "type": "buy", "shares": 100}`,
            expectedStatus: http.StatusBadRequest,
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            db := testutil.SetupTestDB(t)
            svc := testutil.NewTestTransactionService(t, db)
            handler := handlers.NewTransactionHandler(svc)

            req := httptest.NewRequest(http.MethodPost, "/api/transaction", strings.NewReader(tc.body))
            req.Header.Set("Content-Type", "application/json")
            w := httptest.NewRecorder()

            handler.CreateTransaction(w, req)

            assert.Equal(t, tc.expectedStatus, w.Code)
        })
    }
}
```

---

## Frontend Testing

### Overview

**Framework**: Jest + React Testing Library

The frontend uses **Jest** with **React Testing Library** for testing React components, hooks, and utility functions.

### Coverage

**Overall** (Business Logic Only): 93.11% lines, 90.74% statements

**By Category**:
- **Utility Functions**: 100% coverage
- **Hooks**: 89-94% coverage (target exceeded)
- **Context Providers**: 87% coverage
- **Shared Components**: 79% coverage

**Tests**: 584 tests (575 passing, 9 skipped)

### Coverage Thresholds (Mandatory)

Tests will **fail** if coverage drops below:
- Lines: 90%
- Statements: 90%
- Branches: 84%
- Functions: 80%

These thresholds are enforced in pre-commit hooks and CI/CD pipelines.

### Running Frontend Tests

**Unit Tests (Jest + React Testing Library):**
```bash
cd frontend

# Run all unit tests with coverage
pnpm test

# Run tests in watch mode (interactive)
pnpm run test:watch

# Run with coverage report
pnpm run test:coverage

# Run tests in CI mode (non-interactive)
pnpm run test:ci

# View HTML coverage report
pnpm run test:coverage && open coverage/index.html
```

**E2E Tests (Playwright):**
```bash
cd frontend

# Run all E2E tests (headless)
pnpm run test:e2e

# Run with interactive UI
pnpm run test:e2e:ui

# Run in headed mode (see browser)
pnpm run test:e2e:headed

# Debug mode (step through tests)
pnpm run test:e2e:debug
```

### Test Structure

**Directory Layout:**
```
frontend/src/
├── components/
│   ├── ComponentName.js
│   └── __tests__/
│       └── ComponentName.test.js
├── hooks/
│   ├── useHookName.js
│   └── __tests__/
│       └── useHookName.test.js
├── utils/
│   ├── utility.js
│   └── __tests__/
│       └── utility.test.js
├── setupTests.js              # Jest setup
└── __mocks__/
    └── fileMock.js           # Mock for static assets
```

**Test File Naming:**
- Co-located with source: `__tests__/` directory next to source files
- Suffix: `.test.js` or `.spec.js`
- Name matches source: `ComponentName.test.js` for `ComponentName.js`

### Configuration

**Jest Configuration** is in `frontend/package.json`. Key settings:
- **Test Environment**: jsdom (simulates browser DOM)
- **Setup**: `setupTests.js` configures @testing-library/jest-dom matchers
- **Transforms**: babel-jest for JSX/modern JavaScript
- **Coverage Collection**: Focused on business logic only (UI components excluded)
- **Coverage Thresholds**: Mandatory minimums enforced in CI/CD

**Mandatory Coverage Thresholds:**
```json
{
  "coverageThreshold": {
    "global": {
      "branches": 84,
      "functions": 80,
      "lines": 90,
      "statements": 90
    }
  }
}
```

Tests will **fail** if coverage drops below these thresholds. See [Coverage Monitoring Guide](../frontend/tests/docs/infrastructure/COVERAGE_MONITORING.md) for details.

**Actual Coverage Achieved:**
- **Utilities**: 100% coverage
- **Hooks**: 89-94% coverage
- **Context Providers**: 87% coverage
- **Shared Components**: 79% coverage

### Writing Frontend Tests

#### Utility Function Test Template

```javascript
/**
 * @fileoverview Test suite for [utility name]
 *
 * Tests [utility purpose] including:
 * - [Key functionality 1]
 * - [Key functionality 2]
 * - Edge cases: zero, negative, null/undefined, large numbers
 *
 * Total: X tests
 */
import { functionName } from '../utility';

describe('Utility Name', () => {
  describe('functionName', () => {
    test('handles normal case', () => {
      expect(functionName(input)).toBe(expected);
    });

    test('handles edge case', () => {
      expect(functionName(edgeInput)).toBe(edgeExpected);
    });

    test('handles null/undefined', () => {
      expect(functionName(null)).toBe(defaultValue);
      expect(functionName(undefined)).toBe(defaultValue);
    });
  });
});
```

#### Custom Hook Test Template

```javascript
/**
 * @fileoverview Test suite for [hook name]
 *
 * Tests [hook purpose] that handles:
 * - [Key functionality 1]
 * - [Key functionality 2]
 * - State updates and side effects
 *
 * Total: X tests
 */
import { renderHook, act } from '@testing-library/react';
import useCustomHook from '../useCustomHook';

describe('useCustomHook', () => {
  test('initializes with correct state', () => {
    const { result } = renderHook(() => useCustomHook());

    expect(result.current.value).toBe(initialValue);
  });

  test('updates state correctly', () => {
    const { result } = renderHook(() => useCustomHook());

    act(() => {
      result.current.updateValue(newValue);
    });

    expect(result.current.value).toBe(newValue);
  });

  test('handles async operations', async () => {
    const { result } = renderHook(() => useCustomHook());

    await act(async () => {
      await result.current.asyncFunction();
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.data).toBeDefined();
  });
});
```

#### Component Test Template

```javascript
/**
 * @fileoverview Test suite for [Component Name]
 *
 * Tests [component purpose] including:
 * - Rendering with different props
 * - User interactions
 * - Error states and edge cases
 *
 * Total: X tests
 */
import { render, screen, fireEvent } from '@testing-library/react';
import ComponentName from '../ComponentName';

describe('ComponentName', () => {
  test('renders with required props', () => {
    render(<ComponentName title="Test" />);

    expect(screen.getByText('Test')).toBeInTheDocument();
  });

  test('handles user interaction', () => {
    const onClick = jest.fn();
    render(<ComponentName onClick={onClick} />);

    fireEvent.click(screen.getByRole('button'));

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  test('shows error state', () => {
    render(<ComponentName error="Error message" />);

    expect(screen.getByText('Error message')).toBeInTheDocument();
  });
});
```

### Testing Best Practices

#### 1. Document All Test Files

**All test files MUST include JSDoc comments** explaining:
- What functionality is tested
- Important behavior and edge cases
- Known issues or limitations
- Total test count

```javascript
/**
 * @fileoverview Test suite for numeric input parsing
 *
 * Important Parsing Behavior:
 * The hook treats the FIRST separator (. or ,) as the decimal point.
 * Example: "€ 1.234,56" → "1.23456" (first . becomes decimal)
 *
 * Total: 17 tests (all passing)
 */
```

#### 2. Test User Behavior, Not Implementation

**Good** (tests what user experiences):
```javascript
test('shows error when form is invalid', () => {
  render(<Form />);
  fireEvent.click(screen.getByText('Submit'));
  expect(screen.getByText('Required field')).toBeVisible();
});
```

**Bad** (tests implementation details):
```javascript
test('calls validateForm', () => {
  const { instance } = render(<Form />);
  expect(instance.validateForm).toHaveBeenCalled();
});
```

#### 3. Use act() for State Updates

```javascript
// For synchronous updates
act(() => {
  result.current.updateValue('new');
});

// For asynchronous updates
await act(async () => {
  await result.current.fetchData();
});
```

#### 4. Test Edge Cases

```javascript
describe('edge cases', () => {
  test('handles zero', () => {
    expect(calculate(0, 100)).toBe(0);
  });

  test('handles negative numbers', () => {
    expect(calculate(-10, 50)).toBe(-500);
  });

  test('handles null/undefined', () => {
    expect(formatCurrency(null)).toBe('');
    expect(formatCurrency(undefined)).toBe('');
  });

  test('handles very large numbers', () => {
    expect(calculate(1000000, 100)).toBe(100000000);
  });
});
```

#### 5. Mock External Dependencies

```javascript
// Mock API calls
jest.mock('../utils/api', () => ({
  get: jest.fn(() => Promise.resolve({ data: [] })),
  post: jest.fn(() => Promise.resolve({ data: {} })),
}));

// Mock context providers
const mockFormatContext = {
  formatCurrency: (value) => `€${value}`,
  formatNumber: (value) => value.toString(),
};

render(
  <FormatContext.Provider value={mockFormatContext}>
    <Component />
  </FormatContext.Provider>
);
```

#### 6. Use Descriptive Test Names

**Good**:
```javascript
test('formats currency with Euro symbol and European formatting')
test('parses on blur with both comma and period as decimal separators')
test('validates stock dividend with future buy order date')
```

**Bad**:
```javascript
test('works')
test('test1')
test('currency')
```

### CI/CD Integration

Frontend tests run automatically in GitHub Actions (`.github/workflows/frontend-ci.yml`):

```yaml
- name: Install dependencies
  run: pnpm install --frozen-lockfile

- name: Run tests with coverage
  run: pnpm run test:ci

- name: Run linting and formatting
  run: pnpm run check
```

**CI Requirements for PR Merge:**
- All tests pass
- Linting passes
- Coverage thresholds met (when enabled)

### Known Issues

#### React 19 + Jest Async Timing

**Issue**: 3 edge case tests in `useApiState.test.js` are skipped due to React 19 async state update timing issues:
- `clearError()` method after error state is set
- Return value from `execute()` on success
- Error throwing from `execute()` on failure

**Root Cause**: React 19 changed how async state updates are batched. When testing certain edge cases, `result.current` becomes `null` after async operations complete, causing "Cannot read properties of null" errors.

**Impact**: Minimal - these 3 edge cases are covered by other tests that verify the same functionality through different approaches. Core hook functionality (loading/error/data states, callbacks, manual updates) is fully tested with 15/18 tests passing.

**Attempted Solutions**:
1. Using `waitFor()` after `act()` - causes timing conflicts
2. Using different `act()` patterns - same issue with edge cases
3. Updating to latest React Testing Library v16.3.0 - already using latest

**Workaround**: Tests are marked with `test.skip()` and documented in JSDoc with references to equivalent passing tests.

**Example**:
```javascript
test.skip('clearError clears error state', async () => {
  // Skip due to React 19 timing issue: result.current becomes null
  // Functionality IS tested by "setData clears error" test
});
```

#### European Number Format Parsing

**Behavior**: The `useNumericInput` hook treats the FIRST separator as the decimal point:
- Input: `"€ 1.234,56"`
- Parsed: `1.23456` (NOT `1234.56`)
- First `.` becomes decimal, `,` is removed

**Tests verify this actual behavior**, not the potentially expected behavior.

### Troubleshooting

**Tests fail with "Cannot read properties of null":**
- Ensure you're using `act()` for state updates
- Check for missing `await` in async tests
- Verify context providers are wrapped correctly

**Tests fail with "window.matchMedia is not a function":**
- Already mocked in `setupTests.js`
- If still failing, check import order

**Coverage reports show unexpected uncovered lines:**
- Check `collectCoverageFrom` in package.json
- Ensure test file names match pattern: `**/__tests__/**/*.js` or `**/*.test.js`
- Run with `--coverage --verbose` to debug

**Linter fails on test files:**
- Test files require JSDoc documentation
- Biome recognizes test globals (`describe`, `test`, `expect`)
- Check `biome.json` configuration

### Detailed Frontend Test Documentation

For comprehensive information about frontend testing, including:
- Complete test file inventory (29 test files, 584+ unit tests, 40+ E2E tests)
- Coverage breakdown by category
- Testing patterns and best practices
- Known issues (React 19 async timing)
- Coverage monitoring and alerting setup
- Troubleshooting guides

**See**: [frontend/tests/docs/README.md](../frontend/tests/docs/README.md)

### Related Documentation

- **Internal**: [Frontend Test Coverage Summary](../frontend/tests/docs/UNIT_TEST_COVERAGE_SUMMARY.md)
- **Internal**: [Coverage Monitoring Guide](../frontend/tests/docs/infrastructure/COVERAGE_MONITORING.md)
- **External**: [Jest Documentation](https://jestjs.io/docs/getting-started)
- **External**: [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/)
- **External**: [Testing Library Queries](https://testing-library.com/docs/queries/about/)
- **External**: [Jest DOM Matchers](https://github.com/testing-library/jest-dom)
- **External**: [Playwright Documentation](https://playwright.dev/docs/intro)

---

## Docker Integration Testing

### Overview

**Purpose**: Prevent production deployment failures by testing Docker container builds, startup, and integration

The Docker integration test workflow validates:
- Container builds succeed
- Services start without errors
- Health endpoints respond correctly
- Frontend-backend networking works
- Custom container name configurations work

### Running Docker Tests Locally

**Automated Test Script (Recommended):**
```bash
# Run the complete Docker integration test suite
./scripts/test-docker-integration.sh
```

This script automatically:
- Builds Docker images
- Starts containers
- Tests backend health (from inside container)
- Verifies backend response
- Tests frontend static content serving
- Tests frontend-backend proxy integration
- Cleans up containers on exit

**Manual Testing:**
```bash
# Run all Docker integration tests manually
docker compose build
docker compose up -d

# Test backend directly (from inside container)
docker compose exec backend wget -qO- http://localhost:5000/api/system/health

# Test frontend serves static content
curl http://localhost/

# Test API proxy through frontend (production architecture)
curl http://localhost/api/system/health
curl http://localhost/api/system/version

# Check logs
docker compose logs backend
docker compose logs frontend

# Cleanup
docker compose down -v
```

### Architecture Note

**IMPORTANT**: The backend port 5000 is NOT exposed to the host. The backend is only accessible:
- From inside the Docker network (via `docker compose exec`)
- Through the frontend Nginx proxy at `http://localhost/api/`

This is the production architecture and what the tests verify.

### CI/CD Integration

Docker tests run automatically in GitHub Actions (`.github/workflows/docker-test.yml`) when:
- Dockerfiles are modified
- docker-compose.yml changes
- Go module dependencies change (go.mod, go.sum)
- Nginx configuration changes

**Test Jobs:**
1. **docker-integration-test**: Tests default configuration
2. **docker-custom-hostname-test**: Tests custom BACKEND_HOST

### Test Coverage

#### Container Build
- Backend builds successfully with Go multi-stage build
- Frontend builds successfully with Node 24
- Multi-stage builds optimize image size
- Dependencies install correctly

#### Container Startup
- Backend starts and creates database
- Frontend starts and serves static files
- Services connect via Docker network
- Auto-generated INTERNAL_API_KEY works

#### Health Checks
- Backend health endpoint returns 200
- Backend health response includes database status
- Backend version endpoint returns version info
- Health checks complete within 60 seconds

#### Frontend-Backend Integration
- Frontend serves index page
- Frontend proxies /api/ requests to backend
- Nginx template substitution works correctly
- API responses return through proxy

#### Custom Configuration
- Custom BACKEND_HOST values work
- Custom container names work correctly
- Environment variable substitution works

### Hybrid Testing Approach

The Docker integration test workflow uses a **three-layer testing approach**:

**Step 1: Backend Health (Internal)**
- Tests backend independently from inside container network
- Uses `docker compose exec -T backend wget -qO- http://localhost:5000/api/system/health`
- Catches backend-specific failures (database, startup, configuration)
- Verifies backend is healthy before testing integration

**Step 2: Frontend Serves Static Content**
- Tests frontend container independently from host
- Uses `curl http://localhost/`
- Catches frontend failures (Nginx configuration, static files)
- Verifies frontend container is running

**Step 3: Frontend-Backend Integration (API Proxy)**
- Tests production architecture end-to-end
- Uses `curl http://localhost/api/system/health`
- Catches networking issues (proxy configuration, DNS resolution)
- Verifies complete stack works together

This approach ensures failures can be isolated to specific layers, making debugging faster and more accurate.

### Testing Custom Configurations

**Test custom container names:**
```bash
cat > docker-compose.override.yml <<EOF
services:
  backend:
    container_name: custom-backend-name
  frontend:
    build:
      args:
        - BACKEND_HOST=custom-backend-name
    environment:
      - BACKEND_HOST=custom-backend-name
EOF

docker compose build
docker compose up -d
curl http://localhost/api/system/health
docker compose down -v
```

**Test custom domain:**
```bash
export DOMAIN=test.local
export USE_HTTPS=false

docker compose build
docker compose up -d
curl http://localhost/api/system/version
docker compose down -v
```

### Troubleshooting Docker Tests

**Backend fails health check:**
- Check backend logs: `docker compose logs backend`
- Verify database directory permissions
- Ensure INTERNAL_API_KEY is generated
- Test backend from inside container: `docker compose exec backend wget -qO- http://localhost:5000/api/system/health`

**Frontend proxy fails:**
- Check BACKEND_HOST environment variable
- Verify nginx template substitution in logs: `docker compose logs frontend`
- Check network connectivity between containers: `docker network inspect investment-portfolio-manager_app-network`
- Test frontend serves static files: `curl http://localhost/`

**Containers won't start:**
- Check port conflicts: `lsof -i :80` (backend port 5000 is internal only)
- Verify Docker resources (memory/disk)
- Check build logs: `docker compose build --progress=plain`
- Clean Docker cache: `docker system prune -a`

**Tests pass locally but fail in CI:**
- Check GitHub Actions logs for specific error
- Verify jq is available (used for JSON parsing)
- Check timeout values (60 seconds for health checks)
- Ensure no port conflicts in CI environment

### Best Practices

1. **Test before deploying**: Always run Docker tests before production deployment
2. **Test custom configs**: If using custom container names, test with docker-compose.override.yml
3. **Monitor health endpoints**: Use /api/system/health for monitoring
4. **Check logs on failure**: Always examine container logs when tests fail
5. **Clean between runs**: Use `docker compose down -v` to remove volumes

### Related Documentation

- [Docker Configuration Guide](DOCKER.md)
- [CI/CD Workflows](.github/workflows/docker-test.yml)

---

## Troubleshooting

### Issue: Import Errors (Backend)

**Problem**: Package not found or compilation errors

**Solution**:
- Ensure you're running from the correct directory
- Run `go mod tidy` to clean up dependencies
- Check that the module path is correct in `go.mod`

### Issue: Database Schema Errors

**Problem**: `no such table` or schema-related errors in tests

**Solution**:
```bash
# Test database setup creates schema automatically via testutil.SetupTestDB()
# If schema is out of date, check internal/testutil/database.go
cd backend
go test ./internal/testutil/ -v
```

### Issue: Tests Hang

**Problem**: Tests don't complete

**Solution**:
- Check for infinite loops or blocking operations
- Use the `-timeout` flag to set a maximum test duration:
  ```bash
  go test -timeout 30s ./...
  ```
- Check for unclosed database connections

### Issue: Race Conditions

**Problem**: Tests fail intermittently with race detector

**Solution**:
- Always run with `-race` flag during development: `go test -race ./...`
- Ensure no shared mutable state between tests
- Use `testutil.SetupTestDB(t)` for isolated database instances per test

---

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD) or alongside implementation
2. **Test happy path and edge cases**
3. **Use builder factories** for test data setup
4. **Include WHY comments** explaining test purpose
5. **Maintain test coverage** above 75% (enforced in CI)
6. **Use table-driven tests** for validation and edge case scenarios

---

## Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Go Test Flags](https://pkg.go.dev/cmd/go/internal/test)
- [testify/assert](https://pkg.go.dev/github.com/stretchr/testify/assert) - Assertion library
- [testify/require](https://pkg.go.dev/github.com/stretchr/testify/require) - Fatal assertion library
- [httptest](https://pkg.go.dev/net/http/httptest) - HTTP testing utilities

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
