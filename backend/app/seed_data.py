from datetime import datetime, timedelta
from .models import db, Portfolio, Fund, PortfolioFund, Transaction, FundPrice

def seed_database():
    """Seed the database with initial data"""
    
    # Clear existing data
    Transaction.query.delete()
    PortfolioFund.query.delete()
    FundPrice.query.delete()
    Fund.query.delete()
    Portfolio.query.delete()
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

    # Create Funds
    funds = [
        Fund(
            name="Goldman Sachs Enhanced Index Sustainable Equity",
            isin="NL0012125736",
            currency="EUR",
            exchange="XAMS"
        ),
        Fund(
            name="Goldman Sachs Enhanced Index Sustainable EM Equity",
            isin="NL0006311771",
            currency="EUR",
            exchange="XAMS"
        ),
        Fund(
            name="Test Fund",
            isin="TEST000TEST",
            currency="USD",
            exchange="NYSE"
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

    # # Create historical prices for the last 30 days
    # today = datetime.now().date()
    # for fund in funds:
    #     # Different base prices for different funds
    #     if fund.currency == 'EUR':
    #         base_price = 31.67  # Example EUR price
    #     else:
    #         base_price = 35.00  # Example USD price

    #     for days_ago in range(30, -1, -1):
    #         date = today - timedelta(days=days_ago)
    #         # Create some price variation (more realistic)
    #         daily_change = base_price * (0.002 * (days_ago % 5 - 2))  # 0.2% daily variation
    #         price = base_price + daily_change
            
    #         fund_price = FundPrice(
    #             fund_id=fund.id,
    #             date=date,
    #             price=price
    #         )
    #         db.session.add(fund_price)
    #         base_price = price  # Use this as the base for the next day
    
    # db.session.commit()

    # # Create sample transactions
    # transactions = [
    #     # Marit's Portfolio transactions
    #     Transaction(
    #         portfolio_fund_id=portfolio_funds[0].id,
    #         date=(today - timedelta(days=25)),
    #         type='buy',
    #         shares=15.787812,
    #         cost_per_share=31.67
    #     ),
        
    #     # Tobias's Portfolio transactions
    #     Transaction(
    #         portfolio_fund_id=portfolio_funds[1].id,
    #         date=(today - timedelta(days=20)),
    #         type='buy',
    #         shares=12.345678,
    #         cost_per_share=31.89
    #     ),

    #     # PytNick's Portfolio transactions
    #     Transaction(
    #         portfolio_fund_id=portfolio_funds[2].id,  # Enhanced Index Sustainable Equity
    #         date=(today - timedelta(days=15)),
    #         type='buy',
    #         shares=20.123456,
    #         cost_per_share=31.95
    #     ),
    #     Transaction(
    #         portfolio_fund_id=portfolio_funds[3].id,  # Enhanced Index Sustainable EM Equity
    #         date=(today - timedelta(days=10)),
    #         type='buy',
    #         shares=18.765432,
    #         cost_per_share=31.78
    #     ),

    #     # Archived Portfolio transactions
    #     Transaction(
    #         portfolio_fund_id=portfolio_funds[4].id,
    #         date=(today - timedelta(days=5)),
    #         type='buy',
    #         shares=10.0,
    #         cost_per_share=35.00
    #     )
    # ]
    # db.session.add_all(transactions)
    db.session.commit() 