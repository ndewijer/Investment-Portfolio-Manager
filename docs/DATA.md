# Data Structure and Management

For detailed information about data models and their relationships, see [Data Models](MODELS.md).

## Database Management

### Version Control
The application uses Alembic for database migrations with:
- Central VERSION file tracking application version
- Automatic version table management
- Safe handling of fresh and existing databases

### Migration Process
```bash
# Create new migration
flask db revision -m "description"

# Apply migrations
flask db upgrade

# Rollback migrations
flask db downgrade
```

### Database Schema
Core tables:
- Portfolios: Portfolio management
- Funds: Fund information and metadata
- Transactions: Buy/sell records
- Dividends: Dividend payments and reinvestment
- Price History: Historical price data
- System Logs: Application logging

Performance optimized indexes:
- Transaction dates and portfolio relationships
- Fund price history lookups
- Dividend record tracking
- Exchange rate history
- Realized gain/loss calculations

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

## Data Management
- Automatic daily price updates via Yahoo Finance (weekdays at 23:55)
- Protected endpoints for automated tasks
- CSV import/export functionality
- Data validation and sanitization
- Error logging and tracking
- Time-accurate logging with proper timestamps

## Database Initialization
- Fresh databases automatically initialize at current version
- Existing databases safely upgrade through migrations
- Version control prevents duplicate schema changes
- Error handling for common database operations
