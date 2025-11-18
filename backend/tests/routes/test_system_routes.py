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
