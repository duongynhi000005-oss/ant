package dialect_test

import (
	"testing"

	"github.com/skudra-go/ant/dialect"
	"github.com/skudra-go/ant/dialect/mysql"
	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/dialect/sqlite"
	"github.com/skudra-go/ant/dialect/sqlserver"
)

func TestDialects(t *testing.T) {
	cases := []struct {
		d                         dialect.Dialect
		name, driver, ph2, quoted string
		returning                 bool
	}{
		{postgres.Dialect, "postgres", "pgx", "$2", `"a""b"`, true},
		{mysql.Dialect, "mysql", "mysql", "?", "`a\"b`", false},
		{mysql.MariaDB, "mariadb", "mysql", "?", "`a\"b`", true},
		{sqlite.Dialect, "sqlite", "sqlite", "?", `"a""b"`, true},
		{sqlserver.Dialect, "sqlserver", "sqlserver", "@p2", `[a"b]`, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.d.Name(); got != c.name {
				t.Errorf("Name = %q", got)
			}

			if got := c.d.DriverName(); got != c.driver {
				t.Errorf("DriverName = %q", got)
			}

			if got := c.d.Placeholder(2); got != c.ph2 {
				t.Errorf("Placeholder = %q", got)
			}

			if got := c.d.Quote(`a"b`); got != c.quoted {
				t.Errorf("Quote = %q", got)
			}

			if got := c.d.SupportsReturning(); got != c.returning {
				t.Errorf("SupportsReturning = %v", got)
			}
		})
	}
}

func TestQuoteEscapesClosingDelimiter(t *testing.T) {
	if got := sqlserver.Dialect.Quote("a]b"); got != "[a]]b]" {
		t.Errorf("got %q", got)
	}

	if got := mysql.Dialect.Quote("a`b"); got != "`a``b`" {
		t.Errorf("got %q", got)
	}
}
