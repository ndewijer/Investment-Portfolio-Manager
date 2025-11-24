"""
IBKR API namespace for Interactive Brokers integration.

This namespace provides endpoints for:
- IBKR Flex Query configuration
- Transaction import from IBKR
- Transaction inbox management and allocation
- Automated import scheduling
"""

from datetime import datetime

from flask import request
from flask_restx import Namespace, Resource, fields
from werkzeug.exceptions import HTTPException

from ..models import LogCategory, LogLevel
from ..services.ibkr_config_service import IBKRConfigService
from ..services.ibkr_flex_service import IBKRFlexService
from ..services.ibkr_transaction_service import IBKRTransactionService
from ..services.logging_service import logger

# Create namespace
ns = Namespace("ibkr", description="Interactive Brokers integration operations")

# Define models
ibkr_config_model = ns.model(
    "IBKRConfig",
    {
        "configured": fields.Boolean(required=True, description="Whether IBKR is configured"),
        "flex_query_id": fields.String(description="Flex Query ID"),
        "token_expires_at": fields.String(description="Token expiration date (ISO format)"),
        "token_warning": fields.String(description="Token expiration warning"),
        "last_import_date": fields.String(description="Last import date"),
        "auto_import_enabled": fields.Boolean(description="Auto import enabled"),
        "enabled": fields.Boolean(description="Configuration enabled"),
        "created_at": fields.String(description="Configuration creation date"),
        "updated_at": fields.String(description="Last update date"),
    },
)

ibkr_config_create_model = ns.model(
    "IBKRConfigCreate",
    {
        "flex_token": fields.String(
            required=True, description="Flex API token", example="XXXXXXXXXXXXXX"
        ),
        "flex_query_id": fields.String(
            required=True, description="Flex Query ID", example="123456"
        ),
        "token_expires_at": fields.String(description="Token expiration date (ISO format)"),
        "auto_import_enabled": fields.Boolean(description="Enable automatic import", default=False),
        "enabled": fields.Boolean(description="Enable configuration", default=True),
    },
)

ibkr_transaction_model = ns.model(
    "IBKRTransaction",
    {
        "id": fields.String(required=True, description="Transaction ID"),
        "ibkr_transaction_id": fields.String(required=True, description="IBKR Transaction ID"),
        "transaction_date": fields.String(required=True, description="Transaction date"),
        "symbol": fields.String(required=True, description="Security symbol"),
        "isin": fields.String(description="ISIN"),
        "description": fields.String(description="Description"),
        "transaction_type": fields.String(required=True, description="Transaction type"),
        "quantity": fields.Float(required=True, description="Quantity"),
        "price": fields.Float(required=True, description="Price"),
        "total_amount": fields.Float(required=True, description="Total amount"),
        "currency": fields.String(required=True, description="Currency"),
        "fees": fields.Float(description="Fees"),
        "status": fields.String(required=True, description="Processing status"),
        "imported_at": fields.String(description="Import timestamp"),
        "processed_at": fields.String(description="Processing timestamp"),
    },
)

allocation_model = ns.model(
    "Allocation",
    {
        "portfolio_id": fields.String(
            required=True, description="Portfolio ID", example="uuid-here"
        ),
        "percentage": fields.Float(
            required=True, description="Allocation percentage (0-100)", example=50.0
        ),
    },
)

allocation_request_model = ns.model(
    "AllocationRequest",
    {
        "allocations": fields.List(
            fields.Nested(allocation_model),
            required=True,
            description="List of portfolio allocations",
        )
    },
)

error_model = ns.model(
    "Error",
    {
        "error": fields.String(required=True, description="Error message"),
        "details": fields.String(description="Additional error details"),
    },
)


@ns.route("/config")
class IBKRConfig(Resource):
    """IBKR configuration endpoint."""

    @ns.doc("get_ibkr_config")
    @ns.response(200, "Success", ibkr_config_model)
    def get(self):
        """
        Get IBKR configuration status.

        Returns information about the current IBKR Flex Query configuration,
        including:
        - Connection status
        - Token expiration warnings
        - Last import timestamp
        - Auto-import settings

        Note: The Flex API token is never returned for security reasons.
        """
        config_status = IBKRConfigService.get_config_status()
        return config_status, 200

    @ns.doc("save_ibkr_config")
    @ns.expect(ibkr_config_create_model, validate=True)
    @ns.response(200, "Configuration saved")
    @ns.response(400, "Validation error", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Create or update IBKR configuration.

        Configures IBKR Flex Query credentials for automated transaction import.

        The Flex API token is encrypted before storage for security.

        Required permissions:
        - Flex Query access in IBKR
        - Query ID must be pre-configured in IBKR portal
        """
        data = request.get_json()

        if not data or "flex_token" not in data or "flex_query_id" not in data:
            return {"error": "Missing required fields: flex_token and flex_query_id"}, 400

        try:
            # Parse token expiration date if provided
            token_expires_at = None
            if data.get("token_expires_at"):
                try:
                    token_expires_at = datetime.fromisoformat(data["token_expires_at"])
                except (ValueError, TypeError):
                    return {"error": "Invalid token_expires_at format. Use ISO format."}, 400

            # Save config using service
            config = IBKRConfigService.save_config(
                flex_token=data["flex_token"],
                flex_query_id=data["flex_query_id"],
                token_expires_at=token_expires_at,
                auto_import_enabled=data.get("auto_import_enabled"),
                enabled=data.get("enabled"),
            )

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR configuration saved",
                details={"query_id": config.flex_query_id},
            )

            return {"success": True, "message": "Configuration saved successfully"}, 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to save IBKR configuration",
                details={"error": str(e)},
            )
            return {"error": "Failed to save configuration", "details": str(e)}, 500

    @ns.doc("delete_ibkr_config")
    @ns.response(200, "Configuration deleted")
    @ns.response(404, "Configuration not found", error_model)
    @ns.response(500, "Server error", error_model)
    def delete(self):
        """
        Delete IBKR configuration.

        Removes all IBKR Flex Query configuration.
        Does not delete imported transactions.
        """
        try:
            config_details = IBKRConfigService.delete_config()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR configuration deleted",
                details=config_details,
            )

            return {"success": True, "message": "Configuration deleted successfully"}, 200

        except ValueError as e:
            return {"error": str(e)}, 404

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to delete IBKR configuration",
                details={"error": str(e)},
            )
            return {"error": "Failed to delete configuration", "details": str(e)}, 500


@ns.route("/config/test")
class IBKRConfigTest(Resource):
    """IBKR configuration test endpoint."""

    @ns.doc("test_ibkr_connection")
    @ns.expect(ibkr_config_create_model)
    @ns.response(200, "Connection successful")
    @ns.response(400, "Connection failed", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Test IBKR connection with provided credentials.

        Tests the Flex Query connection without saving credentials.
        Useful for validating credentials before saving configuration.
        """
        data = request.get_json()

        if not data or "flex_token" not in data or "flex_query_id" not in data:
            return {"error": "Missing required fields: flex_token and flex_query_id"}, 400

        try:
            service = IBKRFlexService()
            result = service.test_connection(data["flex_token"], data["flex_query_id"])

            if result["success"]:
                return result, 200
            else:
                # Use the service-provided HTTP status code
                status_code = result.get("http_status", 400)
                return result, status_code

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error testing IBKR connection",
                details={"error": str(e)},
            )
            return {"error": "Connection test failed", "details": str(e)}, 500


@ns.route("/import")
class IBKRImport(Resource):
    """IBKR import trigger endpoint."""

    @ns.doc("trigger_ibkr_import")
    @ns.response(200, "Import successful")
    @ns.response(400, "IBKR not configured", error_model)
    @ns.response(403, "IBKR disabled", error_model)
    @ns.response(500, "Import failed", error_model)
    def post(self):
        """
        Manually trigger IBKR transaction import.

        Initiates an import of transactions from IBKR Flex Query.
        Normally this runs automatically on a schedule.

        The import process:
        1. Fetches transactions from IBKR Flex Query
        2. Parses and validates transaction data
        3. Stores new transactions in inbox
        4. Returns summary of imported transactions
        """
        config = IBKRConfigService.get_first_config()

        if not config:
            return {"error": "IBKR not configured"}, 400

        if not config.enabled:
            return {"error": "IBKR integration is disabled"}, 403

        try:
            service = IBKRFlexService()

            # Decrypt token
            token = service._decrypt_token(config.flex_token)

            # Fetch statement
            xml_data = service.fetch_statement(token, config.flex_query_id, use_cache=True)

            if not xml_data:
                return {"error": "Failed to fetch statement from IBKR"}, 500

            # Parse and import transactions
            result = service.parse_and_import_transactions(xml_data)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR import completed",
                details=result,
            )

            return result, 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="IBKR import failed",
                details={"error": str(e)},
            )
            return {"error": "Import failed", "details": str(e)}, 500


@ns.route("/inbox")
class IBKRInbox(Resource):
    """IBKR transaction inbox endpoint."""

    @ns.doc("list_ibkr_transactions")
    @ns.param("status", "Filter by status (pending, processed, ignored)", _in="query")
    @ns.param("transaction_type", "Filter by transaction type", _in="query")
    @ns.response(200, "Success", [ibkr_transaction_model])
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get all IBKR transactions in inbox.

        Returns a list of imported transactions from IBKR.
        By default shows pending transactions that need portfolio allocation.

        Query parameters:
        - status: Filter by status (pending, processed, ignored)
        - transaction_type: Filter by transaction type (buy, sell, dividend, etc.)
        """
        try:
            status = request.args.get("status", "pending")
            transaction_type = request.args.get("transaction_type")

            transactions = IBKRTransactionService.get_inbox(status, transaction_type)

            return transactions, 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error retrieving IBKR inbox",
                details={"error": str(e)},
            )
            return {"error": "Error retrieving inbox", "details": str(e)}, 500


@ns.route("/inbox/count")
class IBKRInboxCount(Resource):
    """IBKR inbox count endpoint."""

    @ns.doc("get_inbox_count")
    @ns.param("status", "Filter by status (default: pending)", _in="query")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get count of IBKR transactions.

        Returns the number of transactions matching the status filter.
        Useful for displaying inbox badge counts.

        Query parameters:
        - status: Filter by status (pending, processed, ignored) - defaults to 'pending'
        """
        try:
            status = request.args.get("status", "pending")
            count = IBKRTransactionService.get_inbox_count(status=status)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Retrieved IBKR inbox count for status '{status}'",
                details={"status": status, "count": count},
            )

            return {"count": count}, 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error retrieving IBKR inbox count",
                details={"error": str(e)},
            )
            return {"error": "Failed to retrieve count", "details": str(e)}, 500


@ns.route("/inbox/bulk-allocate")
class IBKRBulkAllocate(Resource):
    """IBKR bulk allocation endpoint."""

    @ns.doc("bulk_allocate_transactions")
    @ns.expect(allocation_request_model, validate=True)
    @ns.response(200, "Transactions allocated")
    @ns.response(400, "Invalid request", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self):
        """
        Allocate multiple IBKR transactions with same allocations.

        Processes multiple pending transactions using the same portfolio allocation percentages.
        Useful for batch processing similar transactions.

        Request body:
        {
            "transaction_ids": ["id1", "id2", ...],
            "allocations": [
                {"portfolio_id": "uuid", "percentage": 50.0},
                ...
            ]
        }
        """
        data = request.get_json()

        if not data or "transaction_ids" not in data or "allocations" not in data:
            return {"error": "Missing transaction_ids or allocations"}, 400

        transaction_ids = data["transaction_ids"]
        allocations = data["allocations"]

        result = IBKRTransactionService.bulk_allocate_transactions(transaction_ids, allocations)

        if result["success"]:
            return result, 200
        else:
            return result, 400


@ns.route("/portfolios")
class IBKRPortfolios(Resource):
    """IBKR portfolios list endpoint."""

    @ns.doc("get_portfolios_for_allocation")
    @ns.response(200, "Success")
    def get(self):
        """
        Get available portfolios for transaction allocation.

        Returns a list of active portfolios that can be used for
        allocating IBKR transactions.
        """
        from ..services.portfolio_service import PortfolioService

        portfolios = PortfolioService.get_active_portfolios()

        return [{"id": p.id, "name": p.name, "description": p.description} for p in portfolios], 200


@ns.route("/dividends/pending")
class IBKRPendingDividends(Resource):
    """IBKR pending dividends endpoint."""

    @ns.doc("get_pending_dividends")
    @ns.param("symbol", "Filter by symbol", _in="query")
    @ns.param("isin", "Filter by ISIN", _in="query")
    @ns.response(200, "Success")
    @ns.response(500, "Server error", error_model)
    def get(self):
        """
        Get pending dividend records for matching.

        Returns dividend records that can be matched to IBKR dividend transactions.
        Useful for reconciling dividend payments.

        Query parameters:
        - symbol: Filter by trading symbol (optional)
        - isin: Filter by ISIN (optional)
        """
        symbol = request.args.get("symbol")
        isin = request.args.get("isin")

        dividends = IBKRTransactionService.get_pending_dividends(symbol, isin)

        return dividends, 200


@ns.route("/inbox/<string:transaction_id>")
@ns.param("transaction_id", "Transaction unique identifier")
class IBKRTransaction(Resource):
    """IBKR transaction detail endpoint."""

    @ns.doc("get_ibkr_transaction")
    @ns.response(200, "Success", ibkr_transaction_model)
    @ns.response(404, "Transaction not found", error_model)
    def get(self, transaction_id):
        """
        Get IBKR transaction details.

        Returns detailed information about a specific IBKR transaction,
        including any existing portfolio allocations.
        """
        try:
            transaction_detail = IBKRTransactionService.get_transaction_detail(transaction_id)
            return transaction_detail, 200
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error retrieving IBKR transaction",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": "Error retrieving transaction", "details": str(e)}, 500

    @ns.doc("delete_ibkr_transaction")
    @ns.response(200, "Transaction deleted")
    @ns.response(400, "Cannot delete processed transaction", error_model)
    @ns.response(404, "Transaction not found", error_model)
    def delete(self, transaction_id):
        """
        Delete an IBKR transaction.

        Removes a transaction from the inbox.
        Cannot delete transactions that have already been processed.
        """
        try:
            response, status = IBKRTransactionService.delete_transaction(transaction_id)
            return response, status
        except HTTPException:
            raise
        except Exception as e:
            return {"error": "Error deleting transaction", "details": str(e)}, 500


@ns.route("/inbox/<string:transaction_id>/allocate")
@ns.param("transaction_id", "Transaction unique identifier")
class IBKRTransactionAllocate(Resource):
    """IBKR transaction allocation endpoint."""

    @ns.doc("allocate_transaction")
    @ns.expect(allocation_request_model, validate=True)
    @ns.response(200, "Transaction allocated")
    @ns.response(400, "Invalid allocation", error_model)
    @ns.response(404, "Transaction not found", error_model)
    @ns.response(500, "Server error", error_model)
    def post(self, transaction_id):
        """
        Allocate IBKR transaction to portfolios.

        Allocates an imported IBKR transaction to one or more portfolios
        with specified percentages.

        Validation rules:
        - Allocations must sum to 100% (±0.01 tolerance)
        - Each allocation percentage must be > 0
        - Portfolio must exist
        - Transaction must be in 'pending' status

        Creates:
        - Fund record (if doesn't exist)
        - Portfolio-Fund relationship (if doesn't exist)
        - Transaction records in each portfolio
        """
        try:
            data = request.get_json()
            allocations = data.get("allocations", [])

            if not allocations:
                return {"error": "No allocations provided"}, 400

            result = IBKRTransactionService.process_transaction_allocation(
                transaction_id, allocations
            )

            if result["success"]:
                return result, 200
            else:
                return result, 400

        except ValueError as e:
            # Return 404 if it's a "not found" error, otherwise 400 for validation errors
            error_message = str(e)
            if "not found" in error_message.lower():
                return {"error": error_message}, 404
            return {"error": error_message}, 400
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error allocating transaction",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": "Error allocating transaction", "details": str(e)}, 500


@ns.route("/inbox/<string:transaction_id>/ignore")
@ns.param("transaction_id", "Transaction unique identifier")
class IBKRTransactionIgnore(Resource):
    """IBKR transaction ignore endpoint."""

    @ns.doc("ignore_transaction")
    @ns.response(200, "Transaction ignored")
    @ns.response(400, "Cannot ignore processed transaction", error_model)
    @ns.response(404, "Transaction not found", error_model)
    def post(self, transaction_id):
        """
        Mark IBKR transaction as ignored.

        Marks a transaction as ignored, removing it from the pending inbox
        without creating portfolio transactions.

        Useful for:
        - Duplicate transactions
        - Transactions not relevant to tracked portfolios
        - Test transactions
        """
        try:
            response, status = IBKRTransactionService.ignore_transaction(transaction_id)
            return response, status
        except HTTPException:
            raise
        except Exception as e:
            return {"error": "Error ignoring transaction", "details": str(e)}, 500


@ns.route("/inbox/<string:transaction_id>/eligible-portfolios")
@ns.param("transaction_id", "Transaction unique identifier")
class IBKRTransactionEligiblePortfolios(Resource):
    """IBKR transaction eligible portfolios endpoint."""

    @ns.doc("get_eligible_portfolios")
    @ns.response(200, "Success")
    @ns.response(404, "Transaction not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, transaction_id):
        """
        Get eligible portfolios for allocating this transaction.

        Returns portfolios that have the fund/stock matching this transaction
        (by ISIN or symbol).

        Response includes:
        - match_info: Details about fund matching
        - portfolios: List of eligible portfolios
        - warning: Optional warning if no match or no portfolios
        """
        from werkzeug.exceptions import HTTPException

        from ..services.fund_matching_service import FundMatchingService

        try:
            # Get the transaction using the service
            transaction = IBKRTransactionService.get_transaction(transaction_id)

            # Find eligible portfolios using the matching service
            result = FundMatchingService.get_eligible_portfolios_for_transaction(transaction)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Retrieved eligible portfolios for transaction {transaction_id}",
                details={
                    "fund_found": result["match_info"]["found"],
                    "portfolio_count": len(result["portfolios"]),
                    "matched_by": result["match_info"]["matched_by"],
                },
            )

            return result, 200

        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message=f"Error getting eligible portfolios: {e!s}",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": "Error getting eligible portfolios", "details": str(e)}, 500


@ns.route("/inbox/<string:transaction_id>/match-dividend")
@ns.param("transaction_id", "Transaction unique identifier")
class IBKRTransactionMatchDividend(Resource):
    """IBKR dividend matching endpoint."""

    @ns.doc("match_dividend")
    @ns.response(200, "Dividend matched")
    @ns.response(400, "Invalid request", error_model)
    @ns.response(404, "Transaction not found", error_model)
    def post(self, transaction_id):
        """
        Match IBKR dividend transaction to existing dividend records.

        Associates an imported IBKR dividend transaction with one or more
        existing Dividend records for reconciliation.

        Request body:
        {
            "dividend_ids": ["id1", "id2", ...]
        }
        """
        data = request.get_json()

        if not data or "dividend_ids" not in data:
            return {"error": "Missing dividend_ids"}, 400

        result = IBKRTransactionService.match_dividend(transaction_id, data["dividend_ids"])

        if result["success"]:
            return result, 200
        else:
            return result, 400


@ns.route("/inbox/<string:transaction_id>/unallocate")
@ns.param("transaction_id", "Transaction unique identifier")
class IBKRTransactionUnallocate(Resource):
    """IBKR transaction unallocation endpoint."""

    @ns.doc("unallocate_transaction")
    @ns.response(200, "Transaction unallocated")
    @ns.response(400, "Cannot unallocate", error_model)
    @ns.response(404, "Transaction not found", error_model)
    def post(self, transaction_id):
        """
        Unallocate a processed IBKR transaction.

        Deletes all portfolio transactions and allocations,
        reverting the IBKR transaction status to pending.

        Useful for correcting allocation mistakes.
        """
        response, status = IBKRTransactionService.unallocate_transaction(transaction_id)
        return response, status


@ns.route("/inbox/<string:transaction_id>/allocations")
@ns.param("transaction_id", "Transaction unique identifier")
class IBKRTransactionAllocations(Resource):
    """IBKR transaction allocations management endpoint."""

    @ns.doc("get_transaction_allocations")
    @ns.response(200, "Success")
    @ns.response(404, "Transaction not found", error_model)
    def get(self, transaction_id):
        """
        Get allocation details for a processed IBKR transaction.

        Returns allocation information grouped by portfolio,
        combining stock and fee transactions.
        """
        response, status = IBKRTransactionService.get_transaction_allocations(transaction_id)
        return response, status

    @ns.doc("modify_transaction_allocations")
    @ns.expect(allocation_request_model, validate=True)
    @ns.response(200, "Allocations modified")
    @ns.response(400, "Invalid request", error_model)
    @ns.response(404, "Transaction not found", error_model)
    def put(self, transaction_id):
        """
        Modify allocation percentages for a processed IBKR transaction.

        Updates the allocation percentages without unallocating and re-allocating.
        This preserves the transaction while adjusting the distribution.

        Request body:
        {
            "allocations": [
                {"portfolio_id": "uuid", "percentage": 60.0},
                {"portfolio_id": "uuid", "percentage": 40.0}
            ]
        }
        """
        data = request.get_json()

        if not data or "allocations" not in data:
            return {"error": "Missing allocations"}, 400

        try:
            result = IBKRTransactionService.modify_allocations(transaction_id, data["allocations"])
            return result, 200

        except ValueError as e:
            # Return 404 if it's a "not found" error, otherwise 400 for validation errors
            error_message = str(e)
            if "not found" in error_message.lower():
                return {"error": error_message}, 404
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to modify allocations",
                details={"transaction_id": transaction_id, "error": error_message},
            )
            return {"error": error_message}, 400
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Unexpected error modifying allocations",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": "Unexpected error", "details": str(e)}, 500
