# Docker Configuration

## Container Setup
The application uses a multi-container setup with:
- Frontend (Nginx)
- Backend (Gunicorn)
- Shared volume for data persistence

## Environment Variables
```bash
# Common
DOMAIN=your-domain.com
USE_HTTPS=true/false  # Controls protocol for API calls

# Backend
DB_DIR=/data/db
LOG_DIR=/data/logs
INTERNAL_API_KEY=your-secure-key-here  # Optional; auto-generated if not provided

# Frontend
BACKEND_HOST=investment-portfolio-backend  # Backend container hostname (default: investment-portfolio-backend)
# Inherits DOMAIN and USE_HTTPS
```

## Example .env file
```bash
DOMAIN=ipm.local
USE_HTTPS=false
BACKEND_HOST=investment-portfolio-backend  # Optional: override for custom container names
INTERNAL_API_KEY=your-secure-random-key-here
```

## HTTPS Configuration
The USE_HTTPS flag controls:
- API call protocols
- Frontend-to-backend communication
- Whether to expect SSL termination at reverse proxy

### With External Reverse Proxy (USE_HTTPS=true)
- Assumes SSL termination at reverse proxy
- Frontend makes HTTPS API calls
- Requires valid SSL certificate at proxy

### Direct HTTP Access (USE_HTTPS=false)
- No SSL termination required
- Frontend makes HTTP API calls
- Suitable for local development

## Volume Management

### Default Configuration (Named Volume)

By default, the application uses a Docker-managed named volume called `portfolio-data`:

```yaml
volumes:
  - portfolio-data:/data
```

**Benefits:**
- No host path configuration needed
- Works automatically in all environments (development, CI/CD, production)
- **Data persists between container restarts, rebuilds, and updates**
- Docker manages storage location
- Survives `docker compose down` (without `-v` flag)

### ⚠️ Critical: Data Persistence and Loss Prevention

**Your SQLite database lives in this volume. Protect it!**

#### Safe Commands (Data Preserved):
```bash
# Stop containers - VOLUME REMAINS
docker compose down

# Restart containers - SAME DATA
docker compose up -d

# Update application - DATA PERSISTS
docker compose pull
docker compose up -d --force-recreate

# View logs without affecting data
docker compose logs -f
```

#### ⛔ DANGEROUS Commands (Data Loss):
```bash
# THE -v FLAG DELETES YOUR DATABASE!
docker compose down -v           # ⛔ Deletes volume and all data
docker volume rm portfolio-data  # ⛔ Explicit volume deletion
docker volume prune              # ⛔ May delete your volume
```

**Golden Rule:** Never use `-v` flag with `docker compose down` in production unless you want to delete your database!

### Data Backup and Recovery

**⚠️ Important:** SQLite databases require special backup procedures. File-based copies (tar/cp) can be corrupted if the database is active. Always use SQLite's built-in backup methods for live databases.

#### Method 1: Online Backup API (RECOMMENDED)

Uses SQLite's `.backup` command for safe backup of live databases:

```bash
#!/bin/bash
# backup-portfolio.sh
BACKUP_DIR="/path/to/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/portfolio-backup-$DATE.db"

# Backup using SQLite's Online Backup API
docker compose exec backend sqlite3 /data/db/portfolio_manager.db ".backup '/data/db/backup-temp.db'"
docker compose cp backend:/data/db/backup-temp.db "$BACKUP_FILE"
docker compose exec backend rm /data/db/backup-temp.db

# Compress the backup
gzip "$BACKUP_FILE"
echo "Backup created: portfolio-backup-$DATE.db.gz"

# Keep only last 30 days
find "$BACKUP_DIR" -name "portfolio-backup-*.db.gz" -mtime +30 -delete
```

**Advantages:**
- ✅ Safe for live databases
- ✅ Allows concurrent database access
- ✅ Fast and efficient
- ✅ Creates exact binary copy

**Restore from Online Backup:**
```bash
# Stop the application
docker compose down

# Decompress and restore backup
gunzip -c "$BACKUP_DIR/portfolio-backup-YYYYMMDD_HHMMSS.db.gz" > portfolio_manager.db
docker compose cp ./portfolio_manager.db backend:/data/db/portfolio_manager.db
rm portfolio_manager.db

# Start the application
docker compose up -d
```

#### Method 2: VACUUM INTO (ALTERNATIVE)

Creates a vacuumed copy of the database. **Requires SQLite 3.27.0 or higher.**

```bash
#!/bin/bash
# backup-portfolio-vacuum.sh
BACKUP_DIR="/path/to/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup using VACUUM INTO (requires SQLite 3.27+)
docker compose exec backend sqlite3 /data/db/portfolio_manager.db \
  "VACUUM INTO '/data/db/backup-temp.db'"
docker compose cp backend:/data/db/backup-temp.db "$BACKUP_DIR/portfolio-backup-$DATE.db"
docker compose exec backend rm /data/db/backup-temp.db

gzip "$BACKUP_DIR/portfolio-backup-$DATE.db"
echo "Vacuumed backup created: portfolio-backup-$DATE.db.gz"
```

**Advantages:**
- ✅ Safe for live databases
- ✅ Creates optimized/vacuumed copy
- ✅ Removes fragmentation

**Restore from Backup:**

Same restore procedure as Method 1 above.

**Inspect Volume Location:**
```bash
# Find where Docker stores the volume
docker volume inspect investment-portfolio-manager_portfolio-data

# Output shows Mountpoint (actual filesystem location):
# "Mountpoint": "/var/lib/docker/volumes/investment-portfolio-manager_portfolio-data/_data"
```

### Testing vs Production: Why the Difference?

**You may see `-v` used in test commands. This is intentional!**

```bash
# IN TESTS (CI/CD, pre-commit hooks):
docker compose down -v  # ✅ Correct - clean up test data

# IN PRODUCTION:
docker compose down     # ✅ Correct - preserve your database
```

**Why?**
- **Tests** create temporary data that should be cleaned up
- **Production** has real user data that must be preserved
- Test commands ensure clean state for next test run
- Production commands ensure data survives container updates

**Warning:** Don't copy test commands for production use without removing `-v` flag!

### Custom Host Path (Optional)

If you need data on a specific host path (e.g., for NAS storage), create a `docker-compose.override.yml`:

```yaml
# docker-compose.override.yml
services:
  backend:
    volumes:
      - /path/to/your/data:/data  # Override with your path
```

This file is automatically merged with `docker-compose.yml` and is gitignored.

**Example for Unraid:**
```yaml
services:
  backend:
    volumes:
      - /mnt/user/appdata/investment-portfolio:/data
```

## Network Configuration
- Private network for container communication
- Frontend exposed on port 80
- Backend accessible only through frontend proxy

## Development vs Production
- Development: Webpack dev server & Flask
- Production: Nginx & Gunicorn

## Custom Container Names
If you use custom container names (e.g., on Unraid or in custom docker-compose files), set the `BACKEND_HOST` environment variable:

```yaml
services:
  backend:
    container_name: myapp-backend  # Custom name
    # ... rest of config

  frontend:
    container_name: myapp-frontend
    environment:
      - BACKEND_HOST=myapp-backend  # Must match backend container name
    # ... rest of config
```

Or using environment variables:
```bash
# .env file
BACKEND_HOST=myapp-backend
```

The frontend will use this hostname to proxy `/api/` requests to the backend.

## API Key Management
The INTERNAL_API_KEY can be handled in two ways:

1. Manually specified:
   ```bash
   INTERNAL_API_KEY=your-secure-key-here
   ```

2. Auto-generated:
   - If not provided, a secure random key is generated at container start
   - The generated key is logged at startup for reference
   - The key persists for the container's lifetime
   - A new key is generated if the container is recreated
