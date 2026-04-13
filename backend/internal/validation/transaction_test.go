package validation

import (
	"errors"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

const testUUID = "550e8400-e29b-41d4-a716-446655440000"

func validCreateTransactionRequest() request.CreateTransactionRequest {
	return request.CreateTransactionRequest{
		PortfolioFundID: testUUID,
		Date:            "2024-01-15",
		Type:            "buy",
		Shares:          10.0,
		CostPerShare:    50.0,
	}
}

func TestValidateCreateTransaction(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*request.CreateTransactionRequest)
		wantErr    bool
		fieldCheck string
		isUUIDErr  bool
	}{
		{"valid request", nil, false, "", false},
		{"invalid portfolio fund ID", func(r *request.CreateTransactionRequest) { r.PortfolioFundID = "bad" }, true, "", true},
		{"empty portfolio fund ID", func(r *request.CreateTransactionRequest) { r.PortfolioFundID = "" }, true, "", true},
		{"empty date", func(r *request.CreateTransactionRequest) { r.Date = "" }, true, "date", false},
		{"whitespace date", func(r *request.CreateTransactionRequest) { r.Date = "   " }, true, "date", false},
		{"invalid date format", func(r *request.CreateTransactionRequest) { r.Date = "01-15-2024" }, true, "date", false},
		{"valid date", func(r *request.CreateTransactionRequest) { r.Date = "2024-12-31" }, false, "", false},
		{"empty type", func(r *request.CreateTransactionRequest) { r.Type = "" }, true, "transactionType", false},
		{"invalid type", func(r *request.CreateTransactionRequest) { r.Type = "trade" }, true, "transactionType", false},
		{"buy type", func(r *request.CreateTransactionRequest) { r.Type = "buy" }, false, "", false},
		{"sell type", func(r *request.CreateTransactionRequest) { r.Type = "sell" }, false, "", false},
		{"dividend type", func(r *request.CreateTransactionRequest) { r.Type = "dividend" }, false, "", false},
		{"fee type", func(r *request.CreateTransactionRequest) { r.Type = "fee" }, false, "", false},
		{"zero shares", func(r *request.CreateTransactionRequest) { r.Shares = 0.0 }, true, "shares", false},
		{"negative shares", func(r *request.CreateTransactionRequest) { r.Shares = -1.0 }, true, "shares", false},
		{"positive shares", func(r *request.CreateTransactionRequest) { r.Shares = 0.001 }, false, "", false},
		{"zero cost", func(r *request.CreateTransactionRequest) { r.CostPerShare = 0.0 }, true, "costPerShare", false},
		{"negative cost", func(r *request.CreateTransactionRequest) { r.CostPerShare = -5.0 }, true, "costPerShare", false},
		{"positive cost", func(r *request.CreateTransactionRequest) { r.CostPerShare = 0.01 }, false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreateTransactionRequest()
			if tt.modify != nil {
				tt.modify(&req)
			}
			err := ValidateCreateTransaction(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.isUUIDErr && err != nil {
				if !errors.Is(err, ErrInvalidUUID) {
					t.Errorf("expected ErrInvalidUUID, got %v", err)
				}
			}
			if tt.fieldCheck != "" && err != nil {
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

func TestValidateUpdateTransaction(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name       string
		req        request.UpdateTransactionRequest
		wantErr    bool
		fieldCheck string
		isUUIDErr  bool
	}{
		{"all nil", request.UpdateTransactionRequest{}, false, "", false},
		{"valid portfolio fund ID", request.UpdateTransactionRequest{PortfolioFundID: strPtr(testUUID)}, false, "", false},
		{"invalid portfolio fund ID", request.UpdateTransactionRequest{PortfolioFundID: strPtr("bad")}, true, "", true},
		{"valid date", request.UpdateTransactionRequest{Date: strPtr("2024-06-15")}, false, "", false},
		{"empty date", request.UpdateTransactionRequest{Date: strPtr("")}, true, "date", false},
		{"invalid date", request.UpdateTransactionRequest{Date: strPtr("not-a-date")}, true, "date", false},
		{"valid type", request.UpdateTransactionRequest{Type: strPtr("sell")}, false, "", false},
		{"empty type", request.UpdateTransactionRequest{Type: strPtr("")}, true, "transactionType", false},
		{"invalid type", request.UpdateTransactionRequest{Type: strPtr("swap")}, true, "transactionType", false},
		{"positive shares", request.UpdateTransactionRequest{Shares: floatPtr(5.0)}, false, "", false},
		{"zero shares", request.UpdateTransactionRequest{Shares: floatPtr(0.0)}, true, "shares", false},
		{"negative shares", request.UpdateTransactionRequest{Shares: floatPtr(-1.0)}, true, "shares", false},
		{"positive cost", request.UpdateTransactionRequest{CostPerShare: floatPtr(10.0)}, false, "", false},
		{"zero cost", request.UpdateTransactionRequest{CostPerShare: floatPtr(0.0)}, true, "costPerShare", false},
		{"negative cost", request.UpdateTransactionRequest{CostPerShare: floatPtr(-5.0)}, true, "costPerShare", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateTransaction(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.isUUIDErr && err != nil {
				if !errors.Is(err, ErrInvalidUUID) {
					t.Errorf("expected ErrInvalidUUID, got %v", err)
				}
			}
			if tt.fieldCheck != "" && err != nil {
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
