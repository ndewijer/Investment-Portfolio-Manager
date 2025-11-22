# Route Business Logic Refactoring - Prioritized Remediation Plan

## Phase 1: CRITICAL (Blocks Integration Testing)
These must be fixed first - they have complex orchestration logic that makes testing impossible without database access.

### 1.1 IBKR Import Orchestration (ibkr_routes.py:174-237)
**Current Problem:** Route directly orchestrates multi-step IBKR import workflow
**Impact:** Cannot test IBKR import flow without actual database and IBKR service
**Dependencies:** IBKRFlexService, IBKRConfigService
**Effort:** 2-3 hours

**Implementation Steps:**
1. Create `IBKRFlexService.execute_full_import(config)` method
   - Takes IBKRConfig object
   - Internally handles: decrypt token, fetch statement, parse, import, update config
   - Returns: (results dict, http_status_code)

2. Move logic from trigger_import:
   - Token decryption (currently line 194)
   - Statement fetching (lines 197)
   - Transaction parsing (line 203)
   - Import execution (line 206)
   - Config state update (lines 209-210)

3. Simplify route to:
```python
@ibkr.route("/ibkr/import", methods=["POST"])
def trigger_import():
    config = IBKRConfig.query.first()
    if not config or not config.enabled:
        return appropriate_error()
    try:
        result, status = IBKRFlexService.execute_full_import(config)
        return jsonify(result), status
    except Exception as e:
        return error_response(e)
```

**Testing:** Unit test can mock IBKRFlexService
**Files to Modify:** ibkr_routes.py, ibkr_flex_service.py

---

### 1.2 Portfolio Fund Deletion with Confirmation (portfolio_routes.py:320-396)
**Current Problem:** Route handles complex deletion confirmation logic with direct queries
**Impact:** Cannot unit test deletion confirmation flow without database
**Dependencies:** PortfolioService
**Effort:** 2-3 hours

**Implementation Steps:**
1. Create service methods:
   - `PortfolioService.get_deletion_impact_analysis(portfolio_fund_id)`
     - Returns: {transaction_count, dividend_count, fund_name, portfolio_name, fund_object}
   - `PortfolioService.delete_portfolio_fund_confirmed(portfolio_fund_id, confirmed=False)`
     - Replaces current delete logic
     - Handles confirmation checking internally
     - Returns: {success, message, requires_confirmation, details}

2. Move from route (lines 340-368):
   - Query for PortfolioFund with eager loading
   - Count transactions and dividends
   - Format response with fund/portfolio names

3. Simplify route to:
```python
@portfolios.route("/portfolio-funds/<string:portfolio_fund_id>", methods=["DELETE"])
def delete_portfolio_fund(portfolio_fund_id):
    confirmed = request.args.get("confirm") == "true"
    try:
        result = PortfolioService.delete_portfolio_fund_confirmed(portfolio_fund_id, confirmed)
        if result["requires_confirmation"]:
            return jsonify(result), 409
        return "", 204
    except ValueError as e:
        return error_response(str(e), 404)
```

**Testing:** Mock PortfolioService to test confirmation logic
**Files to Modify:** portfolio_routes.py, portfolio_service.py

---

### 1.3 IBKR Transaction Unallocation (ibkr_routes.py:621-684)
**Current Problem:** Route contains complex cascade deletion logic with loop
**Impact:** Cannot test unallocation without database cascade behavior
**Dependencies:** IBKRTransactionService
**Effort:** 2 hours

**Implementation Steps:**
1. Create service method:
   - `IBKRTransactionService.unallocate_transaction(transaction_id)`
     - Gets allocations
     - Deletes all associated transactions
     - Reverts IBKR transaction status
     - Returns: {success, message, deleted_count}
   - Handles all cascade logic internally

2. Move from route (lines 627-650):
   - Query for allocations
   - Loop and delete transactions
   - Update IBKR transaction status
   - Database commit

3. Simplify route to:
```python
@ibkr.route("/ibkr/inbox/<transaction_id>/unallocate", methods=["POST"])
def unallocate_transaction(transaction_id):
    try:
        ibkr_txn = IBKRTransaction.query.get_or_404(transaction_id)
        if ibkr_txn.status != "processed":
            return jsonify({"error": "Not processed"}), 400
        result = IBKRTransactionService.unallocate_transaction(transaction_id)
        return jsonify(result), 200
    except Exception as e:
        return error_response(e)
```

**Testing:** Unit test can mock cascade behavior
**Files to Modify:** ibkr_routes.py, ibkr_transaction_service.py

---

## Phase 2: HIGH PRIORITY (Needed for Complete Test Coverage)
These enable testing of bulk operations and complex filtering scenarios.

### 2.1 Bulk IBKR Transaction Allocation (ibkr_routes.py:813-918)
**Current Problem:** 106 lines of batch processing with nested error handling
**Impact:** Cannot test bulk allocation error scenarios in isolation
**Dependencies:** IBKRTransactionService
**Effort:** 2-3 hours

**Implementation Steps:**
1. Create service method:
   - `IBKRTransactionService.bulk_process_allocations(transaction_ids, allocations)`
     - Validates allocations sum to 100%
     - Processes each transaction
     - Aggregates results and errors
     - Returns: {success, processed_count, failed_count, errors}

2. Move from route (lines 848-897):
   - Allocation validation (lines 849-851)
   - Loop iteration (lines 858-883)
   - Error collection
   - Result aggregation

3. Simplify route to:
```python
@ibkr.route("/ibkr/inbox/bulk-allocate", methods=["POST"])
def bulk_allocate_transactions():
    data = request.get_json()
    if not data or "transaction_ids" not in data:
        return jsonify({"error": "Missing fields"}), 400
    try:
        result = IBKRTransactionService.bulk_process_allocations(
            data["transaction_ids"],
            data["allocations"]
        )
        return jsonify(result), 200
    except ValueError as e:
        return jsonify({"error": str(e)}), 400
```

**Testing:** Mock process_transaction_allocation to test error scenarios
**Files to Modify:** ibkr_routes.py, ibkr_transaction_service.py

---

### 2.2 Bulk Fund Price Update (fund_routes.py:528-575)
**Current Problem:** 48 lines of loop with multiple service calls and error aggregation
**Impact:** Cannot test bulk price update error handling without actual price service
**Dependencies:** FundService, HistoricalPriceService
**Effort:** 1.5 hours

**Implementation Steps:**
1. Create service method:
   - `FundService.update_all_fund_prices()`
     - Gets all funds with symbols
     - Calls HistoricalPriceService for each
     - Aggregates results
     - Returns: {success, updated_funds, errors, total_updated, total_errors}

2. Move from route (lines 529-575):
   - Fund query
   - Loop iteration
   - Error handling
   - Result aggregation

3. Simplify route to:
```python
@funds.route("/funds/update-all-prices", methods=["POST"])
@require_api_key
def update_all_fund_prices():
    try:
        result = FundService.update_all_fund_prices()
        return jsonify(result), 200
    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500
```

**Testing:** Mock HistoricalPriceService to test different error scenarios
**Files to Modify:** fund_routes.py, fund_service.py

---

### 2.3 Log Filtering and Pagination (developer_routes.py:634-700)
**Current Problem:** 67 lines of query building with complex filtering and pagination
**Impact:** Cannot test filter combinations without database queries
**Dependencies:** DeveloperService
**Effort:** 2 hours

**Implementation Steps:**
1. Create service method:
   - `DeveloperService.get_filtered_logs(filters_dict, sort_by, sort_dir, page, per_page)`
     - Takes dict: {levels, categories, start_date, end_date, source}
     - Builds query with all filters
     - Handles sorting and pagination
     - Formats response
     - Returns: {logs, total, pages, current_page}

2. Move from route (lines 642-700):
   - Query building (lines 643-662)
   - Sorting logic (lines 665-671)
   - Pagination (lines 674-676)
   - Response formatting (lines 680-695)

3. Simplify route to:
```python
@developer.route("/logs", methods=["GET"])
def get_logs():
    filters = {
        "levels": request.args.get("level"),
        "categories": request.args.get("category"),
        "start_date": request.args.get("start_date"),
        "end_date": request.args.get("end_date"),
        "source": request.args.get("source"),
    }
    try:
        result = DeveloperService.get_filtered_logs(
            filters,
            sort_by=request.args.get("sort_by", "timestamp"),
            sort_dir=request.args.get("sort_dir", "desc"),
            page=int(request.args.get("page", 1)),
            per_page=int(request.args.get("per_page", 50)),
        )
        return jsonify(result), 200
    except Exception as e:
        return error_response(e)
```

**Testing:** Can test filter combinations with mock data
**Files to Modify:** developer_routes.py, developer_service.py

---

## Phase 3: MEDIUM PRIORITY (Code Quality & Maintainability)
These improve code quality but aren't blocking. Should be done before integration tests.

### 3.1 CSV File Handling Utilities (developer_routes.py:184-507)
**Current Problem:** Duplicated file validation and CSV parsing logic across 3 endpoints
**Impact:** Bug fixes need to be made in multiple places
**Dependencies:** DeveloperService
**Effort:** 1.5 hours

**Implementation Steps:**
1. Create utility methods in DeveloperService:
   - `validate_csv_file_request(request, expected_headers, file_field_name="file")`
     - Checks file exists
     - Checks file type is .csv
     - Reads and validates headers
     - Returns: (file_content, portfolio_fund_or_fund_object, error_response_or_none)

   - `detect_csv_type(headers_set)`
     - Returns: "transaction", "prices", "unknown"
     - Used to prevent wrong file type uploads

2. Apply to endpoints:
   - import_transactions: Lines 184-240
   - import_fund_prices: Lines 441-507
   - Both share 70% of code

3. After consolidation, endpoints become:
```python
@developer.route("/import-transactions", methods=["POST"])
def import_transactions():
    file_content, pf, error = DeveloperService.validate_csv_file_request(
        request, {"date", "type", "shares", "cost_per_share"}
    )
    if error:
        return error
    try:
        count = DeveloperService.import_transactions_csv(file_content, pf.id)
        return jsonify({"message": f"Imported {count} transactions"}), 200
    except ValueError as e:
        return error_response(str(e), 400)
```

**Testing:** Can test file validation separately from import logic
**Files to Modify:** developer_routes.py, developer_service.py

---

### 3.2 Transaction Realized Gain/Loss Queries (transaction_routes.py:98-209)
**Current Problem:** Two separate query patterns for realized gain/loss in create/update
**Impact:** Hard to maintain consistent realized gain/loss behavior
**Dependencies:** TransactionService
**Effort:** 1 hour

**Implementation Steps:**
1. Create service method:
   - `TransactionService.get_realized_gain_loss_info(transaction)`
     - Handles both sell transaction patterns
     - Returns dict: {realized_gain_loss, found: bool} or None
     - Encapsulates query logic

2. Replace in both endpoints:
   - create_transaction: Lines 98-105
   - update_transaction: Lines 195-209

3. Simplify to:
```python
response = service.format_transaction(transaction)
if transaction.type == "sell":
    rgl_info = service.get_realized_gain_loss_info(transaction)
    if rgl_info:
        response["realized_gain_loss"] = rgl_info["realized_gain_loss"]
return jsonify(response)
```

**Testing:** Can unit test realized gain/loss query separately
**Files to Modify:** transaction_routes.py, transaction_service.py

---

### 3.3 Portfolio Funds Data Enrichment (portfolio_routes.py:249-264)
**Current Problem:** Dividend type enrichment loop with direct db.session calls in route
**Impact:** Cannot test data enrichment without database
**Dependencies:** PortfolioService
**Effort:** 1 hour

**Implementation Steps:**
1. Create service method:
   - `PortfolioService.enrich_portfolio_funds_with_dividend_types(portfolio_funds_list)`
     - Takes list of portfolio fund dicts
     - Adds dividend_type for each
     - Returns enriched list

2. Move from route (lines 258-262):
   - The enrichment loop

3. Simplify route to:
```python
if request.method == "GET":
    portfolio_id = request.args.get("portfolio_id")
    if portfolio_id:
        portfolio_funds = PortfolioService.get_portfolio_funds(portfolio_id)
    else:
        portfolio_funds = PortfolioService.get_all_portfolio_funds()

    portfolio_funds = PortfolioService.enrich_portfolio_funds_with_dividend_types(
        portfolio_funds
    )
    return jsonify(portfolio_funds)
```

**Testing:** Can unit test enrichment logic
**Files to Modify:** portfolio_routes.py, portfolio_service.py

---

### 3.4 Portfolio Totals Calculation (portfolio_routes.py:62-70)
**Current Problem:** Financial aggregation logic in formatting method
**Impact:** Difficult to test independently; mixed concerns
**Dependencies:** PortfolioService
**Effort:** 30 minutes

**Implementation Steps:**
1. Create service method:
   - `PortfolioService.calculate_portfolio_totals(portfolio_funds_data)`
     - Takes list of fund metrics
     - Returns dict with all totals
     - Cleanly separates calculation from formatting

2. Use in _format_portfolio_detail:
```python
@staticmethod
def _format_portfolio_detail(portfolio, portfolio_funds_data):
    totals = PortfolioService.calculate_portfolio_totals(portfolio_funds_data)
    return {
        "id": portfolio.id,
        "name": portfolio.name,
        **totals,
    }
```

**Testing:** Can unit test calculations independently
**Files to Modify:** portfolio_routes.py, portfolio_service.py

---

## Phase 4: LOW PRIORITY (Nice-to-Have Refactoring)
These are low-complexity, should be done as "polish" work.

### 4.1 IBKR Transaction Status Updates (ibkr_routes.py:378-405, 419-445)
**Issue:** Direct status updates in routes
**Service Methods Needed:**
- `IBKRTransactionService.mark_as_ignored(transaction_id)`
- `IBKRTransactionService.delete(transaction_id)`

---

### 4.2 Fund Service Enhancements (fund_routes.py:177-261)
**Issues:**
- Direct price queries (lines 177-182)
- Duplicate symbol lookup logic (lines 86-106, 244-261)

**Service Methods Needed:**
- `FundService.get_latest_price(fund_id)`
- `FundService.lookup_symbol_info_with_fallback(symbol, force_refresh)`

---

### 4.3 System Service Additions (system_routes.py:55-56, developer_routes.py:581-595)
**Issues:**
- Raw SQL in health check
- Logging settings persistence

**Service Methods Needed:**
- `SystemService.check_database_connection()`
- `SystemService.update_logging_configuration(enabled, level)` or `DeveloperService.update_logging_settings()`

---

### 4.4 IBKR Allocation Operations (ibkr_routes.py:701-746, 761-809)
**Issues:**
- Complex allocation detail aggregation (lines 710-728)
- Data transformation mixed with queries
- Duplicate allocation percentage validation

**Service Methods Needed:**
- `IBKRTransactionService.get_allocation_details_formatted(transaction_id)`
- `IBKRTransactionService.validate_allocations_percentage(allocations)` - static/class method

---

## Implementation Schedule

### Week 1 (Phase 1 - CRITICAL)
- Day 1-2: IBKR Import Orchestration
- Day 3-4: Portfolio Fund Deletion
- Day 5: IBKR Transaction Unallocation

### Week 2 (Phase 2 - HIGH PRIORITY)
- Day 1-2: Bulk IBKR Allocation
- Day 3: Bulk Fund Price Update
- Day 4-5: Log Filtering & Pagination

### Week 3 (Phase 3 - MEDIUM)
- Day 1-2: CSV File Handling
- Day 3: Transaction Realized Gain/Loss
- Day 4: Portfolio Funds Enrichment
- Day 5: Portfolio Totals Calculation

### Week 4 (Phase 4 - LOW PRIORITY)
- Complete remaining low-priority refactoring
- Integration test writing

---

## Testing Strategy

For each refactored piece:
1. Unit test the service method with mocks
2. Unit test the route with mocked service
3. Integration test the complete flow (end-to-end with real dependencies)
4. Regression test related functionality

Example test structure:
```python
# Unit test - service
def test_execute_full_import_success(self):
    config = Mock()
    service = IBKRFlexService()
    service.fetch_statement = Mock(return_value="<xml>...</xml>")
    service.parse_flex_statement = Mock(return_value=[...])
    service.import_transactions = Mock(return_value={...})

    result, status = service.execute_full_import(config)

    assert status == 200
    assert result["success"]

# Unit test - route
def test_trigger_import_calls_service(self):
    with patch('IBKRFlexService.execute_full_import') as mock:
        mock.return_value = ({...}, 200)
        response = client.post('/ibkr/import')
        assert response.status_code == 200

# Integration test - full flow
def test_trigger_import_full_flow(self):
    # Use real services but mock external APIs
    response = client.post('/ibkr/import')
    assert response.status_code == 200
    # Verify database state
```
