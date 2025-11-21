# Route Files Business Logic Analysis Report

## Analysis Methodology
Reviewed all 7 route files for:
1. Direct database queries/ORM operations beyond simple gets
2. Complex business logic (>10 lines in single endpoint)
3. Data transformation/calculation logic
4. Multiple database operations in single endpoint
5. Transaction management logic
6. Code difficult to unit test without hitting database

## VIOLATIONS FOUND

### portfolio_routes.py

#### Violation 1: Complex portfolio fund deletion logic with embedded queries
**Location:** Lines 320-396 (`delete_portfolio_fund` endpoint)
**Severity:** CRITICAL
**Description:**
- 77 lines of complex business logic in route handler
- Performs confirmation checking, conditional query logic, and transaction management
- Directly queries Transaction and Dividend models (lines 347-352)
- Uses db.joinedload for eager loading (lines 342-343)
- Handles cascade deletion validation
- Complex error handling with conditional responses

**Current Code Issues:**
```python
# Lines 340-368: Direct queries in route
portfolio_fund = PortfolioFund.query.options(...).get(portfolio_fund_id)
transaction_count = Transaction.query.filter_by(...).count()
dividend_count = Dividend.query.filter_by(...).count()
```

**Suggested Service Method:** `PortfolioService.prepare_deletion_confirmation_info(portfolio_fund_id)`
**Additional Methods Needed:**
- `PortfolioService.get_deletion_impact_analysis(portfolio_fund_id)` - returns transaction/dividend counts
- `PortfolioService.validate_portfolio_fund_deletable(portfolio_fund_id)` - checks deletion constraints

---

#### Violation 2: Data transformation logic in get endpoint
**Location:** Lines 249-264 (`handle_portfolio_funds` GET method)
**Severity:** MEDIUM
**Description:**
- Manual loop to add dividend_type from related Fund model (lines 258-262)
- Data enrichment logic that should be in service
- Direct database session access: `db.session.get(Fund, pf["fund_id"])` (line 260)

**Current Code:**
```python
for pf in portfolio_funds:
    if "fund_id" in pf:
        fund = db.session.get(Fund, pf["fund_id"])
        if fund:
            pf["dividend_type"] = fund.dividend_type.value
```

**Suggested Service Method:** `PortfolioService.enrich_portfolio_funds_with_dividend_type(portfolio_funds)`

---

#### Violation 3: Complex portfolio value calculation with loop aggregation
**Location:** Lines 62-70 (`_format_portfolio_detail` method)
**Severity:** MEDIUM
**Description:**
- Aggregation of multiple financial metrics across portfolio funds
- Multiple sum() operations on calculated fields
- Financial calculation logic in formatting method

**Current Code:**
```python
"totalValue": sum(pf["current_value"] for pf in portfolio_funds_data),
"totalCost": sum(pf["total_cost"] for pf in portfolio_funds_data),
"totalDividends": sum(pf["total_dividends"] for pf in portfolio_funds_data),
# etc...
```

**Suggested Service Method:** `PortfolioService.calculate_portfolio_totals(portfolio_funds_data)` - returns aggregated dictionary

---

### transaction_routes.py

#### Violation 4: Conditional queries for realized gain/loss in create endpoint
**Location:** Lines 98-105 (`create_transaction` endpoint)
**Severity:** MEDIUM
**Description:**
- Direct query for RealizedGainLoss records (line 100)
- Conditional logic based on transaction type
- Business logic mixed with response formatting

**Current Code:**
```python
if data["type"] == "sell":
    realized_records = RealizedGainLoss.query.filter_by(
        transaction_id=transaction.id
    ).first()
    if realized_records:
        response["realized_gain_loss"] = realized_records.realized_gain_loss
```

**Suggested Service Method:** `TransactionService.enrich_transaction_response_with_realized_gain_loss(transaction, transaction_data)`

---

#### Violation 5: Complex realized gain/loss query in update endpoint
**Location:** Lines 195-209 (`update_transaction` endpoint)
**Severity:** MEDIUM
**Description:**
- Complex query with multiple filters and ordering (lines 198-205)
- Conditional business logic for sell transactions
- Navigation through relationships (transaction.portfolio_fund)

**Current Code:**
```python
realized_records = (
    RealizedGainLoss.query.filter_by(
        portfolio_id=portfolio_fund.portfolio_id,
        fund_id=portfolio_fund.fund_id,
        transaction_date=transaction.date,
    )
    .order_by(RealizedGainLoss.created_at.desc())
    .first()
)
```

**Suggested Service Method:** `TransactionService.get_realized_gain_loss_for_transaction(transaction)`

---

### dividend_routes.py

#### Violation 6: Pre-deletion data gathering in delete endpoint
**Location:** Lines 222-229 (`delete_dividend` endpoint)
**Severity:** LOW-MEDIUM
**Description:**
- Queries dividend record before deletion to get details for logging
- Navigation through relationships (dividend.fund)
- Should be encapsulated in service

**Current Code:**
```python
dividend = Dividend.query.get_or_404(dividend_id)
dividend_details = {
    "fund_name": dividend.fund.name,
    "total_amount": dividend.total_amount,
    ...
}
```

**Suggested Service Method:** `DividendService.get_deletion_details(dividend_id)` called before delete

---

### fund_routes.py

#### Violation 7: Direct fund price query in get_fund endpoint
**Location:** Lines 177-182 (`get_fund` endpoint)
**Severity:** LOW
**Description:**
- Direct database query for latest price
- Query construction in route handler
- Should be encapsulated in service

**Current Code:**
```python
price_record = (
    FundPrice.query.filter_by(fund_id=fund_id)
    .order_by(FundPrice.date.desc())
    .first()
)
if price_record:
    latest_price = price_record.price
```

**Suggested Service Method:** `FundService.get_latest_price(fund_id)` - returns price or None

---

#### Violation 8: Symbol info lookup with conditional retry logic
**Location:** Lines 86-106 (`create_fund` endpoint)
**Severity:** MEDIUM
**Description:**
- Conditional external service call based on input
- Error handling with fallback (lines 100-106)
- External API integration logic in route

**Current Code:**
```python
if data.get("symbol"):
    try:
        symbol_info = SymbolLookupService.get_symbol_info(
            data["symbol"], force_refresh=True
        )
        # ...
    except Exception as e:
        logger.log(...)
```

**Suggested Service Method:** `FundService.lookup_and_enrich_symbol_info(symbol)` - encapsulates retry/fallback logic

---

#### Violation 9: Symbol info lookup in update with same pattern
**Location:** Lines 244-261 (`update_fund` endpoint)
**Severity:** MEDIUM
**Description:**
- Duplicate symbol info lookup logic from create endpoint
- Conditional based on symbol_changed flag
- Repeated error handling pattern

**Suggested Service Method:** Extract to shared `FundService.refresh_symbol_info_if_needed(fund, symbol_changed)`

---

#### Violation 10: Bulk fund price update with loop and multiple service calls
**Location:** Lines 528-575 (`update_all_fund_prices` endpoint)
**Severity:** HIGH
**Description:**
- 48 lines of complex loop logic in route
- Multiple database queries within loop (lines 530, 536-546)
- Error aggregation and result collection
- Transaction management for batch operations

**Current Code:**
```python
for fund in funds_with_symbols:
    try:
        result, status = HistoricalPriceService.update_historical_prices(fund.id)
        if status == 200:
            updated_funds.append(...)
        else:
            errors.append(...)
    except Exception as e:
        errors.append(...)
```

**Suggested Service Method:** `FundService.update_all_fund_prices()` - encapsulates entire bulk operation with error handling and aggregation

---

### ibkr_routes.py

#### Violation 11: Complex IBKR import workflow in trigger_import
**Location:** Lines 174-237 (`trigger_import` endpoint)
**Severity:** CRITICAL
**Description:**
- 64 lines of orchestration logic in route
- Multiple service method calls in sequence (lines 194-206)
- Token decryption logic (line 194)
- Database state update (lines 209-210)
- Complex result aggregation and formatting (lines 219-226)
- Error handling with logging

**Current Code:**
```python
# Line 194: Token decryption in route
token = service._decrypt_token(config.flex_token)
# Lines 197-206: Sequential operations
xml_data = service.fetch_statement(token, config.flex_query_id, use_cache=True)
transactions = service.parse_flex_statement(xml_data)
results = service.import_transactions(transactions)
# Lines 209-210: State update
config.last_import_date = datetime.now()
db.session.commit()
```

**Suggested Service Method:** `IBKRFlexService.execute_full_import()` - orchestrates entire workflow internally
**Additional Methods:**
- Move token decryption inside service: `IBKRFlexService.fetch_and_parse_statement(config)`
- Move last_import_date update: `IBKRConfigService.update_last_import_date()`

---

#### Violation 12: Inline IBKR transaction status update
**Location:** Lines 378-405 (`ignore_transaction` endpoint)
**Severity:** LOW
**Description:**
- Direct status field update with timestamp (lines 384-386)
- Inline database commit

**Current Code:**
```python
txn.status = "ignored"
txn.processed_at = datetime.now()
db.session.commit()
```

**Suggested Service Method:** `IBKRTransactionService.mark_transaction_as_ignored(transaction_id)` - encapsulates state mutation

---

#### Violation 13: Direct transaction deletion in delete endpoint
**Location:** Lines 419-445 (`delete_transaction` endpoint)
**Severity:** LOW
**Description:**
- Direct db.session.delete() call
- Simple but should be encapsulated

**Suggested Service Method:** `IBKRTransactionService.delete_ibkr_transaction(transaction_id)` already may exist, ensure it's used

---

#### Violation 14: Complex unallocation logic with cascade handling
**Location:** Lines 621-684 (`unallocate_transaction` endpoint)
**Severity:** CRITICAL
**Description:**
- 64 lines of complex cascade deletion logic
- Multiple model queries (IBKRTransactionAllocation, Transaction)
- Loop-based deletion logic (lines 637-644)
- Multiple state mutations (status, processed_at)
- Detailed error handling with transaction rollback

**Current Code:**
```python
allocations = IBKRTransactionAllocation.query.filter_by(...).all()
deleted_count = 0
for allocation in allocations:
    if allocation.transaction_id:
        transaction = db.session.get(Transaction, allocation.transaction_id)
        if transaction:
            db.session.delete(transaction)
            deleted_count += 1
ibkr_txn.status = "pending"
ibkr_txn.processed_at = None
db.session.commit()
```

**Suggested Service Method:** `IBKRTransactionService.unallocate_transaction(transaction_id)` - encapsulates entire cascade logic

---

#### Violation 15: Allocation detail aggregation in get_transaction_allocations
**Location:** Lines 701-746 (`get_transaction_allocations` endpoint)
**Severity:** MEDIUM
**Description:**
- Complex loop building allocation details (lines 710-728)
- Multiple relationship navigations
- Conditional transaction lookup and date formatting
- Data transformation logic mixed with query

**Current Code:**
```python
for allocation in allocations:
    transaction = (
        db.session.get(Transaction, allocation.transaction_id)
        if allocation.transaction_id
        else None
    )
    allocation_details.append({
        # ... multiple field transformations
        "transaction_date": transaction.date.isoformat() if transaction else None,
    })
```

**Suggested Service Method:** `IBKRTransactionService.get_allocation_details(transaction_id)` - returns fully formatted allocation list

---

#### Violation 16: Bulk allocation with loop and nested error handling
**Location:** Lines 813-918 (`bulk_allocate_transactions` endpoint)
**Severity:** HIGH
**Description:**
- 106 lines of complex batch processing logic
- Validation logic (lines 849-851)
- Nested try-except blocks (lines 858-883)
- Error aggregation and reporting
- Multiple sequential API calls

**Current Code:**
```python
total_percentage = sum(a.get("percentage", 0) for a in allocations)
if abs(total_percentage - 100) > 0.01:
    return jsonify({"error": "Allocations must sum to exactly 100%"}), 400

for transaction_id in transaction_ids:
    try:
        result = IBKRTransactionService.process_transaction_allocation(
            transaction_id, allocations
        )
        # ... error handling and aggregation
```

**Suggested Service Method:** `IBKRTransactionService.bulk_process_allocations(transaction_ids, allocations)` - encapsulates validation, iteration, and error aggregation

---

#### Violation 17: Allocation percentage validation repeated
**Location:** Lines 849-851 (`bulk_allocate_transactions` endpoint)
**Severity:** LOW
**Description:**
- Validation logic (sum to 100%) hardcoded in route
- Same validation likely exists in service but called per-transaction

**Suggested Enhancement:** Move to `IBKRTransactionService.validate_allocations(allocations)` static method, reuse in both endpoints

---

### system_routes.py

#### Violation 18: Direct database query in health check
**Location:** Lines 55-56 (`health_check` endpoint)
**Severity:** LOW
**Description:**
- Direct raw SQL query in route
- Should be encapsulated in service

**Current Code:**
```python
db.session.execute(db.text("SELECT 1"))
```

**Suggested Service Method:** `SystemService.check_database_health()` - returns connection status

---

### developer_routes.py

#### Violation 19: Exchange rate parsing and validation in endpoint
**Location:** Lines 52-55 (`get_exchange_rate` endpoint)
**Severity:** LOW
**Description:**
- Date parsing logic in route
- Conditional date handling

**Suggested Enhancement:** Move date parsing to `DeveloperService.get_exchange_rate()` as optional parameter

---

#### Violation 20: CSV header validation logic duplicated
**Location:** Lines 222-239 (`import_transactions` endpoint) and Lines 479-496 (`import_fund_prices` endpoint)
**Severity:** MEDIUM
**Description:**
- Identical CSV validation logic in two endpoints
- Header checking and comparison (lines 223-225, 480-482)
- Found in both places but not extracted

**Current Code (appears in both):**
```python
first_line = decoded_content.split("\n")[0].strip()
expected_headers = {"date", "type", "shares", "cost_per_share"}
found_headers = {h.strip() for h in first_line.split(",")}
if not expected_headers.issubset(found_headers):
    # ... error handling
```

**Suggested Service Method:** `DeveloperService.validate_csv_headers(content, expected_headers)` - reusable validation

---

#### Violation 21: CSV file validation logic duplicated
**Location:** Lines 207-215 (`import_transactions` endpoint), Lines 464-472 (`import_fund_prices` endpoint), Lines 499-507 (type checking)
**Severity:** MEDIUM
**Description:**
- File existence check repeated (lines 184-192, 441-449)
- File type validation repeated (lines 207-215, 464-472)
- File type detection logic (lines 499-507)

**Suggested Service Method:** `DeveloperService.validate_and_read_csv_file(request, expected_headers, file_field_name)` - consolidates all file handling

---

#### Violation 22: Complex logging settings persistence in update endpoint
**Location:** Lines 581-595 (`update_logging_settings` endpoint)
**Severity:** LOW
**Description:**
- Direct SystemSetting queries and mutations
- Conditional object creation
- Database persistence

**Current Code:**
```python
enabled_setting = SystemSetting.query.filter_by(...).first()
if not enabled_setting:
    enabled_setting = SystemSetting(key=...)
enabled_setting.value = str(data["enabled"]).lower()
# ... repeated for level_setting
db.session.add(enabled_setting)
db.session.add(level_setting)
db.session.commit()
```

**Suggested Service Method:** `DeveloperService.update_logging_settings(enabled, level)` or `SystemService.update_logging_configuration()`

---

#### Violation 23: Complex log filtering and pagination query
**Location:** Lines 634-700 (`get_logs` endpoint)
**Severity:** HIGH
**Description:**
- 67 lines of query building logic
- Multiple conditional filters (lines 645-662)
- Complex enum conversions (lines 647, 651)
- Sorting logic (lines 665-671)
- Pagination (lines 674-676)
- Response formatting loop (lines 680-695)

**Current Code:**
```python
query = Log.query
if levels:
    levels_list = levels.split(",")
    level_filters = [Log.level == LogLevel(lvl) for lvl in levels_list]
    query = query.filter(db.or_(*level_filters))
# ... 7 more filter blocks
if sort_dir == "desc":
    query = query.order_by(getattr(Log, sort_by).desc())
# ... pagination and formatting
```

**Suggested Service Method:** `DeveloperService.get_filtered_logs(filters, sort_by, sort_dir, page, per_page)` - encapsulates entire query building, pagination, and formatting

---

#### Violation 24: Fund price CSV file handling with type detection
**Location:** Lines 498-507 (`import_fund_prices` endpoint)
**Severity:** MEDIUM
**Description:**
- Duplicate file type detection logic
- Conditional error response for wrong file type

**Current Code:**
```python
if "type" in found_headers and "shares" in found_headers:
    response, status = logger.log(
        level=LogLevel.ERROR,
        category=LogCategory.SYSTEM,
        message="This appears to be a transaction file...",
        ...
    )
```

**Suggested Enhancement:** Extract to `DeveloperService.detect_csv_file_type(headers)` method

---

## Summary Statistics

Total Violations Found: 24
- Critical: 3 (bulk operations, IBKR import, unallocation)
- High: 3 (bulk price update, bulk allocation, log filtering)
- Medium: 11 (data enrichment, queries, validation, file handling)
- Low: 7 (simple encapsulation needed)

### Most Critical Areas
1. IBKR routes have significant business logic orchestration in endpoints
2. Bulk operations (fund prices, IBKR transactions) have complex error handling and aggregation
3. CSV file processing has duplicated validation logic
4. Complex deletion workflows need service encapsulation
