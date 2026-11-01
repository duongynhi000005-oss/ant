# Feature examples

Each folder here is a section of the [roadmap](../../docs/ROADMAP.md), and each file shows one feature: how to use it, then how it behaves at the edges. Start with the test name and the comment above it, and read the subtests as a list of rules.

| Folder | Covers |
| --- | --- |
| [model_definition](model_definition/) | Structs, `ant.Base`, probes, names, `Configure`, nullable and custom types, field roles, embedded structs, views |
| [keys_and_identity](keys_and_identity/) | Primary keys: convention, explicit, generated, string, UUID, and composite |
| [querying](querying/) | Conditions, ordering, aggregates, grouping, subqueries, unions, CTEs, window functions, locks, raw SQL |
| [relations](relations/) | Conventions, configuration, joins, `Has`, `Project`, many-to-many |
| [loading](loading/) | `Include`, shaped and nested includes, `Load`, `Related`, relation aggregates |
| [writing](writing/) | `Insert`, `Update`, `Delete`, the no-condition guard, upserts, bulk writes, `Reload` |
| [change_tracking](change_tracking/) | `SaveChanges`, identity, states, changing relations, graphs, sessions, random edits |
| [transactions](transactions/) | `Tx`, savepoints, `Begin`, isolation, retries, replicas |
| [lifecycle](lifecycle/) | Soft deletes, timestamps, global filters, scopes, hooks, observers, optimistic concurrency |
| [value_handling](value_handling/) | JSON columns, PostgreSQL arrays, full-text search, encrypted fields, points, vectors |
| [performance](performance/) | `Chunk`, `ChunkByKey`, `Page`, `After` |
| [developer_tooling](developer_tooling/) | Interceptors, logging, `ToSQL`, `Explain`, error classification |
| [testing](testing/) | `anttest`: isolation, fixtures, factories |
| [migrations](migrations/) | `diff`, `up`, failures, locking, repeatable files, drift, adopting a database, custom storage and names, squashing, scripts, data migrations, seeds |

Most examples run with tracking off, so that each read shows what the database holds. Those about tracking use `fixture.RunTracked`. Tracking is on by default in applications.

The shared models and helpers are in [fixture](fixture/fixture.go). `fixture.Run` gives each example an empty database, and `fixture.Seed` adds Alice (30), Bob (17), and Carol (40, with a bio).

## Running them

```sh
make db-up DATABASES=postgres
make test-pg
```

Without a database they skip, so `go test ./...` still passes.
