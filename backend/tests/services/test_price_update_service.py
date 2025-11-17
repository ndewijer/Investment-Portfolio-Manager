"""
Comprehensive test suite for PriceUpdateService.

Tests TodayPriceService and HistoricalPriceService functionality including:
- Latest available price updates
- Historical price updates
- Missing date detection
- yfinance API integration (mocked)
- Duplicate price handling
- Error handling
"""

import uuid
from datetime import date, datetime, timedelta
from unittest.mock import MagicMock, patch

import pandas as pd
from app.models import Fund, FundPrice, InvestmentType, Portfolio, PortfolioFund, Transaction, db
from app.services.price_update_service import HistoricalPriceService, TodayPriceService
from tests.test_helpers import make_id, make_isin, make_symbol


class TestTodayPriceService:
    """Tests for TodayPriceService - latest available price updates."""

    def test_get_latest_available_date(self, app_context):
        """Test that latest available date returns yesterday."""
        latest_date = TodayPriceService.get_latest_available_date()

        # Should be yesterday
        yesterday = datetime.now().date() - timedelta(days=1)
        assert latest_date == yesterday

    def test_update_todays_price_no_symbol(self, app_context, db_session):
        """Test price update fails when fund has no symbol."""
        # Create fund without symbol
        fund = Fund(
            id=make_id(),
            name="Fund Without Symbol",
            isin=make_isin("US"),
            symbol=None,  # No symbol
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Try to update price
        response, status = TodayPriceService.update_todays_price(fund.id)

        # Should fail with 400
        assert status == 400
        assert "No symbol available" in response["message"]

    def test_update_todays_price_already_exists(self, app_context, db_session):
        """Test that existing latest price is not duplicated."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Apple Inc",
            isin=make_isin("US"),
            symbol=make_symbol("AAPL"),
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Create price for yesterday (latest available date)
        yesterday = datetime.now().date() - timedelta(days=1)
        existing_price = FundPrice(id=make_id(), fund_id=fund.id, date=yesterday, price=150.00)
        db.session.add(existing_price)
        db.session.commit()

        # Try to update today's price
        response, status = TodayPriceService.update_todays_price(fund.id)

        # Should return existing price without creating duplicate
        assert status == 200
        assert "already exists" in response["message"]

        # Verify no duplicate created
        prices = FundPrice.query.filter_by(fund_id=fund.id, date=yesterday).all()
        assert len(prices) == 1

    @patch("app.services.price_update_service.yf.Ticker")
    def test_update_todays_price_success(self, mock_ticker, app_context, db_session):
        """Test successful price update from yfinance."""
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

        # Mock yfinance response
        mock_history = pd.DataFrame(
            {"Close": [150.25, 151.50, 152.75]},
            index=pd.DatetimeIndex(
                [
                    datetime.now().date() - timedelta(days=3),
                    datetime.now().date() - timedelta(days=2),
                    datetime.now().date() - timedelta(days=1),
                ]
            ),
        )
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.history.return_value = mock_history
        mock_ticker.return_value = mock_ticker_instance

        # Update today's price
        response, status = TodayPriceService.update_todays_price(fund.id)

        # Should succeed
        assert status == 200
        assert "Updated latest price" in response["message"]

        # Verify price created in database
        yesterday = datetime.now().date() - timedelta(days=1)
        price = FundPrice.query.filter_by(fund_id=fund.id, date=yesterday).first()
        assert price is not None
        assert price.price == 152.75

    @patch("app.services.price_update_service.yf.Ticker")
    def test_update_todays_price_no_data(self, mock_ticker, app_context, db_session):
        """Test handling when yfinance returns no data."""
        # Create fund
        fund = Fund(
            id=make_id(),
            name="Delisted Stock",
            isin=make_isin("US"),
            symbol="DELIST",
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Mock empty yfinance response
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.history.return_value = pd.DataFrame()  # Empty
        mock_ticker.return_value = mock_ticker_instance

        # Try to update price
        response, status = TodayPriceService.update_todays_price(fund.id)

        # Should return 404
        assert status == 404
        assert "No recent price data available" in response["message"]

    @patch("app.services.price_update_service.yf.Ticker")
    def test_update_todays_price_latest_already_exists(self, mock_ticker, app_context, db_session):
        """Test when yfinance returns data but we already have that date."""
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

        # Create price for the date yfinance will return
        last_date = datetime.now().date() - timedelta(days=1)
        existing_price = FundPrice(id=make_id(), fund_id=fund.id, date=last_date, price=150.00)
        db.session.add(existing_price)
        db.session.commit()

        # Mock yfinance to return same date
        mock_history = pd.DataFrame({"Close": [150.00]}, index=pd.DatetimeIndex([last_date]))
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.history.return_value = mock_history
        mock_ticker.return_value = mock_ticker_instance

        # Try to update
        response, status = TodayPriceService.update_todays_price(fund.id)

        # Should recognize existing price
        assert status == 200
        assert "already exists" in response["message"]

    @patch("app.services.price_update_service.yf.Ticker")
    def test_update_todays_price_exception_handling(self, mock_ticker, app_context, db_session):
        """Test exception handling during price update."""
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

        # Mock yfinance to raise exception
        mock_ticker.side_effect = Exception("Network error")

        # Try to update price
        response, status = TodayPriceService.update_todays_price(fund.id)

        # Should return error
        assert status == 500
        assert "Error updating latest price" in response["message"]


class TestHistoricalPriceService:
    """Tests for HistoricalPriceService - historical price updates."""

    def test_get_oldest_transaction_date_no_transactions(self, app_context, db_session):
        """Test getting oldest transaction date when no transactions exist."""
        # Create fund with no transactions
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

        # Get oldest date
        oldest_date = HistoricalPriceService.get_oldest_transaction_date(fund.id)

        # Should be None
        assert oldest_date is None

    def test_get_oldest_transaction_date_with_transactions(self, app_context, db_session):
        """Test getting oldest transaction date with multiple transactions."""
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
        portfolio = Portfolio(id=make_id(), name=f"Portfolio {uuid.uuid4().hex[:6]}")
        db.session.add(portfolio)

        # Create portfolio-fund relationship
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transactions on different dates
        txn1 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 3, 15),
            type="buy",
            shares=100,
            cost_per_share=150.00,
        )
        txn2 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 10),  # Oldest
            type="buy",
            shares=50,
            cost_per_share=140.00,
        )
        txn3 = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 5, 20),
            type="buy",
            shares=25,
            cost_per_share=160.00,
        )
        db.session.add_all([txn1, txn2, txn3])
        db.session.commit()

        # Get oldest date
        oldest_date = HistoricalPriceService.get_oldest_transaction_date(fund.id)

        # Should be Jan 10, 2024
        assert oldest_date == date(2024, 1, 10)

    def test_get_missing_dates_no_transactions(self, app_context, db_session):
        """Test missing dates when no transactions exist."""
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

        # Get missing dates
        missing = HistoricalPriceService.get_missing_dates(fund.id)

        # Should be empty (no transactions = no required dates)
        assert missing == []

    def test_get_missing_dates_all_prices_exist(self, app_context, db_session):
        """Test missing dates when all prices already exist."""
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
        portfolio = Portfolio(id=make_id(), name=f"Portfolio {uuid.uuid4().hex[:6]}")
        db.session.add(portfolio)

        # Create portfolio-fund relationship
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transaction 3 days ago
        txn_date = date(2024, 1, 1)
        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=txn_date,
            type="buy",
            shares=100,
            cost_per_share=150.00,
        )
        db.session.add(txn)

        # Create prices for all dates from transaction to today
        current_date = txn_date
        today = datetime.now().date()
        while current_date <= today:
            price = FundPrice(id=make_id(), fund_id=fund.id, date=current_date, price=150.00)
            db.session.add(price)
            current_date += timedelta(days=1)

        db.session.commit()

        # Get missing dates
        missing = HistoricalPriceService.get_missing_dates(fund.id)

        # Should be empty
        assert missing == []

    def test_get_missing_dates_some_missing(self, app_context, db_session):
        """Test missing dates when some prices are missing."""
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
        portfolio = Portfolio(id=make_id(), name=f"Portfolio {uuid.uuid4().hex[:6]}")
        db.session.add(portfolio)

        # Create portfolio-fund relationship
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transaction on Jan 1
        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=150.00,
        )
        db.session.add(txn)

        # Create prices for Jan 1 and Jan 3 only (Jan 2 missing)
        price1 = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 1, 1), price=150.00)
        price3 = FundPrice(id=make_id(), fund_id=fund.id, date=date(2024, 1, 3), price=152.00)
        db.session.add_all([price1, price3])
        db.session.commit()

        # Get missing dates
        missing = HistoricalPriceService.get_missing_dates(fund.id)

        # Should include Jan 2 and all dates from Jan 4 to today
        assert date(2024, 1, 2) in missing
        assert date(2024, 1, 1) not in missing  # Exists
        assert date(2024, 1, 3) not in missing  # Exists

    def test_update_historical_prices_no_symbol(self, app_context, db_session):
        """Test historical price update fails when fund has no symbol."""
        # Create fund without symbol
        fund = Fund(
            id=make_id(),
            name="Fund Without Symbol",
            isin=make_isin("US"),
            symbol=None,
            currency="USD",
            exchange="NASDAQ",
            investment_type=InvestmentType.STOCK,
        )
        db.session.add(fund)
        db.session.commit()

        # Try to update historical prices
        response, status = HistoricalPriceService.update_historical_prices(fund.id)

        # Should fail with 400
        assert status == 400
        assert "No symbol available" in response["message"]

    def test_update_historical_prices_no_missing_dates(self, app_context, db_session):
        """Test historical update when no dates are missing."""
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

        # No transactions = no missing dates

        # Try to update
        response, status = HistoricalPriceService.update_historical_prices(fund.id)

        # Should return success with no updates
        assert status == 200
        assert "No missing dates" in response["message"]

    @patch("app.services.price_update_service.yf.Ticker")
    def test_update_historical_prices_success(self, mock_ticker, app_context, db_session):
        """Test successful historical price update."""
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
        portfolio = Portfolio(id=make_id(), name=f"Portfolio {uuid.uuid4().hex[:6]}")
        db.session.add(portfolio)

        # Create portfolio-fund relationship
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transaction on Jan 1
        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=150.00,
        )
        db.session.add(txn)
        db.session.commit()

        # Mock yfinance to return prices for Jan 1-3
        mock_history = pd.DataFrame(
            {"Close": [150.00, 151.00, 152.00]},
            index=pd.DatetimeIndex([date(2024, 1, 1), date(2024, 1, 2), date(2024, 1, 3)]),
        )
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.history.return_value = mock_history
        mock_ticker.return_value = mock_ticker_instance

        # Update historical prices
        response, status = HistoricalPriceService.update_historical_prices(fund.id)

        # Should succeed
        assert status == 200
        assert "Updated historical prices" in response["message"]

        # Verify prices created in database
        prices = FundPrice.query.filter_by(fund_id=fund.id).all()
        assert len(prices) >= 3

    @patch("app.services.price_update_service.yf.Ticker")
    def test_update_historical_prices_partial_data(self, mock_ticker, app_context, db_session):
        """Test historical update when yfinance has partial data."""
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
        portfolio = Portfolio(id=make_id(), name=f"Portfolio {uuid.uuid4().hex[:6]}")
        db.session.add(portfolio)

        # Create portfolio-fund relationship
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transaction on Jan 1
        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=150.00,
        )
        db.session.add(txn)
        db.session.commit()

        # Mock yfinance to return data for only Jan 1 and 3 (missing Jan 2)
        # This simulates weekend/holiday gaps
        mock_history = pd.DataFrame(
            {"Close": [150.00, 152.00]},
            index=pd.DatetimeIndex([date(2024, 1, 1), date(2024, 1, 3)]),
        )
        mock_ticker_instance = MagicMock()
        mock_ticker_instance.history.return_value = mock_history
        mock_ticker.return_value = mock_ticker_instance

        # Update historical prices
        _response, status = HistoricalPriceService.update_historical_prices(fund.id)

        # Should succeed but only update available dates
        assert status == 200

        # Verify only available dates were added
        jan1_price = FundPrice.query.filter_by(fund_id=fund.id, date=date(2024, 1, 1)).first()
        jan3_price = FundPrice.query.filter_by(fund_id=fund.id, date=date(2024, 1, 3)).first()

        assert jan1_price is not None
        assert jan3_price is not None

    @patch("app.services.price_update_service.yf.Ticker")
    def test_update_historical_prices_exception_handling(
        self, mock_ticker, app_context, db_session
    ):
        """Test exception handling during historical update."""
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
        portfolio = Portfolio(id=make_id(), name=f"Portfolio {uuid.uuid4().hex[:6]}")
        db.session.add(portfolio)

        # Create portfolio-fund relationship
        pf = PortfolioFund(id=make_id(), portfolio_id=portfolio.id, fund_id=fund.id)
        db.session.add(pf)
        db.session.commit()

        # Create transaction
        txn = Transaction(
            id=make_id(),
            portfolio_fund_id=pf.id,
            date=date(2024, 1, 1),
            type="buy",
            shares=100,
            cost_per_share=150.00,
        )
        db.session.add(txn)
        db.session.commit()

        # Mock yfinance to raise exception
        mock_ticker.side_effect = Exception("API error")

        # Try to update
        response, status = HistoricalPriceService.update_historical_prices(fund.id)

        # Should return error
        assert status == 500
        assert "Error updating historical prices" in response["message"]
