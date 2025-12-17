# SystemService Test Documentation

## Overview

The SystemService handles system-level functionality including version management, migration status checking, and feature availability determination. This service is critical for ensuring proper application/database compatibility and preventing user-facing errors when versions are mismatched.

**Test Coverage**: 100% (79/79 statements) ✅
**Test Count**: 32 comprehensive tests
**Bugs Found**: 1 (IBKR integration version check logic error)
**Updated**: Version 1.3.5 (Dynamic versioning for tests)

## Test Organization

Tests are organized into three main classes:

### `TestSystemService` (27 tests)
Core functionality testing including version retrieval, migration checking, and feature availability.

### `TestSystemServiceLogging` (4 tests)
Verification that all error conditions and operations are properly logged.

### `TestSystemServiceIntegration` (3 tests)
Integration tests with real file system and database operations.

## Service-Specific Patterns

### Dynamic Version Detection (v1.3.5)
Tests now dynamically determine the latest migration version instead of hardcoding values:

```python
def get_latest_migration_version():
    """Get the latest migration version from migrations directory."""
    migrations_dir = Path(__file__).parent.parent.parent / "migrations" / "versions"
    migration_files = list(migrations_dir.glob("[0-9]*.py"))
    versions = [file.name.split("_")[0] for file in migration_files]
    versions.sort(key=lambda v: [int(x) for x in v.split(".")])
    return versions[-1]

LATEST_MIGRATION_VERSION = get_latest_migration_version()
```

**Benefits**:
- Tests automatically adapt to version bumps
- No need to manually update hardcoded version strings
- Reduces test maintenance overhead
- Prevents false failures when creating new migrations

### Database Setup Pattern
Since SystemService queries the `alembic_version` table which doesn't exist in test databases, most tests use this pattern:

```python
def test_with_database(self, app_context, db_session):
    # Create alembic_version table and insert latest migration version
    db_session.execute(db.text("CREATE TABLE IF NOT EXISTS alembic_version (version_num VARCHAR(32) NOT NULL)"))
    db_session.execute(db.text("DELETE FROM alembic_version"))
    db_session.execute(
        db.text(f"INSERT INTO alembic_version (version_num) VALUES ('{LATEST_MIGRATION_VERSION}')")
    )
    db_session.commit()
    # Test continues...
```

**Note**: Uses `LATEST_MIGRATION_VERSION` constant instead of hardcoded version string.

### Mock/Patch Testing Pattern
For error conditions and edge cases, extensive mocking is used:

```python
with patch("builtins.open", side_effect=FileNotFoundError("File not found")):
    version = SystemService.get_app_version()
    assert version == "unknown"
```

### No Factory Pattern
SystemService doesn't use factory patterns since it deals with system-level data (VERSION file, alembic tables) rather than business entities.

## Test Suite Walkthrough

### Core Version Management Tests

#### Test 1: `test_get_app_version_success`
**Purpose**: Verify successful application version retrieval from VERSION file

**Scenario**: Read actual VERSION file in repository
**Test Data**: Uses real VERSION file (currently "1.3.2")
**Action**: `SystemService.get_app_version()`
**Expected**: Returns "1.3.2", not "unknown", in valid version format
**Why it matters**: Core functionality - app needs to know its own version

**Code walkthrough**:
```python
def test_get_app_version_success(self, app_context):
    version = SystemService.get_app_version()

    assert version != "unknown"  # ← Should successfully read file
    assert isinstance(version, str)  # ← Must be string
    assert len(version.split(".")) >= 2  # ← Must be major.minor format
    assert version.startswith("1.3")  # ← Current version family
```

#### Test 2: `test_get_app_version_file_not_found`
**Purpose**: Handle missing VERSION file gracefully

**Scenario**: VERSION file doesn't exist
**Test Data**: Mock FileNotFoundError
**Action**: `SystemService.get_app_version()`
**Expected**: Returns "unknown" without crashing
**Why it matters**: Defensive programming - prevents crashes during deployment issues

#### Test 3: `test_get_app_version_read_error`
**Purpose**: Handle VERSION file permission errors

**Scenario**: VERSION file exists but can't be read
**Test Data**: Mock PermissionError
**Action**: `SystemService.get_app_version()`
**Expected**: Returns "unknown" without crashing
**Why it matters**: Handles filesystem permission issues gracefully

#### Test 4: `test_get_db_version_success`
**Purpose**: Verify successful database version retrieval

**Scenario**: Database has alembic_version table with valid version
**Test Data**: Creates table with version "1.3.1"
**Action**: `SystemService.get_db_version()`
**Expected**: Returns "1.3.1" from database
**Why it matters**: Database version is critical for migration decisions

**Code walkthrough**:
```python
def test_get_db_version_success(self, app_context, db_session):
    # ARRANGE - Set up test database with version
    db_session.execute(db.text("CREATE TABLE IF NOT EXISTS alembic_version (version_num VARCHAR(32) NOT NULL)"))
    db_session.execute(db.text("DELETE FROM alembic_version"))
    db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
    db_session.commit()

    # ACT
    version = SystemService.get_db_version()

    # ASSERT
    assert version == "1.3.1"  # ← Should match inserted version
```

#### Test 5: `test_get_db_version_no_table`
**Purpose**: Handle missing alembic_version table

**Scenario**: Database doesn't have alembic_version table
**Test Data**: Mock database exception
**Action**: `SystemService.get_db_version()`
**Expected**: Returns "unknown" without crashing
**Why it matters**: Handles uninitialized databases gracefully

#### Test 6: `test_get_db_version_no_result`
**Purpose**: Handle empty alembic_version table

**Scenario**: Table exists but has no version records
**Test Data**: Mock empty query result
**Action**: `SystemService.get_db_version()`
**Expected**: Returns "unknown"
**Why it matters**: Handles partially initialized databases

### Migration Status Tests

#### Test 7: `test_check_pending_migrations_no_pending`
**Purpose**: Verify correct behavior when no migrations are pending

**Scenario**: Database is at current head revision
**Test Data**: Creates alembic_version with "1.3.1" (current head)
**Action**: `SystemService.check_pending_migrations()`
**Expected**: Returns (False, None)
**Why it matters**: **Core fix** - prevents false migration warnings

**Code walkthrough**:
```python
def test_check_pending_migrations_no_pending(self, app_context, db_session):
    # ARRANGE - Set database to current head
    db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
    db_session.commit()

    # ACT
    has_pending, error = SystemService.check_pending_migrations()

    # ASSERT
    assert has_pending is False  # ← No migrations needed
    assert error is None  # ← No error occurred
```

#### Test 8: `test_check_pending_migrations_with_pending`
**Purpose**: Verify detection of actual pending migrations

**Scenario**: Database revision is behind head revision
**Test Data**: Mock current="1.2.0", head="1.3.1"
**Action**: `SystemService.check_pending_migrations()`
**Expected**: Returns (True, None)
**Why it matters**: Ensures real migration needs are detected

#### Test 9: `test_check_pending_migrations_uninitialized_db`
**Purpose**: Handle completely uninitialized database

**Scenario**: Database has no revision history
**Test Data**: Mock current_revision=None
**Action**: `SystemService.check_pending_migrations()`
**Expected**: Returns (True, "Database not initialized")
**Why it matters**: Provides clear messaging for setup issues

#### Test 10: `test_check_pending_migrations_error`
**Purpose**: Handle Alembic errors gracefully

**Scenario**: Alembic configuration or connection fails
**Test Data**: Mock database connection error
**Action**: `SystemService.check_pending_migrations()`
**Expected**: Returns (True, "Error checking migrations: ...")
**Why it matters**: Prevents crashes during infrastructure issues

### Feature Availability Tests

#### Test 11-17: Version-Specific Feature Tests
**Purpose**: Verify correct feature flags for different database versions

**Test Data**:
- "unknown" → All features False (except basic)
- "1.0.0" → Only basic_portfolio_management
- "1.1.0" → basic + exclude_from_overview
- "1.1.1" → basic + exclude + realized_gain_loss
- "1.3.0" → All features enabled
- "1.3.1" → All features enabled (current)
- "v1.3.1" → All features enabled (handles 'v' prefix)

**Why it matters**: Feature flags control UI availability and prevent crashes when users try to use features not supported by their database schema.

#### Test 18: `test_check_feature_availability_invalid_version`
**Purpose**: Handle malformed version strings gracefully

**Scenario**: Version string can't be parsed
**Test Data**: "invalid.version" string
**Action**: `SystemService.check_feature_availability()`
**Expected**: Returns default features (basic only)
**Why it matters**: Defensive programming against corrupted version data

#### Test 19: `test_check_feature_availability_major_version_2`
**Purpose**: **Bug fix validation** - Ensure future major versions work correctly

**Scenario**: Database at hypothetical version "2.0.0"
**Test Data**: "2.0.0"
**Action**: `SystemService.check_feature_availability()`
**Expected**: All features enabled
**Why it matters**: **Bug discovered during testing** - original logic `major >= 1 and minor >= 3` would fail for major version 2, returning `ibkr_integration: False` incorrectly.

**Bug Fix Applied**:
```python
# OLD (buggy):
if major >= 1 and minor >= 3:
    features["ibkr_integration"] = True

# NEW (fixed):
if major > 1 or (major == 1 and minor >= 3):
    features["ibkr_integration"] = True
```

### Complete Version Info Tests

#### Test 20: `test_get_version_info_no_migrations_needed`
**Purpose**: Test complete version info assembly when system is up-to-date

**Scenario**: App 1.3.2, DB 1.3.1, no pending migrations
**Test Data**: Sets up database at 1.3.1
**Action**: `SystemService.get_version_info()`
**Expected**: Complete dictionary with migration_needed=False
**Why it matters**: **Primary use case** - this is the normal operating state

**Code walkthrough**:
```python
def test_get_version_info_no_migrations_needed(self, app_context, db_session):
    # ARRANGE
    db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
    db_session.commit()

    # ACT
    version_info = SystemService.get_version_info()

    # ASSERT - Verify structure
    required_keys = {"app_version", "db_version", "features", "migration_needed", "migration_message"}
    assert set(version_info.keys()) == required_keys  # ← Complete response

    # ASSERT - Verify values
    assert version_info["migration_needed"] is False  # ← Key fix
    assert version_info["migration_message"] is None  # ← No warning
```

#### Test 21: `test_get_version_info_migrations_needed`
**Purpose**: Test version info when migrations are actually needed

**Scenario**: Database behind, real migrations pending
**Test Data**: Mock pending migrations detected
**Action**: `SystemService.get_version_info()`
**Expected**: migration_needed=True with helpful message
**Why it matters**: Ensures users get actionable guidance

#### Test 22: `test_get_version_info_migration_error`
**Purpose**: Handle migration check errors in version info

**Scenario**: Migration check fails with error
**Test Data**: Mock migration error
**Action**: `SystemService.get_version_info()`
**Expected**: migration_needed=True, message=error details
**Why it matters**: Propagates technical errors to user for troubleshooting

#### Test 23: `test_get_version_info_unknown_db_version`
**Purpose**: Handle unknown database version in complete info

**Scenario**: Database version can't be determined
**Test Data**: Mock db_version="unknown"
**Action**: `SystemService.get_version_info()`
**Expected**: Specific "could not be determined" message
**Why it matters**: Different messaging for version detection vs migration detection issues

#### Test 24: `test_get_version_info_exception_handling`
**Purpose**: Verify exception propagation in get_version_info

**Scenario**: Underlying service method throws exception
**Test Data**: Mock exception in get_app_version
**Action**: `SystemService.get_version_info()`
**Expected**: Exception propagates (not caught by get_version_info)
**Why it matters**: Ensures calling code can handle exceptions appropriately

### Logging Verification Tests

#### Test 25: `test_get_app_version_logs_error`
**Purpose**: Verify error logging for app version failures

**Scenario**: VERSION file read fails
**Test Data**: Mock FileNotFoundError
**Action**: `SystemService.get_app_version()`
**Expected**: Error logged with ERROR level, SYSTEM category
**Why it matters**: Ensures ops teams can diagnose deployment issues

#### Test 26: `test_get_db_version_logs_error`
**Purpose**: Verify error logging for database version failures

**Scenario**: Database query fails
**Test Data**: Mock database exception
**Action**: `SystemService.get_db_version()`
**Expected**: Error logged with specific error details
**Why it matters**: Database connectivity issues need clear logging

#### Test 27: `test_check_pending_migrations_logs_error`
**Purpose**: Verify error logging for migration check failures

**Scenario**: Alembic operations fail
**Test Data**: Mock migration system error
**Action**: `SystemService.check_pending_migrations()`
**Expected**: Error logged with migration-specific context
**Why it matters**: Migration issues are complex, need detailed logging

#### Test 28: `test_check_feature_availability_logs_warning`
**Purpose**: Verify warning logging for version parsing failures

**Scenario**: Version string is malformed
**Test Data**: "invalid.version.format"
**Action**: `SystemService.check_feature_availability()`
**Expected**: WARNING level log with version details
**Why it matters**: Helps diagnose data corruption or migration issues

#### Test 29: `test_get_version_info_logs_request`
**Purpose**: Verify request logging for version checks

**Scenario**: Normal version info request
**Test Data**: Valid database setup
**Action**: `SystemService.get_version_info()`
**Expected**: INFO level log with complete version details
**Why it matters**: Audit trail for version checks, useful for support

### Integration Tests

#### Test 30: `test_real_version_check_integration`
**Purpose**: End-to-end test with real file system and database

**Scenario**: Complete system test using actual VERSION file
**Test Data**: Real VERSION file + test database
**Action**: `SystemService.get_version_info()`
**Expected**: All components work together correctly
**Why it matters**: Validates real-world operation

#### Test 31: `test_alembic_configuration_paths`
**Purpose**: Verify Alembic file paths are correct

**Scenario**: Check that migration files can be found
**Test Data**: Real migrations directory
**Action**: `SystemService.check_pending_migrations()`
**Expected**: No path-related errors
**Why it matters**: Prevents deployment issues with missing migration files

#### Test 32: `test_version_file_path`
**Purpose**: Verify VERSION file path is correct

**Scenario**: Test actual file system access
**Test Data**: Real VERSION file
**Action**: `SystemService.get_app_version()`
**Expected**: Successfully reads "1.3.2"
**Why it matters**: Validates deployment assumptions about file locations

## Coverage Analysis

**Current Coverage**: 100% (79/79 statements) ✅

```
Coverage % = (Executed Lines / Total Lines) × 100
100% = (79 / 79) × 100
```

**Uncovered Lines**: None

**Coverage Breakdown**:
- `get_app_version`: 100% (all error paths tested)
- `get_db_version`: 100% (all error paths tested)
- `check_pending_migrations`: 100% (all error and success paths tested)
- `check_feature_availability`: 100% (all version parsing paths tested)
- `get_version_info`: 100% (all integration paths tested)

## Bug Discovery and Fixes

### Bug 1: IBKR Integration Version Logic Error

**Description**: The version check for IBKR integration feature was incorrectly written, causing future major versions (2.x+) to incorrectly return `ibkr_integration: false`.

**Root Cause**: Logic `major >= 1 and minor >= 3` requires BOTH conditions, so major version 2 with minor version 0 would fail the `minor >= 3` check.

**The Fix**:
```python
# BEFORE (buggy)
if major >= 1 and minor >= 3:
    features["ibkr_integration"] = True

# AFTER (fixed)
if major > 1 or (major == 1 and minor >= 3):
    features["ibkr_integration"] = True
```

**Test Validation**: Test `test_check_feature_availability_major_version_2` specifically validates this fix.

**Impact**: Would have caused feature regression if not caught during testing.

## Running Tests

### Run All SystemService Tests
```bash
pytest tests/services/test_system_service.py -v
```

### Run All Tests with Coverage
```bash
pytest tests/services/test_system_service.py --cov=app.services.system_service --cov-report=term-missing
```

### Run Specific Test Classes
```bash
# Core functionality only
pytest tests/services/test_system_service.py::TestSystemService -v

# Logging tests only
pytest tests/services/test_system_service.py::TestSystemServiceLogging -v

# Integration tests only
pytest tests/services/test_system_service.py::TestSystemServiceIntegration -v
```

### Run Specific Tests
```bash
# Test the core version fix
pytest tests/services/test_system_service.py::TestSystemService::test_get_version_info_no_migrations_needed -v

# Test the bug fix validation
pytest tests/services/test_system_service.py::TestSystemService::test_check_feature_availability_major_version_2 -v
```

## Related Documentation

- [infrastructure/TESTING_INFRASTRUCTURE.md](../infrastructure/TESTING_INFRASTRUCTURE.md) - Testing setup and fixtures
- [phases/BUG_FIXES_1.3.3.md](../phases/BUG_FIXES_1.3.3.md) - Bug discovery during testing
- [services/TRANSACTION_SERVICE_TESTS.md](TRANSACTION_SERVICE_TESTS.md) - Similar service testing patterns
- [services/LOGGING_SERVICE_TESTS.md](LOGGING_SERVICE_TESTS.md) - Related system service testing

## Key Achievements

1. **100% Coverage** - All code paths tested including error conditions
2. **Comprehensive Error Handling** - All failure modes tested and logged
3. **Bug Discovery** - Found and fixed version logic error before production
4. **Real Integration** - Tests work with actual filesystem and database
5. **Documentation** - Complete test documentation for future maintenance

## Maintenance Notes

### When Adding New Features
1. Add feature flag logic to `check_feature_availability`
2. Add tests for all supported version ranges
3. Update test documentation with new feature descriptions

### When Changing Version Logic
1. Update all relevant test expectations
2. Test edge cases (major version changes, parsing errors)
3. Verify integration tests still pass
4. Update documentation with new logic explanations

### When Updating Migrations (v1.3.5+)
1. **No manual updates needed** - Tests now dynamically detect the latest migration version
2. `LATEST_MIGRATION_VERSION` constant automatically updates based on migration files
3. Simply create new migration files following the naming pattern: `X.Y.Z_description.py`
4. Tests will automatically use the new version for database setup
5. Verify tests pass with `pytest tests/services/test_system_service.py -v`

**Important**: The dynamic versioning pattern eliminates the need to manually update hardcoded version strings throughout the test file when creating new migrations.

This comprehensive test suite ensures the SystemService correctly handles the critical task of version management, preventing user-facing errors and providing clear guidance for database maintenance.
