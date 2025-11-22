"""
Service for processing IBKR transactions and allocations.

This module handles:
- Validating allocations
- Creating Fund and PortfolioFund records
- Creating Transaction records
- Processing transaction allocations
- Dividend matching
"""

from datetime import UTC, datetime

from ..models import (
    Dividend,
    Fund,
    IBKRTransaction,
    IBKRTransactionAllocation,
    InvestmentType,
    LogCategory,
    LogLevel,
    Portfolio,
    PortfolioFund,
    ReinvestmentStatus,
    Transaction,
    db,
)
from ..services.logging_service import logger


class IBKRTransactionService:
    """
    Service class for processing IBKR transactions and allocations.

    Provides methods for:
    - Validating allocations
    - Processing transactions with user allocations
    - Creating Transaction records across portfolios
    - Matching dividends to existing records
    """

    @staticmethod
    def get_transaction(transaction_id: str) -> IBKRTransaction:
        """
        Retrieve an IBKR transaction by ID.

        Args:
            transaction_id: Transaction ID

        Returns:
            IBKRTransaction: The transaction object

        Raises:
            404: If transaction not found
        """
        txn = db.session.get(IBKRTransaction, transaction_id)
        if not txn:
            from flask import abort

            abort(404)
        return txn

    @staticmethod
    def ignore_transaction(transaction_id: str) -> tuple[dict, int]:
        """
        Mark an IBKR transaction as ignored.

        Args:
            transaction_id: Transaction ID

        Returns:
            tuple: (response dict, status code)
        """
        txn = IBKRTransactionService.get_transaction(transaction_id)

        if txn.status == "processed":
            return {"error": "Cannot ignore processed transaction"}, 400

        try:
            txn.status = "ignored"
            txn.processed_at = datetime.now()
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Transaction marked as ignored: {txn.ibkr_transaction_id}",
                details={"transaction_id": transaction_id},
            )

            return {"success": True, "message": "Transaction marked as ignored"}, 200

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to ignore transaction",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": "Failed to ignore transaction", "details": str(e)}, 500

    @staticmethod
    def delete_transaction(transaction_id: str) -> tuple[dict, int]:
        """
        Delete an IBKR transaction (only if not processed).

        Args:
            transaction_id: Transaction ID

        Returns:
            tuple: (response dict, status code)
        """
        txn = IBKRTransactionService.get_transaction(transaction_id)

        if txn.status == "processed":
            return {"error": "Cannot delete processed transaction"}, 400

        try:
            # Store transaction ID before deletion for logging
            ibkr_transaction_id = txn.ibkr_transaction_id

            db.session.delete(txn)
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Transaction deleted: {ibkr_transaction_id}",
                details={"transaction_id": transaction_id},
            )

            return {"success": True, "message": "Transaction deleted"}, 200

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to delete transaction",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            return {"error": "Failed to delete transaction", "details": str(e)}, 500

    @staticmethod
    def validate_allocations(allocations: list[dict]) -> tuple[bool, str]:
        """
        Validate that allocations sum to 100%.

        Args:
            allocations: List of allocation dictionaries

        Returns:
            Tuple of (is_valid, error_message)
        """
        if not allocations:
            return False, "At least one allocation is required"

        total_percentage = sum(alloc["percentage"] for alloc in allocations)

        # Allow small floating point error
        if abs(total_percentage - 100.0) > 0.01:
            return False, f"Allocations must sum to 100% (currently {total_percentage}%)"

        # Validate individual allocations
        for alloc in allocations:
            if alloc["percentage"] <= 0:
                return False, "Allocation percentages must be positive"

            if "portfolio_id" not in alloc:
                return False, "Each allocation must specify a portfolio_id"

        return True, ""

    @staticmethod
    def _get_or_create_fund(symbol: str, isin: str, currency: str) -> Fund:
        """
        Get or create Fund record.

        Args:
            symbol: Trading symbol
            isin: ISIN code
            currency: Currency code

        Returns:
            Fund object
        """
        # Try to find by ISIN first
        if isin:
            fund = Fund.query.filter_by(isin=isin).first()
            if fund:
                return fund

        # Try to find by symbol
        if symbol:
            fund = Fund.query.filter_by(symbol=symbol).first()
            if fund:
                return fund

        # Create new fund
        fund = Fund(
            name=symbol or isin,  # Use symbol as name, fallback to ISIN
            isin=isin or f"UNKNOWN_{symbol}",  # ISIN is required, use placeholder
            symbol=symbol,
            currency=currency,
            exchange="UNKNOWN",  # Will be updated when user provides info
            investment_type=InvestmentType.STOCK,  # Default to stock for IBKR imports
        )
        db.session.add(fund)
        db.session.flush()  # Get ID without committing

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Created new fund: {fund.name}",
            details={"fund_id": fund.id, "symbol": symbol, "isin": isin},
        )

        return fund

    @staticmethod
    def _get_or_create_portfolio_fund(portfolio_id: str, fund_id: str) -> PortfolioFund:
        """
        Get or create PortfolioFund relationship.

        Args:
            portfolio_id: Portfolio ID
            fund_id: Fund ID

        Returns:
            PortfolioFund object
        """
        portfolio_fund = PortfolioFund.query.filter_by(
            portfolio_id=portfolio_id, fund_id=fund_id
        ).first()

        if not portfolio_fund:
            portfolio_fund = PortfolioFund(portfolio_id=portfolio_id, fund_id=fund_id)
            db.session.add(portfolio_fund)
            db.session.flush()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="Created new portfolio-fund relationship",
                details={"portfolio_id": portfolio_id, "fund_id": fund_id},
            )

        return portfolio_fund

    @staticmethod
    def process_transaction_allocation(ibkr_transaction_id: str, allocations: list[dict]) -> dict:
        """
        Process IBKR transaction with user-provided allocations.

        Args:
            ibkr_transaction_id: IBKR transaction ID
            allocations: List of allocation dictionaries with:
                - portfolio_id: Portfolio ID
                - percentage: Allocation percentage

        Returns:
            Dictionary with processing results
        """
        # Get IBKR transaction
        ibkr_txn = db.session.get(IBKRTransaction, ibkr_transaction_id)
        if not ibkr_txn:
            return {"success": False, "error": "Transaction not found"}

        # Check if already processed
        if ibkr_txn.status == "processed":
            return {"success": False, "error": "Transaction already processed"}

        # Validate allocations
        is_valid, error_message = IBKRTransactionService.validate_allocations(allocations)
        if not is_valid:
            return {"success": False, "error": error_message}

        try:
            # Get or create fund
            fund = IBKRTransactionService._get_or_create_fund(
                ibkr_txn.symbol, ibkr_txn.isin, ibkr_txn.currency
            )

            created_transactions = []

            # Process each allocation
            for alloc in allocations:
                portfolio_id = alloc["portfolio_id"]
                percentage = alloc["percentage"]

                # Verify portfolio exists
                portfolio = db.session.get(Portfolio, portfolio_id)
                if not portfolio:
                    raise ValueError(f"Portfolio not found: {portfolio_id}")

                # Calculate allocated amounts
                allocated_amount = (ibkr_txn.total_amount * percentage) / 100.0
                allocated_shares = (
                    (ibkr_txn.quantity * percentage / 100.0) if ibkr_txn.quantity else 0
                )

                # Get or create portfolio-fund relationship
                portfolio_fund = IBKRTransactionService._get_or_create_portfolio_fund(
                    portfolio_id, fund.id
                )

                # Create Transaction record (skip for fee transactions without shares)
                transaction = None
                if ibkr_txn.transaction_type != "fee":
                    # Calculate cost per share
                    cost_per_share = (
                        ibkr_txn.price
                        if ibkr_txn.price
                        else (allocated_amount / allocated_shares if allocated_shares > 0 else 0)
                    )

                    transaction = Transaction(
                        portfolio_fund_id=portfolio_fund.id,
                        date=ibkr_txn.transaction_date,
                        type=ibkr_txn.transaction_type,
                        shares=allocated_shares,
                        cost_per_share=cost_per_share,
                    )
                    db.session.add(transaction)
                    db.session.flush()  # Get transaction ID

                    created_transactions.append(
                        {
                            "transaction_id": transaction.id,
                            "portfolio_name": portfolio.name,
                            "shares": allocated_shares,
                            "amount": allocated_amount,
                        }
                    )

                # Create fee transaction if commission exists
                fee_transaction = None
                if ibkr_txn.fees and ibkr_txn.fees > 0:
                    allocated_fee = (ibkr_txn.fees * percentage) / 100.0

                    fee_transaction = Transaction(
                        portfolio_fund_id=portfolio_fund.id,
                        date=ibkr_txn.transaction_date,
                        type="fee",
                        shares=0,  # Fee transactions have no shares
                        cost_per_share=allocated_fee,  # Store fee amount in cost_per_share
                    )
                    db.session.add(fee_transaction)
                    db.session.flush()  # Get transaction ID

                    created_transactions.append(
                        {
                            "transaction_id": fee_transaction.id,
                            "portfolio_name": portfolio.name,
                            "shares": 0,
                            "amount": allocated_fee,
                            "type": "fee",
                        }
                    )

                # Create allocation record for main transaction
                allocation_record = IBKRTransactionAllocation(
                    ibkr_transaction_id=ibkr_txn.id,
                    portfolio_id=portfolio_id,
                    allocation_percentage=percentage,
                    allocated_amount=allocated_amount,
                    allocated_shares=allocated_shares,
                    transaction_id=transaction.id if transaction else None,
                )
                db.session.add(allocation_record)

                # Create allocation record for fee transaction to link it to IBKR
                if fee_transaction:
                    fee_allocation_record = IBKRTransactionAllocation(
                        ibkr_transaction_id=ibkr_txn.id,
                        portfolio_id=portfolio_id,
                        allocation_percentage=percentage,
                        allocated_amount=allocated_fee,  # Fee amount
                        allocated_shares=0,  # No shares for fees
                        transaction_id=fee_transaction.id,
                    )
                    db.session.add(fee_allocation_record)

            # Update IBKR transaction status
            ibkr_txn.status = "processed"
            ibkr_txn.processed_at = datetime.now(UTC)

            # Commit all changes
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Successfully processed IBKR transaction: {ibkr_txn.ibkr_transaction_id}",
                details={
                    "ibkr_transaction_id": ibkr_txn.ibkr_transaction_id,
                    "transaction_count": len(created_transactions),
                    "allocations": allocations,
                },
            )

            return {
                "success": True,
                "message": "Transaction processed successfully",
                "created_transactions": created_transactions,
                "fund_id": fund.id,
                "fund_name": fund.name,
            }

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to process IBKR transaction",
                details={"ibkr_transaction_id": ibkr_transaction_id, "error": str(e)},
            )
            return {"success": False, "error": f"Processing failed: {e!s}"}

    @staticmethod
    def modify_allocations(transaction_id: str, allocations: list[dict]) -> dict:
        """
        Modify allocation percentages for a processed IBKR transaction.

        This method allows updating allocations after a transaction has been processed.
        It will:
        - Validate that allocations sum to 100%
        - Delete allocations (and their transactions) that are removed
        - Update existing allocations and their associated transactions
        - Create new allocations and transactions for new portfolios

        Args:
            transaction_id: IBKR Transaction ID
            allocations: List of allocation dictionaries with:
                - portfolio_id: Portfolio ID
                - percentage: Allocation percentage

        Returns:
            Dictionary with:
                - success: Boolean indicating success/failure
                - message: Success/error message
                - error: Error details (if applicable)

        Raises:
            ValueError: If transaction not found, not processed, or allocations invalid
        """
        # Get IBKR transaction
        ibkr_txn = db.session.get(IBKRTransaction, transaction_id)
        if not ibkr_txn:
            raise ValueError(f"Transaction {transaction_id} not found")

        if ibkr_txn.status != "processed":
            raise ValueError("Transaction is not processed")

        # Validate allocations
        is_valid, error_message = IBKRTransactionService.validate_allocations(allocations)
        if not is_valid:
            raise ValueError(error_message)

        try:
            # Get existing allocations
            existing_allocations = {
                alloc.portfolio_id: alloc
                for alloc in IBKRTransactionAllocation.query.filter_by(
                    ibkr_transaction_id=transaction_id
                ).all()
            }

            # Track which portfolios are in the new allocation list
            new_portfolio_ids = {a["portfolio_id"] for a in allocations}
            existing_portfolio_ids = set(existing_allocations.keys())

            # Delete allocations for portfolios no longer in the list
            portfolios_to_remove = existing_portfolio_ids - new_portfolio_ids
            for portfolio_id in portfolios_to_remove:
                allocation = existing_allocations[portfolio_id]
                if allocation.transaction_id:
                    transaction = db.session.get(Transaction, allocation.transaction_id)
                    if transaction:
                        # Also delete any fee transactions for this portfolio_fund
                        fee_transactions = Transaction.query.filter_by(
                            portfolio_fund_id=transaction.portfolio_fund_id,
                            date=ibkr_txn.transaction_date,
                            type="fee",
                        ).all()
                        for fee_txn in fee_transactions:
                            db.session.delete(fee_txn)

                        # Delete main transaction - CASCADE DELETE will automatically
                        # delete the corresponding ibkr_transaction_allocation record
                        db.session.delete(transaction)
                else:
                    # If no transaction_id, delete the allocation directly
                    db.session.delete(allocation)

            # Find fund for this IBKR transaction
            from ..services.fund_matching_service import FundMatchingService

            fund = FundMatchingService.find_fund_by_transaction(ibkr_txn)
            if not fund:
                raise ValueError("Fund not found for this IBKR transaction")

            # Update or create allocations
            for alloc_data in allocations:
                portfolio_id = alloc_data["portfolio_id"]
                percentage = alloc_data["percentage"]

                # Calculate allocated amounts
                allocated_amount = (ibkr_txn.total_amount * percentage) / 100
                allocated_shares = (
                    (ibkr_txn.quantity * percentage / 100) if ibkr_txn.quantity else 0
                )

                if portfolio_id in existing_allocations:
                    # Update existing allocation
                    allocation = existing_allocations[portfolio_id]
                    allocation.allocation_percentage = percentage
                    allocation.allocated_amount = allocated_amount
                    allocation.allocated_shares = allocated_shares

                    # Update associated transaction
                    if allocation.transaction_id:
                        transaction = db.session.get(Transaction, allocation.transaction_id)
                        if transaction:
                            transaction.shares = allocated_shares
                            # Cost per share stays the same, shares change

                            # Update fee transaction if commission exists
                            if ibkr_txn.fees and ibkr_txn.fees > 0:
                                allocated_fee = (ibkr_txn.fees * percentage) / 100.0

                                # Find existing fee transaction
                                fee_transaction = Transaction.query.filter_by(
                                    portfolio_fund_id=transaction.portfolio_fund_id,
                                    date=ibkr_txn.transaction_date,
                                    type="fee",
                                ).first()

                                if fee_transaction:
                                    # Update existing fee transaction
                                    fee_transaction.cost_per_share = allocated_fee
                                    # Update the IBKR allocation record for fee transaction
                                    fee_allocation = IBKRTransactionAllocation.query.filter_by(
                                        transaction_id=fee_transaction.id
                                    ).first()
                                    if fee_allocation:
                                        fee_allocation.allocated_amount = allocated_fee
                                else:
                                    # Create new fee transaction if it doesn't exist
                                    fee_transaction = Transaction(
                                        portfolio_fund_id=transaction.portfolio_fund_id,
                                        date=ibkr_txn.transaction_date,
                                        type="fee",
                                        shares=0,
                                        cost_per_share=allocated_fee,
                                    )
                                    db.session.add(fee_transaction)
                                    db.session.flush()

                                    # Create IBKR allocation for new fee transaction
                                    fee_allocation = IBKRTransactionAllocation(
                                        ibkr_transaction_id=transaction_id,
                                        portfolio_id=portfolio_id,
                                        allocation_percentage=percentage,
                                        allocated_amount=allocated_fee,
                                        allocated_shares=0,
                                        transaction_id=fee_transaction.id,
                                    )
                                    db.session.add(fee_allocation)
                else:
                    # Create new allocation
                    # Get or create portfolio fund
                    portfolio_fund = IBKRTransactionService._get_or_create_portfolio_fund(
                        portfolio_id, fund.id
                    )

                    # Create transaction
                    transaction = Transaction(
                        portfolio_fund_id=portfolio_fund.id,
                        date=ibkr_txn.transaction_date,
                        type=ibkr_txn.transaction_type,
                        shares=allocated_shares,
                        cost_per_share=ibkr_txn.price if ibkr_txn.price else 0,
                    )
                    db.session.add(transaction)
                    db.session.flush()

                    # Create fee transaction if commission exists
                    fee_transaction = None
                    if ibkr_txn.fees and ibkr_txn.fees > 0:
                        allocated_fee = (ibkr_txn.fees * percentage) / 100.0

                        fee_transaction = Transaction(
                            portfolio_fund_id=portfolio_fund.id,
                            date=ibkr_txn.transaction_date,
                            type="fee",
                            shares=0,
                            cost_per_share=allocated_fee,
                        )
                        db.session.add(fee_transaction)
                        db.session.flush()

                    # Create allocation for main transaction
                    allocation = IBKRTransactionAllocation(
                        ibkr_transaction_id=transaction_id,
                        portfolio_id=portfolio_id,
                        allocation_percentage=percentage,
                        allocated_amount=allocated_amount,
                        allocated_shares=allocated_shares,
                        transaction_id=transaction.id,
                    )
                    db.session.add(allocation)

                    # Create allocation for fee transaction
                    if fee_transaction:
                        fee_allocation = IBKRTransactionAllocation(
                            ibkr_transaction_id=transaction_id,
                            portfolio_id=portfolio_id,
                            allocation_percentage=percentage,
                            allocated_amount=allocated_fee,
                            allocated_shares=0,
                            transaction_id=fee_transaction.id,
                        )
                        db.session.add(fee_allocation)

            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=(
                    f"Modified allocations for IBKR transaction: {ibkr_txn.ibkr_transaction_id}"
                ),
                details={
                    "allocation_count": len(allocations),
                    "portfolios": list(new_portfolio_ids),
                },
            )

            return {
                "success": True,
                "message": "Allocations modified successfully",
            }

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to modify allocations",
                details={"transaction_id": transaction_id, "error": str(e)},
            )
            raise ValueError(f"Failed to modify allocations: {e!s}") from e

    @staticmethod
    def get_pending_dividends(symbol: str | None = None, isin: str | None = None) -> list[dict]:
        """
        Get pending dividend records for matching.

        Args:
            symbol: Filter by symbol (optional)
            isin: Filter by ISIN (optional)

        Returns:
            List of pending dividend records
        """
        query = Dividend.query.filter_by(reinvestment_status=ReinvestmentStatus.PENDING)

        # Filter by fund if symbol/ISIN provided
        if symbol or isin:
            fund_query = Fund.query
            if isin:
                fund_query = fund_query.filter_by(isin=isin)
            elif symbol:
                fund_query = fund_query.filter_by(symbol=symbol)

            funds = fund_query.all()
            fund_ids = [f.id for f in funds]
            query = query.filter(Dividend.fund_id.in_(fund_ids))

        dividends = query.all()

        return [
            {
                "id": div.id,
                "fund_id": div.fund_id,
                "portfolio_fund_id": div.portfolio_fund_id,
                "record_date": div.record_date.isoformat(),
                "ex_dividend_date": div.ex_dividend_date.isoformat(),
                "shares_owned": div.shares_owned,
                "dividend_per_share": div.dividend_per_share,
                "total_amount": div.total_amount,
            }
            for div in dividends
        ]

    @staticmethod
    def match_dividend(ibkr_transaction_id: str, dividend_ids: list[str]) -> dict:
        """
        Match IBKR dividend transaction to existing Dividend records.

        Args:
            ibkr_transaction_id: IBKR transaction ID
            dividend_ids: List of Dividend IDs to match

        Returns:
            Dictionary with matching results
        """
        # Get IBKR transaction
        ibkr_txn = db.session.get(IBKRTransaction, ibkr_transaction_id)
        if not ibkr_txn:
            return {"success": False, "error": "Transaction not found"}

        if ibkr_txn.transaction_type != "dividend":
            return {"success": False, "error": "Transaction is not a dividend"}

        if ibkr_txn.status == "processed":
            return {"success": False, "error": "Transaction already processed"}

        try:
            # Get dividend records
            dividends = Dividend.query.filter(Dividend.id.in_(dividend_ids)).all()

            if not dividends:
                return {"success": False, "error": "No dividends found"}

            # Update dividend records with total amount from IBKR
            total_shares = sum(div.shares_owned for div in dividends)
            for dividend in dividends:
                # Allocate IBKR amount proportionally based on shares
                dividend.total_amount = ibkr_txn.total_amount * dividend.shares_owned / total_shares

            # Mark IBKR transaction as processed (dividends handled separately)
            ibkr_txn.status = "processed"
            ibkr_txn.processed_at = datetime.now(UTC)

            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Matched dividend transaction: {ibkr_txn.ibkr_transaction_id}",
                details={
                    "ibkr_transaction_id": ibkr_txn.ibkr_transaction_id,
                    "dividend_count": len(dividends),
                    "total_amount": ibkr_txn.total_amount,
                },
            )

            return {
                "success": True,
                "message": "Dividend matched successfully",
                "updated_dividends": len(dividends),
            }

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to match dividend",
                details={"ibkr_transaction_id": ibkr_transaction_id, "error": str(e)},
            )
            return {"success": False, "error": f"Matching failed: {e!s}"}

    @staticmethod
    def get_inbox(status: str = "pending", transaction_type: str | None = None) -> list[dict]:
        """
        Get IBKR inbox transactions with optional filtering.

        Args:
            status: Filter by status (default: "pending")
            transaction_type: Filter by transaction type (optional)

        Returns:
            list[dict]: List of transaction dictionaries with serialized data
        """
        query = IBKRTransaction.query

        # Apply status filter
        if status:
            query = query.filter_by(status=status)

        # Apply transaction type filter
        if transaction_type:
            query = query.filter_by(transaction_type=transaction_type)

        # Order by date descending
        transactions = query.order_by(IBKRTransaction.transaction_date.desc()).all()

        return [
            {
                "id": txn.id,
                "ibkr_transaction_id": txn.ibkr_transaction_id,
                "transaction_date": txn.transaction_date.isoformat(),
                "symbol": txn.symbol,
                "isin": txn.isin,
                "description": txn.description,
                "transaction_type": txn.transaction_type,
                "quantity": txn.quantity,
                "price": txn.price,
                "total_amount": txn.total_amount,
                "currency": txn.currency,
                "fees": txn.fees,
                "status": txn.status,
                "imported_at": txn.imported_at.isoformat(),
            }
            for txn in transactions
        ]

    @staticmethod
    def get_inbox_count(status: str = "pending") -> int:
        """
        Get count of IBKR transactions by status.

        Args:
            status: Filter by status (default: "pending")

        Returns:
            int: Count of transactions matching the status
        """
        return IBKRTransaction.query.filter_by(status=status).count()

    @staticmethod
    def unallocate_transaction(transaction_id: str) -> tuple[dict, int]:
        """
        Unallocate a processed IBKR transaction.

        This deletes all portfolio transactions and allocations,
        reverting the IBKR transaction status to pending.

        Args:
            transaction_id: IBKR Transaction ID

        Returns:
            tuple: (response dict, status code)
        """
        try:
            ibkr_txn = db.session.get(IBKRTransaction, transaction_id)
            if not ibkr_txn:
                return {"error": "Transaction not found"}, 404

            if ibkr_txn.status != "processed":
                return {"error": "Transaction is not processed"}, 400

            # Get all allocations
            allocations = IBKRTransactionAllocation.query.filter_by(
                ibkr_transaction_id=transaction_id
            ).all()

            deleted_count = 0

            # Delete all associated transactions - CASCADE will handle allocations
            for allocation in allocations:
                if allocation.transaction_id:
                    transaction = db.session.get(Transaction, allocation.transaction_id)
                    if transaction:
                        # Delete transaction - CASCADE DELETE will automatically
                        # delete the corresponding ibkr_transaction_allocation record
                        db.session.delete(transaction)
                        deleted_count += 1
                else:
                    # Delete orphaned allocations (allocations without transactions)
                    db.session.delete(allocation)

            # Revert IBKR transaction status
            ibkr_txn.status = "pending"
            ibkr_txn.processed_at = None

            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Unallocated IBKR transaction: {ibkr_txn.ibkr_transaction_id}",
                details={
                    "ibkr_transaction_id": ibkr_txn.ibkr_transaction_id,
                    "deleted_transactions": deleted_count,
                },
            )

            return (
                {
                    "success": True,
                    "message": (
                        f"Transaction unallocated successfully. "
                        f"{deleted_count} portfolio transactions deleted."
                    ),
                },
                200,
            )

        except Exception as e:
            db.session.rollback()
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to unallocate transaction",
                details={"transaction_id": transaction_id, "error": str(e)},
                http_status=500,
            )
            return response, status

    @staticmethod
    def get_transaction_allocations(transaction_id: str) -> tuple[dict, int]:
        """
        Get allocation details for a processed IBKR transaction.

        Groups allocations by portfolio to combine stock and fee transactions.

        Args:
            transaction_id: IBKR Transaction ID

        Returns:
            tuple: (response dict with allocation details, status code)
        """
        try:
            ibkr_txn = db.session.get(IBKRTransaction, transaction_id)
            if not ibkr_txn:
                return {"error": "Transaction not found"}, 404

            # Get grouped allocations (combines stock and fee transactions per portfolio)
            allocation_details = IBKRTransactionService.get_grouped_allocations(transaction_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Retrieved allocations for IBKR transaction {transaction_id}",
                details={"portfolio_count": len(allocation_details)},
            )

            return (
                {
                    "ibkr_transaction_id": ibkr_txn.id,
                    "status": ibkr_txn.status,
                    "allocations": allocation_details,
                },
                200,
            )

        except Exception as e:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to retrieve allocations",
                details={"transaction_id": transaction_id, "error": str(e)},
                http_status=500,
            )
            return response, status
