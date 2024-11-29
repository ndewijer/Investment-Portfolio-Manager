"""
Service class for transaction-related operations.

This module provides methods for:
- Creating and managing transactions
- Retrieving transaction history
- Formatting transaction data
"""

from datetime import datetime

from ..models import PortfolioFund, Transaction, db, RealizedGainLoss


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

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            list: List of formatted transaction dictionaries
        """
        transactions = (
            Transaction.query.join(PortfolioFund)
            .filter(PortfolioFund.portfolio_id == portfolio_id)
            .all()
        )
        return [TransactionService.format_transaction(t) for t in transactions]

    @staticmethod
    def format_transaction(transaction):
        """
        Format a transaction object into a dictionary.

        Args:
            transaction (Transaction): Transaction object

        Returns:
            dict: Formatted transaction containing:
                - id: Transaction ID
                - portfolio_fund_id: Portfolio fund ID
                - fund_name: Fund name
                - date: ISO format date
                - type: Transaction type
                - shares: Number of shares
                - cost_per_share: Cost per share
        """
        return {
            "id": transaction.id,
            "portfolio_fund_id": transaction.portfolio_fund_id,
            "fund_name": transaction.portfolio_fund.fund.name,
            "date": transaction.date.isoformat(),
            "type": transaction.type,
            "shares": transaction.shares,
            "cost_per_share": transaction.cost_per_share,
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
                old_gain = RealizedGainLoss.query.filter_by(
                    transaction_id=transaction_id
                ).first()
                if old_gain:
                    db.session.delete(old_gain)

            # Create new realized gain if it's now a sell transaction
            if transaction.type == "sell":
                current_position = TransactionService.calculate_current_position(
                    transaction.portfolio_fund_id
                )
                if (
                    current_position["total_shares"] + transaction.shares
                    < transaction.shares  # noqa: W503
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
        Delete a transaction.

        Args:
            transaction_id (str): Transaction identifier

        Raises:
            404: If transaction not found
        """
        transaction = Transaction.query.get_or_404(transaction_id)
        db.session.delete(transaction)
        db.session.commit()

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

        total_shares = 0
        total_cost = 0

        for transaction in transactions:
            if transaction.type == "buy":
                total_shares += transaction.shares
                total_cost += transaction.shares * transaction.cost_per_share
            elif transaction.type == "sell":
                if total_shares >= transaction.shares:
                    # Calculate cost basis for sold shares
                    cost_per_share = total_cost / total_shares
                    cost_basis = cost_per_share * transaction.shares

                    # Update totals
                    total_shares -= transaction.shares
                    total_cost -= cost_basis
                else:
                    raise ValueError("Insufficient shares for sale")

        return {
            "total_shares": total_shares,
            "total_cost": total_cost,
            "average_cost": total_cost / total_shares if total_shares > 0 else 0,
        }

    @staticmethod
    def process_sell_transaction(portfolio_fund_id, shares, price, date):
        """Process a sell transaction and record realized gains/losses."""
        pf = PortfolioFund.query.get_or_404(portfolio_fund_id)

        # Create the sell transaction first
        transaction = Transaction(
            portfolio_fund_id=portfolio_fund_id,
            date=date,
            type="sell",
            shares=shares,
            cost_per_share=price,
        )
        db.session.add(transaction)

        # Calculate and record the realized gain/loss
        current_position = TransactionService.calculate_current_position(portfolio_fund_id)
        if current_position["total_shares"] < shares:
            raise ValueError("Insufficient shares for sale")

        cost_basis = current_position["average_cost"] * shares
        sale_proceeds = shares * price
        realized_gain_loss = sale_proceeds - cost_basis

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
        db.session.commit()

        return {"transaction": transaction, "realized_gain_loss": realized_gain_loss}
