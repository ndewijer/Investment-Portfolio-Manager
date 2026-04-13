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
2.0.0
```
- Single source of truth for application version
- Updated when releasing new features
- Injected at build time via `-ldflags` in Go (see `version` package)

**Frontend Version**: `frontend/package.json`
```json
{
  "name": "investment-portfolio-manager",
  "version": "2.0.0",
  ...
}
```
- Must be kept in sync with backend VERSION
- Updated whenever backend version changes
- **Important**: While backend and frontend are in the same repository, versions must match

**Database Version**: Managed by Goose migrations
- Goose tracks applied migrations in the `goose_db_version` table
- Migrations run automatically on server startup
- The application maps Goose migration state to a semantic version for feature flag evaluation

### 2. Feature Flag System

**Location**: `backend/internal/service/system_service.go`

Returns a dictionary of features based on database schema version:

```go
func (s *SystemService) CheckFeatureAvailability(dbVersion string) map[string]bool {
    features := map[string]bool{
        "ibkr_integration":     false,
        "realized_gain_loss":   false,
        "exclude_from_overview": false,
    }

    // Parse version
    parts := strings.Split(dbVersion, ".")
    major, _ := strconv.Atoi(parts[0])
    minor, _ := strconv.Atoi(parts[1])
    patch := 0
    if len(parts) >= 3 {
        patch, _ = strconv.Atoi(parts[2])
    }

    // Version 1.1.0+: exclude_from_overview
    if major >= 1 && minor >= 1 {
        features["exclude_from_overview"] = true
    }

    // Version 1.1.1+: realized_gain_loss
    if major >= 1 && minor >= 1 {
        if minor > 1 || (minor == 1 && patch >= 1) {
            features["realized_gain_loss"] = true
        }
    }

    // Version 1.3.0+: IBKR integration
    if major >= 1 && minor >= 3 {
        features["ibkr_integration"] = true
    }

    return features
}
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
  "app_version": "2.0.0",
  "db_version": "2.0.0",
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
  "app_version": "2.0.0",
  "db_version": "2.0.0",
  "features": {
    "ibkr_integration": true,
    "realized_gain_loss": true,
    "exclude_from_overview": true
  },
  "migration_needed": false,
  "migration_message": null
}
```

**When migration needed** (app_version != db_version):
```json
{
  "app_version": "2.1.0",
  "db_version": "2.0.0",
  "features": {
    "ibkr_integration": true,
    "realized_gain_loss": true,
    "exclude_from_overview": true
  },
  "migration_needed": true,
  "migration_message": "Database schema (v2.0.0) is behind application version (v2.1.0). Restart the server to apply pending migrations."
}
```

**Note**: In the Go backend, migrations run automatically on startup. If a migration is pending, restarting the server will apply it. There is no manual `flask db upgrade` equivalent.

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

Create a new SQL migration file in `backend/internal/database/migrations/`:

```sql
-- +goose Up
-- Add enabled column to ibkr_config table.

ALTER TABLE ibkr_config ADD COLUMN enabled BOOLEAN DEFAULT TRUE;

UPDATE ibkr_config SET enabled = TRUE WHERE enabled IS NULL;

-- +goose Down
-- SQLite does not support DROP COLUMN directly.
-- For downgrade, recreate the table without the column if needed.
```

**Key Points**:
- Use Goose `-- +goose Up` / `-- +goose Down` directives
- Migrations run automatically on next server start
- Set sensible defaults for existing data
- Include both upgrade and downgrade sections

For complex migrations that require application logic, use Go-coded migrations registered via `registerGoMigrationVersion`.

#### 3. Update Model

```go
// backend/internal/model/ibkr_config.go

type IBKRConfig struct {
    ID                string     `json:"id"`
    FlexToken         string     `json:"-"`
    FlexQueryID       string     `json:"flexQueryId"`
    TokenExpiresAt    *time.Time `json:"tokenExpiresAt,omitempty"`
    LastImportDate    *time.Time `json:"lastImportDate,omitempty"`
    AutoImportEnabled bool       `json:"autoImportEnabled"`
    Enabled           bool       `json:"enabled"`  // NEW
    CreatedAt         time.Time  `json:"createdAt"`
    UpdatedAt         time.Time  `json:"updatedAt"`
}
```

#### 4. Update Feature Flags (Optional)

If the new feature needs a separate flag (not always necessary):

```go
// backend/internal/service/system_service.go

func (s *SystemService) CheckFeatureAvailability(dbVersion string) map[string]bool {
    features := map[string]bool{
        "ibkr_integration":    false,
        "ibkr_enable_toggle":  false,  // NEW
        "realized_gain_loss":  false,
        "exclude_from_overview": false,
    }

    // ... version parsing ...

    // Version 1.3.0+: IBKR integration
    if major >= 1 && minor >= 3 {
        features["ibkr_integration"] = true

        // Version 1.3.1+: IBKR enable/disable toggle
        if minor > 3 || (minor == 3 && patch >= 1) {
            features["ibkr_enable_toggle"] = true
        }
    }

    return features
}
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
cp data/portfolio_manager.db data/portfolio_manager.db.backup

# Restart server — migrations apply automatically
cd backend
go run ./cmd/server/main.go

# Verify via API
curl http://localhost:5000/api/system/version
```

#### 8. Update Documentation

- Update RELEASE_NOTES.md
- Update ARCHITECTURE.md if needed
- Update MODELS.md with new column

## Schema Change Checklist

Use this checklist when adding a new feature with schema changes to ensure nothing is missed:

### Pre-Implementation
- [ ] Decide on version number (major.minor.patch)
- [ ] Determine if feature flag is needed

### Database Changes
- [ ] Create Goose migration file in `backend/internal/database/migrations/`
- [ ] Make migration idempotent where possible
- [ ] Set appropriate defaults for existing data
- [ ] Test both upgrade and downgrade
- [ ] Update model structs in `backend/internal/model/`

### Version Updates (CRITICAL - Often Missed!)
- [ ] Update `backend/VERSION` file to new version
- [ ] Update `frontend/package.json` version field to match
- [ ] Verify both versions match exactly

### Feature Flag (if needed)
- [ ] Add feature flag to `SystemService.CheckFeatureAvailability()`
- [ ] Add version check logic
- [ ] Test feature flag returns correct value

### Documentation
- [ ] Add feature to "Current Feature Flags" table in VERSIONING_AND_FEATURES.md
- [ ] Add version to "Version History" table with date and changes
- [ ] Update ARCHITECTURE.md if architectural changes
- [ ] Update TESTING.md if new tests added
- [ ] Create/update relevant documentation files

### Testing
- [ ] Write comprehensive tests for new functionality
- [ ] Test migration applies correctly on server restart
- [ ] Verify feature flag works correctly
- [ ] Test with both old and new schema versions

### Release
- [ ] Create RELEASE_NOTES entry
- [ ] Commit all changes together
- [ ] Tag release in git if applicable

**Why This Checklist Exists**: Previous implementations have missed version updates and feature flags, causing confusion about schema compatibility. This checklist ensures completeness.

## Current Feature Flags

| Feature | Version | Description |
|---------|---------|-------------|
| `exclude_from_overview` | 1.1.0+ | Hide portfolios from overview |
| `realized_gain_loss` | 1.1.1+ | Track realized gains/losses |
| `ibkr_integration` | 1.3.0+ | IBKR Flex Web Service integration |
| `materialized_view_performance` | 1.4.0+ | Portfolio history materialized view (160x performance improvement) |

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.1.0 | 2024-11 | Added exclude_from_overview to portfolios |
| 1.1.1 | 2024-11 | Added realized_gain_loss tracking |
| 1.1.2 | 2024-11 | Performance indexes (no feature flag) |
| 1.3.0 | 2024-11 | IBKR Flex integration |
| 1.3.1 | 2024-11-07 | IBKR enable/disable toggle, bulk transaction processing, allocation presets, mobile UI fixes, chart improvements |
| 1.4.0 | 2026-01-11 | Portfolio history materialized view for 160x performance improvement, CLI management tools |
| 2.0.0 | 2026-03-27 | Backend rewritten from Python/Flask to Go/Chi. Same API contract, same database schema. Goose replaces Alembic for migrations (run automatically on startup). ApexCharts replaces Recharts for charting. |

## Best Practices

### Version Alignment

**Critical Rule**: While backend and frontend are in the same repository, their versions **must always match**.

**When bumping version**:
1. Update `backend/VERSION`
2. Update `frontend/package.json` version field
3. Create database migration if schema changes (in `backend/internal/database/migrations/`)
4. Commit all version changes together

**Why this matters**:
- Single repository = single release unit
- Version mismatch indicates incomplete/broken update
- Simplifies deployment and rollback procedures
- Clear correlation between frontend features and backend capabilities
- Easier for users to report issues ("I'm on version 2.0.0")

**Example version update**:
```bash
# Update backend
echo "2.1.0" > backend/VERSION

# Update frontend (edit package.json)
# Change: "version": "2.0.0" → "version": "2.1.0"

# Commit together
git add backend/VERSION frontend/package.json
git commit -m "Bump version to 2.1.0"
```

**If repositories separate in the future**: This requirement may be relaxed, allowing independent versioning. Update this document accordingly.

### When to Add a Feature Flag

Add feature flag when:
- Feature requires new database tables
- Feature requires new columns that change app behavior
- Feature is opt-in and not universally available
- Feature can be disabled without breaking existing functionality

Don't add feature flag when:
- Bug fixes with no schema change
- Performance improvements with no schema change
- Adding indexes or constraints
- Minor UI tweaks

### Migration Best Practices

1. **Always backup before upgrading**
   ```bash
   cp data/portfolio_manager.db data/portfolio_manager.db.backup
   ```

2. **Migrations run automatically on startup**
   - No manual commands needed
   - Just restart the server

3. **Set sensible defaults**
   - For existing data, set appropriate default values
   - Consider backwards compatibility

4. **Test migrations thoroughly**
   - Create a test database with realistic data
   - Start the server and verify migrations apply cleanly
   - Check the version endpoint: `curl http://localhost:5000/api/system/version`

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

```go
// Use feature flags from the system service
features := systemService.CheckFeatureAvailability(dbVersion)
if features["ibkr_integration"] {
    // Enable IBKR-related functionality
}
```

## Troubleshooting

### "Migration needed" message shows

**Cause**: App version ahead of database version

**Solution**:
```bash
# Restart the backend — migrations apply automatically
docker compose restart backend
```

### Feature not showing after migration

**Cause**: Frontend cached old version info

**Solution**: Refresh browser or check `AppContext` is re-fetching

### Migration fails on startup

**Cause**: Schema change conflicts with existing data

**Solution**:
1. Check backend logs: `docker compose logs backend`
2. Backup and restore if needed
3. Fix migration file and rebuild

## References

- [Goose Documentation](https://pressly.github.io/goose/)
- [Semantic Versioning](https://semver.org/)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [DEVELOPMENT.md](DEVELOPMENT.md)

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
