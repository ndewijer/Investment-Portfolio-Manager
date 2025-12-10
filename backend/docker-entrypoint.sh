#!/bin/sh
set -e

echo "========================================="
echo "Investment Portfolio Manager - Starting"
echo "========================================="

# Generate INTERNAL_API_KEY if not provided
if [ -z "$INTERNAL_API_KEY" ]; then
    KEY=$(python -c "import secrets; print(secrets.token_urlsafe(32))")
    echo "[INFO] Generated INTERNAL_API_KEY: $KEY"
    export INTERNAL_API_KEY=$KEY
fi

# Database file path
DB_FILE="${DB_DIR:-/data/db}/portfolio_manager.db"
DB_IS_NEW=false

# Check if database exists
if [ ! -f "$DB_FILE" ]; then
    echo "[INFO] Database not found - this is a fresh installation"
    DB_IS_NEW=true
else
    echo "[INFO] Database found at: $DB_FILE"
fi

# Always run migrations (safe operation - does nothing if already up to date)
echo "[INFO] Running database migrations..."
flask db upgrade

if [ "$?" -ne 0 ]; then
    echo "[ERROR] Database migration failed!"
    exit 1
fi

echo "[INFO] Database migrations completed successfully"

# Seed database if this is a fresh installation
if [ "$DB_IS_NEW" = true ]; then
    echo "[INFO] Fresh installation detected - seeding database with sample data..."

    if flask seed-db 2>&1; then
        echo "[INFO] Database seeded successfully with sample data"
    else
        echo "[ERROR] Database seeding failed"
        echo "[INFO] Your database schema is ready - the app will work with an empty database"
        echo "[INFO] You can add data through the UI"
    fi
else
    echo "[INFO] Existing database found - skipping seed data"
fi

echo "========================================="
echo "Starting application server..."
echo "========================================="

# Execute the main command (gunicorn)
exec "$@"
