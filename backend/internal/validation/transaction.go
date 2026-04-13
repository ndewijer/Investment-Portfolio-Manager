package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// ValidTransactionType is an alias for model.ValidTransactionTypes for use within validation functions.
// The authoritative definition lives in model.ValidTransactionTypes.
var ValidTransactionType = model.ValidTransactionTypes

// ValidateCreateTransaction validates a transaction creation request.
// Checks all required fields and validates their formats and constraints.
//
// Required fields:
//   - portfolioFundId: Must be a valid UUID
//   - date: Must be in YYYY-MM-DD format
//   - type: Must be one of: buy, sell, dividend, fee
//   - shares: Must be non-zero
//   - costPerShare: Must be non-zero
//
// Returns a validation Error with field-specific error messages if validation fails.
func ValidateCreateTransaction(req request.CreateTransactionRequest) error {
	errors := make(map[string]string)

	portfolioFundErr := ValidateUUID(req.PortfolioFundID)
	if portfolioFundErr != nil {
		return portfolioFundErr
	}

	if strings.TrimSpace(req.Date) == "" {
		errors["date"] = "date is required"
	} else {
		_, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			errors["date"] = err.Error()
		}
	}

	if strings.TrimSpace(req.Type) == "" {
		errors["transactionType"] = "type is required"
	} else if !ValidTransactionType[model.TransactionType(req.Type)] {
		errors["transactionType"] = fmt.Sprintf("invalid type: %s", req.Type)
	}

	if req.Shares <= 0.0 {
		errors["shares"] = "shares must be positive"
	}

	if req.CostPerShare <= 0.0 {
		errors["costPerShare"] = "costPerShare must be positive"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil
}

// ValidateUpdateTransaction validates a transaction update request.
// All fields are optional, but if provided, they must meet the same constraints as create.
//
// Optional fields (validated if provided):
//   - portfolioFundId: Must be a valid UUID if provided
//   - date: Must be in YYYY-MM-DD format if provided
//   - type: Must be one of: buy, sell, dividend, fee if provided
//   - shares: Must be non-zero if provided
//   - costPerShare: Must be non-zero if provided
//
// Returns a validation Error with field-specific error messages if validation fails.
func ValidateUpdateTransaction(req request.UpdateTransactionRequest) error {
	errors := make(map[string]string)

	if req.PortfolioFundID != nil {
		portfolioFundErr := ValidateUUID(*req.PortfolioFundID)
		if portfolioFundErr != nil {
			return portfolioFundErr
		}
	}
	if req.Date != nil {
		if strings.TrimSpace(*req.Date) == "" {
			errors["date"] = "date is required"
		}
		_, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			errors["date"] = err.Error()
		}
	}
	if req.Type != nil {
		if strings.TrimSpace(*req.Type) == "" {
			errors["transactionType"] = "type is required"
		} else if !ValidTransactionType[model.TransactionType(*req.Type)] {
			errors["transactionType"] = fmt.Sprintf("invalid type: %s", *req.Type)
		}
	}
	if req.Shares != nil {
		if *req.Shares <= 0.0 {
			errors["shares"] = "shares must be positive"
		}
	}
	if req.CostPerShare != nil {
		if *req.CostPerShare <= 0.0 {
			errors["costPerShare"] = "costPerShare must be positive"
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil
}
