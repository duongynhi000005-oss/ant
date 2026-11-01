# ant — Implementation Handoff Brief

Oct 31, 2026 · @Alberts

## Purpose of this brief

This brief hands the **ant** project (a Go ORM) to an implementing AI. Read it fully once, then execute Steps 1 to 4 in order. Each step produces files in the repository. Stop after each step and present the result for review before starting the next.

The design decisions below were already made with the project owner. Treat them as fixed unless a step explicitly asks you to evaluate them. If something in this brief is impossible or contradicts the Go spec, stop and report it instead of silently working around it.

## Project summary

ant is a Go ORM that brings the developer experience of Entity Framework Core and Laravel Eloquent to Go without codegen, without stringly-typed queries, and without hiding Go's explicit error handling.

**The problem.** Existing Go ORMs fall into two camps. Reflection-based ones (GORM) are convenient but stringly typed (`Where("age > ?", 18)`), full of zero-value surprises, and lose compile-time safety. Codegen-based ones (ent, sqlboiler, go-jet) are typed but generate large clients or require a schema DSL, which feels foreign. Nothing combines plain structs, compile-time-checked queries, and a small footprint.

**The goal.** Users define plain Go structs, declare an EF-style typed database context, and write queries where column references, value types, relations, and projections are all checked by the compiler. Renaming a struct field in an editor updates every query.

**Baseline facts.**

- Minimum Go version: **1.27**. The API depends on generic methods (new in 1.27), `new(expr)` (1.26), self-referential generic constraints (1.26), field-selector keys in struct literals (1.27), and `iter.Seq2` (1.23).
- Package name: `ant`. Module path placeholder: `github.com/<org>/ant` (owner will supply the real path).
- Target databases: PostgreSQL, MySQL, MariaDB, SQLite, SQL Server. PostgreSQL is the reference dialect; every feature lands there first.
- Built on `database/sql`. No hard dependency on a specific driver in the core package.

## Core design decisions

These are fixed. Every API and implementation choice must be consistent with them.

1. **Plain structs.** Models are ordinary Go structs. No required base type, no generated model code. An optional embeddable `ant.Base` provides `ID`, `CreatedAt`, `UpdatedAt`.
2. **Field probes instead of strings.** `ant.Fields[T]()` returns a `*T` probe. At first use, reflection maps each field's memory address within the probe to its column. Queries reference columns as `&U.Age`. Generic condition functions (`ant.Gt[V](field *V, v V)`) make value-type mismatches compile errors.
3. **Typed end to end.** Ordering, grouping, eager loading, joins, projections, and assignments all take typed field pointers or typed closures. No `any` in the public query API except raw SQL arguments.
4. **Generic methods for fluent chains.** Type-changing operations (`Join[J]`, `Select[R]`, `Include[R]`) are generic methods on concrete builder types. Because Go forbids generic methods in interfaces, builders are concrete structs, never interfaces.
5. **EF-style context.** Users declare a struct embedding `ant.Context` with `ant.Table[T]` fields. `ant.Open[DB]` populates it. Transactions hand back the same typed context.
6. **Errors only at terminal calls.** Builder methods never return errors. They accumulate them, and terminal calls (`All`, `First`, `Find`, `Exec`, `Insert`, `SaveChanges`) return them. Every terminal call takes `context.Context` first.
7. **No hidden I/O.** No lazy loading through field access. Eager loading is explicit and batched to avoid N+1.
8. **No zero-value guessing.** The ORM never infers "unset" from a zero value. Partial updates use change tracking or explicit `ant.Set` assignments. Nullable columns use pointers or `sql.Null[T]`.
9. **Opt-in change tracking.** Entities loaded through the context are snapshotted. `SaveChanges` diffs and writes minimal `UPDATE`s. A `NoTracking()` builder option disables it.
10. **Escape hatches are first-class.** Raw SQL that scans into the same structs is trivial and documented.
11. **Dialects are pluggable** behind one interface from day one.
12. **Readability beats cleverness.** Public API names read like English (`Where`, `OrderBy`, `Include`, `All`). Internals use `unsafe` only where documented and tested.

## Target API

This is the user-facing API the project aims for. It is the north star for Steps 1, 3 and 4. Exact signatures may be refined, but the reading experience must stay this clean.

**Models and context**

```go
type User struct {
    ant.Base               // ID, CreatedAt, UpdatedAt
    Name  string
    Email string `ant:"unique"`
    Age   int
    Bio   *string          // nullable
    Posts []Post           // 1:N via Post.UserID (convention)
}

type Post struct {
    ant.Base
    UserID    int64
    Title     string
    Published bool
    User      *User        // N:1
}

type AppDB struct {
    ant.Context
    Users ant.Table[User]
    Posts ant.Table[Post]
}

// Field probes, declared once per model
var (
    U = ant.Fields[User]()
    P = ant.Fields[Post]()
)
```

**Optional fluent configuration** (overrides conventions; tags stay minimal)

```go
type Entity[E Entity[E]] interface {
    Configure(b *ant.ModelBuilder[E])
}

func (User) Configure(b *ant.ModelBuilder[User]) {
    b.Table("users")
    b.Index(&U.Email).Unique()
    b.HasMany(&U.Posts).ForeignKey(&P.UserID).OnDelete(ant.Cascade)
}
```

**Operations**

```go
db, err := ant.Open[AppDB](ctx, postgres.Dialect, dsn)

// Insert a graph
alice := &User{Name: "Alice", Email: "alice@example.com", Age: 30,
    Posts: []Post{{Title: "Hello"}, {Title: "Draft ideas"}}}
err = db.Users.Insert(ctx, alice)

// Find by primary key
user, err := db.Users.Find(ctx, alice.ID)
if errors.Is(err, ant.ErrNotFound) { /* ... */ }

// Change tracking
user.Age++
user.Bio = new("Gopher since 2012")
err = db.SaveChanges(ctx)

// Typed query with eager loading
adults, err := db.Users.
    Where(ant.Gt(&U.Age, 18)).
    OrderBy(&U.Name).
    Include(&U.Posts).
    All(ctx)

// Bulk update and delete
_, err = db.Posts.Where(ant.Eq(&P.UserID, user.ID)).Update(ctx, ant.Set(&P.Published, true))
_, err = db.Posts.Where(ant.Like(&P.Title, "Draft%")).Delete(ctx)

// Join through a relation and project
type Summary struct{ Author, Title string }
summaries, err := db.Posts.
    Join(&P.User).
    Where(ant.Eq(&P.Published, true)).
    Select(func(p *Post, u *User) Summary { return Summary{u.Name, p.Title} }).
    All(ctx)

// Streaming
for post, err := range db.Posts.Where(ant.Eq(&P.Published, true)).Each(ctx) {
    if err != nil { return err }
    fmt.Println(post.Title)
}

// Transactions return the same typed context
err = db.Tx(ctx, func(tx *AppDB) error {
    bob := &User{Name: "Bob", Email: "bob@example.com"}
    if err := tx.Users.Insert(ctx, bob); err != nil { return err }
    return tx.Posts.Insert(ctx, &Post{UserID: bob.ID, Title: "First!"})
})

// Raw SQL into the same structs
rows, err := ant.Raw[User](db, "SELECT * FROM users WHERE age > $1", 18).All(ctx)
```

**Key generic signatures**

```go
func Fields[T any]() *T
func Eq[V comparable](field *V, v V) Cond
func Gt[V cmp.Ordered](field *V, v V) Cond
func EqCol[V comparable](a, b *V) Cond
func Set[V any](field *V, v V) Assignment

func (q *Query[T]) Where(c ...Cond) *Query[T]
func (q *Query[T]) OrderBy[V cmp.Ordered](field *V) *Query[T]
func (q *Query[T]) Include[R any](rel *[]R) *Query[T]      // 1:N
func (q *Query[T]) IncludeOne[R any](rel **R) *Query[T]    // N:1, 1:1
func (q *Query[T]) Join[J any](rel **J) *Join2[T, J]
func (j *Join2[A, B]) Select[R any](f func(*A, *B) R) *Projection[R]
func (q *Query[T]) Each(ctx context.Context) iter.Seq2[T, error]
```

## Internal architecture

The public API lives in the root `ant` package. Internals are split so that SQL generation, metadata, and execution can be tested independently. Data flows one way: model metadata feeds the query builder, which produces an AST, which a dialect renders to SQL, which the executor runs and the scanner maps back to structs.

| Package | Responsibility |
| --- | --- |
| `ant` (root) | Public API: `Open`, `Context`, `Table[T]`, `Query[T]`, `Join2`, `Projection`, conditions, `Set`, errors, `Raw` |
| `internal/meta` | Model registry: reflect structs once, column names, PKs, relations, conventions, probe address → column map |
| `internal/qb` | Dialect-neutral query AST (select, insert, update, delete, joins, conditions) |
| `internal/track` | Change tracker: identity map, snapshots, diffing, unit of work ordering |
| `internal/scan` | Row scanning into structs, nested relation stitching, uses `sql.ConvertAssign` |
| `internal/exec` | Small non-generic `Executor` interface over `*sql.DB`, `*sql.Tx`, `*sql.Conn` (the test seam) |
| `dialect` | `Dialect` interface plus shared helpers |
| `dialect/postgres`, `dialect/mysql`, `dialect/sqlite`, `dialect/sqlserver` | Placeholders, quoting, `RETURNING` / `LastInsertId`, upserts, type mapping, DDL |
| `migrate` | Later phase: schema snapshot, diff, DDL generation, CLI |
| `anttest` | Public test helpers for users (fixtures, transactional test isolation) |
| `internal/testdb` | Multi-database harness for ant's own integration tests |

## Step 1: Project documentation

Write three Markdown files that explain why ant exists and what it will be. Audience: Go developers evaluating the project and future contributors.

**Deliverables**

- `README.md`: one-paragraph pitch, a 30-line "taste" example taken from the Target API, status badge placeholder, links to the docs below.
- `docs/01-problem.md`: the pain points with concrete before/after code. Cover stringly-typed queries, zero-value update bugs, N+1 from implicit loading, heavy codegen clients, missing change tracking, poor relation ergonomics, weak migration stories.
- `docs/02-existing-solutions.md`: a short assessment of GORM, ent, Bun, sqlc, sqlboiler, go-jet, Bob, sqlx, upper/db, xorm, and pop. For each: approach (reflection, codegen, SQL-first, builder), strengths, weaknesses, maintenance status. End with a comparison table across typed queries, codegen required, relations, change tracking, migrations, and dialects.
- `docs/03-vision.md`: the desired result. Restate the core design decisions, show the Target API, explain the field-probe mechanism, and list explicit non-goals (no NoSQL, no lazy loading through field access, no active record `user.Save()` in v1).

**Acceptance criteria**

- Every claim about another library is checked against its current official docs, with links.
- Code examples compile conceptually against the Target API (they will be turned into real tests later).
- No marketing language. Plain, specific, technical prose.

## Step 2: Feature reference of other ORMs

Build an exhaustive reference of what other projects let their users do. This is a catalog, not an analysis: long bullet lists, one short line per feature, no explanations. It will be used to make sure the roadmap misses nothing.

**Deliverables**

- `docs/reference/ef-core.md`: Entity Framework Core (latest stable).
- `docs/reference/eloquent.md`: Laravel Eloquent, plus the Laravel query builder, migrations, factories and seeders where they relate to models.
- `docs/reference/go-orms.md`: one section per library: GORM, ent, Bun, sqlc, sqlboiler, go-jet, Bob, sqlx, upper/db, xorm, pop, and Atlas (migrations only).

**Required structure for each library.** Group bullets under these headings, and add headings when a library has features outside them:

- Model definition and mapping (conventions, attributes/tags, fluent config, naming, owned/embedded types, inheritance)
- Keys and identity (PK types, composite keys, generated values, UUIDs, sequences)
- Relations (1:1, 1:N, N:M, polymorphic, self-referencing, through/pivot tables, pivot attributes)
- Querying (filters, operators, ordering, paging, grouping, aggregates, subqueries, unions, CTEs, window functions, JSON queries, full-text, raw fragments)
- Loading (eager, lazy, explicit, filtered includes, split queries, counts of relations)
- Writing (insert, bulk insert, update, bulk update, upsert, delete, soft delete, restore, increment/decrement)
- Change tracking and unit of work (states, attach/detach, identity map, no-tracking, concurrency tokens)
- Transactions and connections (savepoints, isolation levels, retries, read replicas, pooling)
- Lifecycle (hooks, events, observers, interceptors, global query filters, scopes)
- Value handling (type converters, casts, accessors/mutators, enums, JSON columns, encryption, computed columns)
- Schema and migrations (code-first, DB-first, diffing, rollback, seeding, scaffolding)
- Performance (compiled queries, batching, caching, streaming, chunking, cursors)
- Developer tooling (logging, SQL inspection, diagnostics, CLI, codegen, testing utilities, factories)
- Database support (dialects, provider-specific features)

**Acceptance criteria**

- At least 150 bullets for EF Core and 120 for Eloquent; Go libraries as many as they genuinely support.
- Each file cites the documentation pages used, with links and the version covered.
- No feature invented or assumed. If unsure, omit it.

## Step 3: Feature roadmap

Write `docs/ROADMAP.md`: the full, exhaustive list of features ant will implement over time, organized into phases. Derive it from the Step 2 reference. Every feature found there must appear either in a phase or in an explicit "Not planned" section with a one-line reason.

**Format**

- One `- [ ]` checkbox per feature, one line each, grouped by phase and then by category (same categories as Step 2).
- Each feature line names the public API it maps to where known, for example `- [ ] Upsert on conflict: Table[T].Upsert(ctx, v, ant.OnConflict(&U.Email))`.
- Each phase ends with exit criteria (what must pass for the phase to be done).

**Phase skeleton to start from** (refine and extend)

1. **v0.1 Foundations**: model registry, probes, conventions, PostgreSQL dialect, Insert/Find/Where/OrderBy/Limit/All/First, Update/Delete, errors, raw SQL, test harness.
2. **v0.2 Relations**: 1:N, N:1, 1:1, Include/IncludeOne with batched loading, joins via relations, projections with Select.
3. **v0.3 Unit of work**: change tracking, SaveChanges, identity map, NoTracking, graph inserts, transactions, savepoints.
4. **v0.4 Dialects**: MySQL, MariaDB, SQLite, SQL Server at parity with PostgreSQL.
5. **v0.5 Query power**: aggregates, GroupBy/Having, subqueries, Exists, unions, CTEs, paging helpers, streaming, upserts, bulk operations.
6. **v0.6 Modeling depth**: N:M with pivot tables, composite keys, embedded/owned types, enums, JSON columns, value converters, soft deletes, timestamps, global filters, scopes.
7. **v0.7 Lifecycle and ops**: hooks/interceptors, logging, tracing (OpenTelemetry), query inspection, optimistic concurrency.
8. **v0.8 Migrations**: schema snapshot, diff, generated migrations, apply/rollback, seeding, CLI.
9. **v0.9 Tooling**: `go vet` analyzer that flags non-probe field pointers, fixtures and factories in `anttest`, documentation site.
10. **v1.0 Stability**: API freeze, performance benchmarks against GORM/Bun/sqlc, security review, upgrade guide.

## Step 4: Package skeleton and test harness

Create a compiling skeleton of the full public API from the Target API section, plus a test harness that runs the same integration tests against every supported database. Most method bodies return `ant.ErrNotImplemented` at this stage. The point is that the shape compiles, the API tests express the requirements, and the database matrix runs.

**Repository layout**

```text
ant/
├── go.mod                      // go 1.27
├── ant.go                      // Open, Context, errors
├── table.go                    // Table[T]
├── query.go                    // Query[T], Join2, Projection
├── cond.go                     // Cond, Eq, Gt, Lt, In, Like, IsNull, And, Or, Not, EqCol
├── assign.go                   // Assignment, Set
├── fields.go                   // Fields[T] probes
├── config.go                   // Entity, ModelBuilder[E]
├── raw.go                      // Raw[T]
├── tx.go
├── internal/{meta,qb,track,scan,exec,testdb}/
├── dialect/{dialect.go,postgres,mysql,sqlite,sqlserver}/
├── anttest/
├── tests/                      // integration tests (API requirements)
│   ├── models_test.go          // User, Post, AppDB fixtures
│   ├── crud_test.go
│   ├── query_test.go
│   ├── relations_test.go
│   ├── tracking_test.go
│   └── tx_test.go
├── docker-compose.yml
├── Makefile
└── .github/workflows/ci.yml
```

**Core internal contracts**

```go
// internal/exec
type Executor interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// dialect
type Dialect interface {
    Name() string
    Placeholder(n int) string        // "$1" or "?"
    Quote(ident string) string
    SupportsReturning() bool
    Render(stmt qb.Statement) (sql string, args []any, err error)
}
```

**Test harness.** Unit tests (SQL rendering, metadata, probes, diffing) run with plain `go test ./...` and need no database. Integration tests in `tests/` run against every database listed in `ANT_TEST_DATABASES` and skip cleanly when one is unreachable.

```go
// internal/testdb
func ForEach(t *testing.T, fn func(t *testing.T, target Target)) {
    for _, target := range Targets() {          // from ANT_TEST_DATABASES
        t.Run(target.Name, func(t *testing.T) {
            target.Connect(t)                   // t.Skip if unreachable
            target.ResetSchema(t)
            fn(t, target)
        })
    }
}

// tests/query_test.go
func TestWhereOrderByAll(t *testing.T) {
    testdb.ForEach(t, func(t *testing.T, target testdb.Target) {
        app := openApp(t, target)          // ant.Open[AppDB] with the target's dialect and DSN
        seedUsers(t, app, "Alice", 30, "Bob", 17)
        got, err := app.Users.Where(ant.Gt(&U.Age, 18)).OrderBy(&U.Name).All(t.Context())
        require.NoError(t, err)
        require.Equal(t, []string{"Alice"}, names(got))
    })
}
```

**Databases**

| Target | How it runs | Default DSN env var |
| --- | --- | --- |
| PostgreSQL 17 | docker-compose service | `ANT_PG_DSN` |
| MySQL 8.4 | docker-compose service | `ANT_MYSQL_DSN` |
| MariaDB 11 | docker-compose service | `ANT_MARIADB_DSN` |
| SQLite | in-process (pure-Go driver, no cgo) | none needed |
| SQL Server 2022 | docker-compose service | `ANT_MSSQL_DSN` |

Verify image tags and driver choices are current before pinning them.

**Makefile targets**: `make test` (unit only), `make db-up` / `make db-down`, `make test-integration` (all databases), `make test-pg` (and one per database), `make lint` (`go vet` plus staticcheck).

**CI**: a GitHub Actions workflow with a unit job and an integration matrix job (one entry per database, using service containers).

**Acceptance criteria**

- `go build ./...` and `go vet ./...` pass on Go 1.27.
- `make test` passes. Unit tests cover probe address-to-column mapping for `User` and `Post`.
- `make test-integration` runs every database in the matrix. API tests compile and fail only with `ErrNotImplemented`, never with build errors.
- The integration suite covers every operation in the Target API section, so it doubles as the requirements checklist.
- Every exported identifier has a doc comment.

## Working rules for the implementing AI

- Work one step at a time. After each step, summarize what you produced, list open questions, and wait for review.
- Verify facts about Go and other libraries against official documentation before writing them down. Do not rely on memory for versions, APIs, or feature lists.
- Go 1.27 features (generic methods) are required. If a toolchain without them is detected, stop and report.
- Keep the public API exactly as readable as the Target API section. If an implementation detail forces an uglier API, propose alternatives instead of shipping it.
- No new dependencies in the root package beyond the standard library. Test-only dependencies (assertion library, database drivers) are fine in test files and `internal/testdb`.
- Prefer small, well-named files and functions. Every exported identifier gets a doc comment.
- When a requirement in this brief is ambiguous, pick the option closest to EF Core behavior, note the decision in `docs/DECISIONS.md`, and continue.
