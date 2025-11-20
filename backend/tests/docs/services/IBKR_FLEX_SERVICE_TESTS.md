# IBKRFlexService Test Suite Documentation

**File**: `tests/test_ibkr_flex_service.py`\
**Service**: `app/services/ibkr_flex_service.py`\
**Tests**: 31 tests\
**Coverage**: 77% (116/150 statements)\
**Created**: Version 1.3.3 (Phase 4)

## Overview

Comprehensive test suite for the IBKRFlexService class, which handles integration with Interactive Brokers (IBKR) Flex Web Service API. This service manages:
- Token encryption/decryption for secure storage
- Flex report retrieval via HTTP API
- XML statement parsing and validation
- Transaction import with duplicate detection
- Cache management for performance
- Multi-currency statement support

The test suite achieves 77% coverage, testing all public methods, error handling paths, and integration points with the IBKR API.

## Test Structure

### Test Classes

#### 1. TestTokenEncryption (3 tests)
Tests token encryption and decryption functionality:
- `test_encrypt_token` - Encrypts plaintext token
- `test_decrypt_token` - Decrypts encrypted token
- `test_encrypt_decrypt_roundtrip` - Full encryption cycle validation

#### 2. TestFlexStatementRetrieval (8 tests)
Tests IBKR Flex API interaction and HTTP communication:
- `test_request_statement_success` - Successful statement request
- `test_request_statement_error_1003` - Statement not ready (transient error)
- `test_request_statement_error_1012` - Invalid query ID
- `test_request_statement_error_1015` - Query not found
- `test_fetch_statement_success` - Successful XML fetch
- `test_fetch_statement_error` - Fetch failure handling
- `test_fetch_statement_bypass_cache` - Cache bypass behavior
- `test_get_statement_complete_flow` - End-to-end statement retrieval

#### 3. TestStatementParsing (6 tests)
Tests XML parsing and transaction extraction:
- `test_parse_statement_success` - Valid XML parsing
- `test_parse_statement_buy_transaction` - Buy transaction parsing
- `test_parse_statement_sell_transaction` - Sell transaction parsing
- `test_parse_statement_dividend_transaction` - Dividend transaction parsing
- `test_parse_statement_fee_transaction` - Fee transaction parsing
- `test_parse_statement_missing_currency` - Default currency handling

#### 4. TestStatementImport (6 tests)
Tests transaction import with duplicate detection:
- `test_import_statement_success` - Import new transactions
- `test_import_statement_duplicate_detection` - Skip existing transactions
- `test_import_statement_partial_duplicates` - Import only new transactions
- `test_import_statement_mixed_types` - Multiple transaction types
- `test_import_statement_empty` - Handle empty statements
- `test_import_statement_multi_currency` - Multiple currencies in one statement

#### 5. TestCacheManagement (5 tests)
Tests cache creation, retrieval, and cleanup:
- `test_cache_created_on_fetch` - Cache entry creation
- `test_cache_prevents_refetch` - Cache hit behavior
- `test_cache_cleanup_deletes_old_entries` - Automatic cleanup
- `test_cache_cleanup_preserves_recent_entries` - Recent cache preservation
- `test_cache_stores_statement_metadata` - Metadata storage

#### 6. TestEdgeCases (3 tests)
Tests error conditions and boundary cases:
- `test_get_statement_with_encryption_error` - Encryption failure handling
- `test_import_statement_parse_error` - Invalid XML handling
- `test_request_statement_http_error` - HTTP request failure

## Testing Strategy

### API Mocking with `responses`
All HTTP requests to IBKR API are mocked using the `responses` library:
```python
import responses

@responses.activate
def test_request_statement_success(self, app_context, ibkr_service):
    responses.add(
        responses.GET,
        "https://gdcdyn.interactivebrokers.com/Universal/servlet/FlexStatementService.SendRequest",
        body="<Status>Success</Status><ReferenceCode>1234567890</ReferenceCode>",
        status=200
    )
```

**Benefits**:
- No actual API calls during testing
- Predictable responses for all scenarios
- Fast test execution
- Can test error conditions easily

### Encryption Key Management
Tests use a centrally generated Fernet key from `test_config.py`:
```python
# In test_config.py
from cryptography.fernet import Fernet

TEST_ENCRYPTION_KEY = Fernet.generate_key().decode()

TEST_CONFIG = {
    "TESTING": True,
    # ... other config ...
    "IBKR_ENCRYPTION_KEY": TEST_ENCRYPTION_KEY,
}

# In test_ibkr_flex_service.py
@pytest.fixture
def ibkr_service(app_context):
    """
    Create IBKRFlexService instance with test config.

    Note: IBKR_ENCRYPTION_KEY is already set in TEST_CONFIG
    and applied during app creation.
    """
    return IBKRFlexService()
```

**Why this matters**:
- Tests actual encryption behavior, not mocked
- Validates key format requirements
- Ensures roundtrip encryption works correctly
- Consistent encryption key across all tests in the session

### Test Isolation
Each test uses unique identifiers to avoid conflicts:
```python
unique_query_id = f"query_{uuid.uuid4()}"
unique_reference_code = f"ref_{uuid.uuid4()}"
```

**Benefits**:
- No test pollution from shared data
- Tests can run in any order
- Parallel test execution safe
- Cache collisions prevented

## Service Methods Tested

### Token Security
- `_encrypt_token(token)` - Encrypts plaintext token using Fernet encryption
- `_decrypt_token(encrypted_token)` - Decrypts token for API use
- **Security**: Tokens never stored in plaintext, encryption validated

### Statement Retrieval
- `request_statement(query_id, token)` - Step 1: Request statement from IBKR
  - Returns reference code for fetching
  - Handles IBKR error codes (1003, 1012, 1015)
  - Validates response XML format

- `fetch_statement(reference_code, token)` - Step 2: Fetch statement XML
  - Retrieves XML using reference code
  - Creates cache entry
  - Returns raw XML string

- `get_statement(query_id, token, bypass_cache)` - Complete retrieval flow
  - Combines request + fetch steps
  - Checks cache before requesting
  - Handles full error flow

### Statement Processing
- `parse_statement(xml_content)` - Parse XML to transaction list
  - Extracts all transaction types
  - Handles missing optional fields
  - Returns structured transaction dictionaries

- `import_statement(xml_content)` - Import transactions to database
  - Creates IBKRTransaction records
  - Duplicate detection by transaction_id
  - Handles mixed transaction types
  - Multi-currency support

### Cache Management
- `cache_statement(query_id, reference_code, xml, metadata)` - Store in cache
  - Creates IBKRImportCache entry
  - Stores metadata (transaction counts, date ranges)
  - Sets creation timestamp

- `cleanup_old_cache_entries()` - Remove expired cache
  - Deletes entries older than 30 days
  - Preserves recent entries
  - Automatic cleanup

## IBKR API Error Codes Tested

### Error 1003 - Statement Not Ready
**Meaning**: Statement generation in progress, retry later\
**Response**: Transient error, user should retry\
**Tested**: `test_request_statement_error_1003`

```python
# IBKR returns:
<Status>Fail</Status>
<ErrorCode>1003</ErrorCode>
<ErrorMessage>Statement generation in progress</ErrorMessage>

# Service raises:
ValueError("IBKR Error 1003: Statement generation in progress. Please try again...")
```

### Error 1012 - Invalid Query ID
**Meaning**: Query ID format is wrong\
**Response**: User must correct the query ID\
**Tested**: `test_request_statement_error_1012`

```python
# IBKR returns:
<ErrorCode>1012</ErrorCode>

# Service raises:
ValueError("IBKR Error 1012: Invalid Flex Query ID")
```

### Error 1015 - Query Not Found
**Meaning**: Query ID doesn't exist or not accessible\
**Response**: User must verify query exists and is accessible\
**Tested**: `test_request_statement_error_1015`

```python
# IBKR returns:
<ErrorCode>1015</ErrorCode>

# Service raises:
ValueError("IBKR Error 1015: Flex Query not found")
```

## XML Statement Format

### Transaction Types Supported
```xml
<FlexStatement>
  <FlexStatements>
    <FlexStatement accountId="U1234567" ...>
      <Trades>
        <!-- Buy Transaction -->
        <Trade transactionID="12345" ibOrderID="67890"
               symbol="AAPL" description="APPLE INC"
               tradeDate="2024-01-15" tradeTime="143000"
               quantity="10" tradePrice="150.00"
               ibCommission="-1.00" netCash="-1501.00"
               cost="1501.00" fifoPnlRealized="0"
               buySell="BUY" openCloseIndicator="O"
               currency="USD" isin="US0378331005" />

        <!-- Sell Transaction -->
        <Trade transactionID="12346" quantity="-5"
               buySell="SELL" tradePrice="155.00"
               netCash="773.50" fifoPnlRealized="25.00" />

        <!-- Dividend -->
        <Trade transactionID="12347"
               assetCategory="STK" type="Dividend"
               quantity="0" netCash="50.00" />

        <!-- Fee -->
        <Trade transactionID="12348"
               type="Commission"
               quantity="0" netCash="-10.00" />
      </Trades>
    </FlexStatement>
  </FlexStatements>
</FlexStatement>
```

### Multi-Currency Support
The service handles statements with multiple currencies:
```python
# Test validates parsing of USD, EUR, GBP in single statement
transactions = [
    {"currency": "USD", "symbol": "AAPL", "total_amount": -1501.00},
    {"currency": "EUR", "symbol": "BMW", "total_amount": -5250.00},
    {"currency": "GBP", "symbol": "HSBA", "total_amount": -780.00},
]
```

**Tested**: `test_import_statement_multi_currency`

### Missing Currency Default
If currency field is missing, defaults to USD:
```python
# XML without currency attribute
<Trade transactionID="12345" symbol="AAPL" ... />

# Parsed as:
{"currency": "USD", "symbol": "AAPL", ...}
```

**Tested**: `test_parse_statement_missing_currency`

## Duplicate Detection

### Transaction ID Matching
Duplicates detected by IBKR `transactionID` field:
```python
# First import: Creates new record
txn1 = IBKRTransaction(
    ibkr_transaction_id="12345",
    symbol="AAPL",
    ...
)

# Second import: Skips duplicate
if IBKRTransaction.query.filter_by(ibkr_transaction_id="12345").first():
    continue  # Skip this transaction
```

**Tested**:
- `test_import_statement_duplicate_detection` - All duplicates
- `test_import_statement_partial_duplicates` - Mixed new/existing

### Import Results
Service returns import statistics:
```python
result = {
    "success": True,
    "imported_count": 3,      # New transactions created
    "duplicate_count": 2,     # Existing transactions skipped
    "total_count": 5          # Total in statement
}
```

## Cache Management

### Cache Entry Structure
```python
IBKRImportCache(
    cache_key="query_12345_ref_67890",  # Unique key
    query_id="12345",
    reference_code="67890",
    statement_xml="<FlexStatement>...</FlexStatement>",
    metadata={
        "transaction_count": 10,
        "date_range": "2024-01-01 to 2024-01-31",
        "currencies": ["USD", "EUR"]
    },
    created_at=datetime(2024, 1, 15, 10, 30, 0)
)
```

### Cache Expiration
- **Lifetime**: 30 days from creation
- **Cleanup**: Automatic via `cleanup_old_cache_entries()`
- **Bypass**: `get_statement(bypass_cache=True)` forces fresh fetch

**Tested**:
- `test_cache_prevents_refetch` - Cache hit behavior
- `test_fetch_statement_bypass_cache` - Cache bypass
- `test_cache_cleanup_deletes_old_entries` - Expiration
- `test_cache_cleanup_preserves_recent_entries` - Preservation

## Error Scenarios Tested

### Encryption Errors
**Test**: `test_get_statement_with_encryption_error`
- Simulates encryption service failure
- Validates error propagation
- Ensures graceful failure

### XML Parse Errors
**Test**: `test_import_statement_parse_error`
- Provides invalid XML
- Service raises ValueError with "Failed to parse statement"
- Database transaction rolled back (no partial imports)

### HTTP Request Failures
**Test**: `test_request_statement_http_error`
- Simulates network failure (ConnectionError)
- Service raises exception
- No cache entry created

### Empty Statements
**Test**: `test_import_statement_empty`
- Statement XML with zero transactions
- Import succeeds with 0 imported
- No database changes

## Coverage Analysis

### Current Coverage: 77% (116/150 statements)

**Well-Covered Areas**:
- ✅ All public methods (100%)
- ✅ Token encryption/decryption (100%)
- ✅ Statement retrieval flow (100%)
- ✅ XML parsing (95%)
- ✅ Duplicate detection (100%)
- ✅ Cache management (85%)
- ✅ Error handling (80%)

**Uncovered Lines** (34 statements):
- Lines 185-195: Advanced XML parsing edge cases (nested structures)
- Lines 240-245: Rare XML attribute combinations
- Lines 310-320: Database rollback error handling
- Lines 380-385: Network timeout handling (requires complex mocking)

**Why 77% is acceptable**:
1. Exceeds 75% minimum target
2. All critical paths tested (encryption, API, imports)
3. Uncovered lines are extreme edge cases
4. Would require complex mocking with diminishing returns
5. Focus moved to other Phase 4 services

**Path to 80%+** (if needed in future):
- Add tests for nested XML structures
- Mock database failures for rollback paths
- Test timeout scenarios with `responses` timeout parameter
- Add tests for malformed XML variations

## Running Tests

### Run All IBKRFlexService Tests
```bash
pytest tests/test_ibkr_flex_service.py -v
```

### Run Specific Test Class
```bash
pytest tests/test_ibkr_flex_service.py::TestFlexStatementRetrieval -v
```

### Run with Coverage
```bash
pytest tests/test_ibkr_flex_service.py --cov=app/services/ibkr_flex_service --cov-report=term-missing
```

### Run Single Test
```bash
pytest tests/test_ibkr_flex_service.py::TestStatementImport::test_import_statement_success -v
```

## Integration Points

### IBKRConfigService Integration
IBKRFlexService works with IBKRConfigService for configuration:
```python
# Config provides:
config = IBKRConfigService.get_config_status()
query_id = config["flex_query_id"]
token = config["flex_token"]  # Encrypted

# Flex service uses:
statement = IBKRFlexService.get_statement(query_id, token)
```

### Database Models
Creates records in:
- **IBKRTransaction** - Individual transactions from statements
- **IBKRImportCache** - Cached statements for performance

### Logging Integration
All operations logged via `logging_service`:
```python
logger.log(
    level=LogLevel.INFO,
    category=LogCategory.IBKR,
    message="Successfully imported IBKR statement",
    details={"imported": 10, "duplicates": 2}
)
```

## Security Considerations

### Token Protection
1. **Encryption at Rest**: All tokens encrypted in database using Fernet
2. **No Plaintext Logging**: Tokens never appear in logs or errors
3. **Memory Safety**: Tokens decrypted only when needed for API calls
4. **Test Isolation**: Tests use separate encryption keys

### XML Injection Prevention
- Uses standard library `xml.etree.ElementTree` (safe against XXE)
- No dynamic XML generation from user input
- All parsing done on IBKR-provided XML only

### API Security
- HTTPS endpoints only (enforced by IBKR)
- No credential storage (uses tokens only)
- Tokens have expiration dates (validated in IBKRConfigService)

## Performance Considerations

### Caching Strategy
- **Cache Hit**: Statement retrieval ~10ms (database query)
- **Cache Miss**: Statement retrieval ~2-5 seconds (IBKR API roundtrip)
- **30-Day TTL**: Balances freshness with performance

### Batch Import Efficiency
- Single database transaction per import
- Bulk duplicate checking with single query
- Minimal database roundtrips

### Test Performance
- **31 tests**: Complete suite runs in ~0.8 seconds
- **Mocked API**: No network latency in tests
- **Isolated Data**: No cleanup overhead between tests

## Future Enhancements

1. **Automatic Retry**: Retry on error 1003 (statement not ready)
2. **Webhook Support**: IBKR webhook integration for automatic imports
3. **Statement Validation**: Schema validation for XML structure
4. **Performance Metrics**: Track import times and cache hit rates
5. **Multi-Account**: Support for multiple IBKR accounts

## Related Documentation

- **Service Code**: `app/services/ibkr_flex_service.py`
- **Related Tests**: `tests/test_ibkr_config_service.py`, `tests/test_ibkr_transaction_service.py`
- **Test Documentation**: `IBKR_CONFIG_SERVICE_TESTS.md`, `IBKR_TRANSACTION_SERVICE_TESTS.md`
- **Bug Fixes**: `BUG_FIXES_1.3.3.md`
- **Models**: `app/models.py` (IBKRTransaction, IBKRImportCache)

The comprehensive test suite provides confidence in the IBKR Flex integration, ensuring secure token handling, reliable statement imports, and proper error handling for production use.
