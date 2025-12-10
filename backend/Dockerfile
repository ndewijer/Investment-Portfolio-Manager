# Multi-stage build for optimal image size
FROM python:3.13-slim AS builder

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

# Set working directory
WORKDIR /app

# Copy dependency files from project root
COPY pyproject.toml uv.lock ./

# Install dependencies (production only)
RUN uv sync --frozen --no-dev

# Install gunicorn (Docker-only dependency)
RUN uv pip install gunicorn==23.0.0

# Final stage
FROM python:3.13-slim

# Set working directory
WORKDIR /app

# Copy virtual environment from builder
COPY --from=builder /app/.venv /app/.venv

# Copy backend application code
COPY backend/ ./

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    FLASK_APP=run.py \
    DB_DIR=/data/db \
    LOG_DIR=/data/logs \
    DOMAIN=localhost \
    PATH="/app/.venv/bin:$PATH"

# Create directories
RUN mkdir -p $DB_DIR $LOG_DIR

# Copy entrypoint script
COPY backend/docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 5000

ENTRYPOINT ["/entrypoint.sh"]
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "run:app"]
