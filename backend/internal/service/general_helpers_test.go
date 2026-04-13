package service

import (
	"math"
	"testing"
)

func TestRound(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{name: "positive with truncation", input: 123.456789, expected: 123.456789},
		{name: "zero", input: 0, expected: 0},
		{name: "negative value", input: -123.456789, expected: -123.456789},
		{name: "small positive", input: 0.000001, expected: 0.000001},
		{name: "rounds up at boundary", input: 0.0000005, expected: 0.000001},
		{name: "rounds down at boundary", input: 0.0000004, expected: 0},
		{name: "large value", input: 999999.999999, expected: 999999.999999},
		{name: "negative rounds correctly", input: -0.0000005, expected: -0.000001},
		{name: "whole number unchanged", input: 42.0, expected: 42.0},
		{name: "one decimal place", input: 1.5, expected: 1.5},
		{name: "typical monetary value", input: 1234.56, expected: 1234.56},
		{name: "very small negative", input: -0.0000001, expected: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := round(tc.input)
			if math.Abs(got-tc.expected) > 1e-9 {
				t.Errorf("round(%v) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestRoundPrecision(t *testing.T) {
	// Verify RoundingPrecision is 1e6
	if RoundingPrecision != 1e6 {
		t.Errorf("RoundingPrecision = %v, want 1e6", RoundingPrecision)
	}

	// Verify round uses 6 decimal places
	val := 1.1234567
	got := round(val)
	expected := math.Round(val*1e6) / 1e6
	if got != expected {
		t.Errorf("round(%v) = %v, want %v", val, got, expected)
	}
}
