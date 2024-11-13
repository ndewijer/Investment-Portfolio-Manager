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

[Detailed development documentation...]
