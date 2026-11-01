// Package testdb runs ant's integration tests against every database listed in
// ANT_TEST_DATABASES.
package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"  // driver "mysql"
	_ "github.com/jackc/pgx/v5/stdlib"  // driver "pgx"
	_ "github.com/microsoft/go-mssqldb" // driver "sqlserver"
	_ "modernc.org/sqlite"              // driver "sqlite"

	"github.com/skudra-go/ant/dialect"
	"github.com/skudra-go/ant/dialect/mysql"
	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/dialect/sqlite"
	"github.com/skudra-go/ant/dialect/sqlserver"
)

const connectTimeout = 10 * time.Second

// Target is one database in the matrix.
type Target struct {
	Name    string
	Dialect dialect.Dialect
	DSN     string
	DB      *sql.DB // set by Connect
}

type spec struct {
	dialect    dialect.Dialect
	env        string
	defaultDSN string
}

// The default DSNs match docker-compose.yml.
var specs = map[string]spec{
	"postgres": {
		postgres.Dialect,
		"ANT_PG_DSN",
		"postgres://ant:ant@127.0.0.1:55432/ant_test?sslmode=disable",
	},
	"mysql": {
		mysql.Dialect,
		"ANT_MYSQL_DSN",
		"ant:ant@tcp(127.0.0.1:53306)/ant_test?parseTime=true",
	},
	"mariadb": {
		mysql.MariaDB,
		"ANT_MARIADB_DSN",
		"ant:ant@tcp(127.0.0.1:53307)/ant_test?parseTime=true",
	},
	"sqlite": {sqlite.Dialect, "", ""},
	"sqlserver": {
		sqlserver.Dialect,
		"ANT_MSSQL_DSN",
		"sqlserver://sa:Ant_test_1%21@127.0.0.1:51433?database=master&encrypt=disable",
	},
}

// Targets returns the databases named in ANT_TEST_DATABASES, a comma-separated
// list. An unknown name yields a target whose Connect fails the test.
func Targets() []Target {
	var out []Target

	for name := range strings.SplitSeq(os.Getenv("ANT_TEST_DATABASES"), ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		s := specs[name]

		dsn := s.defaultDSN

		if s.env != "" {
			if v := os.Getenv(s.env); v != "" {
				dsn = v
			}
		}

		out = append(out, Target{Name: name, Dialect: s.dialect, DSN: dsn})
	}

	return out
}

// ForEach runs fn once per target, each with a freshly reset schema. It skips
// when no databases are configured, so plain go test needs no database.
func ForEach(t *testing.T, fn func(t *testing.T, target Target)) {
	t.Helper()
	ForOnly(t, nil, fn)
}

// ForOnly is ForEach limited to the named databases, or all of them with no
// names. The others are left alone, not even connected to, because their tables
// are shared with whichever test package is running beside this one.
func ForOnly(t *testing.T, names []string, fn func(t *testing.T, target Target)) {
	t.Helper()

	targets := Targets()

	if len(names) > 0 {
		targets = slices.DeleteFunc(
			targets,
			func(target Target) bool { return !slices.Contains(names, target.Name) },
		)
	}

	if len(targets) == 0 {
		t.Skip("no database in ANT_TEST_DATABASES for this test")
	}

	for _, target := range targets {
		t.Run(target.Name, func(t *testing.T) {
			target.Connect(t)
			target.ResetSchema(t)
			fn(t, target)
		})
	}
}

// Connect opens the target's database. A database named in ANT_TEST_DATABASES
// that cannot be reached fails the test, so that a green run means it ran.
func (target *Target) Connect(t *testing.T) {
	t.Helper()

	if target.Dialect == nil {
		t.Fatalf("unknown database %q in ANT_TEST_DATABASES", target.Name)
	}

	if target.Name == "sqlite" {
		path := filepath.Join(t.TempDir(), "ant.db")
		target.DSN = "file:" + path + "?_pragma=foreign_keys(1)"
	}

	db := target.open(t)

	if target.Name == "postgres" {
		db = target.isolate(t, db)
	}

	target.DB = db
}

func (target *Target) open(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open(target.Dialect.DriverName(), target.DSN)
	if err != nil {
		t.Fatalf("%s: %v", target.Name, err)
	}

	t.Cleanup(func() { db.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), connectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("%s unreachable: %v", target.Name, err)
	}

	return db
}

// isolate moves the test into its own PostgreSQL schema, so test packages can
// run in parallel against one server.
func (target *Target) isolate(t *testing.T, admin *sql.DB) *sql.DB {
	t.Helper()

	schema := "ant_" + rand.Text()

	schema = strings.ToLower(schema)
	if _, err := admin.ExecContext(t.Context(), "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	t.Cleanup(func() {
		if _, err := admin.ExecContext(
			context.Background(),
			"DROP SCHEMA "+schema+" CASCADE",
		); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})

	target.DSN += "&search_path=" + schema

	return target.open(t)
}

// ResetSchema recreates the fixture tables.
func (target *Target) ResetSchema(t *testing.T) {
	t.Helper()

	for _, stmt := range schema(target.Name) {
		if _, err := target.DB.ExecContext(t.Context(), stmt); err != nil {
			t.Fatalf("reset schema: %s: %v", stmt, err)
		}
	}
}
