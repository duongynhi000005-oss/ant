package querying

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// Pass sql.Named arguments and write @name in the query.
func TestRawNamed(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		fixture.Seed(t, db)
		ctx := t.Context()

		t.Run("by name", func(t *testing.T) {
			got, err := ant.Raw[fixture.User](
				db,
				`SELECT * FROM users WHERE age > @min AND name <> @skip`,
				sql.Named("min", 18),
				sql.Named("skip", "Alice"),
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Carol"}, fixture.Names(got))
		})

		t.Run("a name can repeat", func(t *testing.T) {
			got, err := ant.Raw[fixture.User](
				db,
				`SELECT * FROM users WHERE age = @n OR age = @n + 23`,
				sql.Named("n", 17),
			).All(ctx)
			require.NoError(t, err)
			require.Len(t, got, 2)
		})

		t.Run("text inside quotes is left alone", func(t *testing.T) {
			got, err := ant.Raw[string](
				db,
				`SELECT '@min' FROM users WHERE age > @min`,
				sql.Named("min", 30),
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"@min"}, got)
		})

		t.Run("works with Exec", func(t *testing.T) {
			n, err := ant.Raw[any](
				db,
				`UPDATE users SET age = @age WHERE name = 'Bob'`,
				sql.Named("age", 18),
			).Exec(ctx)
			require.NoError(t, err)
			require.EqualValues(t, 1, n)
		})

		t.Run("a missing argument", func(t *testing.T) {
			_, err := ant.Raw[int](db, `SELECT @a + @b`, sql.Named("a", 1)).First(ctx)
			require.ErrorContains(t, err, `no argument named "b"`)
		})

		t.Run("an unused argument", func(t *testing.T) {
			_, err := ant.Raw[int](db, `SELECT 1`, sql.Named("a", 1)).First(ctx)
			require.ErrorContains(t, err, `does not use argument "a"`)
		})

		t.Run("named and positional cannot mix", func(t *testing.T) {
			_, err := ant.Raw[int](db, `SELECT @a + $1`, sql.Named("a", 1), 2).First(ctx)
			require.ErrorContains(t, err, "mixes")
		})
	})
}
