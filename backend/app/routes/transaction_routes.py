"""
API routes for transaction management.

This module provides routes for:
- Creating and managing investment transactions
- Retrieving transaction history
- Transaction filtering by portfolio
"""

from flask import Blueprint, jsonify, request

from ..models import (
    LogCategory,
    LogLevel,
    RealizedGainLoss,
    Transaction,
)
from ..services.logging_service import logger, track_request
from ..services.transaction_service import TransactionService

transactions = Blueprint("transactions", __name__)


@transactions.route("/transactions", methods=["GET"])
@track_request
def get_transactions():
    """
    Retrieve transactions, optionally filtered by portfolio.

    Query Parameters:
        portfolio_id (str, optional): Filter transactions by portfolio
        fund_id (str, optional): Filter by fund
        start_date (str, optional): Start date in YYYY-MM-DD format
        end_date (str, optional): End date in YYYY-MM-DD format
        type (str, optional): Transaction type filter
        sort_by (str, optional): Sort field
        sort_order (str, optional): Sort direction ('asc' or 'desc')

    Returns:
        JSON response containing:
        - List of transactions
        - Total transaction count
        - Aggregated statistics
    """
    try:
        portfolio_id = request.args.get("portfolio_id")
        service = TransactionService()

        if portfolio_id:
            transactions = service.get_portfolio_transactions(portfolio_id)
        else:
            transactions = service.get_all_transactions()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message="Successfully retrieved transactions",
            details={
                "portfolio_id": portfolio_id,
                "transaction_count": len(transactions),
            },
        )

        return jsonify(transactions)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error retrieving transactions: {e!s}",
            details={"portfolio_id": portfolio_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@transactions.route("/transactions", methods=["POST"])
def create_transaction():
    """
    Create a new transaction.

    Request Body:
        portfolio_fund_id (str): Portfolio-Fund relationship identifier
        date (str): Transaction date in YYYY-MM-DD format
        type (str): Transaction type ('buy', 'sell', or 'dividend')
        shares (float): Number of shares
        cost_per_share (float): Cost per share

    Returns:
        JSON response containing created transaction details
    """
    try:
        data = request.json
        service = TransactionService()
        transaction = service.create_transaction(data)

        # Format the transaction response
        response = service.format_transaction(transaction)
        if data["type"] == "sell":
            # Add realized gain/loss info to response for sell transactions
            realized_records = RealizedGainLoss.query.filter_by(
                transaction_id=transaction.id
            ).first()

            if realized_records:
                response["realized_gain_loss"] = realized_records.realized_gain_loss

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully created transaction",
            details={
                "transaction_id": transaction.id,
                "type": transaction.type,
                "shares": transaction.shares,
                "cost_per_share": transaction.cost_per_share,
            },
        )

        return jsonify(response)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error creating transaction: {e!s}",
            details={
                "user_message": "Error creating transaction",
                "error": str(e),
                "request_data": data,
            },
            http_status=500,
        )
        return jsonify(response), status


@transactions.route("/transactions/<string:transaction_id>", methods=["GET"])
@track_request
def get_transaction(transaction_id):
    """
    Retrieve a specific transaction.

    Args:
        transaction_id (str): Transaction identifier

    Returns:
        JSON response containing transaction details
    """
    try:
        transaction = Transaction.query.get_or_404(transaction_id)
        service = TransactionService()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully retrieved transaction {transaction_id}",
            details={"transaction_type": transaction.type},
        )

        return jsonify(service.format_transaction(transaction))
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error retrieving transaction: {e!s}",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=404,
        )
        return jsonify(response), status


@transactions.route("/transactions/<string:transaction_id>", methods=["PUT"])
@track_request
def update_transaction(transaction_id):
    """
    Update an existing transaction.

    Args:
        transaction_id (str): Transaction identifier

    Request Body:
        date (str): Transaction date in YYYY-MM-DD format
        type (str): Transaction type ('buy', 'sell', or 'dividend')
        shares (float): Number of shares
        cost_per_share (float): Cost per share

    Returns:
        JSON response containing updated transaction details
    """
    try:
        data = request.json
        service = TransactionService()
        transaction = service.update_transaction(transaction_id, data)

        # Format the transaction response
        response = service.format_transaction(transaction)
        if transaction.type == "sell":
            # Add realized gain/loss info to response
            portfolio_fund = transaction.portfolio_fund
            realized_records = (
                RealizedGainLoss.query.filter_by(
                    portfolio_id=portfolio_fund.portfolio_id,
                    fund_id=portfolio_fund.fund_id,
                    transaction_date=transaction.date,
                )
                .order_by(RealizedGainLoss.created_at.desc())
                .first()
            )

            if realized_records:
                response["realized_gain_loss"] = realized_records.realized_gain_loss

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully updated transaction {transaction_id}",
            details={
                "type": transaction.type,
                "shares": transaction.shares,
                "cost_per_share": transaction.cost_per_share,
            },
        )

        return jsonify(response)
    except ValueError as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error updating transaction: {e!s}",
            details={
                "user_message": str(e),
                "transaction_id": transaction_id,
                "error": str(e),
                "request_data": data,
            },
            http_status=400,
        )
        return jsonify(response), status


@transactions.route("/transactions/<string:transaction_id>", methods=["DELETE"])
@track_request
def delete_transaction(transaction_id):
    """
    Delete a transaction.

    If the transaction is linked to an IBKR allocation and this is the last
    allocation for that IBKR transaction, the IBKR transaction status will
    automatically revert to 'pending'.

    Args:
        transaction_id (str): Transaction identifier

    Returns:
        Empty response with 204 status on success
    """
    try:
        service = TransactionService()
        result = service.delete_transaction(transaction_id)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully deleted transaction {transaction_id}",
            details={
                **result["transaction_details"],
                "ibkr_transaction_id": result["ibkr_transaction_id"],
                "ibkr_reverted": result["ibkr_reverted"],
                "realized_gain_deleted": result["realized_gain_deleted"],
            },
        )

        return "", 204
    except ValueError as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error deleting transaction: {e!s}",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=400,
        )
        return jsonify(response), status
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Unexpected error deleting transaction: {e!s}",
            details={"transaction_id": transaction_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status
