# Contributing Guidelines

## Development Setup

### Prerequisites
- Docker and Docker Compose
- Node.js 23+
- Python 3.13+
- **uv** package manager

### Install uv
```bash
# macOS/Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# Windows
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"

# Verify
uv --version
```

### Setup Steps
1. **Install pre-commit hooks**:
```bash
uv tool install pre-commit
pre-commit install
```

2. **Install dependencies**:
```bash
# Backend (from root)
uv sync --frozen

# Frontend
cd frontend && npm install
```

## Code Style
- Frontend: ESLint & Prettier
- Backend: Ruff (linting and formatting)
- Pre-commit hooks enforce style

## Testing
```bash
# Backend
uv run pytest backend/tests/ --cov=backend/app

# Frontend
cd frontend && npm test
```

## Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run linting and tests:
   ```bash
   # Backend
   uv run ruff check backend/
   uv run ruff format backend/
   uv run pytest backend/tests/

   # Frontend
   cd frontend
   npm run lint
   npm run format
   npm test
   ```
5. Commit your changes (pre-commit hooks will run automatically)
6. Push to your fork
7. Create a Pull Request

## Code Review
- All PRs require review
- CI must pass
- Documentation must be updated

---

**Last Updated**: 2025-12-02 (Version 1.3.4)
**Maintained By**: @ndewijer
