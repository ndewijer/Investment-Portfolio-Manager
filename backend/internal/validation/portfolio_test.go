package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

func TestValidateCreatePortfolio(t *testing.T) {
	tests := []struct {
		name       string
		req        request.CreatePortfolioRequest
		wantErr    bool
		fieldCheck string // field key to check in Error
	}{
		{"valid", request.CreatePortfolioRequest{Name: "My Portfolio"}, false, ""},
		{"valid with description", request.CreatePortfolioRequest{Name: "Test", Description: "A description"}, false, ""},
		{"empty name", request.CreatePortfolioRequest{Name: ""}, true, "name"},
		{"whitespace name", request.CreatePortfolioRequest{Name: "   "}, true, "name"},
		{"name too long", request.CreatePortfolioRequest{Name: strings.Repeat("a", 101)}, true, "name"},
		{"name exactly 100", request.CreatePortfolioRequest{Name: strings.Repeat("a", 100)}, false, ""},
		{"description too long", request.CreatePortfolioRequest{Name: "Valid", Description: strings.Repeat("a", 501)}, true, "description"},
		{"description exactly 500", request.CreatePortfolioRequest{Name: "Valid", Description: strings.Repeat("a", 500)}, false, ""},
		{"empty description ok", request.CreatePortfolioRequest{Name: "Valid", Description: ""}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreatePortfolio(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreatePortfolio() error = %v, wantErr %v", err, tt.wantErr)
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

func TestValidateUpdatePortfolio(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name       string
		req        request.UpdatePortfolioRequest
		wantErr    bool
		fieldCheck string
	}{
		{"all nil (no update)", request.UpdatePortfolioRequest{}, false, ""},
		{"valid name", request.UpdatePortfolioRequest{Name: strPtr("New Name")}, false, ""},
		{"empty name", request.UpdatePortfolioRequest{Name: strPtr("")}, true, "name"},
		{"whitespace name", request.UpdatePortfolioRequest{Name: strPtr("   ")}, true, "name"},
		{"name too long", request.UpdatePortfolioRequest{Name: strPtr(strings.Repeat("a", 101))}, true, "name"},
		{"name exactly 100", request.UpdatePortfolioRequest{Name: strPtr(strings.Repeat("a", 100))}, false, ""},
		{"description too long", request.UpdatePortfolioRequest{Description: strPtr(strings.Repeat("a", 501))}, true, "description"},
		{"description exactly 500", request.UpdatePortfolioRequest{Description: strPtr(strings.Repeat("a", 500))}, false, ""},
		{"empty description ok", request.UpdatePortfolioRequest{Description: strPtr("")}, false, ""},
		{"valid description", request.UpdatePortfolioRequest{Description: strPtr("Updated desc")}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdatePortfolio(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdatePortfolio() error = %v, wantErr %v", err, tt.wantErr)
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

func TestValidateCreatePortfolioFund(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name    string
		req     request.CreatePortfolioFundRequest
		wantErr bool
	}{
		{"valid", request.CreatePortfolioFundRequest{FundID: validUUID, PortfolioID: validUUID}, false},
		{"invalid fund ID", request.CreatePortfolioFundRequest{FundID: "bad", PortfolioID: validUUID}, true},
		{"invalid portfolio ID", request.CreatePortfolioFundRequest{FundID: validUUID, PortfolioID: "bad"}, true},
		{"both invalid", request.CreatePortfolioFundRequest{FundID: "bad", PortfolioID: "bad"}, true},
		{"empty fund ID", request.CreatePortfolioFundRequest{FundID: "", PortfolioID: validUUID}, true},
		{"empty portfolio ID", request.CreatePortfolioFundRequest{FundID: validUUID, PortfolioID: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreatePortfolioFund(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreatePortfolioFund() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
