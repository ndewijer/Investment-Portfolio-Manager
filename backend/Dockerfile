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

# Create entrypoint wrapper for environment setup
RUN echo '#!/bin/sh\n\
if [ -z "$INTERNAL_API_KEY" ]; then\n\
    KEY=$(python -c "import secrets; print(secrets.token_urlsafe(32))")\n\
    echo "Generated INTERNAL_API_KEY: $KEY"\n\
    export INTERNAL_API_KEY=$KEY\n\
fi\n\
exec "$@"' > /entrypoint.sh && chmod +x /entrypoint.sh

EXPOSE 5000

ENTRYPOINT ["/entrypoint.sh"]
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "run:app"]
