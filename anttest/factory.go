package anttest

import (
	"sync/atomic"
	"testing"

	"github.com/skudra-go/ant"
)

// Factory makes valid entities for tests, so a test states only what it cares
// about. Define one per model, usually in a package variable:
//
//	var Users = anttest.NewFactory(func(n int) *User {
//		return &User{Name: fmt.Sprintf("user %d", n), Email: fmt.Sprintf("u%d@example.com", n)}
//	})
//
//	admin := Users.With(func(u *User) { u.Admin = true })    // a state
//	user := admin.Create(t, &db.Users)                       // in the database
//	plain := Users.Build()                                   // in memory only
//
// A factory is immutable, like a query: With returns a new one, so states stack.
// Sequence numbers are shared with the factories made from it, so values stay
// unique across states.
type Factory[T any] struct {
	make        func(n int) *T
	sequence    *atomic.Int64
	overrides   []func(*T)
	afterBuild  []func(*T)
	afterCreate []func(*T)
}

// NewFactory returns a factory. make receives a number that goes up by one for
// each entity, for values that must be unique.
func NewFactory[T any](make func(n int) *T) *Factory[T] {
	return &Factory[T]{make: make, sequence: new(atomic.Int64)}
}

func (f *Factory[T]) copy() *Factory[T] {
	out := *f

	return &out
}

// With returns a factory that applies overrides, as a named state or a one-off.
func (f *Factory[T]) With(overrides ...func(*T)) *Factory[T] {
	out := f.copy()
	out.overrides = append(append([]func(*T){}, f.overrides...), overrides...)

	return out
}

// AfterBuild returns a factory that calls fn on each entity it builds.
func (f *Factory[T]) AfterBuild(fn func(*T)) *Factory[T] {
	out := f.copy()
	out.afterBuild = append(append([]func(*T){}, f.afterBuild...), fn)

	return out
}

// AfterCreate returns a factory that calls fn on each entity it inserts, with its
// generated key in place, which is the place to add related rows.
func (f *Factory[T]) AfterCreate(fn func(*T)) *Factory[T] {
	out := f.copy()
	out.afterCreate = append(append([]func(*T){}, f.afterCreate...), fn)

	return out
}

// Build returns an entity without touching a database.
func (f *Factory[T]) Build(overrides ...func(*T)) *T {
	e := f.make(int(f.sequence.Add(1)))

	for _, override := range f.overrides {
		override(e)
	}

	for _, override := range overrides {
		override(e)
	}

	for _, fn := range f.afterBuild {
		fn(e)
	}

	return e
}

// BuildMany returns n entities.
func (f *Factory[T]) BuildMany(n int, overrides ...func(*T)) []*T {
	out := make([]*T, n)
	for i := range out {
		out[i] = f.Build(overrides...)
	}

	return out
}

// Create builds an entity and inserts it, and its related entities, through table.
// A failure stops the test.
func (f *Factory[T]) Create(t testing.TB, table *ant.Table[T], overrides ...func(*T)) *T {
	t.Helper()

	e := f.Build(overrides...)
	if err := table.Insert(t.Context(), e); err != nil {
		t.Fatalf("anttest: create: %v", err)
	}

	for _, fn := range f.afterCreate {
		fn(e)
	}

	return e
}

// CreateMany inserts n entities.
func (f *Factory[T]) CreateMany(
	t testing.TB,
	table *ant.Table[T],
	n int,
	overrides ...func(*T),
) []*T {
	t.Helper()

	out := make([]*T, n)
	for i := range out {
		out[i] = f.Create(t, table, overrides...)
	}

	return out
}

// Recycled shares one parent among many children, so a hundred posts do not
// create a hundred authors. Get creates it on first use.
type Recycled[T any] struct {
	factory *Factory[T]
	table   *ant.Table[T]
	value   *T
}

// Recycle returns a shared parent made by f.
func Recycle[T any](f *Factory[T], table *ant.Table[T]) *Recycled[T] {
	return &Recycled[T]{factory: f, table: table}
}

// Get returns the shared entity.
func (r *Recycled[T]) Get(t testing.TB) *T {
	t.Helper()

	if r.value == nil {
		r.value = r.factory.Create(t, r.table)
	}

	return r.value
}
