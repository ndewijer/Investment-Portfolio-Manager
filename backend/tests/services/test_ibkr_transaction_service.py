"""
Tests for IBKRTransactionService - IBKR transaction processing and allocation.

This test suite covers:
- Allocation validation (must sum to 100%)
- Fund creation and matching (ISIN and symbol matching)
- Portfolio-fund relationship management
- Transaction allocation across portfolios
- Modifying existing allocations
- Dividend matching to existing dividend records
"""

from datetime import date

import pytest
from app.models import (
    Dividend,
    Fund,
    IBKRTransaction,
    IBKRTransactionAllocation,
    InvestmentType,
    Portfolio,
    PortfolioFund,
    ReinvestmentStatus,
    Transaction,
    db,
)
from app.services.ibkr_transaction_service import IBKRTransactionService
from tests.test_helpers import (
    make_dividend_txn_id,
    make_ibkr_txn_id,
    make_id,
    make_isin,
    make_symbol,
)


@pytest.fixture
def sample_portfolio(app_context, db_session):
    """Create a test portfolio."""
    portfolio = Portfolio(id=make_id(), name="Test Portfolio")
    db.session.add(portfolio)
    db.session.commit()
    return portfolio


@pytest.fixture
def second_portfolio(app_context, db_session):
    """Create a second test portfolio for multi-allocation tests."""
    portfolio = Portfolio(id=make_id(), name="Second Portfolio")
    db.session.add(portfolio)
    db.session.commit()
    return portfolio


@pytest.fixture
def sample_fund(app_context, db_session):
    """Create a test fund with unique ISIN per test."""
    # Use unique ISIN to avoid conflicts between tests
    unique_isin = make_isin("US")
    fund = Fund(
        id=make_id(),
        name="Apple Inc",
        isin=unique_isin,
        symbol=make_symbol("AAPL"),  # Make symbol unique too
        currency="USD",
        exchange="NASDAQ",
        investment_type=InvestmentType.STOCK,
    )
    db.session.add(fund)
    db.session.commit()
    return fund


@pytest.fixture
def sample_ibkr_transaction(app_context, db_session):
    """Create a sample IBKR transaction."""
    txn = IBKRTransaction(
        id=make_id(),
        ibkr_transaction_id=make_ibkr_txn_id(),
        transaction_date=date(2025, 1, 15),
        symbol="AAPL",
        isin="US0378331005",
        description="APPLE INC",
        transaction_type="buy",
        quantity=100,
        price=150.00,
        total_amount=15000.00,
        currency="USD",
        fees=1.50,
        status="pending",
        raw_data="{}",
    )
    db.session.add(txn)
    db.session.commit()
    return txn


class TestAllocationValidation:
    """Tests for allocation percentage validation."""

    def test_validate_allocations_success_100_percent(self):
        """Test that allocations summing to 100% are valid."""
        allocations = [
            {"portfolio_id": make_id(), "percentage": 50.0},
            {"portfolio_id": make_id(), "percentage": 50.0},
        ]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is True
        assert _error == ""

    def test_validate_allocations_single_100_percent(self):
        """Test that single allocation of 100% is valid."""
        allocations = [{"portfolio_id": make_id(), "percentage": 100.0}]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is True

    def test_validate_allocations_three_way_split(self):
        """Test three-way allocation split."""
        allocations = [
            {"portfolio_id": make_id(), "percentage": 40.0},
            {"portfolio_id": make_id(), "percentage": 35.0},
            {"portfolio_id": make_id(), "percentage": 25.0},
        ]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is True

    def test_validate_allocations_floating_point_acceptable(self):
        """Test that small floating point errors are acceptable."""
        allocations = [
            {"portfolio_id": make_id(), "percentage": 33.33},
            {"portfolio_id": make_id(), "percentage": 33.33},
            {"portfolio_id": make_id(), "percentage": 33.34},
        ]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is True

    def test_validate_allocations_too_high(self):
        """Test that allocations over 100% are invalid."""
        allocations = [
            {"portfolio_id": make_id(), "percentage": 60.0},
            {"portfolio_id": make_id(), "percentage": 50.0},
        ]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is False
        assert "110" in _error

    def test_validate_allocations_too_low(self):
        """Test that allocations under 100% are invalid."""
        allocations = [
            {"portfolio_id": make_id(), "percentage": 40.0},
            {"portfolio_id": make_id(), "percentage": 40.0},
        ]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is False
        assert "80" in _error

    def test_validate_allocations_empty_list(self):
        """Test that empty allocation list is invalid."""
        allocations = []

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is False
        assert "At least one allocation" in _error

    def test_validate_allocations_negative_percentage(self):
        """Test that negative percentages are invalid."""
        allocations = [
            {"portfolio_id": make_id(), "percentage": 120.0},
            {"portfolio_id": make_id(), "percentage": -20.0},
        ]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is False
        assert "positive" in _error.lower()

    def test_validate_allocations_missing_portfolio_id(self):
        """Test that missing portfolio_id is invalid."""
        allocations = [{"percentage": 100.0}]

        is_valid, _error = IBKRTransactionService.validate_allocations(allocations)
        assert is_valid is False
        assert "portfolio_id" in _error


class TestFundCreation:
    """Tests for get_or_create_fund method."""

    def test_get_existing_fund_by_isin(self, app_context, db_session, sample_fund):
        """Test that existing fund is retrieved by ISIN."""
        fund = IBKRTransactionService._get_or_create_fund(
            symbol="DIFFERENT_SYMBOL", isin=sample_fund.isin, currency="USD"
        )

        assert fund.id == sample_fund.id
        # Should use existing fund even with different symbol in request
        assert fund.isin == sample_fund.isin

        # Should not create duplicate
        assert Fund.query.filter_by(isin=sample_fund.isin).count() == 1

    def test_get_existing_fund_by_symbol(self, app_context, db_session):
        """Test that existing fund is retrieved by symbol when ISIN doesn't match."""
        # Use unique symbol to avoid conflicts with other tests
        unique_symbol = make_symbol("AAPL", 6)
        unique_isin = make_isin("US")

        existing = Fund(
            id=make_id(),
            name="Apple",
            isin=unique_isin,
            symbol=unique_symbol,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(existing)
        db.session.commit()

        fund = IBKRTransactionService._get_or_create_fund(
            symbol=unique_symbol, isin=None, currency="USD"
        )

        assert fund.id == existing.id
        # Should not create duplicate - verify only the one we created exists with this symbol
        assert Fund.query.filter_by(symbol=unique_symbol).count() == 1

    def test_create_new_fund_with_symbol_and_isin(self, app_context, db_session):
        """Test creating new fund with both symbol and ISIN."""
        fund = IBKRTransactionService._get_or_create_fund(
            symbol="MSFT", isin="US5949181045", currency="USD"
        )

        assert fund is not None
        assert fund.symbol == "MSFT"
        assert fund.isin == "US5949181045"
        assert fund.currency == "USD"
        assert fund.exchange == "UNKNOWN"  # Default value
        assert fund.investment_type == InvestmentType.STOCK  # Default

        # Verify saved to database
        db_fund = Fund.query.filter_by(isin="US5949181045").first()
        assert db_fund is not None

    def test_create_new_fund_with_only_symbol(self, app_context, db_session):
        """Test creating new fund with only symbol (no ISIN)."""
        fund = IBKRTransactionService._get_or_create_fund(symbol="TSLA", isin=None, currency="USD")

        assert fund is not None
        assert fund.symbol == "TSLA"
        assert fund.isin.startswith("UNKNOWN_")  # Placeholder ISIN

    def test_create_new_fund_name_uses_symbol(self, app_context, db_session):
        """Test that fund name defaults to symbol."""
        fund = IBKRTransactionService._get_or_create_fund(
            symbol="GOOGL", isin="US02079K3059", currency="USD"
        )

        assert fund.name == "GOOGL"


class TestPortfolioFundCreation:
    """Tests for get_or_create_portfolio_fund method."""

    def test_get_existing_portfolio_fund(
        self, app_context, db_session, sample_portfolio, sample_fund
    ):
        """Test retrieving existing portfolio-fund relationship."""
        # Create existing relationship
        existing_pf = PortfolioFund(
            id=make_id(),
            portfolio_id=sample_portfolio.id,
            fund_id=sample_fund.id,
        )
        db.session.add(existing_pf)
        db.session.commit()

        # Should retrieve existing
        pf = IBKRTransactionService._get_or_create_portfolio_fund(
            sample_portfolio.id, sample_fund.id
        )

        assert pf.id == existing_pf.id
        assert PortfolioFund.query.count() == 1

    def test_create_new_portfolio_fund(
        self, app_context, db_session, sample_portfolio, sample_fund
    ):
        """Test creating new portfolio-fund relationship."""
        pf = IBKRTransactionService._get_or_create_portfolio_fund(
            sample_portfolio.id, sample_fund.id
        )

        assert pf is not None
        assert pf.portfolio_id == sample_portfolio.id
        assert pf.fund_id == sample_fund.id

        # Verify saved
        db_pf = PortfolioFund.query.filter_by(
            portfolio_id=sample_portfolio.id, fund_id=sample_fund.id
        ).first()
        assert db_pf is not None


class TestProcessTransactionAllocation:
    """Tests for processing IBKR transactions with allocations."""

    def test_process_single_allocation_100_percent(
        self, app_context, db_session, sample_portfolio, sample_ibkr_transaction
    ):
        """Test processing transaction with single 100% allocation."""
        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(
            sample_ibkr_transaction.id, allocations
        )

        assert result["success"] is True
        assert "fund_id" in result
        assert len(result["created_transactions"]) == 1

        # Verify IBKR transaction marked as processed
        ibkr_txn = IBKRTransaction.query.get(sample_ibkr_transaction.id)
        assert ibkr_txn.status == "processed"
        assert ibkr_txn.processed_at is not None

        # Verify allocation record created
        alloc = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=sample_ibkr_transaction.id
        ).first()
        assert alloc is not None
        assert alloc.allocation_percentage == 100.0
        assert alloc.allocated_amount == 15000.00
        assert alloc.allocated_shares == 100

    def test_process_split_allocation(
        self, app_context, db_session, sample_portfolio, second_portfolio, sample_ibkr_transaction
    ):
        """Test processing transaction split across two portfolios."""
        allocations = [
            {"portfolio_id": sample_portfolio.id, "percentage": 60.0},
            {"portfolio_id": second_portfolio.id, "percentage": 40.0},
        ]

        result = IBKRTransactionService.process_transaction_allocation(
            sample_ibkr_transaction.id, allocations
        )

        assert result["success"] is True
        assert len(result["created_transactions"]) == 2

        # Verify allocations
        allocs = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=sample_ibkr_transaction.id
        ).all()
        assert len(allocs) == 2

        # Check first allocation (60%)
        alloc1 = next(a for a in allocs if a.portfolio_id == sample_portfolio.id)
        assert alloc1.allocation_percentage == 60.0
        assert alloc1.allocated_amount == 9000.00  # 15000 * 0.6
        assert alloc1.allocated_shares == 60  # 100 * 0.6

        # Check second allocation (40%)
        alloc2 = next(a for a in allocs if a.portfolio_id == second_portfolio.id)
        assert alloc2.allocation_percentage == 40.0
        assert alloc2.allocated_amount == 6000.00  # 15000 * 0.4
        assert alloc2.allocated_shares == 40  # 100 * 0.4

    def test_process_creates_portfolio_fund_relationship(
        self, app_context, db_session, sample_portfolio, sample_ibkr_transaction
    ):
        """Test that processing creates portfolio-fund relationship."""
        initial_count = PortfolioFund.query.filter_by(portfolio_id=sample_portfolio.id).count()
        assert initial_count == 0

        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(
            sample_ibkr_transaction.id, allocations
        )

        assert result["success"] is True

        # Verify portfolio-fund created for this portfolio
        final_count = PortfolioFund.query.filter_by(portfolio_id=sample_portfolio.id).count()
        assert final_count == 1

    def test_process_creates_transaction_record(
        self, app_context, db_session, sample_portfolio, sample_ibkr_transaction
    ):
        """Test that processing creates Transaction record."""
        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(
            sample_ibkr_transaction.id, allocations
        )

        assert result["success"] is True

        # Verify transaction created
        txn = Transaction.query.first()
        assert txn is not None
        assert txn.type == "buy"
        assert txn.shares == 100
        assert txn.cost_per_share == 150.00

    def test_process_already_processed_transaction(
        self, app_context, db_session, sample_portfolio, sample_ibkr_transaction
    ):
        """Test that processing already-processed transaction fails."""
        # Mark as already processed
        sample_ibkr_transaction.status = "processed"
        db.session.commit()

        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(
            sample_ibkr_transaction.id, allocations
        )

        assert result["success"] is False
        assert "already processed" in result["error"].lower()

    def test_process_nonexistent_transaction(self, app_context, db_session):
        """Test processing nonexistent transaction."""
        fake_id = make_id()
        allocations = [{"portfolio_id": make_id(), "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(fake_id, allocations)

        assert result["success"] is False
        assert "not found" in result["error"].lower()

    def test_process_invalid_allocations(
        self, app_context, db_session, sample_portfolio, sample_ibkr_transaction
    ):
        """Test processing with invalid allocations."""
        # Allocations don't sum to 100%
        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 50.0}]

        result = IBKRTransactionService.process_transaction_allocation(
            sample_ibkr_transaction.id, allocations
        )

        assert result["success"] is False
        assert "50" in result["error"]  # Shows current percentage

    def test_process_creates_fund_if_not_exists(self, app_context, db_session, sample_portfolio):
        """Test that processing creates fund if it doesn't exist."""
        # Create a unique IBKR transaction with fund that doesn't exist yet
        unique_isin = make_isin("US")
        unique_symbol = make_symbol("TEST")

        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=unique_symbol,
            isin=unique_isin,
            description="TEST COMPANY",
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            fees=1.50,
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Ensure no fund exists with this specific symbol/ISIN
        assert Fund.query.filter_by(isin=unique_isin).first() is None

        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        assert result["success"] is True

        # Verify fund was created with correct ISIN and symbol from transaction
        fund = Fund.query.filter_by(isin=unique_isin).first()
        assert fund is not None
        assert fund.symbol == unique_symbol


class TestDividendMatching:
    """Tests for dividend matching functionality."""

    def test_get_pending_dividends_no_filter(
        self, app_context, db_session, sample_fund, sample_portfolio
    ):
        """Test getting all pending dividends."""
        # Create portfolio-fund relationship
        pf = PortfolioFund(
            id=make_id(),
            portfolio_id=sample_portfolio.id,
            fund_id=sample_fund.id,
        )
        db.session.add(pf)
        db.session.commit()  # Commit so FK constraint is satisfied

        # Create pending dividend
        div = Dividend(
            id=make_id(),
            portfolio_fund_id=pf.id,
            fund_id=sample_fund.id,
            record_date=date(2025, 1, 15),
            ex_dividend_date=date(2025, 1, 10),
            shares_owned=100,
            dividend_per_share=2.50,
            total_amount=250.00,
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        db.session.add(div)
        db.session.commit()

        pending = IBKRTransactionService.get_pending_dividends()

        assert len(pending) == 1
        assert pending[0]["id"] == div.id
        assert pending[0]["shares_owned"] == 100

    def test_get_pending_dividends_filter_by_symbol(
        self, app_context, db_session, sample_fund, sample_portfolio
    ):
        """Test filtering pending dividends by symbol."""
        # Setup
        pf = PortfolioFund(
            id=make_id(),
            portfolio_id=sample_portfolio.id,
            fund_id=sample_fund.id,
        )
        db.session.add(pf)
        db.session.commit()

        div = Dividend(
            id=make_id(),
            portfolio_fund_id=pf.id,
            fund_id=sample_fund.id,
            record_date=date(2025, 1, 15),
            ex_dividend_date=date(2025, 1, 10),
            shares_owned=100,
            dividend_per_share=2.50,
            total_amount=250.00,
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        db.session.add(div)
        db.session.commit()

        # Filter by the actual symbol from sample_fund
        pending = IBKRTransactionService.get_pending_dividends(symbol=sample_fund.symbol)

        assert len(pending) == 1

        # Filter by different symbol should find nothing
        pending = IBKRTransactionService.get_pending_dividends(symbol="NONEXISTENT_SYMBOL")

        assert len(pending) == 0

    def test_match_dividend_single(self, app_context, db_session, sample_fund, sample_portfolio):
        """Test matching IBKR dividend to single existing dividend."""
        # Create pending dividend
        pf = PortfolioFund(
            id=make_id(),
            portfolio_id=sample_portfolio.id,
            fund_id=sample_fund.id,
        )
        db.session.add(pf)
        db.session.commit()

        div = Dividend(
            id=make_id(),
            portfolio_fund_id=pf.id,
            fund_id=sample_fund.id,
            record_date=date(2025, 1, 15),
            ex_dividend_date=date(2025, 1, 10),
            shares_owned=100,
            dividend_per_share=2.50,
            total_amount=0,  # Not set yet
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        db.session.add(div)

        # Create IBKR dividend transaction using sample_fund's actual symbol/ISIN
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_dividend_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=sample_fund.symbol,
            isin=sample_fund.isin,
            description="Dividend",
            transaction_type="dividend",
            quantity=None,
            price=None,
            total_amount=250.00,  # Total dividend amount from IBKR
            currency="USD",
            fees=0,
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Match dividend
        result = IBKRTransactionService.match_dividend(ibkr_txn.id, [div.id])

        assert result["success"] is True
        assert result["updated_dividends"] == 1

        # Verify dividend updated with amount
        updated_div = Dividend.query.get(div.id)
        assert updated_div.total_amount == 250.00

        # Verify IBKR transaction marked as processed
        updated_ibkr = IBKRTransaction.query.get(ibkr_txn.id)
        assert updated_ibkr.status == "processed"

    def test_match_dividend_multiple_portfolios(
        self, app_context, db_session, sample_fund, sample_portfolio, second_portfolio
    ):
        """Test matching dividend split across multiple portfolios."""
        # Create portfolio-fund relationships
        pf1 = PortfolioFund(
            id=make_id(),
            portfolio_id=sample_portfolio.id,
            fund_id=sample_fund.id,
        )
        pf2 = PortfolioFund(
            id=make_id(),
            portfolio_id=second_portfolio.id,
            fund_id=sample_fund.id,
        )
        db.session.add_all([pf1, pf2])
        db.session.commit()

        # Create dividends (60 shares in portfolio1, 40 shares in portfolio2)
        div1 = Dividend(
            id=make_id(),
            portfolio_fund_id=pf1.id,
            fund_id=sample_fund.id,
            record_date=date(2025, 1, 15),
            ex_dividend_date=date(2025, 1, 10),
            shares_owned=60,
            dividend_per_share=2.50,
            total_amount=0,
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        div2 = Dividend(
            id=make_id(),
            portfolio_fund_id=pf2.id,
            fund_id=sample_fund.id,
            record_date=date(2025, 1, 15),
            ex_dividend_date=date(2025, 1, 10),
            shares_owned=40,
            dividend_per_share=2.50,
            total_amount=0,
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        db.session.add_all([div1, div2])

        # Create IBKR dividend (total for 100 shares) using sample_fund's symbol/ISIN
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_dividend_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=sample_fund.symbol,
            isin=sample_fund.isin,
            transaction_type="dividend",
            total_amount=250.00,  # $2.50 * 100 shares
            currency="USD",
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Match both dividends
        result = IBKRTransactionService.match_dividend(ibkr_txn.id, [div1.id, div2.id])

        assert result["success"] is True
        assert result["updated_dividends"] == 2

        # Verify amounts allocated proportionally
        updated_div1 = Dividend.query.get(div1.id)
        updated_div2 = Dividend.query.get(div2.id)

        assert updated_div1.total_amount == 150.00  # 250 * (60/100)
        assert updated_div2.total_amount == 100.00  # 250 * (40/100)

    def test_match_dividend_non_dividend_transaction(
        self, app_context, db_session, sample_ibkr_transaction
    ):
        """Test that matching non-dividend transaction fails."""
        div_id = make_id()

        result = IBKRTransactionService.match_dividend(sample_ibkr_transaction.id, [div_id])

        assert result["success"] is False
        assert "not a dividend" in result["error"].lower()

    def test_match_dividend_already_processed(self, app_context, db_session):
        """Test that matching already-processed dividend fails."""
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_dividend_txn_id(),
            transaction_date=date(2025, 1, 15),
            transaction_type="dividend",
            total_amount=250.00,
            currency="USD",
            status="processed",  # Already processed
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        result = IBKRTransactionService.match_dividend(ibkr_txn.id, [make_id()])

        assert result["success"] is False
        assert "already processed" in result["error"].lower()


class TestModifyAllocations:
    """Tests for modifying existing transaction allocations."""

    def test_modify_allocations_change_percentages(
        self, app_context, db_session, sample_portfolio, second_portfolio, sample_fund
    ):
        """Test modifying allocation percentages for existing portfolios."""
        # Create and process initial transaction with 60/40 split
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=sample_fund.symbol,
            isin=sample_fund.isin,
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Initial allocation: 60/40
        initial_allocs = [
            {"portfolio_id": sample_portfolio.id, "percentage": 60.0},
            {"portfolio_id": second_portfolio.id, "percentage": 40.0},
        ]
        IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, initial_allocs)

        # Modify to 50/50
        new_allocs = [
            {"portfolio_id": sample_portfolio.id, "percentage": 50.0},
            {"portfolio_id": second_portfolio.id, "percentage": 50.0},
        ]
        result = IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocs)

        assert result["success"] is True

        # Verify allocations updated
        allocs = IBKRTransactionAllocation.query.filter_by(ibkr_transaction_id=ibkr_txn.id).all()
        assert len(allocs) == 2

        alloc1 = next(a for a in allocs if a.portfolio_id == sample_portfolio.id)
        assert alloc1.allocation_percentage == 50.0
        assert alloc1.allocated_amount == 7500.00  # 15000 * 0.5
        assert alloc1.allocated_shares == 50  # 100 * 0.5

    def test_modify_allocations_add_portfolio(
        self, app_context, db_session, sample_portfolio, second_portfolio
    ):
        """Test adding a new portfolio to existing allocations."""
        # Create third portfolio
        third_portfolio = Portfolio(id=make_id(), name="Third Portfolio")
        db.session.add(third_portfolio)
        db.session.commit()

        # Create unique fund and transaction
        unique_isin = make_isin("US")
        unique_symbol = make_symbol("TEST")
        fund = Fund(
            id=make_id(),
            name="Test Fund",
            isin=unique_isin,
            symbol=unique_symbol,
            currency="USD",
            exchange="NYSE",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)

        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=unique_symbol,
            isin=unique_isin,
            transaction_type="buy",
            quantity=150,
            price=100.00,
            total_amount=15000.00,
            currency="USD",
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Initial: 100% to first portfolio
        initial_allocs = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]
        IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, initial_allocs)

        # Modify: split across three portfolios
        new_allocs = [
            {"portfolio_id": sample_portfolio.id, "percentage": 40.0},
            {"portfolio_id": second_portfolio.id, "percentage": 30.0},
            {"portfolio_id": third_portfolio.id, "percentage": 30.0},
        ]
        result = IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocs)

        assert result["success"] is True

        # Verify 3 allocations now exist
        allocs = IBKRTransactionAllocation.query.filter_by(ibkr_transaction_id=ibkr_txn.id).all()
        assert len(allocs) == 3

    def test_modify_allocations_remove_portfolio(
        self, app_context, db_session, sample_portfolio, second_portfolio, sample_fund
    ):
        """Test removing a portfolio from allocations."""
        # Create transaction with 50/50 split
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=sample_fund.symbol,
            isin=sample_fund.isin,
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Initial: 50/50
        initial_allocs = [
            {"portfolio_id": sample_portfolio.id, "percentage": 50.0},
            {"portfolio_id": second_portfolio.id, "percentage": 50.0},
        ]
        IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, initial_allocs)

        # Get transaction IDs before modification
        alloc2_before = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=ibkr_txn.id, portfolio_id=second_portfolio.id
        ).first()
        txn2_id = alloc2_before.transaction_id

        # Modify: 100% to first portfolio only
        new_allocs = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]
        result = IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocs)

        assert result["success"] is True

        # Verify only 1 allocation remains
        allocs = IBKRTransactionAllocation.query.filter_by(ibkr_transaction_id=ibkr_txn.id).all()
        assert len(allocs) == 1
        assert allocs[0].portfolio_id == sample_portfolio.id

        # Verify second portfolio's transaction was deleted
        assert Transaction.query.get(txn2_id) is None

    def test_modify_allocations_not_processed(self, app_context, db_session, sample_fund):
        """Test that modifying unprocessed transaction fails."""
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=sample_fund.symbol,
            isin=sample_fund.isin,
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            status="pending",  # Not processed
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        new_allocs = [{"portfolio_id": make_id(), "percentage": 100.0}]

        with pytest.raises(ValueError, match="not processed"):
            IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocs)

    def test_modify_allocations_invalid_percentages(
        self, app_context, db_session, sample_portfolio, sample_fund
    ):
        """Test that invalid allocations fail validation."""
        # Create and process transaction
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol=sample_fund.symbol,
            isin=sample_fund.isin,
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        initial_allocs = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]
        IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, initial_allocs)

        # Try to modify with invalid allocation (only 50%)
        invalid_allocs = [{"portfolio_id": sample_portfolio.id, "percentage": 50.0}]

        with pytest.raises(ValueError, match="50"):
            IBKRTransactionService.modify_allocations(ibkr_txn.id, invalid_allocs)

    def test_modify_allocations_not_found(self, app_context, db_session):
        """Test modifying nonexistent transaction."""
        fake_id = make_id()
        allocs = [{"portfolio_id": make_id(), "percentage": 100.0}]

        with pytest.raises(ValueError, match="not found"):
            IBKRTransactionService.modify_allocations(fake_id, allocs)
