package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/skudra-go/ant"
)

func TestCounts(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.After(
		ctx,
		&ant.Statement{SQL: "/* tag */ SELECT 1"},
		ant.Outcome{Duration: 2 * time.Millisecond},
	)

	m.After(ctx, &ant.Statement{SQL: "insert into x"}, ant.Outcome{Err: errors.New("no")})

	var b strings.Builder
	m.WritePrometheus(&b)

	for _, want := range []string{
		`ant_statements_total{kind="SELECT"} 1`,
		`ant_statement_errors_total{kind="INSERT"} 1`,
		`ant_statement_seconds_bucket{kind="SELECT",le="0.005"} 1`,
		`ant_statement_seconds_bucket{kind="SELECT",le="0.001"} 0`,
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing %s in\n%s", want, b.String())
		}
	}

	if len(m.Snapshot()) != 2 {
		t.Fatalf("snapshot: %+v", m.Snapshot())
	}
}
