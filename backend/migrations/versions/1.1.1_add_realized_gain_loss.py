"""
Add RealizedGainLoss table.

Revision ID: 1.1.1
Revises: None
Create Date: 2024-11-29

"""

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = "1.1.1"
down_revision = "1.1.0"
branch_labels = None
depends_on = None


def upgrade():
    """Add RealizedGainLoss table."""

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
        sa.Column(
            "created_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")
        ),
    )

    # Create indexes for better query performance
    op.create_index(
        "ix_realized_gain_loss_portfolio_id", "realized_gain_loss", ["portfolio_id"]
    )
    op.create_index("ix_realized_gain_loss_fund_id", "realized_gain_loss", ["fund_id"])
    op.create_index(
        "ix_realized_gain_loss_transaction_date",
        "realized_gain_loss",
        ["transaction_date"],
    )
    op.create_index(
        "ix_realized_gain_loss_transaction_id",
        "realized_gain_loss",
        ["transaction_id"],
    )


def downgrade():
    """Drop RealizedGainLoss table."""

    # Drop indexes first
    op.drop_index("ix_realized_gain_loss_transaction_date")
    op.drop_index("ix_realized_gain_loss_fund_id")
    op.drop_index("ix_realized_gain_loss_portfolio_id")
    op.drop_index("ix_realized_gain_loss_transaction_id")

    # Drop the RealizedGainLoss table
    op.drop_table("realized_gain_loss")
