"""
Test data factories for the Investment Portfolio Manager test suite.

This module provides factory_boy factories for creating test data.
Factories make it easy to create model instances with sensible defaults
while allowing customization when needed.

Usage:
    from tests.factories import PortfolioFactory, FundFactory

    # Create with defaults
    portfolio = PortfolioFactory()

    # Create with custom attributes
    portfolio = PortfolioFactory(name="My Test Portfolio", is_archived=True)

    # Create with related objects
    portfolio_fund = PortfolioFundFactory(
        portfolio=portfolio,
        fund=FundFactory(dividend_type=DividendType.CASH)
    )
"""

from datetime import date, datetime, timedelta

import factory
from app.models import (
    Dividend,
    DividendType,
    ExchangeRate,
    Fund,
    FundPrice,
    IBKRConfig,
    IBKRTransaction,
    IBKRTransactionAllocation,
    InvestmentType,
    Portfolio,
    PortfolioFund,
    RealizedGainLoss,
    ReinvestmentStatus,
    Transaction,
    db,
)
from factory import Faker, LazyAttribute, SubFactory


class BaseFactory(factory.alchemy.SQLAlchemyModelFactory):
    """
    Base factory for all model factories.

    Configures SQLAlchemy session and provides common settings.
    """

    class Meta:
        """Factory configuration."""

        abstract = True
        sqlalchemy_session = db.session
        sqlalchemy_session_persistence = "commit"


class PortfolioFactory(BaseFactory):
    """
    Factory for creating Portfolio instances.

    Attributes:
        name: Random company name
        description: Random sentence
        is_archived: False by default
        exclude_from_overview: False by default
    """

    class Meta:
        """Factory configuration."""

        model = Portfolio

    name = Faker("company")
    description = Faker("sentence")
    is_archived = False
    exclude_from_overview = False


class FundFactory(BaseFactory):
    """
    Factory for creating Fund instances.

    Attributes:
        name: Random company name
        isin: Random 12-character ISIN-like code
        symbol: Random 3-5 character symbol
        currency: USD by default
        exchange: NASDAQ by default
        investment_type: FUND by default
        dividend_type: NONE by default
    """

    class Meta:
        """Factory configuration."""

        model = Fund

    name = Faker("company")
    isin = factory.Sequence(lambda n: f"US{str(n).zfill(10)}")
    symbol = Faker("bothify", text="????", letters="ABCDEFGHIJKLMNOPQRSTUVWXYZ")
    currency = "USD"
    exchange = "NASDAQ"
    investment_type = InvestmentType.FUND
    dividend_type = DividendType.NONE


class CashDividendFundFactory(FundFactory):
    """Factory for creating funds that pay CASH dividends."""

    dividend_type = DividendType.CASH


class StockDividendFundFactory(FundFactory):
    """Factory for creating funds that pay STOCK dividends."""

    dividend_type = DividendType.STOCK


class PortfolioFundFactory(BaseFactory):
    """
    Factory for creating PortfolioFund instances (portfolio-fund relationships).

    Attributes:
        portfolio: SubFactory to create a Portfolio
        fund: SubFactory to create a Fund
    """

    class Meta:
        """Factory configuration."""

        model = PortfolioFund

    portfolio = SubFactory(PortfolioFactory)
    fund = SubFactory(FundFactory)


class TransactionFactory(BaseFactory):
    """
    Factory for creating Transaction instances.

    Attributes:
        portfolio_fund: SubFactory to create a PortfolioFund
        date: Today's date by default
        type: 'buy' by default
        shares: Random float between 1 and 100
        cost_per_share: Random float between 10 and 500
    """

    class Meta:
        """Factory configuration."""

        model = Transaction

    portfolio_fund = SubFactory(PortfolioFundFactory)
    date = LazyAttribute(lambda x: date.today())
    type = "buy"
    shares = Faker("pyfloat", min_value=1, max_value=100, right_digits=2)
    cost_per_share = Faker("pyfloat", min_value=10, max_value=500, right_digits=2)


class BuyTransactionFactory(TransactionFactory):
    """Factory for creating BUY transactions."""

    type = "buy"


class SellTransactionFactory(TransactionFactory):
    """Factory for creating SELL transactions."""

    type = "sell"


class DividendTransactionFactory(TransactionFactory):
    """Factory for creating DIVIDEND transactions (stock dividend reinvestments)."""

    type = "dividend"


class FeeTransactionFactory(TransactionFactory):
    """Factory for creating FEE transactions."""

    type = "fee"
    shares = 0  # Fees don't have shares


class DividendFactory(BaseFactory):
    """
    Factory for creating Dividend instances.

    Attributes:
        fund: SubFactory to create a Fund
        portfolio_fund: SubFactory to create a PortfolioFund
        record_date: Today's date by default
        ex_dividend_date: 2 days before record date
        shares_owned: Random float between 1 and 100
        dividend_per_share: Random float between 0.10 and 5.00
        total_amount: Calculated from shares_owned * dividend_per_share
        reinvestment_status: PENDING by default
        buy_order_date: None by default
        reinvestment_transaction_id: None by default
    """

    class Meta:
        """Factory configuration."""

        model = Dividend

    fund = SubFactory(FundFactory)
    portfolio_fund = SubFactory(PortfolioFundFactory)
    record_date = LazyAttribute(lambda x: date.today())
    ex_dividend_date = LazyAttribute(lambda obj: obj.record_date - timedelta(days=2))
    shares_owned = Faker("pyfloat", min_value=1, max_value=100, right_digits=2)
    dividend_per_share = Faker("pyfloat", min_value=0.10, max_value=5.00, right_digits=2)
    total_amount = LazyAttribute(lambda obj: obj.shares_owned * obj.dividend_per_share)
    reinvestment_status = ReinvestmentStatus.PENDING
    buy_order_date = None
    reinvestment_transaction_id = None


class CashDividendFactory(DividendFactory):
    """Factory for creating CASH dividends (auto-completed, no reinvestment)."""

    fund = SubFactory(CashDividendFundFactory)
    reinvestment_status = ReinvestmentStatus.COMPLETED


class StockDividendFactory(DividendFactory):
    """Factory for creating STOCK dividends (with reinvestment)."""

    fund = SubFactory(StockDividendFundFactory)
    reinvestment_status = ReinvestmentStatus.PENDING


class FundPriceFactory(BaseFactory):
    """
    Factory for creating FundPrice instances.

    Attributes:
        fund: SubFactory to create a Fund
        date: Today's date by default
        price: Random float between 10 and 500
    """

    class Meta:
        """Factory configuration."""

        model = FundPrice

    fund = SubFactory(FundFactory)
    date = LazyAttribute(lambda x: date.today())
    price = Faker("pyfloat", min_value=10, max_value=500, right_digits=2)


class ExchangeRateFactory(BaseFactory):
    """
    Factory for creating ExchangeRate instances.

    Attributes:
        from_currency: USD by default
        to_currency: EUR by default
        date: Today's date by default
        rate: Random float between 0.5 and 2.0
    """

    class Meta:
        """Factory configuration."""

        model = ExchangeRate

    from_currency = "USD"
    to_currency = "EUR"
    date = LazyAttribute(lambda x: date.today())
    rate = Faker("pyfloat", min_value=0.5, max_value=2.0, right_digits=4)


class RealizedGainLossFactory(BaseFactory):
    """
    Factory for creating RealizedGainLoss instances.

    Attributes:
        portfolio: SubFactory to create a Portfolio
        fund: SubFactory to create a Fund
        transaction_date: Today's date by default
        shares_sold: Random float between 1 and 50
        sale_price: Random float between 100 and 500
        average_cost: Random float between 50 and 400
        realized_gain_loss: Calculated from (sale_price - average_cost) * shares_sold
    """

    class Meta:
        """Factory configuration."""

        model = RealizedGainLoss

    portfolio = SubFactory(PortfolioFactory)
    fund = SubFactory(FundFactory)
    transaction_date = LazyAttribute(lambda x: date.today())
    shares_sold = Faker("pyfloat", min_value=1, max_value=50, right_digits=2)
    sale_price = Faker("pyfloat", min_value=100, max_value=500, right_digits=2)
    average_cost = Faker("pyfloat", min_value=50, max_value=400, right_digits=2)
    realized_gain_loss = LazyAttribute(
        lambda obj: (obj.sale_price - obj.average_cost) * obj.shares_sold
    )


class IBKRConfigFactory(BaseFactory):
    """
    Factory for creating IBKRConfig instances.

    Note: Only one IBKR config should exist in the system.

    Attributes:
        flex_token: Random encrypted token (placeholder)
        flex_query_id: Random 9-digit query ID
        token_expires_at: 90 days from now
        last_import: Now
    """

    class Meta:
        """Factory configuration."""

        model = IBKRConfig

    flex_token = Faker("sha256")
    flex_query_id = Faker("numerify", text="#########")
    token_expires_at = LazyAttribute(lambda x: datetime.now() + timedelta(days=90))
    last_import = LazyAttribute(lambda x: datetime.now())


class IBKRTransactionFactory(BaseFactory):
    """
    Factory for creating IBKRTransaction instances.

    Attributes:
        transaction_id: Random transaction ID
        transaction_type: Trade by default
        symbol: Random 3-5 character symbol
        isin: Random 12-character ISIN
        date: Today's date by default
        quantity: Random float between 1 and 100
        price: Random float between 10 and 500
        total_amount: Calculated from quantity * price
        currency: USD by default
        exchange: NASDAQ by default
        status: pending by default
        created_at: Now
    """

    class Meta:
        """Factory configuration."""

        model = IBKRTransaction

    transaction_id = Faker("numerify", text="##########")
    transaction_type = "Trade"
    symbol = Faker("bothify", text="????", letters="ABCDEFGHIJKLMNOPQRSTUVWXYZ")
    isin = factory.Sequence(lambda n: f"US{str(n).zfill(10)}")
    date = LazyAttribute(lambda x: date.today())
    quantity = Faker("pyfloat", min_value=1, max_value=100, right_digits=2)
    price = Faker("pyfloat", min_value=10, max_value=500, right_digits=2)
    total_amount = LazyAttribute(
        lambda obj: obj.quantity * obj.price if obj.quantity and obj.price else 0
    )
    currency = "USD"
    exchange = "NASDAQ"
    status = "pending"
    created_at = LazyAttribute(lambda x: datetime.now())
    processed_at = None


class IBKRTransactionAllocationFactory(BaseFactory):
    """
    Factory for creating IBKRTransactionAllocation instances.

    Attributes:
        ibkr_transaction: SubFactory to create an IBKRTransaction
        portfolio: SubFactory to create a Portfolio
        percentage: Random float between 1 and 100
        transaction: None by default (created after allocation is processed)
    """

    class Meta:
        """Factory configuration."""

        model = IBKRTransactionAllocation

    ibkr_transaction = SubFactory(IBKRTransactionFactory)
    portfolio = SubFactory(PortfolioFactory)
    percentage = Faker("pyfloat", min_value=1, max_value=100, right_digits=2)
    transaction_id = None


# Convenience functions for creating common test scenarios


def create_portfolio_with_funds(num_funds=3, **portfolio_kwargs):
    """
    Create a portfolio with multiple funds and portfolio_fund relationships.

    Args:
        num_funds (int): Number of funds to create
        **portfolio_kwargs: Additional arguments to pass to PortfolioFactory

    Returns:
        tuple: (portfolio, list of funds, list of portfolio_funds)
    """
    portfolio = PortfolioFactory(**portfolio_kwargs)
    funds = [FundFactory() for _ in range(num_funds)]
    portfolio_funds = [PortfolioFundFactory(portfolio=portfolio, fund=fund) for fund in funds]
    return portfolio, funds, portfolio_funds


def create_portfolio_with_transactions(num_transactions=5, **portfolio_kwargs):
    """
    Create a portfolio with a fund and multiple transactions.

    Args:
        num_transactions (int): Number of transactions to create
        **portfolio_kwargs: Additional arguments to pass to PortfolioFactory

    Returns:
        tuple: (portfolio, fund, portfolio_fund, list of transactions)
    """
    portfolio = PortfolioFactory(**portfolio_kwargs)
    fund = FundFactory()
    portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
    transactions = [
        TransactionFactory(portfolio_fund=portfolio_fund) for _ in range(num_transactions)
    ]
    return portfolio, fund, portfolio_fund, transactions


def create_dividend_scenario(dividend_type=DividendType.CASH, with_reinvestment=False):
    """
    Create a complete dividend scenario with portfolio, fund, and dividend.

    Args:
        dividend_type (DividendType): Type of dividend
        with_reinvestment (bool): Whether to create a reinvestment transaction

    Returns:
        tuple: (portfolio, fund, portfolio_fund, dividend, [transaction])
    """
    portfolio = PortfolioFactory()
    if dividend_type == DividendType.CASH:
        fund = CashDividendFundFactory()
    elif dividend_type == DividendType.STOCK:
        fund = StockDividendFundFactory()
    else:
        fund = FundFactory(dividend_type=dividend_type)

    portfolio_fund = PortfolioFundFactory(portfolio=portfolio, fund=fund)
    dividend = DividendFactory(fund=fund, portfolio_fund=portfolio_fund)

    if with_reinvestment and dividend_type == DividendType.STOCK:
        transaction = DividendTransactionFactory(portfolio_fund=portfolio_fund)
        dividend.reinvestment_transaction_id = transaction.id
        dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
        db.session.commit()
        return portfolio, fund, portfolio_fund, dividend, transaction

    return portfolio, fund, portfolio_fund, dividend, None
