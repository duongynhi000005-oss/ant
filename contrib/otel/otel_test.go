package otel

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/skudra-go/ant"
)

func TestSpans(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := New(provider.Tracer("test"))

	statement := &ant.Statement{SQL: "SELECT * FROM users WHERE id = $1", Args: []any{"secret"}}

	ctx, err := tracer.Before(context.Background(), statement)
	if err != nil {
		t.Fatal(err)
	}

	tracer.After(ctx, statement, ant.Outcome{Err: errors.New("boom")})

	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Name() != "db.select" {
		t.Fatalf("spans: %+v", spans)
	}

	if spans[0].Status().Code != codes.Error {
		t.Fatalf("status: %+v", spans[0].Status())
	}

	for _, attribute := range spans[0].Attributes() {
		if attribute.Value.AsString() == "secret" {
			t.Fatal("a value reached a span")
		}
	}
}
