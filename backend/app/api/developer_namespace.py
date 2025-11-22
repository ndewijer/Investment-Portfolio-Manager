"""
Developer API namespace for debugging and development utilities.

This namespace provides endpoints for:
- System logs viewing and management
- Database introspection
- Development utilities
- Cache management

These endpoints should be disabled in production or protected with authentication.
"""

from flask_restx import Namespace, Resource, fields

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

error_model = ns.model('Error', {
    'error': fields.String(required=True, description='Error message')
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
        """
        return {"message": "Developer log endpoints - legacy routes still active"}, 200

    @ns.doc('clear_logs')
    @ns.response(200, 'Logs cleared')
    def delete(self):
        """
        Clear all system logs.

        Deletes all log entries from the database.

        Warning: This operation cannot be undone.
        """
        return {"message": "Developer log endpoints - legacy routes still active"}, 200


@ns.route('/exchange-rate')
class DeveloperExchangeRate(Resource):
    """Developer exchange rate endpoint."""

    @ns.doc('test_exchange_rate')
    @ns.param('from_currency', 'Source currency', _in='query')
    @ns.param('to_currency', 'Target currency', _in='query')
    @ns.response(200, 'Success')
    def get(self):
        """
        Test exchange rate lookup.

        Tests the exchange rate service with specific currency pairs.
        Useful for debugging currency conversion issues.
        """
        return {"message": "Developer exchange rate endpoints - legacy routes still active"}, 200
