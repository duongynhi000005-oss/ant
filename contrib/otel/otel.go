// Package otel traces ant's statements with OpenTelemetry. It is a module of its
// own, so that ant itself has no such dependency.
//
//	db, err := ant.Open[AppDB](ctx, postgres.Dialect, dsn, ant.WithInterceptor(otel.New(tracer)))
//
// Each statement becomes a span under the caller's, named after the kind of
// statement, with its SQL in db.query.text. Values are never recorded.
package otel

import (
	"context"
	"strings"

	"github.com/skudra-go/ant"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type spanKey struct{}

// Tracer is an ant.Interceptor that starts a span per statement.
type Tracer struct {
	tracer trace.Tracer
	system string
}

// New returns an interceptor for tracer.
func New(tracer trace.Tracer) *Tracer { return &Tracer{tracer: tracer, system: "postgresql"} }

// Before implements ant.Interceptor.
func (t *Tracer) Before(ctx context.Context, s *ant.Statement) (context.Context, error) {
	name, _, _ := strings.Cut(strings.TrimSpace(s.SQL), " ")
	ctx, span := t.tracer.Start(
		ctx,
		"db."+strings.ToLower(name),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", t.system),
			attribute.String("db.query.text", s.SQL),
		),
	)

	return context.WithValue(ctx, spanKey{}, span), nil
}

// After implements ant.Interceptor.
func (t *Tracer) After(ctx context.Context, _ *ant.Statement, o ant.Outcome) {
	span, ok := ctx.Value(spanKey{}).(trace.Span)
	if !ok {
		return
	}

	if o.Err != nil {
		span.RecordError(o.Err)
		span.SetStatus(codes.Error, o.Err.Error())
	}

	span.End()
}
