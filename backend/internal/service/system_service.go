package service

import (
	"database/sql"
	"fmt"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/database"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/version"
)

var sysLog = logging.NewLogger("system")

// SystemService handles system-related operations
type SystemService struct {
	db *sql.DB
}

// NewSystemService creates a new SystemService
func NewSystemService(db *sql.DB) *SystemService {
	return &SystemService{
		db: db,
	}
}

// CheckHealth checks the health of the system
func (s *SystemService) CheckHealth() error {
	sysLog.Debug("checking system health")
	if err := database.HealthCheck(s.db); err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	return nil
}

// CheckVersion retrieves version information including app version, database version,
// feature availability, and pending migration status.
func (s *SystemService) CheckVersion() (model.VersionInfo, error) {
	sysLog.Debug("checking version info")
	appVersion := version.Version
	dbVersion, err := s.getDbVersion()
	if err != nil {
		sysLog.Warn("failed to get db version", "error", err)
		dbVersion = "unknown"
	}

	features := s.checkFeatureAvailability(dbVersion)
	migrationNeeded, migrationMsg := s.checkPendingMigrations(dbVersion)

	var msgPtr *string
	if migrationMsg != "" {
		msgPtr = &migrationMsg
	}

	return model.VersionInfo{
		AppVersion:       appVersion,
		DbVersion:        dbVersion,
		Features:         features,
		MigrationNeeded:  migrationNeeded,
		MigrationMessage: msgPtr,
	}, nil
}

// getDbVersion retrieves the current database schema version.
// It queries goose_db_version first; for legacy Python-migrated DBs it falls back to alembic_version.
func (s *SystemService) getDbVersion() (string, error) {
	var versionID int64
	err := s.db.QueryRow(
		"SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1",
	).Scan(&versionID)
	if err == nil {
		return fmt.Sprintf("%d", versionID), nil
	}
	// Fallback for legacy Python DBs
	var versionNum string
	err = s.db.QueryRow("SELECT version_num FROM alembic_version").Scan(&versionNum)
	if err != nil {
		return "", fmt.Errorf("get alembic version: %w", err)
	}
	return versionNum, nil
}

// checkFeatureAvailability determines which features are available based on the database version.
func (s *SystemService) checkFeatureAvailability(dbVersion string) map[string]bool {
	features := map[string]bool{
		"basic_portfolio_management":    true, // Introduced 1.1.1
		"realized_gain_loss":            true, // Introduced 1.1.1
		"ibkr_integration":              true, // Introduced 1.3.0
		"materialized_view_performance": true, // Introduced 1.4.0
		"fund_level_materialized_view":  true, // Introduced 1.5.0
		"materialized_sale_proceeds":    true, // Introduced 1.7.0
	}

	_ = dbVersion

	return features
}

// checkPendingMigrations checks whether any migration files exist that have not
// yet been applied to the database. This mirrors the Python Alembic approach:
// compare head migration file version against current DB version, rather than
// comparing app version numbers (which would produce false positives when an
// app version ships without a new migration file).
func (s *SystemService) checkPendingMigrations(dbVersion string) (bool, string) {
	pending, err := database.HasPendingMigrations(s.db)
	if err != nil {
		sysLog.Warn("failed to check pending migrations", "error", err)
		return false, ""
	}
	if pending {
		return true, fmt.Sprintf(
			"Database Migration Needed — Database schema version %s has unapplied migrations; run migrations to upgrade",
			dbVersion,
		)
	}
	return false, ""
}
