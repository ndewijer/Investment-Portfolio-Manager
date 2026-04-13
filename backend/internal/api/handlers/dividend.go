package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/validation"
)

var divLog = logging.NewLogger("dividend")

// DividendHandler handles HTTP requests for dividend endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the dividendService.
type DividendHandler struct {
	dividendService *service.DividendService
}

// NewDividendHandler creates a new DividendHandler with the provided service dependency.
func NewDividendHandler(dividendService *service.DividendService) *DividendHandler {
	return &DividendHandler{
		dividendService: dividendService,
	}
}

// GetAllDividend handles GET requests to retrieve all dividends.
// Returns a list of all available dividends that can be held in portfolios.
//
// Endpoint: GET /api/dividend
// Response: 200 OK with array of Dividend
// Error: 500 Internal Server Error if retrieval fails
func (h *DividendHandler) GetAllDividend(w http.ResponseWriter, r *http.Request) {
	divLog.DebugContext(r.Context(), "get all dividends request")

	dividends, err := h.dividendService.GetAllDividend()
	if err != nil {
		divLog.ErrorContext(r.Context(), "failed to get all dividends", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveDividends.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, dividends)
}

// DividendPerPortfolio handles GET requests to retrieve all dividends for a specific portfolio.
// Returns dividend details including fund information, amounts, dates, and reinvestment status
// for all funds held in the specified portfolio.
//
// Endpoint: GET /api/dividend/portfolio/{uuid}
// Response: 200 OK with array of DividendFund
// Error: 400 Bad Request if portfolio ID is invalid (validated by middleware)
// Error: 500 Internal Server Error if retrieval fails
func (h *DividendHandler) DividendPerPortfolio(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "uuid")

	divLog.DebugContext(r.Context(), "get dividends per portfolio request", "portfolio_id", portfolioID)

	dividends, err := h.dividendService.GetDividendFund(portfolioID, "")
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}
		divLog.ErrorContext(r.Context(), "failed to get dividends per portfolio", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveDividends.Error())
		return
	}
	if len(dividends) == 0 {
		response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), "")
		return
	}

	response.RespondJSON(w, http.StatusOK, dividends)
}

// DividendPerFund handles GET requests to retrieve all dividends for a specific fund.
// Returns dividend details including fund information, amounts, dates, and reinvestment status
// for all portfolio positions that held the specified fund.
//
// Endpoint: GET /api/dividend/fund/{uuid}
// Response: 200 OK with array of DividendFund
// Error: 400 Bad Request if fund ID is invalid (validated by middleware)
// Error: 404 Not Found if no dividends exist for the fund
// Error: 500 Internal Server Error if retrieval fails
func (h *DividendHandler) DividendPerFund(w http.ResponseWriter, r *http.Request) {

	fundID := chi.URLParam(r, "uuid")

	divLog.DebugContext(r.Context(), "get dividends per fund request", "fund_id", fundID)

	dividends, err := h.dividendService.GetDividendFund("", fundID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundNotFound.Error(), "")
			return
		}
		divLog.ErrorContext(r.Context(), "failed to get dividends per fund", "error", err, "fund_id", fundID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveDividends.Error())
		return
	}
	if len(dividends) == 0 {
		response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), "")
		return
	}

	response.RespondJSON(w, http.StatusOK, dividends)
}

// GetDividend handles GET requests to retrieve a single dividend by ID.
//
// Endpoint: GET /api/dividend/{uuid}
// Response: 200 OK with Dividend
// Error: 404 Not Found if dividend not found
// Error: 500 Internal Server Error if retrieval fails
func (h *DividendHandler) GetDividend(w http.ResponseWriter, r *http.Request) {
	dividendID := chi.URLParam(r, "uuid")

	divLog.DebugContext(r.Context(), "get dividend request", "dividend_id", dividendID)

	dividends, err := h.dividendService.GetDividend(dividendID)
	if err != nil {
		if errors.Is(err, apperrors.ErrDividendNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), "")
			return
		}
		divLog.ErrorContext(r.Context(), "failed to get dividend", "error", err, "dividend_id", dividendID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveDividends.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, dividends)
}

// CreateDividend handles POST requests to create a new dividend.
// Validates the request body and creates a dividend record in the database.
//
// Endpoint: POST /api/dividend
// Request Body: CreateDividendRequest (portfolioFundId, recordDate, exDividendDate, dividendPerShare, and optionally, buyOrderDate, reinvestmentShares and reinvestmentPrice)
// Response: 201 Created with DividendFund
// Error: 400 Bad Request if validation fails or request body is invalid
// Error: 500 Internal Server Error if creation fails
func (h *DividendHandler) CreateDividend(w http.ResponseWriter, r *http.Request) {
	divLog.DebugContext(r.Context(), "create dividend request")

	req, err := parseJSON[request.CreateDividendRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateCreateDividend(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	dividend, err := h.dividendService.CreateDividend(r.Context(), req)
	if err != nil {
		divLog.ErrorContext(r.Context(), "failed to create dividend", "error", err)
		response.RespondInternalError(w, r, "failed to create dividend")
		return
	}

	divLog.InfoContext(r.Context(), "dividend created", "id", dividend.ID)
	response.RespondJSON(w, http.StatusCreated, dividend)
}

// UpdateDividend handles PUT requests to update an existing dividend.
// Validates the request body and updates the specified dividend fields.
//
// Endpoint: PUT /api/dividend/{uuid}
// Request Body: UpdateDividendRequest (all fields optional)
// Response: 200 OK with updated DividendFund
// Error: 400 Bad Request if dividend ID is invalid (validated by middleware) or validation fails
// Error: 404 Not Found if dividend not found
// Error: 500 Internal Server Error if update fails
func (h *DividendHandler) UpdateDividend(w http.ResponseWriter, r *http.Request) {
	dividendID := chi.URLParam(r, "uuid")

	divLog.DebugContext(r.Context(), "update dividend request", "dividend_id", dividendID)

	req, err := parseJSON[request.UpdateDividendRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdateDividend(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	dividend, err := h.dividendService.UpdateDividend(r.Context(), dividendID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrDividendNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), "")
			return
		}

		divLog.ErrorContext(r.Context(), "failed to update dividend", "error", err, "dividend_id", dividendID)
		response.RespondInternalError(w, r, "failed to update dividend")
		return
	}

	divLog.InfoContext(r.Context(), "dividend updated", "id", dividend.ID)
	response.RespondJSON(w, http.StatusOK, dividend)
}

// DeleteDividend handles DELETE requests to remove a dividend.
// Validates that the dividend exists before deleting.
//
// Endpoint: DELETE /api/dividend/{uuid}
// Response: 204 No Content on successful deletion
// Error: 400 Bad Request if dividend ID is invalid (validated by middleware)
// Error: 404 Not Found if dividend not found
// Error: 500 Internal Server Error if deletion fails
func (h *DividendHandler) DeleteDividend(w http.ResponseWriter, r *http.Request) {
	dividendID := chi.URLParam(r, "uuid")

	divLog.DebugContext(r.Context(), "delete dividend request", "dividend_id", dividendID)

	err := h.dividendService.DeleteDividend(r.Context(), dividendID)
	if err != nil {
		if errors.Is(err, apperrors.ErrDividendNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), "")
			return
		}

		divLog.ErrorContext(r.Context(), "failed to delete dividend", "error", err, "dividend_id", dividendID)
		response.RespondInternalError(w, r, "failed to delete dividend")
		return
	}

	divLog.InfoContext(r.Context(), "dividend deleted", "dividend_id", dividendID)
	response.RespondJSON(w, http.StatusNoContent, nil)
}
