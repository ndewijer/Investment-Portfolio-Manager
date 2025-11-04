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
            if transaction.type == "buy":
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
        portfolio_fund = PortfolioFund.query.get(data["portfolio_fund_id"])
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
            and data.get("reinvestment_shares")
            and data.get("reinvestment_price")
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
                raise ValueError(f"Invalid reinvestment data: {e!s}")

        try:
            db.session.commit()
            return dividend
        except Exception as e:
            db.session.rollback()
            raise ValueError(f"Error saving dividend: {e!s}")

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
        Format dividend for API response.

        Args:
            dividend (Dividend): Dividend object

        Returns:
            dict: Formatted dividend data

        Raises:
            ValueError: If dividend object is not found
        """
        return {
            "id": dividend.id,
            "fund_id": dividend.fund_id,
            "fund_name": dividend.fund.name,
            "portfolio_fund_id": dividend.portfolio_fund_id,
            "record_date": dividend.record_date.isoformat(),
            "ex_dividend_date": dividend.ex_dividend_date.isoformat(),
            "shares_owned": dividend.shares_owned,
            "dividend_per_share": dividend.dividend_per_share,
            "total_amount": dividend.total_amount,
            "reinvestment_status": dividend.reinvestment_status.value,
            "buy_order_date": (
                dividend.buy_order_date.isoformat() if dividend.buy_order_date else None
            ),
            "reinvestment_transaction_id": dividend.reinvestment_transaction_id,
            "dividend_type": dividend.fund.dividend_type.value,
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
        dividend = Dividend.query.get_or_404(dividend_id)

        try:
            # Delete associated transaction if it exists
            if dividend.reinvestment_transaction_id:
                transaction = Transaction.query.get(dividend.reinvestment_transaction_id)
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
            raise ValueError(f"Error deleting dividend: {e!s}")
