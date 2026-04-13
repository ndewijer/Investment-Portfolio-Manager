package request

import (
	"testing"
)

//nolint:gocyclo // Test functions naturally have high complexity due to many test cases
func TestParseLogFilters(t *testing.T) {
	t.Run("default values when no parameters provided", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.SortDir != "desc" {
			t.Errorf("Expected default SortDir 'desc', got '%s'", filters.SortDir)
		}

		if filters.PerPage != 50 {
			t.Errorf("Expected default PerPage 50, got %d", filters.PerPage)
		}

		if len(filters.Levels) != 0 {
			t.Errorf("Expected empty Levels, got %v", filters.Levels)
		}

		if len(filters.Categories) != 0 {
			t.Errorf("Expected empty Categories, got %v", filters.Categories)
		}
	})

	t.Run("single level filter", func(t *testing.T) {
		filters, err := ParseLogFilters("error", "", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(filters.Levels) != 1 {
			t.Fatalf("Expected 1 level, got %d", len(filters.Levels))
		}

		if filters.Levels[0] != "error" {
			t.Errorf("Expected level 'error', got '%s'", filters.Levels[0])
		}
	})

	t.Run("multiple levels filter", func(t *testing.T) {
		filters, err := ParseLogFilters("error,critical,warning", "", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(filters.Levels) != 3 {
			t.Fatalf("Expected 3 levels, got %d", len(filters.Levels))
		}

		expected := []string{"error", "critical", "warning"}
		for i, level := range filters.Levels {
			if level != expected[i] {
				t.Errorf("Expected level '%s' at index %d, got '%s'", expected[i], i, level)
			}
		}
	})

	t.Run("invalid level returns error", func(t *testing.T) {
		_, err := ParseLogFilters("invalid_level", "", "", "", "", "", "", "", "", "")
		if err == nil {
			t.Error("Expected error for invalid level, got nil")
		}
	})

	t.Run("single category filter", func(t *testing.T) {
		filters, err := ParseLogFilters("", "portfolio", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(filters.Categories) != 1 {
			t.Fatalf("Expected 1 category, got %d", len(filters.Categories))
		}

		if filters.Categories[0] != "portfolio" {
			t.Errorf("Expected category 'portfolio', got '%s'", filters.Categories[0])
		}
	})

	t.Run("multiple categories filter", func(t *testing.T) {
		filters, err := ParseLogFilters("", "portfolio,fund,transaction", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(filters.Categories) != 3 {
			t.Fatalf("Expected 3 categories, got %d", len(filters.Categories))
		}

		expected := []string{"portfolio", "fund", "transaction"}
		for i, category := range filters.Categories {
			if category != expected[i] {
				t.Errorf("Expected category '%s' at index %d, got '%s'", expected[i], i, category)
			}
		}
	})

	t.Run("invalid category returns error", func(t *testing.T) {
		_, err := ParseLogFilters("", "invalid_category", "", "", "", "", "", "", "", "")
		if err == nil {
			t.Error("Expected error for invalid category, got nil")
		}
	})

	t.Run("date range parsing", func(t *testing.T) {
		startDate := "2024-01-01T00:00:00Z"
		endDate := "2024-12-31T23:59:59Z"

		filters, err := ParseLogFilters("", "", startDate, endDate, "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.StartDate == nil {
			t.Error("Expected StartDate to be set")
		}

		if filters.EndDate == nil {
			t.Error("Expected EndDate to be set")
		}

		if filters.StartDate != nil && filters.StartDate.Year() != 2024 {
			t.Errorf("Expected StartDate year 2024, got %d", filters.StartDate.Year())
		}

		if filters.EndDate != nil && filters.EndDate.Year() != 2024 {
			t.Errorf("Expected EndDate year 2024, got %d", filters.EndDate.Year())
		}
	})

	t.Run("invalid start date returns error", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "invalid-date", "", "", "", "", "", "", "")
		if err == nil {
			t.Error("Expected error for invalid start_date, got nil")
		}
	})

	t.Run("invalid end date returns error", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "", "invalid-date", "", "", "", "", "", "")
		if err == nil {
			t.Error("Expected error for invalid end_date, got nil")
		}
	})

	t.Run("source filter", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "PortfolioHandler", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.Source != "PortfolioHandler" {
			t.Errorf("Expected Source 'PortfolioHandler', got '%s'", filters.Source)
		}
	})

	t.Run("message filter", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "", "failed to connect", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.Message != "failed to connect" {
			t.Errorf("Expected Message 'failed to connect', got '%s'", filters.Message)
		}
	})

	t.Run("sort direction asc", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "", "", "asc", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.SortDir != "asc" {
			t.Errorf("Expected SortDir 'asc', got '%s'", filters.SortDir)
		}
	})

	t.Run("sort direction desc", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "", "", "desc", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.SortDir != "desc" {
			t.Errorf("Expected SortDir 'desc', got '%s'", filters.SortDir)
		}
	})

	t.Run("invalid sort direction returns error", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "", "", "", "", "invalid", "", "", "")
		if err == nil {
			t.Error("Expected error for invalid sort_dir, got nil")
		}
	})

	t.Run("custom per_page", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "", "", "", "", "25", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.PerPage != 25 {
			t.Errorf("Expected PerPage 25, got %d", filters.PerPage)
		}
	})

	t.Run("per_page too low returns error", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "", "", "", "", "", "", "0", "")
		if err == nil {
			t.Error("Expected error for per_page < 1, got nil")
		}
	})

	t.Run("per_page too high returns error", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "", "", "", "", "", "", "251", "")
		if err == nil {
			t.Error("Expected error for per_page > 250, got nil")
		}
	})

	t.Run("invalid per_page returns error", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "", "", "", "", "", "", "not-a-number", "")
		if err == nil {
			t.Error("Expected error for non-numeric per_page, got nil")
		}
	})

	t.Run("cursor is stored", func(t *testing.T) {
		cursor := "2024-01-15T10:30:00Z_uuid-here"
		filters, err := ParseLogFilters("", "", "", "", "", "", "", cursor, "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.Cursor != cursor {
			t.Errorf("Expected Cursor '%s', got '%s'", cursor, filters.Cursor)
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		filters, err := ParseLogFilters(
			"error,critical",
			"portfolio,fund",
			"2024-01-01T00:00:00Z",
			"2024-12-31T23:59:59Z",
			"Handler",
			"",
			"asc",
			"",
			"20",
			"",
		)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(filters.Levels) != 2 {
			t.Errorf("Expected 2 levels, got %d", len(filters.Levels))
		}

		if len(filters.Categories) != 2 {
			t.Errorf("Expected 2 categories, got %d", len(filters.Categories))
		}

		if filters.StartDate == nil || filters.EndDate == nil {
			t.Error("Expected date range to be set")
		}

		if filters.Source != "Handler" {
			t.Errorf("Expected Source 'Handler', got '%s'", filters.Source)
		}

		if filters.SortDir != "asc" {
			t.Errorf("Expected SortDir 'asc', got '%s'", filters.SortDir)
		}

		if filters.PerPage != 20 {
			t.Errorf("Expected PerPage 20, got %d", filters.PerPage)
		}
	})

	t.Run("valid skip parameter", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "", "", "", "", "", "100")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.Skip != 100 {
			t.Errorf("Expected Skip 100, got %d", filters.Skip)
		}
	})

	t.Run("invalid skip - not a number", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "", "", "", "", "", "", "", "abc")
		if err == nil {
			t.Error("Expected error for non-numeric skip, got nil")
		}
	})

	t.Run("invalid skip - negative", func(t *testing.T) {
		_, err := ParseLogFilters("", "", "", "", "", "", "", "", "", "-1")
		if err == nil {
			t.Error("Expected error for negative skip, got nil")
		}
	})

	t.Run("skip defaults to 0", func(t *testing.T) {
		filters, err := ParseLogFilters("", "", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if filters.Skip != 0 {
			t.Errorf("Expected Skip 0, got %d", filters.Skip)
		}
	})
}
