# System Routes Integration Tests

**File**: `tests/routes/test_system_routes.py`
**Route File**: `app/routes/system_routes.py`
**Test Count**: 4 tests (2 integration + 2 error path)
**Coverage**: 100% (21/21 statements)
**Status**: ✅ All tests passing

---

## Overview

Integration tests for system information and health check API endpoints. These tests verify system version information retrieval and database health monitoring.

### Endpoints Tested

1. **GET /api/system/version** - Get version information
2. **GET /api/system/health** - Get health status

---

## Test Organization

### Test Classes

1. **TestSystemRoutes** (2 tests)
   - Get version info
   - Get health status

2. **TestSystemErrors** (2 tests)
   - Get version info service error
   - Health check database error

---

## Key Test Patterns

### 1. Testing Version Information

```python
def test_get_version_info(self, app_context, client):
    """Test GET /system/version returns version information."""
    response = client.get("/api/system/version")

    assert response.status_code == 200
    data = response.get_json()
    assert "app_version" in data
    assert "db_version" in data
    assert "features" in data
```

**Why**: Provides version info for clients to determine API compatibility and available features.

### 2. Testing Health Check

```python
def test_get_health_status(self, app_context, client, db_session):
    """Test GET /system/health returns health status."""
    response = client.get("/api/system/health")

    assert response.status_code == 200
    data = response.get_json()
    assert "status" in data
    assert "database" in data
    # Database should be connected in test environment
    assert data["database"] == "connected"
```

**Why**: Allows monitoring systems to verify the application and database are operational.

---

## Error Path Testing (Phase 4b)

### TestSystemErrors Class

Added comprehensive error path tests to achieve 100% coverage on `system_routes.py`.

**Tests Added**:
1. **test_get_version_info_service_error** - Tests GET /system/version handles service exceptions
2. **test_health_check_database_error** - Tests GET /system/health handles database connection failures

**Coverage Improvement**: 71% → 100% (all exception handlers now tested)

**Testing Pattern**:
```python
from unittest.mock import patch

def test_get_version_info_service_error(self, app_context, client, monkeypatch):
    """Test GET /system/version handles service errors."""

    # Mock SystemService.get_version_info to raise exception
    def mock_get_version():
        raise Exception("Version file not found")

    monkeypatch.setattr(
        "app.routes.system_routes.SystemService.get_version_info",
        mock_get_version,
    )

    response = client.get("/api/system/version")

    assert response.status_code == 500
    data = response.get_json()
    assert "error" in data
    assert "details" in data
```

**Database Error Testing**:
```python
def test_health_check_database_error(self, app_context, client):
    """Test GET /system/health handles database connection errors."""
    from unittest.mock import patch

    # Mock db.session.execute to raise exception
    with patch("app.routes.system_routes.db.session.execute") as mock_execute:
        mock_execute.side_effect = Exception("Database connection failed")

        response = client.get("/api/system/health")

        assert response.status_code == 503
        data = response.get_json()
        assert data["status"] == "unhealthy"
        assert data["database"] == "disconnected"
        assert "error" in data
```

**Why This Matters**: System endpoints are critical for monitoring. Error path tests ensure the health check API correctly reports failures with appropriate HTTP status codes (503 Service Unavailable for database errors).

---

## System Data Structure

### Version Info Response Format
```json
{
    "app_version": "1.3.3",
    "db_version": "1.3.2",
    "features": {
        "ibkr_integration": true
    }
}
```

### Health Check Response Format (Success)
```json
{
    "status": "healthy",
    "database": "connected"
}
```

### Health Check Response Format (Database Failure)
```json
{
    "status": "unhealthy",
    "database": "disconnected",
    "error": "Database connection failed"
}
```

---

## Running Tests

### Run all system route tests:
```bash
pytest tests/routes/test_system_routes.py -v
```

### Run specific test class:
```bash
pytest tests/routes/test_system_routes.py::TestSystemRoutes -v
```

### Run without coverage (faster):
```bash
pytest tests/routes/test_system_routes.py -v --no-cov
```

---

## Test Results

**All 4 tests passing** ✅

### Test Execution Time
- **Average**: ~0.15 seconds for full suite
- **Pattern**: Fastest route tests (minimal business logic)

### Coverage
- **Route Coverage**: 100% (21/21 statements, 0 missing lines)
- **Coverage Improvement**: 71% → 100% (Phase 4b error path testing)
- Integration tests verify **all 2 system endpoints** are accessible and return appropriate responses
- Error tests verify **all exception handlers** return appropriate status codes (500 for service errors, 503 for database failures)

---

## Important Notes

### Health Check Status Codes

The health check endpoint uses different status codes for different failure scenarios:

- **200 OK**: System is healthy, database connected
- **503 Service Unavailable**: Database connection failed (system unhealthy)

**Why**: Monitoring systems can distinguish between application errors (500) and infrastructure failures (503).

### Version Management

The version endpoint uses `SystemService` to:
1. Read app version from `VERSION` file
2. Query database schema version from `alembic_version` table
3. Check for pending migrations via Alembic
4. Determine available features based on schema version

This prevents false migration warnings when app version > schema version but no migrations are actually pending.

---

## Related Documentation

- **Service Tests**: `tests/docs/services/SYSTEM_SERVICE_TESTS.md` (32 tests, 100% coverage)
- **Testing Infrastructure**: `tests/docs/infrastructure/TESTING_INFRASTRUCTURE.md`
- **API Routes**: `app/routes/system_routes.py`
- **System Service**: `app/services/system_service.py`

---

**Last Updated**: Phase 5 (Route Integration Tests) + Phase 4b (Error Path Testing)
**Maintainer**: See git history
