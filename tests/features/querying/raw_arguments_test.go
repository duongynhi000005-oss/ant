package querying

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// Raw SQL can use ? instead of $1, and a slice fills an IN list.
func TestRawArguments(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		fixture.Seed(t, db)
		ctx := t.Context()
		ages := []int{17, 30}

		t.Run("? is a placeholder", func(t *testing.T) {
			got, err := ant.Raw[string](
				db,
				`SELECT name FROM users WHERE age > ? AND name <> ?`,
				18,
				"Alice",
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Carol"}, got)
		})

		t.Run("a slice fills IN (?)", func(t *testing.T) {
			got, err := ant.Raw[string](
				db,
				`SELECT name FROM users WHERE age IN (?) ORDER BY name`,
				ages,
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Alice", "Bob"}, got)
		})

		t.Run("a slice fills IN (@name)", func(t *testing.T) {
			got, err := ant.Raw[string](
				db,
				`SELECT name FROM users WHERE age NOT IN (@ages)`,
				sql.Named("ages", ages),
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Carol"}, got)
		})

		t.Run("an empty slice matches nothing", func(t *testing.T) {
			got, err := ant.Raw[string](
				db,
				`SELECT name FROM users WHERE age IN (?)`,
				[]int{},
			).All(ctx)
			require.NoError(t, err)
			require.Empty(t, got)
		})

		// SQL rule: NOT IN (NULL) is never true. ant.NotIn with no values
		// differs, and matches everything.
		t.Run("an empty slice in NOT IN also matches nothing", func(t *testing.T) {
			got, err := ant.Raw[string](
				db,
				`SELECT name FROM users WHERE age NOT IN (?)`,
				[]int{},
			).All(ctx)
			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("outside IN, a slice is one array value", func(t *testing.T) {
			got, err := ant.Raw[string](
				db,
				`SELECT name FROM users WHERE age = ANY($1) ORDER BY name`,
				ages,
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Alice", "Bob"}, got)

			got, err = ant.Raw[string](
				db,
				`SELECT name FROM users WHERE age = ANY(@ages) ORDER BY name`,
				sql.Named("ages", ages),
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Alice", "Bob"}, got)
		})

		t.Run("the jsonb ? operator still works", func(t *testing.T) {
			got, err := ant.Raw[bool](db, `SELECT '{"a": 1}'::jsonb ? $1`, "a").First(ctx)
			require.NoError(t, err)
			require.True(t, *got)

			got, err = ant.Raw[bool](db, `SELECT '{"a": 1}'::jsonb ? 'b'`).First(ctx)
			require.NoError(t, err)
			require.False(t, *got)
		})
	})
}
