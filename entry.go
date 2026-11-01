package ant

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/skudra-go/ant/internal/meta"
	"github.com/skudra-go/ant/internal/track"
)

// State is the lifecycle state of an entity.
type State = track.State

// Entity states.
const (
	// Detached entities are not tracked.
	Detached = track.Detached
	// Added entities are inserted by the next SaveChanges.
	Added = track.Added
	// Unchanged entities match what was loaded.
	Unchanged = track.Unchanged
	// Modified entities are updated by the next SaveChanges.
	Modified = track.Modified
	// Deleted entities are removed by the next SaveChanges.
	Deleted = track.Deleted
)

// Add schedules a new entity for insertion by the next SaveChanges.
func Add[T any](h Handle, entity *T) error { return mark(h, entity, track.Added) }

// Attach tracks an entity as it is, assuming it matches the database.
func Attach[T any](h Handle, entity *T) error { return mark(h, entity, track.Unchanged) }

// MarkModified makes the next SaveChanges write every field of an entity. Use it
// for a disconnected entity whose earlier values are unknown.
func MarkModified[T any](h Handle, entity *T) error { return mark(h, entity, track.Modified) }

// Remove schedules an entity for deletion by the next SaveChanges.
func Remove[T any](h Handle, entity *T) error { return mark(h, entity, track.Deleted) }

func mark[T any](h Handle, entity *T, state track.State) error {
	s, model, err := handleModel[T](h)
	if err != nil {
		return err
	}

	if entity == nil {
		return ErrNilEntity
	}

	if len(model.Keys) == 0 {
		return fmt.Errorf("%w: %s", ErrNoKey, model.Type.Name())
	}

	s.tracker.Track(model, unsafe.Pointer(entity), state)

	return nil
}

func handleModel[T any](h Handle) (*state, *meta.Model, error) {
	if h == nil || h.context().state == nil {
		return nil, nil, ErrNotOpen
	}

	model := configure[T]()

	return h.context().state, model, model.Err()
}

// Detach stops tracking an entity. Unsaved changes to it are forgotten.
func Detach[T any](h Handle, entity *T) {
	if h != nil && h.context().state != nil {
		h.context().state.tracker.Remove(unsafe.Pointer(entity))
	}
}

// Tracked returns the tracked entities of type T, oldest first.
func Tracked[T any](h Handle) []*T {
	s, model, err := handleModel[T](h)
	if err != nil {
		return nil
	}

	var out []*T

	for _, e := range s.tracker.Entries() {
		if e.Model == model {
			out = append(out, (*T)(e.Entity))
		}
	}

	return out
}

// HasChanges reports whether SaveChanges has anything to write.
func (c *Context) HasChanges() bool {
	if c.state == nil {
		return false
	}

	for _, e := range c.state.tracker.Entries() {
		if e.State != track.Unchanged || len(e.Changed()) > 0 {
			return true
		}
	}

	return false
}

// ClearTracked stops tracking everything, forgetting unsaved changes.
func (c *Context) ClearTracked() {
	if c.state != nil {
		c.state.tracker.Clear()
	}
}

// DebugTracked describes every tracked entity, for logs and debugging.
func (c *Context) DebugTracked() string {
	if c.state == nil {
		return ""
	}

	var b strings.Builder

	for _, e := range c.state.tracker.Entries() {
		key, _ := e.Model.KeyOf(e.Entity)
		fmt.Fprintf(&b, "%s {%v} %s", e.Model.Type.Name(), key, entryState(e))

		for _, f := range e.Changed() {
			fmt.Fprintf(&b, " %s", f.Short)
		}

		b.WriteByte('\n')
	}

	return b.String()
}

func entryState(e *track.Entry) track.State {
	if e.State == track.Unchanged && len(e.Changed()) > 0 {
		return track.Modified
	}

	return e.State
}

// EntityEntry inspects one entity's tracking state.
type EntityEntry[T any] struct {
	entry  *track.Entry
	entity *T
}

// Entry returns the tracking state of an entity. An untracked entity is
// Detached.
func Entry[T any](h Handle, entity *T) *EntityEntry[T] {
	out := &EntityEntry[T]{entity: entity}

	if h != nil && h.context().state != nil {
		out.entry = h.context().state.tracker.Get(unsafe.Pointer(entity))
	}

	return out
}

// State returns the entity's state, comparing it with what was loaded.
func (e *EntityEntry[T]) State() State {
	if e.entry == nil {
		return Detached
	}

	return entryState(e.entry)
}

// Changed returns the names of the fields that differ from what was loaded.
func (e *EntityEntry[T]) Changed() []string {
	if e.entry == nil {
		return nil
	}

	var out []string

	for _, f := range e.entry.Changed() {
		out = append(out, f.Short)
	}

	return out
}

// IsDirty reports whether a field differs from what was loaded.
func IsDirty[T, V any](e *EntityEntry[T], field *V) bool {
	f, _, err := fieldOf(field)
	if err != nil || e.entry == nil {
		return false
	}

	for _, changed := range e.entry.Changed() {
		if changed == f {
			return true
		}
	}

	return false
}

// Original returns a field's value as it was loaded. It returns ErrNotTracked
// for an entity with no loaded values.
func Original[T, V any](e *EntityEntry[T], field *V) (V, error) {
	var zero V

	f, _, err := fieldOf(field)
	if err != nil {
		return zero, err
	}

	if e.entry == nil || e.entry.Original() == nil {
		return zero, ErrNotTracked
	}

	return f.Value(e.entry.Original()).Interface().(V), nil
}
