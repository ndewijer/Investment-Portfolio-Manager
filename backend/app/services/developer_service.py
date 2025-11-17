"""
Service class for developer-related operations.

Provides methods for:
- Data import/export operations
- System maintenance functions
- Data sanitization utilities
"""

import csv
import uuid
from datetime import date, datetime

from ..models import (
    ExchangeRate,
    Fund,
    FundPrice,
    Portfolio,
    PortfolioFund,
    Transaction,
    db,
)


class DeveloperService:
    """
    Service class for developer-related operations.

    Provides methods for:
    - Data import/export operations
    - System maintenance functions
    - Data sanitization utilities
    """

    @staticmethod
    def sanitize_string(value):
        """
        Sanitize string input by stripping whitespace.

        Args:
            value: Input value to sanitize

        Returns:
            str: Sanitized string or None if input was None
        """
        if value is None:
            return None
        return str(value).strip()

    @staticmethod
    def sanitize_float(value):
        """
        Sanitize float input by converting string to float.

        Args:
            value: Input value to sanitize

        Returns:
            float: Sanitized float value or None if input was None

        Raises:
            ValueError: If value cannot be converted to float
        """
        if value is None:
            return None
        try:
            return float(str(value).strip())
        except ValueError as e:
            raise ValueError(f"Invalid number format: {value}") from e

    @staticmethod
    def sanitize_date(value):
        """
        Sanitize date input by converting string to date object.

        Args:
            value: Input date string in YYYY-MM-DD format

        Returns:
            date: Sanitized date object or None if input was None

        Raises:
            ValueError: If date format is invalid
        """
        if value is None:
            return None
        try:
            date_str = str(value).strip()
            return datetime.strptime(date_str, "%Y-%m-%d").date()
        except ValueError as e:
            raise ValueError(f"Invalid date format: {value}. Expected format: YYYY-MM-DD") from e

    @staticmethod
    def set_exchange_rate(from_currency, to_currency, rate, date=None):
        """
        Set or update exchange rate for a currency pair.

        Args:
            from_currency (str): Source currency code
            to_currency (str): Target currency code
            rate (float): Exchange rate value
            date (date, optional): Rate date, defaults to current date

        Returns:
            dict: Exchange rate details

        Raises:
            ValueError: If input validation fails
        """
        # Sanitize inputs
        from_currency = DeveloperService.sanitize_string(from_currency)
        to_currency = DeveloperService.sanitize_string(to_currency)
        rate = DeveloperService.sanitize_float(rate)
        date = DeveloperService.sanitize_date(date) or datetime.now().date()

        if not from_currency or not to_currency:
            raise ValueError("Currency codes cannot be empty")
        if not rate or rate <= 0:
            raise ValueError("Rate must be a positive number")

        exchange_rate = ExchangeRate.query.filter_by(
            from_currency=from_currency, to_currency=to_currency, date=date
        ).first()

        if exchange_rate:
            exchange_rate.rate = rate
        else:
            exchange_rate = ExchangeRate(
                from_currency=from_currency,
                to_currency=to_currency,
                rate=rate,
                date=date,
            )
            db.session.add(exchange_rate)

        db.session.commit()
        return {
            "from_currency": exchange_rate.from_currency,
            "to_currency": exchange_rate.to_currency,
            "rate": exchange_rate.rate,
            "date": exchange_rate.date.isoformat(),
        }

    @staticmethod
    def get_funds():
        """
        Retrieve all available funds from the database.

        Returns:
            list: List of Fund objects
        """
        return Fund.query.all()

    @staticmethod
    def get_portfolios():
        """
        Retrieve all available portfolios from the database.

        Returns:
            list: List of Portfolio objects
        """
        return Portfolio.query.all()

    @staticmethod
    def get_csv_template():
        """
        Get CSV template information for transactions.

        Returns:
            dict: CSV template details
        """
        description_text = (
            "CSV file should contain the following columns:\n"
            "- date: Transaction date in YYYY-MM-DD format\n"
            '- type: Transaction type, either "buy" or "sell"\n'
            "- shares: Number of shares (decimal numbers allowed)\n"
            "- cost_per_share: Cost per share in the fund's currency"
        )

        return {
            "headers": ["date", "type", "shares", "cost_per_share"],
            "example": {
                "date": "2024-03-21",
                "type": "buy/sell",
                "shares": "10.5",
                "cost_per_share": "150.75",
            },
            "description": description_text,
        }

    @staticmethod
    def _process_csv_content(file_content, required_fields, row_processor_func):
        """
        Common CSV processing utility that handles BOM, header mapping, and row processing.

        Args:
            file_content (bytes): UTF-8 encoded CSV file content
            required_fields (list): List of required field names
            row_processor_func (callable): Function to process each row, takes (mapped_row, row_num)

        Returns:
            list: List of processed objects

        Raises:
            ValueError: If file format is invalid or processing fails
        """
        results = []

        try:
            decoded_content = file_content.decode("utf-8-sig")
            csv_reader = csv.DictReader(decoded_content.splitlines())
            header_mapping = {}

            # Map headers to standardized names (handles BOM and whitespace)
            for std_name in required_fields:
                for actual_name in csv_reader.fieldnames:
                    if actual_name.lower().strip() == std_name:
                        header_mapping[actual_name] = std_name

            # Validate all required fields are present
            if len(header_mapping) != len(required_fields):
                missing_fields = set(required_fields) - set(header_mapping.values())
                raise ValueError(f"Missing required fields: {', '.join(missing_fields)}")

            for row_num, row in enumerate(csv_reader, start=2):
                try:
                    # Map and sanitize fields
                    mapped_row = {}
                    for actual_name, value in row.items():
                        if actual_name in header_mapping:
                            mapped_row[header_mapping[actual_name]] = value

                    # Process the row using the provided function
                    result = row_processor_func(mapped_row, row_num)
                    if result is not None:
                        results.append(result)

                except ValueError as e:
                    raise ValueError(f"Error in row {row_num}: {e!s}") from e

            if not results:
                raise ValueError("No valid records found in CSV file")

            return results

        except UnicodeDecodeError as e:
            raise ValueError("Invalid file encoding. Please use UTF-8 encoded CSV files.") from e
        except csv.Error as e:
            raise ValueError(f"CSV file error: {e!s}") from e

    @staticmethod
    def validate_utf8(file_content):
        """
        Validate that the file content is UTF-8 encoded.

        Args:
            file_content (bytes): Content to validate

        Returns:
            bool: True if content is valid UTF-8

        Raises:
            ValueError: If content is not valid UTF-8
        """
        try:
            # Try to decode as UTF-8 with strict error handling
            file_content.decode("utf-8", errors="strict")
            return True
        except UnicodeDecodeError as e:
            raise ValueError(
                f"File is not UTF-8 encoded. Please save the file in UTF-8 format. Error: {e!s}"
            ) from e

    @staticmethod
    def import_transactions_csv(file_content, portfolio_fund_id):
        """
        Import transactions from CSV file content.

        Args:
            file_content (bytes): UTF-8 encoded CSV file content
            portfolio_fund_id (str): Portfolio-Fund relationship ID

        Returns:
            int: Number of transactions imported

        Raises:
            ValueError: If file format is invalid or import fails
        """
        portfolio_fund = db.session.get(PortfolioFund, portfolio_fund_id)
        if not portfolio_fund:
            raise ValueError("Portfolio-fund relationship not found")

        def process_transaction_row(mapped_row, row_num):
            """Process a single transaction row."""
            return Transaction(
                id=str(uuid.uuid4()),
                portfolio_fund_id=portfolio_fund.id,
                date=DeveloperService.sanitize_date(mapped_row["date"]),
                type=DeveloperService.sanitize_string(mapped_row["type"]).lower(),
                shares=DeveloperService.sanitize_float(mapped_row["shares"]),
                cost_per_share=DeveloperService.sanitize_float(mapped_row["cost_per_share"]),
            )

        required_fields = ["date", "type", "shares", "cost_per_share"]
        transactions = DeveloperService._process_csv_content(
            file_content, required_fields, process_transaction_row
        )

        try:
            db.session.add_all(transactions)
            db.session.commit()
            return len(transactions)
        except Exception as e:
            db.session.rollback()
            raise ValueError(f"Database error while saving transactions: {e!s}") from e

    @staticmethod
    def get_exchange_rate(from_currency, to_currency, date):
        """
        Get exchange rate for a specific currency pair and date.

        Args:
            from_currency (str): Source currency code
            to_currency (str): Target currency code
            date (date): Date for exchange rate lookup

        Returns:
            dict: Exchange rate details including currencies, rate and date,
                 or None if not found
        """
        exchange_rate = ExchangeRate.query.filter_by(
            from_currency=from_currency, to_currency=to_currency, date=date
        ).first()

        if exchange_rate:
            return {
                "from_currency": exchange_rate.from_currency,
                "to_currency": exchange_rate.to_currency,
                "rate": exchange_rate.rate,
                "date": exchange_rate.date.isoformat(),
            }
        return None

    @staticmethod
    def import_fund_prices_csv(file_content, fund_id):
        """
        Import fund prices from CSV file content.

        Args:
            file_content (bytes): UTF-8 encoded CSV file content
            fund_id (str): Fund ID to associate prices with

        Returns:
            int: Number of prices imported

        Raises:
            ValueError: If file format is invalid or import fails
        """

        def process_fund_price_row(mapped_row, row_num):
            """Process a single fund price row."""
            return FundPrice(
                id=str(uuid.uuid4()),
                fund_id=fund_id,
                date=DeveloperService.sanitize_date(mapped_row["date"]),
                price=DeveloperService.sanitize_float(mapped_row["price"]),
            )

        required_fields = ["date", "price"]
        prices = DeveloperService._process_csv_content(
            file_content, required_fields, process_fund_price_row
        )

        try:
            db.session.add_all(prices)
            db.session.commit()
            return len(prices)
        except Exception as e:
            db.session.rollback()
            raise ValueError(f"Database error: {e!s}") from e

    @staticmethod
    def get_fund_price_csv_template():
        """
        Get CSV template information for fund prices.

        Returns:
            dict: CSV template details
        """
        description_text = (
            "CSV file should contain the following columns:\n"
            "- date: Price date in YYYY-MM-DD format\n"
            "- price: Fund price (decimal numbers)"
        )

        return {
            "headers": ["date", "price"],
            "example": {
                "date": "2024-03-21",
                "price": "150.75",
            },
            "description": description_text,
        }

    @staticmethod
    def get_fund_price(fund_id: str, date_: date) -> dict:
        """
        Get fund price for a specific fund and date.

        Args:
            fund_id (str): Fund identifier
            date_ (date): Date for price lookup

        Returns:
            dict: Fund price details including fund_id, date and price,
                 or None if not found
        """
        fund_price = FundPrice.query.filter_by(fund_id=fund_id, date=date_).first()

        if fund_price:
            return {
                "fund_id": fund_price.fund_id,
                "date": fund_price.date.isoformat(),
                "price": fund_price.price,
            }
        return None

    @staticmethod
    def set_fund_price(fund_id: str, price: float, date_: date | None = None) -> dict:
        """
        Set or update fund price for a specific date.

        Args:
            fund_id (str): Fund identifier
            price (float): Price value
            date_ (date, optional): Price date, defaults to current date

        Returns:
            dict: Fund price details including fund_id, date and price
        """
        if date_ is None:
            date_ = datetime.now().date()

        fund_price = FundPrice.query.filter_by(fund_id=fund_id, date=date_).first()

        if not fund_price:
            fund_price = FundPrice(fund_id=fund_id, price=price, date=date_)
            db.session.add(fund_price)
        else:
            fund_price.price = price

        db.session.commit()

        return {
            "fund_id": fund_price.fund_id,
            "date": fund_price.date.isoformat(),
            "price": fund_price.price,
        }
