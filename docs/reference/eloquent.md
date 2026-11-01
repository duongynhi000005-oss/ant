# Laravel Eloquent feature reference

Catalog of what Laravel Eloquent lets users do, plus the Laravel query builder, migrations, factories, and seeders where they relate to models. Covers **Laravel 13.x** (framework v13.39.0, checked 2026-10-31).

## Model definition and mapping

- Generate model classes with `make:model`
- Inspect a model with `model:show`
- Table name convention
- Custom table name
- Database connection per model
- Default attribute values
- Refresh attributes after writes
- Eloquent strictness configuration
- Prevent lazy loading
- Prevent silently discarding attributes
- Mass assignment with `fillable`
- Mass assignment guard with `guarded`
- Mass assignment and JSON columns
- Mass assignment exceptions
- Hide attributes from serialization
- Append computed values to serialization
- Custom Eloquent collection classes
- Custom pivot models

## Keys and identity

- `id` primary key by convention
- Custom primary key column
- Non-incrementing primary keys
- String primary keys
- UUID keys
- ULID keys
- Composite keys not supported on models, documented limitation
- Compare models with `is` and `isNot`

## Relations

- One to one (`hasOne`)
- One to many (`hasMany`)
- Inverse relation (`belongsTo`)
- Default models for `belongsTo` (`withDefault`)
- Has one of many
- Latest or oldest of many
- Advanced has one of many
- Convert many to has one (`one()`)
- Has one through
- Has many through
- Scoped relationships
- Many to many (`belongsToMany`)
- Retrieve intermediate table columns with `withPivot`
- Custom `pivot` attribute name
- Filter by intermediate columns
- Order by intermediate columns
- Custom intermediate table models
- Pivot timestamps
- Automatically hydrate parent models on children
- Polymorphic one to one
- Polymorphic one to many
- One of many (polymorphic)
- Polymorphic many to many
- Custom polymorphic types (morph map)
- Dynamic relationships
- Relation constraints by chaining query methods
- Dynamic relationship properties

## Querying

- `where`, `orWhere`, and `whereNot`
- `whereAny`, `whereAll`, and `whereNone`
- `whereBetween` and `whereNotBetween`
- `whereIn` and `whereNotIn`
- `whereNull` and `whereNotNull`
- `whereDate`, `whereMonth`, `whereDay`, `whereYear`, and `whereTime`
- `wherePast`, `whereFuture`, `whereToday`, and variants
- `whereColumn`
- Logical grouping with closures
- `whereExists`
- Subquery where clauses
- JSON where clauses
- `whereJsonContains` and `whereJsonLength`
- Full-text where clauses
- Vector similarity clauses
- Raw expressions with `DB::raw`
- `selectRaw`, `whereRaw`, `havingRaw`, `orderByRaw`, and `groupByRaw`
- `select`, `distinct`, and `addSelect`
- `orderBy`, `latest`, `oldest`, and `inRandomOrder`
- `reorder` to remove orderings
- `groupBy` and `having`
- `limit`, `offset`, `skip`, and `take`
- `when` conditional clauses
- `count`, `max`, `min`, `avg`, and `sum`
- `exists` and `doesntExist`
- `pluck` and `value`
- Inner join
- Left and right join
- Cross join
- Advanced join clauses
- Subquery joins
- Lateral joins
- `union` and `unionAll`
- Subquery selects
- Subquery ordering
- Reusable query objects with `tap` and `pipe`
- Reusable query components
- Pessimistic locking with `lockForUpdate` and `sharedLock`
- Query relation existence with `has`
- Query relation existence with `whereHas`
- Inline relation existence with `whereRelation`
- Query relation absence with `doesntHave` and `whereDoesntHave`
- Query `morphTo` relations with `whereHasMorph`
- `whereBelongsTo`
- `find`, `findOr`, and `findOrFail`
- `first`, `firstOr`, and `firstOrFail`
- `firstOrCreate` and `firstOrNew`
- Offset pagination with `paginate`
- `simplePaginate`
- `cursorPaginate`

## Loading

- Lazy loading through dynamic properties
- Eager loading with `with`
- Multiple relations
- Nested eager loading
- Nested `morphTo` eager loading with `morphWith`
- Eager load specific columns
- Eager load by default with `$with`
- `without` and `withOnly`
- Constrained eager loads
- Constrain `morphTo` eager loads
- `withWhereHas`
- Lazy eager loading with `load`
- `loadMissing`
- `loadMorph`
- Automatic eager loading
- Prevent lazy loading
- `withCount`
- `loadCount`
- `withMax`, `withMin`, `withSum`, `withAvg`, and `withExists`
- Count on `morphTo` with `morphWithCount`
- Deferred aggregate loading

## Writing

- `save` a new or existing model
- `create` through mass assignment
- `update`
- Mass update through a query
- `insert` and `insertGetId`
- `insertOrIgnore`
- `insertUsing`
- `updateOrCreate`
- `updateOrInsert`
- `upsert`
- `updateJson` through arrow syntax
- `increment` and `decrement`
- `incrementEach`
- `delete`
- `destroy` by primary key
- Delete through a query
- `truncate`
- Soft deletes
- `trashed`
- `restore`
- `forceDelete`
- `withTrashed` and `onlyTrashed`
- Pruning with `Prunable`
- `MassPrunable`
- `replicate`
- Relation `save` and `saveMany`
- Relation `create` and `createMany`
- `push` to recursively save relations
- `associate` and `dissociate`
- `attach` and `detach`
- `sync` and `syncWithoutDetaching`
- `toggle`
- `updateExistingPivot`
- Touch parent timestamps
- `saveQuietly` and other quiet variants

## Change tracking and unit of work

- `isDirty`
- `isClean`
- `wasChanged`
- `getOriginal`
- `getChanges`
- `refresh` and `fresh`

## Transactions and connections

- `DB::transaction`
- Deadlock retries through a transaction attempts argument
- Manual `beginTransaction`, `commit`, and `rollBack`
- Read and write connections
- `sticky` connections
- Prevent stale reads
- Use the write connection for reads
- Pooled PostgreSQL connections
- Multiple database connections
- Transactional pivot operations
- Observer callbacks after commit
- Connect through URLs
- Vitess analytical queries
- Unprepared statements
- Implicit commits awareness

## Lifecycle

- Model events: `retrieved`, `creating`, `created`, `updating`, `updated`, `saving`, `saved`, `deleting`, `deleted`, `trashed`, `forceDeleting`, `forceDeleted`, `restoring`, `restored`, `replicating`
- Event classes with `$dispatchesEvents`
- Closure listeners in `booted`
- Queueable anonymous listeners
- Observers
- Observers and database transactions
- Mute events with `withoutEvents`
- Save a single model without events
- Global scopes
- Generate scopes with `make:scope`
- Anonymous global scopes
- Remove global scopes
- Local scopes
- Dynamic scopes
- Pending attributes
- Migration events

## Value handling

- Accessors
- Mutators
- Accessor caching
- Value objects from multiple attributes
- Mutate multiple attributes
- `casts` definition
- Primitive casts
- Stringable casting
- Array and JSON casting
- Array object and collection casting
- Vector casting
- Binary casting
- Date casting
- Date serialization format
- Enum casting
- Arrays of enums
- Encrypted casting
- Encrypted cast key rotation
- Query time casting with `withCasts`
- Custom casts
- Value object casting
- Inbound-only casts
- Cast parameters
- Castables
- Anonymous cast classes
- Array and JSON serialization of models
- `toArray` and `toJson`
- `makeVisible`, `makeHidden`, and `setHidden`
- API resources
- Resource collections
- Conditional attributes and relationships
- JSON:API resources

## Schema and migrations

- `make:migration`
- Migration up and down methods
- Migration connection setting
- Skip a migration with `shouldRun`
- `migrate`
- `migrate --isolated`
- `migrate --force`
- `migrate:rollback`
- `migrate:reset`
- `migrate:refresh`
- `migrate:fresh`
- `migrate:status`
- `schema:dump` squashing
- Schema builder for create, update, rename, and drop table
- Table and column existence checks
- Table options: engine, charset, collation, comment, temporary
- Column types, including numeric, string, date, binary, JSON, UUID, ULID, spatial, vector, and morph types
- `foreignId`, `foreignIdFor`, `foreignUlid`, and `foreignUuid`
- Column modifiers
- Default expressions
- Column order with `after` and `first`
- Instant column operations
- DDL locking options
- Modify, rename, and drop columns
- Create, rename, and drop indexes
- Unique, primary, full-text, spatial, and vector index types
- Online index creation
- Foreign key constraints
- Toggle foreign key constraints
- `softDeletes` and `timestamps` column helpers
- Migration events
- Seeders with `make:seeder`
- `db:seed`
- `call` additional seeders
- Mute model events in seeders
- Seed through `migrate --seed`
- Force seeding in production

## Performance

- `chunk`
- `chunkById`
- `lazy` and `lazyById`
- `cursor`
- Eager loading specific columns
- Monitor cumulative query time
- Mass prune without loading models

## Developer tooling

- `dd` and `dump` on queries
- `dumpRawSql` and `ddRawSql`
- `toSql`
- `DB::listen` query events
- `DB::whenQueryingForLongerThan`
- `db` command for the database CLI
- `db:show` and `db:table`
- `db:monitor`
- `model:show`
- `make:model`, `make:factory`, `make:seeder`, `make:observer`, and `make:cast`
- Model factories
- Factory states
- Trashed factory state
- Factory callbacks with `afterMaking` and `afterCreating`
- `make` and `create`
- Attribute overrides
- Factory sequences
- Has-many factory relations
- Belongs-to factory relations
- Many-to-many factory relations with pivot attributes
- Polymorphic factory relations
- Recycle existing models in factories
- Factory discovery conventions
- Eloquent collection helpers: `find`, `fresh`, `load`, `loadMissing`, `modelKeys`, `toQuery`, and more

## Database support

- MariaDB
- MySQL
- PostgreSQL
- SQLite
- SQL Server
- Vitess analytical query mode
- PostgreSQL pooled connections

## Sources

Documentation repository [laravel/docs](https://github.com/laravel/docs), branch `13.x`, revision of 2026-10-27:

- [Eloquent: Getting Started](https://laravel.com/docs/13.x/eloquent)
- [Eloquent: Relationships](https://laravel.com/docs/13.x/eloquent-relationships)
- [Eloquent: Collections](https://laravel.com/docs/13.x/eloquent-collections)
- [Eloquent: Mutators and Casting](https://laravel.com/docs/13.x/eloquent-mutators)
- [Eloquent: API Resources](https://laravel.com/docs/13.x/eloquent-resources)
- [Eloquent: Serialization](https://laravel.com/docs/13.x/eloquent-serialization)
- [Eloquent: Factories](https://laravel.com/docs/13.x/eloquent-factories)
- [Query Builder](https://laravel.com/docs/13.x/queries)
- [Pagination](https://laravel.com/docs/13.x/pagination)
- [Migrations](https://laravel.com/docs/13.x/migrations)
- [Seeding](https://laravel.com/docs/13.x/seeding)
- [Database: Getting Started](https://laravel.com/docs/13.x/database)
