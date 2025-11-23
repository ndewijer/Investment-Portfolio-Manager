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
from werkzeug.exceptions import HTTPException

from ..models import LogCategory, LogLevel, db
from ..services.fund_service import FundService
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
        funds_data = FundService.get_all_funds_formatted()
        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message="Successfully retrieved all funds",
            details={"fund_count": len(funds_data)},
        )
        return jsonify(funds_data)
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
        symbol_info = None
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

        fund = FundService.create_fund(data, symbol_info=symbol_info)

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
        fund = FundService.get_fund(fund_id)

        # Get last known price from database
        latest_price = FundService.get_latest_fund_price(fund_id)

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

    except HTTPException:
        # Re-raise HTTP exceptions (like 404 from service layer)
        raise
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
        fund, symbol_changed = FundService.update_fund(fund_id, data)

        # If symbol changed, try to get symbol info
        if symbol_changed and fund.symbol:
            try:
                symbol_info = SymbolLookupService.get_symbol_info(fund.symbol, force_refresh=True)
                if symbol_info:
                    logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.FUND,
                        message=f"Successfully retrieved symbol info for {fund.symbol}",
                        details=symbol_info,
                    )
            except Exception as e:
                logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"Failed to retrieve symbol info: {e!s}",
                    details={"symbol": fund.symbol},
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
                "investment_type": fund.investment_type.value,
            }
        )
    except ValueError as e:
        return jsonify({"error": str(e)}), 404
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
        usage_info = FundService.check_fund_usage(fund_id)

        if usage_info["in_use"]:
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Fund {fund_id} is in use",
                details=usage_info,
            )

        return jsonify(usage_info)
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
        fund_details = FundService.delete_fund(fund_id)

        response, status = logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully deleted fund {fund_details['fund_name']}",
            details=fund_details,
            http_status=200,
        )
        return jsonify(response), status

    except ValueError as e:
        # Fund not found or fund is in use
        error_message = str(e)

        if "Cannot delete" in error_message and "attached to" in error_message:
            # Fund is in use
            response, status = logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.FUND,
                message="Cannot delete fund while attached to portfolios",
                details={"fund_id": fund_id, "user_message": error_message},
                http_status=409,
            )
            return jsonify(response), status
        else:
            # Fund not found
            return jsonify({"error": error_message}), 404

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
        fund = FundService.get_fund(fund_id)

        # Get all prices for this fund, ordered by date
        prices = FundService.get_fund_price_history(fund_id)

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

    except HTTPException:
        # Re-raise HTTP exceptions (like 404 from service layer)
        raise
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
        result = FundService.update_all_fund_prices()
        return jsonify(result)
    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500
