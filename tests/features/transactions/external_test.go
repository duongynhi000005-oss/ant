package transactions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// UseTx joins a transaction started elsewhere, such as by other libraries. The
// caller decides how it ends.
func TestUseTx(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		ctx := t.Context()
		tx, err := db.DB().BeginTx(ctx, nil)
		require.NoError(t, err)
		bound, err := ant.UseTx(db, tx)
		require.NoError(t, err)

		require.NoError(
			t,
			bound.Users.Insert(ctx, &fixture.User{Name: "Temp", Email: "temp@example.com"}),
		)

		require.NoError(t, tx.Rollback())

		count, err := db.Users.Count(ctx)
		require.NoError(t, err)
		require.Zero(t, count)
	})
}

// A second context type can join the same transaction, which is how two parts of
// a program with different contexts commit together.
type OtherDB struct {
	ant.Context
	Posts ant.Table[fixture.Post]
}

func TestShareATransaction(t *testing.T) {
	fixture.Each(t, func(t *testing.T, target fixture.Target) {
		ctx := t.Context()
		db := fixture.Open[fixture.AppDB](t, target)
		other := fixture.Open[OtherDB](t, target)
		alice, _, _ := fixture.Seed(t, db)

		err := db.Tx(ctx, func(tx *fixture.AppDB) error {
			shared := ant.CurrentTx(tx)
			require.NotNil(t, shared)
			joined, err := ant.UseTx(other, shared)
			require.NoError(t, err)

			require.NoError(
				t,
				joined.Posts.Insert(ctx, &fixture.Post{UserID: alice.ID, Title: "together"}),
			)

			return ant.ErrNotFound // rolls back both
		})
		require.ErrorIs(t, err, ant.ErrNotFound)
		count, err := other.Posts.Count(ctx)
		require.NoError(t, err)
		require.Zero(t, count)

		t.Run("outside a transaction there is none", func(t *testing.T) {
			require.Nil(t, ant.CurrentTx(db))
		})
	})
}

// UseConn keeps everything on one connection, for a session setting or lock.
func TestUseConn(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		ctx := t.Context()
		conn, err := db.DB().Conn(ctx)
		require.NoError(t, err)

		defer conn.Close()

		bound, err := ant.UseConn(db, conn)
		require.NoError(t, err)

		_, err = ant.Raw[any](bound, "SET application_name = 'ant-test'").Exec(ctx)
		require.NoError(t, err)
		name, err := ant.Raw[string](bound, "SHOW application_name").First(ctx)
		require.NoError(t, err)
		require.Equal(t, "ant-test", *name)

		t.Run("transactions stay on it", func(t *testing.T) {
			require.NoError(t, bound.Tx(ctx, func(tx *fixture.AppDB) error {
				name, err := ant.Raw[string](tx, "SHOW application_name").First(ctx)
				require.NoError(t, err)
				require.Equal(t, "ant-test", *name)

				return nil
			}))
		})

		_, err = ant.Raw[any](bound, "RESET application_name").Exec(ctx)
		require.NoError(t, err)
	})
}
