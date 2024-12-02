# Investment Portfolio Manager

The application provides core functionality for managing investment fund portfolios, including transaction tracking, dividend management, and historical price tracking. It supports both cash and stock dividend processing, CSV data imports, and basic portfolio performance visualization. The system is particularly suited for European investment funds, with built-in support for currency conversion and European number formatting.

This application was developed as an exploration of Large Language Model (LLM) assisted coding, specifically using Anthropic's Claude. The project aimed to solve a personal challenge of managing multiple Excel spreadsheets tracking various investment fund portfolios, their transactions, and dividend payments.

While functional for personal use, replacing my manual Excel tracking, it is not intended to compete with professional trading or portfolio management platforms as it lacks advanced features found in professional trading platforms, such as user authentication, real-time pricing, or complex trading tools.

## Quick Start

1. Clone and setup:
```bash
git clone https://github.com/ndewijer/investment-portfolio-manager.git
cd investment-portfolio-manager
# Set DOMAIN, USE_HTTPS, DB_DIR, and LOG_DIR in .env
```

2. Run with Docker:
```bash
docker compose up -d
docker compose exec backend flask seed-db
```

Access at http://localhost (or your configured domain)

## Documentation
- [Architecture](docs/ARCHITECTURE.md)
- [Contributing](docs/CONTRIBUTING.md)
- [Data Management](docs/DATA.md)
- [Data Models](docs/MODELS.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Docker Setup](docs/DOCKER.md)

## Features
- Portfolio management with transaction tracking
- Cash and stock dividend processing
- Automated daily fund price updates
- Fund price history and currency conversion
- CSV import/export functionality
- System logging and monitoring
- Protected API endpoints for automated tasks

## License
Apache License, Version 2.0
