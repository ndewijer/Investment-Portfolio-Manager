# Investment Portfolio Manager

The application provides core functionality for managing investment fund portfolios, including transaction tracking, dividend management, and historical price tracking. It supports both cash and stock dividend processing, CSV data imports, and basic portfolio performance visualization. The system is particularly suited for European investment funds, with built-in support for currency conversion and European number formatting.

![IPM Screenshot](docs/IPM.png)

This application was developed as an exploration of Large Language Model (LLM) assisted coding, specifically using Anthropic's Claude. The project aimed to solve a personal challenge of managing multiple Excel spreadsheets tracking various investment fund portfolios, their transactions, and dividend payments.

While functional for personal use, replacing my manual Excel tracking, it is not intended to compete with professional trading or portfolio management platforms as it lacks advanced features found in professional trading platforms, such as user authentication, real-time pricing, or complex trading tools.

## Quick Start

### Docker (Recommended)
```bash
git clone https://github.com/ndewijer/investment-portfolio-manager.git
cd investment-portfolio-manager
docker compose up -d
```
Access at http://localhost

No configuration needed! On first startup:
- Database migrations run automatically
- Sample data is seeded automatically
- Frontend uses relative URLs that nginx proxies to the backend

### Local Development
**Prerequisites**: Python 3.13+, Node.js 23+, uv

```bash
# Install uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# Backend
uv sync --frozen
cd backend
uv run flask db upgrade
uv run flask seed-db
uv run flask run

# Frontend (new terminal)
cd frontend
npm install
npm start
```

## Documentation
### Architecture
- [Architecture](docs/ARCHITECTURE.md)
- [Data Models](docs/MODELS.md)

### Features
- [IBKR Flex Integration Setup](docs/IBKR_SETUP.md)

### Contributing
- [Contributing](docs/CONTRIBUTING.md)
- [Docker Setup](docs/DOCKER.md)

#### Frontend Development
- [CSS](docs/CSS.md)

#### Backend Development
- [Development Guide](docs/DEVELOPMENT.md)
- [Data Management](docs/DATA.md)






## Features
- Portfolio management with transaction tracking
- Cash and stock dividend processing
- Automated daily fund price updates
- **IBKR Flex integration for automatic transaction imports (v1.3.0+)**
- Fund price history and currency conversion
- CSV import/export functionality
- System logging and monitoring
- Protected API endpoints for automated tasks

## License
Apache License, Version 2.0
