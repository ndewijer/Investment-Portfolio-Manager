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

from ..models import IBKRConfig, IBKRTransaction, LogCategory, LogLevel, Portfolio, db
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
        config = IBKRConfig.query.first()

        if not config:
            return jsonify({"configured": False}), 200

        # Check if token is expiring soon (within 30 days)
        token_warning = None
        if config.token_expires_at:
            # SQLite stores naive datetimes, so compare with naive datetime
            days_until_expiry = (config.token_expires_at - datetime.now()).days
            if days_until_expiry < 30:
                token_warning = f"Token expires in {days_until_expiry} days"

        return jsonify(
            {
                "configured": True,
                "flex_query_id": config.flex_query_id,
                "token_expires_at": (
                    config.token_expires_at.isoformat() if config.token_expires_at else None
                ),
                "token_warning": token_warning,
                "last_import_date": (
                    config.last_import_date.isoformat() if config.last_import_date else None
                ),
                "auto_import_enabled": config.auto_import_enabled,
                "enabled": config.enabled,
                "created_at": config.created_at.isoformat(),
                "updated_at": config.updated_at.isoformat(),
            }
        )

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
            service = IBKRFlexService()

            # Encrypt the token
            encrypted_token = service._encrypt_token(data["flex_token"])

            # Parse token expiration date if provided
            token_expires_at = None
            if data.get("token_expires_at"):
                try:
                    token_expires_at = datetime.fromisoformat(data["token_expires_at"])
                except (ValueError, TypeError):
                    return jsonify({"error": "Invalid token_expires_at format"}), 400

            # Get or create config
            config = IBKRConfig.query.first()

            if config:
                # Update existing
                config.flex_token = encrypted_token
                config.flex_query_id = data["flex_query_id"]
                if token_expires_at:
                    config.token_expires_at = token_expires_at
                if "auto_import_enabled" in data:
                    config.auto_import_enabled = data["auto_import_enabled"]
                if "enabled" in data:
                    config.enabled = data["enabled"]
                config.updated_at = datetime.now()
            else:
                # Create new
                config = IBKRConfig(
                    flex_token=encrypted_token,
                    flex_query_id=data["flex_query_id"],
                    token_expires_at=token_expires_at,
                    auto_import_enabled=data.get("auto_import_enabled", False),
                    enabled=data.get("enabled", True),
                )
                db.session.add(config)

            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR configuration saved",
                details={"query_id": data["flex_query_id"]},
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
        config = IBKRConfig.query.first()

        if not config:
            return jsonify({"error": "No configuration found"}), 404

        try:
            db.session.delete(config)
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR configuration deleted",
                details={},
            )

            return jsonify({"success": True, "message": "Configuration deleted successfully"}), 200

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
    config = IBKRConfig.query.first()

    if not config:
        return jsonify({"error": "IBKR not configured"}), 400

    if not config.enabled:
        return jsonify({"error": "IBKR integration is disabled"}), 403

    try:
        service = IBKRFlexService()

        # Decrypt token
        token = service._decrypt_token(config.flex_token)

        # Fetch statement
        xml_data = service.fetch_statement(token, config.flex_query_id, use_cache=True)

        if not xml_data:
            return jsonify({"error": "Failed to fetch statement from IBKR"}), 500

        # Parse transactions
        transactions = service.parse_flex_statement(xml_data)

        # Import transactions
        results = service.import_transactions(transactions)

        # Update last import date
        config.last_import_date = datetime.now()
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message="Manual IBKR import completed",
            details=results,
        )

        return jsonify(
            {
                "success": True,
                "message": "Import completed",
                "imported": results["imported"],
                "skipped": results["skipped"],
                "errors": results["errors"],
            }
        ), 200

    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Error during IBKR import",
            details={"error": str(e)},
        )
        return jsonify({"error": "Import failed", "details": str(e)}), 500


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
    query = IBKRTransaction.query

    # Apply filters
    status = request.args.get("status", "pending")
    if status:
        query = query.filter_by(status=status)

    transaction_type = request.args.get("transaction_type")
    if transaction_type:
        query = query.filter_by(transaction_type=transaction_type)

    # Order by date descending
    transactions = query.order_by(IBKRTransaction.transaction_date.desc()).all()

    return jsonify(
        [
            {
                "id": txn.id,
                "ibkr_transaction_id": txn.ibkr_transaction_id,
                "transaction_date": txn.transaction_date.isoformat(),
                "symbol": txn.symbol,
                "isin": txn.isin,
                "description": txn.description,
                "transaction_type": txn.transaction_type,
                "quantity": txn.quantity,
                "price": txn.price,
                "total_amount": txn.total_amount,
                "currency": txn.currency,
                "fees": txn.fees,
                "status": txn.status,
                "imported_at": txn.imported_at.isoformat(),
            }
            for txn in transactions
        ]
    )


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
        count = IBKRTransaction.query.filter_by(status=status).count()

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
    txn = IBKRTransaction.query.get_or_404(transaction_id)

    return jsonify(
        {
            "id": txn.id,
            "ibkr_transaction_id": txn.ibkr_transaction_id,
            "transaction_date": txn.transaction_date.isoformat(),
            "symbol": txn.symbol,
            "isin": txn.isin,
            "description": txn.description,
            "transaction_type": txn.transaction_type,
            "quantity": txn.quantity,
            "price": txn.price,
            "total_amount": txn.total_amount,
            "currency": txn.currency,
            "fees": txn.fees,
            "status": txn.status,
            "imported_at": txn.imported_at.isoformat(),
            "processed_at": txn.processed_at.isoformat() if txn.processed_at else None,
            "allocations": [
                {
                    "id": alloc.id,
                    "portfolio_id": alloc.portfolio_id,
                    "allocation_percentage": alloc.allocation_percentage,
                    "allocated_amount": alloc.allocated_amount,
                    "allocated_shares": alloc.allocated_shares,
                    "transaction_id": alloc.transaction_id,
                }
                for alloc in txn.allocations
            ],
        }
    )


@ibkr.route("/ibkr/inbox/<transaction_id>/ignore", methods=["POST"])
def ignore_transaction(transaction_id):
    """
    Mark IBKR transaction as ignored.

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response with success status
    """
    txn = IBKRTransaction.query.get_or_404(transaction_id)

    if txn.status == "processed":
        return jsonify({"error": "Cannot ignore processed transaction"}), 400

    try:
        txn.status = "ignored"
        txn.processed_at = datetime.now()
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Transaction marked as ignored: {txn.ibkr_transaction_id}",
            details={"transaction_id": transaction_id},
        )

        return jsonify({"success": True, "message": "Transaction marked as ignored"}), 200

    except Exception as e:
        db.session.rollback()
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Failed to ignore transaction",
            details={"transaction_id": transaction_id, "error": str(e)},
        )
        return jsonify({"error": "Failed to ignore transaction", "details": str(e)}), 500


@ibkr.route("/ibkr/inbox/<transaction_id>", methods=["DELETE"])
def delete_transaction(transaction_id):
    """
    Delete IBKR transaction (only if not processed).

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response with success status
    """
    txn = IBKRTransaction.query.get_or_404(transaction_id)

    if txn.status == "processed":
        return jsonify({"error": "Cannot delete processed transaction"}), 400

    try:
        db.session.delete(txn)
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Transaction deleted: {txn.ibkr_transaction_id}",
            details={"transaction_id": transaction_id},
        )

        return jsonify({"success": True, "message": "Transaction deleted"}), 200

    except Exception as e:
        db.session.rollback()
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Failed to delete transaction",
            details={"transaction_id": transaction_id, "error": str(e)},
        )
        return jsonify({"error": "Failed to delete transaction", "details": str(e)}), 500


@ibkr.route("/ibkr/portfolios", methods=["GET"])
def get_portfolios_for_allocation():
    """
    Get available portfolios for transaction allocation.

    Returns:
        JSON array of active portfolios
    """
    portfolios = Portfolio.query.filter_by(is_archived=False).all()

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
        # Get the transaction
        transaction = IBKRTransaction.query.get_or_404(transaction_id)

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
    from ..models import IBKRTransactionAllocation

    try:
        ibkr_txn = IBKRTransaction.query.get_or_404(transaction_id)

        if ibkr_txn.status != "processed":
            return jsonify({"error": "Transaction is not processed"}), 400

        # Get all allocations
        allocations = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=transaction_id
        ).all()

        deleted_count = 0

        # Delete all associated transactions - CASCADE will handle allocations
        from ..models import Transaction

        for allocation in allocations:
            if allocation.transaction_id:
                transaction = Transaction.query.get(allocation.transaction_id)
                if transaction:
                    # Delete transaction - CASCADE DELETE will automatically
                    # delete the corresponding ibkr_transaction_allocation record
                    db.session.delete(transaction)
                    deleted_count += 1

        # Revert IBKR transaction status
        ibkr_txn.status = "pending"
        ibkr_txn.processed_at = None

        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Unallocated IBKR transaction: {ibkr_txn.ibkr_transaction_id}",
            details={
                "ibkr_transaction_id": ibkr_txn.ibkr_transaction_id,
                "deleted_transactions": deleted_count,
            },
        )

        return (
            jsonify(
                {
                    "success": True,
                    "message": (
                        f"Transaction unallocated successfully. "
                        f"{deleted_count} portfolio transactions deleted."
                    ),
                }
            ),
            200,
        )

    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Failed to unallocate transaction",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@ibkr.route("/ibkr/inbox/<transaction_id>/allocations", methods=["GET"])
@track_request
def get_transaction_allocations(transaction_id):
    """
    Get allocation details for a processed IBKR transaction.

    Args:
        transaction_id: IBKR Transaction ID

    Returns:
        JSON response containing allocation details
    """
    from ..models import IBKRTransactionAllocation, Transaction

    try:
        ibkr_txn = IBKRTransaction.query.get_or_404(transaction_id)

        # Get all allocations
        allocations = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=transaction_id
        ).all()

        allocation_details = []
        for allocation in allocations:
            transaction = (
                Transaction.query.get(allocation.transaction_id)
                if allocation.transaction_id
                else None
            )

            allocation_details.append(
                {
                    "id": allocation.id,
                    "portfolio_id": allocation.portfolio_id,
                    "portfolio_name": allocation.portfolio.name,
                    "allocation_percentage": allocation.allocation_percentage,
                    "allocated_amount": allocation.allocated_amount,
                    "allocated_shares": allocation.allocated_shares,
                    "transaction_id": allocation.transaction_id,
                    "transaction_date": transaction.date.isoformat() if transaction else None,
                }
            )

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Retrieved allocations for IBKR transaction {transaction_id}",
            details={"allocation_count": len(allocation_details)},
        )

        return (
            jsonify(
                {
                    "ibkr_transaction_id": ibkr_txn.id,
                    "status": ibkr_txn.status,
                    "allocations": allocation_details,
                }
            ),
            200,
        )

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Failed to retrieve allocations",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=500,
        )
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
    from ..services.ibkr_transaction_service import IBKRTransactionService

    data = request.get_json()

    if not data or "transaction_ids" not in data or "allocations" not in data:
        return jsonify({"error": "Missing transaction_ids or allocations"}), 400

    transaction_ids = data["transaction_ids"]
    allocations = data["allocations"]

    if not transaction_ids or len(transaction_ids) == 0:
        return jsonify({"error": "No transactions selected"}), 400

    if not allocations or len(allocations) == 0:
        return jsonify({"error": "No allocations provided"}), 400

    # Validate allocations sum to 100%
    total_percentage = sum(a.get("percentage", 0) for a in allocations)
    if abs(total_percentage - 100) > 0.01:
        return jsonify({"error": "Allocations must sum to exactly 100%"}), 400

    processed_count = 0
    failed_count = 0
    errors = []

    try:
        for transaction_id in transaction_ids:
            try:
                result = IBKRTransactionService.process_transaction_allocation(
                    transaction_id, allocations
                )

                if result["success"]:
                    processed_count += 1
                else:
                    failed_count += 1
                    errors.append(
                        {
                            "transaction_id": transaction_id,
                            "error": result.get("error", "Unknown error"),
                        }
                    )

            except Exception as e:
                failed_count += 1
                errors.append({"transaction_id": transaction_id, "error": str(e)})
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message=f"Error processing transaction {transaction_id} in bulk operation",
                    details={"transaction_id": transaction_id, "error": str(e)},
                )

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=(
                f"Bulk allocation completed: {processed_count} processed, {failed_count} failed"
            ),
            details={
                "processed": processed_count,
                "failed": failed_count,
                "total": len(transaction_ids),
            },
        )

        return (
            jsonify(
                {
                    "success": True,
                    "processed": processed_count,
                    "failed": failed_count,
                    "errors": errors if errors else None,
                }
            ),
            200,
        )

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Failed to process bulk allocation",
            details={"error": str(e), "transaction_count": len(transaction_ids)},
            http_status=500,
        )
        return jsonify(response), status


# Register view class
ibkr_config_view = IBKRConfigAPI.as_view("ibkr_config")
ibkr.add_url_rule("/ibkr/config", view_func=ibkr_config_view, methods=["GET", "POST", "DELETE"])
