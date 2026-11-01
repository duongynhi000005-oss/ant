package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		name   string
		script string
		want   []string
	}{
		{"statements", "SELECT 1; SELECT 2;", []string{"SELECT 1", "SELECT 2"}},
		{"no final semicolon", "SELECT 1", []string{"SELECT 1"}},
		{"strings", "SELECT 'a;b'; SELECT 'it''s;'", []string{"SELECT 'a;b'", "SELECT 'it''s;'"}},
		{"escape strings", `SELECT E'a\';b'; SELECT 2`, []string{`SELECT E'a\';b'`, "SELECT 2"}},
		{"identifiers", `SELECT "a;b"; SELECT 2`, []string{`SELECT "a;b"`, "SELECT 2"}},
		{
			"line comments stay with their statement",
			"-- one; two\nSELECT 1;",
			[]string{"-- one; two\nSELECT 1"},
		},
		{
			"block comments nest",
			"/* a; /* b; */ c; */ SELECT 1;",
			[]string{"/* a; /* b; */ c; */ SELECT 1"},
		},
		{"only comments is nothing", "-- nothing here\n/* or here */\n", nil},
		{"trailing comment is dropped", "SELECT 1;\n-- done\n", []string{"SELECT 1"}},
		{
			"dollar quoted bodies",
			"CREATE FUNCTION f() RETURNS int AS $$ BEGIN; RETURN 1; END; $$ LANGUAGE plpgsql; SELECT 1",
			[]string{
				"CREATE FUNCTION f() RETURNS int AS $$ BEGIN; RETURN 1; END; $$ LANGUAGE plpgsql",
				"SELECT 1",
			},
		},
		{
			"tagged dollar quotes",
			"SELECT $a$ $$ ; $a$; SELECT 2",
			[]string{"SELECT $a$ $$ ; $a$", "SELECT 2"},
		},
		{"parameters are not quotes", "SELECT $1; SELECT $2", []string{"SELECT $1", "SELECT $2"}},
		{
			"parentheses",
			"CREATE RULE r AS ON INSERT TO t DO ALSO (SELECT 1; SELECT 2); SELECT 3",
			[]string{"CREATE RULE r AS ON INSERT TO t DO ALSO (SELECT 1; SELECT 2)", "SELECT 3"},
		},
		{
			"standard SQL function bodies",
			"CREATE FUNCTION f() RETURNS int BEGIN ATOMIC SELECT 1; SELECT CASE WHEN true THEN 1 END; END; SELECT 2",
			[]string{
				"CREATE FUNCTION f() RETURNS int BEGIN ATOMIC SELECT 1; SELECT CASE WHEN true THEN 1 END; END",
				"SELECT 2",
			},
		},
		{
			"unterminated string ends the script",
			"SELECT 'oops; SELECT 2",
			[]string{"SELECT 'oops; SELECT 2"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, impl{}.Split(c.script))
		})
	}
}

func TestSameExpression(t *testing.T) {
	same := [][2]string{
		// The same check, before and after a dump and restore.
		{
			`((status)::text = ANY ((ARRAY['new'::character varying, 'gone'::character varying])::text[]))`,
			`((status)::text = ANY (ARRAY[('new'::character varying)::text, ('gone'::character varying)::text]))`,
		},
		{`'draft'::text`, `'draft'::character varying`},
		{`('now'::text)::timestamp without time zone`, `'now'::timestamp without time zone`},
		{`(0)::numeric`, `0`},
		{`'x'::character varying(10)`, `'x'`},
		{`'{}'::text[]`, `'{}'`},
		{`AGE > 0`, `age > 0`},
		{`a::"MyType" > 0`, `a > 0`},
		{`x::double precision > 1`, `x > 1`},
	}

	for _, pair := range same {
		require.True(t, impl{}.SameExpression(pair[0], pair[1]), pair[0])
	}

	different := [][2]string{
		{`'a'::text`, `'b'::text`},
		{`'A'::text`, `'a'::text`}, // case matters inside a string
		{`'a b'`, `'ab'`},          // and so does space
		{`age > 0`, `age > 1`},
		{`age > 0`, `age >= 0`},
		{`now()`, `0`},
		{`(status)::text = 'a'`, `(kind)::text = 'a'`},
	}

	for _, pair := range different {
		require.False(t, impl{}.SameExpression(pair[0], pair[1]), pair[0])
	}
}
