"""
Integration tests for system routes (system_routes.py).

Tests System API endpoints:
- GET /api/system/version - Get version info ✅
- GET /api/system/health - Get health status ✅

Test Summary: 2 passing
"""


class TestSystemRoutes:
    """Test system information endpoints."""

    def test_get_version_info(self, app_context, client):
        """Test GET /system/version returns version information."""
        response = client.get("/api/system/version")

        assert response.status_code == 200
        data = response.get_json()
        assert "app_version" in data
        assert "db_version" in data
        assert "features" in data

    def test_get_health_status(self, app_context, client, db_session):
        """Test GET /system/health returns health status."""
        response = client.get("/api/system/health")

        assert response.status_code == 200
        data = response.get_json()
        assert "status" in data
        assert "database" in data
        # Database should be connected in test environment
        assert data["database"] == "connected"


class TestSystemErrors:
    """Test error paths for system routes."""

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
