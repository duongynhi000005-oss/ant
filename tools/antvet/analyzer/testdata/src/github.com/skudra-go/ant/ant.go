// Package ant is a stub with the signatures the analyzer looks at.
package ant

import "context"

type Cond struct{}

func Fields[T any]() *T                   { return new(T) }
func Eq[V comparable](field *V, v V) Cond { return Cond{} }
func Gt[V any](field *V, v V) Cond        { return Cond{} }
func EqCol[V comparable](a, b *V) Cond    { return Cond{} }
func Exists(q any) Cond                   { return Cond{} }

type Query[T any] struct{}

func (q *Query[T]) Where(c ...Cond) *Query[T]                { return q }
func (q *Query[T]) Join() *Query[T]                          { return q }
func (q *Query[T]) All(ctx context.Context) ([]*T, error)    { return nil, nil }
func (q *Query[T]) Count(ctx context.Context) (int64, error) { return 0, nil }
func (q *Query[T]) OrderBy(field any) *Query[T]              { return q }

type Table[T any] struct{}

func (t *Table[T]) Where(c ...Cond) *Query[T] { return nil }
func (t *Table[T]) Find[K comparable](ctx context.Context, key K) (*T, error) {
	return nil, nil
}

type FieldRef struct{}

func F[V any](field *V) FieldRef { return FieldRef{} }

type ModelBuilder[T any] struct{}

func (b *ModelBuilder[T]) Table(name string)                          {}
func (b *ModelBuilder[T]) Key[V any](field *V)                        {}
func (b *ModelBuilder[T]) CompositeKey(fields ...FieldRef)            {}
func (b *ModelBuilder[T]) NoKey()                                     {}
func GenerateKey[E, V any](b *ModelBuilder[E], field *V, fn func() V) {}
