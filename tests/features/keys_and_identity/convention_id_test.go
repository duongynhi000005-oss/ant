package keys_and_identity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// A field named ID is the primary key, whether declared or embedded.
type Tag struct {
	ID   int32
	Name string
}

// A model with no ID has no key, which is fine until a key operation needs one.
type Event struct {
	Name string
}

type TagDB struct {
	ant.Context
	Tags   ant.Table[Tag]
	Events ant.Table[Event]
}

func TestConventionID(t *testing.T) {
	ddl := `CREATE TABLE tags (id SERIAL PRIMARY KEY, name TEXT NOT NULL);
	        CREATE TABLE events (name TEXT NOT NULL)`
	fixture.RunWith(t, ddl, func(t *testing.T, db *TagDB) {
		ctx := t.Context()

		t.Run("ID is the key", func(t *testing.T) {
			tag := &Tag{Name: "go"}
			require.NoError(t, db.Tags.Insert(ctx, tag))
			found, err := db.Tags.Find(ctx, tag.ID)
			require.NoError(t, err)
			require.Equal(t, "go", found.Name)
		})

		t.Run("a model with no key still inserts and queries", func(t *testing.T) {
			require.NoError(t, db.Events.Insert(ctx, &Event{Name: "login"}))
			events, err := db.Events.All(ctx)
			require.NoError(t, err)
			require.Len(t, events, 1)
		})

		t.Run("key operations need a key", func(t *testing.T) {
			_, err := db.Events.Find(ctx, 1)
			require.ErrorIs(t, err, ant.ErrNoKey)
			require.ErrorIs(t, db.Events.DeleteByID(ctx, 1), ant.ErrNoKey)
			require.ErrorIs(t, db.Events.Reload(ctx, &Event{}), ant.ErrNoKey)
		})
	})
}
