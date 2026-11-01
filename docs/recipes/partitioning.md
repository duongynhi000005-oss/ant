# Partitioned tables

A partitioned table is many tables behind one name. Queries that filter on the partition column skip the rest, and dropping a whole partition frees its space at once, where deleting the same rows would take hours and leave the table bloated. It suits append-heavy data with a retention rule.

```go
func (Event) Configure(b *ant.ModelBuilder[Event]) {
    b.CompositeKey(ant.F(&E.ID), ant.F(&E.OccurredAt))
    b.PartitionByRange(ant.F(&E.OccurredAt))
}
```

Migrations create the parent table. A partitioned table's primary key and unique constraints must include the partition column, because PostgreSQL only enforces uniqueness per partition, so the key is composite.

## Keeping the months

Partitions come and go on a schedule, so they are not migrations, and `drift` ignores them. `postgres.MonthlyPartitions` manages them:

```go
months, err := postgres.MonthlyPartitions[Event](db)

months.Ensure(ctx, time.Now(), 3)          // this month and the next three
months.List(ctx)                           // what exists
months.DropBefore(ctx, cutoff)             // whole months that ended by cutoff
```

`Ensure` is safe to repeat, and to run from several instances. Run it at startup, since a row for a month with no partition is an error, and again on a schedule.

## Related tables

A foreign key to a partitioned table needs the whole key, which a relation cannot express. Keep the relation for loading, and declare the constraint apart:

```go
func (Person) Configure(b *ant.ModelBuilder[Person]) {
    b.PartitionByRange(ant.F(&P.OccurredAt))
    b.BelongsTo(&P.Event).ForeignKey(&P.EventID).References(&E.ID).NoForeignKey().NoIndex()
    b.ForeignKey(ant.F(&P.EventID), ant.F(&P.OccurredAt)).
        References(ant.F(&E.ID), ant.F(&E.OccurredAt)).
        OnDelete(ant.Cascade)
}
```

Children copy the parent's time, which puts both rows in matching months. Name both columns in joins, so PostgreSQL pairs each month with its own:

```go
q.JoinOn[Event](ant.EqCol(&P.EventID, &E.ID), ant.EqCol(&P.OccurredAt, &E.OccurredAt))
```

Drop children first: `DropBefore` on `Person`, then on `Event`.

## Limits

- `.Concurrently()` does not work on a partitioned table.
- A row outside every partition is an error. A default partition would hide that, then block creating the month later.
- Filter on the partition column wherever you can, or PostgreSQL searches every month.

The [audit example](https://github.com/skudra-go/ant-examples/tree/main/examples/audit) is built on all of this, with retention on a cron schedule.
