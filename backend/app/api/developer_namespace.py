"""
Developer API namespace for debugging and development utilities.

This namespace provides endpoints for:
- System logs viewing and management
- Database introspection
- Development utilities
- Exchange rate management
- CSV import/export templates

Warning: These endpoints should be disabled in production or protected with authentication.
"""

from datetime import date

from flask import request
from flask_restx import Namespace, Resource, fields

from ..models import LogCategory, LogLevel
from ..services.developer_service import DeveloperService
from ..services.logging_service import logger

# Create namespace
ns = Namespace("developer", description="Developer and debugging operations")

# Define models
log_entry_model = ns.model(
    "LogEntry",
    {
        "id": fields.String(required=True, description="Log entry ID"),
        "timestamp": fields.DateTime(required=True, description="Log timestamp"),
        "level": fields.String(required=True, description="Log level"),
        "category": fields.String(required=True, description="Log category"),
        "message": fields.String(required=True, description="Log message"),
        "details": fields.Raw(description="Additional details"),
    },
)

exchange_rate_model = ns.model(
    "ExchangeRate",
    {
        "fromCurrency": fields.String(required=True, description="Source currency"),
        "toCurrency": fields.String(required=True, description="Target currency"),
        "rate": fields.Float(required=True, description="Exchange rate"),
        "date": fields.String(description="Rate date"),
    },
)

fund_price_model = ns.model(
    "FundPrice",
    {
        "fundId": fields.String(required=True, description="Fund ID"),
        "price": fields.Float(required=True, description="Price"),
        "date": fields.String(required=True, description="Price date"),
    },
)

error_model = ns.model(
    "Error",
    {
        "error": fields.String(required=True, description="Error message"),
        "details": fields.String(description="Additional error details"),
    },
)


@ns.route("/logs")
class DeveloperLogs(Resource):
    """Developer logs endpoint."""

    @ns.doc("get_logs")
    @ns.param("level", "Filter by log level (comma-separated)", _in="query")
    @ns.param("category", "Filter by category (comma-separated)", _in="query")
    @ns.param("startDate", "Filter logs after this date (ISO 8601 format)", _in="query")
    @ns.param("endDate", "Filter logs before this date (ISO 8601 format)", _in="query")
    @ns.param("source", "Filter by source (partial match)", _in="query")
    @ns.param("message", "Filter by message content (partial match)", _in="query")
    @ns.param("sortDir", "Sort direction: 'asc' or 'desc' (default: desc)", _in="query")
    @ns.param("cursor", "Cursor for pagination (timestamp_id format)", _in="query")
    @ns.param("perPage", "Items per page (default: 50)", _in="query", type="integer")
    @ns.response(200, "Success", [log_entry_model])
    def get(self):
        """
        Get system logs with cursor-based pagination.

        Returns recent system logs for debugging and monitoring.
        Uses cursor-based pagination to prevent drift when new logs arrive.

        Warning: This endpoint can return sensitive information.
        Should be disabled or protected in production.

        Query Parameters (all in camelCase):
        - level: Filter by log level (comma-separated: debug,info,warning,error,critical)
        - category: Filter by category (comma-separated: system,fund,portfolio,transaction,etc.)
        - startDate: Filter logs after this date (ISO 8601 format, e.g., 2024-01-01T00:00:00Z)
        - endDate: Filter logs before this date (ISO 8601 format)
        - source: Filter by source (partial match)
        - message: Filter by message content (partial match)
        - sortDir: Sort direction - 'asc' (oldest first) or 'desc' (newest first, default)
        - cursor: Cursor for pagination (format: timestamp_id, e.g., 2024-01-01T12:00:00_abc123)
        - perPage: Items per page (default: 50, max: 200)

        Response (camelCase):
        - logs: Array of log entries
        - nextCursor: Cursor for next page (null if no more results)
        - hasMore: Boolean indicating if more results available
        - count: Number of logs in current page

        Note: Always sorts by timestamp + id for deterministic ordering.
        """
        from ..services.logging_service import LoggingService

        try:
            result = LoggingService.get_logs(
                levels=request.args.get("level"),
                categories=request.args.get("category"),
                start_date=request.args.get("startDate"),
                end_date=request.args.get("endDate"),
                source=request.args.get("source"),
                sort_dir=request.args.get("sortDir", "desc"),
                cursor=request.args.get("cursor"),
                per_page=int(request.args.get("perPage", 50)),
            )

            # Convert response to camelCase for frontend
            return {
                "logs": result["logs"],
                "nextCursor": result["next_cursor"],
                "hasMore": result["has_more"],
                "count": result["count"],
            }, 200
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error retrieving logs",
                details={"error": str(e)},
            )
            return {"error": "Error retrieving logs", "details": str(e)}, 500

    @ns.doc("clear_logs")
    @ns.response(200, "Logs cleared")
    def delete(self):
        """
        Clear all system logs.

        Deletes all log entries from the database.

        Warning: This operation cannot be undone.
        Use with caution in production environments.
        """
        from ..services.logging_service import LoggingService

        try:
            LoggingService.clear_logs()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message="All logs cleared by user",
                source="clear_logs",
            )

            return {"message": "All logs cleared successfully"}, 200
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error clearing logs",
                details={"error": str(e)},
            )
            return {"error": "Error clearing logs", "details": str(e)}, 500


@ns.route("/exchange-rate")
class DeveloperExchangeRate(Resource):
    """Developer exchange rate endpoint."""

    @ns.doc("get_exchange_rate")
    @ns.param("fromCurrency", "Source currency (e.g., USD)", required=True, _in="query")
    @ns.param("toCurrency", "Target currency (e.g., EUR)", required=True, _in="query")
    @ns.param("date", "Rate date (YYYY-MM-DD)", _in="query")
    @ns.response(200, "Success", exchange_rate_model)
    @ns.response(400, "Missing parameters", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get exchange rate for currency pair (camelCase API).

        Tests the exchange rate service with specific currency pairs.
        Useful for debugging currency conversion issues.

        Query Parameters (camelCase):
        - fromCurrency: Source currency code (e.g., USD)
        - toCurrency: Target currency code (e.g., EUR)
        - date: Optional date for historical rates (YYYY-MM-DD)
        """
        # Read camelCase from query params
        from_currency = request.args.get("fromCurrency")
        to_currency = request.args.get("toCurrency")
        date_str = request.args.get("date")

        if not from_currency or not to_currency:
            return {"error": "Missing required parameters: fromCurrency and toCurrency"}, 400

        try:
            rate_date = None
            if date_str:
                rate_date = date.fromisoformat(date_str)

            # Service layer uses snake_case internally
            rate = DeveloperService.get_exchange_rate(from_currency, to_currency, rate_date)

            # Return camelCase response
            return {
                "fromCurrency": from_currency,
                "toCurrency": to_currency,
                "rate": rate,
                "date": rate_date.isoformat() if rate_date else None,
            }, 200

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error retrieving exchange rate",
                details={"error": str(e)},
            )
            return {"error": "Error retrieving exchange rate", "details": str(e)}, 500

    @ns.doc("set_exchange_rate")
    @ns.expect(exchange_rate_model, validate=True)
    @ns.response(200, "Exchange rate set")
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Set exchange rate for currency pair (camelCase API).

        Manually sets an exchange rate for testing or correction purposes.

        Useful for:
        - Testing currency conversions
        - Correcting historical rates
        - Overriding automatic rate lookups

        Request Body (camelCase):
        - fromCurrency: Source currency code
        - toCurrency: Target currency code
        - rate: Exchange rate value
        - date: Optional date (YYYY-MM-DD)
        """
        data = request.json

        try:
            rate_date = None
            if data.get("date"):
                rate_date = date.fromisoformat(data["date"])

            # Convert camelCase to snake_case for service layer
            DeveloperService.set_exchange_rate(
                from_currency=data["fromCurrency"],
                to_currency=data["toCurrency"],
                rate=data["rate"],
                date=rate_date,
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message="Exchange rate set",
                details=data,
            )

            return {"success": True, "message": "Exchange rate set successfully"}, 200

        except ValueError as e:
            # Date format errors return 400 (matching legacy behavior)
            return {"message": str(e)}, 400
        except Exception as e:
            # All other exceptions return 400 (matching legacy behavior)
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error setting exchange rate",
                details={"error": str(e)},
            )
            return {"message": f"Error setting exchange rate: {e!s}"}, 400


@ns.route("/fund-price")
class DeveloperFundPrice(Resource):
    """Developer fund price management endpoint."""

    @ns.doc("get_fund_price")
    @ns.param("fundId", "Fund ID", required=True, _in="query")
    @ns.param("date", "Price date (YYYY-MM-DD)", required=True, _in="query")
    @ns.response(200, "Success", fund_price_model)
    @ns.response(400, "Missing parameters", error_model)
    @ns.response(404, "Price not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get fund price for specific date (camelCase API).

        Retrieves the price for a fund on a specific date.
        Useful for debugging price lookup issues.

        Query Parameters (camelCase):
        - fundId: Fund identifier
        - date: Price date (YYYY-MM-DD)
        """
        # Read camelCase from query params
        fund_id = request.args.get("fundId")
        date_str = request.args.get("date")

        if not fund_id or not date_str:
            return {"error": "Missing required parameters: fundId and date"}, 400

        try:
            price_date = date.fromisoformat(date_str)
            # Service layer uses snake_case
            result = DeveloperService.get_fund_price(fund_id, price_date)

            # Return 404 if result is None (matching legacy behavior)
            if not result:
                return {"message": "Fund price not found"}, 404

            # Convert response to camelCase
            return {
                "fundId": result["fund_id"],
                "price": result["price"],
                "date": result["date"],
            }, 200

        except ValueError as e:
            # Date format errors return 400
            # (matching legacy behavior via exception cascade to general handler)
            return {"message": str(e)}, 400
        except Exception as e:
            # All other exceptions return 500 (matching legacy behavior)
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error retrieving fund price",
                details={"error": str(e)},
            )
            return {"message": f"Error retrieving fund price: {e!s}"}, 500

    @ns.doc("set_fund_price")
    @ns.expect(fund_price_model, validate=True)
    @ns.response(200, "Price set")
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Set fund price for specific date (camelCase API).

        Manually sets a fund price for testing or correction purposes.

        Useful for:
        - Testing portfolio calculations
        - Correcting historical prices
        - Filling gaps in price data

        Request Body (camelCase):
        - fundId: Fund identifier
        - price: Price value
        - date: Price date (YYYY-MM-DD)
        """
        data = request.json

        try:
            price_date = None
            if data.get("date"):
                price_date = date.fromisoformat(data["date"])

            # Convert camelCase to snake_case for service layer
            result = DeveloperService.set_fund_price(
                fund_id=data["fundId"], price=data["price"], date_=price_date
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message="Fund price set",
                details=data,
            )

            return result, 200

        except ValueError as e:
            # Date format errors return 400 (matching legacy behavior)
            return {"message": str(e)}, 400
        except Exception as e:
            # All other exceptions return 400 (matching legacy behavior)
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error setting fund price",
                details={"error": str(e)},
            )
            return {"message": f"Error setting fund price: {e!s}"}, 400


@ns.route("/fund-price/<string:fund_id>")
class DeveloperFundPriceById(Resource):
    """Get fund price by ID endpoint."""

    @ns.doc("get_fund_price_by_id")
    @ns.param("date", "Price date (YYYY-MM-DD)", _in="query")
    @ns.response(200, "Success", fund_price_model)
    @ns.response(404, "Price not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, fund_id):
        """
        Get fund price for specific fund and date.

        Retrieves the price for a fund on a specific date.
        If date is not provided, returns today's price.
        """
        date_str = request.args.get("date")

        try:
            from datetime import datetime

            if date_str:
                price_date = datetime.strptime(date_str, "%Y-%m-%d").date()
            else:
                price_date = datetime.now().date()

            result = DeveloperService.get_fund_price(fund_id, price_date)

            # Return 404 if result is None (matching legacy behavior)
            if not result:
                return {"message": "Fund price not found"}, 404

            # Convert response to camelCase
            return {
                "fundId": result["fund_id"],
                "price": result["price"],
                "date": result["date"],
            }, 200

        except Exception as e:
            # All exceptions return 500 (matching legacy behavior - ValueError also goes here)
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error retrieving fund price",
                details={"error": str(e)},
            )
            return {"message": f"Error retrieving fund price: {e!s}"}, 500


@ns.route("/csv/transactions/template")
class TransactionCSVTemplate(Resource):
    """Transaction CSV template endpoint."""

    @ns.doc("get_transaction_csv_template")
    @ns.response(200, "CSV template")
    def get(self):
        """
        Get CSV template for transaction import.

        Returns a CSV template with headers and example data
        for importing transactions via CSV file.

        Useful for bulk transaction imports.
        """
        try:
            template = DeveloperService.get_csv_template()
            return template, 200
        except Exception as e:
            return {"error": "Error generating template", "details": str(e)}, 500


@ns.route("/csv/fund-prices/template")
class FundPriceCSVTemplate(Resource):
    """Fund price CSV template endpoint."""

    @ns.doc("get_fund_price_csv_template")
    @ns.response(200, "CSV template")
    def get(self):
        """
        Get CSV template for fund price import.

        Returns a CSV template with headers and example data
        for importing fund prices via CSV file.

        Useful for bulk price imports and historical data backfilling.
        """
        try:
            template = DeveloperService.get_fund_price_csv_template()
            return template, 200
        except Exception as e:
            return {"error": "Error generating template", "details": str(e)}, 500


@ns.route("/data/funds")
class DeveloperFunds(Resource):
    """Developer funds data endpoint."""

    @ns.doc("get_funds_data")
    @ns.response(200, "Success")
    def get(self):
        """
        Get all funds data.

        Returns complete fund data for debugging and inspection.
        """
        try:
            funds = DeveloperService.get_funds()
            return funds, 200
        except Exception as e:
            return {"error": "Error retrieving funds", "details": str(e)}, 500


@ns.route("/data/portfolios")
class DeveloperPortfolios(Resource):
    """Developer portfolios data endpoint."""

    @ns.doc("get_portfolios_data")
    @ns.response(200, "Success")
    def get(self):
        """
        Get all portfolios data.

        Returns complete portfolio data for debugging and inspection.
        """
        try:
            portfolios = DeveloperService.get_portfolios()
            return portfolios, 200
        except Exception as e:
            return {"error": "Error retrieving portfolios", "details": str(e)}, 500


@ns.route("/import-transactions")
class ImportTransactions(Resource):
    """CSV transaction import endpoint."""

    @ns.doc("import_transactions")
    @ns.expect(
        ns.parser()
        .add_argument(
            "file",
            location="files",
            type="file",
            required=True,
            help="CSV file with transaction data",
        )
        .add_argument(
            "fundId",
            location="form",
            type=str,
            required=True,
            help="Portfolio-Fund relationship ID",
        )
    )
    @ns.response(200, "Import successful")
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Import transactions from CSV file (camelCase API).

        Accepts a CSV file with transaction data and imports it into the system.

        CSV Format:
        - Headers: date,type,shares,cost_per_share
        - Date format: YYYY-MM-DD
        - Type: buy, sell, dividend, etc.

        Form Data (camelCase):
        - file: CSV file containing transaction data
        - fundId: Portfolio-Fund relationship ID
        """
        try:
            if "file" not in request.files:
                return {"message": "No file provided"}, 400

            file = request.files["file"]
            # Read camelCase from form data
            portfolio_fund_id = request.form.get("fundId")

            if not portfolio_fund_id:
                return {"message": "No fundId provided"}, 400

            if not file.filename.endswith(".csv"):
                return {"message": "File must be CSV"}, 400

            # Read file content and delegate to service
            file_content = file.read()
            count = DeveloperService.import_transactions_csv(file_content, portfolio_fund_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message=f"Successfully imported {count} transactions",
                details={"transaction_count": count, "portfolio_fund_id": portfolio_fund_id},
            )

            return {"message": f"Successfully imported {count} transactions"}, 200

        except ValueError as e:
            return {"message": str(e)}, 400
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error during transaction import",
                details={"error": str(e)},
            )
            return {"error": "Error during transaction import", "details": str(e)}, 500


@ns.route("/import-fund-prices")
class ImportFundPrices(Resource):
    """CSV fund price import endpoint."""

    @ns.doc("import_fund_prices")
    @ns.expect(
        ns.parser()
        .add_argument(
            "file",
            location="files",
            type="file",
            required=True,
            help="CSV file with fund price data",
        )
        .add_argument("fundId", location="form", type=str, required=True, help="Fund ID")
    )
    @ns.response(200, "Import successful")
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Import fund prices from CSV file (camelCase API).

        Accepts a CSV file with fund price data and imports it into the system.

        CSV Format:
        - Headers: date,price
        - Date format: YYYY-MM-DD
        - Price: Decimal number

        Form Data (camelCase):
        - file: CSV file containing fund price data
        - fundId: Fund ID
        """
        try:
            if "file" not in request.files:
                return {"message": "No file provided"}, 400

            file = request.files["file"]
            # Read camelCase from form data
            fund_id = request.form.get("fundId")

            if not fund_id:
                return {"message": "No fundId provided"}, 400

            if not file.filename.endswith(".csv"):
                return {"message": "File must be CSV"}, 400

            # Read file content and delegate to service
            file_content = file.read()
            count = DeveloperService.import_fund_prices_csv(file_content, fund_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message=f"Successfully imported {count} fund prices",
                details={"fund_id": fund_id, "price_count": count},
            )

            return {"message": f"Successfully imported {count} fund prices"}, 200

        except ValueError as e:
            return {"message": str(e)}, 400
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error during fund price import",
                details={"error": str(e)},
            )
            return {"error": "Error during fund price import", "details": str(e)}, 500


@ns.route("/system-settings/logging")
class LoggingSettings(Resource):
    """Logging configuration settings endpoint."""

    @ns.doc("get_logging_settings")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get current logging configuration settings.

        Returns the current logging enabled state and level.
        Useful for debugging and system monitoring.
        """
        try:
            from ..services.logging_service import LoggingService

            settings = LoggingService.get_logging_settings()
            return settings, 200
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error retrieving logging settings",
                details={"error": str(e)},
            )
            return {"error": "Error retrieving logging settings", "details": str(e)}, 500

    @ns.doc("update_logging_settings")
    @ns.expect(
        ns.model(
            "LoggingSettings",
            {
                "enabled": fields.Boolean(required=True, description="Enable/disable logging"),
                "level": fields.String(
                    required=True,
                    description="Logging level (debug, info, warning, error, critical)",
                ),
            },
        ),
        validate=True,
    )
    @ns.response(200, "Settings updated")
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def put(self):
        """
        Update logging configuration settings.

        Updates the system logging enabled state and level.
        Changes take effect immediately.

        Request Body:
        - enabled: Boolean - Enable/disable logging
        - level: String - Logging level (debug, info, warning, error, critical)
        """
        try:
            from ..services.logging_service import LoggingService

            data = request.json
            result = LoggingService.update_logging_settings(
                enabled=data["enabled"], level=data["level"]
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message="Logging settings updated",
                details=result,
            )

            return result, 200
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error updating logging settings",
                details={"error": str(e)},
            )
            return {"error": "Error updating logging settings", "details": str(e)}, 500
