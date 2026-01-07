# Testing Framework Guide

This guide documents the pytest testing framework and provides guidance for adding new tests.

---

## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
- [Test Structure](#test-structure)
- [Available Fixtures](#available-fixtures)
- [Running Tests](#running-tests)
- [Writing New Tests](#writing-new-tests)
- [Best Practices](#best-practices)
- [Examples](#examples)

---

## Overview

The Investment Portfolio Manager uses **pytest** for its testing framework, chosen for its:

- **Simplicity**: Easy to write and understand tests
- **Powerful fixtures**: Reusable test components
- **Great ecosystem**: Extensive plugin support
- **Clear output**: Readable test results

### Coverage

- **Service Tests**: Comprehensive coverage across all major services
  - **Excellent Coverage Services** (95%+):
    - `ibkr_flex_service.py`: 97% (56 tests) ✅
    - `price_update_service.py`: 98% (17 tests) ✅
    - `fund_matching_service.py`: 100% (27 tests) ✅
    - `symbol_lookup_service.py`: 100% (20 tests) ✅
    - `system_service.py`: 100% ✅
    - `ibkr_config_service.py`: 100% ✅
  - **High Coverage Services** (90%+):
    - `dividend_service.py`: 92% (21 tests) ✅
    - `portfolio_service.py`: 91% ✅
    - `transaction_service.py`: 94% (26 tests) ✅
    - `logging_service.py`: 96% ✅
    - `fund_service.py`: 97% ✅
  - **Target Coverage Services** (85%+):
    - `ibkr_transaction_service.py`: 89% (40 tests) ✅
    - `developer_service.py`: 99% ✅

- **Route Tests**: Complete coverage of all route error paths
  - `ibkr_routes.py`: 100% ✅
  - `portfolio_routes.py`: 100% ✅
  - `transaction_routes.py`: 100% ✅
  - `dividend_routes.py`: 100% ✅
  - `system_routes.py`: 100% ✅
  - `fund_routes.py`: 96% ✅
  - `developer_routes.py`: 91% ✅

- **Critical Bugs Fixed Through Testing**: 5 bugs
  - ReinvestmentStatus enum vs string mismatch
  - Dividend share calculation error
  - Cost basis calculation error
  - Validation bypassing with zero values
  - UNIQUE constraint errors with invalid cache

**Overall Test Coverage**: 656+ tests across routes and services, **95.20%** average coverage

### Goals

- **Prevent regressions**: Catch performance/functionality regressions early
- **Document behavior**: Tests serve as executable documentation
- **Enable refactoring**: Confidence to improve code
- **Comprehensive coverage**: Maintain high test coverage across the codebase

---

## Getting Started

### Prerequisites

- Python 3.13+
- Virtual environment activated
- pytest and dependencies installed

### Installation

```bash
cd backend
source .venv/bin/activate

# Install test dependencies
pip install -r requirements.txt

# Verify installation
pytest --version
```

### Dependencies

```
pytest==8.4.2           # Test framework
pytest-flask==1.3.0     # Flask integration
pytest-cov==7.0.0       # Coverage reporting
```

---

## Test Structure

### Directory Layout

```
backend/
├── app/                    # Application code
│   ├── models.py
│   ├── routes/
│   └── services/
├── tests/                  # Test suite
│   ├── __init__.py        # Package marker
│   ├── conftest.py        # Shared fixtures
│   ├── services/          # Service layer tests
│   │   ├── test_dividend_service.py
│   │   ├── test_fund_matching_service.py
│   │   ├── test_fund_service.py
│   │   ├── test_ibkr_config_service.py
│   │   ├── test_ibkr_flex_service.py
│   │   ├── test_ibkr_transaction_service.py
│   │   ├── test_logging_service.py
│   │   ├── test_portfolio_service.py
│   │   ├── test_price_update_service.py
│   │   ├── test_symbol_lookup_service.py
│   │   └── test_transaction_service.py
│   └── docs/              # Test documentation
│       ├── README.md      # Documentation index
│       ├── services/      # Service test documentation
│       │   ├── DIVIDEND_SERVICE_TESTS.md
│       │   ├── FUND_MATCHING_SERVICE_TESTS.md
│       │   ├── FUND_SERVICE_TESTS.md
│       │   ├── IBKR_CONFIG_SERVICE_TESTS.md
│       │   ├── IBKR_FLEX_SERVICE_TESTS.md
│       │   ├── IBKR_TRANSACTION_SERVICE_TESTS.md
│       │   ├── PORTFOLIO_SERVICE_TESTS.md
│       │   ├── PRICE_UPDATE_SERVICE_TESTS.md
│       │   ├── SYMBOL_LOOKUP_SERVICE_TESTS.md
│       │   └── TRANSACTION_SERVICE_TESTS.md
│       ├── phases/        # Development phase documentation
│       │   ├── BUG_FIXES_1.3.3.md
│       │   └── PHASE_3_SUMMARY.md
│       └── infrastructure/ # Testing infrastructure docs
│           ├── PORTFOLIO_PERFORMANCE_TESTS.md
│           └── TESTING_INFRASTRUCTURE.md
├── pytest.ini             # Pytest configuration
└── requirements.txt       # Dependencies
```

### Test Organization

Tests are organized by:

1. **Category**: Services, Routes, Models (in subdirectories)
2. **Module**: `test_<module_name>.py`
3. **Class**: `Test<FeatureName>` (optional grouping)
4. **Function**: `test_<specific_behavior>`

**Example**:
```python
# tests/services/test_portfolio_service.py
class TestPortfolioHistoryPerformance:
    def test_get_portfolio_history_query_count(self):
        pass

    def test_get_portfolio_history_execution_time(self):
        pass

class TestPortfolioHistoryCorrectness:
    def test_portfolio_history_returns_data(self):
        pass
```

---

## Available Fixtures

Fixtures are reusable components defined in `tests/conftest.py`.

### `app` (session scope)

**Purpose**: Flask application instance for testing

**Scope**: Session - created once per test run

**Usage**:
```python
def test_something(app):
    assert app.config["TESTING"] == True
```

**Configuration**:
- Uses existing database (can be configured for test DB)
- TESTING mode enabled
- Same configuration as production app

### `client` (session scope)

**Purpose**: Test client for HTTP requests

**Scope**: Session - created once per test run

**Usage**:
```python
def test_api_endpoint(client):
    response = client.get('/api/portfolio')
    assert response.status_code == 200
```

**Use for**:
- Integration tests
- API endpoint testing
- End-to-end request/response validation

### `app_context` (function scope)

**Purpose**: Application context for database operations

**Scope**: Function - created for each test

**Usage**:
```python
def test_database_query(app_context):
    from app.models import Portfolio
    portfolios = Portfolio.query.all()
    assert len(portfolios) > 0
```

**Required for**:
- Database queries
- Service layer calls
- Any code requiring Flask context

### `query_counter` (function scope)

**Purpose**: Counts SQL queries executed during test

**Scope**: Function - created for each test

**Attributes**:
- `count` (int): Current query count
- `queries` (list): List of SQL statements
- `reset()`: Reset counter to zero

**Usage**:
```python
def test_query_efficiency(app_context, query_counter):
    query_counter.reset()

    # Execute code
    result = PortfolioService.get_portfolio_history()

    # Verify query count
    print(f"Queries: {query_counter.count}")
    assert query_counter.count < 100
```

**Use for**:
- Performance tests
- Query optimization validation
- Regression prevention

### `timer` (function scope)

**Purpose**: Measures execution time

**Scope**: Function - created for each test

**Methods**:
- `start()`: Start timer
- `stop()`: Stop timer and return elapsed seconds
- `elapsed` (property): Current elapsed time

**Usage**:
```python
def test_performance(app_context, timer):
    timer.start()

    # Execute code
    result = PortfolioService.get_portfolio_history()

    elapsed = timer.stop()
    print(f"Time: {elapsed:.3f}s")
    assert elapsed < 1.0
```

**Use for**:
- Performance benchmarks
- Execution time validation
- Speed regression tests

---

## Running Tests

### Basic Commands

```bash
# Run all tests
pytest

# Run with verbose output
pytest -v

# Run with output capture disabled (see print statements)
pytest -s

# Run specific test file
pytest tests/services/test_portfolio_service.py

# Run specific test function
pytest tests/services/test_portfolio_service.py::test_get_portfolio_history_query_count

# Run specific test class
pytest tests/services/test_portfolio_service.py::TestPortfolioHistoryPerformance

# Run all service tests
pytest tests/services/

# Run specific service tests
pytest tests/services/test_logging_service.py
```

### Coverage Reports

```bash
# Run with coverage (automatically enabled in pytest.ini)
pytest

# Generate HTML coverage report
pytest --cov=app --cov-report=html
open htmlcov/index.html

# Show missing lines
pytest --cov=app --cov-report=term-missing
```

### Markers

```bash
# Run only performance tests
pytest -m performance

# Run only unit tests
pytest -m unit

# Run only integration tests
pytest -m integration
```

---

## Writing New Tests

### Test Template

```python
"""
Module for testing [feature name].

Brief description of what this module tests.
"""

import pytest
from app.services.some_service import SomeService


class TestFeatureName:
    """Test suite for [feature]."""

    def test_basic_functionality(self, app_context):
        """Test that basic feature works correctly."""
        # Arrange
        expected_result = "something"

        # Act
        result = SomeService.some_method()

        # Assert
        assert result == expected_result

    def test_edge_case(self, app_context):
        """Test edge case handling."""
        # Test edge case
        pass
```

### Performance Test Template

```python
class TestPerformance:
    """Performance benchmarks for [feature]."""

    def test_query_count(self, app_context, query_counter):
        """Test that query count is within target."""
        query_counter.reset()

        # Execute feature
        result = SomeService.expensive_operation()

        # Verify
        print(f"\n✓ Query count: {query_counter.count}")
        assert query_counter.count < 50, \
            f"Too many queries: {query_counter.count}"

    def test_execution_time(self, app_context, timer):
        """Test that execution completes quickly."""
        timer.start()

        # Execute feature
        result = SomeService.expensive_operation()

        elapsed = timer.stop()
        print(f"\n✓ Execution time: {elapsed:.3f}s")
        assert elapsed < 1.0, \
            f"Too slow: {elapsed:.3f}s"
```

### Integration Test Template

```python
class TestAPIEndpoint:
    """Integration tests for [endpoint]."""

    def test_endpoint_success(self, client):
        """Test successful API response."""
        response = client.get('/api/endpoint')

        assert response.status_code == 200
        data = response.get_json()
        assert 'expected_field' in data

    def test_endpoint_validation(self, client):
        """Test input validation."""
        response = client.post('/api/endpoint', json={
            'invalid': 'data'
        })

        assert response.status_code == 400
```

---

## Best Practices

### 1. Test Naming

**Good**:
```python
def test_get_portfolio_history_returns_list_of_daily_values():
    pass

def test_portfolio_fund_history_filters_by_date_range():
    pass
```

**Bad**:
```python
def test_history():  # Too vague
    pass

def test_func1():  # Meaningless name
    pass
```

### 2. Arrange-Act-Assert Pattern

```python
def test_something(app_context):
    # Arrange: Set up test data
    portfolio_id = "some-id"
    start_date = "2024-01-01"

    # Act: Execute the code under test
    result = PortfolioService.get_portfolio_history(
        start_date=start_date
    )

    # Assert: Verify the results
    assert isinstance(result, list)
    assert len(result) > 0
```

### 3. One Assertion Per Test (When Possible)

**Good**:
```python
def test_returns_list(app_context):
    result = PortfolioService.get_portfolio_history()
    assert isinstance(result, list)

def test_list_not_empty(app_context):
    result = PortfolioService.get_portfolio_history()
    assert len(result) > 0
```

**Acceptable** (related assertions):
```python
def test_response_structure(app_context):
    result = PortfolioService.get_portfolio_history()
    assert isinstance(result, list)
    assert len(result) > 0
    assert 'date' in result[0]
```

### 4. Use Descriptive Assertion Messages

```python
# Good
assert query_counter.count < 100, \
    f"Too many queries: {query_counter.count} (target: < 100)"

assert elapsed < 1.0, \
    f"Too slow: {elapsed:.3f}s (target: < 1.0s)"

# Bad
assert query_counter.count < 100
assert elapsed < 1.0
```

### 5. Test Edge Cases

```python
def test_empty_portfolio(app_context):
    """Test behavior with no transactions."""
    # Test empty case

def test_missing_price_data(app_context):
    """Test behavior when price data unavailable."""
    # Test missing data

def test_date_range_boundaries(app_context):
    """Test start and end date edge cases."""
    # Test boundaries
```

### 6. Use Fixtures for Shared Setup

```python
# In conftest.py
@pytest.fixture
def sample_portfolio(app_context):
    """Create a sample portfolio for testing."""
    portfolio = Portfolio(name="Test Portfolio")
    db.session.add(portfolio)
    db.session.commit()
    return portfolio

# In test file
def test_with_sample(sample_portfolio):
    assert sample_portfolio.name == "Test Portfolio"
```

---

## Examples

### Example 1: Service Layer Test

```python
def test_calculate_portfolio_fund_values(app_context):
    """Test fund value calculation."""
    from app.models import PortfolioFund, Portfolio

    # Get a real portfolio fund
    portfolio = Portfolio.query.filter_by(is_archived=False).first()
    if not portfolio:
        pytest.skip("No portfolio found")

    # Calculate values
    result = PortfolioService.calculate_portfolio_fund_values(
        portfolio.funds
    )

    # Verify structure
    assert isinstance(result, list)
    for fund in result:
        assert 'fund_id' in fund
        assert 'total_shares' in fund
        assert 'current_value' in fund
        assert isinstance(fund['current_value'], (int, float))
```

### Example 2: Performance Test

```python
def test_portfolio_summary_performance(
    app_context, query_counter, timer
):
    """Test portfolio summary performance."""
    query_counter.reset()
    timer.start()

    # Execute
    result = PortfolioService.get_portfolio_summary()

    elapsed = timer.stop()

    # Verify results
    assert isinstance(result, list)

    # Verify performance
    print(f"\n✓ Queries: {query_counter.count}")
    print(f"✓ Time: {elapsed:.3f}s")

    assert query_counter.count < 100
    assert elapsed < 0.5
```

### Example 3: API Integration Test

```python
def test_portfolio_history_endpoint(client):
    """Test /api/portfolio-history endpoint."""
    # Make request
    response = client.get('/api/portfolio-history?days=30')

    # Verify response
    assert response.status_code == 200

    data = response.get_json()
    assert isinstance(data, list)
    assert len(data) <= 31  # 30 days + today

    # Verify structure
    if len(data) > 0:
        assert 'date' in data[0]
        assert 'portfolios' in data[0]
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
npm test

# Run tests in watch mode (interactive)
npm run test:watch

# Run with coverage report
npm run test:coverage

# Run tests in CI mode (non-interactive)
npm run test:ci

# View HTML coverage report
npm run test:coverage && open coverage/index.html
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
  run: npm ci

- name: Run tests with coverage
  run: npm run test:ci

- name: Run linting
  run: npm run lint
```

**CI Requirements for PR Merge:**
- ✅ All tests pass
- ✅ Linting passes
- ✅ Coverage thresholds met (when enabled)

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
- ESLint recognizes Jest globals (`describe`, `test`, `expect`)
- Check `eslint.config.mjs` has test file configuration

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

# Test backend directly (from inside container using Python)
docker compose exec backend python -c "import urllib.request; print(urllib.request.urlopen('http://localhost:5000/api/system/health').read().decode())"

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
- From inside the Docker network (via `docker compose exec` using Python's urllib)
- Through the frontend Nginx proxy at `http://localhost/api/`

This is the production architecture and what the tests verify.

**Note**: The backend container does not include curl. To test the backend from inside the container, use Python's `urllib.request` module which is part of the standard library.

### CI/CD Integration

Docker tests run automatically in GitHub Actions (`.github/workflows/docker-test.yml`) when:
- Dockerfiles are modified
- docker-compose.yml changes
- Python dependencies change (pyproject.toml, uv.lock)
- Nginx configuration changes

**Test Jobs:**
1. **docker-integration-test**: Tests default configuration
2. **docker-custom-hostname-test**: Tests custom BACKEND_HOST

### Test Coverage

#### Container Build
- ✅ Backend builds successfully with uv and Python 3.13
- ✅ Frontend builds successfully with Node 24
- ✅ Multi-stage builds optimize image size
- ✅ Dependencies install correctly

#### Container Startup
- ✅ Backend starts and creates database
- ✅ Frontend starts and serves static files
- ✅ Services connect via Docker network
- ✅ Auto-generated INTERNAL_API_KEY works

#### Health Checks
- ✅ Backend health endpoint returns 200
- ✅ Backend health response includes database status
- ✅ Backend version endpoint returns version info
- ✅ Health checks complete within 60 seconds

#### Frontend-Backend Integration
- ✅ Frontend serves index page
- ✅ Frontend proxies /api/ requests to backend
- ✅ Nginx template substitution works correctly
- ✅ API responses return through proxy

#### Custom Configuration
- ✅ Custom BACKEND_HOST values work
- ✅ Custom container names work correctly
- ✅ Environment variable substitution works

### Hybrid Testing Approach

The Docker integration test workflow uses a **three-layer testing approach**:

**Step 1: Backend Health (Internal)**
- Tests backend independently from inside container network
- Uses `docker compose exec -T backend python -c "import urllib.request; urllib.request.urlopen('http://localhost:5000/api/system/health')"`
- Catches backend-specific failures (database, startup, configuration)
- Verifies backend is healthy before testing integration
- Note: Uses Python's urllib since curl is not available in the backend container

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
- Test backend from inside container: `docker compose exec backend python -c "import urllib.request; print(urllib.request.urlopen('http://localhost:5000/api/system/health').read().decode())"`

**Frontend proxy fails:**
- Check BACKEND_HOST environment variable
- Verify nginx template substitution in logs: `docker compose logs frontend`
- Check network connectivity between containers: `docker network inspect investment-portfolio-manager_app-network`
- Test frontend serves static files: `curl http://localhost/`
- Test backend directly from inside: `docker compose exec backend python -c "import urllib.request; print(urllib.request.urlopen('http://localhost:5000/api/system/health').read().decode())"`

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
- [Health Check API](../backend/app/api/system_namespace.py)

---

## Troubleshooting

### Issue: Import Errors

**Problem**: `ImportError: cannot import name 'create_app'`

**Solution**:
- Ensure `PYTHONPATH` includes backend directory
- Run pytest from backend directory
- Check conftest.py imports

### Issue: Database Not Found

**Problem**: `No such table` errors

**Solution**:
```bash
# Ensure database exists
cd backend
source .venv/bin/activate
python run.py  # Creates tables if needed
pytest
```

### Issue: Fixture Not Found

**Problem**: `fixture 'query_counter' not found`

**Solution**:
- Ensure conftest.py exists in tests directory
- Check fixture name spelling
- Verify conftest.py is not excluded in pytest.ini

### Issue: Tests Hang

**Problem**: Tests don't complete

**Solution**:
- Check for infinite loops
- Verify database queries don't timeout
- Use pytest timeout plugin:
  ```bash
  pip install pytest-timeout
  pytest --timeout=30
  ```

---

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD) or alongside implementation
2. **Test happy path and edge cases**
3. **Add performance tests** for expensive operations
4. **Update this documentation** with new patterns
5. **Maintain test coverage** above 70%

---

## Resources

- [Pytest Documentation](https://docs.pytest.org/)
- [Flask Testing Guide](https://flask.palletsprojects.com/en/latest/testing/)
- [pytest-flask Plugin](https://pytest-flask.readthedocs.io/)
- [Coverage.py Documentation](https://coverage.readthedocs.io/)

---

**Version**: 1.3.5+
**Last Updated**: 2025-12-18
**Maintainer**: @ndewijer
