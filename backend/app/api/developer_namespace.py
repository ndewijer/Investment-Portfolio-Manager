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
ns = Namespace('developer', description='Developer and debugging operations')

# Define models
log_entry_model = ns.model('LogEntry', {
    'id': fields.String(required=True, description='Log entry ID'),
    'timestamp': fields.DateTime(required=True, description='Log timestamp'),
    'level': fields.String(required=True, description='Log level'),
    'category': fields.String(required=True, description='Log category'),
    'message': fields.String(required=True, description='Log message'),
    'details': fields.Raw(description='Additional details')
})

exchange_rate_model = ns.model('ExchangeRate', {
    'from_currency': fields.String(required=True, description='Source currency'),
    'to_currency': fields.String(required=True, description='Target currency'),
    'rate': fields.Float(required=True, description='Exchange rate'),
    'date': fields.String(description='Rate date')
})

fund_price_model = ns.model('FundPrice', {
    'fund_id': fields.String(required=True, description='Fund ID'),
    'price': fields.Float(required=True, description='Price'),
    'date': fields.String(required=True, description='Price date')
})

error_model = ns.model('Error', {
    'error': fields.String(required=True, description='Error message'),
    'details': fields.String(description='Additional error details')
})


@ns.route('/logs')
class DeveloperLogs(Resource):
    """Developer logs endpoint."""

    @ns.doc('get_logs')
    @ns.param('level', 'Filter by log level', _in='query')
    @ns.param('category', 'Filter by category', _in='query')
    @ns.param('limit', 'Limit number of results', _in='query')
    @ns.response(200, 'Success', [log_entry_model])
    def get(self):
        """
        Get system logs.

        Returns recent system logs for debugging and monitoring.
        Supports filtering by level, category, and limit.

        Warning: This endpoint can return sensitive information.
        Should be disabled or protected in production.

        Query Parameters:
        - level: Filter by log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
        - category: Filter by category (SYSTEM, FUND, PORTFOLIO, TRANSACTION, etc.)
        - limit: Maximum number of logs to return (default: 100)
        """
        # Note: The actual log retrieval logic should be implemented in the service layer
        # For now, returning a placeholder message
        return {"message": "Log retrieval - implement in DeveloperService"}, 200

    @ns.doc('clear_logs')
    @ns.response(200, 'Logs cleared')
    def delete(self):
        """
        Clear all system logs.

        Deletes all log entries from the database.

        Warning: This operation cannot be undone.
        Use with caution in production environments.
        """
        # Note: The actual log clearing logic should be implemented in the service layer
        return {"message": "Log clearing - implement in DeveloperService"}, 200


@ns.route('/exchange-rate')
class DeveloperExchangeRate(Resource):
    """Developer exchange rate endpoint."""

    @ns.doc('get_exchange_rate')
    @ns.param('from_currency', 'Source currency (e.g., USD)', required=True, _in='query')
    @ns.param('to_currency', 'Target currency (e.g., EUR)', required=True, _in='query')
    @ns.param('date', 'Rate date (YYYY-MM-DD)', _in='query')
    @ns.response(200, 'Success', exchange_rate_model)
    @ns.response(400, 'Missing parameters', error_model)
    @ns.response(500, 'Server error', error_model)
    def get(self):
        """
        Get exchange rate for currency pair.

        Tests the exchange rate service with specific currency pairs.
        Useful for debugging currency conversion issues.

        Query Parameters:
        - from_currency: Source currency code (e.g., USD)
        - to_currency: Target currency code (e.g., EUR)
        - date: Optional date for historical rates (YYYY-MM-DD)
        """
        from_currency = request.args.get('from_currency')
        to_currency = request.args.get('to_currency')
        date_str = request.args.get('date')

        if not from_currency or not to_currency:
            return {"error": "Missing required parameters: from_currency and to_currency"}, 400

        try:
            rate_date = None
            if date_str:
                rate_date = date.fromisoformat(date_str)

            rate = DeveloperService.get_exchange_rate(from_currency, to_currency, rate_date)

            return {
                "from_currency": from_currency,
                "to_currency": to_currency,
                "rate": rate,
                "date": rate_date.isoformat() if rate_date else None
            }, 200

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error retrieving exchange rate",
                details={"error": str(e)},
            )
            return {"error": "Error retrieving exchange rate", "details": str(e)}, 500

    @ns.doc('set_exchange_rate')
    @ns.expect(exchange_rate_model, validate=True)
    @ns.response(200, 'Exchange rate set')
    @ns.response(400, 'Validation error', error_model)
    @ns.response(500, 'Server error', error_model)
    def post(self):
        """
        Set exchange rate for currency pair.

        Manually sets an exchange rate for testing or correction purposes.

        Useful for:
        - Testing currency conversions
        - Correcting historical rates
        - Overriding automatic rate lookups
        """
        data = request.json

        try:
            rate_date = None
            if data.get('date'):
                rate_date = date.fromisoformat(data['date'])

            DeveloperService.set_exchange_rate(
                from_currency=data['from_currency'],
                to_currency=data['to_currency'],
                rate=data['rate'],
                date=rate_date
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message="Exchange rate set",
                details=data,
            )

            return {"success": True, "message": "Exchange rate set successfully"}, 200

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error setting exchange rate",
                details={"error": str(e)},
            )
            return {"error": "Error setting exchange rate", "details": str(e)}, 500


@ns.route('/fund-price')
class DeveloperFundPrice(Resource):
    """Developer fund price management endpoint."""

    @ns.doc('get_fund_price')
    @ns.param('fund_id', 'Fund ID', required=True, _in='query')
    @ns.param('date', 'Price date (YYYY-MM-DD)', required=True, _in='query')
    @ns.response(200, 'Success', fund_price_model)
    @ns.response(400, 'Missing parameters', error_model)
    @ns.response(404, 'Price not found', error_model)
    @ns.response(500, 'Server error', error_model)
    def get(self):
        """
        Get fund price for specific date.

        Retrieves the price for a fund on a specific date.
        Useful for debugging price lookup issues.
        """
        fund_id = request.args.get('fund_id')
        date_str = request.args.get('date')

        if not fund_id or not date_str:
            return {"error": "Missing required parameters: fund_id and date"}, 400

        try:
            price_date = date.fromisoformat(date_str)
            result = DeveloperService.get_fund_price(fund_id, price_date)

            return result, 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error retrieving fund price",
                details={"error": str(e)},
            )
            return {"error": "Error retrieving fund price", "details": str(e)}, 500

    @ns.doc('set_fund_price')
    @ns.expect(fund_price_model, validate=True)
    @ns.response(200, 'Price set')
    @ns.response(400, 'Validation error', error_model)
    @ns.response(500, 'Server error', error_model)
    def post(self):
        """
        Set fund price for specific date.

        Manually sets a fund price for testing or correction purposes.

        Useful for:
        - Testing portfolio calculations
        - Correcting historical prices
        - Filling gaps in price data
        """
        data = request.json

        try:
            price_date = None
            if data.get('date'):
                price_date = date.fromisoformat(data['date'])

            result = DeveloperService.set_fund_price(
                fund_id=data['fund_id'],
                price=data['price'],
                date_=price_date
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DEVELOPER,
                message="Fund price set",
                details=data,
            )

            return result, 200

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DEVELOPER,
                message="Error setting fund price",
                details={"error": str(e)},
            )
            return {"error": "Error setting fund price", "details": str(e)}, 500


@ns.route('/csv/transactions/template')
class TransactionCSVTemplate(Resource):
    """Transaction CSV template endpoint."""

    @ns.doc('get_transaction_csv_template')
    @ns.response(200, 'CSV template')
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


@ns.route('/csv/fund-prices/template')
class FundPriceCSVTemplate(Resource):
    """Fund price CSV template endpoint."""

    @ns.doc('get_fund_price_csv_template')
    @ns.response(200, 'CSV template')
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


@ns.route('/data/funds')
class DeveloperFunds(Resource):
    """Developer funds data endpoint."""

    @ns.doc('get_funds_data')
    @ns.response(200, 'Success')
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


@ns.route('/data/portfolios')
class DeveloperPortfolios(Resource):
    """Developer portfolios data endpoint."""

    @ns.doc('get_portfolios_data')
    @ns.response(200, 'Success')
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
