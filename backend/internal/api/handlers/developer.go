package handlers

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/validation"
)

var devLog = logging.NewLogger("developer")

// DeveloperHandler handles HTTP requests for Developer endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the DeveloperService.
type DeveloperHandler struct {
	DeveloperService *service.DeveloperService
}

// NewDeveloperHandler creates a new DeveloperHandler with the provided service dependency.
func NewDeveloperHandler(DeveloperService *service.DeveloperService) *DeveloperHandler {
	return &DeveloperHandler{
		DeveloperService: DeveloperService,
	}
}

// GetLogFilterOptions handles GET requests to retrieve distinct values for log filter columns.
// Returns sorted lists of levels, categories, sources, and messages for frontend picklists.
//
// Endpoint: GET /api/developer/logs/filter-options
// Response: 200 OK with LogFilterOptions
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetLogFilterOptions(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "get log filter options request")

	options, err := h.DeveloperService.GetLogFilterOptions()
	if err != nil {
		devLog.ErrorContext(r.Context(), "failed to get log filter options", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveLogFilterOpts.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, options)
}

// GetLogs handles GET requests to retrieve system logs with filtering and pagination.
// Returns logs with cursor-based pagination for efficient traversal of large result sets.
// Supports filtering by level, category, date range, source, and message content.
//
// Query Parameters:
//   - level: Comma-separated log levels (debug, info, warning, error, critical)
//   - category: Comma-separated categories (portfolio, fund, transaction, etc.)
//   - startDate: Filter logs from this date (YYYY-MM-DD format)
//   - endDate: Filter logs until this date (YYYY-MM-DD format)
//   - source: Filter by source field (partial match)
//   - message: Filter by message content (partial match)
//   - sortDir: Sort direction (asc or desc, default: desc)
//   - cursor: Pagination cursor from previous response
//   - perPage: Number of results per page (1-250, default: 50)
//   - skip: Number of rows to skip (offset-based pagination, used alongside cursor)
//
// Endpoint: GET /api/developer/logs
// Response: 200 OK with LogResponse containing logs, pagination info
// Error: 400 Bad Request if filter parameters are invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "get logs request",
		"filter_level", r.URL.Query().Get("level"),
		"filter_category", r.URL.Query().Get("category"),
		"filter_sort_dir", r.URL.Query().Get("sortDir"),
	)

	// Parse filter parameters
	filters, err := request.ParseLogFilters(
		r.URL.Query().Get("level"),
		r.URL.Query().Get("category"),
		r.URL.Query().Get("startDate"),
		r.URL.Query().Get("endDate"),
		r.URL.Query().Get("source"),
		r.URL.Query().Get("message"),
		r.URL.Query().Get("sortDir"),
		r.URL.Query().Get("cursor"),
		r.URL.Query().Get("perPage"),
		r.URL.Query().Get("skip"),
	)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid filter parameters", err.Error())
		return
	}

	// Call service with filters
	logs, err := h.DeveloperService.GetLogs(r.Context(), filters)
	if err != nil {
		devLog.ErrorContext(r.Context(), "failed to get logs", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveLogs.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, logs)
}

// GetLoggingConfig handles GET requests to retrieve the current logging configuration.
// Returns the enabled status and logging level settings.
//
// Endpoint: GET /api/developer/system-settings/logging
// Response: 200 OK with LoggingSetting
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetLoggingConfig(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "get logging config request")

	setting, err := h.DeveloperService.GetLoggingConfig()
	if err != nil {
		devLog.ErrorContext(r.Context(), "failed to get logging config", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveLoggingConfig.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, setting)
}

// SetLoggingConfig handles PUT requests to update the logging configuration.
// Accepts a JSON body with enabled (bool) and level (string) fields.
// Both fields are required; level must be one of: debug, info, warning, error, critical.
//
// Endpoint: PUT /api/developer/system-settings/logging
// Response: 200 OK with updated LoggingSetting
// Error: 400 Bad Request if body is invalid or validation fails
// Error: 500 Internal Server Error if update fails
func (h *DeveloperHandler) SetLoggingConfig(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "set logging config request")

	req, err := parseJSON[request.SetLoggingConfig](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateLoggingConfig(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	logSetting, err := h.DeveloperService.SetLoggingConfig(r.Context(), req)
	if err != nil {
		devLog.ErrorContext(r.Context(), "failed to set logging config", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToSetLoggingConfig.Error())
		return
	}

	devLog.InfoContext(r.Context(), "logging config updated", "enabled", logSetting.Enabled, "level", logSetting.Level)
	response.RespondJSON(w, http.StatusOK, logSetting)
}

// GetFundPriceCSVTemplate handles GET requests to retrieve the CSV template for fund price imports.
// Returns the expected CSV headers, an example row, and a description of the format.
//
// Endpoint: GET /api/developer/csv/fund-prices/template
// Response: 200 OK with TemplateModel
func (h *DeveloperHandler) GetFundPriceCSVTemplate(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "get fund price CSV template request")

	headers := []string{"date", "price"}
	example := map[string]string{
		"date":  "2024-03-21",
		"price": "150.75",
	}
	description := `CSV file should contain the following columns:
- date: Price date in YYYY-MM-DD format
- price: Fund price (decimal numbers)`

	template := model.TemplateModel{
		Headers:     headers,
		Example:     example,
		Description: description,
	}

	response.RespondJSON(w, http.StatusOK, template)
}

// GetTransactionCSVTemplate handles GET requests to retrieve the CSV template for transaction imports.
// Returns the expected CSV headers, an example row, and a description of the format.
//
// Endpoint: GET /api/developer/csv/transactions/template
// Response: 200 OK with TemplateModel
func (h *DeveloperHandler) GetTransactionCSVTemplate(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "get transaction CSV template request")

	headers := []string{"date", "type", "shares", "cost_per_share"}
	example := map[string]string{
		"date":           "2024-03-21",
		"type":           "buy",
		"shares":         "10.5",
		"cost_per_share": "150.75",
	}
	description := `CSV file should contain the following columns:
- date: Transaction date in YYYY-MM-DD format
- type: Transaction type; one of "buy", "sell", "dividend", or "fee"
- shares: Number of shares (decimal numbers allowed)
- cost_per_share: Cost per share in the fund's currency`

	template := model.TemplateModel{
		Headers:     headers,
		Example:     example,
		Description: description,
	}

	response.RespondJSON(w, http.StatusOK, template)
}

// GetExchangeRate handles GET requests to retrieve an exchange rate for a specific currency pair and date.
// Returns the exchange rate if found, or null if no rate exists for the given parameters.
// Always returns 200 OK with the wrapper, even if the rate is not found (rate will be null).
//
// Query Parameters:
//   - fromCurrency: Source currency code (required)
//   - toCurrency: Target currency code (required)
//   - date: Date for the exchange rate in YYYY-MM-DD format (required)
//
// Endpoint: GET /api/developer/exchange-rate
// Response: 200 OK with ExchangeRateWrapper (rate may be null if not found)
// Error: 400 Bad Request if required parameters are missing or date format is invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetExchangeRate(w http.ResponseWriter, r *http.Request) {

	fromCurrency := r.URL.Query().Get("fromCurrency")
	toCurrency := r.URL.Query().Get("toCurrency")
	dateStr := r.URL.Query().Get("date")

	devLog.DebugContext(r.Context(), "get exchange rate request", "from", fromCurrency, "to", toCurrency, "date", dateStr)

	if fromCurrency == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidCurrency.Error(), "fromCurrency is required")
		return
	}
	if toCurrency == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidCurrency.Error(), "toCurrency is required")
		return
	}
	if dateStr == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidDate.Error(), "date is required")
		return
	}

	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid date format", "Date must be in YYYY-MM-DD format")
		return
	}
	date := parsedDate.Format("2006-01-02")

	exchangeResponse := model.ExchangeRateWrapper{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Date:         date,
	}

	exchangeRate, err := h.DeveloperService.GetExchangeRate(fromCurrency, toCurrency, parsedDate)
	if err != nil {
		if errors.Is(err, apperrors.ErrExchangeRateNotFound) {
			exchangeResponse.Rate = nil
			response.RespondJSON(w, http.StatusOK, exchangeResponse)
			return
		}
		devLog.ErrorContext(r.Context(), "failed to get exchange rate", "error", err, "from", fromCurrency, "to", toCurrency, "date", dateStr)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveExchangeRate.Error())
		return
	}

	exchangeResponse.Rate = exchangeRate

	response.RespondJSON(w, http.StatusOK, exchangeResponse)
}

// UpdateExchangeRate handles POST requests to create or update an exchange rate.
// Accepts a JSON body with date, fromCurrency, toCurrency, and rate fields.
// Upserts the rate for the given currency pair and date.
//
// Endpoint: POST /api/developer/exchange-rate
// Response: 200 OK with ExchangeRate
// Error: 400 Bad Request if body is invalid or validation fails
// Error: 500 Internal Server Error if update fails
func (h *DeveloperHandler) UpdateExchangeRate(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "update exchange rate request")

	req, err := parseJSON[request.SetExchangeRateRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdateExchangeRate(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	exRate, err := h.DeveloperService.UpdateExchangeRate(r.Context(), req)
	if err != nil {
		devLog.ErrorContext(r.Context(), "failed to update exchange rate", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToUpdateExchangeRate.Error())
		return
	}

	devLog.InfoContext(r.Context(), "exchange rate updated", "from", req.FromCurrency, "to", req.ToCurrency, "date", req.Date)
	response.RespondJSON(w, http.StatusOK, exRate)
}

// GetFundPrice handles GET requests to retrieve a fund's price for a specific date.
// Returns the fund price if found for the given fund and date.
//
// Query Parameters:
//   - fundId: The fund UUID (required)
//   - date: Date for the fund price in YYYY-MM-DD format (required)
//
// Endpoint: GET /api/developer/fund-price
// Response: 200 OK with FundPrice
// Error: 400 Bad Request if required parameters are missing or date format is invalid
// Error: 404 Not Found if no price exists for the given fund and date
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetFundPrice(w http.ResponseWriter, r *http.Request) {
	fundID := r.URL.Query().Get("fundId")
	dateStr := r.URL.Query().Get("date")

	devLog.DebugContext(r.Context(), "get fund price request", "fund_id", fundID, "date", dateStr)

	if fundID == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidFundID.Error(), "fundId is required")
		return
	}
	if dateStr == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidDate.Error(), "date is required")
		return
	}

	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid date format", "Date must be in YYYY-MM-DD format")
		return
	}

	fundPrice, err := h.DeveloperService.GetFundPrice(fundID, parsedDate)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundPriceNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundPriceNotFound.Error(), "")
			return
		}
		devLog.ErrorContext(r.Context(), "failed to get fund price", "error", err, "fund_id", fundID, "date", dateStr)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveFundPrice.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, fundPrice)
}

// UpdateFundPrice handles POST requests to create or update a fund price.
// Accepts a JSON body with date, fundId, and price fields.
// Upserts the price for the given fund and date.
//
// Endpoint: POST /api/developer/fund-price
// Response: 200 OK with FundPrice
// Error: 400 Bad Request if body is invalid or validation fails
// Error: 500 Internal Server Error if update fails
func (h *DeveloperHandler) UpdateFundPrice(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "update fund price request")

	req, err := parseJSON[request.SetFundPriceRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdateFundPrice(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	fundPrice, err := h.DeveloperService.UpdateFundPrice(r.Context(), req)
	if err != nil {
		devLog.ErrorContext(r.Context(), "failed to update fund price", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToUpdateFundPrice.Error())
		return
	}

	devLog.InfoContext(r.Context(), "fund price updated", "fund_id", req.FundID, "date", req.Date)
	response.RespondJSON(w, http.StatusOK, fundPrice)
}

// DeleteLogs handles DELETE requests to clear all logs from the database.
// Records the deletion event as a new log entry after clearing.
//
// Endpoint: DELETE /api/developer/logs
// Response: 204 No Content on success
// Error: 500 Internal Server Error if deletion fails
func (h *DeveloperHandler) DeleteLogs(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "delete logs request")

	ipAddress := getClientIP(r)

	err := h.DeveloperService.DeleteLogs(r.Context(), ipAddress, r.Header.Get("User-Agent"))
	if err != nil {
		devLog.ErrorContext(r.Context(), "failed to delete logs", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToDeleteLogs.Error())
		return
	}

	devLog.InfoContext(r.Context(), "logs deleted")
	response.RespondJSON(w, http.StatusNoContent, nil)
}

// getClientIP extracts the client IP address from a request.
// Checks X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
// Returns nil if the IP cannot be determined.
func getClientIP(r *http.Request) *string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can be "client, proxy1, proxy2" — take the leftmost entry
		if i := strings.IndexByte(ip, ','); i != -1 {
			ip = ip[:i]
		}
		lastIP := strings.TrimSpace(ip)
		return &lastIP
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return &ip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}
	return &ip
}

// ImportFundPrices handles POST requests to import fund prices from a CSV file.
// Accepts multipart form data with fields: file (CSV), fundId (string).
// Validates the fund exists, the file content-type/extension, and CSV structure before importing.
//
// Endpoint: POST /api/developer/import-fund-prices
// Response: 200 OK with count of imported rows
// Error: 400 Bad Request for invalid input or CSV format issues
// Error: 500 Internal Server Error if import fails
func (h *DeveloperHandler) ImportFundPrices(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "import fund prices request")

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB limit
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.RespondError(w, http.StatusBadRequest, "failed to parse form", err.Error())
		return
	}

	fundID := strings.TrimSpace(r.FormValue("fundId"))
	if fundID == "" {
		response.RespondError(w, http.StatusBadRequest, "fundId is required", "")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "file is required", err.Error())
		return
	}
	defer file.Close()

	if err := validateCSVFile(header.Filename, header.Header.Get("Content-Type")); err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid file", err.Error())
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "failed to read file", err.Error())
		return
	}

	count, err := h.DeveloperService.ImportFundPrices(r.Context(), fundID, content)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundNotFound) {
			response.RespondError(w, http.StatusBadRequest, "fund not found", err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInvalidCSVHeaders) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidCSVHeaders.Error(), err.Error())
			return
		}
		devLog.ErrorContext(r.Context(), "failed to import fund prices", "error", err, "fund_id", fundID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToImportFundPrices.Error())
		return
	}

	devLog.InfoContext(r.Context(), "fund prices imported", "fund_id", fundID, "count", count)
	response.RespondJSON(w, http.StatusOK, map[string]int{"imported": count})
}

// ImportTransactions handles POST requests to import transactions from a CSV file.
// Accepts multipart form data with fields: file (CSV), fundId (portfolio-fund relationship ID).
// Validates the portfolio-fund exists, the file content-type/extension, and CSV structure before importing.
//
// Endpoint: POST /api/developer/import-transactions
// Response: 200 OK with count of imported rows
// Error: 400 Bad Request for invalid input or CSV format issues
// Error: 500 Internal Server Error if import fails
func (h *DeveloperHandler) ImportTransactions(w http.ResponseWriter, r *http.Request) {
	devLog.DebugContext(r.Context(), "import transactions request")

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB limit
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.RespondError(w, http.StatusBadRequest, "failed to parse form", err.Error())
		return
	}

	portfolioFundID := strings.TrimSpace(r.FormValue("fundId"))
	if portfolioFundID == "" {
		response.RespondError(w, http.StatusBadRequest, "fundId is required", "")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "file is required", err.Error())
		return
	}
	defer file.Close()

	if err := validateCSVFile(header.Filename, header.Header.Get("Content-Type")); err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid file", err.Error())
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "failed to read file", err.Error())
		return
	}

	count, err := h.DeveloperService.ImportTransactions(r.Context(), portfolioFundID, content)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			response.RespondError(w, http.StatusBadRequest, "portfolio-fund not found", err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInvalidCSVHeaders) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidCSVHeaders.Error(), err.Error())
			return
		}
		devLog.ErrorContext(r.Context(), "failed to import transactions", "error", err, "portfolio_fund_id", portfolioFundID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToImportTransactions.Error())
		return
	}

	devLog.InfoContext(r.Context(), "transactions imported", "portfolio_fund_id", portfolioFundID, "count", count)
	response.RespondJSON(w, http.StatusOK, map[string]int{"imported": count})
}

// validateCSVFile checks that the uploaded file has a .csv extension and an acceptable Content-Type.
func validateCSVFile(filename, contentType string) error {
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		return fmt.Errorf("file must have a .csv extension, got %q", filename)
	}
	ct := strings.ToLower(contentType)
	allowed := []string{"text/csv", "text/plain", "application/csv", "application/vnd.ms-excel", "application/octet-stream"}
	for _, a := range allowed {
		if strings.Contains(ct, a) {
			return nil
		}
	}
	return fmt.Errorf("unexpected content-type %q; expected a CSV file", contentType)
}
