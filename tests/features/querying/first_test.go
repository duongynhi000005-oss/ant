package querying

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// First returns one row, or ant.ErrNotFound.
func TestFirst(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		fixture.Seed(t, db)
		ctx := t.Context()

		t.Run("the first in order", func(t *testing.T) {
			user, err := db.Users.OrderByDesc(&U.Age).First(ctx)
			require.NoError(t, err)
			require.Equal(t, "Carol", user.Name)
		})

		t.Run("no match", func(t *testing.T) {
			user, err := db.Users.Where(ant.Eq(&U.Name, "Nobody")).First(ctx)
			require.ErrorIs(t, err, ant.ErrNotFound)
			require.Nil(t, user)
		})

		t.Run("Offset skips rows first", func(t *testing.T) {
			user, err := db.Users.OrderBy(&U.Name).Offset(1).First(ctx)
			require.NoError(t, err)
			require.Equal(t, "Bob", user.Name)
		})

		t.Run("Limit zero finds nothing", func(t *testing.T) {
			_, err := db.Users.OrderBy(&U.Name).Limit(0).First(ctx)
			require.ErrorIs(t, err, ant.ErrNotFound)
		})
	})
}
