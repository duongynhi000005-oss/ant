# Multi-tenancy

To keep customers apart in one database, every query needs a tenant condition. The way to make that reliable is to make it impossible to forget.

## A global filter

A filter applies to every query of a model, and it reaches joins, `Include`, `Has`, and subqueries by itself, because they are all built on the same model. It runs when the query runs, so it can read the request's context.

```go
type tenantKey struct{}

func WithTenant(ctx context.Context, id int64) context.Context {
    return context.WithValue(ctx, tenantKey{}, id)
}

func (Invoice) Configure(b *ant.ModelBuilder[Invoice]) {
    b.Filter("tenant", func(ctx context.Context) ant.Cond {
        id, ok := ctx.Value(tenantKey{}).(int64)
        if !ok {
            return ant.FailedCond(ErrNoTenant) // fail closed
        }
        return ant.Eq(&I.TenantID, id)
    })
}
```

**Fail closed.** With no tenant in the context, the filter returns an error, and the query fails. Returning an empty condition instead would show every tenant's rows to code that forgot to set one, which is the worst way this can go wrong.

## Every model at once

An embedded struct with `ConfigureMixin` brings the column, the filter, and the value on insert, so a model cannot get part of it wrong:

```go
type Tenanted struct {
    TenantID int64
}

func (Tenanted) ConfigureMixin(m *ant.Mixin) {
    m.Filter("tenant", func(ctx context.Context) ant.Cond {
        id, err := TenantOf(ctx)
        if err != nil {
            return ant.FailedCond(err)
        }
        return m.Eq("tenant_id", id)
    })
}

func (t *Tenanted) BeforeInsert(ctx context.Context) error {
    id, err := TenantOf(ctx)
    t.TenantID = id
    return err
}

type Invoice struct {
    Tenanted
    ID    int64
    Total Money
}
```

The [ledger example](https://github.com/skudra-go/ant-examples/tree/main/examples/ledger) is built this way.

## What a filter does not cover

- `ant.Raw` runs your SQL as written, so a raw query has to name the tenant itself.
- `q.IgnoreFilters()` turns filters off, on purpose, for admin tools. Search for it in reviews.
- Unique constraints are per table, so a value unique per tenant needs the tenant in the key: `b.Index(&U.TenantID).And(&U.Email).Unique()`.

For a harder boundary, PostgreSQL's row-level security enforces the same rule in the database. Set the tenant per transaction with `SET LOCAL`, inside `db.Tx`.
