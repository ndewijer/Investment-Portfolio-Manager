"""
System API namespace for health checks and version information.

This namespace provides endpoints for:
- Application version information
- Database version and migration status
- Feature availability flags
- Health checks
"""

from flask_restx import Namespace, Resource, fields

from ..models import LogCategory, LogLevel, db
from ..services.logging_service import logger
from ..services.system_service import SystemService

# Create namespace
ns = Namespace("system", description="System health and version information")

# Define models for documentation
version_info_model = ns.model(
    "VersionInfo",
    {
        "app_version": fields.String(
            required=True, description="Application version from VERSION file"
        ),
        "db_version": fields.String(
            required=True, description="Current database schema version from Alembic"
        ),
        "features": fields.Raw(
            required=True, description="Dictionary of available features based on schema version"
        ),
        "migration_needed": fields.Boolean(
            required=True, description="Whether database migration is required"
        ),
        "migration_message": fields.String(
            description="User-friendly message if migration is needed"
        ),
    },
)

health_check_model = ns.model(
    "HealthCheck",
    {
        "status": fields.String(
            required=True, description="Health status", enum=["healthy", "unhealthy"]
        ),
        "database": fields.String(
            required=True,
            description="Database connection status",
            enum=["connected", "disconnected"],
        ),
        "error": fields.String(description="Error message if unhealthy"),
    },
)

error_model = ns.model(
    "Error",
    {
        "error": fields.String(required=True, description="Error message"),
        "details": fields.String(description="Additional error details"),
    },
)


@ns.route("/version")
class VersionInfo(Resource):
    """Version information endpoint."""

    @ns.doc("get_version_info")
    @ns.response(200, "Success", version_info_model)
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get application and database version information.

        Returns version information including:
        - Application version from VERSION file
        - Database schema version from Alembic
        - Available features based on current schema
        - Migration status and recommendations

        This endpoint is useful for:
        - Frontend compatibility checks
        - Determining which features are available
        - Checking if database migrations are needed
        """
        try:
            version_info = SystemService.get_version_info()
            return version_info, 200

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Error getting version info",
                details={"error": str(e)},
            )
            return {"error": "Failed to get version information", "details": str(e)}, 500


@ns.route("/health")
class HealthCheck(Resource):
    """Health check endpoint."""

    @ns.doc("health_check")
    @ns.response(200, "System is healthy", health_check_model)
    @ns.response(503, "System is unhealthy", health_check_model)
    def get(self):
        """
        Perform a basic health check.

        Checks:
        - Database connectivity
        - Application responsiveness

        This endpoint is useful for:
        - Load balancer health checks
        - Monitoring and alerting
        - Container orchestration health probes
        """
        try:
            # Test database connection
            db.session.execute(db.text("SELECT 1"))

            return {
                "status": "healthy",
                "database": "connected",
            }, 200

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Health check failed",
                details={"error": str(e)},
            )
            return {
                "status": "unhealthy",
                "database": "disconnected",
                "error": str(e),
            }, 503
