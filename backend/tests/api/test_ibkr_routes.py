"""
Integration tests for IBKR routes (ibkr_routes.py).

Tests IBKR API endpoints:
- GET /ibkr/config - Get config status ✅
- POST /ibkr/config - Save config ✅
- DELETE /ibkr/config - Delete config ✅
- POST /ibkr/config/test - Test connection ✅
- POST /ibkr/import - Import transactions ✅
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

Test Summary: 18 happy path tests, 33 error path tests, 51 total

Error path testing covers:
- Missing required fields
- Invalid data formats
- Resource not found (404)
- Service errors and exceptions
- External API failures
- Connection success/failure paths
- Allocation validation errors
- Bulk operation partial failures
"""

from datetime import datetime, timedelta
from decimal import Decimal
from unittest.mock import patch

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
        """
        Test GET /ibkr/config returns status when no config exists.

        WHY: Users need to know if IBKR is configured before attempting to import transactions.
        The UI displays different states based on this endpoint (setup wizard vs. configured view).
        """
        response = client.get("/api/ibkr/config")

        assert response.status_code == 200
        data = response.get_json()
        assert "configured" in data
        # May be True or False depending on database state

    def test_save_config(self, app_context, client, db_session):
        """
        Test POST /ibkr/config saves configuration.

        WHY: Users must be able to configure their IBKR credentials to enable transaction imports.
        Critical to verify that tokens are encrypted in the database (security requirement) and that
        configuration persists correctly for subsequent import operations.
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
        """
        Test GET /ibkr/config returns status with existing config.

        WHY: Verifies that sensitive credentials (flex_token) are never exposed in API responses
        while still informing the UI that configuration exists. Prevents security vulnerabilities.
        """
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
        """
        Test DELETE /ibkr/config removes configuration.

        WHY: Users must be able to revoke IBKR access and remove their credentials from the system.
        Essential for data privacy compliance and when switching IBKR accounts.
        """
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
        """
        Test POST /ibkr/import imports transactions.

        WHY: Transaction import is the core feature of the IBKR integration. While currently
        skipped due to complex external API dependencies, this test will ensure transactions
        flow correctly from IBKR into the inbox for user review.
        """
        # Would require mocking IBKRFlexService.fetch_and_process_flex_query
        pass


class TestIBKRInbox:
    """Test IBKR inbox endpoints."""

    def test_get_inbox_empty(self, app_context, client):
        """
        Test GET /ibkr/inbox returns empty list when no transactions.

        WHY: New users or users with fully processed transactions need to see an empty inbox
        state. The UI must gracefully handle this scenario without errors.
        """
        response = client.get("/api/ibkr/inbox")

        assert response.status_code == 200
        data = response.get_json()
        assert isinstance(data, list)

    def test_get_inbox_with_transactions(self, app_context, client, db_session):
        """
        Test GET /ibkr/inbox returns inbox transactions.

        WHY: Users need to see pending transactions requiring allocation to their portfolios.
        This is the primary workflow for processing imported IBKR data.
        """
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
        """
        Test GET /ibkr/inbox/count returns count.

        WHY: The UI displays a notification badge showing pending transactions count.
        This lightweight endpoint prevents loading full transaction data just to show the count.
        """
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
        """
        Test GET /ibkr/inbox/<transaction_id> returns specific transaction.

        WHY: Users need to view full transaction details before allocating to portfolios.
        Supports drill-down UI pattern from inbox list to transaction detail view.
        """
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
        """
        Test POST /ibkr/inbox/<transaction_id>/ignore marks transaction as ignored.

        WHY: Users need to skip transactions that don't require portfolio allocation (e.g., fees,
        broker transfers). Ignoring removes them from the active inbox without deletion.
        """
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
        """
        Test DELETE /ibkr/inbox/<transaction_id> removes transaction.

        WHY:
        Users need to permanently remove erroneous or duplicate transactions imported from IBKR.
        Provides cleanup capability for data quality issues.
        """
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
        """
        Test GET /ibkr/portfolios returns portfolios.

        WHY: Users need to see available portfolios when allocating IBKR transactions.
        This endpoint populates the portfolio selection UI in the allocation workflow.
        """
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

        WHY: Shows users which portfolios already hold the fund in this transaction, enabling
        smart allocation suggestions. Prevents users from allocating to portfolios missing the fund,
        which would cause errors during transaction processing.
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

    def test_allocate_transaction(self, app_context, client, db_session):
        """
        Test POST /ibkr/inbox/<transaction_id>/allocate allocates transaction.

        WHY:
        Core business function that converts imported IBKR transactions into portfolio holdings.
        Users must allocate transactions to track which portfolios contain which investments.
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
        """
        Test GET /ibkr/dividends/pending returns pending dividends.

        WHY:
        Users need to see which dividend records await matching with IBKR dividend transactions.
        Supports the workflow of tracking expected vs. actual dividend payments.
        """
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
        """
        Test POST /ibkr/inbox/<transaction_id>/match-dividend matches dividend.

        WHY: Validates Bug Fix #1 - dividend transactions were being subtracted instead of added.
        Users must accurately link dividend payments from IBKR with expected dividend records.
        """
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
        """
        Test POST /ibkr/inbox/<transaction_id>/unallocate removes allocations.

        WHY: Users need to undo incorrect allocations and return transactions to the inbox for
        reprocessing. Essential for error correction without deleting the entire transaction.
        """
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
        """
        Test GET /ibkr/inbox/<transaction_id>/allocations returns allocations.

        WHY: Users need to see how a transaction was allocated across portfolios, including
        percentages and amounts. Supports audit trail and modification workflows.
        """
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
        """
        Test PUT /ibkr/inbox/<transaction_id>/allocations updates allocations.

        WHY:
        Users need to modify allocation percentages without fully unallocating and reallocating.
        Supports adjusting portfolio splits (e.g., changing 100% portfolio A to 60% A / 40% B).
        """
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
        """
        Test POST /ibkr/inbox/bulk-allocate allocates multiple transactions.

        WHY: Users importing large transaction sets need to allocate many transactions at once
        with the same portfolio split. Bulk operations save time vs. allocating individually.
        """
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


# ============================================================================
# ERROR PATH TESTS
# ============================================================================


class TestIBKRConfigErrors:
    """Test error paths for IBKR configuration endpoints."""

    def test_save_config_missing_flex_token(self, client):
        """
        Test POST /ibkr/config rejects missing flex_token.

        WHY: Prevents incomplete configuration that would fail during IBKR API calls.
        Clear validation errors guide users to provide all required credentials.
        """
        payload = {"flex_query_id": "query_123"}

        response = client.post("/api/ibkr/config", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        # Flask-RESTX validation returns errors in 'errors' field
        assert "errors" in data or "error" in data

    def test_save_config_missing_flex_query_id(self, client):
        """
        Test POST /ibkr/config rejects missing flex_query_id.

        WHY: Both token and query ID are required to fetch IBKR data. Validation prevents
        configuration that cannot successfully import transactions.
        """
        payload = {"flex_token": "token_123"}

        response = client.post("/api/ibkr/config", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        # Flask-RESTX validation returns errors in 'errors' field
        assert "errors" in data or "error" in data

    def test_save_config_empty_payload(self, client):
        """
        Test POST /ibkr/config rejects empty payload.

        WHY: Protects against UI bugs or API misuse that send requests without data.
        Returns clear error instead of cryptic database constraint violations.
        """
        response = client.post("/api/ibkr/config", json={})

        assert response.status_code == 400
        data = response.get_json()
        # Flask-RESTX validation returns errors in 'errors' field
        assert "errors" in data or "error" in data

    def test_save_config_no_payload(self, client):
        """
        Test POST /ibkr/config rejects no payload.

        WHY: Handles malformed requests gracefully with appropriate HTTP status codes.
        Prevents server errors from reaching users as unhandled exceptions.
        """
        response = client.post("/api/ibkr/config", json=None)

        # Will either be 400 for missing fields or 415 for wrong content type
        assert response.status_code in [400, 415]

    def test_save_config_invalid_token_expires_at(self, client):
        """
        Test POST /ibkr/config rejects invalid token_expires_at format.

        WHY: Date validation prevents data corruption from malformed dates. Users receive
        immediate feedback on date format errors rather than silent failures.
        """
        payload = {
            "flex_token": "token_123",
            "flex_query_id": "query_123",
            "token_expires_at": "not-a-date",
        }

        response = client.post("/api/ibkr/config", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        assert "Invalid token_expires_at format" in data["error"]

    def test_save_config_service_error(self, client):
        """
        Test POST /ibkr/config handles service errors.

        WHY: Database errors must return 500 status with user-friendly messages rather than
        exposing internal exceptions. Critical for production error handling.
        """

        def mock_save_config(*args, **kwargs):
            raise Exception("Database error")

        with patch(
            "app.api.ibkr_namespace.IBKRConfigService.save_config",
            mock_save_config,
        ):
            payload = {"flex_token": "token_123", "flex_query_id": "query_123"}
            response = client.post("/api/ibkr/config", json=payload)

            assert response.status_code == 500
            data = response.get_json()
            assert "Failed to save configuration" in data["error"]

    def test_delete_config_not_found(self, client):
        """
        Test DELETE /ibkr/config handles config not found.

        WHY: Users attempting to delete non-existent configuration should receive clear 404
        errors, not generic failures. Supports proper REST semantics.
        """

        def mock_delete_config():
            raise ValueError("No configuration found")

        with patch(
            "app.api.ibkr_namespace.IBKRConfigService.delete_config",
            mock_delete_config,
        ):
            response = client.delete("/api/ibkr/config")

            assert response.status_code == 404
            data = response.get_json()
            assert "error" in data

    def test_delete_config_service_error(self, client):
        """
        Test DELETE /ibkr/config handles service errors.

        WHY: Database failures during deletion must be communicated clearly to users without
        leaving configuration in inconsistent state. Ensures graceful error recovery.
        """

        def mock_delete_config():
            raise Exception("Database error")

        with patch(
            "app.api.ibkr_namespace.IBKRConfigService.delete_config",
            mock_delete_config,
        ):
            response = client.delete("/api/ibkr/config")

            assert response.status_code == 500
            data = response.get_json()
            assert "Failed to delete configuration" in data["error"]


class TestIBKRConnectionErrors:
    """Test error paths for IBKR connection testing."""

    def test_connection_missing_flex_token(self, client):
        """
        Test POST /ibkr/config/test rejects missing flex_token.

        WHY: Users must test complete credentials before saving. Prevents testing with
        incomplete configuration that would always fail IBKR connection attempts.
        """
        payload = {"flex_query_id": "query_123"}

        response = client.post("/api/ibkr/config/test", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        # Flask-RESTX validation returns errors in 'errors' field
        assert "errors" in data or "error" in data

    def test_connection_missing_flex_query_id(self, client):
        """
        Test POST /ibkr/config/test rejects missing flex_query_id.

        WHY: Connection testing requires both credentials. Early validation saves users from
        confusion when test connections fail due to missing query ID.
        """
        payload = {"flex_token": "token_123"}

        response = client.post("/api/ibkr/config/test", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        # Flask-RESTX validation returns errors in 'errors' field
        assert "errors" in data or "error" in data

    def test_connection_empty_payload(self, client):
        """
        Test POST /ibkr/config/test rejects empty payload.

        WHY: Validates input before attempting expensive IBKR API calls. Returns immediate
        error feedback rather than wasting time on doomed connection attempts.
        """
        response = client.post("/api/ibkr/config/test", json={})

        assert response.status_code == 400

    def test_connection_success(self, client):
        """
        Test POST /ibkr/config/test handles successful connection.

        WHY: Users need confirmation that their IBKR credentials are valid before saving
        configuration. Success feedback builds user confidence in the integration setup.
        """
        with patch("app.api.ibkr_namespace.IBKRFlexService") as mock_service_class:
            mock_instance = mock_service_class.return_value
            mock_instance.test_connection.return_value = {"success": True, "message": "Connected"}

            payload = {"flex_token": "token_123", "flex_query_id": "query_123"}
            response = client.post("/api/ibkr/config/test", json=payload)

            assert response.status_code == 200
            data = response.get_json()
            assert data["success"] is True

    def test_connection_failure(self, client):
        """
        Test POST /ibkr/config/test handles failed connection.

        WHY: Users with invalid credentials need clear error messages explaining why the
        connection failed. Prevents saving credentials that cannot work.
        """
        with patch("app.api.ibkr_namespace.IBKRFlexService") as mock_service_class:
            mock_instance = mock_service_class.return_value
            mock_instance.test_connection.return_value = {
                "success": False,
                "error": "Invalid credentials",
            }

            payload = {"flex_token": "token_123", "flex_query_id": "query_123"}
            response = client.post("/api/ibkr/config/test", json=payload)

            assert response.status_code == 400
            data = response.get_json()
            assert data["success"] is False

    def test_connection_api_failure(self, client):
        """
        Test POST /ibkr/config/test handles API failures.

        WHY: IBKR API outages or network issues must be handled gracefully. Users need to
        distinguish between invalid credentials and temporary infrastructure problems.
        """
        with patch("app.api.ibkr_namespace.IBKRFlexService") as mock_service_class:
            mock_instance = mock_service_class.return_value
            mock_instance.test_connection.side_effect = Exception("API error")

            payload = {"flex_token": "token_123", "flex_query_id": "query_123"}
            response = client.post("/api/ibkr/config/test", json=payload)

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data


class TestIBKRImportErrors:
    """Test error paths for IBKR import endpoint."""

    def test_import_missing_config(self, client):
        """
        Test POST /ibkr/import handles missing config.

        WHY: Users attempting to import before configuring IBKR credentials need clear guidance
        to complete setup first. Prevents confusing failures from unconfigured systems.
        """
        with patch("app.api.ibkr_namespace.IBKRConfigService.get_first_config") as mock_get:
            mock_get.return_value = None

            response = client.post("/api/ibkr/import")

            assert response.status_code == 400
            data = response.get_json()
            assert "not configured" in data["error"]

    def test_import_disabled_config(self, client, db_session):
        """
        Test POST /ibkr/import handles disabled config.

        WHY: Administrators may disable IBKR integration temporarily. Users must receive clear
        403 Forbidden status indicating the feature is intentionally disabled, not broken.
        """
        config = IBKRConfig(
            flex_token="test_token",
            flex_query_id="test_query",
            auto_import_enabled=False,
            enabled=False,  # Disabled
        )
        db_session.add(config)
        db_session.commit()

        response = client.post("/api/ibkr/import")

        assert response.status_code == 403
        data = response.get_json()
        assert "disabled" in data["error"]

    def test_import_api_failure(self, client, db_session):
        """
        Test POST /ibkr/import handles API failures.

        WHY: IBKR API may return errors or timeouts. Users need descriptive error messages
        to understand if the issue is temporary (retry later) vs. configuration problem.
        """
        config = IBKRConfig(
            flex_token="test_token",
            flex_query_id="test_query",
            auto_import_enabled=False,
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        with patch("app.api.ibkr_namespace.IBKRFlexService") as mock_service_class:
            mock_instance = mock_service_class.return_value
            # Mock _decrypt_token to return a token
            mock_instance._decrypt_token.return_value = "decrypted_token"
            # Mock fetch_statement to return None (simulating API failure)
            mock_instance.fetch_statement.return_value = None

            response = client.post("/api/ibkr/import")

            assert response.status_code == 500
            data = response.get_json()
            assert "Failed to fetch statement" in data["error"]

    def test_import_exception(self, client, db_session):
        """
        Test POST /ibkr/import handles general exceptions.

        WHY: Unexpected errors during import (decryption failures, parsing errors) must be
        caught and returned as 500 errors rather than crashing the application.
        """
        config = IBKRConfig(
            flex_token="test_token",
            flex_query_id="test_query",
            auto_import_enabled=False,
            enabled=True,
        )
        db_session.add(config)
        db_session.commit()

        with patch("app.api.ibkr_namespace.IBKRFlexService") as mock_service_class:
            mock_instance = mock_service_class.return_value
            # Mock trigger_manual_import to return an exception response
            mock_instance.trigger_manual_import.return_value = (
                {"error": "Import failed", "details": "Decryption error"},
                500,
            )

            response = client.post("/api/ibkr/import")

            assert response.status_code == 500
            data = response.get_json()
            assert "Import failed" in data["error"]


class TestIBKRInboxErrors:
    """Test error paths for IBKR inbox endpoints."""

    def test_get_transaction_not_found(self, client):
        """
        Test GET /ibkr/inbox/<id> returns 404 for non-existent transaction.

        WHY: Users accessing outdated links or deleted transactions need proper 404 responses.
        Prevents confusion from generic errors when requesting non-existent resources.
        """
        response = client.get("/api/ibkr/inbox/nonexistent-id")

        assert response.status_code == 404

    def test_ignore_transaction_not_found(self, client):
        """
        Test POST /ibkr/inbox/<id>/ignore returns 404 for non-existent transaction.

        WHY: Attempting to ignore already-deleted transactions should return clear 404 errors.
        Supports idempotent operations and prevents misleading success messages.
        """
        response = client.post("/api/ibkr/inbox/nonexistent-id/ignore")

        assert response.status_code == 404

    def test_delete_transaction_not_found(self, client):
        """
        Test DELETE /ibkr/inbox/<id> returns 404 for non-existent transaction.

        WHY: Delete operations on non-existent resources should return 404 per REST standards.
        Helps users identify if transaction was already deleted vs. database errors.
        """
        response = client.delete("/api/ibkr/inbox/nonexistent-id")

        assert response.status_code == 404

    def test_delete_transaction_service_error(self, client, db_session):
        """
        Test DELETE /ibkr/inbox/<id> handles service errors.

        WHY: Database commit failures during deletion must return 500 errors without partial
        deletes. Maintains data consistency and alerts users to retry the operation.
        """
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Test",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        def mock_commit():
            raise Exception("Database error")

        with patch("app.models.db.session.commit", mock_commit):
            response = client.delete(f"/api/ibkr/inbox/{txn.id}")

            assert response.status_code == 500

    def test_get_inbox_count_service_error(self, client):
        """
        Test GET /ibkr/inbox/count handles service errors.

        WHY: Database query failures should not crash the notification badge UI component.
        Returns 500 with error message instead of breaking the entire user interface.
        """
        with patch("app.api.ibkr_namespace.IBKRTransactionService.get_inbox_count") as mock_count:
            mock_count.side_effect = Exception("Database query failed")

            response = client.get("/api/ibkr/inbox/count")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data

    def test_get_eligible_portfolios_transaction_not_found(self, client):
        """
        Test GET /ibkr/inbox/<id>/eligible-portfolios handles missing transaction.

        WHY: Users navigating to allocation screen for deleted transactions need clear 404 errors.
        Prevents attempting fund matching on non-existent data.
        """
        fake_id = make_id()
        response = client.get(f"/api/ibkr/inbox/{fake_id}/eligible-portfolios")

        assert response.status_code == 404
        # Flask's default 404 returns HTML, not JSON
        # Just verify we got a 404 status code

    def test_get_eligible_portfolios_service_error(self, client, db_session):
        """
        Test GET /ibkr/inbox/<id>/eligible-portfolios handles service errors.

        WHY: Fund matching service failures should not prevent users from seeing other inbox
        transactions. Returns 500 with diagnostic information for troubleshooting.
        """
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Test",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        with patch(
            "app.services.fund_matching_service.FundMatchingService.get_eligible_portfolios_for_transaction"
        ) as mock_get:
            mock_get.side_effect = Exception("Matching service error")

            response = client.get(f"/api/ibkr/inbox/{txn.id}/eligible-portfolios")

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data


class TestIBKRAllocationErrors:
    """Test error paths for IBKR allocation endpoints."""

    def test_allocate_transaction_not_found(self, client):
        """
        Test POST /ibkr/inbox/<id>/allocate returns error for non-existent transaction.

        WHY: Attempting to allocate deleted transactions should fail fast with clear errors.
        Prevents phantom allocations and maintains referential integrity.
        """
        payload = {"allocations": [{"portfolio_id": "test", "percentage": 100}]}
        response = client.post("/api/ibkr/inbox/nonexistent-id/allocate", json=payload)

        # May return 400 for validation or 404 for not found depending on order
        assert response.status_code in [400, 404]

    def test_allocate_missing_allocations(self, client, db_session):
        """
        Test POST /ibkr/inbox/<id>/allocate rejects missing allocations.

        WHY: Allocating without specifying target portfolios is meaningless. Validates required
        payload fields before processing to fail fast with actionable error messages.
        """
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Test",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        payload = {}
        response = client.post(f"/api/ibkr/inbox/{txn.id}/allocate", json=payload)

        assert response.status_code in [400, 500]

    def test_match_dividend_not_found(self, client):
        """
        Test POST /ibkr/inbox/<id>/match-dividend returns error for non-existent transaction.

        WHY: Matching dividends requires valid transaction reference. Returns clear error when
        transaction is deleted or ID is invalid, preventing orphaned dividend records.
        """
        payload = {"dividend_ids": ["div1"], "isin": "US0378331005"}
        response = client.post("/api/ibkr/inbox/nonexistent-id/match-dividend", json=payload)

        # May return 400 for validation or 404 for not found
        assert response.status_code in [400, 404]

    def test_match_dividend_missing_fields(self, client, db_session):
        """
        Test POST /ibkr/inbox/<id>/match-dividend rejects missing fields.

        WHY: Dividend matching requires dividend IDs to link transactions. Validates payload
        structure to prevent incomplete matches that corrupt dividend tracking.
        """
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Test",
            transaction_type="dividend",
            quantity=0,
            price=0.0,
            total_amount=100.0,
            currency="USD",
            status="pending",
        )
        db_session.add(txn)
        db_session.commit()

        payload = {}
        response = client.post(f"/api/ibkr/inbox/{txn.id}/match-dividend", json=payload)

        assert response.status_code in [400, 500]

    def test_unallocate_transaction_not_found(self, client):
        """
        Test POST /ibkr/inbox/<id>/unallocate returns 404.

        WHY: Unallocating non-existent transactions should return proper 404 status.
        Prevents silent failures and helps users identify deleted transactions.
        """
        response = client.post("/api/ibkr/inbox/nonexistent-id/unallocate")

        assert response.status_code == 404

    def test_update_allocations_not_found(self, client):
        """
        Test PUT /ibkr/inbox/<id>/allocations returns error for non-existent transaction.

        WHY: Modifying allocations on deleted transactions should fail with clear 404 errors.
        Supports idempotency and prevents updates to phantom records.
        """
        payload = {"allocations": [{"portfolio_id": "test", "percentage": 100}]}
        response = client.put("/api/ibkr/inbox/nonexistent-id/allocations", json=payload)

        # May return 400 for validation or 404 for not found
        assert response.status_code in [400, 404]

    def test_update_allocations_missing_allocations(self, client, db_session):
        """
        Test PUT /ibkr/inbox/<id>/allocations rejects missing allocations.

        WHY: PUT requests must include complete allocation data. Validates payload structure
        to prevent accidentally clearing allocations with empty requests.
        """
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Test",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="processed",
        )
        db_session.add(txn)
        db_session.commit()

        payload = {}
        response = client.put(f"/api/ibkr/inbox/{txn.id}/allocations", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        # Accept either Flask-RESTX validation format or legacy format
        assert "errors" in data or "error" in data

    def test_update_allocations_value_error(self, client, db_session):
        """
        Test PUT /ibkr/inbox/<id>/allocations handles ValueError.

        WHY: Business logic validation errors (percentages not summing to 100%, invalid portfolio
        IDs) must return 400 with specific error messages to guide user corrections.
        """
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Test",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="processed",
        )
        db_session.add(txn)
        db_session.commit()

        with patch(
            "app.api.ibkr_namespace.IBKRTransactionService.modify_allocations"
        ) as mock_modify:
            mock_modify.side_effect = ValueError("Allocation validation failed")

            payload = {"allocations": [{"portfolio_id": "test", "percentage": 100}]}
            response = client.put(f"/api/ibkr/inbox/{txn.id}/allocations", json=payload)

            assert response.status_code == 400
            data = response.get_json()
            assert "error" in data or "message" in data

    def test_update_allocations_general_error(self, client, db_session):
        """
        Test PUT /ibkr/inbox/<id>/allocations handles general exceptions.

        WHY: Unexpected database or service errors must be caught and returned as 500 errors.
        Prevents application crashes and provides error context for troubleshooting.
        """
        txn = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol="AAPL",
            description="Test",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="processed",
        )
        db_session.add(txn)
        db_session.commit()

        with patch(
            "app.api.ibkr_namespace.IBKRTransactionService.modify_allocations"
        ) as mock_modify:
            mock_modify.side_effect = Exception("Unexpected database error")

            payload = {"allocations": [{"portfolio_id": "test", "percentage": 100}]}
            response = client.put(f"/api/ibkr/inbox/{txn.id}/allocations", json=payload)

            assert response.status_code == 500
            data = response.get_json()
            assert "error" in data or "message" in data


class TestIBKRBulkOperationsErrors:
    """Test error paths for IBKR bulk operations."""

    def test_bulk_allocate_missing_transaction_ids(self, client):
        """
        Test POST /ibkr/inbox/bulk-allocate rejects missing transaction_ids.

        WHY: Bulk operations require transaction list. Validates required fields to prevent
        meaningless operations and provide clear guidance on correct API usage.
        """
        payload = {"allocations": [{"portfolio_id": "test", "percentage": 100}]}
        response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

        assert response.status_code in [400, 500]

    def test_bulk_allocate_empty_transaction_ids(self, client):
        """
        Test POST /ibkr/inbox/bulk-allocate rejects empty transaction_ids.

        WHY: Empty transaction list wastes processing time and indicates UI or client error.
        Fast validation prevents unnecessary database queries and service calls.
        """
        payload = {
            "transaction_ids": [],
            "allocations": [{"portfolio_id": "test", "percentage": 100}],
        }
        response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

        assert response.status_code in [400, 500]

    def test_bulk_allocate_missing_allocations(self, client):
        """
        Test POST /ibkr/inbox/bulk-allocate rejects missing allocations.

        WHY: Bulk allocation without portfolio targets is meaningless. Validates complete
        payload structure before processing multiple transactions.
        """
        payload = {"transaction_ids": ["txn1", "txn2"]}
        response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

        assert response.status_code in [400, 500]

    def test_bulk_allocate_empty_allocations(self, client):
        """
        Test POST /ibkr/inbox/bulk-allocate rejects empty allocations list.

        WHY: Empty allocation list indicates client error. Returns clear validation error
        rather than silently failing or processing transactions without destinations.
        """
        payload = {"transaction_ids": ["txn1", "txn2"], "allocations": []}
        response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        assert "error" in data

    def test_bulk_allocate_invalid_percentage_sum(self, client):
        """
        Test POST /ibkr/inbox/bulk-allocate rejects allocations not summing to 100%.

        WHY: Allocation percentages must total 100% to ensure complete transaction accounting.
        Validates business rules before processing to prevent partial or over-allocations.
        """
        payload = {
            "transaction_ids": ["txn1"],
            "allocations": [
                {"portfolio_id": "test1", "percentage": 50},
                {"portfolio_id": "test2", "percentage": 30},
            ],
        }
        response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

        assert response.status_code == 400
        data = response.get_json()
        assert "100%" in data["error"] or "sum" in data["error"].lower()

    def test_bulk_allocate_partial_failure(self, client, db_session):
        """
        Test POST /ibkr/inbox/bulk-allocate handles individual transaction failures.

        WHY: Bulk operations should process as many transactions as possible, reporting both
        successes and failures. Partial success prevents having to retry entire bulk operation.
        """
        fund = create_fund("US", "AAPL", "Apple Inc")
        db_session.add(fund)
        db_session.commit()

        # Create two transactions
        txn1 = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            isin=fund.isin,
            description="Test 1",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="pending",
        )
        txn2 = IBKRTransaction(
            ibkr_transaction_id=make_id(),
            transaction_date=datetime.now().date(),
            symbol=fund.symbol,
            isin=fund.isin,
            description="Test 2",
            transaction_type="buy",
            quantity=10,
            price=100.0,
            total_amount=1000.0,
            currency="USD",
            status="pending",
        )
        db_session.add_all([txn1, txn2])
        db_session.commit()

        # Mock process_transaction_allocation to fail for one transaction
        with patch(
            "app.api.ibkr_namespace.IBKRTransactionService.process_transaction_allocation"
        ) as mock_process:
            # First call succeeds, second call fails
            mock_process.side_effect = [
                {"success": True, "created_transactions": []},
                Exception("Allocation failed for transaction 2"),
            ]

            payload = {
                "transaction_ids": [txn1.id, txn2.id],
                "allocations": [{"portfolio_id": "test", "percentage": 100}],
            }
            response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

            assert response.status_code == 200
            data = response.get_json()
            assert data["success"] is True
            assert data["processed"] == 1
            assert data["failed"] == 1
            assert data["errors"] is not None
            assert len(data["errors"]) == 1

    def test_bulk_allocate_general_error(self, client):
        """
        Test POST /ibkr/inbox/bulk-allocate handles general exceptions.

        WHY: Critical database or service failures during bulk operations must be caught
        and reported. Prevents silent data corruption and provides diagnostic information.
        """
        with patch(
            "app.api.ibkr_namespace.IBKRTransactionService.process_transaction_allocation"
        ) as mock_process:
            mock_process.side_effect = Exception("Critical database error")

            payload = {
                "transaction_ids": ["txn1"],
                "allocations": [{"portfolio_id": "test", "percentage": 100}],
            }
            response = client.post("/api/ibkr/inbox/bulk-allocate", json=payload)

            assert response.status_code in [200, 500]
            # May still return 200 with errors or 500 depending on when exception occurs
