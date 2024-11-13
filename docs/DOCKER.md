# Docker Configuration

## Container Setup
The application uses a multi-container setup with:
- Frontend (Nginx)
- Backend (Gunicorn)
- Shared volume for data persistence

## Environment Variables
```bash
# Backend
DB_DIR=/data/db
LOG_DIR=/data/logs
DOMAIN=your-domain.com

# Frontend
DOMAIN=your-domain.com
```

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
