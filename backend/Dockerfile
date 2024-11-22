FROM python:3.13-slim

# Set working directory
WORKDIR /app

RUN pip install gunicorn

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    FLASK_APP=run.py \
    DB_DIR=/data/db \
    LOG_DIR=/data/logs \
    DOMAIN=localhost

# Copy requirements file
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application
COPY . .

# Create directories for database and logs
RUN mkdir -p $DB_DIR $LOG_DIR

# Generate random API key if not provided
RUN echo '#!/bin/sh\n\
if [ -z "$INTERNAL_API_KEY" ]; then\n\
  export INTERNAL_API_KEY=$(python -c "import secrets; print(secrets.token_urlsafe(32))")\n\
  echo "Generated INTERNAL_API_KEY: $INTERNAL_API_KEY"\n\
fi\n\
exec "$@"' > /entrypoint.sh && chmod +x /entrypoint.sh

# Expose port
EXPOSE 5000

# Set the entrypoint
ENTRYPOINT ["/entrypoint.sh"]

# Run the application
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "run:app"]
