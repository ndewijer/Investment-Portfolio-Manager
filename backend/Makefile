.PHONY: build run clean test seed fix-style

# Variables
IMAGE_NAME = investment-portfolio-backend
CONTAINER_NAME = investment-portfolio-backend
PORT = 5000
DB_DIR ?= $(shell pwd)/data/db
LOG_DIR ?= $(shell pwd)/data/logs

# Create required directories
setup-dirs:
	mkdir -p $(DB_DIR) $(LOG_DIR)

# Build the Docker image
build: setup-dirs
	docker build -t $(IMAGE_NAME) .

# Run the container
run:
	docker run -d \
		--name $(CONTAINER_NAME) \
		-p $(PORT):$(PORT) \
		-v $(DB_DIR):/data/db \
		-v $(LOG_DIR):/data/logs \
		--env-file .env \
		$(IMAGE_NAME)

# Stop and remove the container
clean:
	docker stop $(CONTAINER_NAME) || true
	docker rm $(CONTAINER_NAME) || true

# Run tests
test:
	docker exec $(CONTAINER_NAME) python -m pytest

# Seed database
seed:
	docker exec $(CONTAINER_NAME) flask seed-db

# Rebuild and restart
restart: clean build run

# Show logs
logs:
	docker exec $(CONTAINER_NAME) cat /data/logs/app.log

# Clear logs
clear-logs:
	docker exec $(CONTAINER_NAME) sh -c "truncate -s 0 /data/logs/app.log"
