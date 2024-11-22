"""Migration script to add exclude_from_overview column to portfolio table."""

import os
import sqlite3


def migrate(db_path):
    """
    Add exclude_from_overview column to portfolio table if it doesn't exist.

    Args:
        db_path (str): Path to SQLite database file
    """
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    # Check if column exists
    cursor.execute(
        "SELECT COUNT(*) FROM pragma_table_info('portfolio') WHERE name='exclude_from_overview'"
    )
    column_exists = cursor.fetchone()[0] > 0

    if not column_exists:
        print("Adding exclude_from_overview column to portfolio table...")
        try:
            # Add the column
            cursor.execute(
                "ALTER TABLE portfolio ADD COLUMN exclude_from_overview BOOLEAN DEFAULT FALSE"
            )

            # Set default value for existing rows
            cursor.execute(
                "UPDATE portfolio SET exclude_from_overview = FALSE WHERE exclude_from_overview IS NULL"  # noqa: E501
            )

            # If the old is_hidden column exists, migrate its values
            cursor.execute(
                "SELECT COUNT(*) FROM pragma_table_info('portfolio') WHERE name='is_hidden'"
            )
            if cursor.fetchone()[0] > 0:
                cursor.execute("UPDATE portfolio SET exclude_from_overview = is_hidden")
                cursor.execute("ALTER TABLE portfolio DROP COLUMN is_hidden")

            conn.commit()
            print("Migration successful!")

        except sqlite3.Error as e:
            print(f"Error during migration: {e}")
            conn.rollback()
    else:
        print("Column exclude_from_overview already exists in portfolio table")

    conn.close()


if __name__ == "__main__":
    # Get database directory from environment variable or use default
    db_dir = os.environ.get("DB_DIR", os.path.join(os.getcwd(), "data/db"))

    # Ensure the directory exists
    os.makedirs(db_dir, exist_ok=True)

    # Configure database path
    db_path = os.path.join(db_dir, "portfolio_manager.db")
    print(f"Using database at: {db_path}")

    migrate(db_path)
