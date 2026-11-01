package tests

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/dialect"
	"github.com/skudra-go/ant/internal/testdb"
)

// These run on every database, and each covers something the dialects write
// differently. Where a database lacks a feature, the test asks for the error
// instead, because a clear refusal is part of what a dialect promises.

func openAndSeed(t *testing.T, target testdb.Target) *AppDB {
	t.Helper()
	app := openApp(t, target)
	seedUsers(t, app, "alice", 30, "bob", 25, "carol", 35)

	return app
}

func has(
	target testdb.Target,
	f dialect.Feature,
) bool {
	return dialect.Supports(target.Dialect, f)
}

func TestConformancePaging(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()

		page, err := app.Users.OrderBy(&U.Name).Limit(2).Offset(1).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"bob", "carol"}, names(page))

		rest, err := app.Users.OrderBy(&U.Name).Offset(2).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"carol"}, names(rest), "OFFSET alone")

		some, err := app.Users.Limit(2).All(ctx)
		require.NoError(t, err)
		require.Len(t, some, 2, "LIMIT with no ORDER BY")

		first, err := app.Users.OrderByDesc(&U.Age).First(ctx)
		require.NoError(t, err)
		require.Equal(t, "carol", first.Name)
	})
}

func TestConformanceConditions(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()

		found, err := app.Users.Where(ant.ILike(&U.Name, "ALI%")).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"alice"}, names(found), "ILIKE")

		found, err = app.Users.Where(ant.Between(&U.Age, 26, 34)).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"alice"}, names(found))

		found, err = app.Users.Where(ant.InTuples(
			[]ant.FieldRef{ant.F(&U.Name), ant.F(&U.Age)},
			[]any{"alice", 30},
			[]any{"bob", 99},
		)).
			All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"alice"}, names(found), "multi-column IN")

		exists, err := app.Users.Where(ant.Eq(&U.Name, "bob")).Exists(ctx)
		require.NoError(t, err)
		require.True(t, exists)

		exists, err = app.Users.Where(ant.Eq(&U.Name, "nobody")).Exists(ctx)
		require.NoError(t, err)
		require.False(t, exists)
	})
}

func TestConformanceNullsOrder(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()
		bio := "hello"
		require.NoError(
			t,
			app.Users.Insert(ctx, &User{Name: "with", Email: "with@example.com", Bio: &bio}),
		)

		require.NoError(
			t,
			app.Users.Insert(ctx, &User{Name: "without", Email: "without@example.com"}),
		)

		found, err := app.Users.Order(ant.Col(&U.Bio).Asc().NullsFirst()).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"without", "with"}, names(found))

		found, err = app.Users.Order(ant.Col(&U.Bio).Asc().NullsLast()).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"with", "without"}, names(found))
	})
}

func TestConformanceAggregates(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()

		type row struct {
			Total, Adults, Young int64
			Longest              int
		}

		R := ant.Fields[row]()
		rows, err := ant.Project[row](app.Users.Where(ant.Gte(&U.Age, 0)),
			ant.IntoExpr(&R.Total, ant.Count()),
			ant.IntoExpr(&R.Adults, ant.Count().Filter(ant.Gte(&U.Age, 30))),
			ant.IntoExpr(&R.Young, ant.CountOf(&U.Age).Filter(ant.Lt(&U.Age, 30))),
		).All(ctx)
		require.NoError(t, err)
		require.Equal(t, row{Total: 3, Adults: 2, Young: 1}, rows[0], "FILTER")

		lengths, err := ant.Project[row](app.Users.Where(ant.Eq(&U.Name, "carol")),
			ant.IntoExpr(&R.Longest, ant.Length(&U.Name))).All(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, lengths[0].Longest, "LENGTH")
	})
}

func TestConformanceSets(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		found, err := app.Users.Where(ant.Eq(&U.Name, "alice")).
			Union(app.Users.Where(ant.Eq(&U.Name, "bob"))).All(ctx)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"alice", "bob"}, names(found))
	})
}

// Wanted is a row that exists only in memory, to join against.
type Wanted struct {
	Name string
	Rank int
}

var W = ant.Fields[Wanted]()

func TestConformanceValuesSource(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		wanted := ant.Values(app, Wanted{"alice", 1}, Wanted{"carol", 2})

		listed, err := wanted.OrderBy(&W.Rank).All(ctx)
		require.NoError(t, err)
		require.Equal(t, "carol", listed[1].Name)

		found, err := ant.JoinSub(app.Users.OrderBy(&U.Name), wanted, ant.EqCol(&U.Name, &W.Name)).
			Select(func(u *User, w *Wanted) string { return u.Name }).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"alice", "carol"}, found)
	})
}

func TestConformanceBulkInsertKeys(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()

		users := []*User{
			{Name: "a", Email: "a@x"},
			{Name: "b", Email: "b@x"},
			{Name: "c", Email: "c@x"},
		}
		n, err := app.Users.InsertMany(ctx, users)
		require.NoError(t, err)
		require.EqualValues(t, 3, n)

		for _, u := range users {
			stored, err := app.Users.Find(ctx, u.ID)
			require.NoError(t, err)
			require.Equal(t, u.Name, stored.Name, "the key belongs to this row")
		}

		// SaveChanges batches these where it can.
		more := []*User{{Name: "d", Email: "d@x"}, {Name: "e", Email: "e@x"}}
		for _, u := range more {
			require.NoError(t, ant.Add(app, u))
		}

		require.NoError(t, app.SaveChanges(ctx))

		for _, u := range more {
			stored, err := app.Users.Find(ctx, u.ID)
			require.NoError(t, err)
			require.Equal(t, u.Name, stored.Name)
			require.False(t, stored.CreatedAt.IsZero())
		}
	})
}

func TestConformanceUpsert(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()

		conflict := ant.OnConflict(&U.Email).DoUpdate(ant.F(&U.Age))

		if !has(target, dialect.OnConflict) {
			_, err := app.Users.Upsert(ctx, &User{Name: "a", Email: "a@x", Age: 1}, conflict)
			require.ErrorIs(t, err, ant.ErrUnsupported)

			return
		}

		first := &User{Name: "first", Email: "same@x", Age: 1}
		inserted, err := app.Users.Upsert(ctx, first, conflict)
		require.NoError(t, err)
		require.True(t, inserted)
		require.NotZero(t, first.ID)

		second := &User{Name: "second", Email: "same@x", Age: 2}
		_, err = app.Users.Upsert(ctx, second, conflict)
		require.NoError(t, err)
		require.Equal(t, first.ID, second.ID, "the stored row came back")
		require.Equal(t, "first", second.Name)

		stored, err := app.Users.NoTracking().Where(ant.Eq(&U.ID, first.ID)).First(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, stored.Age)

		third := &User{Name: "third", Email: "same@x", Age: 3}
		wrote, err := app.Users.Upsert(ctx, third, ant.OnConflict(&U.Email).DoNothing())
		require.NoError(t, err)
		require.False(t, wrote)
		require.Zero(t, third.ID)

		n, err := app.Users.InsertMany(
			ctx,
			[]*User{{Name: "fourth", Email: "same@x"}, {Name: "fifth", Email: "new@x"}},
			ant.WithConflict(ant.OnConflict(&U.Email).DoNothing()),
		)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
	})
}

func TestConformanceReturning(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()

		updated, err := app.Users.Where(ant.Eq(&U.Name, "bob")).
			UpdateReturning(ctx, ant.Set(&U.Age, 26))

		if has(target, dialect.ReturningUpdate) {
			require.NoError(t, err)
			require.Equal(t, 26, updated[0].Age)
		} else {
			require.ErrorIs(t, err, ant.ErrUnsupported)
		}

		removed, err := app.Users.Where(ant.Eq(&U.Name, "bob")).DeleteReturning(ctx)

		if has(target, dialect.ReturningDelete) {
			require.NoError(t, err)
			require.Equal(t, "bob", removed[0].Name)
		} else {
			require.ErrorIs(t, err, ant.ErrUnsupported)
		}
	})
}

func TestConformanceLimitedWrites(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		changed, err := app.Users.Where(ant.Gt(&U.Age, 0)).
			OrderBy(&U.Age).
			Limit(1).
			Update(ctx, ant.Set(&U.Age, 99))
		require.NoError(t, err)
		require.EqualValues(t, 1, changed)

		youngest, err := app.Users.NoTracking().Where(ant.Eq(&U.Name, "bob")).First(ctx)
		require.NoError(t, err)
		require.Equal(t, 99, youngest.Age)

		removed, err := app.Users.Where(ant.Gt(&U.Age, 0)).OrderBy(&U.Name).Limit(2).Delete(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 2, removed)

		left, err := app.Users.Count(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 1, left)
	})
}

func TestConformanceLocks(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		err := app.Tx(t.Context(), func(tx *AppDB) error {
			_, err := tx.Users.Where(ant.Eq(&U.Name, "bob")).ForUpdate().First(t.Context())

			return err
		})

		if has(target, dialect.RowLocks) {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, ant.ErrUnsupported)
		}
	})
}

func TestConformanceMissingFeatures(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		_, err := app.Users.FullJoinOn[Post](ant.EqCol(&U.ID, &P.UserID)).
			Select(func(u *User, p *Post) string { return u.Name }).All(ctx)

		if has(target, dialect.FullJoin) {
			require.NoError(t, err)

			return
		}

		require.ErrorIs(t, err, ant.ErrUnsupported)
		require.Contains(t, err.Error(), target.Dialect.Name(), "the error names the database")
	})
}

func TestConformanceSavepoints(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()
		err := app.Tx(ctx, func(tx *AppDB) error {
			require.NoError(t, tx.Users.Insert(ctx, &User{Name: "kept", Email: "kept@x"}))

			inner := tx.Tx(ctx, func(nested *AppDB) error {
				require.NoError(
					t,
					nested.Users.Insert(ctx, &User{Name: "undone", Email: "undone@x"}),
				)

				return sql.ErrNoRows
			})
			require.ErrorIs(t, inner, sql.ErrNoRows)

			return tx.Users.Insert(ctx, &User{Name: "after", Email: "after@x"})
		})
		require.NoError(t, err)
		found, err := app.Users.OrderBy(&U.Name).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"after", "kept"}, names(found))
	})
}

func TestConformanceErrors(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()

		err := app.Users.Insert(ctx, &User{Name: "copy", Email: "alice@example.com"})
		require.Error(t, err)
		require.True(t, ant.IsUniqueViolation(err), "%v", err)

		err = app.Posts.Insert(ctx, &Post{UserID: 999999, Title: "orphan"})
		require.Error(t, err)
		require.True(t, ant.IsForeignKeyViolation(err), "%v", err)
	})
}

func TestConformanceTimes(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()
		before := time.Now().Add(-time.Minute)
		u := &User{Name: "t", Email: "t@x"}
		require.NoError(t, app.Users.Insert(ctx, u))
		stored, err := app.Users.Find(ctx, u.ID)
		require.NoError(t, err)
		require.WithinDuration(t, time.Now(), stored.CreatedAt, 2*time.Minute)

		found, err := app.Users.Where(ant.Gt(&U.CreatedAt, before)).Count(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 1, found, "a time compares as a time")
	})
}

func TestConformanceQuoting(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()
		tricky := `it's "quoted" ` + "`and`" + ` [bracketed]`
		u := &User{Name: tricky, Email: strings.Repeat("e", 40) + "@x"}
		require.NoError(t, app.Users.Insert(ctx, u))
		stored, err := app.Users.Where(ant.Eq(&U.Name, tricky)).First(ctx)
		require.NoError(t, err)
		require.Equal(t, tricky, stored.Name)
	})
}

func TestConformanceDates(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()

		// An hour away, as CURRENT_TIMESTAMP has whole seconds on some databases.
		for _, shift := range []time.Duration{-time.Hour, time.Hour} {
			_, err := app.Users.Where(ant.Gt(&U.Age, 0)).
				Update(ctx, ant.Set(&U.CreatedAt, time.Now().Add(shift)))
			require.NoError(t, err)
			past, err := app.Users.Where(ant.Past(&U.CreatedAt)).Count(ctx)
			require.NoError(t, err)
			future, err := app.Users.Where(ant.Future(&U.CreatedAt)).Count(ctx)
			require.NoError(t, err)

			if shift < 0 {
				require.EqualValues(t, []int64{3, 0}, []int64{past, future})
			} else {
				require.EqualValues(t, []int64{0, 3}, []int64{past, future})
			}
		}

		_, err := app.Users.Where(ant.Today(&U.CreatedAt)).Count(ctx)

		if has(target, dialect.DateCast) {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, ant.ErrUnsupported)
		}

		type row struct{ Year float64 }

		R := ant.Fields[row]()

		years, err := ant.Project[row](
			app.Users.Limit(1),
			ant.IntoExpr(&R.Year, ant.DatePart(&U.CreatedAt, "year")),
		).All(ctx)

		if has(target, dialect.DatePart) {
			require.NoError(t, err)
			require.InDelta(t, time.Now().Year(), years[0].Year, 1)
		} else {
			require.ErrorIs(t, err, ant.ErrUnsupported)
		}
	})
}

func TestConformanceRandomOrder(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		found, err := app.Users.Query().OrderByRandom().All(t.Context())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"alice", "bob", "carol"}, names(found))
	})
}

func TestConformanceCursorPaging(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		first, err := app.Users.OrderBy(&U.ID).After(ctx, "", 2)
		require.NoError(t, err)
		require.Equal(t, []string{"alice", "bob"}, names(first.Items))
		next, err := app.Users.OrderBy(&U.ID).After(ctx, first.Next, 2)
		require.NoError(t, err)
		require.Equal(t, []string{"carol"}, names(next.Items))
	})
}

func TestConformanceGroups(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()
		seedUsers(t, app, "a", 20, "b", 20, "c", 30)

		type bucket struct {
			Age   int
			Users int64
		}

		B := ant.Fields[bucket]()
		got, err := ant.Project[bucket](
			app.Users.GroupBy(&U.Age).Having(ant.Count().Gt(1)),
			ant.Into(&B.Age, &U.Age),
			ant.IntoExpr(&B.Users, ant.Count()),
		).All(ctx)
		require.NoError(t, err)
		require.Equal(t, []bucket{{20, 2}}, got)
	})
}

func TestConformanceWindows(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)

		type row struct {
			Name     string
			Position int64
		}

		R := ant.Fields[row]()
		got, err := ant.Project[row](
			app.Users.OrderBy(&U.Name),
			ant.Into(&R.Name, &U.Name),
			ant.IntoExpr(
				&R.Position,
				ant.RowNumber().Over(ant.OrderedBy(ant.Col(&U.Age).Asc())),
			),
		).All(t.Context())
		require.NoError(t, err)
		require.Equal(t, []row{{"alice", 2}, {"bob", 1}, {"carol", 3}}, got)
	})
}

func TestConformanceRelations(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		users, err := app.Users.OrderBy(&U.Name).NoTracking().All(ctx)
		require.NoError(t, err)

		for i, title := range []string{"one", "two", "three"} {
			require.NoError(
				t,
				app.Posts.Insert(ctx, &Post{UserID: users[0].ID, Title: title, Published: i > 0}),
			)
		}

		writers, err := app.Users.Where(ant.Has(&U.Posts, ant.Eq(&P.Published, true))).
			NoTracking().
			All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"alice"}, names(writers), "EXISTS with a boolean")

		top, err := app.Users.Where(ant.Eq(&U.Name, "alice")).NoTracking().
			Include(&U.Posts, func(q *ant.Query[Post]) *ant.Query[Post] {
				return q.OrderByDesc(&P.Title).Limit(2)
			}).
			All(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"two", "three"}, titles(top[0].Posts), "a limit per parent")
	})
}

func TestConformanceMerge(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		source := ant.Values(app, Wanted{"alice", 41})
		merge := ant.Merge(&app.Users, source, ant.EqCol(&U.Name, &W.Name)).
			WhenMatched(ant.SetExpr(&U.Age, ant.Col(&W.Rank)))
		changed, err := merge.Exec(ctx)

		if !has(target, dialect.Merge) {
			require.ErrorIs(t, err, ant.ErrUnsupported)

			return
		}

		if target.Name == "postgres" {
			version, err := ant.Raw[int](
				app,
				"SELECT current_setting('server_version_num')::int",
			).First(ctx)
			require.NoError(t, err)

			if *version < 150000 {
				t.Skip("MERGE needs PostgreSQL 15")
			}
		}

		require.NoError(t, err)
		require.EqualValues(t, 1, changed)

		alice, err := app.Users.NoTracking().Where(ant.Eq(&U.Name, "alice")).First(ctx)
		require.NoError(t, err)
		require.Equal(t, 41, alice.Age)
	})
}

// More rows than any database takes parameters for in one statement: SQL Server
// stops at 2100, so inserts and loads have to split their work.
func TestConformanceParameterLimits(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openApp(t, target)
		ctx := t.Context()

		users := make([]*User, 2500)
		for i := range users {
			users[i] = &User{Name: "u", Email: fmt.Sprintf("u%d@x", i), Age: 20}
		}

		_, err := app.Users.InsertMany(ctx, users)
		require.NoError(t, err)

		require.NoError(
			t,
			app.Posts.Insert(ctx, &Post{UserID: users[len(users)-1].ID, Title: "last"}),
		)

		loaded, err := app.Users.NoTracking().OrderBy(&U.ID).Include(&U.Posts).All(ctx)
		require.NoError(t, err)
		require.Len(t, loaded, len(users))
		require.Equal(t, []string{"last"}, titles(loaded[len(loaded)-1].Posts))
	})
}

func TestConformanceLateralJoin(t *testing.T) {
	testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
		app := openAndSeed(t, target)
		ctx := t.Context()
		alice, err := app.Users.NoTracking().Where(ant.Eq(&U.Name, "alice")).First(ctx)
		require.NoError(t, err)

		for _, title := range []string{"older", "newer"} {
			require.NoError(t, app.Posts.Insert(ctx, &Post{UserID: alice.ID, Title: title}))
		}

		first := app.Posts.Where(ant.EqCol(&P.UserID, &U.ID)).OrderBy(&P.Title).Limit(1)

		newest, err := ant.JoinLateral(app.Users.Query(), first).
			Select(func(u *User, p *Post) string { return p.Title }).
			All(ctx)

		if !has(target, dialect.LateralJoin) {
			require.ErrorIs(t, err, ant.ErrUnsupported)

			return
		}

		require.NoError(t, err)
		require.Equal(t, []string{"newer"}, newest)
	})
}
