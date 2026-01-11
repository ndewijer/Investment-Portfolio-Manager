"""
Add portfolio_history_materialized table for performance optimization.

Revision ID: 1.4.0
Revises: 1.3.5
Create Date: 2026-01-11
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.4.0"
down_revision = "1.3.5"
branch_labels = None
depends_on = None


def upgrade():
    """Create portfolio_history_materialized table for caching portfolio history calculations."""
    op.create_table(
        "portfolio_history_materialized",
        sa.Column("id", sa.String(), nullable=False),
        sa.Column("portfolio_id", sa.String(), nullable=False),
        sa.Column("date", sa.String(), nullable=False),
        sa.Column("value", sa.Float(), nullable=False),
        sa.Column("cost", sa.Float(), nullable=False),
        sa.Column("realized_gain", sa.Float(), nullable=False),
        sa.Column("unrealized_gain", sa.Float(), nullable=False),
        sa.Column("total_dividends", sa.Float(), nullable=False),
        sa.Column("total_sale_proceeds", sa.Float(), nullable=False),
        sa.Column("total_original_cost", sa.Float(), nullable=False),
        sa.Column("total_gain_loss", sa.Float(), nullable=False),
        sa.Column("is_archived", sa.Integer(), nullable=False),
        sa.Column(
            "calculated_at",
            sa.DateTime(),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.PrimaryKeyConstraint("id"),
        sa.ForeignKeyConstraint(
            ["portfolio_id"],
            ["portfolio.id"],
            ondelete="CASCADE",
        ),
        sa.UniqueConstraint("portfolio_id", "date", name="uq_portfolio_date"),
    )

    # Create index on portfolio_id and date for efficient range queries
    op.create_index(
        "idx_portfolio_history_mat_portfolio_date",
        "portfolio_history_materialized",
        ["portfolio_id", "date"],
    )

    # Create index on date for cross-portfolio queries
    op.create_index(
        "idx_portfolio_history_mat_date",
        "portfolio_history_materialized",
        ["date"],
    )


def downgrade():
    """Drop portfolio_history_materialized table and its indexes."""
    op.drop_index(
        "idx_portfolio_history_mat_date",
        table_name="portfolio_history_materialized",
    )
    op.drop_index(
        "idx_portfolio_history_mat_portfolio_date",
        table_name="portfolio_history_materialized",
    )
    op.drop_table("portfolio_history_materialized")
