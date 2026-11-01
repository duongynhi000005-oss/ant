package querying

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

type label struct{ Text string }

var Lb = ant.Fields[label]()

// Expressions compute values in SQL, with their Go type checked.
func TestExpressions(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		fixture.Seed(t, db)
		ctx := t.Context()
		bob := db.Users.Where(ant.Eq(&U.Name, "Bob"))
		text := func(t *testing.T, e ant.Expr[string]) string {
			t.Helper()

			got, err := ant.Project[label](bob, ant.IntoExpr(&Lb.Text, e)).First(ctx)
			require.NoError(t, err)

			return got.Text
		}

		t.Run("Case", func(t *testing.T) {
			kind := ant.Case[string]().When(ant.Lt(&U.Age, 18), "minor").Else("adult")
			require.Equal(t, "minor", text(t, kind))
		})

		t.Run("Lower, Upper, Coalesce", func(t *testing.T) {
			require.Equal(t, "bob", text(t, ant.Lower(&U.Name)))
			require.Equal(t, "BOB", text(t, ant.Upper(&U.Name)))
		})

		t.Run("Func calls any database function", func(t *testing.T) {
			require.Equal(
				t,
				"B0b",
				text(t, ant.Func[string]("REPLACE", ant.Col(&U.Name), "o", "0")),
			)
		})

		t.Run("SQL mixes columns and bound values", func(t *testing.T) {
			require.Equal(t, "Bob/x", text(t, ant.SQL[string]("? || ?", ant.Col(&U.Name), "/x")))
		})

		t.Run("a bound value is never text in the query", func(t *testing.T) {
			require.Equal(
				t,
				"'); DROP TABLE users; --",
				text(t, ant.SQL[string]("?::text", "'); DROP TABLE users; --")),
			)
		})

		t.Run("Collate", func(t *testing.T) {
			got, err := db.Users.Order(ant.Collate(&U.Name, "C").Desc()).First(ctx)
			require.NoError(t, err)
			require.Equal(t, "Carol", got.Name)
		})

		t.Run("expressions in conditions", func(t *testing.T) {
			got, err := db.Users.Where(ant.Lower(&U.Name).Eq("bob")).All(ctx)
			require.NoError(t, err)
			require.Len(t, got, 1)

			got, err = db.Users.Where(ant.Col(&U.Age).Plus(1).Eq(18)).All(ctx)
			require.NoError(t, err)
			require.Len(t, got, 1)

			got, err = db.Users.Where(ant.Length(&U.Name).Gte(5)).OrderBy(&U.Name).All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Alice", "Carol"}, fixture.Names(got))
		})

		t.Run("SQLCond", func(t *testing.T) {
			got, err := db.Users.Where(ant.SQLCond("? % ? = 0", ant.Col(&U.Age), 10)).
				OrderBy(&U.Name).
				All(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"Alice", "Carol"}, fixture.Names(got))
		})

		t.Run("too many arguments for the fragment", func(t *testing.T) {
			_, err := db.Users.Where(ant.SQLCond("age > 1", 5)).All(ctx)
			require.ErrorContains(t, err, "unused")
		})

		t.Run("ILike ignores case", func(t *testing.T) {
			got, err := db.Users.Where(ant.ILike(&U.Name, "ALICE")).All(ctx)
			require.NoError(t, err)
			require.Len(t, got, 1)
		})

		t.Run("AnyOf, AllOf, NoneOf look across columns", func(t *testing.T) {
			got, err := db.Users.Where(ant.AnyOf("Bob", &U.Name, &U.Email)).All(ctx)
			require.NoError(t, err)
			require.Len(t, got, 1)

			got, err = db.Users.Where(ant.AllOf("Bob", &U.Name, &U.Email)).All(ctx)
			require.NoError(t, err)
			require.Empty(t, got)

			got, err = db.Users.Where(ant.NoneOf("Bob", &U.Name, &U.Email)).All(ctx)
			require.NoError(t, err)
			require.Len(t, got, 2)
		})
	})
}

// A CASE branch can be a column, with WhenExpr and ElseExpr.
func TestCaseWithExpressions(t *testing.T) {
	fixture.Run(t, func(t *testing.T, db *fixture.AppDB) {
		ctx := t.Context()
		fixture.Seed(t, db)

		label := ant.Case[string]().
			WhenExpr(ant.Lt(&U.Age, 18), ant.SQL[string]("'minor'")).
			ElseExpr(ant.Col(&U.Name))

		type row struct{ Label string }

		r := ant.Fields[row]()
		rows, err := ant.Project[row](
			db.Users.OrderBy(&U.Age),
			ant.IntoExpr(&r.Label, label),
		).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []row{{"minor"}, {"Alice"}, {"Carol"}}, rows)
	})
}
