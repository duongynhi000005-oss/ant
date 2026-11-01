package dialect_test

import (
	"reflect"
	"testing"

	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/qb"
)

func TestRenderPostgres(t *testing.T) {
	users := &qb.TableRef{Name: "users"}
	sales := &qb.TableRef{Schema: "sales", Name: "orders"}
	age := qb.Column{Table: users, Name: "age"}
	name := qb.Column{Table: users, Name: "name"}

	selected := func(c ...qb.Expr) []qb.Selected {
		out := make([]qb.Selected, len(c))

		for i, e := range c {
			out[i] = qb.Selected{Value: e}
		}

		return out
	}

	param := func(v any) qb.Expr { return qb.Param{Value: v} }
	sub := &qb.Select{Columns: selected(age), From: qb.Source{Table: users}}
	cases := []struct {
		name string
		stmt qb.Statement
		sql  string
		args []any
	}{
		{
			"select",
			&qb.Select{Columns: selected(age, name), From: qb.Source{Table: users}},
			`SELECT "users"."age", "users"."name" FROM "users"`,
			nil,
		},
		{
			"where order limit offset",
			&qb.Select{
				Columns: selected(age),
				From:    qb.Source{Table: users},
				Where:   []qb.Cond{qb.Compare{Left: age, Op: qb.OpGt, Right: param(18)}},
				Order:   []qb.Order{{Value: name}, {Value: age, Desc: true, Nulls: qb.NullsLast}},
				Limit:   new(10),
				Offset:  new(20),
			},
			`SELECT "users"."age" FROM "users" WHERE "users"."age" > $1 ORDER BY "users"."name", "users"."age" DESC NULLS LAST LIMIT $2 OFFSET $3`,
			[]any{18, 10, 20},
		},
		{
			"schema",
			&qb.Select{
				Columns: selected(qb.Column{Table: sales, Name: "id"}),
				From:    qb.Source{Table: sales},
			},
			`SELECT "orders"."id" FROM "sales"."orders"`,
			nil,
		},
		{
			"joins and aliases",
			&qb.Select{
				Columns: []qb.Selected{
					{Value: name},
					{Value: qb.Column{Alias: "t2", Name: "name"}, Alias: "parent"},
				},
				From: qb.Source{Table: users},
				Joins: []qb.Join{
					{
						Kind:   qb.LeftJoin,
						Source: qb.Source{Table: users, Alias: "t2"},
						On: []qb.Cond{
							qb.Compare{
								Left:  age,
								Op:    qb.OpEq,
								Right: qb.Column{Alias: "t2", Name: "age"},
							},
						},
					},
					{Kind: qb.CrossJoin, Source: qb.Source{Table: sales}},
					{
						Kind:   qb.InnerJoin,
						Source: qb.Source{Table: sales, Alias: "s"},
						Using:  []string{"id"},
					},
				},
			},
			`SELECT "users"."name", "t2"."name" AS "parent" FROM "users" LEFT JOIN "users" AS "t2" ON "users"."age" = "t2"."age" CROSS JOIN "sales"."orders" JOIN "sales"."orders" AS "s" USING ("id")`,
			nil,
		},
		{
			"group having distinct",
			&qb.Select{
				Distinct: true,
				Columns:  selected(age, qb.Func{Name: "COUNT", Args: []qb.Expr{qb.Star{}}}),
				From:     qb.Source{Table: users},
				GroupBy:  []qb.Expr{age},
				Having: []qb.Cond{
					qb.Compare{
						Left:  qb.Func{Name: "COUNT", Args: []qb.Expr{qb.Star{}}},
						Op:    qb.OpGt,
						Right: param(1),
					},
				},
			},
			`SELECT DISTINCT "users"."age", COUNT(*) FROM "users" GROUP BY "users"."age" HAVING COUNT(*) > $1`,
			[]any{1},
		},
		{
			"function details",
			&qb.Select{
				Columns: selected(
					qb.Func{
						Name:     "COUNT",
						Args:     []qb.Expr{age},
						Distinct: true,
						Filter:   []qb.Cond{qb.IsNull{Left: name}},
					},
					qb.Func{
						Name: "RANK",
						Over: &qb.Window{
							PartitionBy: []qb.Expr{age},
							OrderBy:     []qb.Order{{Value: name}},
						},
					},
				),
				From: qb.Source{Table: users},
			},
			`SELECT COUNT(DISTINCT "users"."age") FILTER (WHERE "users"."name" IS NULL), RANK() OVER (PARTITION BY "users"."age" ORDER BY "users"."name") FROM "users"`,
			nil,
		},
		{
			"expressions",
			&qb.Select{
				Columns: selected(
					qb.Binary{Left: age, Op: "+", Right: param(1)},
					qb.Fragment{SQL: "lower(?) || '?'", Args: []qb.Expr{name}},
					qb.Case{
						Whens: []qb.When{{If: qb.IsNull{Left: name}, Then: param("x")}},
						Else:  param("y"),
					},
					qb.Collate{Value: name, Collation: "C"},
					qb.Subquery{Select: sub},
				),
			},
			`SELECT ("users"."age" + $1), lower("users"."name") || '?', CASE WHEN "users"."name" IS NULL THEN $2 ELSE $3 END, "users"."name" COLLATE "C", (SELECT "users"."age" FROM "users")`,
			[]any{1, "x", "y"},
		},
		{
			"with and sets",
			&qb.Select{
				With:      []qb.CTE{{Name: "a", Select: sub}},
				Recursive: true,
				Columns:   []qb.Selected{{Value: qb.Star{}}},
				From:      qb.Source{Name: "a"},
				Sets:      []qb.SetOp{{Kind: "UNION", All: true, Select: sub}},
				Order:     []qb.Order{{Value: qb.Column{Name: "age"}}},
			},
			`WITH RECURSIVE "a" AS (SELECT "users"."age" FROM "users") SELECT * FROM "a" UNION ALL (SELECT "users"."age" FROM "users") ORDER BY "age"`,
			nil,
		},
		{
			"lock and comment",
			&qb.Select{
				Comment: "tag */",
				Columns: selected(age),
				From:    qb.Source{Table: users},
				Lock:    &qb.Lock{Mode: "UPDATE", SkipLocked: true},
			},
			`/* tag * / */ SELECT "users"."age" FROM "users" FOR UPDATE SKIP LOCKED`,
			nil,
		},
		{
			"from subquery lateral",
			&qb.Select{
				Columns: selected(qb.Star{}),
				From:    qb.Source{Sub: sub, Alias: "s"},
				Joins: []qb.Join{{
					Kind:    qb.LeftJoin,
					Lateral: true,
					Source:  qb.Source{Sub: sub, Alias: "l"},
					On:      []qb.Cond{qb.And{}},
				}},
			},
			`SELECT * FROM (SELECT "users"."age" FROM "users") AS "s" LEFT JOIN LATERAL (SELECT "users"."age" FROM "users") AS "l" ON (1 = 1)`,
			nil,
		},
		{
			"several conditions",
			&qb.Delete{
				Table: users,
				Where: []qb.Cond{
					qb.Compare{Left: age, Op: qb.OpEq, Right: param(1)},
					qb.Or{
						Children: []qb.Cond{
							qb.IsNull{Left: name},
							qb.Not{Child: qb.Between{Left: age, Low: param(1), High: param(2)}},
						},
					},
				},
			},
			`DELETE FROM "users" WHERE "users"."age" = $1 AND ("users"."name" IS NULL OR NOT ("users"."age" BETWEEN $2 AND $3))`,
			[]any{1, 1, 2},
		},
		{
			"in",
			&qb.Delete{
				Table: users,
				Where: []qb.Cond{
					qb.In{Left: age, Values: []qb.Expr{param(1), param(2)}, Negate: true},
				},
			},
			`DELETE FROM "users" WHERE "users"."age" NOT IN ($1, $2)`,
			[]any{1, 2},
		},
		{
			"in subquery and exists",
			&qb.Delete{
				Table: users,
				Where: []qb.Cond{
					qb.In{
						Left: age,
						Sub:  sub,
					},
					qb.Exists{Select: sub, Negate: true},
					qb.Fragment{SQL: "a = ?", Args: []qb.Expr{param(1)}},
				},
			},
			`DELETE FROM "users" WHERE "users"."age" IN (SELECT "users"."age" FROM "users") AND NOT EXISTS (SELECT "users"."age" FROM "users") AND (a = $1)`,
			[]any{1},
		},
		{
			"empty in matches nothing",
			&qb.Delete{Table: users, Where: []qb.Cond{qb.In{Left: age}}},
			`DELETE FROM "users" WHERE 1 = 0`,
			nil,
		},
		{
			"empty not in matches everything",
			&qb.Delete{Table: users, Where: []qb.Cond{qb.In{Left: age, Negate: true}}},
			`DELETE FROM "users" WHERE 1 = 1`,
			nil,
		},
		{
			"empty and is true",
			&qb.Delete{Table: users, Where: []qb.Cond{qb.And{}}},
			`DELETE FROM "users" WHERE (1 = 1)`,
			nil,
		},
		{
			"empty or is false",
			&qb.Delete{Table: users, Where: []qb.Cond{qb.Or{}}},
			`DELETE FROM "users" WHERE (1 = 0)`,
			nil,
		},
		{
			"insert",
			&qb.Insert{
				Table:     users,
				Columns:   []string{"id", "age"},
				Rows:      [][]qb.Expr{{qb.Default{}, param(1)}, {param(9), param(2)}},
				Returning: []qb.Selected{{Value: qb.Column{Name: "id"}}},
			},
			`INSERT INTO "users" ("id", "age") VALUES (DEFAULT, $1), ($2, $3) RETURNING "id"`,
			[]any{1, 9, 2},
		},
		{
			"insert defaults",
			&qb.Insert{Table: users, Returning: []qb.Selected{{Value: qb.Column{Name: "id"}}}},
			`INSERT INTO "users" DEFAULT VALUES RETURNING "id"`,
			nil,
		},
		{
			"insert select",
			&qb.Insert{Table: users, Columns: []string{"age"}, From: sub},
			`INSERT INTO "users" ("age") SELECT "users"."age" FROM "users"`,
			nil,
		},
		{
			"upsert",
			&qb.Insert{
				Table:   users,
				Columns: []string{"age"},
				Rows:    [][]qb.Expr{{param(1)}},
				Conflict: &qb.Conflict{
					Columns: []string{"age"},
					Set: []qb.Assignment{
						{Column: "age", Value: qb.Fragment{SQL: "EXCLUDED.age"}},
					},
					Where: []qb.Cond{qb.IsNull{Left: name}},
				},
			},
			`INSERT INTO "users" ("age") VALUES ($1) ON CONFLICT ("age") DO UPDATE SET "age" = EXCLUDED.age WHERE "users"."name" IS NULL`,
			[]any{1},
		},
		{
			"conflict does nothing",
			&qb.Insert{
				Table:    users,
				Columns:  []string{"age"},
				Rows:     [][]qb.Expr{{param(1)}},
				Conflict: &qb.Conflict{},
			},
			`INSERT INTO "users" ("age") VALUES ($1) ON CONFLICT DO NOTHING`,
			[]any{1},
		},
		{
			"update",
			&qb.Update{
				Table: users,
				Set: []qb.Assignment{
					{Column: "age", Value: param(1)},
					{Column: "name", Value: param("x")},
				},
				Where:     []qb.Cond{qb.Compare{Left: age, Op: qb.OpEq, Right: param(2)}},
				Returning: []qb.Selected{{Value: qb.Column{Name: "age"}}},
			},
			`UPDATE "users" SET "age" = $1, "name" = $2 WHERE "users"."age" = $3 RETURNING "age"`,
			[]any{1, "x", 2},
		},
		{
			"delete returning",
			&qb.Delete{Table: users, Returning: []qb.Selected{{Value: qb.Column{Name: "age"}}}},
			`DELETE FROM "users" RETURNING "age"`,
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sql, args, err := postgres.Dialect.Render(c.stmt)
			if err != nil {
				t.Fatal(err)
			}

			if sql != c.sql {
				t.Errorf("sql:\n got %s\nwant %s", sql, c.sql)
			}

			if !reflect.DeepEqual(args, c.args) {
				t.Errorf("args = %v, want %v", args, c.args)
			}
		})
	}
}

func TestRenderFollowsTableRename(t *testing.T) {
	ref := &qb.TableRef{Name: "users"}
	stmt := &qb.Delete{Table: ref}
	ref.Name = "people"

	sql, _, err := postgres.Dialect.Render(stmt)
	if err != nil || sql != `DELETE FROM "people"` {
		t.Errorf("got %q, %v", sql, err)
	}
}

func TestRenderFragmentArguments(t *testing.T) {
	stmt := &qb.Select{
		Columns: []qb.Selected{{Value: qb.Fragment{SQL: "1", Args: []qb.Expr{qb.Param{Value: 1}}}}},
	}

	if _, _, err := postgres.Dialect.Render(stmt); err == nil {
		t.Error("want an error for an unused argument")
	}
}

func TestMatchesAll(t *testing.T) {
	cond := qb.Compare{Left: qb.Param{Value: 1}, Op: qb.OpEq, Right: qb.Param{Value: 1}}
	for name, c := range map[string]struct {
		cond qb.Cond
		want bool
	}{
		"empty and":     {qb.And{}, true},
		"empty or":      {qb.Or{}, false},
		"real":          {cond, false},
		"and with real": {qb.And{Children: []qb.Cond{qb.And{}, cond}}, false},
		"or with empty": {qb.Or{Children: []qb.Cond{qb.And{}, cond}}, true},
		"not in empty":  {qb.In{Negate: true}, true},
		"in empty":      {qb.In{}, false},
		"in subquery":   {qb.In{Negate: true, Sub: &qb.Select{}}, false},
		"not nothing":   {qb.Not{Child: qb.In{}}, true},
		"not all":       {qb.Not{Child: qb.And{}}, false},
		"not and none":  {qb.Not{Child: qb.And{Children: []qb.Cond{qb.Or{}, cond}}}, true},
		"not or none":   {qb.Not{Child: qb.Or{Children: []qb.Cond{qb.Or{}, qb.In{}}}}, true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := qb.MatchesAll(c.cond); got != c.want {
				t.Errorf("got %v", got)
			}
		})
	}
}
