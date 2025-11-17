"""
Comprehensive test suite for LoggingService.

Tests logging functionality including:
- Database and file logging integration
- Log level filtering based on system settings
- Request tracking and context
- Error handling and fallback behavior
- System setting interactions
"""

import json
import logging
import os
from unittest.mock import MagicMock, patch

import pytest
from app.models import Log, LogCategory, LogLevel, SystemSetting, SystemSettingKey, db
from app.services.logging_service import LoggingService, logger, track_request
from flask import g
from tests.test_helpers import make_id


class TestLoggingServiceInit:
    """Tests for LoggingService initialization."""

    def test_init_creates_file_handler(self, app_context):
        """Test that initialization sets up file logging."""
        service = LoggingService()

        assert service.logger is not None
        assert service.logger.name == "app"
        assert len(service.logger.handlers) > 0

        # Check that log directory exists
        assert os.path.exists("logs")

    def test_setup_file_logging_creates_directory(self, app_context):
        """Test that setup creates logs directory if it doesn't exist."""
        # Remove logs directory if it exists
        if os.path.exists("logs"):
            import shutil

            shutil.rmtree("logs")

        LoggingService()  # Initialize service to create log directory

        assert os.path.exists("logs")
        assert os.path.exists("logs/app.log")


class TestShouldLog:
    """Tests for should_log method."""

    def test_should_log_when_logging_disabled(self, app_context, db_session):
        """Test that should_log returns False when logging is disabled."""
        # Clear existing settings to prevent UNIQUE constraint errors
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()

        # Set logging disabled
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="false")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()
        result = service.should_log(LogLevel.INFO)

        assert result is False

    def test_should_log_when_logging_enabled(self, app_context, db_session):
        """Test that should_log returns True when logging is enabled."""
        # Clear existing settings to prevent UNIQUE constraint errors
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()

        # Set logging enabled
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()
        result = service.should_log(LogLevel.INFO)

        assert result is True

    def test_should_log_respects_level_threshold(self, app_context, db_session):
        """Test that should_log respects the configured log level threshold."""
        # Clear existing settings to prevent UNIQUE constraint errors
        db.session.query(SystemSetting).filter(
            SystemSetting.key.in_(
                [SystemSettingKey.LOGGING_ENABLED, SystemSettingKey.LOGGING_LEVEL]
            )
        ).delete()
        db.session.commit()

        # Set logging enabled with WARNING level
        settings = [
            SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true"),
            SystemSetting(
                id=make_id(), key=SystemSettingKey.LOGGING_LEVEL, value=LogLevel.WARNING.value
            ),
        ]
        db.session.add_all(settings)
        db.session.commit()

        service = LoggingService()

        # Below threshold
        assert service.should_log(LogLevel.DEBUG) is False
        assert service.should_log(LogLevel.INFO) is False

        # At or above threshold
        assert service.should_log(LogLevel.WARNING) is True
        assert service.should_log(LogLevel.ERROR) is True
        assert service.should_log(LogLevel.CRITICAL) is True

    def test_should_log_defaults_to_true_on_error(self, app_context, db_session):
        """Test that should_log defaults to True if there's an error reading settings."""
        service = LoggingService()

        # Force an error by mocking a database exception
        with patch.object(SystemSetting, "get_value", side_effect=Exception("Database error")):
            result = service.should_log(LogLevel.INFO)
            assert result is True

    def test_should_log_with_default_settings(self, app_context, db_session):
        """Test should_log behavior with default system settings."""
        # Clear any existing settings to test defaults
        db.session.query(SystemSetting).filter(
            SystemSetting.key.in_(
                [SystemSettingKey.LOGGING_ENABLED, SystemSettingKey.LOGGING_LEVEL]
            )
        ).delete()
        db.session.commit()

        service = LoggingService()

        # Default should be enabled with INFO level
        assert service.should_log(LogLevel.DEBUG) is False  # Below INFO
        assert service.should_log(LogLevel.INFO) is True  # At INFO
        assert service.should_log(LogLevel.ERROR) is True  # Above INFO


class TestLogMethod:
    """Tests for the main log method."""

    def test_log_creates_database_entry(self, app_context, db_session):
        """Test that logging creates an entry in the database."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        response, _status = service.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Test message",
            details={"key": "value"},
            source="test_source",
            http_status=200,
        )

        # Check database entry
        log_entry = Log.query.filter_by(message="Test message").first()
        assert log_entry is not None
        assert log_entry.level == LogLevel.INFO
        assert log_entry.category == LogCategory.SYSTEM
        assert log_entry.message == "Test message"
        assert json.loads(log_entry.details) == {"key": "value"}
        assert log_entry.source == "test_source"
        assert log_entry.http_status == 200

        # Check response
        assert response["status"] == "success"
        assert response["message"] == "Test message"
        assert _status == 200

    def test_log_with_error_level(self, app_context, db_session):
        """Test logging with ERROR level returns error status."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        response, _status = service.log(
            level=LogLevel.ERROR, category=LogCategory.SYSTEM, message="Error message"
        )

        assert response["status"] == "error"
        assert response["message"] == "Error message"
        assert _status == 500

    def test_log_with_critical_level(self, app_context, db_session):
        """Test logging with CRITICAL level returns error status."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        response, _status = service.log(
            level=LogLevel.CRITICAL, category=LogCategory.SYSTEM, message="Critical message"
        )

        assert response["status"] == "error"
        assert response["message"] == "Critical message"
        assert _status == 500

    def test_log_with_user_message_override(self, app_context, db_session):
        """Test that user_message in details overrides the main message."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        response, _status = service.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Technical message",
            details={"user_message": "User-friendly message"},
        )

        assert response["message"] == "User-friendly message"

        # Database should still have the technical message
        log_entry = Log.query.filter_by(message="Technical message").first()
        assert log_entry is not None
        assert log_entry.message == "Technical message"

    def test_log_skipped_when_should_log_false(self, app_context, db_session):
        """Test that logging is skipped when should_log returns False."""
        # Disable logging
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="false")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        response, _status = service.log(
            level=LogLevel.INFO, category=LogCategory.SYSTEM, message="Should be skipped"
        )

        # Should still return response but not create database entry
        assert response["status"] == "success"
        assert response["message"] == "Should be skipped"
        assert _status == 200

        # No database entry should be created
        log_count = Log.query.filter_by(message="Should be skipped").count()
        assert log_count == 0

    @patch("app.services.logging_service.request")
    def test_log_captures_request_context(self, mock_request, app_context, db_session):
        """Test that logging captures request context when available."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        mock_request.remote_addr = "192.168.1.1"
        mock_request.user_agent = MagicMock()
        mock_request.user_agent.string = "Mozilla/5.0"

        # Set request ID in flask.g
        g.request_id = "test-request-123"

        service = LoggingService()
        service.log(level=LogLevel.INFO, category=LogCategory.SYSTEM, message="Request test")

        log_entry = Log.query.filter_by(message="Request test").first()
        assert log_entry is not None
        assert log_entry.request_id == "test-request-123"
        assert log_entry.ip_address == "192.168.1.1"
        assert log_entry.user_agent == "Mozilla/5.0"

    def test_log_handles_database_failure(self, app_context, db_session):
        """Test that logging falls back to file when database fails."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        with (
            patch.object(service.logger, "error") as mock_error,
            patch.object(service.logger, "log") as mock_log,
            patch.object(db.session, "add", side_effect=Exception("Database error")),
        ):
            response, _status = service.log(
                level=LogLevel.INFO,
                category=LogCategory.SYSTEM,
                message="Database failure test",
                details={"key": "value"},
            )

            # Should still return response
            assert response["status"] == "success"
            assert response["message"] == "Database failure test"

            # Should have logged the error and fallen back to file
            assert mock_error.called
            assert mock_log.called

    def test_log_with_stack_trace(self, app_context, db_session):
        """Test logging with explicit stack trace."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        service.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message="Error with trace",
            stack_trace="Traceback (most recent call last):\n  File test.py",
        )

        log_entry = Log.query.filter_by(message="Error with trace").first()
        assert log_entry is not None
        assert log_entry.stack_trace == "Traceback (most recent call last):\n  File test.py"

    def test_log_auto_generates_stack_trace_for_errors(self, app_context, db_session):
        """Test that ERROR level automatically captures stack trace."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        with patch("app.services.logging_service.traceback.format_exc") as mock_trace:
            mock_trace.return_value = "Auto-generated trace"

            service.log(
                level=LogLevel.ERROR, category=LogCategory.SYSTEM, message="Auto trace test"
            )

            log_entry = Log.query.filter_by(message="Auto trace test").first()
            assert log_entry is not None
            assert log_entry.stack_trace == "Auto-generated trace"

    def test_log_no_auto_trace_for_non_errors(self, app_context, db_session):
        """Test that non-ERROR levels don't auto-generate stack traces."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        service.log(level=LogLevel.INFO, category=LogCategory.SYSTEM, message="Info message")

        log_entry = Log.query.filter_by(message="Info message").first()
        assert log_entry is not None
        assert log_entry.stack_trace is None

    def test_log_with_custom_http_status(self, app_context, db_session):
        """Test logging with custom HTTP status."""
        # Ensure logging is enabled
        db.session.query(SystemSetting).filter_by(key=SystemSettingKey.LOGGING_ENABLED).delete()
        db.session.commit()
        setting = SystemSetting(id=make_id(), key=SystemSettingKey.LOGGING_ENABLED, value="true")
        db.session.add(setting)
        db.session.commit()

        service = LoggingService()

        _response, _status = service.log(
            level=LogLevel.WARNING,
            category=LogCategory.SYSTEM,
            message="Warning message",
            http_status=422,
        )

        assert _status == 422

        log_entry = Log.query.filter_by(message="Warning message").first()
        assert log_entry is not None
        assert log_entry.http_status == 422


class TestGetLoggingLevel:
    """Tests for _get_logging_level method."""

    def test_get_logging_level_conversion(self, app_context):
        """Test conversion from LogLevel enum to Python logging levels."""
        service = LoggingService()

        assert service._get_logging_level(LogLevel.DEBUG) == logging.DEBUG
        assert service._get_logging_level(LogLevel.INFO) == logging.INFO
        assert service._get_logging_level(LogLevel.WARNING) == logging.WARNING
        assert service._get_logging_level(LogLevel.ERROR) == logging.ERROR
        assert service._get_logging_level(LogLevel.CRITICAL) == logging.CRITICAL


class TestSingletonLogger:
    """Tests for the singleton logger instance."""

    def test_logger_is_logging_service_instance(self, app_context):
        """Test that the global logger is a LoggingService instance."""
        assert isinstance(logger, LoggingService)

    def test_logger_singleton_behavior(self, app_context):
        """Test that multiple imports return the same logger instance."""
        from app.services.logging_service import logger as logger2

        assert logger is logger2


class TestTrackRequestDecorator:
    """Tests for the track_request decorator."""

    def test_track_request_sets_request_id(self, app_context):
        """Test that track_request decorator sets a request ID."""

        @track_request
        def test_function():
            return g.request_id

        request_id = test_function()

        assert request_id is not None
        assert isinstance(request_id, str)
        assert len(request_id) > 0

    def test_track_request_preserves_function_result(self, app_context):
        """Test that decorator preserves the original function's return value."""

        @track_request
        def test_function(value):
            return f"result: {value}"

        result = test_function("test")
        assert result == "result: test"

    def test_track_request_preserves_function_args(self, app_context):
        """Test that decorator passes through function arguments."""

        @track_request
        def test_function(*args, **kwargs):
            return (args, kwargs)

        result = test_function("arg1", "arg2", key1="value1", key2="value2")
        expected = (("arg1", "arg2"), {"key1": "value1", "key2": "value2"})
        assert result == expected

    def test_track_request_generates_unique_ids(self, app_context):
        """Test that each request gets a unique ID."""

        @track_request
        def test_function():
            return g.request_id

        id1 = test_function()
        id2 = test_function()

        assert id1 != id2

    def test_track_request_handles_exceptions(self, app_context):
        """Test that decorator doesn't interfere with exception handling."""

        @track_request
        def test_function():
            raise ValueError("Test exception")

        with pytest.raises(ValueError, match="Test exception"):
            test_function()

        # Request ID should still be set even if function raises
        assert hasattr(g, "request_id")
