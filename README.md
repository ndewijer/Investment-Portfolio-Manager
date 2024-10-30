# Investment Portfolio Manager

A web application for managing investment portfolios, tracking fund performance, and analyzing investment returns.

## Features

- Multiple portfolio management
- Fund tracking with historical prices
- Transaction management (buy/sell)
- Performance analysis and visualization
- Currency exchange rate support
- CSV import functionality for transactions and fund prices
- European/US number format support

## Project Structure
Investment-Portfolio-Manager/
├── backend/ # Flask backend
│ ├── app/ # Application code
│ │ ├── models/ # Database models
│ │ ├── routes/ # API endpoints
│ │ └── services/ # Business logic
│ ├── venv/ # Python virtual environment
│ └── run.py # Application entry point
└── frontend/ # React frontend
├── public/ # Static files
├── src/ # Source code
└── package.json # Node.js dependencies


## Setup

### Backend Setup

1. Create and activate a Python virtual environment:
```bash
cd backend
python -m venv venv
source venv/bin/activate # On Windows: venv\Scripts\activate
```

2. Install dependencies:
```bash
pip install -r requirements.txt
```

3. Initialize the database:
```bash
flask seed-db
```

4. Start the Flask server:
```bash
flask run
```


### Frontend Setup

1. Install Node.js dependencies:
```bash
cd frontend
npm install
```

2. Start the React development server:
```bash
npm start
```


## Development

The application will be available at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:5000

## Technologies Used

### Backend
- Flask (Web Framework)
- SQLAlchemy (ORM)
- SQLite (Database)
- Flask-CORS (Cross-Origin Resource Sharing)
- Python-dotenv (Environment Variables)

### Frontend
- React
- React Router (Routing)
- Axios (HTTP Client)
- Recharts (Charts)
- FontAwesome (Icons)
- React-DatePicker (Date Selection)
- React-Multi-Select-Component (Multi-select Dropdowns)

## Features in Detail

### Portfolio Management
- Create, edit, and delete portfolios
- Archive/unarchive portfolios
- Track multiple portfolios simultaneously
- View portfolio performance metrics

### Fund Management
- Add and manage investment funds
- Track fund prices over time
- Import historical fund prices via CSV
- Support for different currencies

### Transaction Management
- Record buy/sell transactions
- Import transactions via CSV
- Edit and delete transactions
- Filter and sort transaction history

### Analysis Tools
- Portfolio value tracking over time
- Performance comparison between portfolios
- Gain/loss calculations
- Currency conversion support

### Developer Tools
- Set exchange rates
- Import historical data
- Manage fund prices
- Configure number format preferences

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request
