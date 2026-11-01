# Several databases

## Replicas

For read replicas of one database, use `ant.WithReplicas`, which sends plain reads to replicas in turn, and everything else to the primary.

```go
replica, err := sql.Open("pgx", replicaDSN)

db, err := ant.Open[AppDB](ctx, postgres.Dialect, primaryDSN,
    ant.WithReplicas(replica),            // any number of *sql.DB
    ant.WithStickyWrites(2*time.Second),  // read your own writes
)
```

`ant.OnPrimary(ctx)` sends one call to the primary.

## Different databases

For separate databases, such as one per service area, declare a context for each. Each has its own models, migrations, and `cmd/migrate`.

```go
type AppDB struct { ant.Context; Users ant.Table[User] }
type BillingDB struct { ant.Context; Invoices ant.Table[Invoice] }
```

A relation cannot cross databases. Keep the key, and load the other side in a second query. Mark such a relation `.NoForeignKey()`.

## Routing by tenant

To give each large tenant a database of its own, keep a map of open contexts, and choose one per request. Open them once: an ant context owns a connection pool.

```go
type Router struct {
    mu  sync.RWMutex
    dbs map[string]*AppDB
}

func (r *Router) For(ctx context.Context, tenant string) (*AppDB, error) {
    r.mu.RLock()
    db := r.dbs[tenant]
    r.mu.RUnlock()
    if db != nil {
        return db, nil
    }
    r.mu.Lock()
    defer r.mu.Unlock()
    if db := r.dbs[tenant]; db != nil {
        return db, nil
    }
    db, err := ant.Open[AppDB](ctx, postgres.Dialect, dsnFor(tenant))
    if err != nil {
        return nil, err
    }
    r.dbs[tenant] = db
    return db, nil
}
```

Apply migrations to each with `migrate.Up`, which is safe to call from several processes at once.

A transaction cannot span databases, and ant has no two-phase commit. Use an outbox to carry a change from one to the other.
