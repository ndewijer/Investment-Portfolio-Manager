"""
Comprehensive test suite for DeveloperService.

Tests data import/export operations, system maintenance functions,
and data sanitization utilities.
"""

import uuid
from datetime import date, datetime
from unittest.mock import patch

import pytest
from app.models import (
    DividendType,
    ExchangeRate,
    Fund,
    FundPrice,
    InvestmentType,
    Portfolio,
    PortfolioFund,
    Transaction,
    db,
)
from app.services.developer_service import DeveloperService
from tests.test_helpers import make_id, make_isin


class TestSanitizationMethods:
    """Tests for data sanitization utility methods."""

    def test_sanitize_string_normal_input(self, app_context):
        """Test sanitizing normal string input."""
        result = DeveloperService.sanitize_string("  hello world  ")
        assert result == "hello world"

    def test_sanitize_string_none_input(self, app_context):
        """Test sanitizing None input returns None."""
        result = DeveloperService.sanitize_string(None)
        assert result is None

    def test_sanitize_string_empty_input(self, app_context):
        """Test sanitizing empty string."""
        result = DeveloperService.sanitize_string("   ")
        assert result == ""

    def test_sanitize_string_non_string_input(self, app_context):
        """Test sanitizing non-string input converts to string."""
        result = DeveloperService.sanitize_string(12345)
        assert result == "12345"

    def test_sanitize_float_normal_input(self, app_context):
        """Test sanitizing normal float input."""
        result = DeveloperService.sanitize_float("  123.45  ")
        assert result == 123.45

    def test_sanitize_float_integer_input(self, app_context):
        """Test sanitizing integer input."""
        result = DeveloperService.sanitize_float("100")
        assert result == 100.0

    def test_sanitize_float_none_input(self, app_context):
        """Test sanitizing None input returns None."""
        result = DeveloperService.sanitize_float(None)
        assert result is None

    def test_sanitize_float_invalid_format(self, app_context):
        """Test sanitizing invalid float format raises ValueError."""
        with pytest.raises(ValueError, match="Invalid number format: abc123"):
            DeveloperService.sanitize_float("abc123")

    def test_sanitize_float_empty_string(self, app_context):
        """Test sanitizing empty string raises ValueError."""
        with pytest.raises(ValueError, match="Invalid number format"):
            DeveloperService.sanitize_float("")

    def test_sanitize_date_normal_input(self, app_context):
        """Test sanitizing normal date input."""
        result = DeveloperService.sanitize_date("  2024-03-15  ")
        assert result == date(2024, 3, 15)

    def test_sanitize_date_none_input(self, app_context):
        """Test sanitizing None input returns None."""
        result = DeveloperService.sanitize_date(None)
        assert result is None

    def test_sanitize_date_invalid_format(self, app_context):
        """Test sanitizing invalid date format raises ValueError."""
        with pytest.raises(ValueError, match="Invalid date format: 15/03/2024"):
            DeveloperService.sanitize_date("15/03/2024")

    def test_sanitize_date_invalid_date(self, app_context):
        """Test sanitizing invalid date raises ValueError."""
        with pytest.raises(ValueError, match="Invalid date format: 2024-13-45"):
            DeveloperService.sanitize_date("2024-13-45")


class TestExchangeRateManagement:
    """Tests for exchange rate management methods."""

    def test_set_exchange_rate_new_rate(self, app_context, db_session):
        """Test setting a new exchange rate."""
        result = DeveloperService.set_exchange_rate("USD", "EUR", 0.85)

        assert result["from_currency"] == "USD"
        assert result["to_currency"] == "EUR"
        assert result["rate"] == 0.85
        assert result["date"] == datetime.now().date().isoformat()

        # Verify database entry
        rate = ExchangeRate.query.filter_by(from_currency="USD", to_currency="EUR").first()
        assert rate is not None
        assert rate.rate == 0.85

    def test_set_exchange_rate_update_existing(self, app_context, db_session):
        """Test updating an existing exchange rate."""
        test_date = date(2024, 3, 15)

        # Create initial rate
        initial_rate = ExchangeRate(
            from_currency="USD", to_currency="EUR", rate=0.85, date=test_date
        )
        db.session.add(initial_rate)
        db.session.commit()

        # Update the rate
        result = DeveloperService.set_exchange_rate("USD", "EUR", 0.90, test_date)

        assert result["rate"] == 0.90
        assert result["date"] == test_date.isoformat()

        # Verify only one record exists with updated rate
        rates = ExchangeRate.query.filter_by(
            from_currency="USD", to_currency="EUR", date=test_date
        ).all()
        assert len(rates) == 1
        assert rates[0].rate == 0.90

    def test_set_exchange_rate_with_specific_date(self, app_context, db_session):
        """Test setting exchange rate with specific date."""
        test_date = date(2024, 1, 1)
        result = DeveloperService.set_exchange_rate("GBP", "USD", 1.25, test_date)

        assert result["date"] == "2024-01-01"

    def test_set_exchange_rate_validation_empty_currencies(self, app_context, db_session):
        """Test validation fails with empty currency codes."""
        with pytest.raises(ValueError, match="Currency codes cannot be empty"):
            DeveloperService.set_exchange_rate("", "EUR", 0.85)

        with pytest.raises(ValueError, match="Currency codes cannot be empty"):
            DeveloperService.set_exchange_rate("USD", None, 0.85)

    def test_set_exchange_rate_validation_invalid_rate(self, app_context, db_session):
        """Test validation fails with invalid rate values."""
        with pytest.raises(ValueError, match="Rate must be a positive number"):
            DeveloperService.set_exchange_rate("USD", "EUR", 0)

        with pytest.raises(ValueError, match="Rate must be a positive number"):
            DeveloperService.set_exchange_rate("USD", "EUR", -0.5)

    def test_get_exchange_rate_found(self, app_context, db_session):
        """Test getting existing exchange rate."""
        # Use unique currencies to avoid UNIQUE constraint conflicts
        from_curr = f"TE{uuid.uuid4().hex[:2].upper()}"  # e.g., "TEAB"
        to_curr = f"TE{uuid.uuid4().hex[:2].upper()}"  # e.g., "TECD"
        test_date = date(2024, 3, 15)

        rate = ExchangeRate(from_currency=from_curr, to_currency=to_curr, rate=0.85, date=test_date)
        db.session.add(rate)
        db.session.commit()

        result = DeveloperService.get_exchange_rate(from_curr, to_curr, test_date)

        assert result["from_currency"] == from_curr
        assert result["to_currency"] == to_curr
        assert result["rate"] == 0.85
        assert result["date"] == "2024-03-15"

    def test_get_exchange_rate_not_found(self, app_context, db_session):
        """Test getting non-existent exchange rate returns None."""
        result = DeveloperService.get_exchange_rate("USD", "JPY", date(2024, 3, 15))
        assert result is None


class TestDatabaseQueries:
    """Tests for database query methods."""

    def test_get_funds(self, app_context, db_session):
        """Test retrieving all funds."""
        # Create test funds with unique ISINs
        fund1 = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=make_isin("US"),
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        fund2 = Fund(
            id=make_id(),
            symbol="MSFT",
            name="Microsoft Corp",
            isin=make_isin("US"),
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        db.session.add_all([fund1, fund2])
        db.session.commit()

        result = DeveloperService.get_funds()

        assert len(result) >= 2
        fund_symbols = [f.symbol for f in result]
        assert "AAPL" in fund_symbols
        assert "MSFT" in fund_symbols

    def test_get_portfolios(self, app_context, db_session):
        """Test retrieving all portfolios."""
        # Create test portfolios
        portfolio1 = Portfolio(id=make_id(), name="Test Portfolio 1")
        portfolio2 = Portfolio(id=make_id(), name="Test Portfolio 2")
        db.session.add_all([portfolio1, portfolio2])
        db.session.commit()

        result = DeveloperService.get_portfolios()

        assert len(result) >= 2
        portfolio_names = [p.name for p in result]
        assert "Test Portfolio 1" in portfolio_names
        assert "Test Portfolio 2" in portfolio_names


class TestCSVTemplates:
    """Tests for CSV template methods."""

    def test_get_csv_template(self, app_context):
        """Test getting transaction CSV template."""
        result = DeveloperService.get_csv_template()

        assert "headers" in result
        assert "example" in result
        assert "description" in result

        assert result["headers"] == ["date", "type", "shares", "cost_per_share"]
        assert result["example"]["date"] == "2024-03-21"
        assert result["example"]["type"] == "buy/sell"
        assert "Transaction date in YYYY-MM-DD format" in result["description"]

    def test_get_fund_price_csv_template(self, app_context):
        """Test getting fund price CSV template."""
        result = DeveloperService.get_fund_price_csv_template()

        assert "headers" in result
        assert "example" in result
        assert "description" in result

        assert result["headers"] == ["date", "price"]
        assert result["example"]["date"] == "2024-03-21"
        assert result["example"]["price"] == "150.75"
        assert "Price date in YYYY-MM-DD format" in result["description"]


class TestCSVProcessing:
    """Tests for CSV processing utilities."""

    def test_validate_utf8_valid_content(self, app_context):
        """Test validating valid UTF-8 content."""
        content = b"test,data\n1,2"
        result = DeveloperService.validate_utf8(content)
        assert result is True

    def test_validate_utf8_invalid_content(self, app_context):
        """Test validating invalid UTF-8 content raises ValueError."""
        # Create invalid UTF-8 content
        content = b"test,data\n1,\xff\xfe"

        with pytest.raises(ValueError, match="File is not UTF-8 encoded"):
            DeveloperService.validate_utf8(content)

    def test_process_csv_content_valid_data(self, app_context):
        """Test processing valid CSV content."""
        csv_content = "date,price\n2024-03-15,150.75\n2024-03-16,151.25"
        file_content = csv_content.encode("utf-8")

        def process_row(mapped_row, row_num):
            return {
                "date": mapped_row["date"],
                "price": float(mapped_row["price"]),
                "row_num": row_num,
            }

        result = DeveloperService._process_csv_content(file_content, ["date", "price"], process_row)

        assert len(result) == 2
        assert result[0]["date"] == "2024-03-15"
        assert result[0]["price"] == 150.75
        assert result[0]["row_num"] == 2
        assert result[1]["date"] == "2024-03-16"
        assert result[1]["price"] == 151.25

    def test_process_csv_content_with_bom(self, app_context):
        """Test processing CSV content with BOM (Byte Order Mark)."""
        csv_content = "date,price\n2024-03-15,150.75"
        file_content = csv_content.encode("utf-8-sig")  # Includes BOM

        def process_row(mapped_row, row_num):
            return {"date": mapped_row["date"], "price": float(mapped_row["price"])}

        result = DeveloperService._process_csv_content(file_content, ["date", "price"], process_row)

        assert len(result) == 1
        assert result[0]["date"] == "2024-03-15"

    def test_process_csv_content_missing_fields(self, app_context):
        """Test processing CSV with missing required fields."""
        csv_content = "date,amount\n2024-03-15,150.75"  # Missing 'price' field
        file_content = csv_content.encode("utf-8")

        def process_row(mapped_row, row_num):
            return mapped_row

        with pytest.raises(ValueError, match="Missing required fields: price"):
            DeveloperService._process_csv_content(file_content, ["date", "price"], process_row)

    def test_process_csv_content_invalid_encoding(self, app_context):
        """Test processing CSV with invalid encoding."""
        file_content = b"date,price\n2024-03-15,\xff\xfe"  # Invalid UTF-8

        def process_row(mapped_row, row_num):
            return mapped_row

        with pytest.raises(ValueError, match="Invalid file encoding"):
            DeveloperService._process_csv_content(file_content, ["date", "price"], process_row)

    def test_process_csv_content_no_valid_records(self, app_context):
        """Test processing CSV that results in no valid records."""
        csv_content = "date,price\n"  # Header only, no data
        file_content = csv_content.encode("utf-8")

        def process_row(mapped_row, row_num):
            return None  # Process returns None for all rows

        with pytest.raises(ValueError, match="No valid records found in CSV file"):
            DeveloperService._process_csv_content(file_content, ["date", "price"], process_row)

    def test_process_csv_content_row_processing_error(self, app_context):
        """Test processing CSV with row processing error."""
        csv_content = "date,price\n2024-03-15,invalid_price"
        file_content = csv_content.encode("utf-8")

        def process_row(mapped_row, row_num):
            # This will raise ValueError for invalid_price
            return {"price": float(mapped_row["price"])}

        with pytest.raises(ValueError, match="Error in row 2"):
            DeveloperService._process_csv_content(file_content, ["date", "price"], process_row)


class TestTransactionImport:
    """Tests for transaction CSV import functionality."""

    def test_import_transactions_csv_success(self, app_context, db_session):
        """Test successful transaction import from CSV."""
        # Create test data
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add_all([fund, portfolio, portfolio_fund])
        db.session.commit()

        # Create CSV content
        csv_content = (
            "date,type,shares,cost_per_share\n2024-03-15,buy,100,150.75\n2024-03-16,sell,50,155.25"
        )
        file_content = csv_content.encode("utf-8")

        result = DeveloperService.import_transactions_csv(file_content, portfolio_fund.id)

        assert result == 2

        # Verify transactions were created
        transactions = Transaction.query.filter_by(portfolio_fund_id=portfolio_fund.id).all()
        assert len(transactions) == 2

        buy_txn = next(t for t in transactions if t.type == "buy")
        assert buy_txn.shares == 100.0
        assert buy_txn.cost_per_share == 150.75
        assert buy_txn.date == date(2024, 3, 15)

        sell_txn = next(t for t in transactions if t.type == "sell")
        assert sell_txn.shares == 50.0
        assert sell_txn.cost_per_share == 155.25

    def test_import_transactions_csv_invalid_portfolio_fund(self, app_context, db_session):
        """Test import fails with invalid portfolio-fund ID."""
        csv_content = "date,type,shares,cost_per_share\n2024-03-15,buy,100,150.75"
        file_content = csv_content.encode("utf-8")

        with pytest.raises(ValueError, match="Portfolio-fund relationship not found"):
            DeveloperService.import_transactions_csv(file_content, "invalid-id")

    def test_import_transactions_csv_database_error(self, app_context, db_session):
        """Test import handles database errors gracefully."""
        # Create test data
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add_all([fund, portfolio, portfolio_fund])
        db.session.commit()

        csv_content = "date,type,shares,cost_per_share\n2024-03-15,buy,100,150.75"
        file_content = csv_content.encode("utf-8")

        # Mock database error
        with (
            patch.object(db.session, "commit", side_effect=Exception("Database error")),
            pytest.raises(ValueError, match="Database error while saving transactions"),
        ):
            DeveloperService.import_transactions_csv(file_content, portfolio_fund.id)

    def test_import_transactions_csv_invalid_data_format(self, app_context, db_session):
        """Test import fails with invalid transaction data."""
        # Create test data
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        portfolio_fund = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add_all([fund, portfolio, portfolio_fund])
        db.session.commit()

        # Invalid date format
        csv_content = "date,type,shares,cost_per_share\n15/03/2024,buy,100,150.75"
        file_content = csv_content.encode("utf-8")

        with pytest.raises(ValueError, match="Error in row 2"):
            DeveloperService.import_transactions_csv(file_content, portfolio_fund.id)


class TestFundPriceManagement:
    """Tests for fund price management methods."""

    def test_get_fund_price_found(self, app_context, db_session):
        """Test getting existing fund price."""
        # Create test data
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        test_date = date(2024, 3, 15)
        fund_price = FundPrice(id=make_id(), fund_id=fund.id, date=test_date, price=150.75)
        db.session.add_all([fund, fund_price])
        db.session.commit()

        result = DeveloperService.get_fund_price(fund.id, test_date)

        assert result["fund_id"] == fund.id
        assert result["date"] == "2024-03-15"
        assert result["price"] == 150.75

    def test_get_fund_price_not_found(self, app_context, db_session):
        """Test getting non-existent fund price returns None."""
        fund_id = make_id()
        result = DeveloperService.get_fund_price(fund_id, date(2024, 3, 15))
        assert result is None

    def test_set_fund_price_new_price(self, app_context, db_session):
        """Test setting a new fund price."""
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        db.session.add(fund)
        db.session.commit()

        test_date = date(2024, 3, 15)
        result = DeveloperService.set_fund_price(fund.id, 150.75, test_date)

        assert result["fund_id"] == fund.id
        assert result["date"] == "2024-03-15"
        assert result["price"] == 150.75

        # Verify database entry
        fund_price = FundPrice.query.filter_by(fund_id=fund.id, date=test_date).first()
        assert fund_price is not None
        assert fund_price.price == 150.75

    def test_set_fund_price_update_existing(self, app_context, db_session):
        """Test updating existing fund price."""
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        test_date = date(2024, 3, 15)
        fund_price = FundPrice(id=make_id(), fund_id=fund.id, date=test_date, price=150.75)
        db.session.add_all([fund, fund_price])
        db.session.commit()

        # Update the price
        result = DeveloperService.set_fund_price(fund.id, 155.25, test_date)

        assert result["price"] == 155.25

        # Verify only one record exists with updated price
        prices = FundPrice.query.filter_by(fund_id=fund.id, date=test_date).all()
        assert len(prices) == 1
        assert prices[0].price == 155.25

    def test_set_fund_price_default_date(self, app_context, db_session):
        """Test setting fund price with default date (today)."""
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        db.session.add(fund)
        db.session.commit()

        result = DeveloperService.set_fund_price(fund.id, 150.75)

        assert result["date"] == datetime.now().date().isoformat()
        assert result["price"] == 150.75


class TestFundPriceImport:
    """Tests for fund price CSV import functionality."""

    def test_import_fund_prices_csv_success(self, app_context, db_session):
        """Test successful fund price import from CSV."""
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        db.session.add(fund)
        db.session.commit()

        csv_content = "date,price\n2024-03-15,150.75\n2024-03-16,151.25\n2024-03-17,149.50"
        file_content = csv_content.encode("utf-8")

        result = DeveloperService.import_fund_prices_csv(file_content, fund.id)

        assert result == 3

        # Verify prices were created
        prices = FundPrice.query.filter_by(fund_id=fund.id).all()
        assert len(prices) == 3

        # Check specific prices
        price_data = {p.date: p.price for p in prices}
        assert price_data[date(2024, 3, 15)] == 150.75
        assert price_data[date(2024, 3, 16)] == 151.25
        assert price_data[date(2024, 3, 17)] == 149.50

    def test_import_fund_prices_csv_database_error(self, app_context, db_session):
        """Test import handles database errors gracefully."""
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        db.session.add(fund)
        db.session.commit()

        csv_content = "date,price\n2024-03-15,150.75"
        file_content = csv_content.encode("utf-8")

        # Mock database error
        with (
            patch.object(db.session, "commit", side_effect=Exception("Database error")),
            pytest.raises(ValueError, match="Database error"),
        ):
            DeveloperService.import_fund_prices_csv(file_content, fund.id)

    def test_import_fund_prices_csv_invalid_data(self, app_context, db_session):
        """Test import fails with invalid price data."""
        # Create unique ISIN for each test
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            symbol="AAPL",
            name="Apple Inc",
            isin=unique_isin,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
            dividend_type=DividendType.CASH,
        )
        db.session.add(fund)
        db.session.commit()

        # Invalid price format
        csv_content = "date,price\n2024-03-15,invalid_price"
        file_content = csv_content.encode("utf-8")

        with pytest.raises(ValueError, match="Error in row 2"):
            DeveloperService.import_fund_prices_csv(file_content, fund.id)
