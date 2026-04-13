package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

//go:embed testdata/golden_schema.sql
var goldenSchema string

// Migrate runs all pending migrations.
// On a fresh DB this creates the full schema.
// On an existing DB it applies only new migrations.
func Migrate(db *sql.DB) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// ApplyGoldenSchema creates all tables and indexes by executing the golden schema DDL directly.
// This is much faster than running goose migrations (no goose overhead, no version tracking)
// and is intended for test databases that need a fresh schema quickly.
// It does NOT insert seed data or populate goose_db_version.
func ApplyGoldenSchema(db *sql.DB) error {
	// The golden schema is dumped from sqlite_master ORDER BY name, so indexes
	// may appear before their parent tables. Execute CREATE TABLE/CREATE INDEX
	// in two passes to avoid "no such table" errors.
	var tables, indexes []string
	for stmt := range strings.SplitSeq(goldenSchema, "\n\n") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if strings.HasPrefix(stmt, "CREATE INDEX") {
			indexes = append(indexes, stmt)
		} else if strings.HasPrefix(stmt, "CREATE TABLE sqlite_") {
			// Skip internal SQLite tables (e.g. sqlite_sequence) — created automatically
			continue
		} else {
			tables = append(tables, stmt)
		}
	}
	for _, stmt := range tables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("golden schema exec failed on %q: %w", truncate(stmt, 80), err)
		}
	}
	for _, stmt := range indexes {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("golden schema exec failed on %q: %w", truncate(stmt, 80), err)
		}
	}
	return nil
}

// truncate returns at most n characters of s, appending "…" if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// HasPendingMigrations reports whether any migration files in the embedded FS
// have not yet been applied to the database. It compares the highest migration
// file version against the current DB version — mirroring the Python Alembic
// head_rev vs current_rev check. This avoids false positives when the app
// version advances without a corresponding new migration file.
func HasPendingMigrations(db *sql.DB) (bool, error) {
	var dbVersion int64
	err := db.QueryRow(
		"SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1",
	).Scan(&dbVersion)
	if err != nil {
		// goose_db_version table doesn't exist yet — DB needs initialising
		return true, nil
	}

	headVersion, err := headMigrationVersion()
	if err != nil {
		return false, err
	}

	return dbVersion < headVersion, nil
}

// goMigrationVersions collects versions from Go migrations that self-register
// via registerGoMigrationVersion in their init() functions.
var goMigrationVersions []int64

// registerGoMigrationVersion records a Go migration's version so that
// headMigrationVersion can discover it. Call this from each Go migration's init().
func registerGoMigrationVersion(v int64) {
	goMigrationVersions = append(goMigrationVersions, v)
}

// headMigrationVersion returns the highest version number across both embedded SQL
// migrations (migrations/*.sql) and registered Go migrations.
func headMigrationVersion() (int64, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return 0, fmt.Errorf("read migration dir: %w", err)
	}

	var head int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		if v > head {
			head = v
		}
	}

	for _, v := range goMigrationVersions {
		if v > head {
			head = v
		}
	}

	return head, nil
}
