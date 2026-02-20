"""
Change dividend.reinvestment_transaction_id FK from CASCADE to RESTRICT.

Previously ON DELETE CASCADE meant deleting a reinvestment transaction
directly would silently delete the parent dividend record. This is wrong
because the dividend is the primary owner of the relationship.

With ON DELETE RESTRICT, direct deletion of a transaction that is
referenced by a dividend is blocked at the database level. The dividend
endpoint is the only correct way to delete dividend+transaction pairs,
and it explicitly handles both deletions in sequence.

Revision ID: 1.5.5
Revises: 1.5.4
Create Date: 2026-02-20
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.5.5"
down_revision = "1.5.4"
branch_labels = None
depends_on = None


def upgrade():
    """
    Recreate dividend table with ON DELETE RESTRICT on reinvestment_transaction_id.

    SQLite does not support ALTER TABLE to modify FK constraints, so the
    table must be recreated.
    """
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    if "dividend" not in inspector.get_table_names():
        return

    conn.execute(sa.text("PRAGMA foreign_keys=OFF"))

    try:
        # Step 1: Create new table with RESTRICT instead of CASCADE
        conn.execute(
            sa.text("""
            CREATE TABLE dividend_new (
                "id"                        VARCHAR(36)  NOT NULL,
                "fund_id"                   VARCHAR(36)  NOT NULL,
                "portfolio_fund_id"         VARCHAR(36)  NOT NULL,
                "record_date"               DATE         NOT NULL,
                "ex_dividend_date"          DATE         NOT NULL,
                "shares_owned"              FLOAT        NOT NULL,
                "dividend_per_share"        FLOAT        NOT NULL,
                "total_amount"              FLOAT        NOT NULL,
                "reinvestment_status"       VARCHAR(9)   NOT NULL,
                "buy_order_date"            DATE,
                "reinvestment_transaction_id" VARCHAR(36),
                "created_at"               DATETIME     DEFAULT CURRENT_TIMESTAMP,
                PRIMARY KEY("id"),
                FOREIGN KEY("fund_id") REFERENCES "fund"("id"),
                FOREIGN KEY("portfolio_fund_id")
                    REFERENCES "portfolio_fund"("id") ON DELETE CASCADE,
                FOREIGN KEY("reinvestment_transaction_id")
                    REFERENCES "transaction"("id") ON DELETE RESTRICT
            )
        """)
        )

        # Step 2: Copy all data
        conn.execute(
            sa.text("""
            INSERT INTO dividend_new
            SELECT * FROM dividend
        """)
        )

        # Step 3: Drop old table
        conn.execute(sa.text("DROP TABLE dividend"))

        # Step 4: Rename new table
        conn.execute(sa.text("ALTER TABLE dividend_new RENAME TO dividend"))

        # Step 5: Recreate indexes
        conn.execute(sa.text("CREATE INDEX ix_dividend_fund_id ON dividend(fund_id)"))
        conn.execute(
            sa.text("CREATE INDEX ix_dividend_portfolio_fund_id ON dividend(portfolio_fund_id)")
        )
        conn.execute(sa.text("CREATE INDEX ix_dividend_record_date ON dividend(record_date)"))

    finally:
        conn.execute(sa.text("PRAGMA foreign_keys=ON"))


def downgrade():
    """Recreate dividend table restoring ON DELETE CASCADE on reinvestment_transaction_id."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    if "dividend" not in inspector.get_table_names():
        return

    conn.execute(sa.text("PRAGMA foreign_keys=OFF"))

    try:
        conn.execute(
            sa.text("""
            CREATE TABLE dividend_new (
                "id"                        VARCHAR(36)  NOT NULL,
                "fund_id"                   VARCHAR(36)  NOT NULL,
                "portfolio_fund_id"         VARCHAR(36)  NOT NULL,
                "record_date"               DATE         NOT NULL,
                "ex_dividend_date"          DATE         NOT NULL,
                "shares_owned"              FLOAT        NOT NULL,
                "dividend_per_share"        FLOAT        NOT NULL,
                "total_amount"              FLOAT        NOT NULL,
                "reinvestment_status"       VARCHAR(9)   NOT NULL,
                "buy_order_date"            DATE,
                "reinvestment_transaction_id" VARCHAR(36),
                "created_at"               DATETIME     DEFAULT CURRENT_TIMESTAMP,
                PRIMARY KEY("id"),
                FOREIGN KEY("fund_id") REFERENCES "fund"("id"),
                FOREIGN KEY("portfolio_fund_id")
                    REFERENCES "portfolio_fund"("id") ON DELETE CASCADE,
                FOREIGN KEY("reinvestment_transaction_id")
                    REFERENCES "transaction"("id") ON DELETE CASCADE
            )
        """)
        )

        conn.execute(
            sa.text("""
            INSERT INTO dividend_new
            SELECT * FROM dividend
        """)
        )

        conn.execute(sa.text("DROP TABLE dividend"))
        conn.execute(sa.text("ALTER TABLE dividend_new RENAME TO dividend"))

        conn.execute(sa.text("CREATE INDEX ix_dividend_fund_id ON dividend(fund_id)"))
        conn.execute(
            sa.text("CREATE INDEX ix_dividend_portfolio_fund_id ON dividend(portfolio_fund_id)")
        )
        conn.execute(sa.text("CREATE INDEX ix_dividend_record_date ON dividend(record_date)"))

    finally:
        conn.execute(sa.text("PRAGMA foreign_keys=ON"))
