# IBKRConfigService Test Suite Documentation

**File**: `tests/services/test_ibkr_config_service.py`\
**Service**: `app/services/ibkr_config_service.py`\
**Tests**: 22 tests\
**Coverage**: 100%\
**Created**: Version 1.3.3 (Phase 3)\
**Updated**: Version 1.3.5 (Default Allocation Feature)

## Overview

Comprehensive test suite for the IBKRConfigService class, covering IBKR (Interactive Brokers) configuration management with complete test coverage. Tests all operations including configuration status retrieval, creation, updates, token encryption handling, and security features.

## Test Structure

### Test Classes

#### 1. TestConfigRetrieval (5 tests)
Tests configuration status and retrieval operations:
- `test_get_config_status_not_configured` - No configuration exists
- `test_get_config_status_configured` - Basic configuration status
- `test_get_config_status_with_expiration_far_future` - Token expires >30 days (no warning)
- `test_get_config_status_with_expiration_warning` - Token expires <30 days (shows warning)
- `test_get_config_status_with_last_import` - Configuration with import history

#### 2. TestConfigSave (6 tests)
Tests configuration creation and update operations:
- `test_save_config_create_new` - Create new configuration
- `test_save_config_with_all_fields` - Create with all optional fields
- `test_save_config_update_existing` - Update existing configuration
- `test_save_config_update_partial_fields` - Partial updates preserve other fields
- `test_save_config_token_encryption` - Token encryption functionality
- `test_save_config_disable_config` - Enable/disable configuration

#### 3. TestConfigDeletion (2 tests)
Tests configuration deletion operations:
- `test_delete_config_success` - Delete existing configuration
- `test_delete_config_not_found` - Error handling for missing configuration

#### 4. TestEdgeCases (5 tests)
Tests edge cases and boundary conditions:
- `test_get_config_status_token_expires_today` - Token expires same day
- `test_save_config_enabled_defaults_to_true` - Default enabled behavior
- `test_save_config_auto_import_disabled_by_default` - Default auto_import behavior
- `test_save_config_update_token_expires_at` - Update expiration dates
- `test_get_config_status_excludes_token` - Security: never expose encrypted tokens

#### 5. TestDefaultAllocationConfig (4 tests)
Tests default allocation preset functionality (added in v1.3.5):
- `test_save_config_with_default_allocations` - Save config with default allocation settings
- `test_get_config_status_includes_default_allocations` - Status includes allocation fields
- `test_update_config_default_allocations` - Update existing config with allocations
- `test_disable_default_allocation` - Disable allocation while keeping preset

## Testing Strategy

### Complete Coverage Achievement
This test suite achieves **100% code coverage** by testing:
- All public methods and their branches
- All conditional logic paths
- Token expiration warning calculations
- Default value assignments
- Error conditions and edge cases

### Security-First Testing
- **Token Encryption**: Mocked for testing, validates integration points
- **Token Exclusion**: Ensures encrypted tokens never appear in status responses
- **Input Validation**: Tests all parameter combinations and defaults

### Singleton Pattern Testing
IBKRConfig uses singleton pattern (one config per system):
- **Isolation Fixture**: `clean_ibkr_config` ensures test isolation
- **Update vs Create**: Tests proper detection of existing configurations
- **State Management**: Validates configuration state transitions

## Service Methods Tested

### Configuration Status
- `get_config_status()` - Get current configuration status:
  - Returns `{"configured": False}` when no config exists
  - Returns full status with security exclusions when configured
  - Calculates token expiration warnings (< 30 days)
  - Formats dates in ISO format for API consumption

### Configuration Management
- `save_config(flex_token, flex_query_id, **options)` - Create or update:
  - Creates new configuration if none exists
  - Updates existing configuration preserving unspecified fields
  - Encrypts tokens using IBKRFlexService integration
  - Handles optional parameters with proper defaults

### Configuration Deletion
- `delete_config()` - Remove configuration:
  - Returns configuration details before deletion
  - Raises ValueError if no configuration exists
  - Completely removes configuration from system

## Security Features Tested

### Token Encryption Integration
```python
# Mocked encryption for testing
@pytest.fixture
def mock_encryption():
    with patch('app.services.ibkr_flex_service.IBKRFlexService._encrypt_token'):
        mock_encrypt.side_effect = lambda token: f"encrypted_{token}"
        yield mock_encrypt, mock_decrypt
```

### Token Security Validation
- **Never Exposed**: Encrypted tokens never appear in status responses
- **Encryption Required**: All token storage uses encrypted values
- **Secure Defaults**: Configuration defaults prioritize security

### Expiration Warning System
- **30-Day Threshold**: Warnings appear when tokens expire within 30 days
- **Dynamic Calculation**: Uses current datetime for accurate warnings
- **User-Friendly Messages**: Clear expiration information

## Configuration Model Tested

### IBKRConfig Fields
```python
config = IBKRConfig(
    flex_token="encrypted_token_value",      # Encrypted IBKR token
    flex_query_id="123456",                  # IBKR Flex Query ID
    token_expires_at=datetime.now() + timedelta(days=90),  # Optional
    last_import_date=datetime.now(),         # Optional import tracking
    auto_import_enabled=False,               # Default: False
    enabled=True,                           # Default: True
    created_at=datetime.now(),              # Auto-populated
    updated_at=datetime.now()               # Auto-updated
)
```

### Status Response Format
```python
{
    "configured": True,
    "flex_query_id": "123456",
    "token_expires_at": "2024-04-15T10:30:00",  # ISO format
    "token_warning": "Token expires in 15 days",  # If < 30 days
    "last_import_date": "2024-01-15T10:30:00",   # ISO format
    "auto_import_enabled": False,
    "enabled": True,
    "created_at": "2024-01-01T00:00:00",
    "updated_at": "2024-01-15T10:30:00"
    # Note: flex_token is NEVER included for security
}
```

## Error Scenarios Tested

### Configuration Not Found
1. **Status Check**: Returns `{"configured": False}` gracefully
2. **Deletion**: Raises `ValueError("No configuration found")`
3. **Update Safety**: Creates new configuration if none exists

### Token Expiration Handling
1. **Far Future**: No warning for tokens expiring > 30 days
2. **Near Future**: Warning message for tokens expiring < 30 days
3. **Same Day**: Handles tokens expiring within hours

### Data Validation
1. **Required Fields**: Validates flex_token and flex_query_id
2. **Optional Fields**: Proper default handling for optional parameters
3. **Boolean Logic**: Correct handling of enabled/disabled states

## Advanced Features Tested

### Partial Updates
The service supports partial updates that preserve existing values:
```python
# Only updates query_id, preserves other fields
updated_config = IBKRConfigService.save_config(
    flex_token="new_token",
    flex_query_id="new_query_id"
    # auto_import_enabled and enabled are preserved
)
```

### Token Change Detection
All token updates require re-encryption:
```python
# Every save_config call encrypts the provided token
config = IBKRConfigService.save_config(
    flex_token="plaintext_token",  # Always encrypted before storage
    flex_query_id="123456"
)
assert config.flex_token != "plaintext_token"  # Stored encrypted
```

### Expiration Warning Logic
Dynamic warning calculation based on current time:
```python
days_until_expiry = (config.token_expires_at - datetime.now()).days
if days_until_expiry < 30:
    token_warning = f"Token expires in {days_until_expiry} days"
```

## Test Isolation Strategy

### Auto-Cleanup Fixture
```python
@pytest.fixture(autouse=True)
def clean_ibkr_config(db_session):
    """Clean up IBKRConfig before each test for isolation."""
    IBKRConfig.query.delete()
    db_session.commit()
    yield
    IBKRConfig.query.delete()
    db_session.commit()
```

### Benefits of Isolation
- **No Test Pollution**: Each test starts with clean slate
- **Predictable State**: No dependencies on test execution order
- **Singleton Testing**: Proper testing of singleton configuration pattern

## Integration Points

### IBKRFlexService Integration
- **Token Encryption**: Delegates to IBKRFlexService for secure token storage
- **Decryption Support**: Integration point for token retrieval when needed
- **Error Handling**: Proper error propagation from encryption service

### Datetime Handling
- **ISO Formatting**: All datetime fields formatted for API consumption
- **Timezone Awareness**: Handles naive datetime objects from SQLite
- **Dynamic Calculations**: Real-time expiration warning calculations

### Database Relationships
- **Singleton Pattern**: Only one IBKRConfig record allowed per system
- **Audit Trail**: created_at and updated_at tracking
- **Safe Deletion**: Complete removal with proper error handling

## Security Considerations

### Token Protection
1. **Never Exposed**: Tokens never appear in status responses
2. **Always Encrypted**: All tokens encrypted before database storage
3. **No Logging**: Sensitive data excluded from logs and responses

### Access Control
1. **System-Wide Config**: One configuration affects entire system
2. **Enable/Disable**: Master toggle for IBKR integration
3. **Import Control**: Separate toggle for automated imports

## Performance Considerations

### Efficient Operations
- **Single Query**: Status check uses single database query
- **Minimal Updates**: Only touches database when changes occur
- **Fast Lookups**: Simple primary key operations for configuration access

### Resource Management
- **Small Footprint**: Singleton pattern minimizes database usage
- **Quick Responses**: Status checks return immediately
- **Efficient Updates**: Only updates specified fields

## Default Allocation Feature (v1.3.5)

### Overview
Version 1.3.5 introduced configurable default allocations for automated IBKR imports. This feature allows users to define a preset allocation strategy that automatically applies to imported transactions.

### New Configuration Fields
```python
config = IBKRConfig(
    # ... existing fields ...
    default_allocation_enabled=False,     # Toggle for default allocation feature
    default_allocations=None             # JSON: [{"portfolio_id": "...", "percentage": 60.0}]
)
```

### Test Coverage for Default Allocation

#### Test 1: Save Config with Default Allocations
**Purpose**: Verify saving configuration with default allocation settings
```python
def test_save_config_with_default_allocations():
    allocations_json = (
        '[{"portfolio_id": "p1", "percentage": 60.0}, '
        '{"portfolio_id": "p2", "percentage": 40.0}]'
    )
    config = IBKRConfigService.save_config(
        flex_token="test_token",
        flex_query_id="123456",
        default_allocation_enabled=True,
        default_allocations=allocations_json,
    )
    assert config.default_allocation_enabled is True
    assert config.default_allocations == allocations_json
```

#### Test 2: Status Includes Default Allocations
**Purpose**: Ensure status endpoint returns default allocation fields
```python
def test_get_config_status_includes_default_allocations():
    # Status response includes new fields
    status = IBKRConfigService.get_config_status()
    assert status["default_allocation_enabled"] is True
    assert status["default_allocations"] == allocations_json
```

#### Test 3: Update Default Allocations
**Purpose**: Verify updating existing config with allocation changes
**Scenario**: Config starts with allocations disabled, then enabled
**Validates**: Partial updates work correctly, preserving other fields

#### Test 4: Disable Default Allocation
**Purpose**: Verify disabling allocation while keeping preset
**Scenario**: Enable allocations, configure preset, then disable
**Validates**: Preset is preserved when disabled (can re-enable without reconfiguring)

### Integration with Automated Import
The default allocation feature integrates with the automated IBKR import task:
1. Import completes successfully
2. If `default_allocation_enabled` is True and `default_allocations` is set
3. System automatically applies allocations to all pending transactions
4. Transactions are allocated using the configured percentages
5. Failed allocations are logged but don't stop other transactions

See `app/tasks/ibkr_import.py` for implementation details.

## Future Enhancements

1. **Token Rotation**: Automated token refresh before expiration
2. **Audit Logging**: Track all configuration changes
3. **Backup Config**: Support for configuration backup/restore
4. **Health Monitoring**: Integration with system monitoring
5. **Multi-Environment**: Support for different environments
6. **Allocation Validation**: Validate that portfolios exist before saving preset

## Bug Prevention

This comprehensive test suite prevents:
- **Security Leaks**: Token exposure in responses or logs
- **Configuration Loss**: Accidental deletion or corruption
- **Integration Failures**: Proper encryption service integration
- **UI Issues**: Correct status formatting for frontend consumption
- **Business Logic Errors**: Proper warning calculations and state management

The 100% coverage provides complete confidence in the IBKR configuration management system and ensures secure, reliable operation of the Interactive Brokers integration.
