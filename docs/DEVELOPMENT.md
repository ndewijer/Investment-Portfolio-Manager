# Development Setup

## Prerequisites
- Docker and Docker Compose (recommended)
- Go 1.26+ (for local backend development)
- Node.js 23+ (for local frontend development)
- [golangci-lint](https://golangci-lint.run/) (for backend linting)

## Local Development

### Backend Setup

#### Using Make (Recommended)
```bash
# From project root
make deps       # Download and tidy Go dependencies
make run        # Run backend with version injection
make test       # Run all tests with race detector
make lint       # Run golangci-lint
```

#### Using Go Directly
```bash
cd backend
go run ./cmd/server/main.go

# Or build and run
go build -o bin/server ./cmd/server/main.go
./bin/server
```

The server runs on `http://localhost:5000` by default. The database is created automatically on first run with all migrations applied.

### Frontend Setup
```bash
cd frontend
pnpm install
pnpm start
```

### Playwright E2E Tests
E2E tests require both backend and frontend servers to be running.

**Recommended: Run E2E tests with Docker (easiest):**
```bash
# Run Docker integration tests with E2E tests included
RUN_E2E_TESTS=true ./scripts/test-docker-integration.sh
```

This will:
1. Build and start the full Docker stack
2. Wait for services to be healthy
3. Run API integration tests
4. Run Playwright E2E tests against the Docker stack
5. Clean up containers when done

**Alternative: Run E2E tests against local services:**
```bash
# Terminal 1: Start backend
cd backend
go run ./cmd/server/main.go

# Terminal 2: Start frontend
cd frontend
pnpm start

# Terminal 3: Install Playwright browsers (first time only)
cd frontend
npx playwright install chromium

# Terminal 3: Run E2E tests
pnpm run test:e2e
```

**Note**: E2E tests are **NOT** run in pre-commit hooks by default (too slow). They run in CI automatically.

## Development Tools
- Biome for frontend (linting and formatting)
- golangci-lint for backend (linting)
- `go fmt` / `goimports` for backend formatting
- Pre-commit hooks for code quality

### Pre-commit Hooks

The project uses pre-commit hooks to ensure code quality before commits. Hooks are configured in `.pre-commit-config.yaml`.

**Installation:**
```bash
pip install pre-commit
pre-commit install
```

**Available Hooks:**
- **Code formatting**: Trailing whitespace, end-of-file fixer
- **Validation**: YAML and JSON syntax checking
- **Backend**: golangci-lint, go test
- **Frontend**: Biome (lint + format)
- **Docker**: Integration tests (when Docker-related files change)

**Docker Integration Tests:**

The Docker integration test hook runs automatically when you modify Docker-related files:
- `backend/Dockerfile`
- `frontend/Dockerfile`
- `docker-compose.yml`
- `backend/go.mod` / `backend/go.sum`
- `frontend/nginx.conf`

**How it works:**
- Executes `scripts/test-docker-integration.sh`
- Builds containers, tests health endpoints, verifies integration
- Automatically cleans up on exit (success or failure)

**Requirements:**
- **Docker Desktop** or **Docker Engine** must be installed and running
- If Docker is not available, the hook will skip gracefully with a warning

**Running manually:**
```bash
./scripts/test-docker-integration.sh
```

**Running Hooks Manually:**
```bash
# Run all hooks on changed files
pre-commit run

# Run all hooks on all files
pre-commit run --all-files

# Run specific hook
pre-commit run docker-integration-test

# Run Docker tests directly (faster for debugging)
./scripts/test-docker-integration.sh

# Skip hooks for a commit (not recommended)
git commit --no-verify
```

**Expected Behavior:**
- If Docker isn't installed: Hook skips with warning message
- If Docker is installed: Runs integration test (waits up to 60 seconds for backend)
- Success: Containers build, all health checks pass, cleanup completes
- Failure: Shows detailed error messages and container logs, prevents commit

**Troubleshooting:**
```bash
# If Docker tests fail locally:
docker compose down -v  # Clean up any stuck containers
docker system prune     # Clean Docker cache if needed

# Run the test script with verbose output:
./scripts/test-docker-integration.sh

# If you need to commit without Docker tests:
git commit --no-verify  # Use sparingly!
```

## Dependency Management

### Backend (Go Modules)

**Adding Dependencies:**
```bash
cd backend
go get github.com/some/package@v1.2.3
go mod tidy
```

**Updating Dependencies:**
```bash
cd backend
# Update all to latest compatible versions
go get -u ./...
go mod tidy

# Update specific package
go get -u github.com/some/package
go mod tidy
```

### Common Make Targets

| Target | Description |
|--------|-------------|
| `make run` | Run the application with version injection |
| `make build` | Build binary to `bin/server` |
| `make test` | Run all tests with race detector |
| `make test-short` | Skip slow tests |
| `make lint` | Run golangci-lint |
| `make coverage` | Run tests with coverage summary |
| `make coverage-html` | Generate HTML coverage report |
| `make fmt` | Format code |
| `make deps` | Download and tidy dependencies |
| `make ci-local` | Run full CI pipeline locally |
| `make clean` | Remove build artifacts and coverage files |

## Environment Setup
```bash
# Security (see docs/SECURITY.md for details)
INTERNAL_API_KEY=your-secure-key-here  # For automated tasks
IBKR_ENCRYPTION_KEY=<auto-generated>   # Optional: Encrypts IBKR tokens

# Server
SERVER_PORT=5000                       # HTTP server port (default: 5000)
SERVER_HOST=0.0.0.0                    # HTTP server bind address

# Database
DB_PATH=./data/portfolio_manager.db    # Path to SQLite database file
DB_DIR=/path/to/data/db                # Directory for database file (Docker, takes precedence over DB_PATH)

# Logging
LOG_DIR=./data/logs                    # Directory for log files

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000  # Comma-separated list of allowed origins
DOMAIN=example.com                          # Generates http:// and https:// origins
```

See [docs/CONFIGURATION.md](CONFIGURATION.md) for the complete environment variable reference.

**Security Configuration:**
See [docs/SECURITY.md](SECURITY.md) for complete information on:
- API key management
- IBKR token encryption
- Key generation and rotation
- Backup and migration procedures

### Structured Logging

The Go backend uses structured logging with database-configurable levels. Logging levels and categories can be adjusted at runtime via the `/api/developer/system-settings/logging` endpoints.

Each request gets a unique request ID injected by middleware, making it easy to trace individual requests through the log output.

**Use Cases:**
- Performance profiling during development
- Debugging slow API endpoints
- Monitoring request patterns
- Testing optimization improvements

## Automated Tasks
The application includes automated tasks for:
- Daily fund price updates (weekdays at 00:55 UTC)
- Weekly IBKR transaction imports (Tue-Sat between 05:30 and 07:30 UTC)
- System maintenance and cleanup

These tasks require:
- Valid INTERNAL_API_KEY in environment
- Scheduled via in-process `robfig/cron`
- Both use `SkipIfStillRunning` to prevent overlap
- 15-minute timeouts per task

## Database Management

### Version Control
The application uses Goose for database migrations. Migrations are embedded in the Go binary and run automatically on server startup.

```bash
# Check current version
cat backend/VERSION

# Migrations run automatically on startup
# No manual migration commands needed
```

### Database Setup
```bash
# Fresh installation — just start the server
cd backend
go run ./cmd/server/main.go
# Database is created and all migrations applied automatically

# Inspect the database
sqlite3 ./data/portfolio_manager.db ".tables"
sqlite3 ./data/portfolio_manager.db ".schema funds"
```

### Migration Development
Migrations are managed by Goose and stored in `backend/internal/database/migrations/`. Both SQL and Go-coded migrations are supported.

When creating new migrations:
1. Update VERSION file with new version
2. Create migration file in `backend/internal/database/migrations/`
3. Include proper error handling

Example SQL migration:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS new_table (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS new_table;
```

Go-coded migrations are available for operations that require application logic (e.g., data backfills). These are registered via `registerGoMigrationVersion` in the `database` package and run automatically alongside SQL migrations.

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
