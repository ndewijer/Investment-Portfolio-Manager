"""
Unit tests for DividendService.

Tests cover:
- Share calculation with buy/sell scenarios
- CASH vs STOCK dividend creation
- Reinvestment handling
- Dividend updates
- Edge cases and error handling
"""

from datetime import date

import pytest
from app.models import DividendType, ReinvestmentStatus
from app.services.dividend_service import DividendService
from tests.factories import (
    FundFactory,
    PortfolioFactory,
    PortfolioFundFactory,
)
from tests.test_helpers import make_id


class TestCalculateSharesOwned:
    """Test share calculation on record dates."""

    def test_calculate_shares_buy_only(self, app_context, db_session):
        """Test share calculation with buy transactions only."""
        portfolio = PortfolioFactory()
        fund = FundFactory()
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create buy transactions directly

        from app.models import Transaction

        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=50,
            cost_per_share=10.0,
            date=date(2024, 2, 1),
        )
        db_session.add(txn1)
        db_session.add(txn2)
        db_session.commit()

        # Calculate shares on record date
        shares = DividendService.calculate_shares_owned(portfolio_fund.id, date(2024, 3, 1))

        assert shares == 150  # 100 + 50

    def test_calculate_shares_buy_and_sell(self, app_context, db_session):
        """Test share calculation with buy and sell transactions."""
        portfolio = PortfolioFactory()
        fund = FundFactory()
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create transactions directly

        from app.models import Transaction

        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="sell",
            shares=30,
            cost_per_share=10.0,
            date=date(2024, 2, 1),
        )
        txn3 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=50,
            cost_per_share=10.0,
            date=date(2024, 3, 1),
        )
        db_session.add(txn1)
        db_session.add(txn2)
        db_session.add(txn3)
        db_session.commit()

        # Calculate shares on record date
        shares = DividendService.calculate_shares_owned(portfolio_fund.id, date(2024, 4, 1))

        assert shares == 120  # 100 - 30 + 50

    def test_calculate_shares_before_first_transaction(self, app_context, db_session):
        """Test share calculation before any transactions."""
        portfolio = PortfolioFactory()
        fund = FundFactory()
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create transaction directly

        from app.models import Transaction

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 2, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Calculate shares before first transaction
        shares = DividendService.calculate_shares_owned(portfolio_fund.id, date(2024, 1, 1))

        assert shares == 0

    def test_calculate_shares_only_counts_up_to_record_date(self, app_context, db_session):
        """Test that only transactions up to record date are counted."""
        portfolio = PortfolioFactory()
        fund = FundFactory()
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create transactions directly

        from app.models import Transaction

        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        # This should not be counted
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=50,
            cost_per_share=10.0,
            date=date(2024, 3, 1),
        )
        db_session.add(txn1)
        db_session.add(txn2)
        db_session.commit()

        shares = DividendService.calculate_shares_owned(portfolio_fund.id, date(2024, 2, 1))

        assert shares == 100  # Only first transaction

    def test_calculate_shares_with_dividend_transactions(self, app_context, db_session):
        """Test that dividend transactions are included in share calculation."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create transactions directly

        from app.models import Transaction

        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        # Dividend reinvestment transaction
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="dividend",
            shares=5,
            cost_per_share=10.0,
            date=date(2024, 2, 1),
        )
        db_session.add(txn1)
        db_session.add(txn2)
        db_session.commit()

        shares = DividendService.calculate_shares_owned(portfolio_fund.id, date(2024, 3, 1))

        assert shares == 105  # 100 + 5


class TestCreateDividendCash:
    """Test CASH dividend creation."""

    def test_create_cash_dividend_auto_completed(self, app_context, db_session):
        """Test that CASH dividends are automatically marked as completed."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.CASH)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create buy transaction directly

        from app.models import Transaction

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Create cash dividend
        dividend_data = {
            "portfolio_fund_id": portfolio_fund.id,
            "record_date": "2024-02-15",
            "ex_dividend_date": "2024-02-10",
            "dividend_per_share": 0.50,
        }

        dividend = DividendService.create_dividend(dividend_data)

        assert dividend.shares_owned == 100
        assert dividend.dividend_per_share == 0.50
        assert dividend.total_amount == 50.0  # 100 * 0.50
        assert dividend.reinvestment_status == ReinvestmentStatus.COMPLETED
        assert dividend.reinvestment_transaction_id is None  # No transaction for CASH

    def test_create_cash_dividend_no_transaction_created(self, app_context, db_session):
        """Test that CASH dividends do not create transactions."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.CASH)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create buy transaction directly

        from app.models import Transaction

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Create cash dividend
        dividend_data = {
            "portfolio_fund_id": portfolio_fund.id,
            "record_date": "2024-02-15",
            "ex_dividend_date": "2024-02-10",
            "dividend_per_share": 0.50,
        }

        dividend = DividendService.create_dividend(dividend_data)

        assert dividend.reinvestment_transaction_id is None
        # Verify no transaction was created
        from app.models import Transaction

        txn = Transaction.query.filter_by(
            portfolio_fund_id=portfolio_fund.id, type="dividend"
        ).first()
        assert txn is None


class TestCreateDividendStock:
    """Test STOCK dividend creation."""

    def test_create_stock_dividend_without_reinvestment(self, app_context, db_session):
        """Test that STOCK dividends without reinvestment are pending."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create buy transaction directly

        from app.models import Transaction

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Create stock dividend without reinvestment
        dividend_data = {
            "portfolio_fund_id": portfolio_fund.id,
            "record_date": "2024-02-15",
            "ex_dividend_date": "2024-02-10",
            "dividend_per_share": 0.50,
        }

        dividend = DividendService.create_dividend(dividend_data)

        assert dividend.reinvestment_status == ReinvestmentStatus.PENDING
        assert dividend.reinvestment_transaction_id is None

    def test_create_stock_dividend_with_reinvestment(self, app_context, db_session):
        """Test STOCK dividend with immediate reinvestment."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create buy transaction directly

        from app.models import Transaction

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Create stock dividend with reinvestment
        dividend_data = {
            "portfolio_fund_id": portfolio_fund.id,
            "record_date": "2024-02-15",
            "ex_dividend_date": "2024-02-10",
            "dividend_per_share": 0.50,
            "buy_order_date": "2024-02-20",
            "reinvestment_shares": 2.5,
            "reinvestment_price": 20.0,
        }

        dividend = DividendService.create_dividend(dividend_data)

        assert dividend.reinvestment_status == ReinvestmentStatus.COMPLETED
        assert dividend.reinvestment_transaction_id is not None
        # buy_order_date is set to transaction date (ex_dividend_date),
        # not the provided buy_order_date
        assert dividend.buy_order_date == date(2024, 2, 10)

        # Verify transaction was created
        from app.models import Transaction

        txn = Transaction.query.get(dividend.reinvestment_transaction_id)
        assert txn is not None
        assert txn.type == "dividend"
        assert txn.shares == 2.5
        assert txn.cost_per_share == 20.0
        assert txn.date == date(2024, 2, 10)  # Uses ex_dividend_date

    def test_create_stock_dividend_reinvestment_validation(self, app_context, db_session):
        """Test validation of reinvestment data."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create buy transaction directly

        from app.models import Transaction

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Test with negative shares
        dividend_data = {
            "portfolio_fund_id": portfolio_fund.id,
            "record_date": "2024-02-15",
            "ex_dividend_date": "2024-02-10",
            "dividend_per_share": 0.50,
            "reinvestment_shares": -2.5,  # Invalid
            "reinvestment_price": 20.0,
        }

        with pytest.raises(ValueError, match="Reinvestment shares and price must be positive"):
            DividendService.create_dividend(dividend_data)

        # Test with zero price
        dividend_data["reinvestment_shares"] = 2.5
        dividend_data["reinvestment_price"] = 0  # Invalid

        with pytest.raises(ValueError, match="Reinvestment shares and price must be positive"):
            DividendService.create_dividend(dividend_data)


class TestUpdateDividend:
    """Test dividend update operations."""

    def test_update_dividend_basic_fields(self, app_context, db_session):
        """Test updating basic dividend fields."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.CASH)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)

        # Create dividend directly

        from app.models import Dividend

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=portfolio_fund.id,
            record_date=date(2024, 3, 1),
            ex_dividend_date=date(2024, 2, 28),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.COMPLETED,
        )
        db_session.add(dividend)
        db_session.commit()

        # Update dividend
        update_data = {
            "record_date": "2024-03-15",
            "ex_dividend_date": "2024-03-10",
            "dividend_per_share": 0.75,
        }

        updated_dividend, original_values = DividendService.update_dividend(
            dividend.id, update_data
        )

        assert updated_dividend.dividend_per_share == 0.75
        assert updated_dividend.total_amount == 75.0  # 100 * 0.75 (recalculated)
        assert updated_dividend.record_date == date(2024, 3, 15)
        assert original_values["dividend_per_share"] == 0.50

    def test_update_stock_dividend_add_reinvestment(self, app_context, db_session):
        """Test adding reinvestment to pending STOCK dividend."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)

        # Create dividend directly

        from app.models import Dividend

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=portfolio_fund.id,
            record_date=date(2024, 3, 1),
            ex_dividend_date=date(2024, 2, 28),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.PENDING,
            reinvestment_transaction_id=None,
        )
        db_session.add(dividend)
        db_session.commit()

        # Add reinvestment
        update_data = {
            "record_date": dividend.record_date.isoformat(),
            "ex_dividend_date": dividend.ex_dividend_date.isoformat(),
            "dividend_per_share": 0.50,
            "buy_order_date": "2024-02-20",
            "reinvestment_shares": 2.5,
            "reinvestment_price": 20.0,
        }

        updated_dividend, _ = DividendService.update_dividend(dividend.id, update_data)

        assert updated_dividend.reinvestment_status == ReinvestmentStatus.COMPLETED
        assert updated_dividend.reinvestment_transaction_id is not None

        # Verify transaction was created
        from app.models import Transaction

        txn = Transaction.query.get(updated_dividend.reinvestment_transaction_id)
        assert txn.type == "dividend"
        assert txn.shares == 2.5

    def test_update_stock_dividend_modify_reinvestment(self, app_context, db_session):
        """Test modifying existing reinvestment transaction."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)

        # Create dividend with reinvestment transaction

        from app.models import Transaction

        transaction = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            date=date(2024, 2, 10),
            type="dividend",
            shares=2.5,
            cost_per_share=20.0,
        )
        db_session.add(transaction)
        db_session.flush()

        from app.models import Dividend

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=portfolio_fund.id,
            record_date=date(2024, 3, 1),
            ex_dividend_date=date(2024, 2, 28),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.COMPLETED,
            reinvestment_transaction_id=transaction.id,
            buy_order_date=date(2024, 2, 20),
        )
        db_session.add(dividend)
        db_session.commit()

        # Modify reinvestment
        update_data = {
            "record_date": dividend.record_date.isoformat(),
            "ex_dividend_date": dividend.ex_dividend_date.isoformat(),
            "dividend_per_share": 0.50,
            "buy_order_date": "2024-02-20",
            "reinvestment_shares": 3.0,  # Changed
            "reinvestment_price": 22.0,  # Changed
        }

        updated_dividend, _ = DividendService.update_dividend(dividend.id, update_data)

        # Verify transaction was updated
        txn = Transaction.query.get(updated_dividend.reinvestment_transaction_id)
        assert txn.shares == 3.0
        assert txn.cost_per_share == 22.0

    def test_update_stock_dividend_remove_reinvestment(self, app_context, db_session):
        """Test removing reinvestment from STOCK dividend."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)

        # Create dividend with reinvestment transaction

        from app.models import Transaction

        transaction = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            date=date(2024, 2, 10),
            type="dividend",
            shares=2.5,
            cost_per_share=20.0,
        )
        db_session.add(transaction)
        db_session.flush()

        from app.models import Dividend

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=portfolio_fund.id,
            record_date=date(2024, 3, 1),
            ex_dividend_date=date(2024, 2, 28),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.COMPLETED,
            reinvestment_transaction_id=transaction.id,
        )
        db_session.add(dividend)
        db_session.commit()

        transaction_id = dividend.reinvestment_transaction_id

        # Remove reinvestment (don't provide reinvestment data)
        update_data = {
            "record_date": dividend.record_date.isoformat(),
            "ex_dividend_date": dividend.ex_dividend_date.isoformat(),
            "dividend_per_share": 0.50,
        }

        updated_dividend, _ = DividendService.update_dividend(dividend.id, update_data)

        assert updated_dividend.reinvestment_status == ReinvestmentStatus.PENDING
        assert updated_dividend.reinvestment_transaction_id is None

        # Verify transaction was deleted
        from app.models import Transaction

        txn = Transaction.query.get(transaction_id)
        assert txn is None

    def test_update_dividend_not_found(self, app_context, db_session):
        """Test updating non-existent dividend."""
        with pytest.raises(ValueError, match=r"Dividend .* not found"):
            DividendService.update_dividend("non-existent-id", {"dividend_per_share": 0.50})

    def test_update_dividend_validation_negative_reinvestment(self, app_context, db_session):
        """Test validation when updating with negative reinvestment values."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create dividend directly

        from app.models import Dividend

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=portfolio_fund.id,
            record_date=date(2024, 3, 1),
            ex_dividend_date=date(2024, 2, 28),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        db_session.add(dividend)
        db_session.commit()

        # Test with negative reinvestment shares
        with pytest.raises(ValueError, match="Reinvestment shares and price must be positive"):
            DividendService.update_dividend(
                dividend.id,
                {
                    "record_date": "2024-03-01",
                    "ex_dividend_date": "2024-02-28",
                    "dividend_per_share": 0.50,
                    "reinvestment_shares": -2.5,
                    "reinvestment_price": 20.0,
                },
            )

        # Test with zero reinvestment price
        with pytest.raises(ValueError, match="Reinvestment shares and price must be positive"):
            DividendService.update_dividend(
                dividend.id,
                {
                    "record_date": "2024-03-01",
                    "ex_dividend_date": "2024-02-28",
                    "dividend_per_share": 0.50,
                    "reinvestment_shares": 2.5,
                    "reinvestment_price": 0,
                },
            )


class TestDeleteDividend:
    """Test dividend deletion."""

    def test_delete_cash_dividend(self, app_context, db_session):
        """Test deleting CASH dividend (no transaction)."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.CASH)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)

        # Create dividend directly

        from app.models import Dividend

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=portfolio_fund.id,
            record_date=date(2024, 3, 1),
            ex_dividend_date=date(2024, 2, 28),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.COMPLETED,
        )
        db_session.add(dividend)
        db_session.commit()

        dividend_id = dividend.id

        result = DividendService.delete_dividend(dividend_id)

        assert result is True

        # Verify dividend was deleted
        from app.models import Dividend

        deleted = Dividend.query.get(dividend_id)
        assert deleted is None

    def test_delete_stock_dividend_with_transaction(self, app_context, db_session):
        """Test deleting STOCK dividend with reinvestment transaction."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.STOCK)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)

        # Create dividend with transaction

        from app.models import Transaction

        transaction = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            date=date(2024, 2, 10),
            type="dividend",
            shares=2.5,
            cost_per_share=20.0,
        )
        db_session.add(transaction)
        db_session.flush()

        from app.models import Dividend

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=portfolio_fund.id,
            record_date=date(2024, 3, 1),
            ex_dividend_date=date(2024, 2, 28),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.COMPLETED,
            reinvestment_transaction_id=transaction.id,
        )
        db_session.add(dividend)
        db_session.commit()

        dividend_id = dividend.id
        transaction_id = transaction.id

        result = DividendService.delete_dividend(dividend_id)

        assert result is True

        # Verify both dividend and transaction were deleted
        from app.models import Dividend, Transaction

        assert Dividend.query.get(dividend_id) is None
        assert Transaction.query.get(transaction_id) is None


class TestEdgeCases:
    """Test edge cases and error handling."""

    def test_create_dividend_invalid_portfolio_fund(self, app_context, db_session):
        """Test creating dividend with invalid portfolio_fund_id."""
        dividend_data = {
            "portfolio_fund_id": "non-existent-id",
            "record_date": "2024-02-15",
            "ex_dividend_date": "2024-02-10",
            "dividend_per_share": 0.50,
        }

        with pytest.raises(ValueError, match="Portfolio-fund relationship not found"):
            DividendService.create_dividend(dividend_data)

    def test_calculate_shares_zero_shares(self, app_context, db_session):
        """Test share calculation with zero shares."""
        portfolio = PortfolioFactory()
        fund = FundFactory()
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # No transactions created
        shares = DividendService.calculate_shares_owned(portfolio_fund.id, date(2024, 2, 1))

        assert shares == 0

    def test_create_dividend_calculates_correct_total_amount(self, app_context, db_session):
        """Test that total amount is calculated correctly based on shares owned."""
        portfolio = PortfolioFactory()
        fund = FundFactory(dividend_type=DividendType.CASH)
        portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create transaction with fractional shares directly

        from app.models import Transaction

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=portfolio_fund.id,
            type="buy",
            shares=123.456,  # Fractional shares
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        dividend_data = {
            "portfolio_fund_id": portfolio_fund.id,
            "record_date": "2024-02-15",
            "ex_dividend_date": "2024-02-10",
            "dividend_per_share": 0.123,  # Fractional dividend
        }

        dividend = DividendService.create_dividend(dividend_data)

        expected_total = 123.456 * 0.123
        assert abs(dividend.total_amount - expected_total) < 0.0001
