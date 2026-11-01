package writing

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// Reload replaces an entity's columns with what is stored now.
func TestReload(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		alice, _, carol := fixture.Seed(t, db)
		ctx := t.Context()

		t.Run("picks up changes", func(t *testing.T) {
			_, err := db.Users.Where(ant.Eq(&U.ID, alice.ID)).Update(ctx, ant.Set(&U.Age, 31))
			require.NoError(t, err)
			require.NoError(t, db.Users.Reload(ctx, alice))
			require.Equal(t, 31, alice.Age)
		})

		t.Run("discards unsaved changes", func(t *testing.T) {
			alice.Name = "Changed in memory"
			require.NoError(t, db.Users.Reload(ctx, alice))
			require.Equal(t, "Alice", alice.Name)
		})

		t.Run("a NULL clears a pointer", func(t *testing.T) {
			_, err := db.Users.Where(ant.Eq(&U.ID, carol.ID)).Update(ctx, ant.Set(&U.Bio, nil))
			require.NoError(t, err)
			require.NoError(t, db.Users.Reload(ctx, carol))
			require.Nil(t, carol.Bio)
		})

		t.Run("relations are left alone", func(t *testing.T) {
			alice.Posts = []fixture.Post{{Title: "In memory"}}
			require.NoError(t, db.Users.Reload(ctx, alice))
			require.Len(t, alice.Posts, 1)
			alice.Posts = nil
		})

		t.Run("the row is gone", func(t *testing.T) {
			require.NoError(t, db.Users.DeleteByID(ctx, alice.ID))
			require.ErrorIs(t, db.Users.Reload(ctx, alice), ant.ErrNotFound)
		})

		t.Run("a nil entity", func(t *testing.T) {
			require.ErrorIs(t, db.Users.Reload(ctx, nil), ant.ErrNilEntity)
		})
	})
}
