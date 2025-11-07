# Versioning and Feature Flag System

## Overview

The Investment Portfolio Manager uses a **version-based feature flag system** to gracefully handle database schema changes and enable new features based on the deployed database version. This allows:

- Safe rollout of new features behind schema requirements
- Graceful degradation when database is behind app version
- Clear migration prompts for users
- Prevention of breaking changes when schema is outdated

## Architecture Components

### 1. Version Files

**Backend Version**: `backend/VERSION`
```
1.3.1
```
- Single source of truth for application version
- Updated when releasing new features
- Read by `get_app_version()` in `system_routes.py`

**Frontend Version**: `frontend/package.json`
```json
{
  "name": "investment-portfolio-manager",
  "version": "1.3.1",
  ...
}
```
- Must be kept in sync with backend VERSION
- Updated whenever backend version changes
- **Important**: While backend and frontend are in the same repository, versions must match

**Database Version**: Stored in `alembic_version` table
```sql
SELECT version_num FROM alembic_version
```
- Managed by Alembic migrations
- Updated automatically when running `flask db upgrade`
- Read by `get_db_version()` in `system_routes.py`

### 2. Feature Flag System

**Location**: `backend/app/routes/system_routes.py`

**Function**: `check_feature_availability(db_version)`

Returns a dictionary of features based on database schema version:

```python
def check_feature_availability(db_version):
    """
    Check which features are available based on database version.

    Returns:
        dict: Feature availability flags
    """
    features = {
        "ibkr_integration": False,
        "realized_gain_loss": False,
        "exclude_from_overview": False,
    }

    # Parse version
    version_parts = db_version.split(".")
    major = int(version_parts[0])
    minor = int(version_parts[1])
    patch = int(version_parts[2]) if len(version_parts) >= 3 else 0

    # Version 1.1.0+: exclude_from_overview
    if major >= 1 and minor >= 1:
        features["exclude_from_overview"] = True

    # Version 1.1.1+: realized_gain_loss
    if major >= 1 and minor >= 1:
        if minor > 1 or (minor == 1 and patch >= 1):
            features["realized_gain_loss"] = True

    # Version 1.3.0+: IBKR integration
    if major >= 1 and minor >= 3:
        features["ibkr_integration"] = True

    return features
```

### 3. Frontend Context

**Location**: `frontend/src/context/AppContext.js`

The `AppProvider` fetches version info on app load:

```javascript
const fetchVersionInfo = async () => {
  const response = await api.get('system/version');
  setVersionInfo(response.data);
};

// Response structure:
{
  "app_version": "1.3.0",
  "db_version": "1.3.0",
  "features": {
    "ibkr_integration": true,
    "realized_gain_loss": true,
    "exclude_from_overview": true
  },
  "migration_needed": false,
  "migration_message": null
}
```

All components can access features via:
```javascript
const { features } = useApp();

if (features.ibkr_integration) {
  // Show IBKR-specific UI
}
```

### 4. API Endpoint

**Endpoint**: `GET /api/system/version`

**Response**:
```json
{
  "app_version": "1.3.0",
  "db_version": "1.3.0",
  "features": {
    "ibkr_integration": true,
    "realized_gain_loss": true,
    "exclude_from_overview": true
  },
  "migration_needed": false,
  "migration_message": null
}
```

**When migration needed** (app_version ≠ db_version):
```json
{
  "app_version": "1.3.1",
  "db_version": "1.3.0",
  "features": {
    "ibkr_integration": true,
    "realized_gain_loss": true,
    "exclude_from_overview": true
  },
  "migration_needed": true,
  "migration_message": "Database schema (v1.3.0) is behind application version (v1.3.1). Run 'flask db upgrade' to enable new features."
}
```

## Adding a New Feature with Schema Change

### Step-by-Step Process

#### 1. Decide Version Number

Use semantic versioning:
- **Major** (x.0.0): Breaking changes
- **Minor** (1.x.0): New features, schema changes
- **Patch** (1.3.x): Bug fixes, no schema change

For schema changes, bump **minor** or **patch** depending on scope.

**Example**: Adding IBKR enable/disable toggle
- Current: 1.3.0
- New: 1.3.1 (minor schema change)

#### 2. Create Database Migration

```bash
cd backend
source .venv/bin/activate
flask db revision -m "add_ibkr_enabled_field"
```

Edit the generated migration file:
```python
"""
Add enabled field to ibkr_config table.

Revision ID: 1.3.1
Revises: 1.3.0
Create Date: 2025-11-06
"""

import sqlalchemy as sa
from alembic import op

revision = "1.3.1"
down_revision = "1.3.0"
branch_labels = None
depends_on = None


def upgrade():
    """Add enabled column to ibkr_config table."""
    conn = op.get_bind()
    inspector = sa.inspect(conn)
    columns = [col["name"] for col in inspector.get_columns("ibkr_config")]

    if "enabled" not in columns:
        op.add_column(
            "ibkr_config",
            sa.Column("enabled", sa.Boolean(), nullable=True),
        )

        # Set default value: enabled=True for existing configs
        op.execute(
            """
            UPDATE ibkr_config
            SET enabled = TRUE
            WHERE enabled IS NULL
            """
        )

        # Make column non-nullable after setting defaults
        op.alter_column("ibkr_config", "enabled", nullable=False)


def downgrade():
    """Remove enabled column from ibkr_config table."""
    op.drop_column("ibkr_config", "enabled")
```

**Key Points**:
- Use version number as revision ID
- Check if column exists before adding (idempotent)
- Set sensible defaults for existing data
- Include both upgrade and downgrade

#### 3. Update Model

```python
# backend/app/models.py

class IBKRConfig(db.Model):
    """IBKR Flex Web Service configuration."""

    __tablename__ = "ibkr_config"

    id = db.Column(db.String(36), primary_key=True, default=generate_uuid)
    flex_token = db.Column(db.String(500), nullable=False)
    flex_query_id = db.Column(db.String(100), nullable=False)
    token_expires_at = db.Column(db.DateTime, nullable=True)
    last_import_date = db.Column(db.DateTime, nullable=True)
    auto_import_enabled = db.Column(db.Boolean, default=False, nullable=False)
    enabled = db.Column(db.Boolean, default=True, nullable=False)  # NEW
    created_at = db.Column(db.DateTime, default=db.func.current_timestamp())
    updated_at = db.Column(
        db.DateTime,
        default=db.func.current_timestamp(),
        onupdate=db.func.current_timestamp(),
    )
```

#### 4. Update Feature Flags (Optional)

If the new feature needs a separate flag (not always necessary):

```python
# backend/app/routes/system_routes.py

def check_feature_availability(db_version):
    features = {
        "ibkr_integration": False,
        "ibkr_enable_toggle": False,  # NEW
        "realized_gain_loss": False,
        "exclude_from_overview": False,
    }

    # ... version parsing ...

    # Version 1.3.0+: IBKR integration
    if major >= 1 and minor >= 3:
        features["ibkr_integration"] = True

        # Version 1.3.1+: IBKR enable/disable toggle
        if minor > 3 or (minor == 3 and patch >= 1):
            features["ibkr_enable_toggle"] = True

    return features
```

**Note**: For simple column additions, you might not need a new feature flag. Just check if `ibkr_integration` is enabled.

#### 5. Update VERSION Files

**Backend Version**:
```bash
echo "1.3.1" > backend/VERSION
```

**Frontend Version** (package.json):
```bash
# Edit frontend/package.json
# Change "version": "1.3.0" to "version": "1.3.1"
```

**Important**: Both backend and frontend versions **must be updated together** while they remain in the same repository. This ensures:
- Clear version tracking across the entire application
- Consistent release versioning
- Easier debugging and support (version mismatch indicates incomplete update)

#### 6. Update Frontend to Use Feature

```javascript
// frontend/src/pages/Config.js

const { features } = useApp();

// Check if feature is available
{features.ibkr_integration && (
  <div>
    {/* Show IBKR config */}

    {features.ibkr_enable_toggle && (
      <div className="form-group">
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={config.enabled}
            onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
          />
          Enable IBKR Integration
        </label>
      </div>
    )}
  </div>
)}
```

#### 7. Test Migration

```bash
# Backup database first!
cp backend/data/db/portfolio_manager.db backend/data/db/portfolio_manager.db.backup

# Run migration
cd backend
source .venv/bin/activate
flask db upgrade

# Check version
flask shell
>>> from app.routes.system_routes import get_db_version
>>> get_db_version()
'1.3.1'
```

#### 8. Update Documentation

- Update RELEASE_NOTES.md
- Update ARCHITECTURE.md if needed
- Update MODELS.md with new column

## Current Feature Flags

| Feature | Version | Description |
|---------|---------|-------------|
| `exclude_from_overview` | 1.1.0+ | Hide portfolios from overview |
| `realized_gain_loss` | 1.1.1+ | Track realized gains/losses |
| `ibkr_integration` | 1.3.0+ | IBKR Flex Web Service integration |

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.1.0 | 2024-11 | Added exclude_from_overview to portfolios |
| 1.1.1 | 2024-11 | Added realized_gain_loss tracking |
| 1.1.2 | 2024-11 | Performance indexes (no feature flag) |
| 1.3.0 | 2025-11 | IBKR Flex integration |
| 1.3.1 | 2025-11 | IBKR enable/disable toggle, mobile navigation improvements |

## Best Practices

### Version Alignment

**Critical Rule**: While backend and frontend are in the same repository, their versions **must always match**.

**When bumping version**:
1. Update `backend/VERSION`
2. Update `frontend/package.json` version field
3. Create database migration with matching revision ID (if schema changes)
4. Commit all version changes together

**Why this matters**:
- Single repository = single release unit
- Version mismatch indicates incomplete/broken update
- Simplifies deployment and rollback procedures
- Clear correlation between frontend features and backend capabilities
- Easier for users to report issues ("I'm on version 1.3.1")

**Example version update**:
```bash
# Update backend
echo "1.3.1" > backend/VERSION

# Update frontend (edit package.json)
# Change: "version": "1.3.0" → "version": "1.3.1"

# Commit together
git add backend/VERSION frontend/package.json
git commit -m "Bump version to 1.3.1"
```

**If repositories separate in the future**: This requirement may be relaxed, allowing independent versioning. Update this document accordingly.

### When to Add a Feature Flag

✅ **Add feature flag when**:
- Feature requires new database tables
- Feature requires new columns that change app behavior
- Feature is opt-in and not universally available
- Feature can be disabled without breaking existing functionality

❌ **Don't add feature flag when**:
- Bug fixes with no schema change
- Performance improvements with no schema change
- Adding indexes or constraints
- Minor UI tweaks

### Migration Best Practices

1. **Always backup before migrating**
   ```bash
   cp data/db/portfolio_manager.db data/db/portfolio_manager.db.backup
   ```

2. **Make migrations idempotent**
   - Check if column/table exists before adding
   - Use `IF NOT EXISTS` where possible

3. **Set sensible defaults**
   - For existing data, set appropriate default values
   - Consider backwards compatibility

4. **Test both upgrade and downgrade**
   ```bash
   flask db upgrade    # Test upgrade
   flask db downgrade  # Test downgrade
   flask db upgrade    # Upgrade again
   ```

5. **Version numbers match revision IDs**
   - Makes it easy to track: `db_version = "1.3.1"` → revision `"1.3.1"`

### Frontend Feature Checking

```javascript
// GOOD: Check feature flag before rendering
const { features } = useApp();

{features.ibkr_integration && (
  <Link to="/ibkr/inbox">IBKR Inbox</Link>
)}

// BAD: Hard-code feature availability
<Link to="/ibkr/inbox">IBKR Inbox</Link>  // Will break if schema not migrated
```

### Backend Feature Checking

```python
# GOOD: Check if table/column exists before querying
from app.models import IBKRConfig

try:
    config = IBKRConfig.query.first()
    if config and hasattr(config, 'enabled'):
        is_enabled = config.enabled
except Exception:
    is_enabled = False

# BETTER: Use feature flags
features = check_feature_availability(get_db_version())
if features['ibkr_integration']:
    config = IBKRConfig.query.first()
```

## Troubleshooting

### "Migration needed" message shows

**Cause**: App version ahead of database version

**Solution**:
```bash
docker compose exec backend flask db upgrade
docker compose restart
```

### Feature not showing after migration

**Cause**: Frontend cached old version info

**Solution**: Refresh browser or check `AppContext` is re-fetching

### Migration fails

**Cause**: Schema change conflicts with existing data

**Solution**:
1. Check migration file logic
2. Backup and restore if needed
3. Fix migration script and try again

## References

- [Alembic Documentation](https://alembic.sqlalchemy.org/)
- [Semantic Versioning](https://semver.org/)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [DEVELOPMENT.md](DEVELOPMENT.md)
