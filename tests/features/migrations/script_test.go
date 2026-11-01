package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant/migrate"
)

// script exports everything as one file, for a database ant cannot reach. It can
// run any number of times, and shares its record with up.
func TestIdempotentScript(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "create_accounts")

		p.write(
			"versions/0002_quotes.up.sql",
			"CREATE TABLE quoted (t text DEFAULT 'it''s; $$ fine');\nCREATE FUNCTION two() RETURNS int AS $$ SELECT 2; $$ LANGUAGE sql;\n",
		)

		p.write("code/view.sql", "CREATE OR REPLACE VIEW v AS SELECT 1 AS n;")
		text := must[V1DB](p, "script")

		t.Run("it can run more than once", func(t *testing.T) {
			p.sql(text)
			p.sql(text)
			require.NotEmpty(t, p.columns("accounts"))
			require.NotEmpty(t, p.columns("v"))
			require.Equal(t, 2, scalar[int](p, "SELECT two()"))
			require.Equal(t, 3, scalar[int](p, "SELECT count(*) FROM ant_migrations"))
		})

		t.Run("up agrees that everything is done", func(t *testing.T) {
			require.Contains(t, must[V1DB](p, "up"), "Nothing to apply")
		})

		t.Run("a changed repeatable file runs again", func(t *testing.T) {
			p.write("code/view.sql", "CREATE OR REPLACE VIEW v AS SELECT 1 AS n, 2 AS m;")
			p.sql(must[V1DB](p, "script"))
			require.Contains(t, p.columns("v"), "m")
			require.Contains(t, must[V1DB](p, "up"), "Nothing to apply")
		})
	})
}

// A migration that needs to run outside a transaction cannot sit in a DO block.
func TestScriptRefusesNoTransaction(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		p.write("versions/0001_index.up.sql", "-- ant:no-transaction\nSELECT 1;")
		_, err := run[V1DB](p, "script")
		require.ErrorIs(t, err, migrate.ErrBadMigration)
		require.ErrorContains(t, err, "--plain")
	})
}

// --plain is the way out: plain SQL, in order, for any other tool or a person.
func TestPlainScript(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "start")
		text := must[V1DB](p, "script", "--plain")
		require.Contains(t, text, `CREATE TABLE "accounts"`)
		require.NotContains(t, text, "ant_migrations")
		p.sql(text)
		require.NotEmpty(t, p.columns("accounts"))
	})
}

// schema-dump prints the whole schema, with no database.
func TestSchemaDump(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		p.config.DSN = ""
		require.Contains(t, must[V3DB](p, "schema-dump"), `CREATE TABLE "notes"`)

		t.Run("--snapshot uses the snapshot", func(t *testing.T) {
			_, err := run[V3DB](p, "schema-dump", "--snapshot")
			require.ErrorContains(t, err, "no snapshot yet")
			must[V1DB](p, "diff", "start")
			text := must[V3DB](p, "schema-dump", "--snapshot")
			require.NotContains(t, text, "notes")
		})
	})
}

// inspect prints what the models describe, with no database.
func TestInspect(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		p.config.DSN = ""
		out := must[V3DB](p, "inspect")
		require.Contains(t, out, "accounts")
		require.Contains(t, out, "display_name text NOT NULL")
		require.Contains(t, out, "foreign key (account_id) to accounts (id)")

		t.Run("one table", func(t *testing.T) {
			require.NotContains(t, must[V3DB](p, "inspect", "notes"), "display_name")
			_, err := run[V3DB](p, "inspect", "ghosts")
			require.ErrorContains(t, err, "no table ghosts")
		})
	})
}

// db-show summarizes a live database.
func TestDBShow(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "start")
		must[V1DB](p, "up")
		out := must[V1DB](p, "db-show")
		require.Contains(t, out, "on postgres")
		require.Contains(t, out, "accounts")
	})
}
