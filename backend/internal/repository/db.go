// Package repository provides data-access implementations for all domain entities.
// Each repository wraps a *sql.DB and exposes query/mutation methods that accept
// either a *sql.DB or a *sql.Tx via the [Querier] interface, enabling services to
// compose multiple repository calls within a single database transaction.
package repository

import (
	"context"
	"database/sql"
)

// Querier is satisfied by both *sql.DB and *sql.Tx, allowing repository methods to participate in a transaction or run standalone.
type Querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
