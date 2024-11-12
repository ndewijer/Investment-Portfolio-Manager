"""
Service class for transaction-related operations.

This module provides methods for:
- Creating and managing transactions
- Retrieving transaction history
- Formatting transaction data
"""

from datetime import datetime

from ..models import PortfolioFund, Transaction, db


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
        transaction = Transaction(
            portfolio_fund_id=data["portfolio_fund_id"],
            date=datetime.strptime(data["date"], "%Y-%m-%d").date(),
            type=data["type"],
            shares=data["shares"],
            cost_per_share=data["cost_per_share"],
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
            data (dict): Updated transaction data

        Returns:
            Transaction: Updated transaction object

        Raises:
            404: If transaction not found
        """
        transaction = Transaction.query.get_or_404(transaction_id)
        transaction.date = datetime.strptime(data["date"], "%Y-%m-%d").date()
        transaction.type = data["type"]
        transaction.shares = data["shares"]
        transaction.cost_per_share = data["cost_per_share"]
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
