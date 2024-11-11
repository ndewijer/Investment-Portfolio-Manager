FROM python:3.13-slim

# Set working directory
WORKDIR /app

RUN pip install gunicorn

# Copy requirements file
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application
COPY . .

# Create directories for database and logs with environment variables
ENV DB_DIR=/data/db \
    LOG_DIR=/data/logs \
    DOMAIN=localhost

RUN mkdir -p $DB_DIR $LOG_DIR

# Expose port
EXPOSE 5000

# Run the application
#CMD ["flask", "run", "--app", "run:app", "--host=0.0.0.0"]
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "run:app"]