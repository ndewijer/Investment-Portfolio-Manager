package logging

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// Logger provides categorized logging that always delegates to slog.Default().
// Unlike slog.Default().With(), this correctly picks up handler changes made
// by Init() after package-level variables are initialized.
type Logger struct {
	category string
}

// NewLogger creates a Logger with the given category.
// The category is stored as a "category" attr on the handler. "category",
// "status", and "source" are reserved attr keys that map to dedicated DB
// columns. Callers must not use these as log arg keys unless intentionally
// overriding (e.g. the middleware passes "source" to set the route pattern).
// Use prefixed keys for data values (e.g. "filter_category").
func NewLogger(category string) *Logger {
	return &Logger{category: category}
}

// handler returns the current default handler with the category pre-bound.
func (l *Logger) handler() slog.Handler {
	return slog.Default().With("category", l.category).Handler()
}

// log is the core method. Callers: runtime.Callers → log → public method → caller.
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	h := l.handler()
	if !h.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip: runtime.Callers, log, public method
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = h.Handle(ctx, r) //nolint:errcheck // Handler.Handle returns nil by contract; nothing to do on failure.
}

// Debug logs a message at DEBUG level with no context.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(context.Background(), slog.LevelDebug, msg, args...)
}

// DebugContext logs a message at DEBUG level with the provided context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelDebug, msg, args...)
}

// Info logs a message at INFO level with no context.
func (l *Logger) Info(msg string, args ...any) {
	l.log(context.Background(), slog.LevelInfo, msg, args...)
}

// InfoContext logs a message at INFO level with the provided context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelInfo, msg, args...)
}

// Warn logs a message at WARN level with no context.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(context.Background(), slog.LevelWarn, msg, args...)
}

// WarnContext logs a message at WARN level with the provided context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelWarn, msg, args...)
}

// Error logs a message at ERROR level with no context.
func (l *Logger) Error(msg string, args ...any) {
	l.log(context.Background(), slog.LevelError, msg, args...)
}

// ErrorContext logs a message at ERROR level with the provided context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelError, msg, args...)
}

// Log logs at the specified level. Use for LevelCritical:
//
//	log.Log(ctx, logging.LevelCritical, "data integrity issue", "table", "fund")
func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.log(ctx, level, msg, args...)
}
