package validation

import (
	"fmt"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

// ValidDividendType contains the allowed dividend type values for funds.
var ValidDividendType = map[string]bool{
	"CASH": true, "STOCK": true, "NONE": true,
}

// ValidInvestmentType contains the allowed investment type values for funds.
var ValidInvestmentType = map[string]bool{
	"FUND": true, "STOCK": true,
}

// ValidateCreateFund validates a fund creation request.
// Checks all required fields and validates their formats and constraints.
//
// Required fields:
//   - name: Must be non-empty and max 100 characters
//   - isin: Must be non-empty and match format: 2 letters + 9 alphanumeric + 1 digit
//   - currency: Must be non-empty and max 3 characters (e.g., USD, EUR)
//   - exchange: Must be non-empty and max 15 characters (e.g., NYSE, AMS)
//   - dividend_type: Must be one of: CASH, STOCK, NONE
//   - investment_type: Must be one of: FUND, STOCK
//
// Optional fields:
//   - symbol: Max 10 characters if provided
//
// Returns a validation Error with field-specific error messages if validation fails.
//
//nolint:gocyclo // Comprehensive validation of fund creation, cannot be split well.
func ValidateCreateFund(req request.CreateFundRequest) error {
	errors := make(map[string]string)

	// Required field
	if strings.TrimSpace(req.Name) == "" {
		errors["name"] = "name is required"
	} else if len(req.Name) > 100 {
		errors["name"] = "name must be 100 characters or less"
	}
	if strings.TrimSpace(req.Isin) == "" {
		errors["isin"] = "isin is required"
	}
	if !validateISIN(req.Isin) {
		errors["isin"] = "isin structure is not correct"
	}

	if strings.TrimSpace(req.Currency) == "" {
		errors["currency"] = "currency is required"
	} else if len(req.Currency) > 3 {
		errors["currency"] = "currency must be 3 characters or less (USD, EUR)"
	}

	if strings.TrimSpace(req.Exchange) == "" {
		errors["exchange"] = "exchange is required"
	} else if len(req.Exchange) > 15 {
		errors["exchange"] = "exchange must be 15 characters or less (NYSE, AMS)"
	}

	if strings.TrimSpace(req.DividendType) == "" {
		errors["dividendType"] = "dividend type is required"
	} else if !ValidDividendType[req.DividendType] {
		errors["dividendType"] = fmt.Sprintf("invalid dividend type: %s", req.DividendType)
	}

	if strings.TrimSpace(req.InvestmentType) == "" {
		errors["investmentType"] = "investment type is required"
	} else if !ValidInvestmentType[req.InvestmentType] {
		errors["investmentType"] = fmt.Sprintf("invalid investment type: %s", req.InvestmentType)
	}

	// optional
	if len(req.Symbol) > 10 {
		errors["symbol"] = "symbol must be 10 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

// ValidateUpdateFund validates a fund update request.
// All fields are optional, but if provided, they must meet the same constraints as create.
//
// Optional fields (validated if provided):
//   - name: Max 100 characters if provided
//   - isin: Must match format: 2 letters + 9 alphanumeric + 1 digit if provided
//   - currency: Max 3 characters if provided (e.g., USD, EUR)
//   - exchange: Max 15 characters if provided (e.g., NYSE, AMS)
//   - dividend_type: Must be one of: CASH, STOCK, NONE if provided
//   - investment_type: Must be one of: FUND, STOCK if provided
//   - symbol: Max 10 characters if provided
//
// Returns a validation Error with field-specific error messages if validation fails.
//
//nolint:gocyclo // Comprehensive validation of fund updates, cannot be split well.
func ValidateUpdateFund(req request.UpdateFundRequest) error {
	errors := make(map[string]string)

	// Required field
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errors["name"] = "name is required"
		} else if len(*req.Name) > 100 {
			errors["name"] = "name must be 100 characters or less"
		}
	}
	if req.Isin != nil {
		if strings.TrimSpace(*req.Isin) == "" {
			errors["isin"] = "isin is required"
		}

		if !validateISIN(*req.Isin) {
			errors["isin"] = "isin structure is not correct"
		}
	}
	if req.Currency != nil {
		if strings.TrimSpace(*req.Currency) == "" {
			errors["currency"] = "currency is required"
		} else if len(*req.Currency) > 3 {
			errors["currency"] = "currency must be 3 characters or less (USD, EUR)"
		}
	}
	if req.Exchange != nil {
		if strings.TrimSpace(*req.Exchange) == "" {
			errors["exchange"] = "exchange is required"
		} else if len(*req.Exchange) > 15 {
			errors["exchange"] = "exchange must be 15 characters or less (NYSE, AMS)"
		}
	}
	if req.DividendType != nil {
		if strings.TrimSpace(*req.DividendType) == "" {
			errors["dividendType"] = "dividend type is required"
		} else if !ValidDividendType[*req.DividendType] {
			errors["dividendType"] = fmt.Sprintf("invalid dividend type: %s", *req.DividendType)
		}
	}
	if req.InvestmentType != nil {
		if strings.TrimSpace(*req.InvestmentType) == "" {
			errors["investmentType"] = "investment type is required"
		} else if !ValidInvestmentType[*req.InvestmentType] {
			errors["investmentType"] = fmt.Sprintf("invalid investment type: %s", *req.InvestmentType)
		}
	}
	// optional
	if req.Symbol != nil && len(*req.Symbol) > 10 {
		errors["symbol"] = "symbol must be 10 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}
