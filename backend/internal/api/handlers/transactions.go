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

var txLog = logging.NewLogger("transaction")

// TransactionHandler handles HTTP requests for transaction endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the transactionService.
type TransactionHandler struct {
	transactionService *service.TransactionService
}

// NewTransactionHandler creates a new TransactionHandler with the provided service dependency.
func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// TransactionPerPortfolio handles GET requests to retrieve all transactions for a specific portfolio.
// Returns transaction details including fund information, dates, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction/portfolio/{uuid}
// Response: 200 OK with array of TransactionResponse
// Error: 400 Bad Request if portfolio ID is invalid (validated by middleware)
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) TransactionPerPortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "uuid")

	txLog.DebugContext(r.Context(), "get transactions per portfolio request", "portfolio_id", portfolioID)

	transactions, err := h.transactionService.GetTransactionsperPortfolio(portfolioID)
	if err != nil {
		txLog.ErrorContext(r.Context(), "failed to get transactions per portfolio", "error", err, "portfolio_id", portfolioID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveTransactions.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transactions)
}

// AllTransactions handles GET requests to retrieve all transactions across all portfolios.
// Returns transaction details including fund information, dates, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction
// Response: 200 OK with array of TransactionResponse
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) AllTransactions(w http.ResponseWriter, r *http.Request) {
	txLog.DebugContext(r.Context(), "get all transactions request")

	transactions, err := h.transactionService.GetTransactionsperPortfolio("")
	if err != nil {
		txLog.ErrorContext(r.Context(), "failed to get all transactions", "error", err)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveTransactions.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transactions)
}

// GetTransaction handles GET requests to retrieve a single transaction by ID.
// Returns transaction details including fund information, date, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction/{uuid}
// Response: 200 OK with TransactionResponse
// Error: 400 Bad Request if transaction ID is invalid (validated by middleware)
// Error: 404 Not Found if transaction not found
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {

	transactionID := chi.URLParam(r, "uuid")

	txLog.DebugContext(r.Context(), "get transaction request", "transaction_id", transactionID)

	transaction, err := h.transactionService.GetTransaction(transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrTransactionNotFound.Error(), "")
			return
		}
		txLog.ErrorContext(r.Context(), "failed to get transaction", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, apperrors.ErrFailedToRetrieveTransaction.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transaction)
}

// CreateTransaction handles POST requests to create a new transaction.
// Validates the request body and creates a transaction record in the database.
//
// Endpoint: POST /api/transaction
// Request Body: CreateTransactionRequest (portfolioFundId, date, type, shares, costPerShare)
// Response: 201 Created with TransactionResponse
// Error: 400 Bad Request if validation fails or request body is invalid
// Error: 500 Internal Server Error if creation fails
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	txLog.DebugContext(r.Context(), "create transaction request")

	req, err := parseJSON[request.CreateTransactionRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateCreateTransaction(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	transaction, err := h.transactionService.CreateTransaction(r.Context(), req)
	if err != nil {
		if errors.Is(err, apperrors.ErrInsufficientShares) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrInsufficientShares.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrPortfolioFundNotFound.Error(), "")
			return
		}
		txLog.ErrorContext(r.Context(), "failed to create transaction", "error", err)
		response.RespondInternalError(w, r, "failed to create transaction")
		return
	}

	txLog.InfoContext(r.Context(), "transaction created", "id", transaction.ID, "type", transaction.Type)
	response.RespondJSON(w, http.StatusCreated, transaction)
}

// UpdateTransaction handles PUT requests to update an existing transaction.
// Validates the request body and updates the specified transaction fields.
//
// Endpoint: PUT /api/transaction/{uuid}
// Request Body: UpdateTransactionRequest (all fields optional)
// Response: 200 OK with updated TransactionResponse
// Error: 400 Bad Request if transaction ID is invalid (validated by middleware) or validation fails
// Error: 404 Not Found if transaction not found
// Error: 500 Internal Server Error if update fails
func (h *TransactionHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	txLog.DebugContext(r.Context(), "update transaction request", "transaction_id", transactionID)

	req, err := parseJSON[request.UpdateTransactionRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdateTransaction(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	transaction, err := h.transactionService.UpdateTransaction(r.Context(), transactionID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrTransactionNotFound.Error(), "")
			return
		}
		if errors.Is(err, apperrors.ErrInsufficientShares) {
			response.RespondError(w, http.StatusBadRequest, apperrors.ErrInsufficientShares.Error(), "")
			return
		}

		txLog.ErrorContext(r.Context(), "failed to update transaction", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, "failed to update transaction")
		return
	}

	txLog.InfoContext(r.Context(), "transaction updated", "id", transaction.ID)
	response.RespondJSON(w, http.StatusOK, transaction)
}

// DeleteTransaction handles DELETE requests to remove a transaction.
// Validates that the transaction exists before deleting.
//
// Endpoint: DELETE /api/transaction/{uuid}
// Response: 204 No Content on successful deletion
// Error: 400 Bad Request if transaction ID is invalid (validated by middleware)
// Error: 404 Not Found if transaction not found
// Error: 500 Internal Server Error if deletion fails
func (h *TransactionHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "uuid")

	txLog.DebugContext(r.Context(), "delete transaction request", "transaction_id", transactionID)

	err := h.transactionService.DeleteTransaction(r.Context(), transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrTransactionNotFound.Error(), "")
			return
		}

		txLog.ErrorContext(r.Context(), "failed to delete transaction", "error", err, "transaction_id", transactionID)
		response.RespondInternalError(w, r, "failed to delete transaction")
		return
	}

	txLog.InfoContext(r.Context(), "transaction deleted", "transaction_id", transactionID)
	response.RespondJSON(w, http.StatusNoContent, nil)
}
