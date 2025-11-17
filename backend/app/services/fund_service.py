"""
Service class for fund management operations.

This module provides methods for:
- Creating and managing funds
- Retrieving fund information
- Fund CRUD operations
- Checking fund usage in portfolios
"""

from ..models import DividendType, Fund, FundPrice, InvestmentType, PortfolioFund, Transaction, db


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
    def create_fund(data, symbol_info=None):
        """
        Create a new fund with optional symbol lookup integration.

        Args:
            data (dict): Fund data containing:
                - name: Fund name
                - isin: International Securities Identification Number
                - currency: Trading currency code
                - exchange: Trading exchange
                - symbol (optional): Trading symbol
                - investment_type (optional): 'stock' or 'fund'
            symbol_info (dict, optional): Symbol information from lookup service

        Returns:
            Fund: Created fund object

        Raises:
            IntegrityError: If ISIN is not unique
        """
        # Get investment_type from request, default to 'fund' if not provided
        investment_type_str = data.get("investment_type", "fund")
        investment_type = (
            InvestmentType.STOCK if investment_type_str == "stock" else InvestmentType.FUND
        )

        fund = Fund(
            name=data["name"],
            isin=data["isin"],
            symbol=data.get("symbol"),
            currency=data["currency"],
            exchange=data["exchange"],
            investment_type=investment_type,
            dividend_type=DividendType.NONE,
        )
        db.session.add(fund)
        db.session.commit()
        return fund

    @staticmethod
    def update_fund(fund_id, data):
        """
        Update an existing fund with symbol and type support.

        Args:
            fund_id (str): Fund identifier
            data (dict): Updated fund data containing:
                - name: Fund name
                - isin: ISIN
                - symbol (optional): Trading symbol
                - currency: Currency code
                - exchange: Exchange name
                - dividend_type (optional): Dividend type
                - investment_type (optional): Investment type

        Returns:
            tuple: (fund, symbol_changed) where symbol_changed indicates if symbol was modified

        Raises:
            ValueError: If fund not found
        """
        fund = db.session.get(Fund, fund_id)
        if not fund:
            raise ValueError(f"Fund {fund_id} not found")

        fund.name = data["name"]
        fund.isin = data["isin"]

        # Track symbol changes for caller to handle symbol lookup
        symbol_changed = False
        if data.get("symbol"):
            old_symbol = fund.symbol
            new_symbol = data["symbol"]
            if old_symbol != new_symbol:
                fund.symbol = new_symbol
                symbol_changed = True
        else:
            fund.symbol = None  # Clear symbol if not provided

        fund.currency = data["currency"]
        fund.exchange = data["exchange"]

        if "dividend_type" in data:
            fund.dividend_type = DividendType(data["dividend_type"])

        if "investment_type" in data:
            investment_type_str = data["investment_type"]
            fund.investment_type = (
                InvestmentType.STOCK if investment_type_str == "stock" else InvestmentType.FUND
            )

        db.session.add(fund)
        db.session.commit()
        return fund, symbol_changed

    @staticmethod
    def check_fund_usage(fund_id):
        """
        Check if a fund is being used in any portfolios.

        Args:
            fund_id (str): Fund identifier

        Returns:
            dict: Usage information containing:
                - in_use (bool): Whether fund is in use
                - portfolios (list, optional): List of portfolios using the fund with
                  transaction counts
        """
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
        if not portfolio_funds:
            return {"in_use": False}

        # Get portfolios and their transaction counts
        portfolio_data = []
        for pf in portfolio_funds:
            transaction_count = Transaction.query.filter_by(portfolio_fund_id=pf.id).count()
            if transaction_count > 0:
                portfolio_data.append(
                    {
                        "id": pf.portfolio.id,
                        "name": pf.portfolio.name,
                        "transaction_count": transaction_count,
                    }
                )

        if portfolio_data:
            return {"in_use": True, "portfolios": portfolio_data}

        return {"in_use": False}

    @staticmethod
    def delete_fund(fund_id):
        """
        Delete a fund if it's not being used in any portfolios.

        Args:
            fund_id (str): Fund identifier

        Returns:
            dict: Deletion result with fund details

        Raises:
            ValueError: If fund not found or fund is in use
        """
        fund = db.session.get(Fund, fund_id)
        if not fund:
            raise ValueError(f"Fund {fund_id} not found")

        # Check for any portfolio-fund relationships
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
        if portfolio_funds:
            # Get list of portfolios this fund is attached to
            portfolio_info = [
                {"name": pf.portfolio.name, "id": pf.portfolio.id} for pf in portfolio_funds
            ]

            portfolio_names = ", ".join(pf["name"] for pf in portfolio_info)
            raise ValueError(
                f"Cannot delete {fund.name} because it is still attached to the "
                f"following portfolios: {portfolio_names}. Please remove the fund from "
                f"these portfolios first."
            )

        # Store fund details before deletion
        fund_details = {"fund_id": fund_id, "fund_name": fund.name}

        # Delete any fund prices
        FundPrice.query.filter_by(fund_id=fund_id).delete()

        # Delete the fund
        db.session.delete(fund)
        db.session.commit()

        return fund_details
