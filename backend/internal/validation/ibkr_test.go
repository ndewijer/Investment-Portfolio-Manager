package validation

import (
	"errors"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

func TestValidateUpdateIbkrConfig(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }
	strPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }

	validToken := "123456789012345678901234" // 24 digits

	tests := []struct {
		name       string
		req        request.UpdateIbkrConfigRequest
		wantErr    bool
		fieldCheck string
	}{
		{
			"disabled - no validation",
			request.UpdateIbkrConfigRequest{Enabled: boolPtr(false)},
			false, "",
		},
		{
			"enabled nil - no validation",
			request.UpdateIbkrConfigRequest{},
			false, "",
		},
		{
			"enabled with valid token",
			request.UpdateIbkrConfigRequest{
				Enabled:   boolPtr(true),
				FlexToken: strPtr(validToken),
			},
			false, "",
		},
		{
			"enabled with short token",
			request.UpdateIbkrConfigRequest{
				Enabled:   boolPtr(true),
				FlexToken: strPtr("123"),
			},
			true, "flexToken",
		},
		{
			"enabled with non-numeric token",
			request.UpdateIbkrConfigRequest{
				Enabled:   boolPtr(true),
				FlexToken: strPtr("abcdefghijklmnopqrstuvwx"),
			},
			true, "flexToken",
		},
		{
			"enabled with empty token (trimmed)",
			request.UpdateIbkrConfigRequest{
				Enabled:   boolPtr(true),
				FlexToken: strPtr(""),
			},
			false, "", // empty token with trim is fine (skipped)
		},
		{
			"enabled with whitespace token",
			request.UpdateIbkrConfigRequest{
				Enabled:   boolPtr(true),
				FlexToken: strPtr("   "),
			},
			false, "", // whitespace-only is trimmed to empty, skipped
		},
		{
			"enabled with valid flex query ID",
			request.UpdateIbkrConfigRequest{
				Enabled:     boolPtr(true),
				FlexQueryID: strPtr("12345"),
			},
			false, "",
		},
		{
			"enabled with empty flex query ID",
			request.UpdateIbkrConfigRequest{
				Enabled:     boolPtr(true),
				FlexQueryID: strPtr(""),
			},
			true, "flexQueryId",
		},
		{
			"enabled with too long flex query ID",
			request.UpdateIbkrConfigRequest{
				Enabled:     boolPtr(true),
				FlexQueryID: strPtr("12345678901"),
			},
			true, "flexQueryId",
		},
		{
			"enabled with non-numeric flex query ID",
			request.UpdateIbkrConfigRequest{
				Enabled:     boolPtr(true),
				FlexQueryID: strPtr("abc"),
			},
			true, "flexQueryId",
		},
		{
			"enabled with valid token expires at",
			request.UpdateIbkrConfigRequest{
				Enabled:        boolPtr(true),
				TokenExpiresAt: strPtr("2024-12-31"),
			},
			false, "",
		},
		{
			"enabled with invalid token expires at",
			request.UpdateIbkrConfigRequest{
				Enabled:        boolPtr(true),
				TokenExpiresAt: strPtr("not-a-date"),
			},
			true, "tokenExpiresAt",
		},
		{
			"enabled with default allocation enabled but empty",
			request.UpdateIbkrConfigRequest{
				Enabled:                  boolPtr(true),
				DefaultAllocationEnabled: boolPtr(true),
				DefaultAllocations:       []request.Allocation{},
			},
			true, "defaultAllocations",
		},
		{
			"enabled with allocations summing to 100",
			request.UpdateIbkrConfigRequest{
				Enabled:                  boolPtr(true),
				DefaultAllocationEnabled: boolPtr(true),
				DefaultAllocations: []request.Allocation{
					{PortfolioID: strPtr("portfolio1"), Percentage: floatPtr(60.0)},
					{PortfolioID: strPtr("portfolio2"), Percentage: floatPtr(40.0)},
				},
			},
			false, "",
		},
		{
			"enabled with allocations not summing to 100",
			request.UpdateIbkrConfigRequest{
				Enabled:                  boolPtr(true),
				DefaultAllocationEnabled: boolPtr(true),
				DefaultAllocations: []request.Allocation{
					{PortfolioID: strPtr("portfolio1"), Percentage: floatPtr(30.0)},
					{PortfolioID: strPtr("portfolio2"), Percentage: floatPtr(40.0)},
				},
			},
			true, "defaultAllocations",
		},
		{
			"enabled with zero percentage allocation",
			request.UpdateIbkrConfigRequest{
				Enabled:                  boolPtr(true),
				DefaultAllocationEnabled: boolPtr(true),
				DefaultAllocations: []request.Allocation{
					{PortfolioID: strPtr("portfolio1"), Percentage: floatPtr(0.0)},
				},
			},
			true, "defaultAllocations",
		},
		{
			"enabled with nil portfolio in allocation",
			request.UpdateIbkrConfigRequest{
				Enabled:                  boolPtr(true),
				DefaultAllocationEnabled: boolPtr(true),
				DefaultAllocations: []request.Allocation{
					{PortfolioID: nil, Percentage: floatPtr(100.0)},
				},
			},
			true, "defaultAllocations",
		},
		{
			"enabled with empty portfolio in allocation",
			request.UpdateIbkrConfigRequest{
				Enabled:                  boolPtr(true),
				DefaultAllocationEnabled: boolPtr(true),
				DefaultAllocations: []request.Allocation{
					{PortfolioID: strPtr(""), Percentage: floatPtr(100.0)},
				},
			},
			true, "defaultAllocations",
		},
		{
			"default allocation disabled - no validation",
			request.UpdateIbkrConfigRequest{
				Enabled:                  boolPtr(true),
				DefaultAllocationEnabled: boolPtr(false),
			},
			false, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateIbkrConfig(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateIbkrConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.fieldCheck != "" {
				var valErr *Error
				if errors.As(err, &valErr) {
					if _, ok := valErr.Fields[tt.fieldCheck]; !ok {
						t.Errorf("expected error on field %q, got fields: %v", tt.fieldCheck, valErr.Fields)
					}
				}
			}
		})
	}
}

func TestValidateTestConnection(t *testing.T) {
	validToken := "123456789012345678901234"

	tests := []struct {
		name       string
		req        request.TestIbkrConnectionRequest
		wantErr    bool
		fieldCheck string
	}{
		{"valid", request.TestIbkrConnectionRequest{FlexToken: validToken, FlexQueryID: "12345"}, false, ""},
		{"short token", request.TestIbkrConnectionRequest{FlexToken: "123", FlexQueryID: "12345"}, true, "flexToken"},
		{"non-numeric token", request.TestIbkrConnectionRequest{FlexToken: "abcdefghijklmnopqrstuvwx", FlexQueryID: "12345"}, true, "flexToken"},
		{"empty token", request.TestIbkrConnectionRequest{FlexToken: "", FlexQueryID: "12345"}, true, "flexToken"},
		{"empty query ID", request.TestIbkrConnectionRequest{FlexToken: validToken, FlexQueryID: ""}, true, "flexQueryId"},
		{"query ID too long", request.TestIbkrConnectionRequest{FlexToken: validToken, FlexQueryID: "12345678901"}, true, "flexQueryId"},
		{"non-numeric query ID", request.TestIbkrConnectionRequest{FlexToken: validToken, FlexQueryID: "abc"}, true, "flexQueryId"},
		{"both invalid", request.TestIbkrConnectionRequest{FlexToken: "short", FlexQueryID: ""}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTestConnection(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTestConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.fieldCheck != "" {
				var valErr *Error
				if errors.As(err, &valErr) {
					if _, ok := valErr.Fields[tt.fieldCheck]; !ok {
						t.Errorf("expected error on field %q, got fields: %v", tt.fieldCheck, valErr.Fields)
					}
				}
			}
		})
	}
}

func TestValidateAllocateTransaction(t *testing.T) {
	tests := []struct {
		name       string
		allocs     []request.AllocationEntry
		wantErr    bool
		fieldCheck string
	}{
		{
			"valid single 100%",
			[]request.AllocationEntry{{PortfolioID: testUUID, Percentage: 100.0}},
			false, "",
		},
		{
			"valid multiple summing to 100",
			[]request.AllocationEntry{
				{PortfolioID: testUUID, Percentage: 60.0},
				{PortfolioID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", Percentage: 40.0},
			},
			false, "",
		},
		{
			"empty allocations",
			[]request.AllocationEntry{},
			true, "allocations",
		},
		{
			"nil allocations",
			nil,
			true, "allocations",
		},
		{
			"not summing to 100",
			[]request.AllocationEntry{
				{PortfolioID: testUUID, Percentage: 50.0},
			},
			true, "allocations",
		},
		{
			"empty portfolio ID",
			[]request.AllocationEntry{
				{PortfolioID: "", Percentage: 100.0},
			},
			true, "allocations[0].portfolioId",
		},
		{
			"invalid UUID portfolio",
			[]request.AllocationEntry{
				{PortfolioID: "not-a-uuid", Percentage: 100.0},
			},
			true, "allocations[0].portfolioId",
		},
		{
			"zero percentage",
			[]request.AllocationEntry{
				{PortfolioID: testUUID, Percentage: 0.0},
			},
			true, "allocations[0].percentage",
		},
		{
			"negative percentage",
			[]request.AllocationEntry{
				{PortfolioID: testUUID, Percentage: -10.0},
			},
			true, "allocations[0].percentage",
		},
		{
			"close to 100 within tolerance",
			[]request.AllocationEntry{
				{PortfolioID: testUUID, Percentage: 99.995},
				{PortfolioID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", Percentage: 0.005},
			},
			false, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAllocateTransaction(tt.allocs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllocateTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.fieldCheck != "" {
				var valErr *Error
				if errors.As(err, &valErr) {
					if _, ok := valErr.Fields[tt.fieldCheck]; !ok {
						t.Errorf("expected error on field %q, got fields: %v", tt.fieldCheck, valErr.Fields)
					}
				}
			}
		})
	}
}

func TestValidateBulkAllocate(t *testing.T) {
	validUUID2 := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

	tests := []struct {
		name    string
		req     request.BulkAllocateRequest
		wantErr bool
	}{
		{
			"valid with allocations",
			request.BulkAllocateRequest{
				TransactionIDs: []string{testUUID},
				Allocations:    []request.AllocationEntry{{PortfolioID: validUUID2, Percentage: 100.0}},
			},
			false,
		},
		{
			"valid without allocations (auto-allocate)",
			request.BulkAllocateRequest{
				TransactionIDs: []string{testUUID},
			},
			false,
		},
		{
			"empty transaction IDs",
			request.BulkAllocateRequest{
				TransactionIDs: []string{},
			},
			true,
		},
		{
			"invalid transaction ID",
			request.BulkAllocateRequest{
				TransactionIDs: []string{"not-a-uuid"},
			},
			true,
		},
		{
			"valid IDs but invalid allocations",
			request.BulkAllocateRequest{
				TransactionIDs: []string{testUUID},
				Allocations:    []request.AllocationEntry{{PortfolioID: "", Percentage: 100.0}},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBulkAllocate(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBulkAllocate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMatchDividend(t *testing.T) {
	tests := []struct {
		name    string
		req     request.MatchDividendRequest
		wantErr bool
	}{
		{"valid single", request.MatchDividendRequest{DividendIDs: []string{testUUID}}, false},
		{"valid multiple", request.MatchDividendRequest{DividendIDs: []string{testUUID, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"}}, false},
		{"empty", request.MatchDividendRequest{DividendIDs: []string{}}, true},
		{"nil", request.MatchDividendRequest{}, true},
		{"invalid UUID", request.MatchDividendRequest{DividendIDs: []string{"bad"}}, true},
		{"mixed valid invalid", request.MatchDividendRequest{DividendIDs: []string{testUUID, "bad"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMatchDividend(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMatchDividend() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFlexToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"all digits", "1234567890", true},
		{"with letters", "123abc", false},
		{"empty", "", true},          // no characters to fail
		{"spaces only", "   ", true}, // validateFlexToken iterates runes; space is a digit-class pass-through
		{"with special chars", "123-456", false},
		{"long numeric", "12345678901234567890123456789012345678901234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateFlexToken(tt.token)
			if got != tt.want {
				t.Errorf("validateFlexToken(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}
