// Package scan maps database rows onto Go values.
package scan

import (
	"database/sql"
	"reflect"
	"unsafe"

	"github.com/skudra-go/ant/internal/meta"
)

// For returns a function that scans the current row into a *T. A value type
// such as int takes the single result column. A struct takes columns by name:
// columns with no field are ignored, and fields with no column are left alone.
func For[T any](rows *sql.Rows) (func(*T) error, error) {
	if meta.IsValue(reflect.TypeFor[T]()) {
		return func(dst *T) error { return rows.Scan(dst) }, nil
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	model := meta.Of[T]()

	fields := make([]*meta.Field, len(columns))

	for i, c := range columns {
		fields[i] = model.Column(c)
	}

	dest := make([]any, len(columns))

	return func(dst *T) error {
		for i, f := range fields {
			if f == nil {
				dest[i] = new(any)
			} else {
				dest[i] = f.Addr(unsafe.Pointer(dst))
			}
		}

		return rows.Scan(dest...)
	}, nil
}
