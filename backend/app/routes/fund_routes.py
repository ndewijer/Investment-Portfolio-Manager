"""
API routes for fund management.

This module provides routes for:
- Creating and managing funds
- Retrieving fund information and prices
- Updating fund prices from external sources
- Symbol information lookup
"""

from flask import Blueprint, jsonify, request
from sqlalchemy.exc import IntegrityError

from ..models import (
    DividendType,
    Fund,
    FundPrice,
    InvestmentType,
    LogCategory,
    LogLevel,
    PortfolioFund,
    Transaction,
    db,
)
from ..services.logging_service import logger, track_request
from ..services.price_update_service import HistoricalPriceService, TodayPriceService
from ..services.symbol_lookup_service import SymbolLookupService
from ..utils.security import require_api_key

funds = Blueprint("funds", __name__)


@funds.route("/funds", methods=["GET"])
@track_request
def get_funds():
    """
    Retrieve all funds from the database.

    Returns:
        JSON response containing list of funds with their details
    """
    try:
        funds = Fund.query.all()
        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message="Successfully retrieved all funds",
            details={"fund_count": len(funds)},
        )
        return jsonify(
            [
                {
                    "id": f.id,
                    "name": f.name,
                    "isin": f.isin,
                    "symbol": f.symbol,
                    "currency": f.currency,
                    "exchange": f.exchange,
                    "dividend_type": f.dividend_type.value,
                    "investment_type": f.investment_type.value,
                }
                for f in funds
            ]
        )
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error retrieving funds: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@funds.route("/funds", methods=["POST"])
@track_request
def create_fund():
    """
    Create a new fund.

    Request Body:
        name (str): Fund name
        isin (str): International Securities Identification Number
        symbol (str, optional): Trading symbol
        currency (str): Trading currency code
        exchange (str): Exchange where fund is traded

    Returns:
        JSON response containing created fund details
    """
    try:
        data = request.json

        # If symbol is provided, try to get symbol info before creating fund
        if data.get("symbol"):
            try:
                symbol_info = SymbolLookupService.get_symbol_info(
                    data["symbol"], force_refresh=True
                )
                if symbol_info:
                    logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.FUND,
                        message=f"Successfully retrieved symbol info for {data['symbol']}",
                        details=symbol_info,
                    )
            except Exception as e:
                logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"Failed to retrieve symbol info: {e!s}",
                    details={"symbol": data["symbol"]},
                )

        # Get investment_type from request, default to 'fund' if not provided
        investment_type_str = data.get("investment_type", "fund")
        investment_type = (
            InvestmentType.STOCK if investment_type_str == "stock" else InvestmentType.FUND
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

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully created fund {fund.name}",
            details={"fund_id": fund.id, "isin": fund.isin, "symbol": fund.symbol},
        )

        return jsonify(
            {
                "id": fund.id,
                "name": fund.name,
                "isin": fund.isin,
                "symbol": fund.symbol,
                "currency": fund.currency,
                "exchange": fund.exchange,
                "dividend_type": fund.dividend_type.value,
            }
        )
    except IntegrityError as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message="ISIN must be unique",
            details={
                "user_message": "A fund with this ISIN already exists",
                "error": str(e),
            },
            http_status=400,
        )
        return jsonify(response), status
    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error creating fund: {e!s}",
            details={"user_message": "Error creating fund", "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@funds.route("/funds/<string:fund_id>", methods=["GET"])
@track_request
def get_fund(fund_id):
    """
    Retrieve details for a specific fund.

    Args:
        fund_id (str): Fund identifier

    Query Parameters:
        include_prices (bool, optional): Include price history
        include_dividends (bool, optional): Include dividend history

    Returns:
        JSON response containing:
        - Fund details
        - Latest price
        - Price history (if requested)
        - Dividend history (if requested)
    """
    try:
        fund = Fund.query.get_or_404(fund_id)

        # Get last known price from database
        latest_price = None
        price_record = (
            FundPrice.query.filter_by(fund_id=fund_id).order_by(FundPrice.date.desc()).first()
        )
        if price_record:
            latest_price = price_record.price

        response = {
            "id": fund.id,
            "name": fund.name,
            "symbol": fund.symbol,
            "isin": fund.isin,
            "currency": fund.currency,
            "exchange": fund.exchange,
            "investment_type": fund.investment_type.value,
            "dividend_type": fund.dividend_type.value,
            "latest_price": latest_price,
        }

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully retrieved fund {fund.name}",
            details={"fund_id": fund_id, "has_latest_price": latest_price is not None},
        )

        return jsonify(response)

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error retrieving fund: {e!s}",
            details={
                "fund_id": fund_id,
                "error": str(e),
                "user_message": "Unable to retrieve fund details. Please try again later.",
            },
            http_status=500,
        )
        return jsonify(response), status


@funds.route("/funds/<string:fund_id>", methods=["PUT"])
@track_request
def update_fund(fund_id):
    """
    Update an existing fund's details.

    Args:
        fund_id (str): Fund identifier

    Request Body:
        name (str): Fund name
        isin (str): International Securities Identification Number
        symbol (str, optional): Trading symbol
        currency (str): Trading currency code
        exchange (str): Exchange where fund is traded
        dividend_type (str, optional): Type of dividend

    Returns:
        JSON response containing updated fund details
    """
    try:
        data = request.json
        fund = Fund.query.get_or_404(fund_id)
        fund.name = data["name"]
        fund.isin = data["isin"]

        # Handle symbol update
        if data.get("symbol"):
            old_symbol = fund.symbol
            new_symbol = data["symbol"]

            # Only lookup if symbol has changed
            if old_symbol != new_symbol:
                fund.symbol = new_symbol
                # Try to get symbol info and store it
                try:
                    symbol_info = SymbolLookupService.get_symbol_info(
                        new_symbol, force_refresh=True
                    )
                    if symbol_info:
                        logger.log(
                            level=LogLevel.INFO,
                            category=LogCategory.FUND,
                            message=f"Successfully retrieved symbol info for {new_symbol}",
                            details=symbol_info,
                        )
                except Exception as e:
                    logger.log(
                        level=LogLevel.WARNING,
                        category=LogCategory.FUND,
                        message=f"Failed to retrieve symbol info: {e!s}",
                        details={"symbol": new_symbol},
                    )
        else:
            fund.symbol = None  # Clear symbol if not provided

        fund.currency = data["currency"]
        fund.exchange = data["exchange"]
        if "dividend_type" in data:
            fund.dividend_type = DividendType(data["dividend_type"])
        if "investment_type" in data:
            investment_type_str = data["investment_type"]
            fund.investment_type = (
                InvestmentType.STOCK if investment_type_str == "stock" else InvestmentType.FUND
            )

        db.session.add(fund)
        db.session.commit()

        return jsonify(
            {
                "id": fund.id,
                "name": fund.name,
                "isin": fund.isin,
                "symbol": fund.symbol,
                "currency": fund.currency,
                "exchange": fund.exchange,
                "dividend_type": fund.dividend_type.value,
                "investment_type": fund.investment_type.value,
            }
        )
    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error updating fund: {e!s}",
            details={"fund_id": fund_id, "error": str(e), "request_data": data},
            http_status=400,
        )
        return jsonify(response), status


@funds.route("/funds/<string:fund_id>/check-usage", methods=["GET"])
@track_request
def check_fund_usage(fund_id):
    """
    Check if a fund is being used in any portfolios.

    Args:
        fund_id (str): Fund identifier

    Returns:
        JSON response containing:
        - Usage status
        - List of portfolios using the fund
        - Transaction count
        - Dividend count
    """
    try:
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
        if portfolio_funds:
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
                logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.FUND,
                    message=f"Fund {fund_id} is in use",
                    details={"in_use": True, "portfolios": portfolio_data},
                )
                return jsonify({"in_use": True, "portfolios": portfolio_data})
        return jsonify({"in_use": False})
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error checking fund usage: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@funds.route("/funds/<string:fund_id>", methods=["DELETE"])
@track_request
def delete_fund(fund_id):
    """
    Delete a fund if it's not being used in any portfolios.

    Args:
        fund_id (str): Fund identifier

    Returns:
        JSON response confirming deletion or error if fund is in use
    """
    try:
        # First check if the fund exists
        fund = Fund.query.get_or_404(fund_id)

        # Check for any portfolio-fund relationships
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
        if portfolio_funds:
            # Get list of portfolios this fund is attached to
            portfolio_info = [
                {"name": pf.portfolio.name, "id": pf.portfolio.id} for pf in portfolio_funds
            ]

            response, status = logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.FUND,
                message="Cannot delete fund while attached to portfolios",
                details={
                    "fund_id": fund_id,
                    "fund_name": fund.name,
                    "portfolios": portfolio_info,
                    "user_message": (
                        "Cannot delete {} because it is still attached to the "
                        "following portfolios: {}. Please remove the fund from "
                        "these portfolios first."
                    ).format(fund.name, ", ".join(pf["name"] for pf in portfolio_info)),
                },
                http_status=409,
            )
            return jsonify(response), status

        # If no portfolio relationships exist, proceed with deletion
        # Delete any fund prices
        FundPrice.query.filter_by(fund_id=fund_id).delete()

        # Delete the fund
        db.session.delete(fund)
        db.session.commit()

        response, status = logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully deleted fund {fund.name}",
            details={"fund_id": fund_id},
            http_status=200,
        )
        return jsonify(response), status

    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error deleting fund: {e!s}",
            details={"fund_id": fund_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@funds.route("/lookup-symbol-info/<string:symbol>", methods=["GET"])
@track_request
def lookup_symbol_info(symbol):
    """
    Look up information for a trading symbol.

    Args:
        symbol (str): Trading symbol to look up

    Query Parameters:
        force_refresh (bool, optional): Force refresh from external source

    Returns:
        JSON response containing symbol information
    """
    try:
        # Use SymbolLookupService to get info (checks cache first)
        force_refresh = request.args.get("force_refresh", "false").lower() == "true"
        symbol_info = SymbolLookupService.get_symbol_info(symbol, force_refresh=force_refresh)

        if symbol_info:
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Successfully retrieved symbol info for {symbol}",
                details={
                    "symbol": symbol,
                    "source": "cache" if not force_refresh else "yfinance",
                    "info": symbol_info,
                },
            )
            return jsonify(symbol_info)
        else:
            response, status = logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.FUND,
                message=f"No information found for symbol {symbol}",
                details={"symbol": symbol},
                http_status=404,
            )
            return jsonify(response), status

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error looking up symbol: {e!s}",
            details={"symbol": symbol, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


# Add this new route to get fund prices
@funds.route("/fund-prices/<string:fund_id>", methods=["GET"])
@track_request
def get_fund_prices(fund_id):
    """
    Retrieve price history for a fund.

    Args:
        fund_id (str): Fund identifier

    Returns:
        JSON response containing list of historical prices
    """
    try:
        # Get the fund to ensure it exists
        fund = Fund.query.get_or_404(fund_id)

        # Get all prices for this fund, ordered by date
        prices = FundPrice.query.filter_by(fund_id=fund_id).order_by(FundPrice.date.desc()).all()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully retrieved price history for fund {fund.name}",
            details={"fund_id": fund_id, "price_count": len(prices)},
        )

        return jsonify(
            [
                {"id": price.id, "date": price.date.isoformat(), "price": price.price}
                for price in prices
            ]
        )

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error retrieving fund prices: {e!s}",
            details={"fund_id": fund_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@funds.route("/fund-prices/<string:fund_id>/update", methods=["POST"])
@track_request
def update_fund_prices(fund_id):
    """
    Update fund prices from external source.

    Args:
        fund_id (str): Fund identifier

    Query Parameters:
        type (str): Update type ('today' or 'historical')

    Returns:
        JSON response containing update results
    """
    try:
        update_type = request.args.get("type", "today")  # 'today' or 'historical'

        if update_type == "today":
            response, status = TodayPriceService.update_todays_price(fund_id)
        else:
            response, status = HistoricalPriceService.update_historical_prices(fund_id)

        return jsonify(response), status

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error updating fund prices: {e!s}",
            details={"fund_id": fund_id, "update_type": update_type, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@funds.route("/funds/update-all-prices", methods=["POST"])
@require_api_key
def update_all_fund_prices():
    """
    Update prices for all funds with symbols.

    This endpoint is meant to be called once daily.
    Protected by API key and time-based token.

    Returns:
        JSON response containing update results
    """
    try:
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

        return jsonify(
            {
                "success": True,
                "updated_funds": updated_funds,
                "errors": errors,
                "total_updated": len(updated_funds),
                "total_errors": len(errors),
            }
        )

    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500
