from datetime import datetime
from flask_sqlalchemy import SQLAlchemy

db = SQLAlchemy()

class Portfolio(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False)
    description = db.Column(db.Text)
    is_archived = db.Column(db.Boolean, default=False)
    funds = db.relationship('PortfolioFund', backref='portfolio', lazy=True, 
                          cascade='all, delete-orphan')

class Fund(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False)
    isin = db.Column(db.String(12), unique=True, nullable=False)
    currency = db.Column(db.String(3), nullable=False)
    exchange = db.Column(db.String(50), nullable=False)
    historical_prices = db.relationship('FundPrice', backref='fund', lazy=True)
    portfolios = db.relationship('PortfolioFund', backref='fund', lazy=True)

class PortfolioFund(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    portfolio_id = db.Column(db.Integer, db.ForeignKey('portfolio.id', ondelete='CASCADE'), nullable=False)
    fund_id = db.Column(db.Integer, db.ForeignKey('fund.id'), nullable=False)
    transactions = db.relationship('Transaction', backref='portfolio_fund', lazy=True,
                                 cascade='all, delete-orphan')
    
    __table_args__ = (
        db.UniqueConstraint('portfolio_id', 'fund_id', name='unique_portfolio_fund'),
    )

class Transaction(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    portfolio_fund_id = db.Column(db.Integer, db.ForeignKey('portfolio_fund.id', ondelete='CASCADE'), nullable=False)
    date = db.Column(db.Date, nullable=False)
    type = db.Column(db.String(4), nullable=False)  # 'buy' or 'sell'
    shares = db.Column(db.Float, nullable=False)
    cost_per_share = db.Column(db.Float, nullable=False)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

class FundPrice(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    fund_id = db.Column(db.Integer, db.ForeignKey('fund.id'), nullable=False)
    date = db.Column(db.Date, nullable=False)
    price = db.Column(db.Float, nullable=False)

class ExchangeRate(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    from_currency = db.Column(db.String(3), nullable=False)
    to_currency = db.Column(db.String(3), nullable=False)
    rate = db.Column(db.Float, nullable=False)
    date = db.Column(db.Date, nullable=False)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    __table_args__ = (
        db.UniqueConstraint('from_currency', 'to_currency', 'date', name='unique_exchange_rate'),
    )
