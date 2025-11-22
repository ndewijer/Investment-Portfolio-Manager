# System Routes Integration Tests

**File**: `tests/routes/test_system_routes.py`
**Route File**: `app/routes/system_routes.py`
**Test Count**: 4 tests (2 integration + 2 error paths)
**Coverage**: 100% (21/21 statements)
**Status**: ✅ All tests passing

> **💡 Detailed Test Information**: For detailed explanations of each test including
> WHY it exists and what business logic it validates, see the docstrings in the test file.
> Your IDE will show these when hovering over test names.

---

## Overview

Integration tests for system information and health check API endpoints. These tests verify system version information retrieval and database health monitoring for use by frontends and monitoring systems.

### Endpoints Tested

- **GET /api/system/version** - Get application and database version info
- **GET /api/system/health** - Get health status and database connectivity

---

## Test Organization

### TestSystemRoutes (2 tests)
- `test_get_version_info` - Verifies version endpoint returns app/db versions and features
- `test_get_health_status` - Verifies health check confirms database connectivity

### TestSystemErrors (2 tests)
- `test_get_version_info_service_error` - Tests service failure handling (500)
- `test_health_check_database_error` - Tests database failure handling (503)

---

## Key Patterns

**Version Management**: Uses SystemService to read VERSION file and query Alembic schema version

**Health Check Status Codes**:
- 200 OK - System healthy, database connected
- 503 Service Unavailable - Database connection failed (infrastructure issue)
- 500 Internal Server Error - Application error (e.g., missing VERSION file)

---

## Running Tests

```bash
# Run all system route tests
pytest tests/routes/test_system_routes.py -v

# Run specific test class
pytest tests/routes/test_system_routes.py::TestSystemRoutes -v

# Run without coverage (faster)
pytest tests/routes/test_system_routes.py -v --no-cov
```

---

## Related Documentation

- **Service Tests**: `tests/docs/services/SYSTEM_SERVICE_TESTS.md` (32 tests, 100% coverage)
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **System Service**: `app/services/system_service.py`

---

**Last Updated**: Phase 5 (Route Integration Tests) + Documentation Condensing
**Coverage**: 100% (all exception handlers tested)
