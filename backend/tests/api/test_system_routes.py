"""
Integration tests for system routes (system_routes.py).

Tests System API endpoints:
- GET /api/system/version - Get version info ✅
- GET /api/system/health - Get health status ✅

Test Summary: 4 tests (2 integration + 2 error paths)
"""

from unittest.mock import patch


class TestSystemRoutes:
    """Test system information endpoints."""

    def test_get_version_info(self, app_context, client):
        """
        Verify version endpoint returns application and database versions.

        WHY: Frontend needs version info to display in UI and determine API
        compatibility. This ensures the endpoint correctly retrieves app version
        from VERSION file, database schema version from Alembic, and feature flags.
        """
        response = client.get("/api/system/version")

        assert response.status_code == 200
        data = response.get_json()
        assert "app_version" in data
        assert "db_version" in data
        assert "features" in data

    def test_get_health_status(self, app_context, client, db_session):
        """
        Verify health check endpoint confirms database connectivity.

        WHY: Monitoring systems and load balancers use /health to determine if
        the application is ready to serve requests. This validates that a healthy
        system returns status="healthy" and database="connected".
        """
        response = client.get("/api/system/health")

        assert response.status_code == 200
        data = response.get_json()
        assert "status" in data
        assert "database" in data
        # Database should be connected in test environment
        assert data["database"] == "connected"


class TestSystemErrors:
    """Test error paths for system routes."""

    def test_get_version_info_service_error(self, app_context, client):
        """
        Verify version endpoint handles service failures gracefully.

        WHY: If VERSION file is missing or corrupted, the endpoint should return
        500 with error details rather than crashing. This ensures monitoring can
        detect configuration issues.
        """

        # Mock SystemService.get_version_info to raise exception
        def mock_get_version():
            raise Exception("Version file not found")

        with patch(
            "app.api.system_namespace.SystemService.get_version_info",
            mock_get_version,
        ):
            response = client.get("/api/system/version")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data
            assert "details" in data

    def test_health_check_database_error(self, app_context, client):
        """
        Verify health check returns 503 when database is unavailable.

        WHY: Load balancers need to distinguish between application errors (500)
        and infrastructure failures (503) to route traffic appropriately. When
        the database is down, health check must return 503 Service Unavailable.
        """
        # Mock db.session.execute to raise exception
        with patch("app.api.system_namespace.db.session.execute") as mock_execute:
            mock_execute.side_effect = Exception("Database connection failed")

            response = client.get("/api/system/health")

            assert response.status_code == 503
            data = response.get_json()
            assert data["status"] == "unhealthy"
            assert data["database"] == "disconnected"
            assert "error" in data
