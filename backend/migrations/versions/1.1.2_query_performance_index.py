"""
Add indexes for better query performance.

Revision ID: 1.1.2
Revises: 1.1.1
Create Date: 2024-12-01

"""

from alembic import op
from sqlalchemy.exc import OperationalError

# revision identifiers, used by Alembic.
revision = "1.1.2"
down_revision = "1.1.1"
branch_labels = None
depends_on = None


def upgrade():
    """Add indexes for better query performance."""
    # Add each index in a try/except block
    for index_name, table, columns in [
        ("ix_dividend_fund_id", "dividend", ["fund_id"]),
        ("ix_dividend_portfolio_fund_id", "dividend", ["portfolio_fund_id"]),
        ("ix_dividend_record_date", "dividend", ["record_date"]),
        ("ix_exchange_rate_date", "exchange_rate", ["date"]),
        ("ix_fund_price_date", "fund_price", ["date"]),
        ("ix_fund_price_fund_id", "fund_price", ["fund_id"]),
        ("ix_fund_price_fund_id_date", "fund_price", ["fund_id", "date"]),
        ("ix_realized_gain_loss_fund_id", "realized_gain_loss", ["fund_id"]),
        ("ix_realized_gain_loss_portfolio_id", "realized_gain_loss", ["portfolio_id"]),
        (
            "ix_realized_gain_loss_transaction_date",
            "realized_gain_loss",
            ["transaction_date"],
        ),
        (
            "ix_realized_gain_loss_transaction_id",
            "realized_gain_loss",
            ["transaction_id"],
        ),
        ("ix_transaction_date", "transaction", ["date"]),
        ("ix_transaction_portfolio_fund_id", "transaction", ["portfolio_fund_id"]),
        (
            "ix_transaction_portfolio_fund_id_date",
            "transaction",
            ["portfolio_fund_id", "date"],
        ),
    ]:
        try:
            op.create_index(index_name, table, columns)
        except OperationalError as e:
            if "already exists" not in str(e):
                raise e


def downgrade():
    """Remove performance indexes."""
    # Drop each index in a try/except block
    for index_name in [
        "ix_dividend_fund_id",
        "ix_dividend_portfolio_fund_id",
        "ix_dividend_record_date",
        "ix_exchange_rate_date",
        "ix_fund_price_date",
        "ix_fund_price_fund_id",
        "ix_fund_price_fund_id_date",
        "ix_realized_gain_loss_fund_id",
        "ix_realized_gain_loss_portfolio_id",
        "ix_realized_gain_loss_transaction_date",
        "ix_realized_gain_loss_transaction_id",
        "ix_transaction_date",
        "ix_transaction_portfolio_fund_id",
        "ix_transaction_portfolio_fund_id_date",
    ]:
        try:
            op.drop_index(index_name)
        except OperationalError as e:
            if "no such index" not in str(e):
                raise e
