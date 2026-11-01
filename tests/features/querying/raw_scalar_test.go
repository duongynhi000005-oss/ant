package querying

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// ant.Raw also scans a single column into a value type.
func TestRawScalar(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		fixture.Seed(t, db)
		ctx := t.Context()

		t.Run("one value", func(t *testing.T) {
			count, err := ant.Raw[int](db, `SELECT count(*) FROM users`).First(ctx)
			require.NoError(t, err)
			require.Equal(t, 3, *count)
		})

		t.Run("a column of values", func(t *testing.T) {
			names, err := ant.Raw[string](db, `SELECT name FROM users ORDER BY name`).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Alice", "Bob", "Carol"}, names)
		})

		t.Run("time.Time is a value, not a model", func(t *testing.T) {
			now, err := ant.Raw[time.Time](db, `SELECT now()`).First(ctx)
			require.NoError(t, err)
			require.False(t, now.IsZero())
		})

		t.Run("NULL needs a nullable type", func(t *testing.T) {
			bio, err := ant.Raw[sql.Null[string]](
				db,
				`SELECT bio FROM users WHERE name = 'Bob'`,
			).First(ctx)
			require.NoError(t, err)
			require.False(t, bio.Valid)

			_, err = ant.Raw[string](db, `SELECT bio FROM users WHERE name = 'Bob'`).First(ctx)
			require.Error(t, err)
		})

		t.Run("several columns do not fit one value", func(t *testing.T) {
			_, err := ant.Raw[int](db, `SELECT 1, 2`).First(ctx)
			require.Error(t, err)
		})
	})
}
