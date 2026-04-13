package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "subdir", "test.db")

	if err := EnsureDir(dbPath); err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmp, "subdir"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestEnsureDir_ExistingDirectory(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	// tmp already exists; EnsureDir should succeed without error.
	if err := EnsureDir(dbPath); err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}
}

func TestEnsureDir_NestedPath(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "a", "b", "c", "test.db")

	if err := EnsureDir(dbPath); err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "a", "b", "c")); err != nil {
		t.Fatalf("nested directory not created: %v", err)
	}
}

func TestEnsureDir_DotPath(t *testing.T) {
	// Dir is "." — should be a no-op.
	if err := EnsureDir("test.db"); err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}
}

func TestEnsureDir_EmptyDir(t *testing.T) {
	// filepath.Dir("") returns "." so this should be a no-op.
	if err := EnsureDir(""); err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}
}

func TestOpen_ValidPath(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	// Verify foreign keys are enabled.
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys error: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	// Verify WAL mode.
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode error: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}

	// Verify busy_timeout.
	var bt int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&bt); err != nil {
		t.Fatalf("PRAGMA busy_timeout error: %v", err)
	}
	if bt != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", bt)
	}
}

func TestOpen_DSNWithExistingQueryParam(t *testing.T) {
	tmp := t.TempDir()
	// Path already contains a query parameter.
	dbPath := filepath.Join(tmp, "test.db") + "?cache=shared"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	// Just verify it opened successfully.
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := HealthCheck(db); err != nil {
		t.Errorf("HealthCheck() error: %v", err)
	}
}

func TestHealthCheck_ClosedDB(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	db.Close()

	if err := HealthCheck(db); err == nil {
		t.Error("HealthCheck() expected error for closed DB, got nil")
	}
}
