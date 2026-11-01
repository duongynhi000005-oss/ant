package querying

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// ant.EqCol compares two columns of the same type.
func TestEqCol(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		ctx := t.Context()
		twin := &fixture.User{Name: "same", Email: "same", Age: 1}
		other := &fixture.User{Name: "a", Email: "b", Age: 2}
		require.NoError(t, db.Users.Insert(ctx, twin, other))

		t.Run("matches", func(t *testing.T) {
			got, err := db.Users.Where(ant.EqCol(&U.Name, &U.Email)).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"same"}, fixture.Names(got))
		})

		t.Run("Not inverts it", func(t *testing.T) {
			got, err := db.Users.Where(ant.Not(ant.EqCol(&U.Name, &U.Email))).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"a"}, fixture.Names(got))
		})

		t.Run("both columns must be in the query", func(t *testing.T) {
			_, err := db.Users.Where(ant.EqCol(&U.ID, &P.UserID)).All(ctx)
			require.ErrorIs(t, err, ant.ErrWrongModel)
		})
	})
}
