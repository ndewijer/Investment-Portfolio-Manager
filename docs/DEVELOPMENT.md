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

[Detailed development documentation...]
