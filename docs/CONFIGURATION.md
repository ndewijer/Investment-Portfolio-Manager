# Configuration

Configuration is loaded from environment variables, with `.env` file support via [godotenv](https://github.com/joho/godotenv).

## Environment Variables

### Server

| Variable      | Default     | Description             |
|---------------|-------------|-------------------------|
| `SERVER_PORT` | `5000`      | HTTP server port        |
| `SERVER_HOST` | `0.0.0.0`   | HTTP server bind address|

### Database

| Variable  | Default                          | Description                          |
|-----------|----------------------------------|--------------------------------------|
| `DB_PATH` | `./data/portfolio_manager.db`    | Path to SQLite database file         |
| `DB_DIR`  | *(unset)*                        | Directory for database file (Docker). Takes precedence over `DB_PATH` -- the database will be at `$DB_DIR/portfolio_manager.db` |

### Logging

| Variable  | Default        | Description              |
|-----------|----------------|--------------------------|
| `LOG_DIR` | `./data/logs`  | Directory for log files  |

Logging levels and categories are configurable at runtime via the `/api/developer/system-settings/logging` endpoints.

### CORS

| Variable               | Default                  | Description                                |
|------------------------|--------------------------|--------------------------------------------|
| `CORS_ALLOWED_ORIGINS` | *(unset)*                | Comma-separated list of allowed origins    |
| `DOMAIN`               | *(unset)*                | Generates `http://` and `https://` origins |

Resolution order: `CORS_ALLOWED_ORIGINS` -> `DOMAIN` -> `http://localhost:3000`.

### Security

| Variable              | Default     | Description                                              |
|-----------------------|-------------|----------------------------------------------------------|
| `IBKR_ENCRYPTION_KEY` | *(unset)*   | Fernet key for IBKR credential encryption. Auto-generated and saved to `data/.ibkr_encryption_key` if not provided |
| `INTERNAL_API_KEY`    | *(unset)*   | API key for protected endpoints (e.g., scheduled price updates) |

## .env File

Copy `.env.example` and edit:

```bash
cp .env.example .env
```

The `.env` file is loaded automatically on startup. It is ignored if missing (e.g., when using Docker environment variables instead).

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
