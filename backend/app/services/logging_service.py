"""
Service class for logging operations.

Provides methods for:
- Unified logging to both database and file
- Request tracking
"""

import json
import logging
import os
import traceback
import uuid
from functools import wraps

from flask import g, request

from ..models import Log, LogCategory, LogLevel, SystemSetting, SystemSettingKey, db


class LoggingService:
    """
    Service class for logging operations.

    Provides methods for:
    - Unified logging to both database and file
    - Request tracking
    """

    def __init__(self):
        """Set up file logging as fallback."""
        self.setup_file_logging()

    def setup_file_logging(self):
        """Set up file logging as fallback."""
        log_dir = "logs"
        if not os.path.exists(log_dir):
            os.makedirs(log_dir)

        file_handler = logging.FileHandler(f"{log_dir}/app.log")
        file_handler.setFormatter(logging.Formatter("%(asctime)s - %(levelname)s - %(message)s"))

        self.logger = logging.getLogger("app")
        self.logger.addHandler(file_handler)
        self.logger.setLevel(logging.INFO)

    def should_log(self, level: LogLevel) -> bool:
        """Check if logging is enabled and if the level should be logged.

        Args:
            level (LogLevel): The level to check

        Returns:
            bool: True if logging is enabled and the level should be logged,
            False otherwise

        Raises:
            None
        """
        try:
            logging_enabled = (
                SystemSetting.get_value(SystemSettingKey.LOGGING_ENABLED, "true").lower() == "true"
            )
            if not logging_enabled:
                return False

            configured_level = SystemSetting.get_value(
                SystemSettingKey.LOGGING_LEVEL, LogLevel.INFO.value
            )
            log_levels = {
                LogLevel.DEBUG.value: 0,
                LogLevel.INFO.value: 1,
                LogLevel.WARNING.value: 2,
                LogLevel.ERROR.value: 3,
                LogLevel.CRITICAL.value: 4,
            }

            return log_levels[level.value] >= log_levels[configured_level]
        except Exception:
            # If there's any error checking settings, default to logging everything
            return True

    def log(
        self,
        level: LogLevel,
        category: LogCategory,
        message: str,
        details: dict | None = None,
        source: str | None = None,
        http_status: int | None = None,
        stack_trace: str | None = None,
    ):
        """Unified logging function that logs to both database and file.

        Args:
            level (LogLevel): The level of the log
            category (LogCategory): The category of the log
            message (str): The message of the log
            details (dict): The details of the log
            source (str): The source of the log
            http_status (int): The HTTP status of the log
            stack_trace (str): The stack trace of the log

        Returns:
            tuple: A tuple containing the response and the HTTP status

        Raises:
            None
        """

        if not self.should_log(level):
            # If logging is disabled or level is below threshold, just return the response
            response = {
                "status": ("error" if level in [LogLevel.ERROR, LogLevel.CRITICAL] else "success"),
                "message": message,
            }
            if details and "user_message" in details:
                response["message"] = details["user_message"]
            return response, http_status or (500 if level == LogLevel.ERROR else 200)

        # Create log entry
        log_entry = Log(
            id=str(uuid.uuid4()),
            level=level,
            category=category,
            message=message,
            details=json.dumps(details) if details else None,
            source=source or traceback.extract_stack()[-2][2],
            request_id=getattr(g, "request_id", None),
            stack_trace=stack_trace
            or (traceback.format_exc() if level == LogLevel.ERROR else None),
            http_status=http_status,
            ip_address=request.remote_addr if request else None,
            user_agent=request.user_agent.string if request else None,
        )

        # Try to save to database
        try:
            db.session.add(log_entry)
            db.session.commit()
        except Exception as e:
            # Fallback to file logging if database is unavailable
            self.logger.error(f"Failed to write to database, falling back to file: {e!s}")
            self.logger.log(
                self._get_logging_level(level),
                f"{category.value.upper()} - {message} - {json.dumps(details) if details else ''}",
            )

        # Return formatted response for API
        response = {
            "status": ("error" if level in [LogLevel.ERROR, LogLevel.CRITICAL] else "success"),
            "message": message,
        }

        if details and "user_message" in details:
            response["message"] = details["user_message"]

        return response, http_status or (500 if level == LogLevel.ERROR else 200)

    def _get_logging_level(self, level: LogLevel):
        """Convert our LogLevel to Python's logging level.

        Args:
            level (LogLevel): The level to convert

        Returns:
            logging.Level: The logging level

        Raises:
            None
        """
        return {
            LogLevel.DEBUG: logging.DEBUG,
            LogLevel.INFO: logging.INFO,
            LogLevel.WARNING: logging.WARNING,
            LogLevel.ERROR: logging.ERROR,
            LogLevel.CRITICAL: logging.CRITICAL,
        }[level]


# Create singleton instance
logger = LoggingService()


def track_request(f):
    """Decorator for request tracking.

    Args:
        f (function): The function to decorate

    Returns:
        function: The decorated function

    Raises:
        None
    """

    @wraps(f)
    def decorated(*args, **kwargs):
        """Decorated function implementation.

        Args:
            *args: The arguments to the function
            **kwargs: The keyword arguments to the function

        Returns:
            The result of the decorated function

        Raises:
            None
        """
        g.request_id = str(uuid.uuid4())
        return f(*args, **kwargs)

    return decorated
