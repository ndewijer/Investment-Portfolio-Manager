"""
Task to update prices for all funds with symbols.

Should be run once per day after market close.
"""

import requests
from datetime import datetime, UTC
import hashlib
import os
from ..services.logging_service import logger
from ..models import LogLevel, LogCategory


def update_all_fund_prices():
    """Update prices for all funds with symbols."""

    try:
        # Get API key from environment
        api_key = os.environ.get("INTERNAL_API_KEY")
        if not api_key:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="INTERNAL_API_KEY not set in environment",
                details={"error": "Missing required environment variable"},
            )
            return

        # Generate time-based token
        current_hour = datetime.now(UTC).strftime("%Y-%m-%d-%H")
        time_token = hashlib.sha256(f"{api_key}{current_hour}".encode()).hexdigest()

        # Set up headers
        headers = {"X-API-Key": api_key, "X-Time-Token": time_token}

        # Make the API call with headers
        response = requests.post(
            "http://localhost:5000/api/funds/update-all-prices", headers=headers
        )

        if response.status_code == 200:
            result = response.json()
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.SYSTEM,
                message=f"Successfully updated {result['total_updated']} funds",
                details={
                    "updated_funds": result["updated_funds"],
                    "total_updated": result["total_updated"],
                },
            )

            if result["errors"]:
                logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.SYSTEM,
                    message=f"Encountered {len(result['errors'])} errors during fund price updates",
                    details={
                        "errors": result["errors"],
                        "total_errors": len(result["errors"]),
                    },
                )
        else:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Failed to update fund prices",
                details={
                    "status_code": response.status_code,
                    "response": response.text,
                },
            )

    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message="Error running price updates",
            details={"error": str(e)},
        )


if __name__ == "__main__":
    update_all_fund_prices()
