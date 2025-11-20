"""
Service class for IBKR configuration management operations.

This module provides methods for:
- Retrieving IBKR configuration status
- Saving/updating IBKR configuration
- Deleting IBKR configuration
- Token expiration warnings
"""

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

        return {
            "configured": True,
            "flex_query_id": config.flex_query_id,
            "token_expires_at": (
                config.token_expires_at.isoformat() if config.token_expires_at else None
            ),
            "token_warning": token_warning,
            "last_import_date": (
                config.last_import_date.isoformat() if config.last_import_date else None
            ),
            "auto_import_enabled": config.auto_import_enabled,
            "enabled": config.enabled,
            "created_at": config.created_at.isoformat(),
            "updated_at": config.updated_at.isoformat(),
        }

    @staticmethod
    def save_config(
        flex_token, flex_query_id, token_expires_at=None, auto_import_enabled=None, enabled=None
    ):
        """
        Save or update IBKR configuration.

        Args:
            flex_token (str): Flex token (will be encrypted)
            flex_query_id (str): Flex query ID
            token_expires_at (datetime, optional): Token expiration datetime
            auto_import_enabled (bool, optional): Auto import enabled flag
            enabled (bool, optional): Configuration enabled flag

        Returns:
            IBKRConfig: Created or updated config object

        Raises:
            ValueError: If token encryption fails
        """
        from ..services.ibkr_flex_service import IBKRFlexService

        service = IBKRFlexService()

        # Encrypt the token
        encrypted_token = service._encrypt_token(flex_token)

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
