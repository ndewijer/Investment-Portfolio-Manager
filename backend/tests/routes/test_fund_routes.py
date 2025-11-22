"""
Integration tests for fund routes (fund_routes.py).

Tests all Fund API endpoints:
- GET /funds - List all funds ✅
- POST /funds - Create fund ✅
- GET /funds/<fund_id> - Get fund detail (SKIPPED - requires route refactoring)
- PUT /funds/<fund_id> - Update fund ✅
- DELETE /funds/<fund_id> - Delete fund ✅
- GET /funds/<fund_id>/check-usage - Check fund usage ✅
- GET /lookup-symbol-info/<symbol> - Lookup symbol info ✅
- GET /fund-prices/<fund_id> - Get fund prices (SKIPPED - requires route refactoring)
- POST /fund-prices/<fund_id>/update - Update fund prices ✅
- POST /funds/update-all-prices - Update all fund prices ✅

Test Summary: 15 passing, 4 skipped

NOTE: Some tests are skipped because they test endpoints with direct model queries
(Fund.query.get_or_404(), FundPrice.query.filter_by()) which have session scoping
issues in the test environment. These endpoints are documented in
todo/ROUTE_REFACTORING_REMEDIATION_PLAN.md and will be fixed during route refactoring.
"""

from datetime import datetime
from decimal import Decimal

from app.models import Fund, FundPrice, Portfolio, PortfolioFund, Transaction, db
from tests.test_helpers import make_id, make_isin, make_symbol


def create_fund(
    isin_prefix="US",
    symbol_prefix="TEST",
    name="Test Fund",
    currency="USD",
    exchange="NYSE",
):
    """Helper to create a Fund with all required fields."""
    return Fund(
        isin=make_isin(isin_prefix),
        symbol=make_symbol(symbol_prefix),
        name=name,
        currency=currency,
        exchange=exchange,
    )


class TestFundListAndCreate:
    """Test fund listing and creation endpoints."""

    def test_list_funds_empty(self, app_context, client):
        """
        Verify GET /funds returns a valid empty list when no funds exist.

        WHY: The frontend relies on this endpoint to populate the fund selection
        dropdown. If it doesn't return a valid list structure (even when empty),
        the UI will crash preventing users from accessing any portfolio functionality.
        """
        response = client.get("/api/funds")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        # May have funds from other tests, just verify it's a list
        assert data is not None

    def test_list_funds(self, app_context, client, db_session):
        """
        Verify GET /funds returns complete fund data including IDs and names.

        WHY: Users need to see all available funds to make investment decisions.
        Missing fund data or incorrect structure would prevent fund selection and
        portfolio creation, blocking core application functionality.
        """
        fund1 = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        fund2 = create_fund("US", "VOO", "Vanguard S&P 500 ETF")
        db_session.add_all([fund1, fund2])
        db_session.commit()

        response = client.get("/api/funds")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 2
        assert all("id" in f for f in data)
        assert all("name" in f for f in data)

    def test_create_fund(self, app_context, client, db_session):
        """
        Verify POST /funds successfully creates a new fund with all required fields.

        WHY: Fund creation is the starting point for all investment tracking. Users
        must be able to add new funds (stocks, ETFs) to track their investments.
        Failure here prevents users from building their portfolios.
        """
        payload = {
            "name": "Test Fund",
            "isin": make_isin("US"),
            "symbol": "TEST",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert "id" in data
        assert data["name"] == "Test Fund"
        assert data["symbol"] == "TEST"

        # Verify database
        fund = db.session.get(Fund, data["id"])
        assert fund is not None
        assert fund.name == "Test Fund"

    def test_create_fund_duplicate_isin(self, app_context, client, db_session):
        """
        Verify POST /funds prevents duplicate ISINs to maintain data integrity.

        WHY: Each ISIN uniquely identifies a security globally. Allowing duplicates
        would create data inconsistencies, confuse portfolio calculations, and could
        lead to incorrect valuations or regulatory compliance issues.
        """
        isin = make_isin("US")
        fund = create_fund("US", "VTI", "Existing Fund")
        fund.isin = isin
        db_session.add(fund)
        db_session.commit()

        payload = {
            "name": "Duplicate Fund",
            "isin": isin,
            "symbol": "DUP",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        assert "error" in data or "message" in data


class TestFundRetrieveUpdateDelete:
    """Test fund retrieval, update, and deletion endpoints."""

    def test_get_fund_detail(self, app_context, client, db_session):
        """
        Verify GET /funds/<fund_id> returns complete fund information.

        WHY: Users need detailed fund information to make informed investment decisions
        and verify they're tracking the correct security. Missing data could lead to
        investment mistakes or loss of confidence in the application.
        """
        fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        db_session.add(fund)
        db_session.commit()

        response = client.get(f"/api/funds/{fund.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert data["id"] == fund.id
        assert data["name"] == "Vanguard Total Stock Market ETF"
        assert data["symbol"] == fund.symbol
        assert "latest_price" in data

    def test_get_fund_with_latest_price(self, app_context, client, db_session):
        """
        Verify GET /funds/<fund_id> includes the most recent price data.

        WHY: Current price is essential for portfolio valuation and performance tracking.
        Users expect to see up-to-date valuations; missing or stale prices would make
        the portfolio manager useless for real-time investment decisions.
        """
        fund = create_fund("US", "VOO", "Vanguard S&P 500 ETF")
        db_session.add(fund)
        db_session.commit()

        # Add price data
        price = FundPrice(
            fund_id=fund.id,
            date=datetime.now().date(),
            price=float(Decimal("450.00")),
        )
        db_session.add(price)
        db_session.commit()

        response = client.get(f"/api/funds/{fund.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert data["latest_price"] is not None
        assert data["latest_price"] == 450.00

    def test_get_fund_not_found(self, app_context, client):
        """
        Verify GET /funds/<fund_id> returns 404 for non-existent funds.

        WHY: Proper error handling prevents the UI from crashing when users access
        invalid fund IDs (deleted funds, bookmarked URLs, stale data). Clear 404
        responses allow graceful error messages instead of application failures.
        """
        fake_id = make_id()
        response = client.get(f"/api/funds/{fake_id}")

        assert response.status_code == 404

    def test_update_fund(self, app_context, client, db_session):
        """
        Verify PUT /funds/<fund_id> successfully updates fund attributes.

        WHY: Fund information changes over time (ticker symbol changes, exchange
        listings, corrections to initial data). Users need to update funds to
        maintain accurate records and ensure price updates continue working.
        """
        fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        db_session.add(fund)
        db_session.commit()

        payload = {
            "name": "Updated Fund Name",
            "isin": fund.isin,
            "symbol": "NEWVTI",
            "currency": "USD",
            "exchange": "NASDAQ",
        }

        response = client.put(f"/api/funds/{fund.id}", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["name"] == "Updated Fund Name"
        assert data["symbol"] == "NEWVTI"
        assert data["exchange"] == "NASDAQ"

        # Verify database
        db_session.refresh(fund)
        assert fund.name == "Updated Fund Name"
        assert fund.symbol == "NEWVTI"

    def test_update_fund_not_found(self, app_context, client):
        """
        Verify PUT /funds/<fund_id> returns error for non-existent funds.

        WHY: Attempting to update deleted or invalid funds should fail gracefully
        with clear feedback. This prevents data corruption and provides users with
        actionable error messages instead of silent failures.
        """
        fake_id = make_id()
        payload = {
            "name": "Updated Name",
            "isin": make_isin("US"),
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.put(f"/api/funds/{fake_id}", json=payload)

        assert response.status_code in [400, 404]

    def test_delete_fund(self, app_context, client, db_session):
        """
        Verify DELETE /funds/<fund_id> successfully removes unused funds.

        WHY: Users need to clean up incorrect entries or funds no longer tracked.
        Inability to delete would lead to cluttered fund lists making it harder
        to find relevant funds and potentially causing confusion.
        """
        fund = create_fund("US", "TEST", "Test Fund to Delete")
        db_session.add(fund)
        db_session.commit()
        fund_id = fund.id

        response = client.delete(f"/api/funds/{fund_id}")

        assert response.status_code == 200

        # Verify database
        deleted = db.session.get(Fund, fund_id)
        assert deleted is None

    def test_delete_fund_in_use(self, app_context, client, db_session):
        """
        Verify DELETE /funds/<fund_id> prevents deletion of funds in active portfolios.

        WHY: Deleting a fund that has transactions would orphan transaction data,
        corrupt portfolio valuations, and cause data integrity violations. This
        protection prevents catastrophic data loss and broken portfolio calculations.
        """
        fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        # Add fund to portfolio
        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        response = client.delete(f"/api/funds/{fund.id}")

        assert response.status_code == 409  # Conflict
        data = response.get_json()
        assert "message" in data or "error" in data

        # Verify fund still exists
        fund_exists = db.session.get(Fund, fund.id)
        assert fund_exists is not None


class TestFundUsage:
    """Test fund usage checking endpoint."""

    def test_check_fund_usage_in_use(self, app_context, client, db_session):
        """
        Verify GET /funds/<fund_id>/check-usage correctly identifies funds with transactions.

        WHY: Before deletion, users must know which portfolios use a fund to avoid
        data loss. This check prevents accidental deletion of funds with transaction
        history and shows users where the fund is being used.
        """
        fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        # Add fund to portfolio with transaction
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

        response = client.get(f"/api/funds/{fund.id}/check-usage")

        assert response.status_code == 200
        data = response.get_json()
        assert data["in_use"] is True
        assert "portfolios" in data
        assert len(data["portfolios"]) >= 1

    def test_check_fund_usage_not_in_use(self, app_context, client, db_session):
        """
        Verify GET /funds/<fund_id>/check-usage correctly identifies unused funds.

        WHY: Users should know when funds are safe to delete. Incorrectly reporting
        a fund as in-use would prevent legitimate deletions, while false negatives
        would allow catastrophic data loss.
        """
        fund = create_fund("US", "TEST", "Test Fund Not In Use")
        db_session.add(fund)
        db_session.commit()

        response = client.get(f"/api/funds/{fund.id}/check-usage")

        assert response.status_code == 200
        data = response.get_json()
        assert data["in_use"] is False
        # portfolios key might not be present if not in use
        assert "portfolios" not in data or data["portfolios"] == []


class TestSymbolLookup:
    """Test symbol lookup endpoint."""

    def test_lookup_symbol_info_mock(self, app_context, client, monkeypatch):
        """
        Verify GET /lookup-symbol-info/<symbol> returns symbol metadata from lookup service.

        WHY: Users need to quickly find and verify fund information when creating new
        funds. Auto-populated data (name, currency, exchange) reduces manual entry
        errors and speeds up the fund creation workflow.
        """

        # Mock SymbolLookupService to avoid external API calls
        def mock_get_symbol_info(symbol, force_refresh=False):
            if symbol == "VTI":
                return {
                    "symbol": "VTI",
                    "name": "Vanguard Total Stock Market ETF",
                    "currency": "USD",
                    "exchange": "PCX",
                }
            return None

        from app.services import symbol_lookup_service

        monkeypatch.setattr(
            symbol_lookup_service.SymbolLookupService,
            "get_symbol_info",
            staticmethod(mock_get_symbol_info),
        )

        response = client.get("/api/lookup-symbol-info/VTI")

        assert response.status_code == 200
        data = response.get_json()
        assert data["symbol"] == "VTI"
        assert "name" in data

    def test_lookup_symbol_not_found(self, app_context, client, monkeypatch):
        """
        Verify GET /lookup-symbol-info/<symbol> returns 404 for invalid symbols.

        WHY: Users may mistype symbols or try non-existent tickers. Proper 404 handling
        allows the UI to show clear "symbol not found" messages instead of crashing,
        guiding users to correct their input.
        """

        # Mock to return None for invalid symbol
        def mock_get_symbol_info(symbol, force_refresh=False):
            return None

        from app.services import symbol_lookup_service

        monkeypatch.setattr(
            symbol_lookup_service.SymbolLookupService,
            "get_symbol_info",
            staticmethod(mock_get_symbol_info),
        )

        response = client.get("/api/lookup-symbol-info/INVALID")

        assert response.status_code == 404


class TestFundPrices:
    """Test fund price endpoints."""

    def test_get_fund_prices(self, app_context, client, db_session):
        """
        Verify GET /fund-prices/<fund_id> returns complete historical price data.

        WHY: Users need historical prices for performance analysis, charting, and
        calculating returns over time. Missing price history breaks portfolio analytics
        and prevents users from tracking investment performance.
        """
        fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        db_session.add(fund)
        db_session.commit()

        # Add price data
        price1 = FundPrice(
            fund_id=fund.id,
            date=datetime.now().date(),
            price=float(Decimal("250.00")),
        )
        price2 = FundPrice(
            fund_id=fund.id,
            date=datetime(2024, 1, 1).date(),
            price=float(Decimal("240.00")),
        )
        db_session.add_all([price1, price2])
        db_session.commit()

        response = client.get(f"/api/fund-prices/{fund.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 2
        assert all("date" in p for p in data)
        assert all("price" in p for p in data)

    def test_update_todays_price(self, app_context, client, db_session, monkeypatch):
        """
        Verify POST /fund-prices/<fund_id>/update?type=today fetches current price.

        WHY: Users need real-time portfolio valuations throughout the trading day.
        Today's price update enables accurate current value calculations without
        waiting for end-of-day historical updates.
        """
        fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        db_session.add(fund)
        db_session.commit()

        # Mock TodayPriceService to avoid external API calls
        def mock_update_todays_price(fund_id):
            return {"message": "Price updated successfully", "prices_added": 1}, 200

        from app.services import price_update_service

        monkeypatch.setattr(
            price_update_service.TodayPriceService,
            "update_todays_price",
            staticmethod(mock_update_todays_price),
        )

        response = client.post(f"/api/fund-prices/{fund.id}/update?type=today")

        assert response.status_code == 200
        data = response.get_json()
        assert "message" in data

    def test_update_historical_prices(self, app_context, client, db_session, monkeypatch):
        """
        Verify POST /fund-prices/<fund_id>/update?type=historical backfills price history.

        WHY: Historical prices are needed for performance calculations, charts, and
        accurate portfolio valuations at past dates. This backfill is essential when
        adding new funds or filling gaps in price data.
        """
        fund = create_fund("US", "VOO", "Vanguard S&P 500 ETF")
        db_session.add(fund)
        db_session.commit()

        # Mock HistoricalPriceService to avoid external API calls
        def mock_update_historical_prices(fund_id):
            return {"message": "Historical prices updated", "prices_added": 5}, 200

        from app.services import price_update_service

        monkeypatch.setattr(
            price_update_service.HistoricalPriceService,
            "update_historical_prices",
            staticmethod(mock_update_historical_prices),
        )

        response = client.post(f"/api/fund-prices/{fund.id}/update?type=historical")

        assert response.status_code == 200
        data = response.get_json()
        assert "message" in data


class TestUpdateAllPrices:
    """Test update all fund prices endpoint."""

    def test_update_all_fund_prices(self, app_context, client, db_session, monkeypatch):
        """
        Verify POST /funds/update-all-prices bulk updates all funds via cron job.

        WHY: Automated batch updates keep all portfolio valuations current without
        manual intervention. This scheduled job is critical for maintaining accurate
        daily portfolio values across all users and funds.
        """
        import hashlib
        from datetime import UTC, datetime

        # Create funds with symbols
        fund1 = create_fund("US", "VTI", "Vanguard Total Stock Market ETF")
        fund2 = create_fund("US", "VOO", "Vanguard S&P 500 ETF")
        db_session.add_all([fund1, fund2])
        db_session.commit()

        # Mock HistoricalPriceService to avoid external API calls
        def mock_update_historical_prices(fund_id):
            return {"message": "Prices updated", "prices_added": 3}, 200

        from app.services import price_update_service

        monkeypatch.setattr(
            price_update_service.HistoricalPriceService,
            "update_historical_prices",
            staticmethod(mock_update_historical_prices),
        )

        # Set up API key authentication
        api_key = "test_api_key_12345"
        monkeypatch.setenv("INTERNAL_API_KEY", api_key)

        # Generate time-based token (same logic as decorator)
        current_hour = datetime.now(UTC).strftime("%Y-%m-%d-%H")
        time_token = hashlib.sha256(f"{api_key}{current_hour}".encode()).hexdigest()

        # Make request with authentication headers
        headers = {"X-API-Key": api_key, "X-Time-Token": time_token}

        response = client.post("/api/funds/update-all-prices", headers=headers)

        assert response.status_code == 200
        data = response.get_json()
        assert "updated_funds" in data
        assert "errors" in data
        assert data["success"] is True


class TestFundCRUDErrors:
    """Test error paths for fund CRUD operations."""

    def test_create_fund_missing_name(self, app_context, client):
        """
        Verify POST /funds rejects requests without fund name.

        WHY: Fund name is required for display and user identification. Without
        validation, missing names would create unusable fund entries that break
        UI displays and confuse users.
        """
        payload = {
            "isin": make_isin("US"),
            "symbol": "TEST",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code in [400, 500]

    def test_create_fund_missing_isin(self, app_context, client):
        """
        Verify POST /funds rejects requests without ISIN.

        WHY: ISIN is a required unique identifier for securities. Missing ISIN
        validation would allow incomplete fund records that can't be properly
        tracked or reconciled with external systems.
        """
        payload = {
            "name": "Test Fund",
            "symbol": "TEST",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code in [400, 500]

    def test_create_fund_missing_currency(self, app_context, client):
        """
        Verify POST /funds rejects requests without currency.

        WHY: Currency is essential for accurate valuation calculations. Without it,
        multi-currency portfolios would have incorrect valuations and users couldn't
        properly track returns in their base currency.
        """
        payload = {
            "name": "Test Fund",
            "isin": make_isin("US"),
            "symbol": "TEST",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code in [400, 500]

    def test_create_fund_database_error(self, app_context, client, monkeypatch):
        """
        Verify POST /funds returns 500 with error message on database failures.

        WHY: Database outages or connection issues should return informative errors
        rather than crashing. Proper error handling enables graceful degradation and
        helps with debugging production issues.
        """

        # Mock FundService.create_fund to raise exception
        def mock_create_fund(data, symbol_info=None):
            raise Exception("Database connection failed")

        monkeypatch.setattr("app.routes.fund_routes.FundService.create_fund", mock_create_fund)

        payload = {
            "name": "Test Fund",
            "isin": make_isin("US"),
            "symbol": "TEST",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_update_fund_missing_isin(self, app_context, client, db_session):
        """
        Verify PUT /funds/<fund_id> rejects updates without ISIN.

        WHY: ISIN must remain consistent for fund identity. Allowing ISIN to be
        removed would break fund tracking, price updates, and regulatory compliance.
        Validation prevents data corruption through incomplete updates.
        """
        fund = create_fund("US", "VTI", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        payload = {
            "name": "Updated Name",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.put(f"/api/funds/{fund.id}", json=payload)

        assert response.status_code in [400, 500]

    def test_update_fund_database_error(self, app_context, client, db_session, monkeypatch):
        """
        Verify PUT /funds/<fund_id> returns 400 with error message on database failures.

        WHY: Database errors during updates should be caught and reported clearly
        to prevent partial updates and silent failures. Users need to know if their
        changes didn't save so they can retry.
        """
        fund = create_fund("US", "VTI", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        # Mock FundService.update_fund to raise exception
        def mock_update_fund(fund_id, data):
            raise Exception("Database error")

        monkeypatch.setattr("app.routes.fund_routes.FundService.update_fund", mock_update_fund)

        payload = {
            "name": "Updated Name",
            "isin": make_isin("US"),
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.put(f"/api/funds/{fund.id}", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_delete_fund_not_found(self, app_context, client):
        """
        Verify DELETE /funds/<fund_id> returns 404 for non-existent funds.

        WHY: Attempting to delete already-deleted or invalid funds should return
        clear 404 errors. This prevents confusion and helps users understand the
        current state of their data.
        """
        fake_id = make_id()
        response = client.delete(f"/api/funds/{fake_id}")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_delete_fund_database_error(self, app_context, client, db_session, monkeypatch):
        """
        Verify DELETE /funds/<fund_id> returns 500 on database failures.

        WHY: Database errors during deletion should be reported, not silently fail.
        Users need to know if deletion didn't complete so they can verify data
        integrity and retry if needed.
        """
        fund = create_fund("US", "TEST", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        # Mock FundService.delete_fund to raise exception (not ValueError)
        def mock_delete_fund(fund_id):
            raise Exception("Database error")

        monkeypatch.setattr("app.routes.fund_routes.FundService.delete_fund", mock_delete_fund)

        response = client.delete(f"/api/funds/{fund.id}")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_check_fund_usage_fund_not_found(self, app_context, client, monkeypatch):
        """
        Verify GET /funds/<fund_id>/check-usage handles non-existent funds.

        WHY: Users may check usage of deleted funds (via stale bookmarks or race
        conditions). Proper error handling prevents crashes and provides clear
        feedback about the fund's non-existence.
        """

        # Mock FundService.check_fund_usage to raise ValueError for not found
        def mock_check_usage(fund_id):
            raise ValueError("Fund not found")

        monkeypatch.setattr("app.routes.fund_routes.FundService.check_fund_usage", mock_check_usage)

        fake_id = make_id()
        response = client.get(f"/api/funds/{fake_id}/check-usage")

        assert response.status_code == 500  # Routes wraps ValueError in 500

    def test_check_fund_usage_database_error(self, app_context, client, monkeypatch):
        """
        Verify GET /funds/<fund_id>/check-usage returns 500 on database failures.

        WHY: Usage checks are critical before deletion to prevent data loss. Database
        errors must be surfaced clearly so users know the check failed and don't
        proceed with potentially destructive deletions.
        """

        # Mock FundService.check_fund_usage to raise exception
        def mock_check_usage(fund_id):
            raise Exception("Database error")

        monkeypatch.setattr("app.routes.fund_routes.FundService.check_fund_usage", mock_check_usage)

        fake_id = make_id()
        response = client.get(f"/api/funds/{fake_id}/check-usage")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_get_fund_database_error(self, app_context, client, monkeypatch):
        """
        Verify GET /funds/<fund_id> returns 500 with error message on database failures.

        WHY: Database connectivity issues should be reported clearly rather than
        causing application crashes. Users need to know when data retrieval fails
        so they can retry or contact support.
        """

        # Mock FundService.get_fund to raise non-HTTP exception
        def mock_get_fund(fund_id):
            raise Exception("Database connection failed")

        monkeypatch.setattr("app.routes.fund_routes.FundService.get_fund", mock_get_fund)

        fake_id = make_id()
        response = client.get(f"/api/funds/{fake_id}")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_get_funds_database_error(self, app_context, client):
        """
        Verify GET /funds returns 500 with error message on database failures.

        WHY: The fund list is a critical entry point to the application. Database
        failures must return proper errors instead of crashing, allowing the UI to
        show maintenance messages or retry logic.
        """
        from unittest.mock import patch

        # Mock Fund.query.all to raise exception
        with patch("app.routes.fund_routes.Fund.query") as mock_query:
            mock_query.all.side_effect = Exception("Database query failed")

            response = client.get("/api/funds")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data


class TestSymbolLookupErrors:
    """Test error paths for symbol lookup endpoint."""

    def test_lookup_symbol_external_api_failure(self, app_context, client, monkeypatch):
        """
        Verify GET /lookup-symbol-info/<symbol> returns 500 on external API failures.

        WHY: Third-party APIs (yfinance) may be down or rate-limited. Proper error
        handling allows the UI to show appropriate error messages and lets users
        enter fund data manually instead of blocking fund creation entirely.
        """

        # Mock SymbolLookupService to raise exception
        def mock_get_symbol_info(symbol, force_refresh=False):
            raise Exception("yfinance API connection failed")

        from app.services import symbol_lookup_service

        monkeypatch.setattr(
            symbol_lookup_service.SymbolLookupService,
            "get_symbol_info",
            staticmethod(mock_get_symbol_info),
        )

        response = client.get("/api/lookup-symbol-info/VTI")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_lookup_symbol_force_refresh_failure(self, app_context, client, monkeypatch):
        """
        Verify GET /lookup-symbol-info with force_refresh handles API failures.

        WHY: Force refresh bypasses cache and hits external APIs directly. Rate
        limits or API outages must be handled gracefully with clear error messages
        rather than crashing the fund creation workflow.
        """

        # Mock SymbolLookupService to raise exception on force refresh
        def mock_get_symbol_info(symbol, force_refresh=False):
            if force_refresh:
                raise Exception("API rate limit exceeded")
            return None

        from app.services import symbol_lookup_service

        monkeypatch.setattr(
            symbol_lookup_service.SymbolLookupService,
            "get_symbol_info",
            staticmethod(mock_get_symbol_info),
        )

        response = client.get("/api/lookup-symbol-info/VTI?force_refresh=true")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_lookup_symbol_empty_symbol(self, app_context, client, monkeypatch):
        """
        Verify GET /lookup-symbol-info handles empty/missing symbol parameter.

        WHY: URL routing edge cases (trailing slashes, empty params) should be
        handled gracefully. Prevents server errors from user input mistakes or
        client-side bugs that send malformed requests.
        """

        # Mock to handle empty symbol
        def mock_get_symbol_info(symbol, force_refresh=False):
            if not symbol or symbol.strip() == "":
                return None
            return None

        from app.services import symbol_lookup_service

        monkeypatch.setattr(
            symbol_lookup_service.SymbolLookupService,
            "get_symbol_info",
            staticmethod(mock_get_symbol_info),
        )

        response = client.get("/api/lookup-symbol-info/")

        # May get 404 from Flask routing or 404 from our handler
        assert response.status_code in [404, 308]  # 308 is redirect

    def test_lookup_symbol_cache_error(self, app_context, client, monkeypatch):
        """
        Verify GET /lookup-symbol-info returns 500 on cache read/write failures.

        WHY: Redis or file cache failures shouldn't break symbol lookup entirely.
        Proper error handling ensures users get feedback about cache issues rather
        than mysterious failures during fund creation.
        """

        # Mock SymbolLookupService to raise cache-related exception
        def mock_get_symbol_info(symbol, force_refresh=False):
            raise Exception("Cache read/write error")

        from app.services import symbol_lookup_service

        monkeypatch.setattr(
            symbol_lookup_service.SymbolLookupService,
            "get_symbol_info",
            staticmethod(mock_get_symbol_info),
        )

        response = client.get("/api/lookup-symbol-info/AAPL")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data


class TestPriceUpdateErrors:
    """Test error paths for fund price update endpoints."""

    def test_get_fund_prices_fund_not_found(self, app_context, client, monkeypatch):
        """
        Verify GET /fund-prices/<fund_id> returns 404 for non-existent funds.

        WHY: Users may request price history for deleted funds. Clear 404 responses
        prevent UI crashes and inform users the fund no longer exists, allowing
        graceful error handling in the frontend.
        """
        from werkzeug.exceptions import NotFound

        # Mock FundService.get_fund to raise 404
        def mock_get_fund(fund_id):
            raise NotFound("Fund not found")

        monkeypatch.setattr("app.routes.fund_routes.FundService.get_fund", mock_get_fund)

        fake_id = make_id()
        response = client.get(f"/api/fund-prices/{fake_id}")

        assert response.status_code == 404

    def test_get_fund_prices_database_error(self, app_context, client, db_session, monkeypatch):
        """
        Verify GET /fund-prices/<fund_id> returns 500 on database query failures.

        WHY: Price history queries may fail due to database issues. Proper error
        handling prevents crashes and allows users to retry or use cached data,
        maintaining application stability during outages.
        """
        fund = create_fund("US", "VTI", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        # Mock FundService.get_fund_price_history to raise exception
        def mock_get_price_history(fund_id):
            raise Exception("Database query failed")

        monkeypatch.setattr(
            "app.routes.fund_routes.FundService.get_fund_price_history",
            mock_get_price_history,
        )

        response = client.get(f"/api/fund-prices/{fund.id}")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_update_fund_prices_api_failure_today(
        self, app_context, client, db_session, monkeypatch
    ):
        """
        Verify POST /fund-prices/<fund_id>/update?type=today returns 500 on API failures.

        WHY: Real-time price APIs may fail due to rate limits, network issues, or
        service outages. Proper error handling allows users to retry or fall back
        to cached prices without breaking portfolio valuation entirely.
        """
        fund = create_fund("US", "VTI", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        # Mock TodayPriceService to raise exception
        def mock_update_todays_price(fund_id):
            raise Exception("External API failure")

        from app.services import price_update_service

        monkeypatch.setattr(
            price_update_service.TodayPriceService,
            "update_todays_price",
            staticmethod(mock_update_todays_price),
        )

        response = client.post(f"/api/fund-prices/{fund.id}/update?type=today")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_update_fund_prices_api_failure_historical(
        self, app_context, client, db_session, monkeypatch
    ):
        """
        Verify POST /fund-prices update?type=historical returns 500 on API failures.

        WHY: Historical price backfills may fail due to API limits or connectivity
        issues. Clear error responses allow users to retry later and prevent silent
        failures that would leave price history incomplete.
        """
        fund = create_fund("US", "VOO", "Test Fund")
        db_session.add(fund)
        db_session.commit()

        # Mock HistoricalPriceService to raise exception
        def mock_update_historical_prices(fund_id):
            raise Exception("yfinance connection timeout")

        from app.services import price_update_service

        monkeypatch.setattr(
            price_update_service.HistoricalPriceService,
            "update_historical_prices",
            staticmethod(mock_update_historical_prices),
        )

        response = client.post(f"/api/fund-prices/{fund.id}/update?type=historical")

        assert response.status_code == 500
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_update_all_prices_missing_api_key(self, app_context, client, monkeypatch):
        """
        Verify POST /funds/update-all-prices requires valid API key authentication.

        WHY: This endpoint is designed for cron jobs only and could be abused to
        trigger expensive API calls. API key authentication prevents unauthorized
        access and protects against DoS attacks on external price APIs.
        """
        # Ensure INTERNAL_API_KEY is set
        monkeypatch.setenv("INTERNAL_API_KEY", "test_key")

        # Make request without authentication headers
        response = client.post("/api/funds/update-all-prices")

        assert response.status_code == 401
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_update_all_prices_invalid_time_token(self, app_context, client, monkeypatch):
        """
        Verify POST /funds/update-all-prices validates time-based token freshness.

        WHY: Time tokens prevent replay attacks and stolen credentials from being
        reused. This security layer ensures that even if an API key leaks, it can
        only be used within a limited time window.
        """
        api_key = "test_api_key_12345"
        monkeypatch.setenv("INTERNAL_API_KEY", api_key)

        # Make request with API key but invalid time token
        headers = {"X-API-Key": api_key, "X-Time-Token": "invalid_token"}

        response = client.post("/api/funds/update-all-prices", headers=headers)

        assert response.status_code == 401

    def test_update_all_prices_database_error(self, app_context, client, monkeypatch):
        """
        Verify POST /funds/update-all-prices returns 500 on database failures.

        WHY: The batch price update job must handle database errors gracefully to
        avoid silent failures in scheduled tasks. Clear error responses enable
        monitoring and alerting when nightly price updates fail.
        """
        import hashlib
        from datetime import UTC, datetime
        from unittest.mock import patch

        # Set up API key authentication
        api_key = "test_api_key_12345"
        monkeypatch.setenv("INTERNAL_API_KEY", api_key)

        # Generate time-based token
        current_hour = datetime.now(UTC).strftime("%Y-%m-%d-%H")
        time_token = hashlib.sha256(f"{api_key}{current_hour}".encode()).hexdigest()

        # Mock Fund.query.filter to raise exception
        with patch("app.routes.fund_routes.Fund.query") as mock_query:
            mock_query.filter.return_value.all.side_effect = Exception("Database connection failed")

            headers = {"X-API-Key": api_key, "X-Time-Token": time_token}

            response = client.post("/api/funds/update-all-prices", headers=headers)

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data
            assert data["success"] is False
