"""
Database seeding module for development and testing.

This module provides functionality to:
- Clear existing data from all tables
- Create sample portfolios, funds, and transactions
- Generate test data for development and testing purposes
- Create sample system logs with various levels and categories
"""

import csv
import os
import random
import uuid
from datetime import datetime, timedelta

from .models import (
    Dividend,
    DividendType,
    Fund,
    FundPrice,
    Log,
    LogCategory,
    LogLevel,
    Portfolio,
    PortfolioFund,
    ReinvestmentStatus,
    SymbolInfo,
    Transaction,
    db,
)


def load_fund_prices(fund_id, isin):
    """Load historical prices from CSV file.

    Args:
        fund_id (str): The ID of the fund to load prices for.
        isin (str): The ISIN of the fund to load prices for.

    Returns:
        list: A list of FundPrice objects.
    """
    prices = []
    csv_path = os.path.join(
        os.path.dirname(__file__),
        "..",
        "data",
        "seed",
        "funds",
        "prices",
        f"{isin}.csv",
    )

    with open(csv_path) as file:
        reader = csv.DictReader(file)
        for row in reader:
            prices.append(
                FundPrice(
                    id=str(uuid.uuid4()),
                    fund_id=fund_id,
                    date=datetime.strptime(row["date"], "%Y-%m-%d").date(),
                    price=float(row["price"]),
                )
            )
    return prices


def generate_transactions(portfolio_fund_id, fund_prices, num_transactions=50):
    """Generate realistic transactions based on price history.

    Args:
        portfolio_fund_id (str): The ID of the portfolio fund to generate transactions for.
        fund_prices (list): A list of FundPrice objects.
        num_transactions (int): The number of transactions to generate.

    Returns:
        list: A list of Transaction objects.
    """
    transactions = []
    sorted_prices = sorted(fund_prices, key=lambda x: x.date)
    available_dates = [price.date for price in sorted_prices]

    # Initial investment
    initial_date = available_dates[0]
    initial_price = next(p.price for p in sorted_prices if p.date == initial_date)
    transactions.append(
        Transaction(
            id=str(uuid.uuid4()),
            portfolio_fund_id=portfolio_fund_id,
            date=initial_date,
            type="buy",
            shares=round(10000 / initial_price, 6),  # €10,000 initial investment
            cost_per_share=initial_price,
        )
    )

    # Generate random transactions throughout the period
    for _ in range(num_transactions - 1):
        date = random.choice(available_dates[1:])
        price = next(p.price for p in sorted_prices if p.date == date)

        # 70% chance of buy, 30% chance of sell
        transaction_type = "buy" if random.random() < 0.7 else "sell"
        amount = random.uniform(500, 5000)  # Random amount between €500 and €5000
        shares = round(amount / price, 6)

        transactions.append(
            Transaction(
                id=str(uuid.uuid4()),
                portfolio_fund_id=portfolio_fund_id,
                date=date,
                type=transaction_type,
                shares=shares,
                cost_per_share=price,
            )
        )

    return sorted(transactions, key=lambda x: x.date)


def generate_dividends(portfolio_fund_id, fund_id, fund_prices, dividend_type):
    """Generate yearly dividend payments.

    Args:
        portfolio_fund_id (str): The ID of the portfolio fund to generate dividends for.
        fund_id (str): The ID of the fund to generate dividends for.
        fund_prices (list): A list of FundPrice objects.
        dividend_type (DividendType): The type of dividend to generate.

    Returns:
        list: A list of Dividend objects.
    """
    dividends = []
    sorted_prices = sorted(fund_prices, key=lambda x: x.date)
    years = {price.date.year for price in sorted_prices}

    # Get all transactions for this portfolio_fund
    all_transactions = Transaction.query.filter_by(portfolio_fund_id=portfolio_fund_id).all()

    for year in years:
        # Set dividend date to April 15th of each year
        dividend_date = datetime(year, 4, 15).date()
        if dividend_date < sorted_prices[0].date:
            continue

        # Find closest price before dividend date
        price = next((p.price for p in sorted_prices if p.date <= dividend_date), None)
        if not price:
            continue

        # Calculate shares owned at dividend date
        shares_owned = 0
        for transaction in all_transactions:
            if transaction.date <= dividend_date:
                if transaction.type == "buy":
                    shares_owned += transaction.shares
                elif transaction.type == "sell":
                    shares_owned -= transaction.shares

        if shares_owned <= 0:
            continue

        # Generate dividend amount (2-4% yearly return)
        dividend_rate = random.uniform(0.02, 0.04)
        dividend_per_share = price * dividend_rate

        # Create dividend record
        dividend = Dividend(
            id=str(uuid.uuid4()),
            fund_id=fund_id,
            portfolio_fund_id=portfolio_fund_id,
            record_date=dividend_date - timedelta(days=10),
            ex_dividend_date=dividend_date - timedelta(days=5),
            shares_owned=shares_owned,
            dividend_per_share=dividend_per_share,
            total_amount=shares_owned * dividend_per_share,
            reinvestment_status=(
                ReinvestmentStatus.PENDING
                if dividend_type == DividendType.STOCK
                else ReinvestmentStatus.COMPLETED
            ),
        )
        dividends.append(dividend)

        # For stock dividends, create reinvestment transaction
        if dividend_type == DividendType.STOCK:
            reinvestment_shares = dividend.total_amount / price
            reinvestment_transaction = Transaction(
                id=str(uuid.uuid4()),
                portfolio_fund_id=portfolio_fund_id,
                date=dividend_date + timedelta(days=2),  # 2 days after dividend date
                type="buy",
                shares=reinvestment_shares,
                cost_per_share=price,
            )
            dividend.reinvestment_transaction_id = reinvestment_transaction.id
            dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
            dividend.buy_order_date = reinvestment_transaction.date

            # Store transaction on dividend object for later extraction
            dividend.reinvestment_transaction = reinvestment_transaction

            # Add reinvestment transaction to list
            all_transactions.append(reinvestment_transaction)

    return dividends


def seed_database():
    """
    Seeds the database with initial test data.

    This function performs the following:
    1. Clears all existing data from relevant tables
    2. Creates sample logs with various levels and categories
    3. Creates test portfolios:
       - Marit's Portfolio
       - Tobias's Portfolio
       - PytNick's Portfolio
       - Test Portfolio (archived)
       - Empty Portfolio
    4. Creates test funds:
       - Goldman Sachs Enhanced Index Sustainable Equity (stock dividends)
       - Goldman Sachs Enhanced Index Sustainable EM Equity (cash dividends)
       - Test Fund (no dividends)
    5. Creates portfolio-fund relationships
    6. Logs the completion of the seeding process

    Returns:
        None

    Note:
        This function is intended for development and testing purposes only.
    """

    # Clear existing data
    Log.query.delete()
    Transaction.query.delete()
    Dividend.query.delete()
    PortfolioFund.query.delete()
    FundPrice.query.delete()
    Fund.query.delete()
    Portfolio.query.delete()
    db.session.commit()

    # Create sample logs
    logs = [
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Application started",
            details='{"startup_time": "2024-01-01T00:00:00Z"}',
            source="seed_database",
            http_status=200,
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.WARNING,
            category=LogCategory.DATABASE,
            message="Database connection pool reaching capacity",
            details='{"pool_size": 80, "max_size": 100}',
            source="connection_pool_monitor",
            http_status=None,
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.ERROR,
            category=LogCategory.SECURITY,
            message="Failed login attempt",
            details='{"ip_address": "192.168.1.1", "attempt_count": 3}',
            source="auth_service",
            http_status=401,
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.INFO,
            category=LogCategory.PORTFOLIO,
            message="Portfolio created",
            details='{"portfolio_name": "Test Portfolio", "user_id": "123"}',
            source="portfolio_service",
            http_status=201,
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.DEBUG,
            category=LogCategory.TRANSACTION,
            message="Transaction validation",
            details='{"validation_rules": ["date_check", "balance_check"]}',
            source="transaction_validator",
            http_status=None,
        ),
    ]
    db.session.add_all(logs)
    db.session.commit()

    # Create Portfolios
    portfolios = [
        Portfolio(name="Kirsten", description="Kirsten's Portfolio", is_archived=False),
        Portfolio(name="Thomas", description="Thomas's Portfolio", is_archived=False),
        Portfolio(
            name="Test Portfolio",
            description="A test portfolio that is archived",
            is_archived=True,
        ),
        Portfolio(
            name="Empty Portfolio",
            description="A portfolio without any funds or transactions",
            is_archived=False,
        ),
    ]
    db.session.add_all(portfolios)
    db.session.commit()

    # Create Funds with real data
    funds = [
        Fund(
            name="Vanguard Total Stock Market ETF",
            isin="US9229087690",
            symbol="VTI",
            currency="USD",
            exchange="XNAS",
            dividend_type=DividendType.CASH,
        ),
        Fund(
            name="Amundi Prime All Country World UCITS ETF Acc",
            isin="IE0003XJA0J9",
            symbol="WEBN.DE",
            currency="EUR",
            exchange="XETR",
            dividend_type=DividendType.STOCK,
        ),
        Fund(
            name="Apple Inc.",
            isin="US0378331005",
            symbol="AAPL",
            currency="USD",
            exchange="XNAS",
            dividend_type=DividendType.CASH,
        ),
    ]
    db.session.add_all(funds)
    db.session.commit()

    # Load historical prices
    fund_prices = []
    for fund in funds:
        fund_prices.extend(load_fund_prices(fund.id, fund.isin))
    db.session.add_all(fund_prices)
    db.session.commit()

    # Create Portfolio-Fund relationships with transactions
    portfolio_funds = []
    reinvestment_transactions = []
    dividends = []

    for portfolio in portfolios[:2]:  # First 2 portfolios
        for fund in funds:
            pf = PortfolioFund(portfolio_id=portfolio.id, fund_id=fund.id)
            portfolio_funds.append(pf)
            db.session.add(pf)
            db.session.commit()

            # Generate transactions and dividends
            fund_specific_prices = [fp for fp in fund_prices if fp.fund_id == fund.id]

            # Generate initial transactions
            portfolio_transactions = generate_transactions(pf.id, fund_specific_prices)

            # Add transactions to database so they're available for dividend calculation
            db.session.add_all(portfolio_transactions)
            db.session.commit()

            # Generate dividends and their reinvestment transactions
            portfolio_dividends = generate_dividends(
                pf.id, fund.id, fund_specific_prices, fund.dividend_type
            )

            # Extract reinvestment transactions from dividends
            for dividend in portfolio_dividends:
                if hasattr(dividend, "reinvestment_transaction"):
                    reinvestment_transactions.append(dividend.reinvestment_transaction)

            dividends.extend(portfolio_dividends)

    # Commit reinvestment transactions first (dividends have foreign keys to these)
    db.session.add_all(reinvestment_transactions)
    db.session.commit()

    # Now commit dividends (after their referenced transactions exist)
    db.session.add_all(dividends)
    db.session.commit()

    # Create SymbolInfo records
    symbol_info = [
        SymbolInfo(
            symbol="GSESA.AS",
            name="Goldman Sachs Enhanced Index Sustainable Equity",
            exchange="XAMS",
            currency="EUR",
            isin="NL0012125736",
            data_source="manual",
            is_valid=True,
        ),
        SymbolInfo(
            symbol="GSEME.AS",
            name="Goldman Sachs Enhanced Index Sustainable EM Equity",
            exchange="XAMS",
            currency="EUR",
            isin="NL0006311771",
            data_source="manual",
            is_valid=True,
        ),
    ]
    db.session.add_all(symbol_info)
    db.session.commit()

    # Log the seeding completion
    completion_log = Log(
        id=str(uuid.uuid4()),
        level=LogLevel.INFO,
        category=LogCategory.SYSTEM,
        message="Database seeding completed",
        details=str(
            {
                "portfolios": len(portfolios),
                "funds": len(funds),
                "portfolio_funds": len(portfolio_funds),
            }
        ),
        source="seed_database",
        http_status=None,
    )
    db.session.add(completion_log)
    db.session.commit()
