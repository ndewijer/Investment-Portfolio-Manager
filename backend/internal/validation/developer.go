package validation

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// ValidLogLevels is an alias for model.ValidLogLevels for use within validation functions.
// The authoritative definition lives in model.ValidLogLevels.
var ValidLogLevels = model.ValidLogLevels

// ValidateUpdateExchangeRate validates a SetExchangeRateRequest.
// Returns a validation Error if date, fromCurrency, toCurrency, or rate is missing or invalid.
func ValidateUpdateExchangeRate(req request.SetExchangeRateRequest) error {
	errors := make(map[string]string)

	if strings.TrimSpace(req.Date) == "" {
		errors["date"] = "date is required"
	} else {
		_, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			errors["date"] = err.Error()
		}
	}

	if strings.TrimSpace(req.ToCurrency) == "" {
		errors["toCurrency"] = "to currency is required"
	}

	if strings.TrimSpace(req.FromCurrency) == "" {
		errors["fromCurrency"] = "from currency is required"
	}

	if strings.TrimSpace(req.Rate) == "" {
		errors["rate"] = "rate is required"
	} else {
		rateFloat, err := strconv.ParseFloat(strings.TrimSpace(req.Rate), 64)
		if err != nil {
			errors["rate"] = "rate not a valid number"
		} else if rateFloat <= 0 {
			errors["rate"] = "rate must be positive"
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil

}

// ValidateUpdateFundPrice validates a SetFundPriceRequest.
// Returns a validation Error if date, fundID, or price is missing or not a valid positive number.
func ValidateUpdateFundPrice(req request.SetFundPriceRequest) error {
	errors := make(map[string]string)

	if strings.TrimSpace(req.Date) == "" {
		errors["date"] = "date is required"
	} else {
		_, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			errors["date"] = err.Error()
		}
	}

	if strings.TrimSpace(req.FundID) == "" {
		errors["fundID"] = "fundID is required"
	}

	if strings.TrimSpace(req.Price) == "" {
		errors["price"] = "price is required"
	} else {
		priceFloat, err := strconv.ParseFloat(strings.TrimSpace(req.Price), 64)
		if err != nil {
			errors["price"] = "price not a valid number"
		} else if priceFloat <= 0 {
			errors["price"] = "price must be positive"
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil

}

// ValidateLoggingConfig validates a SetLoggingConfig request.
// Returns a validation Error if enabled is nil or level is empty/not in ValidLogLevels.
func ValidateLoggingConfig(req request.SetLoggingConfig) error {
	errors := make(map[string]string)

	if req.Enabled == nil {
		errors["enabled"] = "enabled is required"
	}

	if strings.TrimSpace(req.Level) == "" {
		errors["level"] = "level is required"
	} else if !ValidLogLevels[model.LogLevel(req.Level)] {
		errors["level"] = fmt.Sprintf("invalid level: %s", req.Level)
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil

}
