"""
Comprehensive test suite for FundMatchingService.

Tests fund matching logic for IBKR transactions including:
- Symbol normalization with exchange suffixes
- ISIN-based matching (priority 1)
- Exact symbol matching (priority 2)
- Normalized symbol matching (priority 3)
- Portfolio eligibility determination
- Warning message generation
"""

from datetime import date

from app.models import Fund, IBKRTransaction, InvestmentType, Portfolio, PortfolioFund, db
from app.services.fund_matching_service import FundMatchingService
from tests.test_helpers import (
    make_custom_string,
    make_ibkr_transaction_id,
    make_id,
    make_isin,
    make_portfolio_name,
    make_symbol,
)


class TestSymbolNormalization:
    """Tests for exchange-based symbol normalization."""

    def test_normalize_symbol_german_xetra(self):
        """Test normalization for German XETRA exchange."""
        result = FundMatchingService.normalize_symbol("WEBN", "XETRA")
        assert result == "WEBN.DE"

    def test_normalize_symbol_german_ibis(self):
        """Test normalization for German IBIS exchange."""
        result = FundMatchingService.normalize_symbol("BMW", "IBIS")
        assert result == "BMW.DE"

    def test_normalize_symbol_german_fwb(self):
        """Test normalization for Frankfurt exchange."""
        result = FundMatchingService.normalize_symbol("SAP", "FWB")
        assert result == "SAP.DE"

    def test_normalize_symbol_london(self):
        """Test normalization for London Stock Exchange."""
        result = FundMatchingService.normalize_symbol("HSBA", "LSE")
        assert result == "HSBA.L"

    def test_normalize_symbol_switzerland(self):
        """Test normalization for Swiss exchange."""
        result = FundMatchingService.normalize_symbol("NESN", "SIX")
        assert result == "NESN.SW"

    def test_normalize_symbol_amsterdam(self):
        """Test normalization for Amsterdam exchange."""
        result = FundMatchingService.normalize_symbol("ASML", "XAMS")
        assert result == "ASML.AS"

    def test_normalize_symbol_us_nyse(self):
        """Test normalization for US NYSE (no suffix)."""
        result = FundMatchingService.normalize_symbol("IBM", "NYSE")
        assert result == "IBM"

    def test_normalize_symbol_us_nasdaq(self):
        """Test normalization for US NASDAQ (no suffix)."""
        result = FundMatchingService.normalize_symbol("AAPL", "NASDAQ")
        assert result == "AAPL"

    def test_normalize_symbol_no_exchange(self):
        """Test normalization without exchange info."""
        result = FundMatchingService.normalize_symbol("AAPL", None)
        assert result == "AAPL"

    def test_normalize_symbol_unknown_exchange(self):
        """Test normalization for unknown exchange (no suffix)."""
        result = FundMatchingService.normalize_symbol("TEST", "UNKNOWN")
        assert result == "TEST"

    def test_normalize_symbol_empty_symbol(self):
        """Test normalization with empty symbol."""
        result = FundMatchingService.normalize_symbol("", "NYSE")
        assert result == ""

    def test_normalize_symbol_none_symbol(self):
        """Test normalization with None symbol."""
        result = FundMatchingService.normalize_symbol(None, "NYSE")
        assert result is None

    def test_normalize_symbol_case_insensitive(self):
        """Test that exchange matching is case-insensitive."""
        result = FundMatchingService.normalize_symbol("BMW", "xetra")
        assert result == "BMW.DE"


class TestFindFundByTransaction:
    """Tests for finding funds matching IBKR transactions."""

    def test_find_fund_by_isin_match(self, app_context, db_session):
        """Test finding fund by ISIN (highest priority)."""
        # Create fund with unique ISIN
        unique_isin = make_isin("US")
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=unique_isin,
            symbol=make_symbol("AAPL"),
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Create IBKR transaction with matching ISIN
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=fund.symbol,
            isin=unique_isin,  # Matches
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=-15000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Find fund
        result = FundMatchingService.find_fund_by_transaction(txn)

        assert result is not None
        assert result.id == fund.id
        assert result.isin == unique_isin

    def test_find_fund_by_exact_symbol_match(self, app_context, db_session):
        """Test finding fund by exact symbol when ISIN doesn't match."""
        # Create fund with unique ISIN and symbol
        fund_isin = make_isin("US")
        unique_symbol = make_symbol("MSFT")
        fund = Fund(
            id=make_id(),
            name="Microsoft Corp",
            isin=fund_isin,
            symbol=unique_symbol,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Create IBKR transaction with different ISIN but matching symbol
        txn_isin = make_isin("US")
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=unique_symbol,  # Matches
            isin=txn_isin,  # Different ISIN
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=50,
            price=300.00,
            total_amount=-15000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Find fund
        result = FundMatchingService.find_fund_by_transaction(txn)

        assert result is not None
        assert result.id == fund.id
        assert result.symbol == unique_symbol

    def test_find_fund_by_normalized_symbol(self, app_context, db_session):
        """Test finding fund by normalized symbol (e.g., WEBN.DE)."""
        # Create fund with German suffix and unique ISIN
        unique_isin = make_isin("DE")
        unique_base = make_custom_string("WEBN", 2)
        fund = Fund(
            id=make_id(),
            name="Webasto SE",
            isin=unique_isin,
            symbol=f"{unique_base}.DE",  # Has suffix
            currency="EUR",
            exchange="XETRA",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Create IBKR transaction without suffix
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=unique_base,  # No suffix
            isin=None,  # No ISIN
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=100,
            price=50.00,
            total_amount=-5000.00,
            currency="EUR",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Find fund (should try common suffixes)
        result = FundMatchingService.find_fund_by_transaction(txn)

        assert result is not None
        assert result.id == fund.id
        assert result.symbol == f"{unique_base}.DE"

    def test_find_fund_no_match(self, app_context, db_session):
        """Test when no fund matches the transaction."""
        # Create fund that won't match
        fund_isin = make_isin("US")
        fund_symbol = make_symbol("AAPL")
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=fund_isin,
            symbol=fund_symbol,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Create IBKR transaction with different symbol and ISIN
        txn_isin = make_isin("US")
        txn_symbol = make_symbol("TSLA")
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=txn_symbol,  # Different
            isin=txn_isin,  # Different
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=10,
            price=200.00,
            total_amount=-2000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Find fund
        result = FundMatchingService.find_fund_by_transaction(txn)

        assert result is None

    def test_find_fund_isin_priority_over_symbol(self, app_context, db_session):
        """Test that ISIN match takes priority over symbol match."""
        # Create two funds with unique ISINs
        isin_correct = make_isin("US")
        isin_wrong = make_isin("US")
        symbol_correct = make_custom_string("AAPL_OLD", 2)
        symbol_wrong = make_symbol("AAPL")

        fund_correct = Fund(
            id=make_id(),
            name="Apple Inc (Correct)",
            isin=isin_correct,  # Matches
            symbol=symbol_correct,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        fund_wrong = Fund(
            id=make_id(),
            name="Apple Inc (Wrong)",
            isin=isin_wrong,
            symbol=symbol_wrong,  # Matches symbol
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add_all([fund_correct, fund_wrong])
        db.session.commit()

        # Create transaction with ISIN matching fund_correct, symbol matching fund_wrong
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=symbol_wrong,
            isin=isin_correct,  # Matches fund_correct
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=-15000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Find fund - should match by ISIN (priority)
        result = FundMatchingService.find_fund_by_transaction(txn)

        assert result is not None
        assert result.id == fund_correct.id
        assert result.name == "Apple Inc (Correct)"


class TestGetPortfoliosWithFund:
    """Tests for getting portfolios that contain a fund."""

    def test_get_portfolios_with_fund_single_portfolio(self, app_context, db_session):
        """Test getting single portfolio with fund."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=make_isin("US"),
            symbol="AAPL",
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)

        # Create portfolio
        portfolio = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Portfolio"),
            is_archived=False,
        )
        db.session.add(portfolio)

        # Create portfolio-fund relationship
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Get portfolios
        portfolios = FundMatchingService.get_portfolios_with_fund(fund.id)

        assert len(portfolios) == 1
        assert portfolios[0].id == portfolio.id

    def test_get_portfolios_with_fund_multiple_portfolios(self, app_context, db_session):
        """Test getting multiple portfolios with same fund."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=make_isin("US"),
            symbol="AAPL",
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)

        # Create multiple portfolios
        portfolio1 = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Portfolio 1"),
            is_archived=False,
        )
        portfolio2 = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Portfolio 2"),
            is_archived=False,
        )
        db.session.add_all([portfolio1, portfolio2])

        # Create portfolio-fund relationships
        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio1.id, fund_id=fund.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio2.id, fund_id=fund.id)
        db.session.add_all([pf1, pf2])
        db.session.commit()

        # Get portfolios
        portfolios = FundMatchingService.get_portfolios_with_fund(fund.id)

        assert len(portfolios) == 2
        portfolio_ids = {p.id for p in portfolios}
        assert portfolio1.id in portfolio_ids
        assert portfolio2.id in portfolio_ids

    def test_get_portfolios_excludes_archived(self, app_context, db_session):
        """Test that archived portfolios are excluded."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=make_isin("US"),
            symbol="AAPL",
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)

        # Create active and archived portfolios
        portfolio_active = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Active Portfolio"),
            is_archived=False,
        )
        portfolio_archived = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Archived Portfolio"),
            is_archived=True,  # Archived
        )
        db.session.add_all([portfolio_active, portfolio_archived])

        # Create portfolio-fund relationships
        pf1 = PortfolioFund(id=make_id(), portfolio_id=portfolio_active.id, fund_id=fund.id)
        pf2 = PortfolioFund(id=make_id(), portfolio_id=portfolio_archived.id, fund_id=fund.id)
        db.session.add_all([pf1, pf2])
        db.session.commit()

        # Get portfolios
        portfolios = FundMatchingService.get_portfolios_with_fund(fund.id)

        # Should only return active portfolio
        assert len(portfolios) == 1
        assert portfolios[0].id == portfolio_active.id

    def test_get_portfolios_no_portfolios(self, app_context, db_session):
        """Test when fund is not in any portfolio."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=make_isin("US"),
            symbol="AAPL",
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Get portfolios
        portfolios = FundMatchingService.get_portfolios_with_fund(fund.id)

        assert portfolios == []


class TestGetEligiblePortfoliosForTransaction:
    """Tests for complete transaction eligibility checking."""

    def test_eligible_portfolios_success(self, app_context, db_session):
        """Test successful matching with eligible portfolios."""
        # Create fund with unique ISIN
        unique_isin = make_isin("US")
        unique_symbol = make_symbol("AAPL")
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=unique_isin,
            symbol=unique_symbol,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)

        # Create portfolio with fund
        portfolio = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Portfolio"),
            description="Test portfolio",
            is_archived=False,
        )
        db.session.add(portfolio)

        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create IBKR transaction
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=unique_symbol,
            isin=unique_isin,
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=-15000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Get eligible portfolios
        result = FundMatchingService.get_eligible_portfolios_for_transaction(txn)

        assert result["match_info"]["found"] is True
        assert result["match_info"]["matched_by"] == "isin"
        assert result["match_info"]["fund_id"] == fund.id
        assert result["match_info"]["fund_name"] == "Apple Inc"
        assert len(result["portfolios"]) == 1
        assert result["portfolios"][0]["id"] == portfolio.id
        assert result["warning"] is None

    def test_eligible_portfolios_no_fund_match(self, app_context, db_session):
        """Test when no fund matches the transaction."""
        # Create IBKR transaction (no matching fund)
        unique_isin = make_isin("US")
        unique_symbol = make_symbol("TSLA")
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=unique_symbol,
            isin=unique_isin,
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=10,
            price=200.00,
            total_amount=-2000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Get eligible portfolios
        result = FundMatchingService.get_eligible_portfolios_for_transaction(txn)

        assert result["match_info"]["found"] is False
        assert result["match_info"]["matched_by"] is None
        assert result["portfolios"] == []
        assert result["warning"] is not None
        assert "No fund found" in result["warning"]
        assert unique_symbol in result["warning"]

    def test_eligible_portfolios_fund_not_in_portfolio(self, app_context, db_session):
        """Test when fund exists but not in any portfolio."""
        # Create fund (not in any portfolio)
        unique_isin = make_isin("US")
        unique_symbol = make_symbol("AAPL")
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=unique_isin,
            symbol=unique_symbol,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Create IBKR transaction
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=unique_symbol,
            isin=unique_isin,
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=100,
            price=150.00,
            total_amount=-15000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Get eligible portfolios
        result = FundMatchingService.get_eligible_portfolios_for_transaction(txn)

        assert result["match_info"]["found"] is True
        assert result["match_info"]["matched_by"] == "isin"
        assert result["match_info"]["fund_id"] == fund.id
        assert result["portfolios"] == []
        assert result["warning"] is not None
        assert "exists but is not assigned" in result["warning"]
        assert "Apple Inc" in result["warning"]

    def test_eligible_portfolios_matched_by_exact_symbol(self, app_context, db_session):
        """Test match_info shows correct match method for exact symbol."""
        # Create fund with unique ISIN and symbol
        fund_isin = make_isin("US")
        unique_symbol = make_symbol("MSFT")
        fund = Fund(
            id=make_id(),
            name="Microsoft Corp",
            isin=fund_isin,
            symbol=unique_symbol,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)

        # Create portfolio
        portfolio = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Portfolio"),
            is_archived=False,
        )
        db.session.add(portfolio)

        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create IBKR transaction with different ISIN but matching symbol
        txn_isin = make_isin("US")
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=unique_symbol,  # Matches
            isin=txn_isin,  # Different ISIN
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=50,
            price=300.00,
            total_amount=-15000.00,
            currency="USD",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Get eligible portfolios
        result = FundMatchingService.get_eligible_portfolios_for_transaction(txn)

        assert result["match_info"]["found"] is True
        assert result["match_info"]["matched_by"] == "exact_symbol"

    def test_eligible_portfolios_matched_by_normalized_symbol(self, app_context, db_session):
        """Test match_info shows normalized_symbol when matched with suffix."""
        # Create fund with suffix and unique ISIN
        fund_isin = make_isin("DE")
        unique_base = make_custom_string("WEBN", 2)
        fund = Fund(
            id=make_id(),
            name="Webasto SE",
            isin=fund_isin,
            symbol=f"{unique_base}.DE",
            currency="EUR",
            exchange="XETRA",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)

        # Create portfolio
        portfolio = Portfolio(
            id=make_id(),
            name=make_portfolio_name("Portfolio"),
            is_archived=False,
        )
        db.session.add(portfolio)

        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create IBKR transaction without ISIN or exact symbol match
        txn_isin = make_isin("DE")
        txn = IBKRTransaction(
            id=make_id(),
            ibkr_transaction_id=make_ibkr_transaction_id(),
            symbol=unique_base,  # No suffix
            isin=txn_isin,  # Different
            transaction_date=date(2024, 1, 15),
            transaction_type="buy",
            quantity=100,
            price=50.00,
            total_amount=-5000.00,
            currency="EUR",
            status="pending",
            report_date=date(2024, 1, 15),
            notes="",
        )
        db.session.add(txn)
        db.session.commit()

        # Get eligible portfolios
        result = FundMatchingService.get_eligible_portfolios_for_transaction(txn)

        assert result["match_info"]["found"] is True
        assert result["match_info"]["matched_by"] == "normalized_symbol"
