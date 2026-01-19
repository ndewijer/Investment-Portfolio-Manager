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

# Define models (using camelCase for API responses)
transaction_model = ns.model(
    "Transaction",
    {
        "id": fields.String(required=True, description="Transaction unique identifier (UUID)"),
        "portfolioFundId": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "fundName": fields.String(required=True, description="Fund name"),
        "date": fields.String(required=True, description="Transaction date (YYYY-MM-DD)"),
        "type": fields.String(
            required=True, description="Transaction type", enum=["buy", "sell", "dividend", "fee"]
        ),
        "shares": fields.Float(required=True, description="Number of shares"),
        "costPerShare": fields.Float(required=True, description="Cost per share"),
        "ibkrLinked": fields.Boolean(description="Whether transaction is linked to IBKR"),
        "ibkrTransactionId": fields.String(description="IBKR transaction ID if linked"),
        "realizedGainLoss": fields.Float(description="Realized gain/loss (sell transactions only)"),
    },
)

transaction_create_model = ns.model(
    "TransactionCreate",
    {
        "portfolioFundId": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "date": fields.String(
            required=True, description="Transaction date (YYYY-MM-DD)", example="2024-01-15"
        ),
        "type": fields.String(
            required=True, description="Transaction type", enum=["buy", "sell", "dividend", "fee"]
        ),
        "shares": fields.Float(required=True, description="Number of shares", example=10.5),
        "costPerShare": fields.Float(required=True, description="Cost per share", example=150.25),
    },
)

error_model = ns.model(
    "Error", {"error": fields.String(required=True, description="Error message")}
)


@ns.route("")
class TransactionList(Resource):
    """Transaction collection endpoint."""

    @ns.doc("list_transactions")
    @ns.response(200, "Success", [transaction_model])
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get all transactions.

        Returns a list of all transactions across all portfolios.
        Use /transaction/portfolio/<portfolio_id> for portfolio-specific transactions.

        Returns transaction details including realized gain/loss for sell transactions.
        """
        try:
            service = TransactionService()
            transactions = service.get_all_transactions()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.TRANSACTION,
                message="Successfully retrieved all transactions",
                details={
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
                details={"error": str(e)},
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

        Request body should use camelCase (portfolioFundId, costPerShare).
        """
        try:
            data = request.json

            # Convert camelCase input to snake_case for service layer
            service_data = {
                "portfolio_fund_id": data.get("portfolioFundId"),
                "date": data.get("date"),
                "type": data.get("type"),
                "shares": data.get("shares"),
                "cost_per_share": data.get("costPerShare"),
            }

            service = TransactionService()
            transaction = service.create_transaction(service_data)

            # Format the transaction response (already in camelCase from service)
            response = service.format_transaction(transaction)
            if service_data["type"] == "sell":
                # Add realized gain/loss info to response for sell transactions
                realized_records = RealizedGainLoss.query.filter_by(
                    transaction_id=transaction.id
                ).first()

                if realized_records:
                    response["realizedGainLoss"] = realized_records.realized_gain_loss

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


@ns.route("/portfolio/<string:portfolio_id>")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class PortfolioTransactionList(Resource):
    """Portfolio transactions endpoint."""

    @ns.doc("list_portfolio_transactions")
    @ns.response(200, "Success", [transaction_model])
    @ns.response(500, "Server error", error_model)
    def get(self, portfolio_id):
        """
        Get all transactions for a specific portfolio.

        Returns all buy, sell, dividend, and fee transactions associated with
        the specified portfolio, including fund names and realized gain/loss data.

        Args:
            portfolio_id: Portfolio unique identifier

        Returns:
            List of transaction objects in camelCase format
        """
        try:
            service = TransactionService()
            transactions = service.get_portfolio_transactions(portfolio_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.TRANSACTION,
                message="Successfully retrieved portfolio transactions",
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
                message=f"Error retrieving portfolio transactions: {e!s}",
                details={"portfolio_id": portfolio_id, "error": str(e)},
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

        Request body should use camelCase (portfolioFundId, costPerShare).
        """
        try:
            data = request.json

            # Convert camelCase input to snake_case for service layer
            service_data = {
                "date": data.get("date"),
                "type": data.get("type"),
                "shares": data.get("shares"),
                "cost_per_share": data.get("costPerShare"),
            }

            service = TransactionService()
            transaction = service.update_transaction(transaction_id, service_data)

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
