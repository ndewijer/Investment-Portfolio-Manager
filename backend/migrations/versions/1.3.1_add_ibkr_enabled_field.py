"""
Add enabled field to ibkr_config table.

Revision ID: 1.3.1
Revises: 1.3.0
Create Date: 2025-11-07
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.3.1"
down_revision = "1.3.0"
branch_labels = None
depends_on = None


def upgrade():
    """Add enabled column to ibkr_config table."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    # Check if ibkr_config table exists
    if "ibkr_config" not in inspector.get_table_names():
        # Table doesn't exist, skip migration
        return

    columns = [col["name"] for col in inspector.get_columns("ibkr_config")]

    if "enabled" not in columns:
        # Add enabled column
        op.add_column(
            "ibkr_config",
            sa.Column("enabled", sa.Boolean(), nullable=True),
        )

        # Set default value: enabled=True for existing configurations
        # (existing users have IBKR configured and expect it to work)
        op.execute(
            """
            UPDATE ibkr_config
            SET enabled = TRUE
            WHERE enabled IS NULL
            """
        )

        # Make column non-nullable after setting defaults
        with op.batch_alter_table("ibkr_config", schema=None) as batch_op:
            batch_op.alter_column("enabled", nullable=False)


def downgrade():
    """Remove enabled column from ibkr_config table."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)

    if "ibkr_config" not in inspector.get_table_names():
        return

    columns = [col["name"] for col in inspector.get_columns("ibkr_config")]

    if "enabled" in columns:
        op.drop_column("ibkr_config", "enabled")
