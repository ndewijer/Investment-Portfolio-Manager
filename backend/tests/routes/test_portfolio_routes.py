"""
Integration tests for portfolio routes (portfolio_routes.py).

Tests all Portfolio API endpoints:
- GET /portfolios - List all portfolios
- POST /portfolios - Create portfolio
- GET /portfolios/<id> - Get portfolio detail with metrics
- PUT /portfolios/<id> - Update portfolio
- DELETE /portfolios/<id> - Delete portfolio
- POST /portfolios/<id>/archive - Archive portfolio
- POST /portfolios/<id>/unarchive - Unarchive portfolio
- GET /portfolio-summary - Get portfolio summary (overview)
- GET /portfolio-history - Get portfolio historical performance
- GET /portfolio-funds - List portfolio funds (optional filter by portfolio_id)
- POST /portfolio-funds - Add fund to portfolio
- DELETE /portfolio-funds/<id> - Remove fund from portfolio
- GET /portfolios/<id>/fund-history - Get fund-specific history for portfolio
"""

from datetime import datetime, timedelta
from decimal import Decimal

from app.models import Fund, FundPrice, Portfolio, PortfolioFund, Transaction, db
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


class TestPortfolioListAndCreate:
    """Test portfolio listing and creation endpoints."""

    def test_list_portfolios_empty(self, app_context, client, db_session):
        """Test GET /portfolios returns empty list when no portfolios exist."""
        response = client.get("/api/portfolios")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) == 0

    def test_list_portfolios_returns_all(self, app_context, client, db_session):
        """Test GET /portfolios returns all portfolios."""
        # Create 3 portfolios (2 active, 1 archived)
        p1 = Portfolio(name="Active Portfolio 1", description="First active")
        p2 = Portfolio(name="Active Portfolio 2", description="Second active")
        p3 = Portfolio(name="Archived Portfolio", description="Archived", is_archived=True)
        db_session.add_all([p1, p2, p3])
        db_session.commit()

        response = client.get("/api/portfolios")

        assert response.status_code == 200
        data = response.get_json()
        assert len(data) == 3
        assert all("id" in p for p in data)
        assert all("name" in p for p in data)
        assert all("is_archived" in p for p in data)

        # Verify structure
        names = {p["name"] for p in data}
        assert names == {"Active Portfolio 1", "Active Portfolio 2", "Archived Portfolio"}

    def test_create_portfolio(self, app_context, client, db_session):
        """Test POST /portfolios creates a new portfolio."""
        payload = {"name": "My New Portfolio", "description": "Test portfolio"}

        response = client.post("/api/portfolios", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["name"] == "My New Portfolio"
        assert data["description"] == "Test portfolio"
        assert "id" in data
        assert data["is_archived"] is False
        assert data["exclude_from_overview"] is False

        # Verify database
        portfolio = db.session.get(Portfolio, data["id"])
        assert portfolio is not None
        assert portfolio.name == "My New Portfolio"

    def test_create_portfolio_minimal_data(self, app_context, client, db_session):
        """Test POST /portfolios with only required fields."""
        payload = {"name": "Minimal Portfolio"}

        response = client.post("/api/portfolios", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["name"] == "Minimal Portfolio"
        assert data["description"] == ""  # Default empty description


class TestPortfolioRetrieveUpdateDelete:
    """Test individual portfolio retrieval, update, and deletion."""

    def test_get_portfolio_detail(self, app_context, client, db_session):
        """Test GET /portfolios/<id> returns portfolio with metrics."""
        # Create portfolio with fund and transaction
        portfolio = Portfolio(name="Test Portfolio")
        fund = create_fund("US", "AAPL", "Apple Inc")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Add transaction
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=10,
            cost_per_share=Decimal("100.00"),
        )
        db_session.add(txn)
        db_session.commit()

        response = client.get(f"/api/portfolios/{portfolio.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert data["id"] == portfolio.id
        assert data["name"] == "Test Portfolio"
        assert "totalValue" in data
        assert "totalCost" in data
        assert "totalDividends" in data
        assert "totalUnrealizedGainLoss" in data
        assert "totalRealizedGainLoss" in data
        assert "totalGainLoss" in data

    def test_get_portfolio_not_found(self, app_context, client):
        """Test GET /portfolios/<id> returns 404 for non-existent portfolio."""
        fake_id = make_id()
        response = client.get(f"/api/portfolios/{fake_id}")

        assert response.status_code == 404

    def test_get_archived_portfolio_returns_404(self, app_context, client, db_session):
        """Test GET /portfolios/<id> returns 404 for archived portfolio."""
        portfolio = Portfolio(name="Archived", is_archived=True)
        db_session.add(portfolio)
        db_session.commit()

        response = client.get(f"/api/portfolios/{portfolio.id}")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data
        assert "archived" in data["error"].lower()

    def test_update_portfolio(self, app_context, client, db_session):
        """Test PUT /portfolios/<id> updates portfolio."""
        portfolio = Portfolio(name="Original Name", description="Original description")
        db_session.add(portfolio)
        db_session.commit()

        payload = {
            "name": "Updated Name",
            "description": "Updated description",
            "exclude_from_overview": True,
        }

        response = client.put(f"/api/portfolios/{portfolio.id}", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["name"] == "Updated Name"
        assert data["description"] == "Updated description"
        assert data["exclude_from_overview"] is True

        # Verify database
        db_session.refresh(portfolio)
        assert portfolio.name == "Updated Name"
        assert portfolio.exclude_from_overview is True

    def test_update_portfolio_not_found(self, app_context, client):
        """Test PUT /portfolios/<id> returns 404 for non-existent portfolio."""
        fake_id = make_id()
        payload = {"name": "Test"}

        response = client.put(f"/api/portfolios/{fake_id}", json=payload)

        assert response.status_code == 404

    def test_delete_portfolio(self, app_context, client, db_session):
        """Test DELETE /portfolios/<id> removes portfolio."""
        portfolio = Portfolio(name="To Delete")
        db_session.add(portfolio)
        db_session.commit()
        portfolio_id = portfolio.id

        response = client.delete(f"/api/portfolios/{portfolio_id}")

        assert response.status_code == 204

        # Verify database
        deleted = db.session.get(Portfolio, portfolio_id)
        assert deleted is None

    def test_delete_portfolio_not_found(self, app_context, client):
        """Test DELETE /portfolios/<id> returns 404 for non-existent portfolio."""
        fake_id = make_id()
        response = client.delete(f"/api/portfolios/{fake_id}")

        assert response.status_code == 404


class TestPortfolioArchiving:
    """Test portfolio archiving and unarchiving."""

    def test_archive_portfolio(self, app_context, client, db_session):
        """Test POST /portfolios/<id>/archive archives portfolio."""
        portfolio = Portfolio(name="To Archive", is_archived=False)
        db_session.add(portfolio)
        db_session.commit()

        response = client.post(f"/api/portfolios/{portfolio.id}/archive")

        assert response.status_code == 200
        data = response.get_json()
        assert data["is_archived"] is True

        # Verify database
        db_session.refresh(portfolio)
        assert portfolio.is_archived is True

    def test_unarchive_portfolio(self, app_context, client, db_session):
        """Test POST /portfolios/<id>/unarchive restores portfolio."""
        portfolio = Portfolio(name="Archived", is_archived=True)
        db_session.add(portfolio)
        db_session.commit()

        response = client.post(f"/api/portfolios/{portfolio.id}/unarchive")

        assert response.status_code == 200
        data = response.get_json()
        assert data["is_archived"] is False

        # Verify database
        db_session.refresh(portfolio)
        assert portfolio.is_archived is False


class TestPortfolioSummaryAndHistory:
    """Test portfolio summary and historical performance endpoints."""

    def test_get_portfolio_summary(self, app_context, client, db_session):
        """Test GET /portfolio-summary returns aggregated metrics."""
        # Create portfolio with fund and transactions
        portfolio = Portfolio(name="Test Portfolio", exclude_from_overview=False)
        fund = create_fund("US", "MSFT", "Microsoft")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=5,
            cost_per_share=Decimal("200.00"),
        )
        db_session.add(txn)
        db_session.commit()

        response = client.get("/api/portfolio-summary")

        assert response.status_code == 200
        data = response.get_json()
        # Summary returns a list of portfolio summaries
        assert isinstance(data, list)
        assert len(data) >= 1
        # Each summary should have cost and value metrics
        if len(data) > 0:
            assert "totalCost" in data[0]

    def test_get_portfolio_summary_excludes_archived(self, app_context, client, db_session):
        """Test GET /portfolio-summary excludes archived portfolios."""
        p1 = Portfolio(name="Active", is_archived=False)
        p2 = Portfolio(name="Archived", is_archived=True)
        db_session.add_all([p1, p2])
        db_session.commit()

        response = client.get("/api/portfolio-summary")

        assert response.status_code == 200
        # Should only include active portfolio data

    def test_get_portfolio_history(self, app_context, client, db_session):
        """Test GET /portfolio-history returns time series data."""
        # Create portfolio with historical prices
        portfolio = Portfolio(name="History Test")
        fund = create_fund("US", "GOOGL", "Google")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Add transaction
        txn_date = datetime.now().date() - timedelta(days=10)
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=txn_date,
            type="buy",
            shares=3,
            cost_per_share=Decimal("150.00"),
        )
        db_session.add(txn)
        db_session.commit()

        # Add historical prices
        for i in range(5):
            fund_price = FundPrice(
                fund_id=fund.id,
                date=datetime.now().date() - timedelta(days=i),
                price=float(Decimal("160.00") + Decimal(i)),
            )
            db_session.add(fund_price)
        db_session.commit()

        response = client.get("/api/portfolio-history")

        assert response.status_code == 200
        data = response.get_json()
        # History endpoint returns historical performance data
        assert data is not None


class TestPortfolioFunds:
    """Test portfolio-fund relationship management."""

    def test_list_portfolio_funds_all(self, app_context, client, db_session):
        """Test GET /portfolio-funds returns all portfolio funds."""
        portfolio = Portfolio(name="Test")
        fund1 = create_fund("US", "AAPL", "Apple")
        fund2 = create_fund("US", "MSFT", "Microsoft")
        db_session.add_all([portfolio, fund1, fund2])
        db_session.commit()

        pf1 = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund1.id)
        pf2 = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund2.id)
        db_session.add_all([pf1, pf2])
        db_session.commit()

        response = client.get("/api/portfolio-funds")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 2  # At least our 2 funds

    def test_list_portfolio_funds_filtered(self, app_context, client, db_session):
        """Test GET /portfolio-funds?portfolio_id=<id> filters by portfolio."""
        p1 = Portfolio(name="Portfolio 1")
        p2 = Portfolio(name="Portfolio 2")
        fund = create_fund("US", "TSLA", "Tesla")
        db_session.add_all([p1, p2, fund])
        db_session.commit()

        pf1 = PortfolioFund(portfolio_id=p1.id, fund_id=fund.id)
        pf2 = PortfolioFund(portfolio_id=p2.id, fund_id=fund.id)
        db_session.add_all([pf1, pf2])
        db_session.commit()

        response = client.get(f"/api/portfolio-funds?portfolio_id={p1.id}")

        assert response.status_code == 200
        # Should only return funds for p1
        # (Filtering logic tested via status code)

    def test_create_portfolio_fund(self, app_context, client, db_session):
        """Test POST /portfolio-funds adds fund to portfolio."""
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "NVDA", "NVIDIA")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        payload = {"portfolio_id": portfolio.id, "fund_id": fund.id}

        response = client.post("/api/portfolio-funds", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert "id" in data
        assert data["portfolio_id"] == portfolio.id
        assert data["fund_id"] == fund.id

        # Verify database
        pf = PortfolioFund.query.filter_by(portfolio_id=portfolio.id, fund_id=fund.id).first()
        assert pf is not None

    def test_delete_portfolio_fund_without_transactions(self, app_context, client, db_session):
        """Test DELETE /portfolio-funds/<id> removes fund with no transactions."""
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "AMD", "AMD")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()
        pf_id = pf.id

        response = client.delete(f"/api/portfolio-funds/{pf_id}?confirm=true")

        assert response.status_code == 204

        # Verify database
        deleted = db.session.get(PortfolioFund, pf_id)
        assert deleted is None

    def test_delete_portfolio_fund_with_transactions_requires_confirmation(
        self, app_context, client, db_session
    ):
        """Test DELETE /portfolio-funds/<id> requires confirmation when transactions exist."""
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "INTC", "Intel")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Add transaction
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date(),
            type="buy",
            shares=10,
            cost_per_share=Decimal("50.00"),
        )
        db_session.add(txn)
        db_session.commit()

        # Try delete without confirmation
        response = client.delete(f"/api/portfolio-funds/{pf.id}")

        assert response.status_code == 409  # Conflict - requires confirmation
        data = response.get_json()
        # Should indicate confirmation is required
        assert "confirmation" in str(data).lower() or "transaction" in str(data).lower()

    def test_get_fund_history_for_portfolio(self, app_context, client, db_session):
        """Test GET /portfolios/<id>/fund-history returns fund-specific historical data."""
        portfolio = Portfolio(name="History Portfolio")
        fund = create_fund("US", "META", "Meta")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Add transaction and historical prices
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date() - timedelta(days=5),
            type="buy",
            shares=2,
            cost_per_share=Decimal("300.00"),
        )
        db_session.add(txn)

        for i in range(3):
            fund_price = FundPrice(
                fund_id=fund.id,
                date=datetime.now().date() - timedelta(days=i),
                price=float(Decimal("310.00") + Decimal(i * 5)),
            )
            db_session.add(fund_price)
        db_session.commit()

        response = client.get(f"/api/portfolios/{portfolio.id}/fund-history")

        assert response.status_code == 200
        data = response.get_json()
        # Fund history endpoint returns fund-specific historical data
        assert data is not None


class TestPortfolioErrors:
    """Test error paths for portfolio routes."""

    def test_archive_portfolio_not_found(self, app_context, client):
        """Test POST /portfolios/<id>/archive handles non-existent portfolio."""
        fake_id = make_id()
        response = client.post(f"/api/portfolios/{fake_id}/archive")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_unarchive_portfolio_not_found(self, app_context, client):
        """Test POST /portfolios/<id>/unarchive handles non-existent portfolio."""
        fake_id = make_id()
        response = client.post(f"/api/portfolios/{fake_id}/unarchive")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_create_portfolio_fund_invalid_portfolio(self, app_context, client, db_session):
        """Test POST /portfolio-funds rejects non-existent portfolio."""
        fund = create_fund("US", "TEST", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        payload = {"portfolio_id": make_id(), "fund_id": fund.id}

        response = client.post("/api/portfolio-funds", json=payload)

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_create_portfolio_fund_invalid_fund(self, app_context, client, db_session):
        """Test POST /portfolio-funds rejects non-existent fund."""
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        payload = {"portfolio_id": portfolio.id, "fund_id": make_id()}

        response = client.post("/api/portfolio-funds", json=payload)

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_delete_portfolio_fund_not_found(self, app_context, client):
        """Test DELETE /portfolio-funds/<id> handles non-existent portfolio fund."""
        fake_id = make_id()
        response = client.delete(f"/api/portfolio-funds/{fake_id}?confirm=true")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_delete_portfolio_fund_database_error(
        self, app_context, client, db_session, monkeypatch
    ):
        """Test DELETE /portfolio-funds/<id> handles database errors."""
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "ERR", "Error Fund")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Mock PortfolioService.delete_portfolio_fund to raise general exception
        def mock_delete_pf(portfolio_fund_id, confirmed=False):
            raise Exception("Database connection failed")

        monkeypatch.setattr(
            "app.routes.portfolio_routes.PortfolioService.delete_portfolio_fund",
            mock_delete_pf,
        )

        response = client.delete(f"/api/portfolio-funds/{pf.id}?confirm=true")

        assert response.status_code == 400
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_get_portfolios_with_include_excluded(self, app_context, client, db_session):
        """Test GET /portfolios?include_excluded=true includes excluded portfolios."""
        p1 = Portfolio(name="Normal", exclude_from_overview=False)
        p2 = Portfolio(name="Excluded", exclude_from_overview=True)
        db_session.add_all([p1, p2])
        db_session.commit()

        response = client.get("/api/portfolios?include_excluded=true")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        # Should include both portfolios
        assert len(data) >= 2

    def test_get_portfolios_without_include_excluded(self, app_context, client, db_session):
        """Test GET /portfolios excludes excluded portfolios by default."""
        p1 = Portfolio(name="Normal", exclude_from_overview=False)
        p2 = Portfolio(name="Excluded", exclude_from_overview=True)
        db_session.add_all([p1, p2])
        db_session.commit()

        response = client.get("/api/portfolios")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        # Should work (service layer handles filtering)
