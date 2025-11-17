"""
Tests for TransactionService.

This test suite covers:
- Transaction retrieval (all transactions, portfolio-specific)
- Transaction formatting with IBKR allocation handling
- Transaction creation (buy, sell, dividend types)
- Transaction updates with realized gain/loss handling
- Transaction deletion with IBKR allocation cleanup
- Current position calculation
- Sell transaction processing with realized gains
"""

from datetime import date

import pytest
from app.models import (
    Dividend,
    IBKRTransaction,
    IBKRTransactionAllocation,
    RealizedGainLoss,
    ReinvestmentStatus,
    Transaction,
)
from app.services.transaction_service import TransactionService
from tests.factories import FundFactory, PortfolioFactory, PortfolioFundFactory
from tests.test_helpers import make_id


class TestGetTransactions:
    """Tests for transaction retrieval methods."""

    def test_get_all_transactions(self, app_context, db_session):
        """Test retrieving all transactions."""
        # Create multiple portfolios and transactions
        pf1 = PortfolioFundFactory()
        pf2 = PortfolioFundFactory()
        db_session.commit()

        # Create transactions directly
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf1.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf2.id,
            type="sell",
            shares=50,
            cost_per_share=15.0,
            date=date(2024, 2, 1),
        )
        db_session.add(txn1)
        db_session.add(txn2)
        db_session.commit()

        # Get all transactions
        transactions = TransactionService.get_all_transactions()

        # Should return all transactions
        assert len(transactions) == 2
        assert all(isinstance(t, dict) for t in transactions)
        assert any(t["id"] == txn1.id for t in transactions)
        assert any(t["id"] == txn2.id for t in transactions)

    def test_get_portfolio_transactions(self, app_context, db_session):
        """Test retrieving transactions for specific portfolio."""
        # Create two portfolios
        portfolio1 = PortfolioFactory()
        portfolio2 = PortfolioFactory()
        fund = FundFactory()
        db_session.commit()

        pf1 = PortfolioFundFactory(portfolio=portfolio1, fund=fund)
        pf2 = PortfolioFundFactory(portfolio=portfolio2, fund=fund)
        db_session.commit()

        # Create transactions for each portfolio
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf1.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf2.id,
            type="buy",
            shares=50,
            cost_per_share=15.0,
            date=date(2024, 2, 1),
        )
        db_session.add(txn1)
        db_session.add(txn2)
        db_session.commit()

        # Get transactions for portfolio1 only
        transactions = TransactionService.get_portfolio_transactions(portfolio1.id)

        # Should only return portfolio1 transactions
        assert len(transactions) == 1
        assert transactions[0]["id"] == txn1.id
        assert transactions[0]["fund_name"] == fund.name

    def test_get_portfolio_transactions_with_ibkr_allocation(self, app_context, db_session):
        """Test that get_portfolio_transactions includes IBKR allocation data."""
        portfolio = PortfolioFactory()
        fund = FundFactory()
        pf = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create IBKR transaction
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id="IBKR-GET-PORTFOLIO-TX",
            transaction_date=date(2024, 1, 1),
            symbol="VWCE",
            description="Buy 100 VWCE",
            transaction_type="buy",
            quantity=100,
            price=10.0,
            total_amount=-1000.0,
            currency="USD",
            status="processed",
        )
        db_session.add(ibkr_txn)
        db_session.commit()

        # Create transaction linked to IBKR
        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Create allocation
        allocation = IBKRTransactionAllocation(
            id=make_id(),
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=portfolio.id,
            transaction_id=txn.id,
            allocation_percentage=100.0,
            allocated_amount=-1000.0,
            allocated_shares=100.0,
        )
        db_session.add(allocation)
        db_session.commit()

        # Get transactions
        transactions = TransactionService.get_portfolio_transactions(portfolio.id)

        # Should include IBKR allocation data
        assert len(transactions) == 1
        assert transactions[0]["ibkr_linked"] is True
        assert transactions[0]["ibkr_transaction_id"] == ibkr_txn.id


class TestFormatTransaction:
    """Tests for transaction formatting."""

    def test_format_transaction_basic(self, app_context, db_session):
        """Test basic transaction formatting without IBKR allocation."""
        pf = PortfolioFundFactory()
        db_session.commit()

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Format transaction
        formatted = TransactionService.format_transaction(txn)

        # Verify formatting
        assert formatted["id"] == txn.id
        assert formatted["portfolio_fund_id"] == pf.id
        assert formatted["fund_name"] == pf.fund.name
        assert formatted["date"] == "2024-01-01"
        assert formatted["type"] == "buy"
        assert formatted["shares"] == 100
        assert formatted["cost_per_share"] == 10.0
        assert formatted["ibkr_linked"] is False
        assert formatted["ibkr_transaction_id"] is None

    def test_format_transaction_with_ibkr_allocation(self, app_context, db_session):
        """Test formatting transaction with IBKR allocation."""
        portfolio = PortfolioFactory()
        pf = PortfolioFundFactory(portfolio=portfolio)
        db_session.commit()

        # Create IBKR transaction
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id="IBKR-FORMAT-TX",
            transaction_date=date(2024, 1, 1),
            symbol="VWCE",
            description="Buy 100 VWCE",
            transaction_type="buy",
            quantity=100,
            price=10.0,
            total_amount=-1000.0,
            currency="USD",
            status="processed",
        )
        db_session.add(ibkr_txn)

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Create allocation
        allocation = IBKRTransactionAllocation(
            id=make_id(),
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=portfolio.id,
            transaction_id=txn.id,
            allocation_percentage=100.0,
            allocated_amount=-1000.0,
            allocated_shares=100.0,
        )
        db_session.add(allocation)
        db_session.commit()

        # Format transaction (without pre-loaded allocation)
        formatted = TransactionService.format_transaction(txn)

        # Should query and include IBKR data
        assert formatted["ibkr_linked"] is True
        assert formatted["ibkr_transaction_id"] == ibkr_txn.id

    def test_format_transaction_batch_mode(self, app_context, db_session):
        """Test formatting in batch mode with pre-loaded data."""
        pf = PortfolioFundFactory()
        db_session.commit()

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Format with batch mode (pre-loaded lookup)
        portfolio_fund_lookup = {pf.id: pf.fund.name}
        formatted = TransactionService.format_transaction(
            txn, portfolio_fund_lookup=portfolio_fund_lookup, batch_mode=True
        )

        # Should use pre-loaded data
        assert formatted["fund_name"] == pf.fund.name
        assert formatted["ibkr_linked"] is False


class TestCreateTransaction:
    """Tests for transaction creation."""

    def test_create_buy_transaction(self, app_context, db_session):
        """Test creating a buy transaction."""
        pf = PortfolioFundFactory()
        db_session.commit()

        data = {
            "portfolio_fund_id": pf.id,
            "date": "2024-01-01",
            "type": "buy",
            "shares": 100,
            "cost_per_share": 10.0,
        }

        txn = TransactionService.create_transaction(data)

        # Verify transaction created
        assert txn.portfolio_fund_id == pf.id
        assert txn.date == date(2024, 1, 1)
        assert txn.type == "buy"
        assert txn.shares == 100.0
        assert txn.cost_per_share == 10.0

        # Verify in database
        db_txn = Transaction.query.get(txn.id)
        assert db_txn is not None

    def test_create_sell_transaction_with_realized_gain(self, app_context, db_session):
        """Test creating a sell transaction calculates realized gain/loss."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create buy transaction first (establish position)
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Create sell transaction
        data = {
            "portfolio_fund_id": pf.id,
            "date": "2024-02-01",
            "type": "sell",
            "shares": 50,
            "cost_per_share": 15.0,  # Sold at profit
        }

        txn = TransactionService.create_transaction(data)

        # Verify transaction created
        assert txn.type == "sell"
        assert txn.shares == 50.0

        # Verify realized gain/loss record created
        realized_gain = RealizedGainLoss.query.filter_by(transaction_id=txn.id).first()
        assert realized_gain is not None
        assert realized_gain.shares_sold == 50.0
        assert realized_gain.cost_basis == 500.0  # 50 shares * $10 cost
        assert realized_gain.sale_proceeds == 750.0  # 50 shares * $15 sale price
        assert realized_gain.realized_gain_loss == 250.0  # $750 - $500 profit

    def test_create_sell_transaction_insufficient_shares(self, app_context, db_session):
        """Test creating sell with insufficient shares raises error."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # No buy transactions = no shares to sell
        data = {
            "portfolio_fund_id": pf.id,
            "date": "2024-01-01",
            "type": "sell",
            "shares": 50,
            "cost_per_share": 15.0,
        }

        # Should raise ValueError
        with pytest.raises(ValueError, match="Insufficient shares"):
            TransactionService.create_transaction(data)


class TestUpdateTransaction:
    """Tests for transaction updates."""

    def test_update_buy_transaction(self, app_context, db_session):
        """Test updating a buy transaction."""
        pf = PortfolioFundFactory()
        db_session.commit()

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Update transaction
        data = {
            "date": "2024-01-15",
            "type": "buy",
            "shares": 150,
            "cost_per_share": 12.0,
        }

        updated = TransactionService.update_transaction(txn.id, data)

        # Verify updates
        assert updated.date == date(2024, 1, 15)
        assert updated.shares == 150.0
        assert updated.cost_per_share == 12.0

    def test_update_sell_transaction_recalculates_gain(self, app_context, db_session):
        """Test updating sell transaction recalculates realized gain/loss."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create buy transaction
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Create sell transaction
        sell_data = {
            "portfolio_fund_id": pf.id,
            "date": "2024-02-01",
            "type": "sell",
            "shares": 50,
            "cost_per_share": 15.0,
        }
        sell_txn = TransactionService.create_transaction(sell_data)

        # Get original realized gain
        original_gain = RealizedGainLoss.query.filter_by(transaction_id=sell_txn.id).first()
        assert original_gain.realized_gain_loss == 250.0  # (15-10) * 50

        # Update sell transaction (different price)
        update_data = {
            "date": "2024-02-01",
            "type": "sell",
            "shares": 50,
            "cost_per_share": 20.0,  # Higher sale price
        }
        TransactionService.update_transaction(sell_txn.id, update_data)

        # Verify realized gain recalculated
        # After fix: Service correctly calculates based on average cost
        # Buy: 100 shares @ $10 = $1000 total cost
        # First sell (original): 50 shares @ $15
        #   - Average cost before sell: $10
        #   - Cost basis for this sell: 50 * $10 = $500
        #   - Sale proceeds: 50 * $15 = $750
        #   - Realized gain: $750 - $500 = $250
        # Update sell to new price: 50 shares @ $20
        #   - Position after original sell: 50 shares, $500 cost, $10 avg
        #   - Cost basis for this sell: 50 * $10 = $500
        #   - Sale proceeds: 50 * $20 = $1000
        #   - Realized gain: $1000 - $500 = $500
        updated_gain = RealizedGainLoss.query.filter_by(transaction_id=sell_txn.id).first()
        assert updated_gain.realized_gain_loss == 500.0

    def test_update_buy_to_sell_creates_realized_gain(self, app_context, db_session):
        """Test changing buy to sell creates realized gain record."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create two buy transactions (need shares to sell)
        buy1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        buy2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=50,
            cost_per_share=10.0,
            date=date(2024, 1, 15),
        )
        db_session.add(buy1)
        db_session.add(buy2)
        db_session.commit()

        # Change buy2 to sell
        update_data = {
            "date": "2024-02-01",
            "type": "sell",
            "shares": 50,
            "cost_per_share": 15.0,
        }
        TransactionService.update_transaction(buy2.id, update_data)

        # Verify realized gain created
        realized_gain = RealizedGainLoss.query.filter_by(transaction_id=buy2.id).first()
        assert realized_gain is not None
        # After fix: Service correctly calculates based on average cost
        # Buy1: 100 shares @ $10 = $1000
        # Buy2 → Sell: 50 shares @ $15
        #   - Position before sell: 150 shares, $1500 cost
        #   - Average cost: $1500 / 150 = $10
        #   - Cost basis: 50 * $10 = $500
        #   - Sale proceeds: 50 * $15 = $750
        #   - Realized gain: $750 - $500 = $250
        assert realized_gain.realized_gain_loss == 250.0


class TestDeleteTransaction:
    """Tests for transaction deletion."""

    def test_delete_buy_transaction(self, app_context, db_session):
        """Test deleting a buy transaction."""
        pf = PortfolioFundFactory()
        db_session.commit()

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        txn_id = txn.id

        # Delete transaction
        result = TransactionService.delete_transaction(txn_id)

        # Verify deletion
        assert result["transaction_details"]["type"] == "buy"
        assert Transaction.query.get(txn_id) is None

    def test_delete_sell_transaction_removes_realized_gain(self, app_context, db_session):
        """Test deleting sell transaction also deletes realized gain record."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create buy and sell
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(buy_txn)
        db_session.commit()

        sell_data = {
            "portfolio_fund_id": pf.id,
            "date": "2024-02-01",
            "type": "sell",
            "shares": 50,
            "cost_per_share": 15.0,
        }
        sell_txn = TransactionService.create_transaction(sell_data)

        # Verify realized gain exists
        realized_gain = RealizedGainLoss.query.filter_by(transaction_id=sell_txn.id).first()
        assert realized_gain is not None

        # Delete sell transaction
        result = TransactionService.delete_transaction(sell_txn.id)

        # Verify both transaction and realized gain deleted
        assert result["realized_gain_deleted"] is True
        assert Transaction.query.get(sell_txn.id) is None
        assert RealizedGainLoss.query.filter_by(transaction_id=sell_txn.id).first() is None

    def test_delete_transaction_with_ibkr_allocation(self, app_context, db_session):
        """Test deleting transaction with IBKR allocation reverts IBKR status."""
        portfolio = PortfolioFactory()
        pf = PortfolioFundFactory(portfolio=portfolio)
        db_session.commit()

        # Create IBKR transaction
        ibkr_txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id="IBKR-DELETE-TX",
            transaction_date=date(2024, 1, 1),
            symbol="VWCE",
            description="Buy 100 VWCE",
            transaction_type="buy",
            quantity=100,
            price=10.0,
            total_amount=-1000.0,
            currency="USD",
            status="processed",
        )
        db_session.add(ibkr_txn)

        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(txn)
        db_session.commit()

        # Create allocation (only allocation for this IBKR transaction)
        allocation = IBKRTransactionAllocation(
            id=make_id(),
            ibkr_transaction_id=ibkr_txn.id,
            portfolio_id=portfolio.id,
            transaction_id=txn.id,
            allocation_percentage=100.0,
            allocated_amount=-1000.0,
            allocated_shares=100.0,
        )
        db_session.add(allocation)
        db_session.commit()

        # Delete transaction
        result = TransactionService.delete_transaction(txn.id)

        # Verify IBKR transaction reverted to pending
        assert result["ibkr_reverted"] is True
        assert result["ibkr_transaction_id"] == ibkr_txn.id

        # Check IBKR transaction status
        ibkr = IBKRTransaction.query.get(ibkr_txn.id)
        assert ibkr.status == "pending"
        assert ibkr.processed_at is None


class TestCalculateCurrentPosition:
    """Tests for current position calculation."""

    def test_calculate_position_with_buys_only(self, app_context, db_session):
        """Test position calculation with only buy transactions."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create buy transactions
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=50,
            cost_per_share=12.0,
            date=date(2024, 2, 1),
        )
        db_session.add(txn1)
        db_session.add(txn2)
        db_session.commit()

        # Calculate position
        position = TransactionService.calculate_current_position(pf.id)

        # Verify calculation
        # Total shares: 100 + 50 = 150
        # Total cost: (100 * 10) + (50 * 12) = 1000 + 600 = 1600
        # Average cost: 1600 / 150 = 10.666...
        assert position["total_shares"] == 150.0
        assert position["total_cost"] == 1600.0
        assert round(position["average_cost"], 2) == 10.67

    def test_calculate_position_with_buys_and_sells(self, app_context, db_session):
        """Test position calculation with buys and sells."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create transactions
        buy1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        buy2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=50,
            cost_per_share=12.0,
            date=date(2024, 2, 1),
        )
        sell1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="sell",
            shares=30,
            cost_per_share=15.0,
            date=date(2024, 3, 1),
        )
        db_session.add_all([buy1, buy2, sell1])
        db_session.commit()

        # Calculate position
        position = TransactionService.calculate_current_position(pf.id)

        # Verify calculation (after bug fix)
        # Buy 100 @ $10 = $1000
        # Buy 50 @ $12 = $600
        # Position: 150 shares, $1600 total
        # Average cost before sell: $1600 / 150 = $10.666...
        # Sell 30 shares:
        #   - Cost basis reduction: 30 * $10.67 = $320
        #   - Remaining cost: $1600 - $320 = $1280
        # Final position: 120 shares, $1280 cost, $10.67 average
        assert position["total_shares"] == 120.0
        assert position["total_cost"] == 1280.0
        assert round(position["average_cost"], 2) == 10.67

    def test_calculate_position_with_dividend_shares(self, app_context, db_session):
        """Test position calculation includes dividend reinvestment shares."""
        portfolio = PortfolioFactory()
        fund = FundFactory()
        pf = PortfolioFundFactory(portfolio=portfolio, fund=fund)
        db_session.commit()

        # Create buy transaction
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Create dividend with reinvestment transaction
        dividend_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="dividend",
            shares=5,
            cost_per_share=10.0,
            date=date(2024, 2, 1),
        )
        db_session.add(dividend_txn)
        db_session.commit()

        dividend = Dividend(
            id=make_id(),
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=date(2024, 2, 1),
            ex_dividend_date=date(2024, 1, 31),
            dividend_per_share=0.50,
            shares_owned=100,
            total_amount=50.0,
            reinvestment_status=ReinvestmentStatus.COMPLETED,
            reinvestment_transaction_id=dividend_txn.id,
        )
        db_session.add(dividend)
        db_session.commit()

        # Calculate position
        position = TransactionService.calculate_current_position(pf.id)

        # Should include dividend shares
        # Total shares: 100 (buy) + 5 (dividend) = 105
        assert position["total_shares"] == 105.0

    def test_calculate_position_empty(self, app_context, db_session):
        """Test position calculation with no transactions."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Calculate position with no transactions
        position = TransactionService.calculate_current_position(pf.id)

        # Should return zeros
        assert position["total_shares"] == 0.0
        assert position["total_cost"] == 0.0
        assert position["average_cost"] == 0.0

    def test_calculate_position_all_shares_sold(self, app_context, db_session):
        """Test position calculation when all shares are sold."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Buy and sell all
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        sell_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="sell",
            shares=100,
            cost_per_share=15.0,
            date=date(2024, 2, 1),
        )
        db_session.add_all([buy_txn, sell_txn])
        db_session.commit()

        # Calculate position
        position = TransactionService.calculate_current_position(pf.id)

        # Should return zeros when fully sold
        assert position["total_shares"] == 0.0
        assert position["total_cost"] == 0.0
        assert position["average_cost"] == 0.0


class TestProcessSellTransaction:
    """Tests for sell transaction processing."""

    def test_process_sell_transaction(self, app_context, db_session):
        """Test processing sell transaction creates transaction and realized gain."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create buy transaction first
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=10.0,
            date=date(2024, 1, 1),
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Process sell
        result = TransactionService.process_sell_transaction(
            portfolio_fund_id=pf.id,
            shares=50,
            price=15.0,
            date=date(2024, 2, 1),
        )

        # Verify transaction created
        txn = result["transaction"]
        assert txn.type == "sell"
        assert txn.shares == 50
        assert txn.cost_per_share == 15.0

        # Verify realized gain/loss calculated
        assert result["realized_gain_loss"] == 250.0  # (15 - 10) * 50

        # Verify realized gain record in database
        gain_record = RealizedGainLoss.query.filter_by(transaction_id=txn.id).first()
        assert gain_record is not None
        assert gain_record.realized_gain_loss == 250.0

    def test_process_sell_transaction_insufficient_shares(self, app_context, db_session):
        """Test processing sell with insufficient shares raises error."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # No buy = no shares
        with pytest.raises(ValueError, match="Insufficient shares"):
            TransactionService.process_sell_transaction(
                portfolio_fund_id=pf.id, shares=50, price=15.0, date=date(2024, 1, 1)
            )

    def test_process_sell_transaction_realized_loss(self, app_context, db_session):
        """Test processing sell at a loss calculates negative realized gain."""
        pf = PortfolioFundFactory()
        db_session.commit()

        # Create buy transaction
        buy_txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            type="buy",
            shares=100,
            cost_per_share=15.0,  # Buy at $15
            date=date(2024, 1, 1),
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Sell at loss
        result = TransactionService.process_sell_transaction(
            portfolio_fund_id=pf.id,
            shares=50,
            price=10.0,  # Sell at $10
            date=date(2024, 2, 1),
        )

        # Verify loss calculated
        # Cost basis: 50 * $15 = $750
        # Sale proceeds: 50 * $10 = $500
        # Loss: $500 - $750 = -$250
        assert result["realized_gain_loss"] == -250.0


class TestEdgeCases:
    """Tests for edge cases and error handling."""

    def test_delete_nonexistent_transaction(self, app_context, db_session):
        """Test deleting non-existent transaction raises error."""
        with pytest.raises(ValueError, match="not found"):
            TransactionService.delete_transaction("non-existent-id")

    def test_update_nonexistent_transaction(self, app_context, db_session):
        """Test updating non-existent transaction raises 404."""
        from werkzeug.exceptions import NotFound

        with pytest.raises(NotFound):
            TransactionService.update_transaction(
                "non-existent-id",
                {"date": "2024-01-01", "type": "buy", "shares": 100, "cost_per_share": 10.0},
            )

    def test_get_portfolio_transactions_empty_portfolio(self, app_context, db_session):
        """Test getting transactions for portfolio with no transactions."""
        portfolio = PortfolioFactory()
        db_session.commit()

        # Get transactions for empty portfolio
        transactions = TransactionService.get_portfolio_transactions(portfolio.id)

        # Should return empty list
        assert transactions == []
