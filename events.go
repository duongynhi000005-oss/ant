package ant

import (
	"context"
	"reflect"

	"github.com/skudra-go/ant/internal/track"
)

// Events are callbacks on a context's unit of work. Interceptors watch statements,
// and observers watch one model, and these watch the unit of work as a whole. A nil
// callback is skipped. They cannot stop anything: use a SaveInterceptor for that.
type Events struct {
	// Saving runs before SaveChanges writes.
	Saving func(ctx context.Context, changes []Change)
	// Saved runs after SaveChanges commits, or, in a transaction, after it wrote.
	Saved func(ctx context.Context, changes []Change)
	// SaveFailed runs when SaveChanges fails.
	SaveFailed func(ctx context.Context, err error)
	// Tracked runs when the tracker starts following an entity. It must not call
	// back into the context.
	Tracked func(model string, entity any)
	// StateChanged runs when an entity is marked with a new state, such as by
	// Remove. It must not call back into the context.
	StateChanged func(model string, entity any, from, to State)
}

// WithEvents registers callbacks on the context and its sessions.
func WithEvents(e Events) Option { return func(s *settings) { s.events = e } }

func (e Events) attach(t *track.Tracker) {
	if f := e.Tracked; f != nil {
		t.OnTrack = func(entry *track.Entry) {
			f(entry.Model.Type.Name(), reflect.NewAt(entry.Model.Type, entry.Entity).Interface())
		}
	}

	if f := e.StateChanged; f != nil {
		t.OnState = func(entry *track.Entry, from, to track.State) {
			f(
				entry.Model.Type.Name(),
				reflect.NewAt(entry.Model.Type, entry.Entity).Interface(),
				from,
				to,
			)
		}
	}
}
