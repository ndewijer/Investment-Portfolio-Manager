"""
Comprehensive test suite for SymbolLookupService.

Tests symbol lookup functionality including:
- Cache hit/miss scenarios
- Cache expiration logic
- yfinance API integration
- Force refresh behavior
- ISIN-based lookups
- Manual symbol updates
- Error handling
"""

from datetime import UTC, datetime, timedelta
from unittest.mock import MagicMock, patch

from app.models import SymbolInfo, db
from app.services.symbol_lookup_service import SymbolLookupService
from tests.test_helpers import make_id, make_isin, make_symbol


class TestGetSymbolInfo:
    """Tests for get_symbol_info method."""

    def test_get_symbol_info_cache_hit(self, app_context, db_session):
        """Test successful cache hit with valid data."""
        # Create unique symbol
        unique_symbol = make_symbol("AAPL")
        unique_isin = make_isin("US")

        # Create cached symbol info
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Apple Inc.",
            exchange="NASDAQ",
            currency="USD",
            isin=unique_isin,
            last_updated=datetime.now(UTC),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Get symbol info (should use cache)
        result = SymbolLookupService.get_symbol_info(unique_symbol)

        assert result is not None
        assert result["symbol"] == unique_symbol
        assert result["name"] == "Apple Inc."
        assert result["exchange"] == "NASDAQ"
        assert result["currency"] == "USD"
        assert result["isin"] == unique_isin
        assert "last_updated" in result

    def test_get_symbol_info_cache_expired(self, app_context, db_session):
        """Test cache expiration triggers refresh."""
        # Create unique symbol
        unique_symbol = make_symbol("MSFT")

        # Create expired cached symbol info
        expired_time = datetime.now(UTC) - timedelta(days=8)
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Microsoft Corporation (Old)",
            exchange="NASDAQ",
            currency="USD",
            last_updated=expired_time,
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Mock yfinance
        with patch("app.services.symbol_lookup_service.yf.Ticker") as mock_ticker:
            mock_info = {
                "longName": "Microsoft Corporation",
                "exchange": "NASDAQ",
                "currency": "USD",
            }
            mock_ticker_instance = MagicMock()
            mock_ticker_instance.info = mock_info
            mock_ticker.return_value = mock_ticker_instance

            # Get symbol info (should refresh)
            result = SymbolLookupService.get_symbol_info(unique_symbol)

            assert result is not None
            assert result["symbol"] == unique_symbol
            assert result["name"] == "Microsoft Corporation"
            assert result["currency"] == "USD"

    @patch("app.services.symbol_lookup_service.yf.Ticker")
    def test_get_symbol_info_no_cache(self, mock_ticker, app_context, db_session):
        """Test fetching symbol info when not in cache."""
        # Create unique symbol
        unique_symbol = make_symbol("TSLA")

        # Mock yfinance response
        mock_info = {
            "longName": "Tesla Inc",
            "exchange": "NASDAQ",
            "currency": "USD",
        }
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.info = mock_info
        mock_ticker.return_value = mock_ticker_instance

        # Get symbol info
        result = SymbolLookupService.get_symbol_info(unique_symbol)

        assert result is not None
        assert result["symbol"] == unique_symbol
        assert result["name"] == "Tesla Inc"
        assert result["exchange"] == "NASDAQ"
        assert result["currency"] == "USD"

        # Verify cached in database
        cached = SymbolInfo.query.filter_by(symbol=unique_symbol).first()
        assert cached is not None
        assert cached.name == "Tesla Inc"
        assert cached.data_source == "yfinance"
        assert cached.is_valid is True

    @patch("app.services.symbol_lookup_service.yf.Ticker")
    def test_get_symbol_info_force_refresh(self, mock_ticker, app_context, db_session):
        """Test force refresh ignores valid cache."""
        # Create unique symbol
        unique_symbol = make_symbol("GOOGL")

        # Create cached symbol info
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Alphabet Inc. (Old)",
            exchange="NASDAQ",
            currency="USD",
            last_updated=datetime.now(UTC),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Mock yfinance with updated data
        mock_info = {
            "longName": "Alphabet Inc. (Updated)",
            "exchange": "NASDAQ",
            "currency": "USD",
        }
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.info = mock_info
        mock_ticker.return_value = mock_ticker_instance

        # Get symbol info with force refresh
        result = SymbolLookupService.get_symbol_info(unique_symbol, force_refresh=True)

        assert result is not None
        assert result["name"] == "Alphabet Inc. (Updated)"

    @patch("app.services.symbol_lookup_service.yf.Ticker")
    def test_get_symbol_info_yfinance_no_data(self, mock_ticker, app_context, db_session):
        """Test handling when yfinance returns no data."""
        # Create unique symbol
        unique_symbol = make_symbol("INVALID")

        # Mock yfinance with empty response
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.info = None
        mock_ticker.return_value = mock_ticker_instance

        # Get symbol info
        result = SymbolLookupService.get_symbol_info(unique_symbol)

        assert result is None

    @patch("app.services.symbol_lookup_service.yf.Ticker")
    def test_get_symbol_info_invalidates_cache_on_no_data(
        self, mock_ticker, app_context, db_session
    ):
        """Test that invalid symbols mark cache as invalid."""
        # Create unique symbol
        unique_symbol = make_symbol("INVALID")

        # Create cached symbol info
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Invalid Symbol",
            exchange="NASDAQ",
            currency="USD",
            last_updated=datetime.now(UTC) - timedelta(days=8),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Mock yfinance with no data
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.info = None
        mock_ticker.return_value = mock_ticker_instance

        # Get symbol info
        result = SymbolLookupService.get_symbol_info(unique_symbol)

        assert result is None

        # Verify cache marked as invalid
        cached = SymbolInfo.query.filter_by(symbol=unique_symbol).first()
        assert cached.is_valid is False

    @patch("app.services.symbol_lookup_service.yf.Ticker")
    def test_get_symbol_info_uses_short_name_fallback(self, mock_ticker, app_context, db_session):
        """Test using shortName when longName not available."""
        # Create unique symbol
        unique_symbol = make_symbol("AMZN")

        # Mock yfinance with only shortName
        mock_info = {
            "shortName": "AMZN",
            "exchange": "NASDAQ",
            "currency": "USD",
        }
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.info = mock_info
        mock_ticker.return_value = mock_ticker_instance

        # Get symbol info
        result = SymbolLookupService.get_symbol_info(unique_symbol)

        assert result is not None
        assert result["name"] == "AMZN"

    @patch("app.services.symbol_lookup_service.yf.Ticker")
    def test_get_symbol_info_uses_unknown_name_fallback(self, mock_ticker, app_context, db_session):
        """Test using 'Unknown' when no name available."""
        # Create unique symbol
        unique_symbol = make_symbol("UNKNOWN")

        # Mock yfinance with no name fields
        mock_info = {
            "exchange": "NASDAQ",
            "currency": "USD",
        }
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.info = mock_info
        mock_ticker.return_value = mock_ticker_instance

        # Get symbol info
        result = SymbolLookupService.get_symbol_info(unique_symbol)

        assert result is not None
        assert result["name"] == "Unknown"

    @patch("app.services.symbol_lookup_service.yf.Ticker")
    def test_get_symbol_info_handles_exception(self, mock_ticker, app_context, db_session):
        """Test error handling when yfinance raises exception."""
        # Create unique symbol
        unique_symbol = make_symbol("ERROR")

        # Mock yfinance to raise exception
        mock_ticker.side_effect = Exception("Network error")

        # Get symbol info
        result = SymbolLookupService.get_symbol_info(unique_symbol)

        assert result is None

    def test_get_symbol_info_skips_invalid_cache(self, app_context, db_session):
        """Test that invalid cache entries are skipped and updated."""
        # Create unique symbol
        unique_symbol = make_symbol("BADCACHE")

        # Create invalid cached symbol info
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Bad Cache",
            exchange="NASDAQ",
            currency="USD",
            last_updated=datetime.now(UTC),
            data_source="yfinance",
            is_valid=False,  # Invalid
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Mock yfinance
        with patch("app.services.symbol_lookup_service.yf.Ticker") as mock_ticker:
            mock_info = {
                "longName": "Good Data",
                "exchange": "NASDAQ",
                "currency": "USD",
            }
            mock_ticker_instance = MagicMock()
            mock_ticker_instance.info = mock_info
            mock_ticker.return_value = mock_ticker_instance

            # Get symbol info (should skip invalid cache and update it)
            result = SymbolLookupService.get_symbol_info(unique_symbol)

            assert result is not None
            assert result["name"] == "Good Data"

            # Verify cache was updated and marked valid
            cached = SymbolInfo.query.filter_by(symbol=unique_symbol).first()
            assert cached.is_valid is True
            assert cached.name == "Good Data"


class TestGetSymbolByIsin:
    """Tests for get_symbol_by_isin method."""

    def test_get_symbol_by_isin_cache_hit(self, app_context, db_session):
        """Test successful ISIN lookup from cache."""
        # Create unique symbol and ISIN
        unique_symbol = make_symbol("AAPL")
        unique_isin = make_isin("US")

        # Create cached symbol info with ISIN
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Apple Inc.",
            exchange="NASDAQ",
            currency="USD",
            isin=unique_isin,
            last_updated=datetime.now(UTC),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Look up by ISIN
        result = SymbolLookupService.get_symbol_by_isin(unique_isin)

        assert result is not None
        assert result["symbol"] == unique_symbol
        assert result["name"] == "Apple Inc."
        assert result["isin"] == unique_isin

    def test_get_symbol_by_isin_not_found(self, app_context, db_session):
        """Test ISIN lookup when not in cache."""
        # Create unique ISIN
        unique_isin = make_isin("US")

        # Look up non-existent ISIN
        result = SymbolLookupService.get_symbol_by_isin(unique_isin)

        assert result is None

    def test_get_symbol_by_isin_invalid_entry(self, app_context, db_session):
        """Test that invalid entries are skipped."""
        # Create unique symbol and ISIN
        unique_symbol = make_symbol("INVALID")
        unique_isin = make_isin("US")

        # Create invalid cached symbol info
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Invalid Symbol",
            exchange="NASDAQ",
            currency="USD",
            isin=unique_isin,
            last_updated=datetime.now(UTC),
            data_source="yfinance",
            is_valid=False,  # Invalid
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Look up by ISIN (should skip invalid)
        result = SymbolLookupService.get_symbol_by_isin(unique_isin)

        assert result is None

    @patch("app.services.symbol_lookup_service.SymbolInfo.query")
    @patch("app.services.symbol_lookup_service.logger.log")
    def test_get_symbol_by_isin_handles_exception(
        self, mock_log, mock_query, app_context, db_session
    ):
        """Test error handling when database query fails."""
        # Create unique ISIN
        unique_isin = make_isin("US")

        # Mock query to raise exception
        mock_query.filter_by.side_effect = Exception("Database error")

        # Look up by ISIN
        result = SymbolLookupService.get_symbol_by_isin(unique_isin)

        assert result is None
        # Verify error was logged
        assert mock_log.called


class TestUpdateSymbolInfo:
    """Tests for update_symbol_info method."""

    def test_update_existing_symbol(self, app_context, db_session):
        """Test updating an existing symbol."""
        # Create unique symbol
        unique_symbol = make_symbol("AAPL")
        unique_isin = make_isin("US")

        # Create existing symbol info
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Apple Inc. (Old)",
            exchange="NASDAQ",
            currency="USD",
            last_updated=datetime.now(UTC) - timedelta(days=1),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Update symbol info
        result = SymbolLookupService.update_symbol_info(
            unique_symbol,
            {
                "name": "Apple Inc. (Updated)",
                "isin": unique_isin,
            },
        )

        assert result is not None
        assert result["symbol"] == unique_symbol
        assert result["name"] == "Apple Inc. (Updated)"
        assert result["isin"] == unique_isin

        # Verify database update
        cached = SymbolInfo.query.filter_by(symbol=unique_symbol).first()
        assert cached.name == "Apple Inc. (Updated)"
        assert cached.isin == unique_isin
        assert cached.data_source == "manual"

    def test_update_creates_new_symbol(self, app_context, db_session):
        """Test creating new symbol via update."""
        # Create unique symbol and ISIN
        unique_symbol = make_symbol("TSLA")
        unique_isin = make_isin("US")

        # Update non-existent symbol
        result = SymbolLookupService.update_symbol_info(
            unique_symbol,
            {
                "name": "Tesla Inc",
                "exchange": "NASDAQ",
                "currency": "USD",
                "isin": unique_isin,
            },
        )

        assert result is not None
        assert result["symbol"] == unique_symbol
        assert result["name"] == "Tesla Inc"
        assert result["exchange"] == "NASDAQ"
        assert result["isin"] == unique_isin

        # Verify database entry
        cached = SymbolInfo.query.filter_by(symbol=unique_symbol).first()
        assert cached is not None
        assert cached.data_source == "manual"
        assert cached.is_valid is True

    def test_update_symbol_partial_data(self, app_context, db_session):
        """Test updating only some fields."""
        # Create unique symbol and ISIN
        unique_symbol = make_symbol("MSFT")
        unique_isin = make_isin("US")

        # Create existing symbol
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Microsoft Corporation",
            exchange="NASDAQ",
            currency="USD",
            last_updated=datetime.now(UTC),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Update only ISIN
        result = SymbolLookupService.update_symbol_info(
            unique_symbol,
            {"isin": unique_isin},
        )

        assert result is not None
        assert result["name"] == "Microsoft Corporation"  # Unchanged
        assert result["exchange"] == "NASDAQ"  # Unchanged
        assert result["isin"] == unique_isin  # Updated

    def test_update_symbol_all_fields(self, app_context, db_session):
        """Test updating all fields."""
        # Create unique symbol and ISIN
        unique_symbol = make_symbol("GOOGL")
        unique_isin = make_isin("US")

        # Create existing symbol
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Alphabet Inc. (Old)",
            exchange="OLD",
            currency="OLD",
            last_updated=datetime.now(UTC),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()

        # Update all fields
        result = SymbolLookupService.update_symbol_info(
            unique_symbol,
            {
                "name": "Alphabet Inc. Class A",
                "exchange": "NASDAQ",
                "currency": "USD",
                "isin": unique_isin,
            },
        )

        assert result is not None
        assert result["name"] == "Alphabet Inc. Class A"
        assert result["exchange"] == "NASDAQ"
        assert result["currency"] == "USD"
        assert result["isin"] == unique_isin

    @patch("app.services.symbol_lookup_service.SymbolInfo.query")
    @patch("app.services.symbol_lookup_service.logger.log")
    def test_update_symbol_handles_exception(self, mock_log, mock_query, app_context, db_session):
        """Test error handling when update fails."""
        # Create unique symbol
        unique_symbol = make_symbol("ERROR")

        # Mock query to raise exception
        mock_query.filter_by.side_effect = Exception("Database error")

        # Try to update
        result = SymbolLookupService.update_symbol_info(
            unique_symbol,
            {"name": "Test"},
        )

        assert result is None
        # Verify error was logged
        assert mock_log.called

    def test_update_symbol_empty_data(self, app_context, db_session):
        """Test updating with empty data dict."""
        # Create unique symbol
        unique_symbol = make_symbol("AMZN")

        # Create existing symbol
        symbol_info = SymbolInfo(
            id=make_id(),
            symbol=unique_symbol,
            name="Amazon.com Inc.",
            exchange="NASDAQ",
            currency="USD",
            last_updated=datetime.now(UTC) - timedelta(days=1),
            data_source="yfinance",
            is_valid=True,
        )
        db.session.add(symbol_info)
        db.session.commit()
        original_time = symbol_info.last_updated

        # Update with empty data
        result = SymbolLookupService.update_symbol_info(unique_symbol, {})

        assert result is not None
        assert result["name"] == "Amazon.com Inc."  # Unchanged
        assert result["exchange"] == "NASDAQ"  # Unchanged

        # Verify last_updated was updated even with no data changes
        cached = SymbolInfo.query.filter_by(symbol=unique_symbol).first()
        assert cached.last_updated > original_time
        assert cached.data_source == "manual"
