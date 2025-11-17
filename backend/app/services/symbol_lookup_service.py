"""Service class for managing trading symbol information."""

import uuid
from datetime import UTC, datetime, timedelta

import yfinance as yf

from ..models import LogCategory, LogLevel, SymbolInfo, db
from ..services.logging_service import logger


class SymbolLookupService:
    """
    Service class for managing trading symbol information.

    Provides methods for:
    - Looking up symbol information
    - Caching symbol data
    - Managing symbol metadata
    """

    CACHE_DURATION = timedelta(days=7)  # Cache data for 7 days

    @staticmethod
    def get_symbol_info(symbol: str, force_refresh: bool = False) -> dict:
        """
        Get symbol information from cache or yfinance.

        Args:
            symbol (str): Trading symbol to look up
            force_refresh (bool, optional): Force refresh from external source

        Returns:
            dict: Symbol information containing:
                - symbol: Trading symbol
                - name: Company/Fund name
                - exchange: Trading exchange
                - currency: Trading currency
                - isin: ISIN if available
                - last_updated: Timestamp of last update
            None: If symbol not found
        """
        try:
            # Check cache first (regardless of is_valid to avoid UNIQUE constraint errors)
            cached_info = SymbolInfo.query.filter_by(symbol=symbol).first()

            # Check if we can use cached data
            if cached_info and cached_info.is_valid and not force_refresh:
                # Ensure last_updated is timezone-aware
                last_updated = cached_info.last_updated.replace(tzinfo=UTC)
                # Check if cache is still valid
                if datetime.now(UTC) - last_updated < SymbolLookupService.CACHE_DURATION:
                    return {
                        "symbol": cached_info.symbol,
                        "name": cached_info.name,
                        "exchange": cached_info.exchange,
                        "currency": cached_info.currency,
                        "isin": cached_info.isin,
                        "last_updated": last_updated.isoformat(),
                    }

            # Fetch from yfinance
            ticker = yf.Ticker(symbol)
            info = ticker.info

            if not info:
                if cached_info:
                    cached_info.is_valid = False
                    db.session.commit()
                return None

            # Update or create cache entry
            symbol_info = cached_info or SymbolInfo(id=str(uuid.uuid4()), symbol=symbol)

            symbol_info.name = info.get("longName", info.get("shortName", "Unknown"))
            symbol_info.exchange = info.get("exchange")
            symbol_info.currency = info.get("currency")
            symbol_info.last_updated = datetime.now(UTC)
            symbol_info.data_source = "yfinance"
            symbol_info.is_valid = True

            db.session.add(symbol_info)
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.SYSTEM,
                message=f"Updated symbol info for {symbol}",
                details={
                    "symbol": symbol,
                    "name": symbol_info.name,
                    "source": "yfinance",
                },
            )

            return {
                "symbol": symbol_info.symbol,
                "name": symbol_info.name,
                "exchange": symbol_info.exchange,
                "currency": symbol_info.currency,
                "isin": symbol_info.isin,
                "last_updated": symbol_info.last_updated.replace(tzinfo=UTC).isoformat(),
            }

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=f"Error fetching symbol info: {e!s}",
                details={"symbol": symbol, "error": str(e)},
            )
            return None

    @staticmethod
    def get_symbol_by_isin(isin: str) -> dict:
        """
        Look up symbol information using ISIN.

        Args:
            isin (str): ISIN to look up

        Returns:
            dict: Symbol information (same structure as get_symbol_info)
            None: If ISIN not found
        """
        try:
            # Check cache first
            cached_info = SymbolInfo.query.filter_by(isin=isin, is_valid=True).first()
            if cached_info:
                return {
                    "symbol": cached_info.symbol,
                    "name": cached_info.name,
                    "exchange": cached_info.exchange,
                    "currency": cached_info.currency,
                    "isin": cached_info.isin,
                    "last_updated": cached_info.last_updated.replace(tzinfo=UTC).isoformat(),
                }

            # TODO: Implement ISIN lookup logic
            return None

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=f"Error looking up ISIN: {e!s}",
                details={"isin": isin, "error": str(e)},
            )
            return None

    @staticmethod
    def update_symbol_info(symbol: str, data: dict) -> dict:
        """
        Manually update symbol information.

        Args:
            symbol (str): Trading symbol to update
            data (dict): Updated symbol data containing any of:
                - name: Company/Fund name
                - exchange: Trading exchange
                - currency: Trading currency
                - isin: ISIN

        Returns:
            dict: Updated symbol information
            None: If update fails
        """
        try:
            symbol_info = SymbolInfo.query.filter_by(symbol=symbol).first()

            if not symbol_info:
                symbol_info = SymbolInfo(id=str(uuid.uuid4()), symbol=symbol)

            # Update fields if provided
            if "name" in data:
                symbol_info.name = data["name"]
            if "exchange" in data:
                symbol_info.exchange = data["exchange"]
            if "currency" in data:
                symbol_info.currency = data["currency"]
            if "isin" in data:
                symbol_info.isin = data["isin"]

            symbol_info.last_updated = datetime.now(UTC)
            symbol_info.data_source = "manual"
            symbol_info.is_valid = True

            db.session.add(symbol_info)
            db.session.commit()

            return {
                "symbol": symbol_info.symbol,
                "name": symbol_info.name,
                "exchange": symbol_info.exchange,
                "currency": symbol_info.currency,
                "isin": symbol_info.isin,
                "last_updated": symbol_info.last_updated.replace(tzinfo=UTC).isoformat(),
            }

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=f"Error updating symbol info: {e!s}",
                details={"symbol": symbol, "error": str(e)},
            )
            return None
