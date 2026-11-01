package keys_and_identity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant/tests/features/fixture"
)

// A non-zero key is written as given, even on a generated key.
func TestExplicitKeyValue(t *testing.T) {
	ddl := `CREATE TABLE tickets (id BIGSERIAL PRIMARY KEY, title TEXT NOT NULL);
	        CREATE TABLE seats (number INT PRIMARY KEY, row TEXT NOT NULL)`
	fixture.RunWith(t, ddl, func(t *testing.T, db *DeskDB) {
		ctx := t.Context()

		t.Run("the value is kept", func(t *testing.T) {
			ticket := &Ticket{ID: 500, Title: "Imported"}
			require.NoError(t, db.Tickets.Insert(ctx, ticket))
			require.EqualValues(t, 500, ticket.ID)

			found, err := db.Tickets.Find(ctx, 500)
			require.NoError(t, err)
			require.Equal(t, "Imported", found.Title)
		})

		t.Run("a duplicate is the database's error", func(t *testing.T) {
			err := db.Tickets.Insert(ctx, &Ticket{ID: 500, Title: "Again"})
			require.ErrorContains(t, err, "duplicate key")
		})

		t.Run("generated and explicit keys mix in one call", func(t *testing.T) {
			generated, explicit := &Ticket{Title: "G"}, &Ticket{ID: 501, Title: "E"}
			require.NoError(t, db.Tickets.Insert(ctx, generated, explicit))
			require.NotZero(t, generated.ID)
			require.EqualValues(t, 501, explicit.ID)
		})
	})
}
