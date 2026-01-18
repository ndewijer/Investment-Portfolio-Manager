"""
Service class for system-related operations.

This module provides methods for:
- Application and database version management
- Migration status checking
- Feature availability based on schema version
"""

import os

from alembic.config import Config
from alembic.runtime.migration import MigrationContext
from alembic.script import ScriptDirectory

from ..models import LogCategory, LogLevel, db
from ..services.logging_service import logger


class SystemService:
    """
    Service class for system-related operations.

    Provides methods for:
    - Version management and checking
    - Migration status validation
    - Feature availability detection
    """

    @staticmethod
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

    @staticmethod
    def get_db_version():
        """
        Get current database schema version from Alembic.

        Returns:
            str: Database version (e.g., "1.1.2") or "unknown"
        """
        try:
            # Query the alembic_version table to get current revision
            result = db.session.execute(
                db.text("SELECT version_num FROM alembic_version")
            ).fetchone()

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

    @staticmethod
    def check_pending_migrations():
        """
        Check if there are actual pending migrations that need to be applied.

        This is more accurate than just comparing version numbers, as it checks
        if there are actual migration scripts that haven't been applied yet.

        Returns:
            tuple: (has_pending_migrations: bool, error_message: str or None)
        """
        try:
            # Get the Alembic configuration
            migrations_dir = os.path.join(os.path.dirname(__file__), "../../migrations")
            alembic_cfg = Config(os.path.join(migrations_dir, "alembic.ini"))
            alembic_cfg.set_main_option("script_location", migrations_dir)
            # Set path_separator to suppress warning
            alembic_cfg.set_main_option("path_separator", os.pathsep)

            # Get script directory
            script = ScriptDirectory.from_config(alembic_cfg)

            # Get current revision from database
            with db.engine.connect() as connection:
                context = MigrationContext.configure(connection)
                current_rev = context.get_current_revision()

                # Get head revision (latest available migration)
                head_rev = script.get_current_head()

                # If current revision is None, database needs to be initialized
                if current_rev is None:
                    return True, "Database not initialized"

                # If current revision doesn't match head, there are pending migrations
                if current_rev != head_rev:
                    return True, None

                # No pending migrations
                return False, None

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Failed to check pending migrations",
                details={"error": str(e)},
            )
            return True, f"Error checking migrations: {e!s}"

    @staticmethod
    def check_feature_availability(db_version):
        """
        Check which features are available based on database schema version.

        Args:
            db_version (str): Database version to check

        Returns:
            dict: Dictionary of feature availability flags
        """
        features = {
            "basic_portfolio_management": True,  # Always available
            "realized_gain_loss": False,
            "ibkr_integration": False,
            "materialized_view_performance": False,
            "fund_level_materialized_view": False,
        }

        # Parse version and check feature availability
        try:
            if db_version != "unknown":
                # Remove 'v' prefix if present and split version
                version_clean = db_version.lstrip("v")
                parts = version_clean.split(".")
                major, minor, patch = int(parts[0]), int(parts[1]), int(parts[2])

                # Version 1.1.1+: Realized gains/losses
                if (
                    major > 1
                    or (major == 1 and minor > 1)
                    or (major == 1 and minor == 1 and patch >= 1)
                ):
                    features["realized_gain_loss"] = True

                # Check for version 1.1.0+ with specific logic
                if minor > 1 or (minor == 1 and patch >= 1):
                    features["realized_gain_loss"] = True

                # Version 1.3.0+: IBKR integration
                if major > 1 or (major == 1 and minor >= 3):
                    features["ibkr_integration"] = True

                # Version 1.4.0+: Materialized view performance optimization
                if major > 1 or (major == 1 and minor >= 4):
                    features["materialized_view_performance"] = True

                # Version 1.5.0+: Fund-level materialized view
                if major > 1 or (major == 1 and minor >= 5):
                    features["fund_level_materialized_view"] = True

        except (ValueError, IndexError) as e:
            logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.SYSTEM,
                message="Failed to parse database version",
                details={"version": db_version, "error": str(e)},
            )

        return features

    @staticmethod
    def get_version_info():
        """
        Get comprehensive version and migration information.

        Returns:
            dict: Dictionary containing:
                - app_version: Application version
                - db_version: Database schema version
                - features: Available features based on schema
                - migration_needed: Whether migrations are pending
                - migration_message: User-friendly message if migrations needed
        """
        app_version = SystemService.get_app_version()
        db_version = SystemService.get_db_version()
        features = SystemService.check_feature_availability(db_version)

        # Check if migration is actually needed by examining pending migrations
        has_pending, migration_error = SystemService.check_pending_migrations()
        migration_needed = has_pending
        migration_message = None

        if migration_needed:
            if migration_error:
                migration_message = migration_error
            elif db_version == "unknown":
                migration_message = (
                    "Database schema version could not be determined. "
                    "Please run 'flask db upgrade' to ensure database is up to date."
                )
            else:
                migration_message = (
                    f"Database schema updates are available. "
                    f"Current schema: v{db_version}, Application: v{app_version}. "
                    f"Run 'flask db upgrade' to apply new migrations."
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

        return {
            "app_version": app_version,
            "db_version": db_version,
            "features": features,
            "migration_needed": migration_needed,
            "migration_message": migration_message,
        }
