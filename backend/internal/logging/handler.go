package logging

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// Writer constants.
const (
	queueSize      = 1024                   // Buffered channel capacity for log entries.
	maxBatchSize   = 50                     // Max entries per DB transaction.
	flushInterval  = 250 * time.Millisecond // Max time between DB flushes.
	dbWriteTimeout = 5 * time.Second        // Per-batch DB write timeout.
)

// writerCore is the shared state for the single background writer goroutine.
// Created once by NewLogHandler; shared by all WithAttrs/WithGroup clones via pointer.
type writerCore struct {
	db        *sql.DB
	queue     chan model.Log     // buffered (queueSize)
	flushCh   chan chan struct{} // synchronous flush signal
	done      chan struct{}      // closed to signal shutdown
	stopped   chan struct{}      // closed when writer exits
	closeOnce sync.Once
}

// LogHandler is a dual-write slog.Handler that writes structured logs to both
// stderr (console) and the database log table via a background writer.
type LogHandler struct {
	writer  *writerCore  // shared writer goroutine
	console slog.Handler // TextHandler for stderr
	enabled *atomic.Bool
	level   *atomic.Int32 // slog.Level stored as int32
	attrs   []slog.Attr   // pre-bound attributes from WithAttrs
	groups  []string      // pre-bound groups from WithGroup
}

// NewLogHandler creates a LogHandler with defaults (enabled=true, level=INFO)
// and starts the background writer goroutine.
func NewLogHandler(db *sql.DB) *LogHandler {
	enabled := &atomic.Bool{}
	enabled.Store(true)
	level := &atomic.Int32{}
	level.Store(int32(slog.LevelInfo))

	w := &writerCore{
		db:      db,
		queue:   make(chan model.Log, queueSize),
		flushCh: make(chan chan struct{}),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go w.startWriter()

	return &LogHandler{
		writer: w,
		console: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:       slog.LevelDebug, // console doesn't filter; our Enabled() gates
			ReplaceAttr: ReplaceLevel,
		}),
		enabled: enabled,
		level:   level,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.Level(h.level.Load())
}

// Handle writes the log record to console (always) and enqueues for database write (if enabled).
//
//nolint:funlen // Dual-write logic with attr extraction needs the length.
func (h *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Always write to console — the console handler has pre-bound attrs from WithAttrs.
	_ = h.console.Handle(ctx, record) //nolint:errcheck // Console write is best-effort; nothing to do on failure.

	// Only enqueue for DB if enabled.
	if !h.enabled.Load() {
		return nil
	}

	entry := h.buildEntry(ctx, record)

	// Non-blocking send: if queue is full, drop the entry (console already has it).
	select {
	case h.writer.queue <- entry:
	default:
		_, _ = fmt.Fprintf(os.Stderr, "logging: queue full, dropping DB entry: %s\n", entry.Message)
	}

	return nil
}

// buildEntry extracts all fields from the record on the caller's goroutine.
//
//nolint:funlen,gocyclo // Attr extraction mirrors the old Handle logic.
func (h *LogHandler) buildEntry(ctx context.Context, record slog.Record) model.Log {
	// Extract category from pre-bound attrs (default "SYSTEM").
	// IMPORTANT: "category", "status", and "source" are reserved attr keys —
	// they map to dedicated DB columns. If caller code passes any of these as
	// a log arg, the record-level attr overrides the default value. Use
	// prefixed keys instead (e.g. "filter_category", "response_status").
	category := "SYSTEM"
	var httpStatus string
	var sourceOverride string
	var details []string

	for _, a := range h.attrs {
		if a.Key == "category" {
			category = strings.ToUpper(a.Value.String())
			continue
		}
		if a.Key == "status" {
			if a.Value.Kind() == slog.KindInt64 {
				httpStatus = strconv.FormatInt(a.Value.Int64(), 10)
			} else {
				details = append(details, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
			}
			continue
		}
		if a.Key == "source" {
			sourceOverride = a.Value.String()
			continue
		}
		details = append(details, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
	}

	// Also check record attrs for reserved keys; collect everything else as details.
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "category" {
			category = strings.ToUpper(a.Value.String())
			return true
		}
		if a.Key == "status" {
			if a.Value.Kind() == slog.KindInt64 {
				httpStatus = strconv.FormatInt(a.Value.Int64(), 10)
			} else {
				details = append(details, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
			}
			return true
		}
		if a.Key == "source" {
			sourceOverride = a.Value.String()
			return true
		}
		details = append(details, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
		return true
	})

	// Request metadata from context.
	requestID := RequestIDFromContext(ctx)
	ipAddress := IPFromContext(ctx)
	userAgent := UserAgentFromContext(ctx)

	// Source: use explicit override if provided, otherwise derive from PC.
	source := sourceOverride
	if source == "" && record.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := frames.Next()
		if f.Function != "" {
			parts := strings.Split(f.Function, "/")
			source = parts[len(parts)-1]
		}
	}

	// Stack trace for ERROR and CRITICAL.
	var stackTrace string
	if record.Level >= slog.LevelError {
		stackTrace = captureStackTrace(5) // +1 frame vs old Handle() for buildEntry
	}

	return model.Log{
		ID:         uuid.New().String(),
		Timestamp:  record.Time.UTC(),
		Level:      SlogLevelToDBString(record.Level),
		Category:   category,
		Message:    record.Message,
		Details:    strings.Join(details, "; "),
		Source:     source,
		RequestID:  requestID,
		StackTrace: stackTrace,
		HTTPStatus: httpStatus,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}
}

// startWriter is the single background goroutine that batches and flushes entries to the DB.
func (w *writerCore) startWriter() {
	defer close(w.stopped)

	batch := make([]model.Log, 0, maxBatchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-w.queue:
			batch = append(batch, entry)
			if len(batch) >= maxBatchSize {
				w.flushBatch(batch)
				batch = batch[:0]
				ticker.Reset(flushInterval)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.flushBatch(batch)
				batch = batch[:0]
			}

		case ack := <-w.flushCh:
			// Drain queue into batch, then flush everything.
			draining := true
			for draining {
				select {
				case entry := <-w.queue:
					batch = append(batch, entry)
				default:
					draining = false
				}
			}
			if len(batch) > 0 {
				w.flushBatch(batch)
				batch = batch[:0]
			}
			close(ack)

		case <-w.done:
			// Drain remaining entries from queue.
			draining := true
			for draining {
				select {
				case entry := <-w.queue:
					batch = append(batch, entry)
				default:
					draining = false
				}
			}
			if len(batch) > 0 {
				w.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch writes a batch of log entries to the DB in a single transaction.
func (w *writerCore) flushBatch(batch []model.Log) {
	if len(batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "logging: begin tx failed: %v (dropping %d entries)\n", err, len(batch))
		return
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO log (id, timestamp, level, category, message, details, source, request_id, stack_trace, http_status, ip_address, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "logging: prepare stmt failed: %v (dropping %d entries)\n", err, len(batch))
		defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.
		return
	}
	defer stmt.Close()

	for _, e := range batch {
		// Convert HTTPStatus string → *int for the INTEGER column.
		// buildEntry only sets this from KindInt64 attrs (PR #71), so Atoi is safe;
		// the error guard is defensive against any future misuse.
		var httpStatus *int
		if e.HTTPStatus != "" {
			if v, err := strconv.Atoi(e.HTTPStatus); err == nil {
				httpStatus = &v
			} else {
				_, _ = fmt.Fprintf(os.Stderr,
					"logging: non-numeric http_status %q in log entry %s, storing as NULL\n",
					e.HTTPStatus, e.ID)
			}
		}

		_, execErr := stmt.ExecContext(ctx, e.ID, e.Timestamp.Format("2006-01-02 15:04:05.000"),
			e.Level, e.Category, e.Message, e.Details,
			e.Source, e.RequestID, e.StackTrace, httpStatus, e.IPAddress, e.UserAgent)
		if execErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "logging: insert failed: %v (msg=%q)\n", execErr, e.Message)
			defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.
			return
		}
	}

	if err := tx.Commit(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "logging: commit failed: %v (dropping %d entries)\n", err, len(batch))
	}
}

// Flush blocks until all queued entries have been written to the database.
func (h *LogHandler) Flush() {
	ack := make(chan struct{})
	select {
	case h.writer.flushCh <- ack:
		<-ack
	case <-h.writer.stopped:
		// Writer already exited.
	}
}

// Close signals the writer to drain remaining entries and exit, then blocks
// until the writer goroutine has finished. Safe to call multiple times.
func (h *LogHandler) Close() {
	h.writer.closeOnce.Do(func() {
		close(h.writer.done)
	})
	<-h.writer.stopped
}

// WithAttrs returns a new handler with the given attributes pre-bound.
// The new handler shares the writer, enabled, and level with the parent.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)

	return &LogHandler{
		writer:  h.writer, // shared pointer
		console: h.console.WithAttrs(attrs),
		enabled: h.enabled, // shared pointer
		level:   h.level,   // shared pointer
		attrs:   newAttrs,
		groups:  h.groups,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)

	return &LogHandler{
		writer:  h.writer, // shared pointer
		console: h.console.WithGroup(name),
		enabled: h.enabled,
		level:   h.level,
		attrs:   h.attrs,
		groups:  newGroups,
	}
}

// SetEnabled enables or disables database logging at runtime.
func (h *LogHandler) SetEnabled(v bool) {
	h.enabled.Store(v)
}

// SetLevel sets the minimum log level at runtime.
func (h *LogHandler) SetLevel(level slog.Level) {
	h.level.Store(int32(level)) //nolint:gosec // G115: slog levels are small constants (-4 to 12); overflow impossible.
}

// captureStackTrace returns a formatted stack trace, skipping `skip` frames.
func captureStackTrace(skip int) string {
	const maxFrames = 32
	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return ""
	}
	pcs = pcs[:n]
	frames := runtime.CallersFrames(pcs)

	var b strings.Builder
	for {
		f, more := frames.Next()
		fmt.Fprintf(&b, "%s\n\t%s:%d\n", f.Function, f.File, f.Line)
		if !more {
			break
		}
	}
	return b.String()
}
