package model

import "time"

// LogLevel represents an allowed log level value.
type LogLevel string

// Log level constants define the valid severity levels for structured log entries.
const (
	LogLevelDebug    LogLevel = "debug"
	LogLevelInfo     LogLevel = "info"
	LogLevelWarning  LogLevel = "warning"
	LogLevelError    LogLevel = "error"
	LogLevelCritical LogLevel = "critical"
)

// ValidLogLevels is the authoritative set of allowed log level values.
// Use this for validation anywhere log levels are accepted as input.
var ValidLogLevels = map[LogLevel]bool{
	LogLevelDebug:    true,
	LogLevelInfo:     true,
	LogLevelWarning:  true,
	LogLevelError:    true,
	LogLevelCritical: true,
}

// LogCategory represents an allowed log category value.
type LogCategory string

// Log category constants define the valid categories that classify log entries by domain.
const (
	LogCategoryPortfolio   LogCategory = "portfolio"
	LogCategoryFund        LogCategory = "fund"
	LogCategoryTransaction LogCategory = "transaction"
	LogCategoryDividend    LogCategory = "dividend"
	LogCategorySystem      LogCategory = "system"
	LogCategoryDatabase    LogCategory = "database"
	LogCategorySecurity    LogCategory = "security"
	LogCategoryIBKR        LogCategory = "ibkr"
	LogCategoryDeveloper   LogCategory = "developer"
)

// ValidLogCategories is the authoritative set of allowed log category values.
// Use this for validation anywhere log categories are accepted as input.
var ValidLogCategories = map[LogCategory]bool{
	LogCategoryPortfolio:   true,
	LogCategoryFund:        true,
	LogCategoryTransaction: true,
	LogCategoryDividend:    true,
	LogCategorySystem:      true,
	LogCategoryDatabase:    true,
	LogCategorySecurity:    true,
	LogCategoryIBKR:        true,
	LogCategoryDeveloper:   true,
}

// LogFilterOptions contains distinct values for log filter columns.
// Used by the frontend to populate filter picklists.
type LogFilterOptions struct {
	Levels     []string `json:"levels"`
	Categories []string `json:"categories"`
	Sources    []string `json:"sources"`
	Messages   []string `json:"messages"`
}

// LogFilters represents parsed and validated parameters for querying system logs.
// All fields are optional and can be combined. Populated by the HTTP layer and
// consumed directly by the repository — no API-layer dependency required.
type LogFilters struct {
	Levels     []string   // Log levels to filter by (debug, info, warning, error, critical)
	Categories []string   // Categories to filter by (portfolio, fund, transaction, etc.)
	StartDate  *time.Time // Filter logs from this timestamp onwards (inclusive)
	EndDate    *time.Time // Filter logs up to this timestamp (inclusive)
	Source     string     // Filter by source field using partial match
	Message    string     // Filter by message content using partial match
	SortDir    string     // Sort direction: "asc" or "desc" (default: "desc")
	Cursor     string     // Pagination cursor from previous response (format: "timestamp_id")
	PerPage    int        // Number of results per page (1-100, default: 50)
	Skip       int        // Number of entries to skip before returning results (for page jumping)
}

// LogResponse represents a paginated response containing log entries.
// Includes cursor-based pagination information for retrieving subsequent pages.
type LogResponse struct {
	Logs       []Log  `json:"logs"`       // Array of log entries for the current page
	NextCursor string `json:"nextCursor"` // Cursor for fetching the next page (empty if no more pages)
	HasMore    bool   `json:"hasMore"`    // Indicates if more results are available
	Count      int    `json:"count"`      // Number of logs in the current response
}

// Log represents a single system log entry with metadata and optional contextual information.
type Log struct {
	ID         string    `json:"id"`                  // Unique identifier for the log entry
	Timestamp  time.Time `json:"timestamp"`           // When the log was created
	Level      string    `json:"level"`               // Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
	Category   string    `json:"category"`            // Category of the log (portfolio, fund, transaction, etc.)
	Message    string    `json:"message"`             // Primary log message
	Details    string    `json:"details,omitempty"`   // Additional details or context (optional)
	Source     string    `json:"source"`              // Source component that generated the log
	RequestID  string    `json:"requestId,omitempty"` // Request ID for tracing (optional)
	StackTrace string    `json:"stack_trace,omitempty"`
	HTTPStatus string    `json:"httpStatus,omitempty"` // HTTP status code if applicable (optional)
	IPAddress  string    `json:"ipAddress,omitempty"`  // IP address of the request (optional)
	UserAgent  string    `json:"userAgent,omitempty"`  // User agent string (optional)
}

// SystemSetting represents a key-value configuration setting stored in the database.
type SystemSetting struct {
	ID        string     // Unique identifier for the setting
	Key       string     // Setting key name
	Value     any        // Setting value (can be any type)
	UpdatedAt *time.Time // When the setting was last updated (optional)
}

// LoggingSetting represents the current logging configuration.
type LoggingSetting struct {
	Enabled bool   `json:"enabled"` // Whether logging is enabled
	Level   string `json:"level"`   // Current log level (debug, info, warning, error, critical)
}

// ExchangeRateWrapper wraps exchange rate query results.
// The Rate field will be nil if no exchange rate exists for the given parameters.
type ExchangeRateWrapper struct {
	FromCurrency string        `json:"fromCurrency"` // Source currency code
	ToCurrency   string        `json:"toCurrency"`   // Target currency code
	Rate         *ExchangeRate `json:"rate"`         // Exchange rate data (null if not found)
	Date         string        `json:"date"`         // Date for the query in YYYY-MM-DD format
}

// TemplateModel represents a CSV import template structure.
// Provides the expected headers, an example row, and format description.
type TemplateModel struct {
	Headers     []string          `json:"headers"`     // CSV column headers
	Example     map[string]string `json:"example"`     // Example values for each header
	Description string            `json:"description"` // Human-readable format description
}
