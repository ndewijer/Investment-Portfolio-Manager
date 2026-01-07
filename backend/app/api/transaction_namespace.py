"""
Transaction API namespace for managing investment transactions.

This namespace provides endpoints for:
- Creating and retrieving transactions (buy, sell, dividend)
- Transaction history and filtering
- Realized gain/loss tracking
"""

from flask import request
from flask_restx import Namespace, Resource, fields
from werkzeug.exceptions import HTTPException

from ..models import LogCategory, LogLevel, RealizedGainLoss
from ..services.logging_service import logger
from ..services.transaction_service import TransactionService

# Create namespace
ns = Namespace("transaction", description="Transaction management operations")

# Define models
transaction_model = ns.model(
    "Transaction",
    {
        "id": fields.String(required=True, description="Transaction unique identifier (UUID)"),
        "portfolio_fund_id": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "date": fields.String(required=True, description="Transaction date (YYYY-MM-DD)"),
        "type": fields.String(
            required=True, description="Transaction type", enum=["buy", "sell", "dividend", "fee"]
        ),
        "shares": fields.Float(required=True, description="Number of shares"),
        "cost_per_share": fields.Float(required=True, description="Cost per share"),
        "created_at": fields.DateTime(description="Creation timestamp"),
    },
)

transaction_create_model = ns.model(
    "TransactionCreate",
    {
        "portfolio_fund_id": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "date": fields.String(
            required=True, description="Transaction date (YYYY-MM-DD)", example="2024-01-15"
        ),
        "type": fields.String(
            required=True, description="Transaction type", enum=["buy", "sell", "dividend", "fee"]
        ),
        "shares": fields.Float(required=True, description="Number of shares", example=10.5),
        "cost_per_share": fields.Float(required=True, description="Cost per share", example=150.25),
    },
)

error_model = ns.model(
    "Error", {"error": fields.String(required=True, description="Error message")}
)


@ns.route("")
class TransactionList(Resource):
    """Transaction collection endpoint."""

    @ns.doc("list_transactions")
    @ns.param("portfolio_id", "Filter by portfolio ID", _in="query")
    @ns.param("fund_id", "Filter by fund ID", _in="query")
    @ns.param("start_date", "Start date (YYYY-MM-DD)", _in="query")
    @ns.param("end_date", "End date (YYYY-MM-DD)", _in="query")
    @ns.param("type", "Transaction type", _in="query")
    @ns.response(200, "Success", [transaction_model])
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get all transactions with optional filters.

        Retrieves a list of all transactions, optionally filtered by:
        - Portfolio ID
        - Fund ID
        - Date range
        - Transaction type

        Returns transaction details including realized gain/loss for sell transactions.
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

            return transactions, 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=f"Error retrieving transactions: {e!s}",
                details={"portfolio_id": portfolio_id, "error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc("create_transaction")
    @ns.expect(transaction_create_model, validate=True)
    @ns.response(201, "Transaction created", transaction_model)
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Create a new transaction.

        Creates a transaction record for buying, selling, or receiving dividends.

        For sell transactions, the system automatically calculates realized gain/loss
        using FIFO (First In, First Out) accounting method.
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

            return response, 201
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=f"Error creating transaction: {e!s}",
                details={
                    "user_message": "Error creating transaction",
                    "error": str(e),
                    "request_data": data,
                },
            )
            return {"error": str(e)}, 500


@ns.route("/<string:transaction_id>")
@ns.param("transaction_id", "Transaction unique identifier (UUID)")
class Transaction(Resource):
    """Transaction detail endpoint."""

    @ns.doc("get_transaction")
    @ns.response(200, "Success", transaction_model)
    @ns.response(404, "Transaction not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, transaction_id):
        """
        Get transaction details.

        Returns detailed information about a specific transaction.
        """
        try:
            transaction = TransactionService.get_transaction(transaction_id)
            service = TransactionService()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.TRANSACTION,
                message=f"Successfully retrieved transaction {transaction_id}",
                details={"transaction_id": transaction_id},
            )

            return service.format_transaction(transaction), 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=f"Error retrieving transaction: {e!s}",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": str(e)}, 404

    @ns.doc("update_transaction")
    @ns.expect(transaction_create_model, validate=True)
    @ns.response(200, "Transaction updated", transaction_model)
    @ns.response(404, "Transaction not found", error_model)
    @ns.response(500, "Server error", error_model)
    def put(self, transaction_id):
        """
        Update a transaction.

        Updates the details of an existing transaction.
        For sell transactions, this recalculates realized gain/loss.

        Note: Modifying transactions may affect historical calculations
        and should be done with caution.
        """
        try:
            data = request.json
            service = TransactionService()
            transaction = service.update_transaction(transaction_id, data)

            # Format the transaction response with realized gain/loss for sell transactions
            response = service.format_transaction(
                transaction, include_realized_gain=(transaction.type == "sell")
            )

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

            return response, 200
        except ValueError as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=f"Error updating transaction: {e!s}",
                details={
                    "user_message": str(e),
                    "transaction_id": transaction_id,
                    "error": str(e),
                    "request_data": data,
                },
            )
            return {"error": str(e)}, 400
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=f"Error updating transaction: {e!s}",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc("delete_transaction")
    @ns.response(200, "Transaction deleted")
    @ns.response(404, "Transaction not found", error_model)
    @ns.response(500, "Server error", error_model)
    def delete(self, transaction_id):
        """
        Delete a transaction.

        Permanently deletes a transaction. For sell transactions,
        this also removes associated realized gain/loss records.

        This operation cannot be undone.
        """
        try:
            service = TransactionService()
            service.delete_transaction(transaction_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.TRANSACTION,
                message=f"Successfully deleted transaction {transaction_id}",
                details={"transaction_id": transaction_id},
            )

            return {"success": True}, 200
        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=f"Error deleting transaction: {e!s}",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": str(e)}, 500
