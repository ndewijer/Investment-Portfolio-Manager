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

var ibkrLog = logging.NewLogger("ibkr")

// IbkrHandler handles HTTP requests for ibkr endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the ibkrService.
type IbkrHandler struct {
	ibkrService *service.IbkrService
}

// NewIbkrHandler creates a new IbkrHandler with the provided service dependency.
func NewIbkrHandler(ibkrService *service.IbkrService) *IbkrHandler {
	return &IbkrHandler{
		ibkrService: ibkrService,
	}
}

// GetConfig handles GET requests to retrieve the IBKR integration configuration.
// Returns configuration details including flex query ID, token expiration, import settings,
// and default allocation settings.
//
// Endpoint: GET /api/ibkr/config
// Response: 200 OK with IbkrConfig
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "get ibkr config request")

	config, err := h.ibkrService.GetIbkrConfig()
	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "failed to get ibkr config", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveIbkrConfig.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, config)
}

// GetActivePortfolios handles GET requests to retrieve all active portfolios for IBKR import allocation.
// Returns portfolios that are not archived and not excluded from tracking.
//
// Endpoint: GET /api/ibkr/portfolios
// Response: 200 OK with array of Portfolio
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetActivePortfolios(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "get active portfolios request")

	config, err := h.ibkrService.GetActivePortfolios()
	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "failed to get active portfolios", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrievePortfolios.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, config)
}

// GetPendingDividends handles GET requests to retrieve dividend records awaiting processing.
// Returns dividends with reinvestment_status = 'PENDING' that can be matched to IBKR dividend transactions.
//
// Endpoint: GET /api/ibkr/dividend/pending
// Query params:
//   - symbol: Filter by fund trading symbol (optional)
//   - isin: Filter by fund ISIN (optional)
//
// Response: 200 OK with array of PendingDividend
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetPendingDividends(w http.ResponseWriter, r *http.Request) {

	symbol := r.URL.Query().Get("symbol")
	isin := r.URL.Query().Get("isin")

	ibkrLog.DebugContext(r.Context(), "get pending dividends request", "symbol", symbol, "isin", isin)

	pendingDividend, err := h.ibkrService.GetPendingDividends(symbol, isin)

	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "failed to get pending dividends", "error", err, "symbol", symbol, "isin", isin)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrievePendingDividend.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, pendingDividend)
}

// GetInbox handles GET requests to retrieve IBKR imported transactions from the inbox.
// Returns transactions that have been imported from IBKR and are awaiting allocation or processing.
//
// Endpoint: GET /api/ibkr/inbox
// Query params:
//   - status: Filter by transaction status (optional, defaults to "pending")
//   - transaction_type: Filter by transaction type (optional, e.g., "dividend", "trade")
//
// Response: 200 OK with array of IBKRTransaction
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	transactionType := r.URL.Query().Get("transactionType")

	ibkrLog.DebugContext(r.Context(), "get inbox request", "status", status, "transaction_type", transactionType)

	inbox, err := h.ibkrService.GetInbox(status, transactionType)

	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "failed to get inbox", "error", err, "status", status, "transaction_type", transactionType)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveInboxTransactions.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, inbox)
}

// GetInboxCount handles GET requests to retrieve the count of IBKR imported transactions.
// Returns the total number of pending transactions in the inbox.
//
// Endpoint: GET /api/ibkr/inbox/count
//
// Response: 200 OK with {"count": <number>}
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetInboxCount(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "get inbox count request")

	count, err := h.ibkrService.GetInboxCount()

	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "failed to get inbox count", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveInboxTransactions.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, count)
}

// GetTransactionAllocations handles GET /api/ibkr/inbox/{uuid}/allocations
// Retrieves the allocation details for a specific IBKR transaction, showing how it was
// distributed across portfolios including amounts, shares, and fees.
//
// Path parameters:
//   - uuid: UUID of the IBKR transaction (validated by middleware)
//
// Responses:
//   - 200: Success with allocation details
//   - 400: Invalid transaction ID (validated by middleware)
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) GetTransactionAllocations(w http.ResponseWriter, r *http.Request) {
	// UUID is already validated by ValidateUUIDMiddleware
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "get transaction allocations request", "transaction_id", transactionID)

	transactionAllocations, err := h.ibkrService.GetTransactionAllocations(transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}

		ibkrLog.ErrorContext(r.Context(), "failed to get transaction allocations", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetTransactionAllocations.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transactionAllocations)
}

// GetEligiblePortfolios handles GET /api/ibkr/inbox/{uuid}/eligible-portfolios
// Finds portfolios eligible for allocating an IBKR transaction by matching the transaction's
// fund via ISIN or symbol. Returns fund details and the list of portfolios that hold this fund.
//
// Note: This endpoint matches Python API behavior - returns 200 OK with match_info.found=false
// when no fund is found (not 404). This allows clients to handle missing funds gracefully.
//
// Path parameters:
//   - uuid: UUID of the IBKR transaction (validated by middleware)
//
// Responses:
//   - 200: Success with match_info, portfolios, and optional warning (even if fund not found)
//   - 400: Invalid transaction ID (validated by middleware)
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) GetEligiblePortfolios(w http.ResponseWriter, r *http.Request) {
	// UUID is already validated by ValidateUUIDMiddleware
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "get eligible portfolios request", "transaction_id", transactionID)

	eligiblePortfolios, err := h.ibkrService.GetEligiblePortfolios(transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}

		ibkrLog.ErrorContext(r.Context(), "failed to get eligible portfolios", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetEligiblePortfolios.Error())
		return
	}

	// Always return 200 OK if fund is not found, Frontend uses match_info.found = false
	response.RespondJSON(w, http.StatusOK, eligiblePortfolios)
}

// ImportFlexReport triggers an IBKR Flex report import.
// Serves cached data if a valid cache entry exists, otherwise fetches from the IBKR API.
// Returns a JSON response with the number of imported and skipped transactions.
func (h *IbkrHandler) ImportFlexReport(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "import flex report request")

	add, skipped, err := h.ibkrService.ImportFlexReport(r.Context())
	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "failed to import flex report", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetNewFlexReport.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "flex report imported", "imported", add, "skipped", skipped)

	type respStruct struct {
		Success  bool `json:"success"`
		Imported int  `json:"imported"`
		Skipped  int  `json:"skipped"`
	}

	response.RespondJSON(w, http.StatusOK, respStruct{
		Success: true, Imported: add, Skipped: skipped,
	})
}

// UpdateIbkrConfig handles POST requests to create or update the IBKR integration configuration.
// Applies a partial update — only non-nil fields in the request body are written; existing values
// are preserved for omitted fields. If the flex_query_id changes while enabled is true, the old
// config row is replaced to avoid stale data.
//
// Endpoint: POST /api/ibkr/config
// Response: 201 Created with updated IbkrConfig
// Error: 400 Bad Request on invalid body or validation failure
// Error: 500 Internal Server Error if the update fails
func (h *IbkrHandler) UpdateIbkrConfig(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "update ibkr config request")

	req, err := parseJSON[request.UpdateIbkrConfigRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdateIbkrConfig(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	config, err := h.ibkrService.UpdateIbkrConfig(r.Context(), req)
	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "failed to update ibkr config", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToUpdateIbkrConfig.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "ibkr config updated")
	response.RespondJSON(w, http.StatusCreated, config)
}

// DeleteIbkrConfig handles DELETE requests to remove the IBKR integration configuration.
// Deletes the single config row; returns 404 if no config exists yet.
//
// Endpoint: DELETE /api/ibkr/config
// Response: 204 No Content on success
// Error: 404 Not Found if no config is configured
// Error: 500 Internal Server Error if deletion fails
func (h *IbkrHandler) DeleteIbkrConfig(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "delete ibkr config request")

	err := h.ibkrService.DeleteIbkrConfig(r.Context())
	if err != nil {
		if errors.Is(err, apperrors.ErrIbkrConfigNotFound) {

			response.RespondError(w, http.StatusNotFound, apperrors.ErrIbkrConfigNotFound.Error(), "")
			return
		}

		ibkrLog.ErrorContext(r.Context(), "failed to delete ibkr config", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToDeleteIbkrConfig.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "ibkr config deleted")
	response.RespondJSON(w, http.StatusNoContent, nil)
}

// TestIbkrConnection handles POST requests to verify IBKR API credentials without saving them.
// Accepts a plaintext flexToken and flexQueryId in the request body and submits a SendRequest
// call to IBKR to confirm the credentials are accepted.
//
// Endpoint: POST /api/ibkr/config/test
// Response: 200 OK with {"success": true}
// Error: 400 Bad Request if the request body is invalid or credentials fail validation
// Error: 500 Internal Server Error if the IBKR API call fails
func (h *IbkrHandler) TestIbkrConnection(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "test ibkr connection request")

	req, err := parseJSON[request.TestIbkrConnectionRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateTestConnection(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	_, err = h.ibkrService.TestIbkrConnection(r.Context(), req)
	if err != nil {
		ibkrLog.ErrorContext(r.Context(), "ibkr test connection failed", "error", err)
		response.RespondInternalError(w, r, "ibkr test connection failed")
		return
	}

	ibkrLog.InfoContext(r.Context(), "ibkr connection test successful")
	response.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetTransaction handles GET /api/ibkr/inbox/{uuid}
// Retrieves a single IBKR transaction with its allocation details (if processed).
//
// Responses:
//   - 200: Success with transaction detail
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "get ibkr transaction request", "transaction_id", transactionID)

	detail, err := h.ibkrService.GetIbkrTransactionDetail(transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}
		ibkrLog.ErrorContext(r.Context(), "failed to get ibkr transaction detail", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToGetIbkrTransaction.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, detail)
}

// DeleteTransaction handles DELETE /api/ibkr/inbox/{uuid}
// Removes a pending IBKR transaction. Returns 400 if already processed.
//
// Responses:
//   - 204: Successfully deleted
//   - 400: Transaction already processed
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "delete ibkr transaction request", "transaction_id", transactionID)

	err := h.ibkrService.DeleteIbkrTransaction(r.Context(), transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrIBKRTransactionAlreadyProcessed.Error(), "")
			return
		}
		ibkrLog.ErrorContext(r.Context(), "failed to delete ibkr transaction", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToDeleteIbkrTransaction.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "ibkr transaction deleted", "transaction_id", transactionID)
	response.RespondJSON(w, http.StatusNoContent, nil)
}

// IgnoreTransaction handles POST /api/ibkr/inbox/{uuid}/ignore
// Marks a pending IBKR transaction as ignored. Returns 400 if already processed.
//
// Responses:
//   - 200: Successfully ignored
//   - 400: Transaction already processed
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) IgnoreTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "ignore ibkr transaction request", "transaction_id", transactionID)

	err := h.ibkrService.IgnoreIbkrTransaction(r.Context(), transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrIBKRTransactionAlreadyProcessed.Error(), "")
			return
		}
		ibkrLog.ErrorContext(r.Context(), "failed to ignore ibkr transaction", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToIgnoreIbkrTransaction.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "ibkr transaction ignored", "transaction_id", transactionID)
	response.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// AllocateTransaction handles POST /api/ibkr/inbox/{uuid}/allocate
// Allocates a pending IBKR transaction to portfolios. Allocations are optional — if omitted
// and default allocation is enabled in config, the defaults are used.
//
// Responses:
//   - 200: Successfully allocated
//   - 400: Validation error, already processed, or no fund match
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) AllocateTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "allocate ibkr transaction request", "transaction_id", transactionID)

	req, err := parseJSON[request.AllocateTransactionRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Validate allocations only if provided (empty means auto-allocate)
	if len(req.Allocations) > 0 {
		if err := validation.ValidateAllocateTransaction(req.Allocations); err != nil {
			response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
			return
		}
	}

	err = h.ibkrService.AllocateIbkrTransaction(r.Context(), transactionID, req.Allocations)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) ||
			errors.Is(err, apperrors.ErrIBKRInvalidAllocations) ||
			errors.Is(err, apperrors.ErrIBKRFundNotMatched) {
			response.RespondError(w, http.StatusBadRequest, err.Error(), "")
			return
		}
		ibkrLog.ErrorContext(r.Context(), "failed to allocate ibkr transaction", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToAllocateIbkrTransaction.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "ibkr transaction allocated", "transaction_id", transactionID)
	response.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// BulkAllocate handles POST /api/ibkr/inbox/bulk-allocate
// Allocates multiple IBKR transactions using the same allocation split.
// Each transaction is processed independently — partial success is possible.
//
// Responses:
//   - 200: Results with success/failed counts and error details
//   - 400: Validation error
func (h *IbkrHandler) BulkAllocate(w http.ResponseWriter, r *http.Request) {
	ibkrLog.DebugContext(r.Context(), "bulk allocate ibkr transactions request")

	req, err := parseJSON[request.BulkAllocateRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateBulkAllocate(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	result := h.ibkrService.BulkAllocateIbkrTransactions(r.Context(), req)

	ibkrLog.InfoContext(r.Context(), "bulk allocate completed", "transaction_count", len(req.TransactionIDs))
	response.RespondJSON(w, http.StatusOK, result)
}

// ModifyAllocations handles PUT /api/ibkr/inbox/{uuid}/allocations
// Atomically unallocates and reallocates a processed IBKR transaction with new percentages.
//
// Responses:
//   - 200: Successfully modified
//   - 400: Validation error or transaction not processed
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) ModifyAllocations(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "modify allocations request", "transaction_id", transactionID)

	req, err := parseJSON[request.ModifyAllocationsRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateAllocateTransaction(req.Allocations); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	err = h.ibkrService.ModifyAllocations(r.Context(), transactionID, req.Allocations)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrIBKRTransactionAlreadyProcessed.Error(), "")
			return
		}
		ibkrLog.ErrorContext(r.Context(), "failed to modify allocations", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToModifyAllocations.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "allocations modified", "transaction_id", transactionID)
	response.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// UnallocateTransaction handles POST /api/ibkr/inbox/{uuid}/unallocate
// Reverses the allocation of a processed IBKR transaction, deleting linked Transaction
// and allocation records and resetting the status to "pending".
//
// Responses:
//   - 200: Successfully unallocated
//   - 400: Transaction not processed
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) UnallocateTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "unallocate ibkr transaction request", "transaction_id", transactionID)

	err := h.ibkrService.UnallocateIbkrTransaction(r.Context(), transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrIBKRTransactionAlreadyProcessed.Error(), "")
			return
		}
		ibkrLog.ErrorContext(r.Context(), "failed to unallocate ibkr transaction", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToUnallocateIbkrTransaction.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "ibkr transaction unallocated", "transaction_id", transactionID)
	response.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// MatchDividend handles POST /api/ibkr/inbox/{uuid}/match-dividend
// Links a processed IBKR transaction (DRIP) to pending dividend records.
// The transaction must be allocated first.
//
// Responses:
//   - 200: Successfully matched
//   - 400: Validation error, transaction not processed, or dividend already matched
//   - 404: Transaction or dividend not found
//   - 500: Internal server error
func (h *IbkrHandler) MatchDividend(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	ibkrLog.DebugContext(r.Context(), "match dividend request", "transaction_id", transactionID)

	req, err := parseJSON[request.MatchDividendRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateMatchDividend(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	err = h.ibkrService.MatchDividend(r.Context(), transactionID, req.DividendIDs)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrDividendNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrIBKRTransactionAlreadyProcessed) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrIBKRTransactionAlreadyProcessed.Error(), "")
			return
		}
		ibkrLog.ErrorContext(r.Context(), "failed to match dividend", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToMatchDividend.Error())
		return
	}

	ibkrLog.InfoContext(r.Context(), "dividend matched", "transaction_id", transactionID, "dividend_ids", req.DividendIDs)
	response.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}
