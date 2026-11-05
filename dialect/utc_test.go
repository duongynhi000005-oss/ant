package dialect_test

import (
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/skudra-go/ant/dialect"
	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/dialect/sqlite"
	"github.com/skudra-go/ant/dialect/sqlserver"
	"github.com/skudra-go/ant/qb"
)

// stamp is a driver.Valuer that holds a time, as ant.DeletedAt does.
type stamp struct{ at time.Time }

func (s stamp) Value() (driver.Value, error) { return s.at, nil }

// pointerStamp has a pointer receiver, so a nil one must stay NULL.
type pointerStamp struct{ at time.Time }

func (s *pointerStamp) Value() (driver.Value, error) { return s.at, nil }

// A database without zones gets every time in UTC, whatever holds it, from
// queries and from raw SQL alike. Other databases get the time as it was.
func TestTimesBindInUTC(t *testing.T) {
	east := time.FixedZone("UTC+3", 3*60*60)
	at := time.Date(2026, 1, 2, 15, 4, 5, 0, east)
	utc := at.UTC()

	var nilTime *time.Time

	var nilStamp *pointerStamp

	cases := []struct {
		name string
		in   any
		want any
	}{
		{"time.Time", at, utc},
		{"*time.Time", &at, utc},
		{"nil *time.Time", nilTime, nilTime},
		{"sql.Null", sql.Null[time.Time]{V: at, Valid: true}, utc},
		{"invalid sql.Null", sql.Null[time.Time]{V: at}, sql.Null[time.Time]{V: at}},
		{"sql.NullTime", sql.NullTime{Time: at, Valid: true}, utc},
		{"a Valuer", stamp{at}, utc},
		{"a nil pointer Valuer", nilStamp, nilStamp},
		{"a secret keeps its cover", qb.Secret{V: at}, qb.Secret{V: utc}},
		{"not a time", 42, 42},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for _, d := range []dialect.Dialect{sqlite.Dialect, sqlserver.Dialect} {
				_, args, err := d.Render(&qb.Select{
					Columns: []qb.Selected{{Value: qb.Param{Value: tt.in}}},
				})
				if err != nil {
					t.Fatal(err)
				}

				if !same(args[0], tt.want) {
					t.Errorf("%s rendered %#v, want %#v", d.Name(), args[0], tt.want)
				}

				_, args, err = dialect.Bind(d, "SELECT ?", []any{tt.in})
				if err != nil {
					t.Fatal(err)
				}

				if !same(args[0], tt.want) {
					t.Errorf("%s bound %#v, want %#v", d.Name(), args[0], tt.want)
				}

				_, args, err = dialect.Bind(d, "SELECT @at", []any{sql.Named("at", tt.in)})
				if err != nil {
					t.Fatal(err)
				}

				if !same(args[0], tt.want) {
					t.Errorf("%s bound by name %#v, want %#v", d.Name(), args[0], tt.want)
				}
			}

			_, args, err := postgres.Dialect.Render(&qb.Select{
				Columns: []qb.Selected{{Value: qb.Param{Value: tt.in}}},
			})
			if err != nil {
				t.Fatal(err)
			}

			if !same(args[0], tt.in) {
				t.Errorf("postgres changed %#v to %#v", tt.in, args[0])
			}
		})
	}
}

// same compares bound values, times by instant and location.
func same(a, b any) bool {
	if x, ok := a.(time.Time); ok {
		y, ok := b.(time.Time)

		return ok && x.Equal(y) && x.Location() == y.Location()
	}

	if x, ok := a.(qb.Secret); ok {
		y, ok := b.(qb.Secret)

		return ok && same(x.V, y.V)
	}

	return a == b
}
