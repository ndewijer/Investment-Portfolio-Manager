package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

func validCreateFundRequest() request.CreateFundRequest {
	return request.CreateFundRequest{
		Name:           "Test Fund",
		Isin:           "US0378331005",
		Symbol:         "AAPL",
		Currency:       "USD",
		Exchange:       "NYSE",
		DividendType:   "CASH",
		InvestmentType: "FUND",
	}
}

func TestValidateCreateFund(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*request.CreateFundRequest)
		wantErr    bool
		fieldCheck string
	}{
		{"valid request", nil, false, ""},
		{"empty name", func(r *request.CreateFundRequest) { r.Name = "" }, true, "name"},
		{"whitespace name", func(r *request.CreateFundRequest) { r.Name = "   " }, true, "name"},
		{"name too long", func(r *request.CreateFundRequest) { r.Name = strings.Repeat("a", 101) }, true, "name"},
		{"name exactly 100", func(r *request.CreateFundRequest) { r.Name = strings.Repeat("a", 100) }, false, ""},
		{"empty isin", func(r *request.CreateFundRequest) { r.Isin = "" }, true, "isin"},
		{"invalid isin checksum", func(r *request.CreateFundRequest) { r.Isin = "US0378331006" }, true, "isin"},
		{"invalid isin format", func(r *request.CreateFundRequest) { r.Isin = "TOOLONG12345" }, true, "isin"},
		{"empty currency", func(r *request.CreateFundRequest) { r.Currency = "" }, true, "currency"},
		{"currency too long", func(r *request.CreateFundRequest) { r.Currency = "USDX" }, true, "currency"},
		{"currency exactly 3", func(r *request.CreateFundRequest) { r.Currency = "EUR" }, false, ""},
		{"empty exchange", func(r *request.CreateFundRequest) { r.Exchange = "" }, true, "exchange"},
		{"exchange too long", func(r *request.CreateFundRequest) { r.Exchange = strings.Repeat("A", 16) }, true, "exchange"},
		{"exchange exactly 15", func(r *request.CreateFundRequest) { r.Exchange = strings.Repeat("A", 15) }, false, ""},
		{"empty dividend type", func(r *request.CreateFundRequest) { r.DividendType = "" }, true, "dividendType"},
		{"invalid dividend type", func(r *request.CreateFundRequest) { r.DividendType = "INVALID" }, true, "dividendType"},
		{"STOCK dividend type", func(r *request.CreateFundRequest) { r.DividendType = "STOCK" }, false, ""},
		{"NONE dividend type", func(r *request.CreateFundRequest) { r.DividendType = "NONE" }, false, ""},
		{"empty investment type", func(r *request.CreateFundRequest) { r.InvestmentType = "" }, true, "investmentType"},
		{"invalid investment type", func(r *request.CreateFundRequest) { r.InvestmentType = "BOND" }, true, "investmentType"},
		{"STOCK investment type", func(r *request.CreateFundRequest) { r.InvestmentType = "STOCK" }, false, ""},
		{"symbol too long", func(r *request.CreateFundRequest) { r.Symbol = strings.Repeat("A", 11) }, true, "symbol"},
		{"symbol exactly 10", func(r *request.CreateFundRequest) { r.Symbol = strings.Repeat("A", 10) }, false, ""},
		{"empty symbol ok", func(r *request.CreateFundRequest) { r.Symbol = "" }, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreateFundRequest()
			if tt.modify != nil {
				tt.modify(&req)
			}
			err := ValidateCreateFund(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateFund() error = %v, wantErr %v", err, tt.wantErr)
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

func TestValidateCreateFund_MultipleErrors(t *testing.T) {
	req := request.CreateFundRequest{} // all empty
	err := ValidateCreateFund(req)
	if err == nil {
		t.Fatal("expected error for empty request")
	}
	var valErr *Error
	if !errors.As(err, &valErr) {
		t.Fatal("expected *Error type")
	}
	// Should have errors for name, isin, currency, exchange, dividendType, investmentType
	expectedFields := []string{"name", "isin", "currency", "exchange", "dividendType", "investmentType"}
	for _, f := range expectedFields {
		if _, ok := valErr.Fields[f]; !ok {
			t.Errorf("expected error on field %q", f)
		}
	}
}

func TestValidateUpdateFund(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name       string
		req        request.UpdateFundRequest
		wantErr    bool
		fieldCheck string
	}{
		{"all nil", request.UpdateFundRequest{}, false, ""},
		{"valid name", request.UpdateFundRequest{Name: strPtr("New Fund")}, false, ""},
		{"empty name", request.UpdateFundRequest{Name: strPtr("")}, true, "name"},
		{"whitespace name", request.UpdateFundRequest{Name: strPtr("   ")}, true, "name"},
		{"name too long", request.UpdateFundRequest{Name: strPtr(strings.Repeat("a", 101))}, true, "name"},
		{"valid isin", request.UpdateFundRequest{Isin: strPtr("US0378331005")}, false, ""},
		{"empty isin", request.UpdateFundRequest{Isin: strPtr("")}, true, "isin"},
		{"invalid isin", request.UpdateFundRequest{Isin: strPtr("INVALID")}, true, "isin"},
		{"valid currency", request.UpdateFundRequest{Currency: strPtr("EUR")}, false, ""},
		{"empty currency", request.UpdateFundRequest{Currency: strPtr("")}, true, "currency"},
		{"currency too long", request.UpdateFundRequest{Currency: strPtr("EURO")}, true, "currency"},
		{"valid exchange", request.UpdateFundRequest{Exchange: strPtr("AMS")}, false, ""},
		{"empty exchange", request.UpdateFundRequest{Exchange: strPtr("")}, true, "exchange"},
		{"exchange too long", request.UpdateFundRequest{Exchange: strPtr(strings.Repeat("X", 16))}, true, "exchange"},
		{"valid dividend type", request.UpdateFundRequest{DividendType: strPtr("CASH")}, false, ""},
		{"empty dividend type", request.UpdateFundRequest{DividendType: strPtr("")}, true, "dividendType"},
		{"invalid dividend type", request.UpdateFundRequest{DividendType: strPtr("BAD")}, true, "dividendType"},
		{"valid investment type", request.UpdateFundRequest{InvestmentType: strPtr("STOCK")}, false, ""},
		{"empty investment type", request.UpdateFundRequest{InvestmentType: strPtr("")}, true, "investmentType"},
		{"invalid investment type", request.UpdateFundRequest{InvestmentType: strPtr("BOND")}, true, "investmentType"},
		{"symbol too long", request.UpdateFundRequest{Symbol: strPtr(strings.Repeat("X", 11))}, true, "symbol"},
		{"valid symbol", request.UpdateFundRequest{Symbol: strPtr("AAPL")}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateFund(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateFund() error = %v, wantErr %v", err, tt.wantErr)
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
