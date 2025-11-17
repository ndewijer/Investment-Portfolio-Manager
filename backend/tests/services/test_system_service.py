"""
Tests for SystemService class.

This module tests the SystemService functionality including:
- Application version retrieval
- Database version checking
- Migration status validation
- Feature availability detection
- Complete version information assembly
"""

import unittest.mock
from unittest.mock import Mock, patch

import pytest
from app.models import LogCategory, LogLevel, db
from app.services.system_service import SystemService


class TestSystemService:
    """Test class for SystemService functionality."""

    def test_get_app_version_success(self, app_context):
        """Test successful app version retrieval from VERSION file."""
        # Test the actual VERSION file exists and returns a valid version
        version = SystemService.get_app_version()

        assert version != "unknown"
        assert isinstance(version, str)
        assert len(version.split(".")) >= 2  # Should be at least major.minor format
        # Version 1.3.2 should be current
        assert version.startswith("1.3")

    def test_get_app_version_file_not_found(self, app_context):
        """Test app version retrieval when VERSION file doesn't exist."""
        with patch("builtins.open", side_effect=FileNotFoundError("File not found")):
            version = SystemService.get_app_version()
            assert version == "unknown"

    def test_get_app_version_read_error(self, app_context):
        """Test app version retrieval when VERSION file can't be read."""
        with patch("builtins.open", side_effect=PermissionError("Permission denied")):
            version = SystemService.get_app_version()
            assert version == "unknown"

    def test_get_db_version_success(self, app_context, db_session):
        """Test successful database version retrieval."""
        # Create alembic_version table and insert test version
        db_session.execute(
            db.text("CREATE TABLE IF NOT EXISTS alembic_version (version_num VARCHAR(32) NOT NULL)")
        )
        db_session.execute(db.text("DELETE FROM alembic_version"))  # Clear any existing
        db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
        db_session.commit()

        version = SystemService.get_db_version()

        assert version != "unknown"
        assert isinstance(version, str)
        # Should be the version we inserted
        assert version == "1.3.1"

    def test_get_db_version_no_table(self, app_context):
        """Test database version retrieval when alembic_version table doesn't exist."""
        with patch("app.models.db.session.execute") as mock_execute:
            mock_execute.side_effect = Exception("Table doesn't exist")
            version = SystemService.get_db_version()
            assert version == "unknown"

    def test_get_db_version_no_result(self, app_context):
        """Test database version retrieval when no version is found."""
        with patch("app.models.db.session.execute") as mock_execute:
            mock_execute.return_value.fetchone.return_value = None
            version = SystemService.get_db_version()
            assert version == "unknown"

    def test_check_pending_migrations_no_pending(self, app_context, db_session):
        """Test migration check when no migrations are pending."""
        # Create alembic_version table with current head version
        db_session.execute(
            db.text("CREATE TABLE IF NOT EXISTS alembic_version (version_num VARCHAR(32) NOT NULL)")
        )
        db_session.execute(db.text("DELETE FROM alembic_version"))
        db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
        db_session.commit()

        has_pending, error = SystemService.check_pending_migrations()

        # Should be no pending migrations if DB is at current head
        assert has_pending is False
        assert error is None

    def test_check_pending_migrations_with_pending(self, app_context):
        """Test migration check when migrations are pending."""
        with patch("app.models.db.engine.connect") as mock_connect:
            # Mock connection and context
            mock_connection = Mock()
            mock_connect.return_value.__enter__.return_value = mock_connection

            # Mock MigrationContext to return different revisions
            with patch("alembic.runtime.migration.MigrationContext.configure") as mock_configure:
                mock_context = Mock()
                mock_context.get_current_revision.return_value = "1.2.0"  # Old revision
                mock_configure.return_value = mock_context

                # Mock ScriptDirectory to return newer head
                with patch("alembic.script.ScriptDirectory.from_config") as mock_script_dir:
                    mock_script = Mock()
                    mock_script.get_current_head.return_value = "1.3.1"  # Newer head
                    mock_script_dir.return_value = mock_script

                    has_pending, error = SystemService.check_pending_migrations()

                    assert has_pending is True
                    assert error is None

    def test_check_pending_migrations_uninitialized_db(self, app_context):
        """Test migration check when database is not initialized."""
        with patch("app.models.db.engine.connect") as mock_connect:
            # Mock connection and context
            mock_connection = Mock()
            mock_connect.return_value.__enter__.return_value = mock_connection

            # Mock MigrationContext to return None (uninitialized)
            with patch("alembic.runtime.migration.MigrationContext.configure") as mock_configure:
                mock_context = Mock()
                mock_context.get_current_revision.return_value = None
                mock_configure.return_value = mock_context

                has_pending, error = SystemService.check_pending_migrations()

                assert has_pending is True
                assert error == "Database not initialized"

    def test_check_pending_migrations_error(self, app_context):
        """Test migration check when an error occurs."""
        with patch("app.models.db.engine.connect", side_effect=Exception("Database error")):
            has_pending, error = SystemService.check_pending_migrations()

            assert has_pending is True
            assert "Error checking migrations" in error
            assert "Database error" in error

    def test_check_feature_availability_unknown_version(self, app_context):
        """Test feature availability for unknown database version."""
        features = SystemService.check_feature_availability("unknown")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": False,
            "ibkr_integration": False,
        }
        assert features == expected_features

    def test_check_feature_availability_version_1_0_0(self, app_context):
        """Test feature availability for version 1.0.0."""
        features = SystemService.check_feature_availability("1.0.0")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": False,
            "ibkr_integration": False,
        }
        assert features == expected_features

    def test_check_feature_availability_version_1_1_0(self, app_context):
        """Test feature availability for version 1.1.0 (exclude_from_overview added)."""
        features = SystemService.check_feature_availability("1.1.0")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": False,  # Added in 1.1.1
            "ibkr_integration": False,
        }
        assert features == expected_features

    def test_check_feature_availability_version_1_1_1(self, app_context):
        """Test feature availability for version 1.1.1 (realized_gain_loss added)."""
        features = SystemService.check_feature_availability("1.1.1")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": True,
            "ibkr_integration": False,
        }
        assert features == expected_features

    def test_check_feature_availability_version_1_3_0(self, app_context):
        """Test feature availability for version 1.3.0 (IBKR integration added)."""
        features = SystemService.check_feature_availability("1.3.0")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": True,
            "ibkr_integration": True,
        }
        assert features == expected_features

    def test_check_feature_availability_version_1_3_1(self, app_context):
        """Test feature availability for current version 1.3.1."""
        features = SystemService.check_feature_availability("1.3.1")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": True,
            "ibkr_integration": True,
        }
        assert features == expected_features

    def test_check_feature_availability_version_with_v_prefix(self, app_context):
        """Test feature availability for version with 'v' prefix."""
        features = SystemService.check_feature_availability("v1.3.1")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": True,
            "ibkr_integration": True,
        }
        assert features == expected_features

    def test_check_feature_availability_invalid_version(self, app_context):
        """Test feature availability for invalid version string."""
        features = SystemService.check_feature_availability("invalid.version")

        # Should return default features when parsing fails
        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": False,
            "ibkr_integration": False,
        }
        assert features == expected_features

    def test_check_feature_availability_major_version_2(self, app_context):
        """Test feature availability for future major version."""
        features = SystemService.check_feature_availability("2.0.0")

        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": True,
            "ibkr_integration": True,
        }
        assert features == expected_features

    def test_get_version_info_no_migrations_needed(self, app_context, db_session):
        """Test complete version info when no migrations are needed."""
        # Setup database version
        db_session.execute(
            db.text("CREATE TABLE IF NOT EXISTS alembic_version (version_num VARCHAR(32) NOT NULL)")
        )
        db_session.execute(db.text("DELETE FROM alembic_version"))
        db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
        db_session.commit()

        version_info = SystemService.get_version_info()

        # Verify structure
        required_keys = {
            "app_version",
            "db_version",
            "features",
            "migration_needed",
            "migration_message",
        }
        assert set(version_info.keys()) == required_keys

        # Verify current state (app 1.3.2, db 1.3.1, no migrations needed)
        assert version_info["app_version"] == "1.3.2"
        assert version_info["db_version"] == "1.3.1"
        assert version_info["migration_needed"] is False
        assert version_info["migration_message"] is None

        # Verify features for 1.3.1
        expected_features = {
            "basic_portfolio_management": True,
            "realized_gain_loss": True,
            "ibkr_integration": True,
        }
        assert version_info["features"] == expected_features

    def test_get_version_info_migrations_needed(self, app_context):
        """Test complete version info when migrations are needed."""
        with (
            patch.object(SystemService, "check_pending_migrations", return_value=(True, None)),
            patch.object(SystemService, "get_db_version", return_value="1.2.0"),
        ):
            version_info = SystemService.get_version_info()

            assert version_info["migration_needed"] is True
            assert "Database schema updates are available" in version_info["migration_message"]
            assert "v1.2.0" in version_info["migration_message"]
            assert "v1.3.2" in version_info["migration_message"]

    def test_get_version_info_migration_error(self, app_context):
        """Test complete version info when migration check has an error."""
        with patch.object(
            SystemService,
            "check_pending_migrations",
            return_value=(True, "Database connection failed"),
        ):
            version_info = SystemService.get_version_info()

            assert version_info["migration_needed"] is True
            assert version_info["migration_message"] == "Database connection failed"

    def test_get_version_info_unknown_db_version(self, app_context):
        """Test complete version info when database version is unknown."""
        with (
            patch.object(SystemService, "get_db_version", return_value="unknown"),
            patch.object(SystemService, "check_pending_migrations", return_value=(True, None)),
        ):
            version_info = SystemService.get_version_info()

            assert version_info["migration_needed"] is True
            assert (
                "Database schema version could not be determined"
                in version_info["migration_message"]
            )

    def test_get_version_info_exception_handling(self, app_context):
        """Test that get_version_info handles exceptions gracefully."""
        with (
            patch.object(
                SystemService, "get_app_version", side_effect=Exception("Version file error")
            ),
            pytest.raises(Exception, match="Version file error"),
        ):
            # This should raise the exception since get_version_info doesn't catch it
            SystemService.get_version_info()


class TestSystemServiceLogging:
    """Test class for SystemService logging functionality."""

    def test_get_app_version_logs_error(self, app_context):
        """Test that get_app_version logs errors appropriately."""
        with (
            patch("builtins.open", side_effect=FileNotFoundError("File not found")),
            patch("app.services.logging_service.logger.log") as mock_log,
        ):
            version = SystemService.get_app_version()

            assert version == "unknown"
            mock_log.assert_called_once_with(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Failed to read VERSION file",
                details={"error": "File not found"},
            )

    def test_get_db_version_logs_error(self, app_context):
        """Test that get_db_version logs errors appropriately."""
        with (
            patch("app.models.db.session.execute", side_effect=Exception("Database error")),
            patch("app.services.logging_service.logger.log") as mock_log,
        ):
            version = SystemService.get_db_version()

            assert version == "unknown"
            mock_log.assert_called_once_with(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Failed to get database version",
                details={"error": "Database error"},
            )

    def test_check_pending_migrations_logs_error(self, app_context):
        """Test that check_pending_migrations logs errors appropriately."""
        with (
            patch("app.models.db.engine.connect", side_effect=Exception("Migration error")),
            patch("app.services.logging_service.logger.log") as mock_log,
        ):
            has_pending, error = SystemService.check_pending_migrations()

            assert has_pending is True
            assert "Error checking migrations" in error
            mock_log.assert_called_once_with(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Failed to check pending migrations",
                details={"error": "Migration error"},
            )

    def test_check_feature_availability_logs_warning(self, app_context):
        """Test that check_feature_availability logs warnings for invalid versions."""
        with patch("app.services.logging_service.logger.log") as mock_log:
            features = SystemService.check_feature_availability("invalid.version.format")

            # Should return default features
            assert features["basic_portfolio_management"] is True
            assert features["realized_gain_loss"] is False
            assert features["ibkr_integration"] is False

            mock_log.assert_called_once_with(
                level=LogLevel.WARNING,
                category=LogCategory.SYSTEM,
                message="Failed to parse database version",
                details={"version": "invalid.version.format", "error": unittest.mock.ANY},
            )

    def test_get_version_info_logs_request(self, app_context, db_session):
        """Test that get_version_info logs version check requests."""
        # Setup database version
        db_session.execute(
            db.text("CREATE TABLE IF NOT EXISTS alembic_version (version_num VARCHAR(32) NOT NULL)")
        )
        db_session.execute(db.text("DELETE FROM alembic_version"))
        db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
        db_session.commit()

        with patch("app.services.logging_service.logger.log") as mock_log:
            SystemService.get_version_info()

            # Should log the version check request (verify last call)
            mock_log.assert_called_with(
                level=LogLevel.INFO,
                category=LogCategory.SYSTEM,
                message="Version check requested",
                details={
                    "app_version": "1.3.2",
                    "db_version": "1.3.1",
                    "migration_needed": False,
                },
            )


class TestSystemServiceIntegration:
    """Integration tests for SystemService with real database."""

    def test_real_version_check_integration(self, app_context, db_session):
        """Test complete version check with real database and files."""
        # Setup database version
        db_session.execute(
            db.text("CREATE TABLE IF NOT EXISTS alembic_version (version_num VARCHAR(32) NOT NULL)")
        )
        db_session.execute(db.text("DELETE FROM alembic_version"))
        db_session.execute(db.text("INSERT INTO alembic_version (version_num) VALUES ('1.3.1')"))
        db_session.commit()

        version_info = SystemService.get_version_info()

        # Verify we get real, expected values
        assert version_info["app_version"] == "1.3.2"
        assert version_info["db_version"] == "1.3.1"
        assert version_info["migration_needed"] is False  # No migrations pending
        assert version_info["migration_message"] is None

        # Verify all expected features are enabled for current schema
        features = version_info["features"]
        assert features["basic_portfolio_management"] is True
        assert features["realized_gain_loss"] is True
        assert features["ibkr_integration"] is True

    def test_alembic_configuration_paths(self, app_context):
        """Test that Alembic configuration paths are correct."""
        # This tests the actual file paths used in check_pending_migrations
        has_pending, error = SystemService.check_pending_migrations()

        # Should not error on path issues
        assert error is None or "Error checking migrations" not in error
        assert isinstance(has_pending, bool)

    def test_version_file_path(self, app_context):
        """Test that VERSION file path is correct."""
        version = SystemService.get_app_version()

        # Should successfully read the actual VERSION file
        assert version != "unknown"
        assert version == "1.3.2"  # Current version
