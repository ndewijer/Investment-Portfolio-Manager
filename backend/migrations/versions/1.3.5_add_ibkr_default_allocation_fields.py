"""
Add default allocation fields to ibkr_config table.

Revision ID: 1.3.5
Revises: 1.3.1
Create Date: 2025-12-17
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.3.5"
down_revision = "1.3.1"
branch_labels = None
depends_on = None


def upgrade():
    """Add default_allocation_enabled and default_allocations columns to ibkr_config table."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    # Check if ibkr_config table exists
    if "ibkr_config" not in inspector.get_table_names():
        # Table doesn't exist, skip migration
        return

    columns = [col["name"] for col in inspector.get_columns("ibkr_config")]

    # Add default_allocation_enabled column
    if "default_allocation_enabled" not in columns:
        op.add_column(
            "ibkr_config",
            sa.Column("default_allocation_enabled", sa.Boolean(), nullable=True),
        )

        # Set default value: disabled for existing configurations
        # (existing users need to explicitly configure and enable this feature)
        op.execute(
            """
            UPDATE ibkr_config
            SET default_allocation_enabled = FALSE
            WHERE default_allocation_enabled IS NULL
            """
        )

        # Make column non-nullable after setting defaults
        with op.batch_alter_table("ibkr_config", schema=None) as batch_op:
            batch_op.alter_column("default_allocation_enabled", nullable=False)

    # Add default_allocations column (stores JSON as text)
    if "default_allocations" not in columns:
        op.add_column(
            "ibkr_config",
            sa.Column("default_allocations", sa.Text(), nullable=True),
        )
        # No default value needed - NULL indicates no preset configured


def downgrade():
    """Remove default allocation columns from ibkr_config table."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    if "ibkr_config" not in inspector.get_table_names():
        return

    columns = [col["name"] for col in inspector.get_columns("ibkr_config")]

    if "default_allocations" in columns:
        op.drop_column("ibkr_config", "default_allocations")

    if "default_allocation_enabled" in columns:
        op.drop_column("ibkr_config", "default_allocation_enabled")
