.PHONY: build up down clean logs test

# Detect docker compose command
DOCKER_COMPOSE := $(shell command -v docker-compose >/dev/null 2>&1 && echo docker-compose || echo docker compose)

# Build all services
build:
	$(DOCKER_COMPOSE) build

# Start all services
up:
	$(DOCKER_COMPOSE) up -d

# Stop all services
down:
	$(DOCKER_COMPOSE) down

# Clean up containers and images
clean:
	$(DOCKER_COMPOSE) down --rmi all --volumes --remove-orphans

# View logs
logs:
	$(DOCKER_COMPOSE) logs -f

# Initialize database
init-db:
	$(DOCKER_COMPOSE) exec backend flask seed-db
