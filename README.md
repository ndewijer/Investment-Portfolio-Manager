# Investment Portfolio Manager

A web application for managing investment portfolios, tracking transactions, and monitoring dividends.

This application was developed as an exploration of Large Language Model (LLM) assisted coding, specifically using Anthropic's Claude. The project aimed to solve a personal challenge of managing multiple Excel spreadsheets tracking various investment fund portfolios, their transactions, and dividend payments. Through iterative development with AI assistance, it evolved into a functional web application that demonstrates both the capabilities and limitations of LLM-guided development.

The app provides core portfolio management features, including transaction tracking, dividend processing (cash and stock) and basic performance visualization. It caters to European funds, supporting currency conversion and formatting.

While functional for personal use, replacing my manual Excel tracking, it is not intended to compete with professional trading or portfolio management platforms as it lacks advanced features found in professional trading platforms, such as user authentication, real-time pricing, or complex trading tools.

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

## Data Structure

### Historical Price Data
The application includes historical price data for two Goldman Sachs funds:
- Goldman Sachs Enhanced Index Sustainable Equity (NL0012125736)
- Goldman Sachs Enhanced Index Sustainable EM Equity (NL0006311771)

Note: The included CSV files contain historical data up to November 2024. For development and testing purposes, this data is sufficient. In production, rely on the actual historical data from the fund provider.

### CSV Format
Price history files follow this format:
```csv
date,price
2024-10-30,40.17
2024-10-29,40.01
...
```

### Development Data
The application includes:
- Sample portfolios
- Historical price data (CSV)
- Generated transactions
- Simulated dividend payments

When running `flask seed-db`, the application will:
1. Load historical prices from CSV files
2. Generate realistic transactions
3. Create yearly dividend records
4. Set up sample portfolios

## Project Structure
```
Investment-Portfolio-Manager/
├── backend/
│   ├── app/
│   │   ├── models/
│   │   ├── routes/
│   │   └── services/
│   ├── data/
│   │   ├── seed/
│   │   │   └── funds/
│   │   │   │   └── prices/
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
