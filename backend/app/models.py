from datetime import datetime
from flask_sqlalchemy import SQLAlchemy
from enum import Enum
import uuid

db = SQLAlchemy()

def generate_uuid():
    return str(uuid.uuid4())

class DividendType(Enum):
    NONE = 'none'
    CASH = 'cash'
    STOCK = 'stock'

class ReinvestmentStatus(Enum):
    PENDING = 'pending'
    COMPLETED = 'completed'
    PARTIAL = 'partial'

class Portfolio(db.Model):
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    name = db.Column(db.String(100), nullable=False)
    description = db.Column(db.Text)
    is_archived = db.Column(db.Boolean, default=False)
    funds = db.relationship('PortfolioFund', backref='portfolio', lazy=True, 
                          cascade='all, delete-orphan')

class Fund(db.Model):
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    name = db.Column(db.String(100), nullable=False)
    isin = db.Column(db.String(12), unique=True, nullable=False)
    currency = db.Column(db.String(3), nullable=False)
    exchange = db.Column(db.String(50), nullable=False)
    historical_prices = db.relationship('FundPrice', backref='fund', lazy=True)
    portfolios = db.relationship('PortfolioFund', backref='fund', lazy=True)
    dividend_type = db.Column(db.Enum(DividendType), nullable=False, default=DividendType.NONE)
    dividends = db.relationship('Dividend', backref='fund', lazy=True)

class PortfolioFund(db.Model):
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    portfolio_id = db.Column(db.String(36), db.ForeignKey('portfolio.id', ondelete='CASCADE'), nullable=False)
    fund_id = db.Column(db.String(36), db.ForeignKey('fund.id'), nullable=False)
    transactions = db.relationship('Transaction', backref='portfolio_fund', lazy=True,
                                 cascade='all, delete-orphan')
    
    __table_args__ = (
        db.UniqueConstraint('portfolio_id', 'fund_id', name='unique_portfolio_fund'),
    )

class Transaction(db.Model):
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    portfolio_fund_id = db.Column(db.String(36), db.ForeignKey('portfolio_fund.id', ondelete='CASCADE'), nullable=False)
    date = db.Column(db.Date, nullable=False)
    type = db.Column(db.String(10), nullable=False)  # 'buy', 'sell', or 'dividend'
    shares = db.Column(db.Float, nullable=False)
    cost_per_share = db.Column(db.Float, nullable=False)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

class FundPrice(db.Model):
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    fund_id = db.Column(db.String(36), db.ForeignKey('fund.id'), nullable=False)
    date = db.Column(db.Date, nullable=False)
    price = db.Column(db.Float, nullable=False)

class ExchangeRate(db.Model):
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    from_currency = db.Column(db.String(3), nullable=False)
    to_currency = db.Column(db.String(3), nullable=False)
    rate = db.Column(db.Float, nullable=False)
    date = db.Column(db.Date, nullable=False)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    __table_args__ = (
        db.UniqueConstraint('from_currency', 'to_currency', 'date', name='unique_exchange_rate'),
    )

class Dividend(db.Model):
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    fund_id = db.Column(db.String(36), db.ForeignKey('fund.id'), nullable=False)
    portfolio_fund_id = db.Column(db.String(36), db.ForeignKey('portfolio_fund.id'), nullable=False)
    record_date = db.Column(db.Date, nullable=False)
    ex_dividend_date = db.Column(db.Date, nullable=False)
    shares_owned = db.Column(db.Float, nullable=False)
    dividend_per_share = db.Column(db.Float, nullable=False)
    total_amount = db.Column(db.Float, nullable=False)
    reinvestment_status = db.Column(db.Enum(ReinvestmentStatus), nullable=False, default=ReinvestmentStatus.PENDING)
    buy_order_date = db.Column(db.Date, nullable=True)
    reinvestment_transaction_id = db.Column(db.String(36), db.ForeignKey('transaction.id'), nullable=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
