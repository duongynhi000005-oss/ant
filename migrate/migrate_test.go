package migrate_test

import (
	"bytes"
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/dialect/mysql"
	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/migrate"
	"github.com/skudra-go/ant/schema"
)

// These run with plain go test, because generating needs no database.

type user struct {
	ant.Base
	Name string
}

type firstDB struct {
	ant.Context
	Users ant.Table[user]
}

type userWithBio struct {
	ant.Base
	Name string
	Bio  *string
}

func (userWithBio) Configure(b *ant.ModelBuilder[userWithBio]) { b.Table("users") }

type secondDB struct {
	ant.Context
	Users ant.Table[userWithBio]
}

// memory is a migrate.Store with nothing behind it.
type memory struct {
	migrations []migrate.Migration
	snapshot   []byte
}

func (m *memory) List(context.Context) ([]migrate.Migration, error) { return m.migrations, nil }
func (m *memory) Snapshot(context.Context) ([]byte, error)          { return m.snapshot, nil }
func (m *memory) Save(_ context.Context, migration migrate.Migration) error {
	m.migrations = append(m.migrations, migration)

	return nil
}
func (m *memory) Remove(context.Context, migrate.Migration) error { return nil }
func (m *memory) SaveSnapshot(_ context.Context, data []byte) error {
	m.snapshot = data

	return nil
}

func TestChangeBecomesSQLWithNoDatabase(t *testing.T) {
	store := new(memory)
	config := migrate.Config{Source: store, Out: new(bytes.Buffer)}
	ctx := t.Context()

	require.NoError(
		t,
		migrate.Run[firstDB](ctx, postgres.Dialect, []string{"diff", "create users"}, config),
	)

	require.Len(t, store.migrations, 1)
	require.Equal(t, "0001", store.migrations[0].Version)
	require.Equal(t, "create_users", store.migrations[0].Name, "the name is cleaned for a file")
	require.Contains(t, store.migrations[0].SQL, `CREATE TABLE "users"`)
	require.NoError(t, migrate.Run[firstDB](ctx, postgres.Dialect, []string{"check"}, config))

	err := migrate.Run[secondDB](ctx, postgres.Dialect, []string{"check"}, config)
	require.ErrorIs(t, err, migrate.ErrPendingChanges)

	require.NoError(
		t,
		migrate.Run[secondDB](ctx, postgres.Dialect, []string{"diff", "add_bio"}, config),
	)

	require.Len(t, store.migrations, 2)
	require.Equal(t, "0002", store.migrations[1].Version)
	require.Contains(t, store.migrations[1].SQL, `ALTER TABLE "users" ADD COLUMN "bio" text;`)
	require.NoError(t, migrate.Run[secondDB](ctx, postgres.Dialect, []string{"check"}, config))
}

func TestUnsupportedDialect(t *testing.T) {
	err := migrate.Run[firstDB](t.Context(), mysql.Dialect, []string{"check"})
	require.ErrorIs(t, err, schema.ErrUnsupported)
}

func TestHelp(t *testing.T) {
	require.ErrorIs(t, migrate.Run[firstDB](t.Context(), postgres.Dialect, nil), flag.ErrHelp)
}
