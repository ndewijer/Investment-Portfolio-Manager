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

from ..models import Fund, LogCategory, LogLevel, db
from ..services.logging_service import logger
from ..services.portfolio_service import PortfolioService

# Create namespace
ns = Namespace('portfolios', description='Portfolio management operations')

# Define models for documentation
portfolio_list_item_model = ns.model('PortfolioListItem', {
    'id': fields.String(required=True, description='Portfolio unique identifier (UUID)'),
    'name': fields.String(required=True, description='Portfolio name'),
    'description': fields.String(description='Portfolio description'),
    'is_archived': fields.Boolean(required=True, description='Whether portfolio is archived'),
    'exclude_from_overview': fields.Boolean(required=True, description='Whether to exclude from overview')
})

portfolio_detail_model = ns.model('PortfolioDetail', {
    'id': fields.String(required=True, description='Portfolio unique identifier (UUID)'),
    'name': fields.String(required=True, description='Portfolio name'),
    'description': fields.String(description='Portfolio description'),
    'is_archived': fields.Boolean(required=True, description='Whether portfolio is archived'),
    'totalValue': fields.Float(required=True, description='Total current value'),
    'totalCost': fields.Float(required=True, description='Total cost basis'),
    'totalDividends': fields.Float(required=True, description='Total dividends received'),
    'totalUnrealizedGainLoss': fields.Float(required=True, description='Total unrealized gain/loss'),
    'totalRealizedGainLoss': fields.Float(required=True, description='Total realized gain/loss'),
    'totalGainLoss': fields.Float(required=True, description='Total gain/loss (realized + unrealized)')
})

portfolio_create_model = ns.model('PortfolioCreate', {
    'name': fields.String(required=True, description='Portfolio name', example='Retirement Portfolio'),
    'description': fields.String(description='Portfolio description', example='Long-term retirement savings')
})

portfolio_update_model = ns.model('PortfolioUpdate', {
    'name': fields.String(description='Portfolio name', example='Updated Portfolio Name'),
    'description': fields.String(description='Portfolio description'),
    'exclude_from_overview': fields.Boolean(description='Whether to exclude from overview')
})

error_model = ns.model('Error', {
    'error': fields.String(required=True, description='Error message')
})


@ns.route('')
class PortfolioList(Resource):
    """Portfolio collection endpoint."""

    @ns.doc('list_portfolios')
    @ns.response(200, 'Success', [portfolio_list_item_model])
    @ns.response(500, 'Server error', error_model)
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
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message="Error retrieving portfolios",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc('create_portfolio')
    @ns.expect(portfolio_create_model, validate=True)
    @ns.response(201, 'Portfolio created', portfolio_list_item_model)
    @ns.response(400, 'Validation error', error_model)
    @ns.response(500, 'Server error', error_model)
    def post(self):
        """
        Create a new portfolio.

        Creates a new portfolio with the provided name and optional description.
        Returns the created portfolio with all fields populated.
        """
        try:
            data = request.json
            portfolio = PortfolioService.create_portfolio(
                name=data["name"],
                description=data.get("description")
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
            return {"error": str(e)}, 400
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message="Error creating portfolio",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route('/<string:portfolio_id>')
@ns.param('portfolio_id', 'Portfolio unique identifier (UUID)')
class Portfolio(Resource):
    """Portfolio detail endpoint."""

    @ns.doc('get_portfolio')
    @ns.response(200, 'Success', portfolio_detail_model)
    @ns.response(404, 'Portfolio not found', error_model)
    @ns.response(500, 'Server error', error_model)
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
                "totalUnrealizedGainLoss": sum(pf["unrealized_gain_loss"] for pf in portfolio_funds_data),
                "totalRealizedGainLoss": sum(pf["realized_gain_loss"] for pf in portfolio_funds_data),
                "totalGainLoss": sum(pf["total_gain_loss"] for pf in portfolio_funds_data),
            }, 200

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

    @ns.doc('update_portfolio')
    @ns.expect(portfolio_update_model, validate=True)
    @ns.response(200, 'Portfolio updated', portfolio_list_item_model)
    @ns.response(404, 'Portfolio not found', error_model)
    @ns.response(500, 'Server error', error_model)
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
                exclude_from_overview=data.get("exclude_from_overview")
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
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error updating portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500

    @ns.doc('delete_portfolio')
    @ns.response(200, 'Portfolio deleted')
    @ns.response(404, 'Portfolio not found', error_model)
    @ns.response(500, 'Server error', error_model)
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
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error deleting portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route('/<string:portfolio_id>/archive')
@ns.param('portfolio_id', 'Portfolio unique identifier (UUID)')
class PortfolioArchive(Resource):
    """Portfolio archive endpoint."""

    @ns.doc('archive_portfolio')
    @ns.response(200, 'Portfolio archived')
    @ns.response(404, 'Portfolio not found', error_model)
    @ns.response(500, 'Server error', error_model)
    def post(self, portfolio_id):
        """
        Archive a portfolio.

        Archives a portfolio, hiding it from normal views while preserving all data.
        Archived portfolios can be unarchived later.
        """
        try:
            PortfolioService.archive_portfolio(portfolio_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Portfolio archived: {portfolio_id}",
                details={"portfolio_id": portfolio_id},
            )

            return {"success": True}, 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error archiving portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500


@ns.route('/<string:portfolio_id>/unarchive')
@ns.param('portfolio_id', 'Portfolio unique identifier (UUID)')
class PortfolioUnarchive(Resource):
    """Portfolio unarchive endpoint."""

    @ns.doc('unarchive_portfolio')
    @ns.response(200, 'Portfolio unarchived')
    @ns.response(404, 'Portfolio not found', error_model)
    @ns.response(500, 'Server error', error_model)
    def post(self, portfolio_id):
        """
        Unarchive a portfolio.

        Restores an archived portfolio, making it visible in normal views again.
        """
        try:
            PortfolioService.unarchive_portfolio(portfolio_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.PORTFOLIO,
                message=f"Portfolio unarchived: {portfolio_id}",
                details={"portfolio_id": portfolio_id},
            )

            return {"success": True}, 200

        except ValueError as e:
            return {"error": str(e)}, 404
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.PORTFOLIO,
                message=f"Error unarchiving portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500
