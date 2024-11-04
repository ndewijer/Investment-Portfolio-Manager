from datetime import datetime, timedelta
from .models import (
    db, Portfolio, Fund, PortfolioFund, Transaction, FundPrice, 
    DividendType, Dividend, Log, LogLevel, LogCategory
)
import uuid

def seed_database():
    """Seed the database with initial data"""
    
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
            http_status=200
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.WARNING,
            category=LogCategory.DATABASE,
            message="Database connection pool reaching capacity",
            details='{"pool_size": 80, "max_size": 100}',
            source="connection_pool_monitor",
            http_status=None
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.ERROR,
            category=LogCategory.SECURITY,
            message="Failed login attempt",
            details='{"ip_address": "192.168.1.1", "attempt_count": 3}',
            source="auth_service",
            http_status=401
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.INFO,
            category=LogCategory.PORTFOLIO,
            message="Portfolio created",
            details='{"portfolio_name": "Test Portfolio", "user_id": "123"}',
            source="portfolio_service",
            http_status=201
        ),
        Log(
            id=str(uuid.uuid4()),
            level=LogLevel.DEBUG,
            category=LogCategory.TRANSACTION,
            message="Transaction validation",
            details='{"validation_rules": ["date_check", "balance_check"]}',
            source="transaction_validator",
            http_status=None
        )
    ]
    db.session.add_all(logs)
    db.session.commit()

    # Create Portfolios
    portfolios = [
        Portfolio(
            name="Marit",
            description="Marit's Portfolio",
            is_archived=False
        ),
        Portfolio(
            name="Tobias",
            description="Tobias's Portfolio",
            is_archived=False
        ),
        Portfolio(
            name="PytNick",
            description="Pytrik and Nick's Portfolio",
            is_archived=False
        ),
        Portfolio(
            name="Test Portfolio",
            description="A test portfolio that is archived",
            is_archived=True
        ),
        Portfolio(
            name="Empty Portfolio",
            description="A portfolio without any funds or transactions",
            is_archived=False
        )
    ]
    db.session.add_all(portfolios)
    db.session.commit()

    # Create Funds with dividend types
    funds = [
        Fund(
            name="Goldman Sachs Enhanced Index Sustainable Equity",
            isin="NL0012125736",
            currency="EUR",
            exchange="XAMS",
            dividend_type=DividendType.STOCK  # Update dividend type
        ),
        Fund(
            name="Goldman Sachs Enhanced Index Sustainable EM Equity",
            isin="NL0006311771",
            currency="EUR",
            exchange="XAMS",
            dividend_type=DividendType.CASH   # Update dividend type
        ),
        Fund(
            name="Test Fund",
            isin="TEST000TEST",
            currency="USD",
            exchange="NYSE",
            dividend_type=DividendType.NONE   # Update dividend type
        )
    ]
    db.session.add_all(funds)
    db.session.commit()

    # Create Portfolio-Fund relationships
    portfolio_funds = [
        # Marit's Portfolio
        PortfolioFund(
            portfolio_id=portfolios[0].id,
            fund_id=funds[0].id
        ),
        
        # Tobias's Portfolio
        PortfolioFund(
            portfolio_id=portfolios[1].id,
            fund_id=funds[0].id
        ),
       
        # PytNick's Portfolio
        PortfolioFund(
            portfolio_id=portfolios[2].id,
            fund_id=funds[0].id
        ),
        PortfolioFund(
            portfolio_id=portfolios[2].id,
            fund_id=funds[1].id
        ),

        # Archived Portfolio
        PortfolioFund(
            portfolio_id=portfolios[3].id,
            fund_id=funds[2].id
        )
    ]
    db.session.add_all(portfolio_funds)
    db.session.commit()

    # Log the seeding completion
    completion_log = Log(
        id=str(uuid.uuid4()),
        level=LogLevel.INFO,
        category=LogCategory.SYSTEM,
        message="Database seeding completed",
        details=f'{{"portfolios": {len(portfolios)}, "funds": {len(funds)}, "portfolio_funds": {len(portfolio_funds)}}}',
        source="seed_database",
        http_status=None
    )
    db.session.add(completion_log)
    db.session.commit() 