"""
Service class for IBKR configuration management operations.

This module provides methods for:
- Retrieving IBKR configuration status
- Saving/updating IBKR configuration
- Deleting IBKR configuration
- Token expiration warnings
- Managing default allocation presets
"""

import json
from datetime import datetime

from ..models import IBKRConfig, db


class IBKRConfigService:
    """
    Service class for IBKR configuration management operations.

    Provides methods for:
    - Getting configuration status
    - Saving/updating configuration
    - Deleting configuration
    """

    @staticmethod
    def get_first_config():
        """
        Get the first (and only) IBKR configuration.

        Returns:
            IBKRConfig | None: The IBKR configuration object, or None if not configured
        """
        return IBKRConfig.query.first()

    @staticmethod
    def get_config_status():
        """
        Get IBKR configuration status.

        Returns:
            dict: Configuration status with token excluded, or None if not configured
            Contains:
            - configured (bool): Whether IBKR is configured
            - flex_query_id (str, optional): Flex query ID
            - token_expires_at (str, optional): Token expiration date (ISO format)
            - token_warning (str, optional): Warning if token expires soon
            - last_import_date (str, optional): Last import date (ISO format)
            - auto_import_enabled (bool, optional): Auto import status
            - enabled (bool, optional): Configuration enabled status
            - default_allocation_enabled (bool, optional): Default allocation enabled status
            - default_allocations (str, optional): JSON string of default allocation preset
            - created_at (str, optional): Configuration creation date
            - updated_at (str, optional): Last update date
        """
        config = IBKRConfigService.get_first_config()

        if not config:
            return {"configured": False}

        # Check if token is expiring soon (within 30 days)
        token_warning = None
        if config.token_expires_at:
            # SQLite stores naive datetimes, so compare with naive datetime
            days_until_expiry = (config.token_expires_at - datetime.now()).days
            if days_until_expiry < 30:
                token_warning = f"Token expires in {days_until_expiry} days"

        # Parse default_allocations from JSON string to actual JSON
        default_allocations = None
        if config.default_allocations:
            try:
                default_allocations = json.loads(config.default_allocations)
            except (json.JSONDecodeError, TypeError):
                default_allocations = None

        return {
            "configured": True,
            "flexQueryId": str(config.flex_query_id) if config.flex_query_id is not None else None,
            "tokenExpiresAt": (
                config.token_expires_at.isoformat() if config.token_expires_at else None
            ),
            "tokenWarning": token_warning,
            "lastImportDate": (
                config.last_import_date.isoformat() if config.last_import_date else None
            ),
            "autoImportEnabled": config.auto_import_enabled,
            "enabled": config.enabled,
            "defaultAllocationEnabled": config.default_allocation_enabled,
            "defaultAllocations": default_allocations,
            "createdAt": config.created_at.isoformat(),
            "updatedAt": config.updated_at.isoformat(),
        }

    @staticmethod
    def save_config(
        flex_token,
        flex_query_id,
        token_expires_at=None,
        auto_import_enabled=None,
        enabled=None,
        default_allocation_enabled=None,
        default_allocations=None,
    ):
        """
        Save or update IBKR configuration.

        Args:
            flex_token (str): Flex token (will be encrypted)
            flex_query_id (str): Flex query ID
            token_expires_at (datetime, optional): Token expiration datetime
            auto_import_enabled (bool, optional): Auto import enabled flag
            enabled (bool, optional): Configuration enabled flag
            default_allocation_enabled (bool, optional): Default allocation enabled flag
            default_allocations (str, optional): JSON string of default allocation preset

        Returns:
            IBKRConfig: Created or updated config object

        Raises:
            ValueError: If token encryption fails
        """
        from ..services.ibkr_flex_service import IBKRFlexService

        service = IBKRFlexService()

        # Encrypt the token
        encrypted_token = service._encrypt_token(flex_token)

        # Convert default_allocations to JSON string if it's a list/dict
        if default_allocations is not None and not isinstance(default_allocations, str):
            default_allocations = json.dumps(default_allocations)

        # Get or create config
        config = IBKRConfigService.get_first_config()

        if config:
            # Update existing
            config.flex_token = encrypted_token
            config.flex_query_id = flex_query_id
            if token_expires_at:
                config.token_expires_at = token_expires_at
            if auto_import_enabled is not None:
                config.auto_import_enabled = auto_import_enabled
            if enabled is not None:
                config.enabled = enabled
            if default_allocation_enabled is not None:
                config.default_allocation_enabled = default_allocation_enabled
            if default_allocations is not None:
                config.default_allocations = default_allocations
            config.updated_at = datetime.now()
        else:
            # Create new
            config = IBKRConfig(
                flex_token=encrypted_token,
                flex_query_id=flex_query_id,
                token_expires_at=token_expires_at,
                auto_import_enabled=(
                    auto_import_enabled if auto_import_enabled is not None else False
                ),
                enabled=enabled if enabled is not None else True,
                default_allocation_enabled=(
                    default_allocation_enabled if default_allocation_enabled is not None else False
                ),
                default_allocations=default_allocations,
            )
            db.session.add(config)

        db.session.commit()
        return config

    @staticmethod
    def delete_config():
        """
        Delete IBKR configuration.

        Returns:
            dict: Deletion result with config details

        Raises:
            ValueError: If no configuration found
        """
        config = IBKRConfigService.get_first_config()

        if not config:
            raise ValueError("No configuration found")

        # Store details before deletion
        config_details = {
            "flex_query_id": config.flex_query_id,
            "was_enabled": config.enabled,
        }

        db.session.delete(config)
        db.session.commit()

        return config_details
