.PHONY: build up down clean logs test install sync update

# Python dependency management with uv
install:
	uv sync --frozen

sync:
	uv sync

update:
	uv lock --upgrade
	uv sync

# Run backend tests locally
test:
	uv run pytest backend/tests/

# Build all services
build:
	docker-compose build

# Start all services
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Clean up containers and images
clean:
	docker-compose down --rmi all --volumes --remove-orphans

# View logs
logs:
	docker-compose logs -f

# Initialize database
init-db:
	docker-compose exec backend flask seed-db
