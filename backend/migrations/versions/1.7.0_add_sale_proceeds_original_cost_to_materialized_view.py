"""
Add sale_proceeds and original_cost columns to fund_history_materialized.

These columns store cumulative sale proceeds and original cost basis at the fund level,
eliminating the need for expensive correlated subqueries against the RealizedGainLoss table.

Also truncates the materialized table so all data is regenerated fresh with the new columns.

Revision ID: 1.7.0
Revises: 1.5.7
Create Date: 2026-03-13
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.7.0"
down_revision = "1.5.7"
branch_labels = None
depends_on = None


def upgrade():
    """Add sale_proceeds and original_cost columns, then truncate table."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)
    tables = inspector.get_table_names()

    if "fund_history_materialized" not in tables:
        return

    # Check existing columns
    existing_columns = [col["name"] for col in inspector.get_columns("fund_history_materialized")]

    # Add sale_proceeds column if not present
    if "sale_proceeds" not in existing_columns:
        op.add_column(
            "fund_history_materialized",
            sa.Column("sale_proceeds", sa.Float, nullable=False, server_default="0.0"),
        )

    # Add original_cost column if not present
    if "original_cost" not in existing_columns:
        op.add_column(
            "fund_history_materialized",
            sa.Column("original_cost", sa.Float, nullable=False, server_default="0.0"),
        )

    # Truncate the materialized table so everything gets regenerated fresh
    conn.execute(sa.text("DELETE FROM fund_history_materialized"))


def downgrade():
    """Remove sale_proceeds and original_cost columns."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)
    tables = inspector.get_table_names()

    if "fund_history_materialized" not in tables:
        return

    existing_columns = [col["name"] for col in inspector.get_columns("fund_history_materialized")]

    if "original_cost" in existing_columns:
        op.drop_column("fund_history_materialized", "original_cost")

    if "sale_proceeds" in existing_columns:
        op.drop_column("fund_history_materialized", "sale_proceeds")
