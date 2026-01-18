"""
Migrate to fund-level materialized view.

This migration restructures the materialized view from portfolio-level to fund-level.
The new fund_history_materialized table stores atomic fund data which can be aggregated
for portfolio-level queries, eliminating data duplication and improving maintainability.

Revision ID: 1.5.0
Revises: 1.4.0
Create Date: 2026-01-18
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.5.0"
down_revision = "1.4.0"
branch_labels = None
depends_on = None


def upgrade():
    """
    Upgrade to fund-level materialized view.

    Steps:
    1. Drop old portfolio_history_materialized table
    2. Create new fund_history_materialized table
    3. Create indexes for optimal query performance
    """
    conn = op.get_bind()
    inspector = sa.inspect(conn)
    tables = inspector.get_table_names()

    # Drop the old portfolio-level materialized view if it exists
    if "portfolio_history_materialized" in tables:
        # Drop indexes first if they exist
        indexes = [idx["name"] for idx in inspector.get_indexes("portfolio_history_materialized")]

        if "idx_portfolio_history_mat_date" in indexes:
            op.drop_index(
                "idx_portfolio_history_mat_date",
                table_name="portfolio_history_materialized",
            )

        if "idx_portfolio_history_mat_portfolio_date" in indexes:
            op.drop_index(
                "idx_portfolio_history_mat_portfolio_date",
                table_name="portfolio_history_materialized",
            )

        op.drop_table("portfolio_history_materialized")

    # Create the new fund-level materialized view table
    if "fund_history_materialized" not in tables:
        op.create_table(
            "fund_history_materialized",
            sa.Column("id", sa.String(36), primary_key=True, nullable=False),
            sa.Column("portfolio_fund_id", sa.String(36), nullable=False),
            sa.Column("fund_id", sa.String(36), nullable=False),
            sa.Column("date", sa.String(10), nullable=False),
            # Fund metrics
            sa.Column("shares", sa.Float, nullable=False),
            sa.Column("price", sa.Float, nullable=False),
            sa.Column("value", sa.Float, nullable=False),
            sa.Column("cost", sa.Float, nullable=False),
            # Gain/loss metrics
            sa.Column("realized_gain", sa.Float, nullable=False),
            sa.Column("unrealized_gain", sa.Float, nullable=False),
            sa.Column("total_gain_loss", sa.Float, nullable=False),
            # Income/expense metrics
            sa.Column("dividends", sa.Float, nullable=False),
            sa.Column("fees", sa.Float, nullable=False),
            # Metadata
            sa.Column(
                "calculated_at",
                sa.DateTime,
                server_default=sa.func.now(),
                nullable=False,
            ),
            # Foreign key constraint
            sa.ForeignKeyConstraint(
                ["portfolio_fund_id"],
                ["portfolio_fund.id"],
                ondelete="CASCADE",
            ),
            # Unique constraint on portfolio_fund_id + date (defined in table creation for SQLite)
            sa.UniqueConstraint("portfolio_fund_id", "date", name="uq_portfolio_fund_date"),
        )

        # Create indexes for optimal query performance
        op.create_index(
            "idx_fund_history_pf_date",
            "fund_history_materialized",
            ["portfolio_fund_id", "date"],
        )

        op.create_index(
            "idx_fund_history_date",
            "fund_history_materialized",
            ["date"],
        )

        op.create_index(
            "idx_fund_history_fund_id",
            "fund_history_materialized",
            ["fund_id"],
        )


def downgrade():
    """
    Downgrade back to portfolio-level materialized view.

    WARNING: This will lose fund-level historical data granularity.
    Only use if absolutely necessary to rollback.
    """
    conn = op.get_bind()
    inspector = sa.inspect(conn)
    tables = inspector.get_table_names()

    # Drop the fund-level materialized view if it exists
    if "fund_history_materialized" in tables:
        # Drop indexes first if they exist
        indexes = [idx["name"] for idx in inspector.get_indexes("fund_history_materialized")]

        if "idx_fund_history_fund_id" in indexes:
            op.drop_index("idx_fund_history_fund_id", table_name="fund_history_materialized")

        if "idx_fund_history_date" in indexes:
            op.drop_index("idx_fund_history_date", table_name="fund_history_materialized")

        if "idx_fund_history_pf_date" in indexes:
            op.drop_index("idx_fund_history_pf_date", table_name="fund_history_materialized")

        # Drop unique constraint
        op.drop_constraint("uq_portfolio_fund_date", "fund_history_materialized", type_="unique")

        op.drop_table("fund_history_materialized")

    # Recreate the old portfolio-level table structure
    if "portfolio_history_materialized" not in tables:
        op.create_table(
            "portfolio_history_materialized",
            sa.Column("id", sa.String(36), primary_key=True, nullable=False),
            sa.Column("portfolio_id", sa.String(36), nullable=False),
            sa.Column("date", sa.String(10), nullable=False),
            sa.Column("value", sa.Float, nullable=False),
            sa.Column("cost", sa.Float, nullable=False),
            sa.Column("realized_gain", sa.Float, nullable=False),
            sa.Column("unrealized_gain", sa.Float, nullable=False),
            sa.Column("total_dividends", sa.Float, nullable=False),
            sa.Column("total_sale_proceeds", sa.Float, nullable=False),
            sa.Column("total_original_cost", sa.Float, nullable=False),
            sa.Column("total_gain_loss", sa.Float, nullable=False),
            sa.Column("is_archived", sa.Boolean, nullable=True),
            sa.Column(
                "calculated_at",
                sa.DateTime,
                server_default=sa.func.now(),
                nullable=False,
            ),
            sa.ForeignKeyConstraint(
                ["portfolio_id"],
                ["portfolio.id"],
                ondelete="CASCADE",
            ),
        )

        op.create_unique_constraint(
            "uq_portfolio_date",
            "portfolio_history_materialized",
            ["portfolio_id", "date"],
        )

        op.create_index(
            "idx_portfolio_history_mat_portfolio_date",
            "portfolio_history_materialized",
            ["portfolio_id", "date"],
        )

        op.create_index(
            "idx_portfolio_history_mat_date",
            "portfolio_history_materialized",
            ["date"],
        )
