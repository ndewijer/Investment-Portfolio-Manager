"""
API routes for IBKR Flex integration.

This module provides routes for:
- IBKR configuration management
- Transaction imports
- Inbox management
- Transaction allocation and processing
"""

from datetime import datetime

from flask import Blueprint, jsonify, request
from flask.views import MethodView

from ..models import LogCategory, LogLevel, db
from ..services.ibkr_config_service import IBKRConfigService
from ..services.ibkr_flex_service import IBKRFlexService
from ..services.ibkr_transaction_service import IBKRTransactionService
from ..services.logging_service import logger, track_request

ibkr = Blueprint("ibkr", __name__)


class IBKRConfigAPI(MethodView):
    """
    RESTful API for IBKR configuration management.

    Provides endpoints for:
    - Saving IBKR configuration
    - Retrieving configuration status
    - Testing connection
    - Deleting configuration
    """

    def get(self):
        """
        Get IBKR configuration status.

        Returns:
            JSON response with configuration status (token excluded)
        """
        config_status = IBKRConfigService.get_config_status()
        return jsonify(config_status), 200

    def post(self):
        """
        Save or update IBKR configuration.

        Request body:
            {
                "flex_token": "string",
                "flex_query_id": "string",
                "token_expires_at": "ISO datetime string" (optional),
                "auto_import_enabled": boolean (optional)
            }

        Returns:
            JSON response with success status
        """
        data = request.get_json()

        if not data or "flex_token" not in data or "flex_query_id" not in data:
            return jsonify({"error": "Missing required fields"}), 400

        try:
            # Parse token expiration date if provided
            token_expires_at = None
            if data.get("token_expires_at"):
                try:
                    token_expires_at = datetime.fromisoformat(data["token_expires_at"])
                except (ValueError, TypeError):
                    return jsonify({"error": "Invalid token_expires_at format"}), 400

            # Save config using service
            config = IBKRConfigService.save_config(
                flex_token=data["flex_token"],
                flex_query_id=data["flex_query_id"],
                token_expires_at=token_expires_at,
                auto_import_enabled=data.get("auto_import_enabled"),
                enabled=data.get("enabled"),
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR configuration saved",
                details={"query_id": config.flex_query_id},
            )

            return jsonify({"success": True, "message": "Configuration saved successfully"}), 200

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to save IBKR configuration",
                details={"error": str(e)},
            )
            return jsonify({"error": "Failed to save configuration", "details": str(e)}), 500

    def delete(self):
        """
        Delete IBKR configuration.

        Returns:
            JSON response with success status
        """
        try:
            config_details = IBKRConfigService.delete_config()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR configuration deleted",
                details=config_details,
            )

            return jsonify({"success": True, "message": "Configuration deleted successfully"}), 200

        except ValueError as e:
            return jsonify({"error": str(e)}), 404

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to delete IBKR configuration",
                details={"error": str(e)},
            )
            return jsonify({"error": "Failed to delete configuration", "details": str(e)}), 500


@ibkr.route("/ibkr/config/test", methods=["POST"])
def test_connection():
    """
    Test IBKR connection.

    Request body:
        {
            "flex_token": "string",
            "flex_query_id": "string"
        }

    Returns:
        JSON response with test results
    """
    data = request.get_json()

    if not data or "flex_token" not in data or "flex_query_id" not in data:
        return jsonify({"error": "Missing required fields"}), 400

    try:
        service = IBKRFlexService()
        result = service.test_connection(data["flex_token"], data["flex_query_id"])

        if result["success"]:
            return jsonify(result), 200
        else:
            return jsonify(result), 400

    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Error testing IBKR connection",
            details={"error": str(e)},
        )
        return jsonify({"error": "Connection test failed", "details": str(e)}), 500


@ibkr.route("/ibkr/import", methods=["POST"])
def trigger_import():
    """
    Trigger manual IBKR transaction import.

    Returns:
        JSON response with import results
    """
    config = IBKRConfigService.get_first_config()

    if not config:
        return jsonify({"error": "IBKR not configured"}), 400

    if not config.enabled:
        return jsonify({"error": "IBKR integration is disabled"}), 403

    service = IBKRFlexService()
    response, status = service.trigger_manual_import(config)
    return jsonify(response), status


@ibkr.route("/ibkr/inbox", methods=["GET"])
def get_inbox():
    """
    Get pending IBKR transactions.

    Query parameters:
        status: Filter by status (optional)
        transaction_type: Filter by type (optional)

    Returns:
        JSON array of pending transactions
    """
    status = request.args.get("status", "pending")
    transaction_type = request.args.get("transaction_type")

    transactions = IBKRTransactionService.get_inbox(
        status=status, transaction_type=transaction_type
    )

    return jsonify(transactions)


@ibkr.route("/ibkr/inbox/count", methods=["GET"])
def get_inbox_count():
    """
    Get count of IBKR transactions.

    Query parameters:
        status: Filter by status (optional, defaults to 'pending')

    Returns:
        JSON response with count: {"count": 5}
    """
    try:
        status = request.args.get("status", "pending")
        count = IBKRTransactionService.get_inbox_count(status=status)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Retrieved IBKR inbox count for status '{status}'",
            details={"status": status, "count": count},
        )

        return jsonify({"count": count}), 200

    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Error retrieving IBKR inbox count",
            details={"error": str(e)},
        )
        return jsonify({"error": "Failed to retrieve count", "details": str(e)}), 500


@ibkr.route("/ibkr/inbox/<transaction_id>", methods=["GET"])
def get_inbox_transaction(transaction_id):
    """
    Get specific IBKR transaction details.

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON object with transaction details
    """
    transaction_detail = IBKRTransactionService.get_transaction_detail(transaction_id)
    return jsonify(transaction_detail)


@ibkr.route("/ibkr/inbox/<transaction_id>/ignore", methods=["POST"])
def ignore_transaction(transaction_id):
    """
    Mark IBKR transaction as ignored.

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response with success status
    """
    response, status = IBKRTransactionService.ignore_transaction(transaction_id)
    return jsonify(response), status


@ibkr.route("/ibkr/inbox/<transaction_id>", methods=["DELETE"])
def delete_transaction(transaction_id):
    """
    Delete IBKR transaction (only if not processed).

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response with success status
    """
    response, status = IBKRTransactionService.delete_transaction(transaction_id)
    return jsonify(response), status


@ibkr.route("/ibkr/portfolios", methods=["GET"])
def get_portfolios_for_allocation():
    """
    Get available portfolios for transaction allocation.

    Returns:
        JSON array of active portfolios
    """
    from ..services.portfolio_service import PortfolioService

    portfolios = PortfolioService.get_active_portfolios()

    return jsonify([{"id": p.id, "name": p.name, "description": p.description} for p in portfolios])


@ibkr.route("/ibkr/inbox/<transaction_id>/eligible-portfolios", methods=["GET"])
@track_request
def get_eligible_portfolios(transaction_id):
    """
    Get portfolios eligible for allocating a specific IBKR transaction.

    Filters portfolios based on whether they have the fund/stock that matches
    the transaction (by ISIN or symbol).

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response containing:
        - match_info: Details about fund matching (found, matched_by, fund details)
        - portfolios: List of eligible portfolios with this fund
        - warning: Optional warning message if no match or no portfolios
    """
    from ..services.fund_matching_service import FundMatchingService

    try:
        # Get the transaction using the service
        transaction = IBKRTransactionService.get_transaction(transaction_id)

        # Find eligible portfolios using the matching service
        result = FundMatchingService.get_eligible_portfolios_for_transaction(transaction)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Retrieved eligible portfolios for transaction {transaction_id}",
            details={
                "fund_found": result["match_info"]["found"],
                "portfolio_count": len(result["portfolios"]),
                "matched_by": result["match_info"]["matched_by"],
            },
        )

        return jsonify(result), 200

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message=f"Error getting eligible portfolios: {e!s}",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@ibkr.route("/ibkr/inbox/<transaction_id>/allocate", methods=["POST"])
def allocate_transaction(transaction_id):
    """
    Process IBKR transaction with allocations.

    Request body:
        {
            "allocations": [
                {
                    "portfolio_id": "string",
                    "percentage": number
                },
                ...
            ]
        }

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response with processing results
    """
    from ..services.ibkr_transaction_service import IBKRTransactionService

    data = request.get_json()

    if not data or "allocations" not in data:
        return jsonify({"error": "Missing allocations"}), 400

    result = IBKRTransactionService.process_transaction_allocation(
        transaction_id, data["allocations"]
    )

    if result["success"]:
        return jsonify(result), 200
    else:
        return jsonify(result), 400


@ibkr.route("/ibkr/dividends/pending", methods=["GET"])
def get_pending_dividends():
    """
    Get pending dividend records for matching.

    Query parameters:
        symbol: Filter by symbol (optional)
        isin: Filter by ISIN (optional)

    Returns:
        JSON array of pending dividends
    """
    from ..services.ibkr_transaction_service import IBKRTransactionService

    symbol = request.args.get("symbol")
    isin = request.args.get("isin")

    dividends = IBKRTransactionService.get_pending_dividends(symbol, isin)

    return jsonify(dividends)


@ibkr.route("/ibkr/inbox/<transaction_id>/match-dividend", methods=["POST"])
def match_dividend(transaction_id):
    """
    Match IBKR dividend transaction to existing Dividend records.

    Request body:
        {
            "dividend_ids": ["id1", "id2", ...]
        }

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response with matching results
    """
    from ..services.ibkr_transaction_service import IBKRTransactionService

    data = request.get_json()

    if not data or "dividend_ids" not in data:
        return jsonify({"error": "Missing dividend_ids"}), 400

    result = IBKRTransactionService.match_dividend(transaction_id, data["dividend_ids"])

    if result["success"]:
        return jsonify(result), 200
    else:
        return jsonify(result), 400


@ibkr.route("/ibkr/inbox/<transaction_id>/unallocate", methods=["POST"])
@track_request
def unallocate_transaction(transaction_id):
    """
    Unallocate a processed IBKR transaction.

    This deletes all portfolio transactions and allocations,
    reverting the IBKR transaction status to pending.

    Args:
        transaction_id: IBKR Transaction ID

    Returns:
        JSON response with success status
    """
    response, status = IBKRTransactionService.unallocate_transaction(transaction_id)
    return jsonify(response), status


@ibkr.route("/ibkr/inbox/<transaction_id>/allocations", methods=["GET"])
@track_request
def get_transaction_allocations(transaction_id):
    """
    Get allocation details for a processed IBKR transaction.

    Groups allocations by portfolio to combine stock and fee transactions.

    Args:
        transaction_id: IBKR Transaction ID

    Returns:
        JSON response containing allocation details grouped by portfolio
    """
    response, status = IBKRTransactionService.get_transaction_allocations(transaction_id)
    return jsonify(response), status


@ibkr.route("/ibkr/inbox/<transaction_id>/allocations", methods=["PUT"])
@track_request
def modify_transaction_allocations(transaction_id):
    """
    Modify allocation percentages for a processed IBKR transaction.

    Request body:
        {
            "allocations": [
                {
                    "portfolio_id": "string",
                    "percentage": number
                },
                ...
            ]
        }

    Args:
        transaction_id: IBKR Transaction ID

    Returns:
        JSON response with success status
    """
    data = request.get_json()

    if not data or "allocations" not in data:
        return jsonify({"error": "Missing allocations"}), 400

    try:
        result = IBKRTransactionService.modify_allocations(transaction_id, data["allocations"])
        return jsonify(result), 200

    except ValueError as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Failed to modify allocations",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=400,
        )
        return jsonify(response), status
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Unexpected error modifying allocations",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@ibkr.route("/ibkr/inbox/bulk-allocate", methods=["POST"])
@track_request
def bulk_allocate_transactions():
    """
    Process multiple IBKR transactions with the same allocations.

    Request body:
        {
            "transaction_ids": ["id1", "id2", ...],
            "allocations": [
                {
                    "portfolio_id": "string",
                    "percentage": number
                },
                ...
            ]
        }

    Returns:
        JSON response with bulk processing results
    """
    data = request.get_json()

    if not data or "transaction_ids" not in data or "allocations" not in data:
        return jsonify({"error": "Missing transaction_ids or allocations"}), 400

    transaction_ids = data["transaction_ids"]
    allocations = data["allocations"]

    result = IBKRTransactionService.bulk_allocate_transactions(transaction_ids, allocations)

    if result["success"]:
        return jsonify(result), 200
    else:
        # If validation failed, return 400
        return jsonify(result), 400


# Register view class
ibkr_config_view = IBKRConfigAPI.as_view("ibkr_config")
ibkr.add_url_rule("/ibkr/config", view_func=ibkr_config_view, methods=["GET", "POST", "DELETE"])
