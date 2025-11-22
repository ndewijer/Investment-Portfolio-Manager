"""
Common data models and schemas for Flask-RESTX API documentation.

This module defines reusable models that are shared across multiple endpoints.
"""

from flask_restx import fields


def get_error_model(api):
    """
    Get the standard error response model.

    Args:
        api: Flask-RESTX Api instance

    Returns:
        Model definition for error responses
    """
    return api.model('Error', {
        'error': fields.String(required=True, description='Error message'),
        'details': fields.String(description='Additional error details')
    })


def get_success_model(api):
    """
    Get the standard success response model.

    Args:
        api: Flask-RESTX Api instance

    Returns:
        Model definition for success responses
    """
    return api.model('Success', {
        'success': fields.Boolean(required=True, description='Operation success status'),
        'message': fields.String(description='Success message')
    })
