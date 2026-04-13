// Package service implements the business logic layer for the Investment Portfolio Manager.
// Services orchestrate repository calls, enforce business rules, manage transaction boundaries,
// and coordinate with external integrations (Yahoo Finance, IBKR). Each service corresponds
// to a domain (portfolios, funds, transactions, dividends, IBKR, materialized views, etc.).
package service
