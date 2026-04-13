# v2.0.0 Migration Guide: Python/Flask to Go Backend

## Overview

Version 2.0.0 of the Investment Portfolio Manager replaces the Python/Flask backend with a Go backend. This guide covers everything you need to know to upgrade from v1.x to v2.0.0.

### What Changed

- **Backend language**: Python 3.13 (Flask/Gunicorn) -> Go 1.26+ (Chi router)
- **Migration tool**: Alembic -> Goose (embedded in binary, runs automatically on startup)
- **Package management**: uv/pip/pyproject.toml -> Go modules (go.mod/go.sum)
- **Linting**: Ruff -> golangci-lint
- **Testing**: pytest -> Go's built-in `testing` package
- **Docker**: Python base image -> multi-stage Go build (smaller image, faster startup)

### What Did NOT Change

- **API contract**: All endpoints maintain the same paths, request/response formats, and behavior. The frontend is completely unaware of the backend language change.
- **Database**: Same SQLite database file, same schema, same tables. Your existing data is preserved.
- **Frontend**: No changes. Same React 19 app, same components, same build process.
- **Port**: Backend still listens on port 5000 (not exposed to host in Docker; accessed via Nginx proxy).
- **Environment variables**: Same `INTERNAL_API_KEY`, `IBKR_ENCRYPTION_KEY`, `DB_DIR`, `LOG_DIR`, `DOMAIN` variables. A few new ones added (see below).
- **IBKR integration**: Same Flex Query import flow, same inbox, same allocation process.
- **Scheduled tasks**: Same daily price updates and weekly IBKR imports (times adjusted slightly to UTC).

---

## Upgrading: Docker (Recommended)

If you run the application via Docker Compose, the upgrade is straightforward.

### Step 1: Backup Your Database

```bash
# Find your database file (check your docker-compose.yml for the volume mount)
cp data/db/portfolio_manager.db data/db/portfolio_manager.db.v1-backup
```

### Step 2: Pull the New Version

```bash
git pull origin main
# Or if using tags:
git checkout v2.0.0
```

### Step 3: Rebuild and Start

```bash
docker compose build
docker compose up -d
```

That's it. The new Go backend will:
1. Start up and connect to your existing SQLite database
2. Run any pending Goose migrations automatically
3. Serve the same API on the same endpoints

### Step 4: Verify

```bash
# Check health
curl http://localhost/api/system/health

# Check version (should show 2.0.0)
curl http://localhost/api/system/version

# Check logs
docker compose logs backend
```

### Docker Environment Variables

Your existing `.env` file should work as-is. The Go backend recognizes the same variables:

```bash
# These work the same as before
INTERNAL_API_KEY=your-secure-key-here
IBKR_ENCRYPTION_KEY=your-encryption-key
DB_DIR=/data/db
LOG_DIR=/data/logs
DOMAIN=your-domain.com

# New in v2.0.0 (optional)
SERVER_PORT=5000                        # Default: 5000
SERVER_HOST=0.0.0.0                     # Default: 0.0.0.0
CORS_ALLOWED_ORIGINS=http://localhost:3000  # Comma-separated; alternative to DOMAIN
```

---

## Upgrading: Local Development

If you develop locally (not in Docker), you need to switch from Python to Go tooling.

### Step 1: Install Go

```bash
# macOS (Homebrew)
brew install go

# Or download from https://go.dev/dl/
# Verify
go version  # Should be 1.26 or later
```

### Step 2: Install golangci-lint (Optional, for linting)

```bash
brew install golangci-lint
# Or:
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Step 3: Pull the New Version

```bash
git pull origin main
```

### Step 4: Run the Backend

```bash
# Using Make (recommended)
make run

# Or directly
cd backend
go run ./cmd/server/main.go
```

The server starts on `http://localhost:5000`. The database is created/migrated automatically.

### Step 5: Verify

```bash
curl http://localhost:5000/api/system/health
curl http://localhost:5000/api/system/version
```

### Old vs New: Command Reference

| Task | v1.x (Python) | v2.0.0 (Go) |
|------|---------------|--------------|
| **Start backend** | `cd backend && uv run flask run` | `make run` or `cd backend && go run ./cmd/server/main.go` |
| **Run tests** | `uv run pytest backend/tests/` | `make test` or `cd backend && go test -race ./...` |
| **Run linter** | `uv run ruff check backend/` | `make lint` or `cd backend && golangci-lint run` |
| **Format code** | `uv run ruff format backend/` | `make fmt` or `cd backend && go fmt ./...` |
| **Install dependencies** | `uv sync --frozen` | `cd backend && go mod download` |
| **Add dependency** | Edit `pyproject.toml` + `uv lock` | `cd backend && go get package@version` |
| **Build** | N/A (interpreted) | `make build` (creates `bin/server`) |
| **Run migrations** | `flask db upgrade` | Automatic on startup |
| **Check coverage** | `pytest --cov=app` | `make coverage` |
| **Database path** | Configured in Flask | `DB_PATH=./data/portfolio_manager.db` or `DB_DIR` env var |

---

## Database Migration Details

### Alembic to Goose

The v1.x backend used Alembic (Python) for database migrations with revision IDs matching version numbers (e.g., revision `"1.3.0"`). The v2.0.0 backend uses Goose (Go) with numbered SQL files.

**Important**: The Go backend includes a base migration (`162_base.sql`) that establishes the complete schema matching the Python backend's final database state. When Goose runs against an existing database:

1. It detects the existing schema
2. The base migration uses `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS`
3. No data is lost or modified
4. Goose records its own migration state in the `goose_db_version` table

Your existing `alembic_version` table remains in the database but is no longer used. It can be safely left in place or dropped manually if desired.

### What About My Existing Data?

Your data is **100% preserved**. The Go backend reads and writes the same SQLite database file with the same schema. The migration from Alembic to Goose is purely a tooling change -- your portfolio, fund, transaction, dividend, and IBKR data are untouched.

---

## Scheduled Task Timing Changes

The scheduled task times have been adjusted slightly for the Go backend:

| Task | v1.x | v2.0.0 |
|------|------|--------|
| Daily price update | Weekdays at 23:55 (local) | Weekdays at 00:55 UTC |
| IBKR import | Tue-Sat 06:30-08:30 (local) | Tue-Sat 05:30-07:30 UTC |

The Go backend uses UTC times exclusively. If your server was running in a specific timezone, the effective times may differ. The scheduled tasks now also include:
- `SkipIfStillRunning` to prevent overlapping runs
- 15-minute timeouts per task

---

## Rolling Back to v1.x

If you need to roll back to the Python backend:

### Using Git Tags

The last Python version is tagged as `v1.7.0-python-final`:

```bash
# Restore the Python version
git checkout v1.7.0-python-final

# Rebuild and start
docker compose build
docker compose up -d
```

### Database Compatibility

The database file is forward-compatible. If you ran v2.0.0 and want to go back:
- Your data is safe (v2.0.0 uses the same schema)
- The `goose_db_version` table will exist but is ignored by Alembic
- The `alembic_version` table is still present and used by the Python backend

### Restoring from Backup

If you backed up before upgrading:

```bash
# Stop containers
docker compose down

# Restore backup
cp data/db/portfolio_manager.db.v1-backup data/db/portfolio_manager.db

# Checkout Python version and restart
git checkout v1.7.0-python-final
docker compose build
docker compose up -d
```

---

## Breaking Changes

There are **no breaking API changes** in v2.0.0. The API contract is identical.

Minor behavioral differences:
- **Error messages**: Some error message strings may differ slightly (e.g., "portfolio not found" vs "Portfolio not found"). The HTTP status codes and error response structure are the same.
- **Request logging**: Log format has changed to structured logging with request IDs. The log content is similar but the format is different.
- **Startup**: The Go backend starts faster (compiled binary vs Python interpreter startup).

---

## Removed: Python-Specific Files

The following files from v1.x are no longer present:
- `pyproject.toml`, `uv.lock` — replaced by `backend/go.mod`, `backend/go.sum`
- `backend/requirements.txt`, `backend/dev-requirements.txt` — no longer needed
- `backend/app/` — replaced by `backend/internal/` (Go project layout)
- `backend/migrations/` (Alembic) — replaced by `backend/internal/database/migrations/` (Goose)
- `backend/pytest.ini`, `backend/conftest.py` — Go tests use built-in testing framework
- `backend/run.py` — replaced by `backend/cmd/server/main.go`

---

## New: Configuration Reference

See [CONFIGURATION.md](CONFIGURATION.md) for the complete environment variable reference, including new v2.0.0 variables like `SERVER_PORT`, `SERVER_HOST`, and `CORS_ALLOWED_ORIGINS`.

---

## FAQ

**Q: Do I need to re-import my IBKR transactions?**
A: No. Your IBKR configuration, imported transactions, and allocations are all preserved in the database.

**Q: Will my existing Docker volumes work?**
A: Yes. The database file path inside Docker (`/data/db/portfolio_manager.db`) is the same.

**Q: Do I need to regenerate my encryption key?**
A: No. The Go backend uses the same Fernet encryption format. Your existing `IBKR_ENCRYPTION_KEY` works as-is. If you were using the auto-generated key from `data/.ibkr_encryption_key`, that file is also read by the Go backend.

**Q: Can I still use Python for development?**
A: The backend is now Go-only. You still need Node.js/pnpm for frontend development. Python is no longer needed for any part of the project.

**Q: What Go version do I need?**
A: Go 1.26 or later. Check with `go version`.

**Q: Is the API documentation (Swagger UI) still available?**
A: The Flask-RESTX Swagger UI at `/api/docs` is no longer available. Use the endpoint tables in [API_DOCUMENTATION.md](API_DOCUMENTATION.md) for reference. The API itself is unchanged.

---

## Resources

- [Go installation guide](https://go.dev/doc/install)
- [golangci-lint installation](https://golangci-lint.run/welcome/install/)
- [Goose documentation](https://pressly.github.io/goose/)
- [API Documentation](API_DOCUMENTATION.md)
- [Architecture Overview](ARCHITECTURE.md)
- [Configuration Reference](CONFIGURATION.md)

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
