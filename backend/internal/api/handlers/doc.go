// Package handlers contains the HTTP handler types for the Investment Portfolio Manager API.
// Each handler struct corresponds to a domain (portfolios, funds, transactions, dividends,
// IBKR, system, developer) and implements the HTTP layer by parsing requests, delegating to
// the service layer, and writing structured JSON responses.
package handlers
