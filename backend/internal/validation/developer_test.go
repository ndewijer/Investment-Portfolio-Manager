package validation

import (
	"errors"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

func TestValidateUpdateExchangeRate(t *testing.T) {
	tests := []struct {
		name       string
		req        request.SetExchangeRateRequest
		wantErr    bool
		fieldCheck string
	}{
		{
			"valid",
			request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "EUR", Rate: "1.05"},
			false, "",
		},
		{"empty date", request.SetExchangeRateRequest{Date: "", FromCurrency: "USD", ToCurrency: "EUR", Rate: "1.05"}, true, "date"},
		{"invalid date", request.SetExchangeRateRequest{Date: "bad", FromCurrency: "USD", ToCurrency: "EUR", Rate: "1.05"}, true, "date"},
		{"empty from currency", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "", ToCurrency: "EUR", Rate: "1.05"}, true, "fromCurrency"},
		{"whitespace from currency", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "  ", ToCurrency: "EUR", Rate: "1.05"}, true, "fromCurrency"},
		{"empty to currency", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "", Rate: "1.05"}, true, "toCurrency"},
		{"whitespace to currency", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "  ", Rate: "1.05"}, true, "toCurrency"},
		{"empty rate", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "EUR", Rate: ""}, true, "rate"},
		{"non-numeric rate", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "EUR", Rate: "abc"}, true, "rate"},
		{"zero rate", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "EUR", Rate: "0"}, true, "rate"},
		{"negative rate", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "EUR", Rate: "-1.5"}, true, "rate"},
		{"valid small rate", request.SetExchangeRateRequest{Date: "2024-01-15", FromCurrency: "USD", ToCurrency: "EUR", Rate: "0.001"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateExchangeRate(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateExchangeRate() error = %v, wantErr %v", err, tt.wantErr)
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

func TestValidateUpdateExchangeRate_MultipleErrors(t *testing.T) {
	req := request.SetExchangeRateRequest{} // all empty
	err := ValidateUpdateExchangeRate(req)
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *Error
	if !errors.As(err, &valErr) {
		t.Fatal("expected *Error")
	}
	for _, f := range []string{"date", "fromCurrency", "toCurrency", "rate"} {
		if _, ok := valErr.Fields[f]; !ok {
			t.Errorf("expected error on field %q", f)
		}
	}
}

func TestValidateUpdateFundPrice(t *testing.T) {
	tests := []struct {
		name       string
		req        request.SetFundPriceRequest
		wantErr    bool
		fieldCheck string
	}{
		{"valid", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "some-fund", Price: "150.50"}, false, ""},
		{"empty date", request.SetFundPriceRequest{Date: "", FundID: "fund", Price: "100"}, true, "date"},
		{"invalid date", request.SetFundPriceRequest{Date: "bad", FundID: "fund", Price: "100"}, true, "date"},
		{"empty fund ID", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "", Price: "100"}, true, "fundID"},
		{"whitespace fund ID", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "  ", Price: "100"}, true, "fundID"},
		{"empty price", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "fund", Price: ""}, true, "price"},
		{"non-numeric price", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "fund", Price: "abc"}, true, "price"},
		{"zero price", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "fund", Price: "0"}, true, "price"},
		{"negative price", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "fund", Price: "-10"}, true, "price"},
		{"small positive price", request.SetFundPriceRequest{Date: "2024-01-15", FundID: "fund", Price: "0.01"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateFundPrice(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateFundPrice() error = %v, wantErr %v", err, tt.wantErr)
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

func TestValidateLoggingConfig(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name       string
		req        request.SetLoggingConfig
		wantErr    bool
		fieldCheck string
	}{
		{"valid enabled debug", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "debug"}, false, ""},
		{"valid enabled info", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "info"}, false, ""},
		{"valid enabled warning", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "warning"}, false, ""},
		{"valid enabled error", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "error"}, false, ""},
		{"valid enabled critical", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "critical"}, false, ""},
		{"valid disabled", request.SetLoggingConfig{Enabled: boolPtr(false), Level: "info"}, false, ""},
		{"nil enabled", request.SetLoggingConfig{Enabled: nil, Level: "info"}, true, "enabled"},
		{"empty level", request.SetLoggingConfig{Enabled: boolPtr(true), Level: ""}, true, "level"},
		{"whitespace level", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "   "}, true, "level"},
		{"invalid level", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "trace"}, true, "level"},
		{"uppercase level", request.SetLoggingConfig{Enabled: boolPtr(true), Level: "DEBUG"}, true, "level"},
		{"nil enabled and empty level", request.SetLoggingConfig{Enabled: nil, Level: ""}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoggingConfig(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLoggingConfig() error = %v, wantErr %v", err, tt.wantErr)
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
