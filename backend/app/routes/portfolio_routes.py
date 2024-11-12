"""
API routes for portfolio management.

This module provides routes for:
- Creating portfolios
- Retrieving portfolio details
- Updating portfolio information
- Deleting portfolios
- Archiving and unarchiving portfolios
- Managing portfolio-fund relationships
- Retrieving portfolio summary and history
"""

from flask import Blueprint, jsonify, request
from flask.views import MethodView

from ..models import (
    Dividend,
    Fund,
    LogCategory,
    LogLevel,
    Portfolio,
    PortfolioFund,
    Transaction,
    db,
)
from ..services.logging_service import logger, track_request
from ..services.portfolio_service import PortfolioService

portfolios = Blueprint("portfolios", __name__)


class PortfolioAPI(MethodView):
    """
    RESTful API for portfolio management.

    Provides endpoints for:
    - Creating portfolios
    - Retrieving portfolio details
    - Updating portfolio information
    - Deleting portfolios

    All methods include proper error handling and logging.
    """

    def get(self, portfolio_id=None):
        """
        Retrieve portfolio(s).

        Args:
            portfolio_id (str, optional): Portfolio identifier

        Returns:
            JSON response containing portfolio details or list of portfolios
        """
        if portfolio_id is None:
            portfolios = Portfolio.query.all()
            return jsonify(
                [
                    {
                        "id": p.id,
                        "name": p.name,
                        "description": p.description,
                        "is_archived": p.is_archived,
                    }
                    for p in portfolios
                ]
            )
        else:
            portfolio = Portfolio.query.get_or_404(portfolio_id)
            if portfolio.is_archived:
                return jsonify({"error": "Portfolio is archived"}), 404

            service = PortfolioService()
            portfolio_funds_data = service.calculate_portfolio_fund_values(
                portfolio.funds
            )

            # Calculate totals from portfolio_funds_data
            total_value = sum(pf["current_value"] for pf in portfolio_funds_data)
            total_cost = sum(pf["total_cost"] for pf in portfolio_funds_data)
            total_dividends = sum(pf["total_dividends"] for pf in portfolio_funds_data)

            return jsonify(
                {
                    "id": portfolio.id,
                    "name": portfolio.name,
                    "description": portfolio.description,
                    "is_archived": portfolio.is_archived,
                    "totalValue": total_value,
                    "totalCost": total_cost,
                    "totalDividends": total_dividends,
                }
            )

    def post(self):
        """
        Create a new portfolio.

        Request Body:
            name (str): Portfolio name
            description (str, optional): Portfolio description

        Returns:
            JSON response containing created portfolio details
        """
        data = request.json
        portfolio = Portfolio(
            name=data["name"], description=data.get("description", "")
        )
        db.session.add(portfolio)
        db.session.commit()
        return jsonify(
            {
                "id": portfolio.id,
                "name": portfolio.name,
                "description": portfolio.description,
            }
        )

    def put(self, portfolio_id):
        """
        Update an existing portfolio.

        Args:
            portfolio_id (str): Portfolio identifier

        Request Body:
            name (str): Portfolio name
            description (str, optional): Portfolio description

        Returns:
            JSON response containing updated portfolio details
        """
        portfolio = Portfolio.query.get_or_404(portfolio_id)
        data = request.json
        portfolio.name = data["name"]
        portfolio.description = data.get("description", "")
        db.session.commit()
        return jsonify(
            {
                "id": portfolio.id,
                "name": portfolio.name,
                "description": portfolio.description,
            }
        )

    def delete(self, portfolio_id):
        """
        Delete a portfolio.

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            Empty response with 204 status on success
        """
        portfolio = Portfolio.query.get_or_404(portfolio_id)
        db.session.delete(portfolio)
        db.session.commit()
        return "", 204


@portfolios.route("/portfolios/<string:portfolio_id>/archive", methods=["POST"])
def archive_portfolio(portfolio_id):
    """
    Archive a portfolio.

    Args:
        portfolio_id (str): Portfolio identifier

    Returns:
        JSON response containing updated portfolio details
    """
    portfolio = Portfolio.query.get_or_404(portfolio_id)
    portfolio.is_archived = True
    db.session.commit()
    return jsonify(
        {
            "id": portfolio.id,
            "name": portfolio.name,
            "description": portfolio.description,
            "is_archived": portfolio.is_archived,
        }
    )


@portfolios.route("/portfolios/<string:portfolio_id>/unarchive", methods=["POST"])
def unarchive_portfolio(portfolio_id):
    """
    Unarchive a portfolio.

    Args:
        portfolio_id (str): Portfolio identifier

    Returns:
        JSON response containing updated portfolio details
    """
    portfolio = Portfolio.query.get_or_404(portfolio_id)
    portfolio.is_archived = False
    db.session.commit()
    return jsonify(
        {
            "id": portfolio.id,
            "name": portfolio.name,
            "description": portfolio.description,
            "is_archived": portfolio.is_archived,
        }
    )


# Register the views
portfolio_view = PortfolioAPI.as_view("portfolio_api")
portfolios.add_url_rule(
    "/portfolios",
    defaults={"portfolio_id": None},
    view_func=portfolio_view,
    methods=["GET"],
)
portfolios.add_url_rule("/portfolios", view_func=portfolio_view, methods=["POST"])
portfolios.add_url_rule(
    "/portfolios/<string:portfolio_id>",
    view_func=portfolio_view,
    methods=["GET", "PUT", "DELETE"],
)


# Summary and History endpoints
@portfolios.route("/portfolio-summary", methods=["GET"])
def get_portfolio_summary():
    """
    Get summary of all portfolios.

    Query Parameters:
        include_archived (bool, optional): Include archived portfolios

    Returns:
        JSON response containing portfolio summaries with:
        - Total value
        - Total cost
        - Total dividends
        - Fund count
        - Transaction count
    """
    service = PortfolioService()
    return jsonify(service.get_portfolio_summary())


@portfolios.route("/portfolio-history", methods=["GET"])
def get_portfolio_history():
    """
    Get historical data for all portfolios.

    Query Parameters:
        start_date (str, optional): Start date in YYYY-MM-DD format
        end_date (str, optional): End date in YYYY-MM-DD format
        interval (str, optional): Data interval ('daily', 'weekly', 'monthly')

    Returns:
        JSON response containing:
        - Historical values
        - Performance metrics
        - Transaction history
    """
    service = PortfolioService()
    return jsonify(service.get_portfolio_history())


@portfolios.route("/portfolio-funds", methods=["GET", "POST"])
def handle_portfolio_funds():
    """
    Handle portfolio-fund relationships.

    GET:
        Query Parameters:
            portfolio_id (str, optional): Filter by portfolio

    POST:
        Request Body:
            portfolio_id (str): Portfolio identifier
            fund_id (str): Fund identifier

    Returns:
        JSON response containing portfolio-fund relationships or created relationship
    """
    if request.method == "GET":
        portfolio_id = request.args.get("portfolio_id")
        service = PortfolioService()

        if portfolio_id:
            portfolio_funds = service.get_portfolio_funds(portfolio_id)
        else:
            portfolio_funds = service.get_all_portfolio_funds()

        # Add dividend_type to each portfolio fund
        for pf in portfolio_funds:
            fund = Fund.query.get(pf["fund_id"])
            if fund:
                pf["dividend_type"] = fund.dividend_type.value

        return jsonify(portfolio_funds)
    else:
        data = request.json
        portfolio_fund = PortfolioFund(
            portfolio_id=data["portfolio_id"], fund_id=data["fund_id"]
        )
        db.session.add(portfolio_fund)
        db.session.commit()
        return jsonify(
            {
                "id": portfolio_fund.id,
                "portfolio_id": portfolio_fund.portfolio_id,
                "fund_id": portfolio_fund.fund_id,
            }
        )


# Add this new route
@portfolios.route("/portfolios/<string:portfolio_id>/fund-history", methods=["GET"])
def get_portfolio_fund_history(portfolio_id):
    """
    Get historical data for funds in a portfolio.

    Args:
        portfolio_id (str): Portfolio identifier

    Returns:
        JSON response containing fund value history
    """
    service = PortfolioService()
    history = service.get_portfolio_fund_history(portfolio_id)
    return jsonify(history)


@portfolios.route("/portfolio-funds/<string:portfolio_fund_id>", methods=["DELETE"])
@track_request
def delete_portfolio_fund(portfolio_fund_id):
    """
    Delete a portfolio-fund relationship.

    Args:
        portfolio_fund_id (str): Portfolio-Fund relationship identifier

    Query Parameters:
        confirm (bool, optional): Confirmation for deletion with transactions

    Returns:
        Empty response with 204 status on success
    """
    try:
        # Eager load the fund and portfolio relationships
        portfolio_fund = PortfolioFund.query.options(
            db.joinedload(PortfolioFund.fund), db.joinedload(PortfolioFund.portfolio)
        ).get_or_404(portfolio_fund_id)

        # Count associated transactions and dividends
        transaction_count = Transaction.query.filter_by(
            portfolio_fund_id=portfolio_fund_id
        ).count()
        dividend_count = Dividend.query.filter_by(
            portfolio_fund_id=portfolio_fund_id
        ).count()

        # Store fund and portfolio names before potential deletion
        fund_name = portfolio_fund.fund.name
        portfolio_name = portfolio_fund.portfolio.name

        # If there are associated records and no confirmation, return count for confirmation
        if (transaction_count > 0 or dividend_count > 0) and request.args.get(
            "confirm"
        ) != "true":
            response, status = logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Deletion of portfolio fund {portfolio_fund_id} requires confirmation",
                details={
                    "user_message": "Confirmation required for deletion",
                    "requires_confirmation": True,
                    "transaction_count": transaction_count,
                    "dividend_count": dividend_count,
                    "fund_name": fund_name,
                    "portfolio_name": portfolio_name,
                },
                http_status=409,
            )
            return jsonify(response), status

        try:
            # Delete associated records if they exist
            if transaction_count > 0:
                Transaction.query.filter_by(
                    portfolio_fund_id=portfolio_fund_id
                ).delete()
            if dividend_count > 0:
                Dividend.query.filter_by(portfolio_fund_id=portfolio_fund_id).delete()

            # Delete the portfolio-fund relationship
            db.session.delete(portfolio_fund)
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Successfully deleted portfolio fund {portfolio_fund_id}",
                details={
                    "transactions_deleted": transaction_count,
                    "dividends_deleted": dividend_count,
                    "fund_name": fund_name,
                    "portfolio_name": portfolio_name,
                },
            )

            return "", 204

        except Exception as e:
            db.session.rollback()
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error deleting portfolio fund: {str(e)}",
                details={
                    "user_message": "Error deleting fund from portfolio",
                    "error": str(e),
                    "portfolio_fund_id": portfolio_fund_id,
                },
                http_status=400,
            )
            return jsonify(response), status

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.PORTFOLIO,
            message=f"Error accessing portfolio fund: {str(e)}",
            details={
                "user_message": "Error accessing fund relationship",
                "error": str(e),
                "portfolio_fund_id": portfolio_fund_id,
            },
            http_status=400,
        )
        return jsonify(response), status
