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
        # Should create 2 transactions: 1 buy + 1 fee (commission=$1.50)
        assert len(result["created_transactions"]) == 2

        # Verify main transaction
        buy_txn = next(t for t in result["created_transactions"] if t.get("type") != "fee")
        assert buy_txn["shares"] == 100.0
        assert buy_txn["amount"] == 15000.00

        # Verify fee transaction
        fee_txn = next(t for t in result["created_transactions"] if t.get("type") == "fee")
        assert fee_txn["shares"] == 0
        assert fee_txn["amount"] == 1.50

        # Verify IBKR transaction marked as processed
        ibkr_txn = db.session.get(IBKRTransaction, sample_ibkr_transaction.id)
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
        # Should create 4 transactions: 2 buy + 2 fee (commission split 60/40)
        assert len(result["created_transactions"]) == 4

        # Filter transactions by type
        buy_txns = [t for t in result["created_transactions"] if t.get("type") != "fee"]
        fee_txns = [t for t in result["created_transactions"] if t.get("type") == "fee"]

        # Verify 2 buy transactions
        assert len(buy_txns) == 2
        # Verify 2 fee transactions
        assert len(fee_txns) == 2

        # Check commission allocation (total fee = $1.50)
        portfolio_a_fee = next(f for f in fee_txns if f["portfolio_name"] == "Test Portfolio")
        assert portfolio_a_fee["amount"] == 0.90  # 60% of $1.50

        portfolio_b_fee = next(f for f in fee_txns if f["portfolio_name"] == "Second Portfolio")
        assert portfolio_b_fee["amount"] == 0.60  # 40% of $1.50

        # Verify allocations (should be 4 total: 2 main + 2 fee)
        all_allocs = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=sample_ibkr_transaction.id
        ).all()
        assert len(all_allocs) == 4

        # Filter to main transaction allocations (those with shares > 0)
        main_allocs = [a for a in all_allocs if a.allocated_shares > 0]
        assert len(main_allocs) == 2

        # Check first allocation (60%)
        alloc1 = next(a for a in main_allocs if a.portfolio_id == sample_portfolio.id)
        assert alloc1.allocation_percentage == 60.0
        assert alloc1.allocated_amount == 9000.00  # 15000 * 0.6
        assert alloc1.allocated_shares == 60  # 100 * 0.6

        # Check second allocation (40%)
        alloc2 = next(a for a in main_allocs if a.portfolio_id == second_portfolio.id)
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
        updated_div = db.session.get(Dividend, div.id)
        assert updated_div.total_amount == 250.00

        # Verify IBKR transaction marked as processed
        updated_ibkr = db.session.get(IBKRTransaction, ibkr_txn.id)
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
        updated_div1 = db.session.get(Dividend, div1.id)
        updated_div2 = db.session.get(Dividend, div2.id)

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
        assert db.session.get(Transaction, txn2_id) is None

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


class TestCommissionAllocation:
    """Tests for commission/fee allocation functionality."""

    def test_process_allocation_with_zero_commission(
        self, app_context, db_session, sample_portfolio
    ):
        """Test that no fee transaction is created when commission is zero."""
        # Create transaction with zero fees
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol="AAPL",
            isin="US0378331005",
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            fees=0,  # No commission
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        assert result["success"] is True
        # Should only create 1 transaction (buy), no fee transaction
        assert len(result["created_transactions"]) == 1

        # Main transaction doesn't have 'type' field in return dict
        # Verify it has shares and amount (not a fee)
        assert result["created_transactions"][0]["shares"] == 100
        assert result["created_transactions"][0]["amount"] == 15000.00

        # Verify no fee transactions in database
        fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(fee_txns) == 0

    def test_commission_allocated_proportionally(
        self, app_context, db_session, sample_portfolio, second_portfolio
    ):
        """Test that commission is split proportionally across portfolios."""
        # Create transaction with $3.00 commission for easy math
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol="AAPL",
            isin="US0378331005",
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            fees=3.00,
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # 60/40 split
        allocations = [
            {"portfolio_id": sample_portfolio.id, "percentage": 60.0},
            {"portfolio_id": second_portfolio.id, "percentage": 40.0},
        ]

        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        assert result["success"] is True

        # Get fee transactions
        fee_txns = [t for t in result["created_transactions"] if t.get("type") == "fee"]
        assert len(fee_txns) == 2

        # Check proportional allocation
        portfolio_a_fee = next(f for f in fee_txns if f["portfolio_name"] == "Test Portfolio")
        assert portfolio_a_fee["amount"] == 1.80  # 60% of $3.00

        portfolio_b_fee = next(f for f in fee_txns if f["portfolio_name"] == "Second Portfolio")
        assert portfolio_b_fee["amount"] == 1.20  # 40% of $3.00

    def test_commission_rounding_fractional_cents(
        self, app_context, db_session, sample_portfolio, second_portfolio
    ):
        """Test commission allocation with fractional cents (rounding)."""
        # Create third portfolio for 3-way split
        third_portfolio = Portfolio(id=make_id(), name="Third Portfolio")
        db.session.add(third_portfolio)
        db.session.commit()

        # $2.00 commission split 3 ways = $0.6666... each
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol="AAPL",
            isin="US0378331005",
            transaction_type="buy",
            quantity=150,
            price=100.00,
            total_amount=15000.00,
            currency="USD",
            fees=2.00,
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Equal 3-way split
        allocations = [
            {"portfolio_id": sample_portfolio.id, "percentage": 33.33},
            {"portfolio_id": second_portfolio.id, "percentage": 33.33},
            {"portfolio_id": third_portfolio.id, "percentage": 33.34},
        ]

        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        assert result["success"] is True

        # Get fee transactions
        fee_txns = [t for t in result["created_transactions"] if t.get("type") == "fee"]
        assert len(fee_txns) == 3

        # Check each allocation
        fee1 = next(f for f in fee_txns if f["portfolio_name"] == "Test Portfolio")
        assert abs(fee1["amount"] - 0.6666) < 0.0001  # 33.33% of $2.00

        fee2 = next(f for f in fee_txns if f["portfolio_name"] == "Second Portfolio")
        assert abs(fee2["amount"] - 0.6666) < 0.0001  # 33.33% of $2.00

        fee3 = next(f for f in fee_txns if f["portfolio_name"] == "Third Portfolio")
        assert abs(fee3["amount"] - 0.6668) < 0.0001  # 33.34% of $2.00

    def test_modify_allocations_updates_fee_transactions(
        self, app_context, db_session, sample_portfolio, second_portfolio, sample_fund
    ):
        """Test that modifying allocations updates fee transactions correctly."""
        # Create transaction with commission
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
            fees=3.00,
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

        # Verify initial fee allocation
        initial_fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(initial_fee_txns) == 2

        # Modify to 50/50
        new_allocs = [
            {"portfolio_id": sample_portfolio.id, "percentage": 50.0},
            {"portfolio_id": second_portfolio.id, "percentage": 50.0},
        ]
        result = IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocs)

        assert result["success"] is True

        # Verify fee transactions updated
        updated_fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(updated_fee_txns) == 2

        # Check updated amounts (should now be $1.50 each for 50/50 split)
        for fee_txn in updated_fee_txns:
            assert fee_txn.cost_per_share == 1.50  # 50% of $3.00

    def test_modify_allocations_removes_fee_transactions(
        self, app_context, db_session, sample_portfolio, second_portfolio, sample_fund
    ):
        """Test that removing portfolio allocation also removes its fee transaction."""
        # Create transaction with commission
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
            fees=3.00,
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Initial: 50/50 split
        initial_allocs = [
            {"portfolio_id": sample_portfolio.id, "percentage": 50.0},
            {"portfolio_id": second_portfolio.id, "percentage": 50.0},
        ]
        IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, initial_allocs)

        # Verify 2 fee transactions created
        initial_fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(initial_fee_txns) == 2

        # Modify to 100% on first portfolio only
        new_allocs = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]
        result = IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocs)

        assert result["success"] is True

        # Verify only 1 fee transaction remains
        remaining_fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(remaining_fee_txns) == 1
        assert remaining_fee_txns[0].cost_per_share == 3.00  # 100% of commission

    def test_modify_allocations_adds_fee_transactions(
        self, app_context, db_session, sample_portfolio, second_portfolio, sample_fund
    ):
        """Test that adding new portfolio allocation creates new fee transaction."""
        # Create transaction with commission
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
            fees=3.00,
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Initial: 100% to first portfolio
        initial_allocs = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]
        IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, initial_allocs)

        # Verify 1 fee transaction created
        initial_fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(initial_fee_txns) == 1
        assert initial_fee_txns[0].cost_per_share == 3.00

        # Modify to 60/40 split (adding second portfolio)
        new_allocs = [
            {"portfolio_id": sample_portfolio.id, "percentage": 60.0},
            {"portfolio_id": second_portfolio.id, "percentage": 40.0},
        ]
        result = IBKRTransactionService.modify_allocations(ibkr_txn.id, new_allocs)

        assert result["success"] is True

        # Verify 2 fee transactions now exist
        updated_fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(updated_fee_txns) == 2

        # Check amounts
        fee_amounts = sorted([f.cost_per_share for f in updated_fee_txns])
        assert fee_amounts == [1.20, 1.80]  # 40% and 60% of $3.00

    def test_fee_transaction_has_correct_structure(self, app_context, db_session, sample_portfolio):
        """Test that fee transactions have the correct field values."""
        # Create transaction with commission
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol="AAPL",
            isin="US0378331005",
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

        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        assert result["success"] is True

        # Get the fee transaction from database
        fee_txn = Transaction.query.filter_by(type="fee").first()
        assert fee_txn is not None

        # Verify structure
        assert fee_txn.type == "fee"
        assert fee_txn.shares == 0  # Fee transactions have no shares
        assert fee_txn.cost_per_share == 1.50  # Fee amount stored here
        assert fee_txn.date == date(2025, 1, 15)  # Same date as IBKR transaction
        assert fee_txn.portfolio_fund_id is not None  # Linked to portfolio-fund

    def test_fee_transaction_linked_to_ibkr(self, app_context, db_session, sample_portfolio):
        """Test that fee transactions are linked to IBKR via IBKRTransactionAllocation."""
        # Create transaction with commission
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol="AAPL",
            isin="US0378331005",
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

        allocations = [{"portfolio_id": sample_portfolio.id, "percentage": 100.0}]

        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        assert result["success"] is True

        # Get the fee transaction
        fee_txn = Transaction.query.filter_by(type="fee").first()
        assert fee_txn is not None

        # Verify fee transaction is linked to IBKR via allocation
        fee_allocation = IBKRTransactionAllocation.query.filter_by(
            transaction_id=fee_txn.id
        ).first()
        assert fee_allocation is not None
        assert fee_allocation.ibkr_transaction_id == ibkr_txn.id
        assert fee_allocation.portfolio_id == sample_portfolio.id
        assert fee_allocation.allocated_amount == 1.50
        assert fee_allocation.allocated_shares == 0

    def test_fee_transaction_linked_to_ibkr_split_allocation(
        self, app_context, db_session, sample_portfolio, second_portfolio
    ):
        """Test that fee transactions are linked to IBKR in split allocations."""
        # Create transaction with commission
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2025, 1, 15),
            symbol="AAPL",
            isin="US0378331005",
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=15000.00,
            currency="USD",
            fees=3.00,  # $3 commission
            status="pending",
            raw_data="{}",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # 60/40 split
        allocations = [
            {"portfolio_id": sample_portfolio.id, "percentage": 60.0},
            {"portfolio_id": second_portfolio.id, "percentage": 40.0},
        ]

        result = IBKRTransactionService.process_transaction_allocation(ibkr_txn.id, allocations)

        assert result["success"] is True

        # Get all fee transactions
        fee_txns = Transaction.query.filter_by(type="fee").all()
        assert len(fee_txns) == 2

        # Verify both fee transactions are linked to IBKR
        for fee_txn in fee_txns:
            fee_allocation = IBKRTransactionAllocation.query.filter_by(
                transaction_id=fee_txn.id
            ).first()
            assert fee_allocation is not None
            assert fee_allocation.ibkr_transaction_id == ibkr_txn.id

        # Verify allocation amounts
        portfolio_a_fee_txn = (
            Transaction.query.join(PortfolioFund)
            .filter(
                Transaction.type == "fee",
                PortfolioFund.portfolio_id == sample_portfolio.id,
            )
            .first()
        )
        assert portfolio_a_fee_txn is not None
        assert portfolio_a_fee_txn.cost_per_share == 1.80  # 60% of $3.00

        portfolio_b_fee_txn = (
            Transaction.query.join(PortfolioFund)
            .filter(
                Transaction.type == "fee",
                PortfolioFund.portfolio_id == second_portfolio.id,
            )
            .first()
        )
        assert portfolio_b_fee_txn is not None
        assert portfolio_b_fee_txn.cost_per_share == 1.20  # 40% of $3.00


class TestTransactionManagement:
    """Tests for transaction management methods (get, ignore, delete)."""

    def test_get_transaction_success(self, app_context, db_session, sample_ibkr_transaction):
        """Test get_transaction retrieves existing transaction."""
        txn = IBKRTransactionService.get_transaction(sample_ibkr_transaction.id)

        assert txn is not None
        assert txn.id == sample_ibkr_transaction.id
        assert txn.symbol == sample_ibkr_transaction.symbol

    def test_get_transaction_not_found(self, app_context, db_session):
        """Test get_transaction raises 404 for non-existent transaction."""
        fake_id = make_id()

        # Flask abort() raises werkzeug HTTPException
        from werkzeug.exceptions import NotFound

        with pytest.raises(NotFound):
            IBKRTransactionService.get_transaction(fake_id)

    def test_ignore_transaction_success(self, app_context, db_session, sample_ibkr_transaction):
        """Test ignore_transaction marks transaction as ignored."""
        response, status = IBKRTransactionService.ignore_transaction(sample_ibkr_transaction.id)

        assert status == 200
        assert response["success"] is True
        assert "ignored" in response["message"]

        # Verify transaction status updated
        db_session.refresh(sample_ibkr_transaction)
        assert sample_ibkr_transaction.status == "ignored"
        assert sample_ibkr_transaction.processed_at is not None

    def test_ignore_transaction_already_processed(
        self, app_context, db_session, sample_ibkr_transaction
    ):
        """Test ignore_transaction rejects already-processed transaction."""
        # Mark as processed
        sample_ibkr_transaction.status = "processed"
        db_session.commit()

        response, status = IBKRTransactionService.ignore_transaction(sample_ibkr_transaction.id)

        assert status == 400
        assert "error" in response
        assert "processed" in response["error"]

        # Status should remain processed
        db_session.refresh(sample_ibkr_transaction)
        assert sample_ibkr_transaction.status == "processed"

    def test_ignore_transaction_not_found(self, app_context, db_session):
        """Test ignore_transaction handles non-existent transaction."""
        fake_id = make_id()

        from werkzeug.exceptions import NotFound

        with pytest.raises(NotFound):
            IBKRTransactionService.ignore_transaction(fake_id)

    def test_delete_transaction_success(self, app_context, db_session, sample_ibkr_transaction):
        """Test delete_transaction removes transaction."""
        transaction_id = sample_ibkr_transaction.id

        response, status = IBKRTransactionService.delete_transaction(transaction_id)

        assert status == 200
        assert response["success"] is True
        assert "deleted" in response["message"]

        # Verify transaction deleted
        deleted = db.session.get(IBKRTransaction, transaction_id)
        assert deleted is None

    def test_delete_transaction_already_processed(
        self, app_context, db_session, sample_ibkr_transaction
    ):
        """Test delete_transaction rejects already-processed transaction."""
        # Mark as processed
        sample_ibkr_transaction.status = "processed"
        db_session.commit()

        response, status = IBKRTransactionService.delete_transaction(sample_ibkr_transaction.id)

        assert status == 400
        assert "error" in response
        assert "processed" in response["error"]

        # Transaction should still exist
        txn = db.session.get(IBKRTransaction, sample_ibkr_transaction.id)
        assert txn is not None

    def test_delete_transaction_not_found(self, app_context, db_session):
        """Test delete_transaction handles non-existent transaction."""
        fake_id = make_id()

        from werkzeug.exceptions import NotFound

        with pytest.raises(NotFound):
            IBKRTransactionService.delete_transaction(fake_id)


class TestGetInbox:
    """Tests for get_inbox() - Retrieve IBKR inbox transactions with filtering."""

    def test_get_inbox_default_pending(self, app_context, db_session):
        """Test get_inbox returns pending transactions by default."""
        # Create transactions with different statuses
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 1),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="pending",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 2),
            symbol="GOOGL",
            description="Google",
            transaction_type="buy",
            quantity=5,
            price=2000.0,
            total_amount=10000.0,
            currency="USD",
            status="processed",
        )
        txn3 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 3),
            symbol="MSFT",
            description="Microsoft",
            transaction_type="buy",
            quantity=20,
            price=300.0,
            total_amount=6000.0,
            currency="USD",
            status="pending",
        )
        db.session.add_all([txn1, txn2, txn3])
        db.session.commit()

        result = IBKRTransactionService.get_inbox()

        assert len(result) == 2
        assert all(txn["status"] == "pending" for txn in result)
        # Should be ordered by transaction_date descending
        assert result[0]["symbol"] == "MSFT"  # Jan 3
        assert result[1]["symbol"] == "AAPL"  # Jan 1

    def test_get_inbox_filter_by_status(self, app_context, db_session):
        """Test get_inbox filters by status."""
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 1),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="ignored",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 2),
            symbol="GOOGL",
            description="Google",
            transaction_type="buy",
            quantity=5,
            price=2000.0,
            total_amount=10000.0,
            currency="USD",
            status="pending",
        )
        db.session.add_all([txn1, txn2])
        db.session.commit()

        result = IBKRTransactionService.get_inbox(status="ignored")

        assert len(result) == 1
        assert result[0]["symbol"] == "AAPL"
        assert result[0]["status"] == "ignored"

    def test_get_inbox_filter_by_transaction_type(self, app_context, db_session):
        """Test get_inbox filters by transaction type."""
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 1),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="pending",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 2),
            symbol="AAPL",
            description="Apple Dividend",
            transaction_type="dividend",
            quantity=0,
            price=0.0,
            total_amount=50.0,
            currency="USD",
            status="pending",
        )
        db.session.add_all([txn1, txn2])
        db.session.commit()

        result = IBKRTransactionService.get_inbox(transaction_type="dividend")

        assert len(result) == 1
        assert result[0]["transaction_type"] == "dividend"

    def test_get_inbox_empty(self, app_context, db_session):
        """Test get_inbox returns empty list when no transactions match."""
        result = IBKRTransactionService.get_inbox(status="pending")

        assert result == []

    def test_get_inbox_response_format(self, app_context, db_session):
        """Test get_inbox returns correctly formatted transaction data."""
        txn = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 15),
            symbol="AAPL",
            isin="US0378331005",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=150.50,
            total_amount=1505.0,
            currency="USD",
            fees=5.0,
            status="pending",
        )
        db.session.add(txn)
        db.session.commit()

        result = IBKRTransactionService.get_inbox()

        assert len(result) == 1
        txn_data = result[0]
        assert txn_data["id"] == txn.id
        assert txn_data["ibkr_transaction_id"] == txn.ibkr_transaction_id
        assert txn_data["transaction_date"] == "2024-01-15"
        assert txn_data["symbol"] == "AAPL"
        assert txn_data["isin"] == "US0378331005"
        assert txn_data["description"] == "Apple Inc"
        assert txn_data["transaction_type"] == "buy"
        assert txn_data["quantity"] == 10
        assert txn_data["price"] == 150.50
        assert txn_data["total_amount"] == 1505.0
        assert txn_data["currency"] == "USD"
        assert txn_data["fees"] == 5.0
        assert txn_data["status"] == "pending"
        assert "imported_at" in txn_data


class TestGetInboxCount:
    """Tests for get_inbox_count() - Count IBKR transactions by status."""

    def test_get_inbox_count_default_pending(self, app_context, db_session):
        """Test get_inbox_count counts pending transactions by default."""
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 1),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="pending",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 2),
            symbol="GOOGL",
            description="Google",
            transaction_type="buy",
            quantity=5,
            price=2000.0,
            total_amount=10000.0,
            currency="USD",
            status="processed",
        )
        txn3 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 3),
            symbol="MSFT",
            description="Microsoft",
            transaction_type="buy",
            quantity=20,
            price=300.0,
            total_amount=6000.0,
            currency="USD",
            status="pending",
        )
        db.session.add_all([txn1, txn2, txn3])
        db.session.commit()

        count = IBKRTransactionService.get_inbox_count()

        assert count == 2

    def test_get_inbox_count_filter_by_status(self, app_context, db_session):
        """Test get_inbox_count filters by status."""
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 1),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="ignored",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 2),
            symbol="GOOGL",
            description="Google",
            transaction_type="buy",
            quantity=5,
            price=2000.0,
            total_amount=10000.0,
            currency="USD",
            status="pending",
        )
        db.session.add_all([txn1, txn2])
        db.session.commit()

        count = IBKRTransactionService.get_inbox_count(status="ignored")

        assert count == 1

    def test_get_inbox_count_zero(self, app_context, db_session):
        """Test get_inbox_count returns 0 when no transactions match."""
        count = IBKRTransactionService.get_inbox_count(status="pending")

        assert count == 0


class TestUnallocateTransaction:
    """Tests for unallocate_transaction() - Remove allocations and revert to pending."""

    def test_unallocate_transaction_with_transactions(
        self, app_context, db_session, sample_fund, sample_portfolio
    ):
        """Test unallocate_transaction removes allocations and transactions."""
        # Create IBKR transaction
        ibkr_txn = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 15),
            symbol=sample_fund.symbol,
            description="Test Transaction",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="processed",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Create portfolio fund
        pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=sample_fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transaction
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 15),
            type="buy",
            shares=10.0,
            cost_per_share=150.0,
        )
        db.session.add(txn)
        db.session.commit()

        # Create allocation
        allocation = IBKRTransactionAllocation(
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=sample_portfolio.id,
            allocation_percentage=100.0,
            allocated_amount=1500.0,
            allocated_shares=10.0,
            transaction_id=txn.id,
        )
        db.session.add(allocation)
        db.session.commit()

        response, status = IBKRTransactionService.unallocate_transaction(ibkr_txn.id)

        assert status == 200
        assert response["success"] is True
        assert "1 portfolio transactions deleted" in response["message"]

        # Verify IBKR transaction reverted to pending
        ibkr_txn = db.session.get(IBKRTransaction, ibkr_txn.id)
        assert ibkr_txn.status == "pending"
        assert ibkr_txn.processed_at is None

        # Verify allocations deleted
        allocations = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=ibkr_txn.id
        ).all()
        assert len(allocations) == 0

        # Verify transaction deleted
        txn = db.session.get(Transaction, txn.id)
        assert txn is None

    def test_unallocate_transaction_orphaned_allocations(
        self, app_context, db_session, sample_fund, sample_portfolio
    ):
        """Test unallocate_transaction handles orphaned allocations without transactions."""
        # Create IBKR transaction
        ibkr_txn = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 15),
            symbol=sample_fund.symbol,
            description="Test Transaction",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="processed",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Create allocation WITHOUT transaction_id (orphaned)
        allocation = IBKRTransactionAllocation(
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=sample_portfolio.id,
            allocation_percentage=100.0,
            allocated_amount=1500.0,
            allocated_shares=10.0,
            transaction_id=None,  # Orphaned
        )
        db.session.add(allocation)
        db.session.commit()

        response, status = IBKRTransactionService.unallocate_transaction(ibkr_txn.id)

        assert status == 200
        assert response["success"] is True

        # Verify IBKR transaction reverted to pending
        ibkr_txn = db.session.get(IBKRTransaction, ibkr_txn.id)
        assert ibkr_txn.status == "pending"

        # Verify orphaned allocation deleted
        allocations = IBKRTransactionAllocation.query.filter_by(
            ibkr_transaction_id=ibkr_txn.id
        ).all()
        assert len(allocations) == 0

    def test_unallocate_transaction_not_found(self, app_context, db_session):
        """Test unallocate_transaction handles non-existent transaction."""
        fake_id = make_id()

        response, status = IBKRTransactionService.unallocate_transaction(fake_id)

        assert status == 404
        assert "error" in response

    def test_unallocate_transaction_not_processed(self, app_context, db_session):
        """Test unallocate_transaction rejects non-processed transactions."""
        ibkr_txn = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 15),
            symbol="AAPL",
            description="Test Transaction",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="pending",  # Not processed
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        response, status = IBKRTransactionService.unallocate_transaction(ibkr_txn.id)

        assert status == 400
        assert "error" in response
        assert "not processed" in response["error"]


class TestGetTransactionAllocations:
    """Tests for get_transaction_allocations() - Retrieve allocation details."""

    def test_get_transaction_allocations(
        self, app_context, db_session, sample_fund, sample_portfolio
    ):
        """Test get_transaction_allocations returns allocation details."""
        # Create IBKR transaction
        ibkr_txn = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 15),
            symbol=sample_fund.symbol,
            description="Test Transaction",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="processed",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        # Create portfolio fund
        pf = PortfolioFund(portfolio_id=sample_portfolio.id, fund_id=sample_fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transaction
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 15),
            type="buy",
            shares=10.0,
            cost_per_share=150.0,
        )
        db.session.add(txn)
        db.session.commit()

        # Create allocation
        allocation = IBKRTransactionAllocation(
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=sample_portfolio.id,
            allocation_percentage=100.0,
            allocated_amount=1500.0,
            allocated_shares=10.0,
            transaction_id=txn.id,
        )
        db.session.add(allocation)
        db.session.commit()

        response, status = IBKRTransactionService.get_transaction_allocations(ibkr_txn.id)

        assert status == 200
        assert response["ibkr_transaction_id"] == ibkr_txn.id
        assert response["status"] == "processed"
        assert len(response["allocations"]) == 1

        # Grouped allocations don't include transaction_id/transaction_date
        # They combine stock and fee transactions per portfolio
        alloc_data = response["allocations"][0]
        assert alloc_data["portfolio_id"] == sample_portfolio.id
        assert alloc_data["portfolio_name"] == sample_portfolio.name
        assert alloc_data["allocation_percentage"] == 100.0
        assert alloc_data["allocated_amount"] == 1500.0
        assert alloc_data["allocated_shares"] == 10.0
        assert alloc_data["allocated_commission"] == 0.0  # No commission in this test

    def test_get_transaction_allocations_no_allocations(self, app_context, db_session):
        """Test get_transaction_allocations returns empty list when no allocations."""
        ibkr_txn = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 15),
            symbol="AAPL",
            description="Test Transaction",
            transaction_type="buy",
            quantity=10,
            price=150.0,
            total_amount=1500.0,
            currency="USD",
            status="pending",
        )
        db.session.add(ibkr_txn)
        db.session.commit()

        response, status = IBKRTransactionService.get_transaction_allocations(ibkr_txn.id)

        assert status == 200
        assert response["allocations"] == []

    def test_get_transaction_allocations_not_found(self, app_context, db_session):
        """Test get_transaction_allocations handles non-existent transaction."""
        fake_id = make_id()

        response, status = IBKRTransactionService.get_transaction_allocations(fake_id)

        assert status == 404
        assert "error" in response


class TestGroupedAllocations:
    """Tests for grouped allocation logic (combining stock and fee transactions)."""

    def test_get_grouped_allocations_combines_stock_and_commission(self, app_context, db_session):
        """
        Verify that get_grouped_allocations combines stock and fee transactions per portfolio.

        WHY: When an IBKR transaction is processed with commission, it creates TWO
        IBKRTransactionAllocation records per portfolio (one for stock, one for fee).
        The UI should show ONE card per portfolio with combined data, not two separate
        cards. This method groups allocations by portfolio_id and sums amounts/shares,
        separating commission into its own field for clear display.
        """
        # Create test data
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        fund = Fund(
            name="Apple Inc",
            isin="US0378331005",
            symbol="AAPL",
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db_session.add(fund)
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create IBKR transaction with commission
        ibkr_txn = IBKRTransaction(
            ibkr_transaction_id=make_ibkr_txn_id(),
            transaction_date=date(2024, 1, 15),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=100,
            price=150.0,
            total_amount=15000.0,
            currency="USD",
            fees=3.00,
            status="processed",
        )
        db_session.add(ibkr_txn)
        db_session.commit()

        # Create stock transaction
        stock_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 15),
            type="buy",
            shares=100.0,
            cost_per_share=150.0,
        )
        db_session.add(stock_txn)
        db_session.commit()

        # Create fee transaction
        fee_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 15),
            type="fee",
            shares=0.0,
            cost_per_share=3.00,
        )
        db_session.add(fee_txn)
        db_session.commit()

        # Create allocations for both
        stock_allocation = IBKRTransactionAllocation(
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=portfolio.id,
            allocation_percentage=100.0,
            allocated_amount=15000.0,
            allocated_shares=100.0,
            transaction_id=stock_txn.id,
        )
        fee_allocation = IBKRTransactionAllocation(
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=portfolio.id,
            allocation_percentage=100.0,
            allocated_amount=3.00,
            allocated_shares=0.0,
            transaction_id=fee_txn.id,
        )
        db_session.add_all([stock_allocation, fee_allocation])
        db_session.commit()

        # Get grouped allocations
        result = IBKRTransactionService.get_grouped_allocations(ibkr_txn.id)

        # Should return 1 grouped allocation (not 2!)
        assert len(result) == 1

        alloc = result[0]
        assert alloc["portfolio_id"] == portfolio.id
        assert alloc["portfolio_name"] == "Test Portfolio"
        assert alloc["allocation_percentage"] == 100.0
        assert alloc["allocated_amount"] == 15000.0  # Stock amount
        assert alloc["allocated_shares"] == 100.0  # Stock shares
        assert alloc["allocated_commission"] == 3.00  # Fee amount
