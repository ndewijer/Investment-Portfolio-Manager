package repository

import (
	"testing"
	"time"
)

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestParseTime(t *testing.T) {
	t.Run("valid date-only format", func(t *testing.T) {
		result, err := ParseTime("2024-03-15")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		if !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("valid datetime format", func(t *testing.T) {
		result, err := ParseTime("2024-03-15 10:30:45")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)
		if !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("valid RFC3339 format", func(t *testing.T) {
		result, err := ParseTime("2024-03-15T10:30:45Z")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)
		if !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("RFC3339 with timezone offset returns UTC", func(t *testing.T) {
		result, err := ParseTime("2024-03-15T10:30:45+05:00")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Location() != time.UTC {
			t.Errorf("expected UTC location, got %v", result.Location())
		}
		expected := time.Date(2024, 3, 15, 5, 30, 45, 0, time.UTC)
		if !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("empty string returns error", func(t *testing.T) {
		_, err := ParseTime("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("invalid format returns error", func(t *testing.T) {
		_, err := ParseTime("15-03-2024")
		if err == nil {
			t.Fatal("expected error for invalid format")
		}
	})

	t.Run("garbage string returns error", func(t *testing.T) {
		_, err := ParseTime("not-a-date")
		if err == nil {
			t.Fatal("expected error for garbage string")
		}
	})

	t.Run("partial date returns error", func(t *testing.T) {
		_, err := ParseTime("2024-03")
		if err == nil {
			t.Fatal("expected error for partial date")
		}
	})

	t.Run("date-only result is in UTC", func(t *testing.T) {
		result, err := ParseTime("2024-01-01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Location() != time.UTC {
			t.Errorf("expected UTC, got %v", result.Location())
		}
	})

	t.Run("leap year date", func(t *testing.T) {
		result, err := ParseTime("2024-02-29")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Day() != 29 || result.Month() != 2 {
			t.Errorf("expected Feb 29, got %v", result)
		}
	})

	t.Run("boundary dates", func(t *testing.T) {
		tests := []string{
			"2000-01-01",
			"1970-01-01",
			"2099-12-31",
		}
		for _, dateStr := range tests {
			_, err := ParseTime(dateStr)
			if err != nil {
				t.Errorf("unexpected error for %s: %v", dateStr, err)
			}
		}
	})
}
