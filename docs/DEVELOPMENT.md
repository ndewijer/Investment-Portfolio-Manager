# Development Setup

## Prerequisites
- Docker and Docker Compose (recommended)
- Node.js 23+ and Python 3.13+ (for local development)

## Local Development

### Backend Setup
```bash
cd backend
python -m venv venv
source venv/bin/activate
pip install -r dev-requirements.txt
flask run
```

### Frontend Setup
```bash
cd frontend
npm install
npm start
```

## Development Tools
- ESLint and Prettier for frontend
- Black and Flake8 for backend
- Pre-commit hooks for code quality

## Environment Setup
```bash
# Required environment variables
INTERNAL_API_KEY=your-secure-key-here  # For automated tasks
DB_DIR=/path/to/data/db
LOG_DIR=/path/to/data/logs
```

## Automated Tasks
The application includes automated tasks for:
- Daily fund price updates (weekdays at 23:55)
- System maintenance and cleanup

These tasks require:
- Valid INTERNAL_API_KEY in environment
- Proper application context
- Database access

## Database Management

### Version Control
The application uses a centralized version management system:
```bash
# Check current version
cat backend/VERSION

# Create new migration
flask db revision -m "description"

# Apply migrations
flask db upgrade

# Rollback if needed
flask db downgrade
```

### Database Setup
```bash
# Fresh installation
flask db upgrade   # Creates and initializes database

# Development reset
flask db downgrade base  # Reset to empty state
flask db upgrade        # Apply all migrations
flask seed-db           # Add sample data
```

### Migration Development
When creating new migrations:
1. Update VERSION file with new version
2. Create migration script
3. Include proper error handling for:
   - Existing tables/indexes
   - Missing tables/indexes
   - Transaction management

Example migration pattern:
```python
from sqlalchemy.exc import OperationalError

def upgrade():
    try:
        # Create tables/indexes
    except OperationalError as e:
        if "already exists" not in str(e):
            raise e

def downgrade():
    try:
        # Drop tables/indexes
    except OperationalError as e:
        if "no such table" not in str(e):
            raise e
```

[Detailed development documentation...]
