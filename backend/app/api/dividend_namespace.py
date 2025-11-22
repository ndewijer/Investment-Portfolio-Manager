"""
Dividend API namespace for managing dividend income and reinvestment.

This namespace provides endpoints for:
- Creating and tracking dividend payments
- Managing dividend reinvestment
- Portfolio dividend allocation
"""

from flask_restx import Namespace, Resource, fields

# Create namespace
ns = Namespace('dividends', description='Dividend management operations')

# Define models
dividend_model = ns.model('Dividend', {
    'id': fields.String(required=True, description='Dividend unique identifier (UUID)'),
    'fund_id': fields.String(required=True, description='Fund ID'),
    'ex_date': fields.String(required=True, description='Ex-dividend date'),
    'payment_date': fields.String(required=True, description='Payment date'),
    'amount': fields.Float(required=True, description='Dividend amount per share'),
    'currency': fields.String(required=True, description='Currency code')
})

error_model = ns.model('Error', {
    'error': fields.String(required=True, description='Error message')
})


@ns.route('')
class DividendList(Resource):
    """Dividend collection endpoint."""

    @ns.doc('list_dividends')
    @ns.response(200, 'Success', [dividend_model])
    def get(self):
        """
        Get all dividends.

        Returns a list of all dividend payments across all funds.
        """
        return {"message": "Dividend endpoints - legacy routes still active"}, 200


@ns.route('/<string:dividend_id>')
@ns.param('dividend_id', 'Dividend unique identifier (UUID)')
class Dividend(Resource):
    """Dividend detail endpoint."""

    @ns.doc('get_dividend')
    @ns.response(200, 'Success', dividend_model)
    @ns.response(404, 'Dividend not found', error_model)
    def get(self, dividend_id):
        """
        Get dividend details.

        Returns detailed information about a specific dividend payment.
        """
        return {"message": "Dividend endpoints - legacy routes still active"}, 200
