package logging

import (
	"log/slog"
	"strings"
)

// LevelCritical is a custom slog level for data integrity issues and unrecoverable state.
const LevelCritical = slog.Level(12)

// SlogLevelToDBString converts a slog.Level to the uppercase string stored in the database.
func SlogLevelToDBString(level slog.Level) string {
	switch {
	case level >= LevelCritical:
		return "CRITICAL"
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARNING"
	case level >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// DBStringToSlogLevel converts a database-stored level string to a slog.Level.
func DBStringToSlogLevel(s string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL":
		return LevelCritical
	case "ERROR":
		return slog.LevelError
	case "WARNING":
		return slog.LevelWarn
	case "INFO":
		return slog.LevelInfo
	case "DEBUG":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// ReplaceLevel is a slog.ReplaceAttr function that prints human-readable level names
// (e.g., "WARNING" instead of "WARN", "CRITICAL" instead of "ERROR+4").
func ReplaceLevel(_ []string, a slog.Attr) slog.Attr {
	if a.Key != slog.LevelKey {
		return a
	}
	level, ok := a.Value.Any().(slog.Level)
	if !ok {
		return a
	}
	a.Value = slog.StringValue(SlogLevelToDBString(level))
	return a
}
