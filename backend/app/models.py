"""
Database models for the investment portfolio application.

This module defines SQLAlchemy models for:
- Portfolios and Funds
- Transactions and Dividends
- Exchange Rates and Fund Prices
- System Settings and Logging
- Symbol Information Cache

It also includes enums for various status types and configurations.
"""

import uuid
from enum import Enum

from flask_sqlalchemy import SQLAlchemy
from sqlalchemy import event
from sqlalchemy.engine import Engine

db = SQLAlchemy()


@event.listens_for(Engine, "connect")
def set_sqlite_pragma(dbapi_connection, connection_record):
    """
    SQLite connection event listener to set timezone pragma.

    Args:
        dbapi_connection: SQLite database connection
        connection_record: Connection record object
    """
    cursor = dbapi_connection.cursor()
    cursor.execute("PRAGMA timezone = 'UTC';")
    cursor.close()


def generate_uuid():
    """
    Generate a UUID string.

    Returns:
        str: New UUID in string format
    """
    return str(uuid.uuid4())


class DividendType(Enum):
    """
    Enumeration of possible dividend types for funds.

    Attributes:
        NONE: Fund does not pay dividends
        CASH: Fund pays cash dividends
        STOCK: Fund pays stock dividends
    """

    NONE = "none"
    CASH = "cash"
    STOCK = "stock"


class ReinvestmentStatus(Enum):
    """
    Enumeration of possible dividend reinvestment statuses.

    Attributes:
        PENDING: Reinvestment is pending
        COMPLETED: Reinvestment has been completed
        PARTIAL: Reinvestment is partially completed
    """

    PENDING = "pending"
    COMPLETED = "completed"
    PARTIAL = "partial"


class InvestmentType(Enum):
    """
    Enumeration of possible investment types.

    Attributes:
        FUND: Investment is a mutual fund or ETF
        STOCK: Investment is a stock
    """

    FUND = "fund"
    STOCK = "stock"


class Portfolio(db.Model):
    """
    Represents an investment portfolio in the system.

    Attributes:
        id (str): Unique identifier (UUID)
        name (str): Portfolio name
        description (str): Optional portfolio description
        is_archived (bool): Whether the portfolio is archived
        exclude_from_overview (bool): Whether to exclude this portfolio from overview
        funds (relationship): Related PortfolioFund entries
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    name = db.Column(db.String(100), nullable=False)
    description = db.Column(db.Text)
    is_archived = db.Column(db.Boolean, default=False)
    exclude_from_overview = db.Column(db.Boolean, default=False)
    funds = db.relationship(
        "PortfolioFund", backref="portfolio", lazy=True, cascade="all, delete-orphan"
    )


class Fund(db.Model):
    """
    Represents an investment fund or stock in the system.

    Attributes:
        id (str): Unique identifier (UUID)
        name (str): Fund name
        isin (str): International Securities Identification Number
        symbol (str): Trading symbol (optional)
        currency (str): Trading currency code
        exchange (str): Exchange where the fund is traded
        investment_type (InvestmentType): Type of investment (fund/stock)
        dividend_type (DividendType): Type of dividend the fund pays
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    name = db.Column(db.String(100), nullable=False)
    isin = db.Column(db.String(12), unique=True, nullable=False)
    symbol = db.Column(db.String(10), nullable=True)
    currency = db.Column(db.String(3), nullable=False)
    exchange = db.Column(db.String(50), nullable=False)
    investment_type = db.Column(
        db.Enum(InvestmentType), nullable=False, default=InvestmentType.FUND
    )
    historical_prices = db.relationship("FundPrice", backref="fund", lazy=True)
    portfolios = db.relationship("PortfolioFund", backref="fund", lazy=True)
    dividend_type = db.Column(
        db.Enum(DividendType), nullable=False, default=DividendType.NONE
    )
    dividends = db.relationship("Dividend", backref="fund", lazy=True)


class PortfolioFund(db.Model):
    """
    Represents a relationship between a portfolio and a fund.

    Attributes:
        id (str): Unique identifier (UUID)
        portfolio_id (str): Foreign key to Portfolio
        fund_id (str): Foreign key to Fund
        transactions (relationship): Related Transaction entries
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    portfolio_id = db.Column(
        db.String(36), db.ForeignKey("portfolio.id", ondelete="CASCADE"), nullable=False
    )
    fund_id = db.Column(db.String(36), db.ForeignKey("fund.id"), nullable=False)
    transactions = db.relationship(
        "Transaction", backref="portfolio_fund", lazy=True, cascade="all, delete-orphan"
    )

    __table_args__ = (
        db.UniqueConstraint("portfolio_id", "fund_id", name="unique_portfolio_fund"),
    )


class Transaction(db.Model):
    """
    Represents a transaction in a portfolio.

    Attributes:
        id (str): Unique identifier (UUID)
        portfolio_fund_id (str): Foreign key to PortfolioFund
        date (date): Transaction date
        type (str): Transaction type ('buy', 'sell', or 'dividend')
        shares (float): Number of shares involved
        cost_per_share (float): Cost per share in fund's currency
        created_at (datetime): Transaction creation timestamp
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    portfolio_fund_id = db.Column(
        db.String(36),
        db.ForeignKey("portfolio_fund.id", ondelete="CASCADE"),
        nullable=False,
    )
    date = db.Column(db.Date, nullable=False)
    type = db.Column(db.String(10), nullable=False)  # 'buy', 'sell', or 'dividend'
    shares = db.Column(db.Float, nullable=False)
    cost_per_share = db.Column(db.Float, nullable=False)
    created_at = db.Column(db.DateTime, default=db.func.current_timestamp())


class FundPrice(db.Model):
    """
    Represents historical price data for a fund.

    Attributes:
        id (str): Unique identifier (UUID)
        fund_id (str): Foreign key to Fund
        date (date): Price date
        price (float): Price value in fund's currency
        created_at (datetime): Price record creation timestamp
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    fund_id = db.Column(db.String(36), db.ForeignKey("fund.id"), nullable=False)
    date = db.Column(db.Date, nullable=False)
    price = db.Column(db.Float, nullable=False)


class ExchangeRate(db.Model):
    """
    Represents exchange rates between currency pairs.

    Attributes:
        id (str): Unique identifier (UUID)
        from_currency (str): Source currency code
        to_currency (str): Target currency code
        rate (float): Exchange rate value
        date (date): Rate date
        created_at (datetime): Rate creation timestamp
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    from_currency = db.Column(db.String(3), nullable=False)
    to_currency = db.Column(db.String(3), nullable=False)
    rate = db.Column(db.Float, nullable=False)
    date = db.Column(db.Date, nullable=False)
    created_at = db.Column(db.DateTime, default=db.func.current_timestamp())

    __table_args__ = (
        db.UniqueConstraint(
            "from_currency", "to_currency", "date", name="unique_exchange_rate"
        ),
    )


class Dividend(db.Model):
    """
    Represents dividend payments for funds.

    Attributes:
        id (str): Unique identifier (UUID)
        fund_id (str): Foreign key to Fund
        portfolio_fund_id (str): Foreign key to PortfolioFund
        record_date (date): Dividend record date
        ex_dividend_date (date): Ex-dividend date
        shares_owned (float): Number of shares owned
        dividend_per_share (float): Dividend amount per share
        total_amount (float): Total dividend amount
        reinvestment_status (ReinvestmentStatus): Current reinvestment status
        buy_order_date (date): Date of reinvestment purchase
        reinvestment_transaction_id (str): Foreign key to reinvestment Transaction
        created_at (datetime): Dividend record creation timestamp
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    fund_id = db.Column(db.String(36), db.ForeignKey("fund.id"), nullable=False)
    portfolio_fund_id = db.Column(
        db.String(36), db.ForeignKey("portfolio_fund.id"), nullable=False
    )
    record_date = db.Column(db.Date, nullable=False)
    ex_dividend_date = db.Column(db.Date, nullable=False)
    shares_owned = db.Column(db.Float, nullable=False)
    dividend_per_share = db.Column(db.Float, nullable=False)
    total_amount = db.Column(db.Float, nullable=False)
    reinvestment_status = db.Column(
        db.Enum(ReinvestmentStatus), nullable=False, default=ReinvestmentStatus.PENDING
    )
    buy_order_date = db.Column(db.Date, nullable=True)
    reinvestment_transaction_id = db.Column(
        db.String(36), db.ForeignKey("transaction.id"), nullable=True
    )
    created_at = db.Column(db.DateTime, default=db.func.current_timestamp())


class LogLevel(Enum):
    """
    Enumeration of possible log levels.

    Attributes:
        DEBUG: Debug-level messages
        INFO: Informational messages
        WARNING: Warning messages
        ERROR: Error messages
        CRITICAL: Critical error messages
    """

    DEBUG = "debug"
    INFO = "info"
    WARNING = "warning"
    ERROR = "error"
    CRITICAL = "critical"


class LogCategory(Enum):
    """
    Enumeration of possible log categories.

    Attributes:
        PORTFOLIO: Portfolio-related logs
        FUND: Fund-related logs
        TRANSACTION: Transaction-related logs
        DIVIDEND: Dividend-related logs
        SYSTEM: System-level logs
        DATABASE: Database-related logs
        SECURITY: Security-related logs
    """

    PORTFOLIO = "portfolio"
    FUND = "fund"
    TRANSACTION = "transaction"
    DIVIDEND = "dividend"
    SYSTEM = "system"
    DATABASE = "database"
    SECURITY = "security"


class Log(db.Model):
    """
    System logging model for tracking application events.

    Attributes:
        id (str): Unique identifier (UUID)
        timestamp (datetime): Event timestamp
        level (LogLevel): Severity level
        category (LogCategory): Event category
        message (str): Log message
        details (str): JSON-formatted additional details
        source (str): Event source (function/method name)
        user_id (str): Associated user ID if applicable
        request_id (str): Associated request ID
        stack_trace (str): Error stack trace if applicable
        http_status (int): HTTP status code if applicable
        ip_address (str): Client IP address
        user_agent (str): Client user agent string
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    timestamp = db.Column(
        db.DateTime, nullable=False, default=db.func.current_timestamp()
    )
    level = db.Column(db.Enum(LogLevel), nullable=False)
    category = db.Column(db.Enum(LogCategory), nullable=False)
    message = db.Column(db.Text, nullable=False)
    details = db.Column(db.Text, nullable=True)  # JSON string for structured data
    source = db.Column(db.String(255), nullable=False)  # Function/method name
    user_id = db.Column(db.String(36), nullable=True)  # For future user authentication
    request_id = db.Column(db.String(36), nullable=True)  # To track request flow
    stack_trace = db.Column(db.Text, nullable=True)
    http_status = db.Column(db.Integer, nullable=True)
    ip_address = db.Column(db.String(45), nullable=True)
    user_agent = db.Column(db.String(255), nullable=True)

    def __repr__(self):
        """
        String representation of the Log object.

        Returns:
            str: Log details in a formatted string
        """
        return f"<Log {self.timestamp} {self.level.value}: {self.message}>"


class SystemSettingKey(Enum):
    """
    Enumeration of system setting keys.

    Attributes:
        LOGGING_ENABLED: Whether logging is enabled
        LOGGING_LEVEL: Current logging level
    """

    LOGGING_ENABLED = "logging_enabled"
    LOGGING_LEVEL = "logging_level"


class SystemSetting(db.Model):
    """
    Represents system-wide configuration settings.

    Attributes:
        id (str): Unique identifier (UUID)
        key (SystemSettingKey): Setting key
        value (str): Setting value
        updated_at (datetime): Last update timestamp
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    key = db.Column(db.Enum(SystemSettingKey), nullable=False, unique=True)
    value = db.Column(db.String(255), nullable=False)
    updated_at = db.Column(
        db.DateTime,
        default=db.func.current_timestamp(),
        onupdate=db.func.current_timestamp(),
    )

    @staticmethod
    def get_value(key: SystemSettingKey, default=None):
        """
        Get the value of a system setting.

        Args:
            key (SystemSettingKey): Setting key
            default: Default value if setting is not found

        Returns:
            str: Setting value or default if not found
        """
        setting = SystemSetting.query.filter_by(key=key).first()
        return setting.value if setting else default


class SymbolInfo(db.Model):
    """
    Cache table for symbol information from external sources.

    Attributes:
        id (str): Unique identifier (UUID)
        symbol (str): Trading symbol
        name (str): Symbol name
        exchange (str): Exchange where the symbol is traded
        currency (str): Currency code
    """

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    symbol = db.Column(db.String(10), unique=True, nullable=False)
    name = db.Column(db.String(255), nullable=False)
    exchange = db.Column(db.String(50))
    currency = db.Column(db.String(3))
    isin = db.Column(db.String(12), unique=True)  # ISIN if available
    last_updated = db.Column(
        db.DateTime,
        default=db.func.current_timestamp(),
        onupdate=db.func.current_timestamp(),
    )
    data_source = db.Column(db.String(50))  # e.g., 'yfinance', 'manual'
    is_valid = db.Column(db.Boolean, default=True)  # Flag for valid/invalid symbols

    def __repr__(self):
        """
        String representation of the SymbolInfo object.

        Returns:
            str: Symbol details in a formatted string
        """
        return f"<SymbolInfo {self.symbol} ({self.name})>"


class RealizedGainLoss(db.Model):
    """
    Tracks realized gains and losses from selling investments.
    
    This allows historical tracking of portfolio performance even after
    shares are sold and reinvested.
    """
    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    portfolio_id = db.Column(
        db.String(36), 
        db.ForeignKey("portfolio.id", ondelete="CASCADE"), 
        nullable=False
    )
    fund_id = db.Column(
        db.String(36), 
        db.ForeignKey("fund.id"), 
        nullable=False
    )
    transaction_date = db.Column(db.Date, nullable=False)
    shares_sold = db.Column(db.Float, nullable=False)
    cost_basis = db.Column(db.Float, nullable=False)  # Original purchase cost
    sale_proceeds = db.Column(db.Float, nullable=False)  # Amount received from sale
    realized_gain_loss = db.Column(db.Float, nullable=False)  # sale_proceeds - cost_basis
    created_at = db.Column(db.DateTime, default=db.func.current_timestamp())

    # Add relationships
    portfolio = db.relationship("Portfolio", backref="realized_gains_losses")
    fund = db.relationship("Fund", backref="realized_gains_losses")
