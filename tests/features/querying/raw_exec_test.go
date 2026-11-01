package querying

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// Exec runs SQL that returns no rows.
func TestRawExec(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		fixture.Seed(t, db)
		ctx := t.Context()

		t.Run("returns the rows affected", func(t *testing.T) {
			n, err := ant.Raw[any](
				db,
				`UPDATE users SET age = age + 1 WHERE age > $1`,
				18,
			).Exec(ctx)
			require.NoError(t, err)
			require.EqualValues(t, 2, n)
		})

		t.Run("no rows affected is not an error", func(t *testing.T) {
			n, err := ant.Raw[any](db, `DELETE FROM users WHERE age > 1000`).Exec(ctx)
			require.NoError(t, err)
			require.Zero(t, n)
		})

		t.Run("a failure is the database's error", func(t *testing.T) {
			_, err := ant.Raw[any](db, `UPDATE no_such_table SET a = 1`).Exec(ctx)
			require.ErrorContains(t, err, "no_such_table")
		})
	})
}
