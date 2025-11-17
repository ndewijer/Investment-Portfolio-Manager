"""
API routes for system information and health checks.

This module provides routes for:
- Version checking
- Feature availability based on database schema
- Health checks
"""

from flask import Blueprint, jsonify

from ..models import LogCategory, LogLevel, db
from ..services.logging_service import logger
from ..services.system_service import SystemService

system = Blueprint("system", __name__)


@system.route("/system/version", methods=["GET"])
def get_version_info():
    """
    Get application and database version information.

    Returns:
        JSON response with:
        - app_version: Application version from VERSION file
        - db_version: Current database schema version from Alembic
        - features: Dict of available features based on schema version
        - migration_needed: Boolean indicating if migration is required
        - migration_message: User-friendly message if migration needed
    """
    try:
        version_info = SystemService.get_version_info()
        return jsonify(version_info), 200

    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message="Error getting version info",
            details={"error": str(e)},
        )
        return jsonify({"error": "Failed to get version information", "details": str(e)}), 500


@system.route("/system/health", methods=["GET"])
def health_check():
    """
    Basic health check endpoint.

    Returns:
        JSON response with health status
    """
    try:
        # Test database connection
        db.session.execute(db.text("SELECT 1"))

        return jsonify(
            {
                "status": "healthy",
                "database": "connected",
            }
        ), 200

    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message="Health check failed",
            details={"error": str(e)},
        )
        return jsonify(
            {
                "status": "unhealthy",
                "database": "disconnected",
                "error": str(e),
            }
        ), 503
