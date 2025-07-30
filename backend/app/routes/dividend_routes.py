"""
API routes for dividend management.

This module provides routes for:
- Creating and managing dividend records
- Retrieving dividend information by fund or portfolio
- Managing dividend reinvestment
"""

import uuid
from datetime import datetime

from flask import Blueprint, jsonify, request

from ..models import (
    Dividend,
    DividendType,
    LogCategory,
    LogLevel,
    ReinvestmentStatus,
    Transaction,
    db,
)
from ..services.dividend_service import DividendService
from ..services.logging_service import logger, track_request

dividends = Blueprint("dividends", __name__)


@dividends.route("/dividends", methods=["POST"])
@track_request
def create_dividend():
    """
    Create a new dividend record.

    Request Body:
        fund_id (str): Fund identifier
        portfolio_fund_id (str): Portfolio-Fund relationship identifier
        record_date (str): Record date in YYYY-MM-DD format
        ex_dividend_date (str): Ex-dividend date in YYYY-MM-DD format
        shares_owned (float): Number of shares owned
        dividend_per_share (float): Dividend amount per share

    Returns:
        JSON response containing created dividend details
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

        return jsonify(DividendService.format_dividend(dividend))
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.DIVIDEND,
            message=f"Error creating dividend: {str(e)}",
            details={
                "user_message": "Error creating dividend",
                "error": str(e),
                "request_data": data,
            },
            http_status=400,
        )
        return jsonify(response), status


@dividends.route("/dividends/fund/<string:fund_id>", methods=["GET"])
@track_request
def get_fund_dividends(fund_id):
    """
    Retrieve all dividends for a specific fund.

    Args:
        fund_id (str): Fund identifier

    Query Parameters:
        start_date (str, optional): Start date in YYYY-MM-DD format
        end_date (str, optional): End date in YYYY-MM-DD format
        status (str, optional): Filter by reinvestment status

    Returns:
        JSON response containing:
        - List of dividends
        - Total dividend amount
        - Reinvestment statistics
    """
    try:
        dividends = DividendService.get_fund_dividends(fund_id)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.DIVIDEND,
            message=f"Successfully retrieved fund dividends",
            details={"fund_id": fund_id, "dividend_count": len(dividends)},
        )

        return jsonify([DividendService.format_dividend(d) for d in dividends])
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.DIVIDEND,
            message=f"Error retrieving fund dividends: {str(e)}",
            details={"fund_id": fund_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@dividends.route("/dividends/portfolio/<string:portfolio_id>", methods=["GET"])
@track_request
def get_portfolio_dividends(portfolio_id):
    """
    Retrieve all dividends for a specific portfolio.

    Args:
        portfolio_id (str): Portfolio identifier

    Returns:
        JSON response containing list of portfolio's dividends
    """
    try:
        dividends = DividendService.get_portfolio_dividends(portfolio_id)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.DIVIDEND,
            message=f"Successfully retrieved portfolio dividends",
            details={"portfolio_id": portfolio_id, "dividend_count": len(dividends)},
        )

        return jsonify([DividendService.format_dividend(d) for d in dividends])
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.DIVIDEND,
            message=f"Error retrieving portfolio dividends: {str(e)}",
            details={"portfolio_id": portfolio_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@dividends.route("/dividends/<string:dividend_id>", methods=["PUT"])
@track_request
def update_dividend(dividend_id):
    """
    Update an existing dividend record.

    Args:
        dividend_id (str): Dividend identifier

    Request Body:
        record_date (str): Record date in YYYY-MM-DD format
        ex_dividend_date (str): Ex-dividend date in YYYY-MM-DD format
        dividend_per_share (float): Dividend amount per share
        buy_order_date (str, optional): Buy order date for reinvestment
        reinvestment_shares (float, optional): Number of shares for reinvestment
        reinvestment_price (float, optional): Price per share for reinvestment

    Returns:
        JSON response containing updated dividend details
    """
    try:
        data = request.json
        dividend = Dividend.query.get_or_404(dividend_id)

        # Store original values for logging
        original_values = {
            "record_date": dividend.record_date,
            "ex_dividend_date": dividend.ex_dividend_date,
            "dividend_per_share": dividend.dividend_per_share,
            "reinvestment_status": dividend.reinvestment_status.value,
        }

        # Update basic dividend information
        dividend.record_date = datetime.strptime(data["record_date"], "%Y-%m-%d").date()
        dividend.ex_dividend_date = datetime.strptime(data["ex_dividend_date"], "%Y-%m-%d").date()
        dividend.dividend_per_share = float(data["dividend_per_share"])

        # Update buy_order_date if provided
        if "buy_order_date" in data and data["buy_order_date"]:
            dividend.buy_order_date = datetime.strptime(data["buy_order_date"], "%Y-%m-%d").date()

        # Recalculate total amount
        dividend.total_amount = dividend.shares_owned * dividend.dividend_per_share

        # Handle stock dividend updates
        if dividend.fund.dividend_type == DividendType.STOCK:
            if data.get("reinvestment_shares") and data.get("reinvestment_price"):
                if dividend.reinvestment_transaction_id:
                    # Update existing reinvestment transaction
                    transaction = Transaction.query.get(dividend.reinvestment_transaction_id)
                    if transaction:
                        transaction.date = dividend.buy_order_date or dividend.ex_dividend_date
                        transaction.shares = float(data["reinvestment_shares"])
                        transaction.cost_per_share = float(data["reinvestment_price"])
                        db.session.add(transaction)
                        dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
                else:
                    # Create new reinvestment transaction
                    transaction = Transaction(
                        id=str(uuid.uuid4()),
                        portfolio_fund_id=dividend.portfolio_fund_id,
                        date=dividend.buy_order_date or dividend.ex_dividend_date,
                        type="dividend",
                        shares=float(data["reinvestment_shares"]),
                        cost_per_share=float(data["reinvestment_price"]),
                    )
                    db.session.add(transaction)
                    db.session.flush()
                    dividend.reinvestment_transaction_id = transaction.id
                    dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
            elif dividend.reinvestment_transaction_id:
                # If reinvestment data is removed, delete the transaction
                transaction = Transaction.query.get(dividend.reinvestment_transaction_id)
                if transaction:
                    db.session.delete(transaction)
                dividend.reinvestment_transaction_id = None
                dividend.reinvestment_status = ReinvestmentStatus.PENDING
        else:
            # For cash dividends, always set status to COMPLETED
            dividend.reinvestment_status = ReinvestmentStatus.COMPLETED

        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.DIVIDEND,
            message=f"Successfully updated dividend {dividend_id}",
            details={
                "original_values": original_values,
                "new_values": {
                    "record_date": dividend.record_date.isoformat(),
                    "ex_dividend_date": dividend.ex_dividend_date.isoformat(),
                    "dividend_per_share": dividend.dividend_per_share,
                    "reinvestment_status": dividend.reinvestment_status.value,
                },
            },
        )

        return jsonify(DividendService.format_dividend(dividend))
    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.DIVIDEND,
            message=f"Error updating dividend: {str(e)}",
            details={"dividend_id": dividend_id, "error": str(e), "request_data": data},
            http_status=400,
        )
        return jsonify(response), status


@dividends.route("/dividends/<string:dividend_id>", methods=["DELETE"])
@track_request
def delete_dividend(dividend_id):
    """
    Delete a dividend record.

    Args:
        dividend_id (str): Dividend identifier

    Returns:
        Empty response with 204 status on success
    """
    try:
        # Store dividend details before deletion for logging
        dividend = Dividend.query.get_or_404(dividend_id)
        dividend_details = {
            "fund_name": dividend.fund.name,
            "total_amount": dividend.total_amount,
            "dividend_type": dividend.fund.dividend_type.value,
            "reinvestment_status": dividend.reinvestment_status.value,
        }

        DividendService.delete_dividend(dividend_id)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.DIVIDEND,
            message=f"Successfully deleted dividend {dividend_id}",
            details=dividend_details,
        )

        return "", 204
    except ValueError as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.DIVIDEND,
            message=f"Error deleting dividend: {str(e)}",
            details={"dividend_id": dividend_id, "error": str(e)},
            http_status=400,
        )
        return jsonify(response), status
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.DIVIDEND,
            message=f"Unexpected error deleting dividend: {str(e)}",
            details={"dividend_id": dividend_id, "error": str(e)},
            http_status=500,
        )
        return jsonify(response), status
