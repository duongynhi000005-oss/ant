# Existing solutions

This is a short assessment of eleven Go database libraries. It records what each project's own documentation says, plus repository status as of 2026-10-31. "Not documented" means the pages reviewed do not describe the feature. It does not prove the feature is absent.

Release dates come from the GitHub releases API, or the Gitea tags API for xorm. A library that is quiet is not necessarily abandoned: small, stable libraries can need few changes.

## GORM

- **Approach:** Reflection-based ORM. An optional CLI adds code generation.
- **Strengths:** Broad [feature set](https://gorm.io/docs/index.html): associations (including many-to-many, polymorphism, and single-table inheritance), hooks, eager loading, and auto migration. [Officially supports](https://gorm.io/docs/connecting_to_the_database.html) MySQL, PostgreSQL, GaussDB, SQLite, SQL Server, TiDB, and Oracle. The [generics API](https://gorm.io/docs/the_generics_way.html) (v1.30.0 and later) adds typed results and removes `Save` and `FirstOrCreate`. The [GORM CLI](https://github.com/go-gorm/cli) generates typed field helpers, as in `generated.User.Age.Gt(18)`.
- **Weaknesses:** Conditions are strings unless you adopt the generator, and relation names in `Preload` are strings. [Struct updates skip zero values](https://gorm.io/docs/update.html). `AutoMigrate` [does not delete unused columns](https://gorm.io/docs/migration.html).
- **Maintenance:** Active. Latest release v1.31.2 (June 2026), repository activity in October 2026.

GORM with its CLI is the closest existing option to ant's type-safety goal. The difference is that its field helpers are generated, while ant's come from the struct itself.

## ent

- **Approach:** Code generation from a [schema written in a Go DSL](https://entgo.io/docs/getting-started).
- **Strengths:** Typed queries and mutations through generated predicates. [Eager loading](https://entgo.io/docs/eager-load) with per-edge filters and nesting. [Versioned migrations](https://entgo.io/docs/versioned-migrations) diffed with Atlas. [Upserts](https://entgo.io/docs/crud) behind a feature flag.
- **Weaknesses:** Models are generated, not plain structs. Relations sit under `Edges`. [Updates](https://entgo.io/docs/crud) use explicit setter chains, and no change tracking is documented. The [dialect list](https://entgo.io/docs/dialects) has no SQL Server.
- **Maintenance:** Active. Latest release v0.14.6 (March 2026), repository activity in October 2026.

## Bun

- **Approach:** ["SQL-first"](https://bun.uptrace.dev/guide/) query builder and ORM layer over struct tags, without code generation.
- **Strengths:** Queries read like SQL. [Relations](https://bun.uptrace.dev/guide/relations.html) include has-one, belongs-to, has-many, polymorphic has-many, and many-to-many. [Go or SQL migrations](https://bun.uptrace.dev/guide/migrations.html), an [`AutoMigrator`](https://pkg.go.dev/github.com/uptrace/bun/migrate) that compares models with the live database, and [fixtures](https://bun.uptrace.dev/guide/fixtures.html). [Dialects](https://bun.uptrace.dev/guide/drivers.html) for PostgreSQL, MySQL and MariaDB, SQL Server, SQLite, and Oracle. [Bulk updates](https://bun.uptrace.dev/guide/query-update.html).
- **Weaknesses:** Conditions and relation names are strings, so a mistake appears at runtime. Change tracking is not documented. `OmitZero` is an explicit choice, but it has the same zero-value ambiguity when used.
- **Maintenance:** Active. Latest release v1.3.0 (October 2026).

## sqlc

- **Approach:** SQL-first code generation. You write SQL, and [sqlc generates typed Go code](https://docs.sqlc.dev/en/latest/index.html).
- **Strengths:** Queries are real SQL, verified at generation time. The call sites are plain typed functions. [Go support](https://docs.sqlc.dev/en/latest/reference/language-support.html) is Stable for MySQL and PostgreSQL, Beta for SQLite, ClickHouse, DuckDB, Spanner, and SQL Server.
- **Weaknesses:** Query shape is fixed at generation time. Runtime-composed filters have no native builder: the common technique is [nullable parameters](https://docs.sqlc.dev/en/latest/howto/named_parameters.html) in SQL, and a request for dynamic queries is [an open discussion](https://github.com/sqlc-dev/sqlc/discussions/364). It is not an ORM, so relations and change tracking are out of scope. It [does not run migrations](https://docs.sqlc.dev/en/latest/howto/ddl.html), but parses six migration tools' files.
- **Maintenance:** Active. Latest release v1.31.1 (April 2026), repository activity in October 2026.

## SQLBoiler

- **Approach:** [Database-first code generation](https://github.com/aarondl/sqlboiler). It generates models from an existing schema.
- **Strengths:** Generated typed filters (`models.PilotWhere.Name.EQ("Tim")`) and typed relation names (`models.JetRels.Pilot`). Automatic timestamps. Supports PostgreSQL, MySQL, SQL Server 2012+, and SQLite, plus CockroachDB through a third-party driver.
- **Weaknesses:** String query mods coexist with the typed helpers. `boil.Infer()` has the zero-value problem. It does not manage migrations, and multi-column foreign keys are not supported.
- **Maintenance:** In maintenance mode. The README says it generally accepts no new features and recommends Bob or sqlc as actively maintained alternatives. Latest release v4.19.5 (June 2025).

## go-jet

- **Approach:** [Typed SQL builder](https://github.com/go-jet/jet) with code generated from a live database. Its README states it is "not an ORM".
- **Strengths:** Column types are enforced at compile time, as in `Film.Length.GT(Int32(180))`. Maps joined results into nested structs in one query. Covers INSERT, UPDATE, and DELETE, including `ON_CONFLICT` and `RETURNING`.
- **Weaknesses:** Requires generation against a database. Queries are written in an uppercase SQL DSL. No SQL Server support: PostgreSQL, MySQL, and SQLite are supported, with CockroachDB and MariaDB tested through their compatible protocols. Migrations and change tracking are not documented.
- **Maintenance:** Active. Latest release v2.16.1 (October 2026).

## Bob

- **Approach:** [Toolkit](https://bob.stephenafamo.com/docs/) with a dialect-specific query builder, database-first model generation, factories, and typed functions generated from SQL.
- **Strengths:** The query builder aims to allow nearly any query a dialect supports. [Generated typed filters](https://bob.stephenafamo.com/docs/code-generation/usage) (`models.SelectWhere.Jets.ID.EQ(100)`). Setter structs with `omit.Val` handle "unset" explicitly, which avoids the zero-value problem. Relations load through joins or extra queries, and test factories are generated.
- **Weaknesses:** Models are generated, not hand-written. Supports PostgreSQL, MySQL and MariaDB, and SQLite, and the [generator list](https://bob.stephenafamo.com/docs/code-generation/intro) has no SQL Server. Migrations are left to another tool. Change tracking is not documented.
- **Maintenance:** Active, pre-1.0. Latest release v0.50.0 (August 2026).

## sqlx

- **Approach:** [Extensions to `database/sql`](https://pkg.go.dev/github.com/jmoiron/sqlx). You write SQL, and sqlx scans results.
- **Strengths:** Small, and existing `database/sql` code can adopt it gradually. Struct scanning, named parameters, and `In` for slice expansion. Works with drivers registered for its bind styles, and `BindDriver` adds more at runtime.
- **Weaknesses:** Queries are unchecked strings, and result mapping fails at runtime. Relations, change tracking, and migrations are out of scope.
- **Maintenance:** Quiet. Latest release v1.4.0 (April 2024), last repository push August 2024. Not archived.

## upper/db

- **Approach:** [Data access layer](https://upper.io/v4/getting-started/) with an agnostic API, a [SQL builder](https://upper.io/v4/getting-started/sql-builder-api/), and an optional ORM-like layer.
- **Strengths:** One API across [PostgreSQL, MySQL, CockroachDB, SQL Server, SQLite, QL, and MongoDB](https://upper.io/v4/adapter/). [Records, stores, and hooks](https://upper.io/v4/tutorial/record-store-and-hooks/) are opt-in.
- **Weaknesses:** Conditions are `db.Cond` maps keyed by column name, for example `db.Cond{"id": 7808}`, so they have no compile-time checks. Hooks run only through record methods. Relation loading, change tracking, and migrations are not documented.
- **Maintenance:** Moderate. Latest release v4.10.0 (March 2025), repository activity in October 2026.

## xorm

- **Approach:** [Reflection-based ORM](https://gitea.com/xorm/xorm/src/branch/main/README.md) with struct tags and chainable APIs.
- **Strengths:** Schema sync, read/write splitting, optimistic locking through a [`version` tag](https://xorm.io/docs/chapter-02/4.columns/), and a built-in LRU cache. Supports MySQL, MariaDB, TiDB, PostgreSQL, CockroachDB, SQLite, and SQL Server, with experimental Oracle support. Database reverse engineering through a separate tool.
- **Weaknesses:** Conditions and joins are strings. Update skips zero values by default. Relation support is described only as "simple cascade loading". `Sync` is its schema tool, and the README describes no versioned migrations or change tracking.
- **Maintenance:** Active. Latest tag v1.4.3 (September 2026). Hosted on Gitea, not GitHub.

## pop

- **Approach:** [ORM-like layer over sqlx](https://gobuffalo.io/documentation/database/pop/) with ActiveRecord-influenced conventions, part of the Buffalo toolkit but usable alone.
- **Strengths:** Conventions for table names and timestamps. [Struct-tag associations](https://gobuffalo.io/documentation/database/relations/) with a batched `EagerPreload` mode. [Migrations](https://gobuffalo.io/documentation/database/migrations/) in Fizz or SQL with up and down files, run by the Soda CLI. Supports PostgreSQL, CockroachDB, MySQL, and SQLite.
- **Weaknesses:** [Queries](https://gobuffalo.io/documentation/database/querying/) and association names are strings. The default `Eager` mode runs one query per parent row. No SQL Server support listed.
- **Maintenance:** Active. Latest release v6.4.1 (September 2026).

## Comparison

The Typed queries column asks whether a mistyped column or wrong value type fails at compile time. The Change tracking column means EF Core-style snapshot diffing with a save call. "Not documented" is explained at the top of this page.

| Library | Typed queries | Codegen required | Relations | Change tracking | Migrations | Dialects |
| --- | --- | --- | --- | --- | --- | --- |
| GORM | Only with optional generated helpers | No, optional | Yes, strings in `Preload` | Not documented | AutoMigrate | MySQL, PostgreSQL, GaussDB, SQLite, SQL Server, TiDB, Oracle |
| ent | Yes, generated | Yes | Yes, typed `With<Edge>` | Not documented | Automatic, or versioned with Atlas | MySQL, MariaDB, PostgreSQL, SQLite; preview CockroachDB, TiDB; experimental Gremlin |
| Bun | No | No | Yes, strings | Not documented | Go or SQL files, AutoMigrator from models | PostgreSQL, MySQL, MariaDB, SQL Server, SQLite, Oracle |
| sqlc | Yes, static SQL checked at generation | Yes | Out of scope, write JOINs | No, out of scope | No, parses other tools' files | Go: MySQL, PostgreSQL stable; SQLite, ClickHouse, DuckDB, Spanner, SQL Server beta |
| SQLBoiler | Yes, generated, strings also accepted | Yes | Yes, typed or strings | Not documented | No | PostgreSQL, MySQL, SQL Server, SQLite; CockroachDB via third party |
| go-jet | Yes, generated | Yes | Nested result mapping | Not documented | Not documented | PostgreSQL, MySQL, SQLite; CockroachDB, MariaDB via compatible protocols |
| Bob | Yes, generated | For models | Yes, generated | Not documented | No | PostgreSQL, MySQL, MariaDB, SQLite |
| sqlx | No | No | Out of scope | No | No | `database/sql` drivers |
| upper/db | No | No | Not documented, JOINs in builder | Not documented | Not documented | PostgreSQL, MySQL, CockroachDB, SQL Server, SQLite, QL, MongoDB |
| xorm | No | No | Simple cascade loading, string joins | Not documented | Sync | MySQL, MariaDB, TiDB, PostgreSQL, CockroachDB, SQLite, SQL Server; experimental Oracle |
| pop | No | No | Yes, strings | Not documented | Fizz or SQL, up and down | PostgreSQL, CockroachDB, MySQL, SQLite |
| ant (target) | Yes, with [limits](03-vision.md#known-limits) | No | Yes, typed pointers | Yes | Structs diffed against a snapshot, into plain SQL files | PostgreSQL, with MySQL, MariaDB, SQLite, and SQL Server planned |

The gap ant targets is the combination of hand-written structs, compiler-checked queries, no generation step, and change tracking. No row above has all four. ant does not exist yet, so its row states a goal, not a result.
