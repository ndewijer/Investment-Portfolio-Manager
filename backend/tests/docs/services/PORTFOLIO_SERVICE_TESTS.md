# PortfolioService Test Suite Documentation

**File**: `tests/services/test_portfolio_service.py`\
**Service**: `app/services/portfolio_service.py`\
**Tests**: 46 tests\
**Coverage**: 90%\
**Created**: Version 1.3.3 (Phase 3)\
**Updated**: Phase 5 - Phase 2b (Service Layer Refactoring)

## Overview

Comprehensive test suite for the PortfolioService class, covering all CRUD operations, relationship management, value calculations, and error handling scenarios.

## Test Structure

### Test Classes

#### 1. TestPortfolioCRUD (13 tests)
Tests basic portfolio operations:
- `test_create_portfolio` - Create portfolio with name and description
- `test_create_portfolio_minimal` - Create portfolio with name only
- `test_update_portfolio` - Update portfolio fields
- `test_update_nonexistent_portfolio` - Error handling for invalid portfolio ID
- `test_delete_portfolio` - Delete portfolio
- `test_delete_nonexistent_portfolio` - Error handling for invalid deletion
- `test_update_archive_status` - Archive/unarchive portfolios
- `test_get_portfolios_list_default` - Get portfolios excluding excluded ones
- `test_get_portfolios_list_include_excluded` - Get all portfolios including excluded
- **[Phase 1b]** `test_get_all_portfolios` - Get all portfolios without filtering
- **[Phase 1b]** `test_get_portfolio` - Retrieve single portfolio by ID
- **[Phase 1b]** `test_get_portfolio_not_found` - 404 handling for missing portfolio

#### 2. TestPortfolioFundManagement (11 tests)
Tests portfolio-fund relationship operations:
- `test_create_portfolio_fund` - Create fund relationship
- `test_create_portfolio_fund_invalid_portfolio` - Error handling for invalid portfolio
- `test_create_portfolio_fund_invalid_fund` - Error handling for invalid fund
- `test_get_all_portfolio_funds` - Retrieve all relationships
- `test_delete_portfolio_fund_no_transactions` - Delete relationship without transactions
- `test_delete_portfolio_fund_with_transactions_no_confirmation` - Require confirmation for deletion with transactions
- `test_delete_portfolio_fund_with_transactions_confirmed` - Delete relationship with cascade
- **[Phase 1b]** `test_get_portfolio_fund_without_relationships` - Basic PortfolioFund retrieval
- **[Phase 1b]** `test_get_portfolio_fund_with_relationships` - PortfolioFund with eager loading
- **[Phase 1b]** `test_get_portfolio_fund_not_found` - Handle missing PortfolioFund
- **[Phase 1b]** `test_count_portfolio_fund_transactions` - Count transactions for a portfolio fund
- **[Phase 1b]** `test_count_portfolio_fund_dividends` - Count dividends for a portfolio fund

#### 3. TestPortfolioCalculations (5 tests)
Tests portfolio value and performance calculations:
- `test_calculate_portfolio_fund_values_basic` - Calculate current values
- `test_calculate_portfolio_fund_values_with_sell` - Handle sell transactions in calculations
- `test_get_portfolio_funds` - Get portfolio funds with values
- `test_get_portfolio_summary` - Get summary of all portfolios
- `test_get_portfolio_summary_with_realized_gains` - Include realized gains in summary

#### 4. TestEdgeCases (3 tests)
Tests error handling and boundary conditions:
- `test_update_archive_status_nonexistent` - Error handling for invalid archive operation
- `test_delete_portfolio_fund_nonexistent` - Error handling for invalid relationship deletion
- `test_calculate_portfolio_fund_values_no_price` - Handle missing price data

#### 5. TestPortfolioHistoricalMethods (8 tests)
Tests portfolio historical analysis functionality:
- `test_get_portfolio_history_no_portfolios` - Empty portfolio history handling
- `test_get_portfolio_history_no_transactions` - Portfolios without transactions
- `test_get_portfolio_history_basic` - Historical value calculations over time
- `test_get_portfolio_fund_history_invalid_portfolio` - Error handling for invalid portfolio
- `test_get_portfolio_fund_history_no_transactions` - Empty fund history handling
- `test_get_portfolio_fund_history_basic` - Fund-level historical analysis
- `test_get_portfolio_summary_no_portfolios` - Empty portfolio summary
- `test_get_portfolio_summary_with_dividend_reinvestment` - Summary with dividend data

#### 6. TestPortfolioHelperMethods (3 tests)
Tests internal helper methods and edge cases:
- `test_process_transactions_sell_to_zero` - Transaction processing when all shares sold
- `test_process_transactions_sell_below_zero` - Overselling scenario handling
- `test_calculate_fund_metrics_historical_format` - Historical format calculations

#### 7. TestGetActivePortfolios (3 tests)
Tests `get_active_portfolios()` method for retrieving non-archived portfolios (Version 1.3.3 Phase 2b):
- `test_get_active_portfolios` - Returns only non-archived portfolios
- `test_get_active_portfolios_empty` - Returns empty list when all portfolios are archived
- `test_get_active_portfolios_none_exist` - Returns empty list when no portfolios exist

## Testing Strategy

### Test Isolation Pattern
Tests use **Query-Specific Data** approach:
- Each test creates its own test data with unique identifiers
- Tests query for specific data they created, not total counts
- Assertions verify specific records exist correctly
- No database cleanup required between tests

**Example Pattern**:
```python
def test_get_portfolios_list_default(self, app_context, db_session):
    # Create test data
    p1 = Portfolio(name="Normal Portfolio", exclude_from_overview=False)
    p2 = Portfolio(name="Excluded Portfolio", exclude_from_overview=True)

    # Test the service
    portfolios = PortfolioService.get_portfolios_list()

    # Assert specific data exists (not totals)
    portfolio_ids = {p.id for p in portfolios}
    assert p1.id in portfolio_ids      # ✅ Verify our data exists
    assert p2.id not in portfolio_ids  # ✅ Verify filtering works
```

### Key Testing Principles
1. **Direct Object Creation**: Uses explicit database models with `str(uuid.uuid4())` IDs
2. **Unique Test Data**: Each test generates unique ISINs, names, and IDs
3. **Specific Assertions**: Tests verify their data exists, not database state
4. **Error Scenario Coverage**: Tests invalid operations and edge cases

## Service Methods Tested

### Portfolio CRUD Operations
- `create_portfolio(name, description)` - Create new portfolio
- `update_portfolio(portfolio_id, name, description, exclude_from_overview)` - Update portfolio
- `delete_portfolio(portfolio_id)` - Delete portfolio
- `update_archive_status(portfolio_id, is_archived)` - Archive/unarchive
- `get_portfolios_list(include_excluded=False)` - List portfolios with filtering

### Portfolio-Fund Relationship Management
- `create_portfolio_fund(portfolio_id, fund_id)` - Create relationship
- `delete_portfolio_fund(portfolio_fund_id, confirmed=False)` - Delete with cascade
- `get_all_portfolio_funds()` - Get all relationships

### Portfolio Value Calculations
- `calculate_portfolio_fund_values(portfolio_funds)` - Calculate current values
- `get_portfolio_funds(portfolio_id)` - Get portfolio fund values
- `get_portfolio_summary()` - Get all portfolio summaries

### Historical Analysis Methods
- `get_portfolio_history(start_date=None, end_date=None)` - Get historical portfolio values
  - Returns API response with camelCase field names (e.g., `totalValue`, `totalRealizedGainLoss`)
  - Internal calculation uses snake_case; conversion happens at API boundary
- `get_portfolio_fund_history(portfolio_id, start_date=None, end_date=None)` - Get fund-level history
  - Returns fund-level data with snake_case field names (internal format)

### Helper Methods
- `_process_transactions_for_date(transactions, date, dividend_shares=0)` - Process transactions for specific date
- `_load_historical_data_batch(portfolio_ids, start_date, end_date)` - Batch load historical data
- `_build_date_lookup_tables(batch_data)` - Build lookup tables for efficient calculations

## Coverage Analysis

**Current Coverage**: 91% (346 of 380 statements)

### Well-Covered Areas
- Basic CRUD operations (create, update, delete)
- Portfolio-fund relationship management
- Value calculations and portfolio summaries
- Historical portfolio analysis (get_portfolio_history, get_portfolio_fund_history)
- Helper methods for transaction processing and calculations
- Error handling for invalid IDs and edge cases
- Complex scenarios with dividends and realized gains

### Areas Needing Coverage (34 missed statements)
- Some error handling paths in complex calculations
- Edge cases in date parsing and parameter validation
- Advanced aggregation query optimization paths
- Some legacy method branches

## Test Data Patterns

### Portfolio Creation
```python
portfolio = Portfolio(
    id=str(uuid.uuid4()),
    name="Test Portfolio",
    description="Test description",
    is_archived=False,
    exclude_from_overview=False
)
```

### Fund Creation with Unique ISIN
```python
fund = Fund(
    id=str(uuid.uuid4()),
    name="Test Fund",
    isin=f"US{uuid.uuid4().hex[:10].upper()}",  # Unique ISIN
    currency="USD",
    exchange="NYSE"
)
```

### Portfolio-Fund Relationship
```python
pf = PortfolioFund(
    id=str(uuid.uuid4()),
    portfolio_id=portfolio.id,
    fund_id=fund.id
)
```

## Error Scenarios Tested

1. **Invalid Portfolio ID**: Update/delete operations with non-existent portfolio
2. **Invalid Fund ID**: Create relationship with invalid fund
3. **Missing Confirmation**: Delete relationship with transactions without confirmation
4. **Missing Price Data**: Value calculations without current prices
5. **Cascade Deletion**: Proper handling of dependent data deletion

## Future Enhancements

1. **Performance Tests**: Add query count monitoring for value calculations
2. **Integration Tests**: Test with real price data updates
3. **Concurrency Tests**: Test multiple portfolio operations simultaneously
4. **Historical Analysis**: Test portfolio performance over time ranges
5. **Data Validation**: Enhanced input validation testing

## Dependencies

- **Models**: Portfolio, Fund, PortfolioFund, Transaction, FundPrice, Dividend, RealizedGainLoss
- **Services**: None (isolated service testing)
- **External**: None (uses test database)

## Bug Fixes Validated

This test suite validates fixes to ensure:
- Portfolio CRUD operations work correctly
- Fund relationships are properly managed
- Value calculations use correct formulas
- Error handling is comprehensive
- Data integrity is maintained

The comprehensive test coverage provides confidence in the PortfolioService implementation and helps prevent regression bugs in future development.

---

**Last Updated**: 2026-01-13 (Version 1.4.1)
**Maintained By**: @ndewijer
