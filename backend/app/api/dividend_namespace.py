"""
Dividend API namespace for managing dividend income and reinvestment.

This namespace provides endpoints for:
- Creating and tracking dividend payments
- Managing dividend reinvestment
- Portfolio dividend allocation
"""

from flask import request
from flask_restx import Namespace, Resource, fields
from werkzeug.exceptions import HTTPException

from ..models import LogCategory, LogLevel
from ..services.dividend_service import DividendService
from ..services.logging_service import logger

# Create namespace
ns = Namespace("dividend", description="Dividend management operations")

# Define models (using camelCase for API responses)
dividend_model = ns.model(
    "Dividend",
    {
        "id": fields.String(required=True, description="Dividend unique identifier (UUID)"),
        "fundId": fields.String(required=True, description="Fund ID"),
        "fundName": fields.String(required=True, description="Fund name"),
        "portfolioFundId": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "recordDate": fields.String(required=True, description="Record date (YYYY-MM-DD)"),
        "exDividendDate": fields.String(required=True, description="Ex-dividend date (YYYY-MM-DD)"),
        "sharesOwned": fields.Float(required=True, description="Shares owned on record date"),
        "dividendPerShare": fields.Float(required=True, description="Dividend amount per share"),
        "totalAmount": fields.Float(required=True, description="Total dividend amount"),
        "reinvestmentStatus": fields.String(
            required=True,
            description="Reinvestment status",
            enum=["PENDING", "COMPLETED", "PARTIAL"],
        ),
        "buyOrderDate": fields.String(description="Buy order date for reinvestment"),
        "reinvestmentTransactionId": fields.String(description="Reinvestment transaction ID"),
        "dividendType": fields.String(description="Dividend type", enum=["NONE", "CASH", "STOCK"]),
    },
)

dividend_create_model = ns.model(
    "DividendCreate",
    {
        "portfolioFundId": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "recordDate": fields.String(
            required=True, description="Record date (YYYY-MM-DD)", example="2024-01-15"
        ),
        "exDividendDate": fields.String(
            required=True, description="Ex-dividend date (YYYY-MM-DD)", example="2024-01-10"
        ),
        "dividendPerShare": fields.Float(
            required=True, description="Dividend per share", example=0.50
        ),
        "buyOrderDate": fields.String(
            description="Buy order date for stock dividends (YYYY-MM-DD)"
        ),
        "reinvestmentShares": fields.Float(
            description="Shares from reinvestment (for stock dividends)"
        ),
        "reinvestmentPrice": fields.Float(description="Price per share for reinvestment"),
    },
)

dividend_update_model = ns.model(
    "DividendUpdate",
    {
        "recordDate": fields.String(required=True, description="Record date (YYYY-MM-DD)"),
        "exDividendDate": fields.String(required=True, description="Ex-dividend date (YYYY-MM-DD)"),
        "dividendPerShare": fields.Float(required=True, description="Dividend amount per share"),
        "reinvestmentShares": fields.Float(description="Shares from reinvestment"),
        "reinvestmentPrice": fields.Float(description="Price per share for reinvestment"),
        "buyOrderDate": fields.String(description="Buy order date (YYYY-MM-DD)"),
    },
)

error_model = ns.model(
    "Error",
    {
        "error": fields.String(required=True, description="Error message"),
        "details": fields.String(description="Additional error details"),
    },
)


@ns.route("")
class DividendList(Resource):
    """Dividend collection endpoint."""

    @ns.doc("create_dividend")
    @ns.expect(dividend_create_model, validate=True)
    @ns.response(201, "Dividend created", dividend_model)
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Create a new dividend record.

        Creates a dividend record for a fund in a portfolio.
        Automatically calculates:
        - Shares owned on record date (based on transaction history)
        - Total dividend amount (shares * dividend per share)
        - Initial reinvestment status based on dividend type

        For stock dividends, optionally include reinvestment details.

        Request body should use camelCase (portfolioFundId, recordDate, etc.).
        """
        try:
            data = request.json

            # Convert camelCase input to snake_case for service layer
            service_data = {
                "portfolio_fund_id": data.get("portfolioFundId"),
                "record_date": data.get("recordDate"),
                "ex_dividend_date": data.get("exDividendDate"),
                "dividend_per_share": data.get("dividendPerShare"),
                "buy_order_date": data.get("buyOrderDate"),
                "reinvestment_shares": data.get("reinvestmentShares"),
                "reinvestment_price": data.get("reinvestmentPrice"),
            }

            dividend = DividendService.create_dividend(service_data)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DIVIDEND,
                message=f"Successfully created dividend",
                details={
                    "dividend_id": dividend.id,
                    "fund_name": dividend.fund.name,
                    "total_amount": dividend.total_amount,
                    "dividend_type": dividend.fund.dividend_type.value,
                },
            )

            return DividendService.format_dividend(dividend), 201

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DIVIDEND,
                message=f"Error creating dividend: {e!s}",
                details={
                    "user_message": "Error creating dividend",
                    "error": str(e),
                    "request_data": data,
                },
            )
            return {"error": "Error creating dividend", "details": str(e)}, 500


@ns.route("/fund/<string:fund_id>")
@ns.param("fund_id", "Fund unique identifier (UUID)")
class FundDividendList(Resource):
    """Fund dividend list endpoint."""

    @ns.doc("get_fund_dividends")
    @ns.response(200, "Success", [dividend_model])
    @ns.response(500, "Server error", error_model)
    def get(self, fund_id):
        """
        Get all dividends for a specific fund.

        Returns all dividend payments for a fund across all portfolios.
        Useful for tracking fund dividend history and reinvestment status.
        """
        try:
            dividends = DividendService.get_fund_dividends(fund_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DIVIDEND,
                message=f"Successfully retrieved fund dividends",
                details={"fund_id": fund_id, "dividend_count": len(dividends)},
            )

            return [DividendService.format_dividend(d) for d in dividends], 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DIVIDEND,
                message=f"Error retrieving fund dividends: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
            )
            return {"error": "Error retrieving dividends", "details": str(e)}, 500


@ns.route("/portfolio/<string:portfolio_id>")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class PortfolioDividendList(Resource):
    """Portfolio dividend list endpoint."""

    @ns.doc("get_portfolio_dividends")
    @ns.response(200, "Success", [dividend_model])
    @ns.response(500, "Server error", error_model)
    def get(self, portfolio_id):
        """
        Get all dividends for a specific portfolio.

        Returns all dividend payments across all funds in a portfolio.
        Useful for tracking portfolio dividend income and tax reporting.
        """
        try:
            dividends = DividendService.get_portfolio_dividends(portfolio_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DIVIDEND,
                message=f"Successfully retrieved portfolio dividends",
                details={"portfolio_id": portfolio_id, "dividend_count": len(dividends)},
            )

            return [DividendService.format_dividend(d) for d in dividends], 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DIVIDEND,
                message=f"Error retrieving portfolio dividends: {e!s}",
                details={"portfolio_id": portfolio_id, "error": str(e)},
            )
            return {"error": "Error retrieving dividends", "details": str(e)}, 500


@ns.route("/<string:dividend_id>")
@ns.param("dividend_id", "Dividend unique identifier (UUID)")
class Dividend(Resource):
    """Dividend detail endpoint."""

    @ns.doc("get_dividend")
    @ns.response(200, "Success", dividend_model)
    @ns.response(404, "Dividend not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, dividend_id):
        """
        Get dividend details.

        Returns detailed information about a specific dividend payment,
        including reinvestment status and transaction details.
        """
        try:
            dividend = DividendService.get_dividend(dividend_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DIVIDEND,
                message=f"Successfully retrieved dividend {dividend_id}",
                details={"dividend_id": dividend_id},
            )

            return DividendService.format_dividend(dividend), 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DIVIDEND,
                message=f"Error retrieving dividend: {e!s}",
                details={"dividend_id": dividend_id, "error": str(e)},
            )
            return {"error": "Error retrieving dividend", "details": str(e)}, 500

    @ns.doc("update_dividend")
    @ns.expect(dividend_update_model, validate=True)
    @ns.response(200, "Dividend updated", dividend_model)
    @ns.response(404, "Dividend not found", error_model)
    @ns.response(500, "Server error", error_model)
    def put(self, dividend_id):
        """
        Update dividend reinvestment details.

        Updates reinvestment information for a dividend, typically used
        when processing stock dividend reinvestment.

        Request body should use camelCase (recordDate, exDividendDate, dividendPerShare, etc.).
        """
        try:
            data = request.json

            # Convert camelCase input to snake_case for service layer
            service_data = {
                "record_date": data.get("recordDate"),
                "ex_dividend_date": data.get("exDividendDate"),
                "dividend_per_share": data.get("dividendPerShare"),
                "reinvestment_shares": data.get("reinvestmentShares"),
                "reinvestment_price": data.get("reinvestmentPrice"),
                "buy_order_date": data.get("buyOrderDate"),
            }

            dividend, _original_values = DividendService.update_dividend(dividend_id, service_data)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DIVIDEND,
                message=f"Successfully updated dividend {dividend_id}",
                details={"dividend_id": dividend_id},
            )

            return DividendService.format_dividend(dividend), 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DIVIDEND,
                message=f"Error updating dividend: {e!s}",
                details={"dividend_id": dividend_id, "error": str(e)},
            )
            return {"error": "Error updating dividend", "details": str(e)}, 500

    @ns.doc("delete_dividend")
    @ns.response(200, "Dividend deleted")
    @ns.response(404, "Dividend not found", error_model)
    @ns.response(500, "Server error", error_model)
    def delete(self, dividend_id):
        """
        Delete a dividend record.

        Permanently deletes a dividend record and associated reinvestment transactions.
        This operation cannot be undone.
        """
        try:
            DividendService.delete_dividend(dividend_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.DIVIDEND,
                message=f"Successfully deleted dividend {dividend_id}",
                details={"dividend_id": dividend_id},
            )

            return {"success": True}, 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.DIVIDEND,
                message=f"Error deleting dividend: {e!s}",
                details={"dividend_id": dividend_id, "error": str(e)},
            )
            return {"error": "Error deleting dividend", "details": str(e)}, 500
