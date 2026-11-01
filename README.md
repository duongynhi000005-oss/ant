# ant

![status: pre-alpha](https://img.shields.io/badge/status-pre--alpha-lightgrey)

ant is a Go ORM for Go 1.27 and later, in which models are plain structs and queries are checked by the compiler, with no code generation step. You declare a typed database context in the style of Entity Framework Core, reference columns through field pointers such as `&U.Age` instead of strings, and get change tracking through `SaveChanges`. Errors stay explicit and appear at terminal calls. ant targets PostgreSQL, MySQL, MariaDB, SQLite, and SQL Server on top of `database/sql`.

ant is in development, and has not been released. Models, relations, change tracking, transactions, global filters, soft deletes, bulk writes, interceptors, and migrations are in place on PostgreSQL. MySQL, MariaDB, SQLite, and SQL Server run the same queries and writes, without migrations yet: [databases](docs/databases.md) has the details. The [roadmap](docs/ROADMAP.md) shows the current state, the [feature examples](tests/features/) show how everything is used, the [recipes](docs/recipes/) cover common jobs, and [ant-examples](https://github.com/skudra-go/ant-examples) has small projects that use it.

## A taste

```go
type User struct {
    ant.Base               // ID, CreatedAt, UpdatedAt
    Name  string
    Age   int
    Posts []Post           // 1:N via Post.UserID
}

type Post struct {
    ant.Base
    UserID int64
    Title  string
}

type AppDB struct {
    ant.Context
    Users ant.Table[User]
    Posts ant.Table[Post]
}

var U = ant.Fields[User]() // field probe: &U.Age identifies the age column
db, err := ant.Open[AppDB](ctx, postgres.Dialect, dsn)
alice := &User{Name: "Alice", Age: 30}
err = db.Users.Insert(ctx, alice)
user, err := db.Users.Find(ctx, alice.ID)
user.Age++
err = db.SaveChanges(ctx) // UPDATE users SET age = $1, ... WHERE id = $2

adults, err := db.Users.
    Where(ant.Gt(&U.Age, 18)). // ant.Gt(&U.Age, "18") does not compile
    Include(&U.Posts).         // batched, no N+1
    All(ctx)
```

## Documentation

- [The problem](docs/01-problem.md): what ant is meant to fix, with before and after code.
- [Existing solutions](docs/02-existing-solutions.md): an assessment of eleven Go database libraries, with a comparison table.
- [Vision](docs/03-vision.md): design decisions, the target API, how field probes work, known limits, and non-goals.
- [Migrations](docs/migrations.md): generating and applying schema changes, adopting an existing database, and leaving ant's tooling.
- [Decisions](docs/DECISIONS.md): refinements to the original brief and the reasons for them.
- [Roadmap](docs/ROADMAP.md): every planned feature by phase, and what is not planned.
- [Reference](docs/reference/): feature catalogs of EF Core, Eloquent, and Go libraries.
- [Feature examples](tests/features/): one small test per feature, showing how to use it and how it behaves at the edges.

## Development

Requires Go 1.27 or later, Docker, `make`, and [staticcheck](https://staticcheck.dev/).

```sh
make test              # unit tests, no database needed
make fmt               # lay the code out, in every module
make lint              # go vet, staticcheck, and fail on anything fmt would change
make db-up DATABASES=postgres  # start PostgreSQL, or leave DATABASES out for all
make test-pg           # run the API tests and feature examples on PostgreSQL
make bench             # compare ant with plain database/sql
make test-integration  # run them on all five databases
make db-down           # stop the databases and discard their data
```

Tests for features that do not exist yet fail with `ErrNotImplemented`.

## License

Apache License 2.0. See [LICENSE](LICENSE).
