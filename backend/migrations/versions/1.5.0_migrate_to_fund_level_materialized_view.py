"""
Migrate to fund-level materialized view and fix CASCADE DELETE constraints.

This migration:
1. Restructures the materialized view from portfolio-level to fund-level
2. Fixes missing CASCADE DELETE constraints on dividend table
3. Fixes missing CASCADE DELETE on portfolio_fund.fund_id

The new fund_history_materialized table stores atomic fund data which can be aggregated
for portfolio-level queries, eliminating data duplication and improving maintainability.

Revision ID: 1.5.0
Revises: 1.4.0
Create Date: 2026-01-18
Updated: 2026-01-31 (added CASCADE DELETE fixes)
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.5.0"
down_revision = "1.4.0"
branch_labels = None
depends_on = None


def _fix_portfolio_fund_cascade(conn, inspector):
    """
    Fix missing CASCADE DELETE on portfolio_fund.fund_id.

    SQLite doesn't support altering foreign keys, so we:
    1. Disable foreign key enforcement
    2. Rename old table
    3. Create new table with correct foreign keys
    4. Copy data
    5. Drop old table
    6. Re-enable foreign key enforcement
    """
    tables = inspector.get_table_names()
    if "portfolio_fund" not in tables:
        return

    # Disable foreign key enforcement
    op.execute("PRAGMA foreign_keys=OFF")

    # Rename old table
    op.rename_table("portfolio_fund", "portfolio_fund_old")

    # Create new portfolio_fund table with CASCADE DELETE on fund_id
    op.create_table(
        "portfolio_fund",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column(
            "portfolio_id",
            sa.String(36),
            sa.ForeignKey("portfolio.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column(
            "fund_id",
            sa.String(36),
            sa.ForeignKey("fund.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.UniqueConstraint("portfolio_id", "fund_id", name="unique_portfolio_fund"),
    )

    # Copy data
    op.execute("INSERT INTO portfolio_fund SELECT * FROM portfolio_fund_old")

    # Drop old table
    op.drop_table("portfolio_fund_old")

    # Re-enable foreign key enforcement
    op.execute("PRAGMA foreign_keys=ON")


def _fix_dividend_cascade(conn, inspector):
    """
    Fix missing CASCADE DELETE on dividend foreign keys.

    Adds CASCADE DELETE to:
    - portfolio_fund_id
    - reinvestment_transaction_id
    """
    tables = inspector.get_table_names()
    if "dividend" not in tables:
        return

    # Disable foreign key enforcement
    op.execute("PRAGMA foreign_keys=OFF")

    # Drop existing indexes first
    existing_indexes = [idx["name"] for idx in inspector.get_indexes("dividend")]
    for index_name in [
        "ix_dividend_fund_id",
        "ix_dividend_portfolio_fund_id",
        "ix_dividend_record_date",
    ]:
        if index_name in existing_indexes:
            op.drop_index(index_name, table_name="dividend")

    # Rename old table
    op.rename_table("dividend", "dividend_old")

    # Create new dividend table with CASCADE DELETE
    op.create_table(
        "dividend",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("fund_id", sa.String(36), sa.ForeignKey("fund.id"), nullable=False),
        sa.Column(
            "portfolio_fund_id",
            sa.String(36),
            sa.ForeignKey("portfolio_fund.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column("record_date", sa.Date(), nullable=False),
        sa.Column("ex_dividend_date", sa.Date(), nullable=False),
        sa.Column("shares_owned", sa.Float(), nullable=False),
        sa.Column("dividend_per_share", sa.Float(), nullable=False),
        sa.Column("total_amount", sa.Float(), nullable=False),
        sa.Column("reinvestment_status", sa.String(9), nullable=False),
        sa.Column("buy_order_date", sa.Date(), nullable=True),
        sa.Column(
            "reinvestment_transaction_id",
            sa.String(36),
            sa.ForeignKey("transaction.id", ondelete="CASCADE"),
            nullable=True,
        ),
        sa.Column("created_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")),
    )

    # Copy data
    op.execute("INSERT INTO dividend SELECT * FROM dividend_old")

    # Drop old table
    op.drop_table("dividend_old")

    # Recreate indexes
    for index_name, columns in [
        ("ix_dividend_fund_id", ["fund_id"]),
        ("ix_dividend_portfolio_fund_id", ["portfolio_fund_id"]),
        ("ix_dividend_record_date", ["record_date"]),
    ]:
        op.create_index(index_name, "dividend", columns)

    # Re-enable foreign key enforcement
    op.execute("PRAGMA foreign_keys=ON")


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

    # Fix CASCADE DELETE constraints on portfolio_fund table
    _fix_portfolio_fund_cascade(conn, inspector)

    # Fix CASCADE DELETE constraints on dividend table
    _fix_dividend_cascade(conn, inspector)


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
