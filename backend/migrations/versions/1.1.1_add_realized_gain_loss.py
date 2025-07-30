"""
Add RealizedGainLoss table.

Revision ID: 1.1.1
Revises: 1.1.0
Create Date: 2024-11-29

"""

import sqlalchemy as sa
from alembic import op
from sqlalchemy.exc import OperationalError

# revision identifiers, used by Alembic.
revision = "1.1.1"
down_revision = "1.1.0"
branch_labels = None
depends_on = None


def upgrade():
    """Add RealizedGainLoss table."""
    try:
        # Create RealizedGainLoss table
        op.create_table(
            "realized_gain_loss",
            sa.Column("id", sa.String(36), primary_key=True),
            sa.Column(
                "portfolio_id",
                sa.String(36),
                sa.ForeignKey("portfolio.id", ondelete="CASCADE"),
                nullable=False,
            ),
            sa.Column("fund_id", sa.String(36), sa.ForeignKey("fund.id"), nullable=False),
            sa.Column(
                "transaction_id",
                sa.String(36),
                sa.ForeignKey("transaction.id", ondelete="CASCADE"),
                nullable=False,
            ),
            sa.Column("transaction_date", sa.Date(), nullable=False),
            sa.Column("shares_sold", sa.Float(), nullable=False),
            sa.Column("cost_basis", sa.Float(), nullable=False),
            sa.Column("sale_proceeds", sa.Float(), nullable=False),
            sa.Column("realized_gain_loss", sa.Float(), nullable=False),
            sa.Column("created_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")),
        )
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

    # Create indexes - these are idempotent in SQLite
    try:
        for index_name, table, columns in [
            (
                "ix_realized_gain_loss_portfolio_id",
                "realized_gain_loss",
                ["portfolio_id"],
            ),
            ("ix_realized_gain_loss_fund_id", "realized_gain_loss", ["fund_id"]),
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
        ]:
            op.create_index(index_name, table, columns)
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e


def downgrade():
    """Drop RealizedGainLoss table."""
    try:
        for index_name in [
            "ix_realized_gain_loss_transaction_date",
            "ix_realized_gain_loss_fund_id",
            "ix_realized_gain_loss_portfolio_id",
            "ix_realized_gain_loss_transaction_id",
        ]:
            op.drop_index(index_name)
    except OperationalError as e:
        if "no such index" not in str(e):
            raise e

    try:
        # Drop the RealizedGainLoss table
        op.drop_table("realized_gain_loss")
    except OperationalError as e:
        if "no such table" not in str(e):
            raise e
