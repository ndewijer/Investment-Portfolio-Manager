"""Security utilities for the API."""

import hashlib
import os
from datetime import UTC, datetime
from functools import wraps

from flask import jsonify, request


def require_api_key(f):
    """
    Decorator to require an API key and time-based token for protected routes.

    Args:
        f: The function to decorate

    Returns:
        Decorated function to check for API key and time-based token
    """

    @wraps(f)
    def decorated_function(*args, **kwargs):
        """
        Decorated function to check for API key and time-based token.

        Args:
            *args: Positional arguments
            **kwargs: Keyword arguments

        Returns:
            JSON response with error message if unauthorized
        """
        # Get the API key from environment variable
        valid_api_key = os.environ.get("INTERNAL_API_KEY")

        # Check if API key is provided and valid
        api_key = request.headers.get("X-API-Key")
        if not api_key or api_key != valid_api_key:
            return jsonify({"error": "Unauthorized"}), 401

        # Generate a time-based token (changes every hour)
        current_hour = datetime.now(UTC).strftime("%Y-%m-%d-%H")
        time_token = hashlib.sha256(f"{valid_api_key}{current_hour}".encode()).hexdigest()

        # Check if time token matches
        provided_token = request.headers.get("X-Time-Token")
        if not provided_token or provided_token != time_token:
            return jsonify({"error": "Invalid token"}), 401

        return f(*args, **kwargs)

    return decorated_function
