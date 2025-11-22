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
            error_levels = [LogLevel.ERROR, LogLevel.CRITICAL]
            status = http_status or (500 if level in error_levels else 200)
            return response, status

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

        error_levels = [LogLevel.ERROR, LogLevel.CRITICAL]
        status = http_status or (500 if level in error_levels else 200)
        return response, status

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

    @staticmethod
    def get_logging_settings() -> dict:
        """
        Get current logging configuration settings.

        Returns:
            dict: Dictionary containing 'enabled' (bool) and 'level' (str)

        Raises:
            Exception: If unable to retrieve settings
        """
        settings = {
            "enabled": SystemSetting.get_value(SystemSettingKey.LOGGING_ENABLED, "true").lower()
            == "true",
            "level": SystemSetting.get_value(SystemSettingKey.LOGGING_LEVEL, LogLevel.INFO.value),
        }
        return settings

    @staticmethod
    def update_logging_settings(enabled: bool, level: str) -> dict:
        """
        Update logging configuration settings.

        Args:
            enabled: Enable/disable logging
            level: Logging level value

        Returns:
            dict: Dictionary containing updated 'enabled' (bool) and 'level' (str)

        Raises:
            Exception: If unable to update settings
        """
        from sqlalchemy import select

        # Get or create enabled setting
        stmt = select(SystemSetting).where(SystemSetting.key == SystemSettingKey.LOGGING_ENABLED)
        enabled_setting = db.session.execute(stmt).scalar_one_or_none()

        if not enabled_setting:
            enabled_setting = SystemSetting(key=SystemSettingKey.LOGGING_ENABLED)
        enabled_setting.value = str(enabled).lower()

        # Get or create level setting
        stmt = select(SystemSetting).where(SystemSetting.key == SystemSettingKey.LOGGING_LEVEL)
        level_setting = db.session.execute(stmt).scalar_one_or_none()

        if not level_setting:
            level_setting = SystemSetting(key=SystemSettingKey.LOGGING_LEVEL)
        level_setting.value = level

        db.session.add(enabled_setting)
        db.session.add(level_setting)
        db.session.commit()

        return {
            "enabled": enabled_setting.value.lower() == "true",
            "level": level_setting.value,
        }

    @staticmethod
    def get_logs(
        levels: str | None = None,
        categories: str | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        source: str | None = None,
        sort_by: str = "timestamp",
        sort_dir: str = "desc",
        page: int = 1,
        per_page: int = 50,
    ) -> dict:
        """
        Retrieve filtered, sorted, and paginated system logs.

        Args:
            levels: Comma-separated list of log levels to filter
            categories: Comma-separated list of log categories to filter
            start_date: Start date in ISO format
            end_date: End date in ISO format
            source: Source filter (partial match)
            sort_by: Field to sort by (default: timestamp)
            sort_dir: Sort direction ('asc' or 'desc', default: desc)
            page: Page number (default: 1)
            per_page: Items per page (default: 50)

        Returns:
            dict: Dictionary containing 'logs' list, 'total', 'pages', and 'current_page'

        Raises:
            Exception: If unable to retrieve logs
        """
        from datetime import datetime

        from sqlalchemy import or_, select

        # Build base query
        stmt = select(Log)

        # Apply filters
        if levels:
            levels_list = levels.split(",")
            level_filters = [Log.level == LogLevel(lvl.lower()) for lvl in levels_list]
            stmt = stmt.where(or_(*level_filters))

        if categories:
            category_list = categories.split(",")
            category_filters = [Log.category == LogCategory(cat.lower()) for cat in category_list]
            stmt = stmt.where(or_(*category_filters))

        if start_date:
            # Parse ISO timestamp string (already in UTC)
            start_datetime = datetime.fromisoformat(start_date.replace("Z", "+00:00"))
            stmt = stmt.where(Log.timestamp >= start_datetime)

        if end_date:
            # Parse ISO timestamp string (already in UTC)
            end_datetime = datetime.fromisoformat(end_date.replace("Z", "+00:00"))
            stmt = stmt.where(Log.timestamp <= end_datetime)

        if source:
            stmt = stmt.where(Log.source.like(f"%{source}%"))

        # Apply sorting
        if sort_dir == "desc":
            stmt = stmt.order_by(getattr(Log, sort_by).desc())
        else:
            stmt = stmt.order_by(getattr(Log, sort_by).asc())

        # Get paginated results
        pagination = db.paginate(stmt, page=page, per_page=per_page)

        return {
            "logs": [
                {
                    "id": log.id,
                    "timestamp": log.timestamp.strftime("%Y-%m-%d %H:%M:%S.%f"),
                    "level": log.level.value,
                    "category": log.category.value,
                    "message": log.message,
                    "details": json.loads(log.details) if log.details else None,
                    "source": log.source,
                    "request_id": log.request_id,
                    "http_status": log.http_status,
                    "ip_address": log.ip_address,
                    "user_agent": log.user_agent,
                }
                for log in pagination.items
            ],
            "total": pagination.total,
            "pages": pagination.pages,
            "current_page": pagination.page,
        }

    @staticmethod
    def clear_logs() -> None:
        """
        Clear all system logs from the database.

        Returns:
            None

        Raises:
            Exception: If unable to clear logs
        """
        from sqlalchemy import delete

        stmt = delete(Log)
        db.session.execute(stmt)
        db.session.commit()


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
