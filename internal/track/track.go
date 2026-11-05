// Package track implements the change tracker: identity map, snapshots, and
// diffing. Entities are held by pointer, so the tracker keeps them alive until
// it is cleared.
package track

import (
	"cmp"
	"slices"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/skudra-go/ant/internal/meta"
)

// State is the lifecycle state of an entity.
type State int

// Entity states, matching EF Core's.
const (
	Detached State = iota
	Added
	Unchanged
	Modified
	Deleted
)

// String returns the state's name.
func (s State) String() string {
	return [...]string{"Detached", "Added", "Unchanged", "Modified", "Deleted"}[s]
}

// Entry is one tracked entity.
type Entry struct {
	Model  *meta.Model
	Entity unsafe.Pointer
	// State is Added, Unchanged, Modified, or Deleted. Unchanged means nothing
	// has been marked. Changed tells whether the fields differ.
	State    State
	snapshot unsafe.Pointer
	id       identity
	keyed    bool
	seq      uint64
	// Members holds the entries of a loaded one-to-many relation by key, and
	// Parents the key of a loaded many-to-one relation, to detect changes.
	Members map[*meta.Relation]map[any]*Entry
	Parents map[*meta.Relation]any
}

// Refresh makes the current values the snapshot.
func (e *Entry) Refresh() { e.snapshot = snapshotOf(e.Model, e.Entity) }

// Changed returns the stored fields that differ from the snapshot. The key is
// never included.
func (e *Entry) Changed() []*meta.Field {
	if e.snapshot == nil {
		return nil
	}

	var out []*meta.Field

	for _, f := range e.Model.Updatable() {
		if !equal(f, e.Entity, e.snapshot) {
			out = append(out, f)
		}
	}

	return out
}

// KeyChanged reports whether the primary key differs from the snapshot.
func (e *Entry) KeyChanged() bool {
	if e.snapshot == nil {
		return false
	}

	return slices.ContainsFunc(
		e.Model.Keys,
		func(f *meta.Field) bool { return !equal(f, e.Entity, e.snapshot) },
	)
}

// Holds reports whether the entity at entity is this entry's row. Slices reuse
// memory, so an address alone does not say which row it holds.
func (e *Entry) Holds(entity unsafe.Pointer) bool {
	if !e.keyed {
		return true
	}

	key, ok := e.Model.KeyOf(entity)

	return ok && key == e.id.key
}

// Original returns the snapshot's copy of the entity, or nil if there is none.
func (e *Entry) Original() unsafe.Pointer { return e.snapshot }

// Scratch returns a private copy of the snapshot that a caller may change.
func (e *Entry) Scratch() unsafe.Pointer {
	if e.snapshot == nil {
		return nil
	}

	return snapshotOf(e.Model, e.snapshot)
}

// Saved records a state and snapshot as saved, and returns how to undo that.
func (e *Entry) Saved(state State) (undo func()) {
	oldState, oldSnapshot := e.State, e.snapshot
	e.State = state
	e.Refresh()

	return func() { e.State, e.snapshot = oldState, oldSnapshot }
}

type identity struct {
	model *meta.Model
	key   any
}

// Tracker holds the tracked entities of one context. It is safe for concurrent
// use, but an entity should not change while SaveChanges reads it.
type Tracker struct {
	mu      sync.Mutex
	busy    atomic.Bool
	seq     uint64
	entries map[unsafe.Pointer]*Entry
	first   map[identity]*Entry
	// OnTrack and OnState report entities starting to be tracked and changing
	// state. They run with the tracker locked, so they must not call back into it.
	OnTrack func(*Entry)
	OnState func(e *Entry, from, to State)
}

// Claim reserves the tracker for one operation, and reports false if another
// holds it. Operations that load into the tracker or save from it write into
// the entities it holds, so two at once would write into the same ones.
func (t *Tracker) Claim() bool { return t.busy.CompareAndSwap(false, true) }

// Release ends the operation that claimed the tracker.
func (t *Tracker) Release() { t.busy.Store(false) }

// New returns an empty tracker.
func New() *Tracker {
	return &Tracker{entries: map[unsafe.Pointer]*Entry{}, first: map[identity]*Entry{}}
}

func keyOf(m *meta.Model, entity unsafe.Pointer) (identity, bool) {
	v, ok := m.KeyOf(entity)

	return identity{m, v}, ok
}

func (t *Tracker) add(m *meta.Model, entity unsafe.Pointer, state State) *Entry {
	t.seq++

	e := &Entry{
		Model:  m,
		Entity: entity,
		State:  state,
		seq:    t.seq,
	}

	if state != Added {
		e.Refresh()
	}

	t.entries[entity] = e
	t.identify(e)

	if t.OnTrack != nil {
		t.OnTrack(e)
	}

	return e
}

// identify records the entry's key, if it has one yet. An entity's memory can
// later hold another row, as when a slice shifts, so the key is kept here.
func (t *Tracker) identify(e *Entry) {
	if id, ok := keyOf(e.Model, e.Entity); ok {
		e.id, e.keyed = id, true

		if _, seen := t.first[id]; !seen {
			t.first[id] = e
		}
	}
}

// Load tracks a loaded entity and returns the instance to use: an already
// tracked one with the same key, or entity itself. Models with no key are not
// tracked.
func (t *Tracker) Load(m *meta.Model, entity unsafe.Pointer) unsafe.Pointer {
	t.mu.Lock()
	defer t.mu.Unlock()

	id, ok := keyOf(m, entity)
	if !ok {
		return entity
	}

	if existing, seen := t.first[id]; seen {
		return existing.Entity
	}

	return t.add(m, entity, Unchanged).Entity
}

// Copy tracks a loaded value that lives inside a relation slice. Such a value
// cannot be swapped for an existing instance, so it gets its own entry, and
// changes to either are saved.
func (t *Tracker) Copy(m *meta.Model, entity unsafe.Pointer) *Entry {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(m.Keys) == 0 {
		return nil
	}

	return t.put(m, entity, Unchanged)
}

// Track starts tracking entity in the given state, or changes its state.
func (t *Tracker) Track(m *meta.Model, entity unsafe.Pointer, state State) *Entry {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.put(m, entity, state)
}

func (t *Tracker) put(m *meta.Model, entity unsafe.Pointer, state State) *Entry {
	if e, ok := t.entries[entity]; ok {
		if e.State != state && t.OnState != nil {
			t.OnState(e, e.State, state)
		}

		e.State = state

		return e
	}

	return t.add(m, entity, state)
}

// Rekey records the key of an entity that received one on insert.
func (t *Tracker) Rekey(e *Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.identify(e)
}

// Move records that an entry's entity now lives at a new address, as happens
// when appending to a relation slice reallocates it.
func (t *Tracker) Move(e *Entry, entity unsafe.Pointer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.entries[e.Entity] == e {
		delete(t.entries, e.Entity)
	}

	e.Entity = entity
	t.entries[entity] = e
}

// Get returns the entry for an entity, or nil.
func (t *Tracker) Get(entity unsafe.Pointer) *Entry {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.entries[entity]
}

// Find returns the first tracked instance with the given key, or nil.
func (t *Tracker) Find(m *meta.Model, key any) *Entry {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.first[identity{m, key}]
}

// Remove stops tracking an entity.
func (t *Tracker) Remove(entity unsafe.Pointer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if e, ok := t.entries[entity]; ok {
		t.forget(e)
	}
}

// Forget stops tracking an entry, which may no longer own its address.
func (t *Tracker) Forget(e *Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.forget(e)
}

// ForgetRow stops tracking every instance of a row, and returns how to undo that.
func (t *Tracker) ForgetRow(m *meta.Model, key any) (undo func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	id := identity{m, key}

	var gone []*Entry

	for _, e := range t.entries {
		if e.keyed && e.id == id {
			gone = append(gone, e)
		}
	}

	for _, e := range gone {
		t.forget(e)
	}

	return func() {
		for _, e := range gone {
			t.Restore(e)
		}
	}
}

func (t *Tracker) forget(e *Entry) {
	if t.entries[e.Entity] == e {
		delete(t.entries, e.Entity)
	}

	if e.keyed && t.first[e.id] == e {
		delete(t.first, e.id)
	}
}

// Restore tracks an entry again, undoing Forget.
func (t *Tracker) Restore(e *Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries[e.Entity] = e

	if e.keyed {
		if _, seen := t.first[e.id]; !seen {
			t.first[e.id] = e
		}
	}
}

// Clear stops tracking everything.
func (t *Tracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	clear(t.entries)
	clear(t.first)
}

// Entries returns every entry, oldest first.
func (t *Tracker) Entries() []*Entry {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]*Entry, 0, len(t.entries))

	for _, e := range t.entries {
		out = append(out, e)
	}

	slices.SortFunc(out, func(a, b *Entry) int { return cmp.Compare(a.seq, b.seq) })

	return out
}

// Len returns the number of tracked entities.
func (t *Tracker) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.entries)
}
