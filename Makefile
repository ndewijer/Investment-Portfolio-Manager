.PHONY: build up down clean logs test

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