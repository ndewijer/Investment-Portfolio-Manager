"""
Tests for FundService.

This test suite covers:
- Fund retrieval operations (get_all, get by ID)
- Fund creation with different investment types
- Fund update operations with symbol tracking
- Fund deletion and usage checking
- Error handling for invalid operations
"""

import uuid

import pytest
from app.models import (
    DividendType,
    Fund,
    FundPrice,
    InvestmentType,
    Portfolio,
    PortfolioFund,
    Transaction,
    db,
)
from app.services.fund_service import FundService
from tests.test_helpers import make_id, make_isin
from werkzeug.exceptions import NotFound


class TestFundRetrieval:
    """Tests for fund retrieval operations."""

    def test_get_all_funds(self, app_context, db_session):
        """Test retrieving all funds."""
        # Create funds
        fund1 = Fund(
            id=make_id(),
            name="Fund 1",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        fund2 = Fund(
            id=make_id(),
            name="Fund 2",
            isin=make_isin("US"),
            currency="EUR",
            exchange="LSE",
        )
        db_session.add_all([fund1, fund2])
        db_session.commit()

        # Get all funds
        funds = FundService.get_all_funds()

        # Verify our funds are in the results
        fund_ids = {f.id for f in funds}
        assert fund1.id in fund_ids
        assert fund2.id in fund_ids

    def test_get_fund_success(self, app_context, db_session):
        """Test retrieving a specific fund by ID."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NASDAQ",
        )
        db_session.add(fund)
        db_session.commit()

        # Get fund
        retrieved = FundService.get_fund(fund.id)

        # Verify
        assert retrieved.id == fund.id
        assert retrieved.name == "Test Fund"
        assert retrieved.currency == "USD"
        assert retrieved.exchange == "NASDAQ"

    def test_get_fund_not_found(self, app_context, db_session):
        """Test get_fund raises 404 for nonexistent fund."""
        fake_id = make_id()

        with pytest.raises(NotFound):
            FundService.get_fund(fake_id)

    def test_get_latest_fund_price(self, app_context, db_session):
        """Test get_latest_fund_price returns most recent price."""
        fund = Fund(
            isin=make_isin("US"),
            symbol="VTI",
            name="Vanguard Total Stock Market ETF",
            currency="USD",
            exchange="NASDAQ",
        )
        db_session.add(fund)
        db_session.commit()

        # Add multiple prices
        from datetime import date

        price1 = FundPrice(fund_id=fund.id, date=date(2024, 1, 1), price=200.00)
        price2 = FundPrice(fund_id=fund.id, date=date(2024, 1, 15), price=250.00)
        db_session.add_all([price1, price2])
        db_session.commit()

        latest_price = FundService.get_latest_fund_price(fund.id)

        assert latest_price == 250.00

    def test_get_latest_fund_price_no_prices(self, app_context, db_session):
        """Test get_latest_fund_price returns None when no prices exist."""
        fund = Fund(
            isin=make_isin("US"),
            symbol="VTI",
            name="Vanguard Total Stock Market ETF",
            currency="USD",
            exchange="NASDAQ",
        )
        db_session.add(fund)
        db_session.commit()

        latest_price = FundService.get_latest_fund_price(fund.id)

        assert latest_price is None

    def test_get_fund_price_history(self, app_context, db_session):
        """Test get_fund_price_history returns all prices ordered by date."""
        fund = Fund(
            isin=make_isin("US"),
            symbol="VTI",
            name="Vanguard Total Stock Market ETF",
            currency="USD",
            exchange="NASDAQ",
        )
        db_session.add(fund)
        db_session.commit()

        # Add multiple prices
        from datetime import date

        price1 = FundPrice(fund_id=fund.id, date=date(2024, 1, 1), price=200.00)
        price2 = FundPrice(fund_id=fund.id, date=date(2024, 1, 15), price=250.00)
        price3 = FundPrice(fund_id=fund.id, date=date(2024, 1, 10), price=225.00)
        db_session.add_all([price1, price2, price3])
        db_session.commit()

        prices = FundService.get_fund_price_history(fund.id)

        assert len(prices) == 3
        # Should be ordered by date descending (newest first)
        assert prices[0].price == 250.00
        assert prices[1].price == 225.00
        assert prices[2].price == 200.00

    def test_get_fund_price_history_empty(self, app_context, db_session):
        """Test get_fund_price_history returns empty list when no prices exist."""
        fund = Fund(
            isin=make_isin("US"),
            symbol="VTI",
            name="Vanguard Total Stock Market ETF",
            currency="USD",
            exchange="NASDAQ",
        )
        db_session.add(fund)
        db_session.commit()

        prices = FundService.get_fund_price_history(fund.id)

        assert prices == []


class TestFundCreation:
    """Tests for fund creation operations."""

    def test_create_fund_minimal(self, app_context, db_session):
        """Test creating fund with minimal required fields."""
        data = {
            "name": "Test Fund",
            "isin": make_isin("US"),
            "currency": "USD",
            "exchange": "NYSE",
        }

        fund = FundService.create_fund(data)

        # Verify creation
        assert fund.id is not None
        assert fund.name == "Test Fund"
        assert fund.currency == "USD"
        assert fund.exchange == "NYSE"
        assert fund.investment_type == InvestmentType.FUND  # Default
        assert fund.dividend_type == DividendType.NONE  # Default
        assert fund.symbol is None

        # Verify in database
        db_fund = db.session.get(Fund, fund.id)
        assert db_fund is not None
        assert db_fund.name == "Test Fund"

    def test_create_fund_with_symbol(self, app_context, db_session):
        """Test creating fund with symbol."""
        data = {
            "name": "Apple Inc.",
            "isin": make_isin("US"),
            "currency": "USD",
            "exchange": "NASDAQ",
            "symbol": "AAPL",
        }

        fund = FundService.create_fund(data)

        assert fund.symbol == "AAPL"

    def test_create_fund_as_stock(self, app_context, db_session):
        """Test creating fund with investment_type='stock'."""
        data = {
            "name": "Tesla Inc.",
            "isin": make_isin("US"),
            "currency": "USD",
            "exchange": "NASDAQ",
            "symbol": "TSLA",
            "investment_type": "stock",
        }

        fund = FundService.create_fund(data)

        assert fund.investment_type == InvestmentType.STOCK

    def test_create_fund_as_fund(self, app_context, db_session):
        """Test creating fund with investment_type='fund'."""
        data = {
            "name": "Vanguard S&P 500",
            "isin": make_isin("US"),
            "currency": "USD",
            "exchange": "NYSE",
            "investment_type": "fund",
        }

        fund = FundService.create_fund(data)

        assert fund.investment_type == InvestmentType.FUND


class TestFundUpdate:
    """Tests for fund update operations."""

    def test_update_fund_basic(self, app_context, db_session):
        """Test updating basic fund fields."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Original Name",
            isin="US1234567890",
            currency="USD",
            exchange="NYSE",
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        # Update fund
        update_data = {
            "name": "Updated Name",
            "isin": "US0987654321",
            "currency": "EUR",
            "exchange": "LSE",
        }
        updated_fund, symbol_changed = FundService.update_fund(fund_id, update_data)

        # Verify update
        assert updated_fund.name == "Updated Name"
        assert updated_fund.isin == "US0987654321"
        assert updated_fund.currency == "EUR"
        assert updated_fund.exchange == "LSE"
        assert symbol_changed is False  # No symbol change

    def test_update_fund_add_symbol(self, app_context, db_session):
        """Test adding symbol to fund."""
        # Create fund without symbol
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        # Add symbol
        update_data = {
            "name": "Test Fund",
            "isin": fund.isin,
            "currency": "USD",
            "exchange": "NYSE",
            "symbol": "AAPL",
        }
        updated_fund, symbol_changed = FundService.update_fund(fund_id, update_data)

        assert updated_fund.symbol == "AAPL"
        assert symbol_changed is True

    def test_update_fund_change_symbol(self, app_context, db_session):
        """Test changing fund symbol."""
        # Create fund with symbol
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
            symbol="OLD",
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        # Change symbol
        update_data = {
            "name": "Test Fund",
            "isin": fund.isin,
            "currency": "USD",
            "exchange": "NYSE",
            "symbol": "NEW",
        }
        updated_fund, symbol_changed = FundService.update_fund(fund_id, update_data)

        assert updated_fund.symbol == "NEW"
        assert symbol_changed is True

    def test_update_fund_remove_symbol(self, app_context, db_session):
        """Test removing symbol from fund."""
        # Create fund with symbol
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
            symbol="AAPL",
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        # Remove symbol (don't include in update data)
        update_data = {
            "name": "Test Fund",
            "isin": fund.isin,
            "currency": "USD",
            "exchange": "NYSE",
        }
        updated_fund, symbol_changed = FundService.update_fund(fund_id, update_data)

        assert updated_fund.symbol is None
        assert symbol_changed is False  # No change tracked when clearing

    def test_update_fund_dividend_type(self, app_context, db_session):
        """Test updating dividend type."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
            dividend_type=DividendType.NONE,
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        # Update dividend type (use lowercase enum value)
        update_data = {
            "name": "Test Fund",
            "isin": fund.isin,
            "currency": "USD",
            "exchange": "NYSE",
            "dividend_type": "cash",  # Lowercase enum value
        }
        updated_fund, _ = FundService.update_fund(fund_id, update_data)

        assert updated_fund.dividend_type == DividendType.CASH

    def test_update_fund_investment_type(self, app_context, db_session):
        """Test updating investment type from fund to stock."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
            investment_type=InvestmentType.FUND,
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        # Update to stock
        update_data = {
            "name": "Test Stock",
            "isin": fund.isin,
            "currency": "USD",
            "exchange": "NYSE",
            "investment_type": "stock",
        }
        updated_fund, _ = FundService.update_fund(fund_id, update_data)

        assert updated_fund.investment_type == InvestmentType.STOCK
        assert updated_fund.name == "Test Stock"

    def test_update_fund_not_found(self, app_context, db_session):
        """Test update raises ValueError for nonexistent fund."""
        fake_id = make_id()
        update_data = {
            "name": "Test",
            "isin": "US1234567890",
            "currency": "USD",
            "exchange": "NYSE",
        }

        with pytest.raises(ValueError, match="not found"):
            FundService.update_fund(fake_id, update_data)


class TestFundDeletion:
    """Tests for fund deletion and usage checking."""

    def test_check_fund_usage_not_in_use(self, app_context, db_session):
        """Test checking usage for fund not in any portfolio."""
        # Create fund not attached to any portfolio
        fund = Fund(
            id=make_id(),
            name="Unused Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        db_session.add(fund)
        db_session.commit()

        # Check usage
        usage = FundService.check_fund_usage(fund.id)

        assert usage["in_use"] is False
        assert "portfolios" not in usage

    def test_check_fund_usage_in_portfolio_no_transactions(self, app_context, db_session):
        """Test fund in portfolio but no transactions."""
        # Create portfolio and fund
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, pf])
        db_session.commit()

        # Check usage
        usage = FundService.check_fund_usage(fund.id)

        # Not considered "in use" if no transactions
        assert usage["in_use"] is False

    def test_check_fund_usage_with_transactions(self, app_context, db_session):
        """Test fund with transactions shows as in use."""
        # Create portfolio, fund, and transaction
        portfolio = Portfolio(id=str(uuid.uuid4()), name="Active Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, pf])
        db_session.commit()

        # Add transaction
        from datetime import date

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        db_session.add(txn)
        db_session.commit()

        # Check usage
        usage = FundService.check_fund_usage(fund.id)

        assert usage["in_use"] is True
        assert "portfolios" in usage
        assert len(usage["portfolios"]) == 1
        assert usage["portfolios"][0]["name"] == "Active Portfolio"
        assert usage["portfolios"][0]["transaction_count"] == 1

    def test_delete_fund_success(self, app_context, db_session):
        """Test deleting fund not in any portfolio."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Fund to Delete",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id
        fund_name = fund.name

        # Delete fund
        result = FundService.delete_fund(fund_id)

        # Verify deletion
        assert result["fund_id"] == fund_id
        assert result["fund_name"] == fund_name

        # Verify fund no longer in database
        deleted_fund = db.session.get(Fund, fund_id)
        assert deleted_fund is None

    def test_delete_fund_with_prices(self, app_context, db_session):
        """Test deleting fund also deletes associated prices."""
        from datetime import date

        # Create fund with prices
        fund = Fund(
            id=make_id(),
            name="Fund to Delete",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        price1 = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 1, 1), price=10.0)
        price2 = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 1, 2), price=11.0)
        db_session.add_all([fund, price1, price2])
        db_session.commit()
        fund_id = fund.id

        # Delete fund
        FundService.delete_fund(fund_id)

        # Verify prices also deleted
        remaining_prices = FundPrice.query.filter_by(fund_id=fund_id).all()
        assert len(remaining_prices) == 0

    def test_delete_fund_in_portfolio(self, app_context, db_session):
        """Test deleting fund in portfolio raises ValueError."""
        # Create portfolio and fund
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Protected Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add_all([portfolio, fund, pf])
        db_session.commit()
        fund_id = fund.id

        # Attempt to delete fund
        with pytest.raises(ValueError, match=r"Cannot delete.*still attached"):
            FundService.delete_fund(fund_id)

        # Verify fund still exists
        existing_fund = db.session.get(Fund, fund_id)
        assert existing_fund is not None

    def test_delete_fund_not_found(self, app_context, db_session):
        """Test deleting nonexistent fund raises ValueError."""
        fake_id = make_id()

        with pytest.raises(ValueError, match="not found"):
            FundService.delete_fund(fake_id)


class TestEdgeCases:
    """Tests for edge cases and error handling."""

    def test_create_fund_duplicate_isin(self, app_context, db_session):
        """Test creating fund with duplicate ISIN raises error."""
        from sqlalchemy.exc import IntegrityError

        isin = make_isin("US")

        # Create first fund
        data1 = {"name": "Fund 1", "isin": isin, "currency": "USD", "exchange": "NYSE"}
        FundService.create_fund(data1)

        # Attempt to create second fund with same ISIN
        data2 = {"name": "Fund 2", "isin": isin, "currency": "EUR", "exchange": "LSE"}

        with pytest.raises(IntegrityError):
            FundService.create_fund(data2)
            db_session.commit()

    def test_update_fund_same_symbol_no_change(self, app_context, db_session):
        """Test updating fund with same symbol doesn't trigger change."""
        # Create fund with symbol
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
            symbol="AAPL",
        )
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        # Update with same symbol
        update_data = {
            "name": "Updated Name",
            "isin": fund.isin,
            "currency": "USD",
            "exchange": "NYSE",
            "symbol": "AAPL",  # Same symbol
        }
        updated_fund, symbol_changed = FundService.update_fund(fund_id, update_data)

        assert updated_fund.symbol == "AAPL"
        assert symbol_changed is False  # No change

    def test_check_fund_usage_multiple_portfolios(self, app_context, db_session):
        """Test fund used in multiple portfolios."""
        from datetime import date

        # Create fund and multiple portfolios
        fund = Fund(
            id=make_id(),
            name="Popular Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        portfolio1 = Portfolio(id=make_id(), name="Portfolio 1")
        portfolio2 = Portfolio(id=make_id(), name="Portfolio 2")

        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio1.id, fund_id=fund.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund.id)
        db_session.add_all([fund, portfolio1, portfolio2, pf1, pf2])
        db_session.commit()

        # Add transactions to both
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf1.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf2.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=50,
            cost_per_share=10.0,
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        # Check usage
        usage = FundService.check_fund_usage(fund.id)

        assert usage["in_use"] is True
        assert len(usage["portfolios"]) == 2
        portfolio_names = {p["name"] for p in usage["portfolios"]}
        assert "Portfolio 1" in portfolio_names
        assert "Portfolio 2" in portfolio_names
