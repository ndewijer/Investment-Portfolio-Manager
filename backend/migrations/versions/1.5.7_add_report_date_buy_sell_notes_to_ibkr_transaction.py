"""Add report_date and notes to IBKRTransaction.

Revision ID: 1.5.7
Revises: 1.5.5
Create Date: 2026-02-26
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.5.7"
down_revision = "1.5.5"
branch_labels = None
depends_on = None


def upgrade():
    """Add report_date and notes columns to ibkr_transaction."""
    # Step 1: Add columns as nullable
    op.add_column("ibkr_transaction", sa.Column("report_date", sa.Date(), nullable=True))
    op.add_column("ibkr_transaction", sa.Column("notes", sa.String(length=255), nullable=True))

    # Step 2: Backfill existing rows
    op.execute(
        "UPDATE ibkr_transaction SET report_date = transaction_date WHERE report_date IS NULL"
    )
    op.execute("UPDATE ibkr_transaction SET notes = '' WHERE notes IS NULL")

    # Step 3: Make columns NOT NULL now that all rows are populated
    with op.batch_alter_table("ibkr_transaction") as batch_op:
        batch_op.alter_column("report_date", existing_type=sa.Date(), nullable=False)
        batch_op.alter_column("notes", existing_type=sa.String(length=255), nullable=False)


def downgrade():
    """Remove report_date and notes columns from ibkr_transaction."""
    op.drop_column("ibkr_transaction", "notes")
    op.drop_column("ibkr_transaction", "report_date")
