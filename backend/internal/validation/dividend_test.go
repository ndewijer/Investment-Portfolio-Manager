package validation

import (
	"errors"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

func validCreateDividendRequest() request.CreateDividendRequest {
	return request.CreateDividendRequest{
		PortfolioFundID:  testUUID,
		RecordDate:       "2024-01-15",
		ExDividendDate:   "2024-01-10",
		DividendPerShare: 1.50,
	}
}

func TestValidateCreateDividend(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*request.CreateDividendRequest)
		wantErr    bool
		fieldCheck string
		isUUIDErr  bool
	}{
		{"valid request", nil, false, "", false},
		{"valid with optionals", func(r *request.CreateDividendRequest) {
			r.BuyOrderDate = "2024-01-20"
			r.ReinvestmentShares = 5.0
			r.ReinvestmentPrice = 10.0
		}, false, "", false},
		{"invalid portfolio fund ID", func(r *request.CreateDividendRequest) { r.PortfolioFundID = "bad" }, true, "", true},
		{"empty portfolio fund ID", func(r *request.CreateDividendRequest) { r.PortfolioFundID = "" }, true, "", true},
		{"empty record date", func(r *request.CreateDividendRequest) { r.RecordDate = "" }, true, "recordDate", false},
		{"invalid record date", func(r *request.CreateDividendRequest) { r.RecordDate = "bad-date" }, true, "recordDate", false},
		{"empty ex-dividend date", func(r *request.CreateDividendRequest) { r.ExDividendDate = "" }, true, "exDividendDate", false},
		{"invalid ex-dividend date", func(r *request.CreateDividendRequest) { r.ExDividendDate = "bad" }, true, "exDividendDate", false},
		{"zero dividend per share", func(r *request.CreateDividendRequest) { r.DividendPerShare = 0.0 }, true, "dividendPerShare", false},
		{"negative dividend per share", func(r *request.CreateDividendRequest) { r.DividendPerShare = -1.0 }, true, "dividendPerShare", false},
		{"invalid buy order date", func(r *request.CreateDividendRequest) { r.BuyOrderDate = "not-a-date" }, true, "buyOrderDate", false},
		{"empty buy order date ok", func(r *request.CreateDividendRequest) { r.BuyOrderDate = "" }, false, "", false},
		{"negative reinvestment shares", func(r *request.CreateDividendRequest) { r.ReinvestmentShares = -1.0 }, true, "reinvestmentShares", false},
		{"zero reinvestment shares ok", func(r *request.CreateDividendRequest) { r.ReinvestmentShares = 0.0 }, false, "", false},
		{"negative reinvestment price", func(r *request.CreateDividendRequest) { r.ReinvestmentPrice = -1.0 }, true, "reinvestmentPrice", false},
		{"zero reinvestment price ok", func(r *request.CreateDividendRequest) { r.ReinvestmentPrice = 0.0 }, false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreateDividendRequest()
			if tt.modify != nil {
				tt.modify(&req)
			}
			err := ValidateCreateDividend(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateDividend() error = %v, wantErr %v", err, tt.wantErr)
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

func TestValidateUpdateDividend(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name       string
		req        request.UpdateDividendRequest
		wantErr    bool
		fieldCheck string
		isUUIDErr  bool
	}{
		{"all nil", request.UpdateDividendRequest{}, false, "", false},
		{"valid portfolio fund ID", request.UpdateDividendRequest{PortfolioFundID: strPtr(testUUID)}, false, "", false},
		{"invalid portfolio fund ID", request.UpdateDividendRequest{PortfolioFundID: strPtr("bad")}, true, "", true},
		{"valid record date", request.UpdateDividendRequest{RecordDate: strPtr("2024-06-15")}, false, "", false},
		{"empty record date", request.UpdateDividendRequest{RecordDate: strPtr("")}, true, "recordDate", false},
		{"invalid record date", request.UpdateDividendRequest{RecordDate: strPtr("bad")}, true, "recordDate", false},
		{"valid ex-dividend date", request.UpdateDividendRequest{ExDividendDate: strPtr("2024-06-10")}, false, "", false},
		{"empty ex-dividend date", request.UpdateDividendRequest{ExDividendDate: strPtr("")}, true, "exDividendDate", false},
		{"invalid ex-dividend date", request.UpdateDividendRequest{ExDividendDate: strPtr("bad")}, true, "exDividendDate", false},
		{"positive dividend per share", request.UpdateDividendRequest{DividendPerShare: floatPtr(2.0)}, false, "", false},
		{"zero dividend per share", request.UpdateDividendRequest{DividendPerShare: floatPtr(0.0)}, true, "dividendPerShare", false},
		{"negative dividend per share", request.UpdateDividendRequest{DividendPerShare: floatPtr(-1.0)}, true, "dividendPerShare", false},
		{"valid buy order date", request.UpdateDividendRequest{BuyOrderDate: strPtr("2024-06-20")}, false, "", false},
		{"empty buy order date", request.UpdateDividendRequest{BuyOrderDate: strPtr("")}, true, "buyOrderDate", false},
		{"invalid buy order date", request.UpdateDividendRequest{BuyOrderDate: strPtr("bad")}, true, "buyOrderDate", false},
		{"positive reinvestment shares", request.UpdateDividendRequest{ReinvestmentShares: floatPtr(5.0)}, false, "", false},
		{"zero reinvestment shares", request.UpdateDividendRequest{ReinvestmentShares: floatPtr(0.0)}, true, "reinvestmentShares", false},
		{"negative reinvestment shares", request.UpdateDividendRequest{ReinvestmentShares: floatPtr(-1.0)}, true, "reinvestmentShares", false},
		{"positive reinvestment price", request.UpdateDividendRequest{ReinvestmentPrice: floatPtr(10.0)}, false, "", false},
		{"zero reinvestment price", request.UpdateDividendRequest{ReinvestmentPrice: floatPtr(0.0)}, true, "reinvestmentPrice", false},
		{"negative reinvestment price", request.UpdateDividendRequest{ReinvestmentPrice: floatPtr(-1.0)}, true, "reinvestmentPrice", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateDividend(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateDividend() error = %v, wantErr %v", err, tt.wantErr)
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
