package validation

import (
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

// ValidateCreateDividend validates a dividend creation request.
// Checks all required fields and validates their formats and constraints.
//
// Required fields:
//   - portfolioFundId: Must be a valid UUID
//   - recordDate: Must be in YYYY-MM-DD format
//   - exDividendDate: Must be in YYYY-MM-DD format
//   - dividendPerShare: Must be positive
//
// Optional fields (validated if provided):
//   - buyOrderDate: Must be in YYYY-MM-DD format
//   - reinvestmentShares: Must be positive
//   - reinvestmentPrice: Must be positive
//
// Returns a validation Error with field-specific error messages if validation fails.
func ValidateCreateDividend(req request.CreateDividendRequest) error {
	errors := make(map[string]string)

	portfolioFundErr := ValidateUUID(req.PortfolioFundID)
	if portfolioFundErr != nil {
		return portfolioFundErr
	}

	if strings.TrimSpace(req.RecordDate) == "" {
		errors["recordDate"] = "date is required"
	} else {
		_, err := time.Parse("2006-01-02", req.RecordDate)
		if err != nil {
			errors["recordDate"] = err.Error()
		}
	}

	if strings.TrimSpace(req.ExDividendDate) == "" {
		errors["exDividendDate"] = "date is required"
	} else {
		_, err := time.Parse("2006-01-02", req.ExDividendDate)
		if err != nil {
			errors["exDividendDate"] = err.Error()
		}
	}

	if req.DividendPerShare <= 0.0 {
		errors["dividendPerShare"] = "dividendPerShare must be positive"
	}

	// optionals

	if req.BuyOrderDate != "" {
		_, err := time.Parse("2006-01-02", req.BuyOrderDate)
		if err != nil {
			errors["buyOrderDate"] = err.Error()
		}
	}

	if req.ReinvestmentShares < 0.0 {
		errors["reinvestmentShares"] = "reinvestmentShares must be positive"
	}

	if req.ReinvestmentPrice < 0.0 {
		errors["reinvestmentPrice"] = "reinvestmentPrice must be positive"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil
}

// ValidateUpdateDividend validates a dividend update request.
// All fields are optional, but if provided, they must meet the same constraints as create.
//
// Optional fields (validated if provided):
//   - portfolioFundId: Must be a valid UUID if provided
//   - recordDate: Must be in YYYY-MM-DD format if provided
//   - exDividendDate: Must be in YYYY-MM-DD format if provided
//   - dividendPerShare: Must be positive if provided
//   - buyOrderDate: Must be in YYYY-MM-DD format if provided
//   - reinvestmentShares: Must be positive if provided
//   - reinvestmentPrice: Must be positive if provided
//
// Returns a validation Error with field-specific error messages if validation fails.
//
//nolint:gocyclo // Comprehensive validation of dividend updates, cannot be split well.
func ValidateUpdateDividend(req request.UpdateDividendRequest) error {
	errors := make(map[string]string)

	if req.PortfolioFundID != nil {
		portfolioFundErr := ValidateUUID(*req.PortfolioFundID)
		if portfolioFundErr != nil {
			return portfolioFundErr
		}
	}
	if req.RecordDate != nil {
		if strings.TrimSpace(*req.RecordDate) == "" {
			errors["recordDate"] = "recordDate is required"
		}
		_, err := time.Parse("2006-01-02", *req.RecordDate)
		if err != nil {
			errors["recordDate"] = err.Error()
		}
	}

	if req.ExDividendDate != nil {
		if strings.TrimSpace(*req.ExDividendDate) == "" {
			errors["exDividendDate"] = "exDividendDate is required"
		}
		_, err := time.Parse("2006-01-02", *req.ExDividendDate)
		if err != nil {
			errors["exDividendDate"] = err.Error()
		}
	}

	if req.DividendPerShare != nil {
		if *req.DividendPerShare <= 0.0 {
			errors["dividendPerShare"] = "dividendPerShare must be positive"
		}
	}

	if req.BuyOrderDate != nil {
		if strings.TrimSpace(*req.BuyOrderDate) == "" {
			errors["buyOrderDate"] = "buyOrderDate is required"
		}
		_, err := time.Parse("2006-01-02", *req.BuyOrderDate)
		if err != nil {
			errors["buyOrderDate"] = err.Error()
		}
	}

	if req.ReinvestmentShares != nil {
		if *req.ReinvestmentShares <= 0.0 {
			errors["reinvestmentShares"] = "reinvestmentShares must be positive"
		}
	}

	if req.ReinvestmentPrice != nil {
		if *req.ReinvestmentPrice <= 0.0 {
			errors["reinvestmentPrice"] = "reinvestmentPrice must be positive"
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil
}
