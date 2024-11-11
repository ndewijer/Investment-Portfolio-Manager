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

2. Create data directory for persistent storage:
```bash
mkdir -p /path/to/your/data/{db,logs}
```

3. Update docker-compose.yml with your data directory:
```yaml
volumes:
  - /path/to/your/data:/data
```

4. Build and start the containers:
```bash
docker compose up -d
```

5. Initialize the database:
```bash
docker compose exec backend flask seed-db
```

The application will be available at:
- Production: http://localhost:80 (or just http://localhost)
- Development: http://localhost:3000

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

## Architecture

### Frontend
- Production (Docker): 
  - Nginx serving static files on port 80
  - Handles all frontend routes
  - Proxies /api requests to backend
  - Configured for React router support
- Development: Webpack dev server on port 3000

### Backend
- Production (Docker): 
  - Gunicorn WSGI server on port 5000
  - Accessed through Nginx proxy at /api
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
