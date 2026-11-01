package transactions

import (
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// Attempts reruns a transaction that lost a serialization conflict or deadlock.
func TestRetry(t *testing.T) {
	fixture.RunTracked(t, func(t *testing.T, db *AppDB) {
		ctx := t.Context()

		t.Run("conflicting transactions all finish", func(t *testing.T) {
			counter := user("counter")
			require.NoError(t, db.Users.Insert(ctx, counter))

			const workers = 8

			var (
				wg   sync.WaitGroup
				runs atomic.Int32
			)

			errs := make(chan error, workers)

			for range workers {
				wg.Go(func() {
					errs <- ant.Session(db).Tx(
						ctx,
						func(tx *AppDB) error {
							runs.Add(1)

							u, err := tx.Users.NoTracking().Find(ctx, counter.ID)
							if err != nil {
								return err
							}

							_, err = tx.Users.Where(ant.Eq(&U.ID, counter.ID)).
								Update(ctx, ant.Set(&U.Age, u.Age+1))

							return err
						},
						ant.Isolation(sql.LevelSerializable),
						ant.Attempts(50),
					)
				})
			}

			wg.Wait()
			close(errs)

			for err := range errs {
				require.NoError(t, err)
			}

			got, err := db.Users.NoTracking().Find(ctx, counter.ID)
			require.NoError(t, err)
			require.Equal(t, workers, got.Age, "no update was lost")
			require.GreaterOrEqual(t, int(runs.Load()), workers)
		})

		t.Run("without Attempts a conflict is returned", func(t *testing.T) {
			require.True(t, ant.IsRetryable(serializationFailure(t, db)))
		})

		t.Run("an ordinary error is not retried", func(t *testing.T) {
			runs := 0
			err := db.Tx(
				ctx,
				func(tx *AppDB) error {
					runs++

					return errStop
				},
				ant.Attempts(5),
			)
			require.ErrorIs(t, err, errStop)
			require.Equal(t, 1, runs)
		})
	})
}

// serializationFailure forces one conflict between two transactions.
func serializationFailure(t *testing.T, db *AppDB) error {
	t.Helper()
	ctx := t.Context()
	row := user("conflict")
	require.NoError(t, db.Users.Insert(ctx, row))
	one, err := ant.Begin(ctx, ant.Session(db), ant.Isolation(sql.LevelSerializable))
	require.NoError(t, err)

	defer one.Rollback()

	two, err := ant.Begin(ctx, ant.Session(db), ant.Isolation(sql.LevelSerializable))
	require.NoError(t, err)

	defer two.Rollback()

	_, err = one.Users.NoTracking().Find(ctx, row.ID)
	require.NoError(t, err)
	_, err = two.Users.NoTracking().Find(ctx, row.ID)
	require.NoError(t, err)
	_, err = one.Users.Where(ant.Eq(&U.ID, row.ID)).Update(ctx, ant.Set(&U.Age, 1))
	require.NoError(t, err)
	require.NoError(t, one.Commit())

	_, err = two.Users.Where(ant.Eq(&U.ID, row.ID)).Update(ctx, ant.Set(&U.Age, 2))

	return err
}
