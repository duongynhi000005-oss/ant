# The problem

Go database libraries mostly make you choose. Reflection-based ORMs work with plain structs but check little at compile time. Code generators check a lot but add a build step, a generated API, and often a schema source other than your Go types. This document lists the specific problems ant is meant to address. Each "before" comes from the cited library's documentation. Each "after" uses the [target API](03-vision.md#target-api).

Library versions and documentation were checked on 2026-10-31.

## Stringly-typed queries

In GORM's generics API, in Bun, and in xorm, the column name, the operator, and the placeholder are all inside a string. The [GORM generics guide](https://gorm.io/docs/the_generics_way.html), the [Bun guide](https://bun.uptrace.dev/guide/), and the [xorm README](https://gitea.com/xorm/xorm/src/branch/main/README.md) show this form:

```go
// GORM
users, err := gorm.G[User](db).Where("age <= ?", 18).Find(ctx)

// Bun
err := db.NewSelect().Model(&users).Where("name LIKE ?", "%John%").Scan(ctx)

// xorm
has, err := engine.Where("name = ?", name).Desc("id").Get(&user)
```

If a field is renamed, these strings do not change and the compiler has nothing to report. The mistake appears when the query runs. The same is true of a value with the wrong type.

```go
// ant
adults, err := db.Users.Where(ant.Lte(&U.Age, 18)).All(ctx)

// ant.Lte(&U.Age, "18") is a compile error: the value must be an int.
// Renaming User.Age in an editor updates &U.Age everywhere.
```

## Zero-value update bugs

Go cannot tell a field that was set to `0` from a field that was never set. Several libraries treat the zero value as "not set".

The [GORM update docs](https://gorm.io/docs/update.html) state: "When updating with struct, GORM will only update non-zero fields." The [xorm README](https://gitea.com/xorm/xorm/src/branch/main/README.md) says Update "will update non-empty and non-zero fields except when you use Cols, AllCols and so on." The [SQLBoiler README](https://github.com/aarondl/sqlboiler) warns that with `boil.Infer()`, a default-true boolean set to `false` may not be written unless it is whitelisted.

```go
// GORM docs example
db.Model(&user).Updates(User{Name: "hello", Age: 18, Active: false})
// UPDATE users SET name='hello', age=18, updated_at = '2013-11-17 21:34:10' WHERE id = 111;
// Active: false is not written.
```

This code compiles and looks correct, and it can pass any test that does not use zero values.

ant never reads meaning into a zero value. An update comes either from comparing against a snapshot or from an explicit assignment:

```go
// Change tracking: compares against the loaded values, so 0 and false are real changes.
user.Age = 0
user.Active = false
err = db.SaveChanges(ctx)

// Explicit assignment
_, err = db.Users.Where(ant.Eq(&U.ID, id)).Update(ctx,
    ant.Set(&U.Age, 0),
    ant.Set(&U.Active, false))
```

## N+1 from implicit loading

Lazy loading makes a field read look free when it costs a query. The [EF Core documentation](https://learn.microsoft.com/en-us/ef/core/querying/related-data/lazy) warns that lazy loading "can cause unneeded extra database roundtrips to occur (the so-called N+1 problem)."

The same pattern exists in Go without lazy fields. In [pop](https://gobuffalo.io/documentation/database/relations/), the default `Eager` loads each association with one query per parent row, and the batched form must be requested with `EagerPreload`:

```go
// pop: one query for users, then one more per user for that user's books
err := tx.Eager("Books").All(&users)
```

ant has no field access that performs I/O. Loading is explicit and batched, so a field that was not loaded is `nil`, not a hidden query:

```go
// ant: one query for users, one query for their posts
users, err := db.Users.Include(&U.Posts).All(ctx)
```

## Heavy codegen clients

Generators solve the type-safety problem and are the strongest option in Go today. The cost is a second artifact. With [ent](https://entgo.io/docs/getting-started), you write the schema in a Go DSL rather than as plain structs and generate a client, plus files per entity:

```go
func (User) Fields() []ent.Field {
    return []ent.Field{
        field.Int("age").Positive(),
        field.String("name").Default("unknown"),
    }
}
```

```go
//   go generate ./ent
// produces client.go, user.go, user_create.go, user/where.go, and more.
u, err := client.User.Query().Where(user.Name("a8m")).Only(ctx)
```

[go-jet](https://github.com/go-jet/jet) and [Bob](https://bob.stephenafamo.com/docs/code-generation/intro) generate from a live database, so regeneration is part of every schema change. [SQLBoiler](https://github.com/aarondl/sqlboiler) works the same way and is in maintenance mode.

ant's models are the structs you wrote, and the query API is generic code in one package:

```go
type User struct {
    ant.Base
    Name string
    Age  int
}

var U = ant.Fields[User]()

u, err := db.Users.Where(ant.Eq(&U.Name, "a8m")).First(ctx)
```

## Missing change tracking

[EF Core](https://learn.microsoft.com/en-us/ef/core/change-tracking/) tracks queried entities and writes only the values that changed when `SaveChanges` runs. None of the Go libraries reviewed in [existing solutions](02-existing-solutions.md) documents an equivalent. Updates name their fields explicitly, as in [ent](https://entgo.io/docs/crud), or send the whole row, as in [GORM's `Save`](https://gorm.io/docs/update.html), which "will save all fields".

```go
// ent: each changed field is repeated as a setter call.
u, err = u.Update().SetAge(30).Save(ctx)
```

This becomes a problem when business logic changes several entities across several functions. Callers must carry a list of what changed, or write everything.

```go
// ant: change the loaded values, then save the unit of work.
user.Age = 30
order.Status = "paid"
err = db.SaveChanges(ctx) // one minimal UPDATE per changed entity
```

## Poor relation ergonomics

Relation names are strings in GORM, Bun, and pop, including nested paths, and they fail at runtime when wrong ([GORM](https://gorm.io/docs/preload.html), [Bun](https://bun.uptrace.dev/guide/relations.html), [pop](https://gobuffalo.io/documentation/database/relations/)):

```go
db.Preload("Orders.OrderItems.Product").Find(&users) // GORM
```

Generated clients avoid strings but change where the data lives. In ent, loaded relations are reached through a generated field, as in `u.Edges.Pets` ([docs](https://entgo.io/docs/eager-load)).

In ant, a relation is a field on the struct, and it is referenced by a typed pointer. Loading a non-relation field fails to compile:

```go
users, err := db.Users.Include(&U.Posts).All(ctx)
fmt.Println(users[0].Posts[0].Title)

// db.Users.Include(&U.Name) is a compile error: *string does not match *[]R.
```

## Weak migration stories

Migration support varies widely, and several libraries leave it to other tools:

- [GORM's AutoMigrate](https://gorm.io/docs/migration.html) adds schema objects but "WON'T delete unused columns".
- [sqlc](https://docs.sqlc.dev/en/latest/howto/ddl.html) "does not perform database migrations for you". It parses the files of other tools.
- [SQLBoiler](https://github.com/aarondl/sqlboiler) and [Bob](https://bob.stephenafamo.com/docs/code-generation/intro) leave schema changes to a separate tool.
- [Bun](https://bun.uptrace.dev/guide/migrations.html) runs Go or SQL migrations, and its [`AutoMigrator`](https://pkg.go.dev/github.com/uptrace/bun/migrate) compares models with the live database to produce them. This is the closest Go option to the goal below. It needs a live database as the comparison point.
- [ent](https://entgo.io/docs/versioned-migrations) diffs the schema with Atlas and produces versioned SQL files without connecting to production. It requires ent's schema DSL.

ant follows the EF Core model: compare the plain model against a stored snapshot, generate a reviewable migration as plain SQL, then apply it. Generating needs no database, and the files work with any tool. See the [migrations guide](migrations.md).
