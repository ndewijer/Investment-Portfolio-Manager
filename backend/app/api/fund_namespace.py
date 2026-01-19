"""
Fund API namespace for managing investment funds and stocks.

This namespace provides endpoints for:
- Creating, reading, updating funds
- Managing fund prices
- Symbol information lookup
- Price updates from external sources
"""

from flask import request
from flask_restx import Namespace, Resource, fields
from sqlalchemy.exc import IntegrityError
from werkzeug.exceptions import HTTPException

from ..models import LogCategory, LogLevel, db
from ..services.fund_service import FundService
from ..services.logging_service import logger
from ..services.price_update_service import HistoricalPriceService, TodayPriceService
from ..services.symbol_lookup_service import SymbolLookupService
from ..utils.security import require_api_key_restx

# Create namespace
ns = Namespace("fund", description="Fund and stock management operations")

# Define models for documentation
fund_model = ns.model(
    "Fund",
    {
        "id": fields.String(required=True, description="Fund unique identifier (UUID)"),
        "name": fields.String(required=True, description="Fund name"),
        "isin": fields.String(
            required=True, description="International Securities Identification Number"
        ),
        "symbol": fields.String(description="Trading symbol"),
        "currency": fields.String(
            required=True, description="Trading currency code (e.g., USD, EUR)"
        ),
        "exchange": fields.String(required=True, description="Exchange where fund is traded"),
        "dividendType": fields.String(
            required=True, description="Dividend type", enum=["NONE", "CASH", "STOCK"]
        ),
        "investmentType": fields.String(
            required=True, description="Investment type", enum=["FUND", "STOCK"]
        ),
    },
)

fund_detail_model = ns.inherit(
    "FundDetail", fund_model, {"latest_price": fields.Raw(description="Latest price information")}
)

fund_create_model = ns.model(
    "FundCreate",
    {
        "name": fields.String(
            required=True, description="Fund name", example="Vanguard S&P 500 ETF"
        ),
        "isin": fields.String(required=True, description="ISIN", example="US9229083632"),
        "symbol": fields.String(description="Trading symbol", example="VOO"),
        "currency": fields.String(required=True, description="Currency code", example="USD"),
        "exchange": fields.String(required=True, description="Exchange", example="NYSE"),
        "dividendType": fields.String(
            description="Dividend type", enum=["NONE", "CASH", "STOCK"], default="NONE"
        ),
        "investmentType": fields.String(
            description="Investment type", enum=["FUND", "STOCK"], default="FUND"
        ),
    },
)

fund_update_model = ns.model(
    "FundUpdate",
    {
        "name": fields.String(description="Fund name"),
        "isin": fields.String(description="ISIN"),
        "symbol": fields.String(description="Trading symbol"),
        "currency": fields.String(description="Currency code"),
        "exchange": fields.String(description="Exchange"),
        "dividendType": fields.String(description="Dividend type", enum=["NONE", "CASH", "STOCK"]),
        "investmentType": fields.String(description="Investment type", enum=["FUND", "STOCK"]),
    },
)

price_update_model = ns.model(
    "PriceUpdate",
    {
        "success": fields.Boolean(required=True, description="Update success status"),
        "price": fields.Float(description="Updated price"),
        "date": fields.String(description="Price date"),
    },
)

symbol_info_model = ns.model(
    "SymbolInfo",
    {
        "symbol": fields.String(required=True, description="Trading symbol"),
        "name": fields.String(description="Security name"),
        "exchange": fields.String(description="Exchange"),
        "currency": fields.String(description="Currency"),
        "price": fields.Float(description="Current price"),
    },
)

error_model = ns.model(
    "Error",
    {
        "error": fields.String(required=True, description="Error message"),
        "details": fields.String(description="Additional error details"),
    },
)


@ns.route("")
class FundList(Resource):
    """Fund collection endpoint."""

    @ns.doc("list_funds")
    @ns.response(200, "Success", [fund_model])
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get all funds.

        Returns a list of all funds and stocks in the system.
        Each fund includes basic information such as name, ISIN, symbol, and dividend type.
        """
        try:
            funds = FundService.get_all_funds()
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message="Successfully retrieved all funds",
                details={"fund_count": len(funds)},
            )
            return [
                {
                    "id": f.id,
                    "name": f.name,
                    "isin": f.isin,
                    "symbol": f.symbol,
                    "currency": f.currency,
                    "exchange": f.exchange,
                    "dividendType": f.dividend_type.value,
                    "investmentType": f.investment_type.value,
                }
                for f in funds
            ], 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error retrieving funds: {e!s}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc("create_fund")
    @ns.expect(fund_create_model, validate=True)
    @ns.response(201, "Fund created", fund_model)
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Create a new fund.

        Creates a new fund or stock with the provided information.
        If a symbol is provided, the system will attempt to fetch additional information
        from external sources to validate and enrich the fund data.

        Returns the created fund with all fields populated.
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
                except HTTPException:
                    raise
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

            return {
                "id": fund.id,
                "name": fund.name,
                "isin": fund.isin,
                "symbol": fund.symbol,
                "currency": fund.currency,
                "exchange": fund.exchange,
                "dividendType": fund.dividend_type.value,
                "investmentType": fund.investment_type.value,
            }, 201

        except IntegrityError as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message="ISIN must be unique",
                details={
                    "user_message": "A fund with this ISIN already exists",
                    "error": str(e),
                },
            )
            return {"error": "A fund with this ISIN already exists", "details": str(e)}, 400
        except HTTPException:
            raise
        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error creating fund: {e!s}",
                details={"user_message": "Error creating fund", "error": str(e)},
            )
            return {"error": "Error creating fund", "details": str(e)}, 500


@ns.route("/<string:fund_id>")
@ns.param("fund_id", "Fund unique identifier (UUID)")
class FundDetail(Resource):
    """Fund detail endpoint."""

    @ns.doc("get_fund")
    @ns.response(200, "Success", fund_detail_model)
    @ns.response(404, "Fund not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, fund_id):
        """
        Get fund details.

        Returns detailed information about a specific fund, including:
        - Basic information (name, ISIN, symbol)
        - Trading information (currency, exchange)
        - Latest price data

        Query Parameters:
        - include_prices (bool): Include price history
        - include_dividends (bool): Include dividend history
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
                "investmentType": fund.investment_type.value,
                "dividendType": fund.dividend_type.value,
                "latest_price": latest_price,
            }

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Successfully retrieved fund {fund.name}",
                details={"fund_id": fund_id, "has_latest_price": latest_price is not None},
            )

            return response, 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error retrieving fund: {e!s}",
                details={
                    "fund_id": fund_id,
                    "error": str(e),
                    "user_message": "Unable to retrieve fund details. Please try again later.",
                },
            )
            return {"error": "Unable to retrieve fund details", "details": str(e)}, 500

    @ns.doc("update_fund")
    @ns.expect(fund_update_model, validate=True)
    @ns.response(200, "Fund updated", fund_model)
    @ns.response(404, "Fund not found", error_model)
    @ns.response(500, "Server error", error_model)
    def put(self, fund_id):
        """
        Update fund information.

        Updates one or more fields of an existing fund.
        Only provided fields will be updated; omitted fields remain unchanged.

        If the symbol is changed, the system will attempt to fetch and validate
        the new symbol information from external sources.
        """
        try:
            data = request.json
            fund, symbol_changed = FundService.update_fund(fund_id, data)

            # If symbol changed, try to get symbol info
            if symbol_changed and fund.symbol:
                try:
                    symbol_info = SymbolLookupService.get_symbol_info(
                        fund.symbol, force_refresh=True
                    )
                    if symbol_info:
                        logger.log(
                            level=LogLevel.INFO,
                            category=LogCategory.FUND,
                            message=f"Successfully retrieved symbol info for updated fund",
                            details={"fund_id": fund_id, "symbol": fund.symbol},
                        )
                except HTTPException:
                    raise
                except Exception as e:
                    logger.log(
                        level=LogLevel.WARNING,
                        category=LogCategory.FUND,
                        message=f"Failed to retrieve symbol info for updated symbol: {e!s}",
                        details={"fund_id": fund_id, "symbol": fund.symbol},
                    )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Successfully updated fund {fund.name}",
                details={"fund_id": fund_id, "symbol_changed": symbol_changed},
            )

            return {
                "id": fund.id,
                "name": fund.name,
                "isin": fund.isin,
                "symbol": fund.symbol,
                "currency": fund.currency,
                "exchange": fund.exchange,
                "dividendType": fund.dividend_type.value,
                "investmentType": fund.investment_type.value,
            }, 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating fund: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
            )
            return {"error": "Error updating fund", "details": str(e)}, 500

    @ns.doc("delete_fund")
    @ns.response(200, "Fund deleted")
    @ns.response(404, "Fund not found", error_model)
    @ns.response(409, "Fund in use", error_model)
    @ns.response(500, "Server error", error_model)
    def delete(self, fund_id):
        """
        Delete a fund.

        Deletes a fund if it's not being used in any portfolios.
        Cannot delete funds that have transactions or are attached to portfolios.

        Returns error if fund is in use.
        """
        try:
            fund_details = FundService.delete_fund(fund_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Successfully deleted fund {fund_details['fund_name']}",
                details=fund_details,
            )

            return {"success": True, "message": "Fund deleted successfully"}, 200

        except ValueError as e:
            error_message = str(e)

            if "Cannot delete" in error_message and "attached to" in error_message:
                logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message="Cannot delete fund while attached to portfolios",
                    details={"fund_id": fund_id, "user_message": error_message},
                )
                return {"error": error_message}, 409
            else:
                return {"error": error_message}, 404

        except HTTPException:
            raise
        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error deleting fund: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
            )
            return {"error": "Error deleting fund", "details": str(e)}, 500


@ns.route("/<string:fund_id>/check-usage")
@ns.param("fund_id", "Fund unique identifier (UUID)")
class FundUsage(Resource):
    """Fund usage check endpoint."""

    @ns.doc("check_fund_usage")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self, fund_id):
        """
        Check if fund is being used.

        Returns information about whether a fund is being used in any portfolios,
        including:
        - Usage status
        - List of portfolios using the fund
        - Transaction count
        - Dividend count

        Useful before attempting to delete a fund.
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

            return usage_info, 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error checking fund usage: {e!s}",
                details={"error": str(e)},
            )
            return {"error": "Error checking fund usage", "details": str(e)}, 500


@ns.route("/<string:fund_id>/price/today")
@ns.param("fund_id", "Fund unique identifier (UUID)")
class FundTodayPrice(Resource):
    """Fund today price update endpoint."""

    @ns.doc("update_today_price")
    @ns.response(200, "Price updated", price_update_model)
    @ns.response(404, "Fund not found", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self, fund_id):
        """
        Update today's price for a fund.

        Fetches the current price from external sources (Yahoo Finance)
        and updates the database with today's price.

        Useful for real-time price updates during trading hours.
        """
        try:
            TodayPriceService.update_single_fund_price(fund_id)
            return {"success": True}, 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating today's price: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
            )
            return {"error": "Error updating price", "details": str(e)}, 500


@ns.route("/<string:fund_id>/price/historical")
@ns.param("fund_id", "Fund unique identifier (UUID)")
class FundHistoricalPrice(Resource):
    """Fund historical price update endpoint."""

    @ns.doc("update_historical_prices", security="apikey")
    @ns.response(200, "Prices updated", price_update_model)
    @ns.response(404, "Fund not found", error_model)
    @ns.response(500, "Server error", error_model)
    @require_api_key_restx
    def post(self, fund_id):
        """
        Update historical prices for a fund.

        Fetches historical price data from external sources
        and backfills the database with missing price records.

        This endpoint requires API key authentication to prevent abuse.

        Note: This can be a long-running operation for funds with
        extensive history. Consider rate limiting and monitoring.
        """
        try:
            HistoricalPriceService.update_historical_prices(fund_id)
            return {"success": True}, 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating historical prices: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
            )
            return {"error": "Error updating historical prices", "details": str(e)}, 500


@ns.route("/symbol/<string:symbol>")
@ns.param("symbol", "Trading symbol (e.g., AAPL, VOO)")
class SymbolInfo(Resource):
    """Symbol information lookup endpoint."""

    @ns.doc("get_symbol_info")
    @ns.response(200, "Success", symbol_info_model)
    @ns.response(404, "Symbol not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, symbol):
        """
        Get information about a trading symbol.

        Fetches detailed information about a stock or fund symbol
        from external sources (Yahoo Finance).

        Useful for validating symbols before creating funds
        or looking up additional information.

        Query Parameters:
        - force_refresh (bool): Force refresh from external source (default: false)
        """
        try:
            force_refresh = request.args.get("force_refresh", "false").lower() == "true"
            symbol_info = SymbolLookupService.get_symbol_info(symbol, force_refresh=force_refresh)

            if not symbol_info:
                return {"error": "Symbol not found"}, 404

            return symbol_info, 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error looking up symbol: {e!s}",
                details={"symbol": symbol, "error": str(e)},
            )
            return {"error": "Error looking up symbol", "details": str(e)}, 500


@ns.route("/update-all-prices")
class UpdateAllPrices(Resource):
    """Update all fund prices endpoint."""

    @ns.doc("update_all_fund_prices", security="apikey")
    @ns.response(200, "Prices updated")
    @ns.response(500, "Server error", error_model)
    @require_api_key_restx
    def post(self):
        """
        Update prices for all funds.

        Fetches current prices for all funds with symbols from external sources
        and updates the database.

        This endpoint is protected by API key authentication and is typically
        called on a scheduled basis (daily) to keep prices up-to-date.

        Warning: This can be a long-running operation.
        """
        try:
            result = FundService.update_all_fund_prices()
            return result, 200
        except HTTPException:
            raise
        except Exception as e:
            return {"success": False, "error": str(e)}, 500


# Fund prices endpoints
fund_price_model = ns.model(
    "FundPrice",
    {
        "id": fields.String(required=True, description="Price record ID"),
        "date": fields.String(required=True, description="Price date (YYYY-MM-DD)"),
        "price": fields.Float(required=True, description="Price value"),
    },
)


@ns.route("/fund-prices/<string:fund_id>")
@ns.param("fund_id", "Fund unique identifier (UUID)")
class FundPrices(Resource):
    """Fund price history endpoint."""

    @ns.doc("get_fund_prices")
    @ns.response(200, "Success", [fund_price_model])
    @ns.response(404, "Fund not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, fund_id):
        """
        Get price history for a fund.

        Returns all historical prices for a fund, ordered by date.
        Useful for charting and analysis.
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

            return [
                {"id": price.id, "date": price.date.isoformat(), "price": price.price}
                for price in prices
            ], 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error retrieving fund prices: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
            )
            return {"error": "Error retrieving fund prices", "details": str(e)}, 500


@ns.route("/fund-prices/<string:fund_id>/update")
@ns.param("fund_id", "Fund unique identifier (UUID)")
class FundPriceUpdate(Resource):
    """Fund price update endpoint."""

    @ns.doc("update_fund_prices")
    @ns.param("type", "Update type (today or historical)", _in="query")
    @ns.response(200, "Prices updated")
    @ns.response(404, "Fund not found", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self, fund_id):
        """
        Update fund prices from external source.

        Fetches and updates prices from Yahoo Finance.

        Query Parameters:
        - type: 'today' for current day price, 'historical' for complete history (default: 'today')

        Note: Historical updates can be long-running operations.
        """
        try:
            update_type = request.args.get("type", "today")

            if update_type == "today":
                response, status = TodayPriceService.update_todays_price(fund_id)
            else:
                response, status = HistoricalPriceService.update_historical_prices(fund_id)

            return response, status

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating fund prices: {e!s}",
                details={"fund_id": fund_id, "updateType": update_type, "error": str(e)},
            )
            return {"error": "Error updating fund prices", "details": str(e)}, 500


@ns.route("/history/<string:portfolio_id>")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class FundHistory(Resource):
    """Fund history endpoint for a specific portfolio."""

    @ns.doc("get_fund_history")
    @ns.param("start_date", "Start date (YYYY-MM-DD)", _in="query")
    @ns.param("end_date", "End date (YYYY-MM-DD)", _in="query")
    @ns.response(200, "Success")
    @ns.response(404, "Portfolio not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, portfolio_id):
        """
        Get historical fund values for a portfolio.

        Returns time-series data showing individual fund values
        within a portfolio over time.

        Query Parameters:
        - start_date: Start date for historical data (YYYY-MM-DD)
        - end_date: End date for historical data (YYYY-MM-DD)

        Returns daily data with the following structure:
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
                "dividends": 0,
                "fees": 0
              }
            ]
          }
        ]

        Useful for analyzing individual fund performance within a portfolio.
        """
        try:
            start_date = request.args.get("start_date")
            end_date = request.args.get("end_date")

            return FundService.get_fund_history(portfolio_id, start_date, end_date), 200
        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error retrieving fund history for portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500
