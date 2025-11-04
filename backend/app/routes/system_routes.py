"""
API routes for system information and health checks.

This module provides routes for:
- Version checking
- Feature availability based on database schema
- Health checks
"""

import os

from flask import Blueprint, jsonify

from ..models import LogCategory, LogLevel, db
from ..services.logging_service import logger

system = Blueprint("system", __name__)


def get_app_version():
    """
    Get application version from VERSION file.

    Returns:
        str: Application version (e.g., "1.3.0")
    """
    version_file = os.path.join(os.path.dirname(__file__), "../../VERSION")
    try:
        with open(version_file) as f:
            return f.read().strip()
    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message="Failed to read VERSION file",
            details={"error": str(e)},
        )
        return "unknown"


def get_db_version():
    """
    Get current database schema version from Alembic.

    Returns:
        str: Database version (e.g., "1.1.2") or "unknown"
    """
    try:
        # Query the alembic_version table to get current revision
        result = db.session.execute(db.text("SELECT version_num FROM alembic_version")).fetchone()

        if result:
            return result[0]
        return "unknown"
    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message="Failed to get database version",
            details={"error": str(e)},
        )
        return "unknown"


def check_feature_availability(db_version):
    """
    Check which features are available based on database version.

    Args:
        db_version: Current database version string

    Returns:
        dict: Feature availability flags
    """
    features = {
        "ibkr_integration": False,
        "realized_gain_loss": False,
        "exclude_from_overview": False,
    }

    # Parse version (handle "unknown" case)
    try:
        if db_version == "unknown":
            return features

        # Extract major.minor.patch from version string
        version_parts = db_version.split(".")
        if len(version_parts) >= 2:
            major = int(version_parts[0])
            minor = int(version_parts[1])

            # Feature flags based on version
            # Version 1.1.0+: exclude_from_overview
            if major >= 1 and minor >= 1:
                features["exclude_from_overview"] = True

            # Version 1.1.1+: realized_gain_loss
            if major >= 1 and minor >= 1:
                patch = int(version_parts[2]) if len(version_parts) >= 3 else 0
                if minor > 1 or (minor == 1 and patch >= 1):
                    features["realized_gain_loss"] = True

            # Version 1.3.0+: IBKR integration
            if major >= 1 and minor >= 3:
                features["ibkr_integration"] = True

    except (ValueError, IndexError) as e:
        logger.log(
            level=LogLevel.WARNING,
            category=LogCategory.SYSTEM,
            message="Failed to parse database version",
            details={"version": db_version, "error": str(e)},
        )

    return features


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
        app_version = get_app_version()
        db_version = get_db_version()
        features = check_feature_availability(db_version)

        # Check if migration is needed
        migration_needed = app_version != db_version
        migration_message = None

        if migration_needed:
            if db_version == "unknown":
                migration_message = (
                    "Database schema version could not be determined. "
                    "Please run 'flask db upgrade' to ensure database is up to date."
                )
            else:
                migration_message = (
                    f"Database schema (v{db_version}) is behind application "
                    f"version (v{app_version}). "
                    f"Run 'flask db upgrade' to enable new features."
                )

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Version check requested",
            details={
                "app_version": app_version,
                "db_version": db_version,
                "migration_needed": migration_needed,
            },
        )

        return jsonify(
            {
                "app_version": app_version,
                "db_version": db_version,
                "features": features,
                "migration_needed": migration_needed,
                "migration_message": migration_message,
            }
        ), 200

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
