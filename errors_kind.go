package ant

import (
	"errors"

	"github.com/skudra-go/ant/dialect"
	"github.com/skudra-go/ant/internal/exec"
)

// FailedSQL returns the statement behind a query error, or "". Values are not in
// it, only their placeholders, so it is safe for an internal log.
func FailedSQL(err error) string {
	var e *exec.Error
	if errors.As(err, &e) {
		return e.SQL
	}

	return ""
}

// sqlStater is implemented by driver errors that carry a SQLSTATE code.
type sqlStater interface {
	SQLState() string
}

// SQLState returns the database's error code for err, or "". Dialects translate
// their own codes to SQLSTATE, so one value means one thing on every database.
func SQLState(err error) string {
	var e sqlStater
	if errors.As(err, &e) {
		return e.SQLState()
	}

	return dialect.StateOf(err)
}

// IsUniqueViolation reports whether err is a unique constraint failure.
func IsUniqueViolation(err error) bool { return SQLState(err) == "23505" }

// IsForeignKeyViolation reports whether err is a foreign key failure.
func IsForeignKeyViolation(err error) bool { return SQLState(err) == "23503" }

// IsNotNullViolation reports whether err is a NOT NULL failure.
func IsNotNullViolation(err error) bool { return SQLState(err) == "23502" }

// IsCheckViolation reports whether err is a check constraint failure.
func IsCheckViolation(err error) bool { return SQLState(err) == "23514" }

// IsRetryable reports whether a transaction failed from a deadlock or a
// serialization conflict, and would likely work if run again.
func IsRetryable(err error) bool {
	state := SQLState(err)

	return state == "40001" || state == "40P01"
}
