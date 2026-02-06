"""
Service class for fund management operations.

This module provides methods for:
- Creating and managing funds
- Retrieving fund information
- Fund CRUD operations
- Checking fund usage in portfolios
"""

from datetime import datetime

from flask import abort

from ..models import (
    DividendType,
    Fund,
    FundHistoryMaterialized,
    FundPrice,
    InvestmentType,
    LogCategory,
    LogLevel,
    Portfolio,
    PortfolioFund,
    Transaction,
    db,
)
from ..services.logging_service import logger


class FundService:
    """
    Service class for fund management operations.

    Provides methods for:
    - Creating and managing funds
    - Retrieving fund information
    - Fund CRUD operations
    """

    @staticmethod
    def format_fund(fund):
        """
        Format a fund object for API response.

        Args:
            fund: Fund object

        Returns:
            dict: Formatted fund data
        """
        return {
            "id": fund.id,
            "name": fund.name,
            "isin": fund.isin,
            "symbol": fund.symbol,
            "currency": fund.currency,
            "exchange": fund.exchange,
            "dividendType": fund.dividend_type.value,
            "investmentType": fund.investment_type.value,
        }

    @staticmethod
    def get_all_funds():
        """
        Retrieve all funds from the database.

        Returns:
            list: List of Fund objects
        """
        return Fund.query.all()

    @staticmethod
    def get_all_funds_formatted():
        """
        Retrieve all funds with formatting for API response.

        Returns:
            list: List of formatted fund dictionaries
        """
        funds = Fund.query.all()
        return [FundService.format_fund(f) for f in funds]

    @staticmethod
    def get_fund(fund_id):
        """
        Retrieve a specific fund by ID.

        Args:
            fund_id (str): Fund identifier

        Returns:
            Fund: Fund object

        Raises:
            404: If fund not found
        """
        fund = db.session.get(Fund, fund_id)
        if not fund:
            abort(404)
        return fund

    @staticmethod
    def get_latest_fund_price(fund_id):
        """
        Retrieve the latest price for a specific fund.

        Args:
            fund_id (str): Fund identifier

        Returns:
            float: Latest price, or None if no price exists
        """
        price_record = (
            db.session.query(FundPrice)
            .filter_by(fund_id=fund_id)
            .order_by(FundPrice.date.desc())
            .first()
        )
        return price_record.price if price_record else None

    @staticmethod
    def get_fund_price_history(fund_id):
        """
        Retrieve all historical prices for a specific fund.

        Args:
            fund_id (str): Fund identifier

        Returns:
            list[FundPrice]: List of fund price records, ordered by date (newest first)
        """
        return (
            db.session.query(FundPrice)
            .filter_by(fund_id=fund_id)
            .order_by(FundPrice.date.desc())
            .all()
        )

    @staticmethod
    def create_fund(data, symbol_info=None):
        """
        Create a new fund with optional symbol lookup integration.

        Args:
            data (dict): Fund data containing:
                - name: Fund name
                - isin: International Securities Identification Number
                - currency: Trading currency code
                - exchange: Trading exchange
                - symbol (optional): Trading symbol
                - investment_type (optional): 'stock' or 'fund'
            symbol_info (dict, optional): Symbol information from lookup service

        Returns:
            Fund: Created fund object

        Raises:
            IntegrityError: If ISIN is not unique
        """
        # Get investment_type from request, default to 'FUND' if not provided
        investment_type_str = data.get("investmentType", "FUND")
        investment_type = (
            InvestmentType.STOCK if investment_type_str == "STOCK" else InvestmentType.FUND
        )

        fund = Fund(
            name=data["name"],
            isin=data["isin"],
            symbol=data.get("symbol"),
            currency=data["currency"],
            exchange=data["exchange"],
            investment_type=investment_type,
            dividend_type=DividendType.NONE,
        )
        db.session.add(fund)
        db.session.commit()
        return fund

    @staticmethod
    def update_fund(fund_id, data):
        """
        Update an existing fund with symbol and type support.

        Args:
            fund_id (str): Fund identifier
            data (dict): Updated fund data containing:
                - name: Fund name
                - isin: ISIN
                - symbol (optional): Trading symbol
                - currency: Currency code
                - exchange: Exchange name
                - dividend_type (optional): Dividend type
                - investment_type (optional): Investment type

        Returns:
            tuple: (fund, symbol_changed) where symbol_changed indicates if symbol was modified

        Raises:
            ValueError: If fund not found
        """
        fund = db.session.get(Fund, fund_id)
        if not fund:
            raise ValueError(f"Fund {fund_id} not found")

        fund.name = data["name"]
        fund.isin = data["isin"]

        # Track symbol changes for caller to handle symbol lookup
        symbol_changed = False
        if data.get("symbol"):
            old_symbol = fund.symbol
            new_symbol = data["symbol"]
            if old_symbol != new_symbol:
                fund.symbol = new_symbol
                symbol_changed = True
        else:
            fund.symbol = None  # Clear symbol if not provided

        fund.currency = data["currency"]
        fund.exchange = data["exchange"]

        if "dividendType" in data:
            fund.dividend_type = DividendType(data["dividendType"])

        if "investmentType" in data:
            investment_type_str = data["investmentType"]
            fund.investment_type = (
                InvestmentType.STOCK if investment_type_str == "STOCK" else InvestmentType.FUND
            )

        db.session.add(fund)
        db.session.commit()
        return fund, symbol_changed

    @staticmethod
    def check_fund_usage(fund_id):
        """
        Check if a fund is being used in any portfolios.

        Args:
            fund_id (str): Fund identifier

        Returns:
            dict: Usage information containing:
                - in_use (bool): Whether fund is in use
                - portfolios (list, optional): List of portfolios using the fund with
                  transaction counts
        """
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
        if not portfolio_funds:
            return {"in_use": False}

        # Get portfolios and their transaction counts
        portfolio_data = []
        for pf in portfolio_funds:
            transaction_count = Transaction.query.filter_by(portfolio_fund_id=pf.id).count()
            if transaction_count > 0:
                portfolio_data.append(
                    {
                        "id": pf.portfolio.id,
                        "name": pf.portfolio.name,
                        "transaction_count": transaction_count,
                    }
                )

        if portfolio_data:
            return {"in_use": True, "portfolios": portfolio_data}

        return {"in_use": False}

    @staticmethod
    def delete_fund(fund_id):
        """
        Delete a fund if it's not being used in any portfolios.

        Args:
            fund_id (str): Fund identifier

        Returns:
            dict: Deletion result with fund details

        Raises:
            ValueError: If fund not found or fund is in use
        """
        fund = db.session.get(Fund, fund_id)
        if not fund:
            raise ValueError(f"Fund {fund_id} not found")

        # Check for any portfolio-fund relationships
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
        if portfolio_funds:
            # Get list of portfolios this fund is attached to
            portfolio_info = [
                {"name": pf.portfolio.name, "id": pf.portfolio.id} for pf in portfolio_funds
            ]

            portfolio_names = ", ".join(pf["name"] for pf in portfolio_info)
            raise ValueError(
                f"Cannot delete {fund.name} because it is still attached to the "
                f"following portfolios: {portfolio_names}. Please remove the fund from "
                f"these portfolios first."
            )

        # Store fund details before deletion
        fund_details = {"fund_id": fund_id, "fund_name": fund.name}

        # Delete any fund prices
        FundPrice.query.filter_by(fund_id=fund_id).delete()

        # Delete the fund
        db.session.delete(fund)
        db.session.commit()

        return fund_details

    @staticmethod
    def update_all_fund_prices():
        """
        Update prices for all funds with symbols.

        Returns:
            dict: Results with updated_funds and errors lists
        """
        from ..services.price_update_service import HistoricalPriceService

        # Get all funds with symbols
        funds_with_symbols = Fund.query.filter(Fund.symbol.isnot(None), Fund.symbol != "").all()

        updated_funds = []
        errors = []

        for fund in funds_with_symbols:
            try:
                result, status = HistoricalPriceService.update_historical_prices(fund.id)

                if status == 200:
                    updated_funds.append(
                        {
                            "fund_id": fund.id,
                            "name": fund.name,
                            "symbol": fund.symbol,
                            "prices_added": result.get("prices_added", 0),
                        }
                    )
                else:
                    errors.append(
                        {
                            "fund_id": fund.id,
                            "name": fund.name,
                            "symbol": fund.symbol,
                            "error": result.get("message", "Unknown error"),
                        }
                    )
            except Exception as e:
                errors.append(
                    {
                        "fund_id": fund.id,
                        "name": fund.name,
                        "symbol": fund.symbol,
                        "error": str(e),
                    }
                )

        return {
            "success": True,
            "updated_funds": updated_funds,
            "errors": errors,
            "total_updated": len(updated_funds),
            "total_errors": len(errors),
        }

    @staticmethod
    def get_fund_history(portfolio_id, start_date=None, end_date=None):
        """
        Get historical fund data for a specific portfolio.

        This method retrieves pre-calculated fund metrics from the
        fund_history_materialized table. Data is grouped by date with
        all funds for each date.

        If no materialized data is found, this method automatically
        attempts to materialize the portfolio history on-demand. This
        handles cases where the view wasn't populated after upgrade or
        was invalidated and not yet recalculated.

        Args:
            portfolio_id (str): Portfolio identifier
            start_date (str, optional): Start date in YYYY-MM-DD format
            end_date (str, optional): End date in YYYY-MM-DD format

        Returns:
            list: List of daily fund values in the format:
                [
                    {
                        "date": "2021-09-06",
                        "funds": [
                            {
                                "portfolioFundId": "...",
                                "fundId": "...",
                                "fundName": "...",
                                "shares": 15.78,
                                "price": 31.67,
                                "value": 500.00,
                                "cost": 500.00,
                                "realizedGain": 0,
                                "unrealizedGain": 0,
                                "totalGainLoss": 0,
                                "dividends": 0,
                                "fees": 0
                            }
                        ]
                    }
                ]

        Raises:
            ValueError: If portfolio not found
        """
        # Verify portfolio exists
        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            raise ValueError("Portfolio not found")

        # Parse dates
        if start_date:
            try:
                start_date = datetime.strptime(start_date, "%Y-%m-%d").date()
            except ValueError:
                start_date = None

        if end_date:
            try:
                end_date = datetime.strptime(end_date, "%Y-%m-%d").date()
            except ValueError:
                end_date = None

        # Build query
        query = (
            db.session.query(FundHistoryMaterialized, Fund.name.label("fund_name"))
            .join(PortfolioFund, FundHistoryMaterialized.portfolio_fund_id == PortfolioFund.id)
            .join(Fund, FundHistoryMaterialized.fund_id == Fund.id)
            .filter(PortfolioFund.portfolio_id == portfolio_id)
        )

        # Apply date filters
        if start_date:
            query = query.filter(FundHistoryMaterialized.date >= start_date.strftime("%Y-%m-%d"))
        if end_date:
            query = query.filter(FundHistoryMaterialized.date <= end_date.strftime("%Y-%m-%d"))

        # Order by date and fund name
        query = query.order_by(FundHistoryMaterialized.date.asc(), Fund.name.asc())

        # Execute query
        results = query.all()

        # Check if we need to materialize/rematerialize
        # This handles cases where:
        # 1. Materialized view wasn't populated after upgrade (no results)
        # 2. Materialized view is stale (transactions newer than latest materialized date)
        # 3. Recent transactions invalidated old data but recalculation hasn't run yet
        needs_materialization = False
        materialization_reason = None

        if not results:
            needs_materialization = True
            materialization_reason = "no_data"
        else:
            # Check if materialized data is stale
            latest_mat_date_str = max(r[0].date for r in results)
            latest_mat_date = datetime.strptime(latest_mat_date_str, "%Y-%m-%d").date()

            # Check if there are transactions newer than the latest materialized date
            portfolio_funds = PortfolioFund.query.filter_by(portfolio_id=portfolio_id).all()
            if portfolio_funds:
                pf_ids = [pf.id for pf in portfolio_funds]
                from sqlalchemy import func

                latest_txn = (
                    db.session.query(func.max(Transaction.date))
                    .filter(Transaction.portfolio_fund_id.in_(pf_ids))
                    .scalar()
                )

                if latest_txn and latest_txn > latest_mat_date:
                    needs_materialization = True
                    materialization_reason = "stale_data"
                    logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.SYSTEM,
                        message=f"Materialized view is stale for portfolio {portfolio_id}",
                        details={
                            "portfolio_id": portfolio_id,
                            "latest_transaction": latest_txn.isoformat(),
                            "latest_materialized": latest_mat_date.isoformat(),
                            "days_behind": (latest_txn - latest_mat_date).days,
                        },
                    )

        if needs_materialization:
            from ..services.portfolio_history_materialized_service import (
                PortfolioHistoryMaterializedService,
            )

            portfolio_funds = PortfolioFund.query.filter_by(portfolio_id=portfolio_id).all()
            if portfolio_funds:
                try:
                    count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
                        portfolio_id, force_recalculate=False
                    )

                    logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.SYSTEM,
                        message=f"Auto-materialized portfolio history ({materialization_reason})",
                        details={
                            "portfolio_id": portfolio_id,
                            "reason": materialization_reason,
                            "records_created": count,
                        },
                    )

                    if count > 0:
                        # Re-run the query after materialization
                        results = query.all()
                except Exception as e:
                    logger.log(
                        level=LogLevel.ERROR,
                        category=LogCategory.SYSTEM,
                        message="Failed to auto-materialize portfolio history",
                        details={"portfolio_id": portfolio_id, "error": str(e)},
                    )

        # Group results by date
        history_by_date = {}
        for entry, fund_name in results:
            date_str = entry.date
            if date_str not in history_by_date:
                history_by_date[date_str] = []

            history_by_date[date_str].append(
                {
                    "portfolioFundId": entry.portfolio_fund_id,
                    "fundId": entry.fund_id,
                    "fundName": fund_name,
                    "shares": entry.shares,
                    "price": entry.price,
                    "value": entry.value,
                    "cost": entry.cost,
                    "realizedGain": entry.realized_gain,
                    "unrealizedGain": entry.unrealized_gain,
                    "totalGainLoss": entry.total_gain_loss,
                    "dividends": entry.dividends,
                    "fees": entry.fees,
                }
            )

        # Convert to list format
        history = [
            {"date": date, "funds": funds} for date, funds in sorted(history_by_date.items())
        ]

        return history
