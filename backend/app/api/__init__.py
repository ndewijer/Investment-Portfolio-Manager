"""
Flask-RESTX API namespaces and documentation.

This module provides Swagger/OpenAPI documentation for all API endpoints.
Each namespace represents a logical grouping of related endpoints.
"""

from flask_restx import Api

# Initialize the API (this will be configured in run.py)
api = None


def init_api(flask_api_instance):
    """
    Initialize the API instance and register all namespaces.

    Args:
        flask_api_instance: The Flask-RESTX Api instance from run.py
    """
    global api
    api = flask_api_instance

    # Import and register namespaces
    from .system_namespace import ns as system_ns
    from .portfolio_namespace import ns as portfolio_ns
    from .fund_namespace import ns as fund_ns
    from .transaction_namespace import ns as transaction_ns
    from .dividend_namespace import ns as dividend_ns
    from .ibkr_namespace import ns as ibkr_ns
    from .developer_namespace import ns as developer_ns

    api.add_namespace(system_ns, path='/system')
    api.add_namespace(portfolio_ns, path='/portfolios')
    api.add_namespace(fund_ns, path='/funds')
    api.add_namespace(transaction_ns, path='/transactions')
    api.add_namespace(dividend_ns, path='/dividends')
    api.add_namespace(ibkr_ns, path='/ibkr')
    api.add_namespace(developer_ns, path='/developer')
