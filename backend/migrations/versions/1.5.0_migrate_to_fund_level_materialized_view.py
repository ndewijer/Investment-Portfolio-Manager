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

    SQLite doesn't support altering foreign keys, so we use raw SQL to:
    1. Create new table with correct foreign keys
    2. Copy data
    3. Drop old table
    4. Rename new table
    """
    tables = inspector.get_table_names()

    # Clean up any leftover temp tables from failed migrations
    if "portfolio_fund_old" in tables:
        op.drop_table("portfolio_fund_old")

    if "portfolio_fund" not in tables:
        return

    # Check if foreign key already has CASCADE DELETE
    # If it does, skip this fix
    pragma_result = conn.execute(sa.text("PRAGMA foreign_key_list(portfolio_fund)"))
    for row in pragma_result:
        # Row format: (id, seq, table, from, to, on_update, on_delete, match)
        if row[2] == "fund" and row[6] == "CASCADE":
            # Already has CASCADE DELETE on fund_id
            return

    # Use raw SQL for the entire operation in a single transaction
    conn.execute(sa.text("PRAGMA foreign_keys=OFF"))

    try:
        # Create new table with correct foreign keys
        conn.execute(
            sa.text("""
            CREATE TABLE portfolio_fund_new (
                id VARCHAR(36) NOT NULL PRIMARY KEY,
                portfolio_id VARCHAR(36) NOT NULL,
                fund_id VARCHAR(36) NOT NULL,
                FOREIGN KEY(portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
                FOREIGN KEY(fund_id) REFERENCES fund(id) ON DELETE CASCADE,
                CONSTRAINT unique_portfolio_fund UNIQUE (portfolio_id, fund_id)
            )
        """)
        )

        # Copy data
        conn.execute(sa.text("INSERT INTO portfolio_fund_new SELECT * FROM portfolio_fund"))

        # Drop old table
        conn.execute(sa.text("DROP TABLE portfolio_fund"))

        # Rename new table
        conn.execute(sa.text("ALTER TABLE portfolio_fund_new RENAME TO portfolio_fund"))

    finally:
        conn.execute(sa.text("PRAGMA foreign_keys=ON"))


def _fix_dividend_cascade(conn, inspector):
    """
    Fix missing CASCADE DELETE on dividend foreign keys.

    Adds CASCADE DELETE to:
    - portfolio_fund_id
    - reinvestment_transaction_id
    """
    tables = inspector.get_table_names()

    # Clean up any leftover temp tables from failed migrations
    if "dividend_old" in tables:
        conn.execute(sa.text("DROP TABLE IF EXISTS dividend_old"))

    if "dividend" not in tables:
        return

    # Check if foreign keys already have CASCADE DELETE
    pragma_result = conn.execute(sa.text("PRAGMA foreign_key_list(dividend)"))
    has_pf_cascade = False
    has_txn_cascade = False

    for row in pragma_result:
        # Row format: (id, seq, table, from, to, on_update, on_delete, match)
        if row[2] == "portfolio_fund" and row[6] == "CASCADE":
            has_pf_cascade = True
        if row[2] == "transaction" and row[6] == "CASCADE":
            has_txn_cascade = True

    # Skip if already fixed
    if has_pf_cascade and has_txn_cascade:
        return

    # Use raw SQL for the entire operation
    conn.execute(sa.text("PRAGMA foreign_keys=OFF"))

    try:
        # Drop existing indexes first
        existing_indexes = [idx["name"] for idx in inspector.get_indexes("dividend")]
        for index_name in [
            "ix_dividend_fund_id",
            "ix_dividend_portfolio_fund_id",
            "ix_dividend_record_date",
        ]:
            if index_name in existing_indexes:
                conn.execute(sa.text(f"DROP INDEX IF EXISTS {index_name}"))

        # Create new table with correct foreign keys
        conn.execute(
            sa.text("""
            CREATE TABLE dividend_new (
                id VARCHAR(36) NOT NULL PRIMARY KEY,
                fund_id VARCHAR(36) NOT NULL,
                portfolio_fund_id VARCHAR(36) NOT NULL,
                record_date DATE NOT NULL,
                ex_dividend_date DATE NOT NULL,
                shares_owned FLOAT NOT NULL,
                dividend_per_share FLOAT NOT NULL,
                total_amount FLOAT NOT NULL,
                reinvestment_status VARCHAR(9) NOT NULL,
                buy_order_date DATE,
                reinvestment_transaction_id VARCHAR(36),
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY(fund_id) REFERENCES fund(id),
                FOREIGN KEY(portfolio_fund_id) REFERENCES portfolio_fund(id)
                    ON DELETE CASCADE,
                FOREIGN KEY(reinvestment_transaction_id) REFERENCES "transaction"(id)
                    ON DELETE CASCADE
            )
        """)
        )

        # Copy data
        conn.execute(sa.text("INSERT INTO dividend_new SELECT * FROM dividend"))

        # Drop old table
        conn.execute(sa.text("DROP TABLE dividend"))

        # Rename new table
        conn.execute(sa.text("ALTER TABLE dividend_new RENAME TO dividend"))

        # Recreate indexes
        conn.execute(sa.text("CREATE INDEX ix_dividend_fund_id ON dividend(fund_id)"))
        conn.execute(
            sa.text("CREATE INDEX ix_dividend_portfolio_fund_id ON dividend(portfolio_fund_id)")
        )
        conn.execute(sa.text("CREATE INDEX ix_dividend_record_date ON dividend(record_date)"))

    finally:
        conn.execute(sa.text("PRAGMA foreign_keys=ON"))


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
