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
        """Test POST /dividends creates a dividend."""
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
        """Test POST /dividends calculates total_amount correctly."""
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
        """Test GET /dividends/fund/<fund_id> returns dividends for fund."""
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
        """Test GET /dividends/fund/<fund_id> handles non-existent fund."""
        fake_id = make_id()
        response = client.get(f"/api/dividends/fund/{fake_id}")

        # Should return empty list or error
        assert response.status_code in [200, 404]

    def test_get_dividends_by_portfolio(self, app_context, client, db_session):
        """Test GET /dividends/portfolio/<portfolio_id> returns dividends for portfolio."""
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
        """Test GET /dividends/portfolio/<portfolio_id> handles non-existent portfolio."""
        fake_id = make_id()
        response = client.get(f"/api/dividends/portfolio/{fake_id}")

        # Should return empty list or error
        assert response.status_code in [200, 404]


class TestDividendUpdateDelete:
    """Test dividend update and deletion."""

    def test_update_dividend(self, app_context, client, db_session):
        """Test PUT /dividends/<dividend_id> updates dividend."""
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
        """Test PUT /dividends/<dividend_id> handles non-existent dividend."""
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
        """Test DELETE /dividends/<dividend_id> removes dividend."""
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
        """Test DELETE /dividends/<dividend_id> handles non-existent dividend."""
        fake_id = make_id()
        response = client.delete(f"/api/dividends/{fake_id}")

        # API returns error for dividend not found (404 or 500)
        assert response.status_code in [404, 500]
