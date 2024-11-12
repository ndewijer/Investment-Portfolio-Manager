"""
Service class for fund management operations.

This module provides methods for:
- Creating and managing funds
- Retrieving fund information
- Fund CRUD operations
"""

from ..models import Fund, db


class FundService:
    """
    Service class for fund management operations.

    Provides methods for:
    - Creating and managing funds
    - Retrieving fund information
    - Fund CRUD operations
    """

    @staticmethod
    def get_all_funds():
        """
        Retrieve all funds from the database.

        Returns:
            list: List of Fund objects
        """
        return Fund.query.all()

    @staticmethod
    def get_fund(fund_id):
        """
        Retrieve a specific fund by ID.

        Args:
            fund_id (str): Fund identifier

        Returns:
            Fund: Fund object

        Raises:
            404: If fund not found
        """
        return Fund.query.get_or_404(fund_id)

    @staticmethod
    def create_fund(data):
        """
        Create a new fund.

        Args:
            data (dict): Fund data containing:
                - name: Fund name
                - isin: International Securities Identification Number
                - currency: Trading currency code
                - exchange: Trading exchange

        Returns:
            Fund: Created fund object
        """
        fund = Fund(
            name=data["name"],
            isin=data["isin"],
            currency=data["currency"],
            exchange=data["exchange"],
        )
        db.session.add(fund)
        db.session.commit()
        return fund

    @staticmethod
    def update_fund(fund_id, data):
        """
        Update an existing fund.

        Args:
            fund_id (str): Fund identifier
            data (dict): Updated fund data

        Returns:
            Fund: Updated fund object

        Raises:
            404: If fund not found
        """
        fund = Fund.query.get_or_404(fund_id)
        fund.name = data["name"]
        fund.isin = data["isin"]
        fund.currency = data["currency"]
        fund.exchange = data["exchange"]
        db.session.commit()
        return fund

    @staticmethod
    def delete_fund(fund_id):
        """
        Delete a fund.

        Args:
            fund_id (str): Fund identifier

        Raises:
            404: If fund not found
        """
        fund = Fund.query.get_or_404(fund_id)
        db.session.delete(fund)
        db.session.commit()
