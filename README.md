# Investment Portfolio Manager

A web application for managing investment portfolios, tracking transactions, and monitoring dividends.

## Features
- Portfolio management
- Transaction tracking
- Dividend management
- Fund price history
- Exchange rate management
- System logging
- CSV import/export

## Prerequisites
- Docker and Docker Compose (recommended)
- Node.js 23+ (for local development only)
- Python 3.13+ (for local development only)

## Setup and Running

### Docker Compose Setup (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/ndewijer/investment-portfolio-manager.git
cd investment-portfolio-manager
```

2. Create .env file in the root directory:
```bash
# Set your domain (defaults to localhost if not set)
DOMAIN=your-domain.com

# Database and logs location
DB_DIR=/path/to/your/data/db
LOG_DIR=/path/to/your/data/logs
```

3. Build and start the containers:
```bash
docker compose up -d
```

4. Initialize the database:
```bash
docker compose exec backend flask seed-db
```

The application will be available at:
- Production: https://your-domain.com (or http://localhost if no domain set)
- Development: http://localhost:3000

### Domain Configuration

The application supports custom domains through environment variables:

1. Production Setup:
   - Set DOMAIN in .env file
   - Frontend automatically configures Nginx
   - Backend CORS settings adapt to the domain
   - API calls use the correct domain

2. Development Setup:
   - Uses localhost by default
   - Frontend runs on port 3000
   - Backend runs on port 5000
   - No domain configuration needed

### Local Development Setup

#### Backend Setup
1. Navigate to the backend directory:
```bash
cd backend
```

2. Create a virtual environment:
```bash
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

3. Install dependencies:
```bash
pip install -r requirements.txt
```

4. Create .env file:
```bash
FLASK_APP=run.py
FLASK_ENV=development
FLASK_DEBUG=1
DATABASE_URL=sqlite:///portfolio_manager.db
LOG_DIR=logs
```

5. Run the development server:
```bash
flask run
```

#### Frontend Setup
1. Navigate to the frontend directory:
```bash
cd frontend
```

2. Install dependencies:
```bash
npm install
```

3. Create .env file:
```bash
REACT_APP_API_URL=http://localhost:5000/api
```

4. Run the development server (Webpack):
```bash
npm start
```

### Development Tools Setup

#### Pre-commit Hooks
The project uses pre-commit hooks to ensure code quality:

1. Install pre-commit:
```bash
pip install pre-commit
```

2. Install the hooks:
```bash
pre-commit install
```

Pre-commit will now run automatically on `git commit`, checking:
- Code formatting (black, prettier)
- Linting (flake8)
- File formatting (trailing spaces, EOF)
- YAML/JSON validation
- Large file checks

#### Backend Development Dependencies
The backend uses separate requirement files:
- `requirements.txt`: Production dependencies
- `dev-requirements.txt`: Development tools (linting, formatting)

Install development dependencies:
```bash
cd backend
pip install -r dev-requirements.txt
```

Development tools include:
- black: Code formatting
- flake8: Linting with docstring checking
- isort: Import sorting
- autopep8: Code style fixing
- pydocstyle: Docstring style checking

#### Frontend Development Tools
The frontend includes:
- ESLint: JavaScript/React linting
- Prettier: Code formatting
- Webpack: Build and development server

Run checks manually:
```bash
cd frontend
npm run lint    # Run ESLint
npm run format  # Run Prettier
```

### Docker Setup

The application uses a multi-container setup:

#### Backend Container
- Base: Python 3.13-slim
- Gunicorn as WSGI server
- Environment variables:
  - DB_DIR: Database location
  - LOG_DIR: Log files location
  - DOMAIN: Server domain name

#### Frontend Container
- Multi-stage build:
  1. Node.js 23 for building
  2. Nginx for serving
- Environment variables:
  - DOMAIN: Server domain name
- Nginx configured for:
  - Static file serving
  - API proxying
  - React routing support

#### Volume Management
Persistent data is stored in:
```
./data/
  ├── db/    # Database files
  └── logs/  # Application logs
```

#### Domain Configuration
Set your domain in the root .env file:
```bash
DOMAIN=your-domain.com  # Defaults to localhost
```

## Architecture

### Frontend
- Production (Docker):
  - Nginx serving static files on port 80
  - Handles all frontend routes
  - Proxies /api requests to backend
  - Supports custom domain configuration
  - Configured for React router support
- Development: Webpack dev server on port 3000

### Backend
- Production (Docker):
  - Gunicorn WSGI server on port 5000
  - Accessed through Nginx proxy at /api
  - Dynamic CORS configuration based on domain
  - Persistent data storage in mounted volume
- Development: Flask development server on port 5000

## Project Structure
```
Investment-Portfolio-Manager/
├── backend/
│   ├── app/
│   │   ├── models/
│   │   ├── routes/
│   │   └── services/
│   ├── data/
│   │   ├── db/
│   │   └── logs/
│   └── tests/
└── frontend/
    ├── public/
    └── src/
        ├── components/
        ├── pages/
        └── services/
```

## Technologies Used

### Backend
- Python 3.13
- Gunicorn (Production WSGI Server)
- Flask (Development Server)
- SQLAlchemy (ORM)
- SQLite (Database)
- Flask-CORS (Cross-Origin Resource Sharing)
- PyTZ (Timezone handling)

### Frontend
- Node.js 23
- React 18
- Nginx (Production Server)
- Webpack (Development Server)
- React Router (Navigation)
- Axios (HTTP client)
- React-DatePicker (Date selection)
- React-Multi-Select (Advanced select inputs)
- FontAwesome (Icons)


## Features in Detail

### Portfolio Management
- Create and manage multiple investment portfolios
- Track portfolio performance over time
- View portfolio composition and allocation
- Archive unused portfolios
- Calculate total portfolio value in different currencies

### Transaction Management
- Record buy and sell transactions
- Import transactions via CSV
- Filter and sort transactions
- Track transaction history
- Calculate gains and losses

### Dividend Management
- Track cash and stock dividends
- Automatic dividend reinvestment tracking
- Dividend payment status monitoring
- Historical dividend records
- Calculate total dividend income

### System Logging
- Comprehensive logging system
- Multiple log levels (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- Configurable logging settings
- Log viewer with filtering and sorting
- Database and file-based logging

### Developer Tools
- Exchange rate management
- Fund price import/export
- CSV templates for data import
- System settings configuration
- Logging configuration

## Contributing
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License
This project is licensed under the Apache License, Version 2.0 - see the LICENSE file for details
