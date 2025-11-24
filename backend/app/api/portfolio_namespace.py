"""
Portfolio API namespace for managing investment portfolios.

This namespace provides endpoints for:
- Creating, reading, updating, and deleting portfolios
- Archiving and unarchiving portfolios
- Managing portfolio-fund relationships
- Retrieving portfolio summaries and performance metrics
"""

from flask import request
from flask_restx import Namespace, Resource, fields
from werkzeug.exceptions import HTTPException

from ..models import LogCategory, LogLevel
from ..services.logging_service import logger
from ..services.portfolio_service import PortfolioService

# Create namespace
ns = Namespace("portfolios", description="Portfolio management operations")

# Define models for documentation
portfolio_list_item_model = ns.model(
    "PortfolioListItem",
    {
        "id": fields.String(required=True, description="Portfolio unique identifier (UUID)"),
        "name": fields.String(required=True, description="Portfolio name"),
        "description": fields.String(description="Portfolio description"),
        "is_archived": fields.Boolean(required=True, description="Whether portfolio is archived"),
        "exclude_from_overview": fields.Boolean(
            required=True, description="Whether to exclude from overview"
        ),
    },
)

portfolio_detail_model = ns.model(
    "PortfolioDetail",
    {
        "id": fields.String(required=True, description="Portfolio unique identifier (UUID)"),
        "name": fields.String(required=True, description="Portfolio name"),
        "description": fields.String(description="Portfolio description"),
        "is_archived": fields.Boolean(required=True, description="Whether portfolio is archived"),
        "totalValue": fields.Float(required=True, description="Total current value"),
        "totalCost": fields.Float(required=True, description="Total cost basis"),
        "totalDividends": fields.Float(required=True, description="Total dividends received"),
        "totalUnrealizedGainLoss": fields.Float(
            required=True, description="Total unrealized gain/loss"
        ),
        "totalRealizedGainLoss": fields.Float(
            required=True, description="Total realized gain/loss"
        ),
        "totalGainLoss": fields.Float(
            required=True, description="Total gain/loss (realized + unrealized)"
        ),
    },
)

portfolio_create_model = ns.model(
    "PortfolioCreate",
    {
        "name": fields.String(
            required=True, description="Portfolio name", example="Retirement Portfolio"
        ),
        "description": fields.String(
            description="Portfolio description", example="Long-term retirement savings"
        ),
    },
)

portfolio_update_model = ns.model(
    "PortfolioUpdate",
    {
        "name": fields.String(description="Portfolio name", example="Updated Portfolio Name"),
        "description": fields.String(description="Portfolio description"),
        "exclude_from_overview": fields.Boolean(description="Whether to exclude from overview"),
    },
)

error_model = ns.model(
    "Error", {"error": fields.String(required=True, description="Error message")}
)


@ns.route("")
class PortfolioList(Resource):
    """Portfolio collection endpoint."""

    @ns.doc("list_portfolios")
    @ns.response(200, "Success", [portfolio_list_item_model])
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get all portfolios.

        Returns a list of all portfolios in the system, including archived ones.
        Each portfolio includes basic information such as name, description, and archive status.
        """
        try:
            portfolios = PortfolioService.get_all_portfolios()
            return [
                {
                    "id": p.id,
                    "name": p.name,
                    "description": p.description,
                    "is_archived": p.is_archived,
                    "exclude_from_overview": p.exclude_from_overview,
                }
                for p in portfolios
            ], 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message="Error retrieving portfolios",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc("create_portfolio")
    @ns.expect(portfolio_create_model, validate=True)
    @ns.response(201, "Portfolio created", portfolio_list_item_model)
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Create a new portfolio.

        Creates a new portfolio with the provided name and optional description.
        Returns the created portfolio with all fields populated.
        """
        try:
            data = request.json
            portfolio = PortfolioService.create_portfolio(
                name=data["name"], description=data.get("description")
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Portfolio created: {portfolio.name}",
                details={"portfolio_id": portfolio.id},
            )

            return {
                "id": portfolio.id,
                "name": portfolio.name,
                "description": portfolio.description,
                "is_archived": portfolio.is_archived,
                "exclude_from_overview": portfolio.exclude_from_overview,
            }, 201

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message="Error creating portfolio",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route("/<string:portfolio_id>")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class Portfolio(Resource):
    """Portfolio detail endpoint."""

    @ns.doc("get_portfolio")
    @ns.response(200, "Success", portfolio_detail_model)
    @ns.response(404, "Portfolio not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, portfolio_id):
        """
        Get portfolio details.

        Returns detailed information about a specific portfolio, including:
        - Basic information (name, description)
        - Performance metrics (total value, cost, gain/loss)
        - Dividend information

        Archived portfolios will return a 404 error.
        """
        try:
            portfolio = PortfolioService.get_portfolio(portfolio_id)
            if portfolio.is_archived:
                return {"error": "Portfolio is archived"}, 404

            portfolio_funds_data = PortfolioService.calculate_portfolio_fund_values(portfolio.funds)

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
            }, 200

        except HTTPException:
            # Let Flask handle HTTP exceptions (like abort(404))
            raise
        except ValueError as e:
            return {"error": str(e)}, 404
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error retrieving portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc("update_portfolio")
    @ns.expect(portfolio_update_model, validate=True)
    @ns.response(200, "Portfolio updated", portfolio_list_item_model)
    @ns.response(404, "Portfolio not found", error_model)
    @ns.response(500, "Server error", error_model)
    def put(self, portfolio_id):
        """
        Update portfolio information.

        Updates one or more fields of an existing portfolio.
        Only provided fields will be updated; omitted fields remain unchanged.
        """
        try:
            data = request.json
            portfolio = PortfolioService.update_portfolio(
                portfolio_id=portfolio_id,
                name=data.get("name"),
                description=data.get("description"),
                exclude_from_overview=data.get("exclude_from_overview"),
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Portfolio updated: {portfolio.name}",
                details={"portfolio_id": portfolio_id},
            )

            return {
                "id": portfolio.id,
                "name": portfolio.name,
                "description": portfolio.description,
                "is_archived": portfolio.is_archived,
                "exclude_from_overview": portfolio.exclude_from_overview,
            }, 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error updating portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc("delete_portfolio")
    @ns.response(200, "Portfolio deleted")
    @ns.response(404, "Portfolio not found", error_model)
    @ns.response(500, "Server error", error_model)
    def delete(self, portfolio_id):
        """
        Delete a portfolio.

        Permanently deletes a portfolio and all associated data, including:
        - Portfolio-fund relationships
        - Transactions
        - Dividend allocations

        This operation cannot be undone. Consider archiving instead if you want to preserve data.
        """
        try:
            PortfolioService.delete_portfolio(portfolio_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Portfolio deleted: {portfolio_id}",
                details={"portfolio_id": portfolio_id},
            )

            return {"success": True}, 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error deleting portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route("/<string:portfolio_id>/archive")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class PortfolioArchive(Resource):
    """Portfolio archive endpoint."""

    @ns.doc("archive_portfolio")
    @ns.response(200, "Portfolio archived")
    @ns.response(404, "Portfolio not found", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self, portfolio_id):
        """
        Archive a portfolio.

        Archives a portfolio, hiding it from normal views while preserving all data.
        Archived portfolios can be unarchived later.
        """
        try:
            portfolio = PortfolioService.update_archive_status(portfolio_id, True)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Portfolio archived: {portfolio_id}",
                details={"portfolio_id": portfolio_id},
            )

            return PortfolioService.format_portfolio_list_item(portfolio), 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error archiving portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route("/<string:portfolio_id>/unarchive")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class PortfolioUnarchive(Resource):
    """Portfolio unarchive endpoint."""

    @ns.doc("unarchive_portfolio")
    @ns.response(200, "Portfolio unarchived")
    @ns.response(404, "Portfolio not found", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self, portfolio_id):
        """
        Unarchive a portfolio.

        Restores an archived portfolio, making it visible in normal views again.
        """
        try:
            portfolio = PortfolioService.update_archive_status(portfolio_id, False)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Portfolio unarchived: {portfolio_id}",
                details={"portfolio_id": portfolio_id},
            )

            return PortfolioService.format_portfolio_list_item(portfolio), 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error unarchiving portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route("-summary")
class PortfolioSummary(Resource):
    """Portfolio summary endpoint."""

    @ns.doc("get_portfolio_summary")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get summary of all portfolios.

        Returns aggregated summary data across all portfolios,
        including total values, costs, and gains/losses.

        Useful for dashboard overview displays.
        """
        try:
            return PortfolioService.get_portfolio_summary(), 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message="Error retrieving portfolio summary",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route("-history")
class PortfolioHistory(Resource):
    """Portfolio history endpoint."""

    @ns.doc("get_portfolio_history")
    @ns.param("start_date", "Start date (YYYY-MM-DD)", _in="query")
    @ns.param("end_date", "End date (YYYY-MM-DD)", _in="query")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get historical data for all portfolios.

        Returns time-series data showing portfolio values over time.

        Query Parameters:
        - start_date: Start date for historical data (YYYY-MM-DD)
        - end_date: End date for historical data (YYYY-MM-DD)

        Useful for performance charts and analysis.
        """
        try:
            start_date = request.args.get("start_date")
            end_date = request.args.get("end_date")

            return PortfolioService.get_portfolio_history(start_date, end_date), 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message="Error retrieving portfolio history",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route("-funds")
class PortfolioFundsList(Resource):
    """Portfolio-fund relationships endpoint."""

    @ns.doc("get_portfolio_funds")
    @ns.param("portfolio_id", "Filter by portfolio ID", _in="query")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get portfolio-fund relationships.

        Returns all portfolio-fund relationships, optionally filtered by portfolio.

        Query Parameters:
        - portfolio_id: Filter relationships for a specific portfolio

        Each relationship represents a fund held in a portfolio.
        """
        try:
            portfolio_id = request.args.get("portfolio_id")

            if portfolio_id:
                portfolio_funds = PortfolioService.get_portfolio_funds(portfolio_id)
            else:
                portfolio_funds = PortfolioService.get_all_portfolio_funds()

            return portfolio_funds, 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message="Error retrieving portfolio funds",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc("create_portfolio_fund")
    @ns.expect(
        ns.model(
            "PortfolioFundCreate",
            {
                "portfolio_id": fields.String(required=True, description="Portfolio ID"),
                "fund_id": fields.String(required=True, description="Fund ID"),
            },
        ),
        validate=True,
    )
    @ns.response(201, "Portfolio-fund relationship created")
    @ns.response(404, "Portfolio or fund not found", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Create portfolio-fund relationship.

        Adds a fund to a portfolio, creating the relationship
        that allows transactions to be recorded.
        """
        try:
            data = request.json
            portfolio_fund = PortfolioService.create_portfolio_fund(
                portfolio_id=data["portfolio_id"], fund_id=data["fund_id"]
            )

            return {
                "id": portfolio_fund.id,
                "portfolio_id": portfolio_fund.portfolio_id,
                "fund_id": portfolio_fund.fund_id,
            }, 201
        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            return {"error": str(e)}, 500


@ns.route("-funds/<string:portfolio_fund_id>")
@ns.param("portfolio_fund_id", "Portfolio-Fund relationship ID")
class PortfolioFundDetail(Resource):
    """Portfolio-fund relationship detail endpoint."""

    @ns.doc("delete_portfolio_fund")
    @ns.param("confirm", "Confirm deletion with transactions", _in="query")
    @ns.response(204, "Portfolio-fund relationship deleted")
    @ns.response(404, "Not found", error_model)
    @ns.response(409, "Confirmation required", error_model)
    @ns.response(500, "Server error", error_model)
    def delete(self, portfolio_fund_id):
        """
        Delete portfolio-fund relationship.

        Removes a fund from a portfolio. If transactions exist,
        requires confirmation via confirm=true query parameter.

        This will also delete all associated transactions and dividends.
        """
        try:
            confirmed = request.args.get("confirm") == "true"
            PortfolioService.delete_portfolio_fund(portfolio_fund_id, confirmed=confirmed)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Deleted portfolio fund {portfolio_fund_id}",
                details={"portfolio_fund_id": portfolio_fund_id},
            )

            return "", 204

        except ValueError as e:
            error_message = str(e)

            if "Confirmation required" in error_message:
                portfolio_fund = PortfolioService.get_portfolio_fund(
                    portfolio_fund_id, with_relationships=True
                )

                if portfolio_fund:
                    transaction_count = PortfolioService.count_portfolio_fund_transactions(
                        portfolio_fund_id
                    )
                    dividend_count = PortfolioService.count_portfolio_fund_dividends(
                        portfolio_fund_id
                    )

                    return {
                        "error": "Confirmation required for deletion",
                        "requires_confirmation": True,
                        "transaction_count": transaction_count,
                        "dividend_count": dividend_count,
                        "fund_name": portfolio_fund.fund.name,
                        "portfolio_name": portfolio_fund.portfolio.name,
                    }, 409

            return {"error": error_message}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error deleting portfolio fund: {e!s}",
                details={"error": str(e), "portfolio_fund_id": portfolio_fund_id},
            )
            return {"error": "Error deleting fund from portfolio"}, 400


@ns.route("/<string:portfolio_id>/fund-history")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class PortfolioFundHistory(Resource):
    """Portfolio fund history endpoint."""

    @ns.doc("get_portfolio_fund_history")
    @ns.param("start_date", "Start date (YYYY-MM-DD)", _in="query")
    @ns.param("end_date", "End date (YYYY-MM-DD)", _in="query")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self, portfolio_id):
        """
        Get historical fund values for a portfolio.

        Returns time-series data showing individual fund values
        within a portfolio over time.

        Query Parameters:
        - start_date: Start date for historical data (YYYY-MM-DD)
        - end_date: End date for historical data (YYYY-MM-DD)

        Useful for analyzing individual fund performance within a portfolio.
        """
        try:
            start_date = request.args.get("start_date")
            end_date = request.args.get("end_date")

            return PortfolioService.get_portfolio_fund_history(
                portfolio_id, start_date, end_date
            ), 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error retrieving fund history for portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500
