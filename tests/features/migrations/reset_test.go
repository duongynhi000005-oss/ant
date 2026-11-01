package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant/migrate"
)

// reset deletes everything and applies all migrations, which suits a developer's
// own database. It asks for the database's name, so it cannot happen by accident.
func TestReset(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "start")
		must[V1DB](p, "up")

		p.sql(
			"INSERT INTO accounts (created_at, updated_at, name, email) VALUES (now(), now(), 'ada', 'ada@example.com')",
		)

		p.sql("CREATE TABLE stray (id int)")
		database := scalar[string](p, "SELECT current_database()")

		t.Run("with no terminal and no confirmation, it refuses", func(t *testing.T) {
			_, err := run[V1DB](p, "reset")
			require.ErrorIs(t, err, migrate.ErrUnconfirmed)
			require.ErrorContains(t, err, "--confirm "+database)
			require.Equal(t, 1, scalar[int](p, "SELECT count(*) FROM accounts"))
		})

		t.Run("the wrong name is refused", func(t *testing.T) {
			_, err := run[V1DB](p, "reset", "--confirm", "production")
			require.ErrorIs(t, err, migrate.ErrUnconfirmed)
			require.Equal(t, 1, scalar[int](p, "SELECT count(*) FROM accounts"))
		})

		t.Run("the right name wipes it and starts again", func(t *testing.T) {
			must[V1DB](p, "reset", "--confirm", database)
			require.Equal(t, 0, scalar[int](p, "SELECT count(*) FROM accounts"))
			require.Empty(t, p.columns("stray"))
			require.Contains(t, must[V1DB](p, "drift"), "No drift")
		})
	})
}

// A helpful failure beats a stack trace for the common mistakes.
func TestCommandErrors(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		t.Run("no name for diff", func(t *testing.T) {
			_, err := run[V1DB](p, "diff")
			require.ErrorContains(t, err, "diff needs a name")
		})

		t.Run("a name that cleans up to nothing", func(t *testing.T) {
			_, err := run[V1DB](p, "diff", "...")
			require.ErrorContains(t, err, "give the migration a name")
		})

		t.Run("an unknown command", func(t *testing.T) {
			_, err := run[V1DB](p, "uppp")
			require.ErrorContains(t, err, `unknown command "uppp"`)
		})

		t.Run("too many names", func(t *testing.T) {
			_, err := run[V1DB](p, "diff", "add", "bio")
			require.ErrorContains(t, err, "diff takes one name, and got: add bio")
		})

		t.Run("a misspelled directive is not ignored", func(t *testing.T) {
			p.write("versions/0001_typo.up.sql", "-- ant:no-transactions\nSELECT 1;")
			_, err := run[V1DB](p, "up")
			require.ErrorIs(t, err, migrate.ErrBadMigration)
			require.ErrorContains(t, err, "unknown directive ant:no-transactions")
			require.NoError(t, removeFile(p, "versions/0001_typo.up.sql"))
		})

		t.Run("no database", func(t *testing.T) {
			p.config.DSN = ""
			_, err := run[V1DB](p, "up")
			require.ErrorContains(t, err, "pass --dsn, or set DATABASE_URL")
		})

		t.Run("an unknown flag", func(t *testing.T) {
			_, err := run[V1DB](p, "up", "--nope")
			require.ErrorContains(t, err, "not defined")
		})
	})
}
