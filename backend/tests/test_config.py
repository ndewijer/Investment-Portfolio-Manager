"""
Test configuration for the Investment Portfolio Manager test suite.

This module provides configuration for the test database and test environment.
"""

from pathlib import Path

# Get the backend directory path
BACKEND_DIR = Path(__file__).parent.parent
DATA_DIR = BACKEND_DIR / "data" / "db"

# Ensure the data directory exists
DATA_DIR.mkdir(parents=True, exist_ok=True)

# Test database configuration
TEST_DATABASE_PATH = DATA_DIR / "test_portfolio_manager.db"
TEST_DATABASE_URI = f"sqlite:///{TEST_DATABASE_PATH}"

# Test configuration dictionary
TEST_CONFIG = {
    "TESTING": True,
    "SQLALCHEMY_DATABASE_URI": TEST_DATABASE_URI,
    "SQLALCHEMY_TRACK_MODIFICATIONS": False,
    "SQLALCHEMY_ECHO": False,  # Set to True to see SQL queries during tests
    "SECRET_KEY": "test-secret-key-not-for-production",
    # Disable CSRF protection in tests
    "WTF_CSRF_ENABLED": False,
}


def get_test_database_path():
    """
    Get the path to the test database file.

    Returns:
        Path: Path object pointing to test database file
    """
    return TEST_DATABASE_PATH


def cleanup_test_database():
    """
    Remove the test database file if it exists.

    This is useful for cleaning up after tests or ensuring a fresh start.
    """
    if TEST_DATABASE_PATH.exists():
        TEST_DATABASE_PATH.unlink()


def database_exists():
    """
    Check if the test database file exists.

    Returns:
        bool: True if test database exists, False otherwise
    """
    return TEST_DATABASE_PATH.exists()
