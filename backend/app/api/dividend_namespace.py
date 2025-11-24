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
ns = Namespace("dividends", description="Dividend management operations")

# Define models
dividend_model = ns.model(
    "Dividend",
    {
        "id": fields.String(required=True, description="Dividend unique identifier (UUID)"),
        "fund_id": fields.String(required=True, description="Fund ID"),
        "portfolio_fund_id": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "record_date": fields.String(required=True, description="Record date (YYYY-MM-DD)"),
        "ex_dividend_date": fields.String(
            required=True, description="Ex-dividend date (YYYY-MM-DD)"
        ),
        "shares_owned": fields.Float(required=True, description="Shares owned on record date"),
        "dividend_per_share": fields.Float(required=True, description="Dividend amount per share"),
        "total_amount": fields.Float(required=True, description="Total dividend amount"),
        "reinvestment_status": fields.String(
            required=True,
            description="Reinvestment status",
            enum=["pending", "completed", "partial"],
        ),
        "buy_order_date": fields.String(description="Buy order date for reinvestment"),
    },
)

dividend_create_model = ns.model(
    "DividendCreate",
    {
        "portfolio_fund_id": fields.String(
            required=True, description="Portfolio-Fund relationship ID"
        ),
        "record_date": fields.String(
            required=True, description="Record date (YYYY-MM-DD)", example="2024-01-15"
        ),
        "ex_dividend_date": fields.String(
            required=True, description="Ex-dividend date (YYYY-MM-DD)", example="2024-01-10"
        ),
        "dividend_per_share": fields.Float(
            required=True, description="Dividend per share", example=0.50
        ),
        "buy_order_date": fields.String(
            description="Buy order date for stock dividends (YYYY-MM-DD)"
        ),
        "reinvestment_shares": fields.Float(
            description="Shares from reinvestment (for stock dividends)"
        ),
        "reinvestment_price": fields.Float(description="Price per share for reinvestment"),
    },
)

dividend_update_model = ns.model(
    "DividendUpdate",
    {
        "reinvestment_shares": fields.Float(description="Shares from reinvestment"),
        "reinvestment_price": fields.Float(description="Price per share for reinvestment"),
        "buy_order_date": fields.String(description="Buy order date (YYYY-MM-DD)"),
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
        """
        try:
            data = request.json
            dividend = DividendService.create_dividend(data)

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
        """
        try:
            data = request.json
            dividend, _original_values = DividendService.update_dividend(dividend_id, data)

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
