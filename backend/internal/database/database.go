// Package database manages the SQLite database connection and schema migrations for the application.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // SQLite driver
)

// EnsureDir creates the directory tree for the given database file path if it does not already exist.
func EnsureDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o750)
}

// Open opens a connection to the SQLite database.
func Open(dbPath string) (*sql.DB, error) {
	// Build DSN with per-connection PRAGMA parameters.
	//
	// Critically, PRAGMAs must be in the DSN rather than executed via db.Exec
	// after opening, because db.Exec only runs on one connection from the pool.
	// Every subsequent connection the pool opens would miss those settings,
	// causing SQLITE_BUSY failures when busy_timeout is absent or foreign-key
	// violations when foreign_keys is unset.
	//
	// The modernc.org/sqlite driver applies _pragma= parameters to every new
	// connection and always sorts busy_timeout first (before journal_mode, etc.)
	// so the timeout is active before any locking occurs.
	//
	// Parameters:
	//   _texttotime=1          — auto-parse DATE/DATETIME TEXT columns to time.Time (v1.46.0+)
	//   busy_timeout(5000)     — wait up to 5 s when another writer holds the lock
	//   foreign_keys(on)       — enforce FK constraints (off by default in SQLite)
	//   journal_mode(WAL)      — WAL allows concurrent readers alongside a writer
	//   wal_autocheckpoint(100)— checkpoint every ~400 KB instead of the default 4 MB;
	//                            keeps the WAL file small and changes visible sooner
	sep := "?"
	if strings.Contains(dbPath, "?") {
		sep = "&"
	}
	dsn := dbPath + sep +
		"_texttotime=1" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=foreign_keys(on)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=wal_autocheckpoint(100)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// HealthCheck performs a simple health check on the database.
func HealthCheck(db *sql.DB) error {
	return db.Ping()
}
