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
# Inherits DOMAIN and USE_HTTPS
```

## Example .env file
```bash
DOMAIN=ipm.local
USE_HTTPS=false
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
```yaml
volumes:
  - /path/to/your/data:/data
```

## Network Configuration
- Private network for container communication
- Frontend exposed on port 80
- Backend accessible only through frontend proxy

## Development vs Production
- Development: Webpack dev server & Flask
- Production: Nginx & Gunicorn

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
