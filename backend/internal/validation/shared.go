package validation

import (
	"fmt"
	"strings"
)

// Error is a validation error that wraps a human-readable message.
type Error struct {
	Fields map[string]string
}

// Error returns the validation error message.
func (e *Error) Error() string {
	msgs := make([]string, 0, len(e.Fields))
	for field, msg := range e.Fields {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(msgs, "; ")
}
