package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/validation"
)

var pfLog = logging.NewLogger("portfolio")

// PortfolioHandler handles HTTP requests for portfolio endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the PortfolioService.
type PortfolioHandler struct {
	portfolioService    *service.PortfolioService
	fundService         *service.FundService
	materializedService *service.MaterializedService
}

// NewPortfolioHandler creates a new PortfolioHandler with the provided service dependency.
func NewPortfolioHandler(portfolioService *service.PortfolioService, fundService *service.FundService, materializedService *service.MaterializedService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService:    portfolioService,
		fundService:         fundService,
		materializedService: materializedService,
	}
}

// Portfolios handles GET requests to retrieve all portfolios.
// This endpoint returns all portfolios including archived and excluded ones.
//
// Endpoint: GET /api/portfolio
// Response: 200 OK with array of PortfoliosResponse
// Error: 500 Internal Server Error if retrieval fails
func (h *PortfolioHandler) Portfolios(w http.ResponseWriter, r *http.Request) {
	pfLog.DebugContext(r.Context(), "get all portfolios request")

	portfolios, err := h.portfolioService.GetAllPortfolios()
	if err != nil {
		pfLog.ErrorContext(r.Context(), "failed to get all portfolios", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrievePortfolios.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, portfolios)
}

// GetPortfolio handles GET requests to retrieve a single portfolio with its current summary.
// Returns the portfolio details along with current valuations (totalValue, totalCost, etc.).
//
// Endpoint: GET /api/portfolio/{portfolioId}
// Response: 200 OK with PortfolioSummary
// Error: 500 Internal Server Error if retrieval or calculation fails
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "uuid")

	pfLog.DebugContext(r.Context(), "get portfolio request", "portfolio_id", portfolioID)

	if portfolioID == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidPortfolioID.Error(), "")
		return
	}

	summaries, err := h.materializedService.GetPortfolioSummaryWithFallback(portfolioID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}

		pfLog.ErrorContext(r.Context(), "failed to get portfolio summary", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetPortfolioSummary.Error())
		return
	}

	if len(summaries) == 0 {
		// Portfolio exists but has no transactions - return zero values
		//nolint:errcheck // Portfolio validation already performed by earlier steps
		portfolio, _ := h.portfolioService.GetPortfolio(portfolioID)
		response.RespondJSON(w, http.StatusOK, model.PortfolioSummary{
			ID:          portfolio.ID,
			Name:        portfolio.Name,
			Description: portfolio.Description,
			IsArchived:  portfolio.IsArchived,
		})
		return
	}

	// Return the single portfolio summary
	response.RespondJSON(w, http.StatusOK, summaries[0])
}

// PortfolioSummary handles GET requests to retrieve current portfolio summaries.
// Returns comprehensive metrics for all active (non-archived, non-excluded) portfolios
// as of the current time.
//
// Endpoint: GET /api/portfolio/summary
// Response: 200 OK with array of PortfolioSummaryResponse
// Error: 500 Internal Server Error if calculation fails
func (h *PortfolioHandler) PortfolioSummary(w http.ResponseWriter, r *http.Request) {
	pfLog.DebugContext(r.Context(), "get portfolio summary request")

	summaries, err := h.materializedService.GetPortfolioSummaryWithFallback("")
	if err != nil {
		pfLog.ErrorContext(r.Context(), "failed to get portfolio summaries", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetPortfolioSummary.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, summaries)
}

// PortfolioHistory handles GET requests to retrieve historical portfolio valuations.
//
// Query Parameters:
//   - start_date (optional): First date to include (YYYY-MM-DD or RFC3339 format)
//     Defaults to 1970-01-01 if not provided
//   - end_date (optional): Last date to include (YYYY-MM-DD or RFC3339 format)
//     Defaults to current date if not provided
//
// The endpoint returns daily portfolio valuations for the requested date range.
// Only active portfolios are included, and the actual returned range may be
// narrowed to the range where transaction data exists.
//
// Endpoint: GET /api/portfolio/history?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD
// Response: 200 OK with array of PortfolioHistoryResponse (one per day)
// Error: 400 Bad Request if date parsing fails
// Error: 500 Internal Server Error if calculation fails
func (h *PortfolioHandler) PortfolioHistory(w http.ResponseWriter, r *http.Request) {
	pfLog.DebugContext(r.Context(), "get portfolio history request",
		"start_date", r.URL.Query().Get("start_date"),
		"end_date", r.URL.Query().Get("end_date"),
	)

	startDate, endDate, err := parseDateParams(r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid date parameters", err.Error())
		return
	}

	portfolioHistory, err := h.materializedService.GetPortfolioHistoryWithFallback(startDate, endDate, "")
	if err != nil {
		pfLog.ErrorContext(r.Context(), "failed to get portfolio history", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetPortfolioHistory.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, portfolioHistory)
}

// parseDateParams extracts and validates start_date and end_date from query parameters.
func parseDateParams(r *http.Request) (time.Time, time.Time, error) {
	var startDate, endDate time.Time
	var err error

	if r.URL.Query().Get("start_date") == "" {
		startDate, err = time.Parse("2006-01-02", "1970-01-01")
		if err != nil {
			panic("impossible: hardcoded date failed to parse: " + err.Error())
		}
	} else {
		startDate, err = time.Parse("2006-01-02", r.URL.Query().Get("start_date"))
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if r.URL.Query().Get("end_date") == "" {
		endDate = time.Now().UTC()
	} else {
		endDate, err = time.Parse("2006-01-02", r.URL.Query().Get("end_date"))
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return startDate, endDate, nil
}

// PortfolioFunds handles GET requests to retrieve all portfolio-fund relationships.
// Returns a listing of all funds across all portfolios with basic metadata.
//
// Endpoint: GET /api/portfolio/funds
// Response: 200 OK with array of PortfolioFundListing
// Error: 500 Internal Server Error if retrieval fails
func (h *PortfolioHandler) PortfolioFunds(w http.ResponseWriter, r *http.Request) {
	pfLog.DebugContext(r.Context(), "get all portfolio funds request")

	listings, err := h.fundService.GetAllPortfolioFundListings()
	if err != nil {
		pfLog.ErrorContext(r.Context(), "failed to get portfolio fund listings", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrievePortfolioFunds.Error())
		return
	}

	if listings == nil {
		listings = []model.PortfolioFundListing{}
	}

	response.RespondJSON(w, http.StatusOK, listings)
}

// GetPortfolioFunds handles GET requests to retrieve all funds for a specific portfolio.
// Returns detailed fund metrics including shares, cost, value, gains/losses, dividends, and fees
// for each fund held in the specified portfolio.
//
// Endpoint: GET /api/portfolio/funds/{portfolioId}
// Response: 200 OK with array of PortfolioFund
// Error: 500 Internal Server Error if retrieval or calculation fails
func (h *PortfolioHandler) GetPortfolioFunds(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "uuid")

	pfLog.DebugContext(r.Context(), "get portfolio funds request", "portfolio_id", portfolioID)

	if portfolioID == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidPortfolioID.Error(), "")
		return
	}

	portfolioFunds, err := h.fundService.GetPortfolioFunds(portfolioID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}

		pfLog.ErrorContext(r.Context(), "failed to get portfolio funds", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetPortfolioFunds.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, portfolioFunds)
}

// CreatePortfolio handles POST requests to create a new portfolio.
// Creates a portfolio with the provided name, description, and settings.
//
// Endpoint: POST /api/portfolio
// Request: JSON body with CreatePortfolioRequest (name required, description and excludeFromOverview optional)
// Response: 201 Created with created portfolio including generated ID
// Error: 400 Bad Request if JSON is invalid or validation fails
// Error: 500 Internal Server Error if creation fails
func (h *PortfolioHandler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	pfLog.DebugContext(r.Context(), "create portfolio request")

	req, err := parseJSON[request.CreatePortfolioRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateCreatePortfolio(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	portfolio, err := h.portfolioService.CreatePortfolio(r.Context(), req)
	if err != nil {
		pfLog.ErrorContext(r.Context(), "failed to create portfolio", "error", err)
		response.RespondInternalError(w, r, "failed to create portfolio")
		return
	}

	pfLog.InfoContext(r.Context(), "portfolio created", "id", portfolio.ID, "name", portfolio.Name)
	response.RespondJSON(w, http.StatusCreated, portfolio)
}

// UpdatePortfolio handles PUT requests to update an existing portfolio.
// Supports partial updates - only provided fields will be updated.
//
// Endpoint: PUT /api/portfolio/{portfolioId}
// Request: JSON body with UpdatePortfolioRequest (all fields optional)
// Response: 200 OK with updated portfolio
// Error: 400 Bad Request if portfolio ID is invalid, JSON is invalid, or validation fails
// Error: 404 Not Found if portfolio doesn't exist
// Error: 500 Internal Server Error if update fails
func (h *PortfolioHandler) UpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "uuid")

	pfLog.DebugContext(r.Context(), "update portfolio request", "portfolio_id", portfolioID)

	req, err := parseJSON[request.UpdatePortfolioRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdatePortfolio(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(r.Context(), portfolioID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}

		pfLog.ErrorContext(r.Context(), "failed to update portfolio", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, "failed to update portfolio")
		return
	}

	pfLog.InfoContext(r.Context(), "portfolio updated", "id", portfolio.ID)
	response.RespondJSON(w, http.StatusOK, portfolio)
}

// DeletePortfolio handles DELETE requests to delete a portfolio.
// Permanently deletes the portfolio and all related data (cascading delete).
//
// Endpoint: DELETE /api/portfolio/{portfolioId}
// Response: 204 No Content on successful deletion
// Error: 400 Bad Request if portfolio ID is invalid
// Error: 404 Not Found if portfolio doesn't exist
// Error: 500 Internal Server Error if deletion fails
func (h *PortfolioHandler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "uuid")

	pfLog.DebugContext(r.Context(), "delete portfolio request", "portfolio_id", portfolioID)

	err := h.portfolioService.DeletePortfolio(r.Context(), portfolioID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}

		pfLog.ErrorContext(r.Context(), "failed to delete portfolio", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, "failed to delete portfolio")
		return
	}

	pfLog.InfoContext(r.Context(), "portfolio deleted", "portfolio_id", portfolioID)
	response.RespondJSON(w, http.StatusNoContent, nil)
}

// ArchivePortfolio handles POST requests to archive a portfolio.
// Archives the portfolio by setting IsArchived to true.
//
// Endpoint: POST /api/portfolio/{portfolioId}/archive
// Response: 200 OK with updated portfolio
// Error: 400 Bad Request if portfolio ID is invalid
// Error: 404 Not Found if portfolio doesn't exist
// Error: 500 Internal Server Error if update fails
func (h *PortfolioHandler) ArchivePortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "uuid")

	pfLog.DebugContext(r.Context(), "archive portfolio request", "portfolio_id", portfolioID)

	isArchived := true
	req := request.UpdatePortfolioRequest{
		IsArchived: &isArchived,
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(r.Context(), portfolioID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}
		pfLog.ErrorContext(r.Context(), "failed to archive portfolio", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, "failed to archive portfolio")
		return
	}

	pfLog.InfoContext(r.Context(), "portfolio archived", "portfolio_id", portfolioID)
	response.RespondJSON(w, http.StatusOK, portfolio)
}

// UnarchivePortfolio handles POST requests to unarchive a portfolio.
// Unarchives the portfolio by setting IsArchived to false.
//
// Endpoint: POST /api/portfolio/{portfolioId}/unarchive
// Response: 200 OK with updated portfolio
// Error: 400 Bad Request if portfolio ID is invalid
// Error: 404 Not Found if portfolio doesn't exist
// Error: 500 Internal Server Error if update fails
func (h *PortfolioHandler) UnarchivePortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "uuid")

	pfLog.DebugContext(r.Context(), "unarchive portfolio request", "portfolio_id", portfolioID)

	isArchived := false
	req := request.UpdatePortfolioRequest{
		IsArchived: &isArchived,
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(r.Context(), portfolioID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}
		pfLog.ErrorContext(r.Context(), "failed to unarchive portfolio", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, "failed to unarchive portfolio")
		return
	}

	pfLog.InfoContext(r.Context(), "portfolio unarchived", "portfolio_id", portfolioID)
	response.RespondJSON(w, http.StatusOK, portfolio)
}

// CreatePortfolioFund handles POST requests to add a fund to a portfolio.
// Requires valid portfolioId and fundId in the request body.
// Returns 201 Created on success.
//
// Error: 400 Bad Request if request body is invalid
// Error: 404 Not Found if portfolio or fund doesn't exist
// Error: 500 Internal Server Error if creation fails
func (h *PortfolioHandler) CreatePortfolioFund(w http.ResponseWriter, r *http.Request) {
	pfLog.DebugContext(r.Context(), "create portfolio fund request")

	req, err := parseJSON[request.CreatePortfolioFundRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateCreatePortfolioFund(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	err = h.fundService.CreatePortfolioFund(r.Context(), req)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrFundNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundNotFound.Error(), "")
			return
		}
		pfLog.ErrorContext(r.Context(), "failed to create portfolio fund", "error", err)
		response.RespondInternalError(w, r, "failed to create portfolio fund")
		return
	}

	pfLog.InfoContext(r.Context(), "portfolio fund created", "portfolio_id", req.PortfolioID, "fund_id", req.FundID)
	response.RespondJSON(w, http.StatusCreated, nil)
}

// DeletePortfolioFund handles DELETE requests to remove a fund from a portfolio.
// Requires a valid portfolio_fund UUID in the URL path.
// Requires ?confirm=true query parameter to prevent accidental deletions.
// Returns 204 No Content on success.
//
// Error: 409 Conflict if confirm parameter is not "true"
// Error: 404 Not Found if portfolio-fund relationship doesn't exist
// Error: 500 Internal Server Error if deletion fails
func (h *PortfolioHandler) DeletePortfolioFund(w http.ResponseWriter, r *http.Request) {
	portfolioFundID := chi.URLParam(r, "uuid")
	confirm := r.URL.Query().Get("confirm")

	pfLog.DebugContext(r.Context(), "delete portfolio fund request", "portfolio_fund_id", portfolioFundID)

	if confirm != "true" {
		response.RespondError(w, http.StatusConflict, "Confirm deletion", "")
		return
	}

	err := h.fundService.DeletePortfolioFund(r.Context(), portfolioFundID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioFundNotFound.Error(), "")
			return
		}

		pfLog.ErrorContext(r.Context(), "failed to delete portfolio fund", "error", err, "portfolio_fund_id", portfolioFundID)
		response.RespondInternalError(w, r, "failed to delete portfolio fund")
		return
	}

	pfLog.InfoContext(r.Context(), "portfolio fund deleted", "portfolio_fund_id", portfolioFundID)
	response.RespondJSON(w, http.StatusNoContent, nil)
}
