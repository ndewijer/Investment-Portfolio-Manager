"""
Tests for IBKRConfigService.

This test suite covers:
- Configuration status retrieval
- Configuration creation and updates
- Token encryption handling
- Token expiration warnings
- Configuration deletion
- Error handling for invalid operations
"""

from datetime import datetime, timedelta
from unittest.mock import patch

import pytest
from app.models import IBKRConfig, db
from app.services.ibkr_config_service import IBKRConfigService
from tests.test_helpers import make_id


@pytest.fixture
def mock_encryption():
    """Mock encryption/decryption functions for testing."""
    with (
        patch("app.services.ibkr_flex_service.IBKRFlexService._encrypt_token") as mock_encrypt,
        patch("app.services.ibkr_flex_service.IBKRFlexService._decrypt_token") as mock_decrypt,
    ):
        # Simple mock: return token with "encrypted_" prefix
        mock_encrypt.side_effect = lambda token: f"encrypted_{token}"
        mock_decrypt.side_effect = lambda token: token.replace("encrypted_", "")

        yield mock_encrypt, mock_decrypt


@pytest.fixture(autouse=True)
def clean_ibkr_config(db_session):
    """Clean up IBKRConfig before each test for isolation."""
    # Clean before test
    IBKRConfig.query.delete()
    db_session.commit()

    yield

    # Clean after test
    IBKRConfig.query.delete()
    db_session.commit()


class TestConfigRetrieval:
    """Tests for configuration status retrieval."""

    def test_get_config_status_not_configured(self, app_context, db_session):
        """Test getting status when no configuration exists."""
        status = IBKRConfigService.get_config_status()

        assert status["configured"] is False
        assert len(status) == 1  # Only 'configured' field

    def test_get_config_status_configured(self, app_context, db_session):
        """Test getting status when configuration exists."""
        # Create config with mock encrypted token
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token_12345",
            flex_query_id="123456",
            auto_import_enabled=True,
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        # Get status
        status = IBKRConfigService.get_config_status()

        # Verify status
        assert status["configured"] is True
        assert status["flex_query_id"] == "123456"
        assert status["auto_import_enabled"] is True
        assert status["enabled"] is True
        assert status["last_import_date"] is None
        assert status["token_expires_at"] is None
        assert status["token_warning"] is None
        assert "created_at" in status
        assert "updated_at" in status

    def test_get_config_status_with_expiration_far_future(self, app_context, db_session):
        """Test status when token expires in far future (no warning)."""
        # Create config with token expiring in 60 days
        expires_at = datetime.now() + timedelta(days=60)
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token",
            flex_query_id="123456",
            token_expires_at=expires_at,
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        # Get status
        status = IBKRConfigService.get_config_status()

        assert status["configured"] is True
        assert status["token_expires_at"] == expires_at.isoformat()
        assert status["token_warning"] is None  # No warning for 60+ days

    def test_get_config_status_with_expiration_warning(self, app_context, db_session):
        """Test status when token expires soon (warning shown)."""
        # Create config with token expiring in 15 days
        expires_at = datetime.now() + timedelta(days=15)
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token",
            flex_query_id="123456",
            token_expires_at=expires_at,
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        # Get status
        status = IBKRConfigService.get_config_status()

        assert status["configured"] is True
        assert status["token_warning"] is not None
        # Should show warning for tokens expiring within 30 days
        assert "days" in status["token_warning"]
        assert "expires" in status["token_warning"]

    def test_get_config_status_with_last_import(self, app_context, db_session):
        """Test status includes last import date."""
        # Create config with last import date
        last_import = datetime(2024, 1, 15, 10, 30, 0)
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token",
            flex_query_id="123456",
            last_import_date=last_import,
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        # Get status
        status = IBKRConfigService.get_config_status()

        assert status["configured"] is True
        assert status["last_import_date"] == last_import.isoformat()


class TestConfigSave:
    """Tests for configuration creation and updates."""

    def test_save_config_create_new(self, app_context, db_session, mock_encryption):
        """Test creating new configuration."""
        mock_encrypt, _mock_decrypt = mock_encryption

        # Save new config
        config = IBKRConfigService.save_config(
            flex_token="test_token_12345", flex_query_id="654321"
        )

        # Verify creation
        assert config.id is not None
        assert config.flex_query_id == "654321"
        assert config.flex_token == "encrypted_test_token_12345"  # Mocked encryption
        assert config.enabled is True  # Default
        assert config.auto_import_enabled is False  # Default
        assert config.token_expires_at is None
        assert config.last_import_date is None

        # Verify encryption was called
        mock_encrypt.assert_called_once_with("test_token_12345")

        # Verify in database
        db_config = IBKRConfig.query.first()
        assert db_config is not None
        assert db_config.flex_query_id == "654321"

    def test_save_config_with_all_fields(self, app_context, db_session, mock_encryption):
        """Test creating config with all optional fields."""
        _mock_encrypt, _mock_decrypt = mock_encryption
        expires_at = datetime.now() + timedelta(days=90)

        config = IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            token_expires_at=expires_at,
            auto_import_enabled=True,
            enabled=True,
        )

        assert config.flex_query_id == "123456"
        assert config.token_expires_at == expires_at
        assert config.auto_import_enabled is True
        assert config.enabled is True

    def test_save_config_update_existing(self, app_context, db_session, mock_encryption):
        """Test updating existing configuration."""
        _mock_encrypt, _mock_decrypt = mock_encryption

        # Create initial config
        old_config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_old_token",
            flex_query_id="111111",
            auto_import_enabled=False,
            enabled=True,
        )
        db_session.add(old_config)
        db_session.commit()
        config_id = old_config.id

        # Update config
        updated_config = IBKRConfigService.save_config(
            flex_token="new_token", flex_query_id="222222", auto_import_enabled=True
        )

        # Verify update (same record, not new)
        assert updated_config.id == config_id
        assert updated_config.flex_query_id == "222222"
        assert updated_config.flex_token == "encrypted_new_token"  # Token changed
        assert updated_config.auto_import_enabled is True

        # Verify only one config exists
        all_configs = IBKRConfig.query.all()
        assert len(all_configs) == 1

    def test_save_config_update_partial_fields(self, app_context, db_session, mock_encryption):
        """Test updating only some fields preserves others."""
        _mock_encrypt, _mock_decrypt = mock_encryption

        # Create initial config with expiration
        expires_at = datetime.now() + timedelta(days=90)
        old_config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_old_token",
            flex_query_id="111111",
            token_expires_at=expires_at,
            auto_import_enabled=True,
            enabled=True,
        )
        db_session.add(old_config)
        db_session.commit()

        # Update only query ID (should preserve expiration and flags)
        updated_config = IBKRConfigService.save_config(
            flex_token="new_token", flex_query_id="222222"
        )

        # Verify partial update
        assert updated_config.flex_query_id == "222222"
        assert updated_config.token_expires_at == expires_at  # Preserved
        assert updated_config.auto_import_enabled is True  # Preserved
        assert updated_config.enabled is True  # Preserved

    def test_save_config_token_encryption(self, app_context, db_session, mock_encryption):
        """Test that token is properly encrypted."""
        mock_encrypt, mock_decrypt = mock_encryption
        plaintext_token = "plaintext_secret_token_12345"

        config = IBKRConfigService.save_config(flex_token=plaintext_token, flex_query_id="123456")

        # Token should be encrypted (mocked)
        assert config.flex_token == f"encrypted_{plaintext_token}"

        # Verify encryption was called with plaintext
        mock_encrypt.assert_called_once_with(plaintext_token)

        # Should be able to decrypt it (test via mock)
        decrypted = mock_decrypt(config.flex_token)
        assert decrypted == plaintext_token

    def test_save_config_disable_config(self, app_context, db_session, mock_encryption):
        """Test disabling configuration."""
        _mock_encrypt, _mock_decrypt = mock_encryption

        # Create enabled config
        config = IBKRConfigService.save_config(
            flex_token="test_token", flex_query_id="123456", enabled=True
        )
        assert config.enabled is True

        # Disable it
        updated_config = IBKRConfigService.save_config(
            flex_token="test_token", flex_query_id="123456", enabled=False
        )
        assert updated_config.enabled is False


class TestConfigDeletion:
    """Tests for configuration deletion."""

    def test_delete_config_success(self, app_context, db_session):
        """Test deleting existing configuration."""
        # Create config
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token",
            flex_query_id="123456",
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()
        config_id = config.id

        # Delete config
        result = IBKRConfigService.delete_config()

        # Verify result
        assert result["flex_query_id"] == "123456"
        assert result["was_enabled"] is True

        # Verify config deleted from database
        deleted_config = db.session.get(IBKRConfig, config_id)
        assert deleted_config is None

    def test_delete_config_not_found(self, app_context, db_session):
        """Test deleting when no configuration exists."""
        with pytest.raises(ValueError, match="No configuration found"):
            IBKRConfigService.delete_config()


class TestEdgeCases:
    """Tests for edge cases and boundary conditions."""

    def test_get_config_status_token_expires_today(self, app_context, db_session):
        """Test warning when token expires today."""
        # Token expires in a few hours (same day)
        expires_at = datetime.now() + timedelta(hours=5)
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token",
            flex_query_id="123456",
            token_expires_at=expires_at,
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        status = IBKRConfigService.get_config_status()

        assert status["token_warning"] is not None
        assert "0 days" in status["token_warning"]

    def test_save_config_enabled_defaults_to_true(self, app_context, db_session, mock_encryption):
        """Test that enabled defaults to True when not specified."""
        _mock_encrypt, _mock_decrypt = mock_encryption

        config = IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            # enabled not specified
        )

        assert config.enabled is True  # Default

    def test_save_config_auto_import_disabled_by_default(
        self, app_context, db_session, mock_encryption
    ):
        """Test that auto_import defaults to False when not specified."""
        _mock_encrypt, _mock_decrypt = mock_encryption

        config = IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            # auto_import_enabled not specified
        )

        assert config.auto_import_enabled is False  # Default

    def test_save_config_update_token_expires_at(self, app_context, db_session, mock_encryption):
        """Test updating token expiration date."""
        _mock_encrypt, _mock_decrypt = mock_encryption

        # Create config without expiration
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token",
            flex_query_id="123456",
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        # Update with expiration date
        new_expires_at = datetime.now() + timedelta(days=90)
        updated_config = IBKRConfigService.save_config(
            flex_token="test_token", flex_query_id="123456", token_expires_at=new_expires_at
        )

        assert updated_config.token_expires_at == new_expires_at

    def test_get_config_status_excludes_token(self, app_context, db_session):
        """Test that status never includes the encrypted token."""
        encrypted_token = "encrypted_secret_token"
        config = IBKRConfig(
            id=make_id(), flex_token=encrypted_token, flex_query_id="123456", enabled=True
        )
        db_session.add(config)
        db_session.commit()

        status = IBKRConfigService.get_config_status()

        # Verify token is NOT in status
        assert "flex_token" not in status
        assert encrypted_token not in str(status.values())


class TestDefaultAllocationConfig:
    """Test default allocation configuration functionality."""

    def test_save_config_with_default_allocations(self, app_context, db_session, mock_encryption):
        """Test saving config with default allocation settings."""
        allocations_json = (
            '[{"portfolio_id": "p1", "percentage": 60.0}, '
            '{"portfolio_id": "p2", "percentage": 40.0}]'
        )

        config = IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            default_allocation_enabled=True,
            default_allocations=allocations_json,
        )

        assert config.default_allocation_enabled is True
        assert config.default_allocations == allocations_json

    def test_get_config_status_includes_default_allocations(self, app_context, db_session):
        """Test that status includes default allocation fields."""
        allocations_json = '[{"portfolio_id": "p1", "percentage": 100.0}]'
        config = IBKRConfig(
            id=make_id(),
            flex_token="encrypted_test_token",
            flex_query_id="123456",
            enabled=True,
            default_allocation_enabled=True,
            default_allocations=allocations_json,
        )
        db_session.add(config)
        db_session.commit()

        status = IBKRConfigService.get_config_status()

        assert status["default_allocation_enabled"] is True
        assert status["default_allocations"] == allocations_json

    def test_update_config_default_allocations(self, app_context, db_session, mock_encryption):
        """Test updating default allocation settings."""
        # Create initial config
        config = IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            default_allocation_enabled=False,
            default_allocations=None,
        )

        assert config.default_allocation_enabled is False
        assert config.default_allocations is None

        # Update with default allocations
        allocations_json = (
            '[{"portfolio_id": "p1", "percentage": 50.0}, '
            '{"portfolio_id": "p2", "percentage": 50.0}]'
        )
        updated_config = IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            default_allocation_enabled=True,
            default_allocations=allocations_json,
        )

        assert updated_config.default_allocation_enabled is True
        assert updated_config.default_allocations == allocations_json

    def test_disable_default_allocation(self, app_context, db_session, mock_encryption):
        """Test disabling default allocation while keeping preset."""
        allocations_json = '[{"portfolio_id": "p1", "percentage": 100.0}]'

        # Create with enabled
        IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            default_allocation_enabled=True,
            default_allocations=allocations_json,
        )

        # Disable allocation
        updated_config = IBKRConfigService.save_config(
            flex_token="test_token",
            flex_query_id="123456",
            default_allocation_enabled=False,
            default_allocations=allocations_json,  # Keep preset
        )

        assert updated_config.default_allocation_enabled is False
        assert updated_config.default_allocations == allocations_json
