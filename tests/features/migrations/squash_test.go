package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant/migrate"
)

// squash replaces a long history with one file. A database that has the old
// migrations records the new file without running it, and a new database runs it,
// so nothing needs coordinating between environments.
func TestSquash(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "create_accounts")
		must[V2DB](p, "diff", "add_bio")
		must[V2DB](p, "up")

		out := must[V2DB](p, "squash")

		t.Run("one file replaces the rest", func(t *testing.T) {
			require.Contains(t, out, "replacing 2 migrations through 0002")
			require.Equal(t, []string{"0003_baseline.up.sql"}, listFiles(p, "versions"))
			text := p.read("versions/0003_baseline.up.sql")
			require.Contains(t, text, "-- ant:replaces 0002")
			require.Contains(t, text, `"bio" text`)
		})

		t.Run("it warns that hand-written SQL is not carried over", func(t *testing.T) {
			require.Contains(t, out, "Warning: a squash keeps what the models describe")
		})

		t.Run("a database with the old files absorbs it", func(t *testing.T) {
			require.Contains(t, must[V2DB](p, "up"), "already has")
			require.Contains(t, must[V2DB](p, "status"), "squashed")
			require.Contains(t, must[V2DB](p, "up"), "Nothing to apply")
			require.Contains(t, must[V2DB](p, "drift"), "No drift")
		})

		t.Run("check still passes", func(t *testing.T) {
			must[V2DB](p, "check")
		})

		t.Run("a new database runs it", func(t *testing.T) {
			p.sql("DROP TABLE accounts; DROP TABLE ant_migrations")
			must[V2DB](p, "up")
			require.Contains(t, p.columns("accounts"), "bio")
			require.Contains(t, must[V2DB](p, "drift"), "No drift")
		})

		t.Run("the next migration follows on", func(t *testing.T) {
			must[V3DB](p, "diff", "more", "--rename", "accounts.name=display_name")
			require.True(t, p.exists("versions/0004_more.up.sql"))
			must[V3DB](p, "up")
		})
	})
}

// A database that has not reached the squash cannot absorb it, and must not run
// it either, since its tables already exist. It has to catch up first.
func TestDatabaseBehindASquash(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "create_accounts")
		must[V2DB](p, "diff", "add_bio")
		must[V2DB](p, "up", "--to", "0001")
		must[V2DB](p, "squash")

		_, err := run[V2DB](p, "up")
		require.ErrorIs(t, err, migrate.ErrBehindSquash)
	})
}

// --keep joins the old files, so what was written by hand survives.
func TestSquashKeep(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "create_accounts")
		must[V1DB](p, "diff", "--empty", "by_hand")
		p.write("versions/0002_by_hand.up.sql", "CREATE TABLE by_hand (id int);")
		must[V1DB](p, "squash", "everything", "--keep")

		text := p.read("versions/0003_everything.up.sql")
		require.Contains(t, text, "CREATE TABLE by_hand")
		require.Contains(t, text, `CREATE TABLE "accounts"`)
		must[V1DB](p, "up")
		require.NotEmpty(t, p.columns("by_hand"))
	})
}

// squash refuses while models are ahead, so nothing is lost silently.
func TestSquashNeedsACleanState(t *testing.T) {
	each(t, func(t *testing.T, p *project) {
		must[V1DB](p, "diff", "start")
		_, err := run[V2DB](p, "squash")
		require.ErrorIs(t, err, migrate.ErrPendingChanges)
	})
}
