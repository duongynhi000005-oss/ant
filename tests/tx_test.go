package tests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant/internal/testdb"
)

func TestTxCommits(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		err := app.Tx(t.Context(), func(tx *AppDB) error {
			bob := &User{Name: "Bob", Email: "bob@example.com"}
			if err := tx.Users.Insert(t.Context(), bob); err != nil {
				return err
			}

			return tx.Posts.Insert(t.Context(), &Post{UserID: bob.ID, Title: "First!"})
		})
		require.NoError(t, err)
		require.Equal(t, 1, countRows(t, target.DB, "users"))
		require.Equal(t, 1, countRows(t, target.DB, "posts"))
	})
}

func TestTxRollsBack(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		boom := errors.New("boom")
		err := app.Tx(t.Context(), func(tx *AppDB) error {
			if err := tx.Users.Insert(
				t.Context(),
				&User{Name: "Bob", Email: "bob@example.com"},
			); err != nil {
				return err
			}

			return boom
		})
		require.ErrorIs(t, err, boom)
		require.Zero(t, countRows(t, target.DB, "users"))
	})
}
