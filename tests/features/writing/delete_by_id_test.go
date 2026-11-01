package writing

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// DeleteByID removes one row by primary key.
func TestDeleteByID(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		alice, _, _ := fixture.Seed(t, db)
		ctx := t.Context()

		t.Run("removes the row", func(t *testing.T) {
			require.NoError(t, db.Users.DeleteByID(ctx, alice.ID))
			_, err := db.Users.Find(ctx, alice.ID)
			require.ErrorIs(t, err, ant.ErrNotFound)
		})

		t.Run("no such row", func(t *testing.T) {
			require.ErrorIs(t, db.Users.DeleteByID(ctx, alice.ID), ant.ErrNotFound)
		})

		t.Run("a key of the wrong type", func(t *testing.T) {
			require.ErrorIs(t, db.Users.DeleteByID(ctx, "1"), ant.ErrKeyType)
		})
	})
}
