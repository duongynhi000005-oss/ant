package change_tracking

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// hold marks the context.Context of an operation that a holder pauses.
type hold struct{}

// holder pauses the statements of an operation marked with hold until the test
// lets it go, so a second operation can start while the first still runs.
type holder struct {
	mu      sync.Mutex
	started chan struct{}
	let     chan struct{}
}

// Before implements ant.Interceptor.
func (h *holder) Before(ctx context.Context, _ *ant.Statement) (context.Context, error) {
	if ctx.Value(hold{}) == nil {
		return ctx, nil
	}

	h.mu.Lock()
	started, let := h.started, h.let

	select {
	case <-started:
	default:
		close(started)
	}

	h.mu.Unlock()
	<-let

	return ctx, nil
}

// After implements ant.Interceptor.
func (*holder) After(context.Context, *ant.Statement, ant.Outcome) {}

// while starts op in another goroutine, runs during once op is waiting on the
// database, then lets op finish and returns its error.
func (h *holder) while(t *testing.T, op func(ctx context.Context) error, during func()) error {
	t.Helper()

	h.mu.Lock()
	h.started, h.let = make(chan struct{}), make(chan struct{})
	h.mu.Unlock()

	done := make(chan error, 1)

	go func() { done <- op(context.WithValue(t.Context(), hold{}, true)) }()

	select {
	case <-h.started:
	case err := <-done:
		t.Fatalf("the operation ended before it reached the database: %v", err)
	}

	during()
	close(h.let)

	return <-done
}

// byTitle orders included posts.
func byTitle(q *ant.Query[Post]) *ant.Query[Post] { return q.OrderBy(&P.Title) }

// postTitles returns the titles of a user's loaded posts.
func postTitles(u *User) []string {
	out := make([]string, len(u.Posts))

	for i, p := range u.Posts {
		out[i] = p.Title
	}

	return out
}

// A context with change tracking is one unit of work. Two operations on it at
// once would load into the same entities while both change them: two requests
// including the same user's posts would each see the other's posts too. So the
// second operation is refused with ErrConcurrentUse. Give each request its own
// ant.Session, and share a context only with tracking off.
func TestConcurrentUse(t *testing.T) {
	fixture.Each(t, func(t *testing.T, target fixture.Target) {
		h := new(holder)
		db := fixture.Open[AppDB](t, target, ant.WithTracking(true), ant.WithInterceptor(h))
		ctx := t.Context()
		alice, bob, _ := fixture.Seed(t, db)
		require.NoError(t, db.Posts.Insert(ctx,
			&Post{UserID: alice.ID, Title: "A1"},
			&Post{UserID: alice.ID, Title: "A2"},
			&Post{UserID: bob.ID, Title: "B1"}))

		load := func(ctx context.Context) error {
			_, err := db.Users.Include(&U.Posts).All(ctx)

			return err
		}

		t.Run("every tracked operation is refused while another runs", func(t *testing.T) {
			stranger := func() *User { return &User{Name: "Dan", Email: "dan@example.com"} }

			ops := []struct {
				name string
				op   func(ctx context.Context) error
			}{
				{"All", func(ctx context.Context) error {
					_, err := db.Users.All(ctx)

					return err
				}},
				{"First", func(ctx context.Context) error {
					_, err := db.Users.OrderBy(&U.Name).First(ctx)

					return err
				}},
				{"Single", func(ctx context.Context) error {
					_, err := db.Users.Where(ant.Eq(&U.Name, "Alice")).Single(ctx)

					return err
				}},
				{"Query.Find", func(ctx context.Context) error {
					_, err := db.Users.Query().Find(ctx, bob.ID)

					return err
				}},
				{"Page", func(ctx context.Context) error {
					_, err := db.Users.OrderBy(&U.ID).Page(ctx, 1, 2)

					return err
				}},
				{"Chunk", func(ctx context.Context) error {
					return db.Users.Query().Chunk(ctx, 2, func([]*User) error { return nil })
				}},
				{"Include", load},
				{"IncludeOne", func(ctx context.Context) error {
					_, err := db.Posts.IncludeOne(&P.User).All(ctx)

					return err
				}},
				{"IncludeJoined", func(ctx context.Context) error {
					_, err := db.Posts.IncludeJoined(&P.User).All(ctx)

					return err
				}},
				{"Load", func(ctx context.Context) error {
					return db.Users.Load(ctx, &U.Posts, &User{Base: ant.Base{ID: bob.ID}})
				}},
				{"LoadOne", func(ctx context.Context) error {
					return db.Posts.LoadOne(ctx, &P.User, &Post{UserID: bob.ID})
				}},
				{"Insert", func(ctx context.Context) error {
					return db.Users.Insert(ctx, stranger())
				}},
				{"Upsert", func(ctx context.Context) error {
					_, err := db.Users.Upsert(ctx, stranger(), ant.OnConflict(&U.Email).DoNothing())

					return err
				}},
				{"Reload", func(ctx context.Context) error {
					return db.Users.Reload(ctx, &User{Base: ant.Base{ID: bob.ID}})
				}},
				{"SaveChanges", db.SaveChanges},
			}

			for _, tt := range ops {
				t.Run(tt.name, func(t *testing.T) {
					err := h.while(t, load, func() {
						require.ErrorIs(t, tt.op(ctx), ant.ErrConcurrentUse)
					})
					require.NoError(t, err, "the first operation is not disturbed")
				})
			}

			n, err := db.Users.NoTracking().Count(ctx)
			require.NoError(t, err)
			require.EqualValues(t, 3, n, "no refused write ran")
		})

		t.Run("SaveChanges holds the tracker as a load does", func(t *testing.T) {
			bob.Age = 18

			err := h.while(t, db.SaveChanges, func() {
				require.ErrorIs(t, load(ctx), ant.ErrConcurrentUse)
			})
			require.NoError(t, err)
			require.Equal(t, 18, stored(t, db, bob.ID).Age)
		})

		t.Run("a transaction shares its context's tracker", func(t *testing.T) {
			err := h.while(
				t,
				func(ctx context.Context) error {
					return db.Tx(ctx, func(tx *AppDB) error {
						_, err := tx.Users.All(ctx)

						return err
					})
				},
				func() {
					require.ErrorIs(t, load(ctx), ant.ErrConcurrentUse)
				},
			)
			require.NoError(t, err)
		})

		t.Run("work that leaves the tracker alone goes on alongside", func(t *testing.T) {
			err := h.while(t, load, func() {
				n, err := db.Users.Count(ctx)
				require.NoError(t, err)
				require.EqualValues(t, 3, n)

				found, err := db.Users.Where(ant.Eq(&U.Name, "Bob")).Exists(ctx)
				require.NoError(t, err)
				require.True(t, found)

				users, err := db.Users.NoTracking().
					Include(&U.Posts, byTitle).
					OrderBy(&U.Name).
					All(ctx)
				require.NoError(t, err)
				require.Equal(t, []string{"A1", "A2"}, postTitles(users[0]))

				names, err := db.Users.Query().Column(&U.Name).All(ctx)
				require.NoError(t, err)
				require.Len(t, names, 3, "a partial row is never tracked")

				changed, err := db.Users.Where(ant.Eq(&U.Name, "nobody")).
					Update(ctx, ant.Set(&U.Age, 1))
				require.NoError(t, err)
				require.Zero(t, changed)

				session, err := ant.Session(db).Users.Include(&U.Posts).All(ctx)
				require.NoError(t, err, "a session has a tracker of its own")
				require.Len(t, session, 3)
			})
			require.NoError(t, err)
		})

		t.Run("the claim ends with the operation, however it ends", func(t *testing.T) {
			canceled, cancel := context.WithCancel(ctx)
			cancel()

			_, err := db.Users.Include(&U.Posts).All(canceled)
			require.ErrorIs(t, err, context.Canceled)

			_, err = db.Users.Where(ant.SQLCond("no_such_column")).All(ctx)
			require.Error(t, err)

			require.NoError(t, load(ctx))
		})

		t.Run("a query inside the operation shares its claim", func(t *testing.T) {
			users, err := db.Users.Include(&U.Posts, func(q *ant.Query[Post]) *ant.Query[Post] {
				return byTitle(q).IncludeOne(&P.User)
			}).OrderBy(&U.Name).All(ctx)
			require.NoError(t, err, "a nested include")
			require.Equal(t, []string{"A1", "A2"}, postTitles(users[0]))
			require.Equal(t, "Alice", users[0].Posts[0].User.Name)

			seen := 0
			err = db.Users.OrderBy(&U.ID).Chunk(ctx, 2, func(batch []*User) error {
				for _, u := range batch {
					if err := db.Users.Load(ctx, &U.Posts, u); err != nil {
						return err
					}

					seen += len(u.Posts)
				}

				return nil
			})
			require.NoError(t, err, "Chunk lets go between batches")
			require.Equal(t, 3, seen)
		})
	})
}

// Requests that share one context with tracking on get either their own rows or
// ErrConcurrentUse, never another request's. A Session per request, or a shared
// context with tracking off, serves every request.
func TestConcurrentRequests(t *testing.T) {
	const requests = 64

	fixture.Each(t, func(t *testing.T, target fixture.Target) {
		tracked := fixture.Open[AppDB](t, target, ant.WithTracking(true))
		untracked := fixture.Open[AppDB](t, target)

		for _, db := range []*AppDB{tracked, untracked} {
			db.DB().SetMaxOpenConns(8)
		}

		ctx := t.Context()
		alice, bob, carol := fixture.Seed(t, untracked)
		require.NoError(t, untracked.Posts.Insert(ctx,
			&Post{UserID: alice.ID, Title: "A1"},
			&Post{UserID: alice.ID, Title: "A2"},
			&Post{UserID: bob.ID, Title: "B1"}))

		want := map[int64][]string{
			alice.ID: {"A1", "A2"},
			bob.ID:   {"B1"},
			carol.ID: {},
		}

		// request loads every user with posts, then a post with its author, as
		// a handler would. With check, it also reads what it loaded: each
		// relation must hold exactly its rows.
		request := func(db *AppDB, check bool) error {
			users, err := db.Users.Include(&U.Posts, byTitle).All(ctx)
			if err != nil || !check {
				return err
			}

			for _, u := range users {
				if got := postTitles(u); !equalTitles(got, want[u.ID]) {
					return errors.New(u.Name + " has posts " + joinTitles(got))
				}
			}

			posts, err := db.Posts.IncludeOne(&P.User).Where(ant.Eq(&P.Title, "B1")).All(ctx)
			if err != nil {
				return err
			}

			if len(posts) != 1 || posts[0].User == nil || posts[0].User.ID != bob.ID {
				return errors.New("B1 lost its author")
			}

			return nil
		}

		run := func(each func() *AppDB, check bool) (refused int, errs []error) {
			var (
				wg sync.WaitGroup
				mu sync.Mutex
			)

			for range requests {
				wg.Go(func() {
					err := request(each(), check)

					mu.Lock()
					defer mu.Unlock()

					switch {
					case errors.Is(err, ant.ErrConcurrentUse):
						refused++
					case err != nil:
						errs = append(errs, err)
					}
				})
			}

			wg.Wait()

			return refused, errs
		}

		// The requests do not read what they loaded here: one tracking context
		// gives every request the same User for Alice, so reading her posts while
		// the next request includes them again is a data race in the
		// application. That is the reason to share only contexts with tracking off.
		t.Run("one tracking context: refused, never a crash", func(t *testing.T) {
			_, errs := run(func() *AppDB { return tracked }, false)
			require.Empty(t, errs)
		})

		t.Run("a session each: all served", func(t *testing.T) {
			refused, errs := run(func() *AppDB { return ant.Session(tracked) }, true)
			require.Empty(t, errs)
			require.Zero(t, refused)
		})

		t.Run("one context with tracking off: all served", func(t *testing.T) {
			refused, errs := run(func() *AppDB { return untracked }, true)
			require.Empty(t, errs)
			require.Zero(t, refused)
		})

		t.Run("tracked sessions saving at once each save their own", func(t *testing.T) {
			var wg sync.WaitGroup

			errs := make(chan error, requests)

			for i := range requests {
				wg.Go(func() {
					session := ant.Session(tracked)

					users, err := session.Users.Include(&U.Posts).OrderBy(&U.ID).All(ctx)
					if err != nil {
						errs <- err

						return
					}

					users[i%3].Age++
					users[i%3].Posts = append(users[i%3].Posts, Post{Title: "new"})

					errs <- session.SaveChanges(ctx)
				})
			}

			wg.Wait()
			close(errs)

			for err := range errs {
				require.NoError(t, err)
			}

			n, err := untracked.Posts.Where(ant.Eq(&P.Title, "new")).Count(ctx)
			require.NoError(t, err)
			require.EqualValues(t, requests, n, "no post was saved twice or lost")
		})
	})
}

func equalTitles(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func joinTitles(titles []string) string {
	out := "["

	for i, title := range titles {
		if i > 0 {
			out += " "
		}

		out += title
	}

	return out + "]"
}

// An entity passed to Load twice gets its related rows once.
func TestLoadRepeatedParent(t *testing.T) {
	fixture.RunTracked(t, func(t *testing.T, db *AppDB) {
		ctx := t.Context()
		alice, _, _ := fixture.Seed(t, db)
		require.NoError(t, db.Posts.Insert(ctx,
			&Post{UserID: alice.ID, Title: "A1"},
			&Post{UserID: alice.ID, Title: "A2"}))

		require.NoError(t, db.Users.Load(ctx, &U.Posts, alice, alice))
		require.Len(t, alice.Posts, 2)

		posts := []*Post{{UserID: alice.ID}, {UserID: alice.ID}}
		require.NoError(t, db.Posts.LoadOne(ctx, &P.User, posts[0], posts[0], posts[1]))
		require.Equal(t, "Alice", posts[0].User.Name)
		require.Equal(t, "Alice", posts[1].User.Name)
	})
}

// Note queries the database from its hooks, as an audit hook might.
type Note struct {
	ID     int64
	Author string
}

// noteHook is what Note's hooks run, set by each test.
var noteHook func(ctx context.Context) error

func (n *Note) AfterLoad(ctx context.Context) error { return runNoteHook(ctx) }

func (n *Note) BeforeUpdate(ctx context.Context) error { return runNoteHook(ctx) }

// inHook marks a hook's own queries, whose notes would run the hook again.
type inHook struct{}

func runNoteHook(ctx context.Context) error {
	if noteHook == nil || ctx.Value(inHook{}) != nil {
		return nil
	}

	return noteHook(context.WithValue(ctx, inHook{}, true))
}

// NotesDB is a context for Note.
type NotesDB struct {
	ant.Context
	Notes ant.Table[Note]
}

// A hook runs inside the operation that called it. Its queries share that
// operation's claim when they use the context.Context the hook was given. One
// made with another context looks like a second operation, and is refused.
func TestConcurrentUseInHooks(t *testing.T) {
	const ddl = `CREATE TABLE notes (id bigserial PRIMARY KEY, author text NOT NULL)`

	fixture.RunTrackedWith(t, ddl, func(t *testing.T, db *NotesDB) {
		ctx := t.Context()
		t.Cleanup(func() { noteHook = nil })

		require.NoError(t, db.Notes.Insert(ctx, &Note{Author: "ann"}, &Note{Author: "ben"}))

		others := func(ctx context.Context) error {
			_, err := db.Notes.Where(ant.Eq(&N.Author, "ben")).All(ctx)

			return err
		}

		t.Run("with the hook's context", func(t *testing.T) {
			noteHook = others

			notes, err := db.Notes.OrderBy(&N.Author).All(ctx)
			require.NoError(t, err, "AfterLoad inside a load")

			notes[0].Author = "anne"

			require.NoError(t, db.SaveChanges(ctx), "BeforeUpdate inside SaveChanges")
		})

		t.Run("with another context", func(t *testing.T) {
			noteHook = func(context.Context) error { return others(context.Background()) }

			_, err := db.Notes.All(ctx)
			require.ErrorIs(t, err, ant.ErrConcurrentUse)

			noteHook = nil

			require.NoError(t, others(ctx), "the refused load let go")
		})
	})
}

// N is the probe for Note.
var N = ant.Fields[Note]()
