# Contributing Guidelines

## Development Setup

### Prerequisites
- Docker and Docker Compose
- Node.js 23+
- Go 1.26+
- [golangci-lint](https://golangci-lint.run/)

### Install golangci-lint
```bash
# macOS (Homebrew)
brew install golangci-lint

# Linux / macOS (install script)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Verify
golangci-lint --version
```

### Setup Steps
1. **Install pre-commit hooks**:
```bash
pip install pre-commit
pre-commit install
```

2. **Install dependencies**:
```bash
# Backend (from backend/)
cd backend && go mod download

# Frontend
cd frontend && pnpm install
```

## Code Style
- Frontend: Biome (lint + format)
- Backend: golangci-lint (linting), `go fmt` / `goimports` (formatting)
- Pre-commit hooks enforce style

## Testing
```bash
# Backend
cd backend && go test -race ./...

# Or using Make from project root
make test

# Frontend
cd frontend && pnpm test
```

## Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run linting and tests:
   ```bash
   # Backend
   cd backend
   golangci-lint run --timeout=5m
   go test -race ./...

   # Frontend
   cd frontend
   pnpm run check
   pnpm test
   ```
5. Commit your changes (pre-commit hooks will run automatically)
6. Push to your fork
7. Create a Pull Request

## Code Review
- All PRs require review
- CI must pass
- Documentation must be updated

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
