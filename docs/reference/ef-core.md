# Entity Framework Core feature reference

Catalog of what EF Core lets users do. Covers **EF Core 10.0** (latest stable package v10.0.13, checked 2026-10-31). EF Core 11 preview features are excluded.

## Model definition and mapping

- Convention-based model discovery
- Data annotations (mapping attributes)
- Fluent API in `OnModelCreating`
- Grouped configuration with `IEntityTypeConfiguration`
- Remove a built-in convention
- Add a custom convention
- Replace an existing convention
- Pre-convention configuration
- Bulk configuration through the metadata API
- Default type mapping for a CLR type
- Ignore a type in pre-convention configuration
- Include a type in the model
- Exclude a type from the model
- Exclude a type from migrations
- Table name
- Table schema
- Default schema
- Table comment
- Table triggers declaration
- View mapping
- Table-valued function mapping
- Shared-type entity types
- Property bag entity types
- Keyless entity types
- Include or exclude a property
- Column name
- Column data type
- Maximum length
- Precision and scale
- Unicode setting
- Required and optional properties
- Nullable reference type integration
- Column collation
- Database collation
- Column comment
- Column order
- Shadow properties
- Indexer properties
- Backing fields
- Field-only properties
- Field versus property access mode
- Constructor binding to mapped properties
- Read-only properties
- Service injection into entity constructors
- Owned entity types
- Collections of owned types
- Nested owned types
- Owned types in separate tables
- Table splitting
- Entity splitting
- Table-specific facet configuration
- Complex types
- Struct complex types
- Nested complex types
- Optional complex types
- Collections of complex types
- Complex types mapped to JSON
- Complex types with entity inheritance
- Table-per-hierarchy inheritance
- Discriminator configuration
- Shared columns in a hierarchy
- Table-per-type inheritance
- Table-per-concrete-type inheritance
- Indexes, including composite indexes
- Unique indexes
- Index sort order
- Named and multiple indexes
- Filtered indexes
- Included columns
- Indexes on complex type properties
- Check constraints
- Spatial types through NetTopologySuite
- Dynamic models with `IModelCacheKeyFactory`
- Model debug view

## Keys and identity

- Primary key by convention
- Explicit primary key
- Composite primary keys
- Primary key constraint name
- Keys on complex type properties
- Alternate keys
- Generated key values
- Explicit key values
- Temporary key values
- Sequences
- Sequence settings
- Value objects as keys
- Case-insensitive string keys
- Key value comparers

## Relations

- One-to-many, required and optional
- One-to-one, required and optional
- Many-to-many
- Self-referencing one-to-many
- Self-referencing one-to-one
- Self-referencing many-to-many
- Unidirectional relationships
- Relationships with no navigations
- Shadow foreign keys
- Composite foreign keys
- Relationships to alternate keys
- Primary key to primary key one-to-one
- Foreign key constraint name
- Exclude foreign key constraints from migrations
- Foreign key indexes by convention
- Remove foreign key indexes
- Navigation pairing by convention
- `InverseProperty` configuration
- Reference and collection navigations
- Collection type choice for navigations
- Required navigations
- Many-to-many with named join table
- Many-to-many with custom join table foreign key names
- Many-to-many with a join entity class
- Navigations to and from a join entity
- Join table with payload
- Skip navigations
- Many-to-many join table with separate primary key
- Many-to-many without cascading delete
- Cascade delete configuration
- Delete behaviors, including restrict, set null, and client-side variants
- Relationship fixup

## Querying

- LINQ queries against `DbSet`
- Filter with `Where`
- Single entity with `First`, `Single`, and variants
- Projection with `Select`
- Ordering, including descending and secondary orderings
- Offset pagination with `Skip` and `Take`
- Keyset pagination
- Join
- GroupJoin
- SelectMany
- `LeftJoin` and `RightJoin` operators
- GroupBy with aggregates
- Aggregate functions
- Subqueries in LINQ
- Set operations through LINQ
- `FromSql` raw queries
- `FromSqlRaw` and interpolated parameters
- Composing LINQ over raw SQL
- `SqlQuery` for scalar types
- `SqlQuery` for unmapped types
- Unmapped complex types in SQL queries
- Non-query raw SQL with `ExecuteSql`
- `EF.Functions` mappings
- Built-in function mapping
- User-defined scalar function mapping
- Custom SQL in a function mapping
- Queryable table-valued function mapping
- Function nullability configuration
- Niladic, aggregate, scalar, and table-valued functions
- Explicit collation in a query
- JSON column queries
- JSON array element access
- Primitive collection queries
- Primitive collection parameters
- Null comparison semantics
- Relational null semantics option
- Client evaluation in top-level projection
- Explicit client evaluation
- Query tags
- Tag with file name and line number
- Force or prevent query parameterization
- Async LINQ operators
- Full-text search through provider functions
- Vector search on SQL Server
- Spatial queries

## Loading

- Eager loading with `Include`
- Multi-level loading with `ThenInclude`
- Filtered include
- Include on derived types
- Auto-include navigations in the model
- `IgnoreAutoIncludes`
- Explicit loading with `Entry().Reference().Load()` and `Collection().Load()`
- Querying related entities through `Entry().Query()`
- Lazy loading with proxies
- Lazy loading with `ILazyLoader`
- Lazy loading with an injected delegate
- Single query mode
- Split queries with `AsSplitQuery`
- Enable split queries globally
- Consistent ordering for split queries
- Warning for cartesian explosion

## Writing

- Add, update, and remove with `SaveChanges`
- Multiple operations in one `SaveChanges`
- `AddRange`, `UpdateRange`, `AttachRange`, and `RemoveRange`
- Insert a graph of related entities
- Add related entities through a navigation
- Change relationships through navigations or foreign keys
- Remove relationships
- Saving disconnected entities
- Graph handling with `TrackGraph`
- `ExecuteDelete` bulk delete
- `ExecuteUpdate` bulk update
- Update several properties in one `ExecuteUpdate`
- Reference the existing column value in `ExecuteUpdate`
- `ExecuteUpdate` on JSON columns
- Non-expression lambda for `ExecuteUpdateAsync`
- Rows-affected results from bulk operations
- Cascade delete of tracked entities
- Cascade delete in the database
- Cascading nulls
- Batched `SaveChanges` commands
- Soft delete through a query filter, as a documented pattern

## Change tracking and unit of work

- Entity states: Detached, Added, Unchanged, Modified, Deleted
- Property-level change tracking
- Automatic snapshot change detection
- Manual `DetectChanges`
- Disable automatic change detection
- `HasChanges`
- Notification entities
- Change-tracking proxies
- `Attach` and `Update`
- Explicit `Entry().State` changes
- `DbContext.Entry` property access
- Current and original values
- Navigation access through `EntityEntry`
- `ChangeTracker.Entries`
- `DbSet.Local` view
- `Find` and `FindAsync` from tracked entities first
- `ChangeTracker.Clear`
- Identity resolution
- No-tracking queries with `AsNoTracking`
- `AsNoTrackingWithIdentityResolution`
- Default query tracking behavior setting
- Tracking in custom projections
- Temporary values
- Value comparers
- Change tracker debug view
- Change tracker logging
- Native concurrency tokens with `rowversion`
- Application-managed concurrency tokens
- `DbUpdateConcurrencyException` conflict resolution
- `DbContext` as a short-lived unit of work

## Transactions and connections

- Implicit transaction per `SaveChanges`
- Disable automatic transactions
- `BeginTransaction`, commit, and rollback
- Savepoints
- Automatic savepoint in `SaveChanges`
- Isolation levels
- Cross-context transactions
- Provide an external connection
- Share a connection and transaction
- Use an external `DbTransaction`
- `System.Transactions` support
- Connection resiliency with execution strategies
- Custom execution strategy
- Transaction commit failure handling
- `DbContext` pooling
- `DbContext` factory
- `ConfigureDbContext` composition
- Multi-tenancy by database, schema, or filter

## Lifecycle

- `DbContext` events: `SavingChanges`, `SavedChanges`, `SaveChangesFailed`
- `ChangeTracker` events: `Tracked`, `StateChanged`
- Command interceptors
- Connection interceptors
- Transaction interceptors
- `SaveChanges` interceptors
- Materialization interceptors
- Query expression interceptors
- Identity resolution interceptors
- Singleton interceptors
- Global query filters
- Multiple named query filters
- Disable query filters with `IgnoreQueryFilters`
- Disable a named query filter selectively
- Query filters with context data
- Diagnostic listeners

## Value handling

- Value converters
- Pre-defined conversions
- Built-in converter classes
- Bulk-configured value converters
- Enum to string or number conversion
- Composite value object conversion
- Collections of primitives
- Collections of value objects
- Value conversion for encryption, as a documented pattern
- `DateTime.Kind` handling on read
- Default values
- Default value constraint name
- Default SQL expressions
- Computed columns, stored or virtual
- Value generation on add or update
- Creation and update timestamps through database defaults
- Override value generation
- Disable value generation
- JSON columns through `ToJson`
- SQL Server `json` type
- Primitive collection mapping to JSON
- `HierarchyId`

## Schema and migrations

- `migrations add`
- `migrations remove`
- `migrations list`
- `migrations script`
- Idempotent SQL scripts
- `migrations bundle`
- `migrations has-pending-model-changes`
- `database update`
- `database drop`
- Model snapshot file in source control
- Migration history table
- Custom history table name and schema
- Customize generated migration code
- Column rename edits
- Data operations in migrations
- Raw SQL in migrations with `MigrationBuilder.Sql`
- Custom `MigrationOperation`
- Migrations in a separate project
- Migrations for multiple providers
- Team workflows and diverged migration trees
- Rollback scripts
- Reset all migrations
- Apply migrations at runtime
- Migration locking
- `EnsureCreated` and `EnsureDeleted`
- Reverse engineering with `dbcontext scaffold`
- Scaffold selected tables and views
- Scaffold with data annotations
- Customize scaffolding with T4 templates
- Model managed seed data with `HasData`
- `UseSeeding` and `UseAsyncSeeding`
- Custom initialization logic

## Performance

- Compiled queries
- Compiled models
- Precompiled queries
- NativeAOT support
- Query plan caching
- Parameterization for dynamic queries
- Streaming and buffering control
- Internal buffering
- `SaveChanges` batching
- Project only the needed properties
- Inlined and uncorrelated subquery optimizations
- Pruning of unneeded SQL elements
- `DbContext` pooling

## Developer tooling

- Simple logging with `LogTo`
- `Microsoft.Extensions.Logging` integration
- Log filtering by level, category, and event
- Throw or suppress for a specific event
- Sensitive data logging
- Detailed query exceptions
- Log formatting options
- Redaction of inlined constants in logs
- Warning for string concatenation in raw SQL
- `ToQueryString` SQL inspection
- Metrics through `System.Diagnostics.Metrics`
- Event counters (legacy)
- `dotnet ef` CLI
- Package Manager Console tools
- `dbcontext info`
- `dbcontext list`
- `dbcontext optimize`
- `dbcontext script`
- Design-time `DbContext` factory
- In-memory provider
- SQLite in-memory testing
- Testing guidance with a real database

## Database support

- SQL Server and Azure SQL
- SQLite
- PostgreSQL through Npgsql
- MySQL through Pomelo or `MySql.EntityFrameworkCore`
- Oracle
- Azure Cosmos DB
- In-memory provider
- SQL Server vector type
- SQL Server `json` type
- Provider-specific extensions for plugins and services

## Sources

Documentation repository [dotnet/EntityFramework.Docs](https://github.com/dotnet/EntityFramework.Docs), published at [learn.microsoft.com/ef/core](https://learn.microsoft.com/en-us/ef/core/), revision of 2026-10-27:

- [Creating and configuring a model](https://learn.microsoft.com/en-us/ef/core/modeling/)
- [Relationships](https://learn.microsoft.com/en-us/ef/core/modeling/relationships)
- [Querying data](https://learn.microsoft.com/en-us/ef/core/querying/)
- [Loading related data](https://learn.microsoft.com/en-us/ef/core/querying/related-data/)
- [Saving data](https://learn.microsoft.com/en-us/ef/core/saving/)
- [Change tracking](https://learn.microsoft.com/en-us/ef/core/change-tracking/)
- [Transactions](https://learn.microsoft.com/en-us/ef/core/saving/transactions)
- [Connection resiliency](https://learn.microsoft.com/en-us/ef/core/miscellaneous/connection-resiliency)
- [Logging, events, and diagnostics](https://learn.microsoft.com/en-us/ef/core/logging-events-diagnostics/)
- [Managing database schemas](https://learn.microsoft.com/en-us/ef/core/managing-schemas/)
- [Performance](https://learn.microsoft.com/en-us/ef/core/performance/)
- [Testing](https://learn.microsoft.com/en-us/ef/core/testing/)
- [EF Core CLI](https://learn.microsoft.com/en-us/ef/core/cli/dotnet)
- [Database providers](https://learn.microsoft.com/en-us/ef/core/providers/)
- What's new in [EF Core 8](https://learn.microsoft.com/en-us/ef/core/what-is-new/ef-core-8.0/whatsnew), [9](https://learn.microsoft.com/en-us/ef/core/what-is-new/ef-core-9.0/whatsnew), and [10](https://learn.microsoft.com/en-us/ef/core/what-is-new/ef-core-10.0/whatsnew)
