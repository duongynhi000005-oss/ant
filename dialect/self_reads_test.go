package dialect_test

import (
	"context"
	"strings"
	"testing"

	"github.com/skudra-go/ant/dialect"
	"github.com/skudra-go/ant/dialect/mysql"
	"github.com/skudra-go/ant/dialect/postgres"
	"github.com/skudra-go/ant/qb"
)

// MySQL refuses a write whose subquery reads the table being written, so such a
// subquery is wrapped in a derived table there, once, at whatever depth the read
// is. Subqueries of other tables, reads, and other databases are left alone.
func TestSelfReadsAreWrappedOnMySQL(t *testing.T) {
	users := &qb.TableRef{Name: "users"}
	posts := &qb.TableRef{Name: "posts"}
	id := qb.Column{Table: users, Name: "id"}
	userID := qb.Column{Table: posts, Name: "user_id"}

	sel := func(column qb.Expr, from *qb.TableRef, where ...qb.Cond) *qb.Select {
		return &qb.Select{
			Columns: []qb.Selected{{Value: column}},
			From:    qb.Source{Table: from},
			Where:   where,
		}
	}

	ofUsers := sel(id, users)
	ofPosts := sel(userID, posts)
	set := []qb.Assignment{{Column: "age", Value: qb.Param{Value: 1}}}

	update := func(where ...qb.Cond) *qb.Update {
		return &qb.Update{Table: users, Set: set, Where: where}
	}

	const wrap = "SELECT /*+ NO_MERGE(materialized) */ * FROM ("

	cases := []struct {
		name string
		stmt qb.Statement
		want string
	}{
		{
			"IN the same table",
			update(qb.In{Left: id, Sub: ofUsers}),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (" + wrap +
				"SELECT `users`.`id` AS `c1` FROM `users`) AS materialized)",
		},
		{
			"IN another table",
			update(qb.In{Left: id, Sub: ofPosts}),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (SELECT `posts`.`user_id` FROM `posts`)",
		},
		{
			"a read nested in another table's subquery, wrapped once, outside",
			update(qb.In{Left: id, Sub: sel(userID, posts, qb.In{Left: userID, Sub: ofUsers})}),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (" + wrap +
				"SELECT `posts`.`user_id` AS `c1` FROM `posts` WHERE `posts`.`user_id` IN " +
				"(SELECT `users`.`id` FROM `users`)) AS materialized)",
		},
		{
			"EXISTS",
			&qb.Delete{Table: users, Where: []qb.Cond{qb.Exists{Select: ofUsers, Negate: true}}},
			"DELETE FROM `users` WHERE NOT EXISTS (" + wrap +
				"SELECT `users`.`id` AS `c1` FROM `users`) AS materialized)",
		},
		{
			"a value in SET",
			&qb.Update{
				Table: users,
				Set:   []qb.Assignment{{Column: "age", Value: qb.Subquery{Select: ofUsers}}},
			},
			"UPDATE `users` SET `age` = (" + wrap + "SELECT `users`.`id` AS `c1` FROM `users`) AS materialized)",
		},
		{
			"a read the table's other spelling names",
			update(qb.In{Left: id, Sub: sel(id, &qb.TableRef{Schema: "app", Name: "Users"})}),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (" + wrap +
				"SELECT `users`.`id` AS `c1` FROM `app`.`Users`) AS materialized)",
		},
		{
			"raw SQL that names the table",
			update(
				qb.In{
					Left: id,
					Sub:  sel(userID, posts, qb.Fragment{SQL: "user_id IN (SELECT id FROM USERS)"}),
				},
			),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (" + wrap +
				"SELECT `posts`.`user_id` AS `c1` FROM `posts` WHERE (user_id IN (SELECT id FROM USERS))) AS materialized)",
		},
		{
			"a filter built at render time",
			update(qb.In{
				Left: id,
				Sub: sel(userID, posts, qb.Lazy{
					Build: func(context.Context) (qb.Cond, error) {
						return qb.In{Left: userID, Sub: ofUsers}, nil
					},
				}),
			}),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (" + wrap +
				"SELECT `posts`.`user_id` AS `c1` FROM `posts` WHERE `posts`.`user_id` IN " +
				"(SELECT `users`.`id` FROM `users`)) AS materialized)",
		},
		{
			"a limited subquery is a derived table already",
			update(qb.In{
				Left: id,
				Sub: &qb.Select{
					Columns: []qb.Selected{{Value: id}},
					From:    qb.Source{Table: users},
					Limit:   new(1),
				},
			}),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (SELECT * FROM " +
				"(SELECT `users`.`id` FROM `users` LIMIT ?) AS limited)",
		},
		{
			"raw SQL outside a subquery does not mark the next one",
			update(qb.Fragment{SQL: "users.age > 1"}, qb.In{Left: id, Sub: ofPosts}),
			"UPDATE `users` SET `age` = ? WHERE (users.age > 1) AND `users`.`id` IN " +
				"(SELECT `posts`.`user_id` FROM `posts`)",
		},
		{
			"columns of one name get names of their own",
			update(qb.In{
				Left: qb.Row{Values: []qb.Expr{id, id}},
				Sub: &qb.Select{
					Columns: []qb.Selected{
						{Value: id},
						{Value: qb.Column{Table: posts, Name: "id"}},
					},
					From: qb.Source{Table: users},
					Joins: []qb.Join{
						{
							Source: qb.Source{Table: posts},
							On:     []qb.Cond{qb.Compare{Left: userID, Op: qb.OpEq, Right: id}},
						},
					},
				},
			}),
			"UPDATE `users` SET `age` = ? WHERE (`users`.`id`, `users`.`id`) IN (" + wrap +
				"SELECT `users`.`id` AS `c1`, `posts`.`id` AS `c2` FROM `users` JOIN `posts` ON " +
				"`posts`.`user_id` = `users`.`id`) AS materialized)",
		},
		{
			"a source named like the table",
			update(qb.In{
				Left: id,
				Sub: &qb.Select{
					Columns: []qb.Selected{{Value: qb.Column{Name: "id"}}},
					From:    qb.Source{Name: "users"},
				},
			}),
			"UPDATE `users` SET `age` = ? WHERE `users`.`id` IN (" + wrap +
				"SELECT `id` AS `c1` FROM `users`) AS materialized)",
		},
		{
			"a SELECT reads freely",
			&qb.Select{
				Columns: []qb.Selected{{Value: id}},
				From:    qb.Source{Table: users},
				Where:   []qb.Cond{qb.In{Left: id, Sub: ofUsers}},
			},
			"SELECT `users`.`id` FROM `users` WHERE `users`.`id` IN (SELECT `users`.`id` FROM `users`)",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := mysql.Dialect.Render(tt.stmt)
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}

			for _, d := range []dialect.Dialect{mysql.MariaDB, postgres.Dialect} {
				other, _, err := d.Render(tt.stmt)
				if err != nil {
					t.Fatal(err)
				}

				if strings.Contains(other, "materialized") {
					t.Errorf("%s wrapped a subquery: %s", d.Name(), other)
				}
			}
		})
	}
}
