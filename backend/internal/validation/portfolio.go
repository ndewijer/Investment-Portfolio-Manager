package validation

import (
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

// ValidateCreatePortfolio validates a CreatePortfolioRequest.
func ValidateCreatePortfolio(req request.CreatePortfolioRequest) error {
	errors := make(map[string]string)

	// Required field
	if strings.TrimSpace(req.Name) == "" {
		errors["name"] = "name is required"
	} else if len(req.Name) > 100 {
		errors["name"] = "name must be 100 characters or less"
	}

	// Optional but has constraints
	if len(req.Description) > 500 {
		errors["description"] = "description must be 500 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

// ValidateUpdatePortfolio validates an UpdatePortfolioRequest.
func ValidateUpdatePortfolio(req request.UpdatePortfolioRequest) error {
	errors := make(map[string]string)

	// Only validate provided fields
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errors["name"] = "name cannot be empty"
		} else if len(*req.Name) > 100 {
			errors["name"] = "name must be 100 characters or less"
		}
	}

	if req.Description != nil && len(*req.Description) > 500 {
		errors["description"] = "description must be 500 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

// ValidateCreatePortfolioFund validates a CreatePortfolioFundRequest.
func ValidateCreatePortfolioFund(req request.CreatePortfolioFundRequest) error {
	fundErr := ValidateUUID(req.FundID)
	if fundErr != nil {
		return fundErr
	}
	portfolioErr := ValidateUUID(req.PortfolioID)
	if portfolioErr != nil {
		return portfolioErr
	}
	return nil
}
