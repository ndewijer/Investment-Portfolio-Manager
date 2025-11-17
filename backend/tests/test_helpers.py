"""
Standardized test helper utilities for consistent testing patterns.

This module provides common utilities used across all service tests to ensure
consistency in UUID generation, model creation, and test data management.
"""

import uuid


def make_isin(prefix: str = "US") -> str:
    """
    Generate a unique ISIN-like identifier for testing.

    Args:
        prefix: Country code prefix (default: "US")

    Returns:
        Unique ISIN-like string (e.g., "US1234567890")

    Example:
        >>> isin = make_isin("US")
        >>> len(isin) == 12  # US + 10 hex chars
        True
    """
    return f"{prefix}{uuid.uuid4().hex[:10].upper()}"


def make_symbol(base: str, unique_len: int = 4) -> str:
    """
    Generate a unique symbol for testing by appending UUID hex.

    Args:
        base: Base symbol (e.g., "AAPL", "MSFT")
        unique_len: Length of UUID hex to append (default: 4)

    Returns:
        Unique symbol string (e.g., "AAPL1A2B")

    Example:
        >>> symbol = make_symbol("AAPL", 4)
        >>> symbol.startswith("AAPL")
        True
        >>> len(symbol) == 8  # AAPL + 4 hex chars
        True
    """
    return f"{base}{uuid.uuid4().hex[:unique_len].upper()}"


def make_id() -> str:
    """
    Generate a unique ID string for model primary keys.

    Returns:
        UUID string suitable for model.id fields

    Example:
        >>> test_id = make_id()
        >>> len(test_id) == 36  # Standard UUID string length
        True
    """
    return str(uuid.uuid4())


def make_ibkr_transaction_id() -> str:
    """
    Generate a unique IBKR transaction ID for testing.

    Returns:
        Unique IBKR transaction ID (e.g., "TXN1234567890")

    Example:
        >>> txn_id = make_ibkr_transaction_id()
        >>> txn_id.startswith("TXN")
        True
    """
    return f"TXN{uuid.uuid4().hex[:10].upper()}"


def make_ibkr_txn_id() -> str:
    """
    Generate a unique IBKR transaction ID with IBKR prefix for testing.

    Returns:
        Unique IBKR transaction ID (e.g., "IBKR_1234567890")

    Example:
        >>> txn_id = make_ibkr_txn_id()
        >>> txn_id.startswith("IBKR_")
        True
    """
    return f"IBKR_{uuid.uuid4()!s}"


def make_dividend_txn_id() -> str:
    """
    Generate a unique dividend transaction ID for testing.

    Returns:
        Unique dividend transaction ID (e.g., "DIV_1234567890")

    Example:
        >>> div_id = make_dividend_txn_id()
        >>> div_id.startswith("DIV_")
        True
    """
    return f"DIV_{uuid.uuid4()!s}"


def make_portfolio_code(prefix: str = "TEST") -> str:
    """
    Generate a unique portfolio code for testing.

    Args:
        prefix: Portfolio code prefix (default: "TEST")

    Returns:
        Unique portfolio code (e.g., "TEST1A2B")

    Example:
        >>> code = make_portfolio_code("TEST")
        >>> code.startswith("TEST")
        True
    """
    return f"{prefix}{uuid.uuid4().hex[:4].upper()}"


def make_custom_string(prefix: str, length: int = 2) -> str:
    """
    Generate a custom string with prefix and UUID hex of specified length.

    Args:
        prefix: String prefix
        length: Length of UUID hex to append (default: 2)

    Returns:
        Unique string with prefix + hex

    Example:
        >>> custom = make_custom_string("WEBN", 2)
        >>> custom.startswith("WEBN")
        True
        >>> len(custom) == 6  # WEBN + 2 hex chars
        True
    """
    return f"{prefix}{uuid.uuid4().hex[:length].upper()}"


def make_portfolio_name(base: str = "Portfolio") -> str:
    """
    Generate a unique portfolio name for testing.

    Args:
        base: Base name (default: "Portfolio")

    Returns:
        Unique portfolio name

    Example:
        >>> name = make_portfolio_name("Portfolio")
        >>> name.startswith("Portfolio")
        True
    """
    return f"{base} {uuid.uuid4().hex[:6].upper()}"


# Common test data patterns
COMMON_CURRENCIES = ["USD", "EUR", "GBP", "JPY", "CAD"]
COMMON_EXCHANGES = ["NASDAQ", "NYSE", "LSE", "TSE", "XETRA"]
COMMON_COUNTRY_PREFIXES = ["US", "GB", "DE", "FR", "JP", "CA"]

# Testing constants for consistent edge case testing
INVALID_UTF8_BYTES = b"\xff\xfe\xfd"  # Invalid UTF-8 sequence
EMPTY_CSV_CONTENT = "header1,header2\n"  # CSV with headers but no data
ZERO_VALUES = [0, 0.0, "0", "0.0"]  # Common zero representations to test
