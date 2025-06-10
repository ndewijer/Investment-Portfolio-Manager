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

    @staticmethod
    def _format_portfolio_list_item(portfolio):
        """Format a portfolio for list responses."""
        return {
            "id": portfolio.id,
            "name": portfolio.name,
            "description": portfolio.description,
            "is_archived": portfolio.is_archived,
            "exclude_from_overview": portfolio.exclude_from_overview,
        }

    @staticmethod
    def _format_portfolio_detail(portfolio, portfolio_funds_data):
        """Format a portfolio with detailed metrics."""
        return {
            "id": portfolio.id,
            "name": portfolio.name,
            "description": portfolio.description,
            "is_archived": portfolio.is_archived,
            "totalValue": sum(pf["current_value"] for pf in portfolio_funds_data),
            "totalCost": sum(pf["total_cost"] for pf in portfolio_funds_data),
            "totalDividends": sum(pf["total_dividends"] for pf in portfolio_funds_data),
            "totalUnrealizedGainLoss": sum(
                pf["unrealized_gain_loss"] for pf in portfolio_funds_data
            ),
            "totalRealizedGainLoss": sum(
                pf["realized_gain_loss"] for pf in portfolio_funds_data
            ),
            "totalGainLoss": sum(pf["total_gain_loss"] for pf in portfolio_funds_data),
        }

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
            return jsonify([self._format_portfolio_list_item(p) for p in portfolios])

        portfolio = Portfolio.query.get_or_404(portfolio_id)
        if portfolio.is_archived:
            return jsonify({"error": "Portfolio is archived"}), 404

        portfolio_funds_data = PortfolioService.calculate_portfolio_fund_values(
            portfolio.funds
        )

        return jsonify(self._format_portfolio_detail(portfolio, portfolio_funds_data))

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

        return jsonify(self._format_portfolio_list_item(portfolio))

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
        portfolio.exclude_from_overview = data.get("exclude_from_overview", False)

        db.session.commit()
        return jsonify(self._format_portfolio_list_item(portfolio))

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


def _update_portfolio_archive_status(portfolio_id, is_archived):
    """
    Helper function to update portfolio archive status.

    Args:
        portfolio_id (str): Portfolio identifier
        is_archived (bool): Archive status

    Returns:
        JSON response containing updated portfolio details
    """
    portfolio = Portfolio.query.get_or_404(portfolio_id)
    portfolio.is_archived = is_archived
    db.session.commit()

    return jsonify(PortfolioAPI._format_portfolio_list_item(portfolio))


@portfolios.route("/portfolios/<string:portfolio_id>/archive", methods=["POST"])
def archive_portfolio(portfolio_id):
    """Archive a portfolio."""
    return _update_portfolio_archive_status(portfolio_id, True)


@portfolios.route("/portfolios/<string:portfolio_id>/unarchive", methods=["POST"])
def unarchive_portfolio(portfolio_id):
    """Unarchive a portfolio."""
    return _update_portfolio_archive_status(portfolio_id, False)


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

    Returns:
        JSON response containing portfolio summaries
    """
    return jsonify(PortfolioService.get_portfolio_summary())


@portfolios.route("/portfolio-history", methods=["GET"])
def get_portfolio_history():
    """
    Get historical data for all portfolios.

    Returns:
        JSON response containing historical portfolio data
    """
    return jsonify(PortfolioService.get_portfolio_history())


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

        if portfolio_id:
            portfolio_funds = PortfolioService.get_portfolio_funds(portfolio_id)
        else:
            portfolio_funds = PortfolioService.get_all_portfolio_funds()

        # Add dividend_type to each portfolio fund
        for pf in portfolio_funds:
            if "fund_id" in pf:  # Only for detailed fund data
                fund = Fund.query.get(pf["fund_id"])
                if fund:
                    pf["dividend_type"] = fund.dividend_type.value

        return jsonify(portfolio_funds)

    # POST - Create new portfolio-fund relationship
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


@portfolios.route("/portfolios/<string:portfolio_id>/fund-history", methods=["GET"])
def get_portfolio_fund_history(portfolio_id):
    """
    Get historical data for funds in a portfolio.

    Args:
        portfolio_id (str): Portfolio identifier

    Returns:
        JSON response containing fund value history
    """
    return jsonify(PortfolioService.get_portfolio_fund_history(portfolio_id))


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


@portfolios.route("/portfolios", methods=["GET"])
def get_portfolios():
    """Get all portfolios."""
    include_excluded = request.args.get("include_excluded", "false").lower() == "true"

    query = Portfolio.query
    if not include_excluded:
        query = query.filter_by(exclude_from_overview=False)

    portfolios_list = query.all()
    return jsonify(
        [PortfolioAPI._format_portfolio_list_item(p) for p in portfolios_list]
    )
