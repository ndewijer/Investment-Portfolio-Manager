#!/bin/sh
set -e

echo "========================================="
echo "Investment Portfolio Manager - Starting"
echo "========================================="

DB_FILE="${DB_DIR:-/data/db}/portfolio_manager.db"

if [ ! -f "$DB_FILE" ]; then
    echo "[INFO] Database not found - this is a fresh installation"
else
    echo "[INFO] Database found at: $DB_FILE"
fi

echo "[INFO] Running database migrations..."
echo "[INFO] Migrations will run automatically on startup"

echo "========================================="
echo "Starting application server..."
echo "========================================="

exec "$@"
