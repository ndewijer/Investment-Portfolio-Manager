"""
Add exclude_from_overview column to table portfolio.

Revision ID: 1.1.0
Revises: None
Create Date: 2024-11-22
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "1.1.0"
down_revision = None
branch_labels = None
depends_on = None


def upgrade():
    """Add exclude_from_overview column to portfolio table."""
    # Check if exclude_from_overview column already exists
    conn = op.get_bind()
    inspector = sa.inspect(conn)
    columns = [col["name"] for col in inspector.get_columns("portfolio")]

    if "exclude_from_overview" not in columns:
        # Add exclude_from_overview column only if it doesn't exist
        op.add_column(
            "portfolio",
            sa.Column("exclude_from_overview", sa.Boolean(), nullable=True),
        )

        # Set default value for new column
        op.execute(
            """
            UPDATE portfolio
            SET exclude_from_overview = FALSE
            WHERE exclude_from_overview IS NULL
        """
        )

    # Check if is_hidden column exists and migrate its values
    if "is_hidden" in columns:
        # Migrate values from is_hidden to exclude_from_overview
        op.execute(
            """
            UPDATE portfolio
            SET exclude_from_overview = is_hidden
        """
        )

        # Drop the old is_hidden column
        op.drop_column("portfolio", "is_hidden")

    # Make the column non-nullable after setting defaults
    if op.get_bind().dialect.name == "sqlite":
        # SQLite workaround for altering column to NOT NULL
        with op.batch_alter_table("portfolio") as batch_op:
            batch_op.alter_column(
                "exclude_from_overview",
                existing_type=sa.Boolean(),
                nullable=False,
                server_default=sa.text("FALSE"),
            )
    else:
        op.alter_column(
            "portfolio",
            "exclude_from_overview",
            existing_type=sa.Boolean(),
            nullable=False,
            server_default=sa.text("FALSE"),
        )


def downgrade():
    """Remove exclude_from_overview column from portfolio table."""

    # Add back is_hidden column if it existed
    op.add_column("portfolio", sa.Column("is_hidden", sa.Boolean(), nullable=True))

    # Migrate values back to is_hidden
    op.execute(
        """
        UPDATE portfolio
        SET is_hidden = exclude_from_overview
    """
    )

    # Drop the exclude_from_overview column
    op.drop_column("portfolio", "exclude_from_overview")
