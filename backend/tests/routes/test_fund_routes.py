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
        """Test GET /funds returns empty list when no funds exist."""
        response = client.get("/api/funds")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        # May have funds from other tests, just verify it's a list
        assert data is not None

    def test_list_funds(self, app_context, client, db_session):
        """Test GET /funds returns list of funds."""
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
        """Test POST /funds creates a new fund."""
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
        """Test POST /funds rejects duplicate ISIN."""
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
        """Test GET /funds/<fund_id> returns fund details."""
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
        """Test GET /funds/<fund_id> includes latest price."""
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
        """Test GET /funds/<fund_id> handles non-existent fund."""
        fake_id = make_id()
        response = client.get(f"/api/funds/{fake_id}")

        assert response.status_code == 404

    def test_update_fund(self, app_context, client, db_session):
        """Test PUT /funds/<fund_id> updates fund."""
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
        """Test PUT /funds/<fund_id> handles non-existent fund."""
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
        """Test DELETE /funds/<fund_id> removes fund."""
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
        """Test DELETE /funds/<fund_id> rejects deletion of fund in use."""
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
        """Test GET /funds/<fund_id>/check-usage reports fund in use."""
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
        """Test GET /funds/<fund_id>/check-usage reports fund not in use."""
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
        """Test GET /lookup-symbol-info/<symbol> with mocked response."""

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
        """Test GET /lookup-symbol-info/<symbol> handles invalid symbol."""

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
        """Test GET /fund-prices/<fund_id> returns price history."""
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
        """Test POST /fund-prices/<fund_id>/update?type=today updates today's price."""
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
        """Test POST /fund-prices/<fund_id>/update?type=historical updates historical prices."""
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
        """Test POST /funds/update-all-prices updates all funds with symbols."""
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
        """Test POST /funds rejects missing name."""
        payload = {
            "isin": make_isin("US"),
            "symbol": "TEST",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code in [400, 500]

    def test_create_fund_missing_isin(self, app_context, client):
        """Test POST /funds rejects missing ISIN."""
        payload = {
            "name": "Test Fund",
            "symbol": "TEST",
            "currency": "USD",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code in [400, 500]

    def test_create_fund_missing_currency(self, app_context, client):
        """Test POST /funds rejects missing currency."""
        payload = {
            "name": "Test Fund",
            "isin": make_isin("US"),
            "symbol": "TEST",
            "exchange": "NYSE",
        }

        response = client.post("/api/funds", json=payload)

        assert response.status_code in [400, 500]

    def test_create_fund_database_error(self, app_context, client, monkeypatch):
        """Test POST /funds handles database errors."""

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
        """Test PUT /funds/<fund_id> rejects missing ISIN."""
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
        """Test PUT /funds/<fund_id> handles database errors."""
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
        """Test DELETE /funds/<fund_id> handles non-existent fund."""
        fake_id = make_id()
        response = client.delete(f"/api/funds/{fake_id}")

        assert response.status_code == 404
        data = response.get_json()
        assert "error" in data

    def test_delete_fund_database_error(self, app_context, client, db_session, monkeypatch):
        """Test DELETE /funds/<fund_id> handles database errors."""
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
        """Test GET /funds/<fund_id>/check-usage handles non-existent fund."""

        # Mock FundService.check_fund_usage to raise ValueError for not found
        def mock_check_usage(fund_id):
            raise ValueError("Fund not found")

        monkeypatch.setattr("app.routes.fund_routes.FundService.check_fund_usage", mock_check_usage)

        fake_id = make_id()
        response = client.get(f"/api/funds/{fake_id}/check-usage")

        assert response.status_code == 500  # Routes wraps ValueError in 500

    def test_check_fund_usage_database_error(self, app_context, client, monkeypatch):
        """Test GET /funds/<fund_id>/check-usage handles database errors."""

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
        """Test GET /funds/<fund_id> handles database errors."""

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
        """Test GET /funds handles database errors."""
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
        """Test GET /lookup-symbol-info/<symbol> handles external API failures."""

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
        """Test GET /lookup-symbol-info/<symbol>?force_refresh=true handles failures."""

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
        """Test GET /lookup-symbol-info with empty symbol."""

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
        """Test GET /lookup-symbol-info handles cache errors gracefully."""

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
        """Test GET /fund-prices/<fund_id> handles non-existent fund."""
        from werkzeug.exceptions import NotFound

        # Mock FundService.get_fund to raise 404
        def mock_get_fund(fund_id):
            raise NotFound("Fund not found")

        monkeypatch.setattr("app.routes.fund_routes.FundService.get_fund", mock_get_fund)

        fake_id = make_id()
        response = client.get(f"/api/fund-prices/{fake_id}")

        assert response.status_code == 404

    def test_get_fund_prices_database_error(self, app_context, client, db_session, monkeypatch):
        """Test GET /fund-prices/<fund_id> handles database errors."""
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
        """Test POST /fund-prices/<fund_id>/update?type=today handles API failures."""
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
        """Test POST /fund-prices/<fund_id>/update?type=historical handles API failures."""
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
        """Test POST /funds/update-all-prices rejects requests without API key."""
        # Ensure INTERNAL_API_KEY is set
        monkeypatch.setenv("INTERNAL_API_KEY", "test_key")

        # Make request without authentication headers
        response = client.post("/api/funds/update-all-prices")

        assert response.status_code == 401
        data = response.get_json()
        assert "error" in data or "message" in data

    def test_update_all_prices_invalid_time_token(self, app_context, client, monkeypatch):
        """Test POST /funds/update-all-prices rejects invalid time token."""
        api_key = "test_api_key_12345"
        monkeypatch.setenv("INTERNAL_API_KEY", api_key)

        # Make request with API key but invalid time token
        headers = {"X-API-Key": api_key, "X-Time-Token": "invalid_token"}

        response = client.post("/api/funds/update-all-prices", headers=headers)

        assert response.status_code == 401

    def test_update_all_prices_database_error(self, app_context, client, monkeypatch):
        """Test POST /funds/update-all-prices handles database errors."""
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
