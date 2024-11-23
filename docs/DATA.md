# Data Structure and Management

## Historical Price Data
The application includes historical price data for two Goldman Sachs funds:
- Goldman Sachs Enhanced Index Sustainable Equity (NL0012125736)
- Goldman Sachs Enhanced Index Sustainable EM Equity (NL0006311771)

### CSV Format
Price history files follow this format:
```csv
date,price
2024-10-30,40.17
2024-10-29,40.01
```

### Data Organization
```
backend/data/
├── seed/
│   └── funds/
│       └── prices/
│           ├── NL0012125736.csv
│           └── NL0006311771.csv
├── exports/ #TODO
└── imports/ #TODO
```

## Database Schema
- Portfolios
- Funds
- Transactions
- Dividends
- Price History
- System Logs

## Data Management
- Automatic daily price updates via Yahoo Finance (weekdays at 23:55)
- Protected endpoints for automated tasks
- CSV import/export functionality
- Data validation and sanitization
- Error logging and tracking
- Time-accurate logging with proper timestamps
