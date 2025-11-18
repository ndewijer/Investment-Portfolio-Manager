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
    Fund,
    LogCategory,
    LogLevel,
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
            "totalRealizedGainLoss": sum(pf["realized_gain_loss"] for pf in portfolio_funds_data),
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
            portfolios = PortfolioService.get_all_portfolios()
            return jsonify([self._format_portfolio_list_item(p) for p in portfolios])

        portfolio = PortfolioService.get_portfolio(portfolio_id)
        if portfolio.is_archived:
            return jsonify({"error": "Portfolio is archived"}), 404

        portfolio_funds_data = PortfolioService.calculate_portfolio_fund_values(portfolio.funds)

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
        portfolio = PortfolioService.create_portfolio(
            name=data["name"], description=data.get("description", "")
        )

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
        data = request.json

        try:
            portfolio = PortfolioService.update_portfolio(
                portfolio_id=portfolio_id,
                name=data["name"],
                description=data.get("description", ""),
                exclude_from_overview=data.get("exclude_from_overview", False),
            )
            return jsonify(self._format_portfolio_list_item(portfolio))
        except ValueError as e:
            return jsonify({"error": str(e)}), 404

    def delete(self, portfolio_id):
        """
        Delete a portfolio.

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            Empty response with 204 status on success
        """
        try:
            PortfolioService.delete_portfolio(portfolio_id)
            return "", 204
        except ValueError as e:
            return jsonify({"error": str(e)}), 404


def _update_portfolio_archive_status(portfolio_id, is_archived):
    """
    Helper function to update portfolio archive status.

    Args:
        portfolio_id (str): Portfolio identifier
        is_archived (bool): Archive status

    Returns:
        JSON response containing updated portfolio details
    """
    try:
        portfolio = PortfolioService.update_archive_status(portfolio_id, is_archived)
        return jsonify(PortfolioAPI._format_portfolio_list_item(portfolio))
    except ValueError as e:
        return jsonify({"error": str(e)}), 404


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

    Query Parameters:
        start_date (str, optional): Start date in YYYY-MM-DD format
        end_date (str, optional): End date in YYYY-MM-DD format

    Returns:
        JSON response containing historical portfolio data
    """
    start_date = request.args.get("start_date")
    end_date = request.args.get("end_date")

    return jsonify(PortfolioService.get_portfolio_history(start_date, end_date))


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
                fund = db.session.get(Fund, pf["fund_id"])
                if fund:
                    pf["dividend_type"] = fund.dividend_type.value

        return jsonify(portfolio_funds)

    # POST - Create new portfolio-fund relationship
    data = request.json
    try:
        portfolio_fund = PortfolioService.create_portfolio_fund(
            portfolio_id=data["portfolio_id"], fund_id=data["fund_id"]
        )

        return jsonify(
            {
                "id": portfolio_fund.id,
                "portfolio_id": portfolio_fund.portfolio_id,
                "fund_id": portfolio_fund.fund_id,
            }
        )
    except ValueError as e:
        return jsonify({"error": str(e)}), 404


@portfolios.route("/portfolios/<string:portfolio_id>/fund-history", methods=["GET"])
def get_portfolio_fund_history(portfolio_id):
    """
    Get historical data for funds in a portfolio.

    Args:
        portfolio_id (str): Portfolio identifier

    Query Parameters:
        start_date (str, optional): Start date in YYYY-MM-DD format
        end_date (str, optional): End date in YYYY-MM-DD format

    Returns:
        JSON response containing fund value history
    """
    start_date = request.args.get("start_date")
    end_date = request.args.get("end_date")

    return jsonify(PortfolioService.get_portfolio_fund_history(portfolio_id, start_date, end_date))


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
    confirmed = request.args.get("confirm") == "true"

    try:
        result = PortfolioService.delete_portfolio_fund(portfolio_fund_id, confirmed=confirmed)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.PORTFOLIO,
            message=f"Successfully deleted portfolio fund {portfolio_fund_id}",
            details=result,
        )

        return "", 204

    except ValueError as e:
        error_message = str(e)

        # Check if this is a confirmation-required error
        if "Confirmation required" in error_message:
            # Extract counts from error message or query again
            portfolio_fund = PortfolioService.get_portfolio_fund(
                portfolio_fund_id, with_relationships=True
            )

            if portfolio_fund:
                transaction_count = PortfolioService.count_portfolio_fund_transactions(
                    portfolio_fund_id
                )
                dividend_count = PortfolioService.count_portfolio_fund_dividends(portfolio_fund_id)

                response, status = logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.PORTFOLIO,
                    message=f"Deletion of portfolio fund {portfolio_fund_id} requires confirmation",
                    details={
                        "user_message": "Confirmation required for deletion",
                        "requires_confirmation": True,
                        "transaction_count": transaction_count,
                        "dividend_count": dividend_count,
                        "fund_name": portfolio_fund.fund.name,
                        "portfolio_name": portfolio_fund.portfolio.name,
                    },
                    http_status=409,
                )
                return jsonify(response), status

        # Other ValueError (not found, etc.)
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.PORTFOLIO,
            message=f"Error deleting portfolio fund: {error_message}",
            details={
                "user_message": error_message,
                "portfolio_fund_id": portfolio_fund_id,
            },
            http_status=404,
        )
        return jsonify(response), status

    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.PORTFOLIO,
            message=f"Error deleting portfolio fund: {e!s}",
            details={
                "user_message": "Error deleting fund from portfolio",
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

    portfolios_list = PortfolioService.get_portfolios_list(include_excluded=include_excluded)
    return jsonify([PortfolioAPI._format_portfolio_list_item(p) for p in portfolios_list])
