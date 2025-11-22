"""
Integration tests for dividend routes (dividend_routes.py).

Tests all Dividend API endpoints:
- POST /dividends - Create dividend
- GET /dividends/fund/<fund_id> - Get dividends by fund
- GET /dividends/portfolio/<portfolio_id> - Get dividends by portfolio
- PUT /dividends/<dividend_id> - Update dividend
- DELETE /dividends/<dividend_id> - Delete dividend
"""

from datetime import datetime, timedelta
from decimal import Decimal

from app.models import Dividend, Fund, Portfolio, PortfolioFund, Transaction, db
from tests.test_helpers import make_id, make_isin, make_symbol


def create_fund(
    isin_prefix="US",
    symbol_prefix="TEST",
    name="Test Fund",
    currency="USD",
    exchange="NYSE",
    dividend_type="CASH",
):
    """Helper to create a Fund with all required fields."""
    from app.models import DividendType

    return Fund(
        isin=make_isin(isin_prefix),
        symbol=make_symbol(symbol_prefix),
        name=name,
        currency=currency,
        exchange=exchange,
        dividend_type=DividendType[dividend_type],
    )


class TestDividendCreate:
    """Test dividend creation."""

    def test_create_dividend(self, app_context, client, db_session):
        """
        Verify dividend creation with automatic share ownership calculation.

        WHY: Users need accurate dividend records that automatically track how many shares
        they owned at the record date. This prevents manual calculation errors and ensures
        dividend income is correctly attributed to the portfolio position at that point in time.
        """
        portfolio = Portfolio(name="Test Portfolio")
        fund = create_fund("US", "VTI", "Vanguard Total Stock Market ETF", dividend_type="CASH")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create a transaction so shares_owned can be calculated
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date() - timedelta(days=30),
            type="buy",
            shares=100,
            cost_per_share=Decimal("50.00"),
        )
        db_session.add(txn)
        db_session.commit()

        payload = {
            "fund_id": fund.id,
            "portfolio_fund_id": pf.id,
            "record_date": datetime.now().date().isoformat(),
            "ex_dividend_date": (datetime.now().date() - timedelta(days=1)).isoformat(),
            "dividend_per_share": 0.75,
        }

        response = client.post("/api/dividends", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert "id" in data
        assert data["shares_owned"] == 100
        assert data["dividend_per_share"] == 0.75

        # Verify database
        dividend = db.session.get(Dividend, data["id"])
        assert dividend is not None
        assert dividend.shares_owned == 100

    def test_create_dividend_calculates_total(self, app_context, client, db_session):
        """
        Verify automatic calculation of total dividend amount from shares and per-share rate.

        WHY: Accurate total dividend calculation is critical for portfolio income tracking and
        tax reporting. Automating this calculation (shares * rate) eliminates user input errors
        and ensures consistency across dividend records for performance analysis.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "VYM", "Vanguard High Dividend Yield ETF")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create a transaction so shares_owned can be calculated
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date() - timedelta(days=30),
            type="buy",
            shares=50,
            cost_per_share=Decimal("60.00"),
        )
        db_session.add(txn)
        db_session.commit()

        payload = {
            "fund_id": fund.id,
            "portfolio_fund_id": pf.id,
            "record_date": datetime.now().date().isoformat(),
            "ex_dividend_date": datetime.now().date().isoformat(),
            "dividend_per_share": 1.50,
        }

        response = client.post("/api/dividends", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        # Total should be 50 * 1.50 = 75.00
        assert data["total_amount"] == 75.00


class TestDividendRetrieve:
    """Test dividend retrieval endpoints."""

    def test_get_dividends_by_fund(self, app_context, client, db_session):
        """
        Verify retrieval of all dividend records for a specific fund across time.

        WHY: Users need to view historical dividend payments to analyze a fund's dividend
        consistency, growth trends, and income reliability. This supports investment decisions
        and helps users identify changes in dividend policy or payout patterns.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "SCHD", "Schwab US Dividend Equity ETF")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create dividends
        div1 = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=100,
            dividend_per_share=Decimal("0.50"),
            total_amount=Decimal("50.00"),
        )
        div2 = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date() - timedelta(days=90),
            ex_dividend_date=datetime.now().date() - timedelta(days=91),
            shares_owned=100,
            dividend_per_share=Decimal("0.48"),
            total_amount=Decimal("48.00"),
        )
        db_session.add_all([div1, div2])
        db_session.commit()

        response = client.get(f"/api/dividends/fund/{fund.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 2
        assert all("id" in d for d in data)

    def test_get_dividends_by_fund_not_found(self, app_context, client):
        """
        Verify graceful handling when requesting dividends for non-existent fund.

        WHY: Prevents application crashes when users access stale URLs or deleted funds.
        Proper error handling ensures a stable user experience even when navigating to
        invalid fund references from bookmarks or external links.
        """
        fake_id = make_id()
        response = client.get(f"/api/dividends/fund/{fake_id}")

        # Should return empty list or error
        assert response.status_code in [200, 404]

    def test_get_dividends_by_portfolio(self, app_context, client, db_session):
        """
        Verify retrieval of all dividend records across multiple funds in a portfolio.

        WHY: Users need consolidated portfolio-level dividend views to track total income,
        plan cash flow for living expenses, and analyze portfolio income diversification.
        This is essential for income-focused investment strategies and retirement planning.
        """
        portfolio = Portfolio(name="Dividend Portfolio")
        fund1 = create_fund("US", "JEPI", "JPMorgan Equity Premium Income ETF")
        fund2 = create_fund("US", "DIVO", "Amplify CWP Enhanced Dividend Income ETF")
        db_session.add_all([portfolio, fund1, fund2])
        db_session.commit()

        pf1 = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund1.id)
        pf2 = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund2.id)
        db_session.add_all([pf1, pf2])
        db_session.commit()

        # Create dividends for both funds in same portfolio
        div1 = Dividend(
            fund_id=fund1.id,
            portfolio_fund_id=pf1.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=50,
            dividend_per_share=Decimal("0.60"),
            total_amount=Decimal("30.00"),
        )
        div2 = Dividend(
            fund_id=fund2.id,
            portfolio_fund_id=pf2.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=40,
            dividend_per_share=Decimal("0.55"),
            total_amount=Decimal("22.00"),
        )
        db_session.add_all([div1, div2])
        db_session.commit()

        response = client.get(f"/api/dividends/portfolio/{portfolio.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 2

    def test_get_dividends_by_portfolio_not_found(self, app_context, client):
        """
        Verify graceful handling when requesting dividends for non-existent portfolio.

        WHY: Prevents application errors when portfolios are deleted or URLs are invalid.
        Robust error handling maintains system stability and provides clear feedback when
        users attempt to access deleted or moved portfolio data.
        """
        fake_id = make_id()
        response = client.get(f"/api/dividends/portfolio/{fake_id}")

        # Should return empty list or error
        assert response.status_code in [200, 404]


class TestDividendUpdateDelete:
    """Test dividend update and deletion."""

    def test_update_dividend(self, app_context, client, db_session):
        """
        Verify updating dividend records when companies adjust dividend amounts.

        WHY: Fund companies occasionally correct dividend announcements or users may need to
        fix data entry errors. Accurate updates ensure portfolio income calculations remain
        correct for tax reporting and performance tracking without requiring delete/recreate.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "DGRO", "iShares Core Dividend Growth ETF")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create transaction for shares_owned calculation
        txn = Transaction(
            portfolio_fund_id=pf.id,
            date=datetime.now().date() - timedelta(days=30),
            type="buy",
            shares=75,
            cost_per_share=Decimal("45.00"),
        )
        db_session.add(txn)
        db_session.commit()

        div = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=75,
            dividend_per_share=Decimal("0.40"),
            total_amount=Decimal("30.00"),
        )
        db_session.add(div)
        db_session.commit()

        payload = {
            "fund_id": fund.id,
            "portfolio_fund_id": pf.id,
            "record_date": datetime.now().date().isoformat(),
            "ex_dividend_date": (datetime.now().date() - timedelta(days=1)).isoformat(),
            "dividend_per_share": 0.42,  # Changed
        }

        response = client.put(f"/api/dividends/{div.id}", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        # Verify the dividend_per_share was updated
        assert data["dividend_per_share"] == 0.42

        # Verify database
        db_session.refresh(div)
        assert float(div.dividend_per_share) == 0.42

    def test_update_dividend_not_found(self, app_context, client):
        """
        Verify proper error response when attempting to update a non-existent dividend.

        WHY: Prevents silent failures or database corruption when users try to update deleted
        dividends. Clear error responses help users understand that the dividend no longer
        exists and prevent confusion about whether their update was applied.
        """
        fake_id = make_id()
        payload = {
            "fund_id": make_id(),
            "portfolio_fund_id": make_id(),
            "record_date": datetime.now().date().isoformat(),
            "ex_dividend_date": datetime.now().date().isoformat(),
            "shares_owned": 100,
            "dividend_per_share": 0.50,
        }

        response = client.put(f"/api/dividends/{fake_id}", json=payload)

        assert response.status_code in [400, 404]

    def test_delete_dividend(self, app_context, client, db_session):
        """
        Verify complete removal of dividend records from the database.

        WHY: Users need to remove erroneous or duplicate dividend entries to maintain accurate
        portfolio income records. Proper deletion ensures portfolio performance calculations
        and tax reports reflect only actual dividend payments received.
        """
        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "SDY", "SPDR S&P Dividend ETF")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        div = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=60,
            dividend_per_share=Decimal("0.35"),
            total_amount=Decimal("21.00"),
        )
        db_session.add(div)
        db_session.commit()
        div_id = div.id

        response = client.delete(f"/api/dividends/{div_id}")

        assert response.status_code == 204

        # Verify database
        deleted = db.session.get(Dividend, div_id)
        assert deleted is None

    def test_delete_dividend_not_found(self, app_context, client):
        """
        Verify appropriate error handling when deleting a non-existent dividend.

        WHY: Prevents confusion when users attempt to delete already-removed dividends or
        invalid IDs. Proper error responses distinguish between successful deletions and
        failed attempts, avoiding duplicate delete operations and race conditions.
        """
        fake_id = make_id()
        response = client.delete(f"/api/dividends/{fake_id}")

        # API returns error for dividend not found (404 or 500)
        assert response.status_code in [404, 500]


class TestDividendErrors:
    """Test error paths for dividend routes."""

    def test_create_dividend_service_error(self, client, db_session):
        """
        Verify graceful error handling when dividend creation fails due to service errors.

        WHY: Database connection issues or constraint violations during dividend creation
        should return clear errors rather than crashes. This ensures users receive actionable
        feedback and prevents partial data corruption during network or database failures.
        """
        from unittest.mock import patch

        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "TEST", "Test Fund")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        with patch("app.routes.dividend_routes.DividendService.create_dividend") as mock_create:
            mock_create.side_effect = Exception("Database error")

            payload = {
                "fund_id": fund.id,
                "portfolio_fund_id": pf.id,
                "record_date": datetime.now().date().isoformat(),
                "ex_dividend_date": datetime.now().date().isoformat(),
                "dividend_per_share": 0.50,
            }

            response = client.post("/api/dividends", json=payload)

            assert response.status_code == 400
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_get_fund_dividends_service_error(self, client):
        """
        Verify proper error responses when database queries fail during fund dividend retrieval.

        WHY: Database connection losses or query timeouts shouldn't crash the application.
        Users need informative error messages to understand temporary service issues versus
        data problems, enabling appropriate retry strategies or support contact.
        """
        from unittest.mock import patch

        with patch("app.routes.dividend_routes.DividendService.get_fund_dividends") as mock_get:
            mock_get.side_effect = Exception("Database query failed")

            fake_id = make_id()
            response = client.get(f"/api/dividends/fund/{fake_id}")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_get_portfolio_dividends_service_error(self, client):
        """
        Verify proper error responses when database queries fail during portfolio dividend retrieval.

        WHY: Portfolio dividend aggregation involves complex joins that may timeout or fail.
        Proper error handling prevents blank screens and provides users with clear feedback
        about service availability for their income tracking features.
        """
        from unittest.mock import patch

        with patch(
            "app.routes.dividend_routes.DividendService.get_portfolio_dividends"
        ) as mock_get:
            mock_get.side_effect = Exception("Database query failed")

            fake_id = make_id()
            response = client.get(f"/api/dividends/portfolio/{fake_id}")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_update_dividend_value_error(self, client, db_session):
        """
        Verify proper handling of validation errors during dividend updates.

        WHY: Invalid data like negative dividend amounts or invalid dates must be rejected
        with clear validation messages. This prevents data corruption and guides users to
        correct their input, maintaining data integrity for financial calculations.
        """
        from unittest.mock import patch

        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "TEST", "Test Fund")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        div = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=100,
            dividend_per_share=Decimal("0.50"),
            total_amount=Decimal("50.00"),
        )
        db_session.add(div)
        db_session.commit()

        with patch("app.routes.dividend_routes.DividendService.update_dividend") as mock_update:
            mock_update.side_effect = ValueError("Dividend not found")

            payload = {
                "fund_id": fund.id,
                "portfolio_fund_id": pf.id,
                "record_date": datetime.now().date().isoformat(),
                "ex_dividend_date": datetime.now().date().isoformat(),
                "dividend_per_share": 0.55,
            }

            response = client.put(f"/api/dividends/{div.id}", json=payload)

            assert response.status_code == 400
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_update_dividend_general_error(self, client, db_session):
        """
        Verify proper handling of unexpected errors during dividend update operations.

        WHY: Unforeseen errors like database constraint violations or connection failures
        during updates must be caught and logged without exposing system internals. This
        protects security while enabling debugging of production issues.
        """
        from unittest.mock import patch

        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "ERR", "Error Fund")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        div = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=50,
            dividend_per_share=Decimal("0.75"),
            total_amount=Decimal("37.50"),
        )
        db_session.add(div)
        db_session.commit()

        with patch("app.routes.dividend_routes.DividendService.update_dividend") as mock_update:
            mock_update.side_effect = Exception("Unexpected database error")

            payload = {
                "fund_id": fund.id,
                "portfolio_fund_id": pf.id,
                "record_date": datetime.now().date().isoformat(),
                "ex_dividend_date": datetime.now().date().isoformat(),
                "dividend_per_share": 0.80,
            }

            response = client.put(f"/api/dividends/{div.id}", json=payload)

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_delete_dividend_value_error(self, client):
        """
        Verify proper error handling when dividend deletion encounters ValueError.

        WHY: ValueErrors during deletion (like invalid dividend IDs) must return appropriate
        400 status codes with clear messages. This distinguishes user errors from system
        failures and provides actionable feedback for correcting the request.
        """
        from unittest.mock import patch

        with patch("app.routes.dividend_routes.DividendService.get_dividend") as mock_get:
            mock_get.side_effect = ValueError("Dividend not found")

            fake_id = make_id()
            response = client.delete(f"/api/dividends/{fake_id}")

            assert response.status_code == 400
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_delete_dividend_general_error(self, client, db_session):
        """
        Verify proper handling of unexpected errors during dividend deletion.

        WHY: Database constraint violations or foreign key conflicts during deletion must
        be handled gracefully with 500 status codes. This prevents cascading failures and
        provides operators with error information for investigating data integrity issues.
        """
        from unittest.mock import patch

        portfolio = Portfolio(name="Test")
        fund = create_fund("US", "DEL", "Delete Test")
        db_session.add_all([portfolio, fund])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        div = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=25,
            dividend_per_share=Decimal("1.00"),
            total_amount=Decimal("25.00"),
        )
        db_session.add(div)
        db_session.commit()

        # Mock get_dividend to succeed but delete_dividend to fail
        with patch("app.routes.dividend_routes.DividendService.delete_dividend") as mock_delete:
            mock_delete.side_effect = Exception("Database constraint violation")

            response = client.delete(f"/api/dividends/{div.id}")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data
