#!/usr/bin/env bash
# Verify that golden_schema.sql is updated whenever a migration file is staged.
# Run by pre-commit when backend/internal/database/migrations/* or golden_schema.sql is touched.
set -euo pipefail

STAGED=$(git diff --cached --name-only)

MIGRATIONS_STAGED=$(echo "$STAGED" | grep '^backend/internal/database/migrations/' || true)
GOLDEN_STAGED=$(echo "$STAGED" | grep '^backend/internal/database/testdata/golden_schema.sql' || true)

if [ -n "$MIGRATIONS_STAGED" ] && [ -z "$GOLDEN_STAGED" ]; then
    echo "Migration files staged but golden_schema.sql was not updated."
    echo "Run: cd backend && go test ./internal/database/... -update-golden"
    exit 1
fi
