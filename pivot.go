package ant

import (
	"context"
	"fmt"
	"slices"
	"unsafe"

	"github.com/skudra-go/ant/internal/exec"
	"github.com/skudra-go/ant/internal/meta"
	"github.com/skudra-go/ant/qb"
)

// pivotKeys resolves a many-to-many relation for one owner and its targets.
func pivotKeys[T, R any](
	h Handle,
	owner *T,
	rel *[]R,
	targets []*R,
) (*state, *meta.Pivot, any, []any, error) {
	s, model, err := handleModel[T](h)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if owner == nil || slices.Contains(targets, nil) {
		return nil, nil, nil, nil, ErrNilEntity
	}

	r, link, err := linkOf[R](rel)

	switch {
	case err != nil:
		return nil, nil, nil, nil, err
	case r.Owner != model:
		return nil, nil, nil, nil, ErrWrongModel
	case r.Pivot == nil:
		return nil, nil, nil, nil, fmt.Errorf("%w: %s", ErrNotManyToMany, r.Name)
	}

	local, ok := savedKey(link.Local, unsafe.Pointer(owner))
	if !ok {
		return nil, nil, nil, nil, ErrNoKey
	}

	foreign := make([]any, 0, len(targets))

	for _, target := range targets {
		key, ok := savedKey(link.Foreign, unsafe.Pointer(target))
		if !ok {
			return nil, nil, nil, nil, ErrNoKey
		}

		foreign = append(foreign, key)
	}

	return s, r.Pivot, local, foreign, nil
}

// savedKey returns a key, or false for an entity not saved yet.
func savedKey(f *meta.Field, entity unsafe.Pointer) (any, bool) {
	if f.Generated && f.Value(entity).IsZero() {
		return nil, false
	}

	return f.Key(entity)
}

// Link adds rows to a many-to-many relation's join table, skipping pairs that are
// already there. It does not change owner's relation field.
func Link[T, R any](ctx context.Context, h Handle, owner *T, rel *[]R, targets ...*R) error {
	s, pivot, local, foreign, err := pivotKeys(h, owner, rel, targets)
	if err != nil || len(foreign) == 0 {
		return err
	}

	_, err = s.run(ctx, pivotInsert(pivot, local, foreign))

	return err
}

// Unlink removes rows from a many-to-many relation's join table. The targets
// themselves stay.
func Unlink[T, R any](ctx context.Context, h Handle, owner *T, rel *[]R, targets ...*R) error {
	s, pivot, local, foreign, err := pivotKeys(h, owner, rel, targets)
	if err != nil || len(foreign) == 0 {
		return err
	}

	_, err = s.run(ctx, pivotDelete(pivot, local, foreign))

	return err
}

// SyncLinks makes targets the only links of owner in a many-to-many relation.
func SyncLinks[T, R any](ctx context.Context, h Handle, owner *T, rel *[]R, targets ...*R) error {
	s, pivot, local, foreign, err := pivotKeys(h, owner, rel, targets)
	if err != nil {
		return err
	}

	return s.transact(ctx, true, func(to exec.Executor) error {
		clear := pivotDelete(pivot, local, nil)
		clear.Where = clear.Where[:1]

		if len(foreign) > 0 {
			clear.Where = append(
				clear.Where,
				qb.Not{Child: pivotDelete(pivot, local, foreign).Where[1]},
			)
		}

		if _, err := s.runOn(ctx, to, clear); err != nil {
			return err
		}

		if len(foreign) == 0 {
			return nil
		}

		_, err := s.runOn(ctx, to, pivotInsert(pivot, local, foreign))

		return err
	})
}

// Toggle links the targets that are not linked to owner, and unlinks those that
// are.
func Toggle[T, R any](ctx context.Context, h Handle, owner *T, rel *[]R, targets ...*R) error {
	s, pivot, local, foreign, err := pivotKeys(h, owner, rel, targets)
	if err != nil || len(foreign) == 0 {
		return err
	}

	return s.transact(ctx, true, func(to exec.Executor) error {
		// A row is removed in the same statement that finds it, so two callers
		// cannot both see it.
		removal := pivotDelete(pivot, local, foreign)
		removal.Returning = []qb.Selected{{Value: qb.Column{Name: pivot.Foreign}}}

		text, args, err := s.dialect.Render(removal)
		if err != nil {
			return err
		}

		rows, err := to.QueryContext(ctx, text, args...)
		if err != nil {
			return err
		}

		removed, err := collect[any](rows)
		if err != nil {
			return err
		}

		gone := make(map[any]struct{}, len(removed))

		for _, key := range removed {
			if k, ok := meta.KeyValue(key); ok {
				gone[k] = struct{}{}
			}
		}

		added := slices.DeleteFunc(foreign, func(key any) bool {
			_, was := gone[key]

			return was
		})

		if len(added) == 0 {
			return nil
		}

		_, err = s.runOn(ctx, to, pivotInsert(pivot, local, added))

		return err
	})
}
