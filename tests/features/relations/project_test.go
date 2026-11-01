package relations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

type Summary struct {
	Author string
	Title  string
}

var S = ant.Fields[Summary]()

// Project selects only the mapped columns, where Select fetches whole rows.
func TestProject(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		seed(t, db)
		ctx := t.Context()

		t.Run("from a join", func(t *testing.T) {
			got, err := ant.Project[Summary](
				db.Posts.Join(&P.User).Where(ant.Eq(&P.Published, true)).OrderBy(&P.Title),
				ant.Into(&S.Author, &U.Name),
				ant.Into(&S.Title, &P.Title),
			).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []Summary{{"Bob", "Bob's post"}, {"Alice", "Hello"}}, got)
		})

		t.Run("from one table", func(t *testing.T) {
			got, err := ant.Project[Summary](
				db.Users.Where(ant.Eq(&U.Name, "Bob")),
				ant.Into(&S.Author, &U.Name),
			).First(ctx)
			require.NoError(t, err)
			require.Equal(t, Summary{Author: "Bob"}, *got)
		})

		t.Run("an expression", func(t *testing.T) {
			got, err := ant.Project[Summary](db.Users.Where(ant.Eq(&U.Name, "Bob")),
				ant.IntoExpr(&S.Author, ant.SQL[string]("upper(?)", ant.Col(&U.Name)))).First(ctx)
			require.NoError(t, err)
			require.Equal(t, "BOB", got.Author)
		})

		t.Run("no mappings", func(t *testing.T) {
			_, err := ant.Project[Summary](db.Users.Where()).All(ctx)
			require.ErrorIs(t, err, ant.ErrNoColumns)
		})

		t.Run("dst must be a probe of the result type", func(t *testing.T) {
			_, err := ant.Project[Summary](db.Users.Where(), ant.Into(&U.Name, &U.Name)).All(ctx)
			require.ErrorIs(t, err, ant.ErrWrongModel)
		})

		t.Run("src must be in the query", func(t *testing.T) {
			_, err := ant.Project[Summary](db.Users.Where(), ant.Into(&S.Title, &P.Title)).All(ctx)
			require.ErrorIs(t, err, ant.ErrWrongModel)
		})

		t.Run("a NULL needs a nullable field", func(t *testing.T) {
			type row struct{ Bio sql.Null[string] }

			R := ant.Fields[row]()
			got, err := ant.Project[row](db.Users.Where(ant.Eq(&U.Name, "Bob")),
				ant.IntoExpr(&R.Bio, ant.SQL[sql.Null[string]]("?", ant.Col(&U.Bio)))).First(ctx)
			require.NoError(t, err)
			require.False(t, got.Bio.Valid)
		})
	})
}
