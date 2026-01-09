# Development Setup

## Prerequisites
- Docker and Docker Compose (recommended)
- Node.js 23+ and Python 3.13+ (for local development)

## Local Development

### Backend Setup

#### Using uv (Recommended)
```bash
# Install uv if not already installed
curl -LsSf https://astral.sh/uv/install.sh | sh

# Install all dependencies (from project root)
uv sync --frozen

# Run Flask
cd backend
uv run flask run
```

#### Using pip (Deprecated - will be removed in v1.4.0)
```bash
cd backend
python -m venv .venv
source .venv/bin/activate
pip install -r dev-requirements.txt
flask run
```

**Note**: All dependencies are managed in `pyproject.toml`. Requirements files are kept for backward compatibility only.

### Frontend Setup
```bash
cd frontend
npm install
npm start
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
uv run python run.py

# Terminal 2: Start frontend
cd frontend
npm start

# Terminal 3: Install Playwright browsers (first time only)
cd frontend
npx playwright install chromium

# Terminal 3: Run E2E tests
npm run test:e2e
```

**Note**: E2E tests are **NOT** run in pre-commit hooks by default (too slow). They run in CI automatically.

## Development Tools
- ESLint and Prettier for frontend
- Ruff for backend (linting and formatting)
- uv for Python package management
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
- **Python**: Ruff linting and formatting, pytest tests
- **Frontend**: ESLint, Prettier
- **Docker**: Integration tests (when Docker-related files change)

**Docker Integration Tests:**

The Docker integration test hook runs automatically when you modify Docker-related files:
- `backend/Dockerfile`
- `frontend/Dockerfile`
- `docker-compose.yml`
- `pyproject.toml` / `uv.lock`
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
- ⚠️ If Docker isn't installed: Hook skips with warning message
- 🐳 If Docker is installed: Runs integration test (waits up to 60 seconds for backend)
- ✅ Success: Containers build, all health checks pass, cleanup completes
- ❌ Failure: Shows detailed error messages and container logs, prevents commit

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

### Adding Dependencies
**Production** (pinned version):
```bash
# 1. Edit pyproject.toml [project.dependencies], add: "package==1.2.3"
# 2. Update lock and sync
uv lock
uv sync --frozen
```

**Development** (flexible range):
```bash
# 1. Edit pyproject.toml [dependency-groups.dev], add: "package>=1.0.0,<2.0.0"
# 2. Update lock and sync
uv lock
uv sync --frozen
```

### Updating Dependencies
```bash
# Update all to latest compatible versions
uv lock --upgrade
uv sync

# Update specific package
uv lock --upgrade-package package-name
uv sync
```

### Common Commands
| Task | Command |
|------|---------|
| Install dependencies | `uv sync --frozen` |
| Run tests | `uv run pytest backend/tests/` |
| Run Flask | `cd backend && uv run flask run` |
| Run linting | `uv run ruff check backend/` |

## Environment Setup
```bash
# Security (see docs/SECURITY.md for details)
INTERNAL_API_KEY=your-secure-key-here  # For automated tasks
IBKR_ENCRYPTION_KEY=<auto-generated>   # Optional: Encrypts IBKR tokens

# Paths
DB_DIR=/path/to/data/db
LOG_DIR=/path/to/data/logs

# Development/Debugging
FLASK_LOG_RESPONSE_TIME=true           # Optional: Log response times in terminal
```

**📖 Security Configuration:**
See [docs/SECURITY.md](SECURITY.md) for complete information on:
- API key management
- IBKR token encryption
- Key generation and rotation
- Backup and migration procedures

### Response Time Logging

Enable detailed request logging with response times for debugging and performance monitoring.

**Configuration:**
```bash
# In .env file
FLASK_LOG_RESPONSE_TIME=true
```

**Output Format:**
```
127.0.0.1 - - [09/Jan/2026 19:56:14] "GET /api/portfolio/history HTTP/1.1" 200 - 45ms
127.0.0.1 - - [09/Jan/2026 19:56:15] "POST /api/transactions HTTP/1.1" 201 - 123ms
```

**Behavior:**
- **Enabled (`true`)**: Replaces default Flask/Werkzeug logging with custom format including response time
- **Disabled (`false` or not set)**: Uses standard Flask/Werkzeug logging without timing

**Use Cases:**
- Performance profiling during development
- Debugging slow API endpoints
- Monitoring request patterns
- Testing optimization improvements

**Note**: This is a development feature. In production, consider using proper APM tools (Application Performance Monitoring) like New Relic, DataDog, or Prometheus.

## Automated Tasks
The application includes automated tasks for:
- Daily fund price updates (weekdays at 23:55)
- System maintenance and cleanup

These tasks require:
- Valid INTERNAL_API_KEY in environment
- Proper application context
- Database access

## Database Management

### Version Control
The application uses a centralized version management system:
```bash
# Check current version
cat backend/VERSION

# Create new migration
flask db revision -m "description"

# Apply migrations
flask db upgrade

# Rollback if needed
flask db downgrade
```

### Database Setup
```bash
# Fresh installation
flask db upgrade   # Creates and initializes database

# Development reset
flask db downgrade base  # Reset to empty state
flask db upgrade        # Apply all migrations
flask seed-db           # Add sample data

# Update seed price data
flask update-seed-prices          # Fetch latest year of prices from yfinance
flask update-seed-prices --days 730  # Fetch 2 years of historical data
```

### Migration Development
When creating new migrations:
1. Update VERSION file with new version
2. Create migration script
3. Include proper error handling for:
   - Existing tables/indexes
   - Missing tables/indexes
   - Transaction management

Example migration pattern:
```python
from sqlalchemy.exc import OperationalError

def upgrade():
    try:
        # Create tables/indexes
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

def downgrade():
    try:
        # Drop tables/indexes
    except OperationalError as e:
        if "no such table" not in str(e):
            raise e
```

[Detailed development documentation...]
