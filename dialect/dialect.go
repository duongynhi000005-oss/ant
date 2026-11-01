// Package dialect defines the interface each database implements, plus
// helpers they share.
package dialect

import (
	"strings"

	"github.com/skudra-go/ant/qb"
)

// Dialect adapts ant to one database.
type Dialect interface {
	// Name identifies the dialect, such as "postgres".
	Name() string
	// DriverName is the database/sql driver to open. The application imports
	// the driver, so ant has no dependency on it.
	DriverName() string
	// Placeholder returns the bind parameter for position n, counted from 1.
	Placeholder(n int) string
	// Quote quotes an identifier.
	Quote(ident string) string
	// SupportsReturning reports whether an INSERT can hand back columns, through
	// RETURNING or an equivalent. Without it, keys come from LastInsertId.
	SupportsReturning() bool
	// Render converts a statement to SQL and its arguments.
	Render(stmt qb.Statement) (sql string, args []any, err error)
}

// QuoteWith wraps ident in open and close, doubling any embedded close.
func QuoteWith(ident, open, close string) string {
	return open + strings.ReplaceAll(ident, close, close+close) + close
}
