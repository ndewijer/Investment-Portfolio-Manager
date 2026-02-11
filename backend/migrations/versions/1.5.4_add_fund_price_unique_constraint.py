"""
Add unique constraint on fund_price (fund_id, date).

This migration adds a unique constraint to prevent duplicate price records
for the same fund on the same date, matching the pattern already used in
the exchange_rate table.

Changes:
1. Adds UniqueConstraint("fund_id", "date", name="unique_fund_price") to fund_price table
2. Updates TodayPriceService and HistoricalPriceService to use upsert logic

IMPORTANT: Before running this migration, clean up any duplicate records:
    SELECT fund_id, date, COUNT(*) as count
    FROM fund_price
    GROUP BY fund_id, date
    HAVING COUNT(*) > 1;

See docs/FUND_PRICE_UNIQUE_CONSTRAINT.md for cleanup SQL.

Revision ID: 1.5.4
Revises: 1.5.0
Create Date: 2026-02-10
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.5.4"
down_revision = "1.5.0"
branch_labels = None
depends_on = None


def upgrade():
    """
    Add unique constraint to fund_price table.

    SQLite doesn't support adding constraints to existing tables,
    so we need to recreate the table with the constraint using raw SQL.
    """
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    # Check if table exists
    if "fund_price" not in inspector.get_table_names():
        # Table doesn't exist yet, skip
        return

    # Check if constraint already exists
    existing_constraints = inspector.get_unique_constraints("fund_price")
    for constraint in existing_constraints:
        if set(constraint["column_names"]) == {"fund_id", "date"}:
            # Constraint already exists, skip
            return

    # Use raw SQL for the entire operation
    conn.execute(sa.text("PRAGMA foreign_keys=OFF"))

    try:
        # Step 1: Drop existing indexes first
        existing_indexes = [idx["name"] for idx in inspector.get_indexes("fund_price")]
        for index_name in [
            "ix_fund_price_date",
            "ix_fund_price_fund_id",
            "ix_fund_price_fund_id_date",
        ]:
            if index_name in existing_indexes:
                conn.execute(sa.text(f"DROP INDEX IF EXISTS {index_name}"))

        # Step 2: Create new table with unique constraint using raw SQL
        conn.execute(
            sa.text("""
            CREATE TABLE fund_price_new (
                id VARCHAR(36) NOT NULL PRIMARY KEY,
                fund_id VARCHAR(36) NOT NULL,
                date DATE NOT NULL,
                price FLOAT NOT NULL,
                FOREIGN KEY(fund_id) REFERENCES fund(id),
                CONSTRAINT unique_fund_price UNIQUE (fund_id, date)
            )
        """)
        )

        # Step 3: Copy data from old table, keeping only newest record per (fund_id, date)
        # This handles any remaining duplicates by keeping the latest ID (most recent insert)
        conn.execute(
            sa.text("""
            INSERT INTO fund_price_new (id, fund_id, date, price)
            SELECT id, fund_id, date, price
            FROM fund_price
            WHERE id IN (
                SELECT MAX(id)
                FROM fund_price
                GROUP BY fund_id, date
            )
        """)
        )

        # Step 4: Drop old table
        conn.execute(sa.text("DROP TABLE fund_price"))

        # Step 5: Rename new table
        conn.execute(sa.text("ALTER TABLE fund_price_new RENAME TO fund_price"))

        # Step 6: Recreate indexes
        conn.execute(sa.text("CREATE INDEX ix_fund_price_date ON fund_price(date)"))
        conn.execute(sa.text("CREATE INDEX ix_fund_price_fund_id ON fund_price(fund_id)"))
        conn.execute(
            sa.text("CREATE INDEX ix_fund_price_fund_id_date ON fund_price(fund_id, date)")
        )

    finally:
        # Re-enable foreign keys
        conn.execute(sa.text("PRAGMA foreign_keys=ON"))


def downgrade():
    """
    Remove unique constraint from fund_price table.

    Recreates the table without the constraint using raw SQL.
    """
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    if "fund_price" not in inspector.get_table_names():
        return

    # Use raw SQL for the entire operation
    conn.execute(sa.text("PRAGMA foreign_keys=OFF"))

    try:
        # Step 1: Drop existing indexes first
        existing_indexes = [idx["name"] for idx in inspector.get_indexes("fund_price")]
        for index_name in [
            "ix_fund_price_date",
            "ix_fund_price_fund_id",
            "ix_fund_price_fund_id_date",
        ]:
            if index_name in existing_indexes:
                conn.execute(sa.text(f"DROP INDEX IF EXISTS {index_name}"))

        # Step 2: Create new table without unique constraint using raw SQL
        conn.execute(
            sa.text("""
            CREATE TABLE fund_price_new (
                id VARCHAR(36) NOT NULL PRIMARY KEY,
                fund_id VARCHAR(36) NOT NULL,
                date DATE NOT NULL,
                price FLOAT NOT NULL,
                FOREIGN KEY(fund_id) REFERENCES fund(id)
            )
        """)
        )

        # Step 3: Copy data
        conn.execute(
            sa.text("""
            INSERT INTO fund_price_new (id, fund_id, date, price)
            SELECT id, fund_id, date, price
            FROM fund_price
        """)
        )

        # Step 4: Drop old table
        conn.execute(sa.text("DROP TABLE fund_price"))

        # Step 5: Rename new table
        conn.execute(sa.text("ALTER TABLE fund_price_new RENAME TO fund_price"))

        # Step 6: Recreate indexes
        conn.execute(sa.text("CREATE INDEX ix_fund_price_date ON fund_price(date)"))
        conn.execute(sa.text("CREATE INDEX ix_fund_price_fund_id ON fund_price(fund_id)"))
        conn.execute(
            sa.text("CREATE INDEX ix_fund_price_fund_id_date ON fund_price(fund_id, date)")
        )

    finally:
        # Re-enable foreign keys
        conn.execute(sa.text("PRAGMA foreign_keys=ON"))
