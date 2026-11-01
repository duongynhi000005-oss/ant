package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sync"
)

var registry struct {
	sync.Mutex
	migrations map[reflect.Type][]Migration
	seeders    map[reflect.Type][]seeder
}

// Seeder fills a database with data.
type Seeder[DB any] func(ctx context.Context, db *DB) error

type seeder struct {
	name string
	run  func(ctx context.Context, db any) error
}

// GoOption adjusts a Go migration.
type GoOption func(*Migration)

// NoTransaction runs a Go migration outside a transaction, for work on large
// tables that commits in batches. fn receives an ordinary context. If it fails, the
// database is left partway, as with the -- ant:no-transaction directive.
func NoTransaction(m *Migration) { m.noTransaction = true }

// Register adds a data migration written in Go, for changes SQL cannot express.
// It runs in order with the SQL migrations, identified by id, such as
// 0005_backfill_slugs, and only once. tx is a context bound to the migration's
// transaction, so ant queries and ant.Raw join it.
//
// Call Register from an init function, in a package that both cmd/migrate and any
// program calling Up import. A model describes the latest schema, not the schema
// at this point in the sequence, so prefer ant.Raw over models in old migrations.
func Register[DB any](id string, fn func(ctx context.Context, tx *DB) error, options ...GoOption) {
	version, name, ok := splitID(id)
	if !ok {
		panic(fmt.Sprintf("ant: migrate.Register: %q is not like 0005_backfill_slugs", id))
	}

	m := Migration{
		Kind:    Versioned,
		Version: version,
		Name:    name,
		run:     func(ctx context.Context, db any, _ *sql.Tx) error { return fn(ctx, db.(*DB)) },
	}

	for _, option := range options {
		option(&m)
	}

	t := reflect.TypeFor[DB]()

	registry.Lock()
	defer registry.Unlock()

	if registry.migrations == nil {
		registry.migrations = map[reflect.Type][]Migration{}
	}

	registry.migrations[t] = append(registry.migrations[t], m)
}

// Seed adds a seeder, run in registration order by the seed command, each in its
// own transaction. Seeders run whenever asked, so write them to be safe to repeat,
// for example with Upsert or FirstOrCreate. Hooks are off unless Config.SeedHooks
// is set.
func Seed[DB any](name string, fn Seeder[DB]) {
	t := reflect.TypeFor[DB]()

	registry.Lock()
	defer registry.Unlock()

	if registry.seeders == nil {
		registry.seeders = map[reflect.Type][]seeder{}
	}

	registry.seeders[t] = append(
		registry.seeders[t],
		seeder{name, func(ctx context.Context, db any) error {
			return fn(ctx, db.(*DB))
		}},
	)
}

func splitID(id string) (string, string, bool) {
	for i, r := range id {
		if r == '_' || r == '-' {
			return id[:i], id[i+1:], i > 0 && i+1 < len(id)
		}
	}

	return "", "", false
}

func registeredMigrations[DB any]() []Migration {
	registry.Lock()
	defer registry.Unlock()

	return append([]Migration(nil), registry.migrations[reflect.TypeFor[DB]()]...)
}

func registeredSeeders[DB any]() []seeder {
	registry.Lock()
	defer registry.Unlock()

	return append([]seeder(nil), registry.seeders[reflect.TypeFor[DB]()]...)
}
