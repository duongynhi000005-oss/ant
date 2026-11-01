# Soft deletes

A soft delete keeps a row but hides it, so a mistake can be undone and history stays intact. Add a field of type `ant.DeletedAt`, and everything else follows.

```go
type Post struct {
    ant.Base
    Title     string
    DeletedAt ant.DeletedAt
}
```

`Delete`, `DeleteByID`, and `ant.Remove` now mark the row instead of removing it, and every query skips marked rows, including joins, `Include`, `Has`, and subqueries.

```go
db.Posts.Where(ant.Eq(&P.Slug, slug)).Delete(ctx)          // marks it

db.Posts.All(ctx)                                          // live rows
db.Posts.WithDeleted().All(ctx)                            // every row
db.Posts.OnlyDeleted().All(ctx)                            // deleted rows

db.Posts.OnlyDeleted().Where(ant.Eq(&P.ID, id)).Restore(ctx)   // undo
db.Posts.WithDeleted().Where(ant.Eq(&P.ID, id)).ForceDelete(ctx)   // really remove
```

## Unique values

A soft-deleted row still holds its value, so a plain unique constraint blocks reusing an email. Index only the live rows:

```go
b.Index(&U.Email).Unique().Where("deleted_at IS NULL")
```

## Clearing out old rows

Soft-deleted rows pile up. `Prune` removes rows for good, in batches, so it does not hold one long transaction:

```go
removed, err := db.Posts.Prune(ctx, ant.Lt(&P.DeletedAt.V, cutoff))
```

`&P.DeletedAt.V` names the column as a time, which is what a condition on when a row was deleted needs. A row that was never deleted holds NULL, so `Lt` skips it.

Rows in a partitioned table go faster by dropping whole months: see [partitioning](partitioning.md).

The blog and the ledger both use this. A closed ledger account is a soft delete, which keeps its history readable.
