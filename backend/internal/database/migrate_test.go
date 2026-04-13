package database_test

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/database"
	_ "modernc.org/sqlite"
)

var updateGolden = flag.Bool("update-golden", false, "update golden schema file")

// setupFreshDB opens a fresh in-memory SQLite DB with the same PRAGMAs as production.
func setupFreshDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_texttotime=1", uuid.New().String())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("foreign_keys pragma: %v", err)
	}
	return db
}

// dumpSchema returns the full schema as a sorted, newline-joined string of SQL statements.
func dumpSchema(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(
		`SELECT sql FROM sqlite_master WHERE sql IS NOT NULL ORDER BY name`,
	)
	if err != nil {
		t.Fatalf("dumpSchema query: %v", err)
	}
	defer rows.Close()

	var stmts []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("dumpSchema scan: %v", err)
		}
		stmts = append(stmts, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("dumpSchema rows: %v", err)
	}
	return strings.Join(stmts, "\n\n") + "\n"
}

// TestMigrate_NewDatabase verifies that running migrations on a fresh DB creates all expected tables.
func TestMigrate_NewDatabase(t *testing.T) {
	db := setupFreshDB(t)

	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	expectedTables := []string{
		"dividend",
		"exchange_rate",
		"fund",
		"fund_history_materialized",
		"fund_price",
		"goose_db_version",
		"ibkr_config",
		"ibkr_import_cache",
		"ibkr_transaction",
		"ibkr_transaction_allocation",
		"log",
		"portfolio",
		"portfolio_fund",
		"realized_gain_loss",
		"symbol_info",
		"system_setting",
		"transaction",
	}

	for _, table := range expectedTables {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after migration: %v", table, err)
		}
	}
}

// TestMigrate_Idempotent verifies that calling Migrate twice produces no errors and no duplicate rows.
func TestMigrate_Idempotent(t *testing.T) {
	db := setupFreshDB(t)

	if err := database.Migrate(db); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}

	// system_setting seed data should not be duplicated
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM system_setting`).Scan(&count); err != nil {
		t.Fatalf("count system_setting: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 system_setting rows after idempotent migration, got %d", count)
	}
}

// TestMigrate_DefaultSettings verifies that the expected default system settings are seeded.
func TestMigrate_DefaultSettings(t *testing.T) {
	db := setupFreshDB(t)

	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	cases := []struct {
		key           string
		expectedValue string
	}{
		{"LOGGING_ENABLED", "true"},
		{"LOGGING_LEVEL", "info"},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			var value string
			err := db.QueryRow(`SELECT value FROM system_setting WHERE "key" = ?`, tc.key).Scan(&value)
			if err != nil {
				t.Fatalf("system_setting key %q not found: %v", tc.key, err)
			}
			if value != tc.expectedValue {
				t.Errorf("expected %q, got %q", tc.expectedValue, value)
			}
		})
	}
}

// TestMigrate_SchemaMatchesGoldenFile verifies that the migrated schema matches the committed golden file.
// Run with -update-golden to regenerate the golden file after intentional schema changes.
func TestMigrate_SchemaMatchesGoldenFile(t *testing.T) {
	db := setupFreshDB(t)

	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	actual := dumpSchema(t, db)
	goldenPath := "testdata/golden_schema.sql"

	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), 0o600); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	expectedBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file %q: %v (run with -update-golden to generate it)", goldenPath, err)
	}

	expected := string(expectedBytes)
	if actual != expected {
		// Show a simple diff hint
		actualLines := strings.Split(actual, "\n")
		expectedLines := strings.Split(expected, "\n")
		for i, line := range expectedLines {
			if i >= len(actualLines) {
				t.Errorf("line %d: golden has %q, actual is missing", i+1, line)
				break
			}
			if actualLines[i] != line {
				t.Errorf("first difference at line %d:\n  golden: %q\n  actual: %q", i+1, line, actualLines[i])
				break
			}
		}
		t.Error("schema drift detected; run `go test ./internal/database/... -update-golden` to regenerate")
	}
}
