"""
Fund matching service for IBKR transactions.

Provides logic to match IBKR transactions to funds in the database using
ISIN (primary) and symbol normalization (fallback).
"""

from typing import ClassVar

from ..models import Fund, IBKRTransaction, Portfolio, PortfolioFund
from ..services.logging_service import LogCategory, LogLevel, logger


class FundMatchingService:
    """Service for matching IBKR transactions to funds and finding eligible portfolios."""

    # Exchange to yfinance suffix mapping
    EXCHANGE_SUFFIXES: ClassVar[dict[str, str]] = {
        "XETRA": ".DE",  # Germany - XETRA
        "IBIS": ".DE",  # Germany - XETRA (alternative name)
        "FWB": ".DE",  # Germany - Frankfurt
        "GER": ".DE",  # Germany - Generic
        "LSE": ".L",  # UK - London Stock Exchange
        "SIX": ".SW",  # Switzerland
        "EURONEXT": ".AS",  # Amsterdam (default for Euronext)
        "AEB": ".AS",  # Amsterdam - Euronext Amsterdam
        "XAMS": ".AS",  # Amsterdam Stock Exchange
        "NYSE": "",  # US - New York Stock Exchange
        "NASDAQ": "",  # US - NASDAQ
        "ARCA": "",  # US - NYSE Arca
        "BATS": "",  # US - BATS
        "FRA": ".F",  # Frankfurt
        "PAR": ".PA",  # Paris
        "MIL": ".MI",  # Milan
        "MCE": ".MC",  # Madrid
        "EBR": ".BR",  # Brussels
        "LIS": ".LS",  # Lisbon
        "AMS": ".AS",  # Amsterdam (alternative)
    }

    @classmethod
    def normalize_symbol(cls, symbol: str, exchange: str | None = None) -> str:
        """
        Normalize IBKR symbol to yfinance format.

        Args:
            symbol: Raw symbol from IBKR (e.g., "WEBN")
            exchange: Exchange code from IBKR (e.g., "XETRA", "GER")

        Returns:
            Normalized symbol with exchange suffix (e.g., "WEBN.DE")
        """
        if not symbol:
            return symbol

        # If no exchange info, return as-is
        if not exchange:
            return symbol

        # Get suffix for this exchange
        suffix = cls.EXCHANGE_SUFFIXES.get(exchange.upper(), "")

        # Return symbol with suffix
        return f"{symbol}{suffix}"

    @classmethod
    def find_fund_by_transaction(cls, transaction: IBKRTransaction) -> Fund | None:
        """
        Find fund matching an IBKR transaction.

        Matching priority:
        1. ISIN match (most reliable)
        2. Exact symbol match
        3. Normalized symbol match (using exchange info if available)

        Args:
            transaction: IBKRTransaction object

        Returns:
            Matched Fund object or None if no match found
        """
        # Try ISIN match first (most reliable)
        if transaction.isin:
            fund = Fund.query.filter_by(isin=transaction.isin).first()
            if fund:
                logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.IBKR,
                    message=f"Fund matched by ISIN: {transaction.isin}",
                    details={"fund_id": fund.id, "fund_name": fund.name},
                )
                return fund

        # Try exact symbol match
        if transaction.symbol:
            fund = Fund.query.filter_by(symbol=transaction.symbol).first()
            if fund:
                logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.IBKR,
                    message=f"Fund matched by exact symbol: {transaction.symbol}",
                    details={"fund_id": fund.id, "fund_name": fund.name},
                )
                return fund

            # Try normalized symbol match
            # Note: We don't have exchange in IBKRTransaction yet, so we'll try common suffixes
            # TODO: Add exchange field to IBKRTransaction for better matching
            common_suffixes = [".DE", ".AS", ".L", ".SW", ".F", ".PA", ".MI"]
            for suffix in common_suffixes:
                normalized = f"{transaction.symbol}{suffix}"
                fund = Fund.query.filter_by(symbol=normalized).first()
                if fund:
                    logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.IBKR,
                        message=f"Fund matched by normalized symbol: {normalized}",
                        details={"fund_id": fund.id, "fund_name": fund.name},
                    )
                    return fund

        logger.log(
            level=LogLevel.WARNING,
            category=LogCategory.IBKR,
            message="No fund match found for transaction",
            details={
                "transaction_id": transaction.id,
                "symbol": transaction.symbol,
                "isin": transaction.isin,
            },
        )
        return None

    @classmethod
    def get_portfolios_with_fund(cls, fund_id: str) -> list[Portfolio]:
        """
        Get all portfolios that have the specified fund assigned.

        Args:
            fund_id: Fund ID to search for

        Returns:
            List of Portfolio objects that contain this fund
        """
        # Query PortfolioFund to find all portfolios with this fund
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()

        # Get unique portfolios (that are not archived)
        portfolios = []
        seen_ids = set()

        for pf in portfolio_funds:
            if pf.portfolio_id not in seen_ids and not pf.portfolio.is_archived:
                portfolios.append(pf.portfolio)
                seen_ids.add(pf.portfolio_id)

        return portfolios

    @classmethod
    def get_eligible_portfolios_for_transaction(
        cls, transaction: IBKRTransaction
    ) -> dict[str, any]:
        """
        Get portfolios eligible for allocation of an IBKR transaction.

        Returns a dict with:
        - match_info: Information about how the fund was matched
        - portfolios: List of eligible portfolios
        - warning: Warning message if no match found

        Args:
            transaction: IBKRTransaction object

        Returns:
            Dict with match_info, portfolios, and optional warning
        """
        # Try to find matching fund
        fund = cls.find_fund_by_transaction(transaction)

        if not fund:
            # No matching fund found
            logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.IBKR,
                message="No fund found for transaction allocation",
                details={
                    "transaction_id": transaction.id,
                    "symbol": transaction.symbol,
                    "isin": transaction.isin,
                },
            )

            return {
                "match_info": {
                    "found": False,
                    "matched_by": None,
                    "fund_id": None,
                    "fund_name": None,
                    "fund_symbol": None,
                },
                "portfolios": [],
                "warning": (
                    f"No fund found matching this transaction (Symbol: {transaction.symbol}, "
                    f"ISIN: {transaction.isin or 'N/A'}). Please add the fund to the system first."
                ),
            }

        # Determine how we matched
        matched_by = "unknown"
        if transaction.isin and fund.isin == transaction.isin:
            matched_by = "isin"
        elif transaction.symbol and fund.symbol == transaction.symbol:
            matched_by = "exact_symbol"
        else:
            matched_by = "normalized_symbol"

        # Get portfolios with this fund
        portfolios = cls.get_portfolios_with_fund(fund.id)

        if not portfolios:
            # Fund exists but not in any portfolio
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="Fund found but not in any portfolio",
                details={
                    "transaction_id": transaction.id,
                    "fund_id": fund.id,
                    "fund_name": fund.name,
                },
            )

            return {
                "match_info": {
                    "found": True,
                    "matched_by": matched_by,
                    "fund_id": fund.id,
                    "fund_name": fund.name,
                    "fund_symbol": fund.symbol,
                },
                "portfolios": [],
                "warning": (
                    f"Fund '{fund.name}' ({fund.symbol}) exists but is not assigned to any "
                    "portfolio. Please add this fund to a portfolio first."
                ),
            }

        # Success - fund matched and portfolios found
        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Found {len(portfolios)} eligible portfolios for transaction",
            details={
                "transaction_id": transaction.id,
                "fund_id": fund.id,
                "fund_name": fund.name,
                "matched_by": matched_by,
                "portfolio_count": len(portfolios),
            },
        )

        return {
            "match_info": {
                "found": True,
                "matched_by": matched_by,
                "fund_id": fund.id,
                "fund_name": fund.name,
                "fund_symbol": fund.symbol,
            },
            "portfolios": [
                {
                    "id": p.id,
                    "name": p.name,
                    "description": p.description,
                }
                for p in portfolios
            ],
            "warning": None,
        }
