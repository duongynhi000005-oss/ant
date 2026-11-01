package ant

import (
	"context"
	"log/slog"
	"time"

	"github.com/skudra-go/ant/internal/exec"
)

// Statement is one SQL statement on its way to the database. An interceptor can
// change it. Args holds real values, so take care what you record.
type Statement = exec.Statement

// Outcome is what happened to a statement. Duration ends when the database
// answered, before rows are read.
type Outcome = exec.Outcome

// Interceptor observes every statement a context runs, for logging, metrics,
// and tracing. Before can return a new context, as tracing needs, and an error
// from it stops the statement.
type Interceptor = exec.Interceptor

// WithInterceptor adds an interceptor. Several run in order, and After runs in
// reverse order.
func WithInterceptor(i Interceptor) Option {
	return func(s *settings) { s.interceptors = append(s.interceptors, i) }
}

// logging is the Interceptor from WithLogger.
type logging struct {
	logger    *slog.Logger
	slow      time.Duration
	sensitive bool
}

// Before implements Interceptor.
func (l *logging) Before(
	ctx context.Context,
	_ *Statement,
) (context.Context, error) {
	return ctx, nil
}

// After implements Interceptor.
func (l *logging) After(ctx context.Context, s *Statement, o Outcome) {
	attrs := []slog.Attr{
		slog.String("sql", s.SQL),
		slog.Duration("duration", o.Duration),
		slog.Int("args", len(s.Args)),
	}

	if l.sensitive {
		attrs = append(attrs, slog.Any("values", s.Args))
	}

	switch {
	case o.Err != nil:
		l.logger.LogAttrs(
			ctx,
			slog.LevelError,
			"ant: statement failed",
			append(attrs, slog.String("error", o.Err.Error()))...)
	case l.slow > 0 && o.Duration >= l.slow:
		l.logger.LogAttrs(ctx, slog.LevelWarn, "ant: slow statement", attrs...)
	default:
		l.logger.LogAttrs(ctx, slog.LevelDebug, "ant: statement", attrs...)
	}
}

// LogOption configures WithLogger.
type LogOption func(*logging)

// SlowThreshold logs statements at or over d at warning level.
func SlowThreshold(d time.Duration) LogOption { return func(l *logging) { l.slow = d } }

// LogValues includes argument values in logs. They are left out by default,
// because they may be passwords or personal data.
func LogValues() LogOption { return func(l *logging) { l.sensitive = true } }

// WithLogger logs every statement to logger: failures at error level, slow ones
// at warning level, and the rest at debug level. Values are never logged unless
// LogValues asks.
func WithLogger(logger *slog.Logger, options ...LogOption) Option {
	l := &logging{logger: logger}

	for _, option := range options {
		option(l)
	}

	return WithInterceptor(l)
}
