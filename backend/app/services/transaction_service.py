"""
Service class for transaction-related operations.

This module provides methods for:
- Creating and managing transactions
- Retrieving transaction history
- Formatting transaction data
"""

from datetime import datetime

from ..models import Dividend, PortfolioFund, RealizedGainLoss, Transaction, db


class TransactionService:
    """
    Service class for transaction-related operations.

    Provides methods for:
    - Creating and managing transactions
    - Retrieving transaction history
    - Formatting transaction data
    """

    @staticmethod
    def get_all_transactions():
        """
        Retrieve all transactions from the database.

        Returns:
            list: List of formatted transaction dictionaries
        """
        transactions = Transaction.query.all()
        return [TransactionService.format_transaction(t) for t in transactions]

    @staticmethod
    def get_portfolio_transactions(portfolio_id):
        """
        Retrieve all transactions for a specific portfolio.

        Uses batch loading for IBKR allocations to eliminate N+1 queries.

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            list: List of formatted transaction dictionaries
        """
        from ..models import Fund, IBKRTransactionAllocation

        # Get all portfolio_funds for this portfolio
        portfolio_funds = PortfolioFund.query.filter_by(portfolio_id=portfolio_id).all()
        portfolio_fund_ids = [pf.id for pf in portfolio_funds]

        # Build portfolio_fund lookup with fund names
        portfolio_fund_lookup = {}
        fund_ids = [pf.fund_id for pf in portfolio_funds]

        # Batch load all funds
        funds = Fund.query.filter(Fund.id.in_(fund_ids)).all()
        fund_lookup = {f.id: f.name for f in funds}

        # Build portfolio_fund lookup
        for pf in portfolio_funds:
            portfolio_fund_lookup[pf.id] = fund_lookup.get(pf.fund_id, "Unknown")

        # Load transactions without eager loading
        transactions = Transaction.query.filter(
            Transaction.portfolio_fund_id.in_(portfolio_fund_ids)
        ).all()

        # Batch load IBKR allocations for all transactions
        transaction_ids = [t.id for t in transactions]
        ibkr_allocations = {}

        if transaction_ids:
            allocations = IBKRTransactionAllocation.query.filter(
                IBKRTransactionAllocation.transaction_id.in_(transaction_ids)
            ).all()

            # Create lookup dictionary
            for allocation in allocations:
                ibkr_allocations[allocation.transaction_id] = allocation

        # Format transactions with pre-loaded data
        return [
            TransactionService.format_transaction(
                t, ibkr_allocations.get(t.id), portfolio_fund_lookup, batch_mode=True
            )
            for t in transactions
        ]

    @staticmethod
    def format_transaction(
        transaction, ibkr_allocation=None, portfolio_fund_lookup=None, batch_mode=False
    ):
        """
        Format a transaction object into a dictionary.

        Args:
            transaction (Transaction): Transaction object
            ibkr_allocation (IBKRTransactionAllocation, optional): Pre-loaded IBKR
                allocation to avoid N+1 queries. If None and batch_mode is False,
                will query database (for backwards compatibility).
            portfolio_fund_lookup (dict, optional): Pre-loaded portfolio_fund_id to
                fund_name mapping to avoid N+1 queries. If None and batch_mode is
                False, will access relationships (for backwards compatibility).
            batch_mode (bool, optional): If True, skip database queries for missing
                data. Use when calling with pre-loaded data to avoid N+1 queries.
                Default: False.

        Returns:
            dict: Formatted transaction containing:
                - id: Transaction ID
                - portfolio_fund_id: Portfolio fund ID
                - fund_name: Fund name
                - date: ISO format date
                - type: Transaction type
                - shares: Number of shares
                - cost_per_share: Cost per share
                - ibkr_linked: Boolean indicating if transaction came from IBKR
                - ibkr_transaction_id: ID of parent IBKR transaction (if applicable)
        """
        # If IBKR allocation not provided, query it (backwards compatibility)
        # Skip querying if in batch mode to avoid N+1 queries
        if ibkr_allocation is None and not batch_mode:
            from ..models import IBKRTransactionAllocation

            ibkr_allocation = IBKRTransactionAllocation.query.filter_by(
                transaction_id=transaction.id
            ).first()

        # Get fund name from lookup or relationship (backwards compatibility)
        if portfolio_fund_lookup is not None:
            fund_name = portfolio_fund_lookup.get(transaction.portfolio_fund_id, "Unknown")
        else:
            # Only access relationship if not in batch mode
            fund_name = transaction.portfolio_fund.fund.name if not batch_mode else "Unknown"

        return {
            "id": transaction.id,
            "portfolio_fund_id": transaction.portfolio_fund_id,
            "fund_name": fund_name,
            "date": transaction.date.isoformat(),
            "type": transaction.type,
            "shares": transaction.shares,
            "cost_per_share": transaction.cost_per_share,
            "ibkr_linked": bool(ibkr_allocation),
            "ibkr_transaction_id": ibkr_allocation.ibkr_transaction_id if ibkr_allocation else None,
        }

    @staticmethod
    def create_transaction(data):
        """
        Create a new transaction.

        If a sell transaction is created, it will also record the realized gain/loss.

        Args:
            data (dict): Transaction data containing:
                - portfolio_fund_id: Portfolio fund ID
                - date: Transaction date (YYYY-MM-DD)
                - type: Transaction type
                - shares: Number of shares
                - cost_per_share: Cost per share

        Returns:
            Transaction: Created transaction object
        """
        # Convert date string to date object
        transaction_date = datetime.strptime(data["date"], "%Y-%m-%d").date()

        if data["type"] == "sell":
            # Use process_sell_transaction for sell transactions
            result = TransactionService.process_sell_transaction(
                portfolio_fund_id=data["portfolio_fund_id"],
                shares=float(data["shares"]),
                price=float(data["cost_per_share"]),
                date=transaction_date,
            )
            return result["transaction"]
        else:
            # Handle buy transactions normally
            transaction = Transaction(
                portfolio_fund_id=data["portfolio_fund_id"],
                date=transaction_date,
                type=data["type"],
                shares=float(data["shares"]),
                cost_per_share=float(data["cost_per_share"]),
            )
            db.session.add(transaction)
            db.session.commit()
            return transaction

    @staticmethod
    def update_transaction(transaction_id, data):
        """
        Update an existing transaction.

        Args:
            transaction_id (str): Transaction identifier
            data (dict): Updated transaction data containing:
                - date: Transaction date (YYYY-MM-DD)
                - type: Transaction type
                - shares: Number of shares
                - cost_per_share: Cost per share

        Returns:
            Transaction: Updated transaction object

        Raises:
            404: If transaction not found
        """
        transaction = Transaction.query.get_or_404(transaction_id)
        old_type = transaction.type

        transaction.date = datetime.strptime(data["date"], "%Y-%m-%d").date()
        transaction.type = data["type"]
        transaction.shares = float(data["shares"])
        transaction.cost_per_share = float(data["cost_per_share"])

        # Handle realized gains updates for sell transactions
        if old_type == "sell" or transaction.type == "sell":
            portfolio_fund = transaction.portfolio_fund

            # Delete old realized gain if it was a sell transaction
            if old_type == "sell":
                old_gain = RealizedGainLoss.query.filter_by(transaction_id=transaction_id).first()
                if old_gain:
                    db.session.delete(old_gain)

            # Create new realized gain if it's now a sell transaction
            if transaction.type == "sell":
                current_position = TransactionService.calculate_current_position(
                    transaction.portfolio_fund_id
                )
                if (
                    current_position["total_shares"] + transaction.shares < transaction.shares
                ):  # Add back the shares being edited
                    raise ValueError("Insufficient shares for sale")

                cost_basis = current_position["average_cost"] * transaction.shares
                sale_proceeds = transaction.shares * transaction.cost_per_share
                realized_gain_loss = sale_proceeds - cost_basis

                gain_loss_record = RealizedGainLoss(
                    portfolio_id=portfolio_fund.portfolio_id,
                    fund_id=portfolio_fund.fund_id,
                    transaction_id=transaction_id,
                    transaction_date=transaction.date,
                    shares_sold=transaction.shares,
                    cost_basis=cost_basis,
                    sale_proceeds=sale_proceeds,
                    realized_gain_loss=realized_gain_loss,
                )
                db.session.add(gain_loss_record)

        db.session.commit()
        return transaction

    @staticmethod
    def delete_transaction(transaction_id):
        """
        Delete a transaction with proper cleanup.

        This method handles:
        - IBKR allocation cleanup and status reversion
        - Realized gain/loss record deletion for sell transactions
        - Cascading deletions

        Args:
            transaction_id (str): Transaction identifier

        Returns:
            dict: Dictionary with deletion details including:
                - transaction_details: Details of deleted transaction
                - ibkr_transaction_id: ID of associated IBKR transaction (if any)
                - ibkr_reverted: Whether IBKR status was reverted to pending
                - realized_gain_deleted: Whether realized gain/loss was deleted

        Raises:
            ValueError: If transaction not found
        """
        from ..models import IBKRTransaction, IBKRTransactionAllocation, LogCategory, LogLevel
        from ..services.logging_service import logger

        transaction = Transaction.query.get(transaction_id)
        if not transaction:
            raise ValueError(f"Transaction {transaction_id} not found")

        # Store transaction details for logging before deletion
        transaction_details = {
            "type": transaction.type,
            "shares": transaction.shares,
            "cost_per_share": transaction.cost_per_share,
            "date": transaction.date.isoformat(),
        }

        try:
            # Check if this transaction is linked to an IBKR allocation
            ibkr_allocation = IBKRTransactionAllocation.query.filter_by(
                transaction_id=transaction_id
            ).first()

            # Handle IBKR allocation cleanup BEFORE deletion
            ibkr_transaction_id = None
            ibkr_reverted = False
            if ibkr_allocation:
                ibkr_transaction_id = ibkr_allocation.ibkr_transaction_id

                # Check how many allocations exist for this IBKR transaction
                allocation_count = IBKRTransactionAllocation.query.filter_by(
                    ibkr_transaction_id=ibkr_transaction_id
                ).count()

                # If this is the last allocation, revert IBKR transaction to pending
                if allocation_count == 1:
                    ibkr_txn = IBKRTransaction.query.get(ibkr_transaction_id)
                    if ibkr_txn and ibkr_txn.status == "processed":
                        ibkr_txn.status = "pending"
                        ibkr_txn.processed_at = None
                        ibkr_reverted = True
                        transaction_details["ibkr_status_reverted"] = True

                        logger.log(
                            level=LogLevel.INFO,
                            category=LogCategory.IBKR,
                            message="IBKR transaction status reverted to pending",
                            details={
                                "ibkr_transaction_id": ibkr_txn.ibkr_transaction_id,
                                "reason": "All allocations deleted",
                                "allocation_count": 0,
                            },
                        )

            # If this is a sell transaction, delete associated realized gain/loss record
            realized_gain = None
            if transaction.type == "sell":
                portfolio_fund = transaction.portfolio_fund
                realized_gain = RealizedGainLoss.query.filter_by(
                    portfolio_id=portfolio_fund.portfolio_id,
                    fund_id=portfolio_fund.fund_id,
                    transaction_date=transaction.date,
                    shares_sold=transaction.shares,
                ).first()

                if realized_gain:
                    db.session.delete(realized_gain)

            # Delete the transaction (will cascade delete the IBKR allocation)
            db.session.delete(transaction)
            db.session.commit()

            return {
                "transaction_details": transaction_details,
                "ibkr_transaction_id": ibkr_transaction_id,
                "ibkr_reverted": ibkr_reverted,
                "realized_gain_deleted": (
                    bool(realized_gain) if transaction.type == "sell" else None
                ),
            }

        except Exception as e:
            db.session.rollback()
            raise ValueError(f"Error deleting transaction: {e!s}") from e

    @staticmethod
    def calculate_current_position(portfolio_fund_id):
        """
        Calculate the current position (shares and cost basis) for a portfolio fund.

        Args:
            portfolio_fund_id (str): Portfolio fund identifier

        Returns:
            dict: Dictionary containing:
                - total_shares: Current number of shares held
                - total_cost: Total cost basis of current position
                - average_cost: Average cost per share
        """
        transactions = (
            Transaction.query.filter_by(portfolio_fund_id=portfolio_fund_id)
            .order_by(Transaction.date.asc())
            .all()
        )

        # Get dividend shares
        dividend_shares = (
            Dividend.query.filter_by(portfolio_fund_id=portfolio_fund_id)
            # Get the reinvestment transactions for stock dividends
            .join(
                Transaction,
                Dividend.reinvestment_transaction_id == Transaction.id,
                isouter=True,
            )
            .with_entities(Transaction.shares)
            .all()
        )
        total_dividend_shares = sum(d.shares or 0 for d in dividend_shares)

        # Add dividend shares to total
        total_shares = 0 + total_dividend_shares
        total_cost = 0

        for transaction in transactions:
            if transaction.type == "buy":
                total_shares += transaction.shares
                total_cost += transaction.shares * transaction.cost_per_share
            elif transaction.type == "sell":
                if (
                    total_shares >= transaction.shares
                    or abs(total_shares - transaction.shares) < 1e-07
                ):
                    # Calculate average cost before the sale
                    average_cost = total_cost / total_shares if total_shares > 0 else 0
                    total_shares -= transaction.shares
                    # Reduce cost basis by average cost of shares sold, not sale price
                    total_cost -= average_cost * transaction.shares
                else:
                    raise ValueError("Insufficient shares for sale")

        # Clean up near-zero values (floating point precision issues)
        if abs(total_shares) < 1e-07:
            total_shares = 0
            total_cost = 0
        if abs(total_cost) < 1e-07:
            total_cost = 0

        return {
            "total_shares": round(total_shares, 6),
            "total_cost": round(total_cost, 6),
            "average_cost": (
                total_cost / total_shares if total_shares > 0 else 0
            ),  # Set to None if no shares
        }

    @staticmethod
    def process_sell_transaction(portfolio_fund_id, shares, price, date):
        """Process a sell transaction and record realized gains/losses."""
        pf = PortfolioFund.query.get_or_404(portfolio_fund_id)

        # Calculate and record the realized gain/loss
        current_position = TransactionService.calculate_current_position(portfolio_fund_id)
        if current_position["total_shares"] < shares:
            raise ValueError("Insufficient shares for sale")

        cost_basis = current_position["average_cost"] * shares
        sale_proceeds = shares * price
        realized_gain_loss = sale_proceeds - cost_basis

        try:
            db.session.begin_nested()

            # Create the sell transaction first
            transaction = Transaction(
                portfolio_fund_id=portfolio_fund_id,
                date=date,
                type="sell",
                shares=shares,
                cost_per_share=price,
            )
            db.session.add(transaction)
            db.session.flush()  # Flush to get transaction.id without committing

            # Record the realized gain/loss
            gain_loss_record = RealizedGainLoss(
                portfolio_id=pf.portfolio_id,
                fund_id=pf.fund_id,
                transaction_id=transaction.id,
                transaction_date=date,
                shares_sold=shares,
                cost_basis=cost_basis,
                sale_proceeds=sale_proceeds,
                realized_gain_loss=realized_gain_loss,
            )
            db.session.add(gain_loss_record)

            # Commit the savepoint
            db.session.commit()

        except Exception as e:
            db.session.rollback()
            raise e

        return {"transaction": transaction, "realized_gain_loss": realized_gain_loss}
