package validation

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
)

// ValidateUpdateIbkrConfig validates an UpdateIbkrConfigRequest.
//
//nolint:gocyclo // Comprehensive validation of ibkr config updates, cannot be split well.
func ValidateUpdateIbkrConfig(req request.UpdateIbkrConfigRequest) error {
	errors := make(map[string]string)

	// Required field
	if req.Enabled != nil && *req.Enabled {
		if req.FlexToken != nil {
			if strings.TrimSpace(*req.FlexToken) != "" {
				if len(*req.FlexToken) < 24 {
					errors["flexToken"] = "flexToken must be at least 24 characters"
				} else {
					if !validateFlexToken(*req.FlexToken) {
						errors["flexToken"] = "flexToken must be a number"
					}
				}
			}
		}

		if req.FlexQueryID != nil {
			if *req.FlexQueryID == "" {
				errors["flexQueryId"] = "flexQueryId must be set"
			} else if len(*req.FlexQueryID) > 10 {
				errors["flexQueryId"] = "flexQueryId must be 10 characters or less"
			} else if _, err := strconv.Atoi(strings.TrimSpace(*req.FlexQueryID)); err != nil {
				errors["flexQueryId"] = "flexQueryId must be a number"
			}

		}

		if req.TokenExpiresAt != nil {
			time, err := ParseTime(*req.TokenExpiresAt)
			if err != nil {
				errors["tokenExpiresAt"] = fmt.Sprintf("tokenExpiresAt cannot be parsed: %v", err)
			} else if time.IsZero() {
				errors["tokenExpiresAt"] = "tokenExpiresAt must be set"
			}
		}

		if req.DefaultAllocationEnabled != nil && *req.DefaultAllocationEnabled {
			if len(req.DefaultAllocations) == 0 {
				errors["defaultAllocations"] = "defaultAllocations should have at least 1 portfolio when enabled"
			} else {
				var perc float64
				for _, v := range req.DefaultAllocations {
					perc += *v.Percentage
					if v.Percentage == nil || *v.Percentage == 0.0 {
						errors["defaultAllocations"] = "allocation percentage must be set"
					} else if v.PortfolioID == nil || *v.PortfolioID == "" {
						errors["defaultAllocations"] = "portfolio must be set"
					}
				}
				if math.Abs(perc-100) > 0.01 {
					errors["defaultAllocations"] = "defaultAllocations do not add up to 100%"
				}
			}
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

// ValidateTestConnection validates the fields of a TestIbkrConnectionRequest.
// Both flexToken and flexQueryId are required: the token must be at least 24 digits,
// and the queryId must be a numeric string of at most 10 characters.
func ValidateTestConnection(req request.TestIbkrConnectionRequest) error {
	errors := make(map[string]string)

	if len(req.FlexToken) < 24 {
		errors["flexToken"] = "flexToken must be at least 24 characters"
	} else {
		if !validateFlexToken(req.FlexToken) {
			errors["flexToken"] = "flexToken must be a number"
		}

	}

	if req.FlexQueryID == "" {
		errors["flexQueryId"] = "flexQueryId must be set"
	} else if len(req.FlexQueryID) > 10 {
		errors["flexQueryId"] = "flexQueryId must be 10 characters or less"
	} else if _, err := strconv.Atoi(strings.TrimSpace(req.FlexQueryID)); err != nil {
		errors["flexQueryId"] = "flexQueryId must be a number"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

// ValidateAllocateTransaction validates an allocation request.
// Ensures percentages sum to 100 (±0.01), each entry has a valid UUID and positive percentage.
func ValidateAllocateTransaction(allocations []request.AllocationEntry) error {
	errors := make(map[string]string)

	if len(allocations) == 0 {
		errors["allocations"] = "at least one allocation is required"
		return &Error{Fields: errors}
	}

	var total float64
	for i, a := range allocations {
		if a.PortfolioID == "" {
			errors[fmt.Sprintf("allocations[%d].portfolioId", i)] = "portfolioId is required"
		} else if err := ValidateUUID(a.PortfolioID); err != nil {
			errors[fmt.Sprintf("allocations[%d].portfolioId", i)] = "invalid UUID format"
		}

		if a.Percentage <= 0 {
			errors[fmt.Sprintf("allocations[%d].percentage", i)] = "percentage must be positive"
		}
		total += a.Percentage
	}

	if len(errors) == 0 && math.Abs(total-100) > 0.01 {
		errors["allocations"] = "allocations must sum to 100%"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

// ValidateBulkAllocate validates a bulk allocation request.
// Requires at least one transaction ID and valid allocation entries.
func ValidateBulkAllocate(req request.BulkAllocateRequest) error {
	errors := make(map[string]string)

	if len(req.TransactionIDs) == 0 {
		errors["transactionIds"] = "at least one transaction ID is required"
	}

	for i, id := range req.TransactionIDs {
		if err := ValidateUUID(id); err != nil {
			errors[fmt.Sprintf("transactionIds[%d]", i)] = "invalid UUID format"
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	// Allocations are optional for bulk (auto-allocate may apply)
	if len(req.Allocations) > 0 {
		return ValidateAllocateTransaction(req.Allocations)
	}

	return nil
}

// ValidateMatchDividend validates a match-dividend request.
// Requires at least one dividend ID, each being a valid UUID.
func ValidateMatchDividend(req request.MatchDividendRequest) error {
	errors := make(map[string]string)

	if len(req.DividendIDs) == 0 {
		errors["dividendIds"] = "at least one dividend ID is required"
	}

	for i, id := range req.DividendIDs {
		if err := ValidateUUID(id); err != nil {
			errors[fmt.Sprintf("dividendIds[%d]", i)] = "invalid UUID format"
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

// validateFlexToken returns true if flexToken consists entirely of ASCII digits.
// strconv.Atoi is not used here: a 19-digit number overflows int64. The token is 23 or larger.
// Validate digit-by-digit instead.
func validateFlexToken(flexToken string) bool {
	for _, c := range strings.TrimSpace(flexToken) {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
