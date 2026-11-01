package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/migrate"
)

// diff compares the structs with the snapshot, and writes the difference as SQL.
// It needs no database, so it works on a laptop with nothing running.
func TestDiff(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		p.config.DSN = "" // proves no connection is needed

		t.Run("the first migration creates everything", func(t *testing.T) {
			out := must[V1DB](p, "diff", "create_accounts")
			require.Contains(t, out, "0001_create_accounts.up.sql")
			require.Contains(t, out, "create table accounts")

			require.Contains(
				t,
				p.read("versions/0001_create_accounts.up.sql"),
				`CREATE TABLE "accounts"`,
			)

			require.True(t, p.exists("versions/schema.snapshot.json"))
		})

		t.Run("no change writes nothing", func(t *testing.T) {
			require.Contains(t, must[V1DB](p, "diff", "nothing"), "No changes")
			require.False(t, p.exists("versions/0002_nothing.up.sql"))
		})

		t.Run("a change writes only the difference", func(t *testing.T) {
			must[V2DB](p, "diff", "add_bio_and_age")
			text := p.read("versions/0002_add_bio_and_age.up.sql")
			require.Contains(t, text, `ALTER TABLE "accounts" ADD COLUMN "bio" text;`)
			require.NotContains(t, text, "CREATE TABLE")
		})

		t.Run("a new NOT NULL column fills existing rows with its zero value", func(t *testing.T) {
			text := p.read("versions/0002_add_bio_and_age.up.sql")
			require.Contains(
				t,
				text,
				`ALTER TABLE "accounts" ADD COLUMN "age" bigint NOT NULL DEFAULT 0;`,
			)

			require.Contains(t, text, `ALTER TABLE "accounts" ALTER COLUMN "age" DROP DEFAULT;`)
			require.Contains(t, text, "Change the DEFAULT below")
		})
	})
}

// A new name is a drop and an add, and only a person knows which was meant. So
// diff warns and shows the rename to use, and --rename generates it.
func TestRenames(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V2DB](p, "diff", "start")

		t.Run("without --rename, the file warns and shows the rename", func(t *testing.T) {
			out, err := run[V3DB](p, "diff", "preview", "--dry-run")
			require.NoError(t, err)
			require.Contains(t, out, "column accounts.name was removed and display_name was added")

			require.Contains(
				t,
				out,
				`--   ALTER TABLE "accounts" RENAME COLUMN "name" TO "display_name";`,
			)

			require.Contains(t, out, `ALTER TABLE "accounts" DROP COLUMN "name";`)
		})

		t.Run("--dry-run wrote nothing", func(t *testing.T) {
			require.False(t, p.exists("versions/0002_preview.up.sql"))
		})

		t.Run("--rename keeps the data", func(t *testing.T) {
			must[V3DB](p, "diff", "rename_name", "--rename", "accounts.name=display_name")
			text := p.read("versions/0002_rename_name.up.sql")
			require.Contains(
				t,
				text,
				`ALTER TABLE "accounts" RENAME COLUMN "name" TO "display_name";`,
			)

			require.NotContains(t, text, "DROP COLUMN")
		})
	})
}

// diff --empty writes a file for changes models cannot express. The models and
// the snapshot stay as they are.
func TestEmptyMigration(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "start")
		must[V1DB](p, "diff", "--empty", "add_extension")
		require.Contains(t, p.read("versions/0002_add_extension.up.sql"), "Written by hand")

		t.Run("flags can follow the name", func(t *testing.T) {
			must[V1DB](p, "diff", "add_type", "--empty")
			require.True(t, p.exists("versions/0003_add_type.up.sql"))
		})

		t.Run("a file left empty is refused, not recorded as done", func(t *testing.T) {
			_, err := run[V1DB](p, "up")
			require.ErrorIs(t, err, migrate.ErrBadMigration)
			require.ErrorContains(t, err, "0002_add_extension has no statements")
		})
	})
}

// A diff writes SQL with no database, from nothing but the types.
func TestDiffStoresTheSnapshotInOrder(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V3DB](p, "diff", "start")
		first := p.read("versions/schema.snapshot.json")
		require.Contains(t, first, `"latest": "0001_start"`)
		require.Contains(t, first, `"dialect": "postgres"`)

		schema, err := ant.SchemaOf[V3DB](postgres.Dialect)
		require.NoError(t, err)
		require.Len(t, schema.Tables, 2)
	})
}
