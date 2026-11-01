# Go ORM and database library feature reference

Catalog of what Go database libraries let users do. Each library lists only what its documentation or public API shows, checked on 2026-10-31. An absent bullet means it was not found, not that the feature is confirmed missing. Empty headings are omitted.

## GORM

Version v1.31.2. Sources at the end of this section apply to all its headings.

### Model definition and mapping

- Struct models with `gorm` tags
- `gorm.Model` embeddable base
- Pluralized table name convention
- `TableName` method
- Temporary table name with `Table`
- `NamingStrategy`
- Column name tag
- Column type, size, precision, and scale tags
- `notNull`, `unique`, and `default` tags
- `check` tag
- Column `comment` tag
- Field-level permissions with `<-`, `->`, and `-`
- Embedded structs
- `embeddedPrefix`
- Indexes through `index` and `uniqueIndex` tags
- Composite, shared, and multiple indexes
- Index field priority
- Single-table inheritance
- Foreign key constraint tags

### Keys and identity

- `ID` primary key by convention
- `primaryKey` tag
- Composite primary keys
- `autoIncrement` and `autoIncrementIncrement` tags

### Relations

- Belongs to
- Has one
- Has many
- Many to many
- Self-referential has one, has many, and many to many
- Polymorphic associations
- Override foreign key and references
- Back-references for many to many
- Customized join table
- Composite foreign keys
- Foreign key constraint behavior
- Association mode: find, append, replace, delete, clear, and count
- Batch association handling

### Querying

- String, struct, and map conditions
- Inline conditions
- `Not` and `Or` conditions
- Select specific fields
- Smart select into smaller structs
- `Order`
- `Limit` and `Offset`
- `Group` and `Having`
- `Distinct`
- `Joins`
- `Scan` into structs
- Subqueries
- `FROM` subqueries
- Group conditions
- `IN` with multiple columns
- Named arguments
- Find to map
- `FirstOrInit`
- `Pluck`
- Locking clauses
- Optimizer, index, and comment hints
- Raw SQL
- Clause builder and statement modifiers
- Generics API with `gorm.G[T]`
- Generated typed field helpers through the GORM CLI

### Loading

- `Preload` in a separate query
- `Joins` preloading
- Preload all with `clause.Associations`
- Preload with conditions
- Custom preloading SQL
- Nested preloading
- Embedded preloading

### Writing

- `Create`
- Create with selected fields
- Batch insert
- Create from map
- Create from SQL expression
- Create with associations
- Default values
- Upsert with `clause.OnConflict`
- `FirstOrCreate`
- `Save`
- `Update` single column
- `Updates` with struct or map
- Update selected fields
- Batch updates
- Block global updates
- Update with SQL expression
- Update from subquery
- Update without hooks or time tracking
- Returning data from modified rows
- Check whether a field changed in a hook
- `Delete`
- Batch delete
- Block global delete
- Returning data from deleted rows
- Soft delete with `DeletedAt`
- Find soft-deleted records with `Unscoped`
- Delete permanently
- Delete flag
- Auto-save associations, with `FullSaveAssociations` and skip options

### Change tracking and unit of work

- Row-level check of changes in update hooks through `Changed`

### Transactions and connections

- Default transaction per write
- Disable default transaction
- `Transaction` closure
- Nested transactions
- Manual transaction control
- `SavePoint` and `RollbackTo`
- Connection pool settings through `database/sql`
- Read/write splitting through DBResolver
- Load balancing in DBResolver
- Sharding plugin
- Context modes and timeouts

### Lifecycle

- Before and after hooks for create, save, update, and delete
- After find hook
- Skip hooks
- Callbacks API: register, delete, replace, and order
- Plugins
- Scopes
- Dynamic table scopes
- Pagination through scopes
- Session settings with `Set` and `Get`

### Value handling

- `Scanner` and `Valuer` custom types
- `GormDataTypeInterface`
- `GormValuerInterface`
- Clause expressions from data types
- Serializer tag
- JSON, gob, and unixtime serializers
- Custom serializers
- Time tracking with `CreatedAt` and `UpdatedAt`, including Unix and nano variants

### Schema and migrations

- `AutoMigrate`
- Migrator interface for tables, columns, views, constraints, and indexes
- Atlas integration
- Disable foreign key constraints when migrating

### Performance

- Prepared statement cache
- `FindInBatches`
- Iteration with `Rows`
- `CreateBatchSize`
- `QueryFields`
- Index hints

### Developer tooling

- Logger with levels
- Custom logger
- `Debug`
- `DryRun`
- `ToSQL`
- Prometheus metrics plugin
- Dialect-translated errors
- `ErrRecordNotFound`

### Database support

- MySQL
- PostgreSQL
- GaussDB
- SQLite
- SQL Server
- TiDB
- Oracle
- ClickHouse driver

**Sources:** [GORM docs](https://gorm.io/docs/index.html), [models](https://gorm.io/docs/models.html), [conventions](https://gorm.io/docs/conventions.html), [associations](https://gorm.io/docs/associations.html), [query](https://gorm.io/docs/query.html), [advanced query](https://gorm.io/docs/advanced_query.html), [update](https://gorm.io/docs/update.html), [delete](https://gorm.io/docs/delete.html), [hooks](https://gorm.io/docs/hooks.html), [migration](https://gorm.io/docs/migration.html), [DBResolver](https://gorm.io/docs/dbresolver.html), [GORM CLI](https://github.com/go-gorm/cli).

## ent

Version v0.14.6.

### Model definition and mapping

- Schema defined as Go code with `Fields` and `Edges`
- Field types and database type overrides
- Custom Go types for fields
- Default values
- Validators, including built-in validators
- `Optional`, `Nillable`, and `Immutable` fields
- Unique fields
- Comments
- Deprecated fields
- Storage key
- Struct tags
- Additional struct fields
- Sensitive fields
- Mixins
- Indexes, including multi-field, on edges, and on edge fields
- Functional indexes through Atlas
- Annotations for custom table name and schema
- Database comments
- Views

### Keys and identity

- ID field configuration
- Globally unique IDs
- Composite identifiers through edge schema

### Relations

- Edges with `To` and `From`
- One-to-one, two types, same type, and bidirectional
- One-to-many, two types and same type
- Many-to-many, two types, same type, and bidirectional
- Edge fields
- Edge schema for join tables
- Required and immutable edges
- Storage key, struct tags, indexes, and comments on edges

### Querying

- Generated field predicates
- Edge predicates
- `Not`, `Or`, and `And`
- Custom predicates
- JSON predicates
- Comparing fields
- Graph traversals
- `Limit`, `Offset`, and ordering
- Order by edge count or edge field
- Custom ordering
- Aggregation
- Group by, group by edge, and having
- Row-level locks feature
- Custom SQL modifiers feature
- SQL raw API feature
- EntQL filtering feature

### Loading

- Eager loading with `With<Edge>`
- Filtered and limited eager loading
- Nested eager loading
- Named edges

### Writing

- Create, create bulk
- Update one, update by ID, and update many
- Delete one and delete many
- Upsert feature
- Add, clear, and set edge operations

### Transactions and connections

- Transactional client
- Isolation levels
- Transaction hooks
- `sql.DB` configuration
- pgx integration

### Lifecycle

- Runtime hooks
- Schema hooks
- Hook helpers and evaluation order
- Codegen hooks
- Interceptors and traversers
- Privacy policies and rules
- Multi-tenancy through privacy or interceptors
- Soft delete through interceptors

### Schema and migrations

- Automatic migration with `Schema.Create`
- `WithDropColumn` and `WithDropIndex`
- Offline mode
- Migration hooks
- Atlas migration engine
- Versioned migrations
- Data migrations
- Multi-schema migrations
- Generate schemas from a database through a separate tool

### Developer tooling

- Code generation with templates
- Testing with `enttest`
- CI guidance
- GraphQL integration
- gRPC integration
- Extension API

### Database support

- MySQL
- MariaDB
- PostgreSQL
- SQLite
- CockroachDB, preview
- TiDB, preview
- Gremlin, experimental

**Sources:** [ent docs](https://entgo.io/docs/getting-started), [fields](https://entgo.io/docs/schema-fields), [edges](https://entgo.io/docs/schema-edges), [CRUD](https://entgo.io/docs/crud), [predicates](https://entgo.io/docs/predicates), [paging](https://entgo.io/docs/paging), [aggregation](https://entgo.io/docs/aggregate), [hooks](https://entgo.io/docs/hooks), [interceptors](https://entgo.io/docs/interceptors), [privacy](https://entgo.io/docs/privacy), [feature flags](https://entgo.io/docs/feature-flags), [migrations](https://entgo.io/docs/migrate), [dialects](https://entgo.io/docs/dialects).

## Bun

Version v1.3.0.

### Model definition and mapping

- Struct models with `bun` tags
- `bun.BaseModel`
- Table-level tags for table name, alias, and select target
- Field-level tags
- Name conventions and custom names
- Dynamic table names
- Discard unknown columns option

### Relations

- Has one
- Belongs to
- Has many
- Polymorphic has many
- Many to many

### Querying

- `NewSelect` builder
- Count rows
- `EXISTS`
- Joins
- Subqueries
- Raw queries
- `Where` with placeholders
- `QueryBuilder` shared interface
- `WHERE IN`
- `WherePK`
- `WHERE VALUES`
- Where grouping
- Common table expressions
- `VALUES` lists
- `WithOrder`
- Placeholders: basic, positional, `bun.Ident`, `bun.Safe`, model, and global
- `MERGE`

### Loading

- `Relation` loading by field name

### Writing

- `NewInsert`
- Bulk insert
- Upsert
- Insert maps
- `INSERT ... SELECT`
- `NewUpdate`
- `OmitZero`
- `Set` and `SetColumn`
- Bulk update from a slice through a CTE
- `NewDelete`
- Bulk delete
- `DELETE ... USING`
- `TRUNCATE`
- Soft deletes through a model or a table view

### Transactions and connections

- `Begin` and `RunInTx`
- `IDB` interface
- PostgreSQL advisory locks
- Driver and connection settings through `database/sql`

### Lifecycle

- Model hooks: `BeforeAppendModel`, `BeforeScanRow`, `AfterScanRow`
- Model query hooks for select, insert, update, delete, create table, and drop table
- Query hooks with `AddQueryHook`

### Value handling

- `sql.Scanner` and `driver.Valuer` custom types
- PostgreSQL arrays and data types guides

### Schema and migrations

- `NewCreateTable` and `NewDropTable`
- `ResetModel`
- Go-based migrations
- SQL-based migrations
- Migration groups and group rollback
- `AutoMigrator` generating changes from models
- Fixtures

### Performance

- Cursor pagination guide
- Result scanning into structs, maps, slices, and scalars

### Developer tooling

- `bundebug` query logging
- Logrus and Zap hooks
- OpenTelemetry instrumentation
- Prometheus metrics
- Starter kit

### Database support

- PostgreSQL
- MySQL and MariaDB
- SQLite
- SQL Server
- Oracle

**Sources:** [Bun guide](https://bun.uptrace.dev/guide/), [models](https://bun.uptrace.dev/guide/models.html), [relations](https://bun.uptrace.dev/guide/relations.html), [hooks](https://bun.uptrace.dev/guide/hooks.html), [migrations](https://bun.uptrace.dev/guide/migrations.html), [drivers](https://bun.uptrace.dev/guide/drivers.html), [`bun/migrate`](https://pkg.go.dev/github.com/uptrace/bun/migrate).

## sqlc

Version v1.31.1.

### Model definition and mapping

- Structs generated from SQL schema
- Type overrides
- Column and table renames in configuration
- JSON and `db` struct tag generation
- `sqlc.embed` struct embedding

### Querying

- Annotated SQL queries generate typed functions
- `:one`, `:many`, `:exec`, `:execresult`, `:execrows`, and `:execlastid`
- `sqlc.arg` named parameters
- `sqlc.narg` nullable parameters
- `sqlc.slice`
- `sqlc.embed`

### Writing

- `:copyfrom` bulk inserts
- `:batchexec`, `:batchmany`, and `:batchone`

### Transactions and connections

- `WithTx` on generated queries
- `database/sql` and pgx drivers

### Schema and migrations

- Schema parsing from atlas, dbmate, golang-migrate, goose, sql-migrate, and tern
- `sqlc verify` for schema changes

### Performance

- Prepared queries

### Developer tooling

- `sqlc generate`
- `sqlc compile`
- `sqlc diff`
- `sqlc vet` lint rules
- `sqlc push`
- `sqlc fmt`
- `sqlc parse`
- `sqlc analyze`
- `sqlc createdb`
- Plugins
- CI/CD guidance
- Multiple language outputs

### Database support

- Go: MySQL and PostgreSQL, stable
- Go: SQLite, ClickHouse, DuckDB, Spanner, and SQL Server, beta

**Sources:** [sqlc docs](https://docs.sqlc.dev/en/latest/index.html), [query annotations](https://docs.sqlc.dev/en/latest/reference/query-annotations.html), [macros](https://docs.sqlc.dev/en/latest/reference/macros.html), [CLI](https://docs.sqlc.dev/en/latest/reference/cli.html), [configuration](https://docs.sqlc.dev/en/latest/reference/config.html), [language support](https://docs.sqlc.dev/en/latest/reference/language-support.html), [DDL](https://docs.sqlc.dev/en/latest/howto/ddl.html).

## SQLBoiler

Version v4.19.5, maintenance mode.

### Model definition and mapping

- Full model generation from a database
- Table and column whitelist and blacklist
- Aliases
- Type replacement
- Custom struct tags
- Extending generated models
- Views

### Relations

- Generated relationship helpers
- Basic multiple schema support

### Querying

- Query mod system
- Generated typed where helpers
- `qm.Where`, `qm.Or`, `qm.Select`, `qm.From`, and `qm.Limit`
- Raw queries
- `Bind` into arbitrary structs
- Find by primary key
- `Exists` and `Count`

### Loading

- Recursive eager loading with `Load`
- Typed relation names with `{Model}Rels`
- Relationship loading on the `.R` struct

### Writing

- Insert
- Update, with `boil.Infer` and `boil.Whitelist`
- `UpdateAll`
- Delete
- `DeleteAll`
- Upsert
- Automatic `CreatedAt` and `UpdatedAt`
- Skip or override automatic timestamps
- Automatic `DeletedAt` soft delete
- `qm.WithDeleted`
- `Reload` and `ReloadAll`

### Transactions and connections

- `boil.Executor` interface
- Transactions through `sql.Tx`
- `context.Context` support

### Lifecycle

- Before and after hooks for create, select, update, delete, and upsert
- Skip hooks

### Value handling

- One-dimensional arrays, JSON, and hstore
- Enum types
- Generated and computed columns

### Developer tooling

- Template-based generation
- Generated compatibility tests
- Debug logging
- Out-of-band drivers

### Database support

- PostgreSQL
- MySQL
- SQL Server 2012+
- SQLite3
- CockroachDB through a third-party driver

**Sources:** [SQLBoiler README](https://github.com/aarondl/sqlboiler).

## go-jet

Version v2.16.1.

### Model definition and mapping

- Generated data model types for tables, views, and enums
- Generated SQL builder packages

### Relations

- Result mapping into nested destination structs

### Querying

- `SELECT` with `DISTINCT`, `FROM`, `WHERE`, `GROUP BY`, `HAVING`, `ORDER BY`, `LIMIT`, `OFFSET`, `FOR`, and `UNION`
- `SELECT_JSON` statements
- `WITH` statements
- Column-type enforced expressions
- Query result mapping to arbitrary destinations

### Writing

- `INSERT` with `VALUES`, `MODEL`, `MODELS`, `QUERY`, `ON_CONFLICT` or `ON_DUPLICATE_KEY_UPDATE`, and `RETURNING`
- `UPDATE` with `SET`, `MODEL`, `WHERE`, and `RETURNING`
- `DELETE` with `WHERE`, `ORDER_BY`, `LIMIT`, and `RETURNING`
- `LOCK` statements

### Developer tooling

- `jet` generator CLI
- `stmt.Sql()` parameterized output
- `stmt.DebugSql()` inlined output

### Database support

- PostgreSQL
- MySQL
- SQLite
- CockroachDB, tested through compatibility
- MariaDB, tested through compatibility

**Sources:** [go-jet README](https://github.com/go-jet/jet), [wiki](https://github.com/go-jet/jet/wiki).

## Bob

Version v0.50.0.

### Model definition and mapping

- Database-first model generation
- SQL file driver
- Configuration for aliases, constraints, types, replacements, relationships, and inflections
- Views
- Column selection with `Only`, `Except`, `WithParent`, and `WithPrefix`

### Relations

- Relationships from foreign keys or manual configuration
- Related-through relations
- Related-when relations
- Multi-column and cross-schema foreign keys
- Modifying relationships through generated methods

### Querying

- Dialect-specific query builder
- Generated typed filters
- Distinct and `DISTINCT ON`
- `CASE`
- Select from functions
- Window functions
- Common table expressions
- Joins, including `USING` and cross join
- `UNION` and combined args
- Locking clauses
- Collate in ordering
- Generated typed functions from SQL queries
- Query annotations

### Loading

- `Preload` through joins
- `ThenLoad` through extra queries
- Check whether a relationship is loaded
- `PreloadCount` and `ThenLoadCount`
- `InsertThenLoadCount`
- Direct count loading

### Writing

- Insert and insert many
- Update and update many
- Upsert
- Delete and delete many
- Setter structs with `omit.Val` and `omitnull.Val`

### Lifecycle

- Model hooks
- Register and skip hooks

### Value handling

- Enum generation
- Enum value formatting

### Performance

- `Prepare` statements
- `Cursor` iteration
- `Each` iteration

### Developer tooling

- `bobgen-psql`, `bobgen-mysql`, `bobgen-sqlite`, and `bobgen-sql`
- Test factories with column and relationship mods
- Factory templates, `Build`, and `Create`
- Generator plugins, built-in and custom
- `bob.Executor` interface

### Database support

- PostgreSQL
- MySQL and MariaDB
- SQLite

Automatic timestamps and soft deletes are documented as deliberately ignored.

**Sources:** [Bob docs](https://bob.stephenafamo.com/docs/), [usage](https://bob.stephenafamo.com/docs/code-generation/usage), [relationships](https://bob.stephenafamo.com/docs/code-generation/relationships), [factories](https://bob.stephenafamo.com/docs/code-generation/factories), [ignored features](https://bob.stephenafamo.com/docs/code-generation/ignored-features).

## sqlx

Version v1.4.0.

### Model definition and mapping

- `db` struct tags
- `MapperFunc` name mapping
- Embedded struct scanning

### Querying

- `Get` and `Select`
- `Queryx` and `QueryRowx`
- `NamedQuery`
- `In` slice expansion
- `Named` and `BindNamed`
- `Rebind`

### Writing

- `NamedExec`, including batch inserts from slices
- `MustExec`

### Transactions and connections

- `Beginx`, `MustBegin`, and `BeginTxx`
- `Connx`
- `Connect`, `MustConnect`, and `ConnectContext`
- Context variants of query methods

### Performance

- `Preparex` and `PrepareNamed`
- `Stmtx` and `NamedStmt`

### Developer tooling

- `StructScan`, `MapScan`, and `SliceScan`
- `LoadFile` for SQL files
- `Unsafe` for missing destination fields

### Database support

- Any `database/sql` driver
- Bind type registration with `BindDriver`

**Sources:** [sqlx README](https://github.com/jmoiron/sqlx), [package docs](https://pkg.go.dev/github.com/jmoiron/sqlx).

## upper/db

Version v4.10.0.

### Model definition and mapping

- `db` struct tags
- `db.Record` interface
- `db.Store` with custom methods

### Querying

- `Find` result sets
- `db.Cond` conditions with operator keys
- `db.And` and `db.Or`
- `db.In`, `db.NotIn`, `db.IsNull`, `db.IsNotNull`, and `db.Like`
- `And` chaining
- `OrderBy`, `Limit`, `Offset`, and `Select`
- `GroupBy`
- `Count` and `Exists`
- SQL builder: `Select`, `SelectFrom`, joins, and `Where`
- `Join`, `LeftJoin`, `RightJoin`, `FullJoin`, and `CrossJoin`
- `Using` clause
- Raw `Query` and `QueryRow`

### Writing

- `Insert` and `InsertReturning`
- `Update` and `UpdateReturning`
- `Delete`
- `Truncate`
- `sess.Save` and `sess.Delete` for records
- SQL builder `InsertInto`, `Update`, and `DeleteFrom`

### Transactions and connections

- `Tx` and `TxContext`
- Connection lifetime settings

### Lifecycle

- `Validator`
- Before and after hooks for create, update, and delete

### Performance

- `Paginate`, `Page`, `NextPage`, and `PrevPage`
- `Cursor`
- `Iterator`
- Prepared statement cache

### Developer tooling

- Logger

### Database support

- PostgreSQL
- MySQL
- CockroachDB
- SQL Server
- SQLite
- QL
- MongoDB

**Sources:** [upper/db docs](https://upper.io/v4/getting-started/), [agnostic API](https://upper.io/v4/getting-started/agnostic-db-api/), [SQL builder](https://upper.io/v4/getting-started/sql-builder-api/), [records and hooks](https://upper.io/v4/tutorial/record-store-and-hooks/), [adapters](https://upper.io/v4/adapter/), [package docs](https://pkg.go.dev/github.com/upper/db/v4).

## xorm

Version v1.4.3.

### Model definition and mapping

- Struct mapping with `xorm` tags
- Name mappers and prefix, suffix, and cache mapping
- Column type, name, null, default, and comment tags
- Unique and index tags, including combined indexes
- `extends` embedded structs
- `-`, `->`, and `<-` field direction tags

### Keys and identity

- `pk` tag
- `autoincr` tag

### Relations

- Simple cascade loading with `Cascade` and `NoCascade`
- `Join` usage

### Querying

- `Where`, `And`, and `Or`
- `In` and `NotIn`
- `Cols`, `AllCols`, `MustCols`, and `Omit`
- `OrderBy`, `Asc`, and `Desc`
- `Limit`
- `GroupBy` and `Having`
- `Distinct`
- `Get`, `Find`, and `FindAndCount`
- `Exist` and `Count`
- `Sum`, `Sums`, `SumInt`, and `SumsInt`
- `ForUpdate`
- `IndexHint`
- `Unscoped`
- Raw `Query` and `SQL`
- SQL builder through `xorm.io/builder`

### Writing

- `Insert`, `InsertOne`, and `InsertMulti`
- `Update`, with default skipping of zero values
- `Incr`, `Decr`, and `SetExpr`
- `Delete`
- Soft delete with the `deleted` tag
- `created` and `updated` tags
- `NoAutoTime`

### Change tracking and unit of work

- Optimistic locking with the `version` tag

### Transactions and connections

- `Begin`, `Commit`, and `Rollback`
- `Transaction` closure
- `NewSession`
- Engine group for read/write splitting
- Load balance policies
- Connection pool settings

### Lifecycle

- Processors: before and after insert, update, and delete
- `BeforeSet`, `AfterSet`, and `AfterLoad`

### Schema and migrations

- `Sync` and `SyncWithOptions`
- `CreateTables`, `DropTables`, `CreateIndexes`, and `CreateUniques`
- `DumpAll` and `Import`
- Reverse tool, `xorm.io/reverse`

### Performance

- `Iterate`
- `Rows`
- Built-in LRU cache
- Context cache
- `Prepare`

### Developer tooling

- `ShowSQL`
- Logger and `LogLevel`
- `LastSQL`

### Database support

- MySQL 5 and 8
- MariaDB
- TiDB
- PostgreSQL
- CockroachDB
- SQLite
- SQL Server
- Oracle, experimental

**Sources:** [xorm README](https://gitea.com/xorm/xorm/src/branch/main/README.md), [xorm docs](https://xorm.io/docs/), [column definition](https://xorm.io/docs/chapter-02/4.columns/), [package docs](https://pkg.go.dev/xorm.io/xorm).

## pop

Version v6.4.1.

### Model definition and mapping

- Struct models with `db` tags
- `id` column convention
- Pluralized, underscored table names
- Automatic `created_at` and `updated_at`

### Relations

- `has_many`, `belongs_to`, `has_one`, and `many_to_many` tags
- `fk_id` custom foreign key
- `order_by` for associations
- `BelongsTo`, `BelongsToAs`, and `BelongsToThrough` query helpers

### Querying

- `Where`
- `Order`
- `Limit`
- `Select`
- `Join`, `InnerJoin`, `LeftJoin`, `LeftOuterJoin`, `RightJoin`, and `RightOuterJoin`
- `GroupBy` and `Having`
- `Find`, `First`, `Last`, and `All`
- `Count` and `Exists`
- `RawQuery`
- `Paginate` and `PaginateFromParams`
- `Scope`

### Loading

- `Eager` with named or nested associations
- `EagerPreload`
- `Load` associations on an existing model

### Writing

- `Create` and `ValidateAndCreate`
- `Update` and `ValidateAndUpdate`
- `Save` and `ValidateAndSave`
- `UpdateColumns`
- `UpdateQuery`
- `Destroy`
- `Reload`
- `TruncateAll`

### Transactions and connections

- `Transaction` closure
- `NewTransaction` and context variants
- `Rollback`
- Configuration through `database.yml`

### Lifecycle

- Before and after callbacks for create, update, save, destroy, and find
- `BeforeValidate`
- `AfterEagerFind`
- Validations

### Schema and migrations

- Fizz migrations
- SQL migrations
- Up and down files
- Dialect-specific migration files
- `schema_migration` table
- `soda migrate`, `migrate up`, `migrate down`, and `migrate status`
- `soda create`, `drop`, `reset`, and `schema`

### Developer tooling

- Soda CLI
- `soda generate` for fizz, sql, model, and config
- Instrumented connections
- Logger

### Database support

- PostgreSQL
- CockroachDB
- MySQL
- MariaDB dialect
- SQLite3

**Sources:** [Pop docs](https://gobuffalo.io/documentation/database/pop/), [relations](https://gobuffalo.io/documentation/database/relations/), [migrations](https://gobuffalo.io/documentation/database/migrations/), [querying](https://gobuffalo.io/documentation/database/querying/), [mutations](https://gobuffalo.io/documentation/database/mutations/), [package docs](https://pkg.go.dev/github.com/gobuffalo/pop/v6).

## Atlas (migrations only)

Version v1.3.0. Some features are limited to particular editions, and this list does not separate them.

### Schema and migrations

- Declarative workflow comparing a current and desired state
- Versioned workflow with plan, lint, and apply
- Schema as code in HCL, SQL, or ORM schemas
- Schema inspection, including export to code
- Atlas GORM provider, in standalone or Go program mode
- `migrate apply`, `checkpoint`, `diff`, `down`, `edit`, `hash`, `import`, `lint`, `ls`, `new`, `push`, `rebase`, `rm`, `set`, `show`, `status`, `test`, and `validate`
- `schema apply`, `clean`, `diff`, `fmt`, `inspect`, `lint`, `plan`, `push`, `stats`, `test`, and `validate`
- Migration linting
- Pre-migration checks
- Checkpoints
- Drift detection
- Down migrations
- Schema tests and migrate tests
- Data scripts
- Declarative seed data and lookup rows
- Database security as code

### Developer tooling

- GitHub Actions, GitLab CI, Bitbucket Pipes, Azure DevOps, and CircleCI integrations
- Terraform provider
- Kubernetes operator
- Argo CD integration
- Crossplane integration
- Atlas Cloud

### Database support

- PostgreSQL
- MySQL
- MariaDB
- SQLite
- SQL Server
- ClickHouse
- Redshift
- Oracle
- Spanner
- Snowflake
- Databricks
- CockroachDB
- Aurora DSQL
- Azure HorizonDB
- YugabyteDB
- Azure Fabric

**Sources:** [Atlas docs](https://atlasgo.io/docs), [CLI reference](https://atlasgo.io/cli-reference), [GORM guide](https://atlasgo.io/guides/orms/gorm).
