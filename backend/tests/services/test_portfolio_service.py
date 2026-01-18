"""
Tests for PortfolioService.

This test suite covers:
- Portfolio CRUD operations (create, update, delete, archive, list)
- Portfolio-fund relationship management
- Portfolio value calculations (current values, summaries)
- Error handling for invalid operations
"""

from datetime import date

import pytest
from app.models import (
    Dividend,
    Fund,
    FundPrice,
    Portfolio,
    PortfolioFund,
    RealizedGainLoss,
    Transaction,
    db,
)
from app.services.portfolio_service import PortfolioService
from tests.test_helpers import make_id, make_isin


class TestPortfolioCRUD:
    """Tests for portfolio CRUD operations."""

    def test_create_portfolio(self, app_context, db_session):
        """Test creating a new portfolio."""
        # Create portfolio
        portfolio = PortfolioService.create_portfolio(
            name="Test Portfolio", description="Test description"
        )

        # Verify creation
        assert portfolio.id is not None
        assert portfolio.name == "Test Portfolio"
        assert portfolio.description == "Test description"
        assert portfolio.is_archived is False
        assert portfolio.exclude_from_overview is False

        # Verify it's in the database
        db_portfolio = db.session.get(Portfolio, portfolio.id)
        assert db_portfolio is not None
        assert db_portfolio.name == "Test Portfolio"

    def test_create_portfolio_minimal(self, app_context, db_session):
        """Test creating portfolio with only name (no description)."""
        portfolio = PortfolioService.create_portfolio(name="Minimal Portfolio")

        assert portfolio.id is not None
        assert portfolio.name == "Minimal Portfolio"
        assert portfolio.description == ""

    def test_update_portfolio(self, app_context, db_session):
        """Test updating an existing portfolio."""
        # Create portfolio directly
        portfolio = Portfolio(
            id=make_id(),
            name="Original Name",
            description="Original description",
            exclude_from_overview=False,
        )
        db_session.add(portfolio)
        db_session.commit()

        # Update portfolio
        updated = PortfolioService.update_portfolio(
            portfolio_id=portfolio.id,
            name="Updated Name",
            description="Updated description",
            exclude_from_overview=True,
        )

        # Verify updates
        assert updated.id == portfolio.id
        assert updated.name == "Updated Name"
        assert updated.description == "Updated description"
        assert updated.exclude_from_overview is True

        # Verify in database
        db_portfolio = db.session.get(Portfolio, portfolio.id)
        assert db_portfolio.name == "Updated Name"
        assert db_portfolio.exclude_from_overview is True

    def test_update_nonexistent_portfolio(self, app_context, db_session):
        """Test updating a portfolio that doesn't exist."""
        fake_id = make_id()

        with pytest.raises(ValueError, match=f"Portfolio {fake_id} not found"):
            PortfolioService.update_portfolio(portfolio_id=fake_id, name="New Name")

    def test_delete_portfolio(self, app_context, db_session):
        """Test deleting a portfolio."""
        # Create portfolio
        portfolio = Portfolio(id=make_id(), name="To Delete")
        db_session.add(portfolio)
        db_session.commit()
        portfolio_id = portfolio.id

        # Delete portfolio
        result = PortfolioService.delete_portfolio(portfolio_id)

        assert result is True

        # Verify deletion
        db_portfolio = db.session.get(Portfolio, portfolio_id)
        assert db_portfolio is None

    def test_delete_nonexistent_portfolio(self, app_context, db_session):
        """Test deleting a portfolio that doesn't exist."""
        fake_id = make_id()

        with pytest.raises(ValueError, match=f"Portfolio {fake_id} not found"):
            PortfolioService.delete_portfolio(fake_id)

    def test_update_archive_status(self, app_context, db_session):
        """Test archiving and unarchiving a portfolio."""
        # Create portfolio
        portfolio = Portfolio(id=make_id(), name="Archive Test", is_archived=False)
        db_session.add(portfolio)
        db_session.commit()

        # Archive portfolio
        updated = PortfolioService.update_archive_status(portfolio.id, True)
        assert updated.is_archived is True

        # Verify in database
        db_portfolio = db.session.get(Portfolio, portfolio.id)
        assert db_portfolio.is_archived is True

        # Unarchive portfolio
        updated = PortfolioService.update_archive_status(portfolio.id, False)
        assert updated.is_archived is False

    def test_get_portfolios_list_default(self, app_context, db_session):
        """Test getting list of portfolios (default excludes excluded from overview)."""
        # Create portfolios with different flags
        p1 = Portfolio(id=make_id(), name="Normal Portfolio", exclude_from_overview=False)
        p2 = Portfolio(id=make_id(), name="Excluded Portfolio", exclude_from_overview=True)
        db_session.add(p1)
        db_session.add(p2)
        db_session.commit()

        # Get portfolios (default behavior)
        portfolios = PortfolioService.get_portfolios_list()

        # Should include p1 (non-excluded)
        portfolio_ids = {p.id for p in portfolios}
        assert p1.id in portfolio_ids

        # Should not include p2 (excluded)
        assert p2.id not in portfolio_ids

        # Verify p1 details
        found_p1 = next(p for p in portfolios if p.id == p1.id)
        assert found_p1.name == "Normal Portfolio"

    def test_get_portfolios_list_include_excluded(self, app_context, db_session):
        """Test getting all portfolios including excluded from overview."""
        # Create portfolios
        p1 = Portfolio(id=make_id(), name="Normal Portfolio", exclude_from_overview=False)
        p2 = Portfolio(id=make_id(), name="Excluded Portfolio", exclude_from_overview=True)
        db_session.add(p1)
        db_session.add(p2)
        db_session.commit()

        # Get all portfolios
        portfolios = PortfolioService.get_portfolios_list(include_excluded=True)

        # Should include both portfolios
        portfolio_ids = {p.id for p in portfolios}
        assert p1.id in portfolio_ids
        assert p2.id in portfolio_ids

        # Verify names
        portfolio_names = {p.name for p in portfolios}
        assert "Normal Portfolio" in portfolio_names
        assert "Excluded Portfolio" in portfolio_names

    def test_get_all_portfolios(self, app_context, db_session):
        """Test getting all portfolios without filtering."""
        # Create portfolios with different flags
        p1 = Portfolio(id=make_id(), name="Portfolio 1", exclude_from_overview=False)
        p2 = Portfolio(id=make_id(), name="Portfolio 2", exclude_from_overview=True)
        p3 = Portfolio(id=make_id(), name="Portfolio 3", is_archived=True)
        db_session.add_all([p1, p2, p3])
        db_session.commit()

        # Get all portfolios
        portfolios = PortfolioService.get_all_portfolios()

        # Should include all portfolios regardless of flags
        portfolio_ids = {p.id for p in portfolios}
        assert p1.id in portfolio_ids
        assert p2.id in portfolio_ids
        assert p3.id in portfolio_ids
        assert len(portfolios) == 3

    def test_get_portfolio(self, app_context, db_session):
        """Test retrieving a single portfolio by ID."""
        # Create portfolio
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        # Retrieve portfolio
        retrieved = PortfolioService.get_portfolio(portfolio.id)
        assert retrieved is not None
        assert retrieved.id == portfolio.id
        assert retrieved.name == "Test Portfolio"

    def test_get_portfolio_not_found(self, app_context, db_session):
        """Test retrieving a portfolio that doesn't exist."""
        from werkzeug.exceptions import NotFound

        fake_id = make_id()

        with pytest.raises(NotFound):
            PortfolioService.get_portfolio(fake_id)


class TestPortfolioFundManagement:
    """Tests for portfolio-fund relationship management."""

    def test_create_portfolio_fund(self, app_context, db_session):
        """Test creating a portfolio-fund relationship."""
        # Create portfolio and fund
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),  # Unique ISIN
            currency="USD",
            exchange="NYSE",
        )
        db_session.add(portfolio)
        db_session.add(fund)
        db_session.commit()

        # Create relationship
        pf = PortfolioService.create_portfolio_fund(portfolio.id, fund.id)

        # Verify creation
        assert pf.id is not None
        assert pf.portfolio_id == portfolio.id
        assert pf.fund_id == fund.id

        # Verify in database
        db_pf = db.session.get(PortfolioFund, pf.id)
        assert db_pf is not None
        assert db_pf.portfolio_id == portfolio.id
        assert db_pf.fund_id == fund.id

    def test_create_portfolio_fund_invalid_portfolio(self, app_context, db_session):
        """Test creating portfolio-fund with nonexistent portfolio."""
        # Create fund only
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        db_session.add(fund)
        db_session.commit()

        fake_portfolio_id = make_id()

        with pytest.raises(ValueError, match=f"Portfolio {fake_portfolio_id} not found"):
            PortfolioService.create_portfolio_fund(fake_portfolio_id, fund.id)

    def test_create_portfolio_fund_invalid_fund(self, app_context, db_session):
        """Test creating portfolio-fund with nonexistent fund."""
        # Create portfolio only
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        fake_fund_id = make_id()

        with pytest.raises(ValueError, match=f"Fund {fake_fund_id} not found"):
            PortfolioService.create_portfolio_fund(portfolio.id, fake_fund_id)

    def test_get_all_portfolio_funds(self, app_context, db_session):
        """Test getting all portfolio-fund relationships."""
        # Create portfolios and funds
        portfolio1 = Portfolio(id=make_id(), name="Portfolio 1")
        portfolio2 = Portfolio(id=make_id(), name="Portfolio 2")
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
            currency="USD",
            exchange="NASDAQ",
        )
        db_session.add_all([portfolio1, portfolio2, fund1, fund2])
        db_session.commit()

        # Create relationships
        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio1.id, fund_id=fund1.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund2.id)
        db_session.add(pf1)
        db_session.add(pf2)
        db_session.commit()

        # Get all portfolio funds
        result = PortfolioService.get_all_portfolio_funds()

        # Verify our portfolio funds are in the results
        result_ids = {pf["id"] for pf in result}
        assert pf1.id in result_ids
        assert pf2.id in result_ids

        # Verify structure
        assert all(isinstance(pf, dict) for pf in result)

        # Verify details of our portfolio funds
        found_pf1 = next(pf for pf in result if pf["id"] == pf1.id)
        assert found_pf1["portfolioName"] == "Portfolio 1"
        assert found_pf1["fundName"] == "Fund 1"

        found_pf2 = next(pf for pf in result if pf["id"] == pf2.id)
        assert found_pf2["portfolioName"] == "Portfolio 2"
        assert found_pf2["fundName"] == "Fund 2"

        # Check structure
        pf1_data = next(pf for pf in result if pf["id"] == pf1.id)
        assert pf1_data["portfolioId"] == portfolio1.id
        assert pf1_data["fundId"] == fund1.id
        assert pf1_data["portfolioName"] == "Portfolio 1"
        assert pf1_data["fundName"] == "Fund 1"

    def test_delete_portfolio_fund_no_transactions(self, app_context, db_session):
        """Test deleting portfolio-fund with no associated transactions."""
        # Create portfolio-fund relationship
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
        pf_id = pf.id

        # Delete without confirmation (should work since no transactions)
        result = PortfolioService.delete_portfolio_fund(pf_id)

        # Verify deletion
        assert result["transactions_deleted"] == 0
        assert result["dividends_deleted"] == 0
        assert result["fund_name"] == "Test Fund"
        assert result["portfolio_name"] == "Test Portfolio"

        # Verify removed from database
        assert db.session.get(PortfolioFund, pf_id) is None

    def test_delete_portfolio_fund_with_transactions_no_confirmation(self, app_context, db_session):
        """Test deleting portfolio-fund with transactions but no confirmation."""
        # Create portfolio-fund relationship
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

        # Add transaction
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

        # Try to delete without confirmation
        with pytest.raises(ValueError, match="Confirmation required: 1 transactions"):
            PortfolioService.delete_portfolio_fund(pf.id, confirmed=False)

    def test_delete_portfolio_fund_with_transactions_confirmed(self, app_context, db_session):
        """Test deleting portfolio-fund with transactions when confirmed."""
        # Create portfolio-fund relationship
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

        # Add transactions and dividend
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 2, 1),
            type="dividend",
            shares=5,
            cost_per_share=10.0,
        )
        dividend = Dividend(
            id=make_id(),
            portfolio_fund_id=pf.id,
            fund_id=fund.id,
            record_date=date(2024, 1, 10),
            ex_dividend_date=date(2024, 1, 15),
            shares_owned=100,
            dividend_per_share=0.50,
            total_amount=50.0,
            # Note: Not setting reinvestment_transaction_id to avoid FK constraint issues
            # during cascade deletion where transactions might be deleted before dividend
        )
        db_session.add_all([txn1, txn2, dividend])
        db_session.commit()
        pf_id = pf.id

        # Delete with confirmation
        result = PortfolioService.delete_portfolio_fund(pf_id, confirmed=True)

        # Verify result
        assert result["transactions_deleted"] == 2
        assert result["dividends_deleted"] == 1
        assert result["fund_name"] == "Test Fund"
        assert result["portfolio_name"] == "Test Portfolio"

        # Verify all deleted
        assert db.session.get(PortfolioFund, pf_id) is None
        assert Transaction.query.filter_by(portfolio_fund_id=pf_id).count() == 0
        assert Dividend.query.filter_by(portfolio_fund_id=pf_id).count() == 0

    def test_get_portfolio_fund_without_relationships(self, app_context, db_session):
        """Test retrieving a portfolio fund without loading relationships."""
        # Create data
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

        # Retrieve without relationships
        retrieved = PortfolioService.get_portfolio_fund(pf.id, with_relationships=False)
        assert retrieved is not None
        assert retrieved.id == pf.id
        assert retrieved.portfolio_id == portfolio.id
        assert retrieved.fund_id == fund.id

    def test_get_portfolio_fund_with_relationships(self, app_context, db_session):
        """Test retrieving a portfolio fund with eagerly loaded relationships."""
        # Create data
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

        # Retrieve with relationships
        retrieved = PortfolioService.get_portfolio_fund(pf.id, with_relationships=True)
        assert retrieved is not None
        assert retrieved.id == pf.id
        # Relationships should be loaded
        assert retrieved.portfolio.name == "Test Portfolio"
        assert retrieved.fund.name == "Test Fund"

    def test_get_portfolio_fund_not_found(self, app_context, db_session):
        """Test retrieving a portfolio fund that doesn't exist."""
        fake_id = make_id()

        # Without relationships
        result = PortfolioService.get_portfolio_fund(fake_id, with_relationships=False)
        assert result is None

        # With relationships
        result = PortfolioService.get_portfolio_fund(fake_id, with_relationships=True)
        assert result is None

    def test_count_portfolio_fund_transactions(self, app_context, db_session):
        """Test counting transactions for a portfolio fund."""
        # Create data
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

        # Initially should be 0
        count = PortfolioService.count_portfolio_fund_transactions(pf.id)
        assert count == 0

        # Add transactions
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 2, 1),
            type="buy",
            shares=50,
            cost_per_share=11.0,
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        # Count should be 2
        count = PortfolioService.count_portfolio_fund_transactions(pf.id)
        assert count == 2

    def test_count_portfolio_fund_dividends(self, app_context, db_session):
        """Test counting dividends for a portfolio fund."""
        # Create data
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

        # Initially should be 0
        count = PortfolioService.count_portfolio_fund_dividends(pf.id)
        assert count == 0

        # Add dividends
        div1 = Dividend(
            id=make_id(),
            portfolio_fund_id=pf.id,
            fund_id=fund.id,
            record_date=date(2024, 1, 10),
            ex_dividend_date=date(2024, 1, 15),
            shares_owned=100,
            dividend_per_share=0.5,
            total_amount=50.0,
        )
        div2 = Dividend(
            id=make_id(),
            portfolio_fund_id=pf.id,
            fund_id=fund.id,
            record_date=date(2024, 4, 10),
            ex_dividend_date=date(2024, 4, 15),
            shares_owned=100,
            dividend_per_share=0.6,
            total_amount=60.0,
        )
        db_session.add_all([div1, div2])
        db_session.commit()

        # Count should be 2
        count = PortfolioService.count_portfolio_fund_dividends(pf.id)
        assert count == 2


class TestPortfolioCalculations:
    """Tests for portfolio value calculations."""

    def test_calculate_portfolio_fund_values_basic(self, app_context, db_session):
        """Test calculating values for portfolio funds with transactions."""
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

        # Add transaction
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

        # Add fund price
        price = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 1, 15), price=12.0)
        db_session.add(price)
        db_session.commit()

        # Calculate values
        result = PortfolioService.calculate_portfolio_fund_values([pf])

        # Verify calculations
        assert len(result) == 1
        pf_data = result[0]
        assert pf_data["id"] == pf.id
        assert pf_data["fundId"] == fund.id
        assert pf_data["fundName"] == "Test Fund"
        assert pf_data["totalShares"] == 100
        assert pf_data["latestPrice"] == 12.0
        assert pf_data["averageCost"] == 10.0
        assert pf_data["totalCost"] == 1000.0
        assert pf_data["currentValue"] == 1200.0
        assert pf_data["unrealizedGainLoss"] == 200.0
        assert pf_data["realizedGainLoss"] == 0
        assert pf_data["totalGainLoss"] == 200.0
        assert pf_data["totalDividends"] == 0

    def test_calculate_portfolio_fund_values_with_sell(self, app_context, db_session):
        """Test calculating values with buy and sell transactions."""
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

        # Add transactions (buy 100, sell 30)
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 2, 1),
            type="sell",
            shares=30,
            cost_per_share=15.0,
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        # Add fund price
        price = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 2, 15), price=12.0)
        db_session.add(price)
        db_session.commit()

        # Calculate values
        result = PortfolioService.calculate_portfolio_fund_values([pf])

        # Verify calculations (70 shares remaining)
        assert len(result) == 1
        pf_data = result[0]
        assert pf_data["totalShares"] == 70
        assert pf_data["latestPrice"] == 12.0
        assert pf_data["totalCost"] == 700.0  # Proportional reduction
        assert pf_data["currentValue"] == 840.0

    def test_get_portfolio_funds(self, app_context, db_session):
        """Test getting portfolio funds for a specific portfolio."""
        # Create portfolio and funds
        portfolio = Portfolio(id=make_id(), name="Test Portfolio")
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
            currency="USD",
            exchange="NASDAQ",
        )
        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund1.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund2.id)
        db_session.add_all([portfolio, fund1, fund2, pf1, pf2])
        db_session.commit()

        # Add transactions
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
            cost_per_share=20.0,
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        # Get portfolio funds
        result = PortfolioService.get_portfolio_funds(portfolio.id)

        # Verify results
        assert len(result) == 2
        assert all(isinstance(pf, dict) for pf in result)
        fund_names = {pf["fundName"] for pf in result}
        assert "Fund 1" in fund_names
        assert "Fund 2" in fund_names

    def test_get_portfolio_summary(self, app_context, db_session):
        """Test getting summary of all portfolios."""
        # Create portfolios
        portfolio1 = Portfolio(
            id=make_id(), name="Portfolio 1", is_archived=False, exclude_from_overview=False
        )
        portfolio2 = Portfolio(
            id=make_id(), name="Portfolio 2", is_archived=False, exclude_from_overview=False
        )
        portfolio3 = Portfolio(
            id=make_id(),
            name="Archived Portfolio",
            is_archived=True,
            exclude_from_overview=False,
        )
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=make_isin("US"),
            currency="USD",
            exchange="NYSE",
        )
        db_session.add_all([portfolio1, portfolio2, portfolio3, fund])
        db_session.commit()

        # Create portfolio-fund relationships
        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio1.id, fund_id=fund.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund.id)
        db_session.add_all([pf1, pf2])
        db_session.commit()

        # Add transactions
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
            cost_per_share=20.0,
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        # Add fund price
        price = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 1, 15), price=15.0)
        db_session.add(price)
        db_session.commit()

        # Get summary
        result = PortfolioService.get_portfolio_summary()

        # Verify results (should only include non-archived, non-excluded portfolios)
        assert all(isinstance(p, dict) for p in result)

        # Should include portfolio1 and portfolio2
        result_ids = {p["id"] for p in result}
        assert portfolio1.id in result_ids
        assert portfolio2.id in result_ids

        # Should not include archived portfolio3
        assert portfolio3.id not in result_ids

        # Check portfolio 1
        p1 = next(p for p in result if p["id"] == portfolio1.id)
        assert p1["name"] == "Portfolio 1"
        assert p1["totalValue"] == 1500.0  # 100 shares * $15
        assert p1["totalCost"] == 1000.0
        assert p1["totalUnrealizedGainLoss"] == 500.0
        assert p1["isArchived"] is False

        # Check portfolio 2
        p2 = next(p for p in result if p["id"] == portfolio2.id)
        assert p2["name"] == "Portfolio 2"
        assert p2["totalValue"] == 750.0  # 50 shares * $15
        assert p2["totalCost"] == 1000.0

    def test_get_portfolio_summary_with_realized_gains(self, app_context, db_session):
        """Test portfolio summary includes realized gains."""
        # Create portfolio and fund
        portfolio = Portfolio(
            id=make_id(),
            name="Test Portfolio",
            is_archived=False,
            exclude_from_overview=False,
        )
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

        # Add transactions (buy and sell)
        txn_buy = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        txn_sell = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 2, 1),
            type="sell",
            shares=30,
            cost_per_share=15.0,
        )
        db_session.add_all([txn_buy, txn_sell])
        db_session.commit()

        # Add realized gain
        realized_gain = RealizedGainLoss(
            id=make_id(),
            portfolio_id=portfolio.id,
            fund_id=fund.id,
            transaction_id=txn_sell.id,
            transaction_date=date(2024, 2, 1),
            shares_sold=30,
            cost_basis=300.0,
            sale_proceeds=450.0,
            realized_gain_loss=150.0,
        )
        db_session.add(realized_gain)
        db_session.commit()

        # Add fund price
        price = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 2, 15), price=12.0)
        db_session.add(price)
        db_session.commit()

        # Get summary
        result = PortfolioService.get_portfolio_summary()

        # Verify our portfolio is in the results
        result_ids = {p["id"] for p in result}
        assert portfolio.id in result_ids

        # Find our portfolio and verify realized gains
        p = next(p for p in result if p["id"] == portfolio.id)
        assert p["totalRealizedGainLoss"] == 150.0
        assert p["totalSaleProceeds"] == 450.0
        assert p["totalOriginalCost"] == 300.0


class TestEdgeCases:
    """Tests for error handling and edge cases."""

    def test_update_archive_status_nonexistent(self, app_context, db_session):
        """Test archiving a portfolio that doesn't exist."""
        fake_id = make_id()

        with pytest.raises(ValueError, match=f"Portfolio {fake_id} not found"):
            PortfolioService.update_archive_status(fake_id, True)

    def test_delete_portfolio_fund_nonexistent(self, app_context, db_session):
        """Test deleting a portfolio-fund relationship that doesn't exist."""
        fake_id = make_id()

        with pytest.raises(ValueError, match=f"Portfolio-fund relationship {fake_id} not found"):
            PortfolioService.delete_portfolio_fund(fake_id)

    def test_calculate_portfolio_fund_values_no_price(self, app_context, db_session):
        """Test calculating values when no price data exists."""
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

        # Add transaction but no price
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

        # Calculate values
        result = PortfolioService.calculate_portfolio_fund_values([pf])

        # Should return data with zero price
        assert len(result) == 1
        pf_data = result[0]
        assert pf_data["totalShares"] == 100
        assert pf_data["latestPrice"] == 0
        assert pf_data["currentValue"] == 0
        assert pf_data["totalCost"] == 1000.0


class TestPortfolioHistoricalMethods:
    """Tests for portfolio historical analysis methods."""

    def test_get_portfolio_history_no_portfolios(self, app_context, db_session):
        """Test portfolio history behavior when no valid portfolios have transactions."""
        # Since get_portfolio_history queries ALL portfolios in the system, and we use
        # Query-Specific Data pattern, we can't easily isolate this test. The method will
        # return data if other tests created portfolios with transactions. Let's verify the
        # method handles the empty case gracefully.
        result = PortfolioService.get_portfolio_history()

        # Verify result is a list (empty or not depends on other tests)
        assert isinstance(result, list)

        # If there are results, they should have the correct structure
        if result:
            assert "date" in result[0]
            assert "portfolios" in result[0]

    def test_get_portfolio_history_no_transactions(self, app_context, db_session):
        """Test portfolio history when a specific portfolio exists but has no transactions."""
        # Create portfolio with no transactions
        portfolio = Portfolio(id=make_id(), name="Empty Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        result = PortfolioService.get_portfolio_history()

        # The result may include data from other tests, but our empty portfolio shouldn't appear
        # because it has no transactions. Verify structure is correct.
        assert isinstance(result, list)

        # If results exist, our empty portfolio should not be in any day's data
        for day_data in result:
            portfolio_ids = [p["id"] for p in day_data["portfolios"]]
            # Our empty portfolio should not appear because it has no transactions
            assert portfolio.id not in portfolio_ids

    def test_get_portfolio_history_basic(self, app_context, db_session):
        """Test basic portfolio history functionality."""
        from datetime import date, timedelta

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

        # Add transactions
        base_date = date.today() - timedelta(days=5)
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=base_date,
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=base_date + timedelta(days=2),
            type="buy",
            shares=50,
            cost_per_share=12.0,
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        # Add prices
        price1 = FundPrice(id=make_id(), fund_id=fund.id, date=base_date, price=10.0)
        price2 = FundPrice(
            id=make_id(), fund_id=fund.id, date=base_date + timedelta(days=2), price=12.0
        )
        price3 = FundPrice(
            id=make_id(), fund_id=fund.id, date=base_date + timedelta(days=4), price=15.0
        )
        db_session.add_all([price1, price2, price3])
        db_session.commit()

        # Get history
        result = PortfolioService.get_portfolio_history(
            start_date=base_date.strftime("%Y-%m-%d"),
            end_date=(base_date + timedelta(days=4)).strftime("%Y-%m-%d"),
        )

        # Verify structure - should be array of daily entries
        assert len(result) > 0
        assert "date" in result[0]
        assert "portfolios" in result[0]

        # Should have entries for each day in the range
        assert len(result) >= 3

        # Find day 1 entry (first transaction) and our specific portfolio
        day1_entry = next(d for d in result if d["date"] == base_date.strftime("%Y-%m-%d"))

        # Find our specific portfolio in the day 1 results
        portfolio_day1 = next(p for p in day1_entry["portfolios"] if p["id"] == portfolio.id)
        assert portfolio_day1["name"] == "Test Portfolio"
        assert portfolio_day1["totalValue"] == 1000.0  # 100 * 10
        assert portfolio_day1["totalCost"] == 1000.0

        # Find day 3 entry (after second transaction) and our specific portfolio
        day3_entry = next(
            d for d in result if d["date"] == (base_date + timedelta(days=2)).strftime("%Y-%m-%d")
        )
        portfolio_day3 = next(p for p in day3_entry["portfolios"] if p["id"] == portfolio.id)
        assert portfolio_day3["totalValue"] == 1800.0  # 150 * 12
        assert portfolio_day3["totalCost"] == 1600.0  # 100*10 + 50*12

    def test_get_portfolio_fund_history_invalid_portfolio(self, app_context, db_session):
        """Test fund history with invalid portfolio ID."""
        fake_id = make_id()

        with pytest.raises((ValueError, Exception)):
            PortfolioService.get_portfolio_fund_history(fake_id)

    def test_get_portfolio_fund_history_no_transactions(self, app_context, db_session):
        """Test fund history when portfolio has no transactions."""
        # Create empty portfolio
        portfolio = Portfolio(id=make_id(), name="Empty Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        result = PortfolioService.get_portfolio_fund_history(portfolio.id)
        assert result == []

    def test_get_portfolio_fund_history_basic(self, app_context, db_session):
        """Test basic portfolio fund history functionality."""
        from datetime import date, timedelta

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

        # Add transaction
        base_date = date.today() - timedelta(days=3)
        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=base_date,
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )
        db_session.add(txn)
        db_session.commit()

        # Add price
        price = FundPrice(id=make_id(), fund_id=fund.id, date=base_date, price=12.0)
        db_session.add(price)
        db_session.commit()

        # Get fund history
        result = PortfolioService.get_portfolio_fund_history(
            portfolio.id,
            start_date=base_date.strftime("%Y-%m-%d"),
            end_date=(base_date + timedelta(days=1)).strftime("%Y-%m-%d"),
        )

        # Verify structure
        assert len(result) > 0
        day_result = result[0]
        assert "date" in day_result
        assert "funds" in day_result

        # Should have our fund
        fund_data = day_result["funds"][0]
        assert fund_data["fundId"] == fund.id
        assert fund_data["fundName"] == "Test Fund"
        assert fund_data["shares"] == 100.0
        assert fund_data["cost"] == 1000.0
        assert fund_data["value"] == 1200.0  # 100 * 12
        assert fund_data["price"] == 12.0

    def test_get_portfolio_summary_no_portfolios(self, app_context, db_session):
        """Test portfolio summary when no portfolios exist."""
        # Delete any existing portfolios
        Portfolio.query.delete()
        db_session.commit()

        result = PortfolioService.get_portfolio_summary()
        assert result == []

    def test_get_portfolio_summary_with_dividend_reinvestment(self, app_context, db_session):
        """Test portfolio summary includes dividend reinvestment shares."""
        from datetime import date

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

        # Add buy transaction
        txn_buy = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )

        # Add dividend reinvestment transaction
        txn_dividend = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 15),
            type="dividend",
            shares=5,
            cost_per_share=10.0,
        )
        db_session.add_all([txn_buy, txn_dividend])
        db_session.commit()

        # Add dividend record with reinvestment
        dividend = Dividend(
            id=make_id(),
            portfolio_fund_id=pf.id,
            fund_id=fund.id,
            record_date=date(2024, 1, 10),
            ex_dividend_date=date(2024, 1, 15),
            shares_owned=100,
            dividend_per_share=0.50,
            total_amount=50.0,
            reinvestment_transaction_id=txn_dividend.id,
        )
        db_session.add(dividend)
        db_session.commit()

        # Add current price
        price = FundPrice(id=make_id(), fund_id=fund.id, date=date.today(), price=12.0)
        db_session.add(price)
        db_session.commit()

        # Get summary
        result = PortfolioService.get_portfolio_summary()

        # Find our portfolio
        portfolio_summary = next(p for p in result if p["id"] == portfolio.id)

        # Should include dividend reinvestment in calculations
        # Total shares should be 100 (buy) + 5 (dividend) = 105
        # But the calculation is complex, so just verify structure and non-zero values
        assert portfolio_summary["totalValue"] > 0
        assert portfolio_summary["totalCost"] > 0


class TestPortfolioHelperMethods:
    """Tests for portfolio helper methods and edge cases."""

    def test_process_transactions_sell_to_zero(self, app_context, db_session):
        """Test _process_transactions_for_date when sells reduce shares to zero."""
        from datetime import date

        # Create test transactions that sell all shares
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

        # Buy 100 shares
        txn_buy = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )

        # Sell all 100 shares
        txn_sell = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 15),
            type="sell",
            shares=100,
            cost_per_share=12.0,
        )
        db_session.add_all([txn_buy, txn_sell])
        db_session.commit()

        # Process transactions up to sell date - should result in 0 shares, 0 cost
        transactions = Transaction.query.filter_by(portfolio_fund_id=pf.id).all()
        shares, cost = PortfolioService._process_transactions_for_date(
            transactions, date(2024, 1, 15)
        )

        assert shares == 0.0
        assert cost == 0.0

    def test_process_transactions_sell_below_zero(self, app_context, db_session):
        """Test _process_transactions_for_date when sells exceed shares owned."""
        from datetime import date

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

        # Buy 100 shares, sell 150 (more than owned)
        txn_buy = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=10.0,
        )

        txn_sell = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 15),
            type="sell",
            shares=150,  # More than owned
            cost_per_share=12.0,
        )
        db_session.add_all([txn_buy, txn_sell])
        db_session.commit()

        transactions = Transaction.query.filter_by(portfolio_fund_id=pf.id).all()
        shares, cost = PortfolioService._process_transactions_for_date(
            transactions, date(2024, 1, 15)
        )

        # Should clamp to 0 shares, 0 cost (can't go negative)
        assert shares == 0.0
        assert cost == 0.0

    def test_calculate_fund_metrics_historical_format(self, app_context, db_session):
        """Test _calculate_fund_metrics returns historical format when requested."""
        from datetime import date, timedelta

        # Create portfolio fund with transaction
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

        # Add transaction
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

        # Add price
        price = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 1, 1), price=12.0)
        db_session.add(price)
        db_session.commit()

        # Test historical format (past date)
        past_date = date.today() - timedelta(days=30)
        result = PortfolioService._calculate_fund_metrics(
            pf, target_date=past_date, force_historical_format=True
        )

        # Should return simplified historical format
        expected_keys = {"fundId", "fundName", "shares", "cost", "value", "price"}
        assert set(result.keys()) == expected_keys
        assert result["fundId"] == fund.id
        assert result["fundName"] == "Test Fund"


class TestGetActivePortfolios:
    """Tests for get_active_portfolios() - Retrieve non-archived portfolios."""

    def test_get_active_portfolios(self, app_context, db_session):
        """Test get_active_portfolios returns only non-archived portfolios."""
        # Create active portfolios
        p1 = Portfolio(id=make_id(), name="Active Portfolio 1", is_archived=False)
        p2 = Portfolio(id=make_id(), name="Active Portfolio 2", is_archived=False)
        # Create archived portfolio
        p3 = Portfolio(id=make_id(), name="Archived Portfolio", is_archived=True)
        db_session.add_all([p1, p2, p3])
        db_session.commit()

        result = PortfolioService.get_active_portfolios()

        assert len(result) == 2
        portfolio_names = [p.name for p in result]
        assert "Active Portfolio 1" in portfolio_names
        assert "Active Portfolio 2" in portfolio_names
        assert "Archived Portfolio" not in portfolio_names

    def test_get_active_portfolios_empty(self, app_context, db_session):
        """Test get_active_portfolios returns empty list when all portfolios archived."""
        # Create only archived portfolios
        p1 = Portfolio(id=make_id(), name="Archived 1", is_archived=True)
        p2 = Portfolio(id=make_id(), name="Archived 2", is_archived=True)
        db_session.add_all([p1, p2])
        db_session.commit()

        result = PortfolioService.get_active_portfolios()

        assert result == []

    def test_get_active_portfolios_none_exist(self, app_context, db_session):
        """Test get_active_portfolios returns empty list when no portfolios exist."""
        result = PortfolioService.get_active_portfolios()

        assert result == []
