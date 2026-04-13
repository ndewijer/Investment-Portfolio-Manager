# Data Structure and Management

For detailed information about data models and their relationships, see [Data Models](MODELS.md).

## Database Management

### Version Control
The application uses Goose for database migrations with:
- Migrations embedded in the Go binary
- Automatic migration on server startup
- Both SQL and Go-coded migrations supported
- Safe handling of fresh and existing databases

### Migration Process

Migrations run **automatically** when the backend starts. There are no manual migration commands needed for normal operation.

For development, migrations are stored in `backend/internal/database/migrations/` and are embedded into the binary at compile time.

**Creating a new migration:**
```bash
# SQL migration
touch backend/internal/database/migrations/NNN_description.sql
```

Edit the file with Goose directives:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS new_table (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS new_table;
```

Go-coded migrations are also supported for operations that require application logic (e.g., data backfills). These are registered via `registerGoMigrationVersion` in the `database` package.

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
- Automatic daily price updates via Yahoo Finance (weekdays at 00:55 UTC)
- Protected endpoints for automated tasks
- CSV import/export functionality
- Data validation and sanitization
- Error logging and tracking
- Time-accurate logging with proper timestamps

## Database Initialization
- Fresh databases automatically initialize at current version on first server start
- Existing databases safely upgrade through Goose migrations on startup
- Version control prevents duplicate schema changes
- Error handling for common database operations

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
