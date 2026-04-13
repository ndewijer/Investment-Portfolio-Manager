package logging

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite DB with the log + system_setting tables.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_texttotime=1", uuid.New().String())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	for _, ddl := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		`CREATE TABLE IF NOT EXISTS log (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			level VARCHAR(8) NOT NULL,
			category VARCHAR(11) NOT NULL,
			message TEXT NOT NULL,
			details TEXT,
			source VARCHAR(255) NOT NULL,
			user_id VARCHAR(36),
			request_id VARCHAR(36),
			stack_trace TEXT,
			http_status INTEGER,
			ip_address VARCHAR(45),
			user_agent VARCHAR(255)
		)`,
		`CREATE TABLE IF NOT EXISTS system_setting (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			key VARCHAR(255) NOT NULL UNIQUE,
			value TEXT,
			updated_at DATETIME
		)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("exec ddl: %v", err)
		}
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// countLogs returns the number of rows in the log table.
func countLogs(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM log").Scan(&n); err != nil {
		t.Fatalf("count logs: %v", err)
	}
	return n
}

// lastLog returns the most recent log row.
type logRow struct {
	level, category, message, details, source string
	requestID, ipAddress, userAgent           string
	stackTrace                                string
}

func lastLog(t *testing.T, db *sql.DB) logRow {
	t.Helper()
	var r logRow
	var det, src, rid, ip, ua, st sql.NullString
	err := db.QueryRow(`
		SELECT level, category, message, details, source,
		       request_id, ip_address, user_agent, stack_trace
		FROM log ORDER BY timestamp DESC, id DESC LIMIT 1
	`).Scan(&r.level, &r.category, &r.message, &det, &src, &rid, &ip, &ua, &st)
	if err != nil {
		t.Fatalf("last log: %v", err)
	}
	r.details = det.String
	r.source = src.String
	r.requestID = rid.String
	r.ipAddress = ip.String
	r.userAgent = ua.String
	r.stackTrace = st.String
	return r
}

// --- Tests ---

func TestDBHandler_WritesToDB(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Info("hello world", "key", "val")
	h.Flush()

	if got := countLogs(t, db); got != 1 {
		t.Fatalf("expected 1 log row, got %d", got)
	}
	row := lastLog(t, db)
	if row.level != "INFO" {
		t.Errorf("level = %q, want INFO", row.level)
	}
	if row.category != "SYSTEM" {
		t.Errorf("category = %q, want SYSTEM (default)", row.category)
	}
	if row.message != "hello world" {
		t.Errorf("message = %q, want 'hello world'", row.message)
	}
	if row.details != "key=val" {
		t.Errorf("details = %q, want 'key=val'", row.details)
	}
}

func TestDBHandler_TimestampHasMillisecondPrecision(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)

	// Use a known timestamp with 456ms to verify milliseconds survive the round-trip.
	knownTime := time.Date(2026, 3, 27, 0, 55, 0, 456_000_000, time.UTC)
	rec := slog.NewRecord(knownTime, slog.LevelInfo, "millis check", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	h.Flush()

	// Scan into time.Time — the driver auto-parses DATETIME with _texttotime=1.
	var ts time.Time
	err := db.QueryRow(`SELECT timestamp FROM log ORDER BY rowid DESC LIMIT 1`).Scan(&ts)
	if err != nil {
		t.Fatalf("query timestamp: %v", err)
	}

	// Verify the millisecond component survived the round-trip (not truncated to .000).
	gotMs := ts.UnixMilli() % 1000
	if gotMs != 456 {
		t.Errorf("expected 456ms, got %dms — timestamp was %v", gotMs, ts)
	}
}

func TestDBHandler_CategoryFromWithAttrs(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h.WithAttrs([]slog.Attr{slog.String("category", "fund")}))

	logger.Info("price updated")
	h.Flush()

	row := lastLog(t, db)
	if row.category != "FUND" {
		t.Errorf("category = %q, want FUND", row.category)
	}
}

func TestDBHandler_CategoryFromRecordAttr(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Info("test", "category", "ibkr")
	h.Flush()

	row := lastLog(t, db)
	if row.category != "IBKR" {
		t.Errorf("category = %q, want IBKR", row.category)
	}
}

func TestDBHandler_LevelGating(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelWarn) // only WARN and above
	logger := slog.New(h)

	logger.Debug("should be filtered")
	logger.Info("should be filtered")
	logger.Warn("should pass")
	logger.Error("should pass")
	h.Flush()

	if got := countLogs(t, db); got != 2 {
		t.Fatalf("expected 2 log rows (WARN+ERROR), got %d", got)
	}
}

func TestDBHandler_SetLevelAtRuntime(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelError) // start restrictive
	logger := slog.New(h)

	logger.Info("filtered out")
	h.Flush()
	if got := countLogs(t, db); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}

	h.SetLevel(slog.LevelDebug) // open up
	logger.Info("now visible")
	h.Flush()
	if got := countLogs(t, db); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestDBHandler_DisabledSkipsDB(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	h.SetEnabled(false)
	logger := slog.New(h)

	logger.Info("should not be in DB")
	h.Flush()

	if got := countLogs(t, db); got != 0 {
		t.Fatalf("expected 0 log rows (disabled), got %d", got)
	}
}

func TestDBHandler_SetEnabledAtRuntime(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	h.SetEnabled(false)
	logger := slog.New(h)

	logger.Info("invisible")
	h.Flush()
	if got := countLogs(t, db); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}

	h.SetEnabled(true)
	logger.Info("visible")
	h.Flush()
	if got := countLogs(t, db); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestDBHandler_ContextMetadata(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	ctx := WithRequestInfo(context.Background(), "req-123", "10.0.0.1", "TestAgent/1.0")
	logger.InfoContext(ctx, "with context")
	h.Flush()

	row := lastLog(t, db)
	if row.requestID != "req-123" {
		t.Errorf("requestID = %q, want req-123", row.requestID)
	}
	if row.ipAddress != "10.0.0.1" {
		t.Errorf("ipAddress = %q, want 10.0.0.1", row.ipAddress)
	}
	if row.userAgent != "TestAgent/1.0" {
		t.Errorf("userAgent = %q, want TestAgent/1.0", row.userAgent)
	}
}

func TestDBHandler_StackTraceOnError(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Error("something broke")
	h.Flush()
	row := lastLog(t, db)
	if row.stackTrace == "" {
		t.Error("expected stack trace for ERROR, got empty string")
	}

	// DEBUG should NOT have stack trace
	logger.Debug("just debug")
	h.Flush()
	var st sql.NullString
	err := db.QueryRow(`SELECT stack_trace FROM log WHERE message = 'just debug'`).Scan(&st)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if st.Valid && st.String != "" {
		t.Errorf("expected no stack trace for DEBUG, got %q", st.String)
	}
}

func TestDBHandler_CriticalLevel(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Log(context.Background(), LevelCritical, "data integrity issue")
	h.Flush()

	row := lastLog(t, db)
	if row.level != "CRITICAL" {
		t.Errorf("level = %q, want CRITICAL", row.level)
	}
	if row.stackTrace == "" {
		t.Error("expected stack trace for CRITICAL, got empty string")
	}
}

func TestDBHandler_DBFailureFallback(t *testing.T) {
	// Use a DB without the log table to simulate failure.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.New().String())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	// Should not panic — falls back to stderr.
	logger.Info("this should not crash")
	h.Flush()
}

func TestDBHandler_ConcurrentWrites(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	const goroutines = 20
	const msgsPerGoroutine = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range msgsPerGoroutine {
				logger.Info("concurrent", "goroutine", id, "msg", j)
			}
		}(i)
	}
	wg.Wait()
	h.Flush()

	expected := goroutines * msgsPerGoroutine
	if got := countLogs(t, db); got != expected {
		t.Errorf("expected %d logs, got %d", expected, got)
	}
}

func TestDBHandler_WithAttrsSharesAtomics(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelError) // restrictive

	child := h.WithAttrs([]slog.Attr{slog.String("category", "fund")})
	childLogger := slog.New(child)

	childLogger.Info("filtered")
	h.Flush()
	if got := countLogs(t, db); got != 0 {
		t.Fatalf("expected 0 (level=ERROR), got %d", got)
	}

	// Change level on parent — child should see it.
	h.SetLevel(slog.LevelDebug)
	childLogger.Info("now visible")
	h.Flush()
	if got := countLogs(t, db); got != 1 {
		t.Fatalf("expected 1 (parent level changed to DEBUG), got %d", got)
	}
}

func TestDBHandler_Source(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Info("source test")
	h.Flush()

	row := lastLog(t, db)
	if row.source == "" {
		t.Error("expected non-empty source")
	}
	// Source should contain this test function name.
	if got := row.source; got == "" {
		t.Error("source is empty")
	}
}

func TestDBHandler_MultipleDetails(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Info("multi", "a", "1", "b", "2", "c", "3")
	h.Flush()

	row := lastLog(t, db)
	if row.details != "a=1; b=2; c=3" {
		t.Errorf("details = %q, want 'a=1; b=2; c=3'", row.details)
	}
}

func TestLogHandler_HTTPStatus_IntAttr(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Info("request complete", "status", 200)
	h.Flush()

	// Verify http_status is stored as an integer, not a string.
	var status sql.NullInt64
	err := db.QueryRow(`SELECT http_status FROM log WHERE message = 'request complete'`).Scan(&status)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !status.Valid {
		t.Fatal("expected http_status to be non-NULL")
	}
	if status.Int64 != 200 {
		t.Errorf("http_status = %d, want 200", status.Int64)
	}
}

func TestLogHandler_HTTPStatus_StringAttr_GoesToDetails(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	// When "status" is a non-int string, it should go to details, not http_status.
	logger.Info("weird status", "status", "ok")
	h.Flush()

	var status sql.NullInt64
	var details sql.NullString
	err := db.QueryRow(`SELECT http_status, details FROM log WHERE message = 'weird status'`).Scan(&status, &details)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if status.Valid {
		t.Errorf("expected http_status to be NULL for non-int status, got %d", status.Int64)
	}
	if !details.Valid || details.String != "status=ok" {
		t.Errorf("details = %q, want 'status=ok'", details.String)
	}
}

func TestLogHandler_HTTPStatus_Absent_IsNULL(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h)

	logger.Info("no status attr")
	h.Flush()

	var status sql.NullInt64
	err := db.QueryRow(`SELECT http_status FROM log WHERE message = 'no status attr'`).Scan(&status)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if status.Valid {
		t.Errorf("expected http_status to be NULL when no status attr, got %d", status.Int64)
	}
}

func TestLogHandler_HTTPStatus_WithAttrs_IntStatus(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	logger := slog.New(h.WithAttrs([]slog.Attr{slog.Int("status", 404)}))

	logger.Info("not found")
	h.Flush()

	var status sql.NullInt64
	err := db.QueryRow(`SELECT http_status FROM log WHERE message = 'not found'`).Scan(&status)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !status.Valid || status.Int64 != 404 {
		t.Errorf("http_status = %v (valid=%v), want 404", status.Int64, status.Valid)
	}
}

// --- Level conversion tests ---

func TestSlogLevelToDBString(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARNING"},
		{slog.LevelError, "ERROR"},
		{LevelCritical, "CRITICAL"},
		{slog.Level(20), "CRITICAL"}, // above critical
		{slog.Level(-10), "DEBUG"},   // below debug
	}
	for _, tt := range tests {
		if got := SlogLevelToDBString(tt.level); got != tt.want {
			t.Errorf("SlogLevelToDBString(%d) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestDBStringToSlogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"critical", LevelCritical},
		{"CRITICAL", LevelCritical},
		{"  INFO  ", slog.LevelInfo}, // whitespace
		{"unknown", slog.LevelInfo},  // default
		{"", slog.LevelInfo},         // empty
	}
	for _, tt := range tests {
		if got := DBStringToSlogLevel(tt.input); got != tt.want {
			t.Errorf("DBStringToSlogLevel(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestReplaceLevel(t *testing.T) {
	a := slog.Attr{Key: slog.LevelKey, Value: slog.AnyValue(slog.LevelWarn)}
	got := ReplaceLevel(nil, a)
	if got.Value.String() != "WARNING" {
		t.Errorf("ReplaceLevel(WARN) = %q, want WARNING", got.Value.String())
	}

	// Non-level attr should pass through unchanged.
	other := slog.String("foo", "bar")
	got2 := ReplaceLevel(nil, other)
	if got2.Value.String() != "bar" {
		t.Errorf("non-level attr changed: %q", got2.Value.String())
	}
}

// --- Context tests ---

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestInfo(ctx, "req-abc", "192.168.1.1", "Mozilla/5.0")

	if got := RequestIDFromContext(ctx); got != "req-abc" {
		t.Errorf("RequestID = %q, want req-abc", got)
	}
	if got := IPFromContext(ctx); got != "192.168.1.1" {
		t.Errorf("IP = %q, want 192.168.1.1", got)
	}
	if got := UserAgentFromContext(ctx); got != "Mozilla/5.0" {
		t.Errorf("UserAgent = %q, want Mozilla/5.0", got)
	}

	// Empty context should return empty strings.
	empty := context.Background()
	if got := RequestIDFromContext(empty); got != "" {
		t.Errorf("RequestID on empty ctx = %q, want empty", got)
	}
}

// --- Init tests ---

func TestInit_DefaultsWhenNoConfig(t *testing.T) {
	db := setupTestDB(t)

	// No system_setting rows — Init should use defaults.
	h := Init(db)
	defer h.Close()

	if !h.enabled.Load() {
		t.Error("expected enabled=true by default")
	}
	if got := slog.Level(h.level.Load()); got != slog.LevelInfo {
		t.Errorf("expected level=INFO by default, got %d", got)
	}
}

func TestInit_ReadsConfigFromDB(t *testing.T) {
	db := setupTestDB(t)

	// Seed system_setting.
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO system_setting (id, key, value, updated_at) VALUES (?, 'LOGGING_ENABLED', 'false', ?)`,
		uuid.New().String(), now)
	if err != nil {
		t.Fatalf("insert enabled: %v", err)
	}
	_, err = db.Exec(`INSERT INTO system_setting (id, key, value, updated_at) VALUES (?, 'LOGGING_LEVEL', 'error', ?)`,
		uuid.New().String(), now)
	if err != nil {
		t.Fatalf("insert level: %v", err)
	}

	h := Init(db)
	defer h.Close()

	if h.enabled.Load() {
		t.Error("expected enabled=false from DB config")
	}
	if got := slog.Level(h.level.Load()); got != slog.LevelError {
		t.Errorf("expected level=ERROR from DB config, got %d", got)
	}
}

// --- Logger wrapper tests ---

func TestLogger_DelegatesToDefault(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(slog.Default()) })

	log := NewLogger("fund")
	log.Info("test message", "key", "val")
	h.Flush()

	if got := countLogs(t, db); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	row := lastLog(t, db)
	if row.category != "FUND" {
		t.Errorf("category = %q, want FUND", row.category)
	}
	if row.message != "test message" {
		t.Errorf("message = %q, want 'test message'", row.message)
	}
}

func TestLogger_AllLevels(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelDebug)
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(slog.Default()) })

	log := NewLogger("system")
	ctx := context.Background()

	log.Debug("d")
	log.DebugContext(ctx, "dc")
	log.Info("i")
	log.InfoContext(ctx, "ic")
	log.Warn("w")
	log.WarnContext(ctx, "wc")
	log.Error("e")
	log.ErrorContext(ctx, "ec")
	log.Log(ctx, LevelCritical, "c")
	h.Flush()

	if got := countLogs(t, db); got != 9 {
		t.Errorf("expected 9 logs (all levels), got %d", got)
	}
}

func TestLogger_RespectsLevelGating(t *testing.T) {
	db := setupTestDB(t)
	h := NewLogHandler(db)
	defer h.Close()
	h.SetLevel(slog.LevelWarn)
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(slog.Default()) })

	log := NewLogger("fund")
	log.Debug("filtered")
	log.Info("filtered")
	log.Warn("pass")
	log.Error("pass")
	h.Flush()

	if got := countLogs(t, db); got != 2 {
		t.Errorf("expected 2 (WARN+ERROR), got %d", got)
	}
}
