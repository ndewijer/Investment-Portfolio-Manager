"""
Service class for dividend-related operations.

Provides methods for:
- Calculating shares owned on a record date
- Creating new dividend records
- Retrieving fund and portfolio dividends
- Formatting dividend data
- Deleting dividend records
"""

import uuid
from datetime import datetime

from ..models import (
    Dividend,
    DividendType,
    PortfolioFund,
    ReinvestmentStatus,
    Transaction,
    db,
)


class DividendService:
    """
    Service class for dividend-related operations.

    Provides methods for:
    - Calculating shares owned on a record date
    - Creating new dividend records
    - Retrieving fund and portfolio dividends
    - Formatting dividend data
    - Deleting dividend records
    """

    @staticmethod
    def get_dividend(dividend_id):
        """
        Retrieve a dividend by ID.

        Args:
            dividend_id (str): Dividend identifier

        Returns:
            Dividend: Dividend object

        Raises:
            404: If dividend not found
        """
        dividend = db.session.get(Dividend, dividend_id)
        if not dividend:
            from flask import abort

            abort(404)
        return dividend

    @staticmethod
    def calculate_shares_owned(portfolio_fund_id, record_date):
        """
        Calculate shares owned on record date.

        Args:
            portfolio_fund_id (str): Portfolio fund identifier
            record_date (date): Record date

        Returns:
            float: Shares owned

        Raises:
            ValueError: If portfolio-fund relationship not found
        """
        shares = 0
        transactions = (
            Transaction.query.filter(
                Transaction.portfolio_fund_id == portfolio_fund_id,
                Transaction.date <= record_date,
            )
            .order_by(Transaction.date)
            .all()
        )

        for transaction in transactions:
            if transaction.type in ("buy", "dividend"):
                shares += transaction.shares
            else:
                shares -= transaction.shares

        return shares

    @staticmethod
    def create_dividend(data):
        """
        Create a new dividend record.

        Args:
            data (dict): Dividend data

        Returns:
            Dividend: New dividend record

        Raises:
            ValueError: If portfolio-fund relationship not found
            ValueError: If dividend object is not found
        """
        portfolio_fund = db.session.get(PortfolioFund, data["portfolio_fund_id"])
        if not portfolio_fund:
            raise ValueError("Portfolio-fund relationship not found")

        # Calculate shares owned on record date
        shares_owned = DividendService.calculate_shares_owned(
            data["portfolio_fund_id"],
            datetime.strptime(data["record_date"], "%Y-%m-%d").date(),
        )

        # Calculate total amount
        dividend_per_share = float(data["dividend_per_share"])
        total_amount = shares_owned * dividend_per_share

        # Set initial status based on dividend type
        initial_status = ReinvestmentStatus.PENDING
        if portfolio_fund.fund.dividend_type == DividendType.CASH:
            initial_status = ReinvestmentStatus.COMPLETED  # Cash dividends are marked as completed

        dividend = Dividend(
            id=str(uuid.uuid4()),
            fund_id=portfolio_fund.fund_id,
            portfolio_fund_id=data["portfolio_fund_id"],
            record_date=datetime.strptime(data["record_date"], "%Y-%m-%d").date(),
            ex_dividend_date=datetime.strptime(data["ex_dividend_date"], "%Y-%m-%d").date(),
            shares_owned=shares_owned,
            dividend_per_share=dividend_per_share,
            total_amount=total_amount,
            reinvestment_status=initial_status,
            buy_order_date=(
                datetime.strptime(data["buy_order_date"], "%Y-%m-%d").date()
                if data.get("buy_order_date")
                else None
            ),
        )

        db.session.add(dividend)
        db.session.flush()

        # Handle stock dividend with immediate reinvestment
        if (
            portfolio_fund.fund.dividend_type == DividendType.STOCK
            and "reinvestment_shares" in data
            and "reinvestment_price" in data
        ):
            try:
                reinvestment_shares = float(data["reinvestment_shares"])
                reinvestment_price = float(data["reinvestment_price"])

                if reinvestment_shares <= 0 or reinvestment_price <= 0:
                    raise ValueError("Reinvestment shares and price must be positive numbers")

                # Create transaction for reinvested dividend
                transaction = Transaction(
                    id=str(uuid.uuid4()),
                    portfolio_fund_id=data["portfolio_fund_id"],
                    date=datetime.strptime(data["ex_dividend_date"], "%Y-%m-%d").date(),
                    type="dividend",
                    shares=reinvestment_shares,
                    cost_per_share=reinvestment_price,
                )
                db.session.add(transaction)
                db.session.flush()

                # Link transaction to dividend and update status
                dividend.reinvestment_transaction_id = transaction.id
                dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
                dividend.buy_order_date = transaction.date
            except (ValueError, TypeError) as e:
                db.session.rollback()
                raise ValueError(f"Invalid reinvestment data: {e!s}") from e

        try:
            db.session.commit()
            return dividend
        except Exception as e:
            db.session.rollback()
            raise ValueError(f"Error saving dividend: {e!s}") from e

    @staticmethod
    def update_dividend(dividend_id, data):
        """
        Update an existing dividend record.

        Args:
            dividend_id (str): Dividend identifier
            data (dict): Updated dividend data containing:
                - record_date (str): Record date in YYYY-MM-DD format
                - ex_dividend_date (str): Ex-dividend date in YYYY-MM-DD format
                - dividend_per_share (float): Dividend amount per share
                - buy_order_date (str, optional): Buy order date for reinvestment
                - reinvestment_shares (float, optional): Number of shares for reinvestment
                - reinvestment_price (float, optional): Price per share for reinvestment

        Returns:
            tuple: (updated_dividend, original_values) for logging purposes

        Raises:
            ValueError: If dividend not found or invalid data provided
        """
        dividend = db.session.get(Dividend, dividend_id)
        if not dividend:
            raise ValueError(f"Dividend {dividend_id} not found")

        # Store original values for logging
        original_values = {
            "record_date": dividend.record_date,
            "ex_dividend_date": dividend.ex_dividend_date,
            "dividend_per_share": dividend.dividend_per_share,
            "reinvestment_status": dividend.reinvestment_status.value,
        }

        try:
            # Update basic dividend information
            dividend.record_date = datetime.strptime(data["record_date"], "%Y-%m-%d").date()
            dividend.ex_dividend_date = datetime.strptime(
                data["ex_dividend_date"], "%Y-%m-%d"
            ).date()
            dividend.dividend_per_share = float(data["dividend_per_share"])

            # Update buy_order_date if provided
            if data.get("buy_order_date"):
                dividend.buy_order_date = datetime.strptime(
                    data["buy_order_date"], "%Y-%m-%d"
                ).date()

            # Recalculate total amount
            dividend.total_amount = dividend.shares_owned * dividend.dividend_per_share

            # Handle stock dividend updates
            if dividend.fund.dividend_type == DividendType.STOCK:
                if "reinvestment_shares" in data and "reinvestment_price" in data:
                    reinvestment_shares = float(data["reinvestment_shares"])
                    reinvestment_price = float(data["reinvestment_price"])

                    if reinvestment_shares <= 0 or reinvestment_price <= 0:
                        raise ValueError("Reinvestment shares and price must be positive numbers")

                    if dividend.reinvestment_transaction_id:
                        # Update existing reinvestment transaction
                        transaction = db.session.get(
                            Transaction, dividend.reinvestment_transaction_id
                        )
                        if transaction:
                            transaction.date = dividend.buy_order_date or dividend.ex_dividend_date
                            transaction.shares = reinvestment_shares
                            transaction.cost_per_share = reinvestment_price
                            db.session.add(transaction)
                            dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
                    else:
                        # Create new reinvestment transaction
                        transaction = Transaction(
                            id=str(uuid.uuid4()),
                            portfolio_fund_id=dividend.portfolio_fund_id,
                            date=dividend.buy_order_date or dividend.ex_dividend_date,
                            type="dividend",
                            shares=reinvestment_shares,
                            cost_per_share=reinvestment_price,
                        )
                        db.session.add(transaction)
                        db.session.flush()
                        dividend.reinvestment_transaction_id = transaction.id
                        dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
                elif dividend.reinvestment_transaction_id:
                    # If reinvestment data is removed, delete the transaction
                    transaction = db.session.get(Transaction, dividend.reinvestment_transaction_id)
                    if transaction:
                        db.session.delete(transaction)
                    dividend.reinvestment_transaction_id = None
                    dividend.reinvestment_status = ReinvestmentStatus.PENDING
            else:
                # For cash dividends, always set status to COMPLETED
                dividend.reinvestment_status = ReinvestmentStatus.COMPLETED

            db.session.commit()
            return dividend, original_values

        except Exception as e:
            db.session.rollback()
            raise ValueError(f"Error updating dividend: {e!s}") from e

    @staticmethod
    def get_fund_dividends(fund_id):
        """
        Get all dividends for a fund.

        Args:
            fund_id (str): Fund identifier

        Returns:
            list: List of Dividend objects

        Raises:
            ValueError: If fund not found
        """
        return Dividend.query.filter_by(fund_id=fund_id).all()

    @staticmethod
    def get_portfolio_dividends(portfolio_id):
        """
        Get all dividends for a portfolio.

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            list: List of Dividend objects

        Raises:
            ValueError: If portfolio not found
        """
        return (
            Dividend.query.join(PortfolioFund)
            .filter(PortfolioFund.portfolio_id == portfolio_id)
            .all()
        )

    @staticmethod
    def format_dividend(dividend):
        """
        Format dividend for API response with camelCase keys.

        Args:
            dividend (Dividend): Dividend object

        Returns:
            dict: Formatted dividend data with camelCase keys containing:
                - id: Dividend ID
                - fundId: Fund ID
                - fundName: Fund name
                - portfolioFundId: Portfolio fund ID
                - recordDate: ISO format date
                - exDividendDate: ISO format date
                - sharesOwned: Shares owned on record date
                - dividendPerShare: Dividend amount per share
                - totalAmount: Total dividend amount
                - reinvestmentStatus: Reinvestment status (PENDING/COMPLETED/PARTIAL)
                - buyOrderDate: Buy order date if applicable
                - reinvestmentTransactionId: Transaction ID if reinvested
                - dividendType: Dividend type (NONE/CASH/STOCK)

        Raises:
            ValueError: If dividend object is not found
        """
        return {
            "id": dividend.id,
            "fundId": dividend.fund_id,
            "fundName": dividend.fund.name,
            "portfolioFundId": dividend.portfolio_fund_id,
            "recordDate": dividend.record_date.isoformat(),
            "exDividendDate": dividend.ex_dividend_date.isoformat(),
            "sharesOwned": dividend.shares_owned,
            "dividendPerShare": dividend.dividend_per_share,
            "totalAmount": dividend.total_amount,
            "reinvestmentStatus": dividend.reinvestment_status.value,
            "buyOrderDate": (
                dividend.buy_order_date.isoformat() if dividend.buy_order_date else None
            ),
            "reinvestmentTransactionId": dividend.reinvestment_transaction_id,
            "dividendType": dividend.fund.dividend_type.value,
        }

    @staticmethod
    def delete_dividend(dividend_id):
        """
        Delete a dividend and its associated transaction.

        Args:
            dividend_id (str): Dividend identifier

        Returns:
            bool: True if deletion is successful, False otherwise

        Raises:
            ValueError: If there is an error during deletion
        """
        dividend = db.session.get(Dividend, dividend_id)
        if not dividend:
            from flask import abort

            abort(404)

        try:
            # Delete associated transaction if it exists
            if dividend.reinvestment_transaction_id:
                transaction = db.session.get(Transaction, dividend.reinvestment_transaction_id)
                if transaction:
                    print(f"Deleting associated transaction {transaction.id}")
                    db.session.delete(transaction)
                else:
                    print(
                        f"Associated transaction {dividend.reinvestment_transaction_id} not found"
                    )

            print(f"Deleting dividend {dividend.id}")
            db.session.delete(dividend)
            db.session.commit()
            return True
        except Exception as e:
            db.session.rollback()
            print(f"Error deleting dividend: {e!s}")
            raise ValueError(f"Error deleting dividend: {e!s}") from e
