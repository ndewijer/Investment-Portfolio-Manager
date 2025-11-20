"""
Integration tests for IBKR routes (ibkr_routes.py).

Tests IBKR API endpoints:
- GET /ibkr/config - Get config status ✅
- POST /ibkr/config - Save config ✅
- DELETE /ibkr/config - Delete config ✅
- POST /ibkr/config/test - Test connection (SKIPPED - requires external API)
- POST /ibkr/import - Import transactions (SKIPPED - requires external API)
- GET /ibkr/inbox - Get inbox transactions ✅
- GET /ibkr/inbox/count - Get inbox count ✅
- GET /ibkr/inbox/<transaction_id> - Get specific transaction ✅
- POST /ibkr/inbox/<transaction_id>/ignore - Ignore transaction ✅
- DELETE /ibkr/inbox/<transaction_id> - Delete transaction ✅
- GET /ibkr/portfolios - Get portfolios for allocation ✅
- GET /ibkr/inbox/<transaction_id>/eligible-portfolios - Get eligible portfolios ✅
- POST /ibkr/inbox/<transaction_id>/allocate - Allocate transaction ✅
- GET /ibkr/dividends/pending - Get pending dividends ✅
- POST /ibkr/inbox/<transaction_id>/match-dividend - Match dividend ✅
- POST /ibkr/inbox/<transaction_id>/unallocate - Unallocate transaction ✅
- GET /ibkr/inbox/<transaction_id>/allocations - Get transaction allocations ✅
- PUT /ibkr/inbox/<transaction_id>/allocations - Update allocations ✅
- POST /ibkr/inbox/bulk-allocate - Bulk allocate ✅

Test Summary: 8 passing, 12 skipped

NOTE: Tests for endpoints requiring external IBKR Flex API calls are skipped (2 tests).
Tests for endpoints using Query.get_or_404() are skipped due to session scoping issues (3 tests).
Tests for endpoints with unresolved business logic requirements are skipped (7 tests).
"""

from datetime import datetime, timedelta
from decimal import Decimal

import pytest
from app.models import (
    Dividend,
    Fund,
    IBKRConfig,
    IBKRTransaction,
    IBKRTransactionAllocation,
    Portfolio,
    PortfolioFund,
    ReinvestmentStatus,
    db,
)
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


class TestIBKRConfig:
    """Test IBKR configuration endpoints."""

    def test_get_config_status_no_config(self, app_context, client):
        """Test GET /ibkr/config returns status when no config exists."""
        response = client.get("/api/ibkr/config")

        assert response.status_code == 200
        data = response.get_json()
        assert "configured" in data
        # May be True or False depending on database state

    def test_save_config(self, app_context, client, db_session):
        """
        Test POST /ibkr/config saves configuration.

        Validates:
        - Configuration is saved to database
        - Response indicates success
        - Token is encrypted (not stored in plain text)
        - Query ID is stored correctly
        """
        payload = {
            "flex_token": "test_token_123",
            "flex_query_id": "query_456",
            "auto_import_enabled": False,
        }

        response = client.post("/api/ibkr/config", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["success"] is True
        assert "message" in data

        # Verify database
        config = IBKRConfig.query.first()
        assert config is not None
        assert config.flex_query_id == "query_456"
        assert config.auto_import_enabled is False
        # Token should be encrypted (not plain text)
        assert config.flex_token != "test_token_123"

    def test_get_config_status_with_config(self, app_context, client, db_session):
        """Test GET /ibkr/config returns status with existing config."""
        config = IBKRConfig(
            flex_token="test_token",
            flex_query_id="test_query",
            auto_import_enabled=False,
        )
        db_session.add(config)
        db_session.commit()

        response = client.get("/api/ibkr/config")

        assert response.status_code == 200
        data = response.get_json()
        assert data["configured"] is True
        assert "flex_token" not in data  # Token should not be exposed

    def test_delete_config(self, app_context, client, db_session):
        """Test DELETE /ibkr/config removes configuration."""
        config = IBKRConfig(
            flex_token="test_token",
            flex_query_id="test_query",
            auto_import_enabled=False,
        )
        db_session.add(config)
        db_session.commit()

        response = client.delete("/api/ibkr/config")

        assert response.status_code == 200

        # Verify database
        configs = IBKRConfig.query.all()
        assert len(configs) == 0


class TestIBKRImport:
    """Test IBKR import endpoint."""

    @pytest.mark.skip(
        reason="Endpoint requires external IBKR Flex API calls. "
        "Testing requires mocking complex external API interactions."
    )
    def test_import_transactions(self, app_context, client, db_session):
        """Test POST /ibkr/import imports transactions."""
        # Would require mocking IBKRFlexService.fetch_and_process_flex_query
        pass


class TestIBKRInbox:
    """Test IBKR inbox endpoints."""

    def test_get_inbox_empty(self, app_context, client):
        """Test GET /ibkr/inbox returns empty list when no transactions."""
        response = client.get("/api/ibkr/inbox")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)

    def test_get_inbox_with_transactions(self, app_context, client, db_session):
        """Test GET /ibkr/inbox returns inbox transactions."""
        # Create IBKR transaction
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        response = client.get("/api/ibkr/inbox")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 1

    def test_get_inbox_count(self, app_context, client, db_session):
        """Test GET /ibkr/inbox/count returns count."""
        # Create IBKR transactions
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="MSFT",
            description="Microsoft",
            transaction_type="buy",
            quantity=5,
            price=float(Decimal("300.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        response = client.get("/api/ibkr/inbox/count")

        assert response.status_code == 200
        data = response.get_json()
        assert "count" in data
        assert data["count"] >= 2

    def test_get_inbox_transaction(self, app_context, client, db_session):
        """Test GET /ibkr/inbox/<transaction_id> returns specific transaction."""
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        response = client.get(f"/api/ibkr/inbox/{txn.id}")

        assert response.status_code == 200
        data = response.get_json()
        assert data["id"] == txn.id
        assert data["symbol"] == "AAPL"

    def test_ignore_transaction(self, app_context, client, db_session):
        """Test POST /ibkr/inbox/<transaction_id>/ignore marks transaction as ignored."""
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        response = client.post(f"/api/ibkr/inbox/{txn.id}/ignore")

        assert response.status_code == 200

        # Verify database
        db_session.refresh(txn)
        assert txn.status == "ignored"

    def test_delete_transaction(self, app_context, client, db_session):
        """Test DELETE /ibkr/inbox/<transaction_id> removes transaction."""
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()
        txn_id = txn.id

        response = client.delete(f"/api/ibkr/inbox/{txn_id}")

        assert response.status_code == 200

        # Verify database
        deleted = db.session.get(IBKRTransaction, txn_id)
        assert deleted is None


class TestIBKRAllocation:
    """Test IBKR transaction allocation endpoints."""

    def test_get_portfolios_for_allocation(self, app_context, client, db_session):
        """Test GET /ibkr/portfolios returns portfolios."""
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add(portfolio)
        db_session.commit()

        response = client.get("/api/ibkr/portfolios")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)
        assert len(data) >= 1

    def test_get_eligible_portfolios(self, app_context, client, db_session):
        """
        Test GET /ibkr/inbox/<transaction_id>/eligible-portfolios returns eligible portfolios.

        Validates:
        - Endpoint returns portfolios that contain the fund matching the transaction
        - Response includes match information (fund found, matched by ISIN/symbol)
        - Response format is correct
        """
        # Create fund and portfolio
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create IBKR transaction
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        response = client.get(f"/api/ibkr/inbox/{txn.id}/eligible-portfolios")

        if response.status_code != 200:
            print(f"Status: {response.status_code}")
            print(f"Response: {response.get_json()}")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, dict)
        assert "portfolios" in data
        assert "match_info" in data

    # @pytest.mark.skip(
    #     reason="Endpoint returns 400 error. Requires investigation of route's validation "
    #     "and business logic."
    # )
    def test_allocate_transaction(self, app_context, client, db_session):
        """Test POST /ibkr/inbox/<transaction_id>/allocate allocates transaction."""
        # Create fund and portfolio
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create IBKR transaction
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        payload = {"allocations": [{"portfolio_id": portfolio.id, "percentage": 100.0}]}

        response = client.post(f"/api/ibkr/inbox/{txn.id}/allocate", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["success"] is True
        assert "created_transactions" in data

        # Verify allocation created
        allocations = IBKRTransactionAllocation.query.filter_by(ibkr_transaction_id=txn.id).all()
        assert len(allocations) >= 1

    def test_get_pending_dividends(self, app_context, client, db_session):
        """Test GET /ibkr/dividends/pending returns pending dividends."""
        # Create fund, portfolio, and dividend
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        dividend = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=100,
            dividend_per_share=Decimal("0.50"),
            total_amount=Decimal("50.00"),
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        db_session.add(dividend)
        db_session.commit()

        response = client.get("/api/ibkr/dividends/pending")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)

    def test_match_dividend(self, app_context, client, db_session):
        """Test POST /ibkr/inbox/<transaction_id>/match-dividend matches dividend."""
        # Create fund, portfolio, and dividend
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        dividend = Dividend(
            fund_id=fund.id,
            portfolio_fund_id=pf.id,
            record_date=datetime.now().date(),
            ex_dividend_date=datetime.now().date() - timedelta(days=1),
            shares_owned=100,
            dividend_per_share=Decimal("0.50"),
            total_amount=Decimal("0"),  # Will be set by matching
            reinvestment_status=ReinvestmentStatus.PENDING,
        )
        db_session.add(dividend)
        db_session.commit()

        # Create IBKR transaction
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            isin=fund.isin,
            description="Apple Inc - Dividend Reinvestment",
            transaction_type="dividend",
            total_amount=float(Decimal("50.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        payload = {"dividend_ids": [dividend.id]}

        response = client.post(f"/api/ibkr/inbox/{txn.id}/match-dividend", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["success"] is True
        assert data["updated_dividends"] == 1

    def test_unallocate_transaction(self, app_context, client, db_session):
        """Test POST /ibkr/inbox/<transaction_id>/unallocate removes allocations."""
        # Create fund, portfolio, and allocation
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create IBKR transaction
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="processed",
        )
        db_session.add(txn)
        db_session.commit()

        # Create allocation
        allocation = IBKRTransactionAllocation(
            ibkr_transaction_id=txn.id,
            portfolio_id=portfolio.id,
            allocation_percentage=100.0,
            allocated_amount=float(Decimal("1500.00")),
            allocated_shares=10.0,
        )
        db_session.add(allocation)
        db_session.commit()

        response = client.post(f"/api/ibkr/inbox/{txn.id}/unallocate")

        assert response.status_code == 200

        # Verify allocations removed
        allocations = IBKRTransactionAllocation.query.filter_by(ibkr_transaction_id=txn.id).all()
        assert len(allocations) == 0

    def test_get_transaction_allocations(self, app_context, client, db_session):
        """Test GET /ibkr/inbox/<transaction_id>/allocations returns allocations."""
        # Create fund, portfolio, and allocation
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
        db_session.add(pf)
        db_session.commit()

        # Create IBKR transaction
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="processed",
        )
        db_session.add(txn)
        db_session.commit()

        # Create allocation
        allocation = IBKRTransactionAllocation(
            ibkr_transaction_id=txn.id,
            portfolio_id=portfolio.id,
            allocation_percentage=100.0,
            allocated_amount=float(Decimal("1500.00")),
            allocated_shares=10.0,
        )
        db_session.add(allocation)
        db_session.commit()

        response = client.get(f"/api/ibkr/inbox/{txn.id}/allocations")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, dict)
        assert "allocations" in data
        assert len(data["allocations"]) >= 1

    def test_update_transaction_allocations(self, app_context, client, db_session):
        """Test PUT /ibkr/inbox/<transaction_id>/allocations updates allocations."""
        # Create 2 portfolios
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio1 = Portfolio(name="Portfolio 1")
        portfolio2 = Portfolio(name="Portfolio 2")
        db_session.add_all([fund, portfolio1, portfolio2])
        db_session.commit()

        # First allocate the transaction
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            isin=fund.isin,
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        # Allocate to process it first (100% to portfolio1)
        from app.services.ibkr_transaction_service import IBKRTransactionService

        IBKRTransactionService.process_transaction_allocation(
            txn.id, [{"portfolio_id": portfolio1.id, "percentage": 100.0}]
        )

        # Now modify to split 60/40 between two portfolios
        payload = {
            "allocations": [
                {"portfolio_id": portfolio1.id, "percentage": 60.0},
                {"portfolio_id": portfolio2.id, "percentage": 40.0},
            ]
        }

        response = client.put(f"/api/ibkr/inbox/{txn.id}/allocations", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data["success"] is True


class TestIBKRBulkOperations:
    """Test IBKR bulk operation endpoints."""

    def test_bulk_allocate(self, app_context, client, db_session):
        """Test POST /ibkr/inbox/bulk-allocate allocates multiple transactions."""
        # Create fund and portfolio
        fund = create_fund("US", "AAPL", "Apple Inc")
        portfolio = Portfolio(name="Test Portfolio")
        db_session.add_all([fund, portfolio])
        db_session.commit()

        # Create IBKR transactions
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            isin=fund.isin,
            description="Apple Inc",
            transaction_type="buy",
            quantity=10,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("1500.00")),
            currency="USD",
            status="pending",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            isin=fund.isin,
            description="Apple Inc",
            transaction_type="buy",
            quantity=5,
            price=float(Decimal("150.00")),
            total_amount=float(Decimal("750.00")),
            currency="USD",
            status="pending",
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        payload = {
            "transaction_ids": [txn1.id, txn2.id],
            "allocations": [{"portfolio_id": portfolio.id, "percentage": 100.0}],
        }

        response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

        assert response.status_code == 200
        data = response.get_json()
        assert data.get("success") is True or "results" in data
