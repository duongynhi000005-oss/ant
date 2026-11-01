package querying

import (
	"database/sql"
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

//go:embed testdata/*.sql
var queries embed.FS

// ant.RawFile loads a statement from any fs.FS, including go:embed.
func TestRawFile(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		fixture.Seed(t, db)
		ctx := t.Context()

		t.Run("runs the file", func(t *testing.T) {
			got, err := ant.RawFile[fixture.User](
				db,
				queries,
				"testdata/adults.sql",
				sql.Named("min", 18),
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Alice", "Carol"}, fixture.Names(got))
		})

		t.Run("a missing file is reported at the terminal call", func(t *testing.T) {
			query := ant.RawFile[fixture.User](db, queries, "testdata/none.sql")
			_, err := query.All(ctx)
			require.ErrorIs(t, err, fs.ErrNotExist)
			_, err = query.Exec(ctx)
			require.ErrorIs(t, err, fs.ErrNotExist)
		})
	})
}
