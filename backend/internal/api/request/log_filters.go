package request

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// ParseLogFilters extracts and validates log filters from query parameters.
// Converts raw query string parameters into a validated model.LogFilters struct.
//
// Parameters are expected as comma-separated strings (for levels and categories)
// or single values (for other fields). All parameters are optional.
//
// Validation rules:
//   - levels: Must be valid log levels (debug, info, warning, error, critical)
//   - categories: Must be valid categories (portfolio, fund, transaction, etc.)
//   - startDate/endDate: Must be valid date/datetime strings (YYYY-MM-DD or RFC3339)
//   - sortDir: Must be "asc" or "desc" (defaults to "desc")
//   - perPage: Must be between 1 and 100 (defaults to 50)
//
// Returns an error if any parameter fails validation.
//
//nolint:gocyclo // Complex validation logic is intentional and clear
func ParseLogFilters(
	levelsParam, categoriesParam, startDateParam, endDateParam,
	sourceParam, messageParam, sortDirParam, cursorParam, perPageParam, skipParam string,
) (*model.LogFilters, error) {
	filters := &model.LogFilters{
		Cursor:  cursorParam,
		Source:  sourceParam,
		Message: messageParam,
	}

	// Parse levels (comma-separated)
	if levelsParam != "" {
		levels := strings.SplitSeq(levelsParam, ",")
		for level := range levels {
			level = strings.TrimSpace(strings.ToLower(level))
			if !model.ValidLogLevels[model.LogLevel(level)] {
				return nil, fmt.Errorf("invalid log level: %s", level)
			}
			filters.Levels = append(filters.Levels, level)
		}
	}

	// Parse categories (comma-separated)
	if categoriesParam != "" {
		categories := strings.SplitSeq(categoriesParam, ",")
		for category := range categories {
			category = strings.TrimSpace(strings.ToLower(category))
			if !model.ValidLogCategories[model.LogCategory(category)] {
				return nil, fmt.Errorf("invalid category: %s", category)
			}
			filters.Categories = append(filters.Categories, category)
		}
	}

	// Parse start_date
	if startDateParam != "" {
		startTime, err := parseFilterTime(startDateParam)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format: %w", err)
		}
		filters.StartDate = &startTime
	}

	// Parse end_date
	if endDateParam != "" {
		endTime, err := parseFilterTime(endDateParam)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
		filters.EndDate = &endTime
	}

	// Validate sort_dir
	if sortDirParam != "" {
		sortDir := strings.ToLower(sortDirParam)
		if sortDir != "asc" && sortDir != "desc" {
			return nil, fmt.Errorf("invalid sort_dir: must be 'asc' or 'desc'")
		}
		filters.SortDir = sortDir
	} else {
		filters.SortDir = "desc" // Default
	}

	// Parse and validate per_page
	if perPageParam != "" {
		perPage, err := strconv.Atoi(perPageParam)
		if err != nil {
			return nil, fmt.Errorf("invalid per_page: must be a number")
		}
		if perPage < 1 || perPage > 250 {
			return nil, fmt.Errorf("invalid per_page: must be between 1 and 250")
		}
		filters.PerPage = perPage
	} else {
		filters.PerPage = 50 // Default
	}

	// Parse and validate skip
	if skipParam != "" {
		skip, err := strconv.Atoi(skipParam)
		if err != nil {
			return nil, fmt.Errorf("invalid skip: must be a number")
		}
		if skip < 0 {
			return nil, fmt.Errorf("invalid skip: must be non-negative")
		}
		filters.Skip = skip
	}

	return filters, nil
}

// parseFilterTime parses date strings for log filter parameters.
// Accepts YYYY-MM-DD, RFC3339, and RFC3339 with milliseconds formats.
func parseFilterTime(str string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05.000Z07:00"} {
		if t, err := time.Parse(layout, str); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as a date or datetime", str)
}
