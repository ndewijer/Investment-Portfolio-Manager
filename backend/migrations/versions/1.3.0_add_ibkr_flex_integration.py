"""
Add IBKR Flex Integration tables and fee transaction type.

Revision ID: 1.3.0
Revises: 1.1.2
Create Date: 2025-01-03

"""

import sqlalchemy as sa
from alembic import op
from sqlalchemy.exc import OperationalError

# revision identifiers, used by Alembic.
revision = "1.3.0"
down_revision = "1.1.2"
branch_labels = None
depends_on = None


def upgrade():
    """Add IBKR Flex Integration tables and fee transaction type."""
    # Create ibkr_config table
    try:
        op.create_table(
            "ibkr_config",
            sa.Column("id", sa.String(36), primary_key=True),
            sa.Column("flex_token", sa.String(500), nullable=False),  # Encrypted token
            sa.Column("flex_query_id", sa.String(100), nullable=False),
            sa.Column("token_expires_at", sa.DateTime(), nullable=True),  # Token expiration
            sa.Column("last_import_date", sa.DateTime(), nullable=True),
            sa.Column("auto_import_enabled", sa.Boolean(), default=False, nullable=False),
            sa.Column("created_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")),
            sa.Column("updated_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")),
        )
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

    # Create ibkr_transaction table
    try:
        op.create_table(
            "ibkr_transaction",
            sa.Column("id", sa.String(36), primary_key=True),
            sa.Column("ibkr_transaction_id", sa.String(100), unique=True, nullable=False),
            sa.Column("transaction_date", sa.Date(), nullable=False),
            sa.Column("symbol", sa.String(10), nullable=True),
            sa.Column("isin", sa.String(12), nullable=True),
            sa.Column("description", sa.Text(), nullable=True),
            sa.Column("transaction_type", sa.String(20), nullable=False),
            sa.Column("quantity", sa.Float(), nullable=True),
            sa.Column("price", sa.Float(), nullable=True),
            sa.Column("total_amount", sa.Float(), nullable=False),
            sa.Column("currency", sa.String(3), nullable=False),
            sa.Column("fees", sa.Float(), default=0.0, nullable=False),
            sa.Column("status", sa.String(20), default="pending", nullable=False),
            sa.Column("imported_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")),
            sa.Column("processed_at", sa.DateTime(), nullable=True),
            sa.Column("raw_data", sa.Text(), nullable=True),
        )
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

    # Create indexes for ibkr_transaction
    try:
        for index_name, table, columns in [
            ("ix_ibkr_transaction_status", "ibkr_transaction", ["status"]),
            ("ix_ibkr_transaction_date", "ibkr_transaction", ["transaction_date"]),
            ("ix_ibkr_transaction_ibkr_id", "ibkr_transaction", ["ibkr_transaction_id"]),
        ]:
            op.create_index(index_name, table, columns)
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

    # Create ibkr_transaction_allocation table
    try:
        op.create_table(
            "ibkr_transaction_allocation",
            sa.Column("id", sa.String(36), primary_key=True),
            sa.Column(
                "ibkr_transaction_id",
                sa.String(36),
                sa.ForeignKey("ibkr_transaction.id", ondelete="CASCADE"),
                nullable=False,
            ),
            sa.Column(
                "portfolio_id",
                sa.String(36),
                sa.ForeignKey("portfolio.id", ondelete="CASCADE"),
                nullable=False,
            ),
            sa.Column("allocation_percentage", sa.Float(), nullable=False),
            sa.Column("allocated_amount", sa.Float(), nullable=False),
            sa.Column("allocated_shares", sa.Float(), nullable=False),
            sa.Column(
                "transaction_id",
                sa.String(36),
                sa.ForeignKey("transaction.id", ondelete="CASCADE"),
                nullable=True,
            ),
            sa.Column("created_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")),
        )
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

    # Create indexes for ibkr_transaction_allocation
    try:
        for index_name, table, columns in [
            (
                "ix_ibkr_allocation_ibkr_transaction_id",
                "ibkr_transaction_allocation",
                ["ibkr_transaction_id"],
            ),
            ("ix_ibkr_allocation_portfolio_id", "ibkr_transaction_allocation", ["portfolio_id"]),
            (
                "ix_ibkr_allocation_transaction_id",
                "ibkr_transaction_allocation",
                ["transaction_id"],
            ),
        ]:
            op.create_index(index_name, table, columns)
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

    # Create ibkr_import_cache table for caching API responses
    try:
        op.create_table(
            "ibkr_import_cache",
            sa.Column("id", sa.String(36), primary_key=True),
            sa.Column("cache_key", sa.String(255), unique=True, nullable=False),
            sa.Column("data", sa.Text(), nullable=False),
            sa.Column("created_at", sa.DateTime(), server_default=sa.text("CURRENT_TIMESTAMP")),
            sa.Column("expires_at", sa.DateTime(), nullable=False),
        )
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

    # Create index for cache expiration
    try:
        op.create_index("ix_ibkr_cache_expires_at", "ibkr_import_cache", ["expires_at"])
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e


def downgrade():
    """Remove IBKR Flex Integration tables."""
    # Drop indexes first
    try:
        for index_name in [
            "ix_ibkr_cache_expires_at",
            "ix_ibkr_allocation_transaction_id",
            "ix_ibkr_allocation_portfolio_id",
            "ix_ibkr_allocation_ibkr_transaction_id",
            "ix_ibkr_transaction_ibkr_id",
            "ix_ibkr_transaction_date",
            "ix_ibkr_transaction_status",
        ]:
            op.drop_index(index_name)
    except OperationalError as e:
        if "no such index" not in str(e):
            raise e

    # Drop tables
    try:
        for table_name in [
            "ibkr_import_cache",
            "ibkr_transaction_allocation",
            "ibkr_transaction",
            "ibkr_config",
        ]:
            op.drop_table(table_name)
    except OperationalError as e:
        if "no such table" not in str(e):
            raise e
