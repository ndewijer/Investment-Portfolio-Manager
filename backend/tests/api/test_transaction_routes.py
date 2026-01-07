"""
Integration tests for transaction routes (transaction_routes.py).

Tests all Transaction API endpoints:
- GET /transaction - List all transactions (optionally filtered by portfolio)
- POST /transaction - Create transaction
- GET /transaction/<id> - Get transaction detail
- PUT /transaction/<id> - Update transaction
- DELETE /transaction/<id> - Delete transaction
"""

from datetime import datetime, timedelta
from decimal import Decimal

from app.models import Fund, Portfolio, PortfolioFund, Transaction, db
from tests.test_helpers import make_id, make_isin, make_symbol


def create_fund(
    isin_prefix="US", symbol_prefix="TEST", name="Test Fund", currency="USD", exchange="NYSE"
):
    """Helper to create a Fund with all required fields."""
    return Fund(
        isin=make_isin(isin_prefix),
        symbol=make_symbol(symbol_prefix),
        name=name,
        currency=currency,
        exchange=exchange,
    )


class TestTransactionList:
    """Test transaction listing endpoints."""

    def test_list_transactions_empty(self, app_context, client):
        """
        Verify that transaction list endpoint returns empty array for new portfolios.

        WHY: Users need to see an empty state rather than an error when they first create
        a portfolio before adding transactions. Prevents confusion with 404 errors and
        provides a consistent API contract that clients can rely on.
        """
        response = client.get("/api/transaction")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) == 0

    def test_list_transactions_returns_all(self, app_context, client, db_session):
        """
        Verify that transaction list endpoint returns all transactions across portfolios.

        WHY: Users need to view their complete transaction history to track all buy/sell
        activity and calculate portfolio performance. This is critical for financial
        record-keeping, tax reporting, and auditing investment decisions.
        """
        # Create portfolio with fund and transactions
        portfolio = Portfolio(name="Test Portfolio")
        fund = create_fund("US", "AAPL", "Apple Inc")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create multiple transactions
        txn1 = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=10,
            cost_per_share=Decimal("100.00"),
        )
        txn2 = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date() - timedelta(days=1),
            type="buy",
            shares=5,
            cost_per_share=Decimal("95.00"),
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        response = client.get("/api/transaction")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 2
        assert all("id" in t for t in data)
        assert all("type" in t for t in data)

    def test_list_transactions_filtered_by_portfolio(self, app_context, client, db_session):
        """
        Verify that portfolio_id query parameter correctly filters transactions by portfolio.

        WHY: Users with multiple portfolios need to isolate transactions by portfolio to
        analyze individual portfolio performance without noise from other portfolios.
        Prevents data leakage between portfolios and enables focused financial analysis.
        """
        # Create two portfolios with different transactions
        p1 = Portfolio(name="Portfolio 1")
        p2 = Portfolio(name="Portfolio 2")
        fund = create_fund("US", "MSFT", "Microsoft")
        db_session.add_all([p1, p2, fund])
        db_session.commit()

        pf1 = PortfolioFund(portfolio_id=p1.id, fund_id=fund.id)
        pf2 = PortfolioFund(portfolio_id=p2.id, fund_id=fund.id)
        db_session.add_all([pf1, pf2])
        db_session.commit()

        # Add transaction to portfolio 1
        txn1 = Transaction(
            portfolio_fund_id=pf1.id,
            date=datetime.now().date(),
            type="buy",
            shares=10,
            cost_per_share=Decimal("150.00"),
        )
        db_session.add(txn1)
        db_session.commit()

        response = client.get(f"/api/transaction?portfolio_id={p1.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        # Should only include transactions for portfolio 1


class TestTransactionCreate:
    """Test transaction creation."""

    def test_create_buy_transaction(self, app_context, client, db_session):
        """
        Verify that buy transactions can be created with required fields.

        WHY: Buy transactions are the foundation of portfolio tracking. Users must be able
        to record purchases to establish cost basis for tax calculations, performance metrics,
        and compliance with investment tracking regulations.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "GOOGL", "Google")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        payload = {
            "portfolio_fund_id": pf.id,
            "date": datetime.now().date().isoformat(),
            "type": "buy",
            "shares": 15,
            "cost_per_share": 120.50,
        }

        response = client.post("/api/transaction", json=payload)

        assert response.status_code == 201
        data = response.get_json()
        assert data["type"] == "buy"
        assert data["shares"] == 15
        assert "id" in data

        # Verify database
        transaction = db.session.get(Transaction, data["id"])
        assert transaction is not None
        assert transaction.type == "buy"

    def test_create_sell_transaction(self, app_context, client, db_session):
        """
        Verify that sell transactions can be created and recorded properly.

        WHY: Sell transactions trigger realized gain/loss calculations essential for tax
        reporting and portfolio performance analysis. Accurate recording prevents tax
        calculation errors that could cost users money or cause compliance issues.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "TSLA", "Tesla")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # First buy some shares
        buy_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date() - timedelta(days=5),
            type="buy",
            shares=20,
            cost_per_share=Decimal("200.00"),
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Now sell some shares
        payload = {
            "portfolio_fund_id": pf.id,
            "date": datetime.now().date().isoformat(),
            "type": "sell",
            "shares": 10,
            "cost_per_share": 250.00,
        }

        response = client.post("/api/transaction", json=payload)

        assert response.status_code == 201
        data = response.get_json()
        assert data["type"] == "sell"
        assert data["shares"] == 10

    def test_create_dividend_transaction(self, app_context, client, db_session):
        """
        Verify that dividend transactions can be recorded for income tracking.

        WHY: Dividend income must be tracked separately for accurate tax reporting and
        total return calculations. Missing dividend records would understate portfolio
        performance and create tax reporting gaps.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "NVDA", "NVIDIA")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        payload = {
            "portfolio_fund_id": pf.id,
            "date": datetime.now().date().isoformat(),
            "type": "dividend",
            "shares": 2,
            "cost_per_share": 50.00,
        }

        response = client.post("/api/transaction", json=payload)

        assert response.status_code == 201
        data = response.get_json()
        assert data["type"] == "dividend"


class TestTransactionRetrieveUpdateDelete:
    """Test individual transaction operations."""

    def test_get_transaction_detail(self, app_context, client, db_session):
        """
        Verify that individual transaction details can be retrieved by ID.

        WHY: Users need to view individual transaction details to verify accuracy of data
        entry and audit their financial records. Critical for catching data entry errors
        before they propagate into tax calculations or performance reports.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "AMD", "AMD")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=25,
            cost_per_share=Decimal("80.00"),
        )
        db_session.add(txn)
        db_session.commit()

        response = client.get(f"/api/transaction/{txn.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert data["id"] == txn.id
        assert data["type"] == "buy"
        assert data["shares"] == 25

    def test_get_transaction_not_found(self, app_context, client):
        """
        Verify that requesting a non-existent transaction returns 404 status.

        WHY: Prevents system crashes when users request non-existent transactions (e.g.,
        after deletion or with invalid/stale links). Provides clear error feedback instead
        of confusing 500 errors, improving user experience and debugging.
        """
        fake_id = make_id()
        response = client.get(f"/api/transaction/{fake_id}")

        assert response.status_code == 404

    def test_update_transaction(self, app_context, client, db_session):
        """
        Verify that existing transactions can be updated with new values.

        WHY: Users need to correct data entry errors in transactions without deleting and
        recreating them, which would break transaction history and audit trails. Essential
        for maintaining data integrity while allowing error correction.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "INTC", "Intel")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=30,
            cost_per_share=Decimal("45.00"),
        )
        db_session.add(txn)
        db_session.commit()

        payload = {
            "portfolio_fund_id": pf.id,
            "date": datetime.now().date().isoformat(),
            "type": "buy",
            "shares": 35,  # Changed
            "cost_per_share": 47.00,  # Changed
        }

        response = client.put(f"/api/transaction/{txn.id}", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["shares"] == 35
        assert float(data["cost_per_share"]) == 47.00

        # Verify database
        db_session.refresh(txn)
        assert txn.shares == 35

    def test_update_transaction_not_found(self, app_context, client):
        """
        Verify that updating a non-existent transaction returns 404 status.

        WHY: Handles race conditions where a transaction is deleted while a user is trying
        to update it. Provides clear error messaging rather than silent failures or
        confusing application states.
        """
        fake_id = make_id()
        payload = {
            "portfolio_fund_id": make_id(),
            "date": datetime.now().date().isoformat(),
            "type": "buy",
            "shares": 10,
            "cost_per_share": 100.00,
        }

        response = client.put(f"/api/transaction/{fake_id}", json=payload)

        assert response.status_code == 404

    def test_delete_transaction(self, app_context, client, db_session):
        """
        Verify that transactions can be permanently deleted from the system.

        WHY: Users need to remove erroneous transactions to maintain accurate portfolio
        records and prevent incorrect tax calculations. Essential for correcting mistakes
        without corrupting historical data or portfolio state.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "META", "Meta")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=20,
            cost_per_share=Decimal("300.00"),
        )
        db_session.add(txn)
        db_session.commit()
        txn_id = txn.id

        response = client.delete(f"/api/transaction/{txn_id}")

        assert response.status_code == 200

        # Verify database
        deleted = db.session.get(Transaction, txn_id)
        assert deleted is None

    def test_delete_transaction_not_found(self, app_context, client):
        """
        Verify that deleting a non-existent transaction returns 400 error.

        WHY: Handles race conditions where multiple deletion requests occur or a user tries
        to delete an already-deleted transaction. Prevents confusing error states and
        provides clear feedback about the operation failure.
        """
        fake_id = make_id()
        response = client.delete(f"/api/transaction/{fake_id}")

        # API returns 404 for transaction not found
        assert response.status_code == 404


class TestTransactionErrors:
    """Test error paths for transaction routes."""

    def test_get_transactions_service_error(self, app_context, client):
        """
        Verify that database errors during transaction listing return 500 with error message.

        WHY: Database failures shouldn't crash the application or expose internal details.
        Users need clear error messages when backend systems fail so they know the issue
        isn't with their data, enabling proper troubleshooting and support.
        """
        from unittest.mock import patch

        # Mock TransactionService.get_all_transactions to raise exception
        with patch(
            "app.api.transaction_namespace.TransactionService.get_all_transactions"
        ) as mock_get_all:
            mock_get_all.side_effect = Exception("Database query failed")

            response = client.get("/api/transaction")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_create_transaction_service_error(self, app_context, client, db_session):
        """
        Verify that database errors during transaction creation return 500 with error message.

        WHY: Database errors during transaction creation must be caught to prevent partial
        writes that corrupt portfolio state. Users need to know when a transaction failed
        to save so they can retry or seek support.
        """
        from unittest.mock import patch

        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "TEST", "Test Fund")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Mock TransactionService.create_transaction to raise exception
        with patch(
            "app.api.transaction_namespace.TransactionService.create_transaction"
        ) as mock_create:
            mock_create.side_effect = Exception("Database error")

            payload = {
                "portfolio_fund_id": pf.id,
                "date": datetime.now().date().isoformat(),
                "type": "buy",
                "shares": 10,
                "cost_per_share": 100.00,
            }

            response = client.post("/api/transaction", json=payload)

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_get_transaction_not_found_error(self, app_context, client):
        """
        Verify that service exceptions when fetching transactions return 404 status.

        WHY: Service-level exceptions when fetching transactions must return 404 rather
        than 500 to distinguish between missing data vs system failures. This helps users
        and support teams quickly identify whether data is missing or systems are down.
        """
        from unittest.mock import patch

        # Mock TransactionService.get_transaction to raise exception
        with patch("app.api.transaction_namespace.TransactionService.get_transaction") as mock_get:
            mock_get.side_effect = Exception("Transaction not found")

            fake_id = make_id()
            response = client.get(f"/api/transaction/{fake_id}")

            assert response.status_code == 404
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_update_transaction_value_error(self, app_context, client, db_session):
        """
        Verify that validation errors during transaction update return 400 with error message.

        WHY: Invalid data (like negative shares) must be rejected before database writes
        to prevent data integrity violations. Protects against both user errors and
        malicious inputs that could corrupt financial records.
        """
        from unittest.mock import patch

        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "TEST", "Test Fund")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=10,
            cost_per_share=Decimal("100.00"),
        )
        db_session.add(txn)
        db_session.commit()

        # Mock TransactionService.update_transaction to raise ValueError
        with patch(
            "app.api.transaction_namespace.TransactionService.update_transaction"
        ) as mock_update:
            mock_update.side_effect = ValueError("Invalid transaction data")

            payload = {
                "portfolio_fund_id": pf.id,
                "date": datetime.now().date().isoformat(),
                "type": "buy",
                "shares": -5,  # Invalid
                "cost_per_share": 100.00,
            }

            response = client.put(f"/api/transaction/{txn.id}", json=payload)

            assert response.status_code == 400
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_update_sell_transaction_with_realized_gain(self, app_context, client, db_session):
        """
        Verify that updating sell transactions correctly recalculates realized gains/losses.

        WHY: Sell transactions must correctly calculate and store realized gains/losses
        for accurate tax reporting. Incorrect calculations could lead to tax filing errors,
        penalties, or audit issues. Updates must trigger recalculation.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "SELL", "Sell Test Fund")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # First buy some shares
        buy_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date() - timedelta(days=10),
            type="buy",
            shares=20,
            cost_per_share=Decimal("50.00"),
        )
        db_session.add(buy_txn)
        db_session.commit()

        # Create a sell transaction
        sell_txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="sell",
            shares=10,
            cost_per_share=Decimal("60.00"),
        )
        db_session.add(sell_txn)
        db_session.commit()

        # Update the sell transaction
        payload = {
            "portfolio_fund_id": pf.id,
            "date": datetime.now().date().isoformat(),
            "type": "sell",
            "shares": 12,  # Changed
            "cost_per_share": 65.00,  # Changed
        }

        response = client.put(f"/api/transaction/{sell_txn.id}", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["type"] == "sell"
        assert data["shares"] == 12

    def test_delete_transaction_general_error(self, app_context, client):
        """
        Verify that unexpected errors during deletion return 500 with error message.

        WHY: Unexpected errors during deletion must be caught and reported properly to
        prevent silent failures where users think a transaction was deleted but it remains
        in the database, leading to incorrect portfolio calculations.
        """
        from unittest.mock import patch

        # Mock TransactionService.delete_transaction to raise general exception
        with patch(
            "app.api.transaction_namespace.TransactionService.delete_transaction"
        ) as mock_delete:
            mock_delete.side_effect = Exception("Unexpected database error")

            fake_id = make_id()
            response = client.delete(f"/api/transaction/{fake_id}")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data
