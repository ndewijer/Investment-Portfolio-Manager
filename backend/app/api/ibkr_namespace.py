"""
IBKR API namespace for Interactive Brokers integration.

This namespace provides endpoints for:
- IBKR Flex Query configuration
- Transaction import from IBKR
- Transaction inbox management and allocation
- Automated import scheduling
"""

from flask_restx import Namespace, Resource, fields

# Create namespace
ns = Namespace('ibkr', description='Interactive Brokers integration operations')

# Define models
ibkr_config_model = ns.model('IBKRConfig', {
    'is_configured': fields.Boolean(required=True, description='Whether IBKR is configured'),
    'last_import': fields.String(description='Last import timestamp'),
    'auto_import_enabled': fields.Boolean(description='Auto import status')
})

ibkr_transaction_model = ns.model('IBKRTransaction', {
    'id': fields.String(required=True, description='Transaction ID'),
    'symbol': fields.String(required=True, description='Security symbol'),
    'transaction_type': fields.String(required=True, description='Transaction type'),
    'quantity': fields.Float(required=True, description='Quantity'),
    'price': fields.Float(required=True, description='Price'),
    'trade_date': fields.String(required=True, description='Trade date'),
    'status': fields.String(required=True, description='Processing status')
})

allocation_model = ns.model('Allocation', {
    'portfolio_id': fields.String(required=True, description='Portfolio ID'),
    'percentage': fields.Float(required=True, description='Allocation percentage')
})

error_model = ns.model('Error', {
    'error': fields.String(required=True, description='Error message')
})


@ns.route('/config')
class IBKRConfig(Resource):
    """IBKR configuration endpoint."""

    @ns.doc('get_ibkr_config')
    @ns.response(200, 'Success', ibkr_config_model)
    def get(self):
        """
        Get IBKR configuration status.

        Returns information about the current IBKR Flex Query configuration,
        including connection status and last import time.
        """
        return {"message": "IBKR config endpoints - legacy routes still active"}, 200

    @ns.doc('update_ibkr_config')
    @ns.response(200, 'Configuration updated')
    def post(self):
        """
        Create or update IBKR configuration.

        Configure IBKR Flex Query credentials for automated transaction import.
        """
        return {"message": "IBKR config endpoints - legacy routes still active"}, 200


@ns.route('/inbox')
class IBKRInbox(Resource):
    """IBKR transaction inbox endpoint."""

    @ns.doc('list_ibkr_transactions')
    @ns.response(200, 'Success', [ibkr_transaction_model])
    def get(self):
        """
        Get all pending IBKR transactions.

        Returns a list of imported transactions from IBKR
        that are pending portfolio allocation.
        """
        return {"message": "IBKR inbox endpoints - legacy routes still active"}, 200


@ns.route('/inbox/<string:transaction_id>/allocate')
@ns.param('transaction_id', 'Transaction unique identifier')
class IBKRTransactionAllocate(Resource):
    """IBKR transaction allocation endpoint."""

    @ns.doc('allocate_transaction')
    @ns.expect([allocation_model])
    @ns.response(200, 'Transaction allocated')
    @ns.response(400, 'Invalid allocation')
    def post(self, transaction_id):
        """
        Allocate IBKR transaction to portfolios.

        Allocates an imported IBKR transaction to one or more portfolios
        with specified percentages. Allocations must sum to 100%.
        """
        return {"message": "IBKR allocation endpoints - legacy routes still active"}, 200


@ns.route('/import')
class IBKRImport(Resource):
    """IBKR import trigger endpoint."""

    @ns.doc('trigger_ibkr_import')
    @ns.response(200, 'Import triggered')
    def post(self):
        """
        Manually trigger IBKR transaction import.

        Initiates an import of transactions from IBKR Flex Query.
        Normally this runs automatically on a schedule.
        """
        return {"message": "IBKR import endpoints - legacy routes still active"}, 200
