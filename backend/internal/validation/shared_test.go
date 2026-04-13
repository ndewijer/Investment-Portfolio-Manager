package validation

import (
	"strings"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]string
		checks []string // substrings that should appear
	}{
		{
			"single field",
			map[string]string{"name": "name is required"},
			[]string{"name: name is required"},
		},
		{
			"multiple fields",
			map[string]string{
				"name":        "name is required",
				"description": "too long",
			},
			[]string{"name: name is required", "description: too long"},
		},
		{
			"empty fields map",
			map[string]string{},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Error{Fields: tt.fields}
			got := e.Error()
			for _, check := range tt.checks {
				if !strings.Contains(got, check) {
					t.Errorf("Error() = %q, should contain %q", got, check)
				}
			}
		})
	}
}

func TestError_ImplementsErrorInterface(t *testing.T) {
	var err error = &Error{Fields: map[string]string{"test": "msg"}}
	if err.Error() == "" {
		t.Error("Error should return non-empty string")
	}
}
