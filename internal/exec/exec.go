// Package exec defines the seam between ant and database/sql.
package exec

import (
	"context"
	"database/sql"
)

// Executor is the part of *sql.DB, *sql.Tx, and *sql.Conn that ant uses to
// run statements.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

var (
	_ Executor = (*sql.DB)(nil)
	_ Executor = (*sql.Tx)(nil)
	_ Executor = (*sql.Conn)(nil)
)
