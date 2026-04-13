package validation

import (
	"errors"
	"testing"
	"time"
)

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID v4 lowercase", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", false},
		{"valid UUID nil", "00000000-0000-0000-0000-000000000000", false},
		{"empty string", "", true},
		{"too short", "550e8400", true},
		{"missing dashes", "550e8400e29b41d4a716446655440000", false}, // google/uuid accepts this
		{"invalid characters", "gggggggg-gggg-gggg-gggg-gggggggggggg", true},
		{"spaces", "  ", true},
		{"random text", "not-a-uuid-at-all", true},
		{"almost valid but wrong length", "550e8400-e29b-41d4-a716-44665544000", true},
		{"extra characters", "550e8400-e29b-41d4-a716-4466554400001", true},
		{"with braces", "{550e8400-e29b-41d4-a716-446655440000}", false}, // google/uuid accepts braced UUIDs
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUUID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrInvalidUUID) {
				t.Errorf("ValidateUUID(%q) error should wrap ErrInvalidUUID, got %v", tt.id, err)
			}
		})
	}
}

func TestValidateUUIDs(t *testing.T) {
	validUUID1 := "550e8400-e29b-41d4-a716-446655440000"
	validUUID2 := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

	tests := []struct {
		name      string
		ids       []string
		wantErr   bool
		errTarget error
	}{
		{"empty slice", []string{}, true, ErrEmptySlice},
		{"single valid", []string{validUUID1}, false, nil},
		{"multiple valid", []string{validUUID1, validUUID2}, false, nil},
		{"first invalid", []string{"bad", validUUID2}, true, ErrInvalidUUID},
		{"second invalid", []string{validUUID1, "bad"}, true, ErrInvalidUUID},
		{"all invalid", []string{"bad1", "bad2"}, true, ErrInvalidUUID},
		{"nil slice", nil, true, ErrEmptySlice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUIDs(tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUUIDs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errTarget != nil && !errors.Is(err, tt.errTarget) {
				t.Errorf("ValidateUUIDs() error = %v, want %v", err, tt.errTarget)
			}
		})
	}
}

func TestValidateISIN(t *testing.T) {
	tests := []struct {
		name string
		isin string
		want bool
	}{
		// Valid ISINs
		{"US Apple", "US0378331005", true},
		{"NL Unilever", "NL0000009165", true},
		{"GB Vodafone", "GB00BH4HKS39", true},
		{"DE Siemens", "DE0007236101", true},

		// Invalid format
		{"empty string", "", false},
		{"too short", "US037833100", false},
		{"too long", "US03783310051", false},
		{"lowercase country", "us0378331005", false},
		{"no country code", "000378331005", false},
		{"special characters", "US@378331005", false},
		{"spaces in ISIN", "US 378331005", false},

		// Invalid checksum
		{"wrong checksum", "US0378331006", false},
		{"wrong checksum 2", "US0378331009", false},

		// Edge cases
		{"all letters except check", "ABCDEFGHIJK0", false}, // likely wrong checksum
		{"numeric country code", "120378331005", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateISIN(tt.isin)
			if got != tt.want {
				t.Errorf("validateISIN(%q) = %v, want %v", tt.isin, got, tt.want)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		wantErr bool
		wantUTC bool
	}{
		{"date format", "2024-01-15", false, true},
		{"date format edge", "2000-01-01", false, true},
		{"RFC3339 format", "2024-01-15T10:30:00Z", false, true},
		{"RFC3339 with offset", "2024-01-15T10:30:00+05:00", false, true},
		{"empty string", "", true, false},
		{"invalid format", "01-15-2024", true, false},
		{"partial date", "2024-01", true, false},
		{"invalid month", "2024-13-01", true, false},
		{"invalid day", "2024-01-32", true, false},
		{"random text", "not-a-date", true, false},
		{"timestamp without T", "2024-01-15 10:30:00", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTime(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTime(%q) error = %v, wantErr %v", tt.str, err, tt.wantErr)
			}
			if !tt.wantErr && tt.wantUTC && result.Location() != time.UTC {
				t.Errorf("ParseTime(%q) location = %v, want UTC", tt.str, result.Location())
			}
		})
	}
}
