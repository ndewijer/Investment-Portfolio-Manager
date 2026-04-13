# Claude Code Instructions

## Development Tools

**This project uses Go for the backend and pnpm for Node.js package management.**

### Backend Development
- Backend code is in `backend/` with `go.mod` at `backend/go.mod`
- Run from repo root using Make targets, or `cd backend` for direct Go commands
- Examples:
  ```bash
  make run           # Run backend with version injection
  make test          # Run all tests with race detector
  make test-short    # Run tests skipping slow tests
  make lint          # Run golangci-lint
  make coverage      # Run tests with coverage summary
  make ci-local      # Run full CI pipeline locally
  ```
- Direct Go commands (from `backend/`):
  ```bash
  cd backend
  go test -race ./...
  go build ./cmd/server/main.go
  golangci-lint run --timeout=5m
  ```

### Frontend Development
- Use `pnpm` instead of `npm` for all frontend commands
- Examples:
  ```bash
  cd frontend
  pnpm install
  pnpm run build
  pnpm run test:ci
  pnpm run start
  ```

### Docker
```bash
make docker-build   # Build images
make docker-up      # Start containers
make docker-down    # Stop containers
make docker-logs    # Follow logs
```

## Memory System

This project uses Claude's `.claudememory/` system for persistent context across sessions.

**Key Documentation Files:**
- `.claudememory/project_context.md` - Technology stack, architecture patterns, database info
- `.claudememory/development_workflow.md` - Development processes and documentation maintenance
- `.claudememory/release_process.md` - PR and release management
- `.claudememory/testing_documentation.md` - Testing strategy and documentation

**Note**: The `.claudememory/` directory is gitignored and managed by Claude Code automatically.
