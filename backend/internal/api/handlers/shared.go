package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// parseJSON parses a JSON request body into the specified type T.
// Includes security protections: 1MB body size limit and strict parsing (rejects unknown fields).
// Returns an error if the JSON is malformed or exceeds the size limit.
func parseJSON[T any](r *http.Request) (T, error) {
	var req T

	// Limit body size (prevent memory exhaustion attacks)
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB max

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing

	if err := decoder.Decode(&req); err != nil {
		return req, fmt.Errorf("invalid JSON: %w", err)
	}

	return req, nil
}
