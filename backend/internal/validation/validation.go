// Package validation provides input validation helpers used by service and handler layers.
// It defines reusable error types and functions for validating UUIDs, date ranges, and other
// common inputs before they reach the repository layer.
package validation

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// Common validation errors returned by validation functions.
// These can be used with errors.Is() for error type checking.
var (
	ErrInvalidUUID      = fmt.Errorf("invalid UUID format")
	ErrInvalidDateRange = fmt.Errorf("invalid date range")
	ErrEmptySlice       = fmt.Errorf("slice cannot be empty")
)

// ValidateUUID checks if a string is a valid UUID (RFC 4122 format).
// Returns ErrInvalidUUID wrapped with the invalid ID if validation fails.
//
// Parameters:
//   - id: The string to validate as a UUID
//
// Returns:
//   - nil if the UUID is valid
//   - wrapped ErrInvalidUUID if the format is invalid
//
// Example:
//
//	if err := validation.ValidateUUID(portfolioID); err != nil {
//	    // Handle invalid UUID
//	}
func ValidateUUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidUUID, id)
	}
	return nil
}

// ValidateUUIDs validates a slice of UUID strings.
// Returns an error on the first invalid UUID encountered.
//
// Parameters:
//   - ids: Slice of strings to validate as UUIDs
//
// Returns:
//   - ErrEmptySlice if the input slice is empty
//   - wrapped ErrInvalidUUID if any UUID in the slice is invalid
//   - nil if all UUIDs are valid
//
// Example:
//
//	if err := validation.ValidateUUIDs(fundIDs); err != nil {
//	    // Handle validation error
//	}
func ValidateUUIDs(ids []string) error {
	if len(ids) == 0 {
		return ErrEmptySlice
	}
	for _, id := range ids {
		if err := ValidateUUID(id); err != nil {
			return err
		}
	}
	return nil
}

func validateISIN(isin string) bool {
	// Check format first
	isinRegex := regexp.MustCompile(`^([A-Z]{2})([A-Z0-9]{9})([0-9]{1})$`)
	if !isinRegex.MatchString(isin) {
		return false
	}

	// Convert letters to numbers (A=10, B=11, ..., Z=35)
	var digits []int
	for _, char := range isin[:11] {
		if char >= 'A' && char <= 'Z' {
			num := int(char - 'A' + 10)
			digits = append(digits, num/10, num%10)
		} else {
			digits = append(digits, int(char-'0'))
		}
	}

	// Apply Luhn algorithm
	sum := 0
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		if (len(digits)-1-i)%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}

	// Verify checksum
	checkDigit := (10 - (sum % 10)) % 10
	return checkDigit == int(isin[11]-'0')
}

// ParseTime parses a date string in "2006-01-02" or RFC3339 format.
// Note: mirrors repository.ParseTime — both are intentionally kept local to avoid cross-layer imports.
func ParseTime(str string) (time.Time, error) {
	returnTime, err := time.Parse("2006-01-02", str)
	if err != nil {
		returnTime, err = time.Parse(time.RFC3339, str)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse date: %w", err)
		}
	}
	return returnTime.UTC(), nil
}
