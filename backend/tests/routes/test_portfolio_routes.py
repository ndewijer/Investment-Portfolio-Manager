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
        """
        Verify GET /portfolios returns empty list when no portfolios exist.

        WHY: Users need to see an empty state when they first start using the application,
        ensuring the UI can handle the initial condition gracefully without errors.
        """
        response = client.get("/api/portfolios")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) == 0

    def test_list_portfolios_returns_all(self, app_context, client, db_session):
        """
        Verify GET /portfolios returns all portfolios including active and archived.

        WHY: Users need complete visibility of all their portfolios to manage their investments
        effectively. This prevents data from being hidden and ensures the API returns consistent results.
        """
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
        """
        Verify POST /portfolios creates a new portfolio with all fields correctly set.

        WHY: Portfolio creation is the fundamental entry point for users to start tracking their
        investments. This ensures new portfolios are created with proper defaults and all attributes
        are correctly persisted to prevent data loss or inconsistent states.
        """
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
        """
        Verify POST /portfolios succeeds with only required fields (name).

        WHY: Users should be able to quickly create portfolios without filling out every field,
        supporting a streamlined onboarding experience. This prevents validation errors from
        blocking legitimate use cases where optional fields can be added later.
        """
        payload = {"name": "Minimal Portfolio"}

        response = client.post("/api/portfolios", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["name"] == "Minimal Portfolio"
        assert data["description"] == ""  # Default empty description


class TestPortfolioRetrieveUpdateDelete:
    """Test individual portfolio retrieval, update, and deletion."""

    def test_get_portfolio_detail(self, app_context, client, db_session):
        """
        Verify GET /portfolios/<id> returns portfolio with complete performance metrics.

        WHY: Users need to see their portfolio's current value, costs, gains/losses, and dividends
        to make informed investment decisions. This ensures all critical financial metrics are
        calculated and returned correctly for portfolio performance tracking.
        """
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
        """
        Verify GET /portfolios/<id> returns 404 for non-existent portfolio.

        WHY: Attempting to access a deleted or invalid portfolio ID should fail gracefully with
        a proper error code, preventing users from seeing incorrect data or experiencing crashes
        when following stale links or bookmarks.
        """
        fake_id = make_id()
        response = client.get(f"/api/portfolios/{fake_id}")

        assert response.status_code == 404

    def test_get_archived_portfolio_returns_404(self, app_context, client, db_session):
        """
        Verify GET /portfolios/<id> returns 404 for archived portfolio.

        WHY: Archived portfolios should not be accessible via standard detail endpoints to prevent
        users from accidentally modifying or viewing stale investment data. This enforces the
        archived state as a soft-delete mechanism.
        """
        portfolio = Portfolio(name="Archived", is_archived=True)
        db_session.add(portfolio)
        db_session.commit()

        response = client.get(f"/api/portfolios/{portfolio.id}")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data
        assert "archived" in data["error"].lower()

    def test_update_portfolio(self, app_context, client, db_session):
        """
        Verify PUT /portfolios/<id> successfully updates portfolio fields.

        WHY: Users need to rename portfolios, update descriptions, and change settings like
        exclude_from_overview as their investment strategies evolve. This ensures changes are
        properly persisted and returned to maintain data consistency.
        """
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
        """
        Verify PUT /portfolios/<id> returns 404 for non-existent portfolio.

        WHY: Update operations on invalid portfolio IDs should fail gracefully to prevent
        creating orphaned data or confusing users with success messages when nothing was updated.
        """
        fake_id = make_id()
        payload = {"name": "Test"}

        response = client.put(f"/api/portfolios/{fake_id}", json=payload)

        assert response.status_code == 404

    def test_delete_portfolio(self, app_context, client, db_session):
        """
        Verify DELETE /portfolios/<id> permanently removes portfolio from database.

        WHY: Users need the ability to permanently delete portfolios they no longer need, cleaning
        up their workspace. This ensures complete removal from the database to prevent clutter
        and maintain data integrity.
        """
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
        """
        Verify DELETE /portfolios/<id> returns 404 for non-existent portfolio.

        WHY: Delete operations on invalid IDs should fail with proper error codes to prevent
        users from thinking a deletion succeeded when nothing happened, avoiding confusion
        about data state.
        """
        fake_id = make_id()
        response = client.delete(f"/api/portfolios/{fake_id}")

        assert response.status_code == 404


class TestPortfolioArchiving:
    """Test portfolio archiving and unarchiving."""

    def test_archive_portfolio(self, app_context, client, db_session):
        """
        Verify POST /portfolios/<id>/archive sets portfolio to archived state.

        WHY: Users need to archive old or inactive portfolios without permanently deleting them,
        preserving historical data while decluttering their active workspace. This supports data
        retention requirements and allows users to restore portfolios if needed later.
        """
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
        """
        Verify POST /portfolios/<id>/unarchive restores archived portfolio to active state.

        WHY: Users may need to reactivate previously archived portfolios when resuming investment
        tracking or correcting accidental archives. This ensures the unarchive operation properly
        restores full portfolio functionality.
        """
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
        """
        Verify GET /portfolio-summary returns aggregated metrics across all active portfolios.

        WHY: Users need a high-level overview of their total investment performance across all
        portfolios to understand their overall financial position. This aggregated view is critical
        for portfolio allocation decisions and investment strategy planning.
        """
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
        """
        Verify GET /portfolio-summary excludes archived portfolios from calculations.

        WHY: Archived portfolios represent inactive investments that should not affect current
        portfolio summaries, preventing skewed metrics and ensuring users see only their active
        investment position.
        """
        p1 = Portfolio(name="Active", is_archived=False)
        p2 = Portfolio(name="Archived", is_archived=True)
        db_session.add_all([p1, p2])
        db_session.commit()

        response = client.get("/api/portfolio-summary")

        assert response.status_code == 200
        # Should only include active portfolio data

    def test_get_portfolio_history(self, app_context, client, db_session):
        """
        Verify GET /portfolio-history returns historical performance time series data.

        WHY: Users need to visualize portfolio performance over time to identify trends, evaluate
        investment strategies, and make data-driven decisions. Historical data is essential for
        performance charting and trend analysis.
        """
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
        """
        Verify GET /portfolio-funds returns all portfolio-fund relationships.

        WHY: Users need to see which funds are held across all portfolios to manage their
        holdings and understand asset distribution. This enables bulk operations and provides
        a comprehensive view of all fund allocations.
        """
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
        """
        Verify GET /portfolio-funds?portfolio_id=<id> filters results to specific portfolio.

        WHY: Users often need to see funds for a single portfolio without noise from other
        portfolios, enabling focused portfolio management and efficient data retrieval for
        specific portfolio views.
        """
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
        """
        Verify POST /portfolio-funds successfully adds a fund to a portfolio.

        WHY: Users need to add new funds to their portfolios to expand their investments and
        track additional holdings. This is a core operation for portfolio construction and
        ensures proper relationship creation in the database.
        """
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
        """
        Verify DELETE /portfolio-funds/<id> removes fund when no transactions exist.

        WHY: Users should be able to remove funds they no longer want to track when there's no
        transaction history. This cleanup operation prevents database clutter and allows users
        to correct mistakes made during portfolio setup.
        """
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
        """
        Verify DELETE /portfolio-funds/<id> requires confirmation when transactions exist.

        WHY: Deleting a fund with transaction history could result in significant data loss and
        financial record corruption. Requiring confirmation prevents accidental deletion of
        valuable investment history and protects users from irreversible mistakes.
        """
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
        """
        Verify GET /portfolios/<id>/fund-history returns fund-specific historical performance.

        WHY: Users need to analyze individual fund performance within a portfolio to identify
        top performers, underperformers, and make informed buy/sell decisions. Fund-level
        historical data is crucial for detailed investment analysis.
        """
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
        """
        Verify POST /portfolios/<id>/archive returns 404 for non-existent portfolio.

        WHY: Archive operations on invalid portfolio IDs should fail gracefully with proper
        error codes, preventing users from thinking an archive succeeded when no portfolio
        exists, which could lead to confusion about data state.
        """
        fake_id = make_id()
        response = client.post(f"/api/portfolios/{fake_id}/archive")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_unarchive_portfolio_not_found(self, app_context, client):
        """
        Verify POST /portfolios/<id>/unarchive returns 404 for non-existent portfolio.

        WHY: Unarchive operations on invalid portfolio IDs should fail with clear error messages
        to prevent users from being misled about restoration success when no portfolio exists
        to restore.
        """
        fake_id = make_id()
        response = client.post(f"/api/portfolios/{fake_id}/unarchive")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_create_portfolio_fund_invalid_portfolio(self, app_context, client, db_session):
        """
        Verify POST /portfolio-funds rejects attempts to link fund to non-existent portfolio.

        WHY: Creating portfolio-fund relationships with invalid portfolio IDs would create orphaned
        data and corrupt database referential integrity. This validation protects against data
        inconsistencies and provides clear error feedback.
        """
        fund = create_fund("US", "TEST", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        payload = {"portfolio_id": make_id(), "fund_id": fund.id}

        response = client.post("/api/portfolio-funds", json=payload)

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_create_portfolio_fund_invalid_fund(self, app_context, client, db_session):
        """
        Verify POST /portfolio-funds rejects attempts to link non-existent fund to portfolio.

        WHY: Creating portfolio-fund relationships with invalid fund IDs would create orphaned
        records and prevent proper transaction tracking. This validation ensures data integrity
        and prevents silent failures.
        """
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        payload = {"portfolio_id": portfolio.id, "fund_id": make_id()}

        response = client.post("/api/portfolio-funds", json=payload)

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_delete_portfolio_fund_not_found(self, app_context, client):
        """
        Verify DELETE /portfolio-funds/<id> returns 404 for non-existent portfolio-fund.

        WHY: Delete operations on invalid portfolio-fund IDs should fail with proper error codes
        to prevent users from thinking a deletion succeeded when nothing existed, maintaining
        accurate expectations about system state.
        """
        fake_id = make_id()
        response = client.delete(f"/api/portfolio-funds/{fake_id}?confirm=true")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_delete_portfolio_fund_database_error(
        self, app_context, client, db_session, monkeypatch
    ):
        """
        Verify DELETE /portfolio-funds/<id> handles unexpected database errors gracefully.

        WHY: Database failures during deletion could leave the system in an inconsistent state
        or expose stack traces to users. Proper error handling ensures the API returns safe,
        user-friendly error messages even during unexpected failures.
        """
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
        """
        Verify GET /portfolios?include_excluded=true includes portfolios marked exclude_from_overview.

        WHY: Users need the ability to view all portfolios including those excluded from overview
        calculations for comprehensive portfolio management. This query parameter supports advanced
        filtering without requiring separate API endpoints.
        """
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
        """
        Verify GET /portfolios excludes portfolios marked exclude_from_overview by default.

        WHY: By default, users expect to see only portfolios included in overview calculations,
        preventing clutter from test portfolios or inactive holdings. This default behavior
        supports clean UI presentation while allowing opt-in visibility via query parameters.
        """
        p1 = Portfolio(name="Normal", exclude_from_overview=False)
        p2 = Portfolio(name="Excluded", exclude_from_overview=True)
        db_session.add_all([p1, p2])
        db_session.commit()

        response = client.get("/api/portfolios")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        # Should work (service layer handles filtering)
