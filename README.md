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
- Python 3.11+
- Node.js 20+
- Docker (optional)

## Setup and Running

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

5. Initialize the database:
```bash
flask seed-db
```

6. Run the backend:
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
NODE_ENV=development
```

4. Run the frontend:
```bash
npm start
```

### Docker Setup

The application can be run using Docker and Docker Compose with configurable storage locations.

#### Environment Variables
- `DB_DIR`: Location for the SQLite database (default: ./data/db)
- `LOG_DIR`: Location for application logs (default: ./data/logs)

#### Using Docker Compose (Recommended)
1. Build and start all services:
```bash
make build
make up
```

2. Initialize the database:
```bash
make init-db
```

3. View logs:
```bash
make logs
```

4. Stop all services:
```bash
make down
```

#### Using Individual Docker Containers

##### Backend Container
```bash
cd backend
# Build with custom directories
DB_DIR=/custom/path/db LOG_DIR=/custom/path/logs make build
# Run the container
make run
# Seed the database
make seed
```

##### Frontend Container
```bash
cd frontend
make build
make run
```

### Accessing the Application
- Frontend: http://localhost:3000
- Backend API: http://localhost:5000/api

## Development

### Project Structure
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

## Technologies Used

### Backend
- Python 3.11
- Flask (Web Framework)
- SQLAlchemy (ORM)
- SQLite (Database)
- Flask-CORS (Cross-Origin Resource Sharing)
- PyTZ (Timezone handling)

### Frontend
- React 18
- React Router (Navigation)
- Axios (HTTP client)
- React-DatePicker (Date selection)
- React-Multi-Select (Advanced select inputs)
- FontAwesome (Icons)

### Development Tools
- Docker
- Docker Compose
- Make
- Git

### Logging
The application includes comprehensive logging with:
- Multiple log levels (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- Database and file logging
- Configurable log settings via Developer Panel
- Log viewer interface

### Database
SQLite database with support for:
- SQLite with UTC timezone support
- Custom storage location support
- Automatic migrations
- Seed data for testing

## Contributing
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License
This project is licensed under the Apache License, Version 2.0 - see the LICENSE file for details
