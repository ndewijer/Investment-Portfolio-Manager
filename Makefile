.PHONY: run build test test-short test-verbose coverage coverage-html coverage-threshold lint lint-fix fmt govulncheck ci-local clean deps help docker-build docker-up docker-down docker-clean docker-logs

# Go backend settings
MODULE=github.com/ndewijer/Investment-Portfolio-Manager/backend
VERSION_PKG=$(MODULE)/internal/version
BACKEND_DIR=backend

# ─── Backend: Build & Run ────────────────────────────────────────────

run:
	cd $(BACKEND_DIR) && go run -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" ./cmd/server/main.go

build:
	cd $(BACKEND_DIR) && go build -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" -o bin/server ./cmd/server/main.go

# ─── Backend: Testing ────────────────────────────────────────────────

test:
	cd $(BACKEND_DIR) && go test -race ./...

test-short:
	cd $(BACKEND_DIR) && go test -short -race ./...

test-verbose:
	cd $(BACKEND_DIR) && go test -v -race ./...

coverage:
	cd $(BACKEND_DIR) && go test -coverprofile=coverage.out ./...
	@echo ""
	@cd $(BACKEND_DIR) && go tool cover -func=coverage.out | grep total

coverage-html: coverage
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: $(BACKEND_DIR)/coverage.html"

coverage-threshold:
	@echo "Running tests with coverage across all internal packages..."
	@cd $(BACKEND_DIR) && go test -coverpkg=./internal/... -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@cd $(BACKEND_DIR) && go tool cover -func=coverage.out | grep total
	@echo ""
	@cd $(BACKEND_DIR) && COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo "Total coverage: $${COVERAGE}%"; \
	THRESHOLD=75; \
	if [ $$(echo "$${COVERAGE} < $${THRESHOLD}" | bc -l) -eq 1 ]; then \
		echo "Coverage $${COVERAGE}% is below threshold $${THRESHOLD}%"; \
		exit 1; \
	else \
		echo "Coverage $${COVERAGE}% meets threshold $${THRESHOLD}%"; \
	fi

# ─── Backend: Code Quality ──────────────────────────────────────────

fmt:
	cd $(BACKEND_DIR) && go fmt ./...

lint:
	cd $(BACKEND_DIR) && golangci-lint run --timeout=5m

lint-fix:
	cd $(BACKEND_DIR) && golangci-lint run --fix --timeout=5m

govulncheck:
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "govulncheck not found. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	}
	cd $(BACKEND_DIR) && govulncheck ./...

# ─── Backend: Maintenance ───────────────────────────────────────────

deps:
	cd $(BACKEND_DIR) && go mod download && go mod tidy

clean:
	rm -rf $(BACKEND_DIR)/bin/
	rm -f $(BACKEND_DIR)/coverage.out $(BACKEND_DIR)/coverage.html $(BACKEND_DIR)/coverage_*.out

# ─── CI ──────────────────────────────────────────────────────────────

ci-local: lint test coverage-threshold govulncheck
	@echo ""
	@echo "Local CI checks passed"

pre-commit-install:
	@command -v pre-commit >/dev/null 2>&1 || { \
		echo "pre-commit not found. Installing..."; \
		pip install pre-commit || brew install pre-commit; \
	}
	pre-commit install
	@echo "Pre-commit hooks installed"

pre-commit-run:
	pre-commit run --all-files

# ─── Docker ──────────────────────────────────────────────────────────

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-clean:
	docker compose down --rmi all --volumes --remove-orphans

docker-logs:
	docker compose logs -f

# ─── Help ────────────────────────────────────────────────────────────

help:
	@echo "Available targets:"
	@echo ""
	@echo "Backend - Build & Run:"
	@echo "  run              - Run the backend application"
	@echo "  build            - Build the backend binary"
	@echo ""
	@echo "Backend - Testing:"
	@echo "  test             - Run all tests with race detector"
	@echo "  test-short       - Run tests in short mode (skip slow tests)"
	@echo "  test-verbose     - Run tests with verbose output"
	@echo ""
	@echo "Backend - Coverage:"
	@echo "  coverage         - Run tests with coverage summary"
	@echo "  coverage-html    - Generate HTML coverage report"
	@echo "  coverage-threshold - Check coverage meets 75% threshold (for CI)"
	@echo ""
	@echo "Backend - Code Quality:"
	@echo "  fmt              - Format Go code"
	@echo "  lint             - Run golangci-lint"
	@echo "  lint-fix         - Run golangci-lint with auto-fix"
	@echo "  govulncheck      - Run security vulnerability scan"
	@echo ""
	@echo "CI/CD:"
	@echo "  ci-local         - Run all CI checks locally"
	@echo "  pre-commit-install - Install pre-commit hooks"
	@echo "  pre-commit-run   - Run pre-commit hooks on all files"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build     - Build Docker images"
	@echo "  docker-up        - Start containers"
	@echo "  docker-down      - Stop containers"
	@echo "  docker-clean     - Remove containers, images, and volumes"
	@echo "  docker-logs      - Follow container logs"
	@echo ""
	@echo "Maintenance:"
	@echo "  deps             - Download and tidy Go dependencies"
	@echo "  clean            - Clean build artifacts and coverage files"
	@echo "  help             - Display this help message"
