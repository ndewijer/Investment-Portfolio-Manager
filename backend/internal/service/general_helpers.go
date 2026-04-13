package service

import "math"

// round rounds a float64 value to two decimal places using the package RoundingPrecision constant.
// This function is used throughout the service layer to ensure consistent rounding of monetary
// values and share counts in API responses.
//
// The rounding uses the standard "round half up" approach via math.Round.
//
// Parameters:
//   - value: The floating-point value to round
//
// Returns the value rounded to two decimal places (0.01 precision).
//
// Example:
//
//	round(123.456789)  // returns 123.46
//	round(0.005)       // returns 0.01
//	round(1.994)       // returns 1.99
func round(value float64) float64 {
	return math.Round(value*RoundingPrecision) / RoundingPrecision
}
