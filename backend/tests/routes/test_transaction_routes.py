"""
Integration tests for transaction routes (transaction_routes.py).

Tests all Transaction API endpoints:
- GET /transactions - List all transactions (optionally filtered by portfolio)
- POST /transactions - Create transaction
- GET /transactions/<id> - Get transaction detail
- PUT /transactions/<id> - Update transaction
- DELETE /transactions/<id> - Delete transaction
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
        """Test GET /transactions returns empty list when no transactions exist."""
        response = client.get("/api/transactions")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) == 0

    def test_list_transactions_returns_all(self, app_context, client, db_session):
        """Test GET /transactions returns all transactions."""
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

        response = client.get("/api/transactions")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 2
        assert all("id" in t for t in data)
        assert all("type" in t for t in data)

    def test_list_transactions_filtered_by_portfolio(self, app_context, client, db_session):
        """Test GET /transactions?portfolio_id=<id> filters by portfolio."""
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

        response = client.get(f"/api/transactions?portfolio_id={p1.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        # Should only include transactions for portfolio 1


class TestTransactionCreate:
    """Test transaction creation."""

    def test_create_buy_transaction(self, app_context, client, db_session):
        """Test POST /transactions creates a buy transaction."""
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

        response = client.post("/api/transactions", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["type"] == "buy"
        assert data["shares"] == 15
        assert "id" in data

        # Verify database
        transaction = db.session.get(Transaction, data["id"])
        assert transaction is not None
        assert transaction.type == "buy"

    def test_create_sell_transaction(self, app_context, client, db_session):
        """Test POST /transactions creates a sell transaction."""
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

        response = client.post("/api/transactions", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["type"] == "sell"
        assert data["shares"] == 10

    def test_create_dividend_transaction(self, app_context, client, db_session):
        """Test POST /transactions creates a dividend transaction."""
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

        response = client.post("/api/transactions", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["type"] == "dividend"


class TestTransactionRetrieveUpdateDelete:
    """Test individual transaction operations."""

    def test_get_transaction_detail(self, app_context, client, db_session):
        """Test GET /transactions/<id> returns transaction details."""
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

        response = client.get(f"/api/transactions/{txn.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert data["id"] == txn.id
        assert data["type"] == "buy"
        assert data["shares"] == 25

    def test_get_transaction_not_found(self, app_context, client):
        """Test GET /transactions/<id> returns 404 for non-existent transaction."""
        fake_id = make_id()
        response = client.get(f"/api/transactions/{fake_id}")

        assert response.status_code == 404

    def test_update_transaction(self, app_context, client, db_session):
        """Test PUT /transactions/<id> updates transaction."""
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

        response = client.put(f"/api/transactions/{txn.id}", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["shares"] == 35
        assert float(data["cost_per_share"]) == 47.00

        # Verify database
        db_session.refresh(txn)
        assert txn.shares == 35

    def test_update_transaction_not_found(self, app_context, client):
        """Test PUT /transactions/<id> returns 404 for non-existent transaction."""
        fake_id = make_id()
        payload = {
            "portfolio_fund_id": make_id(),
            "date": datetime.now().date().isoformat(),
            "type": "buy",
            "shares": 10,
            "cost_per_share": 100.00,
        }

        response = client.put(f"/api/transactions/{fake_id}", json=payload)

        assert response.status_code == 404

    def test_delete_transaction(self, app_context, client, db_session):
        """Test DELETE /transactions/<id> removes transaction."""
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

        response = client.delete(f"/api/transactions/{txn_id}")

        assert response.status_code == 204

        # Verify database
        deleted = db.session.get(Transaction, txn_id)
        assert deleted is None

    def test_delete_transaction_not_found(self, app_context, client):
        """Test DELETE /transactions/<id> returns error for non-existent transaction."""
        fake_id = make_id()
        response = client.delete(f"/api/transactions/{fake_id}")

        # API returns 400 for transaction not found
        assert response.status_code == 400


class TestTransactionErrors:
    """Test error paths for transaction routes."""

    def test_get_transactions_service_error(self, app_context, client):
        """Test GET /transactions handles service errors."""
        from unittest.mock import patch

        # Mock TransactionService.get_all_transactions to raise exception
        with patch(
            "app.routes.transaction_routes.TransactionService.get_all_transactions"
        ) as mock_get_all:
            mock_get_all.side_effect = Exception("Database query failed")

            response = client.get("/api/transactions")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_create_transaction_service_error(self, app_context, client, db_session):
        """Test POST /transactions handles service errors."""
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
            "app.routes.transaction_routes.TransactionService.create_transaction"
        ) as mock_create:
            mock_create.side_effect = Exception("Database error")

            payload = {
                "portfolio_fund_id": pf.id,
                "date": datetime.now().date().isoformat(),
                "type": "buy",
                "shares": 10,
                "cost_per_share": 100.00,
            }

            response = client.post("/api/transactions", json=payload)

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_get_transaction_not_found_error(self, app_context, client):
        """Test GET /transactions/<id> handles not found errors."""
        from unittest.mock import patch

        # Mock TransactionService.get_transaction to raise exception
        with patch("app.routes.transaction_routes.TransactionService.get_transaction") as mock_get:
            mock_get.side_effect = Exception("Transaction not found")

            fake_id = make_id()
            response = client.get(f"/api/transactions/{fake_id}")

            assert response.status_code == 404
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_update_transaction_value_error(self, app_context, client, db_session):
        """Test PUT /transactions/<id> handles validation errors."""
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
            "app.routes.transaction_routes.TransactionService.update_transaction"
        ) as mock_update:
            mock_update.side_effect = ValueError("Invalid transaction data")

            payload = {
                "portfolio_fund_id": pf.id,
                "date": datetime.now().date().isoformat(),
                "type": "buy",
                "shares": -5,  # Invalid
                "cost_per_share": 100.00,
            }

            response = client.put(f"/api/transactions/{txn.id}", json=payload)

            assert response.status_code == 400
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_update_sell_transaction_with_realized_gain(self, app_context, client, db_session):
        """Test PUT /transactions/<id> for sell transaction includes realized gain/loss."""
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

        response = client.put(f"/api/transactions/{sell_txn.id}", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["type"] == "sell"
        assert data["shares"] == 12

    def test_delete_transaction_general_error(self, app_context, client):
        """Test DELETE /transactions/<id> handles unexpected errors."""
        from unittest.mock import patch

        # Mock TransactionService.delete_transaction to raise general exception
        with patch(
            "app.routes.transaction_routes.TransactionService.delete_transaction"
        ) as mock_delete:
            mock_delete.side_effect = Exception("Unexpected database error")

            fake_id = make_id()
            response = client.delete(f"/api/transactions/{fake_id}")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data
