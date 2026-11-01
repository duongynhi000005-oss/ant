package schema_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/schema"
)

func ddl(t *testing.T) schema.Dialect {
	t.Helper()

	d, err := schema.Of(postgres.Dialect)
	require.NoError(t, err)

	return d
}

func users() *schema.Table {
	return &schema.Table{
		Name: "users",
		Columns: []*schema.Column{
			{Name: "id", Type: "bigint", Identity: true},
			{Name: "name", Type: "text"},
			{Name: "email", Type: "text"},
			{Name: "bio", Type: "text", Nullable: true},
		},
		PrimaryKey: &schema.Key{Name: "users_pkey", Columns: []string{"id"}},
		Uniques:    []*schema.Key{{Name: "users_email_key", Columns: []string{"email"}}},
	}
}

func posts() *schema.Table {
	return &schema.Table{
		Name: "posts",
		Columns: []*schema.Column{
			{Name: "id", Type: "bigint", Identity: true},
			{Name: "user_id", Type: "bigint"},
		},
		PrimaryKey: &schema.Key{Name: "posts_pkey", Columns: []string{"id"}},
		Indexes: []*schema.Index{
			{Name: "posts_user_id_idx", Parts: []schema.IndexPart{{Column: "user_id"}}},
		},
		ForeignKeys: []*schema.ForeignKey{{
			Name:       "posts_user_id_fkey",
			Columns:    []string{"user_id"},
			RefTable:   "users",
			RefColumns: []string{"id"},
			OnDelete:   "CASCADE",
		}},
	}
}

func of(tables ...*schema.Table) *schema.Schema { return &schema.Schema{Tables: tables} }

// run returns the statements of the plan between two schemas.
func run(t *testing.T, from, to *schema.Schema, renames ...string) []string {
	t.Helper()
	d := ddl(t)
	plan, err := schema.Diff(from, to, renames...)
	require.NoError(t, err)
	schema.Backfill(d, plan)
	out, err := schema.Statements(d, plan)
	require.NoError(t, err)

	return out
}

func TestNewTablesComeBeforeTheirForeignKeys(t *testing.T) {
	// posts sorts first, and refers to users, which only works because foreign
	// keys come after every table.
	got := run(t, nil, of(posts(), users()))
	require.Len(t, got, 4)
	require.Contains(t, got[0], `CREATE TABLE "posts"`)
	require.Contains(t, got[1], `CREATE TABLE "users"`)
	require.Equal(t, `CREATE INDEX "posts_user_id_idx" ON "posts" ("user_id")`, got[2])

	require.Equal(
		t,
		`ALTER TABLE "posts" ADD CONSTRAINT "posts_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE`,
		got[3],
	)
}

func TestDroppedTableLosesItsForeignKeysFirst(t *testing.T) {
	got := run(t, of(users(), posts()), of(users()))
	require.Equal(
		t,
		[]string{`ALTER TABLE "posts" DROP CONSTRAINT "posts_user_id_fkey"`, `DROP TABLE "posts"`},
		got,
	)
}

func TestNoChanges(t *testing.T) {
	require.Empty(t, run(t, of(users(), posts()), of(posts(), users())))
}

func TestColumnChanges(t *testing.T) {
	t.Run("a nullable column has nothing to fill", func(t *testing.T) {
		next := users()
		next.Columns = append(
			next.Columns,
			&schema.Column{Name: "nick", Type: "text", Nullable: true},
		)

		require.Equal(
			t,
			[]string{`ALTER TABLE "users" ADD COLUMN "nick" text`},
			run(t, of(users()), of(next)),
		)
	})

	t.Run("a NOT NULL column fills existing rows, then stops having a default", func(t *testing.T) {
		next := users()
		next.Columns = append(next.Columns, &schema.Column{Name: "age", Type: "bigint"})

		require.Equal(
			t,
			[]string{
				`ALTER TABLE "users" ADD COLUMN "age" bigint NOT NULL DEFAULT 0`,
				`ALTER TABLE "users" ALTER COLUMN "age" DROP DEFAULT`,
			},
			run(t, of(users()), of(next)),
		)
	})

	t.Run("a column with its own default keeps it", func(t *testing.T) {
		next := users()
		next.Columns = append(
			next.Columns,
			&schema.Column{Name: "role", Type: "text", Default: "'member'"},
		)

		require.Equal(
			t,
			[]string{`ALTER TABLE "users" ADD COLUMN "role" text NOT NULL DEFAULT 'member'`},
			run(t, of(users()), of(next)),
		)
	})

	t.Run("a type with no zero value says so", func(t *testing.T) {
		next := users()
		next.Columns = append(next.Columns, &schema.Column{Name: "status", Type: "user_status"})
		plan, err := schema.Diff(of(users()), of(next))
		require.NoError(t, err)
		schema.Backfill(ddl(t), plan)
		require.Contains(t, plan.Warnings()[0], "fails if the table has rows")
	})

	t.Run("each facet alters alone", func(t *testing.T) {
		next := users()
		next.Column("name").Type = "varchar(100)"
		next.Column("bio").Nullable = false
		next.Column("email").Default = "''"

		require.Equal(
			t,
			[]string{
				`ALTER TABLE "users" ALTER COLUMN "name" TYPE varchar(100) USING "name"::varchar(100)`,
				`ALTER TABLE "users" ALTER COLUMN "email" SET DEFAULT ''`,
				`ALTER TABLE "users" ALTER COLUMN "bio" SET NOT NULL`,
			},
			run(t, of(users()), of(next)),
		)
	})

	t.Run("a generated column is replaced", func(t *testing.T) {
		next := users()
		next.Column("name").Generated = "upper(email)"

		require.Equal(
			t,
			[]string{
				`ALTER TABLE "users" DROP COLUMN "name"`,
				`ALTER TABLE "users" ADD COLUMN "name" text NOT NULL GENERATED ALWAYS AS (upper(email)) STORED`,
			},
			run(t, of(users()), of(next)),
		)
	})
}

func TestWarnings(t *testing.T) {
	warnings := func(t *testing.T, from, to *schema.Schema) string {
		plan, err := schema.Diff(from, to)
		require.NoError(t, err)

		return strings.Join(plan.Warnings(), "\n")
	}
	t.Run("dropping data", func(t *testing.T) {
		next := users()
		next.Columns = next.Columns[:3]
		require.Contains(t, warnings(t, of(users()), of(next)), "drops column users.bio")
		require.Contains(t, warnings(t, of(users()), of()), "drops table users")
	})

	t.Run("changing a type", func(t *testing.T) {
		next := users()
		next.Column("name").Type = "integer"
		require.Contains(t, warnings(t, of(users()), of(next)), "may fail or lose data")
	})

	t.Run("forbidding NULL", func(t *testing.T) {
		next := users()
		next.Column("bio").Nullable = false
		require.Contains(t, warnings(t, of(users()), of(next)), "fails if any row holds NULL")
	})

	t.Run("a new table has no rows to worry about", func(t *testing.T) {
		require.Empty(t, warnings(t, nil, of(users(), posts())))
	})
}

func TestRenames(t *testing.T) {
	t.Run("a rename keeps the data, and constraints follow it", func(t *testing.T) {
		next := users()
		next.Column("email").Name = "mail"
		next.Uniques[0] = &schema.Key{Name: "users_mail_key", Columns: []string{"mail"}}

		require.Equal(
			t,
			[]string{
				`ALTER TABLE "users" RENAME COLUMN "email" TO "mail"`,
				`ALTER TABLE "users" RENAME CONSTRAINT "users_email_key" TO "users_mail_key"`,
			},
			run(t, of(users()), of(next), "users.email=mail"),
		)
	})

	t.Run("a table rename updates foreign keys that point at it", func(t *testing.T) {
		people := users()
		people.Name = "people"
		people.PrimaryKey.Name = "people_pkey"
		people.Uniques[0].Name = "people_email_key"
		child := posts()
		child.ForeignKeys[0].RefTable = "people"

		require.Equal(
			t,
			[]string{
				`ALTER TABLE "users" RENAME TO "people"`,
				`ALTER TABLE "people" RENAME CONSTRAINT "users_pkey" TO "people_pkey"`,
				`ALTER TABLE "people" RENAME CONSTRAINT "users_email_key" TO "people_email_key"`,
			},
			run(t, of(users(), posts()), of(people, child), "users=people"),
		)
	})

	t.Run(
		"a rename that names nothing is an error, because it was meant to prevent a drop",
		func(t *testing.T) {
			_, err := schema.Diff(of(users()), of(users()), "users.nope=x")
			require.ErrorContains(t, err, "no column nope")
			_, err = schema.Diff(of(users()), of(users()), "ghosts=x")
			require.ErrorContains(t, err, "no table ghosts")
			_, err = schema.Diff(of(users()), of(users()), "users.email=x")
			require.ErrorContains(t, err, "models have no column users.x")
			_, err = schema.Diff(of(users()), of(users()), "nonsense")
			require.ErrorContains(t, err, "want table=new_name")
		},
	)

	t.Run("a drop and add of one type is a hint, not a guess", func(t *testing.T) {
		next := users()
		next.Column("bio").Name = "about"
		plan, err := schema.Diff(of(users()), of(next))
		require.NoError(t, err)
		require.Len(t, plan.Hints, 1)
		require.Equal(t, "users.bio=about", plan.Hints[0].Flag())
		text, err := schema.Script(ddl(t), plan)
		require.NoError(t, err)
		require.Contains(t, text, `--   ALTER TABLE "users" RENAME COLUMN "bio" TO "about";`)
		require.Contains(t, text, `ALTER TABLE "users" DROP COLUMN "bio";`)
	})

	t.Run("a table with the same columns under a new name is a hint", func(t *testing.T) {
		people := users()
		people.Name = "people"
		plan, err := schema.Diff(of(users()), of(people))
		require.NoError(t, err)
		require.Len(t, plan.Hints, 1)
		require.Equal(t, "users=people", plan.Hints[0].Flag())
	})
}

func TestObjectsWithANewNameAreRenamed(t *testing.T) {
	t.Run("an index is not rebuilt for its name", func(t *testing.T) {
		next := posts()
		next.Indexes[0].Name = "posts_author_idx"

		require.Equal(
			t,
			[]string{`ALTER INDEX "posts_user_id_idx" RENAME TO "posts_author_idx"`},
			run(t, of(users(), posts()), of(users(), next)),
		)
	})

	t.Run("a changed index is replaced", func(t *testing.T) {
		next := posts()
		next.Indexes[0].Parts[0].Desc = true

		require.Equal(
			t,
			[]string{
				`DROP INDEX "posts_user_id_idx"`,
				`CREATE INDEX "posts_user_id_idx" ON "posts" ("user_id" DESC)`,
			},
			run(t, of(users(), posts()), of(users(), next)),
		)
	})

	t.Run("a changed foreign key is replaced", func(t *testing.T) {
		next := posts()
		next.ForeignKeys[0].OnDelete = "SET NULL"
		got := run(t, of(users(), posts()), of(users(), next))
		require.Equal(t, `ALTER TABLE "posts" DROP CONSTRAINT "posts_user_id_fkey"`, got[0])
		require.Contains(t, got[1], "ON DELETE SET NULL")
	})

	t.Run("building an index concurrently is not a schema change", func(t *testing.T) {
		next := posts()
		next.Indexes[0].Concurrent = true
		require.Empty(t, run(t, of(users(), posts()), of(users(), next)))
	})
}

func TestConcurrentIndex(t *testing.T) {
	index := &schema.Index{
		Name:       "users_bio_idx",
		Parts:      []schema.IndexPart{{Column: "bio"}},
		Concurrent: true,
	}

	next := users()
	next.Indexes = []*schema.Index{index}

	t.Run(
		"on an existing table it builds online, in a file with no transaction",
		func(t *testing.T) {
			plan, err := schema.Diff(of(users()), of(next))
			require.NoError(t, err)
			require.True(t, plan.Online())
			text, err := schema.Script(ddl(t), plan)
			require.NoError(t, err)
			require.Contains(t, text, schema.NoTransaction)
			require.Contains(t, text, "CREATE INDEX CONCURRENTLY")
		},
	)

	t.Run("on a new table there is nothing to wait for", func(t *testing.T) {
		plan, err := schema.Diff(nil, of(next))
		require.NoError(t, err)
		require.False(t, plan.Online())
	})
}

func TestSchemas(t *testing.T) {
	billing := &schema.Table{
		Schema:  "billing",
		Name:    "invoices",
		Columns: []*schema.Column{{Name: "id", Type: "bigint"}},
	}
	got := run(
		t,
		nil,
		&schema.Schema{Schemas: []string{"billing"}, Tables: []*schema.Table{billing}},
	)
	require.Equal(t, `CREATE SCHEMA IF NOT EXISTS "billing"`, got[0])
	require.Contains(t, got[1], `CREATE TABLE "billing"."invoices"`)
}

func TestSnapshot(t *testing.T) {
	s := of(users(), posts())
	s.Dialect = "postgres"

	t.Run("round trip", func(t *testing.T) {
		data, err := s.Encode()
		require.NoError(t, err)
		back, err := schema.Decode(data)
		require.NoError(t, err)
		require.Empty(t, run(t, s, back))
		require.Equal(t, schema.Format, back.Format)
	})

	t.Run("the same models always make the same bytes", func(t *testing.T) {
		a, err := of(users(), posts()).Encode()
		require.NoError(t, err)
		b, err := of(posts(), users()).Encode()
		require.NoError(t, err)
		require.Equal(t, string(a), string(b))
	})

	t.Run("a newer format is refused, not misread", func(t *testing.T) {
		_, err := schema.Decode([]byte(`{"format": 99}`))
		require.ErrorIs(t, err, schema.ErrNewerFormat)
	})

	t.Run("an empty file is an empty schema", func(t *testing.T) {
		got, err := schema.Decode(nil)
		require.NoError(t, err)
		require.Empty(t, got.Tables)
	})
}

func TestLongNames(t *testing.T) {
	d := ddl(t)
	long := strings.Repeat("x", 80)
	a, b := schema.IndexName(d, long, "a"), schema.IndexName(d, long, "b")
	require.LessOrEqual(t, len(a), 63)
	require.NotEqual(t, a, b, "two long names must not become one")
	require.Equal(t, "users_email_idx", schema.IndexName(d, "users", "email"))
	require.Equal(t, "users_pkey", schema.PrimaryKeyName(d, "users"))
	require.Equal(t, "users_email_key", schema.UniqueName(d, "users", "email"))
	require.Equal(t, "posts_user_id_fkey", schema.ForeignKeyName(d, "posts", "user_id"))
}

func TestUnsupportedDialect(t *testing.T) {
	_, err := schema.Of(struct{ named }{})
	require.ErrorIs(t, err, schema.ErrUnsupported)
	require.ErrorContains(t, err, "mysql")
}

type named struct{}

func (named) Name() string { return "mysql" }

func TestZeroValues(t *testing.T) {
	d := ddl(t)

	for columnType, want := range map[string]string{
		"bigint":        "0",
		"numeric(10,2)": "0",
		"text":          "''",
		"varchar(10)":   "''",
		"boolean":       "false",
		"timestamptz":   "now()",
		"jsonb":         "'null'",
		"text[]":        "'{}'",
		"uuid":          "gen_random_uuid()",
		"my_enum":       "",
	} {
		require.Equal(t, want, d.Zero(columnType), columnType)
	}
}

func TestColumnTypes(t *testing.T) {
	d := ddl(t)

	type Status string

	check := func(want string, value any, facets ...schema.Facets) {
		t.Helper()

		var f schema.Facets

		if len(facets) > 0 {
			f = facets[0]
		}

		got, ok := d.ColumnType(reflect.TypeOf(value), f)
		require.True(t, ok)
		require.Equal(t, want, got)
	}
	check("text", "")
	check("text", Status(""))
	check("varchar(20)", "", schema.Facets{Length: 20})
	check("bigint", 1)
	check("bigint", int64(1))
	check("integer", int32(1))
	check("smallint", int16(1))
	check("boolean", true)
	check("real", float32(1))
	check("double precision", 1.0)
	check("numeric(10,2)", 1.0, schema.Facets{Precision: 10, Scale: 2})
	check("bytea", []byte{})
	check("uuid", [16]byte{})

	_, ok := d.ColumnType(reflect.TypeOf(struct{}{}), schema.Facets{})
	require.False(t, ok)
}
